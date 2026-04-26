package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SquadResponse is the API representation of a KomputerSquad.
type SquadResponse struct {
	Name             string                `json:"name"`
	Namespace        string                `json:"namespace"`
	Phase            string                `json:"phase"`
	PodName          string                `json:"podName,omitempty"`
	Members          []SquadMemberResponse `json:"members"`
	OrphanTTL        string                `json:"orphanTTL,omitempty"`
	OrphanedSince    *time.Time            `json:"orphanedSince,omitempty"`
	Message          string                `json:"message,omitempty"`
	BreakUpRequested bool                  `json:"breakUpRequested,omitempty"`
	CreatedAt        time.Time             `json:"createdAt"`
}

// SquadMemberResponse is the API representation of a squad member status entry.
type SquadMemberResponse struct {
	Name       string `json:"name"`
	Ready      bool   `json:"ready"`
	TaskStatus string `json:"taskStatus,omitempty"`
}

// SquadListResponse wraps a list of squads.
type SquadListResponse struct {
	Squads []SquadResponse `json:"squads"`
}

// CreateSquadRequest is the body for POST /squads.
type CreateSquadRequest struct {
	Name      string                               `json:"name" binding:"required"`
	Namespace string                               `json:"namespace,omitempty"`
	Members   []komputerv1alpha1.KomputerSquadMember `json:"members" binding:"required"`
	OrphanTTL string                               `json:"orphanTTL,omitempty"` // duration string e.g. "10m"
}

// PatchSquadRequest is the body for PATCH /squads/:name — full member list replacement.
type PatchSquadRequest struct {
	Members   []komputerv1alpha1.KomputerSquadMember `json:"members,omitempty"`
	OrphanTTL *string                                `json:"orphanTTL,omitempty"`
}

// AddSquadMemberRequest is the body for POST /squads/:name/members.
type AddSquadMemberRequest struct {
	Ref  *komputerv1alpha1.KomputerSquadMemberRef `json:"ref,omitempty"`
	Spec *komputerv1alpha1.KomputerAgentSpec      `json:"spec,omitempty"`
	// Name is the desired KomputerAgent name when Spec is set. Optional.
	Name string `json:"name,omitempty"`
}

// squadToResponse converts a KomputerSquad CR to a SquadResponse.
func squadToResponse(s komputerv1alpha1.KomputerSquad) SquadResponse {
	members := make([]SquadMemberResponse, 0, len(s.Status.Members))
	for _, m := range s.Status.Members {
		members = append(members, SquadMemberResponse{
			Name:       m.Name,
			Ready:      m.Ready,
			TaskStatus: m.TaskStatus,
		})
	}

	var orphanTTL string
	if s.Spec.OrphanTTL != nil {
		orphanTTL = s.Spec.OrphanTTL.Duration.String()
	}

	var orphanedSince *time.Time
	if s.Status.OrphanedSince != nil {
		t := s.Status.OrphanedSince.UTC()
		orphanedSince = &t
	}

	return SquadResponse{
		Name:             s.Name,
		Namespace:        s.Namespace,
		Phase:            string(s.Status.Phase),
		PodName:          s.Status.PodName,
		Members:          members,
		OrphanTTL:        orphanTTL,
		OrphanedSince:    orphanedSince,
		Message:          s.Status.Message,
		BreakUpRequested: s.Spec.BreakUpRequested,
		CreatedAt:        s.CreationTimestamp.UTC(),
	}
}

// listSquads returns all squads, optionally filtered by namespace.
// @ID listSquads
// @Summary List squads
// @Description Returns all squads with their current status. Pass ?namespace= to filter; omit for all namespaces.
// @Tags squads
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} SquadListResponse "List of squads"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /squads [get]
func listSquads(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		squads, err := k8s.ListSquads(c.Request.Context(), ns)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list squads: " + err.Error()})
			return
		}
		resp := SquadListResponse{Squads: make([]SquadResponse, 0, len(squads))}
		for _, s := range squads {
			resp.Squads = append(resp.Squads, squadToResponse(s))
		}
		c.JSON(http.StatusOK, resp)
	}
}

