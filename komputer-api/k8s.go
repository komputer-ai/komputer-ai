package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnsureNamespaceExists creates the namespace if missing. The apiserver
// rejects writes to a non-existent namespace, so this must run before the
// agent CR is created. Secret mirroring is owned by the operator — this
// function does NOT touch secrets.
func (k *K8sClient) EnsureNamespaceExists(ctx context.Context, ns string) error {
	namespace := &corev1.Namespace{}
	err := k.client.Get(ctx, types.NamespacedName{Name: ns}, namespace)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get namespace %s: %w", ns, err)
	}
	namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}
	if err := k.client.Create(ctx, namespace); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create namespace %s: %w", ns, err)
	}
	return nil
}

type K8sClient struct {
	client           client.Client
	clientset        *kubernetes.Clientset
	restConfig       *rest.Config
	defaultNamespace string
	// progress coalesces the worker's per-event status writes; see progress_coalesce.go.
	progress progressCoalescer
	// s3cfg caches KomputerConfig.spec.s3; see s3_config.go.
	s3cfg s3ConfigCache
}

func NewK8sClient(defaultNamespace string) (*K8sClient, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(komputerv1alpha1.AddToScheme(scheme))

	config, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &K8sClient{client: c, clientset: cs, restConfig: config, defaultNamespace: defaultNamespace}, nil
}

// WakeAgent wakes a sleeping agent by patching its instructions and setting the lifecycle.
// If lifecycle is empty, the agent stays running after task (default). If "Sleep", it sleeps again.
func (k *K8sClient) WakeAgent(ctx context.Context, ns, name, instructions, internalSystemPrompt, systemPrompt, model, lifecycle string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: name, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return err
	}

	// Patch spec: update instructions, system prompts, and lifecycle
	original := agent.DeepCopy()
	agent.Spec.Instructions = instructions
	agent.Spec.InternalSystemPrompt = internalSystemPrompt
	if systemPrompt != "" {
		agent.Spec.SystemPrompt = systemPrompt
	}
	agent.Spec.Lifecycle = komputerv1alpha1.AgentLifecycle(lifecycle)
	if model != "" {
		agent.Spec.Model = model
	}
	if err := k.client.Patch(ctx, agent, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch spec: %w", err)
	}

	// Patch status: clear sleeping phase and task status so operator creates a pod.
	// Optimistic lock (retried on conflict, re-checking each time) so this never
	// overwrites a phase — e.g. Archived — written after the Get.
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := k.client.Get(ctx, key, agent); err != nil {
			return err
		}
		if agentArchived(agent) && !k.S3Enabled(ctx) {
			// Without the S3 session store, never flip an archived record back to
			// Pending — that would rebuild it with nothing to restore.
			return ErrAgentArchived
		}
		original2 := agent.DeepCopy()
		agent.Status.Phase = komputerv1alpha1.AgentPhasePending
		agent.Status.TaskStatus = ""
		agent.Status.LastTaskMessage = ""
		// Waking is activity: restart the sleepTTL idle clock so the freshly woken agent
		// gets a full idle window instead of inheriting the deadline that just slept it.
		wokeAt := metav1.Now()
		agent.Status.LastActivityAt = &wokeAt
		agent.Status.SleepExpiresAt = nil
		// TaskStatus was just reset out of band; the coalescer's memory of it is stale.
		k.progress.forget(ns, name)
		return k.client.Status().Patch(ctx, agent, client.MergeFromWithOptions(original2, client.MergeFromWithOptimisticLock{}))
	})
}

// IdleWakeAgent wakes a sleeping agent WITHOUT running any task — the opposite of
// WakeAgent, which always carries a (possibly empty-string) task. It leaves
// spec.Instructions and status.TaskStatus completely untouched, and instead sets
// the komputer.ai/wake-idle annotation (komputerv1alpha1.AnnotationWakeIdle) to
// mark the explicit "nothing should run" intent.
//
// TaskStatus can't carry this signal by itself: both "task just completed" and
// "idle-woken" leave it at Complete/Error, but the operator treats those very
// differently — Complete/Error + Lifecycle=Sleep (or AutoDelete) is ALSO its rule
// for immediately re-sleeping (or deleting) the agent the moment its pod exists.
// An earlier version of this method forced TaskStatus to Complete to make
// shouldStartIdle start the pod idle; that broke the common case (most sleeping
// agents are asleep BECAUSE of lifecycle Sleep) by making the operator re-sleep
// the freshly woken pod on its very next reconcile. The annotation is what both
// shouldStartIdle (start idle) and reconcileStatus (skip the immediate
// Sleep/AutoDelete transition) now key off instead — see their doc comments.
//
// The annotation is cleared by the API worker's PatchAgentTaskStatus the moment
// a real task actually starts.
//
// If the agent is not Sleeping, this is a no-op and the current agent is returned
// unchanged, with no annotation set — an idle wake is idempotent, not an error.
func (k *K8sClient) IdleWakeAgent(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerAgent, error) {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: name, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return nil, err
	}
	s3On := k.S3Enabled(ctx)
	if agentArchived(agent) && !s3On {
		return nil, ErrAgentArchived
	}
	if !agentDormant(agent, s3On) {
		// Not asleep yet, but a sleep may be pending (the operator waits out a running
		// task): waking is the user's latest intent, so cancel it.
		if _, pending := agent.Annotations[komputerv1alpha1.AnnotationSleepRequested]; pending {
			metaOriginal := agent.DeepCopy()
			delete(agent.Annotations, komputerv1alpha1.AnnotationSleepRequested)
			if err := k.client.Patch(ctx, agent, client.MergeFrom(metaOriginal)); err != nil {
				return nil, fmt.Errorf("failed to cancel pending sleep request: %w", err)
			}
		}
		return agent, nil
	}

	// Metadata patch: set the wake-idle annotation. Separate from the status patch
	// below because annotations live on ObjectMeta, not the status subresource.
	originalMeta := agent.DeepCopy()
	if agent.Annotations == nil {
		agent.Annotations = map[string]string{}
	}
	agent.Annotations[komputerv1alpha1.AnnotationWakeIdle] = "true"
	delete(agent.Annotations, komputerv1alpha1.AnnotationSleepRequested)
	if err := k.client.Patch(ctx, agent, client.MergeFrom(originalMeta)); err != nil {
		return nil, fmt.Errorf("failed to patch annotations: %w", err)
	}

	original := agent.DeepCopy()
	agent.Status.Phase = komputerv1alpha1.AgentPhasePending
	// TaskStatus is deliberately left untouched — see the doc comment above.
	// Waking is activity: restart the sleepTTL idle clock, same as WakeAgent.
	wokeAt := metav1.Now()
	agent.Status.LastActivityAt = &wokeAt
	agent.Status.SleepExpiresAt = nil
	if err := k.client.Status().Patch(ctx, agent, client.MergeFrom(original)); err != nil {
		return nil, fmt.Errorf("failed to patch status: %w", err)
	}
	return agent, nil
}

// ErrAgentBusy is returned by SleepAgentNow when the agent has a task in
// progress (InProgress or Compacting) — the caller must cancel it first rather
// than have the sleep silently kill it.
var ErrAgentBusy = fmt.Errorf("agent is running a task")

// ErrAgentEphemeral is returned by SleepAgentNow for an effectively ephemeral
// (emptyDir, no PVC) agent — sleeping deletes the pod, which would silently
// wipe such an agent's only copy of its workspace.
var ErrAgentEphemeral = fmt.Errorf("ephemeral agents can't sleep: sleeping deletes the pod and would wipe the workspace")

// SleepAgentNow puts an agent to sleep immediately, regardless of wake-idle or
// sleepTTL, and WITHOUT changing spec.Lifecycle — the explicit "sleep now"
// counterpart to IdleWakeAgent's explicit "wake now" (neither one carries a
// task; compare WakeAgent, which always does). This is what POST
// /agents/{name}/sleep calls, instead of the old approach of patching
// spec.lifecycle to Sleep and hoping the operator's automatic immediate-Sleep
// rule happened to apply — that rule is guarded by wake-idle and sleepTTL
// (rightly so, for its own purpose) and permanently changes the lifecycle
// policy, neither of which a one-off "sleep now" click should do.
//
// Already Sleeping is a no-op success (idempotent), matching IdleWakeAgent's
// idempotency. A task in progress returns ErrAgentBusy without cancelling it.
// An effectively ephemeral workspace returns ErrAgentEphemeral.
//
// Non-squad agents: Phase/PodName are operator-owned (see the CR-status
// field-ownership rule), so this only records intent — it sets the
// komputer.ai/sleep-requested annotation (komputerv1alpha1.AnnotationSleepRequested)
// and lets the operator's reconcileStatus perform the actual pod delete and
// Phase=Sleeping transition, sharing that transition with the Lifecycle=Sleep
// immediate rule. See the annotation's doc comment for the operator-side half.
//
// Squad members: the agent controller skips squad-managed agents entirely
// (status.Squad=true), so there is no reconcile loop to hand the annotation
// to. Instead this drives the transition directly via PatchAgentPhase,
// mirroring the existing Lifecycle=Sleep squad-member path in patchAgent — the
// squad controller deletes the shared pod once every member is Sleeping.
func (k *K8sClient) SleepAgentNow(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerAgent, error) {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: name, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return nil, err
	}

	s3On := k.S3Enabled(ctx)
	if agentDormant(agent, s3On) {
		return agent, nil
	}
	if agentArchived(agent) {
		return nil, ErrAgentArchived
	}
	if agentTaskInProgress(agent.Status.TaskStatus) {
		return nil, ErrAgentBusy
	}
	// With the S3 session store an ephemeral agent can sleep: the operator saves
	// its session first and the wake restores it.
	if !s3On && k.ResolveEffectiveEphemeral(ctx, ns, agent.Spec.AgentConfigSpec) {
		return nil, ErrAgentEphemeral
	}

	if agent.Status.Squad {
		if err := k.PatchAgentPhase(ctx, ns, name, komputerv1alpha1.AgentPhaseSleeping, "Sleeping — requested via API"); err != nil {
			return nil, fmt.Errorf("failed to set sleeping phase: %w", err)
		}
		if err := k.clearSleepMetadata(ctx, ns, name); err != nil {
			return nil, err
		}
		return k.GetAgent(ctx, ns, name)
	}

	original := agent.DeepCopy()
	if agent.Annotations == nil {
		agent.Annotations = map[string]string{}
	}
	agent.Annotations[komputerv1alpha1.AnnotationSleepRequested] = "true"
	if err := k.client.Patch(ctx, agent, client.MergeFrom(original)); err != nil {
		return nil, fmt.Errorf("failed to patch annotations: %w", err)
	}
	return agent, nil
}

