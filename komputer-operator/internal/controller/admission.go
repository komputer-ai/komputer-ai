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
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

type admissionDecision struct {
	Admit    bool
	Position int32 // 1-based when not admitted; 0 when admitted
	Reason   string
}

// evaluateAdmission applies the template's MaxConcurrentAgents cap to `agent`.
// `siblings` is the full list of KomputerAgents in the same namespace; this
// function filters by templateRef internally so the caller can pass the raw list.
// Pure function — no I/O.
func evaluateAdmission(agent *komputerv1alpha1.KomputerAgent, siblings []komputerv1alpha1.KomputerAgent, limit int32) admissionDecision {
	if limit <= 0 {
		return admissionDecision{Admit: true}
	}
	tpl := agent.Spec.TemplateRef

	var running int32
	for _, s := range siblings {
		if s.Name == agent.Name {
			continue
		}
		if s.Spec.TemplateRef != tpl {
			continue
		}
		if s.Status.Phase == komputerv1alpha1.AgentPhaseRunning {
			running++
		}
	}

	// Build the contender list (this agent + all queued/pending siblings) and
	// sort by priority then creationTimestamp. The agent is admitted only if it
	// ranks within the available open slots — otherwise a higher-priority
	// sibling could be queued behind it after a slot just opened up.
	contenders := []komputerv1alpha1.KomputerAgent{*agent}
	for _, s := range siblings {
		if s.Name == agent.Name {
			continue
		}
		if s.Spec.TemplateRef != tpl {
			continue
		}
		switch s.Status.Phase {
		case komputerv1alpha1.AgentPhaseQueued, komputerv1alpha1.AgentPhasePending, "":
			contenders = append(contenders, s)
		}
	}
	sort.SliceStable(contenders, func(i, j int) bool {
		if contenders[i].Spec.Priority != contenders[j].Spec.Priority {
			return contenders[i].Spec.Priority > contenders[j].Spec.Priority
		}
		if !contenders[i].CreationTimestamp.Equal(&contenders[j].CreationTimestamp) {
			return contenders[i].CreationTimestamp.Before(&contenders[j].CreationTimestamp)
		}
		// Sub-second tie-breaker so identical creationTimestamps don't yield
		// non-deterministic ordering.
		return contenders[i].Name < contenders[j].Name
	})

	pos := int32(1)
	for _, c := range contenders {
		if c.Name == agent.Name {
			break
		}
		pos++
	}

	// Open slots = limit - running. Admit this agent only if its rank is within
	// the open slots.
	openSlots := limit - running
	if openSlots > 0 && pos <= openSlots {
		return admissionDecision{Admit: true}
	}
	return admissionDecision{
		Admit:    false,
		Position: pos,
		Reason:   fmt.Sprintf("template %q reached maxConcurrentAgents (%d/%d running)", tpl, running, limit),
	}
}

func (r *KomputerAgentReconciler) loadNamespaceSiblings(ctx context.Context, ns string) ([]komputerv1alpha1.KomputerAgent, error) {
	list := &komputerv1alpha1.KomputerAgentList{}
	if err := r.List(ctx, list, client.InNamespace(ns)); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// enqueueQueuedSiblings re-enqueues every Queued sibling in the namespace that
// shares the changed agent's templateRef. Triggered when any agent's status
// changes — a Running→anything transition may free a slot.
func (r *KomputerAgentReconciler) enqueueQueuedSiblings(ctx context.Context, obj client.Object) []reconcile.Request {
	a, ok := obj.(*komputerv1alpha1.KomputerAgent)
	if !ok {
		return nil
	}
	list := &komputerv1alpha1.KomputerAgentList{}
	if err := r.List(ctx, list, client.InNamespace(a.Namespace)); err != nil {
		return nil
	}
	reqs := make([]reconcile.Request, 0)
	for _, s := range list.Items {
		if s.Name == a.Name {
			continue
		}
		if s.Spec.TemplateRef != a.Spec.TemplateRef {
			continue
		}
		if s.Status.Phase == komputerv1alpha1.AgentPhaseQueued {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{Name: s.Name, Namespace: s.Namespace}})
		}
	}
	return reqs
}

// enqueueAgentsForTemplate re-enqueues all agents in the template's namespace
// when MaxConcurrentAgents is changed. For cluster templates, enqueues across
// all namespaces.
func (r *KomputerAgentReconciler) enqueueAgentsForTemplate(ctx context.Context, obj client.Object) []reconcile.Request {
	list := &komputerv1alpha1.KomputerAgentList{}
	var opts []client.ListOption
	if obj.GetNamespace() != "" {
		opts = append(opts, client.InNamespace(obj.GetNamespace()))
	}
	if err := r.List(ctx, list, opts...); err != nil {
		return nil
	}
	reqs := make([]reconcile.Request, 0)
	for _, a := range list.Items {
		if a.Spec.TemplateRef != obj.GetName() {
			continue
		}
		reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{Name: a.Name, Namespace: a.Namespace}})
	}
	return reqs
}
