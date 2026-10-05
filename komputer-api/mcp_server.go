package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/komputer-ai/komputer-api/docs"
)

// newMCPServer builds the komputer-ai MCP server. Every tool is generated from
// the swagger spec and dispatched in-process to the REST handler it describes,
// so MCP exposes exactly the REST API with no logic of its own.
func newMCPServer(h http.Handler, basePath string, ops []mcpOperation) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "komputer-ai",
		Version: "v1",
	}, nil)
	for _, op := range ops {
		srv.AddTool(&mcp.Tool{
			Name:        op.ToolName,
			Description: op.Description,
			InputSchema: op.InputSchema,
		}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return callRESTOperation(ctx, h, basePath, op, req.Params.Arguments), nil
		})
	}
	return srv
}

// mountMCPHandler installs the MCP streamable HTTP endpoint on the Gin router.
// External agents connect by adding a custom MCP connector pointing at <api-url>/mcp.
// No auth: matches the existing API posture; network access controls are assumed.
func mountMCPHandler(r *gin.Engine) {
	basePath, ops, err := parseMCPOperations([]byte(docs.SwaggerInfo.ReadDoc()))
	if err != nil {
		panic(fmt.Errorf("build MCP tools from swagger spec: %w", err))
	}
	srv := newMCPServer(r, basePath, ops)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, nil)
	r.Any("/mcp", gin.WrapH(handler))
	r.Any("/mcp/*any", gin.WrapH(handler))
}

// callRESTOperation runs one tool call through the REST handler and maps the
// HTTP response to a tool result. Handler errors become IsError results so the
// calling model sees the real status and message.
func callRESTOperation(ctx context.Context, h http.Handler, basePath string, op mcpOperation, rawArgs json.RawMessage) *mcp.CallToolResult {
	req, err := op.buildRequest(ctx, basePath, rawArgs)
	if err != nil {
		return toolError(err.Error())
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	if rec.Code < 200 || rec.Code > 299 {
		return toolError(fmt.Sprintf("HTTP %d: %s", rec.Code, body))
	}
	if body == "" {
		body = fmt.Sprintf("HTTP %d", rec.Code)
	}
	res := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: body}}}
	var obj map[string]any
	if json.Unmarshal(rec.Body.Bytes(), &obj) == nil {
		res.StructuredContent = obj
	}
	return res
}

func toolError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: msg}}}
}

// buildRequest maps tool arguments onto the operation's path, query string and
// JSON body. Unknown arguments are rejected so typos don't silently no-op.
func (op mcpOperation) buildRequest(ctx context.Context, basePath string, rawArgs json.RawMessage) (*http.Request, error) {
	args := map[string]json.RawMessage{}
	if len(rawArgs) > 0 && string(rawArgs) != "null" {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return nil, fmt.Errorf("arguments must be a JSON object: %w", err)
		}
	}

	allowed := append(append([]string{}, op.PathParams...), op.QueryParams...)
	if op.HasBody {
		allowed = append(allowed, "body")
	}
	for k := range args {
		if !slices.Contains(allowed, k) {
			sort.Strings(allowed)
			return nil, fmt.Errorf("unknown argument %q; valid arguments: %s", k, strings.Join(allowed, ", "))
		}
	}

	path := op.Path
	for _, p := range op.PathParams {
		vals, err := argValues(args[p])
		if err != nil || len(vals) != 1 || vals[0] == "" {
			return nil, fmt.Errorf("missing required argument %q", p)
		}
		path = strings.ReplaceAll(path, "{"+p+"}", url.PathEscape(vals[0]))
	}

	query := url.Values{}
	for _, p := range op.QueryParams {
		raw, ok := args[p]
		if !ok || string(raw) == "null" {
			continue
		}
		vals, err := argValues(raw)
		if err != nil {
			return nil, fmt.Errorf("argument %q: %w", p, err)
		}
		query[p] = vals
	}

	var body io.Reader
	if raw, ok := args["body"]; ok && op.HasBody && string(raw) != "null" {
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, op.Method, basePath+path, body)
	if err != nil {
		return nil, err
	}
	if req.Body == nil {
		// http.NewRequestWithContext leaves Body nil when no body is given.
		// That's fine for a real client request, but handlers dispatched
		// in-process via ServeHTTP (bypassing the usual server accept path)
		// may read the body directly and panic on a true nil.
		req.Body = http.NoBody
	}
	req.URL.RawQuery = query.Encode()
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// argValues renders a JSON argument as path/query strings: strings unquoted,
// numbers and booleans verbatim, arrays as one value per element.
func argValues(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		var out []string
		for _, e := range arr {
			v, err := argValues(e)
			if err != nil {
				return nil, err
			}
			out = append(out, v...)
		}
		return out, nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []string{s}, nil
	}
	t := strings.TrimSpace(string(raw))
	if strings.HasPrefix(t, "{") {
		return nil, fmt.Errorf("object values are not allowed here")
	}
	return []string{t}, nil
}
