package main

import "testing"

func TestMatchesAllLabels(t *testing.T) {
	cases := []struct {
		have, want map[string]string
		match      bool
	}{
		{nil, nil, true},
		{map[string]string{"a": "1"}, nil, true},
		{nil, map[string]string{"a": "1"}, false},
		{map[string]string{"a": "1"}, map[string]string{"a": "1"}, true},
		{map[string]string{"a": "1", "b": "2"}, map[string]string{"a": "1"}, true},
		{map[string]string{"a": "1"}, map[string]string{"a": "1", "b": "2"}, false},
		{map[string]string{"a": "2"}, map[string]string{"a": "1"}, false},
	}
	for i, c := range cases {
		if got := matchesAllLabels(c.have, c.want); got != c.match {
			t.Errorf("case %d: have=%v want=%v: got %v, want %v", i, c.have, c.want, got, c.match)
		}
	}
}
