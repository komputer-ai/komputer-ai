package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// agentRetryBackoff is the retry policy for direct HTTP calls to agent pods.
// Transient failures (connection refused while the pod is still starting, per-attempt
// timeouts, 5xx) are retried with jittered exponential backoff. Semantic 4xx responses
// (e.g. 409 "agent busy") are returned immediately and never retried.
var agentRetryBackoff = wait.Backoff{
	Steps:    4,                      // up to 4 attempts total
	Duration: 200 * time.Millisecond, // first backoff delay
	Factor:   2.0,                    // 200ms → 400ms → 800ms
	Jitter:   0.2,                    // ±20% to avoid synchronized retries
	Cap:      2 * time.Second,
}

// doAgentRequest performs an HTTP request to an agent pod, retrying transient failures
// with exponential backoff. Each attempt runs under its own perAttemptTimeout derived
// from ctx; the retry loop stops immediately if ctx is cancelled. It returns the
// response body and the final HTTP status code (0 if no response was received).
func (k *K8sClient) doAgentRequest(ctx context.Context, method, url, body string, perAttemptTimeout time.Duration) ([]byte, int, error) {
	var (
		respBody []byte
		status   int
		lastErr  error
		attempt  int
	)

	err := wait.ExponentialBackoffWithContext(ctx, agentRetryBackoff, func(ctx context.Context) (bool, error) {
		attempt++
		b, s, retryable, err := k.tryAgentRequest(ctx, method, url, body, perAttemptTimeout)
		respBody, status = b, s
		if err == nil {
			lastErr = nil
			return true, nil // success
		}
		lastErr = err
		if !retryable {
			return false, err // non-retryable (e.g. 4xx): stop and surface the error
		}
		Logger.Warnw("retrying agent request", "attempt", attempt, "method", method, "url", url, "error", err)
		return false, nil // retryable: let backoff sleep, then try again
	})

	if err != nil {
		// On a non-retryable failure ExponentialBackoff returns our error directly; on
		// exhausted retries or context cancellation it returns its own sentinel — in
		// every case lastErr holds the most informative message.
		if lastErr != nil {
			return respBody, status, lastErr
		}
		return respBody, status, err
	}
	return respBody, status, nil
}

// tryAgentRequest performs a single HTTP attempt. It reports whether a failure is
// retryable: transport errors and 5xx responses are retryable; 4xx responses are not.
func (k *K8sClient) tryAgentRequest(ctx context.Context, method, url, body string, perAttemptTimeout time.Duration) (respBody []byte, status int, retryable bool, err error) {
	attemptCtx, cancel := context.WithTimeout(ctx, perAttemptTimeout)
	defer cancel()

	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(attemptCtx, method, url, reqBody)
	if err != nil {
		return nil, 0, false, err // malformed request: not retryable
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// A cancelled or expired parent context is terminal; a per-attempt timeout or
		// transport error is transient and worth another attempt.
		if ctx.Err() != nil {
			return nil, 0, false, err
		}
		return nil, 0, true, err
	}
	defer resp.Body.Close()

	respBody, _ = io.ReadAll(resp.Body)
	if resp.StatusCode >= 500 {
		return respBody, resp.StatusCode, true, fmt.Errorf("agent returned status %d: %s", resp.StatusCode, string(respBody))
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return respBody, resp.StatusCode, false, fmt.Errorf("agent returned status %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, resp.StatusCode, false, nil
}
