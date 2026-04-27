"use client";

import { useState, useEffect, useCallback } from "react";
import { listSquads } from "@/lib/api";
import type { Squad } from "@/lib/types";

export function useSquads() {
  const [squads, setSquads] = useState<Squad[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await listSquads();
      setSquads(data.squads || []);
      setError(null);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { squads, loading, error, refresh };
}
