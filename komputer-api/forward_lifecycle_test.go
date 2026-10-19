package main

import (
	"context"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func idleFinishedAgent() *komputerv1alpha1.KomputerAgent {
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "next"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			PodName:    "a1-pod",
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
}

func getTestAgent(t *testing.T, k8s *K8sClient) *komputerv1alpha1.KomputerAgent {
	t.Helper()
	a := &komputerv1alpha1.KomputerAgent{}
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, a); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	return a
}

// By the time the task is forwarded, the lifecycle is already changed and the
// previous task's Complete status is gone, so the operator can't act on it.
func TestForwardWithLifecycleChange_ClearsStatusBeforeForward(t *testing.T) {
	existing := idleFinishedAgent()
	k8s := newFakeK8sWithStatusSubresource(t, existing)

	_, err := forwardWithLifecycleChange(context.Background(), k8s, existing, "Archive", func() (int64, error) {
		during := getTestAgent(t, k8s)
		if during.Status.TaskStatus != "" {
			t.Errorf("TaskStatus during forward = %q, want cleared", during.Status.TaskStatus)
		}
		if during.Spec.Lifecycle != komputerv1alpha1.AgentLifecycleArchive {
			t.Errorf("Lifecycle during forward = %q, want Archive", during.Spec.Lifecycle)
		}
		return 0, nil
	})
	if err != nil {
		t.Fatalf("forwardWithLifecycleChange: %v", err)
	}
	if got := getTestAgent(t, k8s).Spec.Lifecycle; got != komputerv1alpha1.AgentLifecycleArchive {
		t.Errorf("Lifecycle = %q, want Archive", got)
	}
}

// A failed forward must leave the agent exactly as it was — the new lifecycle
// applied to the old Complete status would archive (or delete) it at once.
func TestForwardWithLifecycleChange_FailedForwardRevertsLifecycle(t *testing.T) {
	existing := idleFinishedAgent()
	k8s := newFakeK8sWithStatusSubresource(t, existing)

	_, err := forwardWithLifecycleChange(context.Background(), k8s, existing, "Archive", func() (int64, error) {
		return 0, errors.New("pod unreachable")
	})
	if err == nil {
		t.Fatal("want the forward error back")
	}
	got := getTestAgent(t, k8s)
	if got.Spec.Lifecycle != "" {
		t.Errorf("Lifecycle = %q, want reverted to default", got.Spec.Lifecycle)
	}
	if got.Status.TaskStatus != komputerv1alpha1.AgentTaskComplete {
		t.Errorf("TaskStatus = %q, want restored to Complete", got.Status.TaskStatus)
	}
}

// If the agent took the task despite the forward error, the worker's newer
// status must win over the restore.
func TestForwardWithLifecycleChange_RestoreNeverOverwritesNewerStatus(t *testing.T) {
	existing := idleFinishedAgent()
	k8s := newFakeK8sWithStatusSubresource(t, existing)

	_, err := forwardWithLifecycleChange(context.Background(), k8s, existing, "Sleep", func() (int64, error) {
		if _, err := k8s.SwapAgentTaskStatus(context.Background(), "default", "a1", "", komputerv1alpha1.AgentTaskInProgress); err != nil {
			t.Fatalf("simulate worker write: %v", err)
		}
		return 0, errors.New("response timed out")
	})
	if err == nil {
		t.Fatal("want the forward error back")
	}
	if got := getTestAgent(t, k8s).Status.TaskStatus; got != komputerv1alpha1.AgentTaskInProgress {
		t.Errorf("TaskStatus = %q, want the worker's InProgress kept", got)
	}
}
