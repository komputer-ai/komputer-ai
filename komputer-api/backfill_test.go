package main

import (
	"math"
	"strings"
	"testing"
)

// TestRatesForModel verifies per-1M-token pricing is matched most-specific-first
// (e.g. "opus-4-5" must not fall through to the coarser "opus-4" entry), that
// Bedrock-style IDs (region-prefixed inference profiles and ARNs) match the same
// rules as friendly names, and that an unrecognized model id falls back to a
// family-level approximation instead of the stale blanket Opus rate.
func TestRatesForModel(t *testing.T) {
	cases := []struct {
		name  string
		model string
		want  modelRates
	}{
		// Fable / Mythos (Mythos shares Fable's pricing).
		{"fable-5-1", "claude-fable-5-1", modelRates{10, 50, 0.25, 12.50}},
		{"fable-5", "claude-fable-5", modelRates{10, 50, 1, 12.50}},
		{"mythos-5-1", "claude-mythos-5-1", modelRates{10, 50, 0.25, 12.50}},
		{"mythos-5", "claude-mythos-5", modelRates{10, 50, 1, 12.50}},

		// Opus — current, standard legacy tier, and the old $15/$75 tier.
		{"opus-5-5", "claude-opus-5-5", modelRates{4, 20, 0.20, 5}},
		{"opus-5", "claude-opus-5", modelRates{5, 25, 0.50, 6.25}},
		{"opus-4-8", "claude-opus-4-8", modelRates{5, 25, 0.50, 6.25}},
		{"opus-4-7", "claude-opus-4-7", modelRates{5, 25, 0.50, 6.25}},
		{"opus-4-6", "claude-opus-4-6", modelRates{5, 25, 0.50, 6.25}},
		{"opus-4-5", "claude-opus-4-5", modelRates{5, 25, 0.50, 6.25}},
		{"opus-4-1 dated id", "claude-opus-4-1-20250805", modelRates{15, 75, 1.50, 18.75}},
		{"opus-4 dated id", "claude-opus-4-20250514", modelRates{15, 75, 1.50, 18.75}},

		// Sonnet.
		{"sonnet-5-5", "claude-sonnet-5-5", modelRates{2, 10, 0.20, 2.50}},
		{"sonnet-5", "claude-sonnet-5", modelRates{2, 10, 0.20, 2.50}},
		{"sonnet-4-6", "claude-sonnet-4-6", modelRates{3, 15, 0.30, 3.75}},
		{"sonnet-4-5 dated id", "claude-sonnet-4-5-20250929", modelRates{3, 15, 0.30, 3.75}},
		{"sonnet-4 dated id", "claude-sonnet-4-20250514", modelRates{3, 15, 0.30, 3.75}},

		// Haiku. 3.5 Haiku's real id puts the version before the family name
		// ("claude-3-5-haiku-..."), unlike every 4.x+ id, so both orderings
		// must resolve to the same rate.
		{"haiku-4-5", "claude-haiku-4-5", modelRates{1, 5, 0.10, 1.25}},
		{"haiku-3-5 real id ordering", "claude-3-5-haiku-20241022", modelRates{0.80, 4, 0.08, 1}},
		{"haiku-3-5 name-first ordering", "claude-haiku-3-5", modelRates{0.80, 4, 0.08, 1}},

		// Bedrock-style IDs (region-prefixed inference profiles and full ARNs)
		// must match on the same embedded suffix.
		{"bedrock us opus-4-5", "us.anthropic.claude-opus-4-5-20251101-v1:0", modelRates{5, 25, 0.50, 6.25}},
		{"bedrock eu sonnet-4-6", "eu.anthropic.claude-sonnet-4-6", modelRates{3, 15, 0.30, 3.75}},
		{"bedrock apac haiku-3-5", "apac.anthropic.claude-3-5-haiku-20241022-v1:0", modelRates{0.80, 4, 0.08, 1}},
		{
			"bedrock arn opus-5-5",
			"arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.anthropic.claude-opus-5-5",
			modelRates{4, 20, 0.20, 5},
		},

		// Unknown ids fall back per family instead of defaulting to the
		// (stale) blanket Opus rate.
		{"unknown opus family", "claude-opus-9000", modelRates{4, 20, 0.20, 5}},
		{"unknown sonnet family", "claude-sonnet-9000", modelRates{2, 10, 0.20, 2.50}},
		{"unknown haiku family", "claude-haiku-9000", modelRates{1, 5, 0.10, 1.25}},
		{"unknown fable family", "claude-fable-9000", modelRates{10, 50, 0.25, 12.50}},
		{"totally unrecognized id", "some-future-model", modelRates{2, 10, 0.20, 2.50}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ratesForModel(tc.model)
			if got != tc.want {
				t.Fatalf("ratesForModel(%q) = %+v, want %+v", tc.model, got, tc.want)
			}
		})
	}
}

