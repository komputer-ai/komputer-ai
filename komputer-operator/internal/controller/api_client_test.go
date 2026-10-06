package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The operator's taskTimeout cancel must reach the agent in its own namespace.
// Before this it dropped the namespace, the API looked in its default namespace,
// and every agent living elsewhere got a 404 — so taskTimeout never fired for them.
func TestCancelAgentTaskViaAPISendsTheNamespace(t *testing.T) {
	var gotPath, gotNamespace, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotNamespace = r.Method, r.URL.Path, r.URL.Query().Get("namespace")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("KOMPUTER_API_URL", srv.URL)

	if err := cancelAgentTaskViaAPI(context.Background(), nil, "team-a", "my-agent"); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/agents/my-agent/cancel" {
		t.Errorf("got %s %s, want POST /api/v1/agents/my-agent/cancel", gotMethod, gotPath)
	}
	if gotNamespace != "team-a" {
		t.Errorf("namespace = %q, want %q", gotNamespace, "team-a")
	}
}

func TestCancelAgentTaskViaAPIReportsAnAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("KOMPUTER_API_URL", srv.URL)

	if err := cancelAgentTaskViaAPI(context.Background(), nil, "team-a", "my-agent"); err == nil {
		t.Fatal("want an error on 404, got nil")
	}
}
