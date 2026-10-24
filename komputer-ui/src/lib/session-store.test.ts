import { describe, it, expect } from "vitest";
import { formatSessionStore, archivedReadOnly, agentDormant } from "./session-store";
import { buildListAgentsQuery } from "./api";

describe("formatSessionStore", () => {
  const now = new Date("2026-10-24T10:02:00Z");

  it("reports a recent save", () => {
    expect(formatSessionStore({ ok: true, lastSavedAt: "2026-10-24T10:00:00Z" }, now)).toBe("Session saved 2m ago");
  });

  it("reports a failed save", () => {
    expect(formatSessionStore({ ok: false, error: "AccessDenied" }, now)).toBe("Session save failed");
  });

  it("says nothing when the store isn't in use", () => {
    expect(formatSessionStore(undefined, now)).toBe("");
  });
});

describe("archived agents", () => {
  it("are read-only only without the S3 session store", () => {
    expect(archivedReadOnly("Archived", false)).toBe(true);
    expect(archivedReadOnly("Archived", true)).toBe(false);
    expect(archivedReadOnly("Running", false)).toBe(false);
  });

  it("wake like sleeping agents when the store is on", () => {
    expect(agentDormant("Sleeping", false)).toBe(true);
    expect(agentDormant("Archived", true)).toBe(true);
    expect(agentDormant("Archived", false)).toBe(false);
  });
});

describe("buildListAgentsQuery", () => {
  it("hides archived agents for the All filter", () => {
    expect(buildListAgentsQuery(undefined, { hideArchived: true })).toContain("hideArchived=true");
  });

  it("doesn't hide them when filtering for Archived", () => {
    expect(buildListAgentsQuery(undefined, { status: "Archived" })).not.toContain("hideArchived");
  });
});
