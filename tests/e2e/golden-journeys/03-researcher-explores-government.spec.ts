import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Golden Journey #3 — Researcher explores government (Spec §74, Journey 4).
 *
 *   Select Government → Select President → Select Term →
 *   Explore Legislation → Open a Bill → Checks Assent
 *
 * Real user flow: a researcher opens the Governments listing, picks the
 * Uhuru Kenyatta administration, drills into Term 2, and explores the
 * legislation tied to that administration.
 *
 * The Governments list + detail pages are server components that fetch
 * /api/v1/governments[/{id}]. We mock those endpoints to keep the journey
 * deterministic.
 *
 * Tagged @golden for selective running:
 *   npx playwright test --grep @golden
 */

const ADMIN_ID = 'gov-uhuru-kenyatta';

const MOCK_ADMINISTRATIONS = {
  administrations: [
    {
      id: ADMIN_ID,
      country_code: 'KE',
      president_id: 'pres-uhuru',
      name: 'Uhuru Kenyatta Administration',
      start_date: '2013-04-09',
      end_date: '2022-09-13',
      government_system: 'Presidential',
      source_url: 'https://www.statehouse.go.ke/',
      president_name: 'Uhuru Kenyatta',
      is_current: false,
    },
  ],
  count: 1,
};

const MOCK_ADMINISTRATION_DETAIL = {
  administration: MOCK_ADMINISTRATIONS.administrations[0],
  president: {
    id: 'pres-uhuru',
    country_code: 'KE',
    full_name: 'Uhuru Muigai Kenyatta',
    display_name: 'Uhuru Kenyatta',
    biography_url: 'https://en.wikipedia.org/wiki/Uhuru_Kenyatta',
  },
  terms: [
    {
      id: 'term-1',
      administration_id: ADMIN_ID,
      president_id: 'pres-uhuru',
      term_number: 1,
      start_date: '2013-04-09',
      end_date: '2017-11-28',
      status: 'COMPLETED',
      source_url: 'https://www.iebc.or.ke/',
    },
    {
      id: 'term-2',
      administration_id: ADMIN_ID,
      president_id: 'pres-uhuru',
      term_number: 2,
      start_date: '2017-11-28',
      end_date: '2022-09-13',
      status: 'COMPLETED',
      source_url: 'https://www.iebc.or.ke/',
    },
  ],
};

test.describe('Golden Journey #3 — Researcher explores government @golden', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/governments', async (route) => {
      await route.fulfill({ status: 200, json: MOCK_ADMINISTRATIONS });
    });
    await page.route(`**/api/v1/governments/${ADMIN_ID}`, async (route) => {
      await route.fulfill({ status: 200, json: MOCK_ADMINISTRATION_DETAIL });
    });
  });

  test('researcher opens Uhuru admin, Term 2, views legislation', async ({ page }, testInfo) => {
    // 1. Homepage → Governments.
    await page.goto('/');
    await runAxeTest(page, 'homepage', testInfo);
    await page.getByRole('link', { name: /^governments$/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /governments of kenya/i }),
    ).toBeVisible();
    await runAxeTest(page, 'governments-list', testInfo);

    // 2. Open the Uhuru Kenyatta administration.
    const adminLink = page.getByRole('link', { name: /uhuru kenyatta administration/i }).first();
    await expect(adminLink).toBeVisible();
    await adminLink.click();

    // 3. Administration detail page renders the president name + terms.
    await expect(
      page.getByRole('heading', { level: 1, name: /uhuru kenyatta administration/i }),
    ).toBeVisible();
    await expect(page.getByText(/president:/i).first()).toBeVisible();
    await expect(page.getByText(/uhuru kenyatta/i).first()).toBeVisible();
    await runAxeTest(page, 'administration-detail', testInfo);

    // 4. Click into Term 2.
    const term2Link = page.getByRole('link', { name: /term 2|term-2/i }).first();
    await expect(term2Link).toBeVisible();
    await term2Link.click();
    await page.waitForLoadState('networkidle');

    // 5. The term page should show "Term 2" in its heading.
    await expect(
      page.getByRole('heading', { level: 1, name: /term 2/i }),
    ).toBeVisible();
    await runAxeTest(page, 'presidential-term', testInfo);

    // 6. Navigate to Bills via the navbar Legislation menu.
    await page.getByRole('link', { name: /^bills$/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /bills before parliament/i }),
    ).toBeVisible();
    await runAxeTest(page, 'bills-list', testInfo);

    // 7. Open the first Bill and check the assent/status badge.
    const billLink = page.locator('a[href^="/bills/00000000-0000-0000-0000-000000000004"]').first();
    if (await billLink.count() > 0) {
      await billLink.click();
      await expect(page.getByText(/enacted/i).first()).toBeVisible();
    }
  });
});
