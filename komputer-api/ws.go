package main

import (
	"context"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// sendQueueCapacity is how many messages can be buffered per WS connection
// before older ones are dropped. ~3-4 seconds of typical agent output at
// peak rates.
const sendQueueCapacity = 256

// Hub manages WebSocket connections per agent name.
//
// Connections are split into three buckets:
//   - broadcast: every event is delivered to every client (legacy fan-out).
//   - groups: each group is a queue — exactly one client per group receives
//     each event. Coordination across replicas is done via Redis SET NX
//     claim keys, so adding clients does not add Redis connections.
//   - matchers: per-connection matcher list for multi-agent subscriptions.
type Hub struct {
	mu        sync.RWMutex
	broadcast map[string]map[*sendQueue]struct{}            // agentName -> queues
	groups    map[string]map[string]map[*sendQueue]struct{} // agentName -> group -> queues
	matchers  map[*sendQueue]*agentMatcher                  // queue -> matcher
	rdb       *redis.Client
	replicaID string
	claimTTL  time.Duration
}

func NewHub(rdb *redis.Client, replicaID string) *Hub {
	return &Hub{
		broadcast: make(map[string]map[*sendQueue]struct{}),
		groups:    make(map[string]map[string]map[*sendQueue]struct{}),
		matchers:  make(map[*sendQueue]*agentMatcher),
		rdb:       rdb,
		replicaID: replicaID,
		claimTTL:  30 * time.Second,
	}
}

// SubscribeBroadcast adds a queue that receives every event for agentName.
func (h *Hub) SubscribeBroadcast(agentName string, q *sendQueue) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.broadcast[agentName] == nil {
		h.broadcast[agentName] = make(map[*sendQueue]struct{})
	}
	h.broadcast[agentName][q] = struct{}{}
	wsConnectionsActive.WithLabelValues("broadcast").Inc()
}

// SubscribeGroup adds a queue to a named group.
func (h *Hub) SubscribeGroup(agentName, group string, q *sendQueue) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.groups[agentName] == nil {
		h.groups[agentName] = make(map[string]map[*sendQueue]struct{})
	}
	if h.groups[agentName][group] == nil {
		h.groups[agentName][group] = make(map[*sendQueue]struct{})
	}
	h.groups[agentName][group][q] = struct{}{}
	wsConnectionsActive.WithLabelValues("group").Inc()
}

// Unsubscribe removes the queue from broadcast and any groups for the agent.
func (h *Hub) Unsubscribe(agentName string, q *sendQueue) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.broadcast[agentName]; ok {
		if _, had := clients[q]; had {
			delete(clients, q)
			wsConnectionsActive.WithLabelValues("broadcast").Dec()
			if len(clients) == 0 {
				delete(h.broadcast, agentName)
			}
		}
	}
	if perGroup, ok := h.groups[agentName]; ok {
		for group, clients := range perGroup {
			if _, has := clients[q]; has {
				delete(clients, q)
				wsConnectionsActive.WithLabelValues("group").Dec()
				if len(clients) == 0 {
					delete(perGroup, group)
				}
			}
		}
		if len(perGroup) == 0 {
			delete(h.groups, agentName)
		}
	}
}

// SubscribeMatch adds a queue that receives every event whose (agentName, namespace)
// satisfies the given matcher.
func (h *Hub) SubscribeMatch(q *sendQueue, m *agentMatcher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.matchers[q] = m
	wsConnectionsActive.WithLabelValues("match").Inc()
}

// UnsubscribeMatch removes a queue from the match bucket.
func (h *Hub) UnsubscribeMatch(q *sendQueue) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, had := h.matchers[q]; had {
		delete(h.matchers, q)
		wsConnectionsActive.WithLabelValues("match").Dec()
	}
}

