package main

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

// ─── parseStreamMessage ───────────────────────────────────────────────────────

func TestParseStreamMessage(t *testing.T) {
	tests := []struct {
		name      string
		values    map[string]interface{}
		wantAgent string
		wantNS    string
		wantType  string
		wantErr   bool
	}{
		{
			name: "valid message with namespace",
			values: map[string]interface{}{
				"agentName": "my-agent",
				"namespace": "prod",
				"type":      "text",
				"timestamp": "2024-01-01T00:00:00Z",
				"payload":   `{"content":"hello"}`,
			},
			wantAgent: "my-agent",
			wantNS:    "prod",
			wantType:  "text",
		},
		{
			name: "valid message without namespace defaults to default",
			values: map[string]interface{}{
				"agentName": "my-agent",
				"type":      "task_started",
				"timestamp": "2024-01-01T00:00:00Z",
				"payload":   `{}`,
			},
			wantAgent: "my-agent",
			wantNS:    "default",
			wantType:  "task_started",
		},
		{
			name: "empty namespace defaults to default",
			values: map[string]interface{}{
				"agentName": "agent-x",
				"namespace": "",
				"type":      "error",
				"timestamp": "2024-01-01T00:00:00Z",
				"payload":   `{"error":"boom"}`,
			},
			wantAgent: "agent-x",
			wantNS:    "default",
			wantType:  "error",
		},
		{
			name: "missing agentName returns error",
			values: map[string]interface{}{
				"type":      "text",
				"timestamp": "2024-01-01T00:00:00Z",
				"payload":   `{}`,
			},
			wantErr: true,
		},
		{
			name: "missing type returns error",
			values: map[string]interface{}{
				"agentName": "agent",
				"timestamp": "2024-01-01T00:00:00Z",
				"payload":   `{}`,
			},
			wantErr: true,
		},
		{
			name: "missing timestamp returns error",
			values: map[string]interface{}{
				"agentName": "agent",
				"type":      "text",
				"payload":   `{}`,
			},
			wantErr: true,
		},
		{
			name: "missing payload returns error",
			values: map[string]interface{}{
				"agentName": "agent",
				"type":      "text",
				"timestamp": "2024-01-01T00:00:00Z",
			},
			wantErr: true,
		},
		{
			name: "malformed payload JSON returns error",
			values: map[string]interface{}{
				"agentName": "agent",
				"type":      "text",
				"timestamp": "2024-01-01T00:00:00Z",
				"payload":   `{not valid json`,
			},
			wantErr: true,
		},
		{
			name: "payload with nested fields parsed correctly",
			values: map[string]interface{}{
				"agentName": "agent",
				"type":      "tool_call",
				"timestamp": "2024-01-01T00:00:00Z",
				"payload":   `{"tool":"Bash","input":{"command":"ls"}}`,
			},
			wantAgent: "agent",
			wantNS:    "default",
			wantType:  "tool_call",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := redis.XMessage{ID: "1-0", Values: tt.values}
			got, err := parseStreamMessage(msg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.AgentName != tt.wantAgent {
				t.Errorf("AgentName = %q, want %q", got.AgentName, tt.wantAgent)
			}
			if got.Namespace != tt.wantNS {
				t.Errorf("Namespace = %q, want %q", got.Namespace, tt.wantNS)
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
		})
	}
}

// ─── mapEventToTaskStatus ─────────────────────────────────────────────────────

func TestMapEventToTaskStatus(t *testing.T) {
	tests := []struct {
		name        string
		event       AgentEvent
		wantStatus  string
		wantMessage string
	}{
		{
			name:        "task_started",
			event:       AgentEvent{Type: "task_started", Payload: map[string]interface{}{}},
			wantStatus:  "InProgress",
			wantMessage: "Task started",
		},
		{
			name:        "text event",
			event:       AgentEvent{Type: "text", Payload: map[string]interface{}{"content": "Hello world"}},
			wantStatus:  "InProgress",
			wantMessage: "Hello world",
		},
		{
			name:        "thinking event",
			event:       AgentEvent{Type: "thinking", Payload: map[string]interface{}{"content": "..."}},
			wantStatus:  "InProgress",
			wantMessage: "Thinking...",
		},
		{
			name:        "tool_call event",
			event:       AgentEvent{Type: "tool_call", Payload: map[string]interface{}{"tool": "Bash"}},
			wantStatus:  "InProgress",
			wantMessage: "Calling Bash",
		},
		{
			name:        "tool_result event",
			event:       AgentEvent{Type: "tool_result", Payload: map[string]interface{}{"tool": "Read"}},
			wantStatus:  "InProgress",
			wantMessage: "Got result from Read",
		},
		{
			name:        "task_completed",
			event:       AgentEvent{Type: "task_completed", Payload: map[string]interface{}{}},
			wantStatus:  "Complete",
			wantMessage: "Task completed",
		},
		{
			name:        "task_cancelled",
			event:       AgentEvent{Type: "task_cancelled", Payload: map[string]interface{}{}},
			wantStatus:  "Complete",
			wantMessage: "Task cancelled",
		},
		{
			name:        "error event",
			event:       AgentEvent{Type: "error", Payload: map[string]interface{}{"error": "something went wrong"}},
			wantStatus:  "Error",
			wantMessage: "Error: something went wrong",
		},
		{
			name:        "unknown event type returns empty",
			event:       AgentEvent{Type: "unknown_type", Payload: map[string]interface{}{}},
			wantStatus:  "",
			wantMessage: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotMsg := mapEventToTaskStatus(tt.event)
			if gotStatus != tt.wantStatus {
				t.Errorf("taskStatus = %q, want %q", gotStatus, tt.wantStatus)
			}
			if gotMsg != tt.wantMessage {
				t.Errorf("lastMessage = %q, want %q", gotMsg, tt.wantMessage)
			}
		})
	}
}

// ─── truncate (API) ───────────────────────────────────────────────────────────

func TestTruncateAPI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{"empty string", "", 10, ""},
		{"shorter than max", "hello", 10, "hello"},
		{"exactly max", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hello"},
		{"zero max", "hello", 0, ""},
		{"unicode rune-safe", "héllo", 3, "hél"},
		{"emoji rune-safe", "😀😁😂", 2, "😀😁"},
		{"unicode no truncation needed", "こんにちは", 10, "こんにちは"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}
