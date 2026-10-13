import type { AgentResponse } from "./types";

/** Stable identity for an agent across API responses: namespace + name. */
export const agentKey = (a: { name: string; namespace: string }) => `${a.namespace}/${a.name}`;

/** The cards view's friendly sort dropdown values. */
export type CardsSort = "newest" | "oldest" | "recent-active";

/**
 * Maps the cards view's sort dropdown to the API's `sort` query param.
 * Kept separate from the table's sort state so both views can drive the
 * same server-side sort through one shared shape.
 */
export function cardsSortToApiSort(sort: CardsSort): string {
  switch (sort) {
    case "oldest":
      return "created";
    case "recent-active":
      return "-lastActive";
    case "newest":
    default:
      return "-created";
  }
}

/** Column keys the table's sortable headers can drive the API `sort` param with. */
export type AgentTableSortKey = "created" | "name" | "status" | "model" | "lastActive" | "cost";

/** Splits an API sort string like "-cost" into its key and direction. */
export function parseApiSort(apiSort: string): { key: string; desc: boolean } {
  const desc = apiSort.startsWith("-");
  return { key: desc ? apiSort.slice(1) : apiSort, desc };
}

/**
 * Given the currently active API sort string and a newly-clicked column,
 * returns the next API sort string: switching to a new column defaults to
 * descending; clicking the already-active column flips its direction.
 */
export function toggleTableSort(currentApiSort: string, clickedKey: AgentTableSortKey): string {
  const { key, desc } = parseApiSort(currentApiSort);
  if (key === clickedKey) {
    return desc ? clickedKey : `-${clickedKey}`;
  }
  return `-${clickedKey}`;
}

/**
 * Appends a freshly-fetched page to the already-loaded agents, dropping any
 * agent whose key is already present. Infinite-scroll offsets can shift
 * while the user is scrolling (agents created/deleted elsewhere), so a plain
 * concat could otherwise render the same agent twice.
 */
export function appendAgentsDeduped(
  existing: AgentResponse[],
  incoming: AgentResponse[]
): AgentResponse[] {
  const seen = new Set(existing.map(agentKey));
  const deduped = incoming.filter((a) => !seen.has(agentKey(a)));
  return [...existing, ...deduped];
}
