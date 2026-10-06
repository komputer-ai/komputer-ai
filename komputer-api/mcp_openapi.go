package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// mcpOperation is one REST operation exposed as an MCP tool. It is derived from
// the swagger spec, so the MCP surface always mirrors the REST API.
type mcpOperation struct {
	ToolName    string
	Method      string // upper-case HTTP method
	Path        string // swagger path template without basePath, e.g. /agents/{name}
	Description string
	PathParams  []string
	QueryParams []string
	HasBody     bool
	InputSchema map[string]any
}

type swaggerSpec struct {
	BasePath    string                                 `json:"basePath"`
	Paths       map[string]map[string]swaggerOperation `json:"paths"`
	Definitions map[string]json.RawMessage             `json:"definitions"`
}

type swaggerOperation struct {
	OperationID string             `json:"operationId"`
	Summary     string             `json:"summary"`
	Description string             `json:"description"`
	Parameters  []swaggerParameter `json:"parameters"`
}

type swaggerParameter struct {
	Name        string          `json:"name"`
	In          string          `json:"in"`
	Description string          `json:"description"`
	Required    bool            `json:"required"`
	Type        string          `json:"type"`
	Items       json.RawMessage `json:"items"`
	Enum        []any           `json:"enum"`
	Default     any             `json:"default"`
	Schema      json.RawMessage `json:"schema"`
}

var swaggerRefRe = regexp.MustCompile(`"#/definitions/([^"]+)"`)

// mcpToolNameOverrides keeps tool names from the original hand-written MCP
// server stable where they differ from snake_case(operationId).
var mcpToolNameOverrides = map[string]string{
	"compactAgentTask": "compact_agent",
	"namespacesGet":    "list_namespaces",
}

// parseMCPOperations turns a swagger 2.0 spec into one mcpOperation per
// operation that has an operationId. Operations without one are skipped.
func parseMCPOperations(specJSON []byte) (string, []mcpOperation, error) {
	var spec swaggerSpec
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return "", nil, fmt.Errorf("parse swagger spec: %w", err)
	}
	var ops []mcpOperation
	for path, methods := range spec.Paths {
		for method, op := range methods {
			if op.OperationID == "" {
				continue
			}
			mop, err := buildMCPOperation(path, method, op, spec.Definitions)
			if err != nil {
				return "", nil, fmt.Errorf("%s: %w", op.OperationID, err)
			}
			ops = append(ops, mop)
		}
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].ToolName < ops[j].ToolName })
	return spec.BasePath, ops, nil
}

func buildMCPOperation(path, method string, op swaggerOperation, defs map[string]json.RawMessage) (mcpOperation, error) {
	toolName := mcpToolNameOverrides[op.OperationID]
	if toolName == "" {
		toolName = toSnakeCase(op.OperationID)
	}
	m := mcpOperation{
		ToolName:    toolName,
		Method:      strings.ToUpper(method),
		Path:        path,
		Description: strings.TrimSpace(op.Summary + "\n\n" + op.Description),
	}
	props := map[string]any{}
	var required []string
	var bodySchema json.RawMessage
	for _, p := range op.Parameters {
		switch p.In {
		case "path", "query":
			prop := map[string]any{"type": p.Type}
			if p.Description != "" {
				prop["description"] = p.Description
			}
			if len(p.Items) > 0 {
				prop["items"] = p.Items
			}
			if len(p.Enum) > 0 {
				prop["enum"] = p.Enum
			}
			if p.Default != nil {
				prop["default"] = p.Default
			}
			props[p.Name] = prop
			if p.In == "path" {
				m.PathParams = append(m.PathParams, p.Name)
				required = append(required, p.Name)
			} else {
				m.QueryParams = append(m.QueryParams, p.Name)
				if p.Required {
					required = append(required, p.Name)
				}
			}
		case "body":
			var schema map[string]any
			if err := json.Unmarshal(rewriteSwaggerRefs(p.Schema), &schema); err != nil {
				return m, fmt.Errorf("body schema: %w", err)
			}
			if p.Description != "" {
				schema["description"] = p.Description
			}
			props["body"] = schema
			if p.Required {
				required = append(required, "body")
			}
			m.HasBody = true
			bodySchema = p.Schema
		default:
			return m, fmt.Errorf("unsupported parameter location %q for %q", p.In, p.Name)
		}
	}
	m.InputSchema = map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		m.InputSchema["required"] = required
	}
	if bodySchema != nil {
		d, err := collectSchemaDefs(bodySchema, defs)
		if err != nil {
			return m, err
		}
		if len(d) > 0 {
			m.InputSchema["$defs"] = d
		}
	}
	return m, nil
}

// collectSchemaDefs returns every definition reachable from schema, keyed for
// "#/$defs/<name>". komputer's own types (main.*, v1alpha1.*) are expanded;
// upstream Kubernetes types (PodSpec etc.) are collapsed to opaque objects —
// fully expanded they add ~190KB to each tool schema.
func collectSchemaDefs(schema json.RawMessage, defs map[string]json.RawMessage) (map[string]any, error) {
	out := map[string]any{}
	var visit func(raw []byte) error
	visit = func(raw []byte) error {
		for _, match := range swaggerRefRe.FindAllSubmatch(raw, -1) {
			name := string(match[1])
			if _, seen := out[name]; seen {
				continue
			}
			if !strings.HasPrefix(name, "main.") && !strings.HasPrefix(name, "v1alpha1.") {
				out[name] = map[string]any{
					"type":        "object",
					"description": "Kubernetes " + name + " object (see the Kubernetes API reference).",
				}
				continue
			}
			def, ok := defs[name]
			if !ok {
				return fmt.Errorf("undefined schema ref %q", name)
			}
			var v map[string]any
			if err := json.Unmarshal(rewriteSwaggerRefs(def), &v); err != nil {
				return fmt.Errorf("definition %q: %w", name, err)
			}
			out[name] = v
			if err := visit(def); err != nil {
				return err
			}
		}
		return nil
	}
	return out, visit(schema)
}

func rewriteSwaggerRefs(raw []byte) []byte {
	return bytes.ReplaceAll(raw, []byte(`"#/definitions/`), []byte(`"#/$defs/`))
}

// toSnakeCase converts an operationId like getAgentCostBreakdown (or listMCPTools)
// to get_agent_cost_breakdown (list_mcp_tools).
func toSnakeCase(s string) string {
	r := []rune(s)
	var b strings.Builder
	for i, c := range r {
		if unicode.IsUpper(c) {
			if i > 0 && (unicode.IsLower(r[i-1]) || unicode.IsDigit(r[i-1]) ||
				(unicode.IsUpper(r[i-1]) && i+1 < len(r) && unicode.IsLower(r[i+1]))) {
				b.WriteByte('_')
			}
			c = unicode.ToLower(c)
		}
		b.WriteRune(c)
	}
	return b.String()
}
