package controller

import (
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// agentProgressOnly reports whether the only difference between two versions of a
// KomputerAgent is task-progress noise: Status.LastTaskMessage and
// Status.LastActivityAt, which the API worker refreshes on every Redis event while a
// task runs. Nothing the controllers decide depends on those two fields mid-task —
// the idle clock only matters once TaskStatus leaves InProgress, and that transition
// is itself a change this function does not ignore.
func agentProgressOnly(oldA, newA *komputerv1alpha1.KomputerAgent) bool {
	if oldA.GetGeneration() != newA.GetGeneration() ||
		!oldA.GetDeletionTimestamp().Equal(newA.GetDeletionTimestamp()) ||
		!equality.Semantic.DeepEqual(oldA.GetLabels(), newA.GetLabels()) ||
		!equality.Semantic.DeepEqual(oldA.GetAnnotations(), newA.GetAnnotations()) ||
		!equality.Semantic.DeepEqual(oldA.GetFinalizers(), newA.GetFinalizers()) ||
		!equality.Semantic.DeepEqual(oldA.GetOwnerReferences(), newA.GetOwnerReferences()) {
		return false
	}
	oldS, newS := oldA.Status.DeepCopy(), newA.Status.DeepCopy()
	oldS.LastTaskMessage, newS.LastTaskMessage = "", ""
	oldS.LastActivityAt, newS.LastActivityAt = nil, nil
	return equality.Semantic.DeepEqual(oldS, newS)
}

// ignoreAgentProgressUpdates drops KomputerAgent update events that only carry
// task-progress noise (see agentProgressOnly). Without it, every streamed text or
// tool event the API writes re-runs each controller watching agents. A plain
// GenerationChangedPredicate is not enough here: several transitions arrive as
// status-only writes (a schedule waking a sleeping agent sets Phase=Pending, task
// completion sets TaskStatus) and the controllers must still see those.
var ignoreAgentProgressUpdates = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		oldA, ok1 := e.ObjectOld.(*komputerv1alpha1.KomputerAgent)
		newA, ok2 := e.ObjectNew.(*komputerv1alpha1.KomputerAgent)
		if !ok1 || !ok2 {
			return true
		}
		return !agentProgressOnly(oldA, newA)
	},
}
