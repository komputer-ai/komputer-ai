// backfill.go — Session JSONL → Redis event backfill.
//
// When Redis history is empty (wiped or first access), we read the agent's
// Claude session JSONL file and convert it into the same AgentEvent structs
// that the real-time Redis worker produces. The converted events are written
// to Redis so subsequent requests are fast.
//
// This file is the SINGLE place where JSONL→event conversion happens.
// All filtering, restructuring, and normalization of session data lives here.
// The query layer (history.go) assumes Redis contains clean, structured events.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

// fetchSessionEvents reads Claude session JSONL from the agent pod and parses
// it into AgentEvents. Uses HTTP to agent pod, falls back to exec (instant in LOCAL mode).
func fetchSessionEvents(ctx context.Context, k8s *K8sClient, ns, podName, sessionID, agentName string, limit int64, model string, totalCostUSD float64) []AgentEvent {
	var raw []byte

	// Try HTTP first (skipped instantly in LOCAL mode).
	podIP, _ := k8s.GetAgentPodIP(ctx, ns, podName)
	if podIP != "" {
		historyPath := fmt.Sprintf("/history?limit=%d&session_id=%s", limit, sessionID)
		respBody, err := k8s.getFromAgent(ctx, podIP, historyPath)
		if err == nil {
			var result struct {
				Events []struct {
					Type      string                 `json:"type"`
					Timestamp string                 `json:"timestamp"`
					Payload   map[string]interface{} `json:"payload"`
				} `json:"events"`
			}
			if json.Unmarshal(respBody, &result) == nil {
				var events []AgentEvent
				for _, e := range result.Events {
					events = append(events, AgentEvent{
						AgentName: agentName,
						Type:      e.Type,
						Timestamp: e.Timestamp,
						Payload:   e.Payload,
					})
				}
				if limit > 0 && int64(len(events)) > limit {
					events = events[int64(len(events))-limit:]
				}
				return events
			}
		}
	}

	// Exec fallback: cat the JSONL file directly.
	sessionFilePath := fmt.Sprintf("/workspace/.claude/projects/-workspace/%s.jsonl", sessionID)
	var err error
	raw, err = k8s.execInPodWithOutput(ctx, ns, podName, "cat", sessionFilePath)
	if err != nil {
		log.Printf("session JSONL read failed for %s: %v", agentName, err)
		return nil
	}

	return convertSessionJSONL(raw, agentName, limit, model, totalCostUSD)
}

// backfillRedisHistory writes converted session events into Redis history
// so subsequent requests are fast.
func backfillRedisHistory(rdb *redis.Client, agentName string, events []AgentEvent) {
	ctx := context.Background()
	historyKey := fmt.Sprintf("komputer-history:%s", agentName)

	for _, event := range events {
		raw, err := json.Marshal(event)
		if err != nil {
			continue
		}
		rdb.RPush(ctx, historyKey, raw)
	}
	log.Printf("backfilled %d events from session to Redis for %s", len(events), agentName)
}

// ---------------------------------------------------------------------------
// convertSessionJSONL — the central JSONL→AgentEvent converter.
//
// Converts raw Claude session JSONL bytes into the same AgentEvent structs
// that the real-time Redis worker produces. This is the ONLY place where
// session data is interpreted and normalized.
//
// Conversions performed:
//   - queue-operation enqueue → task boundary detection → task_completed events
//   - role=user text blocks → user_message events (with system prompt stripping)
//   - role=user tool_result blocks → merged into preceding tool_call events
//   - role=assistant text blocks → text events
//   - role=assistant thinking blocks → thinking events
//   - role=assistant tool_use blocks → tool_result events (output merged later)
//   - Token usage accumulated per turn for task_completed stats
//
// Filtering performed (JSONL artifacts not present in real-time events):
//   - "[Request interrupted by user]" → converted to task_cancelled event
//   - "This session is being continued..." → converted to compaction event
//   - <ide_*>, <system-reminder*>, <task-notification*> → skipped
//   - "No response requested." → skipped (stub response to interruption)
//   - Compaction summaries ("read the full transcript at:") → skipped
//   - Background task notifications ("*(Background task completed") → skipped
//   - System prompt prefix stripped from first user message per turn
// ---------------------------------------------------------------------------
// outputRatePerM returns the output token price per 1M tokens for a model.
func outputRatePerM(model string) float64 {
	switch {
	case strings.Contains(model, "opus"):
		return 75.0
	case strings.Contains(model, "haiku"):
		return 4.0
	default:
		return 15.0
	}
}

