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
	realRoutes := map[string]bool{}
	for _, rt := range r.Routes() {
		if !strings.HasPrefix(rt.Path, "/api/v1/") {
			continue
		}
		realRoutes[rt.Method+" "+rt.Path] = true
		key := rt.Method + " " + rt.Path
		if _, excluded := mcpExcludedRoutes[key]; excluded {
			continue
		}
		if !exposed[key] {
			t.Errorf("%s has no MCP tool: add @ID/@Router swagger annotations to its handler and run swag init, or add it to mcpExcludedRoutes", key)
		}
	}

	// Reverse direction: every exposed tool must map to a real route, or a
	// stale/typo'd @Router annotation would produce a tool that always 404s.
	for _, op := range ops {
		key := op.Method + " " + basePath + swaggerPathParamRe.ReplaceAllString(op.Path, ":$1")
		if !realRoutes[key] {
			t.Errorf("tool %q maps to %q which is not a real route: fix its @Router annotation / re-run swag", op.ToolName, key)
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

// TestMCPRealSpec_LegacyToolNamesStable guards the 14 MCP tool names that
// existed before this branch's REST-API-wide MCP expansion: none of them may
// be renamed by operationId churn, and no renamed duplicate (e.g.
// compact_agent_task) may appear alongside the original.
func TestMCPRealSpec_LegacyToolNamesStable(t *testing.T) {
	_, ops := realMCPOperations(t)
	names := map[string]bool{}
	for _, op := range ops {
		names[op.ToolName] = true
	}
	for _, want := range []string{
		"list_agents", "get_agent", "compact_agent", "list_schedules", "get_schedule",
		"trigger_schedule", "list_memories", "get_memory", "list_skills", "get_skill",
		"list_connectors", "list_secrets", "list_namespaces", "list_templates",
	} {
		if !names[want] {
			t.Errorf("missing legacy MCP tool %q", want)
		}
	}
	if names["compact_agent_task"] {
		t.Error("tool \"compact_agent_task\" should not exist: it is a rename of the legacy compact_agent tool")
	}
}

// TestMCPRealSpec_QueryParamsExposed guards against a handler reading a query
// param that its swagger annotations don't declare: such a param is invisible
// to MCP clients and would be rejected as an unknown argument.
func TestMCPRealSpec_QueryParamsExposed(t *testing.T) {
	_, ops := realMCPOperations(t)
	byName := map[string]mcpOperation{}
	for _, op := range ops {
		byName[op.ToolName] = op
	}

	hasParam := func(params []string, want string) bool {
		for _, p := range params {
			if p == want {
				return true
			}
		}
		return false
	}

	cases := []struct {
		tool   string
		params []string
	}{
		{"get_agent_events", []string{"before", "after", "around"}},
		{"delete_agent", []string{"recreatePod"}},
		{"list_agents", []string{"status", "label"}},
	}
	for _, tc := range cases {
		op, ok := byName[tc.tool]
		if !ok {
			t.Fatalf("tool %q not found", tc.tool)
		}
		for _, want := range tc.params {
			if !hasParam(op.QueryParams, want) {
				t.Errorf("%s: missing query param %q (QueryParams=%v)", tc.tool, want, op.QueryParams)
			}
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
