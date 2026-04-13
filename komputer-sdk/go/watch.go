// WebSocket streaming for agents — hand-written, preserved across regeneration.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

// AgentEvent represents a single event from an agent's WebSocket stream.
type AgentEvent struct {
	AgentName string                 `json:"agentName"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// AgentEventStream provides an iterator-style interface for reading agent events.
// History events are yielded first, followed by live WebSocket events.
// Duplicate events are suppressed using a timestamp+type dedup key.
type AgentEventStream struct {
	conn    *websocket.Conn
	name    string
	history []*AgentEvent
	seen    map[string]struct{}
}

// dedupKey returns the dedup key for an event.
// "task_started" is normalized to "user_message" to match the REST history type.
func dedupKey(e *AgentEvent) string {
	normType := e.Type
	if normType == "task_started" {
		normType = "user_message"
	}
	return e.Timestamp + ":" + normType
}

// WatchAgent prefetches history via REST (up to 200 events) and opens a
// WebSocket connection to stream live events. History events are yielded first;
// live events that duplicate a history event (by timestamp+type) are dropped.
//
// Events include: task_started, thinking, tool_use, tool_result, text,
// task_completed, task_cancelled, error.
//
// Usage:
//
//	stream, err := client.WatchAgent(ctx, "my-agent")
//	if err != nil { ... }
//	defer stream.Close()
//	for {
//	    event, err := stream.Next()
//	    if err != nil { break }
//	    fmt.Println(event.Type, event.Payload)
//	}
func (c *Client) WatchAgent(ctx context.Context, name string) (*AgentEventStream, error) {
	// 1. Fetch history via REST.
	var history []*AgentEvent
	seen := make(map[string]struct{})

	resp, _, err := c.api.AgentsAPI.GetAgentEvents(ctx, name).Limit(200).Execute()
	if err == nil && resp != nil {
		if raw, ok := resp["events"]; ok {
			if events, ok := raw.([]interface{}); ok {
				for _, item := range events {
					m, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					e := &AgentEvent{
						AgentName: stringField(m, "agentName", name),
						Type:      stringField(m, "type", ""),
						Timestamp: stringField(m, "timestamp", ""),
						Payload:   payloadField(m),
					}
					history = append(history, e)
					seen[dedupKey(e)] = struct{}{}
				}
			}
		}
	}
	// History fetch failure is non-fatal — proceed with live-only.

	// 2. Open WebSocket for live events.
	wsURL := strings.Replace(c.baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	url := fmt.Sprintf("%s/api/v1/agents/%s/ws", wsURL, name)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	return &AgentEventStream{conn: conn, name: name, history: history, seen: seen}, nil
}

// Next reads the next event from the stream. History events are returned first,
// then live WebSocket events. Duplicate live events (by timestamp+type key) are
// skipped automatically. Returns an error when the connection is closed or a
// read error occurs.
func (s *AgentEventStream) Next() (*AgentEvent, error) {
	// Drain history queue first.
	if len(s.history) > 0 {
		e := s.history[0]
		s.history = s.history[1:]
		return e, nil
	}

	// Read from live WebSocket, skipping already-seen events.
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			return nil, err
		}

		var event AgentEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			return nil, fmt.Errorf("unmarshal event: %w", err)
		}
		if event.AgentName == "" {
			event.AgentName = s.name
		}

		key := dedupKey(&event)
		if _, dup := s.seen[key]; dup {
			continue
		}
		s.seen[key] = struct{}{}
		return &event, nil
	}
}

// Close closes the WebSocket connection.
func (s *AgentEventStream) Close() error {
	return s.conn.Close()
}

// stringField extracts a string value from a map, returning fallback if missing or wrong type.
func stringField(m map[string]interface{}, key, fallback string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return fallback
}

// payloadField extracts the payload map from an event map.
func payloadField(m map[string]interface{}) map[string]interface{} {
	if v, ok := m["payload"]; ok {
		if p, ok := v.(map[string]interface{}); ok {
			return p
		}
	}
	return map[string]interface{}{}
}
