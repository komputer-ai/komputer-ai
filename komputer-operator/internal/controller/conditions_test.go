package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// assertCondition checks that condType is set with the given status and reason.
func assertCondition(t *testing.T, a *komputerv1alpha1.KomputerAgent, condType string, status metav1.ConditionStatus, reason string) {
	t.Helper()
	c := meta.FindStatusCondition(a.Status.Conditions, condType)
	if c == nil {
		t.Fatalf("condition %s missing; conditions = %+v", condType, a.Status.Conditions)
	}
	if c.Status != status || c.Reason != reason {
		t.Errorf("condition %s = %s/%s, want %s/%s", condType, c.Status, c.Reason, status, reason)
	}
}

// assertEvent checks that the fake recorder got an event with this reason.
func assertEvent(t *testing.T, rec *record.FakeRecorder, reason string) {
	t.Helper()
	for {
		select {
		case e := <-rec.Events:
			if strings.HasPrefix(e, "Normal "+reason+" ") {
				return
			}
		default:
			t.Errorf("no Normal %s event recorded", reason)
			return
		}
	}
}

func withRecorder(r *KomputerAgentReconciler) *record.FakeRecorder {
	rec := record.NewFakeRecorder(20)
	r.Recorder = rec
	return rec
}

func TestConditions_SleepTTL(t *testing.T) {
	agent := ttlAgent(withCreatedAgo(time.Hour), withLastActivityAgo(time.Hour))
	pod := agentPod("test-agent-pod", "default")
	r := newTTLReconciler(t, agent, pod)
	rec := withRecorder(r)

	if _, _, err := r.applyTTL(context.Background(), agent, pod, "pvc", dur(30*time.Minute), nil, nil, nil, false); err != nil {
		t.Fatal(err)
	}
	assertCondition(t, getAgentT(t, r.Client, "test-agent"), komputerv1alpha1.ConditionAsleep, metav1.ConditionTrue, komputerv1alpha1.ReasonSleepTTL)
	assertEvent(t, rec, komputerv1alpha1.ReasonSleepTTL)
}

// Archiving a sleeping agent records the archive and resolves Asleep.
func TestConditions_ArchiveTTLFromSleep(t *testing.T) {
	agent, _, pvc, _ := archiveTTLAgent(withPhase(komputerv1alpha1.AgentPhaseSleeping))
	agent.Status.SetCondition(komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonSleepTTL, "slept")
	r := newTTLReconciler(t, agent, pvc)
	rec := withRecorder(r)

	if _, _, err := r.applyTTL(context.Background(), agent, nil, "my-agent-pvc", nil, nil, dur(30*time.Minute), nil, false); err != nil {
		t.Fatal(err)
	}
	got := getAgentT(t, r.Client, "my-agent")
	assertCondition(t, got, komputerv1alpha1.ConditionArchived, metav1.ConditionTrue, komputerv1alpha1.ReasonArchiveTTL)
	assertCondition(t, got, komputerv1alpha1.ConditionAsleep, metav1.ConditionFalse, komputerv1alpha1.ReasonArchived)
	assertEvent(t, rec, komputerv1alpha1.ReasonArchiveTTL)
}

// deleteTTL leaves no status behind, so the Event is the only trace.
func TestConditions_DeleteTTLEmitsEvent(t *testing.T) {
	agent := ttlAgent(withCreatedAgo(48 * time.Hour))
	r := newTTLReconciler(t, agent)
	rec := withRecorder(r)

	if _, _, err := r.applyTTL(context.Background(), agent, nil, "pvc", nil, dur(24*time.Hour), nil, nil, false); err != nil {
		t.Fatal(err)
	}
	assertEvent(t, rec, komputerv1alpha1.ReasonDeleteTTL)
}

func TestConditions_TaskTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer srv.Close()
	t.Setenv("KOMPUTER_API_URL", srv.URL)

	agent := ttlAgent(withTaskStatus(komputerv1alpha1.AgentTaskInProgress), withPodName("test-agent-pod"))
	started := metav1.NewTime(time.Now().Add(-time.Hour))
	agent.Status.TaskStartedAt = &started
	r := newTTLReconciler(t, agent)
	rec := withRecorder(r)

	if _, _, err := r.applyTTL(context.Background(), agent, nil, "pvc", nil, nil, nil, dur(30*time.Minute), false); err != nil {
		t.Fatal(err)
	}
	assertCondition(t, getAgentT(t, r.Client, "test-agent"), komputerv1alpha1.ConditionTaskTimedOut, metav1.ConditionTrue, komputerv1alpha1.ReasonTaskTimeout)
	assertEvent(t, rec, komputerv1alpha1.ReasonTaskTimeout)
}

