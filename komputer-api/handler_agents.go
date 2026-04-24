package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

type CreateAgentRequest struct {
	Name         string   `json:"name" binding:"required"`
	Instructions string   `json:"instructions" binding:"required"`
	Model        string   `json:"model"`
	TemplateRef  string   `json:"templateRef"`
	Role         string   `json:"role"`          // "manager" or "" (default manager)
	Namespace    string   `json:"namespace"`     // optional, defaults to server default
	SecretRefs   []string `json:"secretRefs"`   // names of existing K8s Secrets to attach
	Memories     []string `json:"memories"`     // optional KomputerMemory names to attach
	Skills       []string `json:"skills"`       // optional KomputerSkill names to attach
	Connectors   []string `json:"connectors"`   // optional KomputerConnector names to attach
	Lifecycle     string   `json:"lifecycle"`     // "", "Sleep", or "AutoDelete"
	OfficeManager string   `json:"officeManager"` // set by manager MCP tool
	SystemPrompt  string   `json:"systemPrompt"`  // optional custom system prompt
	Priority      int32    `json:"priority,omitempty"` // queue priority; higher = admitted first
	PodSpec       *corev1.PodSpec               `json:"podSpec,omitempty"`
	Storage       *komputerv1alpha1.StorageSpec `json:"storage,omitempty"`
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
	Connectors      []string `json:"connectors,omitempty"`   // KomputerConnector names attached to this agent
	Instructions    string   `json:"instructions,omitempty"` // User task (spec.instructions)
	SystemPrompt    string   `json:"systemPrompt,omitempty"` // Custom system prompt (spec.systemPrompt)
	CreatedAt       string   `json:"createdAt"`
	Priority        int32    `json:"priority"`
	QueuePosition   int32    `json:"queuePosition,omitempty"`
	QueueReason     string   `json:"queueReason,omitempty"`
	PodSpec         *corev1.PodSpec               `json:"podSpec,omitempty"`
	Storage         *komputerv1alpha1.StorageSpec `json:"storage,omitempty"`
	// Errors are non-fatal failures that occurred during the request (e.g. CR was patched
	// but live-pod sync failed). The CR change still took effect; the UI can surface these
	// as toasts so the user knows something didn't fully apply.
	Errors          []string                      `json:"errors,omitempty"`
}

// mergeDefaultSkills adds default skill names to the explicit list, deduplicating.
func mergeDefaultSkills(explicit []string, defaults []string) []string {
	seen := make(map[string]bool, len(explicit))
	for _, s := range explicit {
		seen[s] = true
	}
	merged := append([]string{}, explicit...)
	for _, s := range defaults {
		if !seen[s] {
			merged = append(merged, s)
		}
	}
	return merged
}

type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
}

type PatchAgentRequest struct {
	Model        *string   `json:"model,omitempty"`
	Lifecycle    *string   `json:"lifecycle,omitempty"`
	Instructions *string   `json:"instructions,omitempty"`
	TemplateRef  *string   `json:"templateRef,omitempty"`
	SecretRefs   *[]string `json:"secretRefs,omitempty"`  // full replacement list of K8s secret names
	Memories     *[]string `json:"memories,omitempty"`    // memory names to attach
	Skills       *[]string `json:"skills,omitempty"`      // skill names to attach
	Connectors   *[]string `json:"connectors,omitempty"`  // connector names to attach
	SystemPrompt *string   `json:"systemPrompt,omitempty"` // custom system prompt
	Priority     *int32    `json:"priority,omitempty"`    // pointer so 0 vs unset is distinguishable
	PodSpec      *corev1.PodSpec               `json:"podSpec,omitempty"`
	Storage      *komputerv1alpha1.StorageSpec `json:"storage,omitempty"`
}

