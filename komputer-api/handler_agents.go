package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

type CreateAgentRequest struct {
	Name         string `json:"name" binding:"required"`
	Instructions string `json:"instructions" binding:"required"`
	Model        string `json:"model"`
	// Effort sets the Claude Agent SDK's reasoning effort: "low", "medium",
	// "high", "xhigh", or "max". Empty means unset, so the model's own default
	// effort applies.
	Effort      string   `json:"effort,omitempty"`
	TemplateRef string   `json:"templateRef"`
	Role        string   `json:"role"`       // "manager" or "" (default manager)
	Namespace   string   `json:"namespace"`  // optional, defaults to server default
	SecretRefs  []string `json:"secretRefs"` // names of existing K8s Secrets to attach
	Memories    []string `json:"memories"`   // optional KomputerMemory names to attach
	Skills      []string `json:"skills"`     // optional KomputerSkill names to attach
	Connectors  []string `json:"connectors"` // optional KomputerConnector names to attach
	// AllowedTools restricts the agent to exactly these tools. REPLACES the default
	// built-in set rather than extending it, and connector tools are not auto-added.
	// Supports wildcards, e.g. "mcp__figma__*". Empty keeps default behavior.
	AllowedTools []string `json:"allowedTools"`
	// DisallowedTools removes these tools; everything else stays available.
	// Takes precedence over AllowedTools. Supports wildcards.
	DisallowedTools []string `json:"disallowedTools"`
	Lifecycle       string   `json:"lifecycle"` // "", "Sleep", "AutoDelete", or "Archive"
	// SleepTTL puts the agent to sleep after this long with no activity, as a Go
	// duration string (e.g. "30m", "2h"). Empty means never auto-sleep.
	SleepTTL string `json:"sleepTTL,omitempty"`
	// DeleteTTL deletes the agent this long after creation, as a Go duration string
	// (e.g. "24h"). Absolute — it does not reset on wake. Empty means never auto-delete.
	DeleteTTL string `json:"deleteTTL,omitempty"`
	// TaskTimeout cancels a running task once it has run this long, as a Go duration
	// string (e.g. "30m"). A hard wall-clock cap per task — steering does not extend
	// it. Empty means tasks run without a time limit.
	TaskTimeout string `json:"taskTimeout,omitempty"`
	// BudgetUSD is a SOFT lifetime spending cap across all of this agent's tasks, as
	// a decimal USD string (e.g. "10.00"). Once status.totalCostUSD reaches or
	// exceeds it, new tasks are refused with 409 until it's raised; a task already
	// running is always allowed to finish. Empty means no lifetime cap.
	BudgetUSD string `json:"budgetUSD,omitempty"`
	// TaskBudgetUSD is a HARD per-task spending cap, as a decimal USD string (e.g.
	// "2.50"). The SDK stops a task the instant it is exceeded, mid-task. Empty
	// means tasks run without a per-task cap.
	TaskBudgetUSD string                        `json:"taskBudgetUSD,omitempty"`
	OfficeManager string                        `json:"officeManager"`      // set by manager MCP tool
	SystemPrompt  string                        `json:"systemPrompt"`       // optional custom system prompt
	Priority      int32                         `json:"priority,omitempty"` // queue priority; higher = admitted first
	PodSpec       *corev1.PodSpec               `json:"podSpec,omitempty"`
	Storage       *komputerv1alpha1.StorageSpec `json:"storage,omitempty"`
	// Labels are user-defined key=value labels passed through to the agent CR.
	// Reserved-prefix keys (komputer.ai/*) are rejected except for
	// "komputer.ai/personal-agent" which is allow-listed.
	Labels map[string]string `json:"labels,omitempty"`
}

