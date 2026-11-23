/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// buildAgentEnvVars returns the full set of env vars for an agent container:
// core config, Redis, role, injected secrets (own + inherited from office manager),
// skills, and connectors (MCP servers).
//
// It needs a client.Client to fetch K8s Secrets, KomputerSkills, and
// KomputerConnectors, so it accepts one explicitly so it can be called from
// both the agent reconciler and the squad reconciler.
// buildToolPolicyEnvVars serializes the agent's tool allow/deny lists for the
// agent process. Each var is omitted entirely when its field is empty, so an
// agent with no configured policy sees no env var and keeps default behavior.
func buildToolPolicyEnvVars(spec *komputerv1alpha1.KomputerAgentSpec) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	if len(spec.AllowedTools) > 0 {
		if b, err := json.Marshal(spec.AllowedTools); err == nil {
			envVars = append(envVars, corev1.EnvVar{Name: "KOMPUTER_ALLOWED_TOOLS", Value: string(b)})
		}
	}
	if len(spec.DisallowedTools) > 0 {
		if b, err := json.Marshal(spec.DisallowedTools); err == nil {
			envVars = append(envVars, corev1.EnvVar{Name: "KOMPUTER_DISALLOWED_TOOLS", Value: string(b)})
		}
	}
	return envVars
}

// shouldStartIdle decides whether a pod (solo) or container (squad member)
// being built for an agent should start in wake-idle mode — i.e. expose its
// HTTP server via KOMPUTER_WAKE_IDLE=true without auto-running
// spec.Instructions.
//
// The komputer.ai/wake-idle annotation (see AnnotationWakeIdle) always wins:
// it means an explicit idle wake asked for "nothing should run", regardless
// of what TaskStatus says — necessary because an agent can be annotated while
// TaskStatus is still InProgress/Compacting too (e.g. it was idle-woken right
// after an older task wrapped up but before the API got around to it).
//
// Otherwise, status.TaskStatus records what the agent was doing the last time
// its pod existed, which is exactly what determines whether re-running
// Instructions on a fresh pod is correct:
//   - ""          : a freshly created agent, or a new task just patched into
//     Instructions (see createOrTriggerAgent in komputer-api) —
//     run it.
//   - InProgress,
//     Compacting  : the pod died mid-task — resume by re-running Instructions.
//   - Complete,
//     Error       : the last task already finished — nothing to run, start idle.
func shouldStartIdle(taskStatus komputerv1alpha1.AgentTaskStatus, annotations map[string]string) bool {
	if annotations[komputerv1alpha1.AnnotationWakeIdle] == "true" {
		return true
	}
	switch taskStatus {
	case komputerv1alpha1.AgentTaskComplete, komputerv1alpha1.AgentTaskError:
		return true
	default: // "", InProgress, Compacting
		return false
	}
}

