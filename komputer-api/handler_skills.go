package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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
