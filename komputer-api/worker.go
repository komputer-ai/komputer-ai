package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// AgentEvent matches the JSON schema published by the komputer-agent.
type AgentEvent struct {
	AgentName string                 `json:"agentName"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

type RedisWorkerConfig struct {
	Address  string
	Password string
	DB       int
	Queue    string
}

func StartRedisWorker(ctx context.Context, cfg RedisWorkerConfig, k8s *K8sClient, hub *Hub) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	go func() {
		log.Printf("redis worker started, consuming from queue %q at %s", cfg.Queue, cfg.Address)

		for {
			select {
			case <-ctx.Done():
				log.Println("redis worker shutting down")
				rdb.Close()
				return
			default:
			}

			result, err := rdb.BLPop(ctx, 5*time.Second, cfg.Queue).Result()
			if err != nil {
				if err == redis.Nil || err == context.Canceled {
					continue
				}
				log.Printf("redis worker error: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if len(result) < 2 {
				continue
			}

			raw := result[1]
			log.Printf("agent event: %s", raw)

			var event AgentEvent
			if err := json.Unmarshal([]byte(raw), &event); err != nil {
				log.Printf("failed to parse agent event: %v", err)
				continue
			}

			if event.AgentName == "" {
				continue
			}

			// Broadcast to WebSocket subscribers
			hub.Broadcast(event.AgentName, []byte(raw))

			taskStatus, lastMessage := mapEventToTaskStatus(event)
			if taskStatus == "" {
				continue
			}

			if err := k8s.PatchAgentTaskStatus(ctx, event.AgentName, taskStatus, lastMessage); err != nil {
				log.Printf("failed to patch task status for %s: %v", event.AgentName, err)
			}
		}
	}()
}

func mapEventToTaskStatus(event AgentEvent) (taskStatus string, lastMessage string) {
	switch event.Type {
	case "task_started":
		return "Busy", "Task started"
	case "text":
		content, _ := event.Payload["content"].(string)
		return "Busy", truncate(content, 256)
	case "thinking":
		return "Busy", "Thinking..."
	case "tool_call":
		tool, _ := event.Payload["tool"].(string)
		return "Busy", truncate(fmt.Sprintf("Calling %s", tool), 256)
	case "tool_result":
		tool, _ := event.Payload["tool"].(string)
		return "Busy", truncate(fmt.Sprintf("Got result from %s", tool), 256)
	case "task_completed":
		return "Idle", "Task completed"
	case "task_cancelled":
		return "Idle", "Task cancelled"
	case "error":
		errMsg, _ := event.Payload["error"].(string)
		return "Error", truncate(fmt.Sprintf("Error: %s", errMsg), 256)
	default:
		return "", ""
	}
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
