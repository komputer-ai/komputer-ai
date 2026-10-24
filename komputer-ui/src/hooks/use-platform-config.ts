"use client";

import { useEffect, useState } from "react";
import { getPlatformConfig } from "@/lib/api";
import { createConfigLoader } from "@/lib/platform-config";
import type { ConfigResponse } from "@/lib/types";

// The platform config barely changes, so every component shares one loader:
// one request at a time, re-checked every 30s so a change (e.g. the S3 session
// store turned on or off) reaches open pages without a reload.
const REFRESH_MS = 30_000;
const loadConfig = createConfigLoader(getPlatformConfig, REFRESH_MS);
// Last answer, so a newly mounted component starts from it instead of "unknown".
let lastConfig: ConfigResponse | null = null;

/** Platform settings the UI adapts to. s3Enabled is false until known. */
export function usePlatformConfig(): { s3Enabled: boolean } {
  const [config, setConfig] = useState<ConfigResponse | null>(lastConfig);
  useEffect(() => {
    let alive = true;
    const refresh = () =>
      loadConfig().then((cfg) => {
        lastConfig = cfg;
        if (alive) setConfig(cfg);
      });
    refresh();
    const timer = setInterval(refresh, REFRESH_MS);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, []);
  return { s3Enabled: !!config?.s3?.enabled };
}
