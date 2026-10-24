import { describe, it, expect } from "vitest";
import { formatSessionStore, archivedReadOnly, agentDormant, deleteOffered, ephemeralBlocksSleep } from "./session-store";
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

describe("archiveAgent", () => {
  it("POSTs to the archive-now endpoint", async () => {
    const { archiveAgent } = await import("./api");
    const seen: string[] = [];
    const realFetch = globalThis.fetch;
    globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
      seen.push(`${init?.method} ${String(input)}`);
      return new Response(JSON.stringify({ name: "a1", status: "Running" }), { status: 200 });
    }) as typeof fetch;
    try {
      await archiveAgent("a1", "team-a");
    } finally {
      globalThis.fetch = realFetch;
    }
    expect(seen).toHaveLength(1);
    expect(seen[0]).toMatch(/^POST .*\/api\/v1\/agents\/a1\/archive\?namespace=team-a$/);
  });
});

describe("deleteOffered", () => {
  it("with the S3 session store, offers Delete only once the agent is archived", () => {
    expect(deleteOffered("Running", true)).toBe(false);
    expect(deleteOffered("Sleeping", true)).toBe(false);
    expect(deleteOffered("Archived", true)).toBe(true);
  });
  it("without it, offers Delete on every agent", () => {
    expect(deleteOffered("Running", false)).toBe(true);
    expect(deleteOffered("Archived", false)).toBe(true);
  });
});

describe("ephemeralBlocksSleep", () => {
  it("blocks Sleep for an ephemeral agent only when the S3 session store is off", () => {
    expect(ephemeralBlocksSleep(true, false)).toBe(true);
    expect(ephemeralBlocksSleep(true, true)).toBe(false); // saved to S3 first, restored on wake
    expect(ephemeralBlocksSleep(false, false)).toBe(false);
  });
});
