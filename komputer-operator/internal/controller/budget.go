package controller

import (
	"context"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// isBudgetExhausted reports whether totalCostUSD has reached or passed budgetUSD.
// Pure function. Both are decimal USD strings as stored on the CR (status.totalCostUSD,
// spec.budgetUSD) — see AgentConfigSpec.BudgetUSD for why these are strings rather than
// floats.
//
// An unset or unparseable budgetUSD means "no cap" — a malformed value must not be able
// to block an agent that was never meant to have a lifetime cap in the first place. An
// unparseable totalCostUSD is treated as zero spent, for the same reason: it must never
// falsely report exhaustion. Equality counts as exhausted ("spent >= budget"), matching
// the ">=" semantics documented on BudgetUSD.
func isBudgetExhausted(totalCostUSD, budgetUSD string) bool {
	if budgetUSD == "" {
		return false
	}
	budget, err := strconv.ParseFloat(budgetUSD, 64)
	if err != nil {
		return false
	}
	spent, _ := strconv.ParseFloat(totalCostUSD, 64) // unparseable -> 0, treated as unspent
	return spent >= budget
}

// resolveAgentLifetimeBudget returns an agent's effective lifetime budget
// (budgetUSD): its own spec value if set, else its template's default. Mirrors
// resolveAgentTTLs's agent-over-template resolution, for callers — the schedule
// controller — that never build a merged template. A template that can't be read
// resolves to "" (no cap), matching resolveAgentTTLs's "no defaults" fallback.
func resolveAgentLifetimeBudget(ctx context.Context, c client.Client, agent *komputerv1alpha1.KomputerAgent) string {
	if agent.Spec.BudgetUSD != "" {
		return agent.Spec.BudgetUSD
	}
	templateRef := agent.Spec.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	spec, err := readTemplateSpec(ctx, c, templateRef, agent.Namespace)
	if err != nil {
		return ""
	}
	return spec.BudgetUSD
}

// resolveAgentTaskBudget returns an agent's effective per-task budget
// (taskBudgetUSD): its own spec value if set, else its template's default. Mirrors
// resolveAgentTTLs's agent-over-template resolution, for callers — the squad
// controller — that never build a merged template (the agent controller already has
// one via applyAgentOverrides and should read Spec.TaskBudgetUSD off it directly
// instead of calling this).
func resolveAgentTaskBudget(ctx context.Context, c client.Client, agent *komputerv1alpha1.KomputerAgent) string {
	if agent.Spec.TaskBudgetUSD != "" {
		return agent.Spec.TaskBudgetUSD
	}
	templateRef := agent.Spec.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	spec, err := readTemplateSpec(ctx, c, templateRef, agent.Namespace)
	if err != nil {
		return ""
	}
	return spec.TaskBudgetUSD
}
