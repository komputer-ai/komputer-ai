"use client";

import { useState, useEffect, useCallback } from "react";
import { listSkills } from "@/lib/api";
import type { SkillResponse } from "@/lib/types";

export function useSkills() {
  const [skills, setSkills] = useState<SkillResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await listSkills();
      setSkills(data.skills || []);
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

  return { skills, loading, error, refresh };
}
