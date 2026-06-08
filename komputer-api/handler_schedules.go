package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Name           string                   `json:"name"`
	Namespace      string                   `json:"namespace"`
	Schedule       string                   `json:"schedule"`
	Instructions   string                   `json:"instructions"`
	Timezone       string                   `json:"timezone,omitempty"`
	AutoDelete     bool                     `json:"autoDelete,omitempty"`
	KeepAgents     bool                     `json:"keepAgents,omitempty"`
	Suspended      bool                     `json:"suspended,omitempty"`
	Agent          *CreateScheduleAgentSpec `json:"agent,omitempty"`
	Phase          string                   `json:"phase"`
	AgentName      string                   `json:"agentName,omitempty"`
	NextRunTime    string                   `json:"nextRunTime,omitempty"`
	LastRunTime    string                   `json:"lastRunTime,omitempty"`
	RunCount       int                      `json:"runCount,omitempty"`
	SuccessfulRuns int                      `json:"successfulRuns,omitempty"`
	FailedRuns     int                      `json:"failedRuns,omitempty"`
	TotalCostUSD   string                   `json:"totalCostUSD,omitempty"`
	LastRunCostUSD string                   `json:"lastRunCostUSD,omitempty"`
	TotalTokens    int64                    `json:"totalTokens,omitempty"`
	LastRunTokens  int64                    `json:"lastRunTokens,omitempty"`
	LastRunStatus  string                   `json:"lastRunStatus,omitempty"`
	CreatedAt      string                   `json:"createdAt"`
}

type ScheduleListResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
}

type PatchScheduleRequest struct {
	Schedule     *string                  `json:"schedule,omitempty"`
	Instructions *string                  `json:"instructions,omitempty"`
	Timezone     *string                  `json:"timezone,omitempty"`
	AutoDelete   *bool                    `json:"autoDelete,omitempty"`
	KeepAgents   *bool                    `json:"keepAgents,omitempty"`
	Suspended    *bool                    `json:"suspended,omitempty"`
	AgentName    *string                  `json:"agentName,omitempty"`
	Agent        *CreateScheduleAgentSpec `json:"agent,omitempty"`
}

type TriggerScheduleResponse struct {
	Status    string `json:"status"`
	Name      string `json:"name"`
	AgentName string `json:"agentName"`
}

// scheduleToResponse converts a KomputerSchedule CR to a ScheduleResponse.
func scheduleToResponse(s komputerv1alpha1.KomputerSchedule) ScheduleResponse {
	resp := ScheduleResponse{
		Name:           s.Name,
		Namespace:      s.Namespace,
		Schedule:       s.Spec.Schedule,
		Instructions:   s.Spec.Instructions,
		Timezone:       s.Spec.Timezone,
		AutoDelete:     s.Spec.AutoDelete,
		KeepAgents:     s.Spec.KeepAgents,
		Suspended:      s.Spec.Suspended,
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
	if s.Spec.Agent != nil {
		resp.Agent = &CreateScheduleAgentSpec{
			Model:       s.Spec.Agent.Model,
			Lifecycle:   string(s.Spec.Agent.Lifecycle),
			Role:        s.Spec.Agent.Role,
			TemplateRef: s.Spec.Agent.TemplateRef,
			SecretRefs:  s.Spec.Agent.Secrets,
		}
	}
	return resp
}

// createSchedule creates a new cron schedule that fires agents on a recurring basis.
// @ID createSchedule
// @Summary Create schedule
// @Description Creates a new KomputerSchedule CR that triggers agent tasks on a cron schedule.
// @Tags schedules
// @Accept json
// @Produce json
// @Param request body CreateScheduleRequest true "Schedule creation request"
// @Success 201 {object} ScheduleResponse "Schedule created"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /schedules [post]
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

		if err := k8s.EnsureNamespaceExists(c.Request.Context(), ns); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure namespace: " + err.Error()})
			return
		}

		schedule, err := k8s.CreateSchedule(c.Request.Context(), ns, &req)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "schedule already exists: " + req.Name})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create schedule: " + err.Error()})
			return
		}

		Logger.Infow("created new schedule", "namespace", ns, "name", req.Name)
		c.JSON(http.StatusCreated, scheduleToResponse(*schedule))
	}
}

// listSchedules returns all schedules in a namespace.
// @ID listSchedules
// @Summary List schedules
// @Description Returns all schedules with their current status and run history in the specified namespace.
// @Tags schedules
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} ScheduleListResponse "List of schedules"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /schedules [get]
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

// getSchedule returns details for a single schedule.
// @ID getSchedule
// @Summary Get schedule details
// @Description Returns the current status and run history for a single schedule.
// @Tags schedules
// @Produce json
// @Param name path string true "Schedule name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} ScheduleResponse "Schedule details"
// @Failure 404 {object} map[string]string "Schedule not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /schedules/{name} [get]
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

// deleteSchedule deletes a schedule by name.
// @ID deleteSchedule
// @Summary Delete schedule
// @Description Deletes the schedule CR. Does not delete any agents that were created by the schedule.
// @Tags schedules
// @Produce json
// @Param name path string true "Schedule name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Schedule deleted"
// @Failure 404 {object} map[string]string "Schedule not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /schedules/{name} [delete]
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
		Logger.Infow("deleted schedule", "namespace", ns, "name", name)
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

