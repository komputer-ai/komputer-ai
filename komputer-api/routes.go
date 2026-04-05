package main

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

// isValidK8sName checks if a string is a valid Kubernetes DNS subdomain name.
var k8sNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func isValidK8sName(name string) bool {
	return len(name) > 0 && len(name) <= 63 && k8sNameRegex.MatchString(name)
}

// resolveNamespace returns the namespace from the query param, request body, or default.
func resolveNamespace(c *gin.Context, k8s *K8sClient) string {
	if ns := c.Query("namespace"); ns != "" {
		return ns
	}
	return k8s.defaultNamespace
}

// collectSecretKeys gathers all key names from the agent's referenced K8s Secrets.
func collectSecretKeys(ctx gin.Context, k8s *K8sClient, ns string, secretNames []string) []string {
	var keys []string
	for _, name := range secretNames {
		k, err := k8s.GetSecretKeys(ctx.Request.Context(), ns, name)
		if err != nil {
			continue // secret may have been deleted
		}
		keys = append(keys, k...)
	}
	return keys
}

func SetupRoutes(r *gin.Engine, k8s *K8sClient, hub *Hub, worker *RedisWorker) {
	// Health check endpoints (outside /api/v1 for k8s probes).
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		// Check Redis connectivity.
		if err := worker.Rdb.Ping(c.Request.Context()).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": "redis: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/agents", createOrTriggerAgent(k8s))
		v1.GET("/agents", listAgents(k8s))
		v1.GET("/agents/:name", getAgent(k8s))
		v1.GET("/agents/:name/events", getAgentEvents(worker))
		v1.DELETE("/agents/:name", deleteAgent(k8s, worker))
		v1.PATCH("/agents/:name", patchAgent(k8s))
		v1.POST("/agents/:name/cancel", cancelAgentTask(k8s))
		v1.GET("/agents/:name/ws", HandleAgentWS(hub))

		v1.GET("/offices", listOffices(k8s))
		v1.GET("/offices/:name", getOffice(k8s))
		v1.DELETE("/offices/:name", deleteOffice(k8s, worker))
		v1.GET("/offices/:name/events", getOfficeEvents(k8s, worker))

		v1.POST("/schedules", createSchedule(k8s))
		v1.GET("/schedules", listSchedules(k8s))
		v1.GET("/schedules/:name", getSchedule(k8s))
		v1.DELETE("/schedules/:name", deleteSchedule(k8s))
		v1.PATCH("/schedules/:name", patchSchedule(k8s))

		v1.POST("/memories", createMemory(k8s))
		v1.GET("/memories", listMemories(k8s))
		v1.GET("/memories/:name", getMemory(k8s))
		v1.DELETE("/memories/:name", deleteMemory(k8s))
		v1.PATCH("/memories/:name", patchMemory(k8s))

		v1.POST("/skills", createSkill(k8s))
		v1.GET("/skills", listSkills(k8s))
		v1.GET("/skills/:name", getSkill(k8s))
		v1.DELETE("/skills/:name", deleteSkill(k8s))
		v1.PATCH("/skills/:name", patchSkill(k8s))

		v1.GET("/secrets", listSecrets(k8s))
		v1.POST("/secrets", createManagedSecret(k8s))
		v1.DELETE("/secrets/:name", deleteManagedSecret(k8s))
		v1.PATCH("/secrets/:name", updateManagedSecret(k8s))

		v1.POST("/connectors", createConnector(k8s))
		v1.GET("/connectors", listConnectors(k8s))
		v1.GET("/connectors/:name", getConnector(k8s))
		v1.GET("/connectors/:name/tools", listConnectorTools(k8s))
		v1.DELETE("/connectors/:name", deleteConnector(k8s))

		v1.GET("/connector-templates", listConnectorTemplates())
		v1.GET("/templates", listTemplates(k8s))
		v1.GET("/namespaces", listNamespaces(k8s))

		v1.POST("/oauth/authorize", oauthAuthorize(k8s))
		v1.GET("/oauth/callback", oauthCallback(k8s))
		v1.POST("/oauth/refresh", oauthRefresh(k8s))
	}
}
