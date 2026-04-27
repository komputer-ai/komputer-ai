//go:build test

package main

import (
	"testing"
	"time"
)

func TestPerAgentWS_DeliversEvents(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	c := s.dialAgent("alice")
	defer c.Close()

	// Allow the server-side subscribe to register before we publish.
	time.Sleep(20 * time.Millisecond)

	s.publish("alice", "default", "text", map[string]any{"text": "hello"})

	ev, err := readWithTimeout(t, c, 2*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if ev.AgentName != "alice" || ev.Type != "text" {
		t.Fatalf("got agent=%s type=%s, want alice/text", ev.AgentName, ev.Type)
	}
}

func TestPerAgentWS_DoesNotReceiveOtherAgents(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	c := s.dialAgent("alice")
	defer c.Close()
	time.Sleep(20 * time.Millisecond)

	s.publish("bob", "default", "text", map[string]any{"text": "hi"})

	if _, err := readWithTimeout(t, c, 200*time.Millisecond); err == nil {
		t.Fatalf("expected timeout reading from alice's stream while bob published, got message")
	}
}
