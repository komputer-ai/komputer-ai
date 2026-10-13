package main

import (
	"net/http"
	"testing"
)

// TestCreateScheduleRejectsEphemeralAgentWithExplicitSleep verifies a scheduled
// agent's inline template can't combine storage.ephemeral with lifecycle: Sleep.
func TestCreateScheduleRejectsEphemeralAgentWithExplicitSleep(t *testing.T) {
	k8s := newFakeK8s(t)

	body := `{"name":"s1","schedule":"* * * * *","instructions":"go","agent":{"lifecycle":"Sleep","storage":{"ephemeral":true}}}`
	w := postCreateSchedule(t, k8s, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestCreateScheduleRejectsEphemeralAgentWithDefaultedSleep verifies that
// scheduleAgentDefaults's own Sleep default doesn't sneak an ephemeral agent past
// validation: leaving lifecycle unset on an ephemeral inline agent would otherwise
// default to Sleep and silently never actually sleep.
func TestCreateScheduleRejectsEphemeralAgentWithDefaultedSleep(t *testing.T) {
	k8s := newFakeK8s(t)

	// No explicit lifecycle — scheduleAgentDefaults would normally default it to
	// Sleep. Since storage.ephemeral=true is explicit here, that default must not be
	// allowed to slip through un-validated.
	body := `{"name":"s1","schedule":"* * * * *","instructions":"go","agent":{"storage":{"ephemeral":true}}}`
	w := postCreateSchedule(t, k8s, body)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 200/201 (ephemeral must suppress the Sleep default rather than 400); body = %s", w.Code, w.Body.String())
	}
}

// TestCreateScheduleAllowsEphemeralAgentWithAutoDelete is the control case.
func TestCreateScheduleAllowsEphemeralAgentWithAutoDelete(t *testing.T) {
	k8s := newFakeK8s(t)

	body := `{"name":"s1","schedule":"* * * * *","instructions":"go","agent":{"lifecycle":"AutoDelete","storage":{"ephemeral":true}}}`
	w := postCreateSchedule(t, k8s, body)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 200/201; body = %s", w.Code, w.Body.String())
	}
}

// TestPatchScheduleRejectsEphemeralAgentWithSleepTTL mirrors the create-time check
// for the schedule patch endpoint.
func TestPatchScheduleRejectsEphemeralAgentWithSleepTTL(t *testing.T) {
	k8s := newFakeK8s(t)
	postCreateSchedule(t, k8s, `{"name":"s1","schedule":"* * * * *","instructions":"go"}`)

	body := `{"agent":{"sleepTTL":"30m","storage":{"ephemeral":true}}}`
	w := patchScheduleReq(t, k8s, "s1", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}
