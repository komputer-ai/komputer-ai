package main

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestForwardNewTaskPatchesInstructionsBeforeForwarding verifies that sending a new
// task to a running, idle agent (TaskStatus=Complete, i.e. not mid-task) updates
// spec.Instructions BEFORE the task is forwarded to the pod. This matters because if
// the pod is later rescheduled (eviction, crash, node drain), the operator re-runs
// spec.Instructions (see shouldStartIdle in the operator) — it must be the new task,
// not whatever the agent was created with or last ran.
//
// The actual HTTP forward to the pod will fail in this test (no real pod/network),
// but that happens strictly after the instructions patch, so the CR state is already
// updated regardless of the forward's outcome.
func TestForwardNewTaskPatchesInstructionsBeforeForwarding(t *testing.T) {
	t.Setenv("LOCAL", "true") // skip real network; exercise exec fallback (fails fast: no such pod) so the test is deterministic and quick
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "original task"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			PodName:    "a1-pod",
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	k8s := newFakeK8s(t, agent)

	postCreateAgent(t, k8s, `{"name":"a1","instructions":"brand new task"}`)

	var got komputerv1alpha1.KomputerAgent
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Spec.Instructions != "brand new task" {
		t.Errorf("Spec.Instructions = %q, want %q (new task must patch instructions before forwarding)", got.Spec.Instructions, "brand new task")
	}
}

// TestForwardSteerDoesNotPatchInstructions verifies the opposite: sending a message to
// a BUSY agent (TaskStatus=InProgress) is a steer (follow-up), not a new task, and must
// NOT overwrite spec.Instructions. Overwriting would make an interrupted original task
// resume (after a pod restart) as just the steer text instead of the real task.
func TestForwardSteerDoesNotPatchInstructions(t *testing.T) {
	t.Setenv("LOCAL", "true")
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "original task"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			PodName:    "a1-pod",
			TaskStatus: komputerv1alpha1.AgentTaskInProgress,
		},
	}
	k8s := newFakeK8s(t, agent)

	postCreateAgent(t, k8s, `{"name":"a1","instructions":"also add a summary"}`)

	var got komputerv1alpha1.KomputerAgent
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Spec.Instructions != "original task" {
		t.Errorf("Spec.Instructions = %q, want unchanged %q (a steer must not patch instructions)", got.Spec.Instructions, "original task")
	}
}

// TestForwardSteerDoesNotPatchInstructionsCompacting is the Compacting sub-state of
// InProgress — also busy, must also not patch.
func TestForwardSteerDoesNotPatchInstructionsCompacting(t *testing.T) {
	t.Setenv("LOCAL", "true")
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "original task"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			PodName:    "a1-pod",
			TaskStatus: komputerv1alpha1.AgentTaskCompacting,
		},
	}
	k8s := newFakeK8s(t, agent)

	postCreateAgent(t, k8s, `{"name":"a1","instructions":"also add a summary"}`)

	var got komputerv1alpha1.KomputerAgent
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Spec.Instructions != "original task" {
		t.Errorf("Spec.Instructions = %q, want unchanged %q (compacting is still busy)", got.Spec.Instructions, "original task")
	}
}
