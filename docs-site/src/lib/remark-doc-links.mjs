import path from 'node:path';

// The docs are written for GitHub, where pages link to each other by relative
// file path (`./schedules.md`). On the site those must become page URLs
// (`/docs/concepts/schedules/`), or the browser resolves them against the
// current page and 404s.

const contentDir = path.resolve('content/docs');

/**
 * Turns a link written relative to a page in `dir` (a path under content/docs,
 * '.' for the root) into its site path. Anything that isn't a relative link to
 * a .md/.mdx page inside the docs comes back unchanged.
 */
export function docLinkToSitePath(url, dir) {
  if (/^[a-z][a-z0-9+.-]*:|^\/|^#/i.test(url)) return url;
  const hashAt = url.indexOf('#');
  const target = hashAt === -1 ? url : url.slice(0, hashAt);
  const hash = hashAt === -1 ? '' : url.slice(hashAt);
  if (!/\.mdx?$/.test(target)) return url;
  const resolved = path.posix.normalize(path.posix.join(dir, target));
  if (resolved.startsWith('..')) return url;
  const page = resolved.replace(/\.mdx?$/, '').replace(/(^|\/)index$/, '');
  return `/docs/${page ? `${page}/` : ''}${hash}`;
}

function walk(node, fn) {
  fn(node);
  for (const child of node.children ?? []) walk(child, fn);
}

/** Remark plugin: rewrites relative page links using docLinkToSitePath. */
export function remarkDocLinks() {
  return (tree, file) => {
    const rel = path.relative(contentDir, file.path ?? '');
    if (!file.path || rel.startsWith('..')) return;
    const dir = path.posix.dirname(rel.split(path.sep).join('/'));
    walk(tree, (node) => {
      if ((node.type === 'link' || node.type === 'definition') && typeof node.url === 'string') {
        node.url = docLinkToSitePath(node.url, dir);
      }
    });
  };
}
