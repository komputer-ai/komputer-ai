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
		name        string
		taskStatus  komputerv1alpha1.AgentTaskStatus
		annotations map[string]string
		want        bool
	}{
		{"empty (never run / fresh task)", komputerv1alpha1.AgentTaskStatus(""), nil, false},
		{"InProgress (pod died mid-task, continue)", komputerv1alpha1.AgentTaskInProgress, nil, false},
		{"Compacting (pod died mid-task, continue)", komputerv1alpha1.AgentTaskCompacting, nil, false},
		{"Complete (nothing to run)", komputerv1alpha1.AgentTaskComplete, nil, true},
		{"Error (nothing to run)", komputerv1alpha1.AgentTaskError, nil, true},
		{
			"wake-idle annotation forces idle even with InProgress",
			komputerv1alpha1.AgentTaskInProgress,
			map[string]string{komputerv1alpha1.AnnotationWakeIdle: "true"},
			true,
		},
		{
			"wake-idle annotation forces idle with empty TaskStatus",
			komputerv1alpha1.AgentTaskStatus(""),
			map[string]string{komputerv1alpha1.AnnotationWakeIdle: "true"},
			true,
		},
		{
			"unrelated annotations don't force idle",
			komputerv1alpha1.AgentTaskStatus(""),
			map[string]string{"some.other/annotation": "true"},
			false,
		},
		{
			"wake-idle annotation with a non-'true' value does not force idle",
			komputerv1alpha1.AgentTaskStatus(""),
			map[string]string{komputerv1alpha1.AnnotationWakeIdle: "false"},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldStartIdle(tc.taskStatus, tc.annotations)
			if got != tc.want {
				t.Errorf("shouldStartIdle(%q, %v) = %v, want %v", tc.taskStatus, tc.annotations, got, tc.want)
			}
		})
	}
}
