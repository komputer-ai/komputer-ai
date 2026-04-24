package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// AgentEvent matches the JSON schema published by the komputer-agent.
type AgentEvent struct {
	AgentName string                 `json:"agentName"`
	Namespace string                 `json:"namespace,omitempty"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

// RedisWorkerConfig holds settings for the Redis stream worker.
type RedisWorkerConfig struct {
	Address      string
	Password     string
	DB           int
	StreamPrefix string // e.g. "komputer-events"
	ConsumerName string
}

// RedisWorker manages the Redis client and stream consumption loop.
type RedisWorker struct {
	Rdb          *redis.Client
	StreamPrefix string
}

// StartRedisWorker creates a Redis client, starts the stream consumption loop,
// and returns a RedisWorker so the client can be reused (e.g. for event queries).
func StartRedisWorker(ctx context.Context, cfg RedisWorkerConfig, k8s *K8sClient, hub *Hub) *RedisWorker {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	w := &RedisWorker{Rdb: rdb, StreamPrefix: cfg.StreamPrefix}

	streamPattern := cfg.StreamPrefix + ":*"

	// --- Event history handler (consumer group) ---
	// Exactly one worker processes each event: history RPUSH, K8s status patch, ACK.
	go func() {
		log.Printf("event-history-handler started (consumer=%s), consuming streams %q at %s", cfg.ConsumerName, streamPattern, cfg.Address)

		// One-time cleanup: remove old cursor tracking keys from pre-consumer-group era.
		if oldCursors, err := scanKeys(ctx, rdb, "komputer-worker-last-read:*"); err == nil && len(oldCursors) > 0 {
			rdb.Del(ctx, oldCursors...)
			log.Printf("cleaned up %d old cursor keys", len(oldCursors))
		}

		for {
			if ctx.Err() != nil {
				log.Println("event-history-handler shutting down")
				return
			}

			streams, err := scanStreams(ctx, rdb, streamPattern)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("event-history-handler scan error: %v", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
				}
				continue
			}

			if len(streams) == 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
				continue
			}

			for _, s := range streams {
				if err := ensureConsumerGroup(ctx, rdb, s); err != nil {
					log.Printf("failed to ensure consumer group for %s: %v", s, err)
					continue
				}
				claimPendingMessages(ctx, rdb, s, cfg.ConsumerName, k8s)
			}

			keys := make([]string, 0, len(streams))
			ids := make([]string, 0, len(streams))
			for _, s := range streams {
				keys = append(keys, s)
				ids = append(ids, ">")
			}

			results, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    consumerGroup,
				Consumer: cfg.ConsumerName,
				Streams:  append(keys, ids...),
				Block:    5 * time.Second,
				Count:    100,
			}).Result()
			if err != nil {
				if err == redis.Nil {
					continue
				}
				if ctx.Err() != nil {
					return
				}
				log.Printf("event-history-handler xreadgroup error: %v", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
				}
				continue
			}

			for _, stream := range results {
				for _, msg := range stream.Messages {
					event, err := parseStreamMessage(msg)
					if err != nil {
						log.Printf("failed to parse stream message: %v", err)
						rdb.XAck(ctx, stream.Stream, consumerGroup, msg.ID)
						continue
					}

					if event.AgentName == "" {
						rdb.XAck(ctx, stream.Stream, consumerGroup, msg.ID)
						continue
					}

					// --- metric emission ---
					model := ""
					if m, ok := event.Payload["model"].(string); ok {
						model = m
					}
					agentLabel := agentNameLabel(event.AgentName)

					switch event.Type {
					case "task_started":
						agentTasksTotal.WithLabelValues(event.Namespace, model, "started", agentLabel).Inc()
					case "task_completed":
						agentTasksTotal.WithLabelValues(event.Namespace, model, "completed", agentLabel).Inc()
						if cost, ok := event.Payload["cost_usd"].(float64); ok {
							agentTaskCostUSD.WithLabelValues(event.Namespace, model, agentLabel).Add(cost)
						}
						if dur, ok := event.Payload["duration_ms"].(float64); ok {
							agentTaskDuration.WithLabelValues(event.Namespace, model, agentLabel).Observe(dur / 1000.0)
						}
						if usage, ok := event.Payload["last_usage"].(map[string]interface{}); ok {
							for kindKey, label := range map[string]string{
								"input_tokens":                "input",
								"output_tokens":               "output",
								"cache_read_input_tokens":     "cache_read",
								"cache_creation_input_tokens": "cache_creation",
							} {
								if v, ok := usage[kindKey].(float64); ok {
									agentTaskTokens.WithLabelValues(event.Namespace, model, label, agentLabel).Add(v)
								}
							}
						}
					case "task_cancelled":
						agentTasksTotal.WithLabelValues(event.Namespace, model, "cancelled", agentLabel).Inc()
					case "error":
						agentTasksTotal.WithLabelValues(event.Namespace, model, "errored", agentLabel).Inc()
					case "tool_call":
						if toolName, ok := event.Payload["tool"].(string); ok && toolName != "" {
							if toolUseID, ok := event.Payload["id"].(string); ok && toolUseID != "" {
								toolCallTracker.markStart(event.AgentName, toolUseID, time.Now())
							}
						}
					case "tool_result":
						toolName, _ := event.Payload["tool"].(string)
						toolUseID, _ := event.Payload["id"].(string)
						isError, _ := event.Payload["is_error"].(bool)
						outcome := "success"
						if isError {
							outcome = "error"
						}
						if toolName != "" {
							agentToolInvocations.WithLabelValues(event.Namespace, toolName, outcome, agentLabel).Inc()
							if toolUseID != "" {
								if dur, ok := toolCallTracker.consumeDuration(event.AgentName, toolUseID, time.Now()); ok {
									agentToolDuration.WithLabelValues(event.Namespace, toolName, agentLabel).Observe(dur.Seconds())
								}
							}
						}
					}
					// --- end metric emission ---

					raw, err := json.Marshal(event)
					if err != nil {
						log.Printf("failed to marshal event: %v", err)
						rdb.XAck(ctx, stream.Stream, consumerGroup, msg.ID)
						continue
					}
					log.Printf("history event: %s", raw)

					historyKey := fmt.Sprintf("komputer-history:%s", event.AgentName)
					if err := rdb.RPush(ctx, historyKey, raw).Err(); err != nil {
						log.Printf("failed to append to history for %s: %v", event.AgentName, err)
					}

					taskStatus, lastMessage := mapEventToTaskStatus(event)
					if taskStatus != "" {
						sessionID := ""
						costUSD := 0.0
						var totalTokens int64
						var contextWindow int64
						if event.Type == "task_completed" {
							sessionID, _ = event.Payload["session_id"].(string)
							costUSD, _ = event.Payload["cost_usd"].(float64)
							totalTokens = extractTotalTokens(event.Payload)
							// Use last_usage (single API call) for context size, not accumulated usage.
							// Use last_usage (single API call) for context size, not accumulated usage.
							if lastUsage, ok := event.Payload["last_usage"].(map[string]interface{}); ok {
								var ctx int64
								for _, key := range []string{"input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"} {
									if v, ok := lastUsage[key].(float64); ok {
										ctx += int64(v)
									}
								}
								if ctx > 0 {
									totalTokens = ctx
								}
							}
							if cw, ok := event.Payload["context_window"].(float64); ok {
								contextWindow = int64(cw)
							}
						}

						if err := k8s.PatchAgentTaskStatus(ctx, event.Namespace, event.AgentName, taskStatus, lastMessage, sessionID, costUSD, totalTokens, contextWindow); err != nil {
							log.Printf("failed to patch task status for %s: %v", event.AgentName, err)
						}
					}

					rdb.XAck(ctx, stream.Stream, consumerGroup, msg.ID)
				}
			}
		}
	}()

	// --- Event broadcast handler (plain XREAD, no consumer group) ---
	// Every API instance reads every event and pushes to its local WebSocket clients.
	go func() {
		log.Printf("event-broadcast-handler started, tailing streams %q", streamPattern)

		lastIDs := make(map[string]string)

		for {
			if ctx.Err() != nil {
				log.Println("event-broadcast-handler shutting down")
				return
			}

			streams, err := scanStreams(ctx, rdb, streamPattern)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
				}
				continue
			}

			// Prune lastIDs for streams that no longer exist.
			active := make(map[string]struct{}, len(streams))
			for _, s := range streams {
				active[s] = struct{}{}
			}
			for k := range lastIDs {
				if _, ok := active[k]; !ok {
					delete(lastIDs, k)
				}
			}

			if len(streams) == 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Second):
				}
				continue
			}

			keys := make([]string, 0, len(streams))
			ids := make([]string, 0, len(streams))
			for _, s := range streams {
				keys = append(keys, s)
				id, ok := lastIDs[s]
				if !ok {
					// Start from the beginning of the stream so we don't miss
					// events published before we discovered it. Streams are
					// trimmed on each task_started, so this only replays
					// current-task events.
					id = "0-0"
					lastIDs[s] = id
				}
				ids = append(ids, id)
			}

			results, err := rdb.XRead(ctx, &redis.XReadArgs{
				Streams: append(keys, ids...),
				Block:   5 * time.Second,
				Count:   100,
			}).Result()
			if err != nil {
				if err == redis.Nil {
					continue
				}
				if ctx.Err() != nil {
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
				}
				continue
			}

			for _, stream := range results {
				for _, msg := range stream.Messages {
					lastIDs[stream.Stream] = msg.ID
					redisXreadMessagesTotal.Inc()

					event, err := parseStreamMessage(msg)
					if err != nil {
						continue
					}
					if event.AgentName == "" {
						continue
					}

					raw, err := json.Marshal(event)
					if err != nil {
						continue
					}

					hub.Dispatch(ctx, event.AgentName, msg.ID, raw)
				}
			}
		}
	}()

	return w
}

const consumerGroup = "komputer-workers"

// ensureConsumerGroup creates the consumer group if it doesn't exist.
// Uses "0" as the start ID so the group reads from the beginning of the stream.
func ensureConsumerGroup(ctx context.Context, rdb *redis.Client, stream string) error {
	err := rdb.XGroupCreateMkStream(ctx, stream, consumerGroup, "0").Err()
	if err != nil {
		// "BUSYGROUP" means the group already exists — not an error.
		if strings.Contains(err.Error(), "BUSYGROUP") {
			return nil
		}
		return fmt.Errorf("xgroup create %s: %w", stream, err)
	}
	return nil
}

// claimPendingMessages reads this consumer's pending messages (delivered but not ACKed)
// and processes+ACKs them for crash recovery.
func claimPendingMessages(ctx context.Context, rdb *redis.Client, stream string, consumerName string, k8s *K8sClient) {
	for {
		results, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{stream, "0"},
			Count:    100,
		}).Result()
		if err != nil {
			if err != redis.Nil {
				log.Printf("failed to read pending messages for %s: %v", stream, err)
			}
			return
		}

		for _, s := range results {
			if len(s.Messages) == 0 {
				return
			}
			for _, msg := range s.Messages {
				event, err := parseStreamMessage(msg)
				if err != nil {
					rdb.XAck(ctx, stream, consumerGroup, msg.ID)
					continue
				}
				if event.AgentName == "" {
					rdb.XAck(ctx, stream, consumerGroup, msg.ID)
					continue
				}

				raw, err := json.Marshal(event)
				if err != nil {
					rdb.XAck(ctx, stream, consumerGroup, msg.ID)
					continue
				}
				log.Printf("recovered pending event: %s", raw)

				historyKey := fmt.Sprintf("komputer-history:%s", event.AgentName)
				rdb.RPush(ctx, historyKey, raw)

				taskStatus, lastMessage := mapEventToTaskStatus(event)
				if taskStatus != "" {
					sessionID := ""
					costUSD := 0.0
					var totalTokens int64
					var contextWindow int64
					if event.Type == "task_completed" {
						sessionID, _ = event.Payload["session_id"].(string)
						costUSD, _ = event.Payload["cost_usd"].(float64)
						totalTokens = extractTotalTokens(event.Payload)
						// Use last_usage (single API call) for context size, not accumulated usage.
						if lastUsage, ok := event.Payload["last_usage"].(map[string]interface{}); ok {
							var ctx int64
							for _, key := range []string{"input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"} {
								if v, ok := lastUsage[key].(float64); ok {
									ctx += int64(v)
								}
							}
							if ctx > 0 {
								totalTokens = ctx
							}
						}
						if cw, ok := event.Payload["context_window"].(float64); ok {
							contextWindow = int64(cw)
						}
					}
					k8s.PatchAgentTaskStatus(ctx, event.Namespace, event.AgentName, taskStatus, lastMessage, sessionID, costUSD, totalTokens, contextWindow)
				}

				rdb.XAck(ctx, stream, consumerGroup, msg.ID)
			}
		}
	}
}

// scanKeys uses SCAN to discover keys matching a pattern.
func scanKeys(ctx context.Context, rdb *redis.Client, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		batch, next, err := rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// scanStreams uses SCAN to discover all per-agent event streams.
func scanStreams(ctx context.Context, rdb *redis.Client, pattern string) ([]string, error) {
	var streams []string
	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		streams = append(streams, keys...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return streams, nil
}

// parseStreamMessage converts a Redis stream message into an AgentEvent.
// The "payload" field is a JSON string that gets unmarshalled back to a map.
func parseStreamMessage(msg redis.XMessage) (AgentEvent, error) {
	agentName, ok := msg.Values["agentName"].(string)
	if !ok {
		return AgentEvent{}, fmt.Errorf("missing or invalid agentName field")
	}
	typ, ok := msg.Values["type"].(string)
	if !ok {
		return AgentEvent{}, fmt.Errorf("missing or invalid type field")
	}
	timestamp, ok := msg.Values["timestamp"].(string)
	if !ok {
		return AgentEvent{}, fmt.Errorf("missing or invalid timestamp field")
	}

	// Namespace is optional (backward-compatible with old agents).
	namespace, _ := msg.Values["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	event := AgentEvent{
		AgentName: agentName,
		Namespace: namespace,
		Type:      typ,
		Timestamp: timestamp,
	}

	payloadStr, ok := msg.Values["payload"].(string)
	if !ok {
		return AgentEvent{}, fmt.Errorf("missing or invalid payload field")
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return event, fmt.Errorf("unmarshal payload: %w", err)
	}
	event.Payload = payload

	return event, nil
}

func mapEventToTaskStatus(event AgentEvent) (taskStatus string, lastMessage string) {
	switch event.Type {
	case "task_started":
		return "InProgress", "Task started"
	case "text":
		content, _ := event.Payload["content"].(string)
		return "InProgress", truncate(content, 256)
	case "thinking":
		return "InProgress", "Thinking..."
	case "tool_call":
		tool, _ := event.Payload["tool"].(string)
		return "InProgress", truncate(fmt.Sprintf("Calling %s", tool), 256)
	case "tool_result":
		tool, _ := event.Payload["tool"].(string)
		return "InProgress", truncate(fmt.Sprintf("Got result from %s", tool), 256)
	case "task_completed":
		return "Complete", "Task completed"
	case "task_cancelled":
		return "Complete", "Task cancelled"
	case "error":
		errMsg, _ := event.Payload["error"].(string)
		return "Error", truncate(fmt.Sprintf("Error: %s", errMsg), 256)
	default:
		return "", ""
	}
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max])
	}
	return s
}

// extractTotalTokens reads all input + output tokens from a task_completed payload's
// "usage" map, including cache_read and cache_creation tokens. Returns 0 if absent or malformed.
func extractTotalTokens(payload map[string]interface{}) int64 {
	raw, ok := payload["usage"]
	if !ok || raw == nil {
		return 0
	}
	usageMap, ok := raw.(map[string]interface{})
	if !ok {
		return 0
	}
	var total int64
	for _, key := range []string{"input_tokens", "output_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"} {
		if v, ok := usageMap[key].(float64); ok {
			total += int64(v)
		}
	}
	return total
}

// extractContextTokens returns the total input size for the last turn — used as a proxy for
// context window usage. This is input_tokens + cache_read_input_tokens + cache_creation_input_tokens.
func extractContextTokens(payload map[string]interface{}) int64 {
	raw, ok := payload["usage"]
	if !ok || raw == nil {
		return 0
	}
	usageMap, ok := raw.(map[string]interface{})
	if !ok {
		return 0
	}
	var total int64
	for _, key := range []string{"input_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"} {
		if v, ok := usageMap[key].(float64); ok {
			total += int64(v)
		}
	}
	return total
}
