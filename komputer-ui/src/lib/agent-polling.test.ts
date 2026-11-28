import { describe, expect, it } from "vitest";
import { ACTIVE_POLL_MS, SETTLED_POLL_MS, agentPollInterval } from "./agent-polling";

describe("agentPollInterval", () => {
  it("polls fast while the agent has no phase yet (just created)", () => {
    expect(agentPollInterval({ status: "" })).toBe(ACTIVE_POLL_MS);
  });

  it("polls fast while the agent is waiting to start", () => {
    expect(agentPollInterval({ status: "Pending" })).toBe(ACTIVE_POLL_MS);
    expect(agentPollInterval({ status: "Queued" })).toBe(ACTIVE_POLL_MS);
  });

  it("polls fast while a task is active", () => {
    expect(agentPollInterval({ status: "Running", taskStatus: "InProgress" })).toBe(ACTIVE_POLL_MS);
    expect(agentPollInterval({ status: "Running", taskStatus: "Compacting" })).toBe(ACTIVE_POLL_MS);
  });

  it("polls fast while a running agent hasn't reported its first task", () => {
    expect(agentPollInterval({ status: "Running" })).toBe(ACTIVE_POLL_MS);
  });

  it("keeps polling, slower, once the agent has settled", () => {
    expect(agentPollInterval({ status: "Running", taskStatus: "Complete" })).toBe(SETTLED_POLL_MS);
    expect(agentPollInterval({ status: "Running", taskStatus: "Error" })).toBe(SETTLED_POLL_MS);
    for (const status of ["Sleeping", "Archived", "Succeeded", "Failed"]) {
      expect(agentPollInterval({ status, taskStatus: "Complete" })).toBe(SETTLED_POLL_MS);
      expect(agentPollInterval({ status })).toBe(SETTLED_POLL_MS);
    }
  });
});
