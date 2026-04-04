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
