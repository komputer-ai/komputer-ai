package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
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
					Skills:          mergeDefaultSkills(existing.Spec.Skills, defaultSkills),
					Connectors:      existing.Spec.Connectors,
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
				Skills:          mergeDefaultSkills(existing.Spec.Skills, defaultSkills),
				Connectors:      existing.Spec.Connectors,
				Instructions:    extractUserTask(existing.Spec.Instructions),
				CreatedAt:       existing.CreationTimestamp.UTC().Format(time.RFC3339),
			})
			return
		}

		if err := k8s.EnsureNamespace(c.Request.Context(), ns); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure namespace: " + err.Error()})
			return
		}

		agent, err := k8s.CreateAgent(c.Request.Context(), ns, req.Name, buildSystemPrompt(req.Memories)+"\n\n"+instructions, req.Model, req.TemplateRef, role, req.SecretRefs, req.Memories, req.Skills, req.Connectors, req.Lifecycle, req.OfficeManager)
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
			Skills:       mergeDefaultSkills(agent.Spec.Skills, defaultSkills),
			Connectors:   agent.Spec.Connectors,
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

		events, err := GetAgentEventsBefore(c.Request.Context(), worker.Rdb, name, limit, before, worker.StreamPrefix)
		if err != nil {
			events = nil
		}

		// Hybrid: if Redis is empty (wiped) and agent has a session, backfill from JSONL.
		if len(events) == 0 && before == "" {
			ns := resolveNamespace(c, k8s)
			agent, getErr := k8s.GetAgent(c.Request.Context(), ns, name)
			if getErr == nil && agent.Status.PodName != "" && agent.Status.SessionID != "" {
				// Fetch ALL events (no limit) so we backfill the full history.
				allSessionEvents := fetchSessionEvents(c.Request.Context(), k8s, ns, agent.Status.PodName, agent.Status.SessionID, name, 0)
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
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())
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
				Skills:             mergeDefaultSkills(a.Spec.Skills, defaultSkills),
				Connectors:         a.Spec.Connectors,
				Instructions:       extractUserTask(a.Spec.Instructions),
				CreatedAt:          a.CreationTimestamp.UTC().Format(time.RFC3339),
			})
		}

		c.JSON(http.StatusOK, resp)
	}
}

