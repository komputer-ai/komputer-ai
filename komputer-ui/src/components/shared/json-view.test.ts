import { describe, it, expect } from "vitest";
import { resolveJsonDisplay, tokenizeJson } from "./json-view";

// ─── resolveJsonDisplay ──────────────────────────────────────────────────────

describe("resolveJsonDisplay", () => {
  it("pretty-prints a JSON-encoded string", () => {
    const { text, isJson } = resolveJsonDisplay('{"a":1,"b":[1,2]}');
    expect(isJson).toBe(true);
    expect(text).toBe(JSON.stringify({ a: 1, b: [1, 2] }, null, 2));
  });

  it("passes through a non-JSON string unchanged", () => {
    const raw = "just some log output, not json at all";
    const { text, isJson } = resolveJsonDisplay(raw);
    expect(isJson).toBe(false);
    expect(text).toBe(raw);
  });

  it("passes through a Python-repr string unchanged (legacy events)", () => {
    const raw = "{'ok': True, 'count': 3, 'note': None}";
    const { text, isJson } = resolveJsonDisplay(raw);
    expect(isJson).toBe(false);
    expect(text).toBe(raw);
  });

  it("pretty-prints an object value directly", () => {
    const value = { nested: { x: 1 }, list: [1, "two", null] };
    const { text, isJson } = resolveJsonDisplay(value);
    expect(isJson).toBe(true);
    expect(text).toBe(JSON.stringify(value, null, 2));
    expect(JSON.parse(text)).toEqual(value);
  });

  it("pretty-prints an array value directly", () => {
    const value = [1, 2, { a: "b" }];
    const { text, isJson } = resolveJsonDisplay(value);
    expect(isJson).toBe(true);
    expect(JSON.parse(text)).toEqual(value);
  });

  it("treats an empty string as non-JSON", () => {
    const { text, isJson } = resolveJsonDisplay("");
    expect(isJson).toBe(false);
    expect(text).toBe("");
  });

  it("handles null/undefined input without throwing", () => {
    expect(resolveJsonDisplay(null).isJson).toBe(true);
    expect(resolveJsonDisplay(undefined).isJson).toBe(false);
  });
});

// ─── tokenizeJson ────────────────────────────────────────────────────────────

describe("tokenizeJson", () => {
  it("classifies a key distinctly from a string value", () => {
    const text = JSON.stringify({ name: "alice" }, null, 2);
    const tokens = tokenizeJson(text);
    const key = tokens.find((t) => t.text === '"name"');
    const value = tokens.find((t) => t.text === '"alice"');
    expect(key?.type).toBe("key");
    expect(value?.type).toBe("string");
  });

  it("classifies numbers, booleans, and null", () => {
    const text = JSON.stringify({ n: 42, pi: 3.14, ok: true, bad: false, nothing: null }, null, 2);
    const tokens = tokenizeJson(text);
    expect(tokens.find((t) => t.text === "42")?.type).toBe("number");
    expect(tokens.find((t) => t.text === "3.14")?.type).toBe("number");
    expect(tokens.find((t) => t.text === "true")?.type).toBe("boolean");
    expect(tokens.find((t) => t.text === "false")?.type).toBe("boolean");
    expect(tokens.find((t) => t.text === "null")?.type).toBe("null");
  });

  it("classifies braces, brackets, commas and colons as punctuation", () => {
    const text = JSON.stringify({ a: [1, 2] }, null, 2);
    const tokens = tokenizeJson(text);
    for (const p of ["{", "}", "[", "]", ",", ":"]) {
      const tok = tokens.find((t) => t.text === p);
      expect(tok?.type).toBe("punct");
    }
  });

  it("handles string values containing quotes, colons, braces and commas", () => {
    const tricky = 'a "quoted" word: {weird, stuff}';
    const text = JSON.stringify({ msg: tricky }, null, 2);
    const tokens = tokenizeJson(text);
    // The whole string token (with escaped inner quotes) must come back intact.
    const stringToken = tokens.find((t) => t.type === "string");
    expect(stringToken?.text).toBe(JSON.stringify(tricky));
    // Re-joining every token's raw text must reproduce the original exactly.
    expect(tokens.map((t) => t.text).join("")).toBe(text);
  });

  it("handles escaped backslashes and newlines inside strings", () => {
    const tricky = "line1\\nline2 and a \\\\ backslash";
    const text = JSON.stringify({ msg: tricky }, null, 2);
    const tokens = tokenizeJson(text);
    expect(tokens.map((t) => t.text).join("")).toBe(text);
    const stringToken = tokens.find((t) => t.type === "string" && t.text.includes("line1"));
    expect(stringToken?.text).toBe(JSON.stringify(tricky));
  });

  it("round-trips an array of nested objects with every token concatenated", () => {
    const value = [{ id: 1, tags: ["a", "b"], active: true }, { id: 2, tags: [], active: false }];
    const text = JSON.stringify(value, null, 2);
    const tokens = tokenizeJson(text);
    expect(tokens.map((t) => t.text).join("")).toBe(text);
    // Keys inside array-of-objects are still distinguished from string values.
    expect(tokens.find((t) => t.text === '"id"')?.type).toBe("key");
    expect(tokens.find((t) => t.text === '"tags"')?.type).toBe("key");
    expect(tokens.find((t) => t.text === '"a"')?.type).toBe("string");
  });
});
