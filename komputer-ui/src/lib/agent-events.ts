import type { AgentEvent } from "@/lib/types";

const keys = new WeakMap<AgentEvent, string>();

// Identifies an event across history and the WebSocket, which deliver it as
// separate but identical objects. Timestamps are whole seconds, so the payload
// is part of the key: two tool results in one second are different events.
// task_started (WebSocket) and user_message (history) carry the same message
// in different fields, so a user message is identified by its text.
export function eventKey(e: AgentEvent): string {
  let key = keys.get(e);
  if (key === undefined) {
    const p = e.payload ?? {};
    const user = e.type === "task_started" || e.type === "user_message";
    const identity = user ? String(p.instructions ?? p.message ?? p.content ?? "").trim() : JSON.stringify(p);
    key = `${e.agentName}|${e.timestamp}|${user ? "user_message" : e.type}|${identity}`;
    keys.set(e, key);
  }
  return key;
}

// Merges event lists, keeping the first copy of each event, sorted by time.
export function mergeAgentEvents(...lists: AgentEvent[][]): AgentEvent[] {
  const seen = new Set<string>();
  return lists
    .flat()
    .filter((e) => {
      const key = eventKey(e);
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    })
    .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime());
}
