package controller

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// baseTime is a fixed reference so every case reads as an explicit offset from it.
//
// Only for the pure evaluateTTL tests, which are handed an explicit now derived
// from it. The applyTTL tests below read the real wall clock, so their fixtures
// must be now-relative (withCreatedAgo / withLastActivityAgo) — anchoring those
// to a fixed date rots silently as the date recedes past the TTLs under test.
var baseTime = time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)

func dur(d time.Duration) *metav1.Duration {
	return &metav1.Duration{Duration: d}
}

// ttlAgent builds an agent created at baseTime, with optional status overrides.
func ttlAgent(mutate ...func(*komputerv1alpha1.KomputerAgent)) *komputerv1alpha1.KomputerAgent {
	a := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-agent",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(baseTime),
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	for _, m := range mutate {
		m(a)
	}
	return a
}

func withLastActivity(offset time.Duration) func(*komputerv1alpha1.KomputerAgent) {
	return func(a *komputerv1alpha1.KomputerAgent) {
		t := metav1.NewTime(baseTime.Add(offset))
		a.Status.LastActivityAt = &t
	}
}

// withCreatedAgo backdates creationTimestamp by d from real time.Now(), which is
// what the deleteTTL lifetime cap is measured against on the applyTTL path.
func withCreatedAgo(d time.Duration) func(*komputerv1alpha1.KomputerAgent) {
	return func(a *komputerv1alpha1.KomputerAgent) {
		a.CreationTimestamp = metav1.NewTime(time.Now().Add(-d))
	}
}

// withLastActivityAgo sets the idle clock d before real time.Now(), which is what
// the sleepTTL is measured against on the applyTTL path.
func withLastActivityAgo(d time.Duration) func(*komputerv1alpha1.KomputerAgent) {
	return func(a *komputerv1alpha1.KomputerAgent) {
		t := metav1.NewTime(time.Now().Add(-d))
		a.Status.LastActivityAt = &t
	}
}

func withPhase(p komputerv1alpha1.KomputerAgentPhase) func(*komputerv1alpha1.KomputerAgent) {
	return func(a *komputerv1alpha1.KomputerAgent) { a.Status.Phase = p }
}

func withTaskStatus(s komputerv1alpha1.AgentTaskStatus) func(*komputerv1alpha1.KomputerAgent) {
	return func(a *komputerv1alpha1.KomputerAgent) { a.Status.TaskStatus = s }
}

