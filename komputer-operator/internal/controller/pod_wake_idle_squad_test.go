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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestBuildSquadPodSpecWakeIdleFromTaskStatus mirrors
// TestBuildPodWakeIdleFromTaskStatus for squad member containers: a member
// starts idle (KOMPUTER_WAKE_IDLE=true) when its last task already finished
// (Complete/Error) or it is still Sleeping, and auto-runs otherwise.
func TestBuildSquadPodSpecWakeIdleFromTaskStatus(t *testing.T) {
	cases := []struct {
		name       string
		phase      komputerv1alpha1.KomputerAgentPhase
		taskStatus komputerv1alpha1.AgentTaskStatus
		wantIdle   bool
	}{
		{"empty status auto-runs", komputerv1alpha1.AgentPhaseRunning, komputerv1alpha1.AgentTaskStatus(""), false},
		{"InProgress auto-runs (resume)", komputerv1alpha1.AgentPhaseRunning, komputerv1alpha1.AgentTaskInProgress, false},
		{"Compacting auto-runs (resume)", komputerv1alpha1.AgentPhaseRunning, komputerv1alpha1.AgentTaskCompacting, false},
		{"Complete starts idle", komputerv1alpha1.AgentPhaseRunning, komputerv1alpha1.AgentTaskComplete, true},
		{"Error starts idle", komputerv1alpha1.AgentPhaseRunning, komputerv1alpha1.AgentTaskError, true},
		{"still-Sleeping sibling starts idle regardless of taskStatus", komputerv1alpha1.AgentPhaseSleeping, komputerv1alpha1.AgentTaskInProgress, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := newTestScheme(t)
			template := &komputerv1alpha1.KomputerAgentTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
				Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
					PodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "agent", Image: "komputer-ai/agent:latest"}},
					},
				},
			}
			agent := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{Name: "member-a", Namespace: "default"},
				Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "do stuff"},
				Status: komputerv1alpha1.KomputerAgentStatus{
					Phase:      tc.phase,
					TaskStatus: tc.taskStatus,
					PvcName:    "member-a-pvc",
					Port:       8000,
				},
			}
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(template).Build()
			r := &KomputerSquadReconciler{Client: c, Scheme: scheme}

			squad := testSquad("member-a")
			config := &komputerv1alpha1.KomputerConfig{}

			pod, err := r.buildSquadPodSpec(context.Background(), squad, []*komputerv1alpha1.KomputerAgent{agent}, config)
			if err != nil {
				t.Fatalf("buildSquadPodSpec returned error: %v", err)
			}
			if len(pod.Spec.Containers) != 1 {
				t.Fatalf("expected 1 container, got %d", len(pod.Spec.Containers))
			}

			val, ok := toolPolicyEnvValue(pod.Spec.Containers[0].Env, "KOMPUTER_WAKE_IDLE")
			gotIdle := ok && val == "true"
			if gotIdle != tc.wantIdle {
				t.Errorf("phase=%q taskStatus=%q: KOMPUTER_WAKE_IDLE present=%v value=%q, want idle=%v", tc.phase, tc.taskStatus, ok, val, tc.wantIdle)
			}
		})
	}
}
