package controller

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// 2026-10-24 09:00 UTC = 12:00 in Asia/Jerusalem (UTC+3 in October).
var namingNow = time.Date(2026, 10, 24, 9, 0, 30, 0, time.UTC)

func namingSchedule(naming komputerv1alpha1.ScheduleAgentNaming) *komputerv1alpha1.KomputerSchedule {
	past := metav1.NewTime(namingNow.Truncate(time.Minute)) // the run due at 09:00
	return &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "daily-report", Namespace: "default", UID: "sched-uid"},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Schedule: "0 * * * *", Instructions: "write the report", AgentNaming: naming,
			Agent: &komputerv1alpha1.ScheduleAgentSpec{AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Lifecycle: komputerv1alpha1.AgentLifecycleArchive,
			}},
		},
		Status: komputerv1alpha1.KomputerScheduleStatus{Phase: komputerv1alpha1.SchedulePhaseActive, NextRunTime: &past},
	}
}

func namingReconciler(t *testing.T, objs ...client.Object) (*KomputerScheduleReconciler, client.Client) {
	t.Helper()
	scheme := newTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).
		WithStatusSubresource(&komputerv1alpha1.KomputerSchedule{}, &komputerv1alpha1.KomputerAgent{}).Build()
	return &KomputerScheduleReconciler{Client: c, Scheme: scheme, Now: func() time.Time { return namingNow }}, c
}

func reconcileSchedule(t *testing.T, r *KomputerScheduleReconciler) {
	t.Helper()
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "daily-report", Namespace: "default"}}); err != nil {
		t.Fatal(err)
	}
}

func scheduleAgents(t *testing.T, c client.Client) []komputerv1alpha1.KomputerAgent {
	t.Helper()
	var list komputerv1alpha1.KomputerAgentList
	if err := c.List(context.Background(), &list, client.InNamespace("default")); err != nil {
		t.Fatal(err)
	}
	return list.Items
}

func getSchedule(t *testing.T, c client.Client) *komputerv1alpha1.KomputerSchedule {
	t.Helper()
	var s komputerv1alpha1.KomputerSchedule
	if err := c.Get(context.Background(), types.NamespacedName{Name: "daily-report", Namespace: "default"}, &s); err != nil {
		t.Fatal(err)
	}
	return &s
}

// Each run gets its own agent, named after the run; the schedule tracks it.
func TestScheduleReconcile_DateNaming_CreatesANewAgentPerRun(t *testing.T) {
	r, c := namingReconciler(t, namingSchedule(komputerv1alpha1.ScheduleAgentNamingDate))
	reconcileSchedule(t, r)

	agents := scheduleAgents(t, c)
	if len(agents) != 1 || agents[0].Name != "daily-report-20261024-0900" {
		t.Fatalf("agents = %v, want one named daily-report-20261024-0900", agentNames(agents))
	}
	a := agents[0]
	if a.Spec.Lifecycle != komputerv1alpha1.AgentLifecycleArchive || a.Spec.Instructions != "write the report" || a.Labels["komputer.ai/schedule"] != "daily-report" {
		t.Errorf("agent = lifecycle %q, instructions %q, labels %v", a.Spec.Lifecycle, a.Spec.Instructions, a.Labels)
	}
	s := getSchedule(t, c)
	if s.Status.AgentName != a.Name || s.Status.LastRunStatus != "InProgress" {
		t.Errorf("schedule status = agent %q, last run %q", s.Status.AgentName, s.Status.LastRunStatus)
	}
}

// The next run doesn't reuse the previous run's agent.
func TestScheduleReconcile_RandomNaming_NextRunGetsANewAgent(t *testing.T) {
	sched := namingSchedule(komputerv1alpha1.ScheduleAgentNamingRandom)
	sched.Status.AgentName = "daily-report-prev1"
	sched.Status.LastRunStatus = "Success"
	prev := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "daily-report-prev1", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseArchived, TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	r, c := namingReconciler(t, sched, prev)
	reconcileSchedule(t, r)

	agents := scheduleAgents(t, c)
	if len(agents) != 2 {
		t.Fatalf("agents = %v, want the previous one plus a new one", agentNames(agents))
	}
	s := getSchedule(t, c)
	if s.Status.AgentName == "daily-report-prev1" || !strings.HasPrefix(s.Status.AgentName, "daily-report-") {
		t.Errorf("run agent = %q, want a new daily-report-<suffix>", s.Status.AgentName)
	}
	if got := getAgentT(t, c, "daily-report-prev1"); got.Status.Phase != komputerv1alpha1.AgentPhaseArchived {
		t.Errorf("previous run's agent was touched: phase %q", got.Status.Phase)
	}
}

