package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type TaskBreakdown struct {
	Index             int     `json:"index"`
	StartedAt         string  `json:"startedAt"`
	CompletedAt       string  `json:"completedAt,omitempty"`
	Instruction       string  `json:"instruction"`
	CostUSD           float64 `json:"costUSD"`
	DurationMs        int64   `json:"durationMs"`
	Turns             int     `json:"turns"`
	InputTokens       int64   `json:"inputTokens"`
	OutputTokens      int64   `json:"outputTokens"`
	CacheReadTokens   int64   `json:"cacheReadTokens"`
	CacheCreateTokens int64   `json:"cacheCreateTokens"`
	Steer             bool    `json:"steer,omitempty"`
	// EventCount is the number of history events in this task.
	EventCount int `json:"eventCount"`
	// Events is set only when the request passes events=true.
	Events []AgentEvent `json:"events,omitempty"`
}

type CostBreakdownResponse struct {
	Agent string          `json:"agent"`
	Tasks []TaskBreakdown `json:"tasks"`
	// TotalCost and TaskCount cover every task, even when limit trims Tasks.
	TotalCost float64 `json:"totalCost"`
	TaskCount int     `json:"taskCount"`
	CachedAt  string  `json:"cachedAt,omitempty"`
}

// maxEventTasks caps how many tasks keep their events when events=true.
const maxEventTasks = 50

// costBreakdownBuilder folds history events into per-task totals one event at
// a time, so a long history is never held in memory as a whole.
type costBreakdownBuilder struct {
	includeEvents bool
	limit         int // keep only the most recent N tasks; 0 keeps all

	tasks     []TaskBreakdown
	current   *TaskBreakdown
	idx       int
	taskCount int
	totalCost float64
}

func newCostBreakdownBuilder(limit int, includeEvents bool) *costBreakdownBuilder {
	if includeEvents && (limit <= 0 || limit > maxEventTasks) {
		limit = maxEventTasks
	}
	return &costBreakdownBuilder{includeEvents: includeEvents, limit: limit}
}

func (b *costBreakdownBuilder) startTask(ev AgentEvent, instruction string, steer bool) {
	b.current = &TaskBreakdown{
		Index:       b.idx,
		StartedAt:   ev.Timestamp,
		Instruction: instruction,
		Steer:       steer,
	}
	b.idx++
	b.record(ev)
}

func (b *costBreakdownBuilder) record(ev AgentEvent) {
	b.current.EventCount++
	if b.includeEvents {
		b.current.Events = append(b.current.Events, ev)
	}
}

func (b *costBreakdownBuilder) add(ev AgentEvent) {
	// Count events for the current task.
	if b.current != nil {
		b.record(ev)
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
		b.startTask(ev, instruction, steer)

	case "user_message":
		// Session-backfilled user messages act as task_started
		if b.current == nil {
			content, _ := ev.Payload["content"].(string)
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			b.startTask(ev, content, false)
		}

	// error ends a task too — including one the SDK stopped for exceeding
	// taskBudgetUSD, which carries that task's cost.
	case "task_completed", "task_cancelled", "error":
		if current := b.current; current != nil {
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
			b.taskCount++
			b.totalCost += current.CostUSD
			b.tasks = append(b.tasks, *current)
			if b.limit > 0 && len(b.tasks) > b.limit {
				b.tasks[0] = TaskBreakdown{} // release the dropped task's events
				b.tasks = b.tasks[1:]
			}
			b.current = nil
		}
	}
}

func (b *costBreakdownBuilder) result() CostBreakdownResponse {
	return CostBreakdownResponse{
		Tasks:     b.tasks,
		TotalCost: b.totalCost,
		TaskCount: b.taskCount,
	}
}

// buildCostBreakdown builds the breakdown of every task, without per-task events.
func buildCostBreakdown(events []AgentEvent) CostBreakdownResponse {
	b := newCostBreakdownBuilder(0, false)
	for _, ev := range events {
		b.add(ev)
	}
	return b.result()
}

const costBreakdownTTL = 5 * time.Minute

// getAgentCostBreakdown returns the per-task cost breakdown for an agent.
// @ID getAgentCostBreakdown
// @Summary Get agent cost breakdown
// @Description Returns per-task costs for an agent, computed from its event history. Cached for 5 minutes unless refresh=true. totalCost and taskCount always cover every task.
// @Tags agents
// @Produce json
// @Param name path string true "Agent name"
// @Param refresh query bool false "Bypass the 5-minute cache"
// @Param limit query int false "Return only the most recent N tasks (0 or unset returns all)"
// @Param events query bool false "Include each task's events (returns at most the 50 most recent tasks)"
// @Success 200 {object} CostBreakdownResponse "Cost breakdown"
// @Failure 400 {object} map[string]string "Invalid query parameter"
// @Failure 500 {object} map[string]string "Internal error"
// @Router /agents/{name}/cost [get]
func getAgentCostBreakdown(worker *RedisWorker) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		ctx := c.Request.Context()

		limit := 0
		if raw := c.Query("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a non-negative integer"})
				return
			}
			limit = n
		}
		b := newCostBreakdownBuilder(limit, c.Query("events") == "true")
		cacheKey := fmt.Sprintf("komputer-cost:v2:%s:%d:%t", name, b.limit, b.includeEvents)

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

		// Stream the history in pages and fold it into per-task totals.
		if err := ForEachAgentEvent(ctx, worker.Rdb, name, b.add); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch events: " + err.Error()})
			return
		}

		resp := b.result()
		resp.Agent = name
		resp.CachedAt = time.Now().UTC().Format(time.RFC3339)

		// Cache the result
		if raw, err := json.Marshal(resp); err == nil {
			worker.Rdb.Set(ctx, cacheKey, raw, costBreakdownTTL)
		}

		c.JSON(http.StatusOK, resp)
	}
}