type AgentResponse struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Model     string `json:"model"`
	// Effort is this agent's reasoning effort override ("low", "medium", "high",
	// "xhigh", "max"). Empty means unset — the model's own default effort applies.
	Effort          string `json:"effort,omitempty"`
	Status          string `json:"status"`
	TaskStatus      string `json:"taskStatus,omitempty"`
	LastTaskMessage string `json:"lastTaskMessage,omitempty"`
	Lifecycle       string `json:"lifecycle,omitempty"`
	SleepTTL        string `json:"sleepTTL,omitempty"`    // idle timeout before auto-sleep, e.g. "30m"
	DeleteTTL       string `json:"deleteTTL,omitempty"`   // absolute lifetime before auto-delete, e.g. "24h"
	TaskTimeout     string `json:"taskTimeout,omitempty"` // per-task wall-clock cap, e.g. "30m"
	// BudgetUSD is this agent's own SOFT lifetime spending cap (spec.budgetUSD), as
	// a decimal USD string. Empty when unset on the agent itself — note the agent
	// may still be bound by a template default; BudgetExhausted reflects that
	// effective value even when this field is empty.
	BudgetUSD string `json:"budgetUSD,omitempty"`
	// TaskBudgetUSD is this agent's own HARD per-task spending cap
	// (spec.taskBudgetUSD), as a decimal USD string. Empty when unset on the agent
	// itself (it may still inherit a template default).
	TaskBudgetUSD string `json:"taskBudgetUSD,omitempty"`
	// BudgetExhausted is true when status.totalCostUSD has reached or passed the
	// EFFECTIVE lifetime budget (this agent's own BudgetUSD, else its template's
	// default) — new tasks are refused with 409 until it's raised. Always false
	// when no lifetime budget applies anywhere. Computed by the API so clients
	// don't have to re-derive the template resolution themselves.
	BudgetExhausted bool `json:"budgetExhausted,omitempty"`
	// LastActivityAt is when the agent last saw task activity (RFC3339). The idle
	// clock sleepTTL is measured against.
	LastActivityAt string `json:"lastActivityAt,omitempty"`
	// SleepExpiresAt / DeleteExpiresAt are when the TTLs will fire (RFC3339). Empty
	// when the matching TTL is unset or its countdown isn't currently running.
	SleepExpiresAt  string `json:"sleepExpiresAt,omitempty"`
	DeleteExpiresAt string `json:"deleteExpiresAt,omitempty"`
	// TaskStartedAt is when the current (or most recent) task started (RFC3339).
	// TaskExpiresAt is when taskTimeout will cancel it; empty when no task is running.
	TaskStartedAt      string                        `json:"taskStartedAt,omitempty"`
	TaskExpiresAt      string                        `json:"taskExpiresAt,omitempty"`
	LastTaskCostUSD    string                        `json:"lastTaskCostUSD,omitempty"`
	TotalCostUSD       string                        `json:"totalCostUSD,omitempty"`
	TotalTokens        int64                         `json:"totalTokens,omitempty"`
	ModelContextWindow int64                         `json:"modelContextWindow,omitempty"`
	Secrets            []string                      `json:"secrets,omitempty"`         // Key names from K8s Secrets (not values)
	Memories           []string                      `json:"memories,omitempty"`        // KomputerMemory names attached to this agent
	Skills             []string                      `json:"skills,omitempty"`          // KomputerSkill names attached to this agent
	Connectors         []string                      `json:"connectors,omitempty"`      // KomputerConnector names attached to this agent
	AllowedTools       []string                      `json:"allowedTools,omitempty"`    // Tools this agent is restricted to (empty = defaults)
	DisallowedTools    []string                      `json:"disallowedTools,omitempty"` // Tools removed from this agent
	Instructions       string                        `json:"instructions,omitempty"`    // User task (spec.instructions)
	SystemPrompt       string                        `json:"systemPrompt,omitempty"`    // Custom system prompt (spec.systemPrompt)
	CreatedAt          string                        `json:"createdAt"`
	Priority           int32                         `json:"priority"`
	QueuePosition      int32                         `json:"queuePosition,omitempty"`
	QueueReason        string                        `json:"queueReason,omitempty"`
	Squad              bool                          `json:"squad,omitempty"`     // True when this agent is managed by a KomputerSquad
	SquadName          string                        `json:"squadName,omitempty"` // Name of the squad managing this agent (when Squad=true)
	PodSpec            *corev1.PodSpec               `json:"podSpec,omitempty"`
	Storage            *komputerv1alpha1.StorageSpec `json:"storage,omitempty"`
	// Errors are non-fatal failures that occurred during the request (e.g. CR was patched
	// but live-pod sync failed). The CR change still took effect; the UI can surface these
	// as toasts so the user knows something didn't fully apply.
	Errors         []string          `json:"errors,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	CompletionTime string            `json:"completionTime,omitempty"`
}

// buildAgentInternalSystemPrompt assembles the internal system prompt for an
// agent. It is the shared logic used both when creating a solo agent and when
// creating a squad (inline member specs need this too — otherwise squad
// agents start with empty KOMPUTER_INTERNAL_SYSTEM_PROMPT).
//
// squadName/squadMembers describe the squad the agent belongs to (if any).
// Pass squadName="" for a solo agent.
func buildAgentInternalSystemPrompt(
	ctx context.Context,
	k8s *K8sClient,
	ns, agentName, role string,
	memories []string,
	squadName string,
	squadSiblings []string, // member names other than agentName
) string {
	memorySection, _ := k8s.ResolveMemoryContent(ctx, ns, memories)
	agentHeader := fmt.Sprintf("\n---\n\n**Agent Name:** %s", agentName)
	base := workerSystemPrompt
	if role == "manager" {
		base = managerSystemPrompt
	}
	prompt := base + memorySection + agentHeader
	if squadName != "" {
		prompt += fmt.Sprintf("\nYou are part of squad %q with members: %s. Their workspaces are mounted read/write at /agents/<name>/workspace.", squadName, strings.Join(squadSiblings, ", "))
	}
	return prompt
}

// resolveSquadName returns the squad name managing this agent, or "" if not in a squad.
// Uses agent.Status.Squad as the gate (cheap), then looks up the name via the API client.
// Returns "" silently on lookup failure (the Squad bool flag is the source of truth).
func resolveSquadName(ctx context.Context, k8s *K8sClient, agent *komputerv1alpha1.KomputerAgent) string {
	if !agent.Status.Squad {
		return ""
	}
	squad, err := k8s.FindSquadForAgent(ctx, agent.Namespace, agent.Name)
	if err != nil || squad == nil {
		return ""
	}
	return squad.Name
}

// buildSquadNameMap groups all squads by member name for cheap O(1) lookup during list endpoints.
// Returns map[namespace/agentName] -> squadName.
func buildSquadNameMap(ctx context.Context, k8s *K8sClient, ns string) map[string]string {
	out := make(map[string]string)
	squads, err := k8s.ListSquads(ctx, ns)
	if err != nil {
		return out
	}
	for _, s := range squads {
		for _, m := range s.Spec.Members {
			if m.Ref == nil {
				continue
			}
			memberNs := m.Ref.Namespace
			if memberNs == "" {
				memberNs = s.Namespace
			}
			out[memberNs+"/"+m.Ref.Name] = s.Name
		}
	}
	return out
}

const personalAgentLabel = "komputer.ai/personal-agent"

// validateLabelKey rejects keys with the komputer.ai/ reserved prefix (except
// the personal-agent allow-listed key) and enforces K8s qualified-name rules.
// Shared by validateUserLabels (labels to set) and validateRemoveLabelKeys
// (labels to remove), which don't carry a value to additionally validate.
func validateLabelKey(k string) error {
	if strings.HasPrefix(k, "komputer.ai/") && k != personalAgentLabel {
		return fmt.Errorf("label key %q is reserved", k)
	}
	if errs := validation.IsQualifiedName(k); len(errs) > 0 {
		return fmt.Errorf("invalid label key %q: %s", k, strings.Join(errs, ", "))
	}
	return nil
}

// validateUserLabels rejects keys with the komputer.ai/ reserved prefix
// (except the personal-agent allow-listed key) and validates that every
// key/value conforms to K8s label rules.
func validateUserLabels(labels map[string]string) error {
	for k, v := range labels {
		if err := validateLabelKey(k); err != nil {
			return err
		}
		if errs := validation.IsValidLabelValue(v); len(errs) > 0 {
			return fmt.Errorf("invalid label value for key %q: %s", k, strings.Join(errs, ", "))
		}
	}
	return nil
}

// validateRemoveLabelKeys applies the same reserved-prefix / qualified-name
// rule to label keys requested for removal via PatchAgentRequest.RemoveLabels.
func validateRemoveLabelKeys(keys []string) error {
	for _, k := range keys {
		if err := validateLabelKey(k); err != nil {
			return err
		}
	}
	return nil
}

// formatTime formats a *metav1.Time as RFC3339, returning "" for nil/zero.
func formatTime(t *metav1.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// mergeDefaultSkills adds default skill names to the explicit list, deduplicating.
func mergeDefaultSkills(explicit []string, defaults []string) []string {
	seen := make(map[string]bool, len(explicit))
	for _, s := range explicit {
		seen[s] = true
	}
	merged := append([]string{}, explicit...)
	for _, s := range defaults {
		if !seen[s] {
			merged = append(merged, s)
		}
	}
	return merged
}

type AgentListResponse struct {
	Agents []AgentResponse `json:"agents"`
	// Total is the number of agents matching the status/label/q filters, BEFORE
	// the limit/offset slice is applied. Always populated (even when `limit`
	// was omitted, in which case it simply equals len(Agents)).
	Total int `json:"total"`
	// NextOffset is the offset to request for the next page; omitted once
	// there is no more data (or when `limit` was omitted).
	NextOffset *int `json:"nextOffset,omitempty"`
	// Facets lists every distinct namespace and "key=value" label pair across
	// agents in the requested namespace scope, computed BEFORE status/label/q
	// filtering — see AgentListFacets doc.
	Facets AgentListFacets `json:"facets"`
}

// parseTTL converts a Go duration string into a *metav1.Duration. An empty string
// yields nil (TTL unset). Rejects zero and negative values, which would otherwise
// mean "expire immediately" and delete an agent the moment it is created.
func parseTTL(field, value string) (*metav1.Duration, error) {
	if value == "" {
		return nil, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", field, err)
	}
	if d <= 0 {
		return nil, fmt.Errorf("invalid %s: must be a positive duration, got %q", field, value)
	}
	return &metav1.Duration{Duration: d}, nil
}

// parseTTLUpdate builds a tri-state TTL patch from an optional duration string.
// A nil pointer leaves the field alone; an explicit "" clears it.
func parseTTLUpdate(field string, value *string) (TTLUpdate, error) {
	if value == nil {
		return TTLUpdate{}, nil
	}
	d, err := parseTTL(field, *value)
	if err != nil {
		return TTLUpdate{}, err
	}
	return TTLUpdate{Set: true, Value: d}, nil
}

// withAgentTTL returns resp with the agent's TTL/budget fields populated. A
// pass-through so the inline c.JSON(...) response sites stay single expressions.
// Takes ctx/k8s (rather than being pure like fillAgentTTL) because
// BudgetExhausted needs the effective budget, which may come from the agent's
// template.
func withAgentTTL(ctx context.Context, k8s *K8sClient, resp AgentResponse, agent *komputerv1alpha1.KomputerAgent) AgentResponse {
	fillAgentTTL(&resp, agent)
	resp.BudgetExhausted = k8s.IsAgentBudgetExhausted(ctx, agent)
	return resp
}

// fillAgentTTL copies an agent's TTL spec and status timestamps onto a response.
// Kept as a helper because every response path needs all eight fields.
func fillAgentTTL(resp *AgentResponse, agent *komputerv1alpha1.KomputerAgent) {
	if agent.Spec.SleepTTL != nil {
		resp.SleepTTL = agent.Spec.SleepTTL.Duration.String()
	}
	if agent.Spec.DeleteTTL != nil {
		resp.DeleteTTL = agent.Spec.DeleteTTL.Duration.String()
	}
	if agent.Status.LastActivityAt != nil {
		resp.LastActivityAt = agent.Status.LastActivityAt.Format(time.RFC3339)
	}
	if agent.Status.SleepExpiresAt != nil {
		resp.SleepExpiresAt = agent.Status.SleepExpiresAt.Format(time.RFC3339)
	}
	if agent.Status.DeleteExpiresAt != nil {
		resp.DeleteExpiresAt = agent.Status.DeleteExpiresAt.Format(time.RFC3339)
	}
	if agent.Spec.TaskTimeout != nil {
		resp.TaskTimeout = agent.Spec.TaskTimeout.Duration.String()
	}
	if agent.Status.TaskStartedAt != nil {
		resp.TaskStartedAt = agent.Status.TaskStartedAt.Format(time.RFC3339)
	}
	if agent.Status.TaskExpiresAt != nil {
		resp.TaskExpiresAt = agent.Status.TaskExpiresAt.Format(time.RFC3339)
	}
	resp.BudgetUSD = agent.Spec.BudgetUSD
	resp.TaskBudgetUSD = agent.Spec.TaskBudgetUSD
}

type PatchAgentRequest struct {
	Model *string `json:"model,omitempty"`
	// Effort is "low", "medium", "high", "xhigh", "max", or "" to clear back to
	// unset (model default). Omitting the field leaves it unchanged.
	Effort       *string   `json:"effort,omitempty"`
	Lifecycle    *string   `json:"lifecycle,omitempty"`
	Instructions *string   `json:"instructions,omitempty"`
	TemplateRef  *string   `json:"templateRef,omitempty"`
	SecretRefs   *[]string `json:"secretRefs,omitempty"` // full replacement list of K8s secret names
	Memories     *[]string `json:"memories,omitempty"`   // memory names to attach
	Skills       *[]string `json:"skills,omitempty"`     // skill names to attach
	Connectors   *[]string `json:"connectors,omitempty"` // connector names to attach
	// AllowedTools restricts the agent to these tools; an explicit [] clears the
	// restriction and restores the default tool set.
	AllowedTools *[]string `json:"allowedTools,omitempty"`
	// DisallowedTools removes these tools; an explicit [] clears the list.
	DisallowedTools *[]string `json:"disallowedTools,omitempty"`
	SystemPrompt    *string   `json:"systemPrompt,omitempty"` // custom system prompt
	// SleepTTL / DeleteTTL are Go duration strings (e.g. "30m"). An explicit ""
	// clears the TTL; omitting the field leaves it unchanged.
	SleepTTL    *string `json:"sleepTTL,omitempty"`
	DeleteTTL   *string `json:"deleteTTL,omitempty"`
	TaskTimeout *string `json:"taskTimeout,omitempty"`
	// BudgetUSD / TaskBudgetUSD are decimal USD strings (e.g. "10.00"). An explicit
	// "" clears the cap; omitting the field leaves it unchanged.
	BudgetUSD     *string                       `json:"budgetUSD,omitempty"`
	TaskBudgetUSD *string                       `json:"taskBudgetUSD,omitempty"`
	Priority      *int32                        `json:"priority,omitempty"` // pointer so 0 vs unset is distinguishable
	PodSpec       *corev1.PodSpec               `json:"podSpec,omitempty"`
	Storage       *komputerv1alpha1.StorageSpec `json:"storage,omitempty"`
	Labels        map[string]string             `json:"labels,omitempty"`
	// RemoveLabels are user label keys to remove. A key present in both Labels
	// and RemoveLabels is rejected as ambiguous.
	RemoveLabels []string `json:"removeLabels,omitempty"`
}

// createOrTriggerAgent creates a new agent or sends a task to an existing one.
// @ID createAgent
// @Summary Create agent or send task
// @Description Creates a new agent or sends a task to an existing idle agent (upsert by name).
// @Description If the agent doesn't exist, it is created. If it exists and is idle, the task is forwarded.
// @Tags agents
// @Accept json
// @Produce json
// @Param request body CreateAgentRequest true "Agent creation request"
// @Success 200 {object} AgentResponse "Agent created or task forwarded"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 409 {object} map[string]string "Agent is busy or has no running pod"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents [post]
func createOrTriggerAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAgentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())

		// Validate agent name: must be a valid K8s DNS subdomain name.
		if !isValidK8sName(req.Name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent name: must be lowercase, alphanumeric, hyphens only, max 63 chars (e.g. 'my-agent-1')"})
			return
		}

		ns := req.Namespace
		if ns == "" {
			ns = resolveNamespace(c, k8s)
		}

		if err := validateUserLabels(req.Labels); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Parse TTLs up front so a malformed duration 400s before any side effects
		// (namespace creation, secret mirroring) happen.
		sleepTTL, err := parseTTL("sleepTTL", req.SleepTTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		deleteTTL, err := parseTTL("deleteTTL", req.DeleteTTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		taskTimeout, err := parseTTL("taskTimeout", req.TaskTimeout)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		budgetUSD, err := parseBudget("budgetUSD", req.BudgetUSD)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		taskBudgetUSD, err := parseBudget("taskBudgetUSD", req.TaskBudgetUSD)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Build full instructions with system prompt early — needed for both new agents and wake-up.
		role := req.Role
		if role == "" {
			role = "manager"
		}
		if role != "worker" && role != "manager" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'worker' or 'manager'"})
			return
		}
		// System prompt and user instructions are sent separately to the agent.
		// The agent passes system_prompt to the Claude SDK's system_prompt field
		// (not as part of conversation history — replaced on each task, never accumulates).
		instructions := req.Instructions

		// Build internal system prompt — memory resolution happens after we know if agent exists
		// (existing agents use CR memories, new agents use request memories).
		// Looks up the agent's squad (if any) so the prompt mentions siblings.
		buildInternalSystemPrompt := func(memories []string) string {
			squadName := ""
			var siblings []string
			if squad, err := k8s.FindSquadForAgent(c.Request.Context(), ns, req.Name); err == nil && squad != nil {
				squadName = squad.Name
				for _, m := range squad.Spec.Members {
					if m.Ref != nil && m.Ref.Name != req.Name {
						siblings = append(siblings, m.Ref.Name)
					}
				}
			}
			return buildAgentInternalSystemPrompt(c.Request.Context(), k8s, ns, req.Name, role, memories, squadName, siblings)
		}

		existing, err := k8s.GetAgent(c.Request.Context(), ns, req.Name)
		if err != nil && !errors.IsNotFound(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check agent: " + err.Error()})
			return
		}
		if existing != nil && agentArchived(existing) {
			c.JSON(http.StatusConflict, gin.H{"error": ErrAgentArchived.Error()})
			return
		}

		// On Bedrock, reject friendly model names before they reach the SDK. A new
		// agent requires a valid model (empty would fall back to a friendly default);
		// wake/forward only reject a non-empty bad override (empty = keep existing).
		if err := validateBedrockModel(req.Model, existing == nil); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validateEffort(req.Effort); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// An ephemeral workspace (no PVC) can never be combined with Sleep/sleepTTL —
		// sleeping deletes the pod, which would wipe it. For a new agent this checks
		// the request's own values; for wake (existing != nil), storage can't change
		// here, so it's the existing agent's effective ephemeral/sleepTTL plus any
		// Lifecycle override the wake call is making.
		{
			lifecycleToCheck := req.Lifecycle
			var sleepTTLToCheck *metav1.Duration
			var effectiveEphemeral bool
			if existing != nil {
				effectiveEphemeral = k8s.ResolveEffectiveEphemeral(c.Request.Context(), ns, existing.Spec.AgentConfigSpec)
				if lifecycleToCheck == "" {
					lifecycleToCheck = string(existing.Spec.Lifecycle)
				}
				sleepTTLToCheck = existing.Spec.SleepTTL // wake cannot change sleepTTL
			} else {
				effectiveEphemeral = k8s.ResolveEffectiveEphemeral(c.Request.Context(), ns, komputerv1alpha1.AgentConfigSpec{
					TemplateRef: req.TemplateRef,
					Storage:     req.Storage,
				})
				sleepTTLToCheck = sleepTTL
			}
			if err := validateEphemeralStorage(effectiveEphemeral, lifecycleToCheck, sleepTTLToCheck); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		if existing != nil {
			// Wake-up flow for sleeping agents
			if existing.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
				// A wake-with-task always starts a brand new task, so the lifetime
				// budget gates it the same way it gates a forward below. A plain idle
				// wake (no task) goes through the separate /wake endpoint instead,
				// which never reaches this branch.
				if refuseIfBudgetExhausted(c, k8s, existing) {
					agentActionsTotal.WithLabelValues("wake", "error").Inc()
					return
				}
				// Use CR memories (may have been updated via PATCH since creation)
				wakeMemories := existing.Spec.Memories
				if len(req.Memories) > 0 {
					wakeMemories = req.Memories
				}
				wakeSystemPrompt := req.SystemPrompt
				if wakeSystemPrompt == "" {
					wakeSystemPrompt = existing.Spec.SystemPrompt
				}
				if err := k8s.WakeAgent(c.Request.Context(), ns, req.Name, instructions, buildInternalSystemPrompt(wakeMemories), wakeSystemPrompt, req.Model, req.Lifecycle); err != nil {
					agentActionsTotal.WithLabelValues("wake", "error").Inc()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to wake agent: " + err.Error()})
					return
				}
				// Squad-member wake: if the squad pod is still up (other members weren't
				// sleeping), this member's container is running idle (KOMPUTER_WAKE_IDLE=true).
				// Push the task over HTTP since it won't auto-start. If the pod was deleted,
				// the squad controller will rebuild it and the new container picks up
				// instructions from env vars on startup.
				if existing.Status.Squad && existing.Status.PodName != "" {
					effectiveTaskBudget := k8s.ResolveEffectiveTaskBudgetUSD(c.Request.Context(), ns, existing.Spec.AgentConfigSpec)
					if _, fwdErr := k8s.ForwardTaskToAgent(c.Request.Context(), ns, existing.Status.PodName, existing.Name, instructions, req.Model, buildInternalSystemPrompt(wakeMemories), wakeSystemPrompt, effectiveTaskBudget); fwdErr != nil {
						Logger.Warnw("squad-wake task forward failed (will rely on pod rebuild)", "agent_name", req.Name, "error", fwdErr)
					}
				}
				Logger.Infow("waking sleeping agent", "namespace", ns, "agent_name", req.Name)
				agentActionsTotal.WithLabelValues("wake", "success").Inc()
				c.JSON(http.StatusOK, withAgentTTL(c.Request.Context(), k8s, AgentResponse{
					Name:               existing.Name,
					Namespace:          existing.Namespace,
					Model:              existing.Spec.Model,
					Effort:             existing.Spec.Effort,
					Status:             "Pending",
					Lifecycle:          string(existing.Spec.Lifecycle),
					LastTaskCostUSD:    existing.Status.LastTaskCostUSD,
					TotalCostUSD:       existing.Status.TotalCostUSD,
					TotalTokens:        existing.Status.TotalTokens,
					ModelContextWindow: existing.Status.ModelContextWindow,
					Secrets:            collectSecretKeys(*c, k8s, ns, existing.Spec.Secrets),
					Memories:           existing.Spec.Memories,
					Skills:             mergeDefaultSkills(existing.Spec.Skills, defaultSkills),
					Connectors:         existing.Spec.Connectors,
					AllowedTools:       existing.Spec.AllowedTools,
					DisallowedTools:    existing.Spec.DisallowedTools,
					Instructions:       existing.Spec.Instructions,
					SystemPrompt:       existing.Spec.SystemPrompt,
					CreatedAt:          existing.CreationTimestamp.UTC().Format(time.RFC3339),
					Priority:           existing.Spec.Priority,
					QueuePosition:      existing.Status.QueuePosition,
					QueueReason:        existing.Status.QueueReason,
					PodSpec:            existing.Spec.PodSpec,
					Storage:            existing.Spec.Storage,
					Labels:             existing.Spec.Labels,
					CompletionTime:     formatTime(existing.Status.CompletionTime),
				}, existing))
				return
			}

			if existing.Status.PodName == "" {
				c.JSON(http.StatusConflict, gin.H{"error": "agent exists but has no running pod yet"})
				return
			}

			// If the agent is busy, the message is queued as a steer (follow-up).
			// The agent's /task endpoint handles both new tasks and steers.
			//
			// A NEW task (agent not currently mid-task) must update spec.Instructions
			// before it is forwarded, so that if the pod is later rescheduled the
			// operator re-runs THIS task rather than the stale one from agent
			// creation (see shouldStartIdle in the operator). A steer must NOT do
			// this: overwriting Instructions with just the steer text would make an
			// interrupted original task resume as only the follow-up message.
			// If the forward below fails after this patch, that's harmless — the
			// task never reaches InProgress, so a reschedule won't auto-run it
			// either; the next real task request will patch Instructions again.
			//
			// This is also the one place a NEW task can be refused for an exhausted
			// lifetime budget: a steer (the else case, agent already mid-task) is
			// always allowed to reach the running task, since that task is allowed
			// to finish even if it has pushed spend past the cap.
			if !agentTaskInProgress(existing.Status.TaskStatus) {
				if refuseIfBudgetExhausted(c, k8s, existing) {
					agentActionsTotal.WithLabelValues("wake", "error").Inc()
					return
				}
				if err := k8s.PatchAgentSpec(c.Request.Context(), ns, req.Name, nil, nil, nil, &instructions, nil, nil, nil, TTLUpdate{}, TTLUpdate{}, TTLUpdate{}, StringUpdate{}, StringUpdate{}); err != nil {
					Logger.Warnw("failed to patch instructions before forwarding new task", "agent_name", req.Name, "error", err)
				}
			}

			// Use CR memories for existing agents (may have been updated via PATCH)
			forwardMemories := existing.Spec.Memories
			if len(req.Memories) > 0 {
				forwardMemories = req.Memories
			}
			forwardSystemPrompt := req.SystemPrompt
			if forwardSystemPrompt == "" {
				forwardSystemPrompt = existing.Spec.SystemPrompt
			}
			// Update lifecycle if changed — before forwarding. Every lifecycle
			// transition (Sleep, AutoDelete, Archive) fires on a finished task's
			// Complete/Error status, so for a new task the previous task's status is
			// cleared first: otherwise the operator could act on it before this task
			// reports in, killing the task (and, for Archive, deleting its workspace).
			// A steer leaves TaskStatus at InProgress, which never triggers them.
			newTask := !agentTaskInProgress(existing.Status.TaskStatus)
			restoreTaskStatus := false
			if req.Lifecycle != "" && req.Lifecycle != string(existing.Spec.Lifecycle) {
				if newTask && existing.Status.TaskStatus != "" {
					if err := k8s.SetAgentTaskStatus(c.Request.Context(), ns, req.Name, ""); err != nil {
						Logger.Warnw("failed to clear task status before lifecycle change", "agent_name", req.Name, "error", err)
					} else {
						restoreTaskStatus = true
					}
				}
				if err := k8s.PatchAgentLifecycle(c.Request.Context(), ns, req.Name, req.Lifecycle); err != nil {
					Logger.Warnw("failed to patch lifecycle", "agent_name", req.Name, "error", err)
				}
			}

			effectiveTaskBudget := k8s.ResolveEffectiveTaskBudgetUSD(c.Request.Context(), ns, existing.Spec.AgentConfigSpec)
			cw, err := k8s.ForwardTaskToAgent(c.Request.Context(), ns, existing.Status.PodName, existing.Name, instructions, req.Model, buildInternalSystemPrompt(forwardMemories), forwardSystemPrompt, effectiveTaskBudget)
			if err != nil {
				if restoreTaskStatus {
					if restoreErr := k8s.SetAgentTaskStatus(c.Request.Context(), ns, req.Name, existing.Status.TaskStatus); restoreErr != nil {
						Logger.Warnw("failed to restore task status after failed forward", "agent_name", req.Name, "error", restoreErr)
					}
				}
				agentActionsTotal.WithLabelValues("wake", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to forward task: " + err.Error()})
				return
			}
			if cw > 0 {
				if patchErr := k8s.PatchAgentContextWindow(c.Request.Context(), ns, req.Name, cw); patchErr != nil {
					Logger.Warnw("failed to patch context window", "agent_name", req.Name, "error", patchErr)
				}
			}

			// Update TTLs/budgets if this request carried new ones. Only non-empty
			// values are applied — an omitted TTL/budget on a task request must not
			// silently clear an existing one, since callers routinely send just
			// name + instructions.
			if sleepTTL != nil || deleteTTL != nil || taskTimeout != nil || budgetUSD != "" || taskBudgetUSD != "" {
				if err := k8s.PatchAgentSpec(c.Request.Context(), ns, req.Name, nil, nil, nil, nil, nil, nil, nil,
					TTLUpdate{Set: sleepTTL != nil, Value: sleepTTL},
					TTLUpdate{Set: deleteTTL != nil, Value: deleteTTL},
					TTLUpdate{Set: taskTimeout != nil, Value: taskTimeout},
					StringUpdate{Set: budgetUSD != "", Value: budgetUSD},
					StringUpdate{Set: taskBudgetUSD != "", Value: taskBudgetUSD}); err != nil {
					Logger.Warnw("failed to patch TTLs/budgets", "agent_name", req.Name, "error", err)
				}
			}

			Logger.Infow("forwarded task to existing agent", "namespace", ns, "agent_name", req.Name)
			agentActionsTotal.WithLabelValues("wake", "success").Inc()
			c.JSON(http.StatusOK, withAgentTTL(c.Request.Context(), k8s, AgentResponse{
				Name:               existing.Name,
				Namespace:          existing.Namespace,
				Model:              existing.Spec.Model,
				Effort:             existing.Spec.Effort,
				Status:             string(existing.Status.Phase),
				TaskStatus:         string(existing.Status.TaskStatus),
				LastTaskMessage:    existing.Status.LastTaskMessage,
				Lifecycle:          string(existing.Spec.Lifecycle),
				LastTaskCostUSD:    existing.Status.LastTaskCostUSD,
				TotalCostUSD:       existing.Status.TotalCostUSD,
				TotalTokens:        existing.Status.TotalTokens,
				ModelContextWindow: existing.Status.ModelContextWindow,
				Secrets:            collectSecretKeys(*c, k8s, ns, existing.Spec.Secrets),
				Memories:           existing.Spec.Memories,
				Skills:             mergeDefaultSkills(existing.Spec.Skills, defaultSkills),
				Connectors:         existing.Spec.Connectors,
				AllowedTools:       existing.Spec.AllowedTools,
				DisallowedTools:    existing.Spec.DisallowedTools,
				Instructions:       existing.Spec.Instructions,
				SystemPrompt:       existing.Spec.SystemPrompt,
				CreatedAt:          existing.CreationTimestamp.UTC().Format(time.RFC3339),
				Priority:           existing.Spec.Priority,
				QueuePosition:      existing.Status.QueuePosition,
				QueueReason:        existing.Status.QueueReason,
				PodSpec:            existing.Spec.PodSpec,
				Storage:            existing.Spec.Storage,
				Labels:             existing.Spec.Labels,
				CompletionTime:     formatTime(existing.Status.CompletionTime),
			}, existing))
			return
		}

		// A new agent has no existing connectors to compare against — every connector in
		// the request is a fresh attach, so a disabled one is always rejected here.
		if err := k8s.ValidateConnectorsAttachable(c.Request.Context(), ns, req.Connectors, nil); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Inherit connectors from office manager so sub-agents get the same MCP tools.
		connectors := req.Connectors
		if req.OfficeManager != "" {
			manager, mgrErr := k8s.GetAgent(c.Request.Context(), ns, req.OfficeManager)
			if mgrErr == nil && len(manager.Spec.Connectors) > 0 {
				seen := make(map[string]bool, len(connectors))
				for _, c := range connectors {
					seen[c] = true
				}
				for _, c := range manager.Spec.Connectors {
					if !seen[c] {
						connectors = append(connectors, c)
					}
				}
			}
		}

		if err := k8s.EnsureNamespaceExists(c.Request.Context(), ns); err != nil {
			agentActionsTotal.WithLabelValues("create", "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure namespace: " + err.Error()})
			return
		}

		agent, err := k8s.CreateAgent(c.Request.Context(), ns, req.Name, instructions,
			buildInternalSystemPrompt(req.Memories), req.OfficeManager,
			komputerv1alpha1.AgentConfigSpec{
				TemplateRef:     req.TemplateRef,
				SystemPrompt:    req.SystemPrompt,
				Model:           req.Model,
				Effort:          req.Effort,
				Role:            role,
				Secrets:         req.SecretRefs,
				Skills:          req.Skills,
				Memories:        req.Memories,
				Connectors:      connectors,
				AllowedTools:    req.AllowedTools,
				DisallowedTools: req.DisallowedTools,
				Lifecycle:       komputerv1alpha1.AgentLifecycle(req.Lifecycle),
				SleepTTL:        sleepTTL,
				DeleteTTL:       deleteTTL,
				TaskTimeout:     taskTimeout,
				BudgetUSD:       budgetUSD,
				TaskBudgetUSD:   taskBudgetUSD,
				Priority:        req.Priority,
				PodSpec:         req.PodSpec,
				Storage:         req.Storage,
				Labels:          req.Labels,
			})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "agent already exists: " + req.Name})
				return
			}
			agentActionsTotal.WithLabelValues("create", "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent: " + err.Error()})
			return
		}

		Logger.Infow("created new agent", "namespace", ns, "agent_name", req.Name)
		agentActionsTotal.WithLabelValues("create", "success").Inc()
		c.JSON(http.StatusOK, withAgentTTL(c.Request.Context(), k8s, AgentResponse{
			Name:               agent.Name,
			Namespace:          agent.Namespace,
			Model:              agent.Spec.Model,
			Effort:             agent.Spec.Effort,
			Status:             "Pending",
			Lifecycle:          string(agent.Spec.Lifecycle),
			Secrets:            collectSecretKeys(*c, k8s, ns, agent.Spec.Secrets),
			Memories:           agent.Spec.Memories,
			Skills:             mergeDefaultSkills(agent.Spec.Skills, defaultSkills),
			Connectors:         agent.Spec.Connectors,
			AllowedTools:       agent.Spec.AllowedTools,
			DisallowedTools:    agent.Spec.DisallowedTools,
			Instructions:       agent.Spec.Instructions,
			SystemPrompt:       agent.Spec.SystemPrompt,
			CreatedAt:          agent.CreationTimestamp.UTC().Format(time.RFC3339),
			TotalTokens:        agent.Status.TotalTokens,
			ModelContextWindow: agent.Status.ModelContextWindow,
			Priority:           agent.Spec.Priority,
			PodSpec:            agent.Spec.PodSpec,
			Storage:            agent.Spec.Storage,
			Labels:             agent.Spec.Labels,
			CompletionTime:     formatTime(agent.Status.CompletionTime),
		}, agent))
	}
}

// deleteAgent deletes an agent and cleans up all its resources.
// @ID deleteAgent
// @Summary Delete agent
// @Description Deletes the agent CR, pod, PVC, secrets, and Redis event stream.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Param recreatePod query bool false "If the agent is a squad member, also delete the squad pod so it is rebuilt without this member"
// @Success 200 {object} map[string]string "Agent deleted"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name} [delete]
func deleteAgent(k8s *K8sClient, worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		// Optional flag: when the agent is a squad member, also delete the squad pod
		// after removing the agent. The squad reconciler rebuilds the pod fresh on
		// the next loop so any stale ephemeral container from the removed member is gone.
		recreatePod := c.Query("recreatePod") == "true"

		// Look up squad membership BEFORE deleting the agent — once the CR is gone
		// the FindSquadForAgent lookup would still work via the squad's spec, but
		// resolving it up front keeps the squad-pod delete logic simple.
		var squadName string
		if recreatePod {
			if squad, err := k8s.FindSquadForAgent(c.Request.Context(), ns, name); err == nil && squad != nil {
				squadName = squad.Name
			}
		}

		if err := k8s.DeleteAgent(c.Request.Context(), ns, name); err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
				return
			}
			agentActionsTotal.WithLabelValues("delete", "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete agent: " + err.Error()})
			return
		}
		// Clean up the agent's Redis event stream
		if err := DeleteAgentStream(c.Request.Context(), worker.Rdb, name, worker.StreamPrefix); err != nil {
			Logger.Warnw("failed to delete event stream", "agent_name", name, "error", err)
		}

		// Best-effort: delete the squad pod so the reconciler rebuilds it.
		if squadName != "" {
			if err := k8s.DeletePod(c.Request.Context(), ns, squadName+"-pod"); err != nil && !errors.IsNotFound(err) {
				Logger.Warnw("failed to delete squad pod for recreate", "squad_name", squadName, "error", err)
			} else {
				Logger.Infow("deleted squad pod for recreate", "namespace", ns, "squad_name", squadName)
			}
		}

		Logger.Infow("deleted agent", "namespace", ns, "agent_name", name)
		agentActionsTotal.WithLabelValues("delete", "success").Inc()
		c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
	}
}

type CompactAgentRequest struct {
	Instructions string `json:"instructions,omitempty"`
}

// compactAgentTask manually triggers conversation compaction on an active agent task.
// @ID compactAgentTask
// @Summary Compact agent conversation
// @Description Triggers manual conversation compaction. Older turns are summarized to free context space. Requires an active task — returns 409 if the agent is idle. Optional `instructions` field passes custom guidance to the compactor.
// @Tags agents
// @Accept json
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Param request body CompactAgentRequest false "Optional compaction instructions"
// @Success 200 {object} map[string]string "Compaction triggered"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 409 {object} map[string]string "Agent has no running pod or no active task"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name}/compact [post]
func compactAgentTask(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		var req CompactAgentRequest
		// Body is optional; ignore bind errors so an empty POST still works.
		_ = c.ShouldBindJSON(&req)

		agent, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if agent.Status.PodName == "" {
			c.JSON(http.StatusConflict, gin.H{"error": "agent has no running pod"})
			return
		}

		if err := k8s.CompactAgentTask(c.Request.Context(), ns, agent.Status.PodName, agent.Name, req.Instructions); err != nil {
			agentActionsTotal.WithLabelValues("compact", "error").Inc()
			// The agent returns 409 if no active task; surface that to the caller as 409.
			if strings.Contains(err.Error(), "status 409") || strings.Contains(err.Error(), "No task is currently running") {
				c.JSON(http.StatusConflict, gin.H{"error": "no active task to compact; manual compaction only works while the agent is running a task"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compact: " + err.Error()})
			return
		}

		Logger.Infow("triggered manual compaction on agent", "namespace", ns, "agent_name", name)
		agentActionsTotal.WithLabelValues("compact", "success").Inc()
		c.JSON(http.StatusOK, gin.H{"status": "compacting", "name": name})
	}
}

// cancelAgentTask cancels the running task on an agent.
// @ID cancelAgentTask
// @Summary Cancel agent task
// @Description Gracefully cancels the currently running task. The agent pod stays alive for future tasks.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} map[string]string "Task cancelling"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 409 {object} map[string]string "Agent has no running pod"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name}/cancel [post]
func cancelAgentTask(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)

		agent, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if agent.Status.PodName == "" {
			c.JSON(http.StatusConflict, gin.H{"error": "agent has no running pod"})
			return
		}

		if err := k8s.CancelAgentTask(c.Request.Context(), ns, agent.Status.PodName, agent.Name); err != nil {
			agentActionsTotal.WithLabelValues("cancel", "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel: " + err.Error()})
			return
		}

		Logger.Infow("cancelled task on agent", "namespace", ns, "agent_name", name)
		agentActionsTotal.WithLabelValues("cancel", "success").Inc()
		c.JSON(http.StatusOK, gin.H{"status": "cancelling", "name": name})
	}
}

// wakeAgent wakes a sleeping agent WITHOUT running any task — unlike POST /agents,
// which always carries a task (even an empty-string one counts as "run this").
// Use it to bring an agent (or a sleeping squad member) back up idle, ready for
// commands over its own channel (e.g. a manager delegating later) without kicking
// off work immediately.
//
// Calling it on an agent that isn't Sleeping is a no-op: it returns 200 with the
// agent's current state rather than an error, so callers can wake unconditionally.
// @ID wakeAgent
// @Summary Wake an agent idle
// @Description Wakes a sleeping agent without running any task, leaving its instructions untouched. A no-op (200, current state) if the agent is not sleeping.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} AgentResponse "Agent woken (or already awake)"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 409 {object} map[string]string "Agent is archived"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name}/wake [post]
func wakeAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())

		agent, err := k8s.IdleWakeAgent(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
				return
			}
			agentActionsTotal.WithLabelValues("idle_wake", "error").Inc()
			if err == ErrAgentArchived {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to wake agent: " + err.Error()})
			return
		}

		Logger.Infow("idle-woke agent", "namespace", ns, "agent_name", name)
		agentActionsTotal.WithLabelValues("idle_wake", "success").Inc()
		c.JSON(http.StatusOK, withAgentTTL(c.Request.Context(), k8s, AgentResponse{
			Name:               agent.Name,
			Namespace:          agent.Namespace,
			Model:              agent.Spec.Model,
			Effort:             agent.Spec.Effort,
			Status:             string(agent.Status.Phase),
			TaskStatus:         string(agent.Status.TaskStatus),
			LastTaskMessage:    agent.Status.LastTaskMessage,
			Lifecycle:          string(agent.Spec.Lifecycle),
			LastTaskCostUSD:    agent.Status.LastTaskCostUSD,
			TotalCostUSD:       agent.Status.TotalCostUSD,
			TotalTokens:        agent.Status.TotalTokens,
			ModelContextWindow: agent.Status.ModelContextWindow,
			Secrets:            collectSecretKeys(*c, k8s, ns, agent.Spec.Secrets),
			Memories:           agent.Spec.Memories,
			Skills:             mergeDefaultSkills(agent.Spec.Skills, defaultSkills),
			Connectors:         agent.Spec.Connectors,
			AllowedTools:       agent.Spec.AllowedTools,
			DisallowedTools:    agent.Spec.DisallowedTools,
			Instructions:       agent.Spec.Instructions,
			SystemPrompt:       agent.Spec.SystemPrompt,
			CreatedAt:          agent.CreationTimestamp.UTC().Format(time.RFC3339),
			Priority:           agent.Spec.Priority,
			QueuePosition:      agent.Status.QueuePosition,
			QueueReason:        agent.Status.QueueReason,
			PodSpec:            agent.Spec.PodSpec,
			Storage:            agent.Spec.Storage,
			Squad:              agent.Status.Squad,
			SquadName:          resolveSquadName(c.Request.Context(), k8s, agent),
			Labels:             agent.Spec.Labels,
			CompletionTime:     formatTime(agent.Status.CompletionTime),
		}, agent))
	}
}

// sleepAgent puts a running agent to sleep right now — the real counterpart to
// wakeAgent. Unlike PATCH /agents/{name} with lifecycle=Sleep (which sets a
// policy the operator applies once a task completes, suppressed by a prior
// idle wake or a sleepTTL), this always takes effect immediately and never
// touches spec.lifecycle.
//
// Already Sleeping is a no-op: it returns 200 with the agent's current state,
// matching wakeAgent's idempotency.
// @ID sleepAgent
// @Summary Sleep an agent now
// @Description Sleeps an agent immediately (pod deleted, workspace/PVC kept), regardless of sleepTTL or a prior idle wake, without changing spec.lifecycle. A no-op (200, current state) if already sleeping.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} AgentResponse "Agent sleeping (or already asleep)"
// @Failure 400 {object} map[string]string "Ephemeral-workspace agents can't sleep"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 409 {object} map[string]string "Agent is running a task, or is archived"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name}/sleep [post]
func sleepAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())

		agent, err := k8s.SleepAgentNow(c.Request.Context(), ns, name)
		if err != nil {
			switch {
			case errors.IsNotFound(err):
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			case err == ErrAgentBusy:
				agentActionsTotal.WithLabelValues("sleep_now", "error").Inc()
				c.JSON(http.StatusConflict, gin.H{"error": "agent is running a task; cancel it first"})
			case err == ErrAgentArchived:
				agentActionsTotal.WithLabelValues("sleep_now", "error").Inc()
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			case err == ErrAgentEphemeral:
				agentActionsTotal.WithLabelValues("sleep_now", "error").Inc()
				c.JSON(http.StatusBadRequest, gin.H{"error": ErrAgentEphemeral.Error()})
			default:
				agentActionsTotal.WithLabelValues("sleep_now", "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sleep agent: " + err.Error()})
			}
			return
		}

		Logger.Infow("sleep-requested agent", "namespace", ns, "agent_name", name)
		agentActionsTotal.WithLabelValues("sleep_now", "success").Inc()
		c.JSON(http.StatusOK, withAgentTTL(c.Request.Context(), k8s, AgentResponse{
			Name:               agent.Name,
			Namespace:          agent.Namespace,
			Model:              agent.Spec.Model,
			Effort:             agent.Spec.Effort,
			Status:             string(agent.Status.Phase),
			TaskStatus:         string(agent.Status.TaskStatus),
			LastTaskMessage:    agent.Status.LastTaskMessage,
			Lifecycle:          string(agent.Spec.Lifecycle),
			LastTaskCostUSD:    agent.Status.LastTaskCostUSD,
			TotalCostUSD:       agent.Status.TotalCostUSD,
			TotalTokens:        agent.Status.TotalTokens,
			ModelContextWindow: agent.Status.ModelContextWindow,
			Secrets:            collectSecretKeys(*c, k8s, ns, agent.Spec.Secrets),
			Memories:           agent.Spec.Memories,
			Skills:             mergeDefaultSkills(agent.Spec.Skills, defaultSkills),
			Connectors:         agent.Spec.Connectors,
			AllowedTools:       agent.Spec.AllowedTools,
			DisallowedTools:    agent.Spec.DisallowedTools,
			Instructions:       agent.Spec.Instructions,
			SystemPrompt:       agent.Spec.SystemPrompt,
			CreatedAt:          agent.CreationTimestamp.UTC().Format(time.RFC3339),
			Priority:           agent.Spec.Priority,
			QueuePosition:      agent.Status.QueuePosition,
			QueueReason:        agent.Status.QueueReason,
			PodSpec:            agent.Spec.PodSpec,
			Storage:            agent.Spec.Storage,
			Squad:              agent.Status.Squad,
			SquadName:          resolveSquadName(c.Request.Context(), k8s, agent),
			Labels:             agent.Spec.Labels,
			CompletionTime:     formatTime(agent.Status.CompletionTime),
		}, agent))
	}
}

// getAgent returns details for a single agent.
// @ID getAgent
// @Summary Get agent details
// @Description Returns the current status and metadata for a single agent.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Success 200 {object} AgentResponse "Agent details"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name} [get]
func getAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())
		agent, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err != nil {
			if errors.IsNotFound(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, withAgentTTL(c.Request.Context(), k8s, AgentResponse{
			Name:               agent.Name,
			Namespace:          agent.Namespace,
			Model:              agent.Spec.Model,
			Effort:             agent.Spec.Effort,
			Status:             string(agent.Status.Phase),
			TaskStatus:         string(agent.Status.TaskStatus),
			LastTaskMessage:    agent.Status.LastTaskMessage,
			Lifecycle:          string(agent.Spec.Lifecycle),
			LastTaskCostUSD:    agent.Status.LastTaskCostUSD,
			TotalCostUSD:       agent.Status.TotalCostUSD,
			TotalTokens:        agent.Status.TotalTokens,
			ModelContextWindow: agent.Status.ModelContextWindow,
			Secrets:            agent.Spec.Secrets,
			Memories:           agent.Spec.Memories,
			Skills:             mergeDefaultSkills(agent.Spec.Skills, defaultSkills),
			Connectors:         agent.Spec.Connectors,
			AllowedTools:       agent.Spec.AllowedTools,
			DisallowedTools:    agent.Spec.DisallowedTools,
			Instructions:       agent.Spec.Instructions,
			SystemPrompt:       agent.Spec.SystemPrompt,
			CreatedAt:          agent.CreationTimestamp.UTC().Format(time.RFC3339),
			Priority:           agent.Spec.Priority,
			QueuePosition:      agent.Status.QueuePosition,
			QueueReason:        agent.Status.QueueReason,
			PodSpec:            agent.Spec.PodSpec,
			Storage:            agent.Spec.Storage,
			Squad:              agent.Status.Squad,
			SquadName:          resolveSquadName(c.Request.Context(), k8s, agent),
			Labels:             agent.Spec.Labels,
			CompletionTime:     formatTime(agent.Status.CompletionTime),
		}, agent))
	}
}

// shouldBackfillEvents reports whether an empty Redis result for
// getAgentEvents should trigger the JSONL session backfill. Only uncursored
// requests qualify: a cursored poll (after) or seek (before/around) that
// comes back empty means "no events there", not "Redis was wiped" — and
// backfilling anyway would re-RPush the whole session (no dedupe) and
// return stale events, ignoring the cursor the caller asked for.
func shouldBackfillEvents(before, after, around string) bool {
	return before == "" && after == "" && around == ""
}

// getAgentEvents returns the event history for an agent from Redis.
// @ID getAgentEvents
// @Summary Get agent events
// @Description Returns recent events from the agent's Redis stream in chronological order.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Param limit query int false "Max events to return (1-200)" default(50)
// @Param before query string false "RFC-3339 cursor: return events strictly before this timestamp (pagination)"
// @Param after query string false "RFC-3339 cursor: return events strictly after this timestamp (polling for new events)"
// @Param around query string false "RFC-3339 timestamp to center results on"
// @Success 200 {object} map[string]interface{} "Agent events"
// @Failure 400 {object} map[string]string "Invalid limit parameter"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name}/events [get]
func getAgentEvents(worker *RedisWorker, k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
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

		before := c.Query("before") // RFC-3339 cursor for pagination
		after := c.Query("after")   // RFC-3339 cursor — return events strictly after this timestamp
		around := c.Query("around") // RFC-3339 timestamp to center results on

		var events []AgentEvent
		var err error
		if around != "" {
			events, err = GetAgentEventsAround(c.Request.Context(), worker.Rdb, name, limit, around, worker.StreamPrefix)
		} else if after != "" {
			events, err = GetAgentEventsAfter(c.Request.Context(), worker.Rdb, name, limit, after, worker.StreamPrefix)
		} else {
			events, err = GetAgentEventsBefore(c.Request.Context(), worker.Rdb, name, limit, before, worker.StreamPrefix)
		}
		if err != nil {
			events = nil
		}

		// Hybrid: if Redis is empty (wiped) and agent has a session, backfill from JSONL.
		if len(events) == 0 && shouldBackfillEvents(before, after, around) {
			ns := resolveNamespace(c, k8s)
			agent, getErr := k8s.GetAgent(c.Request.Context(), ns, name)
			if getErr == nil && agent.Status.PodName != "" && agent.Status.SessionID != "" {
				// Fetch ALL events (no limit) so we backfill the full history.
				var totalCost float64
				if agent.Status.TotalCostUSD != "" {
					fmt.Sscanf(agent.Status.TotalCostUSD, "%f", &totalCost)
				}
				allSessionEvents := fetchSessionEvents(c.Request.Context(), k8s, ns, agent.Status.PodName, agent.Status.SessionID, name, 0, agent.Spec.Model, totalCost)
				if len(allSessionEvents) > 0 {
					// Backfill synchronously so pagination works on the next request.
					backfillRedisHistory(worker.Rdb, name, allSessionEvents)
					// Serve only the requested limit.
					if int64(len(allSessionEvents)) > limit {
						events = allSessionEvents[int64(len(allSessionEvents))-limit:]
					} else {
						events = allSessionEvents
					}
				}
			}
		}

		if events == nil {
			events = []AgentEvent{}
		}
		c.JSON(http.StatusOK, gin.H{"agent": name, "events": events})
	}
}

// listAgents returns all agents in a namespace.
//
// Sorting/pagination (limit/offset/sort) and the free-text `q` filter all
// operate over the FULL namespace-scoped list fetched from K8s in one call —
// see filterSortPaginateAgents. `label` filtering is intentionally NOT pushed
// down to the K8s list call (unlike before): facets must reflect every label
// combination in the namespace scope regardless of the label/status/q filters
// in effect, so the full list is always built and then filtered in-memory.
// @ID listAgents
// @Summary List agents
// @Description Returns agents with their current status in the specified namespace. Without `limit`, returns every matching agent (backward compatible). With `limit`, returns one page plus `total`/`nextOffset`/`facets` for infinite-scroll UIs.
// @Tags agents
// @Produce json
// @Param namespace query string false "Kubernetes namespace"
// @Param status query string false "Filter by agent phase (case-insensitive), e.g. Running, Sleeping"
// @Param label query []string false "Label filter key=value; repeat for multiple (AND)" collectionFormat(multi)
// @Param q query string false "Case-insensitive substring match against agent name"
// @Param sort query string false "Sort key: created, name, status, model, lastActive, cost; prefix with - for descending. Default -created" default(-created)
// @Param limit query int false "Max agents to return (1-200). Omit to return everything (legacy behavior)"
// @Param offset query int false "Number of matching agents to skip (used with limit). Default 0"
// @Success 200 {object} AgentListResponse "List of agents"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents [get]
func listAgents(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		ns := c.Query("namespace") // empty = all namespaces
		statusFilter := c.Query("status")
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())

		// Parse repeated ?label=k=v query params.
		rawLabels := c.QueryArray("label")
		labelFilter := map[string]string{}
		for _, p := range rawLabels {
			eq := strings.Index(p, "=")
			if eq <= 0 || eq == len(p)-1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid label filter %q (expected key=value)", p)})
				return
			}
			k := p[:eq]
			v := p[eq+1:]
			if errs := validation.IsQualifiedName(k); len(errs) > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid label key %q: %s", k, strings.Join(errs, ", "))})
				return
			}
			if errs := validation.IsValidLabelValue(v); len(errs) > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid label value for %q: %s", k, strings.Join(errs, ", "))})
				return
			}
			labelFilter[k] = v
		}

		limit, err := parseAgentListLimit(c.Query("limit"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		offset, err := parseAgentListOffset(c.Query("offset"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sortParam := c.Query("sort")
		if _, _, err := parseAgentSort(sortParam); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Fetch every agent in the namespace scope — no label filter pushed down
		// to K8s, since facets must be computed over this same unfiltered set.
		agents, err := k8s.ListAgents(c.Request.Context(), ns, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agents: " + err.Error()})
			return
		}

		squadByAgent := buildSquadNameMap(c.Request.Context(), k8s, ns)

		all := make([]AgentResponse, 0, len(agents))
		metaLabels := make(map[string]map[string]string, len(agents))
		for _, a := range agents {
			metaLabels[a.Namespace+"/"+a.Name] = a.Labels
			all = append(all, withAgentTTL(c.Request.Context(), k8s, AgentResponse{
				Name:               a.Name,
				Namespace:          a.Namespace,
				Model:              a.Spec.Model,
				Effort:             a.Spec.Effort,
				Status:             string(a.Status.Phase),
				TaskStatus:         string(a.Status.TaskStatus),
				LastTaskMessage:    a.Status.LastTaskMessage,
				Lifecycle:          string(a.Spec.Lifecycle),
				LastTaskCostUSD:    a.Status.LastTaskCostUSD,
				TotalCostUSD:       a.Status.TotalCostUSD,
				TotalTokens:        a.Status.TotalTokens,
				ModelContextWindow: a.Status.ModelContextWindow,
				Secrets:            collectSecretKeys(*c, k8s, ns, a.Spec.Secrets),
				Memories:           a.Spec.Memories,
				Skills:             mergeDefaultSkills(a.Spec.Skills, defaultSkills),
				Connectors:         a.Spec.Connectors,
				AllowedTools:       a.Spec.AllowedTools,
				DisallowedTools:    a.Spec.DisallowedTools,
				Instructions:       a.Spec.Instructions,
				SystemPrompt:       a.Spec.SystemPrompt,
				CreatedAt:          a.CreationTimestamp.UTC().Format(time.RFC3339),
				Priority:           a.Spec.Priority,
				QueuePosition:      a.Status.QueuePosition,
				QueueReason:        a.Status.QueueReason,
				PodSpec:            a.Spec.PodSpec,
				Storage:            a.Spec.Storage,
				Squad:              a.Status.Squad,
				SquadName:          squadByAgent[a.Namespace+"/"+a.Name],
				Labels:             a.Spec.Labels,
				CompletionTime:     formatTime(a.Status.CompletionTime),
			}, &a))
		}

		page, total, nextOffset, facets, err := filterSortPaginateAgents(all, AgentListQuery{
			Status:       statusFilter,
			Labels:       labelFilter,
			FilterLabels: metaLabels,
			Q:            c.Query("q"),
			Sort:         sortParam,
			Limit:        limit,
			Offset:       offset,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, AgentListResponse{
			Agents:     page,
			Total:      total,
			NextOffset: nextOffset,
			Facets:     facets,
		})
	}
}

// patchAgent updates fields on an existing agent.
// @ID patchAgent
// @Summary Patch agent
// @Description Updates model, lifecycle, instructions, secretRefs, memories, skills, or connectors on an existing agent.
// @Tags agents
// @Accept json
// @Produce json
// @Param name path string true "Agent name"
// @Param namespace query string false "Kubernetes namespace"
// @Param request body PatchAgentRequest true "Fields to update"
// @Success 200 {object} AgentResponse "Updated agent"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal error"
// @Failure 409 {object} map[string]string "Agent is archived (only labels and deleteTTL can change)"
// @Router /agents/{name} [patch]
func patchAgent(k8s *K8sClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ns := resolveNamespace(c, k8s)
		defaultSkills, _ := k8s.ListDefaultSkillNames(c.Request.Context())

		var req PatchAgentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		if req.Model == nil && req.Effort == nil && req.Lifecycle == nil && req.Instructions == nil && req.TemplateRef == nil && req.SecretRefs == nil && req.Memories == nil && req.Skills == nil && req.Connectors == nil && req.AllowedTools == nil && req.DisallowedTools == nil && req.SystemPrompt == nil && req.Priority == nil && req.PodSpec == nil && req.Storage == nil && req.SleepTTL == nil && req.DeleteTTL == nil && req.TaskTimeout == nil && req.BudgetUSD == nil && req.TaskBudgetUSD == nil && len(req.Labels) == 0 && len(req.RemoveLabels) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}
		// On Bedrock, reject a friendly model override before it reaches the SDK.
		if req.Model != nil {
			if err := validateBedrockModel(*req.Model, false); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		if req.Effort != nil {
			if err := validateEffort(*req.Effort); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		if target, getErr := k8s.GetAgent(c.Request.Context(), ns, name); getErr == nil && agentArchived(target) {
			if err := validateArchivedPatch(req); err != nil {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
		}
		sleepTTL, err := parseTTLUpdate("sleepTTL", req.SleepTTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		deleteTTL, err := parseTTLUpdate("deleteTTL", req.DeleteTTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		taskTimeout, err := parseTTLUpdate("taskTimeout", req.TaskTimeout)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		budgetUSD, err := parseBudgetUpdate("budgetUSD", req.BudgetUSD)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		taskBudgetUSD, err := parseBudgetUpdate("taskBudgetUSD", req.TaskBudgetUSD)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// storage.ephemeral is immutable, and can never combine with Sleep lifecycle or
		// a sleepTTL (sleeping deletes the pod, which would wipe an ephemeral
		// workspace). Checked against the agent as it will be once this patch applies:
		// its own current values for anything the patch doesn't touch.
		if req.Storage != nil || req.Lifecycle != nil || sleepTTL.Set {
			existingAgent, getErr := k8s.GetAgent(c.Request.Context(), ns, name)
			if getErr != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "agent not found: " + getErr.Error()})
				return
			}
			effectiveEphemeral := k8s.ResolveEffectiveEphemeral(c.Request.Context(), ns, existingAgent.Spec.AgentConfigSpec)
			if err := validateEphemeralImmutable(effectiveEphemeral, req.Storage); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			lifecycleToCheck := string(existingAgent.Spec.Lifecycle)
			if req.Lifecycle != nil {
				lifecycleToCheck = *req.Lifecycle
			}
			sleepTTLToCheck := existingAgent.Spec.SleepTTL
			if sleepTTL.Set {
				sleepTTLToCheck = sleepTTL.Value
			}
			if err := validateEphemeralStorage(effectiveEphemeral, lifecycleToCheck, sleepTTLToCheck); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		// Reject a disabled connector only if it's being newly attached — a connector
		// already on the agent (now disabled) must survive a re-sent list or an
		// unrelated-field edit. Fetched and checked before any mutation below.
		if req.Connectors != nil {
			existingForGuard, getErr := k8s.GetAgent(c.Request.Context(), ns, name)
			if getErr == nil {
				if err := k8s.ValidateConnectorsAttachable(c.Request.Context(), ns, *req.Connectors, existingForGuard.Spec.Connectors); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}
		}

		var nonFatalErrors []string

		// Determine action label for metrics: "sleep" if lifecycle → Sleep, else "patch".
		patchAction := "patch"
		if req.Lifecycle != nil && *req.Lifecycle == "Sleep" {
			patchAction = "sleep"
		}

		// 1. Patch CR spec first — this is the source of truth.
		if err := k8s.PatchAgentSpec(c.Request.Context(), ns, name, req.Model, req.Effort, req.Lifecycle, req.Instructions, req.TemplateRef, req.SystemPrompt, req.Priority, sleepTTL, deleteTTL, taskTimeout, budgetUSD, taskBudgetUSD); err != nil {
			agentActionsTotal.WithLabelValues(patchAction, "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch agent: " + err.Error()})
			return
		}

		// 1-sleep-squad. Sleep on a squad member: the agent controller skips squad-managed
		// agents, so Lifecycle=Sleep alone wouldn't take effect. Cancel any in-flight task
		// (best effort) and set Status.Phase=Sleeping directly. The squad controller then
		// deletes the squad pod once *all* members are Sleeping.
		if patchAction == "sleep" {
			if cur, getErr := k8s.GetAgent(c.Request.Context(), ns, name); getErr == nil && cur.Status.Squad {
				if cur.Status.PodName != "" {
					if cancelErr := k8s.CancelAgentTask(c.Request.Context(), ns, cur.Status.PodName, name); cancelErr != nil {
						Logger.Warnw("squad-sleep cancel failed (continuing)", "agent_name", name, "error", cancelErr)
					}
				}
				if patchErr := k8s.PatchAgentPhase(c.Request.Context(), ns, name, komputerv1alpha1.AgentPhaseSleeping, "Sleeping — set by user"); patchErr != nil {
					agentActionsTotal.WithLabelValues(patchAction, "error").Inc()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set sleeping phase: " + patchErr.Error()})
					return
				}
			}
		}

		// 1a-labels. Merge user labels additively (existing keys overwritten) and/or
		// remove the keys in RemoveLabels, in a single patch. A key in both Labels
		// and RemoveLabels is rejected as ambiguous.
		if len(req.Labels) > 0 || len(req.RemoveLabels) > 0 {
			if err := validateUserLabels(req.Labels); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := validateRemoveLabelKeys(req.RemoveLabels); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			for _, rk := range req.RemoveLabels {
				if _, ok := req.Labels[rk]; ok {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("label key %q cannot be in both labels and removeLabels", rk)})
					return
				}
			}
			if err := k8s.UpdateAgentLabels(c.Request.Context(), ns, name, req.Labels, req.RemoveLabels); err != nil {
				agentActionsTotal.WithLabelValues(patchAction, "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch agent labels: " + err.Error()})
				return
			}
		}

		// 1a-override. Patch podSpec / storage overrides if provided.
		if req.PodSpec != nil || req.Storage != nil {
			if err := k8s.PatchAgentOverrides(c.Request.Context(), ns, name, req.PodSpec, req.Storage); err != nil {
				agentActionsTotal.WithLabelValues(patchAction, "error").Inc()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to patch agent overrides: " + err.Error()})
				return
			}
		}

		// 1a. If model changed and pod is running, fetch context window via the agent (it has the API key).
		// This is handled below in step 2 where we forward config to the pod and read the response.

		// 1b. Replace secret refs list if provided.
		if req.SecretRefs != nil {
			k8s.PatchAgentSecretsList(c.Request.Context(), ns, name, *req.SecretRefs)
		}

		// 1d. Update memories if provided.
		if req.Memories != nil {
			k8s.PatchAgentMemoriesList(c.Request.Context(), ns, name, *req.Memories)
		}

		// 1d. Update skills if provided.
		if req.AllowedTools != nil || req.DisallowedTools != nil {
			// Takes effect on the agent's next start — the tool policy is applied
			// when the SDK session is constructed, not at runtime.
			if err := k8s.PatchAgentToolPolicy(c.Request.Context(), ns, name, req.AllowedTools, req.DisallowedTools); err != nil {
				Logger.Warnw("failed to patch agent tool policy", "namespace", ns, "agent_name", name, "error", err)
				nonFatalErrors = append(nonFatalErrors, fmt.Sprintf("failed to update tool policy: %v", err))
			}
		}

		if req.Skills != nil {
			k8s.PatchAgentSkillsList(c.Request.Context(), ns, name, *req.Skills)
			skillFiles, _ := k8s.ResolveSkillFiles(c.Request.Context(), ns, *req.Skills)
			if len(skillFiles) > 0 {
				if configJSON, err := json.Marshal(map[string]interface{}{"skills": skillFiles}); err == nil {
					agentForSkills, getErr := k8s.GetAgent(c.Request.Context(), ns, name)
					if getErr == nil && agentForSkills.Status.PodName != "" && agentForSkills.Status.Phase == "Running" {
						if applyErr := k8s.ApplyAgentConfig(c.Request.Context(), ns, agentForSkills.Status.PodName, agentForSkills.Name, string(configJSON)); applyErr != nil {
							Logger.Errorw("skills config apply to pod failed", "agent_name", name, "error", applyErr)
							nonFatalErrors = append(nonFatalErrors, fmt.Sprintf("skills sync to running pod failed: %v", applyErr))
						}
					}
				}
			}
		}

		// 1e. Update connectors if provided.
		if req.Connectors != nil {
			k8s.PatchAgentConnectorsList(c.Request.Context(), ns, name, *req.Connectors)
		}

		// 2. If pod is running, forward config to the agent so it takes effect immediately.
		// If the model changed, read context_window from the response and patch the CR.
		var freshContextWindow int64
		agent, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err == nil && agent.Status.PodName != "" && agent.Status.Phase == "Running" {
			type agentConfigPayload struct {
				Model        *string                 `json:"model,omitempty"`
				Effort       *string                 `json:"effort,omitempty"`
				Lifecycle    *string                 `json:"lifecycle,omitempty"`
				Instructions *string                 `json:"instructions,omitempty"`
				TemplateRef  *string                 `json:"templateRef,omitempty"`
				Secrets      map[string]string       `json:"secrets,omitempty"`
				McpServers   *map[string]interface{} `json:"mcp_servers,omitempty"`
			}
			payload := agentConfigPayload{
				Model:        req.Model,
				Effort:       req.Effort,
				Lifecycle:    req.Lifecycle,
				Instructions: req.Instructions,
				TemplateRef:  req.TemplateRef,
			}
			// If secretRefs changed, resolve all secrets so the agent can set/remove SECRET_* env vars.
			if req.SecretRefs != nil {
				payload.Secrets = k8s.ResolveSecretEnvVars(c.Request.Context(), ns, *req.SecretRefs)
			}
			// If connectors changed, resolve MCP configs so the agent updates KOMPUTER_MCP_SERVERS.
			if req.Connectors != nil {
				mcpServers := k8s.ResolveConnectorMCPConfigs(c.Request.Context(), ns, *req.Connectors)
				payload.McpServers = &mcpServers
			}
			// Skip if payload has no fields the agent's /config endpoint understands
			// (e.g. only skills/memories changed — those are applied separately above).
			hasPayloadFields := payload.Model != nil || payload.Effort != nil || payload.Lifecycle != nil ||
				payload.Instructions != nil || payload.TemplateRef != nil ||
				payload.Secrets != nil || payload.McpServers != nil
			configPayload, _ := json.Marshal(payload)
			if hasPayloadFields {
				if req.Model != nil && *req.Model != "" {
					// Read response to capture context_window
					if cw := k8s.ApplyAgentConfigGetContextWindow(c.Request.Context(), ns, agent.Status.PodName, agent.Name, string(configPayload)); cw > 0 {
						freshContextWindow = cw
						if patchErr := k8s.PatchAgentContextWindow(c.Request.Context(), ns, name, cw); patchErr != nil {
							Logger.Warnw("failed to patch context window", "agent_name", name, "error", patchErr)
						}
					}
				} else {
					if applyErr := k8s.ApplyAgentConfig(c.Request.Context(), ns, agent.Status.PodName, agent.Name, string(configPayload)); applyErr != nil {
						Logger.Errorw("CR patched but config apply to pod failed", "agent_name", name, "error", applyErr)
						nonFatalErrors = append(nonFatalErrors, fmt.Sprintf("config sync to running pod failed: %v", applyErr))
					}
				}
			}
		}

		// 3. Return updated agent.
		updated, err := k8s.GetAgent(c.Request.Context(), ns, name)
		if err != nil {
			agentActionsTotal.WithLabelValues(patchAction, "error").Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "agent patched but failed to read back: " + err.Error()})
			return
		}
		// Use freshContextWindow if we just fetched it (avoids race with CR status patch propagation).
		modelContextWindow := updated.Status.ModelContextWindow
		if freshContextWindow > 0 {
			modelContextWindow = freshContextWindow
		}
		agentActionsTotal.WithLabelValues(patchAction, "success").Inc()
		c.JSON(http.StatusOK, withAgentTTL(c.Request.Context(), k8s, AgentResponse{
			Name:               updated.Name,
			Namespace:          updated.Namespace,
			Model:              updated.Spec.Model,
			Effort:             updated.Spec.Effort,
			Status:             string(updated.Status.Phase),
			TaskStatus:         string(updated.Status.TaskStatus),
			LastTaskMessage:    updated.Status.LastTaskMessage,
			Lifecycle:          string(updated.Spec.Lifecycle),
			LastTaskCostUSD:    updated.Status.LastTaskCostUSD,
			TotalCostUSD:       updated.Status.TotalCostUSD,
			TotalTokens:        updated.Status.TotalTokens,
			ModelContextWindow: modelContextWindow,
			Secrets:            updated.Spec.Secrets,
			Memories:           updated.Spec.Memories,
			Skills:             mergeDefaultSkills(updated.Spec.Skills, defaultSkills),
			Instructions:       updated.Spec.Instructions,
			SystemPrompt:       updated.Spec.SystemPrompt,
			Connectors:         updated.Spec.Connectors,
			AllowedTools:       updated.Spec.AllowedTools,
			DisallowedTools:    updated.Spec.DisallowedTools,
			CreatedAt:          updated.CreationTimestamp.UTC().Format(time.RFC3339),
			Priority:           updated.Spec.Priority,
			QueuePosition:      updated.Status.QueuePosition,
			QueueReason:        updated.Status.QueueReason,
			PodSpec:            updated.Spec.PodSpec,
			Storage:            updated.Spec.Storage,
			Squad:              updated.Status.Squad,
			SquadName:          resolveSquadName(c.Request.Context(), k8s, updated),
			Errors:             nonFatalErrors,
			Labels:             updated.Spec.Labels,
			CompletionTime:     formatTime(updated.Status.CompletionTime),
		}, updated))
	}
}
