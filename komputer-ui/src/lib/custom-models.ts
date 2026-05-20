/**
 * Persistence layer for user-added model identifiers.
 *
 * Built-in models come from `MODELS` in `./constants`. Anything the user types
 * into the "Add model" form in the picker UI is appended here and survives
 * page reloads on the same browser. The agent / API / operator never see this
 * list — they only ever receive the currently-selected string.
 */

export const STORAGE_KEY = "komputer.customModels";

export function loadCustomModels(): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter((v): v is string => typeof v === "string") : [];
  } catch {
    return [];
  }
}

export function addCustomModel(name: string): string[] {
  const trimmed = name.trim();
  if (!trimmed) return loadCustomModels();
  const current = loadCustomModels();
  if (current.includes(trimmed)) return current;
  const next = [...current, trimmed];
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  return next;
}