func TestEvaluateTTL(t *testing.T) {
	tests := []struct {
		name       string
		agent      *komputerv1alpha1.KomputerAgent
		sleepTTL   *metav1.Duration
		deleteTTL  *metav1.Duration
		now        time.Time
		wantAction ttlAction
		// wantRequeue is checked only when non-zero.
		wantRequeue time.Duration
	}{
		{
			name:       "no TTLs set does nothing",
			agent:      ttlAgent(),
			now:        baseTime.Add(999 * time.Hour),
			wantAction: ttlActionNone,
		},
		{
			name:        "sleepTTL not yet elapsed requeues for the remainder",
			agent:       ttlAgent(withLastActivity(0)),
			sleepTTL:    dur(30 * time.Minute),
			now:         baseTime.Add(10 * time.Minute),
			wantAction:  ttlActionNone,
			wantRequeue: 20 * time.Minute,
		},
		{
			name:       "sleepTTL elapsed sleeps the agent",
			agent:      ttlAgent(withLastActivity(0)),
			sleepTTL:   dur(30 * time.Minute),
			now:        baseTime.Add(31 * time.Minute),
			wantAction: ttlActionSleep,
		},
		{
			name:       "sleepTTL exactly elapsed sleeps the agent",
			agent:      ttlAgent(withLastActivity(0)),
			sleepTTL:   dur(30 * time.Minute),
			now:        baseTime.Add(30 * time.Minute),
			wantAction: ttlActionSleep,
		},
		{
			// The whole point of the idle clock: recent activity pushes the deadline out.
			name:        "recent activity resets the sleep clock",
			agent:       ttlAgent(withLastActivity(25 * time.Minute)),
			sleepTTL:    dur(30 * time.Minute),
			now:         baseTime.Add(31 * time.Minute),
			wantAction:  ttlActionNone,
			wantRequeue: 24 * time.Minute,
		},
		{
			// Guard the issue's spec omission: a creation-clocked sleepTTL would kill
			// in-flight work here.
			name:       "in-progress task is never slept",
			agent:      ttlAgent(withLastActivity(0), withTaskStatus(komputerv1alpha1.AgentTaskInProgress)),
			sleepTTL:   dur(30 * time.Minute),
			now:        baseTime.Add(99 * time.Hour),
			wantAction: ttlActionNone,
		},
		{
			name:       "compacting task is never slept",
			agent:      ttlAgent(withLastActivity(0), withTaskStatus(komputerv1alpha1.AgentTaskCompacting)),
			sleepTTL:   dur(30 * time.Minute),
			now:        baseTime.Add(99 * time.Hour),
			wantAction: ttlActionNone,
		},
		{
			name:       "errored task is idle and can be slept",
			agent:      ttlAgent(withLastActivity(0), withTaskStatus(komputerv1alpha1.AgentTaskError)),
			sleepTTL:   dur(30 * time.Minute),
			now:        baseTime.Add(31 * time.Minute),
			wantAction: ttlActionSleep,
		},
		{
			name:       "already sleeping agent is not slept again",
			agent:      ttlAgent(withLastActivity(0), withPhase(komputerv1alpha1.AgentPhaseSleeping)),
			sleepTTL:   dur(30 * time.Minute),
			now:        baseTime.Add(99 * time.Hour),
			wantAction: ttlActionNone,
		},
		{
			name:       "deleteTTL elapsed deletes the agent",
			agent:      ttlAgent(withLastActivity(0)),
			deleteTTL:  dur(24 * time.Hour),
			now:        baseTime.Add(25 * time.Hour),
			wantAction: ttlActionDelete,
		},
		{
			// deleteTTL is an absolute lifetime cap: activity must not defer it.
			name:       "activity does not defer deleteTTL",
			agent:      ttlAgent(withLastActivity(23 * time.Hour)),
			deleteTTL:  dur(24 * time.Hour),
			now:        baseTime.Add(25 * time.Hour),
			wantAction: ttlActionDelete,
		},
		{
			name:       "deleteTTL applies to a sleeping agent",
			agent:      ttlAgent(withPhase(komputerv1alpha1.AgentPhaseSleeping)),
			deleteTTL:  dur(24 * time.Hour),
			now:        baseTime.Add(25 * time.Hour),
			wantAction: ttlActionDelete,
		},
		{
			name:       "deleteTTL applies to an in-progress task",
			agent:      ttlAgent(withTaskStatus(komputerv1alpha1.AgentTaskInProgress)),
			deleteTTL:  dur(24 * time.Hour),
			now:        baseTime.Add(25 * time.Hour),
			wantAction: ttlActionDelete,
		},
		{
			name:       "delete wins when both elapsed",
			agent:      ttlAgent(withLastActivity(0)),
			sleepTTL:   dur(30 * time.Minute),
			deleteTTL:  dur(24 * time.Hour),
			now:        baseTime.Add(25 * time.Hour),
			wantAction: ttlActionDelete,
		},
		{
			// After a TTL-sleep the lifetime cap must keep ticking, so the requeue has
			// to carry the pending delete deadline rather than dropping to zero.
			name:        "sleep fires but still requeues for the pending delete",
			agent:       ttlAgent(withLastActivity(0)),
			sleepTTL:    dur(30 * time.Minute),
			deleteTTL:   dur(24 * time.Hour),
			now:         baseTime.Add(31 * time.Minute),
			wantAction:  ttlActionSleep,
			wantRequeue: 24*time.Hour - 31*time.Minute,
		},
		{
			name:        "soonest of the two deadlines wins the requeue",
			agent:       ttlAgent(withLastActivity(0)),
			sleepTTL:    dur(30 * time.Minute),
			deleteTTL:   dur(24 * time.Hour),
			now:         baseTime.Add(10 * time.Minute),
			wantAction:  ttlActionNone,
			wantRequeue: 20 * time.Minute,
		},
		{
			// A sub-second remainder must not round to zero: controller-runtime reads
			// RequeueAfter=0 as "never requeue", which would strand the agent.
			name:        "sub-second remainder is floored to the minimum",
			agent:       ttlAgent(withLastActivity(0)),
			sleepTTL:    dur(30 * time.Minute),
			now:         baseTime.Add(30*time.Minute - 10*time.Millisecond),
			wantAction:  ttlActionNone,
			wantRequeue: minTTLRequeue,
		},
		{
			name:       "zero-valued TTL is treated as unset",
			agent:      ttlAgent(withLastActivity(0)),
			sleepTTL:   dur(0),
			deleteTTL:  dur(0),
			now:        baseTime.Add(99 * time.Hour),
			wantAction: ttlActionNone,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateTTL(tc.agent, tc.sleepTTL, tc.deleteTTL, tc.now)
			if got.Action != tc.wantAction {
				t.Errorf("Action = %v, want %v", got.Action, tc.wantAction)
			}
			if tc.wantRequeue != 0 && got.RequeueAfter != tc.wantRequeue {
				t.Errorf("RequeueAfter = %v, want %v", got.RequeueAfter, tc.wantRequeue)
			}
		})
	}
}