func TestConditions_LifecycleAndRequests(t *testing.T) {
	sleepLifecycle := func() *komputerv1alpha1.KomputerAgent {
		a := archiveAgent(komputerv1alpha1.AgentTaskComplete)
		a.Spec.Lifecycle = komputerv1alpha1.AgentLifecycleSleep
		return a
	}
	sleepRequested := func() *komputerv1alpha1.KomputerAgent {
		a := archiveAgent(komputerv1alpha1.AgentTaskComplete)
		a.Spec.Lifecycle = ""
		a.Annotations = map[string]string{komputerv1alpha1.AnnotationSleepRequested: "true"}
		return a
	}
	archiveRequested := func() *komputerv1alpha1.KomputerAgent {
		a := archiveAgent(komputerv1alpha1.AgentTaskComplete)
		a.Spec.Lifecycle = ""
		a.Annotations = map[string]string{komputerv1alpha1.AnnotationArchiveRequested: "true"}
		return a
	}
	for _, tc := range []struct {
		name     string
		agent    func() *komputerv1alpha1.KomputerAgent
		condType string
		reason   string
	}{
		{"lifecycle Sleep", sleepLifecycle, komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonLifecycleSleep},
		{"sleep requested", sleepRequested, komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonSleepRequested},
		{"lifecycle Archive", func() *komputerv1alpha1.KomputerAgent { return archiveAgent(komputerv1alpha1.AgentTaskComplete) },
			komputerv1alpha1.ConditionArchived, komputerv1alpha1.ReasonLifecycleArchive},
		{"archive requested", archiveRequested, komputerv1alpha1.ConditionArchived, komputerv1alpha1.ReasonArchiveRequested},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod, pvc, svc := archiveTestObjects()
			agent := tc.agent()
			r := newTTLReconciler(t, agent, pod, pvc, svc)
			rec := withRecorder(r)

			if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, nil, false); err != nil {
				t.Fatal(err)
			}
			assertCondition(t, getAgentT(t, r.Client, "my-agent"), tc.condType, metav1.ConditionTrue, tc.reason)
			assertEvent(t, rec, tc.reason)
		})
	}
}

func TestConditions_SquadMemberSleepTTL(t *testing.T) {
	ctx := context.Background()
	m := squadMember("member-a", time.Hour)
	m.Spec.SleepTTL = dur(30 * time.Minute)
	squad := testSquad("member-a")
	r := newSquadTTLReconciler(t, squad, m)
	rec := record.NewFakeRecorder(10)
	r.Recorder = rec

	r.applyMemberTTLs(ctx, squad, []*komputerv1alpha1.KomputerAgent{m})

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "member-a", Namespace: "default"}, &got); err != nil {
		t.Fatal(err)
	}
	assertCondition(t, &got, komputerv1alpha1.ConditionAsleep, metav1.ConditionTrue, komputerv1alpha1.ReasonSleepTTL)
	assertEvent(t, rec, komputerv1alpha1.ReasonSleepTTL)
}

// ResolveCondition only flips a True condition; it never invents a False one.
func TestResolveConditionOnlyFlipsTrue(t *testing.T) {
	var s komputerv1alpha1.KomputerAgentStatus
	s.ResolveCondition(komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonWoken, "")
	if len(s.Conditions) != 0 {
		t.Fatalf("conditions = %+v, want none", s.Conditions)
	}
	s.SetCondition(komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonSleepTTL, "")
	s.ResolveCondition(komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonWoken, "")
	if !meta.IsStatusConditionFalse(s.Conditions, komputerv1alpha1.ConditionAsleep) {
		t.Errorf("Asleep = %+v, want False", s.Conditions)
	}
}

// A schedule run wakes a sleeping agent itself, so it resolves Asleep too.
func TestConditions_ScheduleWakeResolvesAsleep(t *testing.T) {
	scheme := newTestScheme(t)
	past := metav1.NewTime(time.Now().Add(-time.Minute))
	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerScheduleSpec{Schedule: "0 0 * * *", Instructions: "go", AgentName: "my-agent"},
		Status:     komputerv1alpha1.KomputerScheduleStatus{Phase: komputerv1alpha1.SchedulePhaseActive, NextRunTime: &past},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseSleeping, TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	agent.Status.SetCondition(komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonSleepTTL, "slept")
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(schedule, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerSchedule{}, &komputerv1alpha1.KomputerAgent{}).Build()
	r := &KomputerScheduleReconciler{Client: c, Scheme: scheme}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "sched1", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
	assertCondition(t, getAgentT(t, c, "my-agent"), komputerv1alpha1.ConditionAsleep, metav1.ConditionFalse, komputerv1alpha1.ReasonWoken)
}

// While a slept pod terminates, its terminal phase briefly flips the agent to
// Failed and the sleep re-runs. The condition is already recorded, so the re-run
// must not emit another event (a burst would also trip client-go's per-object
// event spam filter and drop the next real transition's event).
func TestConditions_RepeatedSleepEmitsOneEvent(t *testing.T) {
	agent := ttlAgent(withCreatedAgo(time.Hour), withLastActivityAgo(time.Hour), withPhase(komputerv1alpha1.AgentPhaseFailed))
	agent.Status.SetCondition(komputerv1alpha1.ConditionAsleep, komputerv1alpha1.ReasonSleepTTL,
		"Sleeping — idle past sleepTTL, workspace preserved. Send a new task to wake up.")
	r := newTTLReconciler(t, agent)
	rec := withRecorder(r)

	if _, _, err := r.applyTTL(context.Background(), agent, nil, "pvc", dur(30*time.Minute), nil, nil, nil, false); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-rec.Events:
		t.Errorf("unexpected event on a repeated sleep: %s", e)
	default:
	}
}
