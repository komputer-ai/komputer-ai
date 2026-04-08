package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// GetAgentEvents reads events from the agent's persistent history list.
// Returns the last `limit` events in chronological order.
func GetAgentEvents(ctx context.Context, rdb *redis.Client, agentName string, limit int64, streamPrefix string) ([]AgentEvent, error) {
	return GetAgentEventsBefore(ctx, rdb, agentName, limit, "", streamPrefix)
}

// GetAgentEventsBefore returns up to `limit` events for the agent.
// When `before` is non-empty it must be an RFC-3339 timestamp; only events
// before that value are returned. Uses binary search on the Redis list index
// to avoid fetching the entire list on each page.
func GetAgentEventsBefore(ctx context.Context, rdb *redis.Client, agentName string, limit int64, before string, streamPrefix string) ([]AgentEvent, error) {
	historyKey := fmt.Sprintf("komputer-history:%s", agentName)

	if before == "" {
		// No cursor — return the last `limit` events.
		// When limit <= 0, fetch all events.
		start := -limit
		if limit <= 0 {
			start = 0
		}
		vals, err := rdb.LRange(ctx, historyKey, start, -1).Result()
		if err != nil {
			return nil, fmt.Errorf("lrange: %w", err)
		}
		events := make([]AgentEvent, 0, len(vals))
		for _, raw := range vals {
			var ev AgentEvent
			if err := json.Unmarshal([]byte(raw), &ev); err != nil {
				continue
			}
			events = append(events, ev)
		}
		return events, nil
	}

	// Cursor-based pagination: find the index of the first event at or after
	// the `before` timestamp using binary search, then fetch `limit` events
	// ending just before that index.
	beforeTime, err := time.Parse(time.RFC3339Nano, before)
	if err != nil {
		return nil, fmt.Errorf("invalid before timestamp: %w", err)
	}

	total, err := rdb.LLen(ctx, historyKey).Result()
	if err != nil || total == 0 {
		return []AgentEvent{}, nil
	}

	// Binary search for the cutoff index: the first event whose timestamp >= before.
	lo, hi := int64(0), total
	for lo < hi {
		mid := (lo + hi) / 2
		raw, err := rdb.LIndex(ctx, historyKey, mid).Result()
		if err != nil {
			lo = mid + 1
			continue
		}
		var ev AgentEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			lo = mid + 1
			continue
		}
		evTime, err := time.Parse(time.RFC3339Nano, ev.Timestamp)
		if err != nil {
			lo = mid + 1
			continue
		}
		if evTime.Before(beforeTime) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	// lo is now the index of the first event >= before.
	// We want `limit` events ending at lo-1.
	end := lo - 1
	start := end - limit + 1
	if start < 0 {
		start = 0
	}
	if end < 0 {
		return []AgentEvent{}, nil
	}

	vals, err := rdb.LRange(ctx, historyKey, start, end).Result()
	if err != nil {
		return nil, fmt.Errorf("lrange: %w", err)
	}

	events := make([]AgentEvent, 0, len(vals))
	for _, raw := range vals {
		var ev AgentEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

// GetAgentEventsAfter returns up to `limit` events strictly after a timestamp.
func GetAgentEventsAfter(ctx context.Context, rdb *redis.Client, agentName string, limit int64, after string, streamPrefix string) ([]AgentEvent, error) {
	historyKey := fmt.Sprintf("komputer-history:%s", agentName)

	afterTime, err := time.Parse(time.RFC3339Nano, after)
	if err != nil {
		return nil, fmt.Errorf("invalid after timestamp: %w", err)
	}

	total, err := rdb.LLen(ctx, historyKey).Result()
	if err != nil || total == 0 {
		return []AgentEvent{}, nil
	}

	// Binary search for the first event > after.
	lo, hi := int64(0), total
	for lo < hi {
		mid := (lo + hi) / 2
		raw, err := rdb.LIndex(ctx, historyKey, mid).Result()
		if err != nil {
			lo = mid + 1
			continue
		}
		var ev AgentEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			lo = mid + 1
			continue
		}
		evTime, err := time.Parse(time.RFC3339Nano, ev.Timestamp)
		if err != nil {
			lo = mid + 1
			continue
		}
		if !evTime.After(afterTime) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	// lo is the first event strictly after afterTime.
	end := lo + limit - 1
	if end >= total {
		end = total - 1
	}
	if lo > end {
		return []AgentEvent{}, nil
	}

	vals, err := rdb.LRange(ctx, historyKey, lo, end).Result()
	if err != nil {
		return nil, fmt.Errorf("lrange: %w", err)
	}

	events := make([]AgentEvent, 0, len(vals))
	for _, raw := range vals {
		var ev AgentEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

// GetAgentEventsAround returns events centered on a timestamp.
// Uses binary search to find the target, then fetches limit/2 before and limit/2 after.
func GetAgentEventsAround(ctx context.Context, rdb *redis.Client, agentName string, limit int64, around string, streamPrefix string) ([]AgentEvent, error) {
	historyKey := fmt.Sprintf("komputer-history:%s", agentName)

	aroundTime, err := time.Parse(time.RFC3339Nano, around)
	if err != nil {
		return nil, fmt.Errorf("invalid around timestamp: %w", err)
	}

	total, err := rdb.LLen(ctx, historyKey).Result()
	if err != nil || total == 0 {
		return []AgentEvent{}, nil
	}

	// Binary search for the index closest to the target timestamp.
	lo, hi := int64(0), total
	for lo < hi {
		mid := (lo + hi) / 2
		raw, err := rdb.LIndex(ctx, historyKey, mid).Result()
		if err != nil {
			lo = mid + 1
			continue
		}
		var ev AgentEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			lo = mid + 1
			continue
		}
		evTime, err := time.Parse(time.RFC3339Nano, ev.Timestamp)
		if err != nil {
			lo = mid + 1
			continue
		}
		if evTime.Before(aroundTime) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	// lo is the index of the first event >= around.
	// Bias toward events after the target (tasks have more content after the start).
	before := limit / 4
	after := limit - before
	start := lo - before
	end := lo + after
	if start < 0 {
		start = 0
	}
	if end >= total {
		end = total - 1
	}

	vals, err := rdb.LRange(ctx, historyKey, start, end).Result()
	if err != nil {
		return nil, fmt.Errorf("lrange: %w", err)
	}

	events := make([]AgentEvent, 0, len(vals))
	for _, raw := range vals {
		var ev AgentEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
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
