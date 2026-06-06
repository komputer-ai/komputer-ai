/**
 * Fetched at build time (Next.js static export). The result is baked into the
 * generated HTML, so visitors never make a GitHub API call themselves.
 * The CI rerun on every push keeps this fresh.
 */

interface ReleaseInfo {
  tag: string;
  name: string;
  url: string;
}

const FALLBACK: ReleaseInfo = {
  tag: 'v0.15.9',
  name: 'v0.15.9 — Namespace & Timestamp Polish',
  url: 'https://github.com/komputer-ai/komputer-ai/releases',
};

let cached: ReleaseInfo | null = null;

export async function getLatestRelease(): Promise<ReleaseInfo> {
  if (cached) return cached;
  try {
    const res = await fetch(
      'https://api.github.com/repos/komputer-ai/komputer-ai/releases/latest',
      {
        headers: { Accept: 'application/vnd.github+json' },
        next: { revalidate: 3600 },
      },
    );
    if (!res.ok) throw new Error(`GitHub API ${res.status}`);
    const data = (await res.json()) as { tag_name: string; name: string; html_url: string };
    cached = {
      tag: data.tag_name,
      name: data.name || data.tag_name,
      url: data.html_url,
    };
    return cached;
  } catch (err) {
    console.warn('[release] falling back to hardcoded release info:', err);
    return FALLBACK;
  }
}
