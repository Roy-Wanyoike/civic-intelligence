import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Critical Failure #1 — API unavailable (Spec §75).
 *
 *   Simulate API down → Frontend shows graceful error, not blank page
 *
 * The platform's spec §75 critical-failure contract requires every page
 * that depends on the Go BFF to degrade gracefully when the API is
 * unreachable. We simulate a complete API outage by aborting every
 * /api/v1/* request, then assert that:
 *
 *   - The homepage (which has no API dependency) still renders.
 *   - The /debt page (server component) renders its hero header AND a
 *     visible error banner — never a blank page or a Next.js error overlay.
 *   - The /scenarios page renders its disclaimer AND a "failed to load"
 *     error banner.
 *   - The axe-core scan still passes on the error state — accessibility
 *     does not regress during an outage.
 *
 * Tagged @critical for selective running:
 *   npx playwright test --grep @critical
 */
test.describe('Critical Failure #1 — API unavailable @critical', () => {
  test.beforeEach(async ({ page }) => {
    // Abort every API call so server components receive a fetch rejection.
    await page.route('**/api/v1/**', (route) => route.abort('failed'));
  });

  test('homepage renders (no API dependency)', async ({ page }, testInfo) => {
    const response = await page.goto('/');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /understand what your government is doing/i }),
    ).toBeVisible();
    await runAxeTest(page, 'homepage-api-down', testInfo);
  });

  test('/debt page renders a graceful error banner, not a blank page', async (
    { page },
    testInfo,
  ) => {
    const response = await page.goto('/debt');
    // The page must return 200 even when the API is down — a 500 would be a
    // critical failure of its own.
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /kenya public debt/i }),
    ).toBeVisible();
    // The error banner is rendered via {loadError && (...)} in the page —
    // it must be visible rather than a silent empty state.
    await expect(page.getByText(/failed to load/i).first()).toBeVisible();
    await runAxeTest(page, 'debt-api-down', testInfo);
  });

  test('/scenarios page renders disclaimer + error banner', async ({ page }, testInfo) => {
    const response = await page.goto('/scenarios');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /^what if\??$/i }),
    ).toBeVisible();
    await expect(page.getByText(/hypothetical/i).first()).toBeVisible();
    // The page surfaces a "Failed to load scenarios" banner when the API
    // rejects — never a blank state.
    await expect(page.getByText(/failed to load scenarios/i).first()).toBeVisible();
    await runAxeTest(page, 'scenarios-api-down', testInfo);
  });

  test('/constitution page degrades to bundled fallback article set', async (
    { page },
    testInfo,
  ) => {
    const response = await page.goto('/constitution');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /the constitution of kenya/i }),
    ).toBeVisible();
    // The page must surface the fallback amber banner OR an article list —
    // never a blank page.
    const fallbackBanner = page.getByText(/constitution api is unreachable/i);
    const chapterHeading = page.getByText(/chapter \d+/i).first();
    const bannerVisible = await fallbackBanner.count();
    const chapterVisible = await chapterHeading.count();
    expect(bannerVisible + chapterVisible).toBeGreaterThan(0);
    await runAxeTest(page, 'constitution-api-down', testInfo);
  });
});
