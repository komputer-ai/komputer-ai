package main

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Bearer-token authentication for the API.
//
// Off unless KOMPUTER_API_TOKEN is set, so existing installs keep working.
// When on, every request needs "Authorization: Bearer <token>", except:
//   - health and metrics endpoints (probes and Prometheus scrapes)
//   - the OAuth callback (a browser redirect from the provider, protected by its state parameter)
//   - CORS preflights
//   - requests the MCP server dispatches in-process (the outer /mcp call was already checked)
//
// Browsers can't set headers on a WebSocket, so the agent WebSocket route also
// accepts the token as an "access_token" query parameter.

type inProcessKey struct{}

// withInProcess marks a request as dispatched in-process by the MCP server.
// The marker lives on the context, so a remote client can't set it.
func withInProcess(ctx context.Context) context.Context {
	return context.WithValue(ctx, inProcessKey{}, true)
}

func isInProcess(r *http.Request) bool {
	v, _ := r.Context().Value(inProcessKey{}).(bool)
	return v
}

var authExemptPaths = map[string]struct{}{
	"/healthz":               {},
	"/readyz":                {},
	"/api/metrics":           {},
	"/agent/metrics":         {},
	"/api/v1/oauth/callback": {},
}

func apiTokenFromEnv() string {
	return strings.TrimSpace(os.Getenv("KOMPUTER_API_TOKEN"))
}

// authMiddleware returns a no-op when token is empty.
func authMiddleware(token string) gin.HandlerFunc {
	if token == "" {
		return func(c *gin.Context) { c.Next() }
	}
	want := []byte(token)
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions || isInProcess(c.Request) {
			c.Next()
			return
		}
		if _, ok := authExemptPaths[c.Request.URL.Path]; ok {
			c.Next()
			return
		}
		got := bearerToken(c.Request)
		if got == "" && isWebSocketUpgrade(c.Request) {
			got = c.Query("access_token")
		}
		if got == "" || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			c.Header("WWW-Authenticate", `Bearer realm="komputer"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// corsMiddleware allows any origin when allowed is empty (the historical
// behavior). With a list, only those origins get CORS headers.
func corsMiddleware(allowed []string) gin.HandlerFunc {
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		set[strings.TrimRight(o, "/")] = struct{}{}
	}
	return func(c *gin.Context) {
		if len(set) == 0 {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			c.Header("Vary", "Origin")
			if origin := c.GetHeader("Origin"); origin != "" {
				if _, ok := set[origin]; ok {
					c.Header("Access-Control-Allow-Origin", origin)
				}
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// corsOriginsFromEnv parses CORS_ALLOWED_ORIGINS (comma-separated).
func corsOriginsFromEnv() []string {
	var out []string
	for _, o := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" && o != "*" {
			out = append(out, o)
		}
	}
	return out
}
