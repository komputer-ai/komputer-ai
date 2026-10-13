package main

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestSleepAgentNow_RunningIdleAgent_SetsSleepRequest verifies the core
// non-squad contract: SleepAgentNow does NOT touch Phase/PodName itself (those
// are operator-owned) — it records intent via the sleep-requested annotation
// and leaves spec.Lifecycle untouched, so the operator's reconcileStatus can
// perform the actual transition.
func TestSleepAgentNow_RunningIdleAgent_SetsSleepRequest(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "done already"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	got, err := k8s.SleepAgentNow(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("SleepAgentNow returned error: %v", err)
	}
	if got.Annotations[komputerv1alpha1.AnnotationSleepRequested] != "true" {
		t.Errorf("Annotations[%s] = %q, want %q", komputerv1alpha1.AnnotationSleepRequested, got.Annotations[komputerv1alpha1.AnnotationSleepRequested], "true")
	}
	if got.Spec.Lifecycle != komputerv1alpha1.AgentLifecycleDefault {
		t.Errorf("Spec.Lifecycle = %q, want unchanged (empty)", got.Spec.Lifecycle)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseRunning {
		t.Errorf("Phase = %q, want unchanged Running (operator applies the actual transition)", got.Status.Phase)
	}
}

// TestSleepAgentNow_AlreadySleeping_Idempotent mirrors IdleWakeAgent's
// idempotency: sleeping an already-sleeping agent is success, not an error, and
// must not set the sleep-requested annotation (nothing to request).
func TestSleepAgentNow_AlreadySleeping_Idempotent(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase: komputerv1alpha1.AgentPhaseSleeping,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	got, err := k8s.SleepAgentNow(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("SleepAgentNow returned error: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase = %q, want unchanged Sleeping", got.Status.Phase)
	}
	if _, ok := got.Annotations[komputerv1alpha1.AnnotationSleepRequested]; ok {
		t.Error("sleep-requested annotation should not be set on a no-op sleep")
	}
}

// TestSleepAgentNow_TaskInProgress_ReturnsErrAgentBusy verifies the busy guard
// covers both InProgress and Compacting (Compacting is a transient sub-state of
// InProgress) and does not mutate the agent at all.
func TestSleepAgentNow_TaskInProgress_ReturnsErrAgentBusy(t *testing.T) {
	for _, status := range []komputerv1alpha1.AgentTaskStatus{
		komputerv1alpha1.AgentTaskInProgress,
		komputerv1alpha1.AgentTaskCompacting,
	} {
		t.Run(string(status), func(t *testing.T) {
			agent := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
				Status: komputerv1alpha1.KomputerAgentStatus{
					Phase:      komputerv1alpha1.AgentPhaseRunning,
					TaskStatus: status,
				},
			}
			k8s := newFakeK8sWithStatusSubresource(t, agent)

			_, err := k8s.SleepAgentNow(context.Background(), "default", "a1")
			if err != ErrAgentBusy {
				t.Fatalf("err = %v, want ErrAgentBusy", err)
			}
		})
	}
}

// TestSleepAgentNow_EphemeralWorkspace_ReturnsErrAgentEphemeral verifies an
// agent whose own storage.ephemeral is true is rejected before any annotation
// is set.
func TestSleepAgentNow_EphemeralWorkspace_ReturnsErrAgentEphemeral(t *testing.T) {
	ephemeral := true
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Storage: &komputerv1alpha1.StorageSpec{Ephemeral: &ephemeral},
			},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	_, err := k8s.SleepAgentNow(context.Background(), "default", "a1")
	if err != ErrAgentEphemeral {
		t.Fatalf("err = %v, want ErrAgentEphemeral", err)
	}

	got, getErr := k8s.GetAgent(context.Background(), "default", "a1")
	if getErr != nil {
		t.Fatalf("GetAgent: %v", getErr)
	}
	if _, ok := got.Annotations[komputerv1alpha1.AnnotationSleepRequested]; ok {
		t.Error("sleep-requested annotation should not be set for an ephemeral agent")
	}
}

// TestSleepAgentNow_SquadMember_PatchesPhaseDirectly verifies the squad-member
// path: the agent controller skips squad-managed agents, so SleepAgentNow must
// drive Phase=Sleeping directly (via PatchAgentPhase) instead of relying on the
// sleep-requested annotation, which no reconciler would ever consume for a
// squad member.
func TestSleepAgentNow_SquadMember_PatchesPhaseDirectly(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "member-1",
			Namespace: "default",
			Annotations: map[string]string{
				komputerv1alpha1.AnnotationWakeIdle: "true",
			},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:          komputerv1alpha1.AgentPhaseRunning,
			TaskStatus:     komputerv1alpha1.AgentTaskComplete,
			Squad:          true,
			PodName:        "squad-pod",
			SleepExpiresAt: &metav1.Time{},
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	got, err := k8s.SleepAgentNow(context.Background(), "default", "member-1")
	if err != nil {
		t.Fatalf("SleepAgentNow returned error: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase = %q, want Sleeping (set directly, no operator involved for squad members)", got.Status.Phase)
	}
	if _, ok := got.Annotations[komputerv1alpha1.AnnotationSleepRequested]; ok {
		t.Error("sleep-requested annotation should not be used for a squad member (no reconciler consumes it)")
	}
	if _, ok := got.Annotations[komputerv1alpha1.AnnotationWakeIdle]; ok {
		t.Error("stale wake-idle annotation should be cleared")
	}
	if got.Status.SleepExpiresAt != nil {
		t.Error("SleepExpiresAt should be cleared")
	}
}

// TestSleepAgentNow_SquadMember_Busy_ReturnsErrAgentBusy verifies the busy
// guard applies to squad members too — a race where a task starts between the
// UI click and this call must not sleep an agent out from under a running task.
func TestSleepAgentNow_SquadMember_Busy_ReturnsErrAgentBusy(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "member-1", Namespace: "default"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskInProgress,
			Squad:      true,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	_, err := k8s.SleepAgentNow(context.Background(), "default", "member-1")
	if err != ErrAgentBusy {
		t.Fatalf("err = %v, want ErrAgentBusy", err)
	}
}
