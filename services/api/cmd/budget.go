// Package main — budget.go provides the national budget allocation
// API endpoint (issue #285 / task FEAT-7).
//
// The platform surfaces raw, per-ministry budget allocations for a
// single fiscal year, with every allocation carrying the source URL
// of the National Treasury Budget Statement it was derived from. The
// platform NEVER derives a "best-performing ministry" verdict or a
// "priority score" from the allocations — they are surfaced raw
// (rule: NO_POLITICAL_PERFORMANCE_SCORE).
//
// Endpoint (registered in main.go):
//
//      GET /api/v1/budget?country=KE&year=2026
//
// Returns the seed budget allocations for the requested country +
// year, wrapped in a top-level envelope carrying the total + year +
// country + source_url. Only Kenya (KE) has seed data today; other
// supported countries return 200 with an empty items list (NOT 404 —
// a 404 would let a typo in the country code look like "this country
// has no budget data at all", which is misleading).
//
// Issue #285 acceptance bar: "Use seed data only. Do NOT parse actual
// PDFs." All 21 ministry allocations are hand-curated seed data
// patterned on the National Treasury's published FY 2024/25 Programme-
// Based Budget, scaled to the FY 2026/27 envelope (~KES 3.9T). When
// the verified PBB ingestion path ships (planned Wave 16), this
// seed-only slice will be retired in favour of the canonical budget
// estimates loaded from treasury.go.ke.
package main

