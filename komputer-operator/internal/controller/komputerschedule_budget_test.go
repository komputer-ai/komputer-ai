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
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TestScheduleReconcile_BudgetExhausted_SkipsRun verifies the schedule controller
// skips firing a scheduled run against an idle agent whose lifetime budget is
// exhausted, and records why in the schedule's status — mirroring the existing
// "agent is busy" skip (see komputerschedule_busy_check_test.go). The agent's
// spec.Instructions must not be touched: the run never started, so there is no
// new task whose instructions a later pod reschedule should re-run.
func TestScheduleReconcile_BudgetExhausted_SkipsRun(t *testing.T) {
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
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{BudgetUSD: "10.00"},
			Instructions:    "original task, now idle",
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseRunning,
			PodName:      "my-agent-pod",
			TaskStatus:   komputerv1alpha1.AgentTaskComplete,
			TotalCostUSD: "12.00",
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
		t.Error("LastRunStatus = InProgress, want the run skipped (agent budget exhausted)")
	}
	if !strings.Contains(gotSchedule.Status.Message, "budget exhausted") {
		t.Errorf("Status.Message = %q, want it to explain the skip reason", gotSchedule.Status.Message)
	}

	var gotAgent komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if gotAgent.Spec.Instructions != "original task, now idle" {
		t.Errorf("Spec.Instructions = %q, want unchanged (the run never started)", gotAgent.Spec.Instructions)
	}
}

// TestScheduleReconcile_BudgetNotExhausted_RunsNormally is the control: an agent
// under its lifetime budget must fire exactly as it would with no budget set at
// all. Pinned to a sleeping agent (the simplest "run proceeds" path) so a failure
// here can't also be explained by some other branch skipping the run.
func TestScheduleReconcile_BudgetNotExhausted_RunsNormally(t *testing.T) {
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
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{BudgetUSD: "10.00"},
			Instructions:    "stale",
		},
		Status: komputerv1alpha1.KomputerAgentStatus{
			Phase:        komputerv1alpha1.AgentPhaseSleeping,
			TaskStatus:   komputerv1alpha1.AgentTaskComplete,
			TotalCostUSD: "1.00",
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
	if gotSchedule.Status.LastRunStatus != "InProgress" {
		t.Errorf("LastRunStatus = %q, want InProgress (run should proceed under budget)", gotSchedule.Status.LastRunStatus)
	}

	var gotAgent komputerv1alpha1.KomputerAgent
	if err := c.Get(context.Background(), types.NamespacedName{Name: "my-agent", Namespace: "default"}, &gotAgent); err != nil {
		t.Fatalf("get agent: %v", err)
	}
	if gotAgent.Spec.Instructions != "scheduled prompt" {
		t.Errorf("Spec.Instructions = %q, want the scheduled prompt (run should have woken the agent)", gotAgent.Spec.Instructions)
	}
}
