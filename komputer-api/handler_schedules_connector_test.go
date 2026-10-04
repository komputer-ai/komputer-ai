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

func postCreateSchedule(t *testing.T, k8s *K8sClient, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/schedules", createSchedule(k8s))
	req := httptest.NewRequest(http.MethodPost, "/schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func patchScheduleReq(t *testing.T, k8s *K8sClient, name, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/schedules/:name", patchSchedule(k8s))
	req := httptest.NewRequest(http.MethodPatch, "/schedules/"+name, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateScheduleRejectsDisabledConnectorInInlineAgent(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	k8s := newFakeK8s(t, disabled)

	body := `{"name":"s1","schedule":"* * * * *","instructions":"go","agent":{"instructions":"go","connectors":["slack"]}}`
	w := postCreateSchedule(t, k8s, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestPatchScheduleRejectsNewlyAttachedDisabledConnector(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Schedule:     "* * * * *",
			Instructions: "go",
			Agent: &komputerv1alpha1.ScheduleAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Connectors: []string{"github"}},
			},
		},
	}
	k8s := newFakeK8s(t, disabled, schedule)

	body := `{"agent":{"instructions":"go","connectors":["github","slack"]}}`
	w := patchScheduleReq(t, k8s, "s1", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestPatchScheduleAllowsResendingExistingDisabledConnector(t *testing.T) {
	disabled := &komputerv1alpha1.KomputerConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "slack", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerConnectorSpec{Service: "slack", URL: "https://mcp.example.com/slack", Disabled: true},
	}
	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Schedule:     "* * * * *",
			Instructions: "go",
			Agent: &komputerv1alpha1.ScheduleAgentSpec{
				AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Connectors: []string{"slack"}},
			},
		},
	}
	k8s := newFakeK8s(t, disabled, schedule)

	body := `{"agent":{"instructions":"go v2","connectors":["slack"]}}`
	w := patchScheduleReq(t, k8s, "s1", body)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}