// estimateCostUSD calculates dollar cost from token counts and model pricing.
func estimateCostUSD(model string, inputTokens, outputTokens, cacheRead, cacheCreate float64) float64 {
	// Pricing per 1M tokens (as of 2026-04)
	var inputRate, outputRate, cacheReadRate, cacheCreateRate float64
	switch {
	case strings.Contains(model, "opus"):
		inputRate, outputRate, cacheReadRate, cacheCreateRate = 15.0, 75.0, 1.50, 18.75
	case strings.Contains(model, "haiku"):
		inputRate, outputRate, cacheReadRate, cacheCreateRate = 0.80, 4.0, 0.08, 1.0
	default: // sonnet
		inputRate, outputRate, cacheReadRate, cacheCreateRate = 3.0, 15.0, 0.30, 3.75
	}
	return (inputTokens*inputRate + outputTokens*outputRate +
		cacheRead*cacheReadRate + cacheCreate*cacheCreateRate) / 1_000_000
}

func convertSessionJSONL(raw []byte, agentName string, limit int64, model string, totalCostUSD float64) []AgentEvent {
	var events []AgentEvent

	// Track tool_use blocks by ID so we can merge output from tool_result entries.
	type toolUseInfo struct {
		name      string
		input     interface{}
		timestamp string
		index     int // position in events slice
	}
	pendingTools := map[string]*toolUseInfo{}

	// Track per-turn stats for task_completed events.
	// Cost is calculated per API call and summed (each call has its own billing).
	var lastAssistantTimestamp string
	var turnCostUSD float64          // accumulated cost across all API calls in the turn
	var turnInputTokens, turnOutputTokens, turnCacheRead, turnCacheCreation float64 // for display
	var turnAssistantMessages int
	var prevCallInput, prevCallCacheRead, prevCallCacheCreate float64 // detect same API call
	var pendingCancel *AgentEvent // deferred cancel — emitted only if not followed by a steer
	var lastToolWasSkill bool      // true if the last assistant tool_use was "Skill" — next user message is skill content, skip it
	firstTurn := true

	emitTaskCompleted := func() {
		if lastAssistantTimestamp == "" {
			return
		}
		payload := map[string]interface{}{
			"num_turns": turnAssistantMessages,
			"cost_usd":  turnCostUSD,
			"usage": map[string]interface{}{
				"input_tokens":                int64(turnInputTokens),
				"output_tokens":               int64(turnOutputTokens),
				"cache_read_input_tokens":     int64(turnCacheRead),
				"cache_creation_input_tokens": int64(turnCacheCreation),
			},
		}
		events = append(events, AgentEvent{
			AgentName: agentName,
			Type:      "task_completed",
			Timestamp: lastAssistantTimestamp,
			Payload:   payload,
		})
		lastAssistantTimestamp = ""
		turnCostUSD = 0
		turnInputTokens = 0
		turnOutputTokens = 0
		turnCacheRead = 0
		turnCacheCreation = 0
		turnAssistantMessages = 0
		prevCallInput = 0
		prevCallCacheRead = 0
		prevCallCacheCreate = 0
	}

	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Detect turn boundaries via queue-operation enqueue.
		entryType, _ := entry["type"].(string)
		if entryType == "queue-operation" {
			op, _ := entry["operation"].(string)
			if op == "enqueue" && !firstTurn {
				if pendingCancel != nil {
					// Task was cancelled — emit cancel instead of task_completed.
					events = append(events, *pendingCancel)
					pendingCancel = nil
					// Reset turn accumulators without emitting task_completed.
					lastAssistantTimestamp = ""
					turnCostUSD = 0
					turnInputTokens = 0
					turnOutputTokens = 0
					turnCacheRead = 0
					turnCacheCreation = 0
					turnAssistantMessages = 0
		prevCallInput = 0
		prevCallCacheRead = 0
		prevCallCacheCreate = 0
				} else {
					emitTaskCompleted()
				}
			}
			firstTurn = false
			continue
		}

		msg, _ := entry["message"].(map[string]interface{})
		if msg == nil {
			continue
		}
		role, _ := msg["role"].(string)
		timestamp, _ := entry["timestamp"].(string)

		// ── User messages ──────────────────────────────────────────────
		if role == "user" {
			content := msg["content"]
			var text string
			switch v := content.(type) {
			case string:
				text = v
			case []interface{}:
				for _, block := range v {
					b, ok := block.(map[string]interface{})
					if !ok {
						continue
					}
					btype, _ := b["type"].(string)
					if btype == "text" {
						if t, ok := b["text"].(string); ok {
							text += t + " "
						}
					} else if btype == "tool_result" {
						// Merge output into the matching tool_call event.
						toolUseID, _ := b["tool_use_id"].(string)
						output := ""
						if c, ok := b["content"].(string); ok {
							output = c
						} else if c, ok := b["content"].([]interface{}); ok && len(c) > 0 {
							if first, ok := c[0].(map[string]interface{}); ok {
								output, _ = first["text"].(string)
							}
						}
						if len(output) > 500 {
							output = output[:500] + "..."
						}
						if info, ok := pendingTools[toolUseID]; ok {
							events[info.index].Payload["output"] = map[string]interface{}{
								"stdout": output,
								"stderr": "",
							}
							events[info.index].Type = "tool_result"
							delete(pendingTools, toolUseID)
						}
					}
				}
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}

			// --- Convert or filter JSONL-specific user messages ---

			// Skip skill content injected as user message after a Skill tool_use.
			if lastToolWasSkill {
				lastToolWasSkill = false
				continue
			}

			// Interruptions: defer decision — if next user message follows immediately
			// (steer), skip the cancel. If a new turn starts or file ends, emit cancel.
			if strings.HasPrefix(text, "[Request interrupted by user") {
				pendingCancel = &AgentEvent{
					AgentName: agentName,
					Type:      "task_cancelled",
					Timestamp: timestamp,
					Payload:   map[string]interface{}{"reason": "Cancelled by user"},
				}
				continue
			}
			// A real user message after an interrupt means it was a steer — drop the cancel.
			if pendingCancel != nil {
				pendingCancel = nil
			}
			// Compaction summaries → compaction indicator
			if strings.HasPrefix(text, "This session is being continued from a previous conversation") {
				events = append(events, AgentEvent{
					AgentName: agentName,
					Type:      "compaction",
					Timestamp: timestamp,
					Payload:   map[string]interface{}{},
				})
				continue
			}
			// Skip: IDE context, system reminders, task notifications
			if strings.HasPrefix(text, "<ide_") ||
				strings.HasPrefix(text, "<system-reminder") ||
				strings.HasPrefix(text, "<task-notification>") {
				continue
			}
			// Strip system prompt prefix (joined by "\n\n").
			if parts := strings.Split(text, "\n\n"); len(parts) > 1 {
				text = strings.TrimSpace(parts[len(parts)-1])
			}
			// After stripping, re-check for system content.
			if text == "" ||
				strings.HasPrefix(text, "<ide_") ||
				strings.HasPrefix(text, "<system-reminder") ||
				strings.HasPrefix(text, "<task-notification>") {
				continue
			}

			events = append(events, AgentEvent{
				AgentName: agentName,
				Type:      "user_message",
				Timestamp: timestamp,
				Payload:   map[string]interface{}{"content": text},
			})

		// ── Assistant messages ──────────────────────────────────────────
		} else if role == "assistant" {
			lastAssistantTimestamp = timestamp
			turnAssistantMessages++
			// Calculate cost per unique API call. The JSONL logs thinking + response
			// as separate entries with identical input/cache tokens. Only count
			// input/cache cost when the context changes (new API call). Always count output.
			if usage, ok := msg["usage"].(map[string]interface{}); ok {
				var inp, out, cr, cc float64
				if v, ok := usage["input_tokens"].(float64); ok {
					inp = v
				}
				if v, ok := usage["output_tokens"].(float64); ok {
					out = v
					turnOutputTokens += out
				}
				if v, ok := usage["cache_read_input_tokens"].(float64); ok {
					cr = v
				}
				if v, ok := usage["cache_creation_input_tokens"].(float64); ok {
					cc = v
				}
				// Detect new API call: input context changed from previous message.
				newCall := inp != prevCallInput || cr != prevCallCacheRead || cc != prevCallCacheCreate
				prevCallInput = inp
				prevCallCacheRead = cr
				prevCallCacheCreate = cc
				// Use per-message model from JSONL (may differ from CR spec).
				msgModel, _ := msg["model"].(string)
				if msgModel == "" {
					msgModel = model // fallback to CR spec model
				}
				if newCall {
					turnInputTokens += inp
					turnCacheRead += cr
					turnCacheCreation += cc
					turnCostUSD += estimateCostUSD(msgModel, inp, out, cr, cc)
				} else {
					// Same API call — only output tokens are new.
					turnCostUSD += (out * outputRatePerM(msgModel)) / 1_000_000
				}
			}
			content, _ := msg["content"].([]interface{})
			if content == nil {
				if s, ok := msg["content"].(string); ok && s != "" {
					events = append(events, AgentEvent{
						AgentName: agentName,
						Type:      "text",
						Timestamp: timestamp,
						Payload:   map[string]interface{}{"content": s},
					})
				}
				continue
			}
			for _, block := range content {
				b, ok := block.(map[string]interface{})
				if !ok {
					continue
				}
				btype, _ := b["type"].(string)
				switch btype {
				case "text":
					text, _ := b["text"].(string)
					if text == "" {
						break
					}
					// Skip JSONL-specific assistant artifacts.
					if strings.Contains(text, "read the full transcript at:") ||
						strings.Contains(text, "Continue the conversation from where it left off") ||
						strings.HasPrefix(text, "*(Background task completed") ||
						strings.HasPrefix(text, "*(Duplicate notification") ||
						text == "No response requested." {
						break
					}
					events = append(events, AgentEvent{
						AgentName: agentName,
						Type:      "text",
						Timestamp: timestamp,
						Payload:   map[string]interface{}{"content": text},
					})
				case "thinking":
					thinking, _ := b["thinking"].(string)
					if thinking != "" {
						events = append(events, AgentEvent{
							AgentName: agentName,
							Type:      "thinking",
							Timestamp: timestamp,
							Payload:   map[string]interface{}{"content": thinking},
						})
					}
				case "tool_use":
					toolID, _ := b["id"].(string)
					toolName, _ := b["name"].(string)
					if toolName == "Skill" {
						lastToolWasSkill = true
					}
					idx := len(events)
					events = append(events, AgentEvent{
						AgentName: agentName,
						Type:      "tool_result",
						Timestamp: timestamp,
						Payload: map[string]interface{}{
							"tool":  toolName,
							"name":  toolName,
							"input": b["input"],
						},
					})
					if toolID != "" {
						pendingTools[toolID] = &toolUseInfo{name: toolName, input: b["input"], timestamp: timestamp, index: idx}
					}
				}
			}
		}
	}

	// Flush pending cancel at end of file — it was a real cancel.
	if pendingCancel != nil {
		events = append(events, *pendingCancel)
		pendingCancel = nil
	}
	// Emit task_completed for the final turn.
	emitTaskCompleted()

	// Scale estimated costs proportionally to match the CR's actual total.
	// The estimation from token counts is approximate; the CR total is authoritative.
	if totalCostUSD > 0 {
		var estimatedTotal float64
		for _, ev := range events {
			if ev.Type == "task_completed" {
				if c, ok := ev.Payload["cost_usd"].(float64); ok {
					estimatedTotal += c
				}
			}
		}
		if estimatedTotal > 0 {
			scale := totalCostUSD / estimatedTotal
			for i := range events {
				if events[i].Type == "task_completed" {
					if c, ok := events[i].Payload["cost_usd"].(float64); ok {
						events[i].Payload["cost_usd"] = c * scale
					}
				}
			}
		}
	}

	// Return last N events.
	if limit > 0 && int64(len(events)) > limit {
		events = events[int64(len(events))-limit:]
	}
	return events
}
