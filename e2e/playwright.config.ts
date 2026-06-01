import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright config for the translator check-in system.
 *
 * Targets the e2e docker-compose stack (docker-compose.e2e.yml, project
 * name "thai-e2e") which exposes the frontend on http://localhost:3001.
 * Bring the stack up before running tests:
 *
 *   npm run stack:up
 *   npm test
 *
 * The reset endpoint at /api/test/reset is only available when the backend
 * is built with `-tags e2e` AND env ENABLE_TEST_RESET=true. Both conditions
 * are baked into the e2e compose stack. See docker-compose.e2e.yml.
 */

const BASE_URL = process.env.E2E_BASE_URL ?? 'http://localhost:3001';

export default defineConfig({
  testDir: './tests',
  globalSetup: './global-setup.ts',
  fullyParallel: false, // Tests share DB state via reset endpoint — serialize
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI
    ? [['github'], ['html', { open: 'never' }]]
    : [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    locale: 'zh-TW',
    timezoneId: 'Asia/Bangkok',
  },
  expect: {
    timeout: 8_000,
  },
  timeout: 60_000,
  projects: [
    {
      name: 'chromium-desktop',
      use: { ...devices['Desktop Chrome'] },
      // responsive.spec.ts asserts mobile-only behaviours (drawer collapse).
      // It cannot pass on a desktop viewport — skip it here.
      testIgnore: /responsive\.spec\.ts/,
    },
    {
      name: 'mobile-chrome',
      use: { ...devices['Pixel 7'] },
      testMatch: /responsive\.spec\.ts/,
    },
  ],
});
