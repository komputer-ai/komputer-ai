package main

import (
	"math"
	"testing"
)

const testProfileARN = "arn:aws:bedrock:eu-central-1:123456789012:application-inference-profile/abc123"

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestRepriceInferenceProfileCost(t *testing.T) {
	// 1M of each token kind on Sonnet 5: 2 + 10 + 0.20 + 2.50.
	sonnetMillion := map[string]interface{}{
		"inputTokens": 1e6, "outputTokens": 1e6, "cacheReadInputTokens": 1e6, "cacheCreationInputTokens": 1e6,
	}

	cases := []struct {
		name     string
		eventTyp string
		payload  map[string]interface{}
		wantCost float64
		wantSDK  bool // sdk_cost_usd recorded
	}{
		{
			name:     "ARN model priced from model_usage key",
			eventTyp: "task_completed",
			payload: map[string]interface{}{
				"cost_usd": 99.0, "model": testProfileARN,
				"model_usage": map[string]interface{}{"claude-sonnet-5": sonnetMillion},
			},
			wantCost: 14.70, wantSDK: true,
		},
		{
			name:     "usage keyed by the ARN falls back to response_model",
			eventTyp: "task_cancelled",
			payload: map[string]interface{}{
				"cost_usd": 99.0, "model": testProfileARN, "response_model": "claude-sonnet-5",
				"model_usage": map[string]interface{}{testProfileARN: sonnetMillion},
			},
			wantCost: 14.70, wantSDK: true,
		},
		{
			name:     "several models are summed, each at its own rate",
			eventTyp: "task_completed",
			payload: map[string]interface{}{
				"cost_usd": 99.0, "model": testProfileARN,
				"model_usage": map[string]interface{}{
					"claude-sonnet-5":  map[string]interface{}{"outputTokens": 1e6},
					"claude-haiku-4-5": map[string]interface{}{"outputTokens": 1e6},
				},
			},
			wantCost: 10 + 5, wantSDK: true,
		},
		{
			name:     "recognisable configured model keeps Claude Code's cost",
			eventTyp: "task_completed",
			payload: map[string]interface{}{
				"cost_usd": 1.23, "model": "eu.anthropic.claude-sonnet-4-6",
				"model_usage": map[string]interface{}{"claude-sonnet-4-6": sonnetMillion},
			},
			wantCost: 1.23,
		},
		{
			name:     "nothing priceable keeps Claude Code's cost",
			eventTyp: "task_completed",
			payload: map[string]interface{}{
				"cost_usd": 1.23, "model": testProfileARN,
				"model_usage": map[string]interface{}{testProfileARN: sonnetMillion},
			},
			wantCost: 1.23,
		},
		{
			name:     "older agent without model_usage is untouched",
			eventTyp: "task_completed",
			payload:  map[string]interface{}{"cost_usd": 1.23, "model": testProfileARN},
			wantCost: 1.23,
		},
		{
			name:     "progress events are ignored",
			eventTyp: "text",
			payload: map[string]interface{}{
				"cost_usd": 1.23, "model": testProfileARN,
				"model_usage": map[string]interface{}{"claude-sonnet-5": sonnetMillion},
			},
			wantCost: 1.23,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev := AgentEvent{AgentName: "a", Type: tc.eventTyp, Payload: tc.payload}
			sdk := tc.payload["cost_usd"]
			repriceInferenceProfileCost(&ev)
			if got := ev.Payload["cost_usd"].(float64); !approx(got, tc.wantCost) {
				t.Errorf("cost_usd = %v, want %v", got, tc.wantCost)
			}
			got, recorded := ev.Payload["sdk_cost_usd"]
			if recorded != tc.wantSDK {
				t.Fatalf("sdk_cost_usd recorded = %v, want %v", recorded, tc.wantSDK)
			}
			if recorded && got != sdk {
				t.Errorf("sdk_cost_usd = %v, want the original %v", got, sdk)
			}
		})
	}
}

// TestRepricedCostReachesTaskAccounting checks the corrected cost is what the
// status writer sees, since the worker reads accounting after parseStreamMessage.
func TestRepricedCostReachesTaskAccounting(t *testing.T) {
	ev := AgentEvent{AgentName: "a", Type: "task_completed", Payload: map[string]interface{}{
		"cost_usd": 99.0, "model": testProfileARN,
		"model_usage": map[string]interface{}{"claude-sonnet-5": map[string]interface{}{"outputTokens": 1e6}},
	}}
	repriceInferenceProfileCost(&ev)
	if _, cost, _, _ := extractTaskAccounting(ev); !approx(cost, 10) {
		t.Errorf("accounted cost = %v, want 10", cost)
	}
}
