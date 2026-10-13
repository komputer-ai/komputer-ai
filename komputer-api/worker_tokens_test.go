package main

import (
	"testing"
)

func TestExtractTotalTokens(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]interface{}
		want    int64
	}{
		{
			name: "both input and output present",
			payload: map[string]interface{}{
				"usage": map[string]interface{}{
					"input_tokens":  float64(100),
					"output_tokens": float64(50),
				},
			},
			want: 150,
		},
		{
			name: "only input present",
			payload: map[string]interface{}{
				"usage": map[string]interface{}{
					"input_tokens": float64(200),
				},
			},
			want: 200,
		},
		{
			name: "usage is nil",
			payload: map[string]interface{}{
				"usage": nil,
			},
			want: 0,
		},
		{
			name:    "no usage key",
			payload: map[string]interface{}{},
			want:    0,
		},
		{
			name: "usage is not a map",
			payload: map[string]interface{}{
				"usage": "invalid",
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTotalTokens(tt.payload)
			if got != tt.want {
				t.Errorf("extractTotalTokens() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestExtractTaskAccounting locks in that a cancelled task's cost/session/token
// fields are read exactly like a completed task's — before the fix, this
// extraction only fired for "task_completed", so every cancelled task's spend
// was silently dropped on the floor and never reached PatchAgentTaskStatus.
func TestExtractTaskAccounting(t *testing.T) {
	tests := []struct {
		name              string
		event             AgentEvent
		wantSessionID     string
		wantCostUSD       float64
		wantTotalTokens   int64
		wantContextWindow int64
	}{
		{
			name: "task_completed carries cost and session",
			event: AgentEvent{
				Type: "task_completed",
				Payload: map[string]interface{}{
					"session_id":     "sess-abc",
					"cost_usd":       0.1234,
					"context_window": float64(200000),
					"usage": map[string]interface{}{
						"input_tokens":  float64(100),
						"output_tokens": float64(50),
					},
				},
			},
			wantSessionID:     "sess-abc",
			wantCostUSD:       0.1234,
			wantTotalTokens:   150,
			wantContextWindow: 200000,
		},
		{
			name: "task_cancelled carries the same accounting fields",
			event: AgentEvent{
				Type: "task_cancelled",
				Payload: map[string]interface{}{
					"reason":         "Cancelled by user",
					"session_id":     "sess-def",
					"cost_usd":       0.0456,
					"context_window": float64(200000),
					"usage": map[string]interface{}{
						"input_tokens":  float64(10),
						"output_tokens": float64(20),
					},
				},
			},
			wantSessionID:     "sess-def",
			wantCostUSD:       0.0456,
			wantTotalTokens:   30,
			wantContextWindow: 200000,
		},
		{
			name: "task_cancelled prefers last_usage for context size",
			event: AgentEvent{
				Type: "task_cancelled",
				Payload: map[string]interface{}{
					"session_id": "sess-ghi",
					"cost_usd":   0.01,
					"usage": map[string]interface{}{
						"input_tokens": float64(1000),
					},
					"last_usage": map[string]interface{}{
						"input_tokens":            float64(5),
						"cache_read_input_tokens": float64(3),
					},
				},
			},
			wantSessionID:   "sess-ghi",
			wantCostUSD:     0.01,
			wantTotalTokens: 8, // from last_usage, not the accumulated "usage" (1000)
		},
		{
			name: "non-terminal event yields zero values",
			event: AgentEvent{
				Type:    "tool_call",
				Payload: map[string]interface{}{"cost_usd": 99.0, "session_id": "should-be-ignored"},
			},
			wantSessionID:   "",
			wantCostUSD:     0,
			wantTotalTokens: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionID, costUSD, totalTokens, contextWindow := extractTaskAccounting(tt.event)
			if sessionID != tt.wantSessionID {
				t.Errorf("sessionID = %q, want %q", sessionID, tt.wantSessionID)
			}
			if costUSD != tt.wantCostUSD {
				t.Errorf("costUSD = %v, want %v", costUSD, tt.wantCostUSD)
			}
			if totalTokens != tt.wantTotalTokens {
				t.Errorf("totalTokens = %d, want %d", totalTokens, tt.wantTotalTokens)
			}
			if contextWindow != tt.wantContextWindow {
				t.Errorf("contextWindow = %d, want %d", contextWindow, tt.wantContextWindow)
			}
		})
	}
}
