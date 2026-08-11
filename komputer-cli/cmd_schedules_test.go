package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

// newScheduleUpdateCmd returns the real `schedule update` command, so these
// tests exercise the production flag registration rather than a copy of it.
func newScheduleUpdateCmd(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "komputer"}
	registerScheduleCommands(root)
	for _, c := range root.Commands() {
		if c.Name() != "schedule" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() == "update" {
				return sub
			}
		}
	}
	t.Fatal("schedule update command not found")
	return nil
}

// setFlags marks each flag as Changed, which is what buildScheduleUpdateBody keys off.
func setFlags(t *testing.T, cmd *cobra.Command, flags map[string]string) {
	t.Helper()
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("set --%s=%q: %v", name, value, err)
		}
	}
}

// existingScheduleAgent is a schedule's spec.agent as it comes back from the
// API: every field populated, nested values in the shape json.Unmarshal
// produces. Returns a fresh copy per call because the merge mutates in place.
func existingScheduleAgent() map[string]interface{} {
	return map[string]interface{}{
		"templateRef":     "big",
		"model":           "claude-sonnet-4-6",
		"role":            "worker",
		"lifecycle":       "Sleep",
		"secrets":         []interface{}{"gh-token"},
		"skills":          []interface{}{"sql", "python-expert"},
		"memories":        []interface{}{"schema"},
		"connectors":      []interface{}{"figma"},
		"allowedTools":    []interface{}{"Read", "Grep"},
		"disallowedTools": []interface{}{"Bash"},
		"systemPrompt":    "be terse",
		"priority":        float64(7),
		"podSpec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name":  "agent",
					"image": "ghcr.io/komputer-ai/agent:1",
					"resources": map[string]interface{}{
						"limits":   map[string]interface{}{"cpu": "2", "memory": "4Gi"},
						"requests": map[string]interface{}{"cpu": "2", "memory": "4Gi"},
					},
				},
			},
		},
		"storage": map[string]interface{}{"size": "20Gi", "storageClassName": "gp3"},
		"labels":  map[string]interface{}{"team": "core"},
	}
}

