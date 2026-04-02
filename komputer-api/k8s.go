package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sClient struct {
	client           client.Client
	clientset        *kubernetes.Clientset
	restConfig       *rest.Config
	defaultNamespace string
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

// EnsureNamespace creates the namespace if it doesn't exist, and copies
// the default KomputerAgentTemplate and required secrets into it.
func (k *K8sClient) EnsureNamespace(ctx context.Context, ns string) error {
	namespace := &corev1.Namespace{}
	err := k.client.Get(ctx, types.NamespacedName{Name: ns}, namespace)
	if err == nil {
		return nil // already exists
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check namespace: %w", err)
	}

	// Create namespace
	namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
			Labels: map[string]string{
				"komputer.ai/managed": "true",
			},
		},
	}
	if err := k.client.Create(ctx, namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Copy secrets referenced by agent templates (e.g., anthropic-api-key).
	// Templates don't need copying — the operator falls back to the default namespace.
	for _, secretName := range []string{"anthropic-api-key", "redis-secret"} {
		srcSecret := &corev1.Secret{}
		if err := k.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: k.defaultNamespace}, srcSecret); err == nil {
			newSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: srcSecret.Data,
				Type: srcSecret.Type,
			}
			if createErr := k.client.Create(ctx, newSecret); createErr != nil && !errors.IsAlreadyExists(createErr) {
				log.Printf("warning: failed to copy secret %s to namespace %s: %v", secretName, ns, createErr)
			}
		}
	}

	return nil
}

// WakeAgent wakes a sleeping agent by patching its instructions and setting the lifecycle.
// If lifecycle is empty, the agent stays running after task (default). If "Sleep", it sleeps again.
func (k *K8sClient) WakeAgent(ctx context.Context, ns, name, instructions, model, lifecycle string) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: name, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return err
	}

	// Patch spec: update instructions and lifecycle
	original := agent.DeepCopy()
	agent.Spec.Instructions = instructions
	agent.Spec.Lifecycle = komputerv1alpha1.AgentLifecycle(lifecycle)
	if model != "" {
		agent.Spec.Model = model
	}
	if err := k.client.Patch(ctx, agent, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch spec: %w", err)
	}

	// Patch status: clear sleeping phase and task status so operator creates a pod
	if err := k.client.Get(ctx, key, agent); err != nil {
		return err
	}
	original2 := agent.DeepCopy()
	agent.Status.Phase = komputerv1alpha1.AgentPhasePending
	agent.Status.TaskStatus = ""
	agent.Status.LastTaskMessage = ""
	return k.client.Status().Patch(ctx, agent, client.MergeFrom(original2))
}

// CreateAgentSecrets creates a K8s Secret with agent-specific secrets.
// Keys are prefixed with SECRET_ (e.g. "GITHUB" becomes "SECRET_GITHUB").
// Returns the secret name.
func (k *K8sClient) CreateAgentSecrets(ctx context.Context, ns, agentName string, secrets map[string]string) (string, error) {
	secretName := agentName + "-secrets"
	data := make(map[string][]byte, len(secrets))
	sanitize := regexp.MustCompile(`[^A-Za-z0-9]`)
	for key, value := range secrets {
		safe := strings.ToUpper(sanitize.ReplaceAllString(key, "_"))
		data["SECRET_"+safe] = []byte(value)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
			Labels: map[string]string{
				"komputer.ai/agent-name": agentName,
			},
		},
		Data: data,
	}

	if err := k.client.Create(ctx, secret); err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing secret.
			existing := &corev1.Secret{}
			if getErr := k.client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ns}, existing); getErr != nil {
				return "", fmt.Errorf("failed to get existing secret: %w", getErr)
			}
			existing.Data = data
			if updateErr := k.client.Update(ctx, existing); updateErr != nil {
				return "", fmt.Errorf("failed to update secret: %w", updateErr)
			}
			return secretName, nil
		}
		return "", fmt.Errorf("failed to create secret: %w", err)
	}
	return secretName, nil
}

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