// triggerScheduleNow is the shared firing routine used by both the HTTP handler
// (POST /schedules/:name/trigger) and the MCP server's trigger_schedule tool.
// Returns (agentName, statusCode, error). statusCode mirrors HTTP semantics
// (200 ok, 404 missing, 409 conflict, 500 internal) so callers can map naturally.
func triggerScheduleNow(ctx context.Context, k8s *K8sClient, ns, name string) (string, int, error) {
	schedule, err := k8s.GetSchedule(ctx, ns, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", http.StatusNotFound, fmt.Errorf("schedule not found")
		}
		return "", http.StatusInternalServerError, err
	}
	if schedule.Status.LastRunStatus == "InProgress" {
		return "", http.StatusConflict, fmt.Errorf("a run is already in progress")
	}
	agentName := schedule.Spec.AgentName
	if agentName == "" {
		agentName = schedule.Name + "-agent"
	}

	agent, agentErr := k8s.GetAgent(ctx, ns, agentName)
	if agentErr != nil && !errors.IsNotFound(agentErr) {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to look up agent: %w", agentErr)
	}

	instructions := schedule.Spec.Instructions

	if errors.IsNotFound(agentErr) {
		if schedule.Spec.Agent == nil {
			return "", http.StatusNotFound, fmt.Errorf("agent %s does not exist and schedule has no agent template", agentName)
		}
		lifecycle := string(schedule.Spec.Agent.Lifecycle)
		if lifecycle == "" {
			lifecycle = string(komputerv1alpha1.AgentLifecycleSleep)
		}
		_, err := k8s.CreateAgent(
			ctx, ns, agentName,
			instructions, "", "",
			schedule.Spec.Agent.Model,
			schedule.Spec.Agent.TemplateRef,
			schedule.Spec.Agent.Role,
			schedule.Spec.Agent.Secrets,
			nil, nil, nil,
			lifecycle, "", 0,
			(*corev1.PodSpec)(nil),
			(*komputerv1alpha1.StorageSpec)(nil),
			map[string]string{"komputer.ai/schedule": schedule.Name},
		)
		if err != nil {
			return "", http.StatusInternalServerError, fmt.Errorf("failed to create agent: %w", err)
		}
	} else {
		if agent.Status.TaskStatus == komputerv1alpha1.AgentTaskInProgress {
			return "", http.StatusConflict, fmt.Errorf("agent %s is currently busy", agentName)
		}
		if agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
			if err := k8s.WakeAgent(ctx, ns, agentName, instructions, "", "", "", string(agent.Spec.Lifecycle)); err != nil {
				return "", http.StatusInternalServerError, fmt.Errorf("failed to wake agent: %w", err)
			}
		} else if agent.Status.PodName != "" {
			if _, err := k8s.ForwardTaskToAgent(ctx, ns, agent.Status.PodName, agentName, instructions, "", "", ""); err != nil {
				return "", http.StatusInternalServerError, fmt.Errorf("failed to forward task: %w", err)
			}
		} else {
			return "", http.StatusConflict, fmt.Errorf("agent %s has no running pod", agentName)
		}
	}

	now := metav1.NewTime(time.Now().UTC())
	if freshSchedule, getErr := k8s.GetSchedule(ctx, ns, name); getErr == nil {
		original := freshSchedule.DeepCopy()
		freshSchedule.Status.LastRunTime = &now
		freshSchedule.Status.RunCount++
		freshSchedule.Status.LastRunStatus = "InProgress"
		freshSchedule.Status.AgentName = agentName
		if patchErr := k8s.PatchScheduleStatus(ctx, original, freshSchedule); patchErr != nil {
			Logger.Warnw("manual trigger: failed to update schedule status", "schedule", name, "error", patchErr)
		}
	}

	return agentName, http.StatusOK, nil
}

// triggerSchedule fires a schedule immediately, outside of its cron cadence.
// Mirrors the operator's reconcile-time fire path: ensures the agent exists (creating
// from spec.Agent template if needed), then wakes or forwards the task. Updates the
// schedule status the same way the operator would.
// @ID triggerSchedule
// @Summary Trigger schedule now
// @Description Fires the schedule immediately, independent of its cron cadence. Returns 409 if a run is already in progress.
// @Tags schedules
// @Produce json
// @Param name path string true "Schedule name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} TriggerScheduleResponse "Triggered"
// @Failure 404 {object} map[string]string "Schedule not found"
// @Failure 409 {object} map[string]string "A run is already in progress"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /schedules/{name}/trigger [post]
func triggerSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		agentName, statusCode, err := triggerScheduleNow(c.Request.Context(), k8s, ns, name)
		if err != nil {
			c.JSON(statusCode, gin.H{"error": err.Error()})
			return
		}
		Logger.Infow("manual trigger fired schedule", "namespace", ns, "schedule", name, "agent", agentName)
		c.JSON(http.StatusOK, TriggerScheduleResponse{
			Status:    "triggered",
			Name:      name,
			AgentName: agentName,
		})
	}
}

// patchSchedule updates the cron expression for a schedule.
// @ID patchSchedule
// @Summary Patch schedule
// @Description Updates the cron expression for an existing schedule.
// @Tags schedules
// @Accept json
// @Produce json
// @Param name path string true "Schedule name"
// @Param namespace query string false "Kubernetes namespace"
// @Param request body PatchScheduleRequest true "Fields to update"
// @Success 200 {object} ScheduleResponse "Updated schedule"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /schedules/{name} [patch]
func patchSchedule(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		var req PatchScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Schedule == nil && req.Instructions == nil && req.Timezone == nil &&
			req.AutoDelete == nil && req.KeepAgents == nil && req.Suspended == nil &&
			req.AgentName == nil && req.Agent == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}
		if err := k8s.PatchScheduleSpec(c.Request.Context(), ns, name, req); err != nil {
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
