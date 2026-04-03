package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
)

type CreateAgentRequest struct {
	Name         string            `json:"name" binding:"required"`
	Instructions string            `json:"instructions" binding:"required"`
	Model        string            `json:"model"`
	TemplateRef  string            `json:"templateRef"`
	Role         string            `json:"role"`      // "manager" or "" (default manager)
	Namespace    string            `json:"namespace"` // optional, defaults to server default
	Secrets      map[string]string `json:"secrets"`    // optional key-value secrets
	SecretRefs   []string          `json:"secretRefs"` // names of existing K8s Secrets to attach
	Memories     []string          `json:"memories"`   // optional KomputerMemory names to attach
	Skills       []string          `json:"skills"`     // optional KomputerSkill names to attach
	Lifecycle     string            `json:"lifecycle"`     // "", "Sleep", or "AutoDelete"
	OfficeManager string            `json:"officeManager"` // set by manager MCP tool
}

type AgentResponse struct {
	Name            string   `json:"name"`
	Namespace       string   `json:"namespace"`
	Model           string   `json:"model"`
	Status          string   `json:"status"`
	TaskStatus      string   `json:"taskStatus,omitempty"`
	LastTaskMessage string   `json:"lastTaskMessage,omitempty"`
	Lifecycle       string   `json:"lifecycle,omitempty"`
	LastTaskCostUSD string   `json:"lastTaskCostUSD,omitempty"`
	TotalCostUSD    string   `json:"totalCostUSD,omitempty"`
	TotalTokens          int64    `json:"totalTokens,omitempty"`
	ModelContextWindow   int64    `json:"modelContextWindow,omitempty"`
	Secrets              []string `json:"secrets,omitempty"`      // Key names from K8s Secrets (not values)
	Memories        []string `json:"memories,omitempty"`     // KomputerMemory names attached to this agent
	Skills          []string `json:"skills,omitempty"`       // KomputerSkill names attached to this agent
	Instructions    string   `json:"instructions,omitempty"` // User task extracted from spec.instructions
	CreatedAt       string   `json:"createdAt"`
}

// extractUserTask extracts the user's task from the full instructions string.
// The system prompt prefix ends at "## Your Task\n" — everything after that marker is the user task.
// If no marker is found, the full instructions are returned.
func extractUserTask(instructions string) string {
	const marker = "## Your Task\n"
	idx := strings.Index(instructions, marker)
	if idx == -1 {
		return instructions
	}
	return strings.TrimSpace(instructions[idx+len(marker):])
}

type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
}

type OfficeResponse struct {
	Name            string                 `json:"name"`
	Namespace       string                 `json:"namespace"`
	Manager         string                 `json:"manager"`
	Phase           string                 `json:"phase"`
	TotalAgents     int                    `json:"totalAgents"`
	ActiveAgents    int                    `json:"activeAgents"`
	CompletedAgents int                    `json:"completedAgents"`
	TotalCostUSD    string                 `json:"totalCostUSD,omitempty"`
	TotalTokens     int64                  `json:"totalTokens,omitempty"`
	Members         []OfficeMemberResponse `json:"members,omitempty"`
	CreatedAt       string                 `json:"createdAt"`
}

type OfficeMemberResponse struct {
	Name            string `json:"name"`
	Role            string `json:"role"`
	TaskStatus      string `json:"taskStatus,omitempty"`
	LastTaskCostUSD string `json:"lastTaskCostUSD,omitempty"`
}

type OfficeListResponse struct {
	Offices []OfficeResponse `json:"offices"`
}

type CreateScheduleRequest struct {
	Name         string                  `json:"name" binding:"required"`
	Schedule     string                  `json:"schedule" binding:"required"`
	Instructions string                  `json:"instructions" binding:"required"`
	Timezone     string                  `json:"timezone"`
	AutoDelete   bool                    `json:"autoDelete"`
	KeepAgents   bool                    `json:"keepAgents"`
	AgentName    string                  `json:"agentName"`
	Agent        *CreateScheduleAgentSpec `json:"agent"`
	Namespace    string                  `json:"namespace"`
}

type CreateScheduleAgentSpec struct {
	Model       string            `json:"model"`
	Lifecycle   string            `json:"lifecycle"`
	Role        string            `json:"role"`
	TemplateRef string            `json:"templateRef"`
	Secrets     map[string]string `json:"secrets"`
}