// createOrTriggerAgent creates a new agent or sends a task to an existing one.
// @ID createAgent
// @Summary Create agent or send task
// @Description Creates a new agent or sends a task to an existing idle agent (upsert by name).
// @Description If the agent doesn't exist, it is created. If it exists and is idle, the task is forwarded.
// @Tags agents
// @Accept json
// @Produce json
// @Param request body CreateAgentRequest true "Agent creation request"
// @Success 200 {object} AgentResponse "Agent created or task forwarded"
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
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())

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

		// Build internal system prompt — memory resolution happens after we know if agent exists
		// (existing agents use CR memories, new agents use request memories)
		buildInternalSystemPrompt := func(memories []string) string {
			memorySection, _ := k8s.ResolveMemoryContent(c.Request.Context(), ns, memories)
			agentHeader := fmt.Sprintf("\n---\n\n**Agent Name:** %s", req.Name)
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
			wakeSystemPrompt := req.SystemPrompt
				if wakeSystemPrompt == "" {
					wakeSystemPrompt = existing.Spec.SystemPrompt
				}
				if err := k8s.WakeAgent(c.Request.Context(), ns, req.Name, instructions, buildInternalSystemPrompt(wakeMemories), wakeSystemPrompt, req.Model, req.Lifecycle); err != nil {
					agentActionsTotal.WithLabelValues("wake", "error").Inc()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to wake agent: " + err.Error()})
					return
				}
				log.Printf("waking sleeping agent %s/%s", ns, req.Name)
				agentActionsTotal.WithLabelValues("wake", "success").Inc()
				c.JSON(http.StatusOK, AgentResponse{
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
					Skills:          mergeDefaultSkills(existing.Spec.Skills, defaultSkills),
					Connectors:      existing.Spec.Connectors,
					Instructions:    existing.Spec.Instructions,
					SystemPrompt:    existing.Spec.SystemPrompt,
					CreatedAt:       existing.CreationTimestamp.UTC().Format(time.RFC3339),
					Priority:        existing.Spec.Priority,
					QueuePosition:   existing.Status.QueuePosition,
					QueueReason:     existing.Status.QueueReason,
					PodSpec:         existing.Spec.PodSpec,
					Storage:         existing.Spec.Storage,
				})
				return
			}

			if existing.Status.PodName == "" {
				c.JSON(http.StatusConflict, gin.H{"error": "agent exists but has no running pod yet"})
				return
			}

			// If the agent is busy, the message is queued as a steer (follow-up).
			// The agent's /task endpoint handles both new tasks and steers.

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
			forwardSystemPrompt := req.SystemPrompt
			if forwardSystemPrompt == "" {
				forwardSystemPrompt = existing.Spec.SystemPrompt
			}
			cw, err := k8s.ForwardTaskToAgent(c.Request.Context(), ns, existing.Status.PodName, podIP, instructions, req.Model, buildInternalSystemPrompt(forwardMemories), forwardSystemPrompt)
			if err != nil {
				agentActionsTotal.WithLabelValues("wake", "error").Inc()
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
			agentActionsTotal.WithLabelValues("wake", "success").Inc()
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
				Skills:          mergeDefaultSkills(existing.Spec.Skills, defaultSkills),
				Connectors:      existing.Spec.Connectors,
				Instructions:    existing.Spec.Instructions,
				SystemPrompt:    existing.Spec.SystemPrompt,
				CreatedAt:       existing.CreationTimestamp.UTC().Format(time.RFC3339),
				Priority:        existing.Spec.Priority,
				QueuePosition:   existing.Status.QueuePosition,
				QueueReason:     existing.Status.QueueReason,
				PodSpec:         existing.Spec.PodSpec,
				Storage:         existing.Spec.Storage,
			})
			return
		}

		if err := k8s.EnsureNamespace(c.Request.Context(), ns); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure namespace: " + err.Error()})
			return
		}

		// Inherit connectors from office manager so sub-agents get the same MCP tools.
		connectors := req.Connectors
		if req.OfficeManager != "" {
			manager, mgrErr := k8s.GetAgent(c.Request.Context(), ns, req.OfficeManager)
			if mgrErr == nil && len(manager.Spec.Connectors) > 0 {
				seen := make(map[string]bool, len(connectors))
				for _, c := range connectors {
					seen[c] = true
				}
				for _, c := range manager.Spec.Connectors {
					if !seen[c] {
						connectors = append(connectors, c)
					}
				}
			}
		}

		agent, err := k8s.CreateAgent(c.Request.Context(), ns, req.Name, instructions, buildInternalSystemPrompt(req.Memories), req.SystemPrompt, req.Model, req.TemplateRef, role, req.SecretRefs, req.Memories, req.Skills, connectors, req.Lifecycle, req.OfficeManager, req.Priority, req.PodSpec, req.Storage)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "agent already exists: " + req.Name})
				return
			}
			agentActionsTotal.WithLabelValues("create", "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent: " + err.Error()})
			return
		}

		log.Printf("created new agent %s/%s", ns, req.Name)
		agentActionsTotal.WithLabelValues("create", "success").Inc()
		c.JSON(http.StatusOK, AgentResponse{
			Name:         agent.Name,
			Namespace:    agent.Namespace,
			Model:        agent.Spec.Model,
			Status:       "Pending",
			Lifecycle:    string(agent.Spec.Lifecycle),
			Secrets:      collectSecretKeys(*c, k8s, ns, agent.Spec.Secrets),
			Memories:     agent.Spec.Memories,
			Skills:       mergeDefaultSkills(agent.Spec.Skills, defaultSkills),
			Connectors:   agent.Spec.Connectors,
			Instructions: agent.Spec.Instructions,
			SystemPrompt: agent.Spec.SystemPrompt,
			CreatedAt:    agent.CreationTimestamp.UTC().Format(time.RFC3339),
			TotalTokens:        agent.Status.TotalTokens,
			ModelContextWindow: agent.Status.ModelContextWindow,
			Priority:     agent.Spec.Priority,
			PodSpec:      agent.Spec.PodSpec,
			Storage:      agent.Spec.Storage,
		})
	}
}

