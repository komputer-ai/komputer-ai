package main

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestPatchAgentTaskStatus_ClearsWakeIdleAnnotation_WhenTaskStarts verifies the
// single clearing point for the komputer.ai/wake-idle annotation: when
// taskStatusStartsTask is true (the agent transitions from not-in-progress to
// in-progress — i.e. a real task actually started), the annotation set by an
// earlier idle wake must be removed. This is what makes a SUBSEQUENT pod
// reschedule correctly auto-run the new task instead of staying idle forever.
func TestPatchAgentTaskStatus_ClearsWakeIdleAnnotation_WhenTaskStarts(t *testing.T) {
	cases := []struct {
		name string
		prev komputerv1alpha1.AgentTaskStatus
	}{
		{"empty -> InProgress", komputerv1alpha1.AgentTaskStatus("")},
		{"Complete -> InProgress", komputerv1alpha1.AgentTaskComplete},
		{"Error -> InProgress", komputerv1alpha1.AgentTaskError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agent := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "a1",
					Namespace:   "default",
					Annotations: map[string]string{komputerv1alpha1.AnnotationWakeIdle: "true"},
				},
				Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: tc.prev},
			}
			k8s := newFakeK8sWithStatusSubresource(t, agent)

			if err := k8s.PatchAgentTaskStatus(context.Background(), "default", "a1", "InProgress", "", "", 0, 0, 0); err != nil {
				t.Fatalf("PatchAgentTaskStatus returned error: %v", err)
			}

			var got komputerv1alpha1.KomputerAgent
			if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got); err != nil {
				t.Fatalf("get agent: %v", err)
			}
			if _, ok := got.Annotations[komputerv1alpha1.AnnotationWakeIdle]; ok {
				t.Errorf("wake-idle annotation still present after task started (prev=%q)", tc.prev)
			}
		})
	}
}

// TestPatchAgentTaskStatus_DoesNotClearWakeIdleAnnotation_OnSteer verifies the
// annotation is left alone on InProgress -> InProgress (and Compacting
// transitions), since that's a steer on an already-running task, not a new
// task starting — taskStatusStartsTask is false for those.
func TestPatchAgentTaskStatus_DoesNotClearWakeIdleAnnotation_OnSteer(t *testing.T) {
	cases := []struct {
		name string
		prev komputerv1alpha1.AgentTaskStatus
		next string
	}{
		{"InProgress -> InProgress (steer)", komputerv1alpha1.AgentTaskInProgress, "InProgress"},
		{"InProgress -> Compacting", komputerv1alpha1.AgentTaskInProgress, "Compacting"},
		{"Compacting -> InProgress", komputerv1alpha1.AgentTaskCompacting, "InProgress"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agent := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "a1",
					Namespace:   "default",
					Annotations: map[string]string{komputerv1alpha1.AnnotationWakeIdle: "true"},
				},
				Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: tc.prev},
			}
			k8s := newFakeK8sWithStatusSubresource(t, agent)

			if err := k8s.PatchAgentTaskStatus(context.Background(), "default", "a1", tc.next, "", "", 0, 0, 0); err != nil {
				t.Fatalf("PatchAgentTaskStatus returned error: %v", err)
			}

			var got komputerv1alpha1.KomputerAgent
			if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got); err != nil {
				t.Fatalf("get agent: %v", err)
			}
			if got.Annotations[komputerv1alpha1.AnnotationWakeIdle] != "true" {
				t.Errorf("wake-idle annotation removed on a steer (prev=%q next=%q), want preserved", tc.prev, tc.next)
			}
		})
	}
}

// TestPatchAgentTaskStatus_NoAnnotation_NoOpPatch verifies PatchAgentTaskStatus
// doesn't choke (or issue a spurious patch) when the annotation was never set —
// the common case for agents that have never been idle-woken.
func TestPatchAgentTaskStatus_NoAnnotation_NoOpPatch(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskStatus("")},
	}
	k8s := newFakeK8sWithStatusSubresource(t, agent)

	if err := k8s.PatchAgentTaskStatus(context.Background(), "default", "a1", "InProgress", "", "", 0, 0, 0); err != nil {
		t.Fatalf("PatchAgentTaskStatus returned error: %v", err)
	}

	var got komputerv1alpha1.KomputerAgent
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.TaskStatus != komputerv1alpha1.AgentTaskInProgress {
		t.Errorf("TaskStatus = %q, want InProgress", got.Status.TaskStatus)
	}
}
