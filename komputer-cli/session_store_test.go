package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func agentsDeleteCmd(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "komputer"}
	root.PersistentFlags().String("namespace", "", "")
	registerAgentCommands(root)
	for _, c := range root.Commands() {
		if c.Name() == "delete" {
			return c
		}
	}
	t.Fatal("delete command not found")
	return nil
}

func TestDeleteForceSendsQueryParam(t *testing.T) {
	cmd := agentsDeleteCmd(t)
	if got := agentDeleteURL("http://api", "a1", cmd); got != "http://api/api/v1/agents/a1" {
		t.Errorf("no flags: %q", got)
	}
	if err := cmd.ParseFlags([]string{"--force"}); err != nil {
		t.Fatal(err)
	}
	if got := agentDeleteURL("http://api", "a1", cmd); got != "http://api/api/v1/agents/a1?force=true" {
		t.Errorf("--force: %q", got)
	}
	if err := cmd.ParseFlags([]string{"--force", "--namespace", "team-a"}); err != nil {
		t.Fatal(err)
	}
	if got := agentDeleteURL("http://api", "a1", cmd); got != "http://api/api/v1/agents/a1?namespace=team-a&force=true" {
		t.Errorf("--force with namespace: %q", got)
	}
}

func TestSessionStoreSummary(t *testing.T) {
	if got := sessionStoreSummary(&SessionStoreView{OK: true, LastSavedAt: "2026-10-24T10:00:00Z"}); !strings.HasPrefix(got, "saved ") {
		t.Errorf("ok: %q", got)
	}
	if got := sessionStoreSummary(&SessionStoreView{OK: false, Error: "AccessDenied"}); got != "save failed: AccessDenied" {
		t.Errorf("failed: %q", got)
	}
	if got := sessionStoreSummary(nil); got != "" {
		t.Errorf("nil: %q", got)
	}
}
