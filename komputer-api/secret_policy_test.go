package main

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func TestCheckSecretsAllowed(t *testing.T) {
	if err := checkSecretsAllowed([]string{"anything"}, nil, ""); err != nil {
		t.Errorf("no allowedSecrets must allow everything, got %v", err)
	}
	if err := checkSecretsAllowed([]string{"github-ro", "github-rw"}, []string{"github-*"}, "t"); err != nil {
		t.Errorf("prefix match failed: %v", err)
	}
	err := checkSecretsAllowed([]string{"github-ro", "prod-db"}, []string{"github-*"}, "")
	if err == nil {
		t.Fatal("prod-db must be rejected")
	}
	if !strings.Contains(err.Error(), "prod-db") || strings.Contains(err.Error(), "github-ro,") || !strings.Contains(err.Error(), `"default"`) {
		t.Errorf("error = %q, want it to name prod-db and the default template", err)
	}
}

func TestTemplateAllowedSecretsLookup(t *testing.T) {
	nsTpl := &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "locked", Namespace: "team-a"},
		Spec:       komputerv1alpha1.KomputerAgentTemplateSpec{AllowedSecrets: []string{"github-*"}},
	}
	clusterTpl := &komputerv1alpha1.KomputerAgentClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec:       komputerv1alpha1.KomputerAgentTemplateSpec{AllowedSecrets: []string{"shared-*"}},
	}
	k := newFakeK8s(t, nsTpl, clusterTpl)
	ctx := context.Background()

	if got, err := k.TemplateAllowedSecrets(ctx, "team-a", "locked"); err != nil || len(got) != 1 || got[0] != "github-*" {
		t.Errorf("namespaced template: got %v, %v", got, err)
	}
	if got, err := k.TemplateAllowedSecrets(ctx, "team-a", ""); err != nil || len(got) != 1 || got[0] != "shared-*" {
		t.Errorf("cluster default template: got %v, %v", got, err)
	}
	if got, err := k.TemplateAllowedSecrets(ctx, "team-a", "missing"); err != nil || got != nil {
		t.Errorf("missing template must mean no restriction: got %v, %v", got, err)
	}
}
