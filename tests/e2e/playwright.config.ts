import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright config for the Civic Intelligence Platform.
 *
 * - Base URL: http://localhost:3000 (Next.js dev server)
 * - Browsers: chromium, firefox, webkit
 * - Retry: 2x on CI
 * - Screenshots: only on failure
 * - Video: retain-on-failure
 * - Trace: on-first-retry
 *
 * Spec §74 (golden journeys) + §75 (critical failure tests) live under
 * tests/e2e/golden-journeys/ and tests/e2e/critical-failures/ respectively.
 *
 * Selective runs:
 *   npx playwright test --grep @golden     # only golden journeys
 *   npx playwright test --grep @critical   # only critical-failure tests
 *   npx playwright test --grep @a11y       # only accessibility tests
 */
export default defineConfig({
  // Config lives at tests/e2e/playwright.config.ts; specs are siblings.
  testDir: '.',
  testMatch: /.*\.spec\.ts$/,
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'html',
  expect: {
    // Default assertion timeout — pages render fast against the mock fallbacks.
    timeout: 7_000,
  },
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
    // Reduce flake from slow CI runners.
    locale: 'en-KE',
    timezoneId: 'Africa/Nairobi',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
  webServer: {
    command: 'npm run start',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
