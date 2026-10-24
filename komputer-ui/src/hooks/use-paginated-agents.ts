"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { listAgents } from "@/lib/api";
import type { AgentResponse, AgentListFacets } from "@/lib/types";
import { appendAgentsDeduped } from "@/lib/agent-list-query";

export const AGENTS_PAGE_SIZE = 50;
// Mirrors MaxAgentListLimit in komputer-api.
const MAX_API_LIMIT = 200;

const EMPTY_FACETS: AgentListFacets = { namespaces: [], labels: [] };

export interface PaginatedAgentsFilters {
  namespace?: string;
  status?: string;
  /** "key=value" strings, ANDed together. */
  labels?: string[];
  q?: string;
  /** API sort string, e.g. "-created" or "cost". */
  sort: string;
  /** Leave out Archived agents (the "All" view shows them only on request). */
  hideArchived?: boolean;
}

/**
 * Drives the agents page's infinite-scroll list: fetches AGENTS_PAGE_SIZE at
 * a time, exposes loadMore() for the scroll sentinel, and refresh() that
 * re-fetches everything loaded so far (so the page's existing
 * periodic/visibility refresh doesn't collapse the list back to page 1).
 *
 * Changing any field of `filters` resets to the first page. This intentionally
 * does NOT serve the unpaginated "give me everything" use case — that's what
 * useAgents() is for (see its callers, e.g. the costs page).
 */
export function usePaginatedAgents(filters: PaginatedAgentsFilters) {
  const [agents, setAgents] = useState<AgentResponse[]>([]);
  const [total, setTotal] = useState(0);
  const [facets, setFacets] = useState<AgentListFacets>(EMPTY_FACETS);
  const [nextOffset, setNextOffset] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Bumped on every filter change so a slow, now-stale request can't clobber
  // state after newer filters have already started loading.
  const requestIdRef = useRef(0);
  const agentsRef = useRef<AgentResponse[]>([]);
  agentsRef.current = agents;

  const filtersKey = JSON.stringify(filters);

  const fetchPage = useCallback(
    (limit: number, offset: number) =>
      listAgents(filters.namespace, {
        status: filters.status,
        hideArchived: filters.hideArchived,
        labelSelectors: filters.labels,
        q: filters.q,
        sort: filters.sort,
        limit,
        offset,
      }),
    // filtersKey captures every field of `filters` by value.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [filtersKey]
  );

  const loadFirstPage = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setLoading(true);
    try {
      const data = await fetchPage(AGENTS_PAGE_SIZE, 0);
      if (requestId !== requestIdRef.current) return; // superseded by newer filters
      setAgents(data.agents || []);
      setTotal(data.total ?? (data.agents || []).length);
      setFacets(data.facets ?? EMPTY_FACETS);
      setNextOffset(data.nextOffset ?? null);
      setError(null);
    } catch (e: unknown) {
      if (requestId !== requestIdRef.current) return;
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      if (requestId === requestIdRef.current) setLoading(false);
    }
  }, [fetchPage]);

  useEffect(() => {
    loadFirstPage();
  }, [loadFirstPage]);

  const loadMore = useCallback(async () => {
    if (nextOffset == null || loadingMore || loading) return;
    const requestId = requestIdRef.current;
    setLoadingMore(true);
    try {
      const data = await fetchPage(AGENTS_PAGE_SIZE, nextOffset);
      if (requestId !== requestIdRef.current) return; // filters changed meanwhile
      setAgents((prev) => appendAgentsDeduped(prev, data.agents || []));
      setTotal(data.total ?? total);
      setFacets(data.facets ?? facets);
      setNextOffset(data.nextOffset ?? null);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoadingMore(false);
    }
  }, [nextOffset, loadingMore, loading, fetchPage, total, facets]);

  // Re-fetches everything loaded so far, so a periodic or
  // visibility-triggered refresh updates in place instead of resetting
  // infinite scroll back to the first 50.
  const refresh = useCallback(async () => {
    const requestId = requestIdRef.current;
    const loadedCount = Math.max(agentsRef.current.length, AGENTS_PAGE_SIZE);
    try {
      // The API caps `limit` at MAX_API_LIMIT, so re-fetch in chunks once more
      // than that has been scrolled into view.
      let data = await fetchPage(Math.min(loadedCount, MAX_API_LIMIT), 0);
      let refreshed = data.agents || [];
      while (data.nextOffset != null && refreshed.length < loadedCount) {
        data = await fetchPage(Math.min(loadedCount - refreshed.length, MAX_API_LIMIT), data.nextOffset);
        refreshed = appendAgentsDeduped(refreshed, data.agents || []);
      }
      if (requestId !== requestIdRef.current) return;
      setAgents(refreshed);
      setTotal(data.total ?? refreshed.length);
      setFacets(data.facets ?? EMPTY_FACETS);
      setNextOffset(data.nextOffset ?? null);
      setError(null);
    } catch (e: unknown) {
      if (requestId !== requestIdRef.current) return;
      setError(e instanceof Error ? e.message : "Unknown error");
    }
  }, [fetchPage]);

  return {
    agents,
    total,
    facets,
    hasMore: nextOffset != null,
    loading,
    loadingMore,
    error,
    loadMore,
    refresh,
  };
}
