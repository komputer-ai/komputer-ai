package controller

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// ─── isBudgetExhausted (pure) ─────────────────────────────────────────────────

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
		{"both unparseable", "nope", "nope", false},
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

// ─── resolveAgentLifetimeBudget / resolveAgentTaskBudget (agent over template) ─

func TestResolveAgentLifetimeBudget(t *testing.T) {
	ctx := context.Background()

	tplWithBudget := func(name, ns string) *komputerv1alpha1.KomputerAgentTemplate {
		return &komputerv1alpha1.KomputerAgentTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
				BudgetUSD:     "20.00",
				TaskBudgetUSD: "2.00",
			},
		}
	}

	t.Run("agent spec wins over the template", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "default"
		agent.Spec.BudgetUSD = "5.00"
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).
			WithObjects(tplWithBudget("default", "default")).Build()

		got := resolveAgentLifetimeBudget(ctx, c, agent)
		if got != "5.00" {
			t.Errorf("resolveAgentLifetimeBudget = %q, want the agent's 5.00", got)
		}
	})

	t.Run("template supplies the default when the agent omits it", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "default"
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).
			WithObjects(tplWithBudget("default", "default")).Build()

		got := resolveAgentLifetimeBudget(ctx, c, agent)
		if got != "20.00" {
			t.Errorf("resolveAgentLifetimeBudget = %q, want the template's 20.00", got)
		}
	})

	t.Run("a missing template resolves to no cap", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "gone"
		var c client.Client = fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

		got := resolveAgentLifetimeBudget(ctx, c, agent)
		if got != "" {
			t.Errorf("resolveAgentLifetimeBudget = %q, want empty (no cap)", got)
		}
	})
}

func TestResolveAgentTaskBudget(t *testing.T) {
	ctx := context.Background()

	tplWithBudget := func(name, ns string) *komputerv1alpha1.KomputerAgentTemplate {
		return &komputerv1alpha1.KomputerAgentTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
				TaskBudgetUSD: "2.00",
			},
		}
	}

	t.Run("agent spec wins over the template", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "default"
		agent.Spec.TaskBudgetUSD = "0.50"
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).
			WithObjects(tplWithBudget("default", "default")).Build()

		got := resolveAgentTaskBudget(ctx, c, agent)
		if got != "0.50" {
			t.Errorf("resolveAgentTaskBudget = %q, want the agent's 0.50", got)
		}
	})

	t.Run("template supplies the default when the agent omits it", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "default"
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).
			WithObjects(tplWithBudget("default", "default")).Build()

		got := resolveAgentTaskBudget(ctx, c, agent)
		if got != "2.00" {
			t.Errorf("resolveAgentTaskBudget = %q, want the template's 2.00", got)
		}
	})

	t.Run("a missing template resolves to no cap", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "gone"
		var c client.Client = fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

		got := resolveAgentTaskBudget(ctx, c, agent)
		if got != "" {
			t.Errorf("resolveAgentTaskBudget = %q, want empty (no cap)", got)
		}
	})
}
