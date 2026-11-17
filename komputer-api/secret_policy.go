package main

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// TemplateAllowedSecrets returns the allowedSecrets of an agent template
// (namespaced first, then cluster-scoped, like the operator). A missing
// template means no restriction.
func (k *K8sClient) TemplateAllowedSecrets(ctx context.Context, ns, templateRef string) ([]string, error) {
	if templateRef == "" {
		templateRef = "default"
	}
	tpl := &komputerv1alpha1.KomputerAgentTemplate{}
	err := k.client.Get(ctx, types.NamespacedName{Name: templateRef, Namespace: ns}, tpl)
	if err == nil {
		return tpl.Spec.AllowedSecrets, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	clusterTpl := &komputerv1alpha1.KomputerAgentClusterTemplate{}
	if err := k.client.Get(ctx, types.NamespacedName{Name: templateRef}, clusterTpl); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return clusterTpl.Spec.AllowedSecrets, nil
}

// checkSecretsAllowed returns an error naming the secrets outside allowed.
func checkSecretsAllowed(secrets, allowed []string, templateRef string) error {
	var denied []string
	for _, s := range secrets {
		if !komputerv1alpha1.SecretAllowed(s, allowed) {
			denied = append(denied, s)
		}
	}
	if len(denied) == 0 {
		return nil
	}
	if templateRef == "" {
		templateRef = "default"
	}
	return fmt.Errorf("secrets not allowed by template %q (allowedSecrets: %s): %s",
		templateRef, strings.Join(allowed, ", "), strings.Join(denied, ", "))
}