// asJSON round-trips a value through JSON so assertions compare what actually
// goes over the wire, not the Go types that happen to hold it.
func asJSON(t *testing.T, v interface{}) map[string]interface{} {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

// agentOf pulls the "agent" object out of a body, failing if it is missing.
func agentOf(t *testing.T, body map[string]interface{}) map[string]interface{} {
	t.Helper()
	got := asJSON(t, body)
	agent, ok := got["agent"].(map[string]interface{})
	if !ok {
		t.Fatalf("body has no agent object: %v", got)
	}
	return agent
}

// TestScheduleUpdateBodyPreservesUnchangedAgentFields is the core guard on the
// read-modify-write: PATCH replaces spec.agent wholesale, so changing one field
// must still send every other field back untouched.
func TestScheduleUpdateBodyPreservesUnchangedAgentFields(t *testing.T) {
	cmd := newScheduleUpdateCmd(t)
	setFlags(t, cmd, map[string]string{"model": "claude-opus-4-6"})

	body := buildScheduleUpdateBody(cmd, existingScheduleAgent())

	got := asJSON(t, body)
	if len(got) != 1 {
		t.Errorf("body should carry only the agent object, got keys %v", got)
	}
	agent := agentOf(t, body)

	want := existingScheduleAgent()
	want["model"] = "claude-opus-4-6"
	if !reflect.DeepEqual(agent, asJSON(t, want)) {
		t.Errorf("agent object was not preserved\n got: %v\nwant: %v", agent, asJSON(t, want))
	}
}

// TestScheduleUpdateBodyOmitsAgentWhenUnchanged guards the agentChanged flag.
// The agent object is pre-populated from the server, so a length check would
// wrongly send it — and the API clears spec.agentName whenever agent is present.
func TestScheduleUpdateBodyOmitsAgentWhenUnchanged(t *testing.T) {
	cmd := newScheduleUpdateCmd(t)
	setFlags(t, cmd, map[string]string{"cron": "5 * * * *"})

	body := buildScheduleUpdateBody(cmd, existingScheduleAgent())

	if _, present := body["agent"]; present {
		t.Errorf("agent key must be absent when no agent flag changed, got %v", asJSON(t, body))
	}
	want := map[string]interface{}{"schedule": "5 * * * *"}
	if !reflect.DeepEqual(asJSON(t, body), want) {
		t.Errorf("body = %v, want %v", asJSON(t, body), want)
	}
}

// TestScheduleUpdateBodyMergesPodSpec covers the sibling-field destruction the
// review caught: --cpu alone must not drop the image or memory overrides.
func TestScheduleUpdateBodyMergesPodSpec(t *testing.T) {
	tests := []struct {
		name  string
		flags map[string]string
		want  map[string]interface{}
	}{
		{
			name:  "cpu only keeps image and memory",
			flags: map[string]string{"cpu": "8"},
			want: map[string]interface{}{
				"name":  "agent",
				"image": "ghcr.io/komputer-ai/agent:1",
				"resources": map[string]interface{}{
					"limits":   map[string]interface{}{"cpu": "8", "memory": "4Gi"},
					"requests": map[string]interface{}{"cpu": "8", "memory": "4Gi"},
				},
			},
		},
		{
			name:  "image only keeps cpu and memory",
			flags: map[string]string{"image": "ghcr.io/komputer-ai/agent:2"},
			want: map[string]interface{}{
				"name":  "agent",
				"image": "ghcr.io/komputer-ai/agent:2",
				"resources": map[string]interface{}{
					"limits":   map[string]interface{}{"cpu": "2", "memory": "4Gi"},
					"requests": map[string]interface{}{"cpu": "2", "memory": "4Gi"},
				},
			},
		},
		{
			name:  "memory only keeps image and cpu",
			flags: map[string]string{"memory-limit": "16Gi"},
			want: map[string]interface{}{
				"name":  "agent",
				"image": "ghcr.io/komputer-ai/agent:1",
				"resources": map[string]interface{}{
					"limits":   map[string]interface{}{"cpu": "2", "memory": "16Gi"},
					"requests": map[string]interface{}{"cpu": "2", "memory": "16Gi"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newScheduleUpdateCmd(t)
			setFlags(t, cmd, tt.flags)

			agent := agentOf(t, buildScheduleUpdateBody(cmd, existingScheduleAgent()))

			podSpec, ok := agent["podSpec"].(map[string]interface{})
			if !ok {
				t.Fatalf("agent has no podSpec: %v", agent)
			}
			containers, ok := podSpec["containers"].([]interface{})
			if !ok || len(containers) != 1 {
				t.Fatalf("podSpec.containers = %v, want exactly one container", podSpec["containers"])
			}
			if !reflect.DeepEqual(containers[0], tt.want) {
				t.Errorf("agent container\n got: %v\nwant: %v", containers[0], tt.want)
			}
		})
	}
}

// TestScheduleUpdateBodyMergesStorage covers the other half of the finding:
// --storage sets the size and must leave storageClassName alone.
func TestScheduleUpdateBodyMergesStorage(t *testing.T) {
	cmd := newScheduleUpdateCmd(t)
	setFlags(t, cmd, map[string]string{"storage": "50Gi"})

	agent := agentOf(t, buildScheduleUpdateBody(cmd, existingScheduleAgent()))

	want := map[string]interface{}{"size": "50Gi", "storageClassName": "gp3"}
	if !reflect.DeepEqual(agent["storage"], want) {
		t.Errorf("agent.storage = %v, want %v", agent["storage"], want)
	}
}

// TestScheduleUpdateBodyWithoutExistingAgent covers a schedule that targets an
// agent by name: there is nothing to merge into, so the flags build a fresh
// inline template and the podSpec/storage fallbacks kick in.
func TestScheduleUpdateBodyWithoutExistingAgent(t *testing.T) {
	t.Run("agent flag builds a fresh template", func(t *testing.T) {
		cmd := newScheduleUpdateCmd(t)
		setFlags(t, cmd, map[string]string{"model": "claude-opus-4-6", "storage": "50Gi"})

		agent := agentOf(t, buildScheduleUpdateBody(cmd, nil))

		want := map[string]interface{}{
			"model":   "claude-opus-4-6",
			"storage": map[string]interface{}{"size": "50Gi"},
		}
		if !reflect.DeepEqual(agent, want) {
			t.Errorf("agent = %v, want %v", agent, want)
		}
	})

	t.Run("cpu falls back to a freshly built podSpec", func(t *testing.T) {
		cmd := newScheduleUpdateCmd(t)
		setFlags(t, cmd, map[string]string{"cpu": "8"})

		agent := agentOf(t, buildScheduleUpdateBody(cmd, nil))

		want := asJSON(t, map[string]interface{}{"podSpec": buildPodSpecOverride("8", "", "")})
		if !reflect.DeepEqual(agent, want) {
			t.Errorf("agent = %v, want %v", agent, want)
		}
	})

	t.Run("non-agent flag still omits the agent key", func(t *testing.T) {
		cmd := newScheduleUpdateCmd(t)
		setFlags(t, cmd, map[string]string{"cron": "5 * * * *"})

		body := buildScheduleUpdateBody(cmd, nil)

		if _, present := body["agent"]; present {
			t.Errorf("agent key must be absent, got %v", asJSON(t, body))
		}
	})
}

// TestMergePodSpecOverrideFallsBackOnUnknownShapes pins the fallback: an
// unexpected podSpec shape must degrade to replace-everything, never panic.
func TestMergePodSpecOverrideFallsBackOnUnknownShapes(t *testing.T) {
	fresh := buildPodSpecOverride("8", "", "")
	tests := []struct {
		name     string
		existing interface{}
	}{
		{"nil", nil},
		{"not an object", "podSpec"},
		{"no containers key", map[string]interface{}{"restartPolicy": "Never"}},
		{"containers not a list", map[string]interface{}{"containers": "agent"}},
		{"no container named agent", map[string]interface{}{
			"containers": []interface{}{map[string]interface{}{"name": "sidecar"}},
		}},
		{"container entries not objects", map[string]interface{}{
			"containers": []interface{}{"agent"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergePodSpecOverride(tt.existing, "8", "", "")
			if !reflect.DeepEqual(got, fresh) {
				t.Errorf("mergePodSpecOverride(%v) = %v, want fallback %v", tt.existing, got, fresh)
			}
		})
	}

	t.Run("no overrides returns nil", func(t *testing.T) {
		if got := mergePodSpecOverride(existingScheduleAgent()["podSpec"], "", "", ""); got != nil {
			t.Errorf("mergePodSpecOverride with no overrides = %v, want nil", got)
		}
	})
}