func TestEvaluateTTLExpiryTimestamps(t *testing.T) {
	t.Run("both deadlines are projected for status", func(t *testing.T) {
		agent := ttlAgent(withLastActivity(5 * time.Minute))
		d := evaluateTTL(agent, dur(30*time.Minute), dur(24*time.Hour), baseTime.Add(10*time.Minute))

		if d.SleepExpiresAt == nil {
			t.Fatal("SleepExpiresAt = nil, want a timestamp")
		}
		// Idle clock starts at lastActivityAt (+5m), so sleep is due at +35m.
		if want := baseTime.Add(35 * time.Minute); !d.SleepExpiresAt.Time.Equal(want) {
			t.Errorf("SleepExpiresAt = %v, want %v", d.SleepExpiresAt.Time, want)
		}
		if d.DeleteExpiresAt == nil {
			t.Fatal("DeleteExpiresAt = nil, want a timestamp")
		}
		// Delete clock starts at creationTimestamp, unaffected by activity.
		if want := baseTime.Add(24 * time.Hour); !d.DeleteExpiresAt.Time.Equal(want) {
			t.Errorf("DeleteExpiresAt = %v, want %v", d.DeleteExpiresAt.Time, want)
		}
	})

	t.Run("sleep expiry is nil while a task is in progress", func(t *testing.T) {
		agent := ttlAgent(withLastActivity(0), withTaskStatus(komputerv1alpha1.AgentTaskInProgress))
		d := evaluateTTL(agent, dur(30*time.Minute), nil, baseTime.Add(10*time.Minute))
		if d.SleepExpiresAt != nil {
			t.Errorf("SleepExpiresAt = %v, want nil (clock not running)", d.SleepExpiresAt)
		}
	})

	t.Run("sleep expiry is nil once asleep", func(t *testing.T) {
		agent := ttlAgent(withPhase(komputerv1alpha1.AgentPhaseSleeping))
		d := evaluateTTL(agent, dur(30*time.Minute), nil, baseTime.Add(10*time.Minute))
		if d.SleepExpiresAt != nil {
			t.Errorf("SleepExpiresAt = %v, want nil (already sleeping)", d.SleepExpiresAt)
		}
	})
}

func TestIdleSince(t *testing.T) {
	t.Run("runs from lastActivityAt", func(t *testing.T) {
		agent := ttlAgent(withLastActivity(10 * time.Minute))
		got, ok := idleSince(agent)
		if !ok {
			t.Fatal("ok = false, want true")
		}
		if want := baseTime.Add(10 * time.Minute); !got.Equal(want) {
			t.Errorf("idleSince = %v, want %v", got, want)
		}
	})

	t.Run("does not run before the first task", func(t *testing.T) {
		agent := ttlAgent()
		// StartTime (pod started) must NOT start the idle clock — existing is not idle.
		start := metav1.NewTime(baseTime.Add(5 * time.Minute))
		agent.Status.StartTime = &start
		if _, ok := idleSince(agent); ok {
			t.Error("ok = true, want false — no task has started yet")
		}
	})

	t.Run("ignores a zero-valued lastActivityAt", func(t *testing.T) {
		agent := ttlAgent()
		agent.Status.LastActivityAt = &metav1.Time{}
		if _, ok := idleSince(agent); ok {
			t.Error("ok = true for a zero timestamp, want false")
		}
	})
}

