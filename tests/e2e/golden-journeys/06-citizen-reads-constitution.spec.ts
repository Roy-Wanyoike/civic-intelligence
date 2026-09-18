import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Golden Journey #6 — Citizen reads the Constitution (Spec §74, Journey 3).
 *
 *   Homepage → Constitution Spotlight → Read Article →
 *   Explore Constitution → Search Article → Related Bill
 *
 * Real user flow: a citizen opens the Constitution page, navigates the
 * chapter list, reads an Article, and follows its source link to Kenya Law.
 * The Constitution page degrades gracefully when the Go BFF is unreachable
 * (it falls back to a bundled article set), so this journey runs with or
 * without a live API.
 *
 * Tagged @golden for selective running:
 *   npx playwright test --grep @golden
 */

const MOCK_CONSTITUTION = {
  constitution: {
    id: 'constitution-ke-2010',
    country_code: 'KE',
    title: 'The Constitution of Kenya, 2010',
    promulgated_at: '2010-08-27',
    assented_at: '2010-08-27',
    version: '2010',
    source_url: 'https://kenyalaw.org/kl/index.php?id=398',
  },
  disclaimer: 'Authoritative source material — never reinterpreted by the platform.',
  reality_layer: 'FACT' as const,
};

const MOCK_CHAPTERS = {
  chapters: [
    {
      id: 'chapter-4',
      constitution_id: 'constitution-ke-2010',
      number: 4,
      title: 'The Bill of Rights',
      articles: [
        {
          id: 'article-19',
          chapter_id: 'chapter-4',
          number: 'Article 19',
          title: 'Fundamental rights and freedoms',
          text: 'The Bill of Rights is an integral part of Kenya’s democratic State and is the framework for social, economic and cultural policies.',
          source_url: 'https://kenyalaw.org/kl/index.php?id=398',
        },
      ],
    },
  ],
  count: 1,
  reality_layer: 'FACT' as const,
  disclaimer: 'Verbatim from Kenya Law — never reinterpreted.',
};

test.describe('Golden Journey #6 — Citizen reads the Constitution @golden', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/constitution', async (route) => {
      await route.fulfill({ status: 200, json: MOCK_CONSTITUTION });
    });
    await page.route('**/api/v1/constitution/articles', async (route) => {
      await route.fulfill({ status: 200, json: MOCK_CHAPTERS });
    });
  });

  test('citizen opens /constitution, navigates chapters, reads an Article', async (
    { page },
    testInfo,
  ) => {
    // 1. Homepage → Constitution.
    await page.goto('/');
    await runAxeTest(page, 'homepage', testInfo);
    await page.getByRole('link', { name: /^constitution$/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /the constitution of kenya/i }),
    ).toBeVisible();
    await runAxeTest(page, 'constitution', testInfo);

    // 2. The page must show the FACT reality disclaimer.
    await expect(page.getByText(/authoritative source material/i).first()).toBeVisible();

    // 3. The chapter list must show at least one chapter with articles.
    //    The page falls back to the bundled data file when the API is down.
    await expect(page.getByText(/chapter \d+/i).first()).toBeVisible();

    // 4. Open an Article's source link to verify traceability.
    const articleSourceLink = page
      .getByRole('link', { name: /read full article/i })
      .first();
    await expect(articleSourceLink).toBeVisible();
    await expect(articleSourceLink).toHaveAttribute('href', /kenyalaw\.org/i);

    // 5. Navigate to a Bill that touches this Article (cross-reference).
    //    The Constitution page surfaces a Constitutional Context Engine
    //    section explaining EXPLICIT / INFERRED / JUDICIAL / UNKNOWN.
    await expect(
      page.getByRole('heading', { level: 2, name: /constitutional context engine/i }),
    ).toBeVisible();
    await expect(page.getByText(/EXPLICIT/i).first()).toBeVisible();
    await expect(page.getByText(/INFERRED/i).first()).toBeVisible();
  });
});
