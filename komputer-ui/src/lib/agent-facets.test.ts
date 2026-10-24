import { describe, expect, it } from 'vitest';
import { normalizeAgentFacets } from './agent-facets';

describe('normalizeAgentFacets', () => {
  it('turns null lists into empty ones (older APIs send null when no agent has labels)', () => {
    expect(normalizeAgentFacets({ namespaces: ['default'], labels: null })).toEqual({ namespaces: ['default'], labels: [] });
    expect(normalizeAgentFacets({ namespaces: null, labels: null })).toEqual({ namespaces: [], labels: [] });
  });

  it('returns undefined when there are no facets', () => {
    expect(normalizeAgentFacets(undefined)).toBeUndefined();
  });

  it('keeps lists that are present', () => {
    expect(normalizeAgentFacets({ namespaces: ['a'], labels: ['team=core'] })).toEqual({ namespaces: ['a'], labels: ['team=core'] });
  });
});
