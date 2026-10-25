package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// archivedMessage is the status message of an agent archived after its task.
const archivedMessage = "Archived — task complete, pod and workspace removed. The agent is kept as a record."

// archiveAgentNow is the Archive transition (Lifecycle=Archive, an archive request,
// or archiveTTL): mark the agent Archived,
// then delete its pod, PVC and Service. The CR — spec, cost, tokens, last task
// message — is kept as a read-only record.
//
// Status is written first on purpose. If the deletes fail, the next reconcile
// takes the reconcileArchived path and retries them; deleting first and failing
// the status write would instead let ensurePod build a fresh pod for an agent
// whose task is already done.
func (r *KomputerAgentReconciler) archiveAgentNow(ctx context.Context, agent *komputerv1alpha1.KomputerAgent, reason, message string) error {
	var changed bool
	if err := r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
		s.Phase = komputerv1alpha1.AgentPhaseArchived
		s.PodName = ""
		s.PvcName = ""
		s.Message = message
		changed = s.SetCondition(komputerv1alpha1.ConditionArchived, reason, message)
		s.ResolveCondition(komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonArchived, "Archived while asleep")
		if s.CompletionTime == nil {
			now := metav1.Now()
			s.CompletionTime = &now
		}
		s.SleepExpiresAt = nil
		s.ArchiveExpiresAt = nil
		s.TaskExpiresAt = nil
	}); err != nil {
		return err
	}
	if changed {
		recordEvent(r.Recorder, agent, reason, message)
	}
	return r.deleteArchivedResources(ctx, agent)
}

// reconcileArchived is the whole reconcile for an archived agent. It re-runs the
// resource cleanup (idempotent — a missing object is not an error) so a delete
// that failed during archiveAgentNow is retried, and enforces deleteTTL: as an
// absolute lifetime cap, it is how a retention window is put on archived records.
// Nothing else applies — an archived agent never gets a pod again.
func (r *KomputerAgentReconciler) reconcileArchived(ctx context.Context, agent *komputerv1alpha1.KomputerAgent) (ctrl.Result, error) {
	if err := r.deleteArchivedResources(ctx, agent); err != nil {
		return ctrl.Result{}, err
	}
	// An archive-now request that couldn't be cleared when it was honored would
	// archive the agent again the moment it's woken.
	if _, pending := agent.Annotations[komputerv1alpha1.AnnotationArchiveRequested]; pending {
		if err := r.clearArchiveRequestAnnotations(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
	}

	_, deleteTTL, _ := resolveAgentTTLs(ctx, r.Client, agent)
	d := evaluateTTL(agent, nil, deleteTTL, time.Now(), false)
	if d.Action == ttlActionDelete {
		logf.FromContext(ctx).Info("deleteTTL elapsed, deleting archived agent",
			"agent", agent.Name, "deleteTTL", deleteTTL.Duration)
		recordEvent(r.Recorder, agent, komputerv1alpha1.ReasonDeleteTTL, deleteTTLMessage(deleteTTL))
		if err := r.Delete(ctx, agent); err != nil && !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if err := r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
		s.DeleteExpiresAt = d.DeleteExpiresAt
	}); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: d.RequeueAfter}, nil
}

// deleteArchivedResources deletes the pod, workspace PVC and Service of an
// archived agent. Each object is read from the cache first, so reconciling an
// archived agent whose resources are already gone (the common case — archived
// records accumulate) sends no API requests. Ephemeral agents have no PVC.
func (r *KomputerAgentReconciler) deleteArchivedResources(ctx context.Context, agent *komputerv1alpha1.KomputerAgent) error {
	for _, res := range []struct {
		name string
		obj  client.Object
	}{
		{agent.Name + "-pod", &corev1.Pod{}},
		{agent.Name + "-pvc", &corev1.PersistentVolumeClaim{}},
		{agentServiceName(agent.Name), &corev1.Service{}},
	} {
		obj := res.obj
		if err := r.Get(ctx, types.NamespacedName{Name: res.name, Namespace: agent.Namespace}, obj); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return err
		}
		if !obj.GetDeletionTimestamp().IsZero() {
			continue
		}
		if err := r.Delete(ctx, obj); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