// deleteAgent deletes an agent and cleans up all its resources.
// @ID deleteAgent
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
			agentActionsTotal.WithLabelValues("delete", "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete agent: " + err.Error()})
			return
		}
		// Clean up the agent's Redis event stream
		if err := DeleteAgentStream(c.Request.Context(), worker.Rdb, name, worker.StreamPrefix); err != nil {
			log.Printf("warning: failed to delete event stream for %s: %v", name, err)
		}
		log.Printf("deleted agent %s/%s", ns, name)
		agentActionsTotal.WithLabelValues("delete", "success").Inc()
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

// cancelAgentTask cancels the running task on an agent.
// @ID cancelAgentTask
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
			agentActionsTotal.WithLabelValues("cancel", "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel: " + err.Error()})
			return
		}

		log.Printf("cancelled task on agent %s/%s", ns, name)
		agentActionsTotal.WithLabelValues("cancel", "success").Inc()
		c.JSON(http.StatusOK, gin.H{"status": "cancelling", "name": name})
	}
}

// getAgent returns details for a single agent.
// @ID getAgent
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
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())
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
			Skills:             mergeDefaultSkills(agent.Spec.Skills, defaultSkills),
			Connectors:         agent.Spec.Connectors,
			Instructions:       agent.Spec.Instructions,
			SystemPrompt:       agent.Spec.SystemPrompt,
			CreatedAt:          agent.CreationTimestamp.UTC().Format(time.RFC3339),
			Priority:           agent.Spec.Priority,
			QueuePosition:      agent.Status.QueuePosition,
			QueueReason:        agent.Status.QueueReason,
			PodSpec:            agent.Spec.PodSpec,
			Storage:            agent.Spec.Storage,
		})
	}
}

