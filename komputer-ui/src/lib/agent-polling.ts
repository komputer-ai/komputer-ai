export const ACTIVE_POLL_MS = 5_000;
export const SETTLED_POLL_MS = 15_000;

// How often the agent page re-reads the agent. Never stops: the operator and other
// clients can change the agent at any time, and a just-created agent has no phase
// or task status yet. Settled states are listed, so anything else polls fast.
export function agentPollInterval({ status, taskStatus }: { status: string; taskStatus?: string }): number {
  const taskDone = taskStatus === "Complete" || taskStatus === "Error";
  const taskActive = taskStatus === "InProgress" || taskStatus === "Compacting";
  const settled =
    (status === "Running" && taskDone) ||
    (["Sleeping", "Archived", "Succeeded", "Failed"].includes(status) && !taskActive);
  return settled ? SETTLED_POLL_MS : ACTIVE_POLL_MS;
}
