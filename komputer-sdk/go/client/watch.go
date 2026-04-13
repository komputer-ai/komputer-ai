// WebSocket streaming for agents — hand-written, preserved across regeneration.

package client

import (
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
type AgentEventStream struct {
	conn *websocket.Conn
	name string
}

// WatchAgent opens a WebSocket connection to stream real-time events from an agent.
//
// Events include: task_started, thinking, tool_use, tool_result, text,
// task_completed, task_cancelled, error.
//
// Usage:
//
//	stream, err := client.WatchAgent("my-agent")
//	if err != nil { ... }
//	defer stream.Close()
//	for {
//	    event, err := stream.Next()
//	    if err != nil { break }
//	    fmt.Println(event.Type, event.Payload)
//	}
func (c *Client) WatchAgent(name string) (*AgentEventStream, error) {
	wsURL := strings.Replace(c.baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	url := fmt.Sprintf("%s/api/v1/agents/%s/ws", wsURL, name)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	return &AgentEventStream{conn: conn, name: name}, nil
}

// Next reads the next event from the stream. Returns an error when the
// connection is closed or a read error occurs.
func (s *AgentEventStream) Next() (*AgentEvent, error) {
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
	return &event, nil
}

// Close closes the WebSocket connection.
func (s *AgentEventStream) Close() error {
	return s.conn.Close()
}
