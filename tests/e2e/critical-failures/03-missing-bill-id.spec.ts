import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Critical Failure #3 — Missing Bill id (Spec §75 variant).
 *
 *   Navigate to /bills/nonexistent-id → 404 page renders
 *
 * The Bill detail page (apps/web/src/app/bills/[id]/page.tsx) calls
 * notFound() when the requested id isn't in mockBills (or in the API
 * result if force-dynamic). The notFound() helper renders the
 * not-found.tsx page with a 404 status. This test verifies:
 *
 *   - The 404 page renders (not a blank page or a 500 server error)
 *   - The page contains a clear "Page not found" heading
 *   - A return-home link is provided so the user can recover
 *   - axe-core still passes on the 404 state
 *
 * Tagged @critical for selective running:
 *   npx playwright test --grep @critical
 */
test.describe('Critical Failure #3 — Missing Bill id @critical', () => {
  test('/bills/<nonexistent-id> renders the 404 page', async ({ page }, testInfo) => {
    const response = await page.goto('/bills/this-bill-id-does-not-exist');
    expect(response?.status()).toBe(404);

    await expect(page.getByText('404', { exact: true })).toBeVisible();
    await expect(
      page.getByRole('heading', { level: 1, name: /page not found/i }),
    ).toBeVisible();
    await expect(
      page.getByText(/does not exist or has been moved/i),
    ).toBeVisible();

    // Recovery path — a "Return home" link must be present.
    const homeLink = page.getByRole('link', { name: /return home/i });
    await expect(homeLink).toBeVisible();

    await runAxeTest(page, 'bill-404', testInfo);
  });

  test('/scenarios/<nonexistent-id> renders the 404 page', async ({ page }, testInfo) => {
    // Same contract — a nonexistent scenario id should 404, not 500.
    // We mock the API to return 404 so the server component calls notFound().
    await page.route('**/api/v1/scenarios/nonexistent-scenario-id', (route) =>
      route.fulfill({ status: 404, json: { error: 'not found' } }),
    );

    const response = await page.goto('/scenarios/nonexistent-scenario-id');
    expect(response?.status()).toBe(404);
    await expect(
      page.getByRole('heading', { level: 1, name: /page not found/i }),
    ).toBeVisible();
    await runAxeTest(page, 'scenario-404', testInfo);
  });

  test('404 recovery returns to the homepage', async ({ page }, testInfo) => {
    await page.goto('/bills/does-not-exist');
    await expect(page.getByText('404', { exact: true })).toBeVisible();
    await page.getByRole('link', { name: /return home/i }).click();
    await page.waitForLoadState('networkidle');
    await expect(
      page.getByRole('heading', { level: 1, name: /understand what your government is doing/i }),
    ).toBeVisible();
  });
});
