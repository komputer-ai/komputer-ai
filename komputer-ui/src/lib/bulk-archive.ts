import { agentKey } from "./agent-list-query";

export type BulkArchiveOutcome = {
  archived: number;
  /** Selection keys (namespace/name) of the agents that weren't archived. */
  failedKeys: string[];
  /** Null when everything was archived. */
  message: string | null;
};

/** Sums up a bulk archive: how many went through, and which didn't and why (grouped by reason). */
export function summarizeBulkArchive(
  agents: { name: string; namespace: string }[],
  results: PromiseSettledResult<unknown>[],
): BulkArchiveOutcome {
  const byReason = new Map<string, string[]>();
  const failedKeys: string[] = [];
  results.forEach((r, i) => {
    if (r.status === "fulfilled") return;
    const reason = r.reason instanceof Error ? r.reason.message : String(r.reason);
    failedKeys.push(agentKey(agents[i]));
    byReason.set(reason, [...(byReason.get(reason) ?? []), agents[i].name]);
  });
  const archived = results.length - failedKeys.length;
  if (failedKeys.length === 0) return { archived, failedKeys, message: null };
  const notArchived = [...byReason].map(([reason, names]) => `${names.join(", ")} (${reason})`).join("; ");
  return { archived, failedKeys, message: `Archived ${archived} of ${results.length}. Not archived: ${notArchived}` };
}
