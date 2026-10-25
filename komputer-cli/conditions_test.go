package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentConditionsJSON(t *testing.T) {
	var a AgentResponse
	in := `{"name":"a","conditions":[{"type":"Asleep","status":"True","reason":"SleepTTL","message":"idle","lastTransitionTime":"2030-01-01T00:00:00Z"}]}`
	if err := json.Unmarshal([]byte(in), &a); err != nil {
		t.Fatal(err)
	}
	if len(a.Conditions) != 1 || a.Conditions[0].Type != "Asleep" || a.Conditions[0].Reason != "SleepTTL" {
		t.Errorf("unexpected conditions: %+v", a.Conditions)
	}

	out, err := json.Marshal(AgentResponse{Name: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "conditions") {
		t.Errorf("unset conditions should be omitted, got %s", out)
	}
}

func TestPrintAgentConditions(t *testing.T) {
	with := captureStdout(t, func() {
		printAgent(AgentResponse{Name: "a", Conditions: []AgentCondition{
			{Type: "Asleep", Status: "True", Reason: "SleepTTL", LastTransitionTime: "2030-01-01T00:00:00Z"},
			{Type: "TaskTimedOut", Status: "False", Reason: "TaskStarted"},
		}})
	})
	for _, want := range []string{"Conditions:", "Asleep=True (SleepTTL) since 2030-01-01T00:00:00Z", "TaskTimedOut=False (TaskStarted)"} {
		if !strings.Contains(with, want) {
			t.Errorf("expected %q in output, got:\n%s", want, with)
		}
	}

	without := captureStdout(t, func() { printAgent(AgentResponse{Name: "a"}) })
	if strings.Contains(without, "Conditions:") {
		t.Errorf("Conditions block should be hidden when empty, got:\n%s", without)
	}
}
