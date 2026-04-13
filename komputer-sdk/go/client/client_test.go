package client

import (
	"testing"
)

func TestNew(t *testing.T) {
	c := New("http://localhost:8080")
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.api == nil {
		t.Fatal("client.api is nil")
	}
}

func TestNewTrailingSlash(t *testing.T) {
	c := New("http://localhost:8080/")
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

// TestMethodsExist verifies all convenience methods compile and are addressable.
func TestMethodsExist(t *testing.T) {
	c := New("http://localhost:8080")

	methods := []interface{}{
		// Agents
		c.CreateAgent,
		c.ListAgents,
		c.GetAgent,
		c.DeleteAgent,
		c.PatchAgent,
		c.CancelAgentTask,
		c.GetAgentEvents,
		c.WatchAgent,
		// Memories
		c.CreateMemory,
		c.ListMemories,
		c.GetMemory,
		c.PatchMemory,
		c.DeleteMemory,
		// Skills
		c.CreateSkill,
		c.ListSkills,
		c.GetSkill,
		c.PatchSkill,
		c.DeleteSkill,
		// Schedules
		c.CreateSchedule,
		c.ListSchedules,
		c.GetSchedule,
		c.PatchSchedule,
		c.DeleteSchedule,
		// Secrets
		c.CreateSecret,
		c.ListSecrets,
		c.UpdateSecret,
		c.DeleteSecret,
		// Connectors
		c.CreateConnector,
		c.ListConnectors,
		c.GetConnector,
		c.DeleteConnector,
		c.ListConnectorTools,
		// Offices
		c.ListOffices,
		c.GetOffice,
		c.DeleteOffice,
		c.GetOfficeEvents,
	}

	for i, m := range methods {
		if m == nil {
			t.Errorf("method %d is nil", i)
		}
	}
}

func TestCreateAgentOpts(t *testing.T) {
	opts := CreateAgentOpts{
		Model:     ptrString("claude-sonnet-4-6"),
		Lifecycle: ptrString("Sleep"),
		Skills:    []string{"skill-1"},
		Memories:  []string{"mem-1"},
	}
	if *opts.Model != "claude-sonnet-4-6" {
		t.Errorf("expected claude-sonnet-4-6, got %s", *opts.Model)
	}
	if *opts.Lifecycle != "Sleep" {
		t.Errorf("expected Sleep, got %s", *opts.Lifecycle)
	}
	if len(opts.Skills) != 1 || opts.Skills[0] != "skill-1" {
		t.Errorf("expected [skill-1], got %v", opts.Skills)
	}
}

func ptrString(s string) *string { return &s }