// getSquad returns details for a single squad.
// @ID getSquad
// @Summary Get squad details
// @Description Returns the current status and member list for a single squad.
// @Tags squads
// @Produce json
// @Param name path string true "Squad name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} SquadResponse "Squad details"
// @Failure 404 {object} map[string]string "Squad not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /squads/{name} [get]
func getSquad(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		squad, err := k8s.GetSquad(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "squad not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, squadToResponse(*squad))
	}
}

// createSquad creates a new KomputerSquad.
// @ID createSquad
// @Summary Create squad
// @Description Creates a new squad with the given members.
// validateMemberRefForAdoption ensures an existing agent being adopted into a squad
// is asleep (no running pod), so that joining the squad doesn't leave a stale solo
// pod alongside the new squad-pod container. Returns (errorMessage, true) to block,
// or ("", false) to allow.
func validateMemberRefForAdoption(c *gin.Context, k8s *K8sClient, ns, agentName string) (string, bool) {
	agent, err := k8s.GetAgent(c.Request.Context(), ns, agentName)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Sprintf("agent %q not found", agentName), true
		}
		return fmt.Sprintf("failed to look up agent %q: %v", agentName, err), true
	}
	// Already in a squad? Let the operator handle it (no transition needed).
	if agent.Status.Squad {
		return "", false
	}
	if agent.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		return fmt.Sprintf("agent %q must be asleep in order to team up (current phase: %s). Sleep the agent first, then retry.", agentName, agent.Status.Phase), true
	}
	return "", false
}

// @Tags squads
// @Accept json
// @Produce json
// @Param request body CreateSquadRequest true "Squad creation request"
// @Success 200 {object} SquadResponse "Created squad"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 409 {object} map[string]string "Squad already exists"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /squads [post]
func createSquad(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateSquadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if !isValidK8sName(req.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid squad name: must be lowercase, alphanumeric, hyphens only, max 63 chars"})
			return
		}

		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}

		// Block team-up of running agents — the existing solo pod must be removed first.
		// Sleeping the agent stops its pod (PVC kept) so the squad pod can adopt it cleanly.
		for _, m := range req.Members {
			if m.Ref == nil {
				continue
			}
			refNs := m.Ref.Namespace
			if refNs == "" {
				refNs = ns
			}
			if msg, blocked := validateMemberRefForAdoption(c, k8s, refNs, m.Ref.Name); blocked {
				c.JSON(http.StatusConflict, gin.H{"error": msg})
				return
			}
		}

		// Compute member names so each inline member's InternalSystemPrompt can
		// reference its squad siblings.
		memberNames := make([]string, 0, len(req.Members))
		for i, m := range req.Members {
			if m.Ref != nil {
				memberNames = append(memberNames, m.Ref.Name)
			} else if m.Name != "" {
				memberNames = append(memberNames, m.Name)
			} else {
				memberNames = append(memberNames, fmt.Sprintf("%s-member-%d", req.Name, i))
			}
		}

		// For each inline-spec member, populate InternalSystemPrompt the same way
		// the solo-agent create flow does. Otherwise squad members start with an
		// empty KOMPUTER_INTERNAL_SYSTEM_PROMPT and lack their role/memories/squad
		// context.
		for i := range req.Members {
			m := &req.Members[i]
			if m.Spec == nil {
				continue
			}
			role := m.Spec.Role
			if role == "" {
				role = "worker"
			}
			myName := memberNames[i]
			siblings := make([]string, 0, len(memberNames)-1)
			for j, n := range memberNames {
				if j != i {
					siblings = append(siblings, n)
				}
			}
			m.Spec.InternalSystemPrompt = buildAgentInternalSystemPrompt(
				c.Request.Context(), k8s, ns, myName, role,
				m.Spec.Memories, req.Name, siblings,
			)
		}

		spec := komputerv1alpha1.KomputerSquadSpec{
			Members: req.Members,
		}

		if req.OrphanTTL != "" {
			d, err := time.ParseDuration(req.OrphanTTL)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid orphanTTL: " + err.Error()})
				return
			}
			spec.OrphanTTL = &metav1.Duration{Duration: d}
		}

		squad := &komputerv1alpha1.KomputerSquad{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: ns,
				Labels: map[string]string{
					"komputer.ai/squad-name": req.Name,
				},
			},
			Spec: spec,
		}

		created, err := k8s.CreateSquad(c.Request.Context(), ns, squad)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "squad already exists: " + req.Name})
				return
			}
			squadActionsTotal.WithLabelValues("create", "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create squad: " + err.Error()})
			return
		}

		Logger.Infow("created squad", "namespace", ns, "squad_name", req.Name)
		squadActionsTotal.WithLabelValues("create", "success").Inc()
		c.JSON(http.StatusOK, squadToResponse(*created))
	}
}

