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

// minTTLRequeue floors the requeue interval so a TTL that is a few milliseconds
// away can't produce a RequeueAfter of 0 — controller-runtime treats 0 as "do
// not requeue", which would strand the agent until some other event woke it.
const minTTLRequeue = time.Second

// ttlAction is the transition an elapsed TTL asks for.
type ttlAction int

const (
	// ttlActionNone means no TTL has elapsed yet.
	ttlActionNone ttlAction = iota
	// ttlActionSleep means SleepTTL elapsed: delete the pod, keep the PVC.
	ttlActionSleep
	// ttlActionDelete means DeleteTTL elapsed: delete the whole agent CR.
	ttlActionDelete
)

// ttlDecision is the full outcome of evaluating an agent's TTLs at one instant.
type ttlDecision struct {
	// Action is the transition to perform now.
	Action ttlAction
	// RequeueAfter is when to re-evaluate. Zero means no TTL is pending.
	RequeueAfter time.Duration
	// SleepExpiresAt / DeleteExpiresAt are the projected transition times, for
	// status. Nil when the corresponding TTL is unset or its clock isn't running.
	SleepExpiresAt  *metav1.Time
	DeleteExpiresAt *metav1.Time
}

// evaluateTTL decides what an agent's sleepTTL/deleteTTL require at time `now`.
// Pure function — reads only the agent and the resolved TTLs, mutates nothing.
//
// The two TTLs measure different things on purpose:
//
//   - sleepTTL is an *idle* timeout, and its clock only starts once the agent has
//     actually begun work: it measures from Status.LastActivityAt, which the API
//     worker first stamps when a task starts. An agent that has never run a task has
//     no idle clock and is never slept — otherwise a sleepTTL shorter than pod
//     startup would sleep an agent before it ever got to run. Any later task event or
//     wake pushes the deadline out. It never fires mid-task or on a sleeping agent.
//   - deleteTTL is an *absolute* lifetime from creationTimestamp. It does not reset on
//     wake and applies in every phase, including Sleeping — a lifetime cap that could
//     be deferred indefinitely wouldn't be a cap.
//
// Delete wins when both have elapsed, since it subsumes sleeping. When only sleep
// fires, RequeueAfter still carries the pending delete deadline so the lifetime cap
// keeps ticking against the now-sleeping agent.
func evaluateTTL(
	agent *komputerv1alpha1.KomputerAgent,
	sleepTTL, deleteTTL *metav1.Duration,
	now time.Time,
) ttlDecision {
	var d ttlDecision
	var sleepDeadline, deleteDeadline *time.Time

	if deleteTTL != nil && deleteTTL.Duration > 0 {
		dl := agent.CreationTimestamp.Time.Add(deleteTTL.Duration)
		deleteDeadline = &dl
		t := metav1.NewTime(dl)
		d.DeleteExpiresAt = &t
	}

	// The sleep clock only runs on an awake agent that has started at least one task
	// and isn't currently mid-task. Leaving SleepExpiresAt nil in the other cases is
	// deliberate: it tells the UI "no pending sleep" rather than showing a deadline
	// that isn't counting down.
	if since, ok := idleSince(agent); ok && sleepTTL != nil && sleepTTL.Duration > 0 &&
		agent.Status.Phase != komputerv1alpha1.AgentPhaseSleeping &&
		!taskInProgress(agent.Status.TaskStatus) {

		dl := since.Add(sleepTTL.Duration)
		sleepDeadline = &dl
		t := metav1.NewTime(dl)
		d.SleepExpiresAt = &t
	}

	switch {
	case deleteDeadline != nil && !now.Before(*deleteDeadline):
		d.Action = ttlActionDelete
		return d
	case sleepDeadline != nil && !now.Before(*sleepDeadline):
		d.Action = ttlActionSleep
	}

	// Next wake-up is the soonest deadline still in the future. A deadline that has
	// already fired isn't pending — its Action handles it this reconcile.
	for _, dl := range []*time.Time{sleepDeadline, deleteDeadline} {
		if dl == nil || !now.Before(*dl) {
			continue
		}
		if remaining := dl.Sub(now); d.RequeueAfter == 0 || remaining < d.RequeueAfter {
			d.RequeueAfter = remaining
		}
	}
	if d.RequeueAfter > 0 && d.RequeueAfter < minTTLRequeue {
		d.RequeueAfter = minTTLRequeue
	}
	return d
}

// taskDeadlineDecision is the outcome of evaluating an agent's taskTimeout at one instant.
type taskDeadlineDecision struct {
	// Cancel is true when the running task has outlived taskTimeout.
	Cancel bool
	// ExpiresAt is the projected cancellation time, for status. Nil when the clock
	// isn't running.
	ExpiresAt *metav1.Time
	// RequeueAfter is when to re-evaluate. Zero means no deadline is pending.
	RequeueAfter time.Duration
}

