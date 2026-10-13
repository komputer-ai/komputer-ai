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
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestReconcileStatus_EphemeralAgentSkipsSleepLifecycle verifies that an agent with
// Lifecycle=Sleep and a completed task is NOT slept (pod deleted) when its workspace
// is an ephemeral emptyDir — sleeping would silently wipe it. A non-ephemeral agent in
// the identical situation (see reconcileStatus's own doc comment / envtest coverage)
// is slept as usual; only the ephemeral path is guarded.
func TestReconcileStatus_EphemeralAgentSkipsSleepLifecycle(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent(func(a *komputerv1alpha1.KomputerAgent) {
		a.Spec.Lifecycle = komputerv1alpha1.AgentLifecycleSleep
		a.Status.TaskStatus = komputerv1alpha1.AgentTaskComplete
	})
	pod := agentPod("test-agent-pod", "default")
	r := newTTLReconciler(t, agent, pod)

	if err := r.reconcileStatus(ctx, agent, pod, "test-agent-pvc", "test-agent-pod", nil, nil, true); err != nil {
		t.Fatalf("reconcileStatus: %v", err)
	}

	// The pod must survive — deleting it would wipe the emptyDir workspace.
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent-pod", Namespace: "default"}, &corev1.Pod{}); err != nil {
		t.Errorf("pod was deleted despite ephemeral guard: %v", err)
	}

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.Phase == komputerv1alpha1.AgentPhaseSleeping {
		t.Error("Phase = Sleeping, want the ephemeral agent to stay awake")
	}
	if got.Status.PvcName != "" {
		t.Errorf("PvcName = %q, want empty for an ephemeral (no-PVC) agent", got.Status.PvcName)
	}
}

// TestReconcileStatus_NonEphemeralAgentStillSleeps is the control case: an identical
// agent with ephemeralWorkspace=false is still slept, so the guard above is scoped
// correctly rather than disabling Sleep lifecycle entirely.
func TestReconcileStatus_NonEphemeralAgentStillSleeps(t *testing.T) {
	ctx := context.Background()
	agent := ttlAgent(func(a *komputerv1alpha1.KomputerAgent) {
		a.Spec.Lifecycle = komputerv1alpha1.AgentLifecycleSleep
		a.Status.TaskStatus = komputerv1alpha1.AgentTaskComplete
	})
	pod := agentPod("test-agent-pod", "default")
	r := newTTLReconciler(t, agent, pod)

	if err := r.reconcileStatus(ctx, agent, pod, "test-agent-pvc", "test-agent-pod", nil, nil, false); err != nil {
		t.Fatalf("reconcileStatus: %v", err)
	}

	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent-pod", Namespace: "default"}, &corev1.Pod{}); err == nil {
		t.Error("pod still exists, want it deleted by Sleep lifecycle")
	}

	var got komputerv1alpha1.KomputerAgent
	if err := r.Get(ctx, types.NamespacedName{Name: "test-agent", Namespace: "default"}, &got); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if got.Status.Phase != komputerv1alpha1.AgentPhaseSleeping {
		t.Errorf("Phase = %q, want Sleeping", got.Status.Phase)
	}
	if got.Status.PvcName != "test-agent-pvc" {
		t.Errorf("PvcName = %q, want the PVC name preserved", got.Status.PvcName)
	}
}