func (k *K8sClient) CreateAgent(ctx context.Context, ns, name, instructions, model, templateRef, role string, secretNames []string, memories []string, skills []string, lifecycle, officeManager string) (*komputerv1alpha1.KomputerAgent, error) {
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	if templateRef == "" {
		templateRef = "default"
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
			TemplateRef:   templateRef,
			Instructions:  instructions,
			Model:         model,
			Role:          role,
			Secrets:       secretNames,
			Memories:      memories,
			Skills:        skills,
			Lifecycle:     komputerv1alpha1.AgentLifecycle(lifecycle),
			OfficeManager: officeManager,
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

func (k *K8sClient) GetAgent(ctx context.Context, ns, name string) (*komputerv1alpha1.KomputerAgent, error) {
	agent := &komputerv1alpha1.KomputerAgent{}
	err := k.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, agent)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func (k *K8sClient) ListAgents(ctx context.Context, ns string) ([]komputerv1alpha1.KomputerAgent, error) {
	list := &komputerv1alpha1.KomputerAgentList{}
	var opts []client.ListOption
	if ns != "" {
		opts = append(opts, client.InNamespace(ns))
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

// CancelAgentTask sends a cancel request to the agent's FastAPI endpoint.
func (k *K8sClient) CancelAgentTask(ctx context.Context, ns, podName, podIP string) error {
	err := k.postToAgent(ctx, podIP, "/cancel", "")
	if err != nil {
		// Fallback: kubectl exec curl inside the pod
		return k.execInPod(ctx, ns, podName, "curl", "-s", "-X", "POST", "http://localhost:8000/cancel")
	}
	return nil
}

// ForwardTaskToAgent sends a task to an agent's FastAPI endpoint, falling back to kubectl exec.
func (k *K8sClient) ForwardTaskToAgent(ctx context.Context, ns, podName, podIP, instructions, model, systemPrompt string) error {
	bodyMap := map[string]string{"instructions": instructions}
	if model != "" {
		bodyMap["model"] = model
	}
	if systemPrompt != "" {
		bodyMap["system_prompt"] = systemPrompt
	}
	bodyJSON, _ := json.Marshal(bodyMap)

	err := k.postToAgent(ctx, podIP, "/task", string(bodyJSON))
	if err != nil {
		// Fallback: kubectl exec curl inside the pod
		return k.execInPod(ctx, ns, podName, "curl", "-s", "-X", "POST",
			"-H", "Content-Type: application/json",
			"-d", string(bodyJSON),
			"http://localhost:8000/task")
	}
	return nil
}

// postToAgent makes a direct HTTP POST to an agent pod. Returns error if unreachable.
func (k *K8sClient) postToAgent(ctx context.Context, podIP, path, body string) error {
	url := fmt.Sprintf("http://%s:8000%s", podIP, path)

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(timeoutCtx, http.MethodPost, url, reqBody)
	if err != nil {
		return err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("agent returned status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// execInPod runs a command inside a pod using the Kubernetes API (equivalent to kubectl exec).
func (k *K8sClient) execInPod(ctx context.Context, ns, podName string, command ...string) error {
	pod := &corev1.Pod{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: podName, Namespace: ns}, pod); err != nil {
		return fmt.Errorf("failed to get pod %s: %w", podName, err)
	}

	execReq := k.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, clientgoscheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(k.restConfig, "POST", execReq.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return fmt.Errorf("exec failed: %w (stderr: %s)", err, stderr.String())
	}

	return nil
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
		lifecycle := komputerv1alpha1.AgentLifecycle(req.Agent.Lifecycle)
		if lifecycle == "" {
			lifecycle = komputerv1alpha1.AgentLifecycleSleep
		}
		agentSpec := &komputerv1alpha1.ScheduleAgentSpec{
			Model:       req.Agent.Model,
			Lifecycle:   lifecycle,
			Role:        req.Agent.Role,
			TemplateRef: req.Agent.TemplateRef,
		}
		// Create K8s Secrets from key-value pairs and store secret names on agent spec
		if len(req.Agent.Secrets) > 0 {
			secretName, err := k.CreateAgentSecrets(ctx, ns, req.Name+"-agent", req.Agent.Secrets)
			if err != nil {
				return nil, fmt.Errorf("failed to create secrets: %w", err)
			}
			agentSpec.Secrets = []string{secretName}
		}
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
	schedule := &komputerv1alpha1.KomputerSchedule{}
	key := types.NamespacedName{Name: name, Namespace: ns}
	if err := k.client.Get(ctx, key, schedule); err != nil {
		return fmt.Errorf("failed to get schedule %s: %w", name, err)
	}
	if schedule.Spec.Schedule == cron {
		return nil
	}
	original := schedule.DeepCopy()
	schedule.Spec.Schedule = cron
	return k.client.Patch(ctx, schedule, client.MergeFrom(original))
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
func (k *K8sClient) PatchAgentSpec(ctx context.Context, ns, agentName string, model, lifecycle, instructions, templateRef *string) error {
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
	if !changed {
		return nil
	}
	return k.client.Patch(ctx, agent, client.MergeFrom(original))
}

// ApplyAgentConfig sends a config update to the agent's FastAPI /config endpoint.
func (k *K8sClient) ApplyAgentConfig(ctx context.Context, ns, podName, podIP string, configPayload string) error {
	err := k.postToAgent(ctx, podIP, "/config", configPayload)
	if err != nil {
		// Fallback: kubectl exec curl inside the pod
		return k.execInPod(ctx, ns, podName, "curl", "-s", "-X", "POST",
			"-H", "Content-Type: application/json",
			"-d", configPayload,
			"http://localhost:8000/config")
	}
	return nil
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

func (k *K8sClient) PatchAgentTaskStatus(ctx context.Context, ns, agentName, taskStatus, lastMessage, sessionID string, costUSD float64) error {
	agent := &komputerv1alpha1.KomputerAgent{}
	key := types.NamespacedName{Name: agentName, Namespace: ns}
	if err := k.client.Get(ctx, key, agent); err != nil {
		return fmt.Errorf("failed to get agent %s: %w", agentName, err)
	}

	original := agent.DeepCopy()
	agent.Status.TaskStatus = komputerv1alpha1.AgentTaskStatus(taskStatus)
	agent.Status.LastTaskMessage = lastMessage
	if sessionID != "" {
		agent.Status.SessionID = sessionID
	}
	if costUSD > 0 {
		agent.Status.LastTaskCostUSD = fmt.Sprintf("%.4f", costUSD)
		// Accumulate total cost.
		var total float64
		if agent.Status.TotalCostUSD != "" {
			fmt.Sscanf(agent.Status.TotalCostUSD, "%f", &total)
		}
		total += costUSD
		agent.Status.TotalCostUSD = fmt.Sprintf("%.4f", total)
	}

	return k.client.Status().Patch(ctx, agent, client.MergeFrom(original))
}
