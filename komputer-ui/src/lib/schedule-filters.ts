import type { ScheduleResponse } from "./types";

export const SCHEDULE_STATUS_FILTERS = ["All", "Active", "Running", "Failed", "Suspended", "Error"] as const;
export type ScheduleStatusFilter = (typeof SCHEDULE_STATUS_FILTERS)[number];

/**
 * Active, Suspended and Error are the schedule's phase; Running means a run is
 * in progress, and Failed means the last run failed.
 */
export function matchesScheduleStatus(
  schedule: Pick<ScheduleResponse, "phase" | "lastRunStatus">,
  filter: ScheduleStatusFilter,
): boolean {
  switch (filter) {
    case "All":
      return true;
    case "Running":
      return schedule.lastRunStatus === "InProgress";
    case "Failed":
      return schedule.lastRunStatus === "Failed";
    default:
      return schedule.phase === filter;
  }
}