func buildAgentEnvVars(ctx context.Context, c client.Client, agent *komputerv1alpha1.KomputerAgent, config *komputerv1alpha1.KomputerConfig) ([]corev1.EnvVar, error) {
	log := logf.FromContext(ctx)
	redis := config.Spec.Redis

	envVars := []corev1.EnvVar{
		{Name: "KOMPUTER_INSTRUCTIONS", Value: agent.Spec.Instructions},
		{Name: "KOMPUTER_INTERNAL_SYSTEM_PROMPT", Value: agent.Spec.InternalSystemPrompt},
		{Name: "KOMPUTER_SYSTEM_PROMPT", Value: agent.Spec.SystemPrompt},
		{Name: "KOMPUTER_MODEL", Value: agent.Spec.Model},
		{Name: "KOMPUTER_EFFORT", Value: agent.Spec.Effort},
		{Name: "KOMPUTER_AGENT_NAME", Value: agent.Name},
		{Name: "KOMPUTER_NAMESPACE", Value: agent.Namespace},
		{Name: "CLAUDE_CONFIG_DIR", Value: "/workspace/.claude"},
		// Redis config as env vars (no ConfigMap needed).
		{Name: "KOMPUTER_REDIS_ADDRESS", Value: redis.Address},
		{Name: "KOMPUTER_REDIS_DB", Value: fmt.Sprintf("%d", redis.DB)},
		{Name: "KOMPUTER_REDIS_STREAM_PREFIX", Value: redis.StreamPrefix},
	}

	// Hard per-task spending cap (agent spec over template default — callers
	// without a merged template use resolveAgentTaskBudget). Only set when a cap
	// is actually configured; the agent prefers a per-request value over this env
	// var, so a changed budget takes effect on the next task without waiting for
	// a pod restart.
	if taskBudget := resolveAgentTaskBudget(ctx, c, agent); taskBudget != "" {
		envVars = append(envVars, corev1.EnvVar{Name: "KOMPUTER_TASK_BUDGET_USD", Value: taskBudget})
	}

	// Redis password from Secret (stays as a Secret, not plaintext).
	if redis.PasswordSecret != nil && redis.PasswordSecret.Name != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name: "KOMPUTER_REDIS_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: redis.PasswordSecret.Name},
					Key:                  redis.PasswordSecret.Key,
				},
			},
		})
	}

	if agent.Spec.Role == "manager" {
		envVars = append(envVars,
			corev1.EnvVar{Name: "KOMPUTER_ROLE", Value: agent.Spec.Role},
			corev1.EnvVar{Name: "KOMPUTER_API_URL", Value: config.Spec.APIURL},
		)
	}

	envVars = append(envVars, buildToolPolicyEnvVars(&agent.Spec)...)

	// S3 session store: the agent saves its session at task end and restores it
	// when it has no local copy. Only present when S3 is configured.
	envVars = append(envVars, buildS3EnvVars(config.Spec.S3, agent)...)

	// Inject env vars from agent secrets as SECRET_<SECRETNAME>_<KEY>.
	injectedSecrets := make(map[string]bool)
	for _, secretName := range agent.Spec.Secrets {
		injectedSecrets[secretName] = true
		secret := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, secret); err != nil {
			log.Error(err, "Failed to get agent secret", "secret", secretName)
			continue
		}
		sanitizedName := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(secretName))
		for key := range secret.Data {
			sanitizedKey := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(key))
			envVars = append(envVars, corev1.EnvVar{
				Name: "SECRET_" + sanitizedName + "_" + sanitizedKey,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
						Key:                  key,
					},
				},
			})
		}
	}

	// Inherit secrets from office manager (sub-agents get the same secrets as their manager).
	if agent.Spec.OfficeManager != "" {
		manager := &komputerv1alpha1.KomputerAgent{}
		if err := c.Get(ctx, types.NamespacedName{Name: agent.Spec.OfficeManager, Namespace: agent.Namespace}, manager); err == nil {
			for _, secretName := range manager.Spec.Secrets {
				if injectedSecrets[secretName] {
					continue
				}
				secret := &corev1.Secret{}
				if err := c.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, secret); err != nil {
					continue
				}
				sanitizedName := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(secretName))
				for key := range secret.Data {
					sanitizedKey := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(key))
					envVars = append(envVars, corev1.EnvVar{
						Name: "SECRET_" + sanitizedName + "_" + sanitizedKey,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
								Key:                  key,
							},
						},
					})
				}
			}
		}
	}

	// Inject skills as SKILL_* env vars (full markdown content).
	// Start with default skills (labeled komputer.ai/default=true), then layer explicit skills on top.
	injectedSkills := make(map[string]bool)
	defaultSkills := &komputerv1alpha1.KomputerSkillList{}
	if err := c.List(ctx, defaultSkills, client.MatchingLabels{"komputer.ai/default": "true"}); err == nil {
		for i := range defaultSkills.Items {
			skill := &defaultSkills.Items[i]
			// Skip if the agent explicitly references this skill (explicit takes precedence).
			if injectedSkills[skill.Name] {
				continue
			}
			injectedSkills[skill.Name] = true
			sanitized := strings.ToUpper(regexp.MustCompile(`[^A-Za-z0-9]`).ReplaceAllString(skill.Name, "_"))
			md := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s", skill.Name, skill.Spec.Description, skill.Spec.Content)
			envVars = append(envVars, corev1.EnvVar{
				Name:  "SKILL_" + sanitized,
				Value: md,
			})
		}
	}

	for _, skillRef := range agent.Spec.Skills {
		skillNs := agent.Namespace
		skillName := skillRef
		if parts := strings.SplitN(skillRef, "/", 2); len(parts) == 2 {
			skillNs = parts[0]
			skillName = parts[1]
		}
		if injectedSkills[skillName] {
			continue
		}
		skill := &komputerv1alpha1.KomputerSkill{}
		if err := c.Get(ctx, types.NamespacedName{Name: skillName, Namespace: skillNs}, skill); err != nil {
			log.Info("Skill not found, skipping", "skill", skillRef)
			continue
		}
		injectedSkills[skillName] = true
		sanitized := strings.ToUpper(regexp.MustCompile(`[^A-Za-z0-9]`).ReplaceAllString(skillName, "_"))
		md := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s", skillName, skill.Spec.Description, skill.Spec.Content)
		envVars = append(envVars, corev1.EnvVar{
			Name:  "SKILL_" + sanitized,
			Value: md,
		})
	}

	// Inherit connectors from office manager (sub-agents get the same connectors as their manager).
	connectors := append([]string{}, agent.Spec.Connectors...)
	if agent.Spec.OfficeManager != "" {
		manager := &komputerv1alpha1.KomputerAgent{}
		if err := c.Get(ctx, types.NamespacedName{Name: agent.Spec.OfficeManager, Namespace: agent.Namespace}, manager); err == nil {
			seen := make(map[string]bool, len(connectors))
			for _, conn := range connectors {
				seen[conn] = true
			}
			for _, conn := range manager.Spec.Connectors {
				if !seen[conn] {
					connectors = append(connectors, conn)
				}
			}
		}
	}

	// Resolve connectors → build MCP server config JSON for the agent SDK.
	// Auth tokens are mounted as separate env vars (CONNECTOR_<NAME>_TOKEN) and
	// referenced in the JSON so secrets are not baked in as plaintext.
	if len(connectors) > 0 {
		type mcpServerEntry struct {
			Type       string `json:"type"`
			URL        string `json:"url"`
			TokenEnv   string `json:"tokenEnv,omitempty"`   // env var name holding the Bearer token
			AuthType   string `json:"authType,omitempty"`   // "token", "oauth", or "header"
			HeaderName string `json:"headerName,omitempty"` // custom header name when authType is "header"
		}
		mcpServers := make(map[string]mcpServerEntry)
		for _, connRef := range connectors {
			connNs := agent.Namespace
			connName := connRef
			if parts := strings.SplitN(connRef, "/", 2); len(parts) == 2 {
				connNs = parts[0]
				connName = parts[1]
			}
			conn := &komputerv1alpha1.KomputerConnector{}
			if err := c.Get(ctx, types.NamespacedName{Name: connName, Namespace: connNs}, conn); err != nil {
				log.Info("Connector not found, skipping", "connector", connRef)
				continue
			}
			if conn.Spec.Disabled {
				log.Info("Connector is disabled, skipping", "connector", connRef)
				continue
			}
			sanitized := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(connName))
			entry := mcpServerEntry{Type: "http", URL: conn.Spec.URL, AuthType: conn.Spec.AuthType, HeaderName: conn.Spec.HeaderName}
			// Mount auth secret as env var and reference it.
			if conn.Spec.AuthSecretKeyRef != nil {
				secretName := conn.Spec.AuthSecretKeyRef.Name
				// Cross-namespace connector: secret lives in the connector's namespace,
				// not the agent's. Sync it into the agent's namespace so the pod can
				// mount it via SecretKeyRef (which only resolves within the same ns).
				if connNs != agent.Namespace {
					synced, err := syncConnectorSecretShared(ctx, c, secretName, connNs, agent.Namespace, connName)
					if err != nil {
						log.Info("Failed to sync connector secret across namespaces, skipping", "connector", connRef, "error", err.Error())
						continue
					}
					secretName = synced
				}
				tokenEnvName := "CONNECTOR_" + sanitized + "_TOKEN"
				entry.TokenEnv = tokenEnvName
				envVars = append(envVars, corev1.EnvVar{
					Name: tokenEnvName,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
							Key:                  conn.Spec.AuthSecretKeyRef.Key,
						},
					},
				})
			}
			mcpServers[connName] = entry
		}
		if len(mcpServers) > 0 {
			if mcpJSON, err := json.Marshal(mcpServers); err == nil {
				envVars = append(envVars, corev1.EnvVar{
					Name:  "KOMPUTER_MCP_SERVERS",
					Value: string(mcpJSON),
				})
			}
		}
	}

	return envVars, nil
}

