package main

import (
	"strings"
	"testing"
)

// TestMapEventToTaskStatus_BudgetExceededErrorMapsToError verifies that the
// agent's error event for a budget-exceeded task (see run_agent's handling of
// ResultMessage subtype "error_max_budget_usd") maps to TaskStatus Error with a
// message that clearly names the reason, just like any other "error" event.
func TestMapEventToTaskStatus_BudgetExceededErrorMapsToError(t *testing.T) {
	event := AgentEvent{
		Type: "error",
		Payload: map[string]interface{}{
			"error": "Task budget exceeded ($2.50)",
		},
	}
	status, msg := mapEventToTaskStatus(event)
	if status != "Error" {
		t.Errorf("status = %q, want Error", status)
	}
	if !strings.Contains(msg, "Task budget exceeded") {
		t.Errorf("message = %q, want it to explain the budget was exceeded", msg)
	}
}

// TestExtractTaskCompletionFields_ErrorStillReportsCost verifies that a task
// stopped mid-flight by error_max_budget_usd still reports what it spent before
// being cut off — the task's cost must not be lost just because it ended in
// error rather than completing normally.
func TestExtractTaskCompletionFields_ErrorStillReportsCost(t *testing.T) {
	event := AgentEvent{
		Type: "error",
		Payload: map[string]interface{}{
			"error":      "Task budget exceeded ($2.50)",
			"cost_usd":   2.50,
			"session_id": "sess-123",
		},
	}
	sessionID, costUSD, _, _ := extractTaskCompletionFields(event)
	if costUSD != 2.50 {
		t.Errorf("costUSD = %v, want 2.50", costUSD)
	}
	if sessionID != "sess-123" {
		t.Errorf("sessionID = %q, want sess-123", sessionID)
	}
}

// TestExtractTaskCompletionFields_OrdinaryErrorHasNoCost verifies a plain error
// event with no cost_usd in its payload (e.g. an unhandled exception, see
// main.py's catch-all) yields zero cost rather than some stale value — the zero
// is what tells PatchAgentTaskStatus not to touch totalCostUSD at all.
func TestExtractTaskCompletionFields_OrdinaryErrorHasNoCost(t *testing.T) {
	event := AgentEvent{
		Type:    "error",
		Payload: map[string]interface{}{"error": "something unrelated broke"},
	}
	_, costUSD, _, _ := extractTaskCompletionFields(event)
	if costUSD != 0 {
		t.Errorf("costUSD = %v, want 0", costUSD)
	}
}

// TestExtractTaskCompletionFields_OtherEventTypesYieldZero guards against a
// regression where tightening the type check to "task_completed" || "error"
// accidentally picked up fields from unrelated event types that happen to share
// key names.
func TestExtractTaskCompletionFields_OtherEventTypesYieldZero(t *testing.T) {
	event := AgentEvent{
		Type:    "text",
		Payload: map[string]interface{}{"cost_usd": 99.0, "session_id": "nope"},
	}
	sessionID, costUSD, totalTokens, contextWindow := extractTaskCompletionFields(event)
	if sessionID != "" || costUSD != 0 || totalTokens != 0 || contextWindow != 0 {
		t.Errorf("got (%q, %v, %v, %v), want all zero for a non-completion event", sessionID, costUSD, totalTokens, contextWindow)
	}
}
