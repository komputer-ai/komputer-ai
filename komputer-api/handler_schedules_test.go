package main

import (
	"encoding/json"
	"testing"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func TestScheduleAgentDefaults_FillsEmpty(t *testing.T) {
	cfg := komputerv1alpha1.AgentConfigSpec{}
	scheduleAgentDefaults(&cfg)

	if cfg.Role != "worker" {
		t.Errorf("role should default to worker for schedules, got %q", cfg.Role)
	}
	if cfg.Lifecycle != komputerv1alpha1.AgentLifecycleSleep {
		t.Errorf("lifecycle should default to Sleep, got %q", cfg.Lifecycle)
	}
}

func TestScheduleAgentDefaults_RespectsExplicit(t *testing.T) {
	cfg := komputerv1alpha1.AgentConfigSpec{
		Role:      "manager",
		Lifecycle: komputerv1alpha1.AgentLifecycleAutoDelete,
	}
	scheduleAgentDefaults(&cfg)

	if cfg.Role != "manager" {
		t.Errorf("explicit role must be preserved, got %q", cfg.Role)
	}
	if cfg.Lifecycle != komputerv1alpha1.AgentLifecycleAutoDelete {
		t.Errorf("explicit lifecycle must be preserved, got %q", cfg.Lifecycle)
	}
}

func TestCreateScheduleRequestParsesFullAgentConfig(t *testing.T) {
	body := `{
      "name":"nightly","schedule":"0 9 * * *","instructions":"report",
      "agent":{
        "model":"claude-opus-4-6","role":"worker","templateRef":"big",
        "secrets":["gh-token"],"skills":["sql"],"memories":["schema"],
        "connectors":["github"],"allowedTools":["Read"],"disallowedTools":["Bash"],
        "systemPrompt":"be terse","priority":50,"lifecycle":"Sleep",
        "storage":{"size":"20Gi"},"labels":{"team":"data"}
      }}`

	var req CreateScheduleRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Agent == nil {
		t.Fatal("agent must parse")
	}
	if req.Agent.Model != "claude-opus-4-6" || req.Agent.Role != "worker" {
		t.Errorf("model/role = %q %q", req.Agent.Model, req.Agent.Role)
	}
	if len(req.Agent.Secrets) != 1 || req.Agent.Secrets[0] != "gh-token" {
		t.Errorf("secrets = %v", req.Agent.Secrets)
	}
	if len(req.Agent.Skills) != 1 || len(req.Agent.Memories) != 1 || len(req.Agent.Connectors) != 1 {
		t.Errorf("skills/memories/connectors = %v %v %v",
			req.Agent.Skills, req.Agent.Memories, req.Agent.Connectors)
	}
	if len(req.Agent.AllowedTools) != 1 || len(req.Agent.DisallowedTools) != 1 {
		t.Errorf("tool policy = %v %v", req.Agent.AllowedTools, req.Agent.DisallowedTools)
	}
	if req.Agent.Priority != 50 || req.Agent.SystemPrompt != "be terse" {
		t.Errorf("priority/systemPrompt = %d %q", req.Agent.Priority, req.Agent.SystemPrompt)
	}
	if req.Agent.Storage == nil || req.Agent.Storage.Size != "20Gi" {
		t.Errorf("storage = %v", req.Agent.Storage)
	}
	if req.Agent.Labels["team"] != "data" {
		t.Errorf("labels = %v", req.Agent.Labels)
	}
}

// The CR field is `secrets`; the old API key was `secretRefs`. encoding/json
// drops unknown keys and the handler does not set DisallowUnknownFields, so a
// stale client's secretRefs is silently ignored rather than rejected. Pin that
// so the behavior is a decision, not an accident.
func TestCreateScheduleRequestIgnoresLegacySecretRefs(t *testing.T) {
	body := `{"name":"n","schedule":"0 9 * * *","instructions":"x",
	          "agent":{"secretRefs":["old-key"]}}`

	var req CreateScheduleRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Agent != nil && len(req.Agent.Secrets) != 0 {
		t.Errorf("legacy secretRefs must not populate Secrets, got %v", req.Agent.Secrets)
	}
}
