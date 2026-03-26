package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
)

type CreateAgentRequest struct {
	Name         string `json:"name" binding:"required"`
	Instructions string `json:"instructions" binding:"required"`
	Model        string `json:"model"`
	TemplateRef  string `json:"templateRef"`
	Role         string `json:"role"` // "manager" or "" (default worker)
}

type AgentResponse struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Model           string `json:"model"`
	Status          string `json:"status"`
	TaskStatus      string `json:"taskStatus,omitempty"`
	LastTaskMessage string `json:"lastTaskMessage,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
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

func createOrTriggerAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAgentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		existing, err := k8s.GetAgent(c.Request.Context(), req.Name)
		if err != nil && !errors.IsNotFound(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check agent: " + err.Error()})
			return
		}

		if existing != nil {
			if existing.Status.PodName == "" {
				c.JSON(http.StatusConflict, gin.H{"error": "agent exists but has no running pod yet"})
				return
			}

			if existing.Status.TaskStatus == komputerv1alpha1.AgentTaskBusy {
				c.JSON(http.StatusConflict, gin.H{"error": "agent is busy with another task"})
				return
			}

			podIP, err := k8s.GetAgentPodIP(c.Request.Context(), existing.Status.PodName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent pod IP: " + err.Error()})
				return
			}

			if err := k8s.ForwardTaskToAgent(c.Request.Context(), existing.Status.PodName, podIP, req.Instructions, req.Model); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to forward task: " + err.Error()})
				return
			}

			log.Printf("forwarded task to existing agent %s", req.Name)
			c.JSON(http.StatusOK, AgentResponse{
				Name:            existing.Name,
				Namespace:       existing.Namespace,
				Model:           existing.Spec.Model,
				Status:          string(existing.Status.Phase),
				TaskStatus:      string(existing.Status.TaskStatus),
				LastTaskMessage: existing.Status.LastTaskMessage,
				CreatedAt:       existing.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			})
			return
		}

		instructions := req.Instructions
		role := req.Role
		if role == "" {
			role = "manager"
		}
		if role != "worker" && role != "manager" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'worker' or 'manager'"})
			return
		}
		if role == "manager" {
			instructions = managerSystemPrompt + "\n---\n\n## Your Task\n" + req.Instructions
		}

		agent, err := k8s.CreateAgent(c.Request.Context(), req.Name, instructions, req.Model, req.TemplateRef, role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent: " + err.Error()})
			return
		}

		log.Printf("created new agent %s", req.Name)
		c.JSON(http.StatusCreated, AgentResponse{
			Name:      agent.Name,
			Namespace: agent.Namespace,
			Model:     agent.Spec.Model,
			Status:    "Pending",
			CreatedAt: agent.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		})
	}
}

func deleteAgent(k8s *K8sClient, worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := k8s.DeleteAgent(c.Request.Context(), name); err != nil {
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
		log.Printf("deleted agent %s", name)
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

func cancelAgentTask(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")

		agent, err := k8s.GetAgent(c.Request.Context(), name)
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

		podIP, err := k8s.GetAgentPodIP(c.Request.Context(), agent.Status.PodName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pod IP: " + err.Error()})
			return
		}

		if err := k8s.CancelAgentTask(c.Request.Context(), agent.Status.PodName, podIP); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel: " + err.Error()})
			return
		}

		log.Printf("cancelled task on agent %s", name)
		c.JSON(http.StatusOK, gin.H{"status": "cancelling", "name": name})
	}
}

func getAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		agent, err := k8s.GetAgent(c.Request.Context(), name)
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

func listAgents(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		agents, err := k8s.ListAgents(c.Request.Context())
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
