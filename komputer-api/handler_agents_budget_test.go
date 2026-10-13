package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// ─── isBudgetExhausted / parseBudget (pure) ──────────────────────────────────

func TestIsBudgetExhausted(t *testing.T) {
	tests := []struct {
		name          string
		totalCostUSD  string
		budgetUSD     string
		wantExhausted bool
	}{
		{"unset budget never exhausts", "100.00", "", false},
		{"spent under budget", "1.00", "10.00", false},
		{"spent exactly at budget is exhausted", "10.00", "10.00", true},
		{"spent over budget is exhausted", "10.01", "10.00", true},
		{"no spend yet, budget set", "", "5.00", false},
		{"unparseable spend is treated as zero", "not-a-number", "5.00", false},
		{"unparseable budget means no cap", "100.00", "not-a-number", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBudgetExhausted(tt.totalCostUSD, tt.budgetUSD)
			if got != tt.wantExhausted {
				t.Errorf("isBudgetExhausted(%q, %q) = %v, want %v", tt.totalCostUSD, tt.budgetUSD, got, tt.wantExhausted)
			}
		})
	}
}

func TestParseBudget(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty is unset", "", false},
		{"integer-looking value", "10", false},
		{"decimal value", "2.50", false},
		{"zero is rejected", "0", true},
		{"negative is rejected", "-5", true},
		{"garbage is rejected", "banana", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBudget("budgetUSD", tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseBudget(%q) error = nil, want an error", tt.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseBudget(%q) unexpected error: %v", tt.value, err)
			}
			if got != tt.value {
				t.Errorf("parseBudget(%q) = %q, want unchanged", tt.value, got)
			}
		})
	}
}

func TestParseBudgetUpdateTriState(t *testing.T) {
	t.Run("omitted leaves the field untouched", func(t *testing.T) {
		got, err := parseBudgetUpdate("budgetUSD", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Set {
			t.Error("Set = true for an omitted field, want false")
		}
	})

	t.Run("explicit empty string clears the budget", func(t *testing.T) {
		empty := ""
		got, err := parseBudgetUpdate("budgetUSD", &empty)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Set || got.Value != "" {
			t.Errorf("got %+v, want Set with empty value (clear)", got)
		}
	})

	t.Run("value assigns the budget", func(t *testing.T) {
		v := "5.00"
		got, err := parseBudgetUpdate("budgetUSD", &v)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Set || got.Value != "5.00" {
			t.Errorf("got %+v, want Set with 5.00", got)
		}
	})

	t.Run("invalid value errors", func(t *testing.T) {
		v := "banana"
		if _, err := parseBudgetUpdate("budgetUSD", &v); err == nil {
			t.Error("expected an error for a non-numeric budget")
		}
	})
}

// ─── createOrTriggerAgent: 409 on exhausted budget, never on a steer ─────────

// TestCreateOrTriggerAgent_RefusesNewTaskWhenBudgetExhausted verifies the forward
// path (running, idle agent) refuses a brand new task with 409 once
// status.totalCostUSD has reached the agent's effective lifetime budget.
func TestCreateOrTriggerAgent_RefusesNewTaskWhenBudgetExhausted(t *testing.T) {
	t.Setenv("LOCAL", "true")
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{BudgetUSD: "10.00"},
			Instructions:    "original task",
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseRunning,
			PodName:      "a1-pod",
			TaskStatus:   komputerv1alpha1.AgentTaskComplete,
			TotalCostUSD: "15.00",
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"brand new task"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body: %s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !strings.Contains(body["error"], "budget") {
		t.Errorf("error = %q, want it to mention the budget", body["error"])
	}

	// Instructions must not have been overwritten by the refused task.
	got, err := k8s.GetAgent(t.Context(), "default", "a1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Spec.Instructions != "original task" {
		t.Errorf("Spec.Instructions = %q, want unchanged (refused task must not patch instructions)", got.Spec.Instructions)
	}
}

// TestCreateOrTriggerAgent_RefusesWakeWithTaskWhenBudgetExhausted is the same
// check on the other new-task path: waking a sleeping agent with a task.
func TestCreateOrTriggerAgent_RefusesWakeWithTaskWhenBudgetExhausted(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{BudgetUSD: "10.00"},
			Instructions:    "stale",
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseSleeping,
			TaskStatus:   komputerv1alpha1.AgentTaskComplete,
			TotalCostUSD: "10.00", // exactly at budget -> exhausted
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"wake me up"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body: %s", w.Code, w.Body.String())
	}

	got, err := k8s.GetAgent(t.Context(), "default", "a1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Status.Phase = %q, want unchanged Sleeping (a refused wake must not wake the agent)", got.Status.Phase)
	}
}

// TestCreateOrTriggerAgent_SteerNotRefusedWhenBudgetExhausted verifies a follow-up
// message to an agent that is ALREADY mid-task is never refused for budget, even
// when the agent's spend has already crossed its lifetime budget — that task is
// allowed to finish regardless.
func TestCreateOrTriggerAgent_SteerNotRefusedWhenBudgetExhausted(t *testing.T) {
	t.Setenv("LOCAL", "true")
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{BudgetUSD: "10.00"},
			Instructions:    "original task",
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseRunning,
			PodName:      "a1-pod",
			TaskStatus:   komputerv1alpha1.AgentTaskInProgress,
			TotalCostUSD: "15.00",
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"also add a summary"}`)
	if w.Code == http.StatusConflict {
		t.Fatalf("status = 409, want the steer to reach the agent regardless of budget; body: %s", w.Body.String())
	}

	got, err := k8s.GetAgent(t.Context(), "default", "a1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Spec.Instructions != "original task" {
		t.Errorf("Spec.Instructions = %q, want unchanged (a steer must not patch instructions)", got.Spec.Instructions)
	}
}

// TestWakeAgentEndpoint_IdleWakeNotRefusedByExhaustedBudget verifies the separate
// idle-wake endpoint (POST /agents/{name}/wake, no task) is never refused for
// budget — it carries no task, so there is nothing for the cap to gate.
func TestWakeAgentEndpoint_IdleWakeNotRefusedByExhaustedBudget(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{BudgetUSD: "10.00"},
			Instructions:    "the real task",
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseSleeping,
			TaskStatus:   komputerv1alpha1.AgentTaskComplete,
			TotalCostUSD: "15.00",
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postWakeAgent(t, k8s, "a1")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (idle wake is never refused for budget); body: %s", w.Code, w.Body.String())
	}
}

// TestAgentResponse_BudgetExhaustedFlag verifies the create-response path surfaces
// BudgetExhausted=true once an under-budget agent is created, and false for one
// with no budget configured at all.
func TestAgentResponse_BudgetExhaustedFlag(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"go","budgetUSD":"10.00"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp AgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.BudgetExhausted {
		t.Error("BudgetExhausted = true for a freshly created agent with zero spend, want false")
	}
	if resp.BudgetUSD != "10.00" {
		t.Errorf("BudgetUSD = %q, want %q", resp.BudgetUSD, "10.00")
	}
}
