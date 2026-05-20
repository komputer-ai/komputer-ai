// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { loadCustomModels, addCustomModel, STORAGE_KEY } from './custom-models';

describe('custom-models', () => {
  beforeEach(() => {
    // jsdom provides window.localStorage; clear between tests.
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('loadCustomModels', () => {
    it('returns empty array when storage is empty', () => {
      expect(loadCustomModels()).toEqual([]);
    });

    it('returns the stored list', () => {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(['a', 'b']));
      expect(loadCustomModels()).toEqual(['a', 'b']);
    });

    it('returns empty array when storage contains garbage', () => {
      window.localStorage.setItem(STORAGE_KEY, 'not-json');
      expect(loadCustomModels()).toEqual([]);
    });

    it('returns empty array when storage contains a non-array value', () => {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ a: 1 }));
      expect(loadCustomModels()).toEqual([]);
    });
  });

  describe('addCustomModel', () => {
    it('persists and returns the new list', () => {
      const out = addCustomModel('us.anthropic.claude-sonnet-4-5-20250929-v1:0');
      expect(out).toEqual(['us.anthropic.claude-sonnet-4-5-20250929-v1:0']);
      expect(JSON.parse(window.localStorage.getItem(STORAGE_KEY)!)).toEqual([
        'us.anthropic.claude-sonnet-4-5-20250929-v1:0',
      ]);
    });

    it('trims whitespace from the input', () => {
      const out = addCustomModel('  foo  ');
      expect(out).toEqual(['foo']);
    });

    it('does not add an empty string', () => {
      const out = addCustomModel('   ');
      expect(out).toEqual([]);
      expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
    });

    it('deduplicates against existing entries', () => {
      addCustomModel('foo');
      const out = addCustomModel('foo');
      expect(out).toEqual(['foo']);
    });
  });
});