// clearSleepMetadata clears the komputer.ai/wake-idle annotation (stale once an
// agent is sleeping) and status.SleepExpiresAt (no future sleep deadline once
// already asleep). Used by SleepAgentNow's squad-member path, which drives the
// sleep transition directly rather than through the operator's reconcileStatus
// (which does the equivalent clearing itself for the non-squad path).
func (k *K8sClient) clearSleepMetadata(ctx context.Context, ns, name string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: name, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return err
	}
	if _, ok := agent.Annotations[komputerv1alpha1.AnnotationWakeIdle]; ok {
		original := agent.DeepCopy()
		delete(agent.Annotations, komputerv1alpha1.AnnotationWakeIdle)
		if err := k.client.Patch(ctx, agent, client.MergeFrom(original)); err != nil {
			return fmt.Errorf("failed to clear wake-idle annotation: %w", err)
		}
	}
	if agent.Status.SleepExpiresAt != nil {
		original := agent.DeepCopy()
		agent.Status.SleepExpiresAt = nil
		if err := k.client.Status().Patch(ctx, agent, client.MergeFrom(original)); err != nil {
			return fmt.Errorf("failed to clear sleepExpiresAt: %w", err)
		}
	}
	return nil
}

// CreateAgentSecrets creates a K8s Secret with agent-specific secrets.
// Keys are prefixed with SECRET_ (e.g. "GITHUB" becomes "SECRET_GITHUB").
// Returns the secret name.
// GetSecretKeys returns the key names from a K8s Secret (not the values).
func (k *K8sClient) GetSecretKeys(ctx context.Context, ns, secretName string) ([]string, error) {
	secret := &corev1.Secret{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, secret); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(secret.Data))
	for key := range secret.Data {
		keys = append(keys, key)
	}
	return keys, nil
}

// GetSecretValue returns a single key's value from a K8s Secret.
func (k *K8sClient) GetSecretValue(ctx context.Context, ns, secretName, key string) (string, error) {
	secret := &corev1.Secret{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, secret); err != nil {
		return "", err
	}
	val, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s", key, secretName)
	}
	return string(val), nil
}

// ResolveSecretEnvVars reads all key-value data from a list of K8s secrets and returns
// a flat map using the SECRET_<SECRETNAME>_<KEY> naming convention.
func (k *K8sClient) ResolveSecretEnvVars(ctx context.Context, ns string, secretNames []string) map[string]string {
	result := make(map[string]string)
	sanitize := strings.NewReplacer("-", "_", ".", "_")
	for _, secretName := range secretNames {
		secret := &corev1.Secret{}
		if err := k.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, secret); err != nil {
			continue
		}
		sanitizedName := strings.ToUpper(sanitize.Replace(secretName))
		for key, val := range secret.Data {
			sanitizedKey := strings.ToUpper(sanitize.Replace(key))
			result["SECRET_"+sanitizedName+"_"+sanitizedKey] = string(val)
		}
	}
	return result
}

// ListSecrets lists K8s Secrets in a namespace. If all=false, only returns secrets
// with the label komputer.ai/managed-by=komputer-ai. If all=true, returns all secrets.
func (k *K8sClient) ListSecrets(ctx context.Context, ns string, all bool) ([]corev1.Secret, error) {
	list := &corev1.SecretList{}
	opts := []client.ListOption{}
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}
	if !all {
		opts = append(opts, client.MatchingLabels{"komputer.ai/managed-by": "komputer-ai"})
	}
	if err := k.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// CreateManagedSecret creates a K8s Secret managed by komputer-ai.
// CreateManagedSecret creates a K8s Secret managed by komputer-ai.
// Keys are stored as-is (no transformation). The operator handles naming conventions
// when injecting as env vars.
func (k *K8sClient) CreateManagedSecret(ctx context.Context, ns, name string, data map[string]string) (*corev1.Secret, error) {
	secretData := make(map[string][]byte, len(data))
	for key, value := range data {
		secretData[key] = []byte(value)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"komputer.ai/managed-by":  "komputer-ai",
				"komputer.ai/secret-name": name,
			},
		},
		Data: secretData,
	}
	if err := k.client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}
	return secret, nil
}

// DeleteManagedSecret deletes a K8s Secret by name.
func (k *K8sClient) DeleteManagedSecret(ctx context.Context, ns, name string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	return k.client.Delete(ctx, secret)
}

// UpdateManagedSecret replaces the data in an existing K8s Secret managed by komputer-ai.
func (k *K8sClient) UpdateManagedSecret(ctx context.Context, ns, name string, data map[string]string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := k.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, secret); err != nil {
		return nil, fmt.Errorf("secret not found: %w", err)
	}
	secretData := make(map[string][]byte, len(data))
	for key, value := range data {
		secretData[key] = []byte(value)
	}
	secret.Data = secretData
	if err := k.client.Update(ctx, secret); err != nil {
		return nil, fmt.Errorf("failed to update secret: %w", err)
	}
	return secret, nil
}

// TTLUpdate is a tri-state TTL change used by patch paths:
// Set=false leaves the field untouched, Set=true with a nil Value clears it,
// and Set=true with a non-nil Value assigns it.
type TTLUpdate struct {
	Set   bool
	Value *metav1.Duration
}

// StringUpdate is a tri-state string change used by patch paths (budgetUSD,
// taskBudgetUSD): Set=false leaves the field untouched, Set=true with Value=""
// clears it, and Set=true with a non-empty Value assigns it. Mirrors TTLUpdate's
// tri-state shape for a plain string field rather than a *metav1.Duration.
type StringUpdate struct {
	Set   bool
	Value string
}

// TouchAgentActivity stamps Status.LastActivityAt to now, resetting the idle clock
// the operator's sleepTTL measures against. Used on paths where the API knows work is
// arriving before the agent's own events can say so (wake, task forward).
//
// Best-effort: a failure here only risks an early sleep, which a wake recovers from,
// so it must never fail the task it precedes. Errors are logged, not returned.
func (k *K8sClient) TouchAgentActivity(ctx context.Context, ns, agentName string) {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		Logger.Warnw("failed to get agent to touch activity", "agent_name", agentName, "error", err)
		return
	}
	original := agent.DeepCopy()
	now := metav1.Now()
	agent.Status.LastActivityAt = &now
	// The pending sleep deadline is stale the moment the clock moves; the operator
	// recomputes it on its next reconcile.
	agent.Status.SleepExpiresAt = nil
	if err := k.client.Status().Patch(ctx, agent, client.MergeFrom(original)); err != nil {
		Logger.Warnw("failed to touch agent activity", "agent_name", agentName, "error", err)
	}
}

// durationEqual compares two optional durations, treating nil as "unset". Used to
// keep patch paths from marking the spec changed when the value is identical.
func durationEqual(a, b *metav1.Duration) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Duration == b.Duration
}

func (k *K8sClient) CreateAgent(ctx context.Context, ns, name, instructions, internalSystemPrompt, officeManager string, cfg komputerv1alpha1.AgentConfigSpec) (*komputerv1alpha1.KomputerAgent, error) {
	if cfg.Model == "" {
		cfg.Model = "claude-opus-5-5"
	}
	if cfg.TemplateRef == "" {
		cfg.TemplateRef = "default"
	}

	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"komputer.ai/agent-name": name,
			},
		},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec:      cfg,
			Instructions:         instructions,
			InternalSystemPrompt: internalSystemPrompt,
			OfficeManager:        officeManager,
		},
	}

	if officeManager != "" {
		agent.Labels["komputer.ai/office"] = officeManager + "-office"
	}

	if err := k.client.Create(ctx, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

// ResolveEffectiveEphemeral resolves whether an agent config is effectively ephemeral:
// its own storage.ephemeral if set, else its template's, else false. Mirrors the
// operator's isAgentEphemeral/applyAgentOverrides resolution so the API rejects the
// same combinations the operator would otherwise have to silently ignore. A template
// that can't be read (not found, RBAC, transient error) resolves to false — this must
// never fail an otherwise-valid request.
func (k *K8sClient) ResolveEffectiveEphemeral(ctx context.Context, ns string, cfg komputerv1alpha1.AgentConfigSpec) bool {
	if cfg.Storage != nil && cfg.Storage.Ephemeral != nil {
		return *cfg.Storage.Ephemeral
	}
	templateRef := cfg.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	tpl := &komputerv1alpha1.KomputerAgentTemplate{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: templateRef, Namespace: ns}, tpl); err == nil {
		return tpl.Spec.Storage.Ephemeral != nil && *tpl.Spec.Storage.Ephemeral
	}
	clusterTpl := &komputerv1alpha1.KomputerAgentClusterTemplate{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: templateRef}, clusterTpl); err == nil {
		return clusterTpl.Spec.Storage.Ephemeral != nil && *clusterTpl.Spec.Storage.Ephemeral
	}
	return false
}

