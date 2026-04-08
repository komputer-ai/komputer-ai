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
func fetchSessionEvents(ctx context.Context, k8s *K8sClient, ns, podName, sessionID, agentName string, limit int64) []AgentEvent {
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

	return convertSessionJSONL(raw, agentName, limit)
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
func convertSessionJSONL(raw []byte, agentName string, limit int64) []AgentEvent {
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
	var lastAssistantTimestamp string
	var turnInputTokens, turnOutputTokens, turnCacheRead, turnCacheCreation float64
	var turnAssistantMessages int
	var pendingCancel *AgentEvent // deferred cancel — emitted only if not followed by a steer
	firstTurn := true

	emitTaskCompleted := func() {
		if lastAssistantTimestamp == "" {
			return
		}
		payload := map[string]interface{}{
			"num_turns": turnAssistantMessages,
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
		turnInputTokens = 0
		turnOutputTokens = 0
		turnCacheRead = 0
		turnCacheCreation = 0
		turnAssistantMessages = 0
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
				// Flush pending cancel before the new turn — it was a real cancel, not a steer.
				if pendingCancel != nil {
					events = append(events, *pendingCancel)
					pendingCancel = nil
				}
				emitTaskCompleted()
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
			// Accumulate usage.
			if usage, ok := msg["usage"].(map[string]interface{}); ok {
				if v, ok := usage["input_tokens"].(float64); ok {
					turnInputTokens += v
				}
				if v, ok := usage["output_tokens"].(float64); ok {
					turnOutputTokens += v
				}
				if v, ok := usage["cache_read_input_tokens"].(float64); ok {
					turnCacheRead += v
				}
				if v, ok := usage["cache_creation_input_tokens"].(float64); ok {
					turnCacheCreation += v
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

	// Return last N events.
	if limit > 0 && int64(len(events)) > limit {
		events = events[int64(len(events))-limit:]
	}
	return events
}
