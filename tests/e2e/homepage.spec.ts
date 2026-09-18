import { test, expect } from '@playwright/test';

test('homepage loads with hero text', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('h1')).toContainText('Understand what your government is doing');
});

test('homepage has search input', async ({ page }) => {
  await page.goto('/');
  const search = page.locator('input[type="search"]');
  await expect(search).toBeVisible();
});

test('bills page loads', async ({ page }) => {
  await page.goto('/bills');
  await expect(page.locator('h1')).toContainText('Bills before Parliament');
});

test('about page loads', async ({ page }) => {
  await page.goto('/about');
  await expect(page.locator('h1')).toContainText('About');
});

test('briefing page loads', async ({ page }) => {
  await page.goto('/briefing');
  await expect(page.locator('h1')).toBeVisible();
});

test('loans page loads', async ({ page }) => {
  await page.goto('/loans');
  await expect(page.locator('h1')).toContainText('Loans');
});

test('grants page loads', async ({ page }) => {
  await page.goto('/grants');
  await expect(page.locator('h1')).toContainText('Grants');
});