// ResolveEffectiveBudgetUSD resolves an agent config's effective lifetime budget
// (budgetUSD): its own value if set, else its template's default, else "" (no cap).
// Mirrors ResolveEffectiveEphemeral's agent-over-template resolution and the
// operator's resolveAgentLifetimeBudget. A template that can't be read (not found,
// RBAC, transient error) resolves to "" — this must never fail an otherwise-valid
// request.
func (k *K8sClient) ResolveEffectiveBudgetUSD(ctx context.Context, ns string, cfg komputerv1alpha1.AgentConfigSpec) string {
	if cfg.BudgetUSD != "" {
		return cfg.BudgetUSD
	}
	templateRef := cfg.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	tpl := &komputerv1alpha1.KomputerAgentTemplate{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: templateRef, Namespace: ns}, tpl); err == nil {
		return tpl.Spec.BudgetUSD
	}
	clusterTpl := &komputerv1alpha1.KomputerAgentClusterTemplate{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: templateRef}, clusterTpl); err == nil {
		return clusterTpl.Spec.BudgetUSD
	}
	return ""
}

// ResolveEffectiveTaskBudgetUSD resolves an agent config's effective per-task budget
// (taskBudgetUSD): its own value if set, else its template's default, else "" (no
// cap). Mirrors ResolveEffectiveBudgetUSD / the operator's resolveAgentTaskBudget.
func (k *K8sClient) ResolveEffectiveTaskBudgetUSD(ctx context.Context, ns string, cfg komputerv1alpha1.AgentConfigSpec) string {
	if cfg.TaskBudgetUSD != "" {
		return cfg.TaskBudgetUSD
	}
	templateRef := cfg.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	tpl := &komputerv1alpha1.KomputerAgentTemplate{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: templateRef, Namespace: ns}, tpl); err == nil {
		return tpl.Spec.TaskBudgetUSD
	}
	clusterTpl := &komputerv1alpha1.KomputerAgentClusterTemplate{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: templateRef}, clusterTpl); err == nil {
		return clusterTpl.Spec.TaskBudgetUSD
	}
	return ""
}

// IsAgentBudgetExhausted reports whether an agent's EFFECTIVE lifetime budget
// (its own budgetUSD, else its template's default) has been reached or passed by
// status.totalCostUSD. Convenience wrapper combining ResolveEffectiveBudgetUSD and
// isBudgetExhausted for the AgentResponse.BudgetExhausted field.
func (k *K8sClient) IsAgentBudgetExhausted(ctx context.Context, agent *komputerv1alpha1.KomputerAgent) bool {
	budget := k.ResolveEffectiveBudgetUSD(ctx, agent.Namespace, agent.Spec.AgentConfigSpec)
	return isBudgetExhausted(agent.Status.TotalCostUSD, budget)
}

// MergeAgentLabels merges the given labels into the agent's Spec.Labels additively.
// Existing keys are overwritten; no keys are removed.
func (k *K8sClient) MergeAgentLabels(ctx context.Context, ns, agentName string, labels map[string]string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	original := agent.DeepCopy()
	if agent.Spec.Labels == nil {
		agent.Spec.Labels = map[string]string{}
	}
	for k2, v := range labels {
		agent.Spec.Labels[k2] = v
	}
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// UpdateAgentLabels applies label additions/overwrites (set) and removals
// (remove) in a single patch, touching both Spec.Labels and ObjectMeta.Labels.
// The metadata side must be patched directly here: the operator's reconcile
// loop (mergeLabels in komputer-operator) overlays Spec.Labels onto
// metadata.labels additively and never removes a key that disappears from
// Spec.Labels, and the list endpoint's ?label= filter matches metadata labels —
// so a spec-only removal would leave a stale label that keeps matching.
// Removing a key that isn't present is a no-op, not an error.
func (k *K8sClient) UpdateAgentLabels(ctx context.Context, ns, agentName string, set map[string]string, remove []string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	original := agent.DeepCopy()
	if agent.Spec.Labels == nil {
		agent.Spec.Labels = map[string]string{}
	}
	for k2, v := range set {
		agent.Spec.Labels[k2] = v
	}
	for _, k2 := range remove {
		delete(agent.Spec.Labels, k2)
		if agent.ObjectMeta.Labels != nil {
			delete(agent.ObjectMeta.Labels, k2)
		}
	}
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// PatchAgentOverrides patches the podSpec and/or storage overrides on a KomputerAgent CR.
func (k *K8sClient) PatchAgentOverrides(ctx context.Context, ns, agentName string, podSpec *corev1.PodSpec, storage *komputerv1alpha1.StorageSpec) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	original := agent.DeepCopy()
	if podSpec != nil {
		if isEmptyPodSpec(podSpec) {
			agent.Spec.PodSpec = nil
		} else {
			agent.Spec.PodSpec = podSpec
		}
	}
	if storage != nil {
		// Merge field-by-field onto whatever the agent already has, rather than
		// replacing wholesale — a patch that only sets size (the common case: see
		// the CLI's --storage flag) must not silently clear storageClassName or
		// ephemeral, and vice versa.
		var merged komputerv1alpha1.StorageSpec
		if agent.Spec.Storage != nil {
			merged = *agent.Spec.Storage
		}
		if storage.Size != "" {
			merged.Size = storage.Size
		}
		if storage.StorageClassName != nil {
			merged.StorageClassName = storage.StorageClassName
		}
		if storage.Ephemeral != nil {
			merged.Ephemeral = storage.Ephemeral
		}
		if merged.Size == "" && merged.StorageClassName == nil && merged.Ephemeral == nil {
			agent.Spec.Storage = nil
		} else {
			agent.Spec.Storage = &merged
		}
	}
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// isEmptyPodSpec reports whether a PodSpec contains no override fields. Used to
// interpret an empty `{}` from a client as "clear the override".
func isEmptyPodSpec(p *corev1.PodSpec) bool {
	if p == nil {
		return true
	}
	if len(p.Containers) > 0 || len(p.InitContainers) > 0 || len(p.Volumes) > 0 {
		return false
	}
	if p.NodeSelector != nil || p.Tolerations != nil || p.Affinity != nil {
		return false
	}
	if p.PriorityClassName != "" || p.RuntimeClassName != nil || p.ServiceAccountName != "" {
		return false
	}
	return true
}

func (k *K8sClient) ListNamespaces(ctx context.Context) ([]string, error) {
	list, err := k.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list.Items))
	for _, ns := range list.Items {
		names = append(names, ns.Name)
	}
	return names, nil
}

func (k *K8sClient) GetAgent(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerAgent, error) {
	agent := &komputerv1alpha1.KomputerAgent{}
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, agent)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (k *K8sClient) ListAgents(ctx context.Context, ns string, labelFilter map[string]string) ([]komputerv1alpha1.KomputerAgent, error) {
	list := &komputerv1alpha1.KomputerAgentList{}
	var opts []client.ListOption
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}
	if len(labelFilter) > 0 {
		opts = append(opts, client.MatchingLabels(labelFilter))
	}
	if err := k.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

type TemplateInfo struct {
	Name      string
	Scope     string // "cluster" or "namespace"
	Namespace string // populated for namespaced templates
}

func (k *K8sClient) ListTemplates(ctx context.Context, ns string) ([]TemplateInfo, error) {
	var templates []TemplateInfo

	// Cluster-scoped templates
	clusterList := &komputerv1alpha1.KomputerAgentClusterTemplateList{}
	if err := k.client.List(ctx, clusterList); err != nil {
		return nil, fmt.Errorf("failed to list cluster templates: %w", err)
	}
	for _, t := range clusterList.Items {
		templates = append(templates, TemplateInfo{Name: t.Name, Scope: "cluster"})
	}

	// Namespaced templates (if namespace provided)
	if ns != "" {
		nsList := &komputerv1alpha1.KomputerAgentTemplateList{}
		if err := k.client.List(ctx, nsList, client.InNamespace(ns)); err != nil {
			return nil, fmt.Errorf("failed to list namespace templates: %w", err)
		}
		for _, t := range nsList.Items {
			templates = append(templates, TemplateInfo{Name: t.Name, Scope: "namespace", Namespace: t.Namespace})
		}
	}

	return templates, nil
}

func (k *K8sClient) GetAgentPodIP(ctx context.Context, ns, podName string) (string, error) {
	pod := &corev1.Pod{}
	err := k.client.Get(ctx, types.NamespacedName{Name: podName, Namespace: ns}, pod)
	if err != nil {
		return "", err
	}
	if pod.Status.PodIP == "" {
		return "", fmt.Errorf("pod %s has no IP yet", podName)
	}
	return pod.Status.PodIP, nil
}

// DeleteAgent deletes a KomputerAgent CR. The operator will clean up the pod, PVC, and ConfigMap.
func (k *K8sClient) DeleteAgent(ctx context.Context, ns, name string) error {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	return k.client.Delete(ctx, agent)
}

// ForceDeleteAgent deletes an agent without waiting for the operator to save its
// session to S3: it deletes the CR and then drops the komputer.ai/session-store
// finalizer that would otherwise hold the delete until the save succeeds.
func (k *K8sClient) ForceDeleteAgent(ctx context.Context, ns, name string) error {
	if err := k.DeleteAgent(ctx, ns, name); err != nil {
		return err
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		agent := &komputerv1alpha1.KomputerAgent{}
		if err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, agent); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		kept := agent.Finalizers[:0:0]
		for _, f := range agent.Finalizers {
			if f != komputerv1alpha1.FinalizerSessionStore {
				kept = append(kept, f)
			}
		}
		if len(kept) == len(agent.Finalizers) {
			return nil
		}
		original := agent.DeepCopy()
		agent.Finalizers = kept
		return k.client.Patch(ctx, agent, client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{}))
	})
}