// evaluateTaskDeadline decides whether an agent's running task has exceeded taskTimeout
// at time `now`. Pure function — reads only the agent and the resolved timeout, mutates
// nothing.
//
// This is the mirror image of the sleepTTL clock in evaluateTTL. That one measures
// idleness and deliberately never fires mid-task; this one measures a single task's
// wall-clock runtime and only ever fires mid-task. It starts at Status.TaskStartedAt,
// which the API worker stamps once per task and does not refresh, so steering cannot
// extend the deadline — that is what makes taskTimeout a hard cap rather than an idle
// bound.
//
// The clock only runs on an agent that is actually mid-task on a live pod. The pod guard
// matters: a task left stuck at InProgress after its pod died would otherwise requeue
// forever, trying to cancel a task on a pod that no longer exists. It is expressed via
// Phase rather than a pod lookup so this stays a pure function and so the squad call
// site — which has no pod object in hand — can use the identical condition.
func evaluateTaskDeadline(
	agent *komputerv1alpha1.KomputerAgent,
	taskTimeout *metav1.Duration,
	now time.Time,
) taskDeadlineDecision {
	var d taskDeadlineDecision

	if taskTimeout == nil || taskTimeout.Duration <= 0 {
		return d
	}
	if !taskInProgress(agent.Status.TaskStatus) {
		return d
	}
	if agent.Status.TaskStartedAt == nil || agent.Status.TaskStartedAt.IsZero() {
		return d
	}
	if agent.Status.PodName == "" || agent.Status.Phase != komputerv1alpha1.AgentPhaseRunning {
		return d
	}

	deadline := agent.Status.TaskStartedAt.Time.Add(taskTimeout.Duration)
	if !now.Before(deadline) {
		d.Cancel = true
		return d
	}

	t := metav1.NewTime(deadline)
	d.ExpiresAt = &t
	d.RequeueAfter = deadline.Sub(now)
	if d.RequeueAfter < minTTLRequeue {
		d.RequeueAfter = minTTLRequeue
	}
	return d
}

// idleSince returns the instant the agent's idle clock started, and whether it is
// running at all. The clock is Status.LastActivityAt, which the API worker stamps on
// every agent event — the first of them when a task starts.
//
// There is deliberately no fallback to StartTime or creationTimestamp: those measure
// how long the agent has *existed*, not how long it has been idle. Falling back to
// them meant a sleepTTL shorter than pod startup could sleep an agent before its
// first task ever ran, losing that task. An agent that has never worked simply has
// no idle clock; deleteTTL is the backstop for one that is never used at all.
func idleSince(agent *komputerv1alpha1.KomputerAgent) (time.Time, bool) {
	if agent.Status.LastActivityAt != nil && !agent.Status.LastActivityAt.IsZero() {
		return agent.Status.LastActivityAt.Time, true
	}
	return time.Time{}, false
}

// taskInProgress reports whether the agent is mid-task, and so must not be slept.
// Compacting is a transient sub-state of InProgress, so it counts as busy too.
func taskInProgress(s komputerv1alpha1.AgentTaskStatus) bool {
	return s == komputerv1alpha1.AgentTaskInProgress || s == komputerv1alpha1.AgentTaskCompacting
}

// resolveAgentTTLs returns an agent's effective TTLs: its own spec values, falling
// back to its template's defaults for whichever field it doesn't set.
//
// The agent controller doesn't need this — it already merges the template via
// applyAgentOverrides. This exists for the squad controller, which never builds a
// merged template. A template that can't be read is treated as having no defaults:
// TTLs must not be able to break squad reconciliation.
func resolveAgentTTLs(ctx context.Context, c client.Client, agent *komputerv1alpha1.KomputerAgent) (sleepTTL, deleteTTL, taskTimeout *metav1.Duration) {
	sleepTTL, deleteTTL, taskTimeout = agent.Spec.SleepTTL, agent.Spec.DeleteTTL, agent.Spec.TaskTimeout
	if sleepTTL != nil && deleteTTL != nil && taskTimeout != nil {
		return sleepTTL, deleteTTL, taskTimeout // nothing left for the template to supply
	}

	templateRef := agent.Spec.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}
	spec, err := readTemplateSpec(ctx, c, templateRef, agent.Namespace)
	if err != nil {
		return sleepTTL, deleteTTL, taskTimeout
	}
	if sleepTTL == nil {
		sleepTTL = spec.SleepTTL
	}
	if deleteTTL == nil {
		deleteTTL = spec.DeleteTTL
	}
	if taskTimeout == nil {
		taskTimeout = spec.TaskTimeout
	}
	return sleepTTL, deleteTTL, taskTimeout
}

