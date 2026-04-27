//go:build test

package main

// This file is shared test infrastructure. It must not be built into prod
// binaries. We use a build tag rather than `_test.go` because helpers are
// referenced from multiple test files.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// wsTestServer is the bag returned by newWSTestServer.
type wsTestServer struct {
	t   *testing.T
	mr  *miniredis.Miniredis
	rdb *redis.Client
	hub *Hub
	srv *httptest.Server
}

// newWSTestServer creates a miniredis instance, a Hub, a Gin engine with the
// per-agent and multi WS routes wired up, and an httptest server fronting it.
//
// The caller is responsible for calling Close().
func newWSTestServer(t *testing.T) *wsTestServer {
	t.Helper()
	// InitLogger must be called before handlers that use Logger are exercised.
	InitLogger()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	hub := NewHub(rdb, "test-replica")

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/agents/:name/ws", HandleAgentWS(hub))
	r.GET("/api/v1/agents/events/ws", HandleMultiAgentWS(hub))

	srv := httptest.NewServer(r)
	return &wsTestServer{t: t, mr: mr, rdb: rdb, hub: hub, srv: srv}
}

func (s *wsTestServer) Close() {
	s.srv.Close()
	_ = s.rdb.Close()
	s.mr.Close()
}

// dialAgent opens a WS to the per-agent endpoint.
func (s *wsTestServer) dialAgent(agentName string) *websocket.Conn {
	s.t.Helper()
	wsURL := "ws" + strings.TrimPrefix(s.srv.URL, "http") +
		"/api/v1/agents/" + agentName + "/ws"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		s.t.Fatalf("dial %s: %v", agentName, err)
	}
	return c
}

// dialMulti opens a WS to the multi endpoint with the given query string
// (e.g. "match=worker-*&namespace=default").
func (s *wsTestServer) dialMulti(query string) *websocket.Conn {
	s.t.Helper()
	wsURL := "ws" + strings.TrimPrefix(s.srv.URL, "http") +
		"/api/v1/agents/events/ws?" + query
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		s.t.Fatalf("dial multi %s: %v", query, err)
	}
	return c
}

// publish synthesizes an event message and dispatches it through the Hub.
// This bypasses the Redis worker pump — adequate for hub/handler tests since
// the worker's job is just to call hub.Dispatch.
func (s *wsTestServer) publish(agentName, namespace, eventType string, payload map[string]any) {
	s.t.Helper()
	if payload == nil {
		payload = map[string]any{}
	}
	ev := AgentEvent{
		AgentName: agentName,
		Namespace: namespace,
		Type:      eventType,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   payload,
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		s.t.Fatalf("marshal: %v", err)
	}
	msgID := fmt.Sprintf("%d-0", time.Now().UnixNano())
	s.hub.Dispatch(context.Background(), agentName, namespace, msgID, raw)
}

// readWithTimeout reads one message with a deadline, returning the decoded
// AgentEvent or an error.
func readWithTimeout(t *testing.T, conn *websocket.Conn, timeout time.Duration) (AgentEvent, error) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		return AgentEvent{}, err
	}
	var ev AgentEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return AgentEvent{}, fmt.Errorf("unmarshal %q: %w", string(raw), err)
	}
	return ev, nil
}

// silence lint for unused on platforms that build _test.go without tag
var _ http.Handler = (http.HandlerFunc)(nil)
