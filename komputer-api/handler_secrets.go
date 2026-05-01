package main

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
)

type SecretResponse struct {
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace"`
	Keys           []string `json:"keys"`
	Managed        bool     `json:"managed"`
	AgentName      string   `json:"agentName,omitempty"`
	AttachedAgents int      `json:"attachedAgents"`
	AgentNames     []string `json:"agentNames,omitempty"`
	CreatedAt      string   `json:"createdAt"`
}

type SecretListResponse struct {
	Secrets []SecretResponse `json:"secrets"`
}

type CreateSecretRequest struct {
	Name      string            `json:"name" binding:"required"`
	Data      map[string]string `json:"data" binding:"required"`
	Namespace string            `json:"namespace"`
}

type UpdateSecretRequest struct {
	Data      map[string]string `json:"data" binding:"required"`
	Namespace string            `json:"namespace"`
}

// listSecrets returns all Kubernetes secrets in a namespace.
// @ID listSecrets
// @Summary List secrets
// @Description Returns all secrets with key names (not values) and attached agent counts in the specified namespace.
// @Tags secrets
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Param all query boolean false "Include all secrets, not just managed ones"
// @Success 200 {object} SecretListResponse "List of secrets"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /secrets [get]
func listSecrets(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		all := c.Query("all") == "true"
		secrets, err := k8s.ListSecrets(c.Request.Context(), ns, all)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list secrets: " + err.Error()})
			return
		}
		// Count how many agents reference each secret and collect their names.
		secretUsage := make(map[string][]string)
		agents, _ := k8s.ListAgents(c.Request.Context(), "", nil)
		for _, a := range agents {
			for _, s := range a.Spec.Secrets {
				key := a.Namespace + "/" + s
				secretUsage[key] = append(secretUsage[key], a.Name)
			}
		}
		resp := make([]SecretResponse, 0, len(secrets))
		for _, s := range secrets {
			keys := make([]string, 0, len(s.Data))
			for k := range s.Data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			resp = append(resp, SecretResponse{
				Name:           s.Name,
				Namespace:      s.Namespace,
				Keys:           keys,
				Managed:        s.Labels["komputer.ai/managed-by"] == "komputer-ai",
				AgentName:      s.Labels["komputer.ai/agent-name"],
				AttachedAgents: len(secretUsage[s.Namespace+"/"+s.Name]),
				AgentNames:     secretUsage[s.Namespace+"/"+s.Name],
				CreatedAt:      s.CreationTimestamp.UTC().Format(time.RFC3339),
			})
		}
		c.JSON(http.StatusOK, SecretListResponse{Secrets: resp})
	}
}

// createManagedSecret creates a new managed Kubernetes secret.
// @ID createSecret
// @Summary Create managed secret
// @Description Creates a new Kubernetes secret managed by komputer.ai that can be attached to agents.
// @Tags secrets
// @Accept json
// @Produce json
// @Param request body CreateSecretRequest true "Secret creation request"
// @Success 201 {object} SecretResponse "Secret created"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /secrets [post]
func createManagedSecret(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateSecretRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if !isValidK8sName(req.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid secret name: must be lowercase letters, numbers, and hyphens"})
			return
		}
		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}
		secret, err := k8s.CreateManagedSecret(c.Request.Context(), ns, req.Name, req.Data)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "secret already exists: " + req.Name})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create secret: " + err.Error()})
			return
		}
		keys := make([]string, 0, len(secret.Data))
		for k := range secret.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		c.JSON(http.StatusCreated, SecretResponse{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Keys:      keys,
			Managed:   true,
			CreatedAt: secret.CreationTimestamp.UTC().Format(time.RFC3339),
		})
	}
}

// deleteManagedSecret deletes a managed secret by name.
// @ID deleteSecret
// @Summary Delete managed secret
// @Description Deletes a managed Kubernetes secret.
// @Tags secrets
// @Produce json
// @Param name path string true "Secret name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Secret deleted"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /secrets/{name} [delete]
func deleteManagedSecret(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		if err := k8s.DeleteManagedSecret(c.Request.Context(), ns, name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete secret: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

// updateManagedSecret updates the data in a managed secret.
// @ID updateSecret
// @Summary Update managed secret
// @Description Replaces the key-value pairs in a managed Kubernetes secret.
// @Tags secrets
// @Accept json
// @Produce json
// @Param name path string true "Secret name"
// @Param namespace query string false "Kubernetes namespace"
// @Param request body UpdateSecretRequest true "Updated secret data"
// @Success 200 {object} map[string]string "Secret updated"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /secrets/{name} [patch]
func updateManagedSecret(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		var req UpdateSecretRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Namespace != "" {
			ns = req.Namespace
		}
		if _, err := k8s.UpdateManagedSecret(c.Request.Context(), ns, name, req.Data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update secret: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": name, "namespace": ns})
	}
}
