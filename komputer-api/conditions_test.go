package main

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func sleptAgent() *komputerv1alpha1.KomputerAgent {
	a := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseSleeping,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	a.Status.SetCondition(komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonSleepTTL, "slept")
	return a
}

func wantCondition(t *testing.T, a *komputerv1alpha1.KomputerAgent, condType string, status metav1.ConditionStatus, reason string) {
	t.Helper()
	c := meta.FindStatusCondition(a.Status.Conditions, condType)
	if c == nil || c.Status != status || c.Reason != reason {
		t.Errorf("condition %s = %+v, want %s/%s", condType, c, status, reason)
	}
}

func TestIdleWakeResolvesAsleep(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, sleptAgent())
	got, err := k8s.IdleWakeAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatal(err)
	}
	wantCondition(t, got, komputerv1alpha1.ConditionAsleep, metav1.ConditionFalse, komputerv1alpha1.ReasonWoken)
}

func TestWakeWithTaskResolvesAsleep(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, sleptAgent())
	if err := k8s.WakeAgent(context.Background(), "default", "a1", "go", "", "", "", ""); err != nil {
		t.Fatal(err)
	}
	got, _ := k8s.GetAgent(context.Background(), "default", "a1")
	wantCondition(t, got, komputerv1alpha1.ConditionAsleep, metav1.ConditionFalse, komputerv1alpha1.ReasonWoken)
}

func TestSquadMemberSleepRecordsAsleep(t *testing.T) {
	a := sleptAgent()
	a.Status.Conditions = nil
	a.Status.Phase = komputerv1alpha1.AgentPhaseRunning
	a.Status.Squad = true
	k8s := newFakeK8sWithStatusSubresource(t, a)
	got, err := k8s.SleepAgentNow(context.Background(), "default", "a1")
	if err != nil {
		t.Fatal(err)
	}
	wantCondition(t, got, komputerv1alpha1.ConditionAsleep, metav1.ConditionTrue, komputerv1alpha1.ReasonSleepRequested)
}

func TestTaskStartResolvesTaskTimedOut(t *testing.T) {
	a := sleptAgent()
	a.Status.Phase = komputerv1alpha1.AgentPhaseRunning
	a.Status.SetCondition(komputerv1alpha1.ConditionTaskTimedOut, komputerv1alpha1.ReasonTaskTimeout, "cancelled")
	k8s := newFakeK8sWithStatusSubresource(t, a)
	if err := k8s.PatchAgentTaskStatus(context.Background(), "default", "a1", "InProgress", "", "", 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	got, _ := k8s.GetAgent(context.Background(), "default", "a1")
	wantCondition(t, got, komputerv1alpha1.ConditionTaskTimedOut, metav1.ConditionFalse, komputerv1alpha1.ReasonTaskStarted)
}

func TestAgentResponseCarriesConditions(t *testing.T) {
	var resp AgentResponse
	fillAgentTTL(&resp, sleptAgent())
	if len(resp.Conditions) != 1 {
		t.Fatalf("conditions = %+v, want 1", resp.Conditions)
	}
	c := resp.Conditions[0]
	if c.Type != "Asleep" || c.Status != "True" || c.Reason != "SleepTTL" || c.Message != "slept" || c.LastTransitionTime == "" {
		t.Errorf("condition = %+v", c)
	}
}
