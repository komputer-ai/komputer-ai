package main

import (
	"context"
	"net/http"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestTriggerScheduleNow_Compacting_TreatedAsBusy is the API-side twin of the
// schedule controller's bug 2: triggerScheduleNow's busy check also only
// tested TaskStatus == InProgress, missing the Compacting sub-state. A manual
// "trigger now" on a compacting agent would patch spec.Instructions (the new
// instructions-before-forward logic right below this check) and attempt to
// forward the scheduled prompt as a steer onto a task still actually running.
func TestTriggerScheduleNow_Compacting_TreatedAsBusy(t *testing.T) {
	t.Setenv("LOCAL", "true") // skip real network; forward attempt (if it happens) fails fast

	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Schedule:     "0 0 * * *",
			Instructions: "scheduled prompt",
			AgentName:    "my-agent",
		},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "original task, still running"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			PodName:    "my-agent-pod",
			TaskStatus: komputerv1alpha1.AgentTaskCompacting,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, schedule, agent)

	_, status, err := triggerScheduleNow(context.Background(), k8s, "default", "sched1")

	var gotAgent komputerv1alpha1.KomputerAgent
	if getErr := k8s.client.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent); getErr != nil {
		t.Fatalf("get agent: %v", getErr)
	}
	if gotAgent.Spec.Instructions != "original task, still running" {
		t.Errorf("Spec.Instructions = %q, want unchanged (a busy/compacting agent must not have its instructions overwritten)", gotAgent.Spec.Instructions)
	}
	if status != http.StatusConflict {
		t.Errorf("status = %d, want 409 Conflict (agent busy); err=%v", status, err)
	}
}
