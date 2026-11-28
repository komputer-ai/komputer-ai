import { describe, expect, it } from "vitest";
import { buildGraph } from "./topology-graph";
import type { AgentResponse, OfficeResponse } from "@/lib/types";

const agent = (name: string): AgentResponse => ({ name, namespace: "default", status: "Running" } as AgentResponse);

const office = {
  name: "eng",
  namespace: "default",
  manager: "mgr",
  members: [
    { name: "mgr", role: "manager" },
    { name: "w1", role: "worker" },
    { name: "w2", role: "worker" },
  ],
} as unknown as OfficeResponse;

const nodeIds = (g: ReturnType<typeof buildGraph>) => g.nodes.map((n) => n.id).sort();
const status = (g: ReturnType<typeof buildGraph>, id: string) =>
  (g.nodes.find((n) => n.id === id)?.data as { status?: string } | undefined)?.status;

describe("buildGraph", () => {
  it("shows office members missing from the agent list as Deleted", () => {
    const g = buildGraph([agent("mgr"), agent("w1")], [office], []);
    expect(status(g, "agent-w2")).toBe("Deleted");
  });

  it("leaves out hidden workers instead of showing them as Deleted", () => {
    const g = buildGraph([agent("mgr"), agent("w1")], [office], [], [], new Set(["w2"]));
    expect(nodeIds(g)).toEqual(["agent-mgr", "agent-w1", "office-eng"]);
    expect(g.edges.some((e) => e.target === "agent-w2")).toBe(false);
  });

  it("hangs workers off the office when the manager is hidden", () => {
    const g = buildGraph([agent("w1"), agent("w2")], [office], [], [], new Set(["mgr"]));
    expect(nodeIds(g)).toEqual(["agent-w1", "agent-w2", "office-eng"]);
    const sources = new Set(g.edges.map((e) => e.source));
    expect(sources).toEqual(new Set(["office-eng"]));
  });
});