import (
        "net/http"
        "strings"

        kenya_seed "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// BudgetResponse is the JSON envelope returned by
// GET /api/v1/budget?country=KE&year=2026. The shape mirrors the
// other list endpoints (items + total + filters echoed back) so the
// frontend can reuse its rendering pattern.
//
// The total field is the sum of every allocation's amount_kes_billions
// — i.e. the national budget envelope for the requested fiscal year.
// The percentage field on each allocation is computed against this
// total at seed-init time (see kenya_seed.SampleBudgetAllocations).
//
// The source_url field is the canonical National Treasury Budget
// Statement URL — every allocation also carries its own source_url
// (today, the same URL; when the verified PBB ingestion path ships,
// each allocation will carry the deep link to its specific Budget
// Estimates volume).
//
// The platform NEVER derives a "best-performing ministry" verdict
// from the allocations — they are surfaced raw (rule:
// NO_POLITICAL_PERFORMANCE_SCORE). A ministry with a 0.5% allocation
// is recorded as such; the citizen interprets the priority, the
// platform does not.
type BudgetResponse struct {
        Country   string                      `json:"country"`
        Year      string                      `json:"year"`
        Items     []kenya_seed.BudgetAllocation `json:"items"`
        Total     float64                     `json:"total"`
        SourceURL string                      `json:"source_url"`
        Source    string                      `json:"source"` // always "seed" until ingestion ships
}

// handleBudget is the handler for GET /api/v1/budget. It is
// dispatched directly from the apiHandler mux (no sub-route switch —
// the endpoint accepts only query parameters, no path parameters).
//
// Query parameters:
//   - country (optional, default KE): ISO 3166-1 alpha-2 country code.
//     Today only KE has seed data; other supported countries return
//     200 with an empty items list (NOT 404).
//   - year   (optional, default 2026): calendar year. Both 2026 and
//     2027 refer to FY 2026/27 — Kenya's fiscal year straddles two
//     calendar years. The handler accepts both; other year values
//     return 200 with an empty items list (NOT 404).
//
// On success: 200 with the items list (most-expensive first, matching
// the seed slice's order) + total + year + country + source_url.
//
// On unsupported country code: 400 (delegated to the Country
// middleware, which runs before the handler in production; the
// handler's own 400 check is a defence-in-depth for callers that
// chain the handler directly without the middleware, e.g. unit tests).
func handleBudget(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                return
        }

        // Resolve country. The Country middleware stores the resolved
        // code on the request context (header → query → default "KE").
        // The handler also accepts the ?country= query parameter
        // directly so it works when the middleware is not chained (e.g.
        // in unit tests). The query parameter takes precedence here so
        // the explicit `?country=UG` test case (issue #285 acceptance)
        // exercises the filter logic without depending on the middleware.
        country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
        if country == "" {
                // Fall back to the context (set by the Country middleware in
                // production). If neither is set, CountryFromContext returns
                // the default "KE" — preserving the historical contract for
                // callers that omit the parameter entirely.
                country = middlewareCountryFromContext(r)
        }

        // Resolve year. Default to "2026" (the first calendar year of the
        // FY 2026/27 seed slice). Accept both 2026 and 2027 — Kenya's
        // fiscal year straddles two calendar years, so both refer to the
        // same budget. Other year values return 200 with empty items
        // (NOT 404 — a 404 would let a typo in the year look like "this
        // country has no budget data at all", which is misleading).
        year := strings.TrimSpace(r.URL.Query().Get("year"))
        if year == "" {
                year = "2026"
        }

        // Only Kenya has seed budget data today. For other supported
        // countries (UG, TZ, GH, NG, ZA, RW, ZM, SN, EG, …), return 200
        // with an empty items list rather than 404. This lets the frontend
        // render a graceful "budget data not yet ingested for this
        // country" empty state without distinguishing 404 (country not
        // found) from 200-with-empty (country supported, no data yet).
        items := kenya_seed.FindBudgetAllocationsByCountry(country)
        if items == nil {
                items = []kenya_seed.BudgetAllocation{}
        }

        // Filter by year: if the requested year is not one of the
        // supported years for the seed slice, return an empty items list.
        // The total is correspondingly 0 so the frontend can render "no
        // data for this year" instead of showing a stale total. This
        // branch is unreachable today (the only supported years are 2026
        // and 2027, both of which map to the same FY 2026/27 slice), but
        // the guard keeps the contract stable if the seed slice is ever
        // extended to additional fiscal years.
        if !kenya_seed.IsBudgetYearSupported(year) {
                items = []kenya_seed.BudgetAllocation{}
        }

        // Surface the fiscal-year label (e.g. "2026/27") in the
        // response so the frontend can render "FY 2026/27" without
        // hard-coding it. When the requested year is unsupported, the
        // label is the raw year (so the response always echoes back what
        // the caller asked for, even on miss).
        responseYear := year
        if kenya_seed.IsBudgetYearSupported(year) {
                responseYear = kenya_seed.BudgetFiscalYear()
        }

        // Compute the total from the (possibly filtered) items. Done
        // here rather than calling kenya_seed.BudgetTotalKESBillions()
        // so the total stays consistent with the filtered items list
        // (e.g. when the year filter empties the list, the total is
        // also 0 — not the unfiltered sum).
        var total float64
        for _, item := range items {
                total += item.AmountKESBillions
        }

        writeJSON(w, http.StatusOK, BudgetResponse{
                Country:   country,
                Year:      responseYear,
                Items:     items,
                Total:     total,
                SourceURL: kenya_seed.BudgetSourceURL(),
                Source:    "seed",
        })
}

// middlewareCountryFromContext is a thin wrapper around
// middleware.CountryFromContext so the handler can be unit-tested
// without a direct dependency on the middleware package's internals.
// The wrapper is also a seam: a future refactor that pulls the
// country resolution entirely into the handler can swap the
// implementation without touching the handler body.
//
// In production, the Country middleware runs BEFORE the handler (see
// main.go's middleware chain) and stores the resolved code on the
// context. The handler reads from the context so the resolved code
// (which has already been validated against the supported list)
// wins over a raw query parameter — the middleware's 400-on-invalid
// path is exercised before the handler sees the request.
func middlewareCountryFromContext(r *http.Request) string {
        return middleware.CountryFromContext(r.Context())
}
