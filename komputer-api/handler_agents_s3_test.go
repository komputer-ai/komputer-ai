package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// With the S3 session store on, an archived agent behaves like a sleeping one:
// every control works and takes effect the next time it runs.

func archivedS3Agent() *komputerv1alpha1.KomputerAgent {
	a := archivedAgent()
	a.Spec.Lifecycle = komputerv1alpha1.AgentLifecycleArchive
	return a
}

func agentPhase(t *testing.T, k8s *K8sClient, name string) komputerv1alpha1.KomputerAgentPhase {
	t.Helper()
	a := &komputerv1alpha1.KomputerAgent{}
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: "default"}, a); err != nil {
		t.Fatal(err)
	}
	return a.Status.Phase
}

func TestS3On_TaskToArchivedAgentWakesIt(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archivedS3Agent(), s3Config(true))
	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"run again"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := agentPhase(t, k8s, "a1"); got != komputerv1alpha1.AgentPhasePending {
		t.Errorf("phase = %q, want Pending", got)
	}
	a, _ := k8s.GetAgent(context.Background(), "default", "a1")
	if a.Spec.Instructions != "run again" {
		t.Errorf("instructions = %q, want patched", a.Spec.Instructions)
	}
}

func TestS3Off_TaskToArchivedAgentStill409(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archivedS3Agent(), s3Config(false))
	if w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"again"}`); w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
}

func TestS3On_PatchArchivedAgentAnyField(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archivedS3Agent(), s3Config(true))
	if w := patchAgentReq(t, k8s, "a1", `{"model":"claude-sonnet-4-6"}`); w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestS3On_IdleWakeArchived(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archivedS3Agent(), s3Config(true))
	if w := postWakeAgent(t, k8s, "a1"); w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := agentPhase(t, k8s, "a1"); got != komputerv1alpha1.AgentPhasePending {
		t.Errorf("phase = %q, want Pending", got)
	}
}

func TestS3On_SleepArchivedIsNoop(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archivedS3Agent(), s3Config(true))
	if w := postSleepAgent(t, k8s, "a1"); w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := agentPhase(t, k8s, "a1"); got != komputerv1alpha1.AgentPhaseArchived {
		t.Errorf("phase = %q, want Archived", got)
	}
}

func TestS3On_EphemeralWithSleepLifecycleAccepted(t *testing.T) {
	k8s := newFakeK8s(t, s3Config(true))
	w := postCreateAgent(t, k8s, `{"name":"e1","instructions":"go","lifecycle":"Sleep","storage":{"ephemeral":true}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestS3Off_EphemeralWithSleepLifecycleRejected(t *testing.T) {
	k8s := newFakeK8s(t, s3Config(false))
	w := postCreateAgent(t, k8s, `{"name":"e1","instructions":"go","lifecycle":"Sleep","storage":{"ephemeral":true}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestS3On_SleepNowEphemeralAccepted(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "e1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Storage: &komputerv1alpha1.StorageSpec{Ephemeral: boolPtrAPI(true)}},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseRunning, TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent, s3Config(true))
	if w := postSleepAgent(t, k8s, "e1"); w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestS3On_ScheduleWithArchiveAccepted(t *testing.T) {
	body := `{"name":"s1","schedule":"* * * * *","instructions":"go","agent":{"lifecycle":"Archive"}}`
	if w := postCreateSchedule(t, newFakeK8s(t, s3Config(true)), body); w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestS3Off_ScheduleWithArchiveRejected(t *testing.T) {
	body := `{"name":"s1","schedule":"* * * * *","instructions":"go","agent":{"lifecycle":"Archive"}}`
	if w := postCreateSchedule(t, newFakeK8s(t, s3Config(false)), body); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestS3On_TriggerScheduleWakesArchivedAgent(t *testing.T) {
	sched := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerScheduleSpec{Schedule: "* * * * *", Instructions: "scheduled run", AgentName: "a1"},
	}
	k8s := newFakeK8sWithStatusSubresource(t, archivedS3Agent(), sched, s3Config(true))
	if _, code, err := triggerScheduleNow(context.Background(), k8s, "default", "s1"); err != nil {
		t.Fatalf("triggerScheduleNow: %d %v", code, err)
	}
	if got := agentPhase(t, k8s, "a1"); got != komputerv1alpha1.AgentPhasePending {
		t.Errorf("phase = %q, want Pending", got)
	}
}

func TestS3On_TeamUpAdoptsArchivedAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	k8s := newFakeK8s(t, archivedS3Agent(), s3Config(true))
	if msg, blocked := validateMemberRefForAdoption(c, k8s, "default", "a1"); blocked {
		t.Fatalf("archived agent should be adoptable with S3 on: %s", msg)
	}
}

func TestListAgents_HideArchived(t *testing.T) {
	running := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseRunning},
	}
	k8s := newFakeK8s(t, archivedS3Agent(), running)

	names := func(query string) []string {
		w := serveListAgents(t, k8s, "?limit=50&"+query)
		var resp AgentListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, a := range resp.Agents {
			out = append(out, a.Name)
		}
		return out
	}
	if got := names("hideArchived=true"); len(got) != 1 || got[0] != "r1" {
		t.Errorf("hideArchived=true: got %v, want [r1]", got)
	}
	if got := names("hideArchived=true&status=Archived"); len(got) != 1 || got[0] != "a1" {
		t.Errorf("status=Archived wins over hideArchived: got %v, want [a1]", got)
	}
	if got := names(""); len(got) != 2 {
		t.Errorf("default: got %v, want both", got)
	}
}

func TestDeleteAgent_ForceRemovesSessionStoreFinalizer(t *testing.T) {
	agent := archivedS3Agent()
	agent.Finalizers = []string{komputerv1alpha1.FinalizerSessionStore}
	k8s := newFakeK8s(t, agent)
	if err := k8s.ForceDeleteAgent(context.Background(), "default", "a1"); err != nil {
		t.Fatal(err)
	}
	err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &komputerv1alpha1.KomputerAgent{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("agent should be gone after a force delete, got err=%v", err)
	}
}
