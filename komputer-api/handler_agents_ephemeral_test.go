package main

import (
	"context"
	"net/http"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestCreateAgentRejectsEphemeralWithSleepLifecycle verifies that a new agent cannot
// combine storage.ephemeral=true with lifecycle: Sleep — sleeping would delete the
// pod and wipe the emptyDir workspace.
func TestCreateAgentRejectsEphemeralWithSleepLifecycle(t *testing.T) {
	k8s := newFakeK8s(t)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"go","lifecycle":"Sleep","storage":{"ephemeral":true}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ephemeral") {
		t.Errorf("error body %q does not mention 'ephemeral'", w.Body.String())
	}
}

// TestCreateAgentRejectsEphemeralWithSleepTTL verifies the same rejection for
// sleepTTL, which implicitly puts the agent to sleep once idle.
func TestCreateAgentRejectsEphemeralWithSleepTTL(t *testing.T) {
	k8s := newFakeK8s(t)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"go","sleepTTL":"30m","storage":{"ephemeral":true}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestCreateAgentAllowsEphemeralWithoutSleep is the control case — ephemeral alone,
// or combined with AutoDelete, must be accepted.
func TestCreateAgentAllowsEphemeralWithoutSleep(t *testing.T) {
	k8s := newFakeK8s(t)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"go","lifecycle":"AutoDelete","storage":{"ephemeral":true}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

// TestCreateAgentRejectsEphemeralInheritedFromTemplateWithSleep verifies the
// effective-value resolution: an agent that doesn't set storage.ephemeral itself,
// but uses a template that defaults to ephemeral=true, is still rejected when
// combined with Sleep.
func TestCreateAgentRejectsEphemeralInheritedFromTemplateWithSleep(t *testing.T) {
	tpl := &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "ci", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
			Storage: komputerv1alpha1.StorageSpec{Ephemeral: boolPtrAPI(true)},
		},
	}
	k8s := newFakeK8s(t, tpl)

	w := postCreateAgent(t, k8s, `{"name":"a1","instructions":"go","templateRef":"ci","lifecycle":"Sleep"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestPatchAgentRejectsFlippingEphemeralOnExistingAgent verifies storage.ephemeral is
// immutable after creation, in both directions.
func TestPatchAgentRejectsFlippingEphemeralOnExistingAgent(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Storage: &komputerv1alpha1.StorageSpec{Ephemeral: boolPtrAPI(true)},
			},
			Instructions: "go",
		},
	}
	k8s := newFakeK8s(t, agent)

	w := patchAgentReq(t, k8s, "a1", `{"storage":{"ephemeral":false}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "immutable") {
		t.Errorf("error body %q does not mention 'immutable'", w.Body.String())
	}
}

// TestPatchAgentRejectsFlippingEphemeralOnPersistentAgent is the mirror-image case:
// a persistent agent can't be patched to ephemeral=true either.
func TestPatchAgentRejectsFlippingEphemeralOnPersistentAgent(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{},
			Instructions:    "go",
		},
	}
	k8s := newFakeK8s(t, agent)

	w := patchAgentReq(t, k8s, "a1", `{"storage":{"ephemeral":true}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestPatchAgentAllowsStorageSizeChangeWithoutTouchingEphemeral verifies a patch that
// only resizes storage (the common CLI --storage path) is unaffected by the
// immutability guard, and doesn't silently clear the agent's ephemeral flag either.
func TestPatchAgentAllowsStorageSizeChangeWithoutTouchingEphemeral(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Storage: &komputerv1alpha1.StorageSpec{Ephemeral: boolPtrAPI(true), Size: "1Gi"},
			},
			Instructions: "go",
		},
	}
	k8s := newFakeK8s(t, agent)

	w := patchAgentReq(t, k8s, "a1", `{"storage":{"size":"5Gi"}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	updated, err := k8s.GetAgent(context.Background(), "default", "a1")
	if err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if updated.Spec.Storage == nil || updated.Spec.Storage.Ephemeral == nil || !*updated.Spec.Storage.Ephemeral {
		t.Errorf("Storage.Ephemeral = %v, want true to survive a size-only patch", updated.Spec.Storage)
	}
	if updated.Spec.Storage.Size != "5Gi" {
		t.Errorf("Storage.Size = %q, want 5Gi", updated.Spec.Storage.Size)
	}
}

// TestPatchAgentRejectsSleepLifecycleOnEphemeralAgent verifies the Sleep/sleepTTL
// conflict check also fires on PATCH when Lifecycle alone changes (storage untouched).
func TestPatchAgentRejectsSleepLifecycleOnEphemeralAgent(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
				Storage: &komputerv1alpha1.StorageSpec{Ephemeral: boolPtrAPI(true)},
			},
			Instructions: "go",
		},
	}
	k8s := newFakeK8s(t, agent)

	w := patchAgentReq(t, k8s, "a1", `{"lifecycle":"Sleep"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}
