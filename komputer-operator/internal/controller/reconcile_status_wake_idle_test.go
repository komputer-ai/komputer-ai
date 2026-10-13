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

// TestReconcileStatus_SleepLifecycle_WakeIdleAnnotation_SkipsImmediateSleep is
// bug 1 from code review: a Lifecycle=Sleep agent that's idle-woken comes back
// up with TaskStatus still Complete (idle wake deliberately leaves it alone).
// Without the wake-idle annotation carve-out, reconcileStatus's immediate
// "Sleep mode: delete pod when task is done" rule would see Lifecycle=Sleep +
// TaskStatus=Complete on the very next reconcile and tear the freshly-woken
// pod right back down — undoing the wake. The annotation must suppress that
// rule while present.
func TestReconcileStatus_SleepLifecycle_WakeIdleAnnotation_SkipsImmediateSleep(t *testing.T) {
	scheme := newTestScheme(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent-pod", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-agent",
			Namespace:   "default",
			Annotations: map[string]string{komputerv1alpha1.AnnotationWakeIdle: "true"},
		},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: komputerv1alpha1.AgentLifecycleSleep},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
		Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus returned error: %v", err)
	}

	var gotPod corev1.Pod
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pod", Namespace: "default"}, &gotPod); err != nil {
		t.Fatalf("pod should NOT have been deleted while wake-idle annotated: %v", err)
	}

	var gotAgent komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if gotAgent.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase = Sleeping, want NOT Sleeping while wake-idle annotated (pod is Running)")
	}
}

// TestReconcileStatus_SleepLifecycle_WithoutAnnotation_StillSleepsImmediately
// is the control case: the existing behavior (no annotation) must be
// unchanged — Lifecycle=Sleep + TaskStatus=Complete deletes the pod and sets
// Phase=Sleeping right away.
func TestReconcileStatus_SleepLifecycle_WithoutAnnotation_StillSleepsImmediately(t *testing.T) {
	scheme := newTestScheme(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent-pod", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: komputerv1alpha1.AgentLifecycleSleep},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
		Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus returned error: %v", err)
	}

	var gotPod corev1.Pod
	err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pod", Namespace: "default"}, &gotPod)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("pod should have been deleted (no annotation), got err=%v", err)
	}

	var gotAgent komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if gotAgent.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase = %q, want Sleeping", gotAgent.Status.Phase)
	}
}

// TestReconcileStatus_AutoDeleteLifecycle_WakeIdleAnnotation_SkipsImmediateDelete
// is the other half of bug 1: an AutoDelete agent that's manually slept then
// idle-woken has TaskStatus=Complete. Without the annotation carve-out,
// reconcileStatus would delete the entire CR on the very next reconcile.
func TestReconcileStatus_AutoDeleteLifecycle_WakeIdleAnnotation_SkipsImmediateDelete(t *testing.T) {
	scheme := newTestScheme(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent-pod", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-agent",
			Namespace:   "default",
			Annotations: map[string]string{komputerv1alpha1.AnnotationWakeIdle: "true"},
		},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: komputerv1alpha1.AgentLifecycleAutoDelete},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
		Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus returned error: %v", err)
	}

	var gotAgent komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent); err != nil {
		t.Fatalf("agent CR should NOT have been deleted while wake-idle annotated: %v", err)
	}
}

// TestReconcileStatus_AutoDeleteLifecycle_WithoutAnnotation_StillDeletes is the
// control case for AutoDelete: existing behavior unchanged without the
// annotation.
func TestReconcileStatus_AutoDeleteLifecycle_WithoutAnnotation_StillDeletes(t *testing.T) {
	scheme := newTestScheme(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent-pod", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: komputerv1alpha1.AgentLifecycleAutoDelete},
		},
		Status: komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
		Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

	if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus returned error: %v", err)
	}

	var gotAgent komputerv1alpha1.KomputerAgent
	err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("agent CR should have been deleted (no annotation), got err=%v", err)
	}
}
