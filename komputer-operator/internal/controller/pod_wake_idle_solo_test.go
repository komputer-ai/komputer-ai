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

// TestBuildPodWakeIdleFromTaskStatus verifies that the solo agent pod builder
// sets KOMPUTER_WAKE_IDLE=true exactly when shouldStartIdle says the agent's
// last task already finished (Complete/Error), and omits it when a task is
// pending or was interrupted ("" / InProgress / Compacting) — so the pod
// auto-runs spec.Instructions in those cases.
func TestBuildPodWakeIdleFromTaskStatus(t *testing.T) {
	cases := []struct {
		name       string
		taskStatus komputerv1alpha1.AgentTaskStatus
		wantIdle   bool
	}{
		{"empty status auto-runs", komputerv1alpha1.AgentTaskStatus(""), false},
		{"InProgress auto-runs (resume)", komputerv1alpha1.AgentTaskInProgress, false},
		{"Compacting auto-runs (resume)", komputerv1alpha1.AgentTaskCompacting, false},
		{"Complete starts idle", komputerv1alpha1.AgentTaskComplete, true},
		{"Error starts idle", komputerv1alpha1.AgentTaskError, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scheme := newTestScheme(t)
			c := fake.NewClientBuilder().WithScheme(scheme).Build()
			r := &KomputerAgentReconciler{Client: c, Scheme: scheme, ControlNamespace: "default"}

			template := &komputerv1alpha1.KomputerAgentTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
				Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
					PodSpec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "agent", Image: "komputer-ai/agent:latest"}},
					},
				},
			}
			agent := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
				Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "do stuff"},
				Status:     komputerv1alpha1.KomputerAgentStatus{TaskStatus: tc.taskStatus},
			}
			config := &komputerv1alpha1.KomputerConfig{}

			pod, err := r.buildPod(context.Background(), agent, template, "my-agent-pvc", "my-agent-pod", config)
			if err != nil {
				t.Fatalf("buildPod returned error: %v", err)
			}

			val, ok := toolPolicyEnvValue(pod.Spec.Containers[0].Env, "KOMPUTER_WAKE_IDLE")
			gotIdle := ok && val == "true"
			if gotIdle != tc.wantIdle {
				t.Errorf("taskStatus=%q: KOMPUTER_WAKE_IDLE present=%v value=%q, want idle=%v", tc.taskStatus, ok, val, tc.wantIdle)
			}
		})
	}
}
