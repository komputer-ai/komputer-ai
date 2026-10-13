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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestScheduleReconcile_Compacting_TreatedAsBusy is bug 2 from code review: the
// schedule controller's busy check only tested TaskStatus == InProgress, so a
// Compacting agent (a transient sub-state of InProgress — see taskInProgress in
// ttl.go) passed the guard as if idle. That let the scheduled run both
// overwrite spec.Instructions (the new instructions-patch-before-forward logic)
// and attempt to forward the scheduled prompt as a steer onto a task that's
// still actually in flight. Use taskInProgress for the busy check so Compacting
// is treated the same as InProgress: skip the run, don't touch Instructions.
func TestScheduleReconcile_Compacting_TreatedAsBusy(t *testing.T) {
	scheme := newTestScheme(t)
	past := metav1.NewTime(time.Now().Add(-time.Minute))

	schedule := &komputerv1alpha1.KomputerSchedule{
		ObjectMeta: metav1.ObjectMeta{Name: "sched1", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerScheduleSpec{
			Schedule:     "0 0 * * *",
			Instructions: "scheduled prompt",
			AgentName:    "my-agent",
		},
		Status: komputerv1alpha1.KomputerScheduleStatus{
			Phase:       komputerv1alpha1.SchedulePhaseActive,
			NextRunTime: &past,
		},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "original task, still running"},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:      komputerv1alpha1.AgentPhaseRunning,
			PodName:    "my-agent-pod",
			TaskStatus: komputerv1alpha1.AgentTaskCompacting,
		},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(schedule, agent).
		WithStatusSubresource(&komputerv1alpha1.KomputerSchedule{}, &komputerv1alpha1.KomputerAgent{}).
		Build()
	r := &KomputerScheduleReconciler{Client: c, Scheme: scheme}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "sched1", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	var gotSchedule komputerv1alpha1.KomputerSchedule
	if err := c.Get(context.Background(), types.NamespacedName{Name: "sched1", Namespace: "default"}, &gotSchedule); err != nil {
		t.Fatalf("get schedule: %v", err)
	}
	if gotSchedule.Status.LastRunStatus == "InProgress" {
		t.Error("LastRunStatus = InProgress, want the run skipped (agent is busy compacting)")
	}

	var gotAgent komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if gotAgent.Spec.Instructions != "original task, still running" {
		t.Errorf("Spec.Instructions = %q, want unchanged (a busy/compacting agent must not have its instructions overwritten)", gotAgent.Spec.Instructions)
	}
}
