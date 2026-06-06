package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newMCPServer builds the komputer-ai MCP server: exposes the highest-leverage
// API capabilities to external Claude / Claude-Code / MCP-aware agents as tools.
// Each tool wraps an existing handler-layer operation via the shared K8sClient.
func newMCPServer(k8s *K8sClient) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "komputer-ai",
		Version: "v1",
	}, nil)

	registerAgentMCPTools(srv, k8s)
	registerScheduleMCPTools(srv, k8s)
	registerMemoryMCPTools(srv, k8s)
	registerSkillMCPTools(srv, k8s)
	registerConnectorMCPTools(srv, k8s)
	registerSecretMCPTools(srv, k8s)
	registerInfraMCPTools(srv, k8s)

	return srv
}

// mountMCPHandler installs the MCP streamable HTTP endpoint on the Gin router.
// External agents connect by adding a custom MCP connector pointing at <api-url>/mcp.
// No auth: matches the existing API posture; network access controls are assumed.
func mountMCPHandler(r *gin.Engine, k8s *K8sClient) {
	srv := newMCPServer(k8s)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, nil)
	r.Any("/mcp", gin.WrapH(handler))
	r.Any("/mcp/*any", gin.WrapH(handler))
}

// ─── Agents ───────────────────────────────────────────────────────────────

type listAgentsArgs struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace; empty = all namespaces"`
}

type getAgentArgs struct {
	Name      string `json:"name" jsonschema:"Agent name"`
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace"`
}

type compactAgentArgs struct {
	Name         string `json:"name" jsonschema:"Agent name"`
	Namespace    string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace"`
	Instructions string `json:"instructions,omitempty" jsonschema:"Optional guidance to the compactor about what to preserve."`
}

func registerAgentMCPTools(srv *mcp.Server, k8s *K8sClient) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_agents",
		Description: "List KomputerAgent resources in a namespace (or all namespaces if omitted). Returns name, phase, model, and task status for each.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listAgentsArgs) (*mcp.CallToolResult, any, error) {
		agents, err := k8s.ListAgents(ctx, args.Namespace, nil)
		if err != nil {
			return nil, nil, err
		}
		out := make([]map[string]any, 0, len(agents))
		for _, a := range agents {
			out = append(out, map[string]any{
				"name":       a.Name,
				"namespace":  a.Namespace,
				"phase":      string(a.Status.Phase),
				"taskStatus": string(a.Status.TaskStatus),
				"model":      a.Spec.Model,
				"lifecycle":  string(a.Spec.Lifecycle),
			})
		}
		return nil, map[string]any{"agents": out}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_agent",
		Description: "Get full details for a single KomputerAgent: spec, status, current task message, and recent costs.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args getAgentArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		a, err := k8s.GetAgent(ctx, ns, args.Name)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"name":            a.Name,
			"namespace":       a.Namespace,
			"phase":           string(a.Status.Phase),
			"taskStatus":      string(a.Status.TaskStatus),
			"lastTaskMessage": a.Status.LastTaskMessage,
			"model":           a.Spec.Model,
			"lifecycle":       string(a.Spec.Lifecycle),
			"instructions":    a.Spec.Instructions,
			"memories":        a.Spec.Memories,
			"skills":          a.Spec.Skills,
			"connectors":      a.Spec.Connectors,
			"totalCostUSD":    a.Status.TotalCostUSD,
			"lastTaskCostUSD": a.Status.LastTaskCostUSD,
		}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "compact_agent",
		Description: "Trigger manual conversation compaction on an agent's active task. Older turns are summarized to free context space. Only works while the agent is actively running a task — returns an error otherwise.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args compactAgentArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		a, err := k8s.GetAgent(ctx, ns, args.Name)
		if err != nil {
			return nil, nil, err
		}
		if a.Status.PodName == "" {
			return nil, nil, fmt.Errorf("agent %s has no running pod", args.Name)
		}
		if err := k8s.CompactAgentTask(ctx, ns, a.Status.PodName, args.Name, args.Instructions); err != nil {
			return nil, nil, fmt.Errorf("failed to compact: %w", err)
		}
		return nil, map[string]any{
			"status": "compacting",
			"name":   args.Name,
		}, nil
	})
}

// ─── Schedules ────────────────────────────────────────────────────────────

type scheduleNameArgs struct {
	Name      string `json:"name" jsonschema:"Schedule name"`
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace"`
}

type listSchedulesArgs struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace; empty = all namespaces"`
}

