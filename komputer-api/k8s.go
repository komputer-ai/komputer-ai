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

	// Copy default template from defaultNamespace to new namespace
	srcTemplate := &komputerv1alpha1.KomputerAgentTemplate{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: "default", Namespace: k.defaultNamespace}, srcTemplate); err == nil {
		newTemplate := &komputerv1alpha1.KomputerAgentTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: ns,
			},
			Spec: *srcTemplate.Spec.DeepCopy(),
		}
		if createErr := k.client.Create(ctx, newTemplate); createErr != nil && !errors.IsAlreadyExists(createErr) {
			log.Printf("warning: failed to copy default template to namespace %s: %v", ns, createErr)
		}
	}

	// Copy secrets referenced by the template (e.g., anthropic-api-key)
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

func (k *K8sClient) CreateAgent(ctx context.Context, ns, name, instructions, model, templateRef, role string) (*komputerv1alpha1.KomputerAgent, error) {
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
			Role:         role,
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