// getAgentEvents returns the event history for an agent from Redis.
// @ID getAgentEvents
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
func getAgentEvents(worker *RedisWorker, k8s *K8sClient) gin.HandlerFunc {
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
		after := c.Query("after")   // RFC-3339 cursor — return events strictly after this timestamp
		around := c.Query("around") // RFC-3339 timestamp to center results on

		var events []AgentEvent
		var err error
		if around != "" {
			events, err = GetAgentEventsAround(c.Request.Context(), worker.Rdb, name, limit, around, worker.StreamPrefix)
		} else if after != "" {
			events, err = GetAgentEventsAfter(c.Request.Context(), worker.Rdb, name, limit, after, worker.StreamPrefix)
		} else {
			events, err = GetAgentEventsBefore(c.Request.Context(), worker.Rdb, name, limit, before, worker.StreamPrefix)
		}
		if err != nil {
			events = nil
		}

		// Hybrid: if Redis is empty (wiped) and agent has a session, backfill from JSONL.
		if len(events) == 0 && before == "" {
			ns := resolveNamespace(c, k8s)
			agent, getErr := k8s.GetAgent(c.Request.Context(), ns, name)
			if getErr == nil && agent.Status.PodName != "" && agent.Status.SessionID != "" {
				// Fetch ALL events (no limit) so we backfill the full history.
				var totalCost float64
				if agent.Status.TotalCostUSD != "" {
					fmt.Sscanf(agent.Status.TotalCostUSD, "%f", &totalCost)
				}
				allSessionEvents := fetchSessionEvents(c.Request.Context(), k8s, ns, agent.Status.PodName, agent.Status.SessionID, name, 0, agent.Spec.Model, totalCost)
				if len(allSessionEvents) > 0 {
					// Backfill synchronously so pagination works on the next request.
					backfillRedisHistory(worker.Rdb, name, allSessionEvents)
					// Serve only the requested limit.
					if int64(len(allSessionEvents)) > limit {
						events = allSessionEvents[int64(len(allSessionEvents))-limit:]
					} else {
						events = allSessionEvents
					}
				}
			}
		}

		if events == nil {
			events = []AgentEvent{}
		}
		c.JSON(http.StatusOK, gin.H{"agent": name, "events": events})
	}
}

// listAgents returns all agents in a namespace.
// @ID listAgents
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
		statusFilter := c.Query("status")
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())
		agents, err := k8s.ListAgents(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agents: " + err.Error()})
			return
		}

		resp := AgentListResponse{Agents: make([]AgentResponse, 0, len(agents))}
		for _, a := range agents {
			if statusFilter != "" && !strings.EqualFold(statusFilter, string(a.Status.Phase)) {
				continue
			}
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
				Skills:             mergeDefaultSkills(a.Spec.Skills, defaultSkills),
				Connectors:         a.Spec.Connectors,
				Instructions:       a.Spec.Instructions,
				SystemPrompt:       a.Spec.SystemPrompt,
				CreatedAt:          a.CreationTimestamp.UTC().Format(time.RFC3339),
				Priority:           a.Spec.Priority,
				QueuePosition:      a.Status.QueuePosition,
				QueueReason:        a.Status.QueueReason,
				PodSpec:            a.Spec.PodSpec,
				Storage:            a.Spec.Storage,
			})
		}

		c.JSON(http.StatusOK, resp)
	}
}