// An agent that has never started a task must never be auto-slept, however long ago
// it was created. Measuring from creationTimestamp meant a sleepTTL shorter than pod
// startup could sleep an agent before its first task ran, losing that task.
func TestEvaluateTTLNeverRanAgentIsNotSlept(t *testing.T) {
	agent := ttlAgent(withPhase(komputerv1alpha1.AgentPhasePending), withTaskStatus(""))
	d := evaluateTTL(agent, dur(30*time.Minute), nil, baseTime.Add(99*time.Hour))
	if d.Action != ttlActionNone {
		t.Errorf("Action = %v, want ttlActionNone (no task has started)", d.Action)
	}
	if d.SleepExpiresAt != nil {
		t.Errorf("SleepExpiresAt = %v, want nil (clock not running)", d.SleepExpiresAt)
	}
	if d.RequeueAfter != 0 {
		t.Errorf("RequeueAfter = %v, want 0 (nothing pending)", d.RequeueAfter)
	}
}

// A slow-starting pod is the case this protects: the agent exists well past its
// sleepTTL but hasn't begun work, so the clock hasn't started.
func TestEvaluateTTLSlowPodStartupIsNotSlept(t *testing.T) {
	agent := ttlAgent(withPhase(komputerv1alpha1.AgentPhasePending), withTaskStatus(""))
	start := metav1.NewTime(baseTime)
	agent.Status.StartTime = &start
	d := evaluateTTL(agent, dur(time.Minute), nil, baseTime.Add(10*time.Minute))
	if d.Action != ttlActionNone {
		t.Errorf("Action = %v, want ttlActionNone — must not sleep before the first task", d.Action)
	}
}

// Once the first task starts, the clock does run and the agent sleeps normally.
func TestEvaluateTTLClockStartsWithFirstTask(t *testing.T) {
	agent := ttlAgent(withLastActivity(0), withTaskStatus(komputerv1alpha1.AgentTaskComplete))
	d := evaluateTTL(agent, dur(30*time.Minute), nil, baseTime.Add(31*time.Minute))
	if d.Action != ttlActionSleep {
		t.Errorf("Action = %v, want ttlActionSleep once a task has run", d.Action)
	}
}

// deleteTTL is the backstop for an agent that is never used: it measures existence,
// not idleness, so it still reclaims a never-tasked agent.
func TestEvaluateTTLDeleteStillReclaimsNeverRanAgent(t *testing.T) {
	agent := ttlAgent(withPhase(komputerv1alpha1.AgentPhasePending), withTaskStatus(""))
	d := evaluateTTL(agent, dur(30*time.Minute), dur(24*time.Hour), baseTime.Add(25*time.Hour))
	if d.Action != ttlActionDelete {
		t.Errorf("Action = %v, want ttlActionDelete", d.Action)
	}
}

func TestTimeEqual(t *testing.T) {
	now := metav1.NewTime(baseTime)
	sameSecond := metav1.NewTime(baseTime.Add(400 * time.Millisecond))
	later := metav1.NewTime(baseTime.Add(2 * time.Second))

	if !timeEqual(nil, nil) {
		t.Error("timeEqual(nil, nil) = false, want true")
	}
	if timeEqual(&now, nil) || timeEqual(nil, &now) {
		t.Error("timeEqual with one nil = true, want false")
	}
	// Sub-second drift must compare equal or status would be patched every reconcile.
	if !timeEqual(&now, &sameSecond) {
		t.Error("timeEqual within the same second = false, want true")
	}
	if timeEqual(&now, &later) {
		t.Error("timeEqual across seconds = true, want false")
	}
}

// ─── applyTTL side effects (fake client) ─────────────────────────────────────

func newTTLReconciler(t *testing.T, objs ...client.Object) *KomputerAgentReconciler {
	t.Helper()
	s := newTestScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objs...).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
		Build()
	return &KomputerAgentReconciler{Client: c, Scheme: s}
}

// agentPod is the pod applyTTL is expected to delete when sleeping.
func agentPod(name, ns string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
}

func TestApplyTTL_SleepDeletesPodAndSetsSleepingPhase(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent(withCreatedAgo(time.Hour), withLastActivityAgo(time.Hour)) // idle for an hour
	pod := agentPod("test-agent-pod", "default")
	r := newTTLReconciler(t, agent, pod)

	res, handled, err := r.applyTTL(ctx, agent, pod, "test-agent-pvc", dur(30*time.Minute), nil)
	if err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true (sleep fired)")
	}
	if res.RequeueAfter != 0 {
		t.Errorf("RequeueAfter = %v, want 0 (no deleteTTL pending)", res.RequeueAfter)
	}

	// Pod must be gone; the PVC is untouched (not owned by this call).
	err = r.Get(ctx, types.NamespacedName{Name: "test-agent-pod", Namespace: "default"}, &corev1.Pod{})
	if err == nil {
		t.Error("pod still exists, want it deleted")
	}

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase = %q, want Sleeping", got.Status.Phase)
	}
	if got.Status.PodName != "" {
		t.Errorf("PodName = %q, want empty", got.Status.PodName)
	}
	if got.Status.PvcName != "test-agent-pvc" {
		t.Errorf("PvcName = %q, want the PVC to be preserved", got.Status.PvcName)
	}
	if got.Status.SleepExpiresAt != nil {
		t.Error("SleepExpiresAt should be cleared once asleep")
	}
}