// readTemplateSpec resolves a template by name: namespaced first, then cluster-scoped,
// matching how the rest of the operator resolves templateRef.
func readTemplateSpec(ctx context.Context, c client.Client, name, namespace string) (*komputerv1alpha1.KomputerAgentTemplateSpec, error) {
	tpl := &komputerv1alpha1.KomputerAgentTemplate{}
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, tpl); err == nil {
		return &tpl.Spec, nil
	}
	clusterTpl := &komputerv1alpha1.KomputerAgentClusterTemplate{}
	if err := c.Get(ctx, types.NamespacedName{Name: name}, clusterTpl); err != nil {
		return nil, err
	}
	return &clusterTpl.Spec, nil
}

// applyTTL evaluates the agent's TTLs and performs any transition they require.
//
// Returns handled=true when it slept or deleted the agent, in which case the caller
// must stop reconciling — the agent is either gone or its pod has been torn down.
// When nothing fired, it persists the projected expiry timestamps and returns the
// requeue hint so the controller wakes exactly when the next TTL comes due.
func (r *KomputerAgentReconciler) applyTTL(
	ctx context.Context,
	agent *komputerv1alpha1.KomputerAgent,
	pod *corev1.Pod,
	pvcName string,
	sleepTTL, deleteTTL *metav1.Duration,
) (ctrl.Result, bool, error) {
	log := logf.FromContext(ctx)

	if sleepTTL == nil && deleteTTL == nil {
		// Clear stale expiry timestamps if the TTLs were removed from the spec.
		if agent.Status.SleepExpiresAt != nil || agent.Status.DeleteExpiresAt != nil {
			if err := r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
				s.SleepExpiresAt = nil
				s.DeleteExpiresAt = nil
			}); err != nil {
				return ctrl.Result{}, false, err
			}
		}
		return ctrl.Result{}, false, nil
	}

	// The issue calls for a warning when the ordering is nonsensical: a deleteTTL at
	// or below sleepTTL means the agent is deleted before it ever gets to sleep.
	if sleepTTL != nil && deleteTTL != nil && deleteTTL.Duration <= sleepTTL.Duration {
		log.Info("deleteTTL is not greater than sleepTTL — agent will be deleted before it can sleep",
			"agent", agent.Name, "sleepTTL", sleepTTL.Duration, "deleteTTL", deleteTTL.Duration)
	}

	d := evaluateTTL(agent, sleepTTL, deleteTTL, time.Now())

	switch d.Action {
	case ttlActionDelete:
		log.Info("deleteTTL elapsed, deleting agent",
			"agent", agent.Name, "deleteTTL", deleteTTL.Duration, "createdAt", agent.CreationTimestamp)
		if err := r.Delete(ctx, agent); err != nil && !errors.IsNotFound(err) {
			return ctrl.Result{}, false, err
		}
		return ctrl.Result{}, true, nil

	case ttlActionSleep:
		idleFrom, _ := idleSince(agent) // always set: the sleep action can't fire without it
		log.Info("sleepTTL elapsed, putting agent to sleep",
			"agent", agent.Name, "sleepTTL", sleepTTL.Duration, "idleSince", idleFrom)
		if pod != nil {
			if err := r.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
				return ctrl.Result{}, false, err
			}
		}
		if err := r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
			s.Phase = komputerv1alpha1.AgentPhaseSleeping
			s.PodName = ""
			s.PvcName = pvcName
			s.Message = "Sleeping — idle past sleepTTL, workspace preserved. Send a new task to wake up."
			// No pending sleep once asleep; the lifetime cap keeps its deadline.
			s.SleepExpiresAt = nil
			s.DeleteExpiresAt = d.DeleteExpiresAt
		}); err != nil {
			return ctrl.Result{}, false, err
		}
		return ctrl.Result{RequeueAfter: d.RequeueAfter}, true, nil
	}

	// Nothing fired — publish the projected deadlines so clients can show a countdown.
	if !timeEqual(agent.Status.SleepExpiresAt, d.SleepExpiresAt) ||
		!timeEqual(agent.Status.DeleteExpiresAt, d.DeleteExpiresAt) {
		if err := r.updateStatus(ctx, agent, func(s *komputerv1alpha1.KomputerAgentStatus) {
			s.SleepExpiresAt = d.SleepExpiresAt
			s.DeleteExpiresAt = d.DeleteExpiresAt
		}); err != nil {
			return ctrl.Result{}, false, err
		}
	}
	return ctrl.Result{RequeueAfter: d.RequeueAfter}, false, nil
}

// timeEqual compares two optional timestamps at second granularity, matching how
// metav1.Time round-trips through the apiserver. Without the truncation every
// reconcile would see sub-second drift and patch status forever.
func timeEqual(a, b *metav1.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Time.Unix() == b.Time.Unix()
}