// agentHTTPBase returns the base URL for an agent's HTTP endpoint via the
// per-agent K8s Service. Works for both solo and squad-member agents — the
// Service routes to the right container port (named "http") inside the pod.
func agentHTTPBase(ns, agentName string) string {
	return fmt.Sprintf("http://%s-agent-svc.%s:8000", agentName, ns)
}

// execInPodCurl curls localhost on the agent container. Each squad member
// container is named after its agent, so passing agentName as containerName
// targets the right one. For solo pods, container name == agent name too.
// The container reads AGENT_PORT itself so the URL works in both modes.
func (k *K8sClient) execInPodCurl(ctx context.Context, ns, podName, agentName, method, path, body string) ([]byte, error) {
	// Pass the body via stdin (-d @-) to avoid shell-quoting hazards from arbitrary user content.
	args := []string{"sh", "-c", fmt.Sprintf("curl -s -X %s -H 'Content-Type: application/json' --data-binary @- http://localhost:${AGENT_PORT:-8000}%s", method, path)}
	return k.execInContainerWithStdin(ctx, ns, podName, agentName, body, args...)
}

// CancelAgentTask sends a cancel request to the agent's FastAPI endpoint.
func (k *K8sClient) CancelAgentTask(ctx context.Context, ns, podName, agentName string) error {
	err := k.postToAgent(ctx, ns, agentName, "/cancel", "")
	if err != nil {
		// Fallback: kubectl exec curl inside the pod (uses $AGENT_PORT to pick the right port).
		_, execErr := k.execInPodCurl(ctx, ns, podName, agentName, "POST", "/cancel", "")
		return execErr
	}
	return nil
}

// CompactAgentTask triggers a manual compaction on the agent's active session.
// The agent forwards `/compact [instructions]` through the bundled Claude Code CLI,
// which fires the PreCompact hook with trigger="manual".
//
// When the agent is idle and we're spawning a fresh /compact-only session, the
// PreCompact hook can't see any AssistantMessage.usage (no message has been sent
// yet), so we seed `prev_tokens` from the CR's status.totalTokens — that's the
// context size at the end of the last completed task, which is what the SDK
// will resume into. The agent uses it as the tokens_before payload.
//
// Returns an error with substring "status 409" when the agent rejected the
// request — the HTTP handler maps that to a 409 to the caller. We have to parse
// the curl-exec response body manually because `curl -s` succeeds on any HTTP
// status; the in-cluster HTTP path does check the status code but the exec
// fallback (used in LOCAL=true mode) does not.
func (k *K8sClient) CompactAgentTask(ctx context.Context, ns, podName, agentName, instructions string) error {
	bodyMap := map[string]interface{}{}
	if instructions != "" {
		bodyMap["instructions"] = instructions
	}
	// Seed prev_tokens from the CR's last known context size, for the idle path.
	if agent, err := k.GetAgent(ctx, ns, agentName); err == nil && agent.Status.TotalTokens > 0 {
		bodyMap["prev_tokens"] = agent.Status.TotalTokens
	}
	bodyJSON, _ := json.Marshal(bodyMap)
	err := k.postToAgent(ctx, ns, agentName, "/compact", string(bodyJSON))
	if err != nil {
		// Fall back to kubectl exec. Read the response body and look for an
		// error indicator — FastAPI's HTTPException serialises as {"detail": "..."}.
		resp, execErr := k.execInPodCurl(ctx, ns, podName, agentName, "POST", "/compact", string(bodyJSON))
		if execErr != nil {
			return execErr
		}
		var parsed struct {
			Detail string `json:"detail"`
			Status string `json:"status"`
		}
		if jerr := json.Unmarshal(resp, &parsed); jerr == nil && parsed.Detail != "" {
			return fmt.Errorf("agent returned status 409: %s", parsed.Detail)
		}
		if parsed.Status == "" {
			return fmt.Errorf("agent returned unexpected response: %s", string(resp))
		}
	}
	return nil
}

// ForwardTaskToAgent sends a task to an agent's FastAPI endpoint, falling back to
// kubectl exec. taskBudgetUSD is the effective per-task hard cap (agent over
// template, see ResolveEffectiveTaskBudgetUSD); when non-empty, the agent prefers
// it over its own KOMPUTER_TASK_BUDGET_USD env var, so a budget raised or lowered
// after pod startup applies to the very next task without a restart.
func (k *K8sClient) ForwardTaskToAgent(ctx context.Context, ns, podName, agentName, instructions, model, internalSystemPrompt, systemPrompt, taskBudgetUSD string) (int64, error) {
	// Push the idle clock out before the task lands. The agent's own task_started
	// event refreshes LastActivityAt a moment later, but until it round-trips through
	// Redis the operator could still see an agent idle past its sleepTTL and tear the
	// pod down under the task we are about to send.
	k.TouchAgentActivity(ctx, ns, agentName)

	bodyMap := map[string]interface{}{"instructions": instructions}
	if model != "" {
		bodyMap["model"] = model
	}
	if internalSystemPrompt != "" {
		bodyMap["internal_system_prompt"] = internalSystemPrompt
	}
	if systemPrompt != "" {
		bodyMap["system_prompt"] = systemPrompt
	}
	// Sent as a number, not a string — the agent's TaskRequest.task_budget_usd is a
	// float field. A value that fails to parse (should never happen; validated on the
	// way in) is dropped rather than sent malformed.
	if taskBudgetUSD != "" {
		if f, err := strconv.ParseFloat(taskBudgetUSD, 64); err == nil {
			bodyMap["task_budget_usd"] = f
		}
	}
	bodyJSON, _ := json.Marshal(bodyMap)

	respBody, err := k.postToAgentWithResponse(ctx, ns, agentName, "/task", string(bodyJSON))
	if err != nil {
		// Fallback: kubectl exec curl inside the pod — capture stdout to get context_window
		out, execErr := k.execInPodCurl(ctx, ns, podName, agentName, "POST", "/task", string(bodyJSON))
		if execErr != nil {
			return 0, execErr
		}
		respBody = out
	}
	var result struct {
		ContextWindow int64 `json:"context_window"`
	}
	json.Unmarshal(respBody, &result)
	return result.ContextWindow, nil
}

// postToAgent makes an HTTP POST to an agent via its per-agent K8s Service, retrying
// transient failures with backoff (see doAgentRequest).
// When LOCAL=true, skips HTTP entirely to force exec fallback (no in-cluster DNS).
func (k *K8sClient) postToAgent(ctx context.Context, ns, agentName, path, body string) error {
	if os.Getenv("LOCAL") == "true" {
		return fmt.Errorf("LOCAL mode: skipping direct pod HTTP")
	}
	url := agentHTTPBase(ns, agentName) + path
	_, _, err := k.doAgentRequest(ctx, http.MethodPost, url, body, 3*time.Second)
	return err
}

// execInPod runs a command inside a pod's first container.
func (k *K8sClient) execInPod(ctx context.Context, ns, podName string, command ...string) error {
	_, err := k.execInPodWithOutput(ctx, ns, podName, command...)
	return err
}

// execInPodWithOutput runs a command inside a pod's first container and returns stdout.
// For squad pods, prefer execInContainerWithOutput so you target the right agent.
func (k *K8sClient) execInPodWithOutput(ctx context.Context, ns, podName string, command ...string) ([]byte, error) {
	return k.execInContainerWithOutput(ctx, ns, podName, "", command...)
}

// execInContainerWithOutput runs a command in a specific container of a pod and returns stdout.
// containerName is treated as a preferred name:
//   - If the pod has a container with that exact name, exec there (correct for squad pods,
//     where each member container is named after its agent).
//   - Otherwise, fall back to the pod's first container (correct for solo pods, where the
//     container is named generically e.g. "agent").
func (k *K8sClient) execInContainerWithOutput(ctx context.Context, ns, podName, containerName string, command ...string) ([]byte, error) {
	pod := &corev1.Pod{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: podName, Namespace: ns}, pod); err != nil {
		return nil, fmt.Errorf("failed to get pod %s: %w", podName, err)
	}
	if len(pod.Spec.Containers) == 0 {
		return nil, fmt.Errorf("pod %s has no containers", podName)
	}

	resolved := ""
	if containerName != "" {
		for _, c := range pod.Spec.Containers {
			if c.Name == containerName {
				resolved = containerName
				break
			}
		}
		// Also check ephemeral containers — squad members adopted into a running
		// pod live there. Without this, exec falls back to pod.Spec.Containers[0]
		// and silently runs in the wrong container, crossing tasks between agents.
		if resolved == "" {
			for _, c := range pod.Spec.EphemeralContainers {
				if c.Name == containerName {
					resolved = containerName
					break
				}
			}
		}
	}
	if resolved == "" {
		resolved = pod.Spec.Containers[0].Name
	}

	execReq := k.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: resolved,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, clientgoscheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(k.restConfig, "POST", execReq.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return nil, fmt.Errorf("exec failed: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// execInContainerWithStdin is like execInContainerWithOutput but pipes `stdin` to the command.
func (k *K8sClient) execInContainerWithStdin(ctx context.Context, ns, podName, containerName, stdin string, command ...string) ([]byte, error) {
	pod := &corev1.Pod{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: podName, Namespace: ns}, pod); err != nil {
		return nil, fmt.Errorf("failed to get pod %s: %w", podName, err)
	}
	if len(pod.Spec.Containers) == 0 {
		return nil, fmt.Errorf("pod %s has no containers", podName)
	}

	resolved := ""
	if containerName != "" {
		for _, c := range pod.Spec.Containers {
			if c.Name == containerName {
				resolved = containerName
				break
			}
		}
		// Also check ephemeral containers — squad members adopted into a running
		// pod live there. Without this, exec falls back to pod.Spec.Containers[0]
		// and silently runs in the wrong container, crossing tasks between agents.
		if resolved == "" {
			for _, c := range pod.Spec.EphemeralContainers {
				if c.Name == containerName {
					resolved = containerName
					break
				}
			}
		}
	}
	if resolved == "" {
		resolved = pod.Spec.Containers[0].Name
	}

	execReq := k.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: resolved,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
		}, clientgoscheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(k.restConfig, "POST", execReq.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  strings.NewReader(stdin),
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return nil, fmt.Errorf("exec failed: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func (k *K8sClient) GetOffice(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerOffice, error) {
	office := &komputerv1alpha1.KomputerOffice{}
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, office)
	if err != nil {
		return nil, err
	}
	return office, nil
}

func (k *K8sClient) ListOffices(ctx context.Context, ns string) ([]komputerv1alpha1.KomputerOffice, error) {
	list := &komputerv1alpha1.KomputerOfficeList{}
	var opts []client.ListOption
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}
	if err := k.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *K8sClient) DeleteOffice(ctx context.Context, ns, name string) error {
	office := &komputerv1alpha1.KomputerOffice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	return k.client.Delete(ctx, office)
}

