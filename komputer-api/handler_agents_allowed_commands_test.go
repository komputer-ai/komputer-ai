package main

import (
	"encoding/json"
	"testing"
)

func TestCreateAgentRequestParsesAllowedCommands(t *testing.T) {
	var req CreateAgentRequest
	if err := json.Unmarshal([]byte(`{"name":"a1","instructions":"go","allowedCommands":["git status*","ls"]}`), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(req.AllowedCommands) != 2 || req.AllowedCommands[0] != "git status*" {
		t.Errorf("AllowedCommands = %v", req.AllowedCommands)
	}

	var none CreateAgentRequest
	if err := json.Unmarshal([]byte(`{"name":"a1","instructions":"go"}`), &none); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if none.AllowedCommands != nil {
		t.Error("allowedCommands must default to nil so Bash stays unrestricted")
	}
}

func TestAgentResponseAllowedCommands(t *testing.T) {
	b, _ := json.Marshal(AgentResponse{Name: "a1"})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, ok := m["allowedCommands"]; ok {
		t.Error("allowedCommands must be omitted when empty")
	}

	b, _ = json.Marshal(AgentResponse{Name: "a1", AllowedCommands: []string{"ls"}})
	m = nil
	_ = json.Unmarshal(b, &m)
	if _, ok := m["allowedCommands"]; !ok {
		t.Error("allowedCommands missing")
	}
}

func TestPatchRequestAllowedCommandsUnsetVsCleared(t *testing.T) {
	var unset PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"model":"x"}`), &unset); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if unset.AllowedCommands != nil {
		t.Error("absent allowedCommands must stay nil")
	}

	var cleared PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"allowedCommands":[]}`), &cleared); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cleared.AllowedCommands == nil || len(*cleared.AllowedCommands) != 0 {
		t.Error("explicit [] must be a non-nil empty slice so it clears the list")
	}
}
