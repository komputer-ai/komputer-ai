package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// sendQueue serializes writes to a single websocket.Conn through a bounded
// buffered channel. Producers call Enqueue, which is non-blocking; if the
// buffer is full, the oldest queued message is dropped to make room and a
// metric is incremented. A single writer goroutine drains the queue and
// performs WriteMessage with a write deadline.
//
// This decouples slow clients from the dispatcher: a client that stops
// reading can never block other clients, only its own queue fills and drops.
type sendQueue struct {
	conn      *websocket.Conn
	ch        chan []byte
	closeOnce sync.Once
	closed    chan struct{}
	mode      string // for metrics labels: broadcast|group|match
}

const sendQueueWriteTimeout = 10 * time.Second

func newSendQueue(conn *websocket.Conn, capacity int) *sendQueue {
	q := &sendQueue{
		conn:   conn,
		ch:     make(chan []byte, capacity),
		closed: make(chan struct{}),
		mode:   "broadcast",
	}
	go q.run()
	return q
}

// newSendQueueWithMode is the same constructor but allows the caller to set
// the metric label (broadcast|group|match).
func newSendQueueWithMode(conn *websocket.Conn, capacity int, mode string) *sendQueue {
	q := newSendQueue(conn, capacity)
	q.mode = mode
	return q
}

// Enqueue tries to add msg to the queue without blocking. If the queue is
// full, the oldest message is dropped first. After Close() returns, Enqueue
// is a no-op.
func (q *sendQueue) Enqueue(msg []byte) {
	select {
	case <-q.closed:
		return
	default:
	}
	for {
		select {
		case q.ch <- msg:
			return
		case <-q.closed:
			return
		default:
			// Drop oldest, then retry. Non-blocking receive — if another
			// goroutine drained between checks, fall through and try send.
			select {
			case <-q.ch:
				wsSendQueueDroppedTotal.WithLabelValues(q.mode).Inc()
			default:
			}
		}
	}
}

// Close stops the writer and closes the underlying connection. Safe to call
// multiple times.
func (q *sendQueue) Close() {
	q.closeOnce.Do(func() {
		close(q.closed)
		_ = q.conn.Close()
	})
}

func (q *sendQueue) run() {
	for {
		select {
		case <-q.closed:
			return
		case msg := <-q.ch:
			_ = q.conn.SetWriteDeadline(time.Now().Add(sendQueueWriteTimeout))
			if err := q.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				// Connection broken — the connection owner will detect this via
				// the read loop and call Close. We just stop draining.
				q.Close()
				return
			}
		}
	}
}
