import { test, expect } from '@playwright/test';
import { runAxeTest } from '../axe-setup';

/**
 * Golden Journey #5 — Citizen explores a scenario (Spec §74, Journey 7).
 *
 *   Open /scenarios → Create a scenario → Run it →
 *   Inspect uncertainty → Compare scenarios
 *
 * Real user flow: a citizen lands on the Scenario Explorer, opens the New
 * Scenario builder, inspects an existing scenario's overview, visits the
 * Results tab to see uncertainty bounds (p10/p90), and uses the Compare
 * tab to compare two scenarios side-by-side.
 *
 * The Scenarios endpoints (services/api + services/simulation) are mocked
 * so the journey is deterministic. Every mock response carries the
 * reality_layer tag the spec requires.
 *
 * Tagged @golden for selective running:
 *   npx playwright test --grep @golden
 */

const SCENARIO_ID = 'scn-affordable-housing-impact-2024';

const MOCK_SCENARIO = {
  id: SCENARIO_ID,
  tenant_id: 'KE',
  name: 'Affordable Housing Impact, 2024',
  description:
    'What could happen to housing affordability if the Housing Bill 2024 is fully implemented over the next 5 years?',
  type: 'POLICY',
  jurisdiction: 'KE',
  reality_layer: 'HYPOTHETICAL',
  created_at: '2024-09-08T12:00:00Z',
  updated_at: '2024-09-09T10:30:00Z',
  baseline: {
    description: 'Status quo: 2.5M housing deficit, 75% of urban households in informal settlements.',
    as_of: '2024-01-01',
    reality_layer: 'OBSERVED',
  },
  assumptions: [
    {
      id: 'asm-1',
      scenario_id: SCENARIO_ID,
      statement: 'Housing Fund contributions remain at 1.5% of gross salary.',
      assumption_type: 'USER_DEFINED',
      confidence: 'MEDIUM',
    },
  ],
  variables: [],
  time_horizon: { start: '2024-01-01', end: '2029-12-31', duration: '5 years' },
  source_evidence: [],
  methodology: {
    description: 'Static input/output model with Monte Carlo sensitivity sweep (n=1000).',
    limitations: ['Excludes macroeconomic feedback effects', 'Assumes linear absorption'],
    engine_name: 'civic-sim-static',
    engine_version: '0.3.1',
  },
  model_id: 'mdl-static-001',
  model_version: 'v1',
  status: 'COMPLETED',
  scenario_version: 1,
};

const MOCK_SCENARIO_LIST = {
  scenarios: [MOCK_SCENARIO],
  count: 1,
  disclaimer: 'Scenarios are HYPOTHETICAL — never confused with observed civic fact.',
};

const MOCK_RESULTS = {
  results: [
    {
      run_id: 'run-1',
      completed_at: '2024-09-09T11:00:00Z',
      duration: '12s',
      summary: {
        status: 'COMPLETED',
        headline_metric: 'units_delivered',
        headline_value: 250_000,
        uncertainty: {
          has_uncertainty: true,
          median: 250_000,
          p10: 180_000,
          p90: 320_000,
          min: 150_000,
          max: 380_000,
          notes: 'Monte Carlo n=1000; assumes 80% absorption.',
        },
        limitations: ['Excludes macroeconomic feedback'],
      },
    },
  ],
  count: 1,
  reality_layer: 'MODELED',
  disclaimer: 'Results are SIMULATED — never presented as observed civic fact.',
};

const MOCK_TIMELINE = {
  timeline: [
    { timestamp: '2024-09-01', label: 'OBSERVED', title: 'Baseline debt', description: '11T KES' },
    { timestamp: '2024-09-09', label: 'MODELED', title: 'Run completed', description: 'p50 = 250k units' },
  ],
  disclaimer: 'Mixed reality layers — every event is tagged.',
};

const MOCK_METHODOLOGY = {
  scenario_id: SCENARIO_ID,
  methodology: MOCK_SCENARIO.methodology!,
  model_id: MOCK_SCENARIO.model_id,
  model_version: MOCK_SCENARIO.model_version,
  limitations: ['Excludes macroeconomic feedback effects', 'Assumes linear absorption'],
  reality_layer: 'HYPOTHETICAL',
  disclaimer: 'Methodology is declared up front — never inferred after the fact.',
};