// patchSquad updates the spec of an existing squad (full member list replacement).
// @ID patchSquad
// @Summary Patch squad
// @Description Replaces the member list and/or orphanTTL on an existing squad. Retries once on 409 conflict.
// @Tags squads
// @Accept json
// @Produce json
// @Param name path string true "Squad name"
// @Param namespace query string false "Kubernetes namespace"
// @Param request body PatchSquadRequest true "Fields to update"
// @Success 200 {object} SquadResponse "Updated squad"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Squad not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /squads/{name} [patch]
func patchSquad(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		var req PatchSquadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		if req.Members == nil && req.OrphanTTL == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		// Retry once on 409 conflict.
		var updated *komputerv1alpha1.KomputerSquad
		for attempt := 0; attempt < 2; attempt++ {
			squad, err := k8s.GetSquad(c.Request.Context(), ns, name)
			if err != nil {
				if errors.IsNotFound(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": "squad not found"})
					return
				}
				squadActionsTotal.WithLabelValues("update", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if req.Members != nil {
				squad.Spec.Members = req.Members
			}
			if req.OrphanTTL != nil {
				if *req.OrphanTTL == "" {
					squad.Spec.OrphanTTL = nil
				} else {
					d, parseErr := time.ParseDuration(*req.OrphanTTL)
					if parseErr != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "invalid orphanTTL: " + parseErr.Error()})
						return
					}
					squad.Spec.OrphanTTL = &metav1.Duration{Duration: d}
				}
			}

			updated, err = k8s.UpdateSquad(c.Request.Context(), squad)
			if err == nil {
				break
			}
			if !errors.IsConflict(err) || attempt == 1 {
				squadActionsTotal.WithLabelValues("update", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update squad: " + err.Error()})
				return
			}
			// 409 conflict on first attempt — retry
			Logger.Infow("squad update conflict, retrying", "squad_name", name)
		}

		Logger.Infow("patched squad", "namespace", ns, "squad_name", name)
		squadActionsTotal.WithLabelValues("update", "success").Inc()
		c.JSON(http.StatusOK, squadToResponse(*updated))
	}
}

// deleteSquad deletes a KomputerSquad CR.
// @ID deleteSquad
// @Summary Delete squad
// @Description Deletes the squad CR. The operator will clean up the shared pod.
// @Tags squads
// @Produce json
// @Param name path string true "Squad name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Squad deleted"
// @Failure 404 {object} map[string]string "Squad not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /squads/{name} [delete]
func deleteSquad(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		if err := k8s.DeleteSquad(c.Request.Context(), ns, name); err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "squad not found"})
				return
			}
			squadActionsTotal.WithLabelValues("delete", "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete squad: " + err.Error()})
			return
		}

		Logger.Infow("deleted squad", "namespace", ns, "squad_name", name)
		squadActionsTotal.WithLabelValues("delete", "success").Inc()
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