type ScheduleResponse struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	Schedule       string `json:"schedule"`
	Timezone       string `json:"timezone,omitempty"`
	AutoDelete     bool   `json:"autoDelete,omitempty"`
	KeepAgents     bool   `json:"keepAgents,omitempty"`
	Phase          string `json:"phase"`
	AgentName      string `json:"agentName,omitempty"`
	NextRunTime    string `json:"nextRunTime,omitempty"`
	LastRunTime    string `json:"lastRunTime,omitempty"`
	RunCount       int    `json:"runCount,omitempty"`
	SuccessfulRuns int    `json:"successfulRuns,omitempty"`
	FailedRuns     int    `json:"failedRuns,omitempty"`
	TotalCostUSD   string `json:"totalCostUSD,omitempty"`
	LastRunCostUSD string `json:"lastRunCostUSD,omitempty"`
	TotalTokens    int64  `json:"totalTokens,omitempty"`
	LastRunTokens  int64  `json:"lastRunTokens,omitempty"`
	LastRunStatus  string `json:"lastRunStatus,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

type ScheduleListResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
}

// --- Secret types ---

type SecretResponse struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Keys      []string `json:"keys"`
	Managed   bool     `json:"managed"`
	AgentName string   `json:"agentName,omitempty"`
	CreatedAt string   `json:"createdAt"`
}

type SecretListResponse struct {
	Secrets []SecretResponse `json:"secrets"`
}

type CreateSecretRequest struct {
	Name      string            `json:"name" binding:"required"`
	Data      map[string]string `json:"data" binding:"required"`
	Namespace string            `json:"namespace"`
}

// isValidK8sName checks if a string is a valid Kubernetes DNS subdomain name.
var k8sNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func isValidK8sName(name string) bool {
	return len(name) > 0 && len(name) <= 63 && k8sNameRegex.MatchString(name)
}

// resolveNamespace returns the namespace from the query param, request body, or default.
func resolveNamespace(c *gin.Context, k8s *K8sClient) string {
	if ns := c.Query("namespace"); ns != "" {
		return ns
	}
	return k8s.defaultNamespace
}

// collectSecretKeys gathers all key names from the agent's referenced K8s Secrets.
func collectSecretKeys(ctx gin.Context, k8s *K8sClient, ns string, secretNames []string) []string {
	var keys []string
	for _, name := range secretNames {
		k, err := k8s.GetSecretKeys(ctx.Request.Context(), ns, name)
		if err != nil {
			continue // secret may have been deleted
		}
		keys = append(keys, k...)
	}
	return keys
}

func SetupRoutes(r *gin.Engine, k8s *K8sClient, hub *Hub, worker *RedisWorker) {
	// Health check endpoints (outside /api/v1 for k8s probes).
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		// Check Redis connectivity.
		if err := worker.Rdb.Ping(c.Request.Context()).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "redis: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/agents", createOrTriggerAgent(k8s))
		v1.GET("/agents", listAgents(k8s))
		v1.GET("/agents/:name", getAgent(k8s))
		v1.GET("/agents/:name/events", getAgentEvents(worker))
		v1.DELETE("/agents/:name", deleteAgent(k8s, worker))
		v1.PATCH("/agents/:name", patchAgent(k8s))
		v1.POST("/agents/:name/cancel", cancelAgentTask(k8s))
		v1.GET("/agents/:name/ws", HandleAgentWS(hub))

		v1.GET("/offices", listOffices(k8s))
		v1.GET("/offices/:name", getOffice(k8s))
		v1.DELETE("/offices/:name", deleteOffice(k8s, worker))
		v1.GET("/offices/:name/events", getOfficeEvents(k8s, worker))

		v1.POST("/schedules", createSchedule(k8s))
		v1.GET("/schedules", listSchedules(k8s))
		v1.GET("/schedules/:name", getSchedule(k8s))
		v1.DELETE("/schedules/:name", deleteSchedule(k8s))
		v1.PATCH("/schedules/:name", patchSchedule(k8s))

		v1.POST("/memories", createMemory(k8s))
		v1.GET("/memories", listMemories(k8s))
		v1.GET("/memories/:name", getMemory(k8s))
		v1.DELETE("/memories/:name", deleteMemory(k8s))
		v1.PATCH("/memories/:name", patchMemory(k8s))

		v1.POST("/skills", createSkill(k8s))
		v1.GET("/skills", listSkills(k8s))
		v1.GET("/skills/:name", getSkill(k8s))
		v1.DELETE("/skills/:name", deleteSkill(k8s))
		v1.PATCH("/skills/:name", patchSkill(k8s))

		v1.GET("/secrets", listSecrets(k8s))
		v1.POST("/secrets", createManagedSecret(k8s))
		v1.DELETE("/secrets/:name", deleteManagedSecret(k8s))

		v1.GET("/templates", listTemplates(k8s))
		v1.GET("/namespaces", listNamespaces(k8s))
	}
}

func listNamespaces(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		names, err := k8s.ListNamespaces(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"namespaces": names})
	}
}

// createOrTriggerAgent creates a new agent or sends a task to an existing one.
// @Summary Create agent or send task
// @Description Creates a new agent or sends a task to an existing idle agent (upsert by name).
// @Description If the agent doesn't exist, it is created. If it exists and is idle, the task is forwarded.
// @Tags agents
// @Accept json
// @Produce json
// @Param request body CreateAgentRequest true "Agent creation request"
// @Success 201 {object} AgentResponse "Agent created"
// @Success 200 {object} AgentResponse "Task forwarded to existing agent"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 409 {object} map[string]string "Agent is busy or has no running pod"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents [post]
func createOrTriggerAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAgentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate agent name: must be a valid K8s DNS subdomain name.
		if !isValidK8sName(req.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent name: must be lowercase, alphanumeric, hyphens only, max 63 chars (e.g. 'my-agent-1')"})
			return
		}

		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}

		// Build full instructions with system prompt early — needed for both new agents and wake-up.
		role := req.Role
		if role == "" {
			role = "manager"
		}
		if role != "worker" && role != "manager" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'worker' or 'manager'"})
			return
		}
		// System prompt and user instructions are sent separately to the agent.
		// The agent passes system_prompt to the Claude SDK's system_prompt field
		// (not as part of conversation history — replaced on each task, never accumulates).
		instructions := req.Instructions

		// Build system prompt — memory resolution happens after we know if agent exists
		// (existing agents use CR memories, new agents use request memories)
		buildSystemPrompt := func(memories []string) string {
			memorySection, _ := k8s.ResolveMemoryContent(c.Request.Context(), ns, memories)
			agentHeader := fmt.Sprintf("\n---\n\n**Agent Name:** %s\n\n## Your Task\n", req.Name)
			if role == "manager" {
				return managerSystemPrompt + memorySection + agentHeader
			}
			return workerSystemPrompt + memorySection + agentHeader
		}

		existing, err := k8s.GetAgent(c.Request.Context(), ns, req.Name)
		if err != nil && !errors.IsNotFound(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check agent: " + err.Error()})
			return
		}

		if existing != nil {
			// Wake-up flow for sleeping agents
			if existing.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
				// Use CR memories (may have been updated via PATCH since creation)
			wakeMemories := existing.Spec.Memories
			if len(req.Memories) > 0 {
				wakeMemories = req.Memories
			}
			if err := k8s.WakeAgent(c.Request.Context(), ns, req.Name, buildSystemPrompt(wakeMemories)+"\n\n"+instructions, req.Model, req.Lifecycle); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to wake agent: " + err.Error()})
					return
				}
				log.Printf("waking sleeping agent %s/%s", ns, req.Name)
				c.JSON(http.StatusAccepted, AgentResponse{
					Name:            existing.Name,
					Namespace:       existing.Namespace,
					Model:           existing.Spec.Model,
					Status:          "Pending",
					Lifecycle:       string(existing.Spec.Lifecycle),
					LastTaskCostUSD: existing.Status.LastTaskCostUSD,
					TotalCostUSD:    existing.Status.TotalCostUSD,
					TotalTokens:          existing.Status.TotalTokens,
						ModelContextWindow:   existing.Status.ModelContextWindow,
					Secrets:         collectSecretKeys(*c, k8s, ns, existing.Spec.Secrets),
					Memories:        existing.Spec.Memories,
					Skills:          existing.Spec.Skills,
					Instructions:    extractUserTask(existing.Spec.Instructions),
					CreatedAt:       existing.CreationTimestamp.UTC().Format(time.RFC3339),
				})
				return
			}

			if existing.Status.PodName == "" {
				c.JSON(http.StatusConflict, gin.H{"error": "agent exists but has no running pod yet"})
				return
			}

			if existing.Status.TaskStatus == komputerv1alpha1.AgentTaskInProgress {
				c.JSON(http.StatusConflict, gin.H{"error": "agent is busy with another task"})
				return
			}

			podIP, err := k8s.GetAgentPodIP(c.Request.Context(), ns, existing.Status.PodName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent pod IP: " + err.Error()})
				return
			}

			// Use CR memories for existing agents (may have been updated via PATCH)
			forwardMemories := existing.Spec.Memories
			if len(req.Memories) > 0 {
				forwardMemories = req.Memories
			}
			cw, err := k8s.ForwardTaskToAgent(c.Request.Context(), ns, existing.Status.PodName, podIP, instructions, req.Model, buildSystemPrompt(forwardMemories))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to forward task: " + err.Error()})
				return
			}
			if cw > 0 {
				if patchErr := k8s.PatchAgentContextWindow(c.Request.Context(), ns, req.Name, cw); patchErr != nil {
					log.Printf("warning: failed to patch context window for %s: %v", req.Name, patchErr)
				}
			}

			// Update lifecycle if changed
			if req.Lifecycle != "" && req.Lifecycle != string(existing.Spec.Lifecycle) {
				if err := k8s.PatchAgentLifecycle(c.Request.Context(), ns, req.Name, req.Lifecycle); err != nil {
					log.Printf("warning: failed to patch lifecycle for %s: %v", req.Name, err)
				}
			}

			log.Printf("forwarded task to existing agent %s/%s", ns, req.Name)
			c.JSON(http.StatusOK, AgentResponse{
				Name:            existing.Name,
				Namespace:       existing.Namespace,
				Model:           existing.Spec.Model,
				Status:          string(existing.Status.Phase),
				TaskStatus:      string(existing.Status.TaskStatus),
				LastTaskMessage: existing.Status.LastTaskMessage,
				Lifecycle:       string(existing.Spec.Lifecycle),
				LastTaskCostUSD: existing.Status.LastTaskCostUSD,
				TotalCostUSD:    existing.Status.TotalCostUSD,
				TotalTokens:          existing.Status.TotalTokens,
						ModelContextWindow:   existing.Status.ModelContextWindow,
				Secrets:         collectSecretKeys(*c, k8s, ns, existing.Spec.Secrets),
				Memories:        existing.Spec.Memories,
				Skills:          existing.Spec.Skills,
				Instructions:    extractUserTask(existing.Spec.Instructions),
				CreatedAt:       existing.CreationTimestamp.UTC().Format(time.RFC3339),
			})
			return
		}

		if err := k8s.EnsureNamespace(c.Request.Context(), ns); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure namespace: " + err.Error()})
			return
		}

		var secretNames []string
		if len(req.Secrets) > 0 {
			secretName, err := k8s.CreateAgentSecrets(c.Request.Context(), ns, req.Name, req.Secrets)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create secrets: " + err.Error()})
				return
			}
			secretNames = []string{secretName}
		}
		secretNames = append(secretNames, req.SecretRefs...)

		agent, err := k8s.CreateAgent(c.Request.Context(), ns, req.Name, buildSystemPrompt(req.Memories)+"\n\n"+instructions, req.Model, req.TemplateRef, role, secretNames, req.Memories, req.Skills, req.Lifecycle, req.OfficeManager)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent: " + err.Error()})
			return
		}

		log.Printf("created new agent %s/%s", ns, req.Name)
		c.JSON(http.StatusCreated, AgentResponse{
			Name:         agent.Name,
			Namespace:    agent.Namespace,
			Model:        agent.Spec.Model,
			Status:       "Pending",
			Lifecycle:    string(agent.Spec.Lifecycle),
			Secrets:      collectSecretKeys(*c, k8s, ns, agent.Spec.Secrets),
			Memories:     agent.Spec.Memories,
			Skills:       agent.Spec.Skills,
			Instructions: extractUserTask(agent.Spec.Instructions),
			CreatedAt:    agent.CreationTimestamp.UTC().Format(time.RFC3339),
			TotalTokens:        agent.Status.TotalTokens,
			ModelContextWindow: agent.Status.ModelContextWindow,
		})
	}
}

// deleteAgent deletes an agent and cleans up all its resources.
// @Summary Delete agent
// @Description Deletes the agent CR, pod, PVC, secrets, and Redis event stream.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Agent deleted"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name} [delete]
func deleteAgent(k8s *K8sClient, worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		if err := k8s.DeleteAgent(c.Request.Context(), ns, name); err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete agent: " + err.Error()})
			return
		}
		// Clean up the agent's Redis event stream
		if err := DeleteAgentStream(c.Request.Context(), worker.Rdb, name, worker.StreamPrefix); err != nil {
			log.Printf("warning: failed to delete event stream for %s: %v", name, err)
		}
		log.Printf("deleted agent %s/%s", ns, name)
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

// cancelAgentTask cancels the running task on an agent.
// @Summary Cancel agent task
// @Description Gracefully cancels the currently running task. The agent pod stays alive for future tasks.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Task cancelling"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 409 {object} map[string]string "Agent has no running pod"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name}/cancel [post]
func cancelAgentTask(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		agent, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if agent.Status.PodName == "" {
			c.JSON(http.StatusConflict, gin.H{"error": "agent has no running pod"})
			return
		}

		podIP, err := k8s.GetAgentPodIP(c.Request.Context(), ns, agent.Status.PodName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pod IP: " + err.Error()})
			return
		}

		if err := k8s.CancelAgentTask(c.Request.Context(), ns, agent.Status.PodName, podIP); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel: " + err.Error()})
			return
		}

		log.Printf("cancelled task on agent %s/%s", ns, name)
		c.JSON(http.StatusOK, gin.H{"status": "cancelling", "name": name})
	}
}

// getAgent returns details for a single agent.
// @Summary Get agent details
// @Description Returns the current status and metadata for a single agent.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} AgentResponse "Agent details"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name} [get]
func getAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		agent, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, AgentResponse{
			Name:            agent.Name,
			Namespace:       agent.Namespace,
			Model:           agent.Spec.Model,
			Status:          string(agent.Status.Phase),
			TaskStatus:      string(agent.Status.TaskStatus),
			LastTaskMessage: agent.Status.LastTaskMessage,
			Lifecycle:       string(agent.Spec.Lifecycle),
			LastTaskCostUSD: agent.Status.LastTaskCostUSD,
			TotalCostUSD:    agent.Status.TotalCostUSD,
			TotalTokens:        agent.Status.TotalTokens,
			ModelContextWindow: agent.Status.ModelContextWindow,
			Secrets:            agent.Spec.Secrets,
			Memories:           agent.Spec.Memories,
			Skills:             agent.Spec.Skills,
			Instructions:       extractUserTask(agent.Spec.Instructions),
			CreatedAt:          agent.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

// getAgentEvents returns the event history for an agent from Redis.
// @Summary Get agent events
// @Description Returns recent events from the agent's Redis stream in chronological order.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Param limit query int false "Max events to return (1-200)" default(50)
// @Success 200 {object} map[string]interface{} "Agent events"
// @Failure 400 {object} map[string]string "Invalid limit parameter"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name}/events [get]
func getAgentEvents(worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		limit := int64(50)
		if l := c.Query("limit"); l != "" {
			parsed, err := strconv.ParseInt(l, 10, 64)
			if err != nil || parsed < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
				return
			}
			if parsed > 200 {
				parsed = 200
			}
			limit = parsed
		}
		before := c.Query("before") // RFC-3339 cursor for pagination
		events, err := GetAgentEventsBefore(c.Request.Context(), worker.Rdb, name, limit, before, worker.StreamPrefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent events: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"agent": name, "events": events})
	}
}

// listAgents returns all agents in a namespace.
// @Summary List agents
// @Description Returns all agents with their current status in the specified namespace.
// @Tags agents
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} AgentListResponse "List of agents"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents [get]
func listAgents(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		agents, err := k8s.ListAgents(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agents: " + err.Error()})
			return
		}

		resp := AgentListResponse{Agents: make([]AgentResponse, 0, len(agents))}
		for _, a := range agents {
			resp.Agents = append(resp.Agents, AgentResponse{
				Name:            a.Name,
				Namespace:       a.Namespace,
				Model:           a.Spec.Model,
				Status:          string(a.Status.Phase),
				TaskStatus:      string(a.Status.TaskStatus),
				LastTaskMessage: a.Status.LastTaskMessage,
				Lifecycle:       string(a.Spec.Lifecycle),
				LastTaskCostUSD: a.Status.LastTaskCostUSD,
				TotalCostUSD:    a.Status.TotalCostUSD,
				TotalTokens:        a.Status.TotalTokens,
				ModelContextWindow: a.Status.ModelContextWindow,
				Secrets:            collectSecretKeys(*c, k8s, ns, a.Spec.Secrets),
				Memories:           a.Spec.Memories,
				Skills:             a.Spec.Skills,
				Instructions:       extractUserTask(a.Spec.Instructions),
				CreatedAt:          a.CreationTimestamp.UTC().Format(time.RFC3339),
			})
		}

		c.JSON(http.StatusOK, resp)
	}
}

// officeToResponse converts a KomputerOffice CR to an OfficeResponse.
func officeToResponse(o komputerv1alpha1.KomputerOffice, includeMembers bool) OfficeResponse {
	resp := OfficeResponse{
		Name:            o.Name,
		Namespace:       o.Namespace,
		Manager:         o.Spec.Manager,
		Phase:           string(o.Status.Phase),
		TotalAgents:     o.Status.TotalAgents,
		ActiveAgents:    o.Status.ActiveAgents,
		CompletedAgents: o.Status.CompletedAgents,
		TotalCostUSD:    o.Status.TotalCostUSD,
		TotalTokens:     o.Status.TotalTokens,
		CreatedAt:       o.CreationTimestamp.UTC().Format(time.RFC3339),
	}
	if includeMembers {
		members := make([]OfficeMemberResponse, 0, len(o.Status.Members)+1)
		// Include manager as a member entry.
		if o.Status.Manager.Name != "" {
			members = append(members, OfficeMemberResponse{
				Name:            o.Status.Manager.Name,
				Role:            o.Status.Manager.Role,
				TaskStatus:      o.Status.Manager.TaskStatus,
				LastTaskCostUSD: o.Status.Manager.LastTaskCostUSD,
			})
		}
		for _, m := range o.Status.Members {
			members = append(members, OfficeMemberResponse{
				Name:            m.Name,
				Role:            m.Role,
				TaskStatus:      m.TaskStatus,
				LastTaskCostUSD: m.LastTaskCostUSD,
			})
		}
		resp.Members = members
	}
	return resp
}

func listOffices(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		offices, err := k8s.ListOffices(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list offices: " + err.Error()})
			return
		}
		resp := OfficeListResponse{Offices: make([]OfficeResponse, 0, len(offices))}
		for _, o := range offices {
			resp.Offices = append(resp.Offices, officeToResponse(o, false))
		}
		c.JSON(http.StatusOK, resp)
	}
}

func getOffice(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		office, err := k8s.GetOffice(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "office not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, officeToResponse(*office, true))
	}
}

func deleteOffice(k8s *K8sClient, worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		// Get the office first so we can clean up member event streams.
		office, err := k8s.GetOffice(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "office not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := k8s.DeleteOffice(c.Request.Context(), ns, name); err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "office not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete office: " + err.Error()})
			return
		}

		// Clean up Redis event streams for all member agents (including manager).
		if office.Status.Manager.Name != "" {
			if err := DeleteAgentStream(c.Request.Context(), worker.Rdb, office.Status.Manager.Name, worker.StreamPrefix); err != nil {
				log.Printf("warning: failed to delete event stream for manager %s: %v", office.Status.Manager.Name, err)
			}
		}
		for _, m := range office.Status.Members {
			if err := DeleteAgentStream(c.Request.Context(), worker.Rdb, m.Name, worker.StreamPrefix); err != nil {
				log.Printf("warning: failed to delete event stream for member %s: %v", m.Name, err)
			}
		}

		log.Printf("deleted office %s/%s", ns, name)
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

// scheduleToResponse converts a KomputerSchedule CR to a ScheduleResponse.
func scheduleToResponse(s komputerv1alpha1.KomputerSchedule) ScheduleResponse {
	resp := ScheduleResponse{
		Name:           s.Name,
		Namespace:      s.Namespace,
		Schedule:       s.Spec.Schedule,
		Timezone:       s.Spec.Timezone,
		AutoDelete:     s.Spec.AutoDelete,
		KeepAgents:     s.Spec.KeepAgents,
		Phase:          string(s.Status.Phase),
		AgentName:      s.Status.AgentName,
		RunCount:       s.Status.RunCount,
		SuccessfulRuns: s.Status.SuccessfulRuns,
		FailedRuns:     s.Status.FailedRuns,
		TotalCostUSD:   s.Status.TotalCostUSD,
		LastRunCostUSD: s.Status.LastRunCostUSD,
		TotalTokens:    s.Status.TotalTokens,
		LastRunTokens:  s.Status.LastRunTokens,
		LastRunStatus:  s.Status.LastRunStatus,
		CreatedAt:      s.CreationTimestamp.UTC().Format(time.RFC3339),
	}
	if s.Status.NextRunTime != nil {
		resp.NextRunTime = s.Status.NextRunTime.Format("2006-01-02T15:04:05Z")
	}
	if s.Status.LastRunTime != nil {
		resp.LastRunTime = s.Status.LastRunTime.Format("2006-01-02T15:04:05Z")
	}
	// Use spec agentName if status doesn't have one yet.
	if resp.AgentName == "" {
		resp.AgentName = s.Spec.AgentName
	}
	return resp
}

func createSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if !isValidK8sName(req.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule name: must be lowercase, alphanumeric, hyphens only, max 63 chars (e.g. 'my-schedule-1')"})
			return
		}

		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}

		if err := k8s.EnsureNamespace(c.Request.Context(), ns); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure namespace: " + err.Error()})
			return
		}

		schedule, err := k8s.CreateSchedule(c.Request.Context(), ns, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create schedule: " + err.Error()})
			return
		}

		log.Printf("created new schedule %s/%s", ns, req.Name)
		c.JSON(http.StatusCreated, scheduleToResponse(*schedule))
	}
}

func listSchedules(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		schedules, err := k8s.ListSchedules(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules: " + err.Error()})
			return
		}
		resp := ScheduleListResponse{Schedules: make([]ScheduleResponse, 0, len(schedules))}
		for _, s := range schedules {
			resp.Schedules = append(resp.Schedules, scheduleToResponse(s))
		}
		c.JSON(http.StatusOK, resp)
	}
}

func getSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		schedule, err := k8s.GetSchedule(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, scheduleToResponse(*schedule))
	}
}

func deleteSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		if err := k8s.DeleteSchedule(c.Request.Context(), ns, name); err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule: " + err.Error()})
			return
		}
		log.Printf("deleted schedule %s/%s", ns, name)
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

type PatchScheduleRequest struct {
	Schedule *string `json:"schedule,omitempty"`
}

func patchSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		var req PatchScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Schedule == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}
		if err := k8s.PatchScheduleCron(c.Request.Context(), ns, name, *req.Schedule); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch schedule: " + err.Error()})
			return
		}
		sched, err := k8s.GetSchedule(c.Request.Context(), ns, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "patched but failed to read back: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, scheduleToResponse(*sched))
	}
}

func getOfficeEvents(k8s *K8sClient, worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		limit := int64(50)
		if l := c.Query("limit"); l != "" {
			parsed, err := strconv.ParseInt(l, 10, 64)
			if err != nil || parsed < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
				return
			}
			if parsed > 200 {
				parsed = 200
			}
			limit = parsed
		}

		office, err := k8s.GetOffice(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "office not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Collect all agent names (manager + members).
		agentNames := make([]string, 0, len(office.Status.Members)+1)
		if office.Status.Manager.Name != "" {
			agentNames = append(agentNames, office.Status.Manager.Name)
		}
		for _, m := range office.Status.Members {
			agentNames = append(agentNames, m.Name)
		}

		// Fetch events for each agent and merge.
		var allEvents []AgentEvent
		for _, agentName := range agentNames {
			events, err := GetAgentEvents(c.Request.Context(), worker.Rdb, agentName, limit, worker.StreamPrefix)
			if err != nil {
				log.Printf("warning: failed to get events for agent %s: %v", agentName, err)
				continue
			}
			allEvents = append(allEvents, events...)
		}

		// Sort by timestamp ascending.
		sort.Slice(allEvents, func(i, j int) bool {
			return allEvents[i].Timestamp < allEvents[j].Timestamp
		})

		// Apply limit to merged results.
		if int64(len(allEvents)) > limit {
			allEvents = allEvents[len(allEvents)-int(limit):]
		}

		c.JSON(http.StatusOK, gin.H{"office": name, "events": allEvents})
	}
}

// --- Memory handlers ---

type CreateMemoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Content     string `json:"content" binding:"required"`
	Description string `json:"description"`
	Namespace   string `json:"namespace"`
}

type MemoryResponse struct {
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace"`
	Content        string   `json:"content"`
	Description    string   `json:"description,omitempty"`
	AttachedAgents int      `json:"attachedAgents"`
	AgentNames     []string `json:"agentNames,omitempty"`
	CreatedAt      string   `json:"createdAt"`
}

