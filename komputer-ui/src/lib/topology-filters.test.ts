import { describe, expect, it } from "vitest";
import { parseHiddenStatuses, serializeHiddenStatuses } from "./topology-filters";

describe("parseHiddenStatuses", () => {
  it("hides Archived when the param is absent", () => {
    expect(parseHiddenStatuses(null)).toEqual(["Archived"]);
  });

  it("hides nothing when the param is empty", () => {
    expect(parseHiddenStatuses("")).toEqual([]);
  });

  it("reads a comma-separated list", () => {
    expect(parseHiddenStatuses("Archived,Failed")).toEqual(["Archived", "Failed"]);
  });
});

describe("serializeHiddenStatuses", () => {
  it("leaves the default out of the URL", () => {
    expect(serializeHiddenStatuses(["Archived"])).toBeNull();
  });

  it("writes an empty param when nothing is hidden", () => {
    expect(serializeHiddenStatuses([])).toBe("");
  });

  it("round-trips other selections", () => {
    const param = serializeHiddenStatuses(["Failed", "Archived"]);
    expect(param).toBe("Archived,Failed");
    expect(parseHiddenStatuses(param)).toEqual(["Archived", "Failed"]);
  });
});
