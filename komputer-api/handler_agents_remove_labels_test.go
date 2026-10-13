package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// ── k8s layer: UpdateAgentLabels ──────────────────────────────────────────────

// TestUpdateAgentLabels_RemovesFromSpecAndMetadata is the core contract: a
// removed key must disappear from BOTH Spec.Labels and ObjectMeta.Labels in one
// patch. The metadata side matters because the operator's reconcile loop only
// ever overlays Spec.Labels onto metadata.labels additively (mergeLabels in
// komputer-operator) and never removes a stale key, and the list endpoint's
// ?label= filter matches metadata labels — so a spec-only removal would leave a
// label that keeps matching list filters.
func TestUpdateAgentLabels_RemovesFromSpecAndMetadata(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a1",
			Namespace: "default",
			Labels:    map[string]string{"team": "core", "env": "prod"},
		},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Labels: map[string]string{"team": "core", "env": "prod"},
			},
		},
	}
	k8s := newFakeK8s(t, agent)

	if err := k8s.UpdateAgentLabels(context.Background(), "default", "a1", nil, []string{"team"}); err != nil {
		t.Fatalf("UpdateAgentLabels: %v", err)
	}

	got, err := k8s.GetAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if _, ok := got.Spec.Labels["team"]; ok {
		t.Error("Spec.Labels still has \"team\", want removed")
	}
	if _, ok := got.ObjectMeta.Labels["team"]; ok {
		t.Error("ObjectMeta.Labels still has \"team\", want removed")
	}
	if got.Spec.Labels["env"] != "prod" {
		t.Errorf("Spec.Labels[\"env\"] = %q, want unchanged \"prod\"", got.Spec.Labels["env"])
	}
	if got.ObjectMeta.Labels["env"] != "prod" {
		t.Errorf("ObjectMeta.Labels[\"env\"] = %q, want unchanged \"prod\"", got.ObjectMeta.Labels["env"])
	}
}

// TestUpdateAgentLabels_SetAndRemoveInOneCall verifies a single call can both
// add/overwrite and remove keys.
func TestUpdateAgentLabels_SetAndRemoveInOneCall(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a1",
			Namespace: "default",
			Labels:    map[string]string{"old": "yes"},
		},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Labels: map[string]string{"old": "yes"},
			},
		},
	}
	k8s := newFakeK8s(t, agent)

	err := k8s.UpdateAgentLabels(context.Background(), "default", "a1",
		map[string]string{"new": "value"}, []string{"old"})
	if err != nil {
		t.Fatalf("UpdateAgentLabels: %v", err)
	}

	got, err := k8s.GetAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if _, ok := got.Spec.Labels["old"]; ok {
		t.Error("Spec.Labels still has \"old\", want removed")
	}
	if _, ok := got.ObjectMeta.Labels["old"]; ok {
		t.Error("ObjectMeta.Labels still has \"old\", want removed")
	}
	if got.Spec.Labels["new"] != "value" {
		t.Errorf("Spec.Labels[\"new\"] = %q, want \"value\"", got.Spec.Labels["new"])
	}
}

// TestUpdateAgentLabels_RemovingMissingKeyIsNoop verifies removing a key that
// isn't present is not an error.
func TestUpdateAgentLabels_RemovingMissingKeyIsNoop(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default", Labels: map[string]string{"env": "prod"}},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Labels: map[string]string{"env": "prod"}},
		},
	}
	k8s := newFakeK8s(t, agent)

	if err := k8s.UpdateAgentLabels(context.Background(), "default", "a1", nil, []string{"does-not-exist"}); err != nil {
		t.Fatalf("UpdateAgentLabels should be a no-op for a missing key, got error: %v", err)
	}

	got, err := k8s.GetAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if got.Spec.Labels["env"] != "prod" {
		t.Errorf("Spec.Labels[\"env\"] = %q, want unchanged \"prod\"", got.Spec.Labels["env"])
	}
}

// ── validation ────────────────────────────────────────────────────────────────

