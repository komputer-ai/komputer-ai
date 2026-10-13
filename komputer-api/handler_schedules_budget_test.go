package main

import (
	"context"
	"net/http"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestTriggerScheduleNow_RefusesWhenBudgetExhausted is the API-side twin of the
// schedule controller's budget check (komputerschedule_budget_test.go in the
// operator): a manual "trigger now" against an idle agent whose lifetime budget
// is already exhausted must be refused with 409, and must not touch
// spec.Instructions — the run never starts.
func TestTriggerScheduleNow_RefusesWhenBudgetExhausted(t *testing.T) {
	t.Setenv("LOCAL", "true")

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
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{BudgetUSD: "10.00"},
			Instructions:    "original task, now idle",
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseRunning,
			PodName:      "my-agent-pod",
			TaskStatus:   komputerv1alpha1.AgentTaskComplete,
			TotalCostUSD: "12.00",
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, schedule, agent)

	_, status, err := triggerScheduleNow(context.Background(), k8s, "default", "sched1")
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want 409 Conflict (budget exhausted); err=%v", status, err)
	}
	if err == nil {
		t.Fatal("expected an error describing the exhausted budget")
	}

	var gotAgent komputerv1alpha1.KomputerAgent
	if getErr := k8s.client.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent); getErr != nil {
		t.Fatalf("get agent: %v", getErr)
	}
	if gotAgent.Spec.Instructions != "original task, now idle" {
		t.Errorf("Spec.Instructions = %q, want unchanged (the run never started)", gotAgent.Spec.Instructions)
	}
}

// TestTriggerScheduleNow_UnderBudgetRunsNormally is the control: an agent under
// its lifetime budget must trigger exactly as it would with no budget set.
func TestTriggerScheduleNow_UnderBudgetRunsNormally(t *testing.T) {
	t.Setenv("LOCAL", "true")

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
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{BudgetUSD: "10.00"},
			Instructions:    "stale",
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseRunning,
			PodName:      "my-agent-pod",
			TaskStatus:   komputerv1alpha1.AgentTaskComplete,
			TotalCostUSD: "1.00",
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, schedule, agent)

	// The forward itself has no real pod/network to land on in this test (same
	// setup as TestTriggerScheduleNow_Compacting_TreatedAsBusy), so it fails after
	// the budget check — the assertion here is specifically that it is NOT refused
	// for budget (409), and that instructions were patched before that forward
	// attempt, proving the run proceeded past the budget gate.
	_, status, _ := triggerScheduleNow(context.Background(), k8s, "default", "sched1")
	if status == http.StatusConflict {
		t.Fatalf("status = 409, want the run to proceed under budget")
	}

	var gotAgent komputerv1alpha1.KomputerAgent
	if getErr := k8s.client.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent); getErr != nil {
		t.Fatalf("get agent: %v", getErr)
	}
	if gotAgent.Spec.Instructions != "scheduled prompt" {
		t.Errorf("Spec.Instructions = %q, want the scheduled prompt (run should have proceeded to forward the task)", gotAgent.Spec.Instructions)
	}
}
