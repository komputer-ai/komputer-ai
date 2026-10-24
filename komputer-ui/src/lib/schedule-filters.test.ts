import { describe, expect, it } from "vitest";
import { matchesScheduleStatus } from "./schedule-filters";

const sched = (phase: "Active" | "Suspended" | "Error", lastRunStatus?: string) => ({ phase, lastRunStatus });

describe("matchesScheduleStatus", () => {
  it("All matches everything", () => {
    expect(matchesScheduleStatus(sched("Error"), "All")).toBe(true);
  });

  it("Running matches a schedule whose run is in progress", () => {
    expect(matchesScheduleStatus(sched("Active", "InProgress"), "Running")).toBe(true);
    expect(matchesScheduleStatus(sched("Active", "Success"), "Running")).toBe(false);
  });

  it("Active, Suspended and Error match the schedule phase", () => {
    expect(matchesScheduleStatus(sched("Active"), "Active")).toBe(true);
    expect(matchesScheduleStatus(sched("Suspended"), "Suspended")).toBe(true);
    expect(matchesScheduleStatus(sched("Suspended"), "Active")).toBe(false);
    expect(matchesScheduleStatus(sched("Error"), "Error")).toBe(true);
  });

  it("Failed matches a schedule whose last run failed", () => {
    expect(matchesScheduleStatus(sched("Active", "Failed"), "Failed")).toBe(true);
    expect(matchesScheduleStatus(sched("Active", "Success"), "Failed")).toBe(false);
  });
});