func TestValidateRemoveLabelKeys(t *testing.T) {
	if err := validateRemoveLabelKeys([]string{"team", "env"}); err != nil {
		t.Errorf("valid keys rejected: %v", err)
	}
	if err := validateRemoveLabelKeys([]string{"komputer.ai/agent-name"}); err == nil {
		t.Error("expected reserved-prefix key to be rejected")
	} else if !strings.Contains(err.Error(), "reserved") {
		t.Errorf("error %q should mention 'reserved'", err.Error())
	}
	// The personal-agent key is allow-listed through the reserved prefix.
	if err := validateRemoveLabelKeys([]string{personalAgentLabel}); err != nil {
		t.Errorf("personal-agent label key should be allowed, got: %v", err)
	}
	if err := validateRemoveLabelKeys([]string{"not a valid key!"}); err == nil {
		t.Error("expected malformed key to be rejected")
	}
}

// ── PatchAgentRequest JSON ────────────────────────────────────────────────────

func TestPatchAgentRequestParsesRemoveLabels(t *testing.T) {
	var req PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"removeLabels":["team","env"]}`), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(req.RemoveLabels) != 2 || req.RemoveLabels[0] != "team" || req.RemoveLabels[1] != "env" {
		t.Errorf("RemoveLabels = %v, want [team env]", req.RemoveLabels)
	}
}

// ── handler level ─────────────────────────────────────────────────────────────

// TestPatchAgentRemoveLabels_EndToEnd exercises the full patch handler: an
// existing label is removed, another is added, and the response plus a
// follow-up GetAgent both reflect spec AND metadata being updated together.
func TestPatchAgentRemoveLabels_EndToEnd(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a1",
			Namespace: "default",
			Labels:    map[string]string{"team": "core", "env": "prod"},
		},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Labels: map[string]string{"team": "core", "env": "prod"},
			},
			Instructions: "go",
		},
	}
	k8s := newFakeK8s(t, agent)

	w := patchAgentReq(t, k8s, "a1", `{"labels":{"stage":"beta"},"removeLabels":["team"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp AgentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := resp.Labels["team"]; ok {
		t.Error("response Labels still has \"team\", want removed")
	}
	if resp.Labels["stage"] != "beta" {
		t.Errorf("response Labels[\"stage\"] = %q, want \"beta\"", resp.Labels["stage"])
	}
	if resp.Labels["env"] != "prod" {
		t.Errorf("response Labels[\"env\"] = %q, want unchanged \"prod\"", resp.Labels["env"])
	}

	got, err := k8s.GetAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if _, ok := got.ObjectMeta.Labels["team"]; ok {
		t.Error("ObjectMeta.Labels still has \"team\" after patch, want removed")
	}
}

// TestPatchAgentRemoveLabels_ReservedKeyRejected verifies a reserved-prefix key
// in removeLabels is a 400, mirroring the existing rule for labels to set.
func TestPatchAgentRemoveLabels_ReservedKeyRejected(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "go"},
	}
	k8s := newFakeK8s(t, agent)

	w := patchAgentReq(t, k8s, "a1", `{"removeLabels":["komputer.ai/agent-name"]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "reserved") {
		t.Errorf("error body %q should mention 'reserved'", w.Body.String())
	}
}

// TestPatchAgentRemoveLabels_KeyInBothRejected verifies a key present in both
// labels and removeLabels is rejected as ambiguous rather than silently
// picking a winner.
func TestPatchAgentRemoveLabels_KeyInBothRejected(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "go"},
	}
	k8s := newFakeK8s(t, agent)

	w := patchAgentReq(t, k8s, "a1", `{"labels":{"team":"core"},"removeLabels":["team"]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "team") {
		t.Errorf("error body %q should name the conflicting key", w.Body.String())
	}
}

// TestPatchAgentRemoveLabelsAlone_IsNotNoFields verifies removeLabels alone
// (no other patch fields) satisfies the "at least one field" check.
func TestPatchAgentRemoveLabelsAlone_IsNotNoFields(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default", Labels: map[string]string{"team": "core"}},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Labels: map[string]string{"team": "core"}},
			Instructions:    "go",
		},
	}
	k8s := newFakeK8s(t, agent)

	w := patchAgentReq(t, k8s, "a1", `{"removeLabels":["team"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

// TestPatchAgentNoFields_StillRejected guards against a regression where adding
// RemoveLabels to the "no fields" check accidentally widened it to always pass.
func TestPatchAgentNoFields_StillRejected(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "go"},
	}
	k8s := newFakeK8s(t, agent)

	w := patchAgentReq(t, k8s, "a1", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}
