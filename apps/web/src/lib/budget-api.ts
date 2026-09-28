// Client for the National Budget Allocations API endpoint (issue #285).

import { getJSON_ as getJSON } from './api';

/**
 * BudgetCategory — the 2-valued enum classifying a budget allocation.
 *
 * The wire values match the Go-side `kenya_seed.BudgetCategory` constants
 * (`BudgetCategoryRecurrent` / `BudgetCategoryDevelopment`) so the frontend
 * can switch on them directly without a mapping table. The frontend renders
 * recurrent cells in forest green (`#0c3b2e`) and development cells in
 * acacia gold (`#d4a017`) — see `apps/web/src/app/budget/budget-treemap.tsx`.
 */
export type BudgetCategory = 'recurrent' | 'development';

/**
 * BudgetAllocation — a single ministry's budget allocation for a single
 * fiscal year. Each row carries its own source_url so the frontend can
 * surface provenance per allocation — a citizen can verify every figure
 * against the Treasury's published Budget Estimates without leaving the
 * platform.
 *
 * The platform NEVER derives a "best-performing ministry" verdict from
 * these figures — they are surfaced raw (rule:
 * `NO_POLITICAL_PERFORMANCE_SCORE`). A ministry with a 0.5% allocation is
 * recorded as such; the citizen interprets the priority, the platform
 * does not.
 */
export interface BudgetAllocation {
  ministry: string;
  amount_kes_billions: number;
  percentage: number;
  category: BudgetCategory;
  source_url: string;
}

/**
 * BudgetResponse — the JSON envelope returned by
 * `GET /api/v1/budget?country=KE&year=2026`.
 *
 * Mirrors the Go-side `BudgetResponse` struct in
 * `services/api/cmd/budget.go` so the client can decode the response
 * without a separate mapping table.
 */
export interface BudgetResponse {
  country: string;
  year: string;
  items: BudgetAllocation[];
  total: number;
  source_url: string;
  source: 'seed';
}

/**
 * getBudget — fetch the budget allocations for the requested country +
 * year. Defaults to `country=KE` + `year=2026` when called with no
 * arguments (mirroring the API's own defaults so a caller that just
 * wants "Kenya's budget" doesn't have to construct the query string).
 *
 * Only Kenya has seed data today; other supported countries return 200
 * with an empty `items` list (NOT 404 — the frontend renders a graceful
 * "budget data not yet ingested for this country" empty state).
 */
export async function getBudget(
  params: { country?: string; year?: string } = {},
): Promise<BudgetResponse> {
  const qs = new URLSearchParams();
  if (params.country) qs.set('country', params.country);
  if (params.year) qs.set('year', params.year);
  const query = qs.toString();
  return getJSON<BudgetResponse>(`/api/v1/budget${query ? `?${query}` : ''}`);
}
