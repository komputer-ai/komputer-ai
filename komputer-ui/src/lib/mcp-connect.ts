// Pure helpers for the Integrations page — building the MCP endpoint URL and
// the copy-pasteable commands/config for each client. No side effects, no
// `window`/`fetch` access, so these are safe to unit test and to call from
// Server/Client components alike (the page passes in `window.location.origin`
// after mount to stay hydration-safe).

export type ClaudeCodeScope = "local" | "user" | "project";

const ABSOLUTE_URL_RE = /^[a-z][a-z0-9+.-]*:\/\//i;

/** Resolve the MCP endpoint URL from the UI's runtime apiUrl.
 *  - absolute apiUrl → strip trailing slashes, append "/mcp"
 *  - relative or empty apiUrl (e.g. "" or "/api-proxy") → resolve against `origin` first
 */
export function mcpEndpointUrl(apiUrl: string, origin: string): string {
  const base = ABSOLUTE_URL_RE.test(apiUrl)
    ? apiUrl
    : `${origin.replace(/\/+$/, "")}${apiUrl ? (apiUrl.startsWith("/") ? apiUrl : `/${apiUrl}`) : ""}`;
  return `${base.replace(/\/+$/, "")}/mcp`;
}

const LOCAL_HOSTS = new Set(["localhost", "127.0.0.1", "::1", "[::1]", "0.0.0.0"]);

/** True when the URL's host is localhost / 127.0.0.1 / ::1 / 0.0.0.0 */
export function isLocalUrl(url: string): boolean {
  try {
    const { hostname } = new URL(url);
    return LOCAL_HOSTS.has(hostname.toLowerCase());
  } catch {
    return false;
  }
}

/** `claude mcp add --transport http komputer <url>`; for scope "user"/"project" insert `--scope <scope>`
 *  right after `add` (i.e. `claude mcp add --scope user --transport http komputer <url>`). "local" = no flag (Claude Code default). */
export function claudeCodeAddCommand(url: string, scope: ClaudeCodeScope): string {
  const scopeFlag = scope === "local" ? "" : ` --scope ${scope}`;
  return `claude mcp add${scopeFlag} --transport http komputer ${url}`;
}

/** Pretty-printed (2-space) JSON for claude_desktop_config.json:
 *  {"mcpServers":{"komputer":{"command":"npx","args":["-y","mcp-remote","<url>"]}}}
 *  and append "--allow-http" to args when url starts with "http://". */
export function claudeDesktopConfig(url: string): string {
  const args = ["-y", "mcp-remote", url];
  if (url.startsWith("http://")) args.push("--allow-http");
  return JSON.stringify(
    { mcpServers: { komputer: { command: "npx", args } } },
    null,
    2,
  );
}
