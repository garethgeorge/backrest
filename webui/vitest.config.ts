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
