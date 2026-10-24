package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func squadMemberAgent() *komputerv1alpha1.KomputerAgent {
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "m1", Namespace: "default", Labels: map[string]string{"komputer.ai/squad": "true"}},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseRunning, Squad: true},
	}
}

func TestCreateAgentSetsArchiveTTL(t *testing.T) {
	k8s := newFakeK8s(t)
	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"go","archiveTTL":"24h"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	got, err := k8s.GetAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Spec.ArchiveTTL == nil || got.Spec.ArchiveTTL.Duration != 24*time.Hour {
		t.Errorf("ArchiveTTL = %v, want 24h", got.Spec.ArchiveTTL)
	}
	var resp AgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ArchiveTTL != "24h0m0s" {
		t.Errorf("response archiveTTL = %q, want 24h0m0s", resp.ArchiveTTL)
	}
}

func TestCreateAgentRejectsBadArchiveTTL(t *testing.T) {
	w := postCreateAgent(t, newFakeK8s(t), `{"name":"a1","instructions":"go","archiveTTL":"7d"}`)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "archiveTTL") {
		t.Fatalf("status = %d, body = %s; want 400 naming archiveTTL", w.Code, w.Body.String())
	}
}

func TestPatchAgentSetsAndClearsArchiveTTL(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"}}
	k8s := newFakeK8s(t, agent)

	if w := patchAgentReq(t, k8s, "a1", `{"archiveTTL":"2h"}`); w.Code != http.StatusOK {
		t.Fatalf("set status = %d; body = %s", w.Code, w.Body.String())
	}
	got, _ := k8s.GetAgent(context.Background(), "default", "a1")
	if got.Spec.ArchiveTTL == nil || got.Spec.ArchiveTTL.Duration != 2*time.Hour {
		t.Fatalf("ArchiveTTL = %v, want 2h", got.Spec.ArchiveTTL)
	}

	if w := patchAgentReq(t, k8s, "a1", `{"archiveTTL":""}`); w.Code != http.StatusOK {
		t.Fatalf("clear status = %d; body = %s", w.Code, w.Body.String())
	}
	got, _ = k8s.GetAgent(context.Background(), "default", "a1")
	if got.Spec.ArchiveTTL != nil {
		t.Errorf("ArchiveTTL = %v, want cleared", got.Spec.ArchiveTTL)
	}
}

func TestPatchSquadMemberRejectsArchiveTTL(t *testing.T) {
	k8s := newFakeK8s(t, squadMemberAgent())
	w := patchAgentReq(t, k8s, "m1", `{"archiveTTL":"2h"}`)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "squad") {
		t.Fatalf("status = %d, body = %s; want 400 about squads", w.Code, w.Body.String())
	}
	// Clearing it is always fine.
	if w := patchAgentReq(t, k8s, "m1", `{"archiveTTL":""}`); w.Code != http.StatusOK {
		t.Errorf("clear status = %d; body = %s", w.Code, w.Body.String())
	}
}

func TestTaskToSquadMemberRejectsArchiveTTL(t *testing.T) {
	w := postCreateAgent(t, newFakeK8s(t, squadMemberAgent()), `{"name":"m1","instructions":"go","archiveTTL":"2h"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestSquadsRejectArchiveTTLMembers(t *testing.T) {
	withTTL := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{ArchiveTTL: &metav1.Duration{Duration: time.Hour}},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseSleeping},
	}
	squad := &komputerv1alpha1.KomputerSquad{ObjectMeta: metav1.ObjectMeta{Name: "sq", Namespace: "default"}}
	k8s := newFakeK8s(t, withTTL, squad)

	cases := map[string]*httptest.ResponseRecorder{
		"create with inline spec": postCreateSquad(t, k8s, `{"name":"sq2","members":[{"spec":{"model":"claude-sonnet-4-6","archiveTTL":"1h"}}]}`),
		"create with ref":         postCreateSquad(t, k8s, `{"name":"sq3","members":[{"ref":{"name":"a1"}}]}`),
		"patch with ref":          patchSquadReq(t, k8s, "sq", `{"members":[{"ref":{"name":"a1"}}]}`),
		"add inline spec":         postAddSquadMember(t, k8s, "sq", `{"spec":{"model":"claude-sonnet-4-6","archiveTTL":"1h"}}`),
		"add ref":                 postAddSquadMember(t, k8s, "sq", `{"ref":{"name":"a1"}}`),
	}
	for name, w := range cases {
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "archiveTTL") {
			t.Errorf("%s: status = %d, body = %s; want 400 naming archiveTTL", name, w.Code, w.Body.String())
		}
	}
}

func TestScheduleArchiveTTLNeedsS3ForFixedNaming(t *testing.T) {
	k8s := newFakeK8s(t)
	fixed := `{"name":"s1","schedule":"* * * * *","instructions":"go","agent":{"archiveTTL":"1h"}}`
	if w := postCreateSchedule(t, k8s, fixed); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "archiveTTL") {
		t.Errorf("fixed naming: status = %d, body = %s; want 400 naming archiveTTL", w.Code, w.Body.String())
	}
	perRun := `{"name":"s2","schedule":"* * * * *","instructions":"go","agentNaming":"Date","agent":{"archiveTTL":"1h"}}`
	if w := postCreateSchedule(t, k8s, perRun); w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("per-run naming: status = %d, want success; body = %s", w.Code, w.Body.String())
	}
}

func TestFillAgentArchiveTTL(t *testing.T) {
	at := metav1.NewTime(time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC))
	agent := &komputerv1alpha1.KomputerAgent{
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{ArchiveTTL: &metav1.Duration{Duration: 24 * time.Hour}},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{ArchiveExpiresAt: &at},
	}
	var resp AgentResponse
	fillAgentTTL(&resp, agent)
	if resp.ArchiveTTL != "24h0m0s" || resp.ArchiveExpiresAt != "2026-08-11T12:00:00Z" {
		t.Errorf("archiveTTL/archiveExpiresAt = %q/%q", resp.ArchiveTTL, resp.ArchiveExpiresAt)
	}
}
