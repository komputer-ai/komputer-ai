"use client";

import { useState, useEffect, useCallback } from "react";
import { listSecrets } from "@/lib/api";
import type { SecretResponse } from "@/lib/types";

export function useSecrets(showAll?: boolean) {
  const [secrets, setSecrets] = useState<SecretResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await listSecrets(undefined, showAll);
      setSecrets(data.secrets || []);
      setError(null);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [showAll]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { secrets, loading, error, refresh };
}
