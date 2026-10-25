import { describe, it, expect } from 'vitest';
import { humanizeConditionReason } from './agent-conditions';

describe('humanizeConditionReason', () => {
  it('maps known reasons', () => {
    expect(humanizeConditionReason('SleepTTL')).toBe('Sleep TTL');
    expect(humanizeConditionReason('ArchiveTTL')).toBe('Archive TTL');
    expect(humanizeConditionReason('DeleteTTL')).toBe('Delete TTL');
    expect(humanizeConditionReason('ArchiveRequested')).toBe('Archive requested');
    expect(humanizeConditionReason('SleepRequested')).toBe('Sleep requested');
    expect(humanizeConditionReason('LifecycleSleep')).toBe('Lifecycle: Sleep');
    expect(humanizeConditionReason('TaskTimeout')).toBe('Task timeout');
    expect(humanizeConditionReason('TaskStarted')).toBe('Task started');
    expect(humanizeConditionReason('Woken')).toBe('Woken');
  });

  it('falls back to splitting CamelCase for unknown reasons', () => {
    expect(humanizeConditionReason('SecretsSynced')).toBe('Secrets synced');
    expect(humanizeConditionReason('SessionUploadFailed')).toBe('Session upload failed');
    expect(humanizeConditionReason('NodeTTLExceeded')).toBe('Node TTL exceeded');
  });

  it('returns empty for missing reason', () => {
    expect(humanizeConditionReason(undefined)).toBe('');
    expect(humanizeConditionReason('')).toBe('');
  });
});
