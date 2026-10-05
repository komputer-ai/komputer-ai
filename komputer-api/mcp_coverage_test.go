package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/komputer-ai/komputer-api/docs"
)

// mcpExcludedRoutes are REST routes deliberately not exposed as MCP tools.
var mcpExcludedRoutes = map[string]string{
	"GET /api/v1/agents/:name/ws":                 "WebSocket stream; poll get_agent_events instead",
	"GET /api/v1/agents/:name/download/*filepath": "binary file download",
	"POST /api/v1/oauth/authorize":                "browser OAuth flow",
	"GET /api/v1/oauth/callback":                  "browser OAuth flow",
	"POST /api/v1/oauth/refresh":                  "internal OAuth token refresh",
}

var swaggerPathParamRe = regexp.MustCompile(`\{([^}]+)\}`)

func realMCPOperations(t *testing.T) (string, []mcpOperation) {
	t.Helper()
	basePath, ops, err := parseMCPOperations([]byte(docs.SwaggerInfo.ReadDoc()))
	if err != nil {
		t.Fatalf("parse embedded swagger: %v", err)
	}
	return basePath, ops
}

// TestMCPToolsCoverEveryAPIRoute fails when a REST route is added without a
// swagger @ID (so it would silently be missing from /mcp).
func TestMCPToolsCoverEveryAPIRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	basePath, ops := realMCPOperations(t)
	exposed := map[string]bool{}
	for _, op := range ops {
		exposed[op.Method+" "+basePath+swaggerPathParamRe.ReplaceAllString(op.Path, ":$1")] = true
	}

	r := gin.New()
	SetupRoutes(r, nil, nil, nil)
	for _, rt := range r.Routes() {
		if !strings.HasPrefix(rt.Path, "/api/v1/") {
			continue
		}
		key := rt.Method + " " + rt.Path
		if _, excluded := mcpExcludedRoutes[key]; excluded {
			continue
		}
		if !exposed[key] {
			t.Errorf("%s has no MCP tool: add @ID/@Router swagger annotations to its handler and run swag init, or add it to mcpExcludedRoutes", key)
		}
	}
}

func TestMCPRealSpec_CoreToolsPresent(t *testing.T) {
	_, ops := realMCPOperations(t)
	names := map[string]bool{}
	for _, op := range ops {
		names[op.ToolName] = true
	}
	for _, want := range []string{
		"create_agent", "get_agent", "patch_agent", "delete_agent", "cancel_agent_task",
		"get_agent_events", "get_agent_cost_breakdown", "create_schedule", "trigger_schedule",
		"create_memory", "create_skill", "create_connector", "list_connector_templates",
		"create_secret", "create_squad", "list_namespaces", "list_templates",
	} {
		if !names[want] {
			t.Errorf("missing MCP tool %q", want)
		}
	}
}

// TestMCPRealSpec_SchemaSizeBudget guards against a request type pulling the
// full Kubernetes type tree into tool schemas (~190KB per tool if expanded).
func TestMCPRealSpec_SchemaSizeBudget(t *testing.T) {
	_, ops := realMCPOperations(t)
	total := 0
	for _, op := range ops {
		b, err := json.Marshal(op.InputSchema)
		if err != nil {
			t.Fatalf("%s: marshal schema: %v", op.ToolName, err)
		}
		total += len(b)
	}
	if total > 100_000 {
		t.Errorf("tool schemas total %d bytes, budget 100000", total)
	}
}
