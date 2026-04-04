package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
)

type CreateScheduleRequest struct {
	Name         string                   `json:"name" binding:"required"`
	Schedule     string                   `json:"schedule" binding:"required"`
	Instructions string                   `json:"instructions" binding:"required"`
	Timezone     string                   `json:"timezone"`
	AutoDelete   bool                     `json:"autoDelete"`
	KeepAgents   bool                     `json:"keepAgents"`
	AgentName    string                   `json:"agentName"`
	Agent        *CreateScheduleAgentSpec `json:"agent"`
	Namespace    string                   `json:"namespace"`
}

type CreateScheduleAgentSpec struct {
	Model       string   `json:"model"`
	Lifecycle   string   `json:"lifecycle"`
	Role        string   `json:"role"`
	TemplateRef string   `json:"templateRef"`
	SecretRefs  []string `json:"secretRefs"`
}

type ScheduleResponse struct {
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	Schedule       string `json:"schedule"`
	Timezone       string `json:"timezone,omitempty"`
	AutoDelete     bool   `json:"autoDelete,omitempty"`
	KeepAgents     bool   `json:"keepAgents,omitempty"`
	Phase          string `json:"phase"`
	AgentName      string `json:"agentName,omitempty"`
	NextRunTime    string `json:"nextRunTime,omitempty"`
	LastRunTime    string `json:"lastRunTime,omitempty"`
	RunCount       int    `json:"runCount,omitempty"`
	SuccessfulRuns int    `json:"successfulRuns,omitempty"`
	FailedRuns     int    `json:"failedRuns,omitempty"`
	TotalCostUSD   string `json:"totalCostUSD,omitempty"`
	LastRunCostUSD string `json:"lastRunCostUSD,omitempty"`
	TotalTokens    int64  `json:"totalTokens,omitempty"`
	LastRunTokens  int64  `json:"lastRunTokens,omitempty"`
	LastRunStatus  string `json:"lastRunStatus,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

type ScheduleListResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
}

type PatchScheduleRequest struct {
	Schedule *string `json:"schedule,omitempty"`
}

// scheduleToResponse converts a KomputerSchedule CR to a ScheduleResponse.
func scheduleToResponse(s komputerv1alpha1.KomputerSchedule) ScheduleResponse {
	resp := ScheduleResponse{
		Name:           s.Name,
		Namespace:      s.Namespace,
		Schedule:       s.Spec.Schedule,
		Timezone:       s.Spec.Timezone,
		AutoDelete:     s.Spec.AutoDelete,
		KeepAgents:     s.Spec.KeepAgents,
		Phase:          string(s.Status.Phase),
		AgentName:      s.Status.AgentName,
		RunCount:       s.Status.RunCount,
		SuccessfulRuns: s.Status.SuccessfulRuns,
		FailedRuns:     s.Status.FailedRuns,
		TotalCostUSD:   s.Status.TotalCostUSD,
		LastRunCostUSD: s.Status.LastRunCostUSD,
		TotalTokens:    s.Status.TotalTokens,
		LastRunTokens:  s.Status.LastRunTokens,
		LastRunStatus:  s.Status.LastRunStatus,
		CreatedAt:      s.CreationTimestamp.UTC().Format(time.RFC3339),
	}
	if s.Status.NextRunTime != nil {
		resp.NextRunTime = s.Status.NextRunTime.Format("2006-01-02T15:04:05Z")
	}
	if s.Status.LastRunTime != nil {
		resp.LastRunTime = s.Status.LastRunTime.Format("2006-01-02T15:04:05Z")
	}
	// Use spec agentName if status doesn't have one yet.
	if resp.AgentName == "" {
		resp.AgentName = s.Spec.AgentName
	}
	return resp
}

func createSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if !isValidK8sName(req.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule name: must be lowercase, alphanumeric, hyphens only, max 63 chars (e.g. 'my-schedule-1')"})
			return
		}

		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}

		if err := k8s.EnsureNamespace(c.Request.Context(), ns); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure namespace: " + err.Error()})
			return
		}

		schedule, err := k8s.CreateSchedule(c.Request.Context(), ns, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create schedule: " + err.Error()})
			return
		}

		log.Printf("created new schedule %s/%s", ns, req.Name)
		c.JSON(http.StatusCreated, scheduleToResponse(*schedule))
	}
}

func listSchedules(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		schedules, err := k8s.ListSchedules(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules: " + err.Error()})
			return
		}
		resp := ScheduleListResponse{Schedules: make([]ScheduleResponse, 0, len(schedules))}
		for _, s := range schedules {
			resp.Schedules = append(resp.Schedules, scheduleToResponse(s))
		}
		c.JSON(http.StatusOK, resp)
	}
}

func getSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		schedule, err := k8s.GetSchedule(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, scheduleToResponse(*schedule))
	}
}

func deleteSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		if err := k8s.DeleteSchedule(c.Request.Context(), ns, name); err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule: " + err.Error()})
			return
		}
		log.Printf("deleted schedule %s/%s", ns, name)
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

func patchSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		var req PatchScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Schedule == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}
		if err := k8s.PatchScheduleCron(c.Request.Context(), ns, name, *req.Schedule); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch schedule: " + err.Error()})
			return
		}
		sched, err := k8s.GetSchedule(c.Request.Context(), ns, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "patched but failed to read back: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, scheduleToResponse(*sched))
	}
}
