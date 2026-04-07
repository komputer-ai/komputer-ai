package main

import "testing"

func TestIsValidK8sName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid names
		{"single char", "a", true},
		{"simple lowercase", "my-agent", true},
		{"alphanumeric", "agent123", true},
		{"numbers only", "123", true},
		{"hyphen in middle", "my-agent-01", true},
		{"exactly 63 chars", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},

		// Invalid names
		{"empty string", "", false},
		{"64 chars exceeds limit", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"starts with hyphen", "-agent", false},
		{"ends with hyphen", "agent-", false},
		{"contains uppercase", "MyAgent", false},
		{"contains dot", "my.agent", false},
		{"contains underscore", "my_agent", false},
		{"contains space", "my agent", false},
		{"only hyphen", "-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidK8sName(tt.input)
			if got != tt.want {
				t.Errorf("isValidK8sName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
