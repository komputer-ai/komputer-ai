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
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func mkAgent(name, tpl string, prio int32, age time.Duration, phase komputerv1alpha1.KomputerAgentPhase) komputerv1alpha1.KomputerAgent {
	return komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: name, CreationTimestamp: metav1.NewTime(time.Now().Add(-age))},
		Spec:       komputerv1alpha1.KomputerAgentSpec{TemplateRef: tpl, Priority: prio},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: phase},
	}
}

func TestEvaluateAdmission_NoCap(t *testing.T) {
	a := mkAgent("a", "default", 0, 0, "")
	d := evaluateAdmission(&a, []komputerv1alpha1.KomputerAgent{a}, 0)
	if !d.Admit {
		t.Fatal("expected admit when cap is 0")
	}
}

func TestEvaluateAdmission_UnderCap(t *testing.T) {
	a := mkAgent("a", "default", 0, 0, "")
	r1 := mkAgent("r1", "default", 0, time.Minute, komputerv1alpha1.AgentPhaseRunning)
	d := evaluateAdmission(&a, []komputerv1alpha1.KomputerAgent{a, r1}, 5)
	if !d.Admit {
		t.Fatal("expected admit when 1 < 5")
	}
}

func TestEvaluateAdmission_FIFO(t *testing.T) {
	r1 := mkAgent("r1", "default", 0, 2*time.Minute, komputerv1alpha1.AgentPhaseRunning)
	q1 := mkAgent("q1", "default", 0, 30*time.Second, komputerv1alpha1.AgentPhaseQueued)
	q2 := mkAgent("q2", "default", 0, 10*time.Second, komputerv1alpha1.AgentPhaseQueued)
	d := evaluateAdmission(&q2, []komputerv1alpha1.KomputerAgent{r1, q1, q2}, 1)
	if d.Admit {
		t.Fatal("expected queued")
	}
	if d.Position != 2 {
		t.Fatalf("expected pos 2, got %d", d.Position)
	}
}

func TestEvaluateAdmission_PriorityWins(t *testing.T) {
	r1 := mkAgent("r1", "default", 0, 2*time.Minute, komputerv1alpha1.AgentPhaseRunning)
	low := mkAgent("low", "default", 0, 30*time.Second, komputerv1alpha1.AgentPhaseQueued)
	high := mkAgent("high", "default", 100, 5*time.Second, komputerv1alpha1.AgentPhaseQueued)
	d := evaluateAdmission(&high, []komputerv1alpha1.KomputerAgent{r1, low, high}, 1)
	if d.Admit {
		t.Fatal("expected queued")
	}
	if d.Position != 1 {
		t.Fatalf("expected high at pos 1, got %d", d.Position)
	}
}

func TestEvaluateAdmission_DifferentTemplateNotCounted(t *testing.T) {
	r1 := mkAgent("r1", "other", 0, time.Minute, komputerv1alpha1.AgentPhaseRunning)
	a := mkAgent("a", "default", 0, 0, "")
	d := evaluateAdmission(&a, []komputerv1alpha1.KomputerAgent{r1, a}, 1)
	if !d.Admit {
		t.Fatal("agents from other templates should not consume this template's slots")
	}
}
