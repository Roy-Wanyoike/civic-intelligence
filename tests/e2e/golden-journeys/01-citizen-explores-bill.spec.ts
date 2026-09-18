import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Golden Journey #1 — Citizen discovers a Bill (Spec §74, Journey 1).
 *
 *   Open Homepage → Search Bill → Open Bill → Understand Summary →
 *   Inspect Timeline → Inspect Evidence
 *
 * Real user flow: a Kenyan citizen lands on the homepage, uses the Bills
 * search filter, opens a Bill, reads its plain-language summary, follows
 * the timeline link to inspect verified events, and finally inspects the
 * Documents sidebar to see version + citation counts.
 *
 * Tagged @golden for selective running:
 *   npx playwright test --grep @golden
 */
test.describe('Golden Journey #1 — Citizen discovers a Bill @golden', () => {
  test('citizen searches, opens, reads summary, inspects timeline + evidence', async (
    { page },
    testInfo,
  ) => {
    // 1. Open homepage. Hero headline is hardcoded so it always renders.
    await page.goto('/');
    await expect(
      page.getByRole('heading', { level: 1, name: /understand what your government is doing/i }),
    ).toBeVisible();
    await runAxeTest(page, 'homepage', testInfo);

    // 2. Navigate to the Bills listing via the homepage's "All Bills" link.
    await page.getByRole('link', { name: /all bills/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /bills before parliament/i }),
    ).toBeVisible();
    await runAxeTest(page, 'bills-list', testInfo);

    // 3. Search the Bills listing for "Housing" — the search input filters the
    //    list in-page. Submitting the form navigates with ?q=housing.
    const searchInput = page.locator('form[aria-label="Filter Bills"] input[name="q"]');
    await searchInput.fill('Housing');
    await searchInput.press('Enter');
    await page.waitForLoadState('networkidle');

    // 4. Open the first matching Bill. The Housing Bill is in mockBills with
    //    id 00000000-0000-0000-0000-000000000001.
    const firstBillLink = page.getByRole('link', { name: /the housing bill, 2024/i }).first();
    await expect(firstBillLink).toBeVisible();
    await firstBillLink.click();

    // 5. Bill detail page — assert the plain-language summary block rendered.
    await expect(
      page.getByRole('heading', { level: 1, name: /the housing bill, 2024/i }),
    ).toBeVisible();
    await expect(
      page.getByRole('heading', { level: 2, name: /in plain language/i }),
    ).toBeVisible();
    await runAxeTest(page, 'bill-detail', testInfo);

    // 6. Inspect the Timeline section. The Bill detail page renders a preview
    //    Timeline section with a "Full timeline →" link.
    await expect(page.getByRole('heading', { level: 2, name: /^timeline$/i })).toBeVisible();
    await page.getByRole('link', { name: /full timeline/i }).click();
    await page.waitForLoadState('networkidle');

    // 7. On the timeline page, at least one timeline event should be visible
    //    (mock data has events for the Housing Bill).
    await expect(
      page.getByRole('heading', { level: 1, name: /timeline.*housing bill/i }),
    ).toBeVisible();
    await runAxeTest(page, 'bill-timeline', testInfo);

    // 8. Inspect evidence — the Documents sidebar on the Bill detail page
    //    shows version_count + citation_count. Navigate back to the detail
    //    page and inspect the sidebar.
    await page.goBack();
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { level: 3, name: /documents/i })).toBeVisible();
    await expect(page.getByText(/version\(s\)/)).toBeVisible();
    await expect(page.getByText(/citation\(s\)/)).toBeVisible();
  });
});
