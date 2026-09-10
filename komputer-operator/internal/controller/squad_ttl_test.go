package controller

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// squadMember builds a squad-managed agent: idle, awake, and with the idle clock
// already running (so sleepTTL is eligible to fire).
func squadMember(name string, idleFor time.Duration, mutate ...func(*komputerv1alpha1.KomputerAgent)) *komputerv1alpha1.KomputerAgent {
	last := metav1.NewTime(time.Now().Add(-idleFor))
	a := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(time.Now().Add(-idleFor)),
			Labels:            map[string]string{"komputer.ai/squad": "true"},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:          komputerv1alpha1.AgentPhaseRunning,
			TaskStatus:     komputerv1alpha1.AgentTaskComplete,
			Squad:          true,
			LastActivityAt: &last,
		},
	}
	for _, m := range mutate {
		m(a)
	}
	return a
}

func testSquad(memberNames ...string) *komputerv1alpha1.KomputerSquad {
	members := make([]komputerv1alpha1.KomputerSquadMember, 0, len(memberNames))
	for _, n := range memberNames {
		members = append(members, komputerv1alpha1.KomputerSquadMember{
			Ref: &komputerv1alpha1.KomputerSquadMemberRef{Name: n, Namespace: "default"},
		})
	}
	return &komputerv1alpha1.KomputerSquad{
		ObjectMeta: metav1.ObjectMeta{Name: "test-squad", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerSquadSpec{Members: members},
	}
}

func newSquadTTLReconciler(t *testing.T, objs ...client.Object) *KomputerSquadReconciler {
	t.Helper()
	s := newTestScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objs...).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}, &komputerv1alpha1.KomputerSquad{}).
		Build()
	return &KomputerSquadReconciler{Client: c, Scheme: s}
}

func TestApplyMemberTTLs_SleepsIdleMemberByPhaseOnly(t *testing.T) {
	ctx := context.Background()
	m := squadMember("member-a", time.Hour)
	m.Spec.SleepTTL = dur(30 * time.Minute)
	squad := testSquad("member-a")
	r := newSquadTTLReconciler(t, squad, m)

	survivors := r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{m})

	// A slept member is still a member — it must not be dropped from the pod build.
	if len(survivors) != 1 {
		t.Fatalf("survivors = %d, want 1 (sleep keeps membership)", len(survivors))
	}

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "member-a", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get member: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase = %q, want Sleeping", got.Status.Phase)
	}
	if got.Status.SleepExpiresAt != nil {
		t.Error("SleepExpiresAt should be cleared once asleep")
	}
	// The shared squad pod is torn down by reconcileSquadPod's all-sleeping check,
	// not here — so the member must still be marked squad-managed.
	if !got.Status.Squad {
		t.Error("Status.Squad was cleared; sleeping a member must not un-squad it")
	}
}

func TestApplyMemberTTLs_DeletesExpiredMemberAndDropsIt(t *testing.T) {
	ctx := context.Background()
	m := squadMember("member-a", 48*time.Hour)
	m.Spec.DeleteTTL = dur(24 * time.Hour)
	squad := testSquad("member-a")
	r := newSquadTTLReconciler(t, squad, m)

	survivors := r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{m})

	if len(survivors) != 0 {
		t.Errorf("survivors = %d, want 0 (deleted member must be dropped)", len(survivors))
	}
	if err := r.Get(ctx, types.NamespacedName{Name: "member-a", Namespace: "default"}, &komputerv1alpha1.KomputerAgent{}); err == nil {
		t.Error("member still exists, want it deleted")
	}
}

func TestApplyMemberTTLs_OnlyAffectsExpiredMembers(t *testing.T) {
	ctx := context.Background()
	expired := squadMember("member-a", time.Hour)
	expired.Spec.SleepTTL = dur(30 * time.Minute)
	fresh := squadMember("member-b", time.Minute)
	fresh.Spec.SleepTTL = dur(30 * time.Minute)
	squad := testSquad("member-a", "member-b")
	r := newSquadTTLReconciler(t, squad, expired, fresh)

	survivors := r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{expired, fresh})
	if len(survivors) != 2 {
		t.Fatalf("survivors = %d, want 2", len(survivors))
	}

	var a, b komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "member-a", Namespace: "default"}, &a); err != nil {
		t.Fatalf("get member-a: %v", err)
	}
	if err := r.Get(ctx, types.NamespacedName{Name: "member-b", Namespace: "default"}, &b); err != nil {
		t.Fatalf("get member-b: %v", err)
	}
	if a.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("member-a Phase = %q, want Sleeping", a.Status.Phase)
	}
	if b.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		t.Error("member-b was slept, but it is still within its sleepTTL")
	}
	if b.Status.SleepExpiresAt == nil {
		t.Error("member-b should publish a pending sleepExpiresAt")
	}
}

