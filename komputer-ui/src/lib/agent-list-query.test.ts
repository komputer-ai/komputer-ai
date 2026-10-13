import { describe, it, expect } from "vitest";
import {
  agentKey,
  cardsSortToApiSort,
  parseApiSort,
  toggleTableSort,
  appendAgentsDeduped,
} from "./agent-list-query";
import type { AgentResponse } from "./types";

function mk(name: string, namespace = "default"): AgentResponse {
  return {
    name,
    namespace,
    model: "claude-sonnet-4-6",
    status: "Running",
    createdAt: "2024-01-01T00:00:00Z",
  } as AgentResponse;
}

describe("agentKey", () => {
  it("combines namespace and name", () => {
    expect(agentKey({ name: "a", namespace: "ns1" })).toBe("ns1/a");
  });
});

describe("cardsSortToApiSort", () => {
  it("maps newest to -created", () => {
    expect(cardsSortToApiSort("newest")).toBe("-created");
  });
  it("maps oldest to created", () => {
    expect(cardsSortToApiSort("oldest")).toBe("created");
  });
  it("maps recent-active to -lastActive", () => {
    expect(cardsSortToApiSort("recent-active")).toBe("-lastActive");
  });
});

describe("parseApiSort", () => {
  it("parses a descending sort", () => {
    expect(parseApiSort("-cost")).toEqual({ key: "cost", desc: true });
  });
  it("parses an ascending sort", () => {
    expect(parseApiSort("name")).toEqual({ key: "name", desc: false });
  });
});

describe("toggleTableSort", () => {
  it("switching to a new column defaults to descending", () => {
    expect(toggleTableSort("-created", "cost")).toBe("-cost");
    expect(toggleTableSort("name", "cost")).toBe("-cost");
  });
  it("clicking the active descending column flips to ascending", () => {
    expect(toggleTableSort("-cost", "cost")).toBe("cost");
  });
  it("clicking the active ascending column flips to descending", () => {
    expect(toggleTableSort("cost", "cost")).toBe("-cost");
  });
});

describe("appendAgentsDeduped", () => {
  it("appends new agents after existing ones", () => {
    const existing = [mk("a"), mk("b")];
    const incoming = [mk("c"), mk("d")];
    expect(appendAgentsDeduped(existing, incoming).map((a) => a.name)).toEqual([
      "a",
      "b",
      "c",
      "d",
    ]);
  });

  it("drops incoming agents whose key is already present", () => {
    const existing = [mk("a"), mk("b")];
    const incoming = [mk("b"), mk("c")]; // offset shifted, "b" reappears
    const result = appendAgentsDeduped(existing, incoming);
    expect(result.map((a) => a.name)).toEqual(["a", "b", "c"]);
  });

  it("treats same name in different namespaces as distinct", () => {
    const existing = [mk("a", "ns1")];
    const incoming = [mk("a", "ns2")];
    const result = appendAgentsDeduped(existing, incoming);
    expect(result).toHaveLength(2);
  });

  it("returns incoming unchanged when existing is empty", () => {
    const incoming = [mk("a"), mk("b")];
    expect(appendAgentsDeduped([], incoming)).toEqual(incoming);
  });
});