// --- KomputerSquad helpers ---

func (k *K8sClient) GetSquad(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerSquad, error) {
	squad := &komputerv1alpha1.KomputerSquad{}
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, squad)
	if err != nil {
		return nil, err
	}
	return squad, nil
}

func (k *K8sClient) ListSquads(ctx context.Context, ns string) ([]komputerv1alpha1.KomputerSquad, error) {
	list := &komputerv1alpha1.KomputerSquadList{}
	var opts []client.ListOption
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}
	if err := k.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *K8sClient) CreateSquad(ctx context.Context, ns string, squad *komputerv1alpha1.KomputerSquad) (*komputerv1alpha1.KomputerSquad, error) {
	if err := k.client.Create(ctx, squad); err != nil {
		return nil, err
	}
	return squad, nil
}

func (k *K8sClient) UpdateSquad(ctx context.Context, squad *komputerv1alpha1.KomputerSquad) (*komputerv1alpha1.KomputerSquad, error) {
	if err := k.client.Update(ctx, squad); err != nil {
		return nil, err
	}
	return squad, nil
}

func (k *K8sClient) DeleteSquad(ctx context.Context, ns, name string) error {
	squad := &komputerv1alpha1.KomputerSquad{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	return k.client.Delete(ctx, squad)
}

// DeletePod deletes a Pod by name. Used to force a squad pod recreation after
// membership changes (the squad reconciler will rebuild it on the next loop).
func (k *K8sClient) DeletePod(ctx context.Context, ns, podName string) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: ns,
		},
	}
	return k.client.Delete(ctx, pod)
}

// FindSquadForAgent lists all squads in the namespace and returns the first one
// that contains agentName as a ref member. Returns nil if the agent is not in any squad.
func (k *K8sClient) FindSquadForAgent(ctx context.Context, ns, agentName string) (*komputerv1alpha1.KomputerSquad, error) {
	squads, err := k.ListSquads(ctx, ns)
	if err != nil {
		return nil, err
	}
	for i := range squads {
		for _, m := range squads[i].Spec.Members {
			if m.Ref != nil && m.Ref.Name == agentName {
				return &squads[i], nil
			}
		}
	}
	return nil, nil
}

// SteerAgent sends a steer message to a running agent pod. It looks up the pod by agent name.
// Returns an error if the agent is not found, has no pod, or the pod is unreachable.
func (k *K8sClient) SteerAgent(ctx context.Context, ns, agentName, message string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: agentName, Namespace: ns}, agent); err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}
	if agent.Status.PodName == "" {
		return fmt.Errorf("agent %s has no running pod", agentName)
	}
	bodyJSON, _ := json.Marshal(map[string]string{"instructions": message})
	_, err := k.postToAgentWithResponse(ctx, ns, agentName, "/task", string(bodyJSON))
	if err != nil {
		// Fallback via kubectl exec (uses $AGENT_PORT inside the container)
		_, err = k.execInPodCurl(ctx, ns, agent.Status.PodName, agentName, "POST", "/task", string(bodyJSON))
	}
	return err
}

func (k *K8sClient) CreateSchedule(ctx context.Context, ns string, req *CreateScheduleRequest) (*komputerv1alpha1.KomputerSchedule, error) {
	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: ns,
			Labels: map[string]string{
				"komputer.ai/schedule-name": req.Name,
			},
		},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Schedule:     req.Schedule,
			Instructions: req.Instructions,
			Timezone:     req.Timezone,
			AutoDelete:   req.AutoDelete,
			KeepAgents:   req.KeepAgents,
			AgentName:    req.AgentName,
		},
	}

	if req.Agent != nil {
		agentSpec := req.Agent.DeepCopy()
		scheduleAgentDefaults(&agentSpec.AgentConfigSpec)
		schedule.Spec.Agent = agentSpec
	}

	if err := k.client.Create(ctx, schedule); err != nil {
		return nil, err
	}
	return schedule, nil
}

func (k *K8sClient) GetSchedule(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerSchedule, error) {
	schedule := &komputerv1alpha1.KomputerSchedule{}
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, schedule)
	if err != nil {
		return nil, err
	}
	return schedule, nil
}

func (k *K8sClient) PatchScheduleCron(ctx context.Context, ns, name, cron string) error {
	return k.PatchScheduleSpec(ctx, ns, name, PatchScheduleRequest{Schedule: &cron})
}

// PatchScheduleSpec patches mutable spec fields on a schedule. nil pointers are skipped.
// AgentName and Agent are mutually exclusive: setting AgentName clears Agent and vice versa,
// matching the create-time validation in KomputerScheduleSpec.
func (k *K8sClient) PatchScheduleSpec(ctx context.Context, ns, name string, req PatchScheduleRequest) error {
	schedule := &komputerv1alpha1.KomputerSchedule{}
	key := types.NamespacedName{Name: name, Namespace: ns}
	if err := k.client.Get(ctx, key, schedule); err != nil {
		return fmt.Errorf("failed to get schedule %s: %w", name, err)
	}
	original := schedule.DeepCopy()
	changed := false
	if req.Schedule != nil && schedule.Spec.Schedule != *req.Schedule {
		schedule.Spec.Schedule = *req.Schedule
		changed = true
	}
	if req.Instructions != nil && schedule.Spec.Instructions != *req.Instructions {
		schedule.Spec.Instructions = *req.Instructions
		changed = true
	}
	if req.Timezone != nil && schedule.Spec.Timezone != *req.Timezone {
		schedule.Spec.Timezone = *req.Timezone
		changed = true
	}
	if req.AutoDelete != nil && schedule.Spec.AutoDelete != *req.AutoDelete {
		schedule.Spec.AutoDelete = *req.AutoDelete
		changed = true
	}
	if req.KeepAgents != nil && schedule.Spec.KeepAgents != *req.KeepAgents {
		schedule.Spec.KeepAgents = *req.KeepAgents
		changed = true
	}
	if req.Suspended != nil && schedule.Spec.Suspended != *req.Suspended {
		schedule.Spec.Suspended = *req.Suspended
		changed = true
	}
	if req.AgentName != nil && schedule.Spec.AgentName != *req.AgentName {
		schedule.Spec.AgentName = *req.AgentName
		if *req.AgentName != "" {
			// Targeting an existing agent — drop the inline template.
			schedule.Spec.Agent = nil
		}
		changed = true
	}
	if req.Agent != nil {
		agentSpec := req.Agent.DeepCopy()
		scheduleAgentDefaults(&agentSpec.AgentConfigSpec)
		schedule.Spec.Agent = agentSpec
		// Inline template replaces any AgentName reference.
		schedule.Spec.AgentName = ""
		changed = true
	}
	if !changed {
		return nil
	}
	return k.client.Patch(ctx, schedule, client.MergeFrom(original))
}

// PatchScheduleStatus applies a status subresource patch built from (original -> updated).
func (k *K8sClient) PatchScheduleStatus(ctx context.Context, original, updated *komputerv1alpha1.KomputerSchedule) error {
	return k.client.Status().Patch(ctx, updated, client.MergeFrom(original))
}

func (k *K8sClient) ListSchedules(ctx context.Context, ns string) ([]komputerv1alpha1.KomputerSchedule, error) {
	list := &komputerv1alpha1.KomputerScheduleList{}
	var opts []client.ListOption
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}
	if err := k.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *K8sClient) DeleteSchedule(ctx context.Context, ns, name string) error {
	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	return k.client.Delete(ctx, schedule)
}

