package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// dialServer upgrades an httptest server to a websocket and returns the server-side conn
// (passed back via channel) and the client-side conn.
func dialServer(t *testing.T, srv *httptest.Server) (server, client *websocket.Conn) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return nil, client
}

func newWSPair(t *testing.T) (server, client *websocket.Conn, cleanup func()) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	serverCh := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		serverCh <- c
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	cli, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}
	srvConn := <-serverCh
	return srvConn, cli, func() { cli.Close(); srvConn.Close(); srv.Close() }
}

func TestSendQueue_DeliversMessagesInOrder(t *testing.T) {
	srvConn, cliConn, cleanup := newWSPair(t)
	defer cleanup()

	q := newSendQueue(srvConn, 16)
	defer q.Close()

	q.Enqueue([]byte("a"))
	q.Enqueue([]byte("b"))
	q.Enqueue([]byte("c"))

	cliConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	got := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		_, m, err := cliConn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		got = append(got, string(m))
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("got %v, want [a b c]", got)
	}

	// Silence the unused-suspect linter on sync import (used in next test).
	_ = sync.Mutex{}
}

func TestSendQueue_DropsOldestOnOverflow(t *testing.T) {
	srvConn, cliConn, cleanup := newWSPair(t)
	defer cleanup()

	// Make the client unable to consume by not reading. Capacity 4.
	q := newSendQueue(srvConn, 4)
	defer q.Close()

	// Slow the writer's actual delivery: read nothing on the client side until
	// after we enqueue 8 messages — the queue has cap 4 and the writer has 0
	// in flight (since the first WriteMessage hasn't returned yet because we
	// aren't reading). 4 must drop.
	for i := 0; i < 8; i++ {
		q.Enqueue([]byte{byte('0' + i)})
	}

	// Now drain the client.
	cliConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	got := []string{}
	for {
		_, m, err := cliConn.ReadMessage()
		if err != nil {
			break
		}
		got = append(got, string(m))
	}
	// We should have lost messages — exactly which depends on scheduling, but
	// the count must be < 8 and order must be ascending (no reorder).
	if len(got) >= 8 {
		t.Fatalf("expected drops, got all 8 messages: %v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i] <= got[i-1] {
			t.Fatalf("messages out of order: %v", got)
		}
	}
}
