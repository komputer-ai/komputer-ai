package main

import (
	"context"
	"sync"
	"time"
)

// activityCoalesceWindow bounds how often a running task's progress events write the
// agent CR. Each streamed text/thinking/tool event used to cost a GET and a status
// PATCH; within this window, repeat progress events for an agent whose task status
// is unchanged are dropped instead. The cost is that Status.LastTaskMessage and
// Status.LastActivityAt lag the stream by up to this long — the live stream reaches
// clients over the WebSocket hub regardless, and the operator's sleepTTL idle clock
// is measured in minutes and only consulted once the task is no longer in progress.
const activityCoalesceWindow = 30 * time.Second

// progressCoalescer remembers, per agent, the task status this API replica last
// wrote and when. It is per-replica and in-memory on purpose: with N replicas the
// bound becomes N writes per window, and a restart just costs one extra write.
type progressCoalescer struct {
	mu   sync.Mutex
	last map[string]progressWrite
}

type progressWrite struct {
	taskStatus string
	at         time.Time
}

// coalescibleEvent reports whether an event type is mid-task progress. Everything
// else — task_started, task_completed, task_cancelled, error, compaction — changes
// the task state or carries accounting, and is always written.
func coalescibleEvent(eventType string) bool {
	switch eventType {
	case "text", "thinking", "tool_call", "tool_result":
		return true
	}
	return false
}

func progressKey(ns, agentName string) string { return ns + "/" + agentName }

// skip reports whether a write for this event can be dropped: it is a progress event,
// it would leave TaskStatus where this replica last wrote it, and that write was less
// than activityCoalesceWindow ago. Comparing the status (not just the time) means a
// progress event that flips Compacting back to InProgress is never dropped.
func (c *progressCoalescer) skip(ns, agentName, eventType, taskStatus string, now time.Time) bool {
	if !coalescibleEvent(eventType) {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	w, ok := c.last[progressKey(ns, agentName)]
	return ok && w.taskStatus == taskStatus && now.Sub(w.at) < activityCoalesceWindow
}

func (c *progressCoalescer) record(ns, agentName, taskStatus string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.last == nil {
		c.last = make(map[string]progressWrite)
	}
	c.last[progressKey(ns, agentName)] = progressWrite{taskStatus: taskStatus, at: now}
}

// forget drops the remembered write for an agent. Called wherever the API changes
// TaskStatus out of band (wake), so the next event is compared against reality.
func (c *progressCoalescer) forget(ns, agentName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.last, progressKey(ns, agentName))
}

// PatchAgentTaskEvent is the worker's entry point for an agent event: it applies the
// progress coalescing above, then defers to PatchAgentTaskStatus for the write.
func (k *K8sClient) PatchAgentTaskEvent(ctx context.Context, eventType, ns, agentName, taskStatus, lastMessage, sessionID string, costUSD float64, totalTokens int64, contextWindow int64) error {
	now := time.Now()
	if k.progress.skip(ns, agentName, eventType, taskStatus, now) {
		return nil
	}
	if err := k.PatchAgentTaskStatus(ctx, ns, agentName, taskStatus, lastMessage, sessionID, costUSD, totalTokens, contextWindow); err != nil {
		return err
	}
	k.progress.record(ns, agentName, taskStatus, now)
	return nil
}
