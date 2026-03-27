package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sClient struct {
	client           client.Client
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

	return &K8sClient{client: c, defaultNamespace: defaultNamespace}, nil
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
	for key, value := range secrets {
		data["SECRET_"+strings.ToUpper(key)] = []byte(value)
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

func (k *K8sClient) CreateAgent(ctx context.Context, ns, name, instructions, model, templateRef, role string, secretNames []string, lifecycle string) (*komputerv1alpha1.KomputerAgent, error) {
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
			TemplateRef:  templateRef,
			Instructions: instructions,
			Model:        model,
			Role:      role,
			Secrets:   secretNames,
			Lifecycle: komputerv1alpha1.AgentLifecycle(lifecycle),
		},
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
	if err := k.client.List(ctx, list, client.InNamespace(ns)); err != nil {
		return nil, err
	}
	return list.Items, nil
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
func (k *K8sClient) ForwardTaskToAgent(ctx context.Context, ns, podName, podIP, instructions, model string) error {
	bodyMap := map[string]string{"instructions": instructions}
	if model != "" {
		bodyMap["model"] = model
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

	config, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	execReq := clientset.CoreV1().RESTClient().Post().
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

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", execReq.URL())
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

// PatchAgentTaskStatus patches only the task-related status fields on a KomputerAgent CR.
func (k *K8sClient) PatchAgentTaskStatus(ctx context.Context, ns, agentName, taskStatus, lastMessage, sessionID string) error {
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

	return k.client.Status().Patch(ctx, agent, client.MergeFrom(original))
}
