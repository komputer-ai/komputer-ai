package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Hub manages WebSocket connections per agent name.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[*websocket.Conn]struct{} // agentName -> set of connections
}

func NewHub() *Hub {
	return &Hub{
		conns: make(map[string]map[*websocket.Conn]struct{}),
	}
}

// Subscribe adds a connection for the given agent.
func (h *Hub) Subscribe(agentName string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[agentName] == nil {
		h.conns[agentName] = make(map[*websocket.Conn]struct{})
	}
	h.conns[agentName][conn] = struct{}{}
}

// Unsubscribe removes a connection.
func (h *Hub) Unsubscribe(agentName string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.conns[agentName]; ok {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(h.conns, agentName)
		}
	}
}

// Broadcast sends a raw JSON message to all connections for the given agent.
func (h *Hub) Broadcast(agentName string, message []byte) {
	h.mu.RLock()
	clients := h.conns[agentName]
	h.mu.RUnlock()

	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("ws write error for %s: %v", agentName, err)
			conn.Close()
			h.Unsubscribe(agentName, conn)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleAgentWS handles GET /api/v1/agents/:name/ws
// @Summary Stream agent events (WebSocket)
// @Description Upgrades to a WebSocket connection to stream real-time agent events. Events include task_started, thinking, tool_call, tool_result, text, task_completed, task_cancelled, and error.
// @Tags agents
// @Param name path string true "Agent name"
// @Router /agents/{name}/ws [get]
func HandleAgentWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		agentName := c.Param("name")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}
		defer conn.Close()

		hub.Subscribe(agentName, conn)
		defer hub.Unsubscribe(agentName, conn)

		log.Printf("ws client connected for agent %s", agentName)

		// Block until the client disconnects
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}
}
