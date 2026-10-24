package main

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func TestCreateSchedule_AgentNaming_StoredAndReturned(t *testing.T) {
	k8s := newFakeK8s(t)
	w := postCreateSchedule(t, k8s, `{"name":"s1","schedule":"0 * * * *","instructions":"go","agentNaming":"Date","agent":{}}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp ScheduleResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.AgentNaming != "Date" {
		t.Errorf("response agentNaming = %q, want Date", resp.AgentNaming)
	}
	s, _ := k8s.GetSchedule(context.Background(), "default", "s1")
	if s.Spec.AgentNaming != komputerv1alpha1.ScheduleAgentNamingDate {
		t.Errorf("spec agentNaming = %q, want Date", s.Spec.AgentNaming)
	}
}

func TestCreateSchedule_AgentNaming_Rejected(t *testing.T) {
	for name, body := range map[string]string{
		"unknown value":          `{"name":"s1","schedule":"0 * * * *","instructions":"go","agentNaming":"Weekly","agent":{}}`,
		"per-run with agentName": `{"name":"s1","schedule":"0 * * * *","instructions":"go","agentNaming":"Random","agentName":"a1"}`,
	} {
		if w := postCreateSchedule(t, newFakeK8s(t), body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400; body = %s", name, w.Code, w.Body.String())
		}
	}
}

// Each run's agent runs once, so it can archive without the S3 session store.
func TestCreateSchedule_PerRunNaming_AllowsArchiveWithoutS3(t *testing.T) {
	body := `{"name":"s1","schedule":"0 * * * *","instructions":"go","agentNaming":"Date","agent":{"lifecycle":"Archive"}}`
	if w := postCreateSchedule(t, newFakeK8s(t, s3Config(false)), body); w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func namingTestSchedule(naming komputerv1alpha1.ScheduleAgentNaming, lifecycle komputerv1alpha1.AgentLifecycle) *komputerv1alpha1.KomputerSchedule {
	return &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "default", UID: "s1-uid"},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Schedule: "0 * * * *", Instructions: "go", AgentNaming: naming,
			Agent: &komputerv1alpha1.ScheduleAgentSpec{AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: lifecycle}},
		},
	}
}

// Switching an Archive schedule back to Fixed would reuse one archived agent.
func TestPatchSchedule_BackToFixed_WithArchiveWithoutS3_Rejected(t *testing.T) {
	k8s := newFakeK8s(t, namingTestSchedule(komputerv1alpha1.ScheduleAgentNamingDate, komputerv1alpha1.AgentLifecycleArchive), s3Config(false))
	if w := patchScheduleReq(t, k8s, "s1", `{"agentNaming":"Fixed"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestPatchSchedule_AgentNaming_Updated(t *testing.T) {
	k8s := newFakeK8s(t, namingTestSchedule("", komputerv1alpha1.AgentLifecycleSleep))
	if w := patchScheduleReq(t, k8s, "s1", `{"agentNaming":"Random"}`); w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	s, _ := k8s.GetSchedule(context.Background(), "default", "s1")
	if s.Spec.AgentNaming != komputerv1alpha1.ScheduleAgentNamingRandom {
		t.Errorf("spec agentNaming = %q, want Random", s.Spec.AgentNaming)
	}
}

// A manual trigger of a per-run schedule creates a new agent owned by the schedule.
func TestTriggerScheduleNow_PerRun_CreatesANewAgent(t *testing.T) {
	sched := namingTestSchedule(komputerv1alpha1.ScheduleAgentNamingRandom, komputerv1alpha1.AgentLifecycleArchive)
	sched.Status.AgentName = "s1-prev1"
	prev := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "s1-prev1", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseArchived, TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	k8s := newFakeK8sWithStatusSubresource(t, sched, prev, s3Config(false))
	name, code, err := triggerScheduleNow(context.Background(), k8s, "default", "s1")
	if err != nil {
		t.Fatalf("triggerScheduleNow: %d %v", code, err)
	}
	if !regexp.MustCompile(`^s1-[a-z0-9]{5}$`).MatchString(name) || name == "s1-prev1" {
		t.Fatalf("agent = %q, want a new s1-<5 chars>", name)
	}
	a, err := k8s.GetAgent(context.Background(), "default", name)
	if err != nil {
		t.Fatal(err)
	}
	if a.Labels["komputer.ai/schedule"] != "s1" || len(a.OwnerReferences) != 1 || a.OwnerReferences[0].Name != "s1" {
		t.Errorf("agent labels %v, owners %v; want the schedule label and owner", a.Labels, a.OwnerReferences)
	}
	if s, _ := k8s.GetSchedule(context.Background(), "default", "s1"); s.Status.AgentName != name {
		t.Errorf("schedule agent = %q, want %q", s.Status.AgentName, name)
	}
}

func TestTriggerScheduleNow_PerRun_PreviousStillWorking_409(t *testing.T) {
	sched := namingTestSchedule(komputerv1alpha1.ScheduleAgentNamingDate, komputerv1alpha1.AgentLifecycleSleep)
	sched.Status.AgentName = "s1-prev1"
	prev := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "s1-prev1", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseRunning, TaskStatus: komputerv1alpha1.AgentTaskInProgress},
	}
	k8s := newFakeK8sWithStatusSubresource(t, sched, prev)
	if _, code, err := triggerScheduleNow(context.Background(), k8s, "default", "s1"); code != http.StatusConflict {
		t.Fatalf("code = %d (%v), want 409", code, err)
	}
}
