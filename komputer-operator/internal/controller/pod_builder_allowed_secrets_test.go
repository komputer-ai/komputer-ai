package controller

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func TestSecretAllowed(t *testing.T) {
	cases := []struct {
		name    string
		allowed []string
		want    bool
	}{
		{"anything", nil, true},
		{"github-token", []string{"github-*"}, true},
		{"github", []string{"github-*"}, false},
		{"slack-token", []string{"slack-token"}, true},
		{"slack-token-2", []string{"slack-token"}, false},
		{"aws-creds", []string{"github-*", "slack-token"}, false},
	}
	for _, tc := range cases {
		if got := komputerv1alpha1.SecretAllowed(tc.name, tc.allowed); got != tc.want {
			t.Errorf("SecretAllowed(%q, %v) = %v, want %v", tc.name, tc.allowed, got, tc.want)
		}
	}
}

func secretEnvNames(env []corev1.EnvVar) []string {
	var out []string
	for _, e := range env {
		if strings.HasPrefix(e.Name, "SECRET_") {
			out = append(out, e.Name)
		}
	}
	return out
}

func newSecret(name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("x")},
	}
}

func TestBuildAgentEnvVarsHonorsTemplateAllowedSecrets(t *testing.T) {
	scheme := newTestScheme(t)
	tpl := &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentTemplateSpec{AllowedSecrets: []string{"github-*"}},
	}
	manager := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "boss", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Secrets: []string{"github-ro", "prod-db"}},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		tpl, manager, newSecret("github-token"), newSecret("aws-creds"), newSecret("github-ro"), newSecret("prod-db"),
	).Build()

	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Secrets: []string{"github-token", "aws-creds"}},
			OfficeManager:   "boss",
			Instructions:    "do stuff",
		},
	}
	env, err := buildAgentEnvVars(context.Background(), c, agent, &komputerv1alpha1.KomputerConfig{})
	if err != nil {
		t.Fatalf("buildAgentEnvVars: %v", err)
	}
	got := strings.Join(secretEnvNames(env), ",")
	for _, want := range []string{"SECRET_GITHUB_TOKEN_TOKEN", "SECRET_GITHUB_RO_TOKEN"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing allowed secret %s in %s", want, got)
		}
	}
	for _, deny := range []string{"SECRET_AWS_CREDS_TOKEN", "SECRET_PROD_DB_TOKEN"} {
		if strings.Contains(got, deny) {
			t.Errorf("secret %s must not be injected (not in allowedSecrets): %s", deny, got)
		}
	}
}

func TestBuildAgentEnvVarsNoTemplateMeansNoRestriction(t *testing.T) {
	scheme := newTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(newSecret("aws-creds")).Build()
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentSpec{
			AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Secrets: []string{"aws-creds"}},
			Instructions:    "do stuff",
		},
	}
	env, err := buildAgentEnvVars(context.Background(), c, agent, &komputerv1alpha1.KomputerConfig{})
	if err != nil {
		t.Fatalf("buildAgentEnvVars: %v", err)
	}
	if got := secretEnvNames(env); len(got) != 1 || got[0] != "SECRET_AWS_CREDS_TOKEN" {
		t.Errorf("secret env = %v, want SECRET_AWS_CREDS_TOKEN", got)
	}
}