// TestOutputRatePerM spot-checks that the per-model output rate used for
// same-API-call token accounting matches the table above, not the old stale
// per-family switch.
func TestOutputRatePerM(t *testing.T) {
	cases := []struct {
		model string
		want  float64
	}{
		{"claude-opus-5-5", 20},
		{"claude-opus-4-5", 25},
		{"claude-sonnet-4-6", 15},
		{"claude-haiku-4-5", 5},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			if got := outputRatePerM(tc.model); got != tc.want {
				t.Fatalf("outputRatePerM(%q) = %v, want %v", tc.model, got, tc.want)
			}
		})
	}
}

// TestConvertSessionJSONL_CancelledSessionCarriesCost reconstructs a session
// whose JSONL ends right on an interruption (no further queue-operation entry
// and no steer) — the EOF flush path. Before the fix:
//   1. the cancel event carried only {"reason": ...}, no cost/usage at all, and
//   2. the unconditional emitTaskCompleted() call right after the EOF flush
//      double-emitted a second, duplicate-cost event for the very same turn
//      (lastAssistantTimestamp was never cleared by the old pendingCancel path).
// Both must be fixed: the cancel event should carry the turn's accumulated
// cost_usd/usage/num_turns, and no trailing task_completed should follow it.
func TestConvertSessionJSONL_CancelledSessionCarriesCost(t *testing.T) {
	raw := []byte(strings.Join([]string{
		`{"timestamp":"2024-01-01T00:00:00Z","message":{"role":"user","content":"do the thing"}}`,
		`{"timestamp":"2024-01-01T00:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"ok working on it"}],"usage":{"input_tokens":100,"output_tokens":50},"model":"claude-sonnet-4-6"}}`,
		`{"timestamp":"2024-01-01T00:00:02Z","message":{"role":"user","content":"[Request interrupted by user]"}}`,
	}, "\n"))

	events := convertSessionJSONL(raw, "agent-1", 0, "claude-sonnet-4-6", 0)

	var cancelled, completed int
	var cancelPayload map[string]interface{}
	for _, ev := range events {
		switch ev.Type {
		case "task_cancelled":
			cancelled++
			cancelPayload = ev.Payload
		case "task_completed":
			completed++
		}
	}

	if cancelled != 1 {
		t.Fatalf("got %d task_cancelled events, want 1 (events: %+v)", cancelled, events)
	}
	if completed != 0 {
		t.Fatalf("got %d task_completed events, want 0 — the EOF flush must not double-emit (events: %+v)", completed, events)
	}

	wantCost := estimateCostUSD("claude-sonnet-4-6", 100, 50, 0, 0)
	if c, ok := cancelPayload["cost_usd"].(float64); !ok || math.Abs(c-wantCost) > 1e-9 {
		t.Errorf("task_cancelled cost_usd = %v, want %v", cancelPayload["cost_usd"], wantCost)
	}
	if cancelPayload["reason"] != "Cancelled by user" {
		t.Errorf("task_cancelled reason = %v, want %q", cancelPayload["reason"], "Cancelled by user")
	}
	usage, ok := cancelPayload["usage"].(map[string]interface{})
	if !ok {
		t.Fatalf("task_cancelled usage missing or wrong type: %v", cancelPayload["usage"])
	}
	if usage["input_tokens"] != int64(100) || usage["output_tokens"] != int64(50) {
		t.Errorf("task_cancelled usage = %+v, want input_tokens=100 output_tokens=50", usage)
	}
}