// buildAgentVolumes returns the pod-level Volume entry for an agent's workspace.
// volumeName is the name to give the Volume object (e.g. "workspace" for solo pods,
// "<agentName>-workspace" for squad pods). The caller controls the naming convention.
//
// When ephemeral is true, the workspace is an emptyDir instead of a PVC — it lives
// exactly as long as the pod, with no backing PersistentVolumeClaim at all. sizeLimit,
// when non-nil, caps the emptyDir's size (from storage.size); nil means unbounded,
// matching emptyDir's default. sizeLimit is ignored when ephemeral is false.
func buildAgentVolumes(volumeName, pvcName string, ephemeral bool, sizeLimit *resource.Quantity) []corev1.Volume {
	if ephemeral {
		return []corev1.Volume{
			{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: sizeLimit,
					},
				},
			},
		}
	}
	return []corev1.Volume{
		{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		},
	}
}

// buildAgentVolumeMounts returns the VolumeMount for the agent's own workspace.
// volumeName must match the Volume name returned by buildAgentVolumes.
// For squad pods, sibling mounts must be appended by the caller after this call.
func buildAgentVolumeMounts(volumeName string) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      volumeName,
			MountPath: "/workspace",
		},
	}
}

// mergeEnvVars merges newVars into base: vars in newVars replace base vars with
// the same name; unknown names are appended. Returns the merged slice.
func mergeEnvVars(base []corev1.EnvVar, newVars []corev1.EnvVar) []corev1.EnvVar {
	result := make([]corev1.EnvVar, len(base))
	copy(result, base)

	index := make(map[string]int, len(result))
	for i, env := range result {
		index[env.Name] = i
	}
	for _, env := range newVars {
		if idx, ok := index[env.Name]; ok {
			result[idx] = env
		} else {
			index[env.Name] = len(result)
			result = append(result, env)
		}
	}
	return result
}

