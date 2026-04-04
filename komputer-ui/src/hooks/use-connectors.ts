"use client";

import { useState, useEffect, useCallback } from "react";
import { listConnectors } from "@/lib/api";
import type { ConnectorResponse } from "@/lib/types";

export function useConnectors() {
  const [connectors, setConnectors] = useState<ConnectorResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await listConnectors();
      setConnectors(data.connectors || []);
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

  return { connectors, loading, error, refresh };
}
