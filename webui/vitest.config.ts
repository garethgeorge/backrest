import { defineConfig } from 'vitest/config';

// Standalone test config, intentionally not merged with vite.config.ts: the
// paraglide/compression plugins aren't needed at test time (paraglide output is
// precompiled by the `compile-i18n` script), and `test.env` below replaces the
// vite `define` block since process.env reads work natively under vitest.
export default defineConfig({
  resolve: {
    tsconfigPaths: true,
  },
  test: {
    environment: 'jsdom',
    globals: true,
    // userEvent-driven component tests run 3-5s each and tip past the default
    // 5s timeout under CI load; give headroom without masking real hangs.
    testTimeout: 20_000,
    setupFiles: ['./src/test/setup.tsx'],
    include: ['src/**/*.test.{ts,tsx}'],
    restoreMocks: true,
    unstubGlobals: true,
    env: {
      UI_OS: 'unix',
      BACKREST_BUILD_VERSION: 'test-build',
      UI_BACKEND_URL: 'http://localhost:9898',
      UI_FEATURES: '',
      TZ: 'UTC',
    },
  },
});