// syncConnectorSecretShared is the package-level counterpart of
// KomputerAgentReconciler.syncConnectorSecret. Both the agent and squad
// reconcilers use it via buildAgentEnvVars.
func syncConnectorSecretShared(ctx context.Context, c client.Client, srcName, srcNs, dstNs, connectorName string) (string, error) {
	src := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: srcName, Namespace: srcNs}, src); err != nil {
		return "", fmt.Errorf("get source secret %s/%s: %w", srcNs, srcName, err)
	}

	syncedName := fmt.Sprintf("%s-from-%s", srcName, srcNs)
	dst := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: syncedName, Namespace: dstNs}, dst)
	if errors.IsNotFound(err) {
		dst = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      syncedName,
				Namespace: dstNs,
				Labels: map[string]string{
					"komputer.ai/synced-from-namespace": srcNs,
					"komputer.ai/synced-for-connector":  connectorName,
				},
				Annotations: map[string]string{
					"komputer.ai/source-secret": srcNs + "/" + srcName,
				},
			},
			Type: src.Type,
			Data: src.Data,
		}
		if createErr := c.Create(ctx, dst); createErr != nil {
			return "", fmt.Errorf("create synced secret %s/%s: %w", dstNs, syncedName, createErr)
		}
		return syncedName, nil
	}
	if err != nil {
		return "", fmt.Errorf("get synced secret %s/%s: %w", dstNs, syncedName, err)
	}

	// Update if data or type drifted from the source.
	if dst.Type != src.Type || !secretDataEqual(dst.Data, src.Data) {
		original := dst.DeepCopy()
		dst.Data = src.Data
		dst.Type = src.Type
		if dst.Annotations == nil {
			dst.Annotations = map[string]string{}
		}
		dst.Annotations["komputer.ai/source-secret"] = srcNs + "/" + srcName
		if patchErr := c.Patch(ctx, dst, client.MergeFrom(original)); patchErr != nil {
			return "", fmt.Errorf("patch synced secret %s/%s: %w", dstNs, syncedName, patchErr)
		}
	}
	return syncedName, nil
}

