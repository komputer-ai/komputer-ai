package main

import (
	"context"
	"fmt"
	"reflect"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// ErrAgentArchived is returned for any request that would run, wake or sleep an
// archived agent. Archiving deletes the pod and workspace and keeps the CR only
// as a record, so the agent can never run again.
var ErrAgentArchived = fmt.Errorf("agent is archived: its pod and workspace were removed and it can't run again — create a new agent, or delete this one to reuse its name")

func agentArchived(agent *komputerv1alpha1.KomputerAgent) bool {
	return agent.Status.Phase == komputerv1alpha1.AgentPhaseArchived
}

// agentDormant reports whether an agent is asleep in the sense every wake path
// cares about: Sleeping, or — with the S3 session store on — Archived. An
// archived agent then works exactly like a sleeping one with no workspace: waking
// it builds a fresh one and the agent restores its session from S3.
func agentDormant(agent *komputerv1alpha1.KomputerAgent, s3On bool) bool {
	return agent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping || (s3On && agentArchived(agent))
}

// validateArchivedPatch allows only labels and deleteTTL on an archived agent —
// the fields that still mean something on a record (organizing it, and how long
// to retain it). Everything else configures a run that will never happen.
// Implemented as an allowlist (clear the allowed fields, require the rest to be
// empty) so a field added to PatchAgentRequest later is refused by default.
func validateArchivedPatch(req PatchAgentRequest) error {
	rest := req
	rest.Labels, rest.RemoveLabels, rest.DeleteTTL = nil, nil, nil
	if !reflect.DeepEqual(rest, PatchAgentRequest{}) {
		return fmt.Errorf("agent is archived: only labels and deleteTTL can be changed")
	}
	return nil
}

// validateScheduleAgentRef rejects pointing a schedule at an existing agent
// that is archived or will archive itself after its next task — the same reason
// validateScheduleLifecycle rejects an Archive template. A missing agent is left
// to the existing "referenced agent not found" handling.
func validateScheduleAgentRef(ctx context.Context, k8s *K8sClient, ns, name string, s3On bool) error {
	if s3On {
		return nil // archived agents run again when the S3 session store is on
	}
	agent, err := k8s.GetAgent(ctx, ns, name)
	if err != nil {
		return nil
	}
	if agentArchived(agent) {
		return fmt.Errorf("agent %q is archived and can't run again", name)
	}
	return validateScheduleLifecycle(agent.Spec.Lifecycle, false)
}

// validateScheduleLifecycle rejects Lifecycle=Archive for a schedule's agent
// unless the S3 session store is on: every run reuses the same agent, and only
// the session store lets an archived agent run again.
func validateScheduleLifecycle(lifecycle komputerv1alpha1.AgentLifecycle, s3On bool) error {
	if lifecycle == komputerv1alpha1.AgentLifecycleArchive && !s3On {
		return fmt.Errorf("lifecycle: Archive on a schedule needs the S3 session store — every run reuses one agent, and an archived agent can only run again when its session is in S3. Use Sleep or AutoDelete, or configure KomputerConfig.spec.s3")
	}
	return nil
}
