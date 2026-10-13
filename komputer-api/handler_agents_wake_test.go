package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func postWakeAgent(t *testing.T, k8s *K8sClient, name string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/agents/:name/wake", wakeAgent(k8s))
	req := httptest.NewRequest(http.MethodPost, "/agents/"+name+"/wake", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestWakeAgentEndpoint_SleepingAgent verifies the HTTP handler: a Sleeping agent
// comes back 200 with Instructions untouched and no longer reports Sleeping.
func TestWakeAgentEndpoint_SleepingAgent(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "the real task"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseSleeping,
			TaskStatus: komputerv1alpha1.AgentTaskComplete,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postWakeAgent(t, k8s, "a1")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp AgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status == string(komputerv1alpha1.AgentPhaseSleeping) {
		t.Errorf("Status = %q, want no longer Sleeping", resp.Status)
	}
	if resp.Instructions != "the real task" {
		t.Errorf("Instructions = %q, want unchanged %q", resp.Instructions, "the real task")
	}
}

// TestWakeAgentEndpoint_NotSleeping_Idempotent verifies waking a non-sleeping agent
// returns 200 with its current state rather than an error.
func TestWakeAgentEndpoint_NotSleeping_Idempotent(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "doing stuff"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			TaskStatus: komputerv1alpha1.AgentTaskInProgress,
		},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	w := postWakeAgent(t, k8s, "a1")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp AgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != string(komputerv1alpha1.AgentPhaseRunning) {
		t.Errorf("Status = %q, want unchanged Running", resp.Status)
	}
}

// TestWakeAgentEndpoint_NotFound verifies a missing agent returns 404.
func TestWakeAgentEndpoint_NotFound(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t)

	w := postWakeAgent(t, k8s, "nope")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", w.Code, w.Body.String())
	}
}