// patchAgent updates fields on an existing agent.
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
		if req.Model == nil && req.Lifecycle == nil && req.Instructions == nil && req.TemplateRef == nil && req.SecretRefs == nil && req.Memories == nil && req.Skills == nil && req.Connectors == nil {
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
								log.Printf("warning: skills config apply to pod failed for %s: %v", name, applyErr)
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
			Skills:             mergeDefaultSkills(updated.Spec.Skills, defaultSkills),
			Instructions:       extractUserTask(updated.Spec.Instructions),
			Connectors:         updated.Spec.Connectors,
			CreatedAt:          updated.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

// parseSessionJSONL converts raw Claude session JSONL bytes into AgentEvent structs.
// fetchSessionEvents reads Claude session JSONL from the agent pod and parses it into AgentEvents.
// Uses HTTP to agent pod, falls back to exec (instant in LOCAL mode).
func fetchSessionEvents(ctx context.Context, k8s *K8sClient, ns, podName, sessionID, agentName string, limit int64) []AgentEvent {
	var raw []byte

	// Try HTTP first (skipped instantly in LOCAL mode).
	podIP, _ := k8s.GetAgentPodIP(ctx, ns, podName)
	if podIP != "" {
		historyPath := fmt.Sprintf("/history?limit=%d&session_id=%s", limit, sessionID)
		respBody, err := k8s.getFromAgent(ctx, podIP, historyPath)
		if err == nil {
			// Parse agent's /history response and convert to AgentEvent.
			var result struct {
				Events []struct {
					Type      string                 `json:"type"`
					Timestamp string                 `json:"timestamp"`
					Payload   map[string]interface{} `json:"payload"`
				} `json:"events"`
			}
			if json.Unmarshal(respBody, &result) == nil {
				var events []AgentEvent
				for _, e := range result.Events {
					events = append(events, AgentEvent{
						AgentName: agentName,
						Type:      e.Type,
						Timestamp: e.Timestamp,
						Payload:   e.Payload,
					})
				}
				if limit > 0 && int64(len(events)) > limit {
					events = events[int64(len(events))-limit:]
				}
				return events
			}
		}
	}

	// Exec fallback: cat the JSONL file directly.
	sessionFilePath := fmt.Sprintf("/workspace/.claude/projects/-workspace/%s.jsonl", sessionID)
	var err error
	raw, err = k8s.execInPodWithOutput(ctx, ns, podName, "cat", sessionFilePath)
	if err != nil {
		log.Printf("session JSONL read failed for %s: %v", agentName, err)
		return nil
	}

	return parseSessionJSONL(raw, agentName, limit)
}

// backfillRedisHistory writes session events into Redis history so subsequent requests are fast.
func backfillRedisHistory(rdb *redis.Client, agentName string, events []AgentEvent) {
	ctx := context.Background()
	historyKey := fmt.Sprintf("komputer-history:%s", agentName)

	for _, event := range events {
		raw, err := json.Marshal(event)
		if err != nil {
			continue
		}
		rdb.RPush(ctx, historyKey, raw)
	}
	log.Printf("backfilled %d events from session to Redis for %s", len(events), agentName)
}

// parseSessionJSONL converts raw Claude session JSONL bytes into AgentEvent structs.
func parseSessionJSONL(raw []byte, agentName string, limit int64) []AgentEvent {
	var events []AgentEvent

	// Track tool_use blocks by ID so we can merge output from tool_result entries.
	type toolUseInfo struct {
		name      string
		input     interface{}
		timestamp string
		index     int // position in events slice
	}
	pendingTools := map[string]*toolUseInfo{}

	// Track per-turn stats for task_completed events.
	var lastAssistantTimestamp string
	var turnInputTokens, turnOutputTokens, turnCacheRead, turnCacheCreation float64
	var turnAssistantMessages int
	firstTurn := true

	emitTaskCompleted := func() {
		if lastAssistantTimestamp == "" {
			return
		}
		payload := map[string]interface{}{
			"num_turns": turnAssistantMessages,
			"usage": map[string]interface{}{
				"input_tokens":                  int64(turnInputTokens),
				"output_tokens":                 int64(turnOutputTokens),
				"cache_read_input_tokens":       int64(turnCacheRead),
				"cache_creation_input_tokens":   int64(turnCacheCreation),
			},
		}
		events = append(events, AgentEvent{
			AgentName: agentName,
			Type:      "task_completed",
			Timestamp: lastAssistantTimestamp,
			Payload:   payload,
		})
		// Reset accumulators.
		lastAssistantTimestamp = ""
		turnInputTokens = 0
		turnOutputTokens = 0
		turnCacheRead = 0
		turnCacheCreation = 0
		turnAssistantMessages = 0
	}

	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Detect turn boundaries via queue-operation enqueue.
		entryType, _ := entry["type"].(string)
		if entryType == "queue-operation" {
			op, _ := entry["operation"].(string)
			if op == "enqueue" && !firstTurn {
				emitTaskCompleted()
			}
			firstTurn = false
			continue
		}

		msg, _ := entry["message"].(map[string]interface{})
		if msg == nil {
			continue
		}
		role, _ := msg["role"].(string)
		timestamp, _ := entry["timestamp"].(string)

		if role == "user" {
			// User messages contain either text (actual user input) or tool_result blocks.
			content := msg["content"]
			var text string
			switch v := content.(type) {
			case string:
				text = v
			case []interface{}:
				for _, block := range v {
					b, ok := block.(map[string]interface{})
					if !ok {
						continue
					}
					btype, _ := b["type"].(string)
					if btype == "text" {
						if t, ok := b["text"].(string); ok {
							text += t + " "
						}
					} else if btype == "tool_result" {
						// Merge output into the matching tool_call event.
						toolUseID, _ := b["tool_use_id"].(string)
						output := ""
						if c, ok := b["content"].(string); ok {
							output = c
						} else if c, ok := b["content"].([]interface{}); ok && len(c) > 0 {
							if first, ok := c[0].(map[string]interface{}); ok {
								output, _ = first["text"].(string)
							}
						}
						if len(output) > 500 {
							output = output[:500] + "..."
						}
						if info, ok := pendingTools[toolUseID]; ok {
							// Wrap in structured format matching real-time agent events.
							events[info.index].Payload["output"] = map[string]interface{}{
								"stdout": output,
								"stderr": "",
							}
							events[info.index].Type = "tool_result"
							delete(pendingTools, toolUseID)
						}
					}
				}
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			// Skip IDE context messages (file open, selection, etc.) — not real user input.
			if strings.HasPrefix(text, "<ide_") || strings.HasPrefix(text, "<system-reminder") {
				continue
			}
			// The agent prepends the system prompt to user messages joined by "\n\n".
			// Extract only the user's actual instruction (the last section).
			if parts := strings.Split(text, "\n\n"); len(parts) > 1 {
				text = strings.TrimSpace(parts[len(parts)-1])
			}
			// After stripping, skip if it now looks like IDE/system context.
			if text == "" || strings.HasPrefix(text, "<ide_") || strings.HasPrefix(text, "<system-reminder") {
				continue
			}
			events = append(events, AgentEvent{
				AgentName: agentName,
				Type:      "user_message",
				Timestamp: timestamp,
				Payload:   map[string]interface{}{"content": text},
			})
		} else if role == "assistant" {
			lastAssistantTimestamp = timestamp
			turnAssistantMessages++
			// Accumulate usage from assistant message.
			if usage, ok := msg["usage"].(map[string]interface{}); ok {
				if v, ok := usage["input_tokens"].(float64); ok {
					turnInputTokens += v
				}
				if v, ok := usage["output_tokens"].(float64); ok {
					turnOutputTokens += v
				}
				if v, ok := usage["cache_read_input_tokens"].(float64); ok {
					turnCacheRead += v
				}
				if v, ok := usage["cache_creation_input_tokens"].(float64); ok {
					turnCacheCreation += v
				}
			}
			content, _ := msg["content"].([]interface{})
			if content == nil {
				if s, ok := msg["content"].(string); ok && s != "" {
					events = append(events, AgentEvent{
						AgentName: agentName,
						Type:      "text",
						Timestamp: timestamp,
						Payload:   map[string]interface{}{"content": s},
					})
				}
				continue
			}
			for _, block := range content {
				b, ok := block.(map[string]interface{})
				if !ok {
					continue
				}
				btype, _ := b["type"].(string)
				switch btype {
				case "text":
					text, _ := b["text"].(string)
					if text != "" {
						events = append(events, AgentEvent{
							AgentName: agentName,
							Type:      "text",
							Timestamp: timestamp,
							Payload:   map[string]interface{}{"content": text},
						})
					}
				case "thinking":
					thinking, _ := b["thinking"].(string)
					if thinking != "" {
						events = append(events, AgentEvent{
							AgentName: agentName,
							Type:      "thinking",
							Timestamp: timestamp,
							Payload:   map[string]interface{}{"content": thinking},
						})
					}
				case "tool_use":
					toolID, _ := b["id"].(string)
					toolName, _ := b["name"].(string)
					idx := len(events)
					events = append(events, AgentEvent{
						AgentName: agentName,
						Type:      "tool_result", // will have output merged from tool_result entry
						Timestamp: timestamp,
						Payload: map[string]interface{}{
							"tool":  toolName,
							"name":  toolName,
							"input": b["input"],
						},
					})
					if toolID != "" {
						pendingTools[toolID] = &toolUseInfo{name: toolName, input: b["input"], timestamp: timestamp, index: idx}
					}
				}
			}
		}
	}

	// Emit task_completed for the final turn.
	emitTaskCompleted()

	// Return last N events.
	if limit > 0 && int64(len(events)) > limit {
		events = events[int64(len(events))-limit:]
	}
	return events
}
