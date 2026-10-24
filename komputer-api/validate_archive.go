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
	return validateScheduleLifecycle(agent.Spec.AgentConfigSpec, false)
}

// validateScheduleLifecycle rejects Lifecycle=Archive or an archiveTTL for a
// schedule's agent unless archiveOK: the S3 session store is on, or the schedule
// creates a new agent per run. Otherwise every run reuses the same agent, and only
// the session store lets an archived agent run again.
func validateScheduleLifecycle(cfg komputerv1alpha1.AgentConfigSpec, archiveOK bool) error {
	if archiveOK {
		return nil
	}
	if cfg.Lifecycle == komputerv1alpha1.AgentLifecycleArchive {
		return fmt.Errorf("lifecycle: Archive on a schedule needs the S3 session store — every run reuses one agent, and an archived agent can only run again when its session is in S3. Use Sleep or AutoDelete, set agentNaming to Date or Random, or configure KomputerConfig.spec.s3")
	}
	if cfg.ArchiveTTL != nil {
		return fmt.Errorf("archiveTTL on a schedule needs the S3 session store — every run reuses one agent, and an archived agent can only run again when its session is in S3. Use sleepTTL, set agentNaming to Date or Random, or configure KomputerConfig.spec.s3")
	}
	return nil
}

// ErrArchiveTTLInSquad is returned when archiveTTL is set on a squad member, or an
// agent with one is added to a squad: archiving deletes the member's PVC, which the
// shared squad pod mounts (the same reason ArchiveAgentNow refuses squad members).
var ErrArchiveTTLInSquad = fmt.Errorf("archiveTTL isn't supported on squad members: remove the agent from the squad, or clear its archiveTTL")

// agentInSquad reports whether the agent is a squad member. The label is written
// when the squad controller creates the agent; Status.Squad when it adopts one.
func agentInSquad(agent *komputerv1alpha1.KomputerAgent) bool {
	return agent.Status.Squad || agent.Labels["komputer.ai/squad"] != ""
}

// validateSquadMembersNoArchiveTTL rejects squad members — inline specs or refs to
// existing agents — that carry their own archiveTTL.
func validateSquadMembersNoArchiveTTL(ctx context.Context, k8s *K8sClient, ns string, members []komputerv1alpha1.KomputerSquadMember) error {
	for _, m := range members {
		if m.Spec != nil && m.Spec.ArchiveTTL != nil {
			return ErrArchiveTTLInSquad
		}
		if m.Ref == nil {
			continue
		}
		refNs := m.Ref.Namespace
		if refNs == "" {
			refNs = ns
		}
		if a, err := k8s.GetAgent(ctx, refNs, m.Ref.Name); err == nil && a.Spec.ArchiveTTL != nil {
			return fmt.Errorf("agent %q has an archiveTTL: %w", m.Ref.Name, ErrArchiveTTLInSquad)
		}
	}
	return nil
}
