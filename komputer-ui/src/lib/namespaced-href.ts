/**
 * Build a path that carries the namespace query param.
 *
 * Most resource pages (agents, schedules, offices, squads) read namespace
 * from `?namespace=…`. Forgetting it sends users to the wrong record (or a
 * 404). Use this helper for every outbound link or programmatic navigation
 * to those pages.
 *
 * If `namespace` is undefined or "default", the path is returned unchanged —
 * "default" is the implicit fallback in the API and adding it adds noise.
 *
 * If the path already contains a query string, the namespace is merged in.
 */
export function namespacedHref(path: string, namespace?: string | null): string {
  if (!namespace || namespace === "default") return path;
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}namespace=${encodeURIComponent(namespace)}`;
}
