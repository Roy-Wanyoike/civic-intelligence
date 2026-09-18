import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Golden Journey #2 — Citizen follows an Act (Spec §74, Journey 2 variant).
 *
 *   Open Act → What happened after assent? → Audit the Act → Follow the law
 *
 * Real user flow: a citizen lands on the homepage, opens the Acts list,
 * selects an Act, reads the "What happened after Presidential Assent?"
 * section, navigates into the Audit page, and uses the Follow-the-Law button.
 *
 * The Act detail page is a server component that fetches from
 * /api/v1/acts/{id}. To make the journey deterministic (not dependent on a
 * live BFF), we intercept the Acts list and detail routes with realistic
 * mock payloads.
 *
 * Tagged @golden for selective running:
 *   npx playwright test --grep @golden
 */

const ACT_ID = 'act-public-finance-management-2012';

const MOCK_ACT = {
  id: ACT_ID,
  title: 'The Public Finance Management Act, 2012',
  citation: 'No. 18 of 2012',
  assent_date: '2012-07-23',
  commencement_date: '2012-08-01',
  source_url: 'https://kenyalaw.org/kl/index.php?id=5315',
  status: 'in_force',
  country: 'KE',
  summary:
    'Provides for the management of public finances, including the Consolidated Fund, the budget process, and oversight of public debt.',
};

const MOCK_ACTS_LIST = {
  items: [MOCK_ACT],
  total: 1,
};

const MOCK_AUDIT = {
  act_id: ACT_ID,
  assent: { status: 'VERIFIED', date: '2012-07-23', source_url: MOCK_ACT.source_url },
  publication: { status: 'VERIFIED', date: '2012-07-30', source_url: MOCK_ACT.source_url },
  commencement: { status: 'VERIFIED', date: '2012-08-01', source_url: MOCK_ACT.source_url },
  regulations: { status: 'NOT_VERIFIED', count: 0 },
  amendments: { status: 'VERIFIED', count: 4 },
  court_decisions: { status: 'NOT_VERIFIED', count: 0 },
  current_status: 'in_force',
  gaps: ['regulations', 'court_decisions'],
};

const MOCK_LINEAGE = {
  act_id: ACT_ID,
  stages: [
    { stage: 'Bill', identifier: 'NA Bill No. 18 of 2012', source_url: MOCK_ACT.source_url },
    { stage: 'Bill Version', identifier: 'v1', source_url: MOCK_ACT.source_url },
    { stage: 'Assent', date: '2012-07-23', source_url: MOCK_ACT.source_url },
    { stage: 'Act', identifier: MOCK_ACT.citation, source_url: MOCK_ACT.source_url },
  ],
};

test.describe('Golden Journey #2 — Citizen follows an Act @golden', () => {
  test.beforeEach(async ({ page }) => {
    // Intercept every Act-related API call so the journey is deterministic
    // regardless of whether the Go BFF is running.
    await page.route('**/api/v1/acts', async (route) => {
      await route.fulfill({ status: 200, json: MOCK_ACTS_LIST });
    });
    await page.route('**/api/v1/acts/*', async (route) => {
      const url = route.request().url();
      if (url.endsWith(`/audit`)) {
        await route.fulfill({ status: 200, json: MOCK_AUDIT });
        return;
      }
      if (url.endsWith(`/lineage`)) {
        await route.fulfill({ status: 200, json: MOCK_LINEAGE });
        return;
      }
      if (url.endsWith(`/events`)) {
        await route.fulfill({ status: 200, json: { items: [], total: 0 } });
        return;
      }
      await route.fulfill({ status: 200, json: MOCK_ACT });
    });
  });

  test('citizen opens an Act, audits it, follows the law', async ({ page }, testInfo) => {
    // 1. Homepage → navigate to Acts.
    await page.goto('/');
    await runAxeTest(page, 'homepage', testInfo);
    // The navbar mega-menu exposes an "Acts" entry under "Legislation".
    await page.getByRole('link', { name: /^Acts$/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /acts of parliament/i }),
    ).toBeVisible();
    await runAxeTest(page, 'acts-list', testInfo);

    // 2. Click the (mocked) Act in the listing.
    const actLink = page.getByRole('link', { name: /public finance management act, 2012/i }).first();
    await expect(actLink).toBeVisible();
    await actLink.click();

    // 3. The Act detail page renders its header + the post-assent section.
    await expect(
      page.getByRole('heading', { level: 1, name: /public finance management act, 2012/i }),
    ).toBeVisible();
    await expect(
      page.getByRole('heading', { level: 2, name: /what happened after presidential assent\?/i }),
    ).toBeVisible();
    await runAxeTest(page, 'act-detail', testInfo);

    // 4. Navigate into the Audit page.
    await page.getByRole('link', { name: /audit an act/i }).click();
    await page.waitForLoadState('networkidle');
    await runAxeTest(page, 'act-audit', testInfo);

    // 5. The Follow-a-Law button should be visible on the Act detail page.
    await page.goBack();
    await page.waitForLoadState('networkidle');
    const followButton = page.getByRole('button', { name: /follow this law/i });
    await expect(followButton).toBeVisible();
  });
});