// Two runs in the same minute (e.g. a manual trigger) still get distinct agents.
func TestScheduleReconcile_DateNaming_NameTaken_AddsASuffix(t *testing.T) {
	taken := &komputerv1alpha1.KomputerAgent{ObjectMeta: metav1.ObjectMeta{Name: "daily-report-20261024-0900", Namespace: "default"}}
	r, c := namingReconciler(t, namingSchedule(komputerv1alpha1.ScheduleAgentNamingDate), taken)
	reconcileSchedule(t, r)

	s := getSchedule(t, c)
	if !regexp.MustCompile(`^daily-report-20261024-0900-[a-z0-9]{4}$`).MatchString(s.Status.AgentName) {
		t.Errorf("run agent = %q, want daily-report-20261024-0900-<4 chars>", s.Status.AgentName)
	}
}

// A reconcile retried after the run's agent was created (e.g. the status write
// conflicted) reuses that agent instead of starting the task a second time.
func TestScheduleReconcile_PerRun_RetryReusesTheRunsAgent(t *testing.T) {
	sched := namingSchedule(komputerv1alpha1.ScheduleAgentNamingDate)
	r, c := namingReconciler(t, sched)
	created := buildScheduledAgent(sched, "daily-report-20261024-0900")
	created.Annotations = map[string]string{scheduleRunAnnotation: sched.Status.NextRunTime.UTC().Format(time.RFC3339)}
	if err := ctrl.SetControllerReference(sched, created, r.Scheme); err != nil {
		t.Fatal(err)
	}
	if err := c.Create(context.Background(), created); err != nil {
		t.Fatal(err)
	}
	reconcileSchedule(t, r)

	if agents := scheduleAgents(t, c); len(agents) != 1 {
		t.Fatalf("agents = %v, want only the run's existing agent", agentNames(agents))
	}
	if s := getSchedule(t, c); s.Status.AgentName != "daily-report-20261024-0900" || s.Status.LastRunStatus != "InProgress" {
		t.Errorf("status = agent %q, last run %q", s.Status.AgentName, s.Status.LastRunStatus)
	}
}

// Fixed (the default) keeps reusing one agent named <schedule>-agent.
func TestScheduleReconcile_FixedNaming_Unchanged(t *testing.T) {
	r, c := namingReconciler(t, namingSchedule(""))
	reconcileSchedule(t, r)
	if s := getSchedule(t, c); s.Status.AgentName != "daily-report-agent" {
		t.Errorf("run agent = %q, want daily-report-agent", s.Status.AgentName)
	}
}

func agentNames(agents []komputerv1alpha1.KomputerAgent) []string {
	var out []string
	for _, a := range agents {
		out = append(out, a.Name)
	}
	return out
}

// A run is skipped while the previous run's agent is still working.
func TestScheduleReconcile_PerRun_PreviousStillWorking_Skips(t *testing.T) {
	sched := namingSchedule(komputerv1alpha1.ScheduleAgentNamingDate)
	sched.Status.AgentName = "daily-report-20261024-0800"
	sched.Status.LastRunStatus = "Success"
	prev := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "daily-report-20261024-0800", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseRunning, TaskStatus: komputerv1alpha1.AgentTaskInProgress},
	}
	r, c := namingReconciler(t, sched, prev)
	reconcileSchedule(t, r)

	if agents := scheduleAgents(t, c); len(agents) != 1 {
		t.Fatalf("agents = %v, want no new agent while the previous run works", agentNames(agents))
	}
	s := getSchedule(t, c)
	if s.Status.AgentName != prev.Name || s.Status.Message != "Skipped: previous run still in progress" || !s.Status.NextRunTime.After(namingNow) {
		t.Errorf("status = agent %q, message %q, next %v", s.Status.AgentName, s.Status.Message, s.Status.NextRunTime)
	}
}
