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
	return cs, &seen
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