func TestApplyTTL_SleepStillWatchesPendingDelete(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent(withCreatedAgo(time.Hour), withLastActivityAgo(time.Hour))
	pod := agentPod("test-agent-pod", "default")
	r := newTTLReconciler(t, agent, pod)

	res, handled, err := r.applyTTL(ctx, agent, pod, "pvc", dur(30*time.Minute), dur(240*time.Hour))
	if err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true")
	}
	// The lifetime cap must keep ticking against the now-sleeping agent.
	if res.RequeueAfter <= 0 {
		t.Errorf("RequeueAfter = %v, want a pending delete deadline", res.RequeueAfter)
	}

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.DeleteExpiresAt == nil {
		t.Error("DeleteExpiresAt should survive the sleep transition")
	}
}

func TestApplyTTL_SleepWithNoPodStillSleeps(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent(withCreatedAgo(time.Hour), withLastActivityAgo(time.Hour))
	r := newTTLReconciler(t, agent)

	// A nil pod (deleted out from under us) must not panic or error.
	_, handled, err := r.applyTTL(ctx, agent, nil, "pvc", dur(30*time.Minute), nil)
	if err != nil {
		t.Fatalf("applyTTL with nil pod: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true")
	}
}

func TestApplyTTL_DeleteRemovesAgent(t *testing.T) {
	ctx := context.Background()
	// Created 48h ago, deleteTTL 24h → past its lifetime.
	agent := ttlAgent()
	agent.CreationTimestamp = metav1.NewTime(time.Now().Add(-48 * time.Hour))
	r := newTTLReconciler(t, agent)

	_, handled, err := r.applyTTL(ctx, agent, nil, "pvc", nil, dur(24*time.Hour))
	if err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true (delete fired)")
	}
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent", Namespace: "default"}, &komputerv1alpha1.KomputerAgent{}); err == nil {
		t.Error("agent still exists, want it deleted")
	}
}

func TestApplyTTL_NoTTLsIsANoOp(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent(withCreatedAgo(time.Hour), withLastActivityAgo(time.Hour))
	pod := agentPod("test-agent-pod", "default")
	r := newTTLReconciler(t, agent, pod)

	res, handled, err := r.applyTTL(ctx, agent, pod, "pvc", nil, nil)
	if err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if handled {
		t.Error("handled = true with no TTLs set, want false")
	}
	if res.RequeueAfter != 0 {
		t.Errorf("RequeueAfter = %v, want 0", res.RequeueAfter)
	}
	// The pod must survive — no TTL means no transition.
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent-pod", Namespace: "default"}, &corev1.Pod{}); err != nil {
		t.Errorf("pod was deleted without a TTL: %v", err)
	}
}

func TestApplyTTL_PublishesExpiryTimestampsWhenPending(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent(withCreatedAgo(time.Hour), withLastActivityAgo(0))
	pod := agentPod("test-agent-pod", "default")
	r := newTTLReconciler(t, agent, pod)

	res, handled, err := r.applyTTL(ctx, agent, pod, "pvc", dur(time.Hour), dur(240*time.Hour))
	if err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if handled {
		t.Fatal("handled = true, want false (nothing elapsed)")
	}
	// Should wake for the sooner of the two (the 1h sleep).
	if res.RequeueAfter <= 0 || res.RequeueAfter > time.Hour {
		t.Errorf("RequeueAfter = %v, want (0, 1h]", res.RequeueAfter)
	}

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.SleepExpiresAt == nil || got.Status.DeleteExpiresAt == nil {
		t.Errorf("expiry timestamps not persisted: sleep=%v delete=%v",
			got.Status.SleepExpiresAt, got.Status.DeleteExpiresAt)
	}
}