// PatchAgentSpec patches mutable spec fields on a KomputerAgent CR.
func (k *K8sClient) PatchAgentSpec(ctx context.Context, ns, agentName string, model, effort, lifecycle, instructions, templateRef, systemPrompt *string, priority *int32, sleepTTL, deleteTTL, taskTimeout TTLUpdate, budgetUSD, taskBudgetUSD StringUpdate) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	original := agent.DeepCopy()
	changed := false
	if model != nil && *model != agent.Spec.Model {
		agent.Spec.Model = *model
		changed = true
	}
	if effort != nil && *effort != agent.Spec.Effort {
		agent.Spec.Effort = *effort
		changed = true
	}
	if lifecycle != nil && string(agent.Spec.Lifecycle) != *lifecycle {
		agent.Spec.Lifecycle = komputerv1alpha1.AgentLifecycle(*lifecycle)
		changed = true
	}
	if instructions != nil && *instructions != agent.Spec.Instructions {
		agent.Spec.Instructions = *instructions
		changed = true
	}
	if templateRef != nil && *templateRef != agent.Spec.TemplateRef {
		agent.Spec.TemplateRef = *templateRef
		changed = true
	}
	if systemPrompt != nil && *systemPrompt != agent.Spec.SystemPrompt {
		agent.Spec.SystemPrompt = *systemPrompt
		changed = true
	}
	if priority != nil && *priority != agent.Spec.Priority {
		agent.Spec.Priority = *priority
		changed = true
	}
	if sleepTTL.Set && !durationEqual(agent.Spec.SleepTTL, sleepTTL.Value) {
		agent.Spec.SleepTTL = sleepTTL.Value
		changed = true
	}
	if deleteTTL.Set && !durationEqual(agent.Spec.DeleteTTL, deleteTTL.Value) {
		agent.Spec.DeleteTTL = deleteTTL.Value
		changed = true
	}
	if taskTimeout.Set && !durationEqual(agent.Spec.TaskTimeout, taskTimeout.Value) {
		agent.Spec.TaskTimeout = taskTimeout.Value
		changed = true
	}
	if budgetUSD.Set && agent.Spec.BudgetUSD != budgetUSD.Value {
		agent.Spec.BudgetUSD = budgetUSD.Value
		changed = true
	}
	if taskBudgetUSD.Set && agent.Spec.TaskBudgetUSD != taskBudgetUSD.Value {
		agent.Spec.TaskBudgetUSD = taskBudgetUSD.Value
		changed = true
	}
	if !changed {
		return nil
	}
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// getFromAgent makes an HTTP GET to an agent via its per-agent K8s Service, retrying
// transient failures with backoff (see doAgentRequest).
func (k *K8sClient) getFromAgent(ctx context.Context, ns, agentName, path string) ([]byte, error) {
	if os.Getenv("LOCAL") == "true" {
		return nil, fmt.Errorf("LOCAL mode: skipping direct pod HTTP")
	}
	url := agentHTTPBase(ns, agentName) + path
	respBody, _, err := k.doAgentRequest(ctx, http.MethodGet, url, "", 5*time.Second)
	return respBody, err
}

// postToAgentWithResponse makes an HTTP POST to an agent via its per-agent K8s Service
// and returns the response body, retrying transient failures with backoff (see
// doAgentRequest).
func (k *K8sClient) postToAgentWithResponse(ctx context.Context, ns, agentName, path, body string) ([]byte, error) {
	if os.Getenv("LOCAL") == "true" {
		return nil, fmt.Errorf("LOCAL mode: skipping direct pod HTTP")
	}
	url := agentHTTPBase(ns, agentName) + path
	respBody, _, err := k.doAgentRequest(ctx, http.MethodPost, url, body, 3*time.Second)
	return respBody, err
}

// ApplyAgentConfig sends a config update to the agent's FastAPI /config endpoint,
// falling back to an in-pod curl if the direct HTTP call fails.
func (k *K8sClient) ApplyAgentConfig(ctx context.Context, ns, podName, agentName string, configPayload string) error {
	err := k.postToAgent(ctx, ns, agentName, "/config", configPayload)
	if err != nil {
		// Fallback: kubectl exec curl inside the pod (uses $AGENT_PORT)
		_, execErr := k.execInPodCurl(ctx, ns, podName, agentName, "POST", "/config", configPayload)
		return execErr
	}
	return nil
}