func createMemory(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateMemoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if !isValidK8sName(req.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid memory name: must be lowercase letters, numbers, and hyphens"})
			return
		}
		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}
		memory, err := k8s.CreateMemory(c.Request.Context(), ns, req.Name, req.Content, req.Description)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create memory: " + err.Error()})
			return
		}
		c.JSON(http.StatusCreated, MemoryResponse{
			Name:      memory.Name,
			Namespace: memory.Namespace,
			Content:   memory.Spec.Content,
			Description: memory.Spec.Description,
			CreatedAt: memory.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

func getMemory(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		memory, err := k8s.GetMemory(c.Request.Context(), ns, name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "memory not found"})
			return
		}
		c.JSON(http.StatusOK, MemoryResponse{
			Name:           memory.Name,
			Namespace:      memory.Namespace,
			Content:        memory.Spec.Content,
			Description:    memory.Spec.Description,
			AttachedAgents: memory.Status.AttachedAgents,
			AgentNames:     memory.Status.AgentNames,
			CreatedAt:      memory.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

func listMemories(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		memories, err := k8s.ListMemories(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list memories: " + err.Error()})
			return
		}
		resp := make([]MemoryResponse, 0, len(memories))
		for _, m := range memories {
			resp = append(resp, MemoryResponse{
				Name:           m.Name,
				Namespace:      m.Namespace,
				Content:        m.Spec.Content,
				Description:    m.Spec.Description,
				AttachedAgents: m.Status.AttachedAgents,
				AgentNames:     m.Status.AgentNames,
				CreatedAt:      m.CreationTimestamp.UTC().Format(time.RFC3339),
			})
		}
		c.JSON(http.StatusOK, gin.H{"memories": resp})
	}
}

type PatchMemoryRequest struct {
	Content     *string `json:"content,omitempty"`
	Description *string `json:"description,omitempty"`
}

func patchMemory(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		var req PatchMemoryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Content == nil && req.Description == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}
		if err := k8s.PatchMemory(c.Request.Context(), ns, name, req.Content, req.Description); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch memory: " + err.Error()})
			return
		}
		memory, err := k8s.GetMemory(c.Request.Context(), ns, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "patched but failed to read back: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, MemoryResponse{
			Name:           memory.Name,
			Namespace:      memory.Namespace,
			Content:        memory.Spec.Content,
			Description:    memory.Spec.Description,
			AttachedAgents: memory.Status.AttachedAgents,
			AgentNames:     memory.Status.AgentNames,
			CreatedAt:      memory.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

func deleteMemory(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		if err := k8s.DeleteMemory(c.Request.Context(), ns, name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete memory: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

// --- Skill handlers ---

type CreateSkillRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	Content     string `json:"content" binding:"required"`
	Namespace   string `json:"namespace"`
}

type SkillResponse struct {
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace"`
	Description    string   `json:"description"`
	Content        string   `json:"content"`
	AttachedAgents int      `json:"attachedAgents"`
	AgentNames     []string `json:"agentNames,omitempty"`
	IsDefault      bool     `json:"isDefault"`
	CreatedAt      string   `json:"createdAt"`
}

type PatchSkillRequest struct {
	Description *string `json:"description,omitempty"`
	Content     *string `json:"content,omitempty"`
}

func createSkill(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateSkillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if !isValidK8sName(req.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid skill name: must be lowercase letters, numbers, and hyphens"})
			return
		}
		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}
		skill, err := k8s.CreateSkill(c.Request.Context(), ns, req.Name, req.Description, req.Content)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create skill: " + err.Error()})
			return
		}
		c.JSON(http.StatusCreated, SkillResponse{
			Name:        skill.Name,
			Namespace:   skill.Namespace,
			Description: skill.Spec.Description,
			Content:     skill.Spec.Content,
			IsDefault:   skill.Labels["komputer.ai/default"] == "true",
			CreatedAt:   skill.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

func getSkill(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		skill, err := k8s.GetSkill(c.Request.Context(), ns, name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
			return
		}
		c.JSON(http.StatusOK, SkillResponse{
			Name:           skill.Name,
			Namespace:      skill.Namespace,
			Description:    skill.Spec.Description,
			Content:        skill.Spec.Content,
			AttachedAgents: skill.Status.AttachedAgents,
			AgentNames:     skill.Status.AgentNames,
			IsDefault:      skill.Labels["komputer.ai/default"] == "true",
			CreatedAt:      skill.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

func listSkills(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		skills, err := k8s.ListSkills(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list skills: " + err.Error()})
			return
		}
		resp := make([]SkillResponse, 0, len(skills))
		for _, s := range skills {
			resp = append(resp, SkillResponse{
				Name:           s.Name,
				Namespace:      s.Namespace,
				Description:    s.Spec.Description,
				Content:        s.Spec.Content,
				AttachedAgents: s.Status.AttachedAgents,
				AgentNames:     s.Status.AgentNames,
				IsDefault:      s.Labels["komputer.ai/default"] == "true",
				CreatedAt:      s.CreationTimestamp.UTC().Format(time.RFC3339),
			})
		}
		c.JSON(http.StatusOK, gin.H{"skills": resp})
	}
}

func patchSkill(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		var req PatchSkillRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Description == nil && req.Content == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}
		if err := k8s.PatchSkill(c.Request.Context(), ns, name, req.Description, req.Content); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch skill: " + err.Error()})
			return
		}
		skill, err := k8s.GetSkill(c.Request.Context(), ns, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "patched but failed to read back: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, SkillResponse{
			Name:           skill.Name,
			Namespace:      skill.Namespace,
			Description:    skill.Spec.Description,
			Content:        skill.Spec.Content,
			AttachedAgents: skill.Status.AttachedAgents,
			AgentNames:     skill.Status.AgentNames,
			IsDefault:      skill.Labels["komputer.ai/default"] == "true",
			CreatedAt:      skill.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

func deleteSkill(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		if err := k8s.DeleteSkill(c.Request.Context(), ns, name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete skill: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

// --- Secret handlers ---

func listSecrets(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		all := c.Query("all") == "true"
		secrets, err := k8s.ListSecrets(c.Request.Context(), ns, all)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list secrets: " + err.Error()})
			return
		}
		resp := make([]SecretResponse, 0, len(secrets))
		for _, s := range secrets {
			keys := make([]string, 0, len(s.Data))
			for k := range s.Data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			resp = append(resp, SecretResponse{
				Name:      s.Name,
				Namespace: s.Namespace,
				Keys:      keys,
				Managed:   s.Labels["komputer.ai/managed-by"] == "komputer-ai",
				AgentName: s.Labels["komputer.ai/agent-name"],
				CreatedAt: s.CreationTimestamp.UTC().Format(time.RFC3339),
			})
		}
		c.JSON(http.StatusOK, SecretListResponse{Secrets: resp})
	}
}

func createManagedSecret(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateSecretRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if !isValidK8sName(req.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid secret name: must be lowercase letters, numbers, and hyphens"})
			return
		}
		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}
		secret, err := k8s.CreateManagedSecret(c.Request.Context(), ns, req.Name, req.Data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create secret: " + err.Error()})
			return
		}
		keys := make([]string, 0, len(secret.Data))
		for k := range secret.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		c.JSON(http.StatusCreated, SecretResponse{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Keys:      keys,
			Managed:   true,
			CreatedAt: secret.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

func deleteManagedSecret(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		if err := k8s.DeleteManagedSecret(c.Request.Context(), ns, name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete secret: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

type TemplateResponse struct {
	Name      string `json:"name"`
	Scope     string `json:"scope"`               // "namespace" or "cluster"
	Namespace string `json:"namespace,omitempty"`  // populated for namespaced templates
}

func listTemplates(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := resolveNamespace(c, k8s)
		templates, err := k8s.ListTemplates(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list templates: " + err.Error()})
			return
		}

		resp := make([]TemplateResponse, 0, len(templates))
		for _, t := range templates {
			resp = append(resp, TemplateResponse{Name: t.Name, Scope: t.Scope, Namespace: t.Namespace})
		}
		c.JSON(http.StatusOK, gin.H{"templates": resp})
	}
}

type PatchAgentRequest struct {
	Model        *string           `json:"model,omitempty"`
	Lifecycle    *string           `json:"lifecycle,omitempty"`
	Instructions *string           `json:"instructions,omitempty"`
	TemplateRef  *string           `json:"templateRef,omitempty"`
	Secrets      map[string]string `json:"secrets,omitempty"`    // key-value pairs to set/update
	SecretRefs   *[]string         `json:"secretRefs,omitempty"` // full replacement list of K8s secret names
	Memories     *[]string         `json:"memories,omitempty"`   // memory names to attach
	Skills       *[]string         `json:"skills,omitempty"`     // skill names to attach
}

func patchAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		var req PatchAgentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Model == nil && req.Lifecycle == nil && req.Instructions == nil && req.TemplateRef == nil && len(req.Secrets) == 0 && req.SecretRefs == nil && req.Memories == nil && req.Skills == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		// 1. Patch CR spec first — this is the source of truth.
		if err := k8s.PatchAgentSpec(c.Request.Context(), ns, name, req.Model, req.Lifecycle, req.Instructions, req.TemplateRef); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch agent: " + err.Error()})
			return
		}

		// 1a. If model changed and pod is running, fetch context window via the agent (it has the API key).
		// This is handled below in step 2 where we forward config to the pod and read the response.

		// 1b. Update secrets if provided.
		if len(req.Secrets) > 0 {
			secretName, err := k8s.CreateAgentSecrets(c.Request.Context(), ns, name, req.Secrets)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update secrets: " + err.Error()})
				return
			}
			// Ensure the secret name is on the CR's spec.secrets list.
			agent, err := k8s.GetAgent(c.Request.Context(), ns, name)
			if err == nil {
				hasSecret := false
				for _, s := range agent.Spec.Secrets {
					if s == secretName {
						hasSecret = true
						break
					}
				}
				if !hasSecret {
					k8s.PatchAgentSecretsList(c.Request.Context(), ns, name, append(agent.Spec.Secrets, secretName))
				}
			}
		}

		// 1c. Replace secret refs list if provided (used for removal).
		if req.SecretRefs != nil {
			k8s.PatchAgentSecretsList(c.Request.Context(), ns, name, *req.SecretRefs)
		}

		// 1d. Update memories if provided.
		if req.Memories != nil {
			k8s.PatchAgentMemoriesList(c.Request.Context(), ns, name, *req.Memories)
		}

		// 1d. Update skills if provided.
		if req.Skills != nil {
			k8s.PatchAgentSkillsList(c.Request.Context(), ns, name, *req.Skills)
			skillFiles, _ := k8s.ResolveSkillFiles(c.Request.Context(), ns, *req.Skills)
			if len(skillFiles) > 0 {
				if configJSON, err := json.Marshal(map[string]interface{}{"skills": skillFiles}); err == nil {
					agentForSkills, getErr := k8s.GetAgent(c.Request.Context(), ns, name)
					if getErr == nil && agentForSkills.Status.PodName != "" && agentForSkills.Status.Phase == "Running" {
						podIP, ipErr := k8s.GetAgentPodIP(c.Request.Context(), ns, agentForSkills.Status.PodName)
						if ipErr == nil {
							if applyErr := k8s.ApplyAgentConfig(c.Request.Context(), ns, agentForSkills.Status.PodName, podIP, string(configJSON)); applyErr != nil {
								log.Printf("warning: skills config apply to pod failed for %s: %v", name, applyErr)
							}
						}
					}
				}
			}
		}

		// 2. If pod is running, forward config to the agent so it takes effect immediately.
		// If the model changed, read context_window from the response and patch the CR.
		var freshContextWindow int64
		agent, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err == nil && agent.Status.PodName != "" && agent.Status.Phase == "Running" {
			// Build config payload. If secretRefs changed, resolve all secrets to env-var key-value
			// pairs so the agent can apply the complete set (and remove any stale SECRET_* vars).
			type agentConfigPayload struct {
				Model        *string            `json:"model,omitempty"`
				Lifecycle    *string            `json:"lifecycle,omitempty"`
				Instructions *string            `json:"instructions,omitempty"`
				TemplateRef  *string            `json:"templateRef,omitempty"`
				Secrets      map[string]string  `json:"secrets"`
			}
			payload := agentConfigPayload{
				Model:        req.Model,
				Lifecycle:    req.Lifecycle,
				Instructions: req.Instructions,
				TemplateRef:  req.TemplateRef,
			}
			if req.SecretRefs != nil {
				// Full replacement: resolve all remaining secrets so the agent can remove stale ones.
				payload.Secrets = k8s.ResolveSecretEnvVars(c.Request.Context(), ns, *req.SecretRefs)
			} else if len(req.Secrets) > 0 {
				payload.Secrets = req.Secrets
			}
			configPayload, _ := json.Marshal(payload)
			podIP, ipErr := k8s.GetAgentPodIP(c.Request.Context(), ns, agent.Status.PodName)
			if ipErr == nil {
				if req.Model != nil && *req.Model != "" {
					// Read response to capture context_window
					if cw := k8s.ApplyAgentConfigGetContextWindow(c.Request.Context(), ns, agent.Status.PodName, podIP, string(configPayload)); cw > 0 {
						freshContextWindow = cw
						if patchErr := k8s.PatchAgentContextWindow(c.Request.Context(), ns, name, cw); patchErr != nil {
							log.Printf("warning: failed to patch context window for %s: %v", name, patchErr)
						}
					}
				} else {
					if applyErr := k8s.ApplyAgentConfig(c.Request.Context(), ns, agent.Status.PodName, podIP, string(configPayload)); applyErr != nil {
						log.Printf("warning: CR patched but config apply to pod failed for %s: %v", name, applyErr)
					}
				}
			}
		}

		// 3. Return updated agent.
		updated, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent patched but failed to read back: " + err.Error()})
			return
		}
		// Use freshContextWindow if we just fetched it (avoids race with CR status patch propagation).
		modelContextWindow := updated.Status.ModelContextWindow
		if freshContextWindow > 0 {
			modelContextWindow = freshContextWindow
		}
		c.JSON(http.StatusOK, AgentResponse{
			Name:            updated.Name,
			Namespace:       updated.Namespace,
			Model:           updated.Spec.Model,
			Status:          string(updated.Status.Phase),
			TaskStatus:      string(updated.Status.TaskStatus),
			LastTaskMessage: updated.Status.LastTaskMessage,
			Lifecycle:       string(updated.Spec.Lifecycle),
			LastTaskCostUSD: updated.Status.LastTaskCostUSD,
			TotalCostUSD:    updated.Status.TotalCostUSD,
			TotalTokens:        updated.Status.TotalTokens,
			ModelContextWindow: modelContextWindow,
			Secrets:            updated.Spec.Secrets,
			Memories:           updated.Spec.Memories,
			Skills:             updated.Spec.Skills,
			Instructions:       extractUserTask(updated.Spec.Instructions),
			CreatedAt:          updated.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}
