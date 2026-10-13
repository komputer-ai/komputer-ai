package main

import "testing"

// A task the SDK stopped for exceeding taskBudgetUSD ends with an "error" event
// that carries its cost. The cost page must close the task on that event and
// attribute the cost, or the budget-stopped task shows $0 there while
// status.totalCostUSD includes it.
func TestBuildCostBreakdown_ErrorEventClosesTaskWithCost(t *testing.T) {
	events := []AgentEvent{
		{Type: "task_started", Timestamp: "2026-10-13T10:00:00Z", Payload: map[string]interface{}{"instructions": "big job"}},
		{Type: "error", Timestamp: "2026-10-13T10:05:00Z", Payload: map[string]interface{}{"error": "Task budget exceeded ($2.00)", "cost_usd": 2.0}},
		{Type: "task_started", Timestamp: "2026-10-13T11:00:00Z", Payload: map[string]interface{}{"instructions": "next"}},
		{Type: "task_completed", Timestamp: "2026-10-13T11:01:00Z", Payload: map[string]interface{}{"cost_usd": 0.5}},
	}
	got := buildCostBreakdown(events)
	if len(got.Tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(got.Tasks))
	}
	if got.Tasks[0].CostUSD != 2.0 || got.Tasks[0].CompletedAt == "" {
		t.Errorf("budget-stopped task: cost=%v completedAt=%q, want 2.0 and set", got.Tasks[0].CostUSD, got.Tasks[0].CompletedAt)
	}
	if got.Tasks[1].CostUSD != 0.5 {
		t.Errorf("next task cost = %v, want 0.5", got.Tasks[1].CostUSD)
	}
}
