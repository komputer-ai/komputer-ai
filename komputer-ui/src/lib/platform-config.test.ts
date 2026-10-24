import { describe, expect, it } from 'vitest';
import { createConfigLoader } from './platform-config';

describe('createConfigLoader', () => {
  it('reuses a fresh answer and asks again once it is stale', async () => {
    let now = 0;
    let s3 = false;
    let calls = 0;
    const load = createConfigLoader(async () => { calls++; return { s3: { enabled: s3 } }; }, 30_000, () => now);

    expect((await load()).s3.enabled).toBe(false);
    s3 = true; // S3 turned on in the cluster
    now = 10_000;
    expect((await load()).s3.enabled).toBe(false); // still fresh: no new request
    expect(calls).toBe(1);

    now = 31_000;
    expect((await load()).s3.enabled).toBe(true); // stale: asked again, sees the change
    expect(calls).toBe(2);
  });

  it('shares one request between concurrent callers', async () => {
    let calls = 0;
    const load = createConfigLoader(async () => { calls++; return { s3: { enabled: true } }; }, 30_000, () => 0);
    await Promise.all([load(), load(), load()]);
    expect(calls).toBe(1);
  });
});
