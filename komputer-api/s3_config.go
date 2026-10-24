package main

import (
	"context"
	"sync"
	"time"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// s3ConfigTTL bounds how stale the API's view of KomputerConfig.spec.s3 can be.
// Handlers check it on most agent requests, so it's cached instead of listed
// every time.
const s3ConfigTTL = 15 * time.Second

// s3ConfigCache holds the last KomputerConfig.spec.s3 read. The zero value is
// ready to use.
type s3ConfigCache struct {
	mu   sync.Mutex
	spec *komputerv1alpha1.S3Spec
	at   time.Time
}

// S3Config returns the S3 session store config from the singleton
// KomputerConfig (the first one listed, as the operator does), cached for
// s3ConfigTTL. Nil when there's no config or it can't be read: the store is
// treated as off.
func (k *K8sClient) S3Config(ctx context.Context) *komputerv1alpha1.S3Spec {
	k.s3cfg.mu.Lock()
	defer k.s3cfg.mu.Unlock()
	if !k.s3cfg.at.IsZero() && time.Since(k.s3cfg.at) < s3ConfigTTL {
		return k.s3cfg.spec
	}
	var list komputerv1alpha1.KomputerConfigList
	var spec *komputerv1alpha1.S3Spec
	if err := k.client.List(ctx, &list); err != nil {
		Logger.Warnw("failed to read KomputerConfig for the S3 session store; treating it as off", "error", err)
	} else if len(list.Items) > 0 {
		spec = list.Items[0].Spec.S3
	}
	k.s3cfg.spec, k.s3cfg.at = spec, time.Now()
	return spec
}

// S3Enabled reports whether the S3 session store is configured.
func (k *K8sClient) S3Enabled(ctx context.Context) bool {
	return k.S3Config(ctx).IsEnabled()
}

// SessionStoreResult is the session_store block an agent puts on its task-end
// event: the outcome of saving its session to S3.
type SessionStoreResult struct {
	OK    bool
	At    time.Time
	Bytes int64
	Error string
}

// extractSessionStore reads an event's session_store block, or nil when the
// event has none (S3 off, or not a task-end event).
func extractSessionStore(event AgentEvent) *SessionStoreResult {
	raw, ok := event.Payload["session_store"].(map[string]interface{})
	if !ok {
		return nil
	}
	res := &SessionStoreResult{At: time.Now()}
	res.OK, _ = raw["ok"].(bool)
	if b, ok := raw["bytes"].(float64); ok {
		res.Bytes = int64(b)
	}
	res.Error, _ = raw["error"].(string)
	if at, ok := raw["at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, at); err == nil {
			res.At = t
		}
	}
	return res
}
