// API bearer token for a komputer-api with auth on (KOMPUTER_API_TOKEN).
// Kept in this browser's localStorage, entered by the user when the API
// answers 401. Never put the token in config.js: that file is public.

const STORAGE_KEY = "komputer.apiToken";
export const UNAUTHORIZED_EVENT = "komputer:unauthorized";

export function getApiToken(): string {
  if (typeof window === "undefined") return "";
  try {
    return window.localStorage.getItem(STORAGE_KEY) ?? "";
  } catch {
    return "";
  }
}

export function setApiToken(token: string) {
  try {
    if (token) window.localStorage.setItem(STORAGE_KEY, token);
    else window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Storage blocked: the token lasts for this page only.
  }
}

export function authHeaders(): Record<string, string> {
  const token = getApiToken();
  return token ? { Authorization: `Bearer ${token}` } : {};
}

// Browsers can't set headers on a WebSocket, so the token goes in the query.
export function withWsToken(url: string): string {
  const token = getApiToken();
  if (!token) return url;
  return `${url}${url.includes("?") ? "&" : "?"}access_token=${encodeURIComponent(token)}`;
}

export function notifyUnauthorized() {
  if (typeof window !== "undefined") window.dispatchEvent(new Event(UNAUTHORIZED_EVENT));
}
