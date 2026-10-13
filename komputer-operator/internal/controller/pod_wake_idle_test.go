package controller

import (
	"testing"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestShouldStartIdle is a table-driven test for the pure run-vs-idle decision
// used by both the solo agent controller and the squad controller when
// building a pod/container. See shouldStartIdle for the rule.
func TestShouldStartIdle(t *testing.T) {
	cases := []struct {
		name       string
		taskStatus komputerv1alpha1.AgentTaskStatus
		want       bool
	}{
		{"empty (never run / fresh task)", komputerv1alpha1.AgentTaskStatus(""), false},
		{"InProgress (pod died mid-task, continue)", komputerv1alpha1.AgentTaskInProgress, false},
		{"Compacting (pod died mid-task, continue)", komputerv1alpha1.AgentTaskCompacting, false},
		{"Complete (nothing to run)", komputerv1alpha1.AgentTaskComplete, true},
		{"Error (nothing to run)", komputerv1alpha1.AgentTaskError, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldStartIdle(tc.taskStatus)
			if got != tc.want {
				t.Errorf("shouldStartIdle(%q) = %v, want %v", tc.taskStatus, got, tc.want)
			}
		})
	}
}
