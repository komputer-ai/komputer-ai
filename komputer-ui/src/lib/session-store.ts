import type { SessionStoreState } from "./types";

/** "Session saved 2m ago" / "Session save failed" for an agent's S3 session store state. */
export function formatSessionStore(s: SessionStoreState | undefined, now: Date = new Date()): string {
  if (!s) return "";
  if (!s.ok) return "Session save failed";
  if (!s.lastSavedAt) return "Session saved";
  const secs = Math.max(0, Math.round((now.getTime() - new Date(s.lastSavedAt).getTime()) / 1000));
  if (secs < 60) return "Session saved just now";
  const mins = Math.round(secs / 60);
  if (mins < 60) return `Session saved ${mins}m ago`;
  const hours = Math.round(mins / 60);
  if (hours < 48) return `Session saved ${hours}h ago`;
  return `Session saved ${Math.round(hours / 24)}d ago`;
}

/**
 * Archived agents are read-only records — unless the S3 session store is on, in
 * which case an archived agent is a sleeping agent with no workspace: every
 * control works and it restores its session when it runs again.
 */
export function archivedReadOnly(status: string | undefined, s3Enabled: boolean): boolean {
  return status === "Archived" && !s3Enabled;
}

/** Whether waking applies: Sleeping, or Archived with the S3 session store on. */
export function agentDormant(status: string | undefined, s3Enabled: boolean): boolean {
  return status === "Sleeping" || (status === "Archived" && s3Enabled);
}
