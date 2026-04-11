package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type TemplateResponse struct {
	Name      string `json:"name"`
	Scope     string `json:"scope"`              // "namespace" or "cluster"
	Namespace string `json:"namespace,omitempty"` // populated for namespaced templates
}

// listTemplates returns available agent templates in the namespace and cluster.
// @ID listTemplates
// @Summary List agent templates
// @Description Returns all agent templates (both namespace-scoped and cluster-scoped) available in the specified namespace.
// @Tags templates
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]interface{} "List of templates"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /templates [get]
func listTemplates(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := resolveNamespace(c, k8s)
		templates, err := k8s.ListTemplates(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list templates: " + err.Error()})
			return
		}

		resp := make([]TemplateResponse, 0, len(templates))
		for _, t := range templates {
			resp = append(resp, TemplateResponse{Name: t.Name, Scope: t.Scope, Namespace: t.Namespace})
		}
		c.JSON(http.StatusOK, gin.H{"templates": resp})
	}
}

// listNamespaces returns all Kubernetes namespaces accessible to the API.
// @Summary List namespaces
// @Description Returns all Kubernetes namespaces the API has access to.
// @Tags templates
// @Produce json
// @Success 200 {object} map[string]interface{} "List of namespaces"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /namespaces [get]
func listNamespaces(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		names, err := k8s.ListNamespaces(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"namespaces": names})
	}
}
