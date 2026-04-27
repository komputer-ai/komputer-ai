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

package webhooks

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// +kubebuilder:webhook:path=/validate-komputer-komputer-ai-v1alpha1-komputersquad,mutating=false,failurePolicy=fail,sideEffects=None,groups=komputer.komputer.ai,resources=komputersquads,verbs=create;update,versions=v1alpha1,name=vkomputersquad.kb.io,admissionReviewVersions=v1

// KomputerSquadValidator validates KomputerSquad resources, rejecting squads where
// any member agent already belongs to another squad in the same namespace.
type KomputerSquadValidator struct {
	Client client.Client
}

// SetupWebhookWithManager registers the validating webhook with the controller-runtime manager.
func (v *KomputerSquadValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&komputerv1alpha1.KomputerSquad{}).
		WithValidator(v).
		Complete()
}

var _ webhook.CustomValidator = &KomputerSquadValidator{}

func (v *KomputerSquadValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	squad := obj.(*komputerv1alpha1.KomputerSquad)
	return nil, v.validate(ctx, squad)
}

func (v *KomputerSquadValidator) ValidateUpdate(ctx context.Context, _ runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	squad := newObj.(*komputerv1alpha1.KomputerSquad)
	return nil, v.validate(ctx, squad)
}

func (v *KomputerSquadValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// validate checks:
//  1. The squad has at least one member (defense-in-depth; CRD MinItems=1 also enforces this).
//  2. No member agent (identified via Ref.Name) already belongs to another squad in the namespace.
func (v *KomputerSquadValidator) validate(ctx context.Context, squad *komputerv1alpha1.KomputerSquad) error {
	if len(squad.Spec.Members) == 0 {
		return fmt.Errorf("squad must have at least one member")
	}

	// Collect names of all ref-based members in this squad.
	requested := make(map[string]bool)
	for _, m := range squad.Spec.Members {
		if m.Ref != nil {
			requested[m.Ref.Name] = true
		}
		// Spec-based members (inline agent definitions) are new agents created by the
		// controller — they can't be in another squad yet, so no overlap check needed.
	}

	// Nothing to check if all members are inline specs.
	if len(requested) == 0 {
		return nil
	}

	var existing komputerv1alpha1.KomputerSquadList
	if err := v.Client.List(ctx, &existing, client.InNamespace(squad.Namespace)); err != nil {
		return fmt.Errorf("list squads: %w", err)
	}

	for _, other := range existing.Items {
		// Skip the squad being created/updated itself (relevant on update).
		if other.Name == squad.Name {
			continue
		}
		for _, m := range other.Spec.Members {
			if m.Ref != nil && requested[m.Ref.Name] {
				return fmt.Errorf("agent %q is already a member of squad %q", m.Ref.Name, other.Name)
			}
		}
	}

	return nil
}