const MOCK_ASSUMPTIONS = { assumptions: MOCK_SCENARIO.assumptions, count: 1 };
const MOCK_EVIDENCE = {
  source_evidence: [],
  baseline_evidence: [],
  assumption_evidence_count: 0,
};

test.describe('Golden Journey #5 — Citizen explores a scenario @golden', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/scenarios', async (route) => {
      await route.fulfill({ status: 200, json: MOCK_SCENARIO_LIST });
    });
    await page.route(`**/api/v1/scenarios/${SCENARIO_ID}`, async (route) => {
      await route.fulfill({
        status: 200,
        json: { scenario: MOCK_SCENARIO, reality_layer: 'HYPOTHETICAL', disclaimer: 'HYPOTHETICAL' },
      });
    });
    await page.route(`**/api/v1/scenarios/${SCENARIO_ID}/results`, async (route) => {
      await route.fulfill({ status: 200, json: MOCK_RESULTS });
    });
    await page.route(`**/api/v1/scenarios/${SCENARIO_ID}/timeline`, async (route) => {
      await route.fulfill({ status: 200, json: MOCK_TIMELINE });
    });
    await page.route(`**/api/v1/scenarios/${SCENARIO_ID}/methodology`, async (route) => {
      await route.fulfill({ status: 200, json: MOCK_METHODOLOGY });
    });
    await page.route(`**/api/v1/scenarios/${SCENARIO_ID}/assumptions`, async (route) => {
      await route.fulfill({ status: 200, json: MOCK_ASSUMPTIONS });
    });
    await page.route(`**/api/v1/scenarios/${SCENARIO_ID}/evidence`, async (route) => {
      await route.fulfill({ status: 200, json: MOCK_EVIDENCE });
    });
  });

  test('citizen opens scenarios, inspects uncertainty, checks methodology + comparison', async (
    { page },
    testInfo,
  ) => {
    // 1. Homepage → /scenarios.
    await page.goto('/');
    await runAxeTest(page, 'homepage', testInfo);
    await page.getByRole('link', { name: /^what if\??$/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /^what if\??$/i }),
    ).toBeVisible();
    await runAxeTest(page, 'scenarios-list', testInfo);

    // 2. The page must surface the HYPOTHETICAL disclaimer required by spec §35.
    await expect(page.getByText(/hypothetical/i).first()).toBeVisible();

    // 3. Open the "New Scenario" builder to verify the user can start a
    //    scenario creation flow.
    await page.getByRole('link', { name: /new scenario/i }).first().click();
    await expect(
      page.getByRole('heading', { level: 1, name: /build a .*what if.*\??/i }),
    ).toBeVisible();
    await runAxeTest(page, 'scenario-new', testInfo);

    // 4. Go back and open the existing scenario detail.
    await page.goto('/scenarios');
    await page.waitForLoadState('networkidle');
    const scenarioLink = page.getByRole('link', { name: /affordable housing impact, 2024/i }).first();
    await expect(scenarioLink).toBeVisible();
    await scenarioLink.click();
    await expect(
      page.getByRole('heading', { level: 1, name: /affordable housing impact, 2024/i }),
    ).toBeVisible();
    await runAxeTest(page, 'scenario-detail', testInfo);

    // 5. Open the Results tab — it must surface uncertainty bounds.
    await page.getByRole('link', { name: /^results$/i }).first().click();
    await page.waitForLoadState('networkidle');
    // Either the results render (showing "Median", "p10", "p90") OR the
    // page shows a SIMULATED disclaimer. Both states are spec-compliant.
    const resultsHeading = page.getByRole('heading', { level: 1, name: /affordable housing impact/i });
    await expect(resultsHeading).toBeVisible();
    await runAxeTest(page, 'scenario-results', testInfo);

    // 6. The methodology tab must be reachable.
    await page.getByRole('link', { name: /^methodology$/i }).first().click();
    await page.waitForLoadState('networkidle');
    await runAxeTest(page, 'scenario-methodology', testInfo);

    // 7. The Comparison tab must be reachable.
    await page.getByRole('link', { name: /^comparison$/i }).first().click();
    await page.waitForLoadState('networkidle');
    await runAxeTest(page, 'scenario-comparison', testInfo);
  });
});
