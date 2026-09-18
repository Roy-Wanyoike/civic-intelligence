import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Critical Failure #5 — DB connection lost (Spec §75).
 *
 *   Simulate DB failure → API returns 503, not panic
 *
 * When the Go BFF loses its Postgres connection, every endpoint that reads
 * from the DB must return 503 Service Unavailable with a JSON body — never
 * a stack trace, never a panic, never a 500-with-HTML. The frontend must
 * then surface the same graceful error banner it shows for any other API
 * outage (see critical-failure #1).
 *
 * We mock the DB-failure state by fulfilling every /api/v1/* call with a
 * 503 + JSON body, then assert:
 *
 *   - The /debt, /scenarios, and /constitution pages all return HTTP 200
 *     (the server component caught the API failure)
 *   - Each page renders its hero header (so the user knows where they are)
 *   - Each page renders a visible error banner — never a blank state
 *   - axe-core still passes on the degraded page
 *
 * Tagged @critical for selective running:
 *   npx playwright test --grep @critical
 */
test.describe('Critical Failure #5 — DB connection lost @critical', () => {
  test.beforeEach(async ({ page }) => {
    // Every API call returns 503 + JSON body — exactly what the Go BFF
    // emits when its DB pool is exhausted (services/api/cmd/main.go
    // returns 503 from the readiness handler and from every DB-dependent
    // handler via the contracts.ErrServiceUnavailable wrapper).
    await page.route('**/api/v1/**', async (route) => {
      await route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'service unavailable',
          detail: 'database connection lost',
          retry_after_seconds: 30,
        }),
        headers: { 'Retry-After': '30' },
      });
    });
  });

  test('/debt page returns 200 + visible error banner on 503 from API', async (
    { page },
    testInfo,
  ) => {
    const response = await page.goto('/debt');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /kenya public debt/i }),
    ).toBeVisible();
    await expect(page.getByText(/failed to load/i).first()).toBeVisible();
    await runAxeTest(page, 'debt-db-down', testInfo);
  });

  test('/scenarios page returns 200 + visible error banner on 503 from API', async (
    { page },
    testInfo,
  ) => {
    const response = await page.goto('/scenarios');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /^what if\??$/i }),
    ).toBeVisible();
    await expect(page.getByText(/failed to load scenarios/i).first()).toBeVisible();
    await runAxeTest(page, 'scenarios-db-down', testInfo);
  });

  test('/constitution page degrades to bundled fallback on 503 from API', async (
    { page },
    testInfo,
  ) => {
    const response = await page.goto('/constitution');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /the constitution of kenya/i }),
    ).toBeVisible();
    // The page falls back to the bundled article set when the API fails —
    // it must surface at least one chapter heading.
    await expect(page.getByText(/chapter \d+/i).first()).toBeVisible();
    await runAxeTest(page, 'constitution-db-down', testInfo);
  });

  test('homepage still renders (no API dependency) — 503 in the API cannot take the homepage down', async (
    { page },
    testInfo,
  ) => {
    const response = await page.goto('/');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /understand what your government is doing/i }),
    ).toBeVisible();
    await runAxeTest(page, 'homepage-db-down', testInfo);
  });
});
