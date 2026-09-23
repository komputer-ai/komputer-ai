import { defineConfig } from 'vitest/config';
import { fileURLToPath } from 'node:url';

export default defineConfig({
  resolve: {
    // Mirror tsconfig's "@/*" → "./src/*" so component modules can be imported in tests.
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  test: {
    environment: 'node',
    environmentMatchGlobs: [['src/components/**', 'jsdom']],
  },
});
