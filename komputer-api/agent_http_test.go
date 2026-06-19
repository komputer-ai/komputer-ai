package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// initialize the global Logger once so the retry-warning path does not nil-panic.
func init() { InitLogger() }

func TestDoAgentRequest_RetriesTransientThenSucceeds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	k := &K8sClient{}
	body, status, err := k.doAgentRequest(context.Background(), http.MethodGet, srv.URL, "", 2*time.Second)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if string(body) != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", string(body))
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestDoAgentRequest_DoesNotRetry4xx(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("agent busy"))
	}))
	defer srv.Close()

	k := &K8sClient{}
	_, status, err := k.doAgentRequest(context.Background(), http.MethodPost, srv.URL, "{}", 2*time.Second)
	if err == nil {
		t.Fatal("expected error for 409 response, got nil")
	}
	if status != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", status)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("expected exactly 1 attempt (no retry on 4xx), got %d", got)
	}
}

func TestDoAgentRequest_ExhaustsRetriesOnPersistentTransient(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	k := &K8sClient{}
	_, _, err := k.doAgentRequest(context.Background(), http.MethodGet, srv.URL, "", 2*time.Second)
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if got := atomic.LoadInt32(&attempts); int(got) != agentRetryBackoff.Steps {
		t.Fatalf("expected %d attempts, got %d", agentRetryBackoff.Steps, got)
	}
}

func TestDoAgentRequest_StopsOnContextCancellation(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel during the first backoff sleep, before retries are exhausted.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	k := &K8sClient{}
	_, _, err := k.doAgentRequest(ctx, http.MethodGet, srv.URL, "", 2*time.Second)
	if err == nil {
		t.Fatal("expected error on context cancellation, got nil")
	}
	if got := atomic.LoadInt32(&attempts); int(got) >= agentRetryBackoff.Steps {
		t.Fatalf("expected fewer than %d attempts due to cancellation, got %d", agentRetryBackoff.Steps, got)
	}
}
