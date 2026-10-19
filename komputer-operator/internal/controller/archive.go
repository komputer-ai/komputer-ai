package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// archiveAgentNow is the Lifecycle=Archive transition: mark the agent Archived,
// then delete its pod, PVC and Service. The CR — spec, cost, tokens, last task
// message — is kept as a read-only record.
//
// Status is written first on purpose. If the deletes fail, the next reconcile
// takes the reconcileArchived path and retries them; deleting first and failing
// the status write would instead let ensurePod build a fresh pod for an agent
// whose task is already done.
func (r *KomputerAgentReconciler) archiveAgentNow(ctx context.Context, agent *komputerv1alpha1.KomputerAgent) error {
	if err := r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
		s.Phase = komputerv1alpha1.AgentPhaseArchived
		s.PodName = ""
		s.PvcName = ""
		s.Message = "Archived — task complete, pod and workspace removed. The agent is kept as a record."
		if s.CompletionTime == nil {
			now := metav1.Now()
			s.CompletionTime = &now
		}
		s.SleepExpiresAt = nil
		s.TaskExpiresAt = nil
	}); err != nil {
		return err
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

	_, deleteTTL, _ := resolveAgentTTLs(ctx, r.Client, agent)
	d := evaluateTTL(agent, nil, deleteTTL, time.Now(), false)
	if d.Action == ttlActionDelete {
		logf.FromContext(ctx).Info("deleteTTL elapsed, deleting archived agent",
			"agent", agent.Name, "deleteTTL", deleteTTL.Duration)
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
// archived agent. Ephemeral agents have no PVC; NotFound is ignored throughout.
func (r *KomputerAgentReconciler) deleteArchivedResources(ctx context.Context, agent *komputerv1alpha1.KomputerAgent) error {
	meta := func(name string) metav1.ObjectMeta {
		return metav1.ObjectMeta{Name: name, Namespace: agent.Namespace}
	}
	for _, obj := range []client.Object{
		&corev1.Pod{ObjectMeta: meta(agent.Name + "-pod")},
		&corev1.PersistentVolumeClaim{ObjectMeta: meta(agent.Name + "-pvc")},
		&corev1.Service{ObjectMeta: meta(agentServiceName(agent.Name))},
	} {
		if err := r.Delete(ctx, obj); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
