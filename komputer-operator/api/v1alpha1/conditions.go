package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Conditions that record why an agent changed state. Phase and Message keep
// saying what the state is; these say what caused it, as a machine-readable
// Reason. Each one is set by whichever component performs the transition (the
// operator for TTLs and lifecycle, the API for wakes, API-driven squad sleeps
// and task starts), in the same status write as the transition itself.
const (
	// ConditionAsleep is True while the agent sleeps. Reasons: SleepTTL,
	// LifecycleSleep, SleepRequested; False with Woken or Archived.
	ConditionAsleep = "Asleep"
	// ConditionArchived is True once the agent is archived. Reasons: ArchiveTTL,
	// LifecycleArchive, ArchiveRequested; False with Woken (S3 session store).
	ConditionArchived = "Archived"
	// ConditionTaskTimedOut is True when taskTimeout cancelled the latest task
	// (reason TaskTimeout); False with TaskStarted once the next task starts.
	ConditionTaskTimedOut = "TaskTimedOut"
)

// Condition reasons.
const (
	ReasonSleepTTL         = "SleepTTL"
	ReasonLifecycleSleep   = "LifecycleSleep"
	ReasonSleepRequested   = "SleepRequested"
	ReasonArchiveTTL       = "ArchiveTTL"
	ReasonLifecycleArchive = "LifecycleArchive"
	ReasonArchiveRequested = "ArchiveRequested"
	ReasonArchived         = "Archived"
	ReasonWoken            = "Woken"
	ReasonTaskTimeout      = "TaskTimeout"
	ReasonTaskStarted      = "TaskStarted"
	// ReasonDeleteTTL is only an Event reason: a deleted agent has no status left.
	ReasonDeleteTTL = "DeleteTTL"
)

// SetCondition records condType as True with reason and message, and reports
// whether that changed anything — callers emit an Event only for a real change.
func (s *KomputerAgentStatus) SetCondition(condType, reason, message string) bool {
	return meta.SetStatusCondition(&s.Conditions, metav1.Condition{
		Type: condType, Status: metav1.ConditionTrue, Reason: reason, Message: message,
	})
}

// ResolveCondition flips condType to False with reason, only if it is currently
// True — an agent that never slept doesn't grow an "Asleep: False" entry.
func (s *KomputerAgentStatus) ResolveCondition(condType, reason, message string) {
	if !meta.IsStatusConditionTrue(s.Conditions, condType) {
		return
	}
	meta.SetStatusCondition(&s.Conditions, metav1.Condition{
		Type: condType, Status: metav1.ConditionFalse, Reason: reason, Message: message,
	})
}
