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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// LabelMirroredFromNs is applied to every mirror Secret to distinguish
// operator-managed mirrors from user-managed secrets.
const LabelMirroredFromNs = "komputer.ai/mirrored-from-ns"

// sourceMissingError is a typed error returned when the source Secret is absent.
type sourceMissingError struct {
	Namespace string
	Name      string
}

func (e *sourceMissingError) Error() string {
	return fmt.Sprintf("source secret %s/%s not found", e.Namespace, e.Name)
}

// isMissingSourceError reports whether err (or any wrapped cause) is a sourceMissingError.
func isMissingSourceError(err error) bool {
	var target *sourceMissingError
	return errors.As(err, &target)
}

// sourceRef records a (namespace, name) pair that was passed to mirrorSourceToNamespace.
type sourceRef struct {
	Namespace string
	Name      string
}

// mirrorSourceToNamespace copies a source Secret into targetNs under the same name.
// When sourceNs == targetNs the source IS the mirror — returns immediately.
// If the source is absent, returns a *sourceMissingError.
// If the mirror already exists and its Data/Type match the source, this is a no-op.
func mirrorSourceToNamespace(ctx context.Context, c client.Client, sourceNs, sourceName, targetNs string) error {
	log := logf.FromContext(ctx)

	// Source IS the mirror — nothing to do.
	if sourceNs == targetNs {
		return nil
	}

	// Fetch source.
	src := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: sourceName, Namespace: sourceNs}, src); err != nil {
		if apierrors.IsNotFound(err) {
			return &sourceMissingError{Namespace: sourceNs, Name: sourceName}
		}
		return fmt.Errorf("get source secret %s/%s: %w", sourceNs, sourceName, err)
	}

	// Get-or-create mirror in targetNs.
	dst := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: sourceName, Namespace: targetNs}, dst)
	if apierrors.IsNotFound(err) {
		dst = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sourceName,
				Namespace: targetNs,
				Labels: map[string]string{
					LabelMirroredFromNs: sourceNs,
				},
			},
			Type: src.Type,
			Data: src.Data,
		}
		if createErr := c.Create(ctx, dst); createErr != nil && !apierrors.IsAlreadyExists(createErr) {
			return fmt.Errorf("create mirror secret %s/%s: %w", targetNs, sourceName, createErr)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get mirror secret %s/%s: %w", targetNs, sourceName, err)
	}

	// Mirror exists — diff and update if needed.
	needsUpdate := dst.Type != src.Type || !secretDataEqual(dst.Data, src.Data)
	labelDrifted := dst.Labels == nil || dst.Labels[LabelMirroredFromNs] != sourceNs

	if !needsUpdate && !labelDrifted {
		return nil
	}

	original := dst.DeepCopy()
	dst.Data = src.Data
	dst.Type = src.Type
	if dst.Labels == nil {
		dst.Labels = map[string]string{}
	}
	dst.Labels[LabelMirroredFromNs] = sourceNs

	if needsUpdate {
		log.Info("Updating drifted mirror secret", "secret", sourceName, "namespace", targetNs)
	}
	if patchErr := c.Patch(ctx, dst, client.MergeFrom(original)); patchErr != nil {
		return fmt.Errorf("patch mirror secret %s/%s: %w", targetNs, sourceName, patchErr)
	}
	return nil
}

// reconcileAgentSecrets resolves the agent's template, builds the set of source
// secrets that must be mirrored into agent.Namespace, and calls
// mirrorSourceToNamespace for each. It returns the list of (ns, name) sources
// that were processed.
//
// Empty Namespace fields in SecretKeyRef default to controlNs (the namespace
// the operator itself runs in).
func reconcileAgentSecrets(
	ctx context.Context,
	c client.Client,
	agent *komputerv1alpha1.KomputerAgent,
	config *komputerv1alpha1.KomputerConfig,
	controlNs string,
) ([]sourceRef, error) {
	log := logf.FromContext(ctx)

	// Resolve the template.
	templateRef := agent.Spec.TemplateRef
	if templateRef == "" {
		templateRef = "default"
	}

	var anthropicRef komputerv1alpha1.SecretKeyRef
	template := &komputerv1alpha1.KomputerAgentTemplate{}
	if err := c.Get(ctx, types.NamespacedName{Name: templateRef, Namespace: agent.Namespace}, template); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("get template %q: %w", templateRef, err)
		}
		// Fall back to cluster-scoped template.
		clusterTemplate := &komputerv1alpha1.KomputerAgentClusterTemplate{}
		if clusterErr := c.Get(ctx, types.NamespacedName{Name: templateRef}, clusterTemplate); clusterErr != nil {
			return nil, fmt.Errorf("get cluster template %q: %w", templateRef, clusterErr)
		}
		anthropicRef = clusterTemplate.Spec.AnthropicKeySecretRef
	} else {
		anthropicRef = template.Spec.AnthropicKeySecretRef
	}

	// Resolve empty Namespace to controlNs.
	if anthropicRef.Namespace == "" {
		anthropicRef.Namespace = controlNs
	}

	// Build the set of sources to mirror.
	sources := []sourceRef{
		{Namespace: anthropicRef.Namespace, Name: anthropicRef.Name},
	}

	if config.Spec.Redis.PasswordSecret != nil && config.Spec.Redis.PasswordSecret.Name != "" {
		redisRef := *config.Spec.Redis.PasswordSecret
		if redisRef.Namespace == "" {
			redisRef.Namespace = controlNs
		}
		sources = append(sources, sourceRef{Namespace: redisRef.Namespace, Name: redisRef.Name})
	}

	// Mirror each source.
	for _, src := range sources {
		if err := mirrorSourceToNamespace(ctx, c, src.Namespace, src.Name, agent.Namespace); err != nil {
			log.Error(err, "Failed to mirror secret", "source", src.Namespace+"/"+src.Name, "target", agent.Namespace)
			return nil, err
		}
	}

	return sources, nil
}
