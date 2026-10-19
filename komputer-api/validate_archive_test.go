package main

import (
	"net/http"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func archivedAgent() *komputerv1alpha1.KomputerAgent {
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: komputerv1alpha1.AgentLifecycleArchive},
			Instructions:    "done",
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseArchived,
			TaskStatus:   komputerv1alpha1.AgentTaskComplete,
			TotalCostUSD: "0.5000",
		},
	}
}

func TestCreateAgentAcceptsArchiveLifecycle(t *testing.T) {
	w := postCreateAgent(t, newFakeK8s(t), `{"name":"a1","instructions":"go","lifecycle":"Archive"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

// Sending a task to an archived agent's name must not try to forward or wake it.
func TestCreateOrTriggerRejectsArchivedAgent(t *testing.T) {
	w := postCreateAgent(t, newFakeK8s(t, archivedAgent()), `{"name":"a1","instructions":"again"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "archived") {
		t.Errorf("error body %q does not mention 'archived'", w.Body.String())
	}
}

func TestWakeAndSleepRejectArchivedAgent(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, archivedAgent())
	if w := postWakeAgent(t, k8s, "a1"); w.Code != http.StatusConflict {
		t.Errorf("wake status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
	if w := postSleepAgent(t, k8s, "a1"); w.Code != http.StatusConflict {
		t.Errorf("sleep status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
}

// Only labels and deleteTTL (the record's retention) can change on an archived agent.
func TestPatchArchivedAgent(t *testing.T) {
	k8s := newFakeK8s(t, archivedAgent())
	if w := patchAgentReq(t, k8s, "a1", `{"model":"claude-sonnet-4-6"}`); w.Code != http.StatusConflict {
		t.Errorf("model patch status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
	if w := patchAgentReq(t, k8s, "a1", `{"deleteTTL":"720h"}`); w.Code != http.StatusOK {
		t.Errorf("deleteTTL patch status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestCreateScheduleRejectsArchiveLifecycle(t *testing.T) {
	body := `{"name":"s1","schedule":"* * * * *","instructions":"go","agent":{"lifecycle":"Archive"}}`
	w := postCreateSchedule(t, newFakeK8s(t), body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestPatchArchivedAgentAllowsLabels(t *testing.T) {
	k8s := newFakeK8s(t, archivedAgent())
	if w := patchAgentReq(t, k8s, "a1", `{"labels":{"team":"billing"}}`); w.Code != http.StatusOK {
		t.Errorf("labels patch status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

// A schedule can't be pointed at an archived agent, or at one that will archive
// itself after its next task.
func TestScheduleRejectsArchivedAgentRef(t *testing.T) {
	archiveLifecycle := archivedAgent()
	archiveLifecycle.Name = "a2"
	archiveLifecycle.Status.Phase = komputerv1alpha1.AgentPhaseRunning
	k8s := newFakeK8s(t, archivedAgent(), archiveLifecycle)

	for _, name := range []string{"a1", "a2"} {
		body := `{"name":"s-` + name + `","schedule":"* * * * *","instructions":"go","agentName":"` + name + `"}`
		if w := postCreateSchedule(t, k8s, body); w.Code != http.StatusBadRequest {
			t.Errorf("create with agentName=%s status = %d, want 400; body = %s", name, w.Code, w.Body.String())
		}
	}
}

func TestPatchSquadRejectsArchivedRef(t *testing.T) {
	squad := &komputerv1alpha1.KomputerSquad{
		ObjectMeta: metav1.ObjectMeta{Name: "sq", Namespace: "default"},
	}
	k8s := newFakeK8s(t, archivedAgent(), squad)
	w := patchSquadReq(t, k8s, "sq", `{"members":[{"ref":{"name":"a1"}}]}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
}
