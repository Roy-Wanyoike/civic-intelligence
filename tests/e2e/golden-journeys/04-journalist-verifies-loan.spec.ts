import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Golden Journey #4 — Journalist verifies a loan (Spec §74, Journey 6).
 *
 *   Open Debt → Select Government → Select Presidential Term →
 *   Select Legislature → Inspect Graph → Open Loan → Inspect Evidence
 *
 * Real user flow: a journalist opens the /debt dashboard, inspects the
 * debt trend graph, navigates to the /loans register, finds a loan,
 * verifies its source URL, and checks the attribution disclaimer.
 *
 * Tagged @golden for selective running:
 *   npx playwright test --grep @golden
 */

const MOCK_DEBT_DASHBOARD = {
  country_code: 'KE',
  total_debt_stock: 11_000_000_000_000,
  domestic_debt: 5_500_000_000_000,
  external_debt: 5_500_000_000_000,
  debt_service: 1_400_000_000_000,
  debt_to_gdp: 70.2,
  currency: 'KES',
  as_of: '2024-09-30',
  source_url: 'https://www.centralbank.go.ke/monthly-economic-indicators/',
  disclaimer: 'The platform does not rank governments or attribute sovereign borrowing personally to a president.',
};

const MOCK_DEBT_TIMELINE = {
  country_code: 'KE',
  points: [
    { date: '2023-06-30', total_debt_stock: 10_500_000_000_000, source_url: 'https://www.centralbank.go.ke/' },
    { date: '2024-06-30', total_debt_stock: 11_000_000_000_000, source_url: 'https://www.centralbank.go.ke/' },
  ],
  disclaimer: MOCK_DEBT_DASHBOARD.disclaimer,
};

test.describe('Golden Journey #4 — Journalist verifies a loan @golden', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/debt', async (route) => {
      await route.fulfill({ status: 200, json: MOCK_DEBT_DASHBOARD });
    });
    await page.route('**/api/v1/debt/timeline', async (route) => {
      await route.fulfill({ status: 200, json: MOCK_DEBT_TIMELINE });
    });
  });

  test('journalist opens /debt, finds a loan, verifies source + attribution', async (
    { page },
    testInfo,
  ) => {
    // 1. Homepage → /debt.
    await page.goto('/');
    await runAxeTest(page, 'homepage', testInfo);
    await page.getByRole('link', { name: /^public debt$/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /kenya public debt/i }),
    ).toBeVisible();
    await runAxeTest(page, 'debt-dashboard', testInfo);

    // 2. The dashboard must surface the explicit no-political-attribution
    //    disclaimer required by spec §46.
    await expect(
      page.getByText(/does not.*rank governments.*or.*attribute.*borrowing/i),
    ).toBeVisible();

    // 3. The debt trend chart section is rendered with a Source link.
    await expect(
      page.getByRole('heading', { level: 2, name: /public debt stock over time/i }),
    ).toBeVisible();
    await expect(page.getByRole('link', { name: /^source$/i }).first()).toBeVisible();

    // 4. Navigate to the Loans register.
    await page.getByRole('link', { name: /^loans$/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /government loans since september 2022/i }),
    ).toBeVisible();
    await runAxeTest(page, 'loans-register', testInfo);

    // 5. Find the first loan and verify it has a source URL link.
    const firstSourceLink = page
      .locator('a[href^="https://"]')
      .filter({ hasText: /imf|world bank|afdb|china|tdb|eib|capital markets/i })
      .first();
    await expect(firstSourceLink).toBeVisible();

    // 6. Attribution warning — every loan must link to a credible source URL.
    //    Confirm at least one source link is present.
    const sourceLinks = page.locator('ul li a[href^="https://"]');
    await expect(sourceLinks.first()).toBeVisible();
  });
});
