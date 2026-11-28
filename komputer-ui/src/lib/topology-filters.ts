import type { AgentStatus } from "./types";

export const TOPOLOGY_AGENT_STATUSES: AgentStatus[] = [
  "Running", "Queued", "Pending", "Sleeping", "Succeeded", "Failed", "Archived",
];

/** Archived agents are finished records, so the topology hides them unless asked. */
export const DEFAULT_HIDDEN_STATUSES: AgentStatus[] = ["Archived"];

/**
 * Reads the `hide` query param: absent means the default, empty means hide
 * nothing, otherwise a comma-separated list of statuses.
 */
export function parseHiddenStatuses(param: string | null): string[] {
  if (param === null) return [...DEFAULT_HIDDEN_STATUSES];
  return param.split(",").filter(Boolean);
}

/** The `hide` query param for hidden, or null when it's the default (so the URL stays clean). */
export function serializeHiddenStatuses(hidden: string[]): string | null {
  const sorted = [...hidden].sort();
  const defaults = [...DEFAULT_HIDDEN_STATUSES].sort();
  if (sorted.length === defaults.length && sorted.every((s, i) => s === defaults[i])) return null;
  return sorted.join(",");
}
