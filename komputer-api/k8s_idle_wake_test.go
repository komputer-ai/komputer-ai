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
// explicit idle wake: it clears the Sleeping phase, restarts the idle clock
// (so the operator creates a pod and SleepTTL gets a fresh window), and sets
// the komputer.ai/wake-idle annotation — but, unlike WakeAgent, it must NOT
// touch spec.Instructions and must NOT touch status.TaskStatus at all. Forcing
// TaskStatus was the old (buggy) way to signal "start idle"; it's wrong because
// TaskStatus=Complete + Lifecycle=Sleep is also the operator's immediate
// "re-sleep this pod" rule, so forcing it would make the agent (the common
// case: slept because of lifecycle Sleep) re-sleep itself right back down on
// the very next reconcile. The annotation is what the operator now keys off.
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
		t.Errorf("TaskStatus = %q, want unchanged Complete (idle wake must not touch TaskStatus)", got.Status.TaskStatus)
	}
	if got.Annotations[komputerv1alpha1.AnnotationWakeIdle] != "true" {
		t.Errorf("Annotations[%s] = %q, want %q", komputerv1alpha1.AnnotationWakeIdle, got.Annotations[komputerv1alpha1.AnnotationWakeIdle], "true")
	}
	if got.Status.SleepExpiresAt != nil {
		t.Error("SleepExpiresAt should be cleared")
	}
	if got.Status.LastActivityAt == nil {
		t.Error("LastActivityAt should be stamped (restarts the idle clock)")
	}
}

// TestIdleWakeAgent_EmptyTaskStatus_LeftUntouched covers an agent that was put
// to sleep before ever completing a task (TaskStatus==""). The old design
// forced TaskStatus to Complete here so the operator's shouldStartIdle (which
// only checked TaskStatus) would start the pod idle; that's no longer needed
// since shouldStartIdle now also checks the wake-idle annotation directly, so
// TaskStatus must be left exactly as it was.
func TestIdleWakeAgent_EmptyTaskStatus_LeftUntouched(t *testing.T) {
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
	if got.Status.TaskStatus != komputerv1alpha1.AgentTaskStatus("") {
		t.Errorf("TaskStatus = %q, want left untouched (empty)", got.Status.TaskStatus)
	}
	if got.Annotations[komputerv1alpha1.AnnotationWakeIdle] != "true" {
		t.Errorf("Annotations[%s] = %q, want %q", komputerv1alpha1.AnnotationWakeIdle, got.Annotations[komputerv1alpha1.AnnotationWakeIdle], "true")
	}
}

// TestIdleWakeAgent_NotSleeping_Idempotent verifies that waking an agent that is
// not Sleeping (e.g. already Running) is a no-op, not an error — the handler maps
// this to "200 OK with the current agent" rather than a conflict. It must not
// set the wake-idle annotation either: the agent was never actually put through
// an idle wake.
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
	if got.Annotations[komputerv1alpha1.AnnotationWakeIdle] == "true" {
		t.Error("wake-idle annotation should not be set on a no-op wake")
	}
}