// patchAgent updates fields on an existing agent.
// @ID patchAgent
// @Summary Patch agent
// @Description Updates model, lifecycle, instructions, secretRefs, memories, skills, or connectors on an existing agent.
// @Tags agents
// @Accept json
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Param request body PatchAgentRequest true "Fields to update"
// @Success 200 {object} AgentResponse "Updated agent"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name} [patch]
func patchAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())

		var req PatchAgentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Model == nil && req.Lifecycle == nil && req.Instructions == nil && req.TemplateRef == nil && req.SecretRefs == nil && req.Memories == nil && req.Skills == nil && req.Connectors == nil && req.SystemPrompt == nil && req.Priority == nil && req.PodSpec == nil && req.Storage == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		var nonFatalErrors []string

		// Determine action label for metrics: "sleep" if lifecycle → Sleep, else "patch".
		patchAction := "patch"
		if req.Lifecycle != nil && *req.Lifecycle == "Sleep" {
			patchAction = "sleep"
		}

		// 1. Patch CR spec first — this is the source of truth.
		if err := k8s.PatchAgentSpec(c.Request.Context(), ns, name, req.Model, req.Lifecycle, req.Instructions, req.TemplateRef, req.SystemPrompt, req.Priority); err != nil {
			agentActionsTotal.WithLabelValues(patchAction, "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch agent: " + err.Error()})
			return
		}

		// 1a-override. Patch podSpec / storage overrides if provided.
		if req.PodSpec != nil || req.Storage != nil {
			if err := k8s.PatchAgentOverrides(c.Request.Context(), ns, name, req.PodSpec, req.Storage); err != nil {
				agentActionsTotal.WithLabelValues(patchAction, "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch agent overrides: " + err.Error()})
				return
			}
		}

		// 1a. If model changed and pod is running, fetch context window via the agent (it has the API key).
		// This is handled below in step 2 where we forward config to the pod and read the response.

		// 1b. Replace secret refs list if provided.
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
								log.Printf("error: skills config apply to pod failed for %s: %v", name, applyErr)
								nonFatalErrors = append(nonFatalErrors, fmt.Sprintf("skills sync to running pod failed: %v", applyErr))
							}
						}
					}
				}
			}
		}

		// 1e. Update connectors if provided.
		if req.Connectors != nil {
			k8s.PatchAgentConnectorsList(c.Request.Context(), ns, name, *req.Connectors)
		}

		// 2. If pod is running, forward config to the agent so it takes effect immediately.
		// If the model changed, read context_window from the response and patch the CR.
		var freshContextWindow int64
		agent, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err == nil && agent.Status.PodName != "" && agent.Status.Phase == "Running" {
			type agentConfigPayload struct {
				Model        *string                 `json:"model,omitempty"`
				Lifecycle    *string                 `json:"lifecycle,omitempty"`
				Instructions *string                 `json:"instructions,omitempty"`
				TemplateRef  *string                 `json:"templateRef,omitempty"`
				Secrets      map[string]string       `json:"secrets,omitempty"`
				McpServers   *map[string]interface{} `json:"mcp_servers,omitempty"`
			}
			payload := agentConfigPayload{
				Model:        req.Model,
				Lifecycle:    req.Lifecycle,
				Instructions: req.Instructions,
				TemplateRef:  req.TemplateRef,
			}
			// If secretRefs changed, resolve all secrets so the agent can set/remove SECRET_* env vars.
			if req.SecretRefs != nil {
				payload.Secrets = k8s.ResolveSecretEnvVars(c.Request.Context(), ns, *req.SecretRefs)
			}
			// If connectors changed, resolve MCP configs so the agent updates KOMPUTER_MCP_SERVERS.
			if req.Connectors != nil {
				mcpServers := k8s.ResolveConnectorMCPConfigs(c.Request.Context(), ns, *req.Connectors)
				payload.McpServers = &mcpServers
			}
			// Skip if payload has no fields the agent's /config endpoint understands
			// (e.g. only skills/memories changed — those are applied separately above).
			hasPayloadFields := payload.Model != nil || payload.Lifecycle != nil ||
				payload.Instructions != nil || payload.TemplateRef != nil ||
				payload.Secrets != nil || payload.McpServers != nil
			configPayload, _ := json.Marshal(payload)
			podIP, ipErr := k8s.GetAgentPodIP(c.Request.Context(), ns, agent.Status.PodName)
			if ipErr == nil && hasPayloadFields {
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
						log.Printf("error: CR patched but config apply to pod failed for %s: %v", name, applyErr)
						nonFatalErrors = append(nonFatalErrors, fmt.Sprintf("config sync to running pod failed: %v", applyErr))
					}
				}
			}
		}

		// 3. Return updated agent.
		updated, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err != nil {
			agentActionsTotal.WithLabelValues(patchAction, "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent patched but failed to read back: " + err.Error()})
			return
		}
		// Use freshContextWindow if we just fetched it (avoids race with CR status patch propagation).
		modelContextWindow := updated.Status.ModelContextWindow
		if freshContextWindow > 0 {
			modelContextWindow = freshContextWindow
		}
		agentActionsTotal.WithLabelValues(patchAction, "success").Inc()
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
			Skills:             mergeDefaultSkills(updated.Spec.Skills, defaultSkills),
			Instructions:       updated.Spec.Instructions,
			SystemPrompt:       updated.Spec.SystemPrompt,
			Connectors:         updated.Spec.Connectors,
			CreatedAt:          updated.CreationTimestamp.UTC().Format(time.RFC3339),
			Priority:           updated.Spec.Priority,
			QueuePosition:      updated.Status.QueuePosition,
			QueueReason:        updated.Status.QueueReason,
			PodSpec:            updated.Spec.PodSpec,
			Storage:            updated.Spec.Storage,
			Errors:             nonFatalErrors,
		})
	}
}
