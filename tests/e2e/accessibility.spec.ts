import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// These tests verify WCAG 2.2 AA compliance using axe-core.
// Run: npx playwright test tests/e2e/accessibility.spec.ts

test('homepage is accessible (WCAG 2.2 AA)', async ({ page }) => {
  await page.goto('/');
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag22aa'])
    .analyze();
  expect(results.violations).toEqual([]);
});

test('bills page is accessible', async ({ page }) => {
  await page.goto('/bills');
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa'])
    .analyze();
  expect(results.violations).toEqual([]);
});

test('about page is accessible', async ({ page }) => {
  await page.goto('/about');
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa'])
    .analyze();
  expect(results.violations).toEqual([]);
});

test('briefing page is accessible', async ({ page }) => {
  await page.goto('/briefing');
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa'])
    .analyze();
  expect(results.violations).toEqual([]);
});
