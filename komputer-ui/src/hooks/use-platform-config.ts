"use client";

import { useEffect, useState } from "react";
import { getPlatformConfig } from "@/lib/api";
import { retryUntilLoaded } from "@/lib/retry";
import type { ConfigResponse } from "@/lib/types";

// The platform config barely changes, so every component shares one fetch.
let cached: ConfigResponse | null = null;
let inflight: Promise<ConfigResponse> | null = null;

function loadConfig(): Promise<ConfigResponse> {
  if (cached) return Promise.resolve(cached);
  if (!inflight) {
    // Keep retrying: one failed request (e.g. the API restarting) must not
    // leave an open page treating S3 as off until it's reloaded.
    inflight = retryUntilLoaded(getPlatformConfig).then((cfg) => {
      cached = cfg;
      return cfg;
    });
  }
  return inflight;
}

/** Platform settings the UI adapts to. s3Enabled is false until known. */
export function usePlatformConfig(): { s3Enabled: boolean } {
  const [config, setConfig] = useState<ConfigResponse | null>(cached);
  useEffect(() => {
    let alive = true;
    loadConfig().then((cfg) => {
      if (alive) setConfig(cfg);
    });
    return () => {
      alive = false;
    };
  }, []);
  return { s3Enabled: !!config?.s3?.enabled };
}
