package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func postSleepAgent(t *testing.T, k8s *K8sClient, name string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/agents/:name/sleep", sleepAgent(k8s))
	req := httptest.NewRequest(http.MethodPost, "/agents/"+name+"/sleep", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestSleepAgentEndpoint_RunningIdleAgent_SetsSleepRequest verifies the happy
// path: 200, the agent's spec.lifecycle is left untouched (this is NOT the
// same as PATCH .../agents/{name} {lifecycle:"Sleep"}), and the request has
// been recorded for the operator to act on.
func TestSleepAgentEndpoint_RunningIdleAgent_SetsSleepRequest(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "done already"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postSleepAgent(t, k8s, "a1")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp AgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Lifecycle != "" {
		t.Errorf("Lifecycle = %q, want unchanged (empty) — sleep now must not set a policy", resp.Lifecycle)
	}

	got, err := k8s.GetAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got.Annotations[komputerv1alpha1.AnnotationSleepRequested] != "true" {
		t.Errorf("sleep-requested annotation not set on agent")
	}
}

// TestSleepAgentEndpoint_AlreadySleeping_Idempotent verifies sleeping an
// already-sleeping agent returns 200 rather than an error.
func TestSleepAgentEndpoint_AlreadySleeping_Idempotent(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseSleeping},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postSleepAgent(t, k8s, "a1")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

// TestSleepAgentEndpoint_Busy_Returns409 verifies both busy task statuses map to
// 409 with a message telling the caller to cancel first.
func TestSleepAgentEndpoint_Busy_Returns409(t *testing.T) {
	for _, status := range []komputerv1alpha1.AgentTaskStatus{
		komputerv1alpha1.AgentTaskInProgress,
		komputerv1alpha1.AgentTaskCompacting,
	} {
		t.Run(string(status), func(t *testing.T) {
			agent := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
				Status: komputerv1alpha1.KomputerAgentStatus{
					Phase:      komputerv1alpha1.AgentPhaseRunning,
					TaskStatus: status,
				},
			}
			k8s := newFakeK8sWithStatusSubresource(t, agent)

			w := postSleepAgent(t, k8s, "a1")
			if w.Code != http.StatusConflict {
				t.Fatalf("status = %d, want 409; body = %s", w.Code, w.Body.String())
			}
		})
	}
}

// TestSleepAgentEndpoint_Ephemeral_Returns400 verifies an ephemeral-workspace
// agent is rejected with 400 rather than silently wiping its workspace.
func TestSleepAgentEndpoint_Ephemeral_Returns400(t *testing.T) {
	ephemeral := true
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Storage: &komputerv1alpha1.StorageSpec{Ephemeral: &ephemeral},
			},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postSleepAgent(t, k8s, "a1")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestSleepAgentEndpoint_NotFound verifies a missing agent returns 404.
func TestSleepAgentEndpoint_NotFound(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t)

	w := postSleepAgent(t, k8s, "nope")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", w.Code, w.Body.String())
	}
}

// TestSleepAgentEndpoint_SquadMember_PatchesPhaseDirectly verifies a squad
// member sleeps via the existing PatchAgentPhase mechanism (used for squad
// sleep/wake since the agent controller skips squad-managed agents), not the
// sleep-requested annotation.
func TestSleepAgentEndpoint_SquadMember_PatchesPhaseDirectly(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "member-1", Namespace: "default"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
			Squad:      true,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postSleepAgent(t, k8s, "member-1")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp AgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != string(komputerv1alpha1.AgentPhaseSleeping) {
		t.Errorf("Status = %q, want Sleeping", resp.Status)
	}
}
