package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_DeleteAgentForce(t *testing.T) {
	var gotForce, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotForce = r.Method, r.URL.Query().Get("force")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"deleted","name":"a1"}`))
	}))
	defer srv.Close()

	if _, _, err := New(srv.URL).DeleteAgentForce(context.Background(), "a1"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotForce != "true" {
		t.Errorf("got %s force=%q, want DELETE force=true", gotMethod, gotForce)
	}
}

func TestClient_GetConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/config" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"s3":{"enabled":true,"bucket":"sessions"}}`))
	}))
	defer srv.Close()

	cfg, _, err := New(srv.URL).GetConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.S3.GetEnabled() || cfg.S3.GetBucket() != "sessions" {
		t.Errorf("config = %+v", cfg)
	}
}
