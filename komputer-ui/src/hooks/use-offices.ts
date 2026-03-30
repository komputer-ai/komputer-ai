"use client";

import { useState, useEffect, useCallback } from "react";
import { listOffices } from "@/lib/api";
import type { OfficeResponse } from "@/lib/types";

export function useOffices() {
  const [offices, setOffices] = useState<OfficeResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await listOffices();
      setOffices(data.offices || []);
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

  return { offices, loading, error, refresh };
}
