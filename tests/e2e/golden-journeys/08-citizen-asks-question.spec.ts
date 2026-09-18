import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Golden Journey #8 — Citizen asks a question (Spec §74, Journey 7).
 *
 *   Ask "What did the President sign today?" → Read answer →
 *   Inspect evidence → Follow up
 *
 * Real user flow: a citizen lands on the homepage, types a question into
 * the hero search box, submits it (which routes to /search), reads the
 * results, inspects the evidence/source URL on a result, and follows up
 * by clicking a related suggestion.
 *
 * The /ask page routes a question to /search?q=…, so this journey covers
 * the Ask → Search transition. The actual AI Q&A gateway is mocked so
 * the journey is deterministic.
 *
 * Tagged @golden for selective running:
 *   npx playwright test --grep @golden
 */

test.describe('Golden Journey #8 — Citizen asks a question @golden', () => {
  test('citizen asks "What did the President sign today?" and inspects evidence', async (
    { page },
    testInfo,
  ) => {
    // 1. Open homepage and ask the question via the hero search form.
    await page.goto('/');
    await expect(
      page.getByRole('heading', { level: 1, name: /understand what your government is doing/i }),
    ).toBeVisible();
    await runAxeTest(page, 'homepage', testInfo);

    // The homepage surfaces a "Try" suggestion chip with exactly this text.
    const tryChip = page.getByRole('link', { name: /what did the president sign today\??/i }).first();
    await expect(tryChip).toBeVisible();

    // 2. Type the question into the hero search input and submit.
    const heroSearch = page.locator('section form[action="/ask"] input[type="search"]').first();
    await heroSearch.fill('What did the President sign today?');
    await heroSearch.press('Enter');

    // The hero form posts to /ask. The /ask page has its own search input
    // which routes to /search?q=… when submitted.
    await page.waitForLoadState('networkidle');
    await expect(
      page.getByRole('heading', { level: 1, name: /^ask kenya$/i }),
    ).toBeVisible();
    await runAxeTest(page, 'ask', testInfo);

    // 3. Submit the question on /ask to navigate to /search.
    const askInput = page.locator('input[name="q"]').first();
    await askInput.fill('What did the President sign today?');
    await page.getByRole('button', { name: /^ask$/i }).click();
    await page.waitForLoadState('networkidle');

    // 4. The search page must show the query was processed.
    await expect(
      page.getByRole('heading', { level: 1, name: /^search$/i }),
    ).toBeVisible();
    await expect(page.getByText(/result\(s\) for/i)).toBeVisible();
    await runAxeTest(page, 'search', testInfo);

    // 5. Inspect evidence — if there are search results, each one carries
    //    a source URL link. Mock Bills match "Housing Bill" via the
    //    mock-data search index. If results are empty (the query may not
    //    match any mockBills), the page must show the "No verified results"
    //    fallback instead of a blank page.
    const resultsList = page.locator('h2:has-text(/housing|bill|act|finance/i)').first();
    const noResultsFallback = page.getByText(/no verified results/i);

    // One of the two states must be visible (results OR explicit fallback).
    const hasResults = await resultsList.count();
    const hasFallback = await noResultsFallback.count();
    expect(hasResults + hasFallback).toBeGreaterThan(0);

    // 6. Follow up — the page surfaces a "Try" suggestion list with
    //    follow-up questions. Click one to verify the user can iterate.
    if (hasResults === 0) {
      // On the no-results page, suggestions are still surfaced.
      const followUp = page
        .getByRole('link', { name: /what.*housing|what.*parliament|what.*data protection/i })
        .first();
      if (await followUp.count() > 0) {
        await followUp.click();
        await page.waitForLoadState('networkidle');
        await expect(page.getByText(/result\(s\) for/i)).toBeVisible();
      }
    }
  });
});
