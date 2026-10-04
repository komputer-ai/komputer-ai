package main

import (
	"reflect"
	"testing"
)

// parseLabelFlags is shared by `agents create`, `agents update` and
// `schedule create`; malformed input exits the process, so only the accepting
// cases are covered here.
func TestParseLabelFlags(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  map[string]string
	}{
		{"no flags", nil, map[string]string{}},
		{"single pair", []string{"team=core"}, map[string]string{"team": "core"}},
		{"multiple pairs", []string{"team=core", "env=prod"}, map[string]string{"team": "core", "env": "prod"}},
		{"splits on first equals", []string{"k=v=with=eq"}, map[string]string{"k": "v=with=eq"}},
		{"last value wins", []string{"team=core", "team=platform"}, map[string]string{"team": "platform"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLabelFlags(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLabelFlags(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSortAgentResponses(t *testing.T) {
	mk := func(name, createdAt, lastActivityAt string) AgentResponse {
		return AgentResponse{Name: name, CreatedAt: createdAt, LastActivityAt: lastActivityAt}
	}

	t.Run("created sorts newest first", func(t *testing.T) {
		agents := []AgentResponse{
			mk("old", "2024-01-01T00:00:00Z", ""),
			mk("new", "2024-03-01T00:00:00Z", ""),
			mk("mid", "2024-02-01T00:00:00Z", ""),
		}
		got := sortAgentResponses(agents, "created")
		want := []string{"new", "mid", "old"}
		for i, w := range want {
			if got[i].Name != w {
				t.Fatalf("sortAgentResponses(created)[%d] = %q, want %q", i, got[i].Name, w)
			}
		}
	})

	t.Run("last-active sorts most recently active first", func(t *testing.T) {
		agents := []AgentResponse{
			mk("stale", "2024-01-01T00:00:00Z", "2024-01-02T00:00:00Z"),
			mk("fresh", "2024-01-01T00:00:00Z", "2024-03-01T00:00:00Z"),
		}
		got := sortAgentResponses(agents, "last-active")
		if got[0].Name != "fresh" || got[1].Name != "stale" {
			t.Fatalf("sortAgentResponses(last-active) = %v, want [fresh, stale]", []string{got[0].Name, got[1].Name})
		}
	})

	t.Run("never-active agents sort last, tie-broken by created desc", func(t *testing.T) {
		agents := []AgentResponse{
			mk("never-old", "2024-01-01T00:00:00Z", ""),
			mk("active", "2024-01-01T00:00:00Z", "2024-01-05T00:00:00Z"),
			mk("never-new", "2024-02-01T00:00:00Z", ""),
		}
		got := sortAgentResponses(agents, "last-active")
		want := []string{"active", "never-new", "never-old"}
		for i, w := range want {
			if got[i].Name != w {
				t.Fatalf("sortAgentResponses(last-active)[%d] = %q, want %q", i, got[i].Name, w)
			}
		}
	})

	t.Run("unknown sort falls back to created", func(t *testing.T) {
		agents := []AgentResponse{
			mk("old", "2024-01-01T00:00:00Z", ""),
			mk("new", "2024-03-01T00:00:00Z", ""),
		}
		got := sortAgentResponses(agents, "bogus")
		if got[0].Name != "new" || got[1].Name != "old" {
			t.Fatalf("sortAgentResponses(bogus) = %v, want [new, old]", []string{got[0].Name, got[1].Name})
		}
	})
}

func TestTruncateCLI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{"empty string", "", 10, ""},
		{"shorter than max", "hello", 10, "hello"},
		{"exactly max", "hello", 5, "hello"},
		{"longer appends ellipsis", "hello world", 5, "hello..."},
		{"zero max", "hello", 0, "..."},
		{"unicode byte-based truncation", "héllo world", 3, "hé..."},
		{"long string truncated", "abcdefghij", 4, "abcd..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}
