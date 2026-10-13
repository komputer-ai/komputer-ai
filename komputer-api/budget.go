package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// isBudgetExhausted reports whether totalCostUSD has reached or passed budgetUSD.
// Pure function — mirrors the operator's identically-named helper
// (komputer-operator/internal/controller/budget.go) since the two components can't
// share a Go package, but must agree on the exact same semantics.
//
// An unset or unparseable budgetUSD means "no cap" — a malformed value must not be
// able to block an agent that was never meant to have a lifetime cap in the first
// place. An unparseable totalCostUSD is treated as zero spent, for the same reason:
// it must never falsely report exhaustion. Equality counts as exhausted
// ("spent >= budget"), matching the ">=" semantics documented on
// AgentConfigSpec.BudgetUSD.
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

// budgetExhaustedError formats the 409 error body for a request refused because the
// agent's lifetime budget is exhausted.
func budgetExhaustedError(agentName, totalCostUSD, budgetUSD string) error {
	return fmt.Errorf("agent %q has exhausted its budget: spent $%s of $%s — raise budgetUSD to continue", agentName, totalCostUSD, budgetUSD)
}

// parseBudget validates a decimal USD string for budgetUSD / taskBudgetUSD: empty is
// allowed (unset), anything else must parse as a number strictly greater than zero.
// Returns the value unchanged (CRDs store it as the string the caller supplied, not a
// re-formatted float) so a round-trip through the API doesn't rewrite "10" to
// "10.000000".
func parseBudget(field, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", fmt.Errorf("invalid %s: %q is not a number", field, value)
	}
	if f <= 0 {
		return "", fmt.Errorf("invalid %s: must be a positive number, got %q", field, value)
	}
	return value, nil
}

// parseBudgetUpdate builds a tri-state budget patch from an optional string, mirroring
// parseTTLUpdate. A nil pointer leaves the field alone; an explicit "" clears it.
func parseBudgetUpdate(field string, value *string) (StringUpdate, error) {
	if value == nil {
		return StringUpdate{}, nil
	}
	v, err := parseBudget(field, *value)
	if err != nil {
		return StringUpdate{}, err
	}
	return StringUpdate{Set: true, Value: v}, nil
}

// checkBudgetExhausted returns budgetExhaustedError when agent's EFFECTIVE
// lifetime budget (its own, else its template's default) has been reached or
// passed by status.totalCostUSD, else nil. Callers check this exactly once,
// right before starting a NEW task — never before a steer onto an already
// in-progress task (which is always allowed to finish) and never on the
// idle-wake endpoint (which carries no task).
//
// Takes context.Context rather than *gin.Context so the schedule-trigger path
// (triggerScheduleNow, which has no gin.Context — it's also called by the MCP
// server) can use it too; refuseIfBudgetExhausted adapts it for gin handlers.
func checkBudgetExhausted(ctx context.Context, k8s *K8sClient, agent *komputerv1alpha1.KomputerAgent) error {
	budget := k8s.ResolveEffectiveBudgetUSD(ctx, agent.Namespace, agent.Spec.AgentConfigSpec)
	if !isBudgetExhausted(agent.Status.TotalCostUSD, budget) {
		return nil
	}
	return budgetExhaustedError(agent.Name, agent.Status.TotalCostUSD, budget)
}

// refuseIfBudgetExhausted writes a 409 response and returns true when
// checkBudgetExhausted reports the agent's lifetime budget exhausted.
func refuseIfBudgetExhausted(c *gin.Context, k8s *K8sClient, agent *komputerv1alpha1.KomputerAgent) bool {
	err := checkBudgetExhausted(c.Request.Context(), k8s, agent)
	if err == nil {
		return false
	}
	c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	return true
}
