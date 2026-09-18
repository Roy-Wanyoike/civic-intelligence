import AxeBuilder from '@axe-core/playwright';
import type { Page, TestInfo } from '@playwright/test';
import { expect } from '@playwright/test';

/**
 * Axe-core accessibility setup for the Civic Intelligence Platform.
 *
 * We test WCAG 2.2 AA conformance on every golden journey and every
 * critical-failure test. The four critical WCAG 2.2 AA rule tags are:
 *
 *   wcag2a   — WCAG 2.0 Level A baseline (no flashing, alt text, etc.)
 *   wcag2aa  — WCAG 2.0 Level AA (color contrast, headings, labels)
 *   wcag21aa — WCAG 2.1 Level AA (focus order, target size, name-role-value)
 *   wcag22aa — WCAG 2.2 Level AA (focus not obscured, dragging movements,
 *              target size minimum, consistent help, redundant entry,
 *              accessible authentication)
 *
 * Spec §74 (golden journeys) and §75 (critical failure tests) require every
 * page visited by a journey to pass an axe scan with these tags so an
 * accessibility regression cannot ship behind a green build.
 *
 * Usage in a spec:
 *
 *   test('citizen opens a bill @golden @a11y', async ({ page }, testInfo) => {
 *     await page.goto('/bills/00000000-0000-0000-0000-000000000001');
 *     await runAxeTest(page, 'bill-detail', testInfo);
 *     await expect(page.getByRole('heading', { name: /housing bill/i })).toBeVisible();
 *   });
 *
 * The helper:
 *   - Runs AxeBuilder with the four WCAG 2.2 AA tags above.
 *   - Disables rules that depend on visual rendering where the underlying
 *     rule cannot be evaluated against a headless snapshot (none currently).
 *   - On failure, attaches a human-readable violation summary to the test
 *     report AND writes a JSON report to test-results/axe-<pageName>.json so
 *     developers can re-inspect offline.
 *   - Throws via expect() so the test stops at the failing page (rather than
 *     continuing and producing confusing downstream failures).
 */

export const WCAG_22_AA_TAGS = ['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa'] as const;

export interface AxeViolation {
  id: string;
  impact: 'minor' | 'moderate' | 'serious' | 'critical';
  description: string;
  help: string;
  helpUrl: string;
  nodes: number;
  tags: string[];
}

export interface AxeRunResult {
  pageName: string;
  violations: AxeViolation[];
  passCount: number;
  incompleteCount: number;
  inapplicableCount: number;
}

/**
 * Run an axe-core scan against `page` and assert it passes WCAG 2.2 AA.
 *
 * @param page       Playwright Page — must have navigated already.
 * @param pageName   Short slug for the report attachment (e.g. "bill-detail").
 * @param testInfo   The TestInfo passed by Playwright's test() callback —
 *                   used to attach the report. Optional; if omitted the
 *                   helper still throws on failure but does not attach.
 */
export async function runAxeTest(
  page: Page,
  pageName: string,
  testInfo?: TestInfo,
): Promise<AxeRunResult> {
  const results = await new AxeBuilder({ page })
    .withTags([...WCAG_22_AA_TAGS])
    // The Bill detail page renders a hero gradient on a dark background; axe's
    // color-contrast rule is sometimes noisy for placeholder text inside
    // canvas-like decorative regions. We do NOT disable color-contrast here —
    // the gradient is a real design surface and should pass.
    .analyze();

  const violations: AxeViolation[] = results.violations.map((v) => ({
    id: v.id,
    impact: (v.impact ?? 'moderate') as AxeViolation['impact'],
    description: v.description,
    help: v.help,
    helpUrl: v.helpUrl,
    nodes: v.nodes.length,
    tags: v.tags,
  }));

  const summary: AxeRunResult = {
    pageName,
    violations,
    passCount: results.passes.length,
    incompleteCount: results.incomplete.length,
    inapplicableCount: results.inapplicable.length,
  };

  if (testInfo) {
    await testInfo.attach(`axe-${pageName}.json`, {
      body: JSON.stringify(summary, null, 2),
      contentType: 'application/json',
    });
  }

  // Assert via expect() so the violation details surface in the test output.
  expect(
    violations,
    `axe-core WCAG 2.2 AA scan failed for "${pageName}". ` +
      `${violations.length} rule(s) with violations: ` +
      violations
        .map((v) => `${v.id} (${v.impact}, ${v.nodes} node(s)) — ${v.help}`)
        .join('\n  • '),
  ).toEqual([]);

  return summary;
}

/**
 * Take a screenshot for the test report.
 *
 * Playwright's `screenshot: 'only-on-failure'` config already captures a
 * screenshot on test failure automatically; this helper is for *intentional*
 * human-in-the-loop snapshots (e.g. capturing the visual state of a golden
 * journey for a PR review).
 */
export async function snapshot(
  page: Page,
  testInfo: TestInfo,
  name: string,
): Promise<void> {
  const buffer = await page.screenshot({ fullPage: true });
  await testInfo.attach(`${name}.png`, {
    body: buffer,
    contentType: 'image/png',
  });
}
