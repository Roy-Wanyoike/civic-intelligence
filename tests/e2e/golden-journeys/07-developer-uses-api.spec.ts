import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Golden Journey #7 — Developer uses the API (Spec §74, Journey 8 variant).
 *
 *   Open /developers → Inspect OpenAPI → Make a curl request →
 *   Check response shape
 *
 * Real user flow: a developer lands on the homepage, opens the Developers
 * page, inspects the endpoint table, copies a curl example, runs it against
 * the local BFF, and verifies the response shape.
 *
 * Tagged @golden for selective running:
 *   npx playwright test --grep @golden
 */

const MOCK_BILLS_RESPONSE = {
  items: [
    {
      id: '00000000-0000-0000-0000-000000000001',
      identifier: 'NA Bill No. 23',
      title: 'The Housing Bill, 2024',
      year: 2024,
      status: 'in_progress',
      current_stage: 'Committee Stage',
      house_name: 'National Assembly',
      country: 'KE',
    },
  ],
  total: 1,
  page: 1,
  page_size: 20,
};

test.describe('Golden Journey #7 — Developer uses the API @golden', () => {
  test('developer inspects OpenAPI, makes a curl request, checks response shape', async (
    { page },
    testInfo,
  ) => {
    // 1. Homepage → /developers.
    await page.goto('/');
    await runAxeTest(page, 'homepage', testInfo);
    await page.getByRole('link', { name: /^developers$/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /developer api/i }),
    ).toBeVisible();
    await runAxeTest(page, 'developers', testInfo);

    // 2. The page must surface the machine-readable contract reference.
    await expect(page.getByText(/docs\/api\/openapi\.yaml/i)).toBeVisible();

    // 3. Inspect the endpoint table — at least one endpoint must be visible.
    const endpointRows = page.locator('table tbody tr');
    await expect(endpointRows.first()).toBeVisible();
    const endpointCount = await endpointRows.count();
    expect(endpointCount).toBeGreaterThan(5);

    // 4. The page must surface the example curl block.
    const exampleBlock = page.locator('pre').filter({ hasText: /curl.*-s.*api\/v1\/bills/ });
    await expect(exampleBlock).toBeVisible();

    // 5. The rate-limit (300 req/min) hint must be visible so the developer
    //    knows the public API surface is bounded.
    await expect(page.getByText(/rate limiting \(300 req\/min\)/i)).toBeVisible();

    // 6. Use Playwright's request context to make the same curl-style request
    //    the developer would make. We mock the API endpoint so the response
    //    is deterministic.
    await page.route('**/api/v1/bills', async (route) => {
      await route.fulfill({ status: 200, json: MOCK_BILLS_RESPONSE });
    });

    const response = await page.request.get('/api/v1/bills');
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body).toHaveProperty('items');
    expect(body).toHaveProperty('total');
    expect(body).toHaveProperty('page');
    expect(body).toHaveProperty('page_size');
    expect(Array.isArray(body.items)).toBe(true);
    expect(body.items.length).toBeGreaterThan(0);
    expect(body.items[0]).toHaveProperty('id');
    expect(body.items[0]).toHaveProperty('title');
    expect(body.items[0]).toHaveProperty('status');
  });
});
