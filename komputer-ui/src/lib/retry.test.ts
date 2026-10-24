import { describe, expect, it } from 'vitest';
import { retryUntilLoaded } from './retry';

describe('retryUntilLoaded', () => {
  it('keeps trying after failures and resolves with the first success', async () => {
    let calls = 0;
    const waits: number[] = [];
    const fetcher = async () => {
      calls++;
      if (calls < 3) throw new Error('API restarting');
      return { s3: { enabled: true } };
    };
    const got = await retryUntilLoaded(fetcher, 3000, async (ms) => { waits.push(ms); });
    expect(got).toEqual({ s3: { enabled: true } });
    expect(calls).toBe(3);
    expect(waits).toEqual([3000, 3000]);
  });

  it('does not wait when the first try succeeds', async () => {
    const waits: number[] = [];
    await retryUntilLoaded(async () => 1, 3000, async (ms) => { waits.push(ms); });
    expect(waits).toEqual([]);
  });
});
