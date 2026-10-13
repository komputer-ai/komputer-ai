package controller

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestBuildAgentEnvVarsCarriesTaskBudget verifies the pod env carries
// KOMPUTER_TASK_BUDGET_USD when the agent (or its template) sets a hard per-task
// cap, and omits it entirely when no cap is configured anywhere.
func TestBuildAgentEnvVarsCarriesTaskBudget(t *testing.T) {
	scheme := newTestScheme(t)
	config := &komputerv1alpha1.KomputerConfig{}

	t.Run("agent's own taskBudgetUSD is injected", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		agent := &komputerv1alpha1.KomputerAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
			Spec: komputerv1alpha1.KomputerAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{TaskBudgetUSD: "2.50"},
				Instructions:    "do stuff",
			},
		}
		envVars, err := buildAgentEnvVars(context.Background(), c, agent, config)
		if err != nil {
			t.Fatalf("buildAgentEnvVars returned error: %v", err)
		}
		val, ok := toolPolicyEnvValue(envVars, "KOMPUTER_TASK_BUDGET_USD")
		if !ok || val != "2.50" {
			t.Errorf("KOMPUTER_TASK_BUDGET_USD = %q, ok=%v, want 2.50/true", val, ok)
		}
	})

	t.Run("falls back to the template's default", func(t *testing.T) {
		tpl := &komputerv1alpha1.KomputerAgentTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
			Spec:       komputerv1alpha1.KomputerAgentTemplateSpec{TaskBudgetUSD: "1.00"},
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tpl).Build()
		agent := &komputerv1alpha1.KomputerAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
			Spec: komputerv1alpha1.KomputerAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{TemplateRef: "default"},
				Instructions:    "do stuff",
			},
		}
		envVars, err := buildAgentEnvVars(context.Background(), c, agent, config)
		if err != nil {
			t.Fatalf("buildAgentEnvVars returned error: %v", err)
		}
		val, ok := toolPolicyEnvValue(envVars, "KOMPUTER_TASK_BUDGET_USD")
		if !ok || val != "1.00" {
			t.Errorf("KOMPUTER_TASK_BUDGET_USD = %q, ok=%v, want 1.00/true", val, ok)
		}
	})

	t.Run("omitted entirely with no cap configured", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		agent := &komputerv1alpha1.KomputerAgent{
			ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
			Spec: komputerv1alpha1.KomputerAgentSpec{
				Instructions: "do stuff",
			},
		}
		envVars, err := buildAgentEnvVars(context.Background(), c, agent, config)
		if err != nil {
			t.Fatalf("buildAgentEnvVars returned error: %v", err)
		}
		if _, ok := toolPolicyEnvValue(envVars, "KOMPUTER_TASK_BUDGET_USD"); ok {
			t.Error("KOMPUTER_TASK_BUDGET_USD present, want omitted when no cap is set")
		}
	})
}