func TestApplyTTL_ClearsStaleExpiriesWhenTTLsRemoved(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent()
	stale := metav1.NewTime(time.Now().Add(time.Hour))
	agent.Status.SleepExpiresAt = &stale
	agent.Status.DeleteExpiresAt = &stale
	r := newTTLReconciler(t, agent)

	if _, handled, err := r.applyTTL(ctx, agent, nil, "pvc", nil, nil); err != nil || handled {
		t.Fatalf("applyTTL: handled=%v err=%v", handled, err)
	}

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.SleepExpiresAt != nil || got.Status.DeleteExpiresAt != nil {
		t.Error("expiry timestamps should be cleared once the TTLs are removed from the spec")
	}
}

func TestApplyTTL_InProgressTaskIsNotSlept(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent(withCreatedAgo(time.Hour), withLastActivityAgo(time.Hour), withTaskStatus(komputerv1alpha1.AgentTaskInProgress))
	pod := agentPod("test-agent-pod", "default")
	r := newTTLReconciler(t, agent, pod)

	_, handled, err := r.applyTTL(ctx, agent, pod, "pvc", dur(30*time.Minute), nil)
	if err != nil {
		t.Fatalf("applyTTL: %v", err)
	}
	if handled {
		t.Error("handled = true, want false — an in-flight task must never be slept")
	}
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent-pod", Namespace: "default"}, &corev1.Pod{}); err != nil {
		t.Errorf("pod of an in-progress agent was deleted: %v", err)
	}
}

// ─── resolveAgentTTLs (agent spec over template defaults) ────────────────────

func TestResolveAgentTTLs(t *testing.T) {
	ctx := context.Background()

	tplWithTTLs := func(name, ns string) *komputerv1alpha1.KomputerAgentTemplate {
		return &komputerv1alpha1.KomputerAgentTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
				SleepTTL:  dur(time.Hour),
				DeleteTTL: dur(48 * time.Hour),
			},
		}
	}

	t.Run("agent spec wins over the template", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "default"
		agent.Spec.SleepTTL = dur(5 * time.Minute)
		agent.Spec.DeleteTTL = dur(6 * time.Hour)
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).
			WithObjects(tplWithTTLs("default", "default")).Build()

		sleep, del := resolveAgentTTLs(ctx, c, agent)
		if sleep.Duration != 5*time.Minute {
			t.Errorf("sleepTTL = %v, want the agent's 5m", sleep.Duration)
		}
		if del.Duration != 6*time.Hour {
			t.Errorf("deleteTTL = %v, want the agent's 6h", del.Duration)
		}
	})

	t.Run("template supplies what the agent omits", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "default"
		agent.Spec.SleepTTL = dur(5 * time.Minute) // only sleep set
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).
			WithObjects(tplWithTTLs("default", "default")).Build()

		sleep, del := resolveAgentTTLs(ctx, c, agent)
		if sleep.Duration != 5*time.Minute {
			t.Errorf("sleepTTL = %v, want the agent's 5m", sleep.Duration)
		}
		if del == nil || del.Duration != 48*time.Hour {
			t.Errorf("deleteTTL = %v, want the template's 48h", del)
		}
	})

	t.Run("falls back to a cluster template", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "shared"
		clusterTpl := &komputerv1alpha1.KomputerAgentClusterTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "shared"},
			Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
				SleepTTL: dur(15 * time.Minute),
			},
		}
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(clusterTpl).Build()

		sleep, _ := resolveAgentTTLs(ctx, c, agent)
		if sleep == nil || sleep.Duration != 15*time.Minute {
			t.Errorf("sleepTTL = %v, want the cluster template's 15m", sleep)
		}
	})

	t.Run("a missing template is not an error", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.TemplateRef = "gone"
		agent.Spec.SleepTTL = dur(5 * time.Minute)
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

		sleep, del := resolveAgentTTLs(ctx, c, agent)
		if sleep.Duration != 5*time.Minute {
			t.Errorf("sleepTTL = %v, want the agent's 5m to survive", sleep.Duration)
		}
		if del != nil {
			t.Errorf("deleteTTL = %v, want nil", del)
		}
	})

	t.Run("skips the template lookup when both are set", func(t *testing.T) {
		agent := ttlAgent()
		agent.Spec.SleepTTL = dur(time.Minute)
		agent.Spec.DeleteTTL = dur(time.Hour)
		// Empty client: a lookup would fail, proving none was attempted.
		c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

		sleep, del := resolveAgentTTLs(ctx, c, agent)
		if sleep.Duration != time.Minute || del.Duration != time.Hour {
			t.Errorf("got %v/%v, want the agent's own values", sleep, del)
		}
	})
}