func TestApplyMemberTTLs_NeverSleepsAMemberMidTask(t *testing.T) {
	ctx := context.Background()
	m := squadMember("member-a", time.Hour, func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.TaskStatus = komputerv1alpha1.AgentTaskInProgress
	})
	m.Spec.SleepTTL = dur(30 * time.Minute)
	squad := testSquad("member-a")
	r := newSquadTTLReconciler(t, squad, m)

	r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{m})

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "member-a", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get member: %v", err)
	}
	if got.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		t.Error("a member running a task was slept")
	}
}

func TestApplyMemberTTLs_NeverSleepsAMemberThatHasNotRunATask(t *testing.T) {
	ctx := context.Background()
	m := squadMember("member-a", 48*time.Hour, func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.LastActivityAt = nil // never started a task
		a.Status.TaskStatus = ""
		a.Status.Phase = komputerv1alpha1.AgentPhasePending
	})
	m.Spec.SleepTTL = dur(30 * time.Minute)
	squad := testSquad("member-a")
	r := newSquadTTLReconciler(t, squad, m)

	r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{m})

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "member-a", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get member: %v", err)
	}
	if got.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		t.Error("a member that never ran a task was slept")
	}
}

func TestApplyMemberTTLs_InheritsTemplateDefaults(t *testing.T) {
	ctx := context.Background()
	m := squadMember("member-a", time.Hour)
	m.Spec.TemplateRef = "default" // no TTLs of its own
	tpl := &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentTemplateSpec{SleepTTL: dur(30 * time.Minute)},
	}
	squad := testSquad("member-a")
	r := newSquadTTLReconciler(t, squad, m, tpl)

	r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{m})

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "member-a", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get member: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase = %q, want Sleeping from the template's sleepTTL", got.Status.Phase)
	}
}

func TestApplyMemberTTLs_NoTTLsIsANoOp(t *testing.T) {
	ctx := context.Background()
	m := squadMember("member-a", 99*time.Hour) // very idle, but no TTLs
	squad := testSquad("member-a")
	r := newSquadTTLReconciler(t, squad, m)

	survivors := r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{m})
	if len(survivors) != 1 {
		t.Fatalf("survivors = %d, want 1", len(survivors))
	}

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "member-a", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get member: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseRunning {
		t.Errorf("Phase = %q, want Running untouched", got.Status.Phase)
	}
}

func TestApplyMemberTTLs_ClearsStaleExpiriesWhenTTLsRemoved(t *testing.T) {
	ctx := context.Background()
	stale := metav1.NewTime(time.Now().Add(time.Hour))
	m := squadMember("member-a", time.Minute, func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.SleepExpiresAt = &stale
		a.Status.DeleteExpiresAt = &stale
	})
	squad := testSquad("member-a")
	r := newSquadTTLReconciler(t, squad, m)

	r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{m})

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "member-a", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get member: %v", err)
	}
	if got.Status.SleepExpiresAt != nil || got.Status.DeleteExpiresAt != nil {
		t.Error("stale expiries should be cleared once the TTLs are gone from the spec")
	}
}

// Deleting every member must leave the squad empty rather than half-built, so the
// next reconcile prunes the refs and routes into handleEmptySquad.
func TestApplyMemberTTLs_AllMembersExpiredLeavesNoSurvivors(t *testing.T) {
	ctx := context.Background()
	a := squadMember("member-a", 48*time.Hour)
	a.Spec.DeleteTTL = dur(24 * time.Hour)
	b := squadMember("member-b", 48*time.Hour)
	b.Spec.DeleteTTL = dur(24 * time.Hour)
	squad := testSquad("member-a", "member-b")
	r := newSquadTTLReconciler(t, squad, a, b)

	survivors := r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{a, b})
	if len(survivors) != 0 {
		t.Errorf("survivors = %d, want 0", len(survivors))
	}
}
