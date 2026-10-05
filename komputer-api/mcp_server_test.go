package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type capturedRequest struct {
	Method, Name, RawQuery, ContentType, Body string
}

// newTestMCPSession wires the testSwagger tools to a gin engine with a fake
// PATCH /api/v1/widgets/:name handler and returns a connected client session.
func newTestMCPSession(t *testing.T) (*mcp.ClientSession, *[]capturedRequest) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var seen []capturedRequest
	r := gin.New()
	r.PATCH("/api/v1/widgets/:name", func(c *gin.Context) {
		b, _ := io.ReadAll(c.Request.Body)
		seen = append(seen, capturedRequest{
			Method: c.Request.Method, Name: c.Param("name"), RawQuery: c.Request.URL.RawQuery,
			ContentType: c.GetHeader("Content-Type"), Body: string(b),
		})
		if c.Param("name") == "nope" {
			c.JSON(http.StatusNotFound, gin.H{"error": "widget not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "name": c.Param("name")})
	})

	basePath, ops := parseTestSwagger(t)
	return connectMCPSession(t, r, basePath, ops), &seen
}

// connectMCPSession wires ops onto r and returns a connected client session.
// Shared by newTestMCPSession and tests that need a custom router (e.g. one
// without gin.Recovery, to exercise callRESTOperation's own panic recovery).
func connectMCPSession(t *testing.T, r *gin.Engine, basePath string, ops []mcpOperation) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	clientT, serverT := mcp.NewInMemoryTransports()
	if _, err := newMCPServer(r, basePath, ops).Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func callTool(t *testing.T, cs *mcp.ClientSession, args map[string]any) (*mcp.CallToolResult, string) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "patch_widget", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool protocol error: %v", err)
	}
	return res, res.Content[0].(*mcp.TextContent).Text
}

func TestMCPListTools(t *testing.T) {
	cs, _ := newTestMCPSession(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) != 1 || res.Tools[0].Name != "patch_widget" {
		t.Fatalf("tools = %+v", res.Tools)
	}
	if res.Tools[0].Description != "Patch widget\n\nUpdates a widget." {
		t.Errorf("description = %q", res.Tools[0].Description)
	}
	// PATCH is neither read-only nor (by convention here) destructive.
	ann := res.Tools[0].Annotations
	if ann == nil || ann.ReadOnlyHint {
		t.Errorf("Annotations = %+v, want non-nil with ReadOnlyHint false", ann)
	}
	if ann != nil && ann.DestructiveHint != nil {
		t.Errorf("DestructiveHint = %v, want nil for a non-DELETE operation", *ann.DestructiveHint)
	}
}

func TestToolAnnotations(t *testing.T) {
	cases := []struct {
		method          string
		wantReadOnly    bool
		wantDestructive bool // only checked when non-nil DestructiveHint is expected
	}{
		{"GET", true, false},
		{"POST", false, false},
		{"PATCH", false, false},
		{"PUT", false, false},
		{"DELETE", false, true},
	}
	for _, tc := range cases {
		ann := toolAnnotations(tc.method)
		if ann.ReadOnlyHint != tc.wantReadOnly {
			t.Errorf("toolAnnotations(%q).ReadOnlyHint = %v, want %v", tc.method, ann.ReadOnlyHint, tc.wantReadOnly)
		}
		gotDestructive := ann.DestructiveHint != nil && *ann.DestructiveHint
		if gotDestructive != tc.wantDestructive {
			t.Errorf("toolAnnotations(%q).DestructiveHint = %v, want %v", tc.method, ann.DestructiveHint, tc.wantDestructive)
		}
	}
}

func TestMCPCall_DispatchesToHandler(t *testing.T) {
	cs, seen := newTestMCPSession(t)
	res, text := callTool(t, cs, map[string]any{
		"name": "w1", "namespace": "team-a", "limit": 5, "body": map[string]any{"size": 3},
	})
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", text)
	}
	if len(*seen) != 1 {
		t.Fatalf("handler called %d times", len(*seen))
	}
	got := (*seen)[0]
	want := capturedRequest{Method: "PATCH", Name: "w1", RawQuery: "limit=5&namespace=team-a",
		ContentType: "application/json", Body: `{"size":3}`}
	if got != want {
		t.Errorf("handler saw %+v, want %+v", got, want)
	}
	sc, ok := res.StructuredContent.(map[string]any)
	if !ok || sc["ok"] != true || sc["name"] != "w1" {
		t.Errorf("StructuredContent = %#v", res.StructuredContent)
	}
}

func TestMCPCall_HandlerErrorIsToolError(t *testing.T) {
	cs, _ := newTestMCPSession(t)
	res, text := callTool(t, cs, map[string]any{"name": "nope"})
	if !res.IsError {
		t.Fatal("IsError = false, want true")
	}
	if !strings.Contains(text, "HTTP 404") || !strings.Contains(text, "widget not found") {
		t.Errorf("text = %q", text)
	}
}

// TestMCPCall_HandlerPanicRecovered uses a router WITHOUT gin.Recovery to
// prove callRESTOperation recovers from a handler panic itself. In
// production, main.go installs gin.Recovery before SetupRoutes, but the MCP
// tool handler runs in a go-sdk goroutine where an unrecovered panic would
// kill the process regardless of gin's own middleware.
func TestMCPCall_HandlerPanicRecovered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New() // deliberately no gin.Recovery()
	r.PATCH("/api/v1/widgets/:name", func(c *gin.Context) {
		panic("boom")
	})
	basePath, ops := parseTestSwagger(t)
	cs := connectMCPSession(t, r, basePath, ops)

	res, text := callTool(t, cs, map[string]any{"name": "w1"})
	if !res.IsError {
		t.Fatal("IsError = false, want true")
	}
	if !strings.Contains(text, "internal error") || !strings.Contains(text, "boom") {
		t.Errorf("text = %q, want it to contain %q and %q", text, "internal error", "boom")
	}
}

func TestMCPCall_MissingPathParam(t *testing.T) {
	cs, seen := newTestMCPSession(t)
	res, text := callTool(t, cs, map[string]any{"namespace": "x"})
	if !res.IsError || !strings.Contains(text, `missing required argument "name"`) {
		t.Errorf("IsError=%v text=%q", res.IsError, text)
	}
	if len(*seen) != 0 {
		t.Error("handler should not be called")
	}
}

func TestMCPCall_UnknownArgument(t *testing.T) {
	cs, seen := newTestMCPSession(t)
	res, text := callTool(t, cs, map[string]any{"name": "w1", "nmespace": "x"})
	if !res.IsError || !strings.Contains(text, `unknown argument "nmespace"`) || !strings.Contains(text, "namespace") {
		t.Errorf("IsError=%v text=%q", res.IsError, text)
	}
	if len(*seen) != 0 {
		t.Error("handler should not be called")
	}
}

func TestArgValues(t *testing.T) {
	for raw, want := range map[string][]string{
		`"team-a"`:  {"team-a"},
		`5`:         {"5"},
		`true`:      {"true"},
		`["a","b"]`: {"a", "b"},
		`[1, 2]`:    {"1", "2"},
	} {
		got, err := argValues(json.RawMessage(raw))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("argValues(%s) = %v, %v; want %v", raw, got, err, want)
		}
	}
	if _, err := argValues(json.RawMessage(`{"a":1}`)); err == nil {
		t.Error("argValues(object) should error")
	}
}
