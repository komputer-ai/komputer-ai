package main

import (
	"fmt"
	"log"
	"net/http"
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
	Lifecycle    string            `json:"lifecycle"` // "", "Sleep", or "AutoDelete"
}

type AgentResponse struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Model           string `json:"model"`
	Status          string `json:"status"`
	TaskStatus      string `json:"taskStatus,omitempty"`
	LastTaskMessage string `json:"lastTaskMessage,omitempty"`
	Lifecycle       string `json:"lifecycle,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
}

// resolveNamespace returns the namespace from the query param, request body, or default.
func resolveNamespace(c *gin.Context, k8s *K8sClient) string {
	if ns := c.Query("namespace"); ns != "" {
		return ns
	}
	return k8s.defaultNamespace
}

func SetupRoutes(r *gin.Engine, k8s *K8sClient, hub *Hub, worker *RedisWorker) {
	v1 := r.Group("/api/v1")
	{
		v1.POST("/agents", createOrTriggerAgent(k8s))
		v1.GET("/agents", listAgents(k8s))
		v1.GET("/agents/:name", getAgent(k8s))
		v1.GET("/agents/:name/events", getAgentEvents(worker))
		v1.DELETE("/agents/:name", deleteAgent(k8s, worker))
		v1.POST("/agents/:name/cancel", cancelAgentTask(k8s))
		v1.GET("/agents/:name/ws", HandleAgentWS(hub))
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
		agentHeader := fmt.Sprintf("\n---\n\n**Agent Name:** %s\n\n## Your Task\n", req.Name)
		instructions := req.Instructions
		if role == "manager" {
			instructions = managerSystemPrompt + agentHeader + req.Instructions
		} else {
			instructions = workerSystemPrompt + agentHeader + req.Instructions
		}

		existing, err := k8s.GetAgent(c.Request.Context(), ns, req.Name)
		if err != nil && !errors.IsNotFound(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check agent: " + err.Error()})
			return
		}

		if existing != nil {
			// Wake-up flow for sleeping agents
			if existing.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
				if err := k8s.WakeAgent(c.Request.Context(), ns, req.Name, instructions, req.Model, req.Lifecycle); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to wake agent: " + err.Error()})
					return
				}
				log.Printf("waking sleeping agent %s/%s", ns, req.Name)
				c.JSON(http.StatusAccepted, AgentResponse{
					Name:      existing.Name,
					Namespace: existing.Namespace,
					Model:     existing.Spec.Model,
					Status:    "Pending",
					CreatedAt: existing.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
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

			if err := k8s.ForwardTaskToAgent(c.Request.Context(), ns, existing.Status.PodName, podIP, req.Instructions, req.Model); err != nil {
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

		agent, err := k8s.CreateAgent(c.Request.Context(), ns, req.Name, instructions, req.Model, req.TemplateRef, role, secretNames, req.Lifecycle)
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
		ns := resolveNamespace(c, k8s)
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
				CreatedAt:       a.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			})
		}

		c.JSON(http.StatusOK, resp)
	}
}
