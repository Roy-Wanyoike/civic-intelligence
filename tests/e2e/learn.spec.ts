import { test, expect } from '@playwright/test';

/**
 * Civic Education Hub — /learn route tests.
 *
 * Covers:
 *   - the landing page at /learn (5 guide cards, hero, CTA),
 *   - the per-guide page at /learn/<slug> (TOC, sections, sources,
 *     related links, feedback widget, breadcrumb),
 *   - SEO: each guide page emits a canonical link + OpenGraph article tags,
 *   - the navbar mega-menu exposes a "Learn" entry under "Resources".
 *
 * These tests run against the same `next start` server as the rest of the
 * Playwright suite — no special fixtures, no API mocking. The guides are
 * authored content shipped with the app, so they are available
 * deterministically at build time.
 */

const GUIDE_SLUGS = [
  'how-a-bill-becomes-law',
  'understanding-parliament',
  'how-to-read-a-bill',
  'your-constitutional-rights',
  'public-participation',
] as const;

test.describe('Civic Education Hub — /learn', () => {
  test('landing page renders all 5 guide cards', async ({ page }) => {
    await page.goto('/learn');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('h1')).toContainText('Parliament works');
    // 5 guide cards, each linking to /learn/<slug>.
    const guideLinks = page.locator('a[href^="/learn/"]');
    await expect(guideLinks).toHaveCount(GUIDE_SLUGS.length);
    for (const slug of GUIDE_SLUGS) {
      await expect(page.locator(`a[href="/learn/${slug}"]`)).toBeVisible();
    }
  });

  test('landing page breadcrumb links back home', async ({ page }) => {
    await page.goto('/learn');
    await page.waitForLoadState('networkidle');
    const breadcrumb = page.locator('nav[aria-label="Breadcrumb"]');
    await expect(breadcrumb).toBeVisible();
    await expect(breadcrumb.locator('a[href="/"]')).toBeVisible();
  });

  test('landing page exposes a canonical URL for SEO', async ({ page }) => {
    await page.goto('/learn');
    await page.waitForLoadState('networkidle');
    const canonical = page.locator('link[rel="canonical"]');
    await expect(canonical).toHaveAttribute('href', /\/learn$/);
  });

  test('navbar mega-menu exposes Learn under Resources', async ({ page }) => {
    await page.goto('/');
    // Wait for React hydration so the Explore button's onClick is wired
    // before we click — otherwise the click lands on a button with no
    // listener attached yet.
    await page.waitForLoadState('networkidle');
    const exploreButton = page.getByRole('button', { name: 'Explore' });
    await exploreButton.click();
    // The mega-menu is rendered with role="menu" once exploreOpen flips.
    const menu = page.locator('[role="menu"]');
    await expect(menu).toBeVisible();
    // The mega-menu should now contain a "Learn" link under the Resources column.
    const learnLink = page.locator('a[href="/learn"]').first();
    await expect(learnLink).toBeVisible();
    await expect(learnLink).toContainText('Learn');
  });

  for (const slug of GUIDE_SLUGS) {
    test(`guide ${slug} renders with TOC + sources + feedback`, async ({ page }) => {
      await page.goto(`/learn/${slug}`);
      await page.waitForLoadState('networkidle');
      // Title + breadcrumb.
      await expect(page.locator('h1')).toBeVisible();
      const breadcrumb = page.locator('nav[aria-label="Breadcrumb"]');
      await expect(breadcrumb).toContainText('Learn');
      await expect(breadcrumb.locator('a[href="/learn"]')).toBeVisible();
      // TOC sidebar.
      const toc = page.locator('nav[aria-label="Table of contents"]');
      await expect(toc).toBeVisible();
      await expect(toc.locator('a[href^="#"]').first()).toBeVisible();
      // Sources section.
      await expect(page.getByRole('heading', { name: 'Sources' })).toBeVisible();
      // At least one external source link (target=_blank).
      const sourceLinks = page.locator('section[aria-labelledby="sources-heading"] a[target="_blank"]');
      const sourceCount = await sourceLinks.count();
      expect(sourceCount).toBeGreaterThan(0);
      // Related links section.
      await expect(page.getByRole('heading', { name: 'Related on the platform' })).toBeVisible();
      // Feedback widget.
      await expect(page.getByRole('heading', { name: 'Was this helpful?' })).toBeVisible();
      // The "Yes" / "No" buttons must be present.
      await expect(page.getByRole('button', { name: /^Yes$/ })).toBeVisible();
      await expect(page.getByRole('button', { name: /^No$/ })).toBeVisible();
    });

    test(`guide ${slug} emits article OpenGraph + canonical URL`, async ({ page }) => {
      await page.goto(`/learn/${slug}`);
      const canonical = page.locator('link[rel="canonical"]');
      await expect(canonical).toHaveAttribute('href', new RegExp(`/learn/${slug}$`));
      const ogType = page.locator('meta[property="og:type"][content="article"]');
      await expect(ogType).toHaveCount(1);
    });

    test(`guide ${slug} feedback widget toggles to thank-you on click`, async ({ page }) => {
      await page.goto(`/learn/${slug}`);
      // The feedback widget is a client component — wait for hydration
      // so the Yes button's onClick handler is attached before we click.
      await page.waitForLoadState('networkidle');
      const yesButton = page.getByRole('button', { name: /^Yes$/ });
      await expect(yesButton).toBeVisible();
      await yesButton.click();
      // The "Thanks for the feedback" status message should appear.
      await expect(page.getByText('Thanks for the feedback')).toBeVisible();
    });
  }

  test('how-a-bill-becomes-law guide shows 8 stage badges', async ({ page }) => {
    await page.goto('/learn/how-a-bill-becomes-law');
    // Stages 1 through 8 each render a badge with text "Stage N".
    for (let i = 1; i <= 8; i++) {
      await expect(page.getByText(`Stage ${i}`, { exact: false }).first()).toBeVisible();
    }
  });

  test('unknown guide slug returns 404', async ({ page }) => {
    const response = await page.goto('/learn/this-slug-does-not-exist');
    expect(response?.status()).toBe(404);
  });
});
