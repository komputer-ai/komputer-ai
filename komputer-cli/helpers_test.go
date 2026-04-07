package main

import "testing"

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
