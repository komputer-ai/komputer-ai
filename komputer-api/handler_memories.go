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

// createMemory creates a new memory resource.
// @ID createMemory
// @Summary Create memory
// @Description Creates a new KomputerMemory CR that can be attached to agents as persistent context.
// @Tags memories
// @Accept json
// @Produce json
// @Param request body CreateMemoryRequest true "Memory creation request"
// @Success 201 {object} MemoryResponse "Memory created"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /memories [post]
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

// getMemory returns details for a single memory.
// @ID getMemory
// @Summary Get memory details
// @Description Returns the content and attached agent count for a single memory.
// @Tags memories
// @Produce json
// @Param name path string true "Memory name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} MemoryResponse "Memory details"
// @Failure 404 {object} map[string]string "Memory not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /memories/{name} [get]
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

// listMemories returns all memories in a namespace.
// @ID listMemories
// @Summary List memories
// @Description Returns all memories with content and attached agent counts in the specified namespace.
// @Tags memories
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]interface{} "List of memories"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /memories [get]
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

// patchMemory updates content or description on an existing memory.
// @ID patchMemory
// @Summary Patch memory
// @Description Updates the content or description of an existing memory.
// @Tags memories
// @Accept json
// @Produce json
// @Param name path string true "Memory name"
// @Param namespace query string false "Kubernetes namespace"
// @Param request body PatchMemoryRequest true "Fields to update"
// @Success 200 {object} MemoryResponse "Updated memory"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /memories/{name} [patch]
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

// deleteMemory deletes a memory by name.
// @ID deleteMemory
// @Summary Delete memory
// @Description Deletes the memory CR.
// @Tags memories
// @Produce json
// @Param name path string true "Memory name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Memory deleted"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /memories/{name} [delete]
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