// TestConvertSessionJSONL_CostScalingIncludesCancelledTasks verifies the final
// scaling pass (which reconciles backfill's per-token cost estimate against
// the CR's authoritative totalCostUSD) also scales task_cancelled events —
// otherwise a session with a cancelled task would under-count estimatedTotal
// and inflate every completed task's displayed cost to compensate.
func TestConvertSessionJSONL_CostScalingIncludesCancelledTasks(t *testing.T) {
	raw := []byte(strings.Join([]string{
		`{"timestamp":"2024-01-01T00:00:00Z","message":{"role":"user","content":"do the thing"}}`,
		`{"timestamp":"2024-01-01T00:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"ok working on it"}],"usage":{"input_tokens":100,"output_tokens":50},"model":"claude-sonnet-4-6"}}`,
		`{"timestamp":"2024-01-01T00:00:02Z","message":{"role":"user","content":"[Request interrupted by user]"}}`,
	}, "\n"))

	estimated := estimateCostUSD("claude-sonnet-4-6", 100, 50, 0, 0)
	authoritative := estimated * 2 // CR's real total is exactly double the per-token estimate

	events := convertSessionJSONL(raw, "agent-1", 0, "claude-sonnet-4-6", authoritative)

	var gotCost float64
	var found bool
	for _, ev := range events {
		if ev.Type == "task_cancelled" {
			gotCost, found = ev.Payload["cost_usd"].(float64)
		}
	}
	if !found {
		t.Fatalf("no task_cancelled event found (events: %+v)", events)
	}
	if math.Abs(gotCost-authoritative) > 1e-9 {
		t.Errorf("scaled task_cancelled cost_usd = %v, want %v (scale factor 2x applied)", gotCost, authoritative)
	}
}

// TestEstimateCostUSD exercises the full cost formula (not just the rate
// lookup), including a Bedrock id, to make sure the per-1M-token division and
// field wiring (input/output/cacheRead/cacheWrite) are all correct.
func TestEstimateCostUSD(t *testing.T) {
	cases := []struct {
		name                                   string
		model                                  string
		input, output, cacheRead, cacheCreate float64
		want                                   float64
	}{
		{"opus-5-5, input+output only", "claude-opus-5-5", 1_000_000, 1_000_000, 0, 0, 4 + 20},
		{"sonnet-4-6, cache read+write only", "claude-sonnet-4-6", 0, 0, 1_000_000, 1_000_000, 0.30 + 3.75},
		{"bedrock opus-4-1, input+output", "us.anthropic.claude-opus-4-1-20250805-v1:0", 1_000_000, 1_000_000, 0, 0, 15 + 75},
		{"zero tokens", "claude-sonnet-5-5", 0, 0, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := estimateCostUSD(tc.model, tc.input, tc.output, tc.cacheRead, tc.cacheCreate)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("estimateCostUSD(%q) = %v, want %v", tc.model, got, tc.want)
			}
		})
	}
}

// Newer Claude Code versions also write "api-request-blob" lines that carry a
// copy of the messages sent to the model, with no timestamp. Only real user and
// assistant entries are the conversation; the copies must not become events.
func TestConvertSessionJSONL_IgnoresRequestBlobs(t *testing.T) {
	raw := strings.Join([]string{
		`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-10-24T12:01:25.949Z"}`,
		`{"type":"user","timestamp":"2026-10-24T12:01:25.986Z","message":{"role":"user","content":"print the date"}}`,
		`{"type":"api-request-blob","message":{"role":"user","content":[{"type":"text","text":"print the date"}]}}`,
		`{"type":"assistant","timestamp":"2026-10-24T12:01:27.490Z","message":{"role":"assistant","content":[{"type":"text","text":"On it."}],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		`{"type":"api-request-blob","message":{"role":"assistant","content":[{"type":"text","text":"On it."}]}}`,
		`{"type":"assistant","timestamp":"2026-10-24T12:01:30.663Z","message":{"role":"assistant","content":[{"type":"text","text":"Done."}],"usage":{"input_tokens":12,"output_tokens":3}}}`,
	}, "\n")

	events := convertSessionJSONL([]byte(raw), "a1", 0, "claude-haiku-4-5", 0)
	var texts []string
	for _, e := range events {
		if e.Timestamp == "" {
			t.Errorf("%s event has no timestamp: %+v", e.Type, e.Payload)
		}
		if e.Type == "text" {
			texts = append(texts, e.Payload["content"].(string))
		}
	}
	if strings.Join(texts, "|") != "On it.|Done." {
		t.Errorf("text events = %q, want each assistant message once", texts)
	}
}
