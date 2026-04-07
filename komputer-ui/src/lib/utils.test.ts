import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { formatCost, formatRelativeTime, cronToHuman } from './utils';

// ─── formatCost ───────────────────────────────────────────────────────────────

describe('formatCost', () => {
  it('returns dash for undefined', () => {
    expect(formatCost(undefined)).toBe('—');
  });

  it('returns dash for empty string', () => {
    expect(formatCost('')).toBe('—');
  });

  it('returns dash for non-numeric string', () => {
    expect(formatCost('abc')).toBe('—');
  });

  it('formats zero correctly', () => {
    expect(formatCost('0')).toBe('$0.0000');
  });

  it('formats typical cost with 4 decimal places', () => {
    expect(formatCost('0.0123')).toBe('$0.0123');
  });

  it('rounds to 4 decimal places', () => {
    expect(formatCost('1.23456789')).toBe('$1.2346');
  });

  it('formats integer string', () => {
    expect(formatCost('5')).toBe('$5.0000');
  });

  it('formats negative value', () => {
    expect(formatCost('-0.0050')).toBe('$-0.0050');
  });
});

// ─── cronToHuman ──────────────────────────────────────────────────────────────

describe('cronToHuman', () => {
  it('returns every N minutes for */N minute pattern', () => {
    expect(cronToHuman('*/15 * * * *')).toBe('Every 15 minutes');
  });

  it('returns every N minutes for */30', () => {
    expect(cronToHuman('*/30 * * * *')).toBe('Every 30 minutes');
  });

  it('returns every N hours for */N hour pattern', () => {
    expect(cronToHuman('0 */2 * * *')).toBe('Every 2 hours');
  });

  it('returns weekdays format for MON-FRI', () => {
    expect(cronToHuman('0 9 * * MON-FRI')).toBe('Weekdays at 9:00');
  });

  it('pads minutes with zero for weekday schedule', () => {
    expect(cronToHuman('5 9 * * MON-FRI')).toBe('Weekdays at 9:05');
  });

  it('returns daily format for daily cron', () => {
    expect(cronToHuman('0 8 * * *')).toBe('Daily at 8:00');
  });

  it('returns monthly format for 1st of month', () => {
    expect(cronToHuman('0 10 1 * *')).toBe('Monthly on the 1st at 10:00');
  });

  it('returns raw cron for unrecognized pattern', () => {
    const cron = '15 14 5 * *';
    expect(cronToHuman(cron)).toBe(cron);
  });

  it('returns raw input for invalid cron (wrong part count)', () => {
    expect(cronToHuman('* * *')).toBe('* * *');
  });

  it('returns raw input for empty string', () => {
    expect(cronToHuman('')).toBe('');
  });
});

// ─── formatRelativeTime ───────────────────────────────────────────────────────

describe('formatRelativeTime', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2024-06-01T12:00:00Z'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns seconds ago for recent timestamp', () => {
    const ts = new Date('2024-06-01T11:59:45Z').toISOString();
    expect(formatRelativeTime(ts)).toBe('15s ago');
  });

  it('returns minutes ago', () => {
    const ts = new Date('2024-06-01T11:55:00Z').toISOString();
    expect(formatRelativeTime(ts)).toBe('5m ago');
  });

  it('returns hours ago', () => {
    const ts = new Date('2024-06-01T09:00:00Z').toISOString();
    expect(formatRelativeTime(ts)).toBe('3h ago');
  });

  it('returns days ago', () => {
    const ts = new Date('2024-05-29T12:00:00Z').toISOString();
    expect(formatRelativeTime(ts)).toBe('3d ago');
  });

  it('returns just now for future timestamp within 60s', () => {
    const ts = new Date('2024-06-01T12:00:30Z').toISOString();
    expect(formatRelativeTime(ts)).toBe('just now');
  });

  it('returns in Xm for future timestamp minutes away', () => {
    const ts = new Date('2024-06-01T12:10:00Z').toISOString();
    expect(formatRelativeTime(ts)).toBe('in 10m');
  });

  it('returns in Xh for future timestamp hours away', () => {
    const ts = new Date('2024-06-01T15:00:00Z').toISOString();
    expect(formatRelativeTime(ts)).toBe('in 3h');
  });

  it('returns in Xd for future timestamp days away', () => {
    const ts = new Date('2024-06-04T12:00:00Z').toISOString();
    expect(formatRelativeTime(ts)).toBe('in 3d');
  });
});
