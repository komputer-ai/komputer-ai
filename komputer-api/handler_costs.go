package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type TaskBreakdown struct {
	Index             int          `json:"index"`
	StartedAt         string       `json:"startedAt"`
	CompletedAt       string       `json:"completedAt,omitempty"`
	Instruction       string       `json:"instruction"`
	CostUSD           float64      `json:"costUSD"`
	DurationMs        int64        `json:"durationMs"`
	Turns             int          `json:"turns"`
	InputTokens       int64        `json:"inputTokens"`
	OutputTokens      int64        `json:"outputTokens"`
	CacheReadTokens   int64        `json:"cacheReadTokens"`
	CacheCreateTokens int64        `json:"cacheCreateTokens"`
	Steer             bool         `json:"steer,omitempty"`
	Events            []AgentEvent `json:"events"`
}

type CostBreakdownResponse struct {
	Agent     string          `json:"agent"`
	Tasks     []TaskBreakdown `json:"tasks"`
	TotalCost float64         `json:"totalCost"`
	TaskCount int             `json:"taskCount"`
	CachedAt  string          `json:"cachedAt,omitempty"`
}

func buildCostBreakdown(events []AgentEvent) CostBreakdownResponse {
	var tasks []TaskBreakdown
	var current *TaskBreakdown
	idx := 0

	for _, ev := range events {
		// Collect events for the current task.
		if current != nil {
			current.Events = append(current.Events, ev)
		}

		switch ev.Type {
		case "task_started":
			instruction, _ := ev.Payload["instructions"].(string)
			if instruction == "" {
				if content, ok := ev.Payload["content"].(string); ok {
					instruction = content
				}
			}
			if len(instruction) > 200 {
				instruction = instruction[:200] + "..."
			}
			steer, _ := ev.Payload["steer"].(bool)
			current = &TaskBreakdown{
				Index:       idx,
				StartedAt:   ev.Timestamp,
				Instruction: instruction,
				Steer:       steer,
				Events:      []AgentEvent{ev},
			}
			idx++

		case "user_message":
			// Session-backfilled user messages act as task_started
			if current == nil {
				content, _ := ev.Payload["content"].(string)
				if len(content) > 200 {
					content = content[:200] + "..."
				}
				current = &TaskBreakdown{
					Index:       idx,
					StartedAt:   ev.Timestamp,
					Instruction: content,
					Events:      []AgentEvent{ev},
				}
				idx++
			}

		case "task_completed":
			if current != nil {
				current.CompletedAt = ev.Timestamp
				if costRaw, ok := ev.Payload["cost_usd"].(float64); ok {
					current.CostUSD = costRaw
				} else if costRaw, ok := ev.Payload["costUSD"].(float64); ok {
					current.CostUSD = costRaw
				}
				if dur, ok := ev.Payload["duration_ms"].(float64); ok {
					current.DurationMs = int64(dur)
				} else if dur, ok := ev.Payload["duration"].(float64); ok {
					current.DurationMs = int64(dur)
				}
				if turns, ok := ev.Payload["turns"].(float64); ok {
					current.Turns = int(turns)
				} else if turns, ok := ev.Payload["num_turns"].(float64); ok {
					current.Turns = int(turns)
				}
				if usage, ok := ev.Payload["usage"].(map[string]interface{}); ok {
					if v, ok := usage["input_tokens"].(float64); ok {
						current.InputTokens = int64(v)
					}
					if v, ok := usage["output_tokens"].(float64); ok {
						current.OutputTokens = int64(v)
					}
					if v, ok := usage["cache_read_input_tokens"].(float64); ok {
						current.CacheReadTokens = int64(v)
					}
					if v, ok := usage["cache_creation_input_tokens"].(float64); ok {
						current.CacheCreateTokens = int64(v)
					}
				}
				tasks = append(tasks, *current)
				current = nil
			}
		}
	}

	var totalCost float64
	for _, t := range tasks {
		totalCost += t.CostUSD
	}

	return CostBreakdownResponse{
		Tasks:     tasks,
		TotalCost: totalCost,
		TaskCount: len(tasks),
	}
}

const costBreakdownTTL = 5 * time.Minute

func getAgentCostBreakdown(worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		cacheKey := fmt.Sprintf("komputer-cost:%s", name)
		ctx := c.Request.Context()

		// Check cache first (skip if ?refresh=true)
		if c.Query("refresh") != "true" {
			if cached, err := worker.Rdb.Get(ctx, cacheKey).Result(); err == nil {
				var resp CostBreakdownResponse
				if json.Unmarshal([]byte(cached), &resp) == nil {
					c.JSON(http.StatusOK, resp)
					return
				}
			}
		}

		// Fetch all events from history (limit=0 fetches all)
		events, err := GetAgentEventsBefore(ctx, worker.Rdb, name, 0, "", worker.StreamPrefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch events: " + err.Error()})
			return
		}

		resp := buildCostBreakdown(events)
		resp.Agent = name
		resp.CachedAt = time.Now().UTC().Format(time.RFC3339)

		// Cache the result
		if raw, err := json.Marshal(resp); err == nil {
			worker.Rdb.Set(ctx, cacheKey, raw, costBreakdownTTL)
		}

		c.JSON(http.StatusOK, resp)
	}
}

