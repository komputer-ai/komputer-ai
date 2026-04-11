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

// createSkill creates a new skill resource.
// @ID createSkill
// @Summary Create skill
// @Description Creates a new KomputerSkill CR with script content that can be attached to agents.
// @Tags skills
// @Accept json
// @Produce json
// @Param request body CreateSkillRequest true "Skill creation request"
// @Success 201 {object} SkillResponse "Skill created"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /skills [post]
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

// getSkill returns details for a single skill.
// @ID getSkill
// @Summary Get skill details
// @Description Returns the content, description, and attached agent count for a single skill.
// @Tags skills
// @Produce json
// @Param name path string true "Skill name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} SkillResponse "Skill details"
// @Failure 404 {object} map[string]string "Skill not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /skills/{name} [get]
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

// listSkills returns all skills in a namespace.
// @ID listSkills
// @Summary List skills
// @Description Returns all skills with content and attached agent counts in the specified namespace.
// @Tags skills
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]interface{} "List of skills"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /skills [get]
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

// patchSkill updates description or content on an existing skill.
// @ID patchSkill
// @Summary Patch skill
// @Description Updates the description or script content of an existing skill.
// @Tags skills
// @Accept json
// @Produce json
// @Param name path string true "Skill name"
// @Param namespace query string false "Kubernetes namespace"
// @Param request body PatchSkillRequest true "Fields to update"
// @Success 200 {object} SkillResponse "Updated skill"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /skills/{name} [patch]
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

// deleteSkill deletes a skill by name.
// @ID deleteSkill
// @Summary Delete skill
// @Description Deletes the skill CR.
// @Tags skills
// @Produce json
// @Param name path string true "Skill name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Skill deleted"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /skills/{name} [delete]
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
