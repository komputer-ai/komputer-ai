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

// Hub manages WebSocket connections per agent name.
//
// Connections are split into two buckets:
//   - broadcast: every event is delivered to every client (legacy fan-out).
//   - groups: each group is a queue — exactly one client per group receives
//     each event. Coordination across replicas is done via Redis SET NX
//     claim keys, so adding clients does not add Redis connections.
type Hub struct {
	mu         sync.RWMutex
	broadcast  map[string]map[*websocket.Conn]struct{}            // agentName -> conns
	groups     map[string]map[string]map[*websocket.Conn]struct{} // agentName -> group -> conns
	rdb        *redis.Client
	replicaID  string
	claimTTL   time.Duration
}

func NewHub(rdb *redis.Client, replicaID string) *Hub {
	return &Hub{
		broadcast: make(map[string]map[*websocket.Conn]struct{}),
		groups:    make(map[string]map[string]map[*websocket.Conn]struct{}),
		rdb:       rdb,
		replicaID: replicaID,
		claimTTL:  30 * time.Second,
	}
}

// SubscribeBroadcast adds a connection that receives every event.
func (h *Hub) SubscribeBroadcast(agentName string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.broadcast[agentName] == nil {
		h.broadcast[agentName] = make(map[*websocket.Conn]struct{})
	}
	h.broadcast[agentName][conn] = struct{}{}
	wsConnectionsActive.WithLabelValues("broadcast").Inc()
}

// SubscribeGroup adds a connection to a named group. Within the group, each
// event is delivered to exactly one client (across all API replicas).
func (h *Hub) SubscribeGroup(agentName, group string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.groups[agentName] == nil {
		h.groups[agentName] = make(map[string]map[*websocket.Conn]struct{})
	}
	if h.groups[agentName][group] == nil {
		h.groups[agentName][group] = make(map[*websocket.Conn]struct{})
	}
	h.groups[agentName][group][conn] = struct{}{}
	wsConnectionsActive.WithLabelValues("group").Inc()
}

// Unsubscribe removes a connection from broadcast and any groups for the agent.
func (h *Hub) Unsubscribe(agentName string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.broadcast[agentName]; ok {
		if _, had := clients[conn]; had {
			delete(clients, conn)
			wsConnectionsActive.WithLabelValues("broadcast").Dec()
			if len(clients) == 0 {
				delete(h.broadcast, agentName)
			}
		}
	}
	if perGroup, ok := h.groups[agentName]; ok {
		for group, clients := range perGroup {
			if _, has := clients[conn]; has {
				delete(clients, conn)
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

// Dispatch sends a message to the broadcast bucket and to one client per group.
// Group routing is coordinated across replicas via SET NX on a per-(group, msgID)
// claim key. The first replica to claim wins and delivers; others skip.
//
// msgID should be a stable identifier for the event (e.g. the Redis stream ID)
// so all replicas race on the same key.
func (h *Hub) Dispatch(ctx context.Context, agentName, msgID string, message []byte) {
	// Snapshot connections under read lock, write outside the lock.
	h.mu.RLock()
	bcast := make([]*websocket.Conn, 0, len(h.broadcast[agentName]))
	for c := range h.broadcast[agentName] {
		bcast = append(bcast, c)
	}
	groupMembers := make(map[string][]*websocket.Conn, len(h.groups[agentName]))
	for group, clients := range h.groups[agentName] {
		conns := make([]*websocket.Conn, 0, len(clients))
		for c := range clients {
			conns = append(conns, c)
		}
		if len(conns) > 0 {
			groupMembers[group] = conns
		}
	}
	h.mu.RUnlock()

	// Broadcast bucket: send to all.
	for _, conn := range bcast {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			wsDispatchTotal.WithLabelValues("broadcast", "write_failed").Inc()
			conn.Close()
			h.Unsubscribe(agentName, conn)
		} else {
			wsDispatchTotal.WithLabelValues("broadcast", "delivered").Inc()
		}
	}

	// Group buckets: per group, claim then deliver to one local member.
	// On write failure, try other local members in the same group before giving up.
	for group, members := range groupMembers {
		claimKey := "wsclaim:" + agentName + ":" + group + ":" + msgID
		ok, err := h.rdb.SetNX(ctx, claimKey, h.replicaID, h.claimTTL).Result()
		if err != nil || !ok {
			wsDispatchTotal.WithLabelValues("group", "claimed_by_other").Inc()
			continue
		}

		// Shuffle a copy so we try members in random order — spreads load and avoids
		// always retrying the same head-of-list candidate.
		order := make([]*websocket.Conn, len(members))
		copy(order, members)
		rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

		delivered := false
		for _, pick := range order {
			if err := pick.WriteMessage(websocket.TextMessage, message); err != nil {
				pick.Close()
				h.Unsubscribe(agentName, pick)
				continue // try next member in this group
			}
			wsDispatchTotal.WithLabelValues("group", "delivered").Inc()
			delivered = true
			break
		}

		if !delivered {
			wsDispatchTotal.WithLabelValues("group", "write_failed").Inc()
			// All local members failed. Release the claim so a future event
			// re-evaluates routing (a different replica may have live members).
			// This event itself is lost for the group; clients reconcile via REST history.
			h.rdb.Del(ctx, claimKey)
		}
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
		defer conn.Close()

		if group != "" {
			hub.SubscribeGroup(agentName, group, conn)
			Logger.Infow("ws client connected", "agent_name", agentName, "group", group)
		} else {
			hub.SubscribeBroadcast(agentName, conn)
			Logger.Infow("ws client connected", "agent_name", agentName, "group", "broadcast")
		}
		defer hub.Unsubscribe(agentName, conn)

		// Ping/pong keepalive: detect silently-dropped clients (browser tab killed,
		// network blip) so they don't leak in the connections gauge forever.
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
				break
			}
		}
	}
}
