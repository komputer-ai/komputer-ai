"use client";

import React from "react";

/**
 * Resolves a value (already-parsed object/array/primitive, or a string) into
 * pretty-printed text plus whether it should be rendered as colored JSON.
 *
 * - A string is parsed with JSON.parse; if that succeeds it's re-pretty-printed
 *   as JSON. If it fails (plain text, Python-repr like "{'a': True}", etc.) the
 *   original text is returned unchanged so legacy/non-JSON events still display.
 * - A non-string value is pretty-printed directly with JSON.stringify.
 */
export function resolveJsonDisplay(value: unknown): { text: string; isJson: boolean } {
  if (value === undefined) {
    return { text: "", isJson: false };
  }
  if (typeof value === "string") {
    if (value.trim() === "") {
      return { text: value, isJson: false };
    }
    try {
      const parsed = JSON.parse(value);
      return { text: JSON.stringify(parsed, null, 2), isJson: true };
    } catch {
      return { text: value, isJson: false };
    }
  }
  try {
    return { text: JSON.stringify(value, null, 2), isJson: true };
  } catch {
    return { text: String(value), isJson: false };
  }
}

export type JsonTokenType =
  | "key"
  | "string"
  | "number"
  | "boolean"
  | "null"
  | "punct"
  | "space";

export interface JsonToken {
  type: JsonTokenType;
  text: string;
}

// Matches, in priority order: a quoted string (handling escaped chars so an
// escaped quote doesn't end the match early), a number, the true/false/null
// keywords, JSON punctuation, runs of whitespace, and finally any single
// leftover character as a safety net. Every character of the input is
// consumed by exactly one alternative, so joining all matches reproduces the
// original string exactly.
const TOKEN_REGEX = /"(?:\\.|[^"\\])*"|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null|[{}[\],:]|\s+|[\s\S]/g;

function classify(token: string): JsonTokenType {
  if (token[0] === '"') return "string";
  if (token === "true" || token === "false") return "boolean";
  if (token === "null") return "null";
  if (token.length === 1 && "{}[],:".includes(token)) return "punct";
  if (/^-?\d/.test(token)) return "number";
  return "space";
}

/** Tokenizes pretty-printed JSON text for syntax coloring. A quoted string
 * immediately followed by a colon (ignoring whitespace) is classified as a
 * "key" rather than a "string". */
export function tokenizeJson(text: string): JsonToken[] {
  const raw = text.match(TOKEN_REGEX) ?? [];
  const tokens: JsonToken[] = raw.map((t) => ({ type: classify(t), text: t }));
  for (let i = 0; i < tokens.length; i++) {
    if (tokens[i].type !== "string") continue;
    let j = i + 1;
    while (j < tokens.length && tokens[j].type === "space") j++;
    if (j < tokens.length && tokens[j].type === "punct" && tokens[j].text === ":") {
      tokens[i] = { ...tokens[i], type: "key" };
    }
  }
  return tokens;
}

const TOKEN_COLOR_CLASS: Record<JsonTokenType, string> = {
  key: "text-[var(--color-brand-blue-light)]",
  string: "text-[var(--color-status-success)]",
  number: "text-[var(--color-brand-violet-light)]",
  boolean: "text-[var(--color-status-pending)]",
  null: "text-[var(--color-text-muted)]",
  punct: "text-[var(--color-text-secondary)]",
  space: "",
};

export type JsonViewProps = {
  /** An already-parsed value, or a string (JSON-encoded or plain text). */
  value: unknown;
  className?: string;
  style?: React.CSSProperties;
};

/**
 * Renders a value as syntax-colored, pretty-printed JSON when possible,
 * falling back to the raw text unstyled when it isn't valid JSON (plain
 * strings, or legacy Python-repr tool output already stored in events).
 * Never uses dangerouslySetInnerHTML — every token is its own text node.
 */
export function JsonView({ value, className, style }: JsonViewProps) {
  const { text, isJson } = resolveJsonDisplay(value);
  if (!isJson) {
    return (
      <pre className={className} style={style}>
        {text}
      </pre>
    );
  }
  const tokens = tokenizeJson(text);
  return (
    <pre className={className} style={style}>
      {tokens.map((tok, i) =>
        tok.type === "space" ? (
          tok.text
        ) : (
          <span key={i} className={TOKEN_COLOR_CLASS[tok.type]}>
            {tok.text}
          </span>
        )
      )}
    </pre>
  );
}
