package controller

import (
	"reflect"
	"testing"
)

func TestMergeLabels_NilInputs(t *testing.T) {
	got := mergeLabels(nil, nil)
	if len(got) != 0 {
		t.Fatalf("want empty map, got %v", got)
	}
}

func TestMergeLabels_UserOnly(t *testing.T) {
	got := mergeLabels(map[string]string{"team": "core"}, nil)
	want := map[string]string{"team": "core"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeLabels_SystemOnly(t *testing.T) {
	got := mergeLabels(nil, map[string]string{"komputer.ai/agent-name": "alice"})
	want := map[string]string{"komputer.ai/agent-name": "alice"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeLabels_SystemOverridesUser(t *testing.T) {
	got := mergeLabels(
		map[string]string{"komputer.ai/agent-name": "ATTACKER", "team": "core"},
		map[string]string{"komputer.ai/agent-name": "alice"},
	)
	want := map[string]string{"komputer.ai/agent-name": "alice", "team": "core"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeLabels_BothPreserved(t *testing.T) {
	got := mergeLabels(
		map[string]string{"team": "core", "env": "prod"},
		map[string]string{"komputer.ai/agent-name": "alice", "komputer.ai/squad": "false"},
	)
	want := map[string]string{
		"team": "core",
		"env":  "prod",
		"komputer.ai/agent-name": "alice",
		"komputer.ai/squad":      "false",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
