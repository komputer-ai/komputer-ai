package main

import (
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
)

type OfficeResponse struct {
	Name            string                 `json:"name"`
	Namespace       string                 `json:"namespace"`
	Manager         string                 `json:"manager"`
	Phase           string                 `json:"phase"`
	TotalAgents     int                    `json:"totalAgents"`
	ActiveAgents    int                    `json:"activeAgents"`
	CompletedAgents int                    `json:"completedAgents"`
	TotalCostUSD    string                 `json:"totalCostUSD,omitempty"`
	TotalTokens     int64                  `json:"totalTokens,omitempty"`
	Members         []OfficeMemberResponse `json:"members,omitempty"`
	CreatedAt       string                 `json:"createdAt"`
}

type OfficeMemberResponse struct {
	Name            string `json:"name"`
	Role            string `json:"role"`
	TaskStatus      string `json:"taskStatus,omitempty"`
	LastTaskCostUSD string `json:"lastTaskCostUSD,omitempty"`
}

type OfficeListResponse struct {
	Offices []OfficeResponse `json:"offices"`
}

// officeToResponse converts a KomputerOffice CR to an OfficeResponse.
func officeToResponse(o komputerv1alpha1.KomputerOffice, includeMembers bool) OfficeResponse {
	resp := OfficeResponse{
		Name:            o.Name,
		Namespace:       o.Namespace,
		Manager:         o.Spec.Manager,
		Phase:           string(o.Status.Phase),
		TotalAgents:     o.Status.TotalAgents,
		ActiveAgents:    o.Status.ActiveAgents,
		CompletedAgents: o.Status.CompletedAgents,
		TotalCostUSD:    o.Status.TotalCostUSD,
		TotalTokens:     o.Status.TotalTokens,
		CreatedAt:       o.CreationTimestamp.UTC().Format(time.RFC3339),
	}
	if includeMembers {
		members := make([]OfficeMemberResponse, 0, len(o.Status.Members)+1)
		// Include manager as a member entry.
		if o.Status.Manager.Name != "" {
			members = append(members, OfficeMemberResponse{
				Name:            o.Status.Manager.Name,
				Role:            o.Status.Manager.Role,
				TaskStatus:      o.Status.Manager.TaskStatus,
				LastTaskCostUSD: o.Status.Manager.LastTaskCostUSD,
			})
		}
		for _, m := range o.Status.Members {
			members = append(members, OfficeMemberResponse{
				Name:            m.Name,
				Role:            m.Role,
				TaskStatus:      m.TaskStatus,
				LastTaskCostUSD: m.LastTaskCostUSD,
			})
		}
		resp.Members = members
	}
	return resp
}

// listOffices returns all offices in a namespace.
// @ID listOffices
// @Summary List offices
// @Description Returns all offices with their current status in the specified namespace.
// @Tags offices
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} OfficeListResponse "List of offices"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /offices [get]
func listOffices(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		offices, err := k8s.ListOffices(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list offices: " + err.Error()})
			return
		}
		resp := OfficeListResponse{Offices: make([]OfficeResponse, 0, len(offices))}
		for _, o := range offices {
			resp.Offices = append(resp.Offices, officeToResponse(o, false))
		}
		c.JSON(http.StatusOK, resp)
	}
}

// getOffice returns details for a single office including all member agents.
// @ID getOffice
// @Summary Get office details
// @Description Returns the current status and member list for a single office.
// @Tags offices
// @Produce json
// @Param name path string true "Office name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} OfficeResponse "Office details"
// @Failure 404 {object} map[string]string "Office not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /offices/{name} [get]
func getOffice(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		office, err := k8s.GetOffice(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "office not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, officeToResponse(*office, true))
	}
}

// deleteOffice deletes an office and cleans up all member agent event streams.
// @ID deleteOffice
// @Summary Delete office
// @Description Deletes the office CR and cleans up Redis event streams for all member agents.
// @Tags offices
// @Produce json
// @Param name path string true "Office name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Office deleted"
// @Failure 404 {object} map[string]string "Office not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /offices/{name} [delete]
func deleteOffice(k8s *K8sClient, worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		// Get the office first so we can clean up member event streams.
		office, err := k8s.GetOffice(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "office not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := k8s.DeleteOffice(c.Request.Context(), ns, name); err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "office not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete office: " + err.Error()})
			return
		}

		// Clean up Redis event streams for all member agents (including manager).
		if office.Status.Manager.Name != "" {
			if err := DeleteAgentStream(c.Request.Context(), worker.Rdb, office.Status.Manager.Name, worker.StreamPrefix); err != nil {
				log.Printf("warning: failed to delete event stream for manager %s: %v", office.Status.Manager.Name, err)
			}
		}
		for _, m := range office.Status.Members {
			if err := DeleteAgentStream(c.Request.Context(), worker.Rdb, m.Name, worker.StreamPrefix); err != nil {
				log.Printf("warning: failed to delete event stream for member %s: %v", m.Name, err)
			}
		}

		log.Printf("deleted office %s/%s", ns, name)
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

// getOfficeEvents returns merged events from all member agents in the office.
// @ID getOfficeEvents
// @Summary Get office events
// @Description Returns merged events from all member agent Redis streams, sorted chronologically.
// @Tags offices
// @Produce json
// @Param name path string true "Office name"
// @Param namespace query string false "Kubernetes namespace"
// @Param limit query int false "Max events to return (1-200)" default(50)
// @Success 200 {object} map[string]interface{} "Office events"
// @Failure 400 {object} map[string]string "Invalid limit parameter"
// @Failure 404 {object} map[string]string "Office not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /offices/{name}/events [get]
func getOfficeEvents(k8s *K8sClient, worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		limit := int64(50)
		if l := c.Query("limit"); l != "" {
			parsed, err := strconv.ParseInt(l, 10, 64)
			if err != nil || parsed < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit parameter"})
				return
			}
			if parsed > 200 {
				parsed = 200
			}
			limit = parsed
		}

		office, err := k8s.GetOffice(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "office not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Collect all agent names (manager + members).
		agentNames := make([]string, 0, len(office.Status.Members)+1)
		if office.Status.Manager.Name != "" {
			agentNames = append(agentNames, office.Status.Manager.Name)
		}
		for _, m := range office.Status.Members {
			agentNames = append(agentNames, m.Name)
		}

		// Fetch events for each agent and merge.
		var allEvents []AgentEvent
		for _, agentName := range agentNames {
			events, err := GetAgentEvents(c.Request.Context(), worker.Rdb, agentName, limit, worker.StreamPrefix)
			if err != nil {
				log.Printf("warning: failed to get events for agent %s: %v", agentName, err)
				continue
			}
			allEvents = append(allEvents, events...)
		}

		// Sort by timestamp ascending.
		sort.Slice(allEvents, func(i, j int) bool {
			return allEvents[i].Timestamp < allEvents[j].Timestamp
		})

		// Apply limit to merged results.
		if int64(len(allEvents)) > limit {
			allEvents = allEvents[len(allEvents)-int(limit):]
		}

		c.JSON(http.StatusOK, gin.H{"office": name, "events": allEvents})
	}
}