// ApplyAgentConfigGetContextWindow sends a config update and returns the context_window from the response.
// Returns 0 if the agent is unreachable or the field is absent.
func (k *K8sClient) ApplyAgentConfigGetContextWindow(ctx context.Context, ns, podName, agentName string, configPayload string) int64 {
	respBody, err := k.postToAgentWithResponse(ctx, ns, agentName, "/config", configPayload)
	if err != nil {
		// Fallback via exec — capture stdout to still get context_window
		out, execErr := k.execInPodCurl(ctx, ns, podName, agentName, "POST", "/config", configPayload)
		if execErr != nil {
			return 0
		}
		respBody = out
	}
	var result struct {
		ContextWindow int64 `json:"context_window"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0
	}
	return result.ContextWindow
}

// PatchAgentSecretsList updates the secrets list on an agent's spec.
func (k *K8sClient) PatchAgentSecretsList(ctx context.Context, ns, agentName string, secrets []string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	original := agent.DeepCopy()
	agent.Spec.Secrets = secrets
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// PatchAgentMemoriesList updates the memories list on an agent's spec.
func (k *K8sClient) PatchAgentMemoriesList(ctx context.Context, ns, agentName string, memories []string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	original := agent.DeepCopy()
	agent.Spec.Memories = memories
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// PatchAgentSkillsList updates the skills list on an agent's spec.
func (k *K8sClient) PatchAgentSkillsList(ctx context.Context, ns, agentName string, skills []string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	original := agent.DeepCopy()
	agent.Spec.Skills = skills
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// PatchAgentToolPolicy updates an agent's tool allow/deny lists. A nil pointer
// means "leave unchanged"; a non-nil empty slice clears that list.
func (k *K8sClient) PatchAgentToolPolicy(ctx context.Context, ns, agentName string, allowed, disallowed *[]string) error {
	if allowed == nil && disallowed == nil {
		return nil
	}
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	original := agent.DeepCopy()
	if allowed != nil {
		agent.Spec.AllowedTools = *allowed
	}
	if disallowed != nil {
		agent.Spec.DisallowedTools = *disallowed
	}
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// --- Memory CRUD ---

func (k *K8sClient) CreateMemory(ctx context.Context, ns, name, content, description string) (*komputerv1alpha1.KomputerMemory, error) {
	memory := &komputerv1alpha1.KomputerMemory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"komputer.ai/memory-name": name,
			},
		},
		Spec: komputerv1alpha1.KomputerMemorySpec{
			Content:     content,
			Description: description,
		},
	}
	if err := k.client.Create(ctx, memory); err != nil {
		return nil, err
	}
	return memory, nil
}

func (k *K8sClient) GetMemory(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerMemory, error) {
	memory := &komputerv1alpha1.KomputerMemory{}
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, memory)
	if err != nil {
		return nil, err
	}
	return memory, nil
}

func (k *K8sClient) ListMemories(ctx context.Context, ns string) ([]komputerv1alpha1.KomputerMemory, error) {
	list := &komputerv1alpha1.KomputerMemoryList{}
	var opts []client.ListOption
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}
	if err := k.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *K8sClient) DeleteMemory(ctx context.Context, ns, name string) error {
	memory := &komputerv1alpha1.KomputerMemory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	return k.client.Delete(ctx, memory)
}

func (k *K8sClient) PatchMemory(ctx context.Context, ns, name string, content, description *string) error {
	memory := &komputerv1alpha1.KomputerMemory{}
	key := types.NamespacedName{Name: name, Namespace: ns}
	if err := k.client.Get(ctx, key, memory); err != nil {
		return fmt.Errorf("failed to get memory %s: %w", name, err)
	}
	original := memory.DeepCopy()
	changed := false
	if content != nil && *content != memory.Spec.Content {
		memory.Spec.Content = *content
		changed = true
	}
	if description != nil && *description != memory.Spec.Description {
		memory.Spec.Description = *description
		changed = true
	}
	if !changed {
		return nil
	}
	return k.client.Patch(ctx, memory, client.MergeFrom(original))
}

// ResolveMemoryContent fetches all referenced memories and returns concatenated content.
// References can be "name" (same namespace) or "namespace/name" (cross-namespace).
func (k *K8sClient) ResolveMemoryContent(ctx context.Context, agentNs string, memoryRefs []string) (string, error) {
	if len(memoryRefs) == 0 {
		return "", nil
	}
	var sections []string
	for _, ref := range memoryRefs {
		ns := agentNs
		name := ref
		if parts := strings.SplitN(ref, "/", 2); len(parts) == 2 {
			ns = parts[0]
			name = parts[1]
		}
		memory, err := k.GetMemory(ctx, ns, name)
		if err != nil {
			continue // skip missing memories
		}
		sections = append(sections, fmt.Sprintf("### %s\n%s", name, memory.Spec.Content))
	}
	if len(sections) == 0 {
		return "", nil
	}
	return "\n## Memory / Knowledge\nThe following knowledge has been provided. Use it as context when relevant.\n\n" + strings.Join(sections, "\n\n"), nil
}

// --- Skill CRUD ---

func (k *K8sClient) CreateSkill(ctx context.Context, ns, name, description, content string) (*komputerv1alpha1.KomputerSkill, error) {
	skill := &komputerv1alpha1.KomputerSkill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"komputer.ai/skill-name": name,
			},
		},
		Spec: komputerv1alpha1.KomputerSkillSpec{
			Description: description,
			Content:     content,
		},
	}
	if err := k.client.Create(ctx, skill); err != nil {
		return nil, err
	}
	return skill, nil
}

func (k *K8sClient) GetSkill(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerSkill, error) {
	skill := &komputerv1alpha1.KomputerSkill{}
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, skill)
	if err != nil {
		return nil, err
	}
	return skill, nil
}

func (k *K8sClient) ListSkills(ctx context.Context, ns string) ([]komputerv1alpha1.KomputerSkill, error) {
	list := &komputerv1alpha1.KomputerSkillList{}
	var opts []client.ListOption
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}
	if err := k.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *K8sClient) ListDefaultSkillNames(ctx context.Context) ([]string, error) {
	list := &komputerv1alpha1.KomputerSkillList{}
	if err := k.client.List(ctx, list, client.MatchingLabels{"komputer.ai/default": "true"}); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list.Items))
	for _, s := range list.Items {
		names = append(names, s.Name)
	}
	return names, nil
}

func (k *K8sClient) DeleteSkill(ctx context.Context, ns, name string) error {
	skill := &komputerv1alpha1.KomputerSkill{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	return k.client.Delete(ctx, skill)
}

func (k *K8sClient) PatchSkill(ctx context.Context, ns, name string, description, content *string) error {
	skill := &komputerv1alpha1.KomputerSkill{}
	key := types.NamespacedName{Name: name, Namespace: ns}
	if err := k.client.Get(ctx, key, skill); err != nil {
		return fmt.Errorf("failed to get skill %s: %w", name, err)
	}
	original := skill.DeepCopy()
	changed := false
	if description != nil && *description != skill.Spec.Description {
		skill.Spec.Description = *description
		changed = true
	}
	if content != nil && *content != skill.Spec.Content {
		skill.Spec.Content = *content
		changed = true
	}
	if !changed {
		return nil
	}
	return k.client.Patch(ctx, skill, client.MergeFrom(original))
}

// --- Connector CRUD ---

func (k *K8sClient) CreateConnector(ctx context.Context, ns, name, service, displayName, url, connType, authType, headerName string, authSecretName, authSecretKey *string) (*komputerv1alpha1.KomputerConnector, error) {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"komputer.ai/service": service,
			},
		},
		Spec: komputerv1alpha1.KomputerConnectorSpec{
			Type:        connType,
			AuthType:    authType,
			HeaderName:  headerName,
			Service:     service,
			DisplayName: displayName,
			URL:         url,
		},
	}
	if authSecretName != nil && authSecretKey != nil {
		conn.Spec.AuthSecretKeyRef = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: *authSecretName},
			Key:                  *authSecretKey,
		}
	}
	if err := k.client.Create(ctx, conn); err != nil {
		return nil, err
	}
	// Set owner reference on the auth secret so it's garbage-collected when the connector is deleted.
	if conn.Spec.AuthSecretKeyRef != nil {
		secret := &corev1.Secret{}
		if err := k.client.Get(ctx, types.NamespacedName{Name: conn.Spec.AuthSecretKeyRef.Name, Namespace: ns}, secret); err == nil {
			ref := metav1.OwnerReference{
				APIVersion: "komputer.komputer.ai/v1alpha1",
				Kind:       "KomputerConnector",
				Name:       conn.Name,
				UID:        conn.UID,
			}
			secret.OwnerReferences = append(secret.OwnerReferences, ref)
			k.client.Update(ctx, secret)
		}
	}
	return conn, nil
}

func (k *K8sClient) GetConnector(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerConnector, error) {
	conn := &komputerv1alpha1.KomputerConnector{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, conn); err != nil {
		return nil, err
	}
	return conn, nil
}

func (k *K8sClient) ListConnectors(ctx context.Context, ns string) ([]komputerv1alpha1.KomputerConnector, error) {
	list := &komputerv1alpha1.KomputerConnectorList{}
	var opts []client.ListOption
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
	}
	if err := k.client.List(ctx, list, opts...); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *K8sClient) DeleteConnector(ctx context.Context, ns, name string) error {
	conn := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
	}
	return k.client.Delete(ctx, conn)
}

// SetConnectorAuthSecret updates a connector CR to reference the given secret key
// and, when authType is non-empty, switches the connector to that auth type.
func (k *K8sClient) SetConnectorAuthSecret(ctx context.Context, ns, name, authType, secretName, secretKey string) (*komputerv1alpha1.KomputerConnector, error) {
	conn, err := k.GetConnector(ctx, ns, name)
	if err != nil {
		return nil, err
	}
	if authType != "" {
		conn.Spec.AuthType = authType
	}
	conn.Spec.AuthSecretKeyRef = &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
		Key:                  secretKey,
	}
	if err := k.client.Update(ctx, conn); err != nil {
		return nil, err
	}
	return conn, nil
}

// SetConnectorDisabled toggles a connector's Spec.Disabled flag. Any caller may
// re-enable a connector — there is no additional authorization check here.
func (k *K8sClient) SetConnectorDisabled(ctx context.Context, ns, name string, disabled bool) (*komputerv1alpha1.KomputerConnector, error) {
	conn, err := k.GetConnector(ctx, ns, name)
	if err != nil {
		return nil, err
	}
	conn.Spec.Disabled = disabled
	if err := k.client.Update(ctx, conn); err != nil {
		return nil, err
	}
	return conn, nil
}

// UpdateSecretKey sets a single key in an existing K8s Secret, leaving other keys untouched.
func (k *K8sClient) UpdateSecretKey(ctx context.Context, ns, name, key, value string) error {
	secret := &corev1.Secret{}
	if err := k.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, secret); err != nil {
		return fmt.Errorf("secret not found: %w", err)
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	secret.Data[key] = []byte(value)
	if err := k.client.Update(ctx, secret); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}
	return nil
}

// SetSecretOwnerRef sets an owner reference on a secret so it is garbage-collected with the connector.
func (k *K8sClient) SetSecretOwnerRef(ctx context.Context, ns, secretName, ownerName, ownerUID string) {
	secret := &corev1.Secret{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, secret); err != nil {
		return
	}
	ref := metav1.OwnerReference{
		APIVersion: "komputer.komputer.ai/v1alpha1",
		Kind:       "KomputerConnector",
		Name:       ownerName,
		UID:        types.UID(ownerUID),
	}
	// Avoid duplicate owner refs.
	for _, existing := range secret.OwnerReferences {
		if existing.Name == ownerName {
			return
		}
	}
	secret.OwnerReferences = append(secret.OwnerReferences, ref)
	k.client.Update(ctx, secret)
}

// GetOAuthTokenData reads a secret key and parses it as a JSON map.
func (k *K8sClient) GetOAuthTokenData(ctx context.Context, ns, secretName, key string) (map[string]interface{}, error) {
	raw, err := k.GetSecretValue(ctx, ns, secretName, key)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// UpdateOAuthTokenData marshals a token map and writes it to a secret key.
func (k *K8sClient) UpdateOAuthTokenData(ctx context.Context, ns, secretName, key string, data map[string]interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, updateErr := k.UpdateManagedSecret(ctx, ns, secretName, map[string]string{key: string(raw)})
	return updateErr
}

func (k *K8sClient) PatchAgentConnectorsList(ctx context.Context, ns, agentName string, connectors []string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	original := agent.DeepCopy()
	agent.Spec.Connectors = connectors
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// ValidateConnectorsAttachable rejects connectors in newRefs that are disabled, unless
// they are already present in existingRefs — attaching a disabled connector is blocked,
// but a connector that was attached before being disabled must not break a patch that
// merely re-sends the existing list or edits an unrelated field. Connector refs may be
// "name" (resolved in defaultNs) or "namespace/name", matching ResolveConnectorMCPConfigs.
// A missing connector is left for existing resolution logic to skip elsewhere; it is not
// treated as an attach error here.
func (k *K8sClient) ValidateConnectorsAttachable(ctx context.Context, defaultNs string, newRefs, existingRefs []string) error {
	existing := make(map[string]bool, len(existingRefs))
	for _, ref := range existingRefs {
		existing[ref] = true
	}
	for _, ref := range newRefs {
		if existing[ref] {
			continue
		}
		connNs := defaultNs
		connName := ref
		if parts := strings.SplitN(ref, "/", 2); len(parts) == 2 {
			connNs = parts[0]
			connName = parts[1]
		}
		conn, err := k.GetConnector(ctx, connNs, connName)
		if err != nil {
			continue
		}
		if conn.Spec.Disabled {
			return fmt.Errorf("connector %q is disabled", connName)
		}
	}
	return nil
}

// ResolveConnectorMCPConfigs resolves connector names to MCP server configs with auth headers.
// Returns {"connName": {"type": "http", "url": "...", "headers": {"Authorization": "Bearer ..."}}}
func (k *K8sClient) ResolveConnectorMCPConfigs(ctx context.Context, agentNs string, connectorNames []string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, connRef := range connectorNames {
		connNs := agentNs
		connName := connRef
		if parts := strings.SplitN(connRef, "/", 2); len(parts) == 2 {
			connNs = parts[0]
			connName = parts[1]
		}
		conn := &komputerv1alpha1.KomputerConnector{}
		if err := k.client.Get(ctx, types.NamespacedName{Name: connName, Namespace: connNs}, conn); err != nil {
			continue
		}
		if conn.Spec.Disabled {
			Logger.Infow("skipping disabled connector", "namespace", connNs, "name", connName)
			continue
		}
		entry := map[string]interface{}{"type": "http", "url": conn.Spec.URL}
		if conn.Spec.AuthSecretKeyRef != nil {
			secret := &corev1.Secret{}
			if err := k.client.Get(ctx, types.NamespacedName{Name: conn.Spec.AuthSecretKeyRef.Name, Namespace: connNs}, secret); err == nil {
				if raw, ok := secret.Data[conn.Spec.AuthSecretKeyRef.Key]; ok {
					tokenStr := string(raw)
					// For OAuth connectors, the secret value is a JSON blob — extract access_token.
					if conn.Spec.AuthType == "oauth" {
						var oauthData struct {
							AccessToken string `json:"access_token"`
						}
						if jsonErr := json.Unmarshal(raw, &oauthData); jsonErr == nil && oauthData.AccessToken != "" {
							tokenStr = oauthData.AccessToken
						}
					}
					if conn.Spec.AuthType == "header" && conn.Spec.HeaderName != "" {
						entry["headers"] = map[string]string{conn.Spec.HeaderName: tokenStr}
					} else {
						entry["headers"] = map[string]string{"Authorization": "Bearer " + tokenStr}
					}
				}
			}
		}
		result[connName] = entry
	}
	return result
}

// ResolveSkillFiles fetches all referenced skills and returns a map of skill name to {"description": ..., "content": ...}.
// References can be "name" (same namespace) or "namespace/name" (cross-namespace).
func (k *K8sClient) ResolveSkillFiles(ctx context.Context, agentNs string, skillRefs []string) (map[string]map[string]string, error) {
	if len(skillRefs) == 0 {
		return nil, nil
	}
	result := make(map[string]map[string]string)
	for _, ref := range skillRefs {
		ns := agentNs
		name := ref
		if parts := strings.SplitN(ref, "/", 2); len(parts) == 2 {
			ns = parts[0]
			name = parts[1]
		}
		skill, err := k.GetSkill(ctx, ns, name)
		if err != nil {
			continue // skip missing skills
		}
		result[name] = map[string]string{
			"description": skill.Spec.Description,
			"content":     skill.Spec.Content,
		}
	}
	return result, nil
}

// PatchAgentTaskStatus patches only the task-related status fields on a KomputerAgent CR.
// PatchAgentLifecycle updates the lifecycle field on an agent's spec.
func (k *K8sClient) PatchAgentLifecycle(ctx context.Context, ns, agentName, lifecycle string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}
	if string(agent.Spec.Lifecycle) == lifecycle {
		return nil // no change needed
	}
	original := agent.DeepCopy()
	agent.Spec.Lifecycle = komputerv1alpha1.AgentLifecycle(lifecycle)
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

func fetchModelContextWindow(ctx context.Context, model string) (int64, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/v1/models/"+model, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var result struct {
		MaxInputTokens int64 `json:"max_input_tokens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.MaxInputTokens, nil
}

func (k *K8sClient) PatchAgentContextWindow(ctx context.Context, ns, agentName string, contextWindow int64) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: agentName, Namespace: ns}, agent); err != nil {
		return err
	}
	original := agent.DeepCopy()
	agent.Status.ModelContextWindow = contextWindow
	return k.client.Status().Patch(ctx, agent, client.MergeFrom(original))
}

