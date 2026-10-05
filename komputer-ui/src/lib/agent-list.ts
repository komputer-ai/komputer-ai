import type { AgentResponse } from "./types";

export type AgentSort = "newest" | "oldest" | "recent-active";

export interface AgentListFilters {
  namespace: string;
  /** "All" matches every status. */
  status: string;
  search: string;
  /** "key=value" strings, ANDed together. */
  labels: string[];
  sort: AgentSort;
}

function matchesLabels(agent: AgentResponse, labelFilters: string[]): boolean {
  if (labelFilters.length === 0) return true;
  if (!agent.labels) return false;
  return labelFilters.every((filter) => {
    const eq = filter.indexOf("=");
    if (eq === -1) return false;
    const key = filter.slice(0, eq);
    const value = filter.slice(eq + 1);
    return agent.labels?.[key] === value;
  });
}

function timeOf(value: string | undefined): number {
  if (!value) return NaN;
  const t = new Date(value).getTime();
  return Number.isNaN(t) ? NaN : t;
}

/** createdAt descending — undated/unparseable entries sort last. */
function compareNewestFirst(a: AgentResponse, b: AgentResponse): number {
  const at = timeOf(a.createdAt);
  const bt = timeOf(b.createdAt);
  if (Number.isNaN(at) && Number.isNaN(bt)) return 0;
  if (Number.isNaN(at)) return 1;
  if (Number.isNaN(bt)) return -1;
  return bt - at;
}

/**
 * lastActivityAt descending; agents that have never run a task (no
 * lastActivityAt) sort last, tie-broken by createdAt descending.
 */
function compareRecentActiveFirst(a: AgentResponse, b: AgentResponse): number {
  const aActive = !!a.lastActivityAt;
  const bActive = !!b.lastActivityAt;
  if (aActive !== bActive) return aActive ? -1 : 1;
  if (aActive && bActive) {
    const diff = timeOf(b.lastActivityAt) - timeOf(a.lastActivityAt);
    if (diff !== 0) return diff;
  }
  return compareNewestFirst(a, b);
}

/**
 * Pure filter + sort for the agents list page — namespace/status/search/label
 * filters are ANDed, then the result is ordered by `sort`. Kept side-effect
 * free so the page component and its tests can share one source of truth.
 */
export function filterAndSortAgents(
  agents: AgentResponse[],
  filters: AgentListFilters
): AgentResponse[] {
  let result = agents;

  if (filters.namespace) {
    result = result.filter((a) => a.namespace === filters.namespace);
  }
  if (filters.status !== "All") {
    result = result.filter((a) => a.status === filters.status);
  }
  if (filters.search.trim()) {
    const q = filters.search.trim().toLowerCase();
    result = result.filter((a) => a.name.toLowerCase().includes(q));
  }
  if (filters.labels.length > 0) {
    result = result.filter((a) => matchesLabels(a, filters.labels));
  }

  const compare = filters.sort === "oldest"
    ? (a: AgentResponse, b: AgentResponse) => -compareNewestFirst(a, b)
    : filters.sort === "recent-active"
      ? compareRecentActiveFirst
      : compareNewestFirst;

  return [...result].sort(compare);
}

/** Unique "key=value" pairs present across the given agents' labels, for filter suggestions. */
export function labelSuggestions(agents: AgentResponse[]): string[] {
  const seen = new Set<string>();
  for (const agent of agents) {
    if (!agent.labels) continue;
    for (const [key, value] of Object.entries(agent.labels)) {
      seen.add(`${key}=${value}`);
    }
  }
  return [...seen];
}
