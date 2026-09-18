import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Critical Failure #2 — Malformed response (Spec §75).
 *
 *   API returns malformed JSON → Frontend handles gracefully
 *
 * The frontend uses a small typed fetch client (apps/web/src/lib/api.ts)
 * that calls resp.json() inside a try/catch. Pages that consume that
 * client must surface a visible error banner rather than crashing.
 *
 * We simulate the API returning an HTTP 200 with a non-JSON body (e.g.
 * the upstream Kenyan source's HTML error page got passed through) and
 * assert:
 *
 *   - The page returns 200 (no 500 server error)
 *   - A visible error banner is rendered
 *   - The homepage hero is unaffected
 *   - axe-core still passes on the error state
 *
 * Tagged @critical for selective running:
 *   npx playwright test --grep @critical
 */
test.describe('Critical Failure #2 — Malformed JSON response @critical', () => {
  test.beforeEach(async ({ page }) => {
    // Serve a malformed (HTML instead of JSON) body with a 200 status. This
    // exercises the resp.json() rejection path in apps/web/src/lib/api.ts.
    await page.route('**/api/v1/debt', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/html',
        body: '<html><body>upstream gateway error — not JSON</body></html>',
      });
    });
    await page.route('**/api/v1/debt/timeline', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/html',
        body: '<html><body>upstream gateway error — not JSON</body></html>',
      });
    });
    await page.route('**/api/v1/scenarios', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/plain',
        body: '503 service unavailable — plain text, not JSON',
      });
    });
    await page.route('**/api/v1/constitution', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/html',
        body: '<html><body>500 from upstream</body></html>',
      });
    });
    await page.route('**/api/v1/constitution/articles', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/html',
        body: '<html><body>500 from upstream</body></html>',
      });
    });
  });

  test('/debt page handles malformed JSON gracefully', async ({ page }, testInfo) => {
    const response = await page.goto('/debt');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /kenya public debt/i }),
    ).toBeVisible();
    await expect(page.getByText(/failed to load/i).first()).toBeVisible();
    await runAxeTest(page, 'debt-malformed', testInfo);
  });

  test('/scenarios page handles malformed JSON gracefully', async ({ page }, testInfo) => {
    const response = await page.goto('/scenarios');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /^what if\??$/i }),
    ).toBeVisible();
    await expect(page.getByText(/failed to load scenarios/i).first()).toBeVisible();
    await runAxeTest(page, 'scenarios-malformed', testInfo);
  });

  test('/constitution page degrades to bundled fallback when API returns HTML', async (
    { page },
    testInfo,
  ) => {
    const response = await page.goto('/constitution');
    expect(response?.status()).toBe(200);
    await expect(
      page.getByRole('heading', { level: 1, name: /the constitution of kenya/i }),
    ).toBeVisible();
    // Either the fallback banner OR an article list is visible — never blank.
    const fallback = page.getByText(/constitution api is unreachable/i);
    const chapters = page.getByText(/chapter \d+/i).first();
    expect((await fallback.count()) + (await chapters.count())).toBeGreaterThan(0);
    await runAxeTest(page, 'constitution-malformed', testInfo);
  });
});
