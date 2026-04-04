package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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

type PatchMemoryRequest struct {
	Content     *string `json:"content,omitempty"`
	Description *string `json:"description,omitempty"`
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