func registerScheduleMCPTools(srv *mcp.Server, k8s *K8sClient) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_schedules",
		Description: "List KomputerSchedule resources. Each schedule fires its agent on a cron cadence with a fixed instructions payload.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listSchedulesArgs) (*mcp.CallToolResult, any, error) {
		schedules, err := k8s.ListSchedules(ctx, args.Namespace)
		if err != nil {
			return nil, nil, err
		}
		out := make([]ScheduleResponse, 0, len(schedules))
		for _, s := range schedules {
			out = append(out, scheduleToResponse(s))
		}
		return nil, map[string]any{"schedules": out}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_schedule",
		Description: "Get full details for a single KomputerSchedule including the instructions it runs each tick.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args scheduleNameArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		s, err := k8s.GetSchedule(ctx, ns, args.Name)
		if err != nil {
			return nil, nil, err
		}
		return nil, scheduleToResponse(*s), nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "trigger_schedule",
		Description: "Fire a schedule immediately, outside its cron cadence. The schedule's normal next run remains scheduled.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args scheduleNameArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		agentName, _, err := triggerScheduleNow(ctx, k8s, ns, args.Name)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"status":    "triggered",
			"name":      args.Name,
			"agentName": agentName,
		}, nil
	})
}

// ─── Memories ─────────────────────────────────────────────────────────────

type memoryNameArgs struct {
	Name      string `json:"name" jsonschema:"Memory name"`
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace"`
}

type listMemoriesArgs struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace; empty = current"`
}

func registerMemoryMCPTools(srv *mcp.Server, k8s *K8sClient) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_memories",
		Description: "List KomputerMemory resources (reusable knowledge blocks that agents can attach to their system prompt).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listMemoriesArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		mems, err := k8s.ListMemories(ctx, ns)
		if err != nil {
			return nil, nil, err
		}
		out := make([]map[string]any, 0, len(mems))
		for _, m := range mems {
			out = append(out, map[string]any{
				"name":        m.Name,
				"namespace":   m.Namespace,
				"description": m.Spec.Description,
			})
		}
		return nil, map[string]any{"memories": out}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_memory",
		Description: "Get a single memory's content and description.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args memoryNameArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		m, err := k8s.GetMemory(ctx, ns, args.Name)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"name":        m.Name,
			"namespace":   m.Namespace,
			"description": m.Spec.Description,
			"content":     m.Spec.Content,
		}, nil
	})
}

// ─── Skills ───────────────────────────────────────────────────────────────

type skillNameArgs struct {
	Name      string `json:"name" jsonschema:"Skill name"`
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace"`
}

func registerSkillMCPTools(srv *mcp.Server, k8s *K8sClient) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_skills",
		Description: "List KomputerSkill resources (reusable scripts/tools agents can invoke).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listMemoriesArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		skills, err := k8s.ListSkills(ctx, ns)
		if err != nil {
			return nil, nil, err
		}
		out := make([]map[string]any, 0, len(skills))
		for _, s := range skills {
			out = append(out, map[string]any{
				"name":        s.Name,
				"namespace":   s.Namespace,
				"description": s.Spec.Description,
			})
		}
		return nil, map[string]any{"skills": out}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_skill",
		Description: "Get a single skill's full body and description.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args skillNameArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		s, err := k8s.GetSkill(ctx, ns, args.Name)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"name":        s.Name,
			"namespace":   s.Namespace,
			"description": s.Spec.Description,
			"body":        s.Spec.Content,
		}, nil
	})
}

// ─── Connectors ───────────────────────────────────────────────────────────

func registerConnectorMCPTools(srv *mcp.Server, k8s *K8sClient) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_connectors",
		Description: "List KomputerConnector resources (configured external MCP servers like GitHub, Slack, Atlassian).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listMemoriesArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		conns, err := k8s.ListConnectors(ctx, ns)
		if err != nil {
			return nil, nil, err
		}
		out := make([]map[string]any, 0, len(conns))
		for _, c := range conns {
			out = append(out, map[string]any{
				"name":      c.Name,
				"namespace": c.Namespace,
				"type":      c.Spec.Type,
				"url":       c.Spec.URL,
			})
		}
		return nil, map[string]any{"connectors": out}, nil
	})
}

// ─── Secrets ──────────────────────────────────────────────────────────────

func registerSecretMCPTools(srv *mcp.Server, k8s *K8sClient) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_secrets",
		Description: "List K8s Secret names visible in a namespace (values are never returned).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listMemoriesArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		secrets, err := k8s.ListSecrets(ctx, ns, false)
		if err != nil {
			return nil, nil, err
		}
		out := make([]map[string]any, 0, len(secrets))
		for _, s := range secrets {
			keys := make([]string, 0, len(s.Data))
			for k := range s.Data {
				keys = append(keys, k)
			}
			out = append(out, map[string]any{
				"name":      s.Name,
				"namespace": s.Namespace,
				"keys":      keys,
			})
		}
		return nil, map[string]any{"secrets": out}, nil
	})
}

// ─── Infrastructure ──────────────────────────────────────────────────────

func registerInfraMCPTools(srv *mcp.Server, k8s *K8sClient) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_namespaces",
		Description: "List Kubernetes namespaces visible to komputer-api.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		ns, err := k8s.ListNamespaces(ctx)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"namespaces": ns}, nil
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_templates",
		Description: "List available KomputerAgentTemplate / KomputerAgentClusterTemplate resources.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args listMemoriesArgs) (*mcp.CallToolResult, any, error) {
		ns := args.Namespace
		if ns == "" {
			ns = k8s.defaultNamespace
		}
		templates, err := k8s.ListTemplates(ctx, ns)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"templates": templates}, nil
	})
}
