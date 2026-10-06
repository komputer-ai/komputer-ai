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

func postCreateSquad(t *testing.T, k8s *K8sClient, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/squads", createSquad(k8s))
	req := httptest.NewRequest(http.MethodPost, "/squads", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func patchSquadReq(t *testing.T, k8s *K8sClient, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/squads/:name", patchSquad(k8s))
	req := httptest.NewRequest(http.MethodPatch, "/squads/"+name, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func postAddSquadMember(t *testing.T, k8s *K8sClient, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/squads/:name/members", addSquadMember(k8s))
	req := httptest.NewRequest(http.MethodPost, "/squads/"+name+"/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateSquadRejectsDisabledConnectorInInlineMember(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	k8s := newFakeK8s(t, disabled)

	body := `{"name":"squad1","members":[{"name":"m1","spec":{"instructions":"go","connectors":["slack"]}}]}`
	w := postCreateSquad(t, k8s, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestAddSquadMemberRejectsDisabledConnector(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	squad := &komputerv1alpha1.KomputerSquad{
		ObjectMeta: metav1.ObjectMeta{Name: "squad1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerSquadSpec{
			Members: []komputerv1alpha1.KomputerSquadMember{
				{Name: "m1", Spec: &komputerv1alpha1.KomputerAgentSpec{Instructions: "go"}},
			},
		},
	}
	k8s := newFakeK8s(t, disabled, squad)

	body := `{"name":"m2","spec":{"instructions":"go","connectors":["slack"]}}`
	w := postAddSquadMember(t, k8s, "squad1", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestPatchSquadRejectsNewlyAttachedDisabledConnector(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	squad := &komputerv1alpha1.KomputerSquad{
		ObjectMeta: metav1.ObjectMeta{Name: "squad1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerSquadSpec{
			Members: []komputerv1alpha1.KomputerSquadMember{
				{Name: "m1", Spec: &komputerv1alpha1.KomputerAgentSpec{Instructions: "go", AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Connectors: []string{"github"}}}},
			},
		},
	}
	k8s := newFakeK8s(t, disabled, squad)

	body := `{"members":[{"name":"m1","spec":{"instructions":"go","connectors":["github","slack"]}}]}`
	w := patchSquadReq(t, k8s, "squad1", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestPatchSquadAllowsResendingExistingDisabledConnector is the crucial nuance: a
// member's connector that was already attached before being disabled must not block
// a patch that re-sends the same member list (matched by member name).
func TestPatchSquadAllowsResendingExistingDisabledConnector(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	squad := &komputerv1alpha1.KomputerSquad{
		ObjectMeta: metav1.ObjectMeta{Name: "squad1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerSquadSpec{
			Members: []komputerv1alpha1.KomputerSquadMember{
				{Name: "m1", Spec: &komputerv1alpha1.KomputerAgentSpec{Instructions: "go", AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Connectors: []string{"slack"}}}},
			},
		},
	}
	k8s := newFakeK8s(t, disabled, squad)

	body := `{"members":[{"name":"m1","spec":{"instructions":"go v2","connectors":["slack"]}}]}`
	w := patchSquadReq(t, k8s, "squad1", body)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}
