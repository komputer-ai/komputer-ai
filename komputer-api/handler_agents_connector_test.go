package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func postCreateAgent(t *testing.T, k8s *K8sClient, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/agents", createOrTriggerAgent(k8s))
	req := httptest.NewRequest(http.MethodPost, "/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func patchAgentReq(t *testing.T, k8s *K8sClient, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/agents/:name", patchAgent(k8s))
	req := httptest.NewRequest(http.MethodPatch, "/agents/"+name, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestCreateAgentRejectsNewDisabledConnector verifies a brand-new agent cannot be
// created with a disabled connector in its connectors list.
func TestCreateAgentRejectsNewDisabledConnector(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	k8s := newFakeK8s(t, disabled)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"go","connectors":["slack"]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "disabled") {
		t.Errorf("error body %q does not mention 'disabled'", w.Body.String())
	}
}

// TestCreateAgentRejectsNewDisabledConnectorCrossNamespace verifies the "namespace/name"
// connector reference form is honored by the attach guard.
func TestCreateAgentRejectsNewDisabledConnectorCrossNamespace(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-slack", Namespace: "shared"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	k8s := newFakeK8s(t, disabled)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"go","connectors":["shared/shared-slack"]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestCreateAgentAllowsEnabledConnector is the control case — an enabled connector
// must not be rejected.
func TestCreateAgentAllowsEnabledConnector(t *testing.T) {
	enabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "github", URL: "https://mcp.example.com/github"},
	}
	k8s := newFakeK8s(t, enabled)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"go","connectors":["github"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

// TestPatchAgentRejectsNewlyAttachedDisabledConnector verifies that adding a disabled
// connector to an existing agent's connector list via PATCH is rejected.
func TestPatchAgentRejectsNewlyAttachedDisabledConnector(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Connectors: []string{"github"}},
			Instructions:    "go",
		},
	}
	k8s := newFakeK8s(t, disabled, agent)

	w := patchAgentReq(t, k8s, "a1", `{"connectors":["github","slack"]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "disabled") {
		t.Errorf("error body %q does not mention 'disabled'", w.Body.String())
	}
}

// TestPatchAgentAllowsResendingExistingDisabledConnector is the crucial nuance: a
// connector that was already attached before being disabled must not block a patch
// that merely re-sends the same connectors list (or edits an unrelated field).
func TestPatchAgentAllowsResendingExistingDisabledConnector(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Connectors: []string{"slack"}},
			Instructions:    "go",
		},
	}
	k8s := newFakeK8s(t, disabled, agent)

	// Re-send the exact same (now-disabled) connectors list.
	w := patchAgentReq(t, k8s, "a1", `{"connectors":["slack"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("re-sending existing connectors: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	// Editing an unrelated field must not be blocked either, even without resending connectors.
	w = patchAgentReq(t, k8s, "a1", `{"instructions":"go v2"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("editing unrelated field: status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}
