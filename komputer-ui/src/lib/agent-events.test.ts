import { describe, expect, it } from "vitest";
import type { AgentEvent } from "./types";
import { eventKey, mergeAgentEvents } from "./agent-events";

const ev = (type: AgentEvent["type"], timestamp: string, payload: Record<string, unknown>, agentName = "a1"): AgentEvent =>
  ({ agentName, namespace: "default", type, timestamp, payload });

// History and the WebSocket deliver the same event as separate objects.
const copy = (e: AgentEvent): AgentEvent => JSON.parse(JSON.stringify(e));

describe("mergeAgentEvents", () => {
  it("keeps two tool results from the same second", () => {
    const update = ev("tool_result", "2026-11-25T21:36:09Z", { tool: "update_pull_request", input: { n: 1 }, output: "ok" });
    const register = ev("tool_result", "2026-11-25T21:36:09Z", { tool: "register_release_fix", input: { v: "2.15" }, output: "ok" });
    expect(mergeAgentEvents([update, register], [])).toEqual([update, register]);
  });

  it("keeps two text blocks from the same second", () => {
    const a = ev("text", "2026-11-25T21:36:09Z", { content: "first" });
    const b = ev("text", "2026-11-25T21:36:09Z", { content: "second" });
    expect(mergeAgentEvents([], [a, b])).toHaveLength(2);
  });

  it("keeps an event once when both history and the WebSocket have it", () => {
    const a = ev("tool_result", "2026-11-25T21:36:09Z", { tool: "create_item", output: "ok" });
    const b = ev("text", "2026-11-25T21:36:10Z", { content: "done" });
    expect(mergeAgentEvents([a, b], [copy(b), copy(a)])).toEqual([a, b]);
  });

  it("drops an event the WebSocket replays", () => {
    const a = ev("text", "2026-11-25T21:36:10Z", { content: "done" });
    expect(mergeAgentEvents([], [a, copy(a)])).toEqual([a]);
  });

  it("treats task_started and user_message with the same text as one message", () => {
    const started = ev("task_started", "2026-11-25T21:36:00Z", { instructions: "Do it ", session_id: "s" });
    const history = ev("user_message", "2026-11-25T21:36:00Z", { content: "Do it" });
    expect(mergeAgentEvents([history], [started])).toEqual([history]);
  });

  it("sorts by time, keeping arrival order within a second", () => {
    const late = ev("text", "2026-11-25T21:36:11Z", { content: "late" });
    const first = ev("tool_result", "2026-11-25T21:36:09Z", { tool: "x" });
    const second = ev("tool_result", "2026-11-25T21:36:09Z", { tool: "y" });
    expect(mergeAgentEvents([late], [first, second])).toEqual([first, second, late]);
  });
});

describe("eventKey", () => {
  it("tells apart the same event from different agents", () => {
    const p = { tool: "x" };
    expect(eventKey(ev("tool_result", "2026-11-25T21:36:09Z", p, "a1")))
      .not.toBe(eventKey(ev("tool_result", "2026-11-25T21:36:09Z", p, "a2")));
  });
});
