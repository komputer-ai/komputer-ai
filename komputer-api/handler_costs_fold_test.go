package main

import (
	"testing"
)

// BenchmarkCostBreakdownFold folds a 100-task, 20k-event history that is
// already in memory, so it measures decoding and folding without Redis.
func BenchmarkCostBreakdownFold(b *testing.B) {
	const tasks, perTask = 100, 200
	var history []string
	gen := syntheticHistory(b, tasks, perTask, historyPageSize)
	for start := int64(0); start < tasks*perTask; start += historyPageSize {
		page, _ := gen(start, start+historyPageSize-1)
		history = append(history, page...)
	}
	lrange := func(start, stop int64) ([]string, error) {
		if start >= int64(len(history)) {
			return nil, nil
		}
		return history[start:min(stop+1, int64(len(history)))], nil
	}
	b.ReportAllocs()
	for b.Loop() {
		bd := newCostBreakdownBuilder(0, false)
		if err := foldHistoryPages(lrange, historyPageSize, bd.addRaw); err != nil {
			b.Fatal(err)
		}
		if bd.result().TaskCount != tasks {
			b.Fatal("wrong task count")
		}
	}
}

func TestEventHead(t *testing.T) {
	cases := []struct {
		raw, typ, ts string
		ok           bool
	}{
		{`{"agentName":"a","type":"tool_result","timestamp":"t1","payload":{"output":"x"}}`, "tool_result", "t1", true},
		{`{"payload":{"type":"nested"},"timestamp":"t2","type":"task_completed"}`, "task_completed", "t2", true},
		{`{"agentName":"a","timestamp":"t3"}`, "", "t3", false},
		{`not json`, "", "", false},
		{`["type"]`, "", "", false},
	}
	for _, tc := range cases {
		typ, ts, ok := eventHead(tc.raw)
		if typ != tc.typ || ts != tc.ts || ok != tc.ok {
			t.Errorf("eventHead(%s) = %q, %q, %v; want %q, %q, %v", tc.raw, typ, ts, ok, tc.typ, tc.ts, tc.ok)
		}
	}
}