// PatchAgentPhase sets Status.Phase (and optionally Message) on a KomputerAgent.
// Used by the API to drive squad-member sleep/wake transitions, since the agent
// controller skips squad-managed agents.
func (k *K8sClient) PatchAgentPhase(ctx context.Context, ns, agentName string, phase komputerv1alpha1.KomputerAgentPhase, message string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: agentName, Namespace: ns}, agent); err != nil {
		return err
	}
	original := agent.DeepCopy()
	agent.Status.Phase = phase
	if message != "" {
		agent.Status.Message = message
	}
	return k.client.Status().Patch(ctx, agent, client.MergeFrom(original))
}

// agentTaskInProgress reports whether a task status means the agent is mid-task.
// Compacting is a transient sub-state of InProgress, so it counts as busy.
// Mirrors taskInProgress in the operator's ttl.go.
func agentTaskInProgress(s komputerv1alpha1.AgentTaskStatus) bool {
	return s == komputerv1alpha1.AgentTaskInProgress || s == komputerv1alpha1.AgentTaskCompacting
}

// taskStatusStartsTask reports whether a status transition marks the start of a new
// task — that is, the agent was not mid-task and now is.
//
// Deliberately false for in-progress → in-progress. A steer keeps the agent in progress
// and must not re-stamp Status.TaskStartedAt, or a user could extend spec.taskTimeout
// indefinitely by steering and the hard cap would degrade into an idle bound.
func taskStatusStartsTask(prev, next komputerv1alpha1.AgentTaskStatus) bool {
	return !agentTaskInProgress(prev) && agentTaskInProgress(next)
}

// statusChangedBeyondActivity reports whether two statuses differ in anything other
// than LastActivityAt.
func statusChangedBeyondActivity(a, b *komputerv1alpha1.KomputerAgentStatus) bool {
	ac, bc := a.DeepCopy(), b.DeepCopy()
	ac.LastActivityAt, bc.LastActivityAt = nil, nil
	return !equality.Semantic.DeepEqual(ac, bc)
}

// SwapAgentTaskStatus sets status.taskStatus to `to` only while it is still
// `from`, under an optimistic lock, so it never overwrites a status the worker
// wrote in between. Returns whether it changed anything. Used by
// forwardWithLifecycleChange.
func (k *K8sClient) SwapAgentTaskStatus(ctx context.Context, ns, name string, from, to komputerv1alpha1.AgentTaskStatus) (bool, error) {
	swapped := false
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		agent := &komputerv1alpha1.KomputerAgent{}
		if err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, agent); err != nil {
			return err
		}
		if agent.Status.TaskStatus != from {
			return nil
		}
		original := agent.DeepCopy()
		agent.Status.TaskStatus = to
		if err := k.client.Status().Patch(ctx, agent, client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{})); err != nil {
			return err
		}
		swapped = true
		return nil
	})
	if swapped {
		k.progress.forget(ns, name)
	}
	return swapped, err
}

func (k *K8sClient) PatchAgentTaskStatus(ctx context.Context, ns, agentName, taskStatus, lastMessage, sessionID string, costUSD float64, totalTokens int64, contextWindow int64) error {
	return k.patchAgentTaskStatus(ctx, ns, agentName, taskStatus, lastMessage, sessionID, costUSD, totalTokens, contextWindow, nil)
}

// sessionSaveFailedPrefix starts the task message when a failed S3 save stops
// Archive or AutoDelete from destroying the agent's only copy of its session.
const sessionSaveFailedPrefix = "Couldn't save the session to S3: "

// patchAgentTaskStatus is PatchAgentTaskStatus plus the task-end event's S3
// session store result (store, nil when the event carried none). It records
// status.sessionStore, and when the save failed on an agent whose lifecycle
// would now destroy its last local copy (Archive deletes the PVC, AutoDelete the
// agent), it records the task as Error instead of Complete — both transitions
// only fire on Complete, so the session survives until the next task or a delete.
func (k *K8sClient) patchAgentTaskStatus(ctx context.Context, ns, agentName, taskStatus, lastMessage, sessionID string, costUSD float64, totalTokens int64, contextWindow int64, store *SessionStoreResult) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}

	original := agent.DeepCopy()
	if store != nil {
		// The worker's own clock, not the pod's: TaskStartedAt is stamped here too
		// (when task_started is processed), and the task-end event always comes
		// after it in the stream, so "saved after the task started" holds even
		// when the worker lags or the pod's clock is skewed.
		at := metav1.Now()
		ss := agent.Status.SessionStore.DeepCopy()
		if ss == nil {
			ss = &komputerv1alpha1.SessionStoreStatus{}
		}
		ss.LastAttemptAt = &at
		ss.OK = store.OK
		ss.Error = store.Error
		if store.OK {
			ss.LastSavedAt = &at
		}
		agent.Status.SessionStore = ss
		lc := agent.Spec.Lifecycle
		if !store.OK && taskStatus == string(komputerv1alpha1.AgentTaskComplete) &&
			(lc == komputerv1alpha1.AgentLifecycleArchive || lc == komputerv1alpha1.AgentLifecycleAutoDelete) {
			taskStatus = string(komputerv1alpha1.AgentTaskError)
			lastMessage = truncate(sessionSaveFailedPrefix+store.Error, 256)
		}
	}
	prevTaskStatus := agent.Status.TaskStatus
	agent.Status.TaskStatus = komputerv1alpha1.AgentTaskStatus(taskStatus)
	agent.Status.LastTaskMessage = lastMessage
	// Every event counts as activity — this is the clock the operator's sleepTTL
	// measures idleness against, so refreshing it here keeps a busy agent awake.
	now := metav1.Now()
	agent.Status.LastActivityAt = &now
	// The task clock, by contrast, is stamped once per task and never refreshed, so
	// spec.taskTimeout stays a hard cap that steering cannot extend.
	if taskStatusStartsTask(prevTaskStatus, agent.Status.TaskStatus) {
		agent.Status.TaskStartedAt = &now
	}
	if sessionID != "" {
		agent.Status.SessionID = sessionID
	}
	if costUSD > 0 {
		agent.Status.LastTaskCostUSD = fmt.Sprintf("%.4f", costUSD)
		var total float64
		if agent.Status.TotalCostUSD != "" {
			fmt.Sscanf(agent.Status.TotalCostUSD, "%f", &total)
		}
		total += costUSD
		agent.Status.TotalCostUSD = fmt.Sprintf("%.4f", total)
	}
	if totalTokens > 0 {
		agent.Status.TotalTokens = totalTokens
	}
	if contextWindow > 0 {
		agent.Status.ModelContextWindow = contextWindow
	}

	// LastActivityAt always moves, so compare everything else: an event that changes
	// nothing but the activity clock inside the coalescing window is not worth a write.
	if !statusChangedBeyondActivity(&original.Status, &agent.Status) &&
		original.Status.LastActivityAt != nil && now.Sub(original.Status.LastActivityAt.Time) < activityCoalesceWindow {
		return nil
	}

	if err := k.client.Status().Patch(ctx, agent, client.MergeFrom(original)); err != nil {
		return err
	}

	// A real task starting is the single point that clears the wake-idle
	// annotation (see komputerv1alpha1.AnnotationWakeIdle) — covering every way a
	// task can start (forward, wake-with-task, schedules, manager tools), since
	// they all funnel through this method. Deliberately NOT cleared on a steer
	// (InProgress -> InProgress/Compacting): taskStatusStartsTask is false there,
	// since the agent was already doing real work, not sitting idle-woken.
	if taskStatusStartsTask(prevTaskStatus, agent.Status.TaskStatus) {
		// A new task supersedes both an idle wake and a sleep request the operator
		// hasn't acted on yet — otherwise the agent would sleep after a task the user
		// started afterwards.
		_, wakeIdle := agent.Annotations[komputerv1alpha1.AnnotationWakeIdle]
		_, sleepRequested := agent.Annotations[komputerv1alpha1.AnnotationSleepRequested]
		if wakeIdle || sleepRequested {
			metaOriginal := agent.DeepCopy()
			delete(agent.Annotations, komputerv1alpha1.AnnotationWakeIdle)
			delete(agent.Annotations, komputerv1alpha1.AnnotationSleepRequested)
			if err := k.client.Patch(ctx, agent, client.MergeFrom(metaOriginal)); err != nil {
				return fmt.Errorf("failed to clear wake-idle/sleep-requested annotations: %w", err)
			}
		}
	}

	return nil
}
