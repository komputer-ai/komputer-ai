import { describe, expect, it } from "vitest";
import { archiveAllowedForSchedule, exampleRunAgentName, isPerRunNaming, scheduleAgentRef } from "./schedule-naming";

describe("schedule agent naming", () => {
  it("treats Date and Random as per-run, Fixed and unset as reuse", () => {
    expect(isPerRunNaming("Date")).toBe(true);
    expect(isPerRunNaming("Random")).toBe(true);
    expect(isPerRunNaming("Fixed")).toBe(false);
    expect(isPerRunNaming(undefined)).toBe(false);
  });

  it("allows Archive with the S3 session store, or when each run gets its own agent", () => {
    expect(archiveAllowedForSchedule(true, "Fixed")).toBe(true);
    expect(archiveAllowedForSchedule(false, "Date")).toBe(true);
    expect(archiveAllowedForSchedule(false, "Fixed")).toBe(false);
    expect(archiveAllowedForSchedule(false, undefined)).toBe(false);
  });

  it("shows what a run's agent will be called", () => {
    expect(exampleRunAgentName("nightly", "Fixed")).toBe("nightly-agent");
    expect(exampleRunAgentName("nightly", "Date")).toBe("nightly-YYYYMMDD-HHMM");
    expect(exampleRunAgentName("", "Random")).toBe("<schedule>-k3x9q");
  });

  it("reports an agent reference only for schedules without a template", () => {
    // agentName on a template schedule is just the agent the last run used.
    expect(scheduleAgentRef({ agentName: "nightly-20261024-0900", agent: { lifecycle: "Archive" } })).toBe("");
    expect(scheduleAgentRef({ agentName: "my-agent" })).toBe("my-agent");
    expect(scheduleAgentRef({})).toBe("");
  });
});
