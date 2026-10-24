package main

import (
	"context"
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestValidateEffort verifies the allowed effort values match exactly what the
// Claude Agent SDK's ClaudeAgentOptions.effort accepts, and that "" (unset /
// model default) is always valid — it must never be rejected, since that's the
// default state of every agent today.
func TestValidateEffort(t *testing.T) {
	cases := []struct {
		effort  string
		wantErr bool
	}{
		{"", false},
		{"low", false},
		{"medium", false},
		{"high", false},
		{"xhigh", false},
		{"max", false},
		{"ultra", true},
		{"HIGH", true},
		{"auto", true},
	}
	for _, tc := range cases {
		t.Run(tc.effort, func(t *testing.T) {
			err := validateEffort(tc.effort)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateEffort(%q) error = %v, wantErr = %v", tc.effort, err, tc.wantErr)
			}
		})
	}
}

// TestPatchAgentSpecEffortSetAndClear verifies the same pointer semantics as
// model: a nil *string leaves effort unchanged, and a non-nil pointer (even to
// "") sets/clears it.
func TestPatchAgentSpecEffortSetAndClear(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Effort: "high"},
		},
	}
	k8s := newFakeK8s(t, agent)

	t.Run("omitted leaves effort unchanged", func(t *testing.T) {
		if err := k8s.PatchAgentSpec(context.Background(), "default", "a1",
			nil, nil, nil, nil, nil, nil, nil,
			TTLUpdate{}, TTLUpdate{}, TTLUpdate{}, TTLUpdate{}, StringUpdate{}, StringUpdate{}); err != nil {
			t.Fatalf("PatchAgentSpec: %v", err)
		}
		got, err := k8s.GetAgent(context.Background(), "default", "a1")
		if err != nil {
			t.Fatalf("GetAgent: %v", err)
		}
		if got.Spec.Effort != "high" {
			t.Errorf("Effort = %q, want unchanged \"high\"", got.Spec.Effort)
		}
	})

	t.Run("explicit value sets effort", func(t *testing.T) {
		low := "low"
		if err := k8s.PatchAgentSpec(context.Background(), "default", "a1",
			nil, &low, nil, nil, nil, nil, nil,
			TTLUpdate{}, TTLUpdate{}, TTLUpdate{}, TTLUpdate{}, StringUpdate{}, StringUpdate{}); err != nil {
			t.Fatalf("PatchAgentSpec: %v", err)
		}
		got, err := k8s.GetAgent(context.Background(), "default", "a1")
		if err != nil {
			t.Fatalf("GetAgent: %v", err)
		}
		if got.Spec.Effort != "low" {
			t.Errorf("Effort = %q, want \"low\"", got.Spec.Effort)
		}
	})

	t.Run("explicit empty string clears effort back to unset", func(t *testing.T) {
		empty := ""
		if err := k8s.PatchAgentSpec(context.Background(), "default", "a1",
			nil, &empty, nil, nil, nil, nil, nil,
			TTLUpdate{}, TTLUpdate{}, TTLUpdate{}, TTLUpdate{}, StringUpdate{}, StringUpdate{}); err != nil {
			t.Fatalf("PatchAgentSpec: %v", err)
		}
		got, err := k8s.GetAgent(context.Background(), "default", "a1")
		if err != nil {
			t.Fatalf("GetAgent: %v", err)
		}
		if got.Spec.Effort != "" {
			t.Errorf("Effort = %q, want cleared to \"\" (unset)", got.Spec.Effort)
		}
	})
}

// TestPatchAgentRequestDistinguishesEffortClearFromOmit mirrors the existing
// model/sleepTTL JSON-decoding test: an omitted "effort" key must decode to a
// nil pointer (no change), while an explicit "" must decode to a non-nil
// pointer to "" (clear back to unset).
func TestPatchAgentRequestDistinguishesEffortClearFromOmit(t *testing.T) {
	var omitted PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"model":"x"}`), &omitted); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if omitted.Effort != nil {
		t.Error("omitted effort must stay nil so a patch doesn't clear it")
	}

	var cleared PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"effort":""}`), &cleared); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cleared.Effort == nil || *cleared.Effort != "" {
		t.Error("explicit \"\" must be distinguishable from omitted")
	}

	var set PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"effort":"xhigh"}`), &set); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if set.Effort == nil || *set.Effort != "xhigh" {
		t.Error("explicit effort value must decode through")
	}
}
