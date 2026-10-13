package main

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// A sleep request the operator hasn't acted on yet (it waits out a task in
// progress, or the agent has no pod yet) must not outlive the user's later
// intent: otherwise the agent goes to sleep after a task the user started
// afterwards. The latest intent wins — a new task starting, or an idle wake,
// cancels a pending sleep request.

func getSleepTestAgent(t *testing.T, k8s *K8sClient) komputerv1alpha1.KomputerAgent {
	t.Helper()
	var got komputerv1alpha1.KomputerAgent
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	return got
}

func TestPatchAgentTaskStatus_NewTaskCancelsPendingSleepRequest(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "a1", Namespace: "default",
			Annotations: map[string]string{komputerv1alpha1.AnnotationSleepRequested: "true"},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	if err := k8s.PatchAgentTaskStatus(context.Background(), "default", "a1", "InProgress", "", "", 0, 0, 0); err != nil {
		t.Fatalf("PatchAgentTaskStatus: %v", err)
	}
	if _, ok := getSleepTestAgent(t, k8s).Annotations[komputerv1alpha1.AnnotationSleepRequested]; ok {
		t.Error("a pending sleep request survived a new task starting")
	}
}

func TestPatchAgentTaskStatus_SteerKeepsPendingSleepRequest(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "a1", Namespace: "default",
			Annotations: map[string]string{komputerv1alpha1.AnnotationSleepRequested: "true"},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskInProgress},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	// InProgress -> InProgress is the same task continuing, not a new one.
	if err := k8s.PatchAgentTaskStatus(context.Background(), "default", "a1", "InProgress", "", "", 0, 0, 0); err != nil {
		t.Fatalf("PatchAgentTaskStatus: %v", err)
	}
	if _, ok := getSleepTestAgent(t, k8s).Annotations[komputerv1alpha1.AnnotationSleepRequested]; !ok {
		t.Error("a steer must not cancel a pending sleep request")
	}
}

func TestIdleWakeAgent_NotSleeping_CancelsPendingSleepRequest(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "a1", Namespace: "default",
			Annotations: map[string]string{komputerv1alpha1.AnnotationSleepRequested: "true"},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	if _, err := k8s.IdleWakeAgent(context.Background(), "default", "a1"); err != nil {
		t.Fatalf("IdleWakeAgent: %v", err)
	}
	if _, ok := getSleepTestAgent(t, k8s).Annotations[komputerv1alpha1.AnnotationSleepRequested]; ok {
		t.Error("waking an agent with a pending sleep request must cancel the request")
	}
}
