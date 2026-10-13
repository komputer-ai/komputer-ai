// Helpers for the agent labels key/value editor. Edits are staged as rows and
// turned into a PATCH body (labels to set + keys to remove) on Save.

export interface LabelRow {
  key: string;
  value: string;
}

export interface LabelRowErrors {
  key?: string;
  value?: string;
}

const SYSTEM_PREFIX = "komputer.ai/";
const NAME_RE = /^[A-Za-z0-9]([-A-Za-z0-9_.]*[A-Za-z0-9])?$/;
const DNS_SUBDOMAIN_RE = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/;

export const isSystemLabel = (key: string) => key.startsWith(SYSTEM_PREFIX);

/** Splits agent labels into editable user rows and read-only komputer.ai/* entries, both sorted by key. */
export function splitLabels(labels: Record<string, string> | undefined): {
  user: LabelRow[];
  system: [string, string][];
} {
  const entries = Object.entries(labels ?? {}).sort(([a], [b]) => a.localeCompare(b));
  return {
    user: entries.filter(([k]) => !isSystemLabel(k)).map(([key, value]) => ({ key, value })),
    system: entries.filter(([k]) => isSystemLabel(k)),
  };
}

const isBlank = (r: LabelRow) => !r.key.trim() && !r.value.trim();

/** Kubernetes label rules (qualified-name key, ≤63-char value), plus reserved prefix and duplicate checks. */
export function validateLabelRows(rows: LabelRow[]): LabelRowErrors[] {
  const seen = new Map<string, number>();
  rows.forEach((r, i) => {
    const k = r.key.trim();
    if (k && !seen.has(k)) seen.set(k, i);
  });
  return rows.map((r, i) => {
    const errs: LabelRowErrors = {};
    if (isBlank(r)) return errs;
    const key = r.key.trim();
    const value = r.value.trim();
    if (!key) {
      errs.key = "Key is required";
    } else if (isSystemLabel(key)) {
      errs.key = "komputer.ai/ is reserved";
    } else if (seen.get(key) !== i) {
      errs.key = "Duplicate key";
    } else {
      const slash = key.indexOf("/");
      const prefix = slash >= 0 ? key.slice(0, slash) : "";
      const name = slash >= 0 ? key.slice(slash + 1) : key;
      if (slash >= 0 && (!prefix || prefix.length > 253 || !DNS_SUBDOMAIN_RE.test(prefix))) {
        errs.key = "Prefix must be a DNS subdomain (e.g. example.com/)";
      } else if (name.length > 63 || !NAME_RE.test(name)) {
        errs.key = "Letters, digits, - _ . (max 63, start/end alphanumeric)";
      }
    }
    if (value && (value.length > 63 || !NAME_RE.test(value))) {
      errs.value = "Letters, digits, - _ . (max 63, start/end alphanumeric)";
    }
    return errs;
  });
}

export const hasLabelErrors = (errs: LabelRowErrors[]) => errs.some((e) => e.key || e.value);

/**
 * Diffs edited rows against the agent's current user labels. Blank rows are
 * ignored; a renamed key becomes a remove of the old key plus a set of the new one.
 */
export function diffLabels(
  original: Record<string, string> | undefined,
  rows: LabelRow[],
): { labels: Record<string, string>; removeLabels: string[] } {
  const before = Object.fromEntries(Object.entries(original ?? {}).filter(([k]) => !isSystemLabel(k)));
  const after: Record<string, string> = {};
  for (const r of rows) {
    if (isBlank(r)) continue;
    after[r.key.trim()] = r.value.trim();
  }
  const labels: Record<string, string> = {};
  for (const [k, v] of Object.entries(after)) {
    if (before[k] !== v) labels[k] = v;
  }
  const removeLabels = Object.keys(before).filter((k) => !(k in after)).sort();
  return { labels, removeLabels };
}
