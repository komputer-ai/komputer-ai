import { describe, it, expect } from "vitest";
import { splitLabels, validateLabelRows, hasLabelErrors, diffLabels } from "./label-edit";

describe("splitLabels", () => {
  it("separates komputer.ai/* labels and sorts both groups", () => {
    const { user, system } = splitLabels({
      "komputer.ai/squad": "s1",
      team: "core",
      env: "prod",
      "komputer.ai/created-by": "manager",
    });
    expect(user).toEqual([{ key: "env", value: "prod" }, { key: "team", value: "core" }]);
    expect(system).toEqual([["komputer.ai/created-by", "manager"], ["komputer.ai/squad", "s1"]]);
  });

  it("handles undefined", () => {
    expect(splitLabels(undefined)).toEqual({ user: [], system: [] });
  });
});

describe("validateLabelRows", () => {
  it("accepts valid rows, empty values, prefixed keys and blank rows", () => {
    const errs = validateLabelRows([
      { key: "team", value: "core" },
      { key: "example.com/tier", value: "" },
      { key: "", value: "" },
    ]);
    expect(hasLabelErrors(errs)).toBe(false);
  });

  it("requires a key when a value is set", () => {
    expect(validateLabelRows([{ key: "", value: "x" }])[0].key).toBe("Key is required");
  });

  it("rejects the reserved prefix", () => {
    expect(validateLabelRows([{ key: "komputer.ai/squad", value: "x" }])[0].key).toMatch(/reserved/);
  });

  it("flags only the later duplicate", () => {
    const errs = validateLabelRows([{ key: "team", value: "a" }, { key: "team", value: "b" }]);
    expect(errs[0].key).toBeUndefined();
    expect(errs[1].key).toBe("Duplicate key");
  });

  it("rejects invalid key and value characters and lengths", () => {
    const errs = validateLabelRows([
      { key: "has space", value: "ok" },
      { key: "ok", value: "-leading" },
      { key: "a".repeat(64), value: "ok" },
      { key: "ok2", value: "v".repeat(64) },
      { key: "Bad_Prefix/name", value: "ok" },
    ]);
    expect(errs[0].key).toBeDefined();
    expect(errs[1].value).toBeDefined();
    expect(errs[2].key).toBeDefined();
    expect(errs[3].value).toBeDefined();
    expect(errs[4].key).toMatch(/Prefix/);
  });
});

describe("diffLabels", () => {
  const original = { team: "core", env: "prod", "komputer.ai/squad": "s1" };

  it("returns an empty diff when nothing changed", () => {
    expect(diffLabels(original, [{ key: "env", value: "prod" }, { key: "team", value: "core" }])).toEqual({
      labels: {},
      removeLabels: [],
    });
  });

  it("sets new and changed values and removes deleted keys, never touching system labels", () => {
    expect(
      diffLabels(original, [
        { key: "team", value: "data" },
        { key: "owner", value: "infra" },
      ]),
    ).toEqual({ labels: { team: "data", owner: "infra" }, removeLabels: ["env"] });
  });

  it("treats a rename as remove + set and ignores blank rows", () => {
    expect(
      diffLabels(original, [
        { key: "environment", value: "prod" },
        { key: "team", value: "core" },
        { key: " ", value: "" },
      ]),
    ).toEqual({ labels: { environment: "prod" }, removeLabels: ["env"] });
  });

  it("trims keys and values", () => {
    expect(diffLabels({}, [{ key: " team ", value: " core " }])).toEqual({ labels: { team: "core" }, removeLabels: [] });
  });
});
