"use client";

import { useEffect, useState } from "react";
import { formatRelativeTime } from "@/lib/utils";
import { Tooltip } from "@/components/kit/tooltip";

type RelativeTimeProps = {
  timestamp: string;
};

export function RelativeTime({ timestamp }: RelativeTimeProps) {
  const [relative, setRelative] = useState(() => formatRelativeTime(timestamp));

  useEffect(() => {
    setRelative(formatRelativeTime(timestamp));
    const interval = setInterval(() => {
      setRelative(formatRelativeTime(timestamp));
    }, 30_000);
    return () => clearInterval(interval);
  }, [timestamp]);

  return (
    <Tooltip content={new Date(timestamp).toLocaleString()} side="bottom">
      <span className="cursor-default text-xs text-[var(--color-text-secondary)]">
        {relative}
      </span>
    </Tooltip>
  );
}
