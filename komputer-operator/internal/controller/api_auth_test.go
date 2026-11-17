package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func TestCancelAgentTaskViaAPISendsTheToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("KOMPUTER_API_URL", srv.URL)
	t.Setenv("KOMPUTER_API_TOKEN", "s3cret")

	if err := cancelAgentTaskViaAPI(context.Background(), nil, "team-a", "my-agent"); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if gotAuth != "Bearer s3cret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer s3cret")
	}
}

func TestSetAPIAuthNoTokenNoHeader(t *testing.T) {
	t.Setenv("KOMPUTER_API_TOKEN", "")
	req, _ := http.NewRequest(http.MethodGet, "http://x", nil)
	setAPIAuth(req)
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want none", got)
	}
}

func TestAPITokenEnvVarsOnlyForManagers(t *testing.T) {
	config := &komputerv1alpha1.KomputerConfig{Spec: komputerv1alpha1.KomputerConfigSpec{
		APITokenSecret: &komputerv1alpha1.SecretKeyRef{Name: "komputer-api-token", Key: "token"},
	}}
	worker := &komputerv1alpha1.KomputerAgent{}
	if env := buildAPITokenEnvVars(worker, config); len(env) != 0 {
		t.Errorf("worker got %d env vars, want none", len(env))
	}

	manager := &komputerv1alpha1.KomputerAgent{Spec: komputerv1alpha1.KomputerAgentSpec{AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Role: "manager"}}}
	env := buildAPITokenEnvVars(manager, config)
	if len(env) != 1 || env[0].Name != "KOMPUTER_API_TOKEN" {
		t.Fatalf("manager env = %+v, want KOMPUTER_API_TOKEN", env)
	}
	ref := env[0].ValueFrom.SecretKeyRef
	if ref == nil || ref.Name != "komputer-api-token" || ref.Key != "token" {
		t.Errorf("secretKeyRef = %+v", ref)
	}
	if env[0].Value != "" {
		t.Error("token must come from the Secret, not a plaintext value")
	}
}

func TestAPITokenEnvVarsOmittedWhenUnset(t *testing.T) {
	manager := &komputerv1alpha1.KomputerAgent{Spec: komputerv1alpha1.KomputerAgentSpec{AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Role: "manager"}}}
	if env := buildAPITokenEnvVars(manager, &komputerv1alpha1.KomputerConfig{}); len(env) != 0 {
		t.Errorf("got %d env vars with no apiTokenSecret, want none", len(env))
	}
}
