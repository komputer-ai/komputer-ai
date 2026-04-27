//go:build test

package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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

func TestMultiWS_RejectsEmptySubscription(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.srv.URL, "http") + "/api/v1/agents/events/ws"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %v", resp)
	}
}

func TestMultiWS_WildcardMatchesAll(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	c := s.dialMulti("match=*")
	defer c.Close()
	time.Sleep(20 * time.Millisecond)

	s.publish("alice", "default", "text", map[string]any{"text": "hi"})
	s.publish("bob", "default", "text", map[string]any{"text": "yo"})

	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		ev, err := readWithTimeout(t, c, 2*time.Second)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		got[ev.AgentName] = true
	}
	if !got["alice"] || !got["bob"] {
		t.Fatalf("expected alice and bob, got %v", got)
	}
}

func TestMultiWS_PrefixGlobFiltersOut(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	c := s.dialMulti("match=worker-*")
	defer c.Close()
	time.Sleep(20 * time.Millisecond)

	s.publish("worker-1", "default", "text", map[string]any{})
	s.publish("manager-1", "default", "text", map[string]any{}) // should not arrive
	s.publish("worker-2", "default", "text", map[string]any{})

	got := []string{}
	for i := 0; i < 2; i++ {
		ev, err := readWithTimeout(t, c, 2*time.Second)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		got = append(got, ev.AgentName)
	}
	if _, err := readWithTimeout(t, c, 200*time.Millisecond); err == nil {
		t.Fatalf("expected no third message (manager-1 filtered out), got one")
	}
	for _, n := range got {
		if !strings.HasPrefix(n, "worker-") {
			t.Fatalf("got non-worker message: %s", n)
		}
	}
}

func TestMultiWS_ExplicitListUnionedWithGlob(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	c := s.dialMulti("match=worker-*&agents=manager-1")
	defer c.Close()
	time.Sleep(20 * time.Millisecond)

	s.publish("worker-1", "default", "text", map[string]any{})
	s.publish("manager-1", "default", "text", map[string]any{})
	s.publish("other", "default", "text", map[string]any{}) // filtered

	want := map[string]bool{"worker-1": true, "manager-1": true}
	for len(want) > 0 {
		ev, err := readWithTimeout(t, c, 2*time.Second)
		if err != nil {
			t.Fatalf("read with %d expected remaining: %v", len(want), err)
		}
		if !want[ev.AgentName] {
			t.Fatalf("got unexpected agent %s", ev.AgentName)
		}
		delete(want, ev.AgentName)
	}
	if _, err := readWithTimeout(t, c, 200*time.Millisecond); err == nil {
		t.Fatalf("expected no extra message after both received")
	}
}

func TestMultiWS_NamespaceFilter(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	c := s.dialMulti("match=*&namespace=team-a")
	defer c.Close()
	time.Sleep(20 * time.Millisecond)

	s.publish("alice", "team-a", "text", map[string]any{})
	s.publish("alice", "team-b", "text", map[string]any{}) // filtered

	ev, err := readWithTimeout(t, c, 2*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if ev.Namespace != "team-a" {
		t.Fatalf("got namespace %s, want team-a", ev.Namespace)
	}
	if _, err := readWithTimeout(t, c, 200*time.Millisecond); err == nil {
		t.Fatalf("expected no message from team-b namespace")
	}
}

func TestMultiWS_DynamicMembership_NewAgentArrives(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	c := s.dialMulti("match=worker-*")
	defer c.Close()
	time.Sleep(20 * time.Millisecond)

	// Initially no events. Agent appears later.
	s.publish("worker-99", "default", "task_started", map[string]any{"prompt": "go"})
	ev, err := readWithTimeout(t, c, 2*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if ev.AgentName != "worker-99" {
		t.Fatalf("got %s, want worker-99", ev.AgentName)
	}
}

func TestMultiWS_SlowClientDoesNotStallOthers(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	slow := s.dialMulti("match=*")
	defer slow.Close()
	fast := s.dialMulti("match=*")
	defer fast.Close()
	time.Sleep(20 * time.Millisecond)

	// Don't read on `slow`. Publish many events. `fast` should still receive.
	for i := 0; i < 1000; i++ {
		s.publish("alice", "default", "text", map[string]any{"i": i})
	}

	count := 0
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		fast.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, _, err := fast.ReadMessage()
		if err != nil {
			break
		}
		count++
	}
	if count < 100 {
		t.Fatalf("fast client only received %d messages, expected lots", count)
	}
}

func TestMultiWS_AgentsParam_UnknownAgentNoOp(t *testing.T) {
	s := newWSTestServer(t)
	defer s.Close()

	// First verify an unknown agent produces no messages (subscribe, wait, no publish).
	cQuiet := s.dialMulti("agents=ghost")
	defer cQuiet.Close()
	time.Sleep(20 * time.Millisecond)
	if _, err := readWithTimeout(t, cQuiet, 200*time.Millisecond); err == nil {
		t.Fatal("expected no message for ghost agent before it publishes")
	}

	// Open a fresh connection (reusing a timed-out conn is unreliable on some
	// platforms) and verify that once "ghost" publishes, it gets through.
	c := s.dialMulti("agents=ghost")
	defer c.Close()
	time.Sleep(20 * time.Millisecond)

	s.publish("ghost", "default", "text", map[string]any{})
	if _, err := readWithTimeout(t, c, 2*time.Second); err != nil {
		t.Fatalf("expected message after ghost publishes: %v", err)
	}
}
