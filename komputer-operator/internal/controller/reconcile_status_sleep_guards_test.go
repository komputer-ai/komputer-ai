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

// The immediate lifecycle-Sleep transition in reconcileStatus has two
// independent guards that arrived in separate changes: the wake-idle annotation
// (an idle-woken agent must not be slept again on the next reconcile) and an
// ephemeral workspace (sleeping deletes the pod, which would wipe an emptyDir).
// This pins the combined condition so neither guard can quietly mask or undo
// the other: the pod is deleted only when NEITHER guard applies.
func TestReconcileStatus_SleepLifecycle_BothGuards(t *testing.T) {
	cases := []struct {
		name         string
		wakeIdle     bool
		ephemeral    bool
		wantPodKept  bool
		wantSleeping bool
	}{
		{name: "neither guard: sleeps", wakeIdle: false, ephemeral: false, wantPodKept: false, wantSleeping: true},
		{name: "wake-idle only: kept", wakeIdle: true, ephemeral: false, wantPodKept: true, wantSleeping: false},
		{name: "ephemeral only: kept", wakeIdle: false, ephemeral: true, wantPodKept: true, wantSleeping: false},
		{name: "both guards: kept", wakeIdle: true, ephemeral: true, wantPodKept: true, wantSleeping: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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
			if tc.wakeIdle {
				agent.Annotations = map[string]string{komputerv1alpha1.AnnotationWakeIdle: "true"}
			}
			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(pod, agent).
				WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
				Build()
			r := &KomputerAgentReconciler{Client: c, Scheme: scheme}

			if err := r.reconcileStatus(context.Background(), agent, pod, "my-agent-pvc", "my-agent-pod", nil, nil, tc.ephemeral); err != nil {
				t.Fatalf("reconcileStatus: %v", err)
			}

			err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent-pod", Namespace: "default"}, &corev1.Pod{})
			podKept := err == nil
			if err != nil && !apierrors.IsNotFound(err) {
				t.Fatalf("get pod: %v", err)
			}
			if podKept != tc.wantPodKept {
				t.Errorf("pod kept = %v, want %v", podKept, tc.wantPodKept)
			}

			var got komputerv1alpha1.KomputerAgent
			if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &got); err != nil {
				t.Fatalf("get agent: %v", err)
			}
			if sleeping := got.Status.Phase == komputerv1alpha1.AgentPhaseSleeping; sleeping != tc.wantSleeping {
				t.Errorf("Phase Sleeping = %v, want %v", sleeping, tc.wantSleeping)
			}
		})
	}
}
