package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func TestParseTTL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantNil bool
		wantErr bool
	}{
		{name: "empty means unset", value: "", wantNil: true},
		{name: "minutes", value: "30m", want: 30 * time.Minute},
		{name: "hours", value: "24h", want: 24 * time.Hour},
		{name: "compound", value: "1h30m", want: 90 * time.Minute},
		{name: "seconds", value: "45s", want: 45 * time.Second},
		{name: "garbage is rejected", value: "banana", wantErr: true},
		{name: "bare number is rejected", value: "30", wantErr: true},
		// Zero and negative would mean "expire immediately", deleting an agent the
		// instant it is created.
		{name: "zero is rejected", value: "0s", wantErr: true},
		{name: "negative is rejected", value: "-5m", wantErr: true},
		{name: "days are not a Go duration unit", value: "7d", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseTTL("sleepTTL", tc.value)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseTTL(%q) error = nil, want an error", tc.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTTL(%q) unexpected error: %v", tc.value, err)
			}
			if tc.wantNil {
				if got != nil {
					t.Errorf("parseTTL(%q) = %v, want nil", tc.value, got)
				}
				return
			}
			if got == nil || got.Duration != tc.want {
				t.Errorf("parseTTL(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseTTLErrorNamesTheField(t *testing.T) {
	_, err := parseTTL("deleteTTL", "nope")
	if err == nil {
		t.Fatal("expected an error")
	}
	// The message goes straight into the 400 body, so it has to say which field.
	if !strings.Contains(err.Error(), "deleteTTL") {
		t.Errorf("error %q does not name the offending field", err.Error())
	}
}

func TestParseTTLUpdateTriState(t *testing.T) {
	t.Run("omitted leaves the field untouched", func(t *testing.T) {
		got, err := parseTTLUpdate("sleepTTL", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Set {
			t.Error("Set = true for an omitted field, want false")
		}
	})

	t.Run("explicit empty string clears the TTL", func(t *testing.T) {
		empty := ""
		got, err := parseTTLUpdate("sleepTTL", &empty)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Set {
			t.Error("Set = false for an explicit \"\", want true (clear)")
		}
		if got.Value != nil {
			t.Errorf("Value = %v, want nil (cleared)", got.Value)
		}
	})

	t.Run("value assigns the TTL", func(t *testing.T) {
		v := "2h"
		got, err := parseTTLUpdate("sleepTTL", &v)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.Set || got.Value == nil || got.Value.Duration != 2*time.Hour {
			t.Errorf("got %+v, want Set with 2h", got)
		}
	})

	t.Run("invalid value errors", func(t *testing.T) {
		v := "later"
		if _, err := parseTTLUpdate("sleepTTL", &v); err == nil {
			t.Error("expected an error for an invalid duration")
		}
	})
}

func TestCreateAgentRequestParsesTTLs(t *testing.T) {
	body := `{"name":"a1","instructions":"go","sleepTTL":"30m","deleteTTL":"24h"}`
	var req CreateAgentRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.SleepTTL != "30m" {
		t.Errorf("SleepTTL = %q, want \"30m\"", req.SleepTTL)
	}
	if req.DeleteTTL != "24h" {
		t.Errorf("DeleteTTL = %q, want \"24h\"", req.DeleteTTL)
	}
}

func TestCreateAgentRequestTTLsOptional(t *testing.T) {
	var req CreateAgentRequest
	if err := json.Unmarshal([]byte(`{"name":"a1","instructions":"go"}`), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.SleepTTL != "" || req.DeleteTTL != "" {
		t.Error("TTLs must default to empty so existing agents keep current behavior")
	}
}

func TestPatchAgentRequestDistinguishesClearFromOmit(t *testing.T) {
	var omitted PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"model":"x"}`), &omitted); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if omitted.SleepTTL != nil {
		t.Error("omitted sleepTTL must stay nil so a patch doesn't clear it")
	}

	var cleared PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"sleepTTL":""}`), &cleared); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cleared.SleepTTL == nil || *cleared.SleepTTL != "" {
		t.Error("explicit \"\" must be distinguishable from omitted")
	}
}

func TestFillAgentTTL(t *testing.T) {
	activity := metav1.NewTime(time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))
	sleepAt := metav1.NewTime(time.Date(2026, 8, 10, 12, 30, 0, 0, time.UTC))
	deleteAt := metav1.NewTime(time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC))

	agent := &komputerv1alpha1.KomputerAgent{
		Spec: komputerv1alpha1.KomputerAgentSpec{
			SleepTTL:  &metav1.Duration{Duration: 30 * time.Minute},
			DeleteTTL: &metav1.Duration{Duration: 24 * time.Hour},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			LastActivityAt:  &activity,
			SleepExpiresAt:  &sleepAt,
			DeleteExpiresAt: &deleteAt,
		},
	}

	var resp AgentResponse
	fillAgentTTL(&resp, agent)

	if resp.SleepTTL != "30m0s" {
		t.Errorf("SleepTTL = %q, want \"30m0s\"", resp.SleepTTL)
	}
	if resp.DeleteTTL != "24h0m0s" {
		t.Errorf("DeleteTTL = %q, want \"24h0m0s\"", resp.DeleteTTL)
	}
	if resp.LastActivityAt != "2026-08-10T12:00:00Z" {
		t.Errorf("LastActivityAt = %q", resp.LastActivityAt)
	}
	if resp.SleepExpiresAt != "2026-08-10T12:30:00Z" {
		t.Errorf("SleepExpiresAt = %q", resp.SleepExpiresAt)
	}
	if resp.DeleteExpiresAt != "2026-08-11T12:00:00Z" {
		t.Errorf("DeleteExpiresAt = %q", resp.DeleteExpiresAt)
	}
}

func TestFillAgentTTLLeavesUnsetFieldsEmpty(t *testing.T) {
	var resp AgentResponse
	fillAgentTTL(&resp, &komputerv1alpha1.KomputerAgent{})

	if resp.SleepTTL != "" || resp.DeleteTTL != "" ||
		resp.LastActivityAt != "" || resp.SleepExpiresAt != "" || resp.DeleteExpiresAt != "" {
		t.Errorf("expected all TTL fields empty, got %+v", resp)
	}
}

func TestAgentResponseOmitsEmptyTTLFields(t *testing.T) {
	b, err := json.Marshal(AgentResponse{Name: "a1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"sleepTTL", "deleteTTL", "lastActivityAt", "sleepExpiresAt", "deleteExpiresAt"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s must be omitted when empty", k)
		}
	}
}

func TestDurationEqual(t *testing.T) {
	thirty := &metav1.Duration{Duration: 30 * time.Minute}
	alsoThirty := &metav1.Duration{Duration: 30 * time.Minute}
	hour := &metav1.Duration{Duration: time.Hour}

	if !durationEqual(nil, nil) {
		t.Error("durationEqual(nil, nil) = false, want true")
	}
	if durationEqual(thirty, nil) || durationEqual(nil, thirty) {
		t.Error("durationEqual with one nil = true, want false")
	}
	if !durationEqual(thirty, alsoThirty) {
		t.Error("equal durations compared unequal")
	}
	if durationEqual(thirty, hour) {
		t.Error("different durations compared equal")
	}
}
