package main

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// newFakeK8sWithStatusSubresource is like newFakeK8s but registers KomputerAgent's
// status subresource, which IdleWakeAgent needs since it writes via
// k.client.Status().Patch — without this the fake client's typed status tracking
// rejects the patch.
func newFakeK8sWithStatusSubresource(t *testing.T, objs ...client.Object) *K8sClient {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := komputerv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
		Build()
	return &K8sClient{client: c, defaultNamespace: "default"}
}

// TestIdleWakeAgent_SleepingAgent_StartsIdle verifies the core contract of an
// explicit idle wake: it clears the Sleeping phase and restarts the idle clock
// (so the operator creates a pod and SleepTTL gets a fresh window), but — unlike
// WakeAgent — it must NOT touch spec.Instructions and must NOT clear a Complete
// TaskStatus, since shouldStartIdle in the operator relies on TaskStatus==Complete
// to start the new pod idle instead of auto-running stale Instructions.
func TestIdleWakeAgent_SleepingAgent_StartsIdle(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "the last real task"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseSleeping,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	got, err := k8s.IdleWakeAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("IdleWakeAgent returned error: %v", err)
	}

	if got.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		t.Error("Phase still Sleeping, want cleared so the operator creates a pod")
	}
	if got.Spec.Instructions != "the last real task" {
		t.Errorf("Spec.Instructions = %q, want unchanged %q (idle wake must not touch instructions)", got.Spec.Instructions, "the last real task")
	}
	if got.Status.TaskStatus != komputerv1alpha1.AgentTaskComplete {
		t.Errorf("TaskStatus = %q, want unchanged Complete", got.Status.TaskStatus)
	}
	if got.Status.SleepExpiresAt != nil {
		t.Error("SleepExpiresAt should be cleared")
	}
	if got.Status.LastActivityAt == nil {
		t.Error("LastActivityAt should be stamped (restarts the idle clock)")
	}
}

// TestIdleWakeAgent_EmptyTaskStatus_ForcedToComplete covers an agent that was put
// to sleep before ever completing a task (TaskStatus==""). Left alone, the
// operator's shouldStartIdle treats "" as "auto-run" — which would resurrect
// Instructions even though this is an explicit idle wake. IdleWakeAgent must force
// TaskStatus to Complete so the new pod actually starts idle.
func TestIdleWakeAgent_EmptyTaskStatus_ForcedToComplete(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "never ran"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseSleeping,
			TaskStatus: komputerv1alpha1.AgentTaskStatus(""),
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	got, err := k8s.IdleWakeAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("IdleWakeAgent returned error: %v", err)
	}
	if got.Status.TaskStatus != komputerv1alpha1.AgentTaskComplete {
		t.Errorf("TaskStatus = %q, want forced to Complete so the pod starts idle", got.Status.TaskStatus)
	}
}

// TestIdleWakeAgent_NotSleeping_Idempotent verifies that waking an agent that is
// not Sleeping (e.g. already Running) is a no-op, not an error — the handler maps
// this to "200 OK with the current agent" rather than a conflict.
func TestIdleWakeAgent_NotSleeping_Idempotent(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "doing stuff"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskInProgress,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	got, err := k8s.IdleWakeAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("IdleWakeAgent returned error: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseRunning {
		t.Errorf("Phase = %q, want unchanged Running", got.Status.Phase)
	}
	if got.Status.TaskStatus != komputerv1alpha1.AgentTaskInProgress {
		t.Errorf("TaskStatus = %q, want unchanged InProgress", got.Status.TaskStatus)
	}
}
