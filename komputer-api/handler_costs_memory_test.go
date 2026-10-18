package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// syntheticHistory serves a fake komputer-history list of tasks×perTask events,
// generating each entry on demand so the test itself holds no full history.
func syntheticHistory(t *testing.T, tasks, perTask int, pageSize int64) func(start, stop int64) ([]string, error) {
	total := int64(tasks * perTask)
	filler := strings.Repeat("x", 1024) // ~1 KiB of tool output per event
	return func(start, stop int64) ([]string, error) {
		if stop-start+1 > pageSize {
			t.Fatalf("LRANGE %d..%d asks for more than a page (%d)", start, stop, pageSize)
		}
		var out []string
		for i := start; i <= stop && i < total; i++ {
			task, pos := int(i)/perTask, int(i)%perTask
			ev := AgentEvent{AgentName: "a", Timestamp: fmt.Sprintf("2026-10-17T00:00:%06dZ", i)}
			switch pos {
			case 0:
				ev.Type = "task_started"
				ev.Payload = map[string]interface{}{"instructions": fmt.Sprintf("run %d %s", task, filler)}
			case perTask - 1:
				ev.Type = "task_completed"
				ev.Payload = map[string]interface{}{"cost_usd": 0.01, "usage": map[string]interface{}{"input_tokens": 10.0}}
			default:
				ev.Type = "tool_result"
				ev.Payload = map[string]interface{}{"output": filler}
			}
			raw, _ := json.Marshal(ev)
			out = append(out, string(raw))
		}
		return out, nil
	}
}

func heapAlloc() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// The cost endpoint OOMKilled 256Mi API pods on agents with hundreds of
// scheduled runs: it loaded the whole history and kept every event per task.
// Folding paged history must retain only per-task totals.
func TestCostBreakdown_LargeHistoryRetainsOnlyTotals(t *testing.T) {
	const tasks, perTask = 600, 200 // 120k events, ~130 MiB of raw history

	before := heapAlloc()
	b := newCostBreakdownBuilder(0, false)
	if err := foldHistoryPages(syntheticHistory(t, tasks, perTask, historyPageSize), historyPageSize, b.add); err != nil {
		t.Fatal(err)
	}
	resp := b.result()
	retained := int64(heapAlloc()) - int64(before)
	runtime.KeepAlive(resp)

	if resp.TaskCount != tasks || len(resp.Tasks) != tasks {
		t.Fatalf("taskCount=%d tasks=%d, want %d", resp.TaskCount, len(resp.Tasks), tasks)
	}
	if got := resp.Tasks[0].EventCount; got != perTask {
		t.Errorf("eventCount = %d, want %d", got, perTask)
	}
	if resp.Tasks[0].Events != nil {
		t.Errorf("events kept without events=true")
	}
	if resp.TotalCost < 5.99 || resp.TotalCost > 6.01 {
		t.Errorf("totalCost = %v, want 6", resp.TotalCost)
	}
	// 600 tasks with a 200-char instruction each is well under 1 MiB.
	if retained > 1<<20 {
		t.Errorf("breakdown retains %d bytes, want < 1 MiB", retained)
	}
	t.Logf("retained %d KiB for %d events", retained>>10, tasks*perTask)
}

func TestCostBreakdown_LimitKeepsTotals(t *testing.T) {
	b := newCostBreakdownBuilder(10, false)
	if err := foldHistoryPages(syntheticHistory(t, 100, 5, 7), 7, b.add); err != nil {
		t.Fatal(err)
	}
	resp := b.result()
	if len(resp.Tasks) != 10 || resp.Tasks[0].Index != 90 || resp.Tasks[9].Index != 99 {
		t.Fatalf("tasks = %d (first index %d), want the 10 most recent", len(resp.Tasks), resp.Tasks[0].Index)
	}
	if resp.TaskCount != 100 || resp.TotalCost < 0.99 || resp.TotalCost > 1.01 {
		t.Errorf("taskCount=%d totalCost=%v, want totals over all 100 tasks", resp.TaskCount, resp.TotalCost)
	}
}

func TestCostBreakdown_EventsOptInIsCapped(t *testing.T) {
	b := newCostBreakdownBuilder(0, true)
	if err := foldHistoryPages(syntheticHistory(t, 120, 4, historyPageSize), historyPageSize, b.add); err != nil {
		t.Fatal(err)
	}
	resp := b.result()
	if len(resp.Tasks) != maxEventTasks {
		t.Fatalf("tasks = %d, want %d", len(resp.Tasks), maxEventTasks)
	}
	if got := len(resp.Tasks[0].Events); got != 4 {
		t.Errorf("events = %d, want 4", got)
	}
	if resp.TaskCount != 120 {
		t.Errorf("taskCount = %d, want 120", resp.TaskCount)
	}
}
