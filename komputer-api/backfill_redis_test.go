package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/redis/go-redis/v9"
)

// recordingHook stands in for Redis: it records every round trip (a single
// command, or a whole pipeline) and answers without touching the network.
type recordingHook struct {
	roundTrips int
	values     []string
}

func (h *recordingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, fmt.Errorf("no network in tests")
	}
}

func (h *recordingHook) record(cmd redis.Cmder) {
	args := cmd.Args()
	for _, a := range args[2:] { // RPUSH key v1 v2 ...
		switch v := a.(type) {
		case []byte:
			h.values = append(h.values, string(v))
		case string:
			h.values = append(h.values, v)
		}
	}
}

func (h *recordingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		h.roundTrips++
		h.record(cmd)
		return nil
	}
}

func (h *recordingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		h.roundTrips++
		for _, c := range cmds {
			h.record(c)
		}
		return nil
	}
}

// A long history must be written back in a few round trips, not one per event:
// each round trip waits on the network, and thousands of them add up to seconds.
func TestBackfillRedisHistory_BatchesTheWrites(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	hook := &recordingHook{}
	rdb.AddHook(hook)

	events := make([]AgentEvent, 2500)
	for i := range events {
		events[i] = AgentEvent{AgentName: "a1", Type: "text", Timestamp: fmt.Sprintf("t%04d", i)}
	}
	backfillRedisHistory(rdb, "a1", events)

	if hook.roundTrips > 2 {
		t.Errorf("round trips = %d, want at most 2 for %d events", hook.roundTrips, len(events))
	}
	if len(hook.values) != len(events) {
		t.Fatalf("pushed %d values, want %d", len(hook.values), len(events))
	}
	for i, raw := range hook.values {
		var ev AgentEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil || ev.Timestamp != events[i].Timestamp {
			t.Fatalf("value %d = %q, want event %s in order", i, raw, events[i].Timestamp)
		}
	}
}
