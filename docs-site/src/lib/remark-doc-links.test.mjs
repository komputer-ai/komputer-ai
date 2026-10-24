// Run with: node --test src/lib
import assert from 'node:assert/strict';
import { test } from 'node:test';
import { docLinkToSitePath } from './remark-doc-links.mjs';

test('relative .md links become site paths', () => {
  assert.equal(docLinkToSitePath('./session-store.md', 'concepts'), '/docs/concepts/session-store/');
  assert.equal(docLinkToSitePath('schedules.md#agent-configuration', 'concepts'), '/docs/concepts/schedules/#agent-configuration');
  assert.equal(docLinkToSitePath('../integration/rest-api.md', 'concepts'), '/docs/integration/rest-api/');
  assert.equal(docLinkToSitePath('./caveats.mdx', 'concepts/squads'), '/docs/concepts/squads/caveats/');
  assert.equal(docLinkToSitePath('concepts/agents.md', '.'), '/docs/concepts/agents/');
  assert.equal(docLinkToSitePath('./index.mdx', 'concepts'), '/docs/concepts/');
});

test('other links are left alone', () => {
  for (const url of ['https://example.com/a.md', '/docs/concepts/agents/', '#lifecycle', 'mailto:a@b.c', './diagram.png', '../../helm/values.yaml']) {
    assert.equal(docLinkToSitePath(url, 'concepts'), url);
  }
});

test('links that leave the docs tree are left alone', () => {
  assert.equal(docLinkToSitePath('../../README.md', 'concepts'), '../../README.md');
});
