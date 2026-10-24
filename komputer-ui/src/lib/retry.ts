const sleepFor = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

/** Calls fetcher until it succeeds, waiting delayMs between tries. */
export async function retryUntilLoaded<T>(
  fetcher: () => Promise<T>,
  delayMs = 3000,
  sleep: (ms: number) => Promise<void> = sleepFor,
): Promise<T> {
  for (;;) {
    try {
      return await fetcher();
    } catch {
      await sleep(delayMs);
    }
  }
}
