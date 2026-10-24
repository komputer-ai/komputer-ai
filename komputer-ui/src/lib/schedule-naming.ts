import type { ScheduleAgentNaming, ScheduleResponse } from "./types";

export const AGENT_NAMING_OPTIONS: { value: ScheduleAgentNaming; label: string }[] = [
  { value: "Fixed", label: "Same agent every run" },
  { value: "Date", label: "New agent each run, named by date" },
  { value: "Random", label: "New agent each run, random suffix" },
];

/** Date and Random naming create a new agent from the template on every run. */
export function isPerRunNaming(naming?: ScheduleAgentNaming): boolean {
  return naming === "Date" || naming === "Random";
}

/**
 * A schedule's agent may use the Archive lifecycle with the S3 session store
 * (the reused agent restores from S3), or when each run gets its own agent.
 */
export function archiveAllowedForSchedule(s3Enabled: boolean, naming?: ScheduleAgentNaming): boolean {
  return s3Enabled || isPerRunNaming(naming);
}

/** Illustrates the name a run's agent gets, for the form's hint text. */
export function exampleRunAgentName(scheduleName: string, naming?: ScheduleAgentNaming): string {
  const base = scheduleName.trim() || "<schedule>";
  if (naming === "Date") return `${base}-YYYYMMDD-HHMM`;
  if (naming === "Random") return `${base}-k3x9q`;
  return `${base}-agent`;
}

/**
 * The existing agent a schedule targets, or "" when it creates agents from its
 * template. The API's agentName is the agent the last run used, so it only names
 * a reference when the schedule has no template.
 */
export function scheduleAgentRef(schedule: Pick<ScheduleResponse, "agentName" | "agent">): string {
  return schedule.agent ? "" : (schedule.agentName ?? "");
}
