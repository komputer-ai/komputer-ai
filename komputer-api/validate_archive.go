package main

import (
	"fmt"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// ErrAgentArchived is returned for any request that would run, wake or sleep an
// archived agent. Archiving deletes the pod and workspace and keeps the CR only
// as a record, so the agent can never run again.
var ErrAgentArchived = fmt.Errorf("agent is archived: its pod and workspace were removed and it can't run again — create a new agent, or delete this one to reuse its name")

func agentArchived(agent *komputerv1alpha1.KomputerAgent) bool {
	return agent.Status.Phase == komputerv1alpha1.AgentPhaseArchived
}

// validateArchivedPatch allows only labels and deleteTTL on an archived agent —
// the fields that still mean something on a record (organizing it, and how long
// to retain it). Everything else configures a run that will never happen.
func validateArchivedPatch(req PatchAgentRequest) error {
	if req.Model != nil || req.Effort != nil || req.Lifecycle != nil || req.Instructions != nil ||
		req.TemplateRef != nil || req.SecretRefs != nil || req.Memories != nil || req.Skills != nil ||
		req.Connectors != nil || req.AllowedTools != nil || req.DisallowedTools != nil ||
		req.SystemPrompt != nil || req.Priority != nil || req.PodSpec != nil || req.Storage != nil ||
		req.SleepTTL != nil || req.TaskTimeout != nil || req.BudgetUSD != nil || req.TaskBudgetUSD != nil {
		return fmt.Errorf("agent is archived: only labels and deleteTTL can be changed")
	}
	return nil
}

// validateScheduleLifecycle rejects Lifecycle=Archive for a schedule's agent:
// every run reuses the same agent, and an archived agent never runs again.
func validateScheduleLifecycle(lifecycle komputerv1alpha1.AgentLifecycle) error {
	if lifecycle == komputerv1alpha1.AgentLifecycleArchive {
		return fmt.Errorf("lifecycle: Archive isn't supported for schedules — every run reuses one agent, and an archived agent can't run again. Use Sleep or AutoDelete")
	}
	return nil
}
