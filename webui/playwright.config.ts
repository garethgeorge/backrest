import { defineConfig, devices } from '@playwright/test';

// NOTE: no webServer / baseURL here. Each test boots its own backrest
// instance on a free port via the `backrest` fixture (see e2e/harness).
export default defineConfig({
  testDir: './e2e/specs',
  globalSetup: './e2e/global-setup.ts',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 4 : undefined,
  timeout: 120_000,
  expect: {
    timeout: 15_000,
  },
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : [['list']],
  use: {
    trace: 'on-first-retry',
    video: 'retain-on-failure',
    locale: 'en-US',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      // Same-tab reload hangs have been WebKit-only (#1306, #1402), so WebKit
      // runs just the specs that reload the page: login, logout and refresh.
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
      testMatch: /(auth-login|page-reload)\.spec\.ts$/,
    },
  ],
});
