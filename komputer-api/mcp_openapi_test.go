package main

import (
	"reflect"
	"strings"
	"testing"
)

// testSwagger is a minimal swagger 2.0 fixture shared by the MCP tests.
const testSwagger = `{
  "basePath": "/api/v1",
  "paths": {
    "/widgets/{name}": {
      "patch": {
        "operationId": "patchWidget",
        "summary": "Patch widget",
        "description": "Updates a widget.",
        "parameters": [
          {"name": "name", "in": "path", "required": true, "type": "string", "description": "Widget name"},
          {"name": "namespace", "in": "query", "type": "string", "description": "Kubernetes namespace"},
          {"name": "limit", "in": "query", "type": "integer", "default": 50},
          {"name": "request", "in": "body", "required": true, "description": "Patch body",
           "schema": {"$ref": "#/definitions/main.PatchWidgetRequest"}}
        ]
      }
    },
    "/widgets/{name}/ws": {
      "get": {"summary": "Stream widget events"}
    }
  },
  "definitions": {
    "main.PatchWidgetRequest": {"type": "object", "properties": {
      "spec": {"$ref": "#/definitions/v1alpha1.WidgetSpec"},
      "pod":  {"$ref": "#/definitions/v1.PodSpec"}
    }},
    "v1alpha1.WidgetSpec": {"type": "object", "properties": {
      "child": {"$ref": "#/definitions/v1alpha1.WidgetSpec"},
      "size":  {"type": "integer"}
    }},
    "v1.PodSpec": {"type": "object", "properties": {"containers": {"type": "array"}}}
  }
}`

func parseTestSwagger(t *testing.T) (string, []mcpOperation) {
	t.Helper()
	basePath, ops, err := parseMCPOperations([]byte(testSwagger))
	if err != nil {
		t.Fatalf("parseMCPOperations: %v", err)
	}
	return basePath, ops
}

func TestParseMCPOperations_SkipsOperationsWithoutID(t *testing.T) {
	basePath, ops := parseTestSwagger(t)
	if basePath != "/api/v1" {
		t.Errorf("basePath = %q, want /api/v1", basePath)
	}
	if len(ops) != 1 {
		t.Fatalf("got %d ops, want 1 (ws has no operationId)", len(ops))
	}
}

func TestParseMCPOperations_OperationShape(t *testing.T) {
	_, ops := parseTestSwagger(t)
	op := ops[0]
	if op.ToolName != "patch_widget" || op.Method != "PATCH" || op.Path != "/widgets/{name}" {
		t.Errorf("got name=%q method=%q path=%q", op.ToolName, op.Method, op.Path)
	}
	if op.Description != "Patch widget\n\nUpdates a widget." {
		t.Errorf("description = %q", op.Description)
	}
	if !reflect.DeepEqual(op.PathParams, []string{"name"}) {
		t.Errorf("PathParams = %v", op.PathParams)
	}
	if !reflect.DeepEqual(op.QueryParams, []string{"namespace", "limit"}) {
		t.Errorf("QueryParams = %v", op.QueryParams)
	}
	if !op.HasBody {
		t.Error("HasBody = false, want true")
	}
}

func TestParseMCPOperations_InputSchema(t *testing.T) {
	_, ops := parseTestSwagger(t)
	s := ops[0].InputSchema
	if s["type"] != "object" {
		t.Fatalf("schema type = %v", s["type"])
	}
	if !reflect.DeepEqual(s["required"], []string{"name", "body"}) {
		t.Errorf("required = %v, want [name body]", s["required"])
	}
	props := s["properties"].(map[string]any)
	name := props["name"].(map[string]any)
	if name["type"] != "string" || name["description"] != "Widget name" {
		t.Errorf("name prop = %v", name)
	}
	if props["limit"].(map[string]any)["default"] != float64(50) {
		t.Errorf("limit default = %v", props["limit"])
	}
	body := props["body"].(map[string]any)
	if body["$ref"] != "#/$defs/main.PatchWidgetRequest" || body["description"] != "Patch body" {
		t.Errorf("body prop = %v", body)
	}

	defs := s["$defs"].(map[string]any)
	if len(defs) != 3 {
		t.Fatalf("got %d $defs, want 3: %v", len(defs), defs)
	}
	req := defs["main.PatchWidgetRequest"].(map[string]any)
	spec := req["properties"].(map[string]any)["spec"].(map[string]any)
	if spec["$ref"] != "#/$defs/v1alpha1.WidgetSpec" {
		t.Errorf("nested ref not rewritten: %v", spec)
	}
	// Self-referencing komputer type is expanded once (cycle-safe).
	ws := defs["v1alpha1.WidgetSpec"].(map[string]any)
	child := ws["properties"].(map[string]any)["child"].(map[string]any)
	if child["$ref"] != "#/$defs/v1alpha1.WidgetSpec" {
		t.Errorf("cyclic ref = %v", child)
	}
	// Upstream Kubernetes types are collapsed to opaque objects.
	pod := defs["v1.PodSpec"].(map[string]any)
	if _, has := pod["properties"]; has || pod["type"] != "object" ||
		!strings.Contains(pod["description"].(string), "v1.PodSpec") {
		t.Errorf("v1.PodSpec not collapsed: %v", pod)
	}
}

func TestParseMCPOperations_UndefinedRef(t *testing.T) {
	spec := `{"basePath":"/api/v1","paths":{"/x":{"post":{"operationId":"createX","parameters":[
	  {"name":"request","in":"body","schema":{"$ref":"#/definitions/main.Missing"}}]}}},"definitions":{}}`
	if _, _, err := parseMCPOperations([]byte(spec)); err == nil || !strings.Contains(err.Error(), "main.Missing") {
		t.Fatalf("err = %v, want undefined ref error", err)
	}
}

func TestParseMCPOperations_UnsupportedParamLocation(t *testing.T) {
	spec := `{"basePath":"/api/v1","paths":{"/x":{"post":{"operationId":"uploadX","parameters":[
	  {"name":"file","in":"formData","type":"file"}]}}},"definitions":{}}`
	if _, _, err := parseMCPOperations([]byte(spec)); err == nil || !strings.Contains(err.Error(), "formData") {
		t.Fatalf("err = %v, want unsupported location error", err)
	}
}

// TestBuildMCPOperation_ToolNameOverride guards legacy MCP tool names: a few
// operationIds would snake_case to a different name than the original
// hand-written MCP server used, so mcpToolNameOverrides keeps them stable.
func TestBuildMCPOperation_ToolNameOverride(t *testing.T) {
	op := swaggerOperation{OperationID: "compactAgentTask", Summary: "Compact agent conversation"}
	m, err := buildMCPOperation("/agents/{name}/compact", "post", op, nil)
	if err != nil {
		t.Fatalf("buildMCPOperation: %v", err)
	}
	if m.ToolName != "compact_agent" {
		t.Errorf("ToolName = %q, want %q (override for compactAgentTask)", m.ToolName, "compact_agent")
	}
}

func TestToSnakeCase(t *testing.T) {
	for in, want := range map[string]string{
		"createAgent":           "create_agent",
		"getAgentCostBreakdown": "get_agent_cost_breakdown",
		"breakUpSquad":          "break_up_squad",
		"listMCPTools":          "list_mcp_tools",
		"list":                  "list",
	} {
		if got := toSnakeCase(in); got != want {
			t.Errorf("toSnakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}