// Dispatch fans an event into the broadcast bucket and the group bucket.
//
// msgID is a stable identifier for the event (e.g. the Redis stream ID) used
// for cross-replica deduplication of group deliveries.
func (h *Hub) Dispatch(ctx context.Context, agentName, namespace, msgID string, message []byte) {
	// Snapshot under read lock; iterate (and Enqueue, which is non-blocking)
	// outside the lock.
	h.mu.RLock()
	bcast := make([]*sendQueue, 0, len(h.broadcast[agentName]))
	for q := range h.broadcast[agentName] {
		bcast = append(bcast, q)
	}
	groupMembers := make(map[string][]*sendQueue, len(h.groups[agentName]))
	for group, clients := range h.groups[agentName] {
		conns := make([]*sendQueue, 0, len(clients))
		for q := range clients {
			conns = append(conns, q)
		}
		if len(conns) > 0 {
			groupMembers[group] = conns
		}
	}
	matchTargets := make([]*sendQueue, 0, len(h.matchers))
	for q, m := range h.matchers {
		if m.matches(agentName, namespace) {
			matchTargets = append(matchTargets, q)
		}
	}
	h.mu.RUnlock()

	// Broadcast bucket: enqueue to all.
	for _, q := range bcast {
		q.Enqueue(message)
		wsDispatchTotal.WithLabelValues("broadcast", "delivered").Inc()
	}

	// Group buckets.
	for group, members := range groupMembers {
		claimKey := "wsclaim:" + agentName + ":" + group + ":" + msgID
		ok, err := h.rdb.SetNX(ctx, claimKey, h.replicaID, h.claimTTL).Result()
		if err != nil || !ok {
			wsDispatchTotal.WithLabelValues("group", "claimed_by_other").Inc()
			continue
		}
		// Pick a random member to deliver to. Enqueue is non-blocking, so we
		// can't observe write failure here — the writer goroutine on a dead
		// queue will close the connection and the read loop will Unsubscribe.
		// That's acceptable: at-most-once with reconcile via REST history.
		idx := rand.Intn(len(members))
		members[idx].Enqueue(message)
		wsDispatchTotal.WithLabelValues("group", "delivered").Inc()
	}

	// Match bucket: enqueue to every matching subscription.
	for _, q := range matchTargets {
		q.Enqueue(message)
		wsDispatchTotal.WithLabelValues("match", "delivered").Inc()
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleAgentWS handles GET /api/v1/agents/:name/ws
// @Summary Stream agent events (WebSocket)
// @Description Upgrades to a WebSocket connection to stream real-time agent events. Pass `?group=<name>` to join a consumer group: each event in the stream is delivered to exactly one client per group across all API replicas (useful for distributed consumers). Without `group`, every connected client receives every event (legacy broadcast).
// @Tags agents
// @Param name path string true "Agent name"
// @Param group query string false "Consumer group name. When set, this connection joins a group and Redis-coordinated single-delivery applies."
// @Router /agents/{name}/ws [get]
func HandleAgentWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		agentName := c.Param("name")
		group := c.Query("group")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			Logger.Errorw("ws upgrade failed", "error", err)
			return
		}

		mode := "broadcast"
		if group != "" {
			mode = "group"
		}
		q := newSendQueueWithMode(conn, sendQueueCapacity, mode)
		defer q.Close()

		if group != "" {
			hub.SubscribeGroup(agentName, group, q)
			Logger.Infow("ws client connected", "agent_name", agentName, "group", group)
		} else {
			hub.SubscribeBroadcast(agentName, q)
			Logger.Infow("ws client connected", "agent_name", agentName, "group", "broadcast")
		}
		defer hub.Unsubscribe(agentName, q)

		runReadLoop(conn)
	}
}

// runReadLoop installs ping/pong and blocks until the connection closes,
// returning so the caller's defers fire (Unsubscribe, Close).
func runReadLoop(conn *websocket.Conn) {
	const (
		pongWait   = 60 * time.Second
		pingPeriod = 30 * time.Second
	)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
