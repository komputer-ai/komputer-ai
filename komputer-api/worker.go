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

	go func() {
		defer rdb.Close()
		log.Printf("redis worker started, consuming streams %q at %s", streamPattern, cfg.Address)

		// Track last-read ID per stream. "$" means only new messages.
		lastIDs := make(map[string]string)

		for {
			if ctx.Err() != nil {
				log.Println("redis worker shutting down")
				return
			}

			// Discover active agent streams via SCAN.
			streams, err := scanStreams(ctx, rdb, streamPattern)
			if err != nil {
				if ctx.Err() != nil {
					log.Println("redis worker shutting down")
					return
				}
				log.Printf("redis worker scan error: %v", err)
				select {
				case <-ctx.Done():
					log.Println("redis worker shutting down")
					return
				case <-time.After(1 * time.Second):
				}
				continue
			}

			// Prune lastIDs for streams that no longer exist.
			activeStreams := make(map[string]struct{}, len(streams))
			for _, s := range streams {
				activeStreams[s] = struct{}{}
			}
			for k := range lastIDs {
				if _, ok := activeStreams[k]; !ok {
					delete(lastIDs, k)
				}
			}

			if len(streams) == 0 {
				// No streams yet — wait and retry.
				select {
				case <-ctx.Done():
					log.Println("redis worker shutting down")
					return
				case <-time.After(2 * time.Second):
				}
				continue
			}

			// Build XREAD args: keys then IDs.
			keys := make([]string, 0, len(streams))
			ids := make([]string, 0, len(streams))
			for _, s := range streams {
				keys = append(keys, s)
				id, ok := lastIDs[s]
				if !ok {
					// Check Redis for a saved cursor from a previous run.
					cursorKey := fmt.Sprintf("komputer-worker-last-read:%s", s)
					saved, err := rdb.Get(ctx, cursorKey).Result()
					if err == nil && saved != "" {
						id = saved
					} else {
						id = "$" // No cursor — only read new messages
					}
				}
				ids = append(ids, id)
			}

			xreadArgs := &redis.XReadArgs{
				Streams: append(keys, ids...),
				Block:   5 * time.Second,
				Count:   100,
			}

			results, err := rdb.XRead(ctx, xreadArgs).Result()
			if err != nil {
				if err == redis.Nil {
					continue
				}
				if ctx.Err() != nil {
					log.Println("redis worker shutting down")
					return
				}
				log.Printf("redis worker xread error: %v", err)
				select {
				case <-ctx.Done():
					log.Println("redis worker shutting down")
					return
				case <-time.After(1 * time.Second):
				}
				continue
			}

			for _, stream := range results {
				for _, msg := range stream.Messages {
					lastIDs[stream.Stream] = msg.ID

					event, err := parseStreamMessage(msg)
					if err != nil {
						log.Printf("failed to parse stream message: %v", err)
						continue
					}

					if event.AgentName == "" {
						continue
					}

					// Re-serialize to JSON for WebSocket broadcast (payload as object, not string).
					raw, err := json.Marshal(event)
					if err != nil {
						log.Printf("failed to marshal event for broadcast: %v", err)
						continue
					}
					log.Printf("agent event: %s", raw)

					hub.Broadcast(event.AgentName, raw)

					// Append to persistent history list for GET /events.
					historyKey := fmt.Sprintf("komputer-history:%s", event.AgentName)
					if err := rdb.RPush(ctx, historyKey, raw).Err(); err != nil {
						log.Printf("failed to append to history for %s: %v", event.AgentName, err)
					}

					taskStatus, lastMessage := mapEventToTaskStatus(event)
					if taskStatus == "" {
						continue
					}

					sessionID := ""
					costUSD := 0.0
					if event.Type == "task_completed" {
						sessionID, _ = event.Payload["session_id"].(string)
						costUSD, _ = event.Payload["cost_usd"].(float64)
					}

					if err := k8s.PatchAgentTaskStatus(ctx, event.Namespace, event.AgentName, taskStatus, lastMessage, sessionID, costUSD); err != nil {
						log.Printf("failed to patch task status for %s: %v", event.AgentName, err)
					}
				}

				// Save cursor so we resume from here on restart
				cursorKey := fmt.Sprintf("komputer-worker-last-read:%s", stream.Stream)
				if err := rdb.Set(ctx, cursorKey, lastIDs[stream.Stream], 0).Err(); err != nil {
					log.Printf("failed to save cursor for %s: %v", stream.Stream, err)
				}
			}
		}
	}()

	return w
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

// GetAgentEvents reads events from the agent's persistent history list.
// Returns the last `limit` events in chronological order.
func GetAgentEvents(ctx context.Context, rdb *redis.Client, agentName string, limit int64, streamPrefix string) ([]AgentEvent, error) {
	historyKey := fmt.Sprintf("komputer-history:%s", agentName)

	// LRANGE with negative indices gets the last N items.
	vals, err := rdb.LRange(ctx, historyKey, -limit, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("lrange: %w", err)
	}

	events := make([]AgentEvent, 0, len(vals))
	for _, raw := range vals {
		var ev AgentEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			log.Printf("skipping malformed history entry: %v", err)
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

// DeleteAgentStream removes an agent's event stream and history from Redis.
func DeleteAgentStream(ctx context.Context, rdb *redis.Client, agentName string, streamPrefix string) error {
	streamKey := fmt.Sprintf("%s:%s", streamPrefix, agentName)
	historyKey := fmt.Sprintf("komputer-history:%s", agentName)
	if err := rdb.Del(ctx, streamKey, historyKey).Err(); err != nil {
		return fmt.Errorf("delete stream/history for %s: %w", agentName, err)
	}
	return nil
}

// ListAgentStreams returns agent names that have active event streams.
func ListAgentStreams(ctx context.Context, rdb *redis.Client, streamPrefix string) ([]string, error) {
	streams, err := scanStreams(ctx, rdb, streamPrefix+":*")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(streams))
	for _, s := range streams {
		name := strings.TrimPrefix(s, streamPrefix+":")
		names = append(names, name)
	}
	return names, nil
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
