import { retryUntilLoaded } from "./retry";

/**
 * Loads a config with one shared request at a time, reusing an answer for
 * ttlMs and asking again after that — so a change such as turning the S3
 * session store on or off reaches open pages without a reload.
 */
export function createConfigLoader<T>(
  fetcher: () => Promise<T>,
  ttlMs = 30_000,
  now: () => number = Date.now,
): () => Promise<T> {
  let cached: { value: T; at: number } | null = null;
  let inflight: Promise<T> | null = null;
  return () => {
    if (cached && now() - cached.at < ttlMs) return Promise.resolve(cached.value);
    if (!inflight) {
      inflight = retryUntilLoaded(fetcher)
        .then((value) => {
          cached = { value, at: now() };
          return value;
        })
        .finally(() => {
          inflight = null;
        });
    }
    return inflight;
  };
}