// stripEnvVar removes all entries with the given name from env and returns the
// filtered slice. Logs at Info level if any entry was removed.
func stripEnvVar(env []corev1.EnvVar, name string, log logr.Logger) []corev1.EnvVar {
	filtered := env[:0:len(env)]
	stripped := false
	for _, e := range env {
		if e.Name == name {
			stripped = true
			continue
		}
		filtered = append(filtered, e)
	}
	if stripped {
		log.Info("Stripped user-supplied env var; operator will inject it from AnthropicKeySecretRef", "envVar", name)
	}
	return filtered
}

// s3KeyEnv is the env var that carries the agent's session object key. A
// container without it predates the S3 session store (or S3 being turned on)
// and can't save its own session — see podHasS3Env.
const s3KeyEnv = "KOMPUTER_S3_KEY"

// defaultS3TerminationGraceSeconds gives an interrupted task time to finish and
// save its session before the kubelet kills the pod (the preStop waits up to
// 20s for the task, then up to 20s more after interrupting it).
const defaultS3TerminationGraceSeconds int64 = 60

// buildS3EnvVars returns the S3 session store env for an agent container, or nil
// when S3 isn't configured. Static credentials come from the mirrored credentials
// Secret and are optional, so a missing or broken Secret never blocks the pod —
// the agent just reports a failed save. Without a Secret, the AWS default
// credential chain applies (an IAM role on the pod's ServiceAccount, or env vars
// added through the agent template).
//
// Every name is KOMPUTER_S3_*: operator env vars replace template env vars of
// the same name, so setting AWS_REGION or AWS_ACCESS_KEY_ID here would clobber
// the agent's own AWS settings (e.g. Bedrock's region and credentials).
func buildS3EnvVars(s3 *komputerv1alpha1.S3Spec, agent *komputerv1alpha1.KomputerAgent) []corev1.EnvVar {
	if !s3.IsEnabled() {
		return nil
	}
	env := []corev1.EnvVar{
		{Name: "KOMPUTER_S3_BUCKET", Value: s3.Bucket},
		{Name: "KOMPUTER_S3_REGION", Value: s3.Region},
		{Name: "KOMPUTER_S3_ENDPOINT", Value: s3.Endpoint},
		{Name: "KOMPUTER_S3_FORCE_PATH_STYLE", Value: fmt.Sprintf("%t", s3.ForcePathStyle)},
		{Name: s3KeyEnv, Value: komputerv1alpha1.SessionStoreKey(s3, agent)},
		{Name: "KOMPUTER_S3_PREVIOUS_KEY", Value: komputerv1alpha1.SessionStorePreviousKey(s3, agent)},
	}
	if ref := s3.CredentialsSecret; ref != nil && ref.Name != "" {
		optional := true
		for _, kv := range [][2]string{
			{"KOMPUTER_S3_ACCESS_KEY_ID", komputerv1alpha1.S3CredentialsAccessKeyKey},
			{"KOMPUTER_S3_SECRET_ACCESS_KEY", komputerv1alpha1.S3CredentialsSecretKeyKey},
		} {
			env = append(env, corev1.EnvVar{
				Name: kv[0],
				ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: ref.Name},
					Key:                  kv[1],
					Optional:             &optional,
				}},
			})
		}
	}
	return env
}

// podHasS3Env reports whether an agent container was started with the S3
// session store env, i.e. whether it saves its own session.
func podHasS3Env(c corev1.Container) bool {
	for _, e := range c.Env {
		if e.Name == s3KeyEnv && e.Value != "" {
			return true
		}
	}
	return false
}
