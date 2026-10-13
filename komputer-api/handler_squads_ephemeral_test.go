package main

import (
	"net/http"
	"testing"
)

// TestCreateSquadRejectsEphemeralMemberWithSleepLifecycle verifies an inline squad
// member can't combine storage.ephemeral with lifecycle: Sleep — sleeping would
// delete the shared squad pod, wiping the member's emptyDir workspace.
func TestCreateSquadRejectsEphemeralMemberWithSleepLifecycle(t *testing.T) {
	k8s := newFakeK8s(t)

	body := `{"name":"squad1","members":[{"name":"m1","spec":{"instructions":"go","lifecycle":"Sleep","storage":{"ephemeral":true}}}]}`
	w := postCreateSquad(t, k8s, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestCreateSquadAllowsEphemeralMemberWithoutSleep is the control case.
func TestCreateSquadAllowsEphemeralMemberWithoutSleep(t *testing.T) {
	k8s := newFakeK8s(t)

	body := `{"name":"squad1","members":[{"name":"m1","spec":{"instructions":"go","storage":{"ephemeral":true}}}]}`
	w := postCreateSquad(t, k8s, body)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 200/201; body = %s", w.Code, w.Body.String())
	}
}

// TestAddSquadMemberRejectsEphemeralWithSleepTTL mirrors the create-time check for
// the add-member endpoint.
func TestAddSquadMemberRejectsEphemeralWithSleepTTL(t *testing.T) {
	k8s := newFakeK8s(t)

	body := `{"name":"m2","spec":{"instructions":"go","sleepTTL":"30m","storage":{"ephemeral":true}}}`
	w := postAddSquadMember(t, k8s, "squad1", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestPatchSquadRejectsEphemeralMemberWithSleepLifecycle mirrors the create-time check
// for the full-member-list patch endpoint.
func TestPatchSquadRejectsEphemeralMemberWithSleepLifecycle(t *testing.T) {
	k8s := newFakeK8s(t)

	body := `{"members":[{"name":"m1","spec":{"instructions":"go","lifecycle":"Sleep","storage":{"ephemeral":true}}}]}`
	w := patchSquadReq(t, k8s, "squad1", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}
