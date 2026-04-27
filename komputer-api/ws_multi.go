package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HandleMultiAgentWS handles GET /api/v1/agents/events/ws
//
// Query parameters:
//
//   - match — comma-separated glob patterns (e.g. "worker-*,manager-*").
//     Glob syntax is path.Match (?, *, [a-z]).
//   - agents — comma-separated explicit agent names (e.g. "alice,bob").
//   - namespace — optional namespace filter. If omitted, events from any
//     namespace are delivered.
//
// At least one of match or agents must be provided. The two are unioned.
//
// Each delivered message is a JSON-encoded AgentEvent including agentName
// and namespace, so the client can demultiplex.
//
// @Summary Stream events from multiple agents (WebSocket)
// @Tags agents
// @Param match query string false "Comma-separated glob patterns"
// @Param agents query string false "Comma-separated agent names"
// @Param namespace query string false "Optional namespace filter"
// @Router /agents/events/ws [get]
func HandleMultiAgentWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		patterns := splitCSV(c.Query("match"))
		agents := splitCSV(c.Query("agents"))
		namespace := c.Query("namespace")

		matcher, err := compileMatcher(patterns, agents, namespace)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			Logger.Errorw("ws upgrade failed", "error", err)
			return
		}

		q := newSendQueueWithMode(conn, sendQueueCapacity, "match")
		defer q.Close()

		hub.SubscribeMatch(q, matcher)
		defer hub.UnsubscribeMatch(q)

		Logger.Infow("ws multi client connected",
			"patterns", patterns,
			"agents", agents,
			"namespace", namespace,
		)

		runReadLoop(conn)
	}
}

// splitCSV trims and splits a comma-separated string, dropping empties.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
