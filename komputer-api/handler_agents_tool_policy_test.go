package main

import (
	"encoding/json"
	"testing"
)

func TestCreateAgentRequestParsesToolPolicy(t *testing.T) {
	body := `{"name":"a1","instructions":"go","allowedTools":["Read","mcp__figma__*"],"disallowedTools":["Bash"]}`

	var req CreateAgentRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(req.AllowedTools) != 2 || req.AllowedTools[0] != "Read" || req.AllowedTools[1] != "mcp__figma__*" {
		t.Errorf("AllowedTools = %v", req.AllowedTools)
	}
	if len(req.DisallowedTools) != 1 || req.DisallowedTools[0] != "Bash" {
		t.Errorf("DisallowedTools = %v", req.DisallowedTools)
	}
}

func TestCreateAgentRequestToolPolicyOptional(t *testing.T) {
	var req CreateAgentRequest
	if err := json.Unmarshal([]byte(`{"name":"a1","instructions":"go"}`), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.AllowedTools != nil || req.DisallowedTools != nil {
		t.Error("tool policy must default to nil so agents keep current behavior")
	}
}

func TestAgentResponseOmitsEmptyToolPolicy(t *testing.T) {
	b, err := json.Marshal(AgentResponse{Name: "a1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["allowedTools"]; ok {
		t.Error("allowedTools must be omitted when empty")
	}
	if _, ok := m["disallowedTools"]; ok {
		t.Error("disallowedTools must be omitted when empty")
	}
}

func TestAgentResponseSerializesToolPolicy(t *testing.T) {
	b, err := json.Marshal(AgentResponse{
		Name:            "a1",
		AllowedTools:    []string{"Read"},
		DisallowedTools: []string{"Bash"},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["allowedTools"]; !ok {
		t.Error("allowedTools missing")
	}
	if _, ok := m["disallowedTools"]; !ok {
		t.Error("disallowedTools missing")
	}
}

func TestPatchRequestDistinguishesUnsetFromCleared(t *testing.T) {
	var unset PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"model":"x"}`), &unset); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if unset.AllowedTools != nil {
		t.Error("absent allowedTools must stay nil")
	}

	var cleared PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"allowedTools":[]}`), &cleared); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cleared.AllowedTools == nil {
		t.Fatal("explicit [] must be non-nil so it can clear the policy")
	}
	if len(*cleared.AllowedTools) != 0 {
		t.Errorf("expected empty slice, got %v", *cleared.AllowedTools)
	}

	var set PatchAgentRequest
	if err := json.Unmarshal([]byte(`{"disallowedTools":["Bash"]}`), &set); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if set.DisallowedTools == nil || len(*set.DisallowedTools) != 1 || (*set.DisallowedTools)[0] != "Bash" {
		t.Errorf("DisallowedTools = %v", set.DisallowedTools)
	}
}
