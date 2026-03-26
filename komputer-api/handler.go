package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
)

type CreateAgentRequest struct {
	Name         string `json:"name" binding:"required"`
	Instructions string `json:"instructions" binding:"required"`
	Model        string `json:"model"`
	TemplateRef  string `json:"templateRef"`
}

type AgentResponse struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Model     string `json:"model"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
}

func SetupRoutes(r *gin.Engine, k8s *K8sClient) {
	v1 := r.Group("/api/v1")
	{
		v1.POST("/agents", createOrTriggerAgent(k8s))
		v1.GET("/agents", listAgents(k8s))
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

			podIP, err := k8s.GetAgentPodIP(c.Request.Context(), existing.Status.PodName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent pod IP: " + err.Error()})
				return
			}

			if err := k8s.ForwardTaskToAgent(c.Request.Context(), podIP, req.Instructions); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to forward task: " + err.Error()})
				return
			}

			log.Printf("forwarded task to existing agent %s", req.Name)
			c.JSON(http.StatusOK, AgentResponse{
				Name:      existing.Name,
				Namespace: existing.Namespace,
				Model:     existing.Spec.Model,
				Status:    string(existing.Status.Phase),
				CreatedAt: existing.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			})
			return
		}

		agent, err := k8s.CreateAgent(c.Request.Context(), req.Name, req.Instructions, req.Model, req.TemplateRef)
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
				Name:      a.Name,
				Namespace: a.Namespace,
				Model:     a.Spec.Model,
				Status:    string(a.Status.Phase),
				CreatedAt: a.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
			})
		}

		c.JSON(http.StatusOK, resp)
	}
}
