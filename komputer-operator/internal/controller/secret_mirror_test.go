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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		t.Fatalf("add k8s scheme: %v", err)
	}
	if err := komputerv1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("add komputer scheme: %v", err)
	}
	return s
}

// TestMirrorSourceToNamespace_HappyPath: source exists, no mirror → mirror created with correct data and label.
func TestMirrorSourceToNamespace_HappyPath(t *testing.T) {
	ctx := context.Background()
	s := newTestScheme(t)

	src := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anthropic-api-key",
			Namespace: "komputer-system",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"api-key": []byte("sk-test-value")},
	}
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(src).Build()

	err := mirrorSourceToNamespace(ctx, c, "komputer-system", "anthropic-api-key", "agent-ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mirror := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: "anthropic-api-key", Namespace: "agent-ns"}, mirror); err != nil {
		t.Fatalf("mirror not found: %v", err)
	}
	if string(mirror.Data["api-key"]) != "sk-test-value" {
		t.Errorf("data mismatch: got %q", mirror.Data["api-key"])
	}
	if mirror.Labels[LabelMirroredFromNs] != "komputer-system" {
		t.Errorf("label missing or wrong: %v", mirror.Labels)
	}
	if mirror.Type != corev1.SecretTypeOpaque {
		t.Errorf("type mismatch: got %v", mirror.Type)
	}
}

// TestMirrorSourceToNamespace_DriftUpdate: mirror exists but data drifted → mirror updated.
func TestMirrorSourceToNamespace_DriftUpdate(t *testing.T) {
	ctx := context.Background()
	s := newTestScheme(t)

	src := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anthropic-api-key",
			Namespace: "komputer-system",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"api-key": []byte("sk-rotated")},
	}
	mirror := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "anthropic-api-key",
			Namespace: "agent-ns",
			Labels:    map[string]string{LabelMirroredFromNs: "komputer-system"},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"api-key": []byte("sk-old")},
	}
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(src, mirror).Build()

	err := mirrorSourceToNamespace(ctx, c, "komputer-system", "anthropic-api-key", "agent-ns")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: "anthropic-api-key", Namespace: "agent-ns"}, updated); err != nil {
		t.Fatalf("get mirror: %v", err)
	}
	if string(updated.Data["api-key"]) != "sk-rotated" {
		t.Errorf("data not updated: got %q", updated.Data["api-key"])
	}
}

// TestMirrorSourceToNamespace_SourceMissing: source absent → typed error returned.
func TestMirrorSourceToNamespace_SourceMissing(t *testing.T) {
	ctx := context.Background()
	s := newTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(s).Build()

	err := mirrorSourceToNamespace(ctx, c, "komputer-system", "anthropic-api-key", "agent-ns")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !isMissingSourceError(err) {
		t.Errorf("expected sourceMissingError, got %T: %v", err, err)
	}
}

// TestMirrorSourceToNamespace_SameNs: sourceNs == targetNs → no-op, no error.
func TestMirrorSourceToNamespace_SameNs(t *testing.T) {
	ctx := context.Background()
	s := newTestScheme(t)
	// No secrets in fake client — if the function tried to read one it would error.
	c := fake.NewClientBuilder().WithScheme(s).Build()

	err := mirrorSourceToNamespace(ctx, c, "komputer-system", "anthropic-api-key", "komputer-system")
	if err != nil {
		t.Fatalf("expected no-op (same ns), got: %v", err)
	}
}

// TestReconcileAgentSecrets_HappyPath: both anthropic and redis secrets mirrored correctly.
func TestReconcileAgentSecrets_HappyPath(t *testing.T) {
	ctx := context.Background()
	s := newTestScheme(t)

	anthropicSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "anthropic-api-key", Namespace: "komputer-system"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"api-key": []byte("sk-test")},
	}
	redisSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "redis-secret", Namespace: "komputer-system"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"password": []byte("redispass")},
	}
	clusterTemplate := &komputerv1alpha1.KomputerAgentClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
			AnthropicKeySecretRef: komputerv1alpha1.SecretKeyRef{
				Name:      "anthropic-api-key",
				Key:       "api-key",
				Namespace: "komputer-system",
			},
		},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "my-agent", Namespace: "agent-ns"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{TemplateRef: "default"},
	}
	config := &komputerv1alpha1.KomputerConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
		Spec: komputerv1alpha1.KomputerConfigSpec{
			Redis: komputerv1alpha1.RedisSpec{
				Address: "redis:6379",
				PasswordSecret: &komputerv1alpha1.SecretKeyRef{
					Name:      "redis-secret",
					Key:       "password",
					Namespace: "komputer-system",
				},
			},
		},
	}

	c := fake.NewClientBuilder().WithScheme(s).WithObjects(anthropicSecret, redisSecret, clusterTemplate, agent).Build()

	sources, err := reconcileAgentSecrets(ctx, c, agent, config, "komputer-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(sources))
	}

	// Verify anthropic mirror in agent-ns
	m := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: "anthropic-api-key", Namespace: "agent-ns"}, m); err != nil {
		t.Errorf("anthropic mirror not found: %v", err)
	}

	// Verify redis mirror in agent-ns
	r := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: "redis-secret", Namespace: "agent-ns"}, r); err != nil {
		t.Errorf("redis mirror not found: %v", err)
	}
}
