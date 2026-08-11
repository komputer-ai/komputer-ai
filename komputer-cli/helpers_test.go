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