// addSquadMember appends a member to an existing squad. Retries once on 409 conflict.
// @ID addSquadMember
// @Summary Add squad member
// @Description Appends a member (by ref or inline spec) to the squad's member list.
// @Tags squads
// @Accept json
// @Produce json
// @Param name path string true "Squad name"
// @Param namespace query string false "Kubernetes namespace"
// @Param request body AddSquadMemberRequest true "Member to add"
// @Success 200 {object} SquadResponse "Updated squad"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Squad not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /squads/{name}/members [post]
func addSquadMember(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		var req AddSquadMemberRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		if req.Ref == nil && req.Spec == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "exactly one of ref or spec must be set"})
			return
		}
		if req.Ref != nil && req.Spec != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "exactly one of ref or spec must be set"})
			return
		}

		// Block adoption of a running agent — must be asleep so the existing solo pod
		// is gone before the squad pod takes over its PVC.
		if req.Ref != nil {
			refNs := req.Ref.Namespace
			if refNs == "" {
				refNs = ns
			}
			if msg, blocked := validateMemberRefForAdoption(c, k8s, refNs, req.Ref.Name); blocked {
				c.JSON(http.StatusConflict, gin.H{"error": msg})
				return
			}
		}

		newMember := komputerv1alpha1.KomputerSquadMember{
			Ref:  req.Ref,
			Spec: req.Spec,
			Name: req.Name,
		}

		// Retry once on 409 conflict.
		var updated *komputerv1alpha1.KomputerSquad
		for attempt := 0; attempt < 2; attempt++ {
			squad, err := k8s.GetSquad(c.Request.Context(), ns, name)
			if err != nil {
				if errors.IsNotFound(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": "squad not found"})
					return
				}
				squadActionsTotal.WithLabelValues("add_member", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// If the new member is an inline spec, populate its InternalSystemPrompt
			// so it starts with the correct role + memories + squad context.
			if newMember.Spec != nil {
				role := newMember.Spec.Role
				if role == "" {
					role = "worker"
				}
				myName := newMember.Name
				if myName == "" {
					myName = fmt.Sprintf("%s-member-%d", squad.Name, len(squad.Spec.Members))
				}
				siblings := make([]string, 0, len(squad.Spec.Members))
				for _, m := range squad.Spec.Members {
					if m.Ref != nil {
						siblings = append(siblings, m.Ref.Name)
					} else if m.Name != "" {
						siblings = append(siblings, m.Name)
					}
				}
				newMember.Spec.InternalSystemPrompt = buildAgentInternalSystemPrompt(
					c.Request.Context(), k8s, ns, myName, role,
					newMember.Spec.Memories, squad.Name, siblings,
				)
			}

			squad.Spec.Members = append(squad.Spec.Members, newMember)

			updated, err = k8s.UpdateSquad(c.Request.Context(), squad)
			if err == nil {
				break
			}
			if !errors.IsConflict(err) || attempt == 1 {
				squadActionsTotal.WithLabelValues("add_member", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member: " + err.Error()})
				return
			}
			// 409 on first attempt — retry
			Logger.Infow("squad add_member conflict, retrying", "squad_name", name)
		}

		Logger.Infow("added member to squad", "namespace", ns, "squad_name", name)
		squadActionsTotal.WithLabelValues("add_member", "success").Inc()

		// Best-effort: steer all other current members to inform them of the new arrival.
		addedName := ""
		if newMember.Ref != nil {
			addedName = newMember.Ref.Name
		}
		if addedName != "" {
			msg := fmt.Sprintf("%s joined the squad. Continue your task.", addedName)
			for _, m := range updated.Spec.Members {
				if m.Ref == nil || m.Ref.Name == addedName {
					continue
				}
				if err := k8s.SteerAgent(c.Request.Context(), ns, m.Ref.Name, msg); err != nil {
					Logger.Warnw("failed to steer squad member on add", "squad_name", name, "member", m.Ref.Name, "error", err)
				}
			}
		}

		c.JSON(http.StatusOK, squadToResponse(*updated))
	}
}

// removeSquadMember removes a named member from a squad. Retries once on 409 conflict.
// @ID removeSquadMember
// @Summary Remove squad member
// @Description Removes the named member from the squad's member list (matched by ref.name or spec-based name).
// @Tags squads
// @Produce json
// @Param name path string true "Squad name"
// @Param agent path string true "Agent name to remove"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} SquadResponse "Updated squad"
// @Failure 404 {object} map[string]string "Squad or member not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /squads/{name}/members/{agent} [delete]
func removeSquadMember(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		squadName := c.Param("name")
		agentName := c.Param("agent")
		ns := resolveNamespace(c, k8s)

		// Retry once on 409 conflict.
		var updated *komputerv1alpha1.KomputerSquad
		for attempt := 0; attempt < 2; attempt++ {
			squad, err := k8s.GetSquad(c.Request.Context(), ns, squadName)
			if err != nil {
				if errors.IsNotFound(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": "squad not found"})
					return
				}
				squadActionsTotal.WithLabelValues("remove_member", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			filtered := make([]komputerv1alpha1.KomputerSquadMember, 0, len(squad.Spec.Members))
			found := false
			for _, m := range squad.Spec.Members {
				if m.Ref != nil && m.Ref.Name == agentName {
					found = true
					continue
				}
				filtered = append(filtered, m)
			}
			if !found {
				c.JSON(http.StatusNotFound, gin.H{"error": "member not found in squad: " + agentName})
				return
			}
			squad.Spec.Members = filtered

			updated, err = k8s.UpdateSquad(c.Request.Context(), squad)
			if err == nil {
				break
			}
			if !errors.IsConflict(err) || attempt == 1 {
				squadActionsTotal.WithLabelValues("remove_member", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member: " + err.Error()})
				return
			}
			// 409 on first attempt — retry
			Logger.Infow("squad remove_member conflict, retrying", "squad_name", squadName)
		}

		Logger.Infow("removed member from squad", "namespace", ns, "squad_name", squadName, "agent_name", agentName)
		squadActionsTotal.WithLabelValues("remove_member", "success").Inc()

		// Best-effort: steer all remaining members to inform them of the departure.
		msg := fmt.Sprintf("%s left the squad. Continue your task.", agentName)
		for _, m := range updated.Spec.Members {
			if m.Ref == nil {
				continue
			}
			if err := k8s.SteerAgent(c.Request.Context(), ns, m.Ref.Name, msg); err != nil {
				Logger.Warnw("failed to steer squad member on remove", "squad_name", squadName, "member", m.Ref.Name, "error", err)
			}
		}

		c.JSON(http.StatusOK, squadToResponse(*updated))
	}
}

// breakUpSquad marks the squad for dissolution. The operator deletes the squad
// CR (and via the cleanup finalizer, clears Status.Squad on each member) once
// every member is asleep. Sending tasks to sleeping members in the meantime is
// allowed — they wake, run, return to Sleeping, and the break-up eventually
// completes. PVCs are preserved.
//
// @ID breakUpSquad
// @Summary Request squad break-up
// @Description Marks the squad for dissolution once all members are asleep.
// @Tags squads
// @Produce json
// @Param name path string true "Squad name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} SquadResponse "Squad with break-up flag set"
// @Failure 404 {object} map[string]string "Squad not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /squads/{name}/break-up [post]
func breakUpSquad(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		// Retry once on 409 conflict.
		var updated *komputerv1alpha1.KomputerSquad
		for attempt := 0; attempt < 2; attempt++ {
			squad, err := k8s.GetSquad(c.Request.Context(), ns, name)
			if err != nil {
				if errors.IsNotFound(err) {
					c.JSON(http.StatusNotFound, gin.H{"error": "squad not found"})
					return
				}
				squadActionsTotal.WithLabelValues("break_up", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if squad.Spec.BreakUpRequested {
				// Idempotent — already requested.
				c.JSON(http.StatusOK, squadToResponse(*squad))
				return
			}

			squad.Spec.BreakUpRequested = true
			updated, err = k8s.UpdateSquad(c.Request.Context(), squad)
			if err == nil {
				break
			}
			if !errors.IsConflict(err) || attempt == 1 {
				squadActionsTotal.WithLabelValues("break_up", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to request break-up: " + err.Error()})
				return
			}
			Logger.Infow("squad break_up conflict, retrying", "squad_name", name)
		}

		Logger.Infow("squad break-up requested", "namespace", ns, "squad_name", name)
		squadActionsTotal.WithLabelValues("break_up", "success").Inc()
		c.JSON(http.StatusOK, squadToResponse(*updated))
	}
}
