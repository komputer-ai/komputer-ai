import type { AgentListFacets } from "./types";

// Older APIs send null for a facet list with no entries (e.g. no agent has labels).
export function normalizeAgentFacets(
  facets: { namespaces?: string[] | null; labels?: string[] | null } | undefined,
): AgentListFacets | undefined {
  if (!facets) return undefined;
  return { namespaces: facets.namespaces ?? [], labels: facets.labels ?? [] };
}
