package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"

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
	Secrets      map[string]string `json:"secrets"`   // optional key-value secrets
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
	Secrets         []string `json:"secrets,omitempty"` // Key names from K8s Secrets (not values)
	CreatedAt       string   `json:"createdAt"`
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
	LastRunStatus  string `json:"lastRunStatus,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

type ScheduleListResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
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
		agentHeader := fmt.Sprintf("\n---\n\n**Agent Name:** %s\n\n## Your Task\n", req.Name)
		var systemPrompt string
		if role == "manager" {
			systemPrompt = managerSystemPrompt + agentHeader
		} else {
			systemPrompt = workerSystemPrompt + agentHeader
		}
		instructions := req.Instructions

		existing, err := k8s.GetAgent(c.Request.Context(), ns, req.Name)
		if err != nil && !errors.IsNotFound(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check agent: " + err.Error()})
			return
		}

		if existing != nil {
			// Wake-up flow for sleeping agents
			if existing.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
				if err := k8s.WakeAgent(c.Request.Context(), ns, req.Name, systemPrompt+"\n\n"+instructions, req.Model, req.Lifecycle); err != nil {
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
					Secrets:         collectSecretKeys(*c, k8s, ns, existing.Spec.Secrets),
					CreatedAt:       existing.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
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

			if err := k8s.ForwardTaskToAgent(c.Request.Context(), ns, existing.Status.PodName, podIP, instructions, req.Model, systemPrompt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to forward task: " + err.Error()})
				return
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
				Secrets:         collectSecretKeys(*c, k8s, ns, existing.Spec.Secrets),
				CreatedAt:       existing.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
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

		agent, err := k8s.CreateAgent(c.Request.Context(), ns, req.Name, systemPrompt+"\n\n"+instructions, req.Model, req.TemplateRef, role, secretNames, req.Lifecycle, req.OfficeManager)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent: " + err.Error()})
			return
		}

		log.Printf("created new agent %s/%s", ns, req.Name)
		c.JSON(http.StatusCreated, AgentResponse{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Model:     agent.Spec.Model,
			Status:    "Pending",
			Lifecycle: string(agent.Spec.Lifecycle),
			Secrets:   collectSecretKeys(*c, k8s, ns, agent.Spec.Secrets),
			CreatedAt: agent.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
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
			Secrets:         agent.Spec.Secrets,
			CreatedAt:       agent.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
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
		events, err := GetAgentEvents(c.Request.Context(), worker.Rdb, name, limit, worker.StreamPrefix)
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
				Secrets:         collectSecretKeys(*c, k8s, ns, a.Spec.Secrets),
				CreatedAt:       a.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
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
		CreatedAt:       o.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
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
		LastRunStatus:  s.Status.LastRunStatus,
		CreatedAt:      s.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
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
