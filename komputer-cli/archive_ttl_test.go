package main

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestArchiveTTLFlagRegistered checks every command that exposes --sleep-ttl
// also exposes --archive-ttl, so the two stay in step.
func TestArchiveTTLFlagRegistered(t *testing.T) {
	root := &cobra.Command{Use: "komputer"}
	registerAgentCommands(root)

	checked := 0
	for _, c := range root.Commands() {
		if c.Flags().Lookup("sleep-ttl") == nil {
			continue
		}
		checked++
		if c.Flags().Lookup("archive-ttl") == nil {
			t.Errorf("%q has --sleep-ttl but no --archive-ttl", c.Name())
		}
	}
	if checked == 0 {
		t.Fatal("no command with --sleep-ttl found")
	}
}

func TestArchiveTTLAgentResponseJSON(t *testing.T) {
	var a AgentResponse
	in := `{"name":"a","archiveTTL":"24h","archiveExpiresAt":"2030-01-01T00:00:00Z"}`
	if err := json.Unmarshal([]byte(in), &a); err != nil {
		t.Fatal(err)
	}
	if a.ArchiveTTL != "24h" || a.ArchiveExpiresAt != "2030-01-01T00:00:00Z" {
		t.Errorf("got ArchiveTTL=%q ArchiveExpiresAt=%q", a.ArchiveTTL, a.ArchiveExpiresAt)
	}

	out, err := json.Marshal(AgentResponse{Name: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "archive") {
		t.Errorf("unset archive fields should be omitted, got %s", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	fn()
	w.Close()
	b, _ := io.ReadAll(r)
	return string(b)
}

func TestPrintAgentArchiveTTL(t *testing.T) {
	with := captureStdout(t, func() {
		printAgent(AgentResponse{Name: "a", ArchiveTTL: "24h", ArchiveExpiresAt: "2999-01-01T00:00:00Z"})
	})
	if !strings.Contains(with, "Archive TTL:") || !strings.Contains(with, "24h") || !strings.Contains(with, "archives in") {
		t.Errorf("expected Archive TTL row with hint, got:\n%s", with)
	}

	without := captureStdout(t, func() { printAgent(AgentResponse{Name: "a"}) })
	if strings.Contains(without, "Archive TTL:") {
		t.Errorf("Archive TTL row should be hidden when unset, got:\n%s", without)
	}

	noHint := captureStdout(t, func() { printAgent(AgentResponse{Name: "a", ArchiveTTL: "24h"}) })
	if !strings.Contains(noHint, "Archive TTL:") || strings.Contains(noHint, "archives") {
		t.Errorf("expected row without hint when no archive is pending, got:\n%s", noHint)
	}
}
