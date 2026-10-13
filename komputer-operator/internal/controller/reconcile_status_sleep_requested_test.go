/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestReconcileStatus_SleepRequested_SleepsNow exercises POST /agents/{name}/sleep's
// operator-side counterpart: the komputer.ai/sleep-requested annotation must sleep
// the agent immediately, bypassing both guards that gate the ordinary
// Lifecycle=Sleep rule (wake-idle, sleepTTL) and regardless of spec.Lifecycle itself
// (the whole point is a real "sleep now" independent of the Sleep policy), and must
// NOT touch spec.Lifecycle in the process.
func TestReconcileStatus_SleepRequested_SleepsNow(t *testing.T) {
	cases := []struct {
		name      string
		lifecycle komputerv1alpha1.AgentLifecycle
		wakeIdle  bool
		sleepTTL  *metav1.Duration
	}{
		{name: "lifecycle empty (default)", lifecycle: komputerv1alpha1.AgentLifecycleDefault},
		{name: "with wake-idle annotation present", lifecycle: komputerv1alpha1.AgentLifecycleDefault, wakeIdle: true},
		{name: "with a sleepTTL set", lifecycle: komputerv1alpha1.AgentLifecycleDefault, sleepTTL: &metav1.Duration{Duration: 30 * 60 * 1e9}},
		{name: "lifecycle Sleep too", lifecycle: komputerv1alpha1.AgentLifecycleSleep},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := newTestScheme(t)
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "my-agent-pod", Namespace: "default"},
				Status:     corev1.PodStatus{Phase: corev1.PodRunning},
			}
			agent := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-agent",
					Namespace: "default",
					Annotations: map[string]string{
						komputerv1alpha1.AnnotationSleepRequested: "true",
					},
				},
				Spec: komputerv1alpha1.KomputerAgentSpec{
					AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: tc.lifecycle},
				},
				Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskComplete},
			}
			if tc.wakeIdle {
				agent.Annotations[komputerv1alpha1.AnnotationWakeIdle] = "true"
			}
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(pod, agent).
				WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
				Build()
			r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

			if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", tc.sleepTTL, nil, false); err != nil {
				t.Fatalf("reconcileStatus: %v", err)
			}

			err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pod", Namespace: "default"}, &corev1.Pod{})
			if !apierrors.IsNotFound(err) {
				t.Fatalf("pod should have been deleted, get err = %v", err)
			}

			var got komputerv1alpha1.KomputerAgent
			if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &got); err != nil {
				t.Fatalf("get agent: %v", err)
			}
			if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
				t.Errorf("Phase = %s, want Sleeping", got.Status.Phase)
			}
			if got.Spec.Lifecycle != tc.lifecycle {
				t.Errorf("spec.Lifecycle changed: got %q, want unchanged %q", got.Spec.Lifecycle, tc.lifecycle)
			}
			if _, ok := got.Annotations[komputerv1alpha1.AnnotationSleepRequested]; ok {
				t.Errorf("AnnotationSleepRequested should have been cleared")
			}
			if _, ok := got.Annotations[komputerv1alpha1.AnnotationWakeIdle]; ok {
				t.Errorf("AnnotationWakeIdle should have been cleared")
			}
		})
	}
}

// TestReconcileStatus_SleepRequested_WaitsForInProgressTask pins the "never kill a
// running task" rule from the AnnotationSleepRequested doc comment: while a task is
// in progress, the pod must be left running and the request annotation must stay
// set so a later reconcile (once the task ends) can retry it.
func TestReconcileStatus_SleepRequested_WaitsForInProgressTask(t *testing.T) {
	for _, status := range []komputerv1alpha1.AgentTaskStatus{
		komputerv1alpha1.AgentTaskInProgress,
		komputerv1alpha1.AgentTaskCompacting,
	} {
		t.Run(string(status), func(t *testing.T) {
			scheme := newTestScheme(t)
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "my-agent-pod", Namespace: "default"},
				Status:     corev1.PodStatus{Phase: corev1.PodRunning},
			}
			agent := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-agent",
					Namespace: "default",
					Annotations: map[string]string{
						komputerv1alpha1.AnnotationSleepRequested: "true",
					},
				},
				Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: status},
			}
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(pod, agent).
				WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
				Build()
			r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

			if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
				t.Fatalf("reconcileStatus: %v", err)
			}

			if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pod", Namespace: "default"}, &corev1.Pod{}); err != nil {
				t.Fatalf("pod should NOT have been deleted while a task is in progress: %v", err)
			}

			var got komputerv1alpha1.KomputerAgent
			if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &got); err != nil {
				t.Fatalf("get agent: %v", err)
			}
			if got.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
				t.Errorf("Phase should NOT be Sleeping while a task is in progress")
			}
			if got.Annotations[komputerv1alpha1.AnnotationSleepRequested] != "true" {
				t.Errorf("AnnotationSleepRequested should remain set so a later reconcile retries")
			}
		})
	}
}

// TestReconcileStatus_SleepRequested_EphemeralWorkspace_NotSlept mirrors the
// Lifecycle=Sleep ephemeral guard: sleeping an effectively ephemeral (emptyDir)
// agent would delete its only copy of the workspace, so the request must not be
// honored even though it was set.
func TestReconcileStatus_SleepRequested_EphemeralWorkspace_NotSlept(t *testing.T) {
	scheme := newTestScheme(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent-pod", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-agent",
			Namespace: "default",
			Annotations: map[string]string{
				komputerv1alpha1.AnnotationSleepRequested: "true",
			},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
		Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileStatus(context.Background(), agent, pod, "", "my-agent-pod", nil, nil, true); err != nil {
		t.Fatalf("reconcileStatus: %v", err)
	}

	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pod", Namespace: "default"}, &corev1.Pod{}); err != nil {
		t.Fatalf("pod should NOT have been deleted for an ephemeral workspace: %v", err)
	}
	var got komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase should NOT be Sleeping for an ephemeral-workspace agent")
	}
}
