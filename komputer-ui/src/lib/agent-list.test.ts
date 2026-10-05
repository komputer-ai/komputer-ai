import { describe, it, expect } from 'vitest';
import {
  filterAndSortAgents,
  labelSuggestions,
  type AgentListFilters,
} from './agent-list';
import type { AgentResponse } from './types';

function mk(overrides: Partial<AgentResponse> & { name: string }): AgentResponse {
  return {
    name: overrides.name,
    namespace: overrides.namespace ?? 'default',
    model: 'claude-sonnet-4-6',
    status: overrides.status ?? 'Running',
    createdAt: overrides.createdAt ?? '2024-01-01T00:00:00Z',
    lastActivityAt: overrides.lastActivityAt,
    labels: overrides.labels,
  } as AgentResponse;
}

const baseFilters: AgentListFilters = {
  namespace: '',
  status: 'All',
  search: '',
  labels: [],
  sort: 'newest',
};

describe('filterAndSortAgents — label filters', () => {
  const agents = [
    mk({ name: 'a', labels: { team: 'core', env: 'prod' } }),
    mk({ name: 'b', labels: { team: 'core', env: 'staging' } }),
    mk({ name: 'c', labels: { team: 'growth' } }),
    mk({ name: 'd' }), // no labels at all
  ];

  it('returns everything when no label filters are set', () => {
    const result = filterAndSortAgents(agents, baseFilters);
    expect(result.map((a) => a.name)).toEqual(['a', 'b', 'c', 'd']);
  });

  it('filters to agents matching a single key=value label', () => {
    const result = filterAndSortAgents(agents, { ...baseFilters, labels: ['team=core'] });
    expect(result.map((a) => a.name).sort()).toEqual(['a', 'b']);
  });

  it('ANDs multiple label filters together', () => {
    const result = filterAndSortAgents(agents, {
      ...baseFilters,
      labels: ['team=core', 'env=prod'],
    });
    expect(result.map((a) => a.name)).toEqual(['a']);
  });

  it('excludes an agent with no labels when a label filter is active', () => {
    const result = filterAndSortAgents(agents, { ...baseFilters, labels: ['team=core'] });
    expect(result.map((a) => a.name)).not.toContain('d');
  });

  it('excludes agents whose label value does not match', () => {
    const result = filterAndSortAgents(agents, { ...baseFilters, labels: ['env=prod'] });
    expect(result.map((a) => a.name)).toEqual(['a']);
  });
});

describe('filterAndSortAgents — namespace/status/search (existing behavior)', () => {
  const agents = [
    mk({ name: 'a', namespace: 'ns1', status: 'Running' }),
    mk({ name: 'b', namespace: 'ns2', status: 'Failed' }),
  ];

  it('filters by namespace', () => {
    const result = filterAndSortAgents(agents, { ...baseFilters, namespace: 'ns1' });
    expect(result.map((a) => a.name)).toEqual(['a']);
  });

  it('filters by status', () => {
    const result = filterAndSortAgents(agents, { ...baseFilters, status: 'Failed' });
    expect(result.map((a) => a.name)).toEqual(['b']);
  });

  it('filters by name search, case-insensitively', () => {
    const result = filterAndSortAgents(agents, { ...baseFilters, search: 'A' });
    expect(result.map((a) => a.name)).toEqual(['a']);
  });
});

describe('filterAndSortAgents — sort modes', () => {
  const agents = [
    mk({ name: 'old', createdAt: '2024-01-01T00:00:00Z' }),
    mk({ name: 'new', createdAt: '2024-03-01T00:00:00Z' }),
    mk({ name: 'mid', createdAt: '2024-02-01T00:00:00Z' }),
  ];

  it('"newest" sorts by createdAt descending', () => {
    const result = filterAndSortAgents(agents, { ...baseFilters, sort: 'newest' });
    expect(result.map((a) => a.name)).toEqual(['new', 'mid', 'old']);
  });

  it('"oldest" sorts by createdAt ascending', () => {
    const result = filterAndSortAgents(agents, { ...baseFilters, sort: 'oldest' });
    expect(result.map((a) => a.name)).toEqual(['old', 'mid', 'new']);
  });

  it('"recent-active" sorts by lastActivityAt descending', () => {
    const active = [
      mk({ name: 'stale', createdAt: '2024-01-01T00:00:00Z', lastActivityAt: '2024-01-02T00:00:00Z' }),
      mk({ name: 'fresh', createdAt: '2024-01-01T00:00:00Z', lastActivityAt: '2024-03-01T00:00:00Z' }),
    ];
    const result = filterAndSortAgents(active, { ...baseFilters, sort: 'recent-active' });
    expect(result.map((a) => a.name)).toEqual(['fresh', 'stale']);
  });

  it('"recent-active" puts never-active agents last, tie-broken by createdAt desc', () => {
    const mixed = [
      mk({ name: 'never-old', createdAt: '2024-01-01T00:00:00Z' }),
      mk({ name: 'active', createdAt: '2024-01-01T00:00:00Z', lastActivityAt: '2024-01-05T00:00:00Z' }),
      mk({ name: 'never-new', createdAt: '2024-02-01T00:00:00Z' }),
    ];
    const result = filterAndSortAgents(mixed, { ...baseFilters, sort: 'recent-active' });
    expect(result.map((a) => a.name)).toEqual(['active', 'never-new', 'never-old']);
  });

  it('"recent-active" tie-breaks two never-active agents by createdAt desc', () => {
    const mixed = [
      mk({ name: 'never-a', createdAt: '2024-01-01T00:00:00Z' }),
      mk({ name: 'never-b', createdAt: '2024-05-01T00:00:00Z' }),
    ];
    const result = filterAndSortAgents(mixed, { ...baseFilters, sort: 'recent-active' });
    expect(result.map((a) => a.name)).toEqual(['never-b', 'never-a']);
  });

  it('is stable (preserves input order) for equal sort keys', () => {
    const sameTime = [
      mk({ name: 'x', createdAt: '2024-01-01T00:00:00Z' }),
      mk({ name: 'y', createdAt: '2024-01-01T00:00:00Z' }),
      mk({ name: 'z', createdAt: '2024-01-01T00:00:00Z' }),
    ];
    const result = filterAndSortAgents(sameTime, { ...baseFilters, sort: 'newest' });
    expect(result.map((a) => a.name)).toEqual(['x', 'y', 'z']);
  });
});

describe('labelSuggestions', () => {
  it('collects unique key=value pairs present across agents', () => {
    const agents = [
      mk({ name: 'a', labels: { team: 'core', env: 'prod' } }),
      mk({ name: 'b', labels: { team: 'core' } }),
      mk({ name: 'c' }),
    ];
    expect(labelSuggestions(agents).sort()).toEqual(['env=prod', 'team=core']);
  });

  it('returns an empty array when no agents have labels', () => {
    expect(labelSuggestions([mk({ name: 'a' })])).toEqual([]);
  });
});
