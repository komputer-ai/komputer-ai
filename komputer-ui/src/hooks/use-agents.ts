"use client";

import { useState, useEffect, useCallback } from "react";
import { listAgents, type ListAgentsOptions } from "@/lib/api";
import type { AgentResponse } from "@/lib/types";

/** `fields` trims each agent to what the caller shows; pass a constant so it stays stable. */
export function useAgents(fields?: ListAgentsOptions["fields"]) {
  const [agents, setAgents] = useState<AgentResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await listAgents(undefined, { fields });
      setAgents(data.agents || []);
      setError(null);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [fields]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { agents, loading, error, refresh };
}
