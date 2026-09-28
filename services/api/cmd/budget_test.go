package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	kenya_seed "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
)

// budgetAllocationJSON is the JSON-decoded form of a single
// BudgetAllocation item. Decoding into a struct (rather than
// map[string]any) lets the tests reference fields by name + type,
// which makes failures easier to diagnose when a refactor renames a
// field or flips an omitempty tag.
type budgetAllocationJSON struct {
	Ministry          string  `json:"ministry"`
	AmountKESBillions float64 `json:"amount_kes_billions"`
	Percentage        float64 `json:"percentage"`
	Category          string  `json:"category"`
	SourceURL         string  `json:"source_url"`
}

// budgetResponseJSON is the JSON envelope returned by
// GET /api/v1/budget?country=KE&year=2026. Only the fields the tests
// assert on are decoded — the rest are ignored.
type budgetResponseJSON struct {
	Country   string                   `json:"country"`
	Year      string                   `json:"year"`
	Items     []budgetAllocationJSON   `json:"items"`
	Total     float64                  `json:"total"`
	SourceURL string                   `json:"source_url"`
	Source    string                   `json:"source"`
}

// validBudgetCategories is the canonical set of BudgetCategory wire
// values. Used by the tests to assert every response item carries one
// of the 2 allowed categories (no typos, no silently-introduced
// "capital" / "ops" / "running" variants). Kept in the test file
// (rather than reading allBudgetCategories from kenya_seed) so the
// test fails loudly if a new category is added without a corresponding
// test update.
var validBudgetCategories = map[string]bool{
	"recurrent":   true,
	"development": true,
}

// expectedBudgetAllocations returns the expected seed allocations for
// the given country. The test uses this to cross-check the response
// items against the seed slice — catches a refactor that loses a row
// or scrambles the per-country filter.
func expectedBudgetAllocations(t *testing.T, country string) []budgetAllocationJSON {
	t.Helper()
	seed := kenya_seed.FindBudgetAllocationsByCountry(country)
	if seed == nil {
		t.Fatalf("expected seed slice to have ≥1 allocation for %s", country)
	}
	out := make([]budgetAllocationJSON, 0, len(seed))
	for _, a := range seed {
		out = append(out, budgetAllocationJSON{
			Ministry:          a.Ministry,
			AmountKESBillions: a.AmountKESBillions,
			Percentage:        a.Percentage,
			Category:          string(a.Category),
			SourceURL:         a.SourceURL,
		})
	}
	return out
}

// TestBudgetHandler_ReturnsAllocations verifies the new
// /api/v1/budget endpoint (issue #285) returns Kenya's national budget
// allocations — every item carrying all 5 required fields, every
// category one of the 2 allowed values, and the items + total
// consistent with the seed slice.
//
// The test drives handleBudget directly (not the full mux) so the
// handler is exercised in isolation — a regression that breaks the
// handler body will fail this test even if the mux registration is
// intact (the registration is exercised separately by the OpenAPI
// contract test suite).
func TestBudgetHandler_ReturnsAllocations(t *testing.T) {
	const country = "KE"
	const year = "2026"
	want := expectedBudgetAllocations(t, country)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/budget?country="+country+"&year="+year, nil)
	rr := httptest.NewRecorder()
	handleBudget(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/budget?country=%s&year=%s, got %d (body=%s)", country, year, rr.Code, rr.Body.String())
	}

	var resp budgetResponseJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Country + year are echoed back. The year is the fiscal-year
	// label ("2026/27"), not the raw query parameter — both 2026
	// and 2027 refer to the same fiscal year, so the response
	// surfaces the label rather than the ambiguous calendar year.
	if resp.Country != country {
		t.Errorf("expected country=%q, got %q", country, resp.Country)
	}
	if resp.Year != kenya_seed.BudgetFiscalYear() {
		t.Errorf("expected year=%q (fiscal-year label), got %q", kenya_seed.BudgetFiscalYear(), resp.Year)
	}
	if resp.Source != "seed" {
		t.Errorf("expected source=seed (issue #285 acceptance: seed data only), got %q", resp.Source)
	}
	if resp.SourceURL == "" {
		t.Errorf("expected source_url to be non-empty (Treasury Budget Statement URL)")
	}
	if resp.SourceURL != kenya_seed.BudgetSourceURL() {
		t.Errorf("expected source_url=%q, got %q", kenya_seed.BudgetSourceURL(), resp.SourceURL)
	}

	// Issue #285 acceptance: 21 ministry allocations totalling
	// ~KES 3.9T. The exact count is asserted here so a refactor
	// that drops a ministry from the seed slice fails loudly.
	if len(resp.Items) != len(want) {
		t.Fatalf("expected %d items (matches seed slice), got %d", len(want), len(resp.Items))
	}

	// Assert every item carries all 5 required fields + every
	// category value is one of the 2 allowed kinds.
	previousAmount := 0.0
	for i, item := range resp.Items {
		if item.Ministry == "" {
			t.Errorf("item[%d]: expected ministry to be non-empty", i)
		}
		if item.AmountKESBillions <= 0 {
			t.Errorf("item[%d]: expected amount_kes_billions to be positive, got %f", i, item.AmountKESBillions)
		}
		if item.Percentage <= 0 || item.Percentage > 100 {
			t.Errorf("item[%d]: expected 0 < percentage ≤ 100, got %f", i, item.Percentage)
		}
		if !validBudgetCategories[item.Category] {
			t.Errorf("item[%d]: expected category to be one of recurrent/development, got %q", i, item.Category)
		}
		if item.SourceURL == "" {
			t.Errorf("item[%d]: expected source_url to be non-empty", i)
		}
		// Items sorted most-expensive-first (largest allocation at
		// the top). The seed slice is already stored
		// most-expensive-first, so the response should preserve
		// that order without re-sorting — this assertion catches a
		// refactor that loses the order.
		if i > 0 && item.AmountKESBillions > previousAmount {
			t.Errorf("item[%d]: amounts not sorted most-expensive-first (previous=%f, current=%f)", i, previousAmount, item.AmountKESBillions)
		}
		previousAmount = item.AmountKESBillions
	}

	// Cross-check the response items against the seed slice
	// (catches a refactor that loses a row or scrambles the
	// per-ministry filter).
	wantByMinistry := map[string]budgetAllocationJSON{}
	for _, w := range want {
		wantByMinistry[w.Ministry] = w
	}
	for _, got := range resp.Items {
		w, ok := wantByMinistry[got.Ministry]
		if !ok {
			t.Errorf("response item with ministry=%q not in seed slice", got.Ministry)
			continue
		}
		if got.AmountKESBillions != w.AmountKESBillions {
			t.Errorf("ministry=%q: expected amount_kes_billions=%f, got %f", got.Ministry, w.AmountKESBillions, got.AmountKESBillions)
		}
		if got.Category != w.Category {
			t.Errorf("ministry=%q: expected category=%q, got %q", got.Ministry, w.Category, got.Category)
		}
		if got.SourceURL != w.SourceURL {
			t.Errorf("ministry=%q: expected source_url=%q, got %q", got.Ministry, w.SourceURL, got.SourceURL)
		}
	}

	// The total MUST match the sum of every item's amount — the
	// handler computes it live from the (possibly filtered) items
	// so the total stays consistent with the items list even when
	// the year filter empties the list.
	wantTotal := 0.0
	for _, w := range want {
		wantTotal += w.AmountKESBillions
	}
	if resp.Total != wantTotal {
		t.Errorf("expected total=%f (sum of seed amounts), got %f", wantTotal, resp.Total)
	}
	// Issue #285 acceptance: total ~KES 3.9T (3,900B ± 100B). The
	// exact value is asserted here so a refactor that changes the
	// seed amounts without adjusting the acceptance bar fails
	// loudly.
	if resp.Total < 3800 || resp.Total > 4000 {
		t.Errorf("expected total ~3,900B (3.9T ± 100B), got %f", resp.Total)
	}

	// Percentages MUST sum to ~100% (rounding to 2 decimal places
	// is applied at seed-init, so the sum may be 99.99% or 100.01%
	// — anything outside [99.5, 100.5] indicates a drift).
	pctSum := 0.0
	for _, item := range resp.Items {
		pctSum += item.Percentage
	}
	if pctSum < 99.5 || pctSum > 100.5 {
		t.Errorf("expected percentages to sum to ~100%%, got %f", pctSum)
	}
}

// TestBudgetHandler_FiltersByCountry verifies the budget handler
// returns an empty items list (NOT 404) for supported countries that
// have no seed data yet. This guards the "every datum is sourced"
// invariant: a 404 would let a typo in the country code look like
// "this country has no budget data at all", which is misleading —
// an empty 200 lets the frontend render a graceful "budget data not
// yet ingested for this country" empty state.
//
// The test exercises three branches:
//   - country=UG (supported, no seed data) → 200 with empty items
//     + total=0 + country echoed
//   - country=NG (supported, no seed data) → 200 with empty items
//   - country=KE (default, has seed data) → 200 with full items
//     (control case — verifies the filter is country-specific, not
//     a blanket "always return empty" bug)
func TestBudgetHandler_FiltersByCountry(t *testing.T) {
	cases := []struct {
		name        string
		country     string
		wantItems   int
		wantTotalGT bool // true → expect total > 0; false → expect total == 0
	}{
		{
			name:        "UG_supported_no_seed_data_returns_empty",
			country:     "UG",
			wantItems:   0,
			wantTotalGT: false,
		},
		{
			name:        "NG_supported_no_seed_data_returns_empty",
			country:     "NG",
			wantItems:   0,
			wantTotalGT: false,
		},
		{
			name:        "ZA_supported_no_seed_data_returns_empty",
			country:     "ZA",
			wantItems:   0,
			wantTotalGT: false,
		},
		{
			name:        "KE_default_has_seed_data_returns_full",
			country:     "KE",
			wantItems:   len(kenya_seed.SampleBudgetAllocations),
			wantTotalGT: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/budget?country="+tc.country+"&year=2026", nil)
			rr := httptest.NewRecorder()
			handleBudget(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200 for country=%s (NOT 404 — empty 200 is the contract for supported countries without seed data), got %d (body=%s)", tc.country, rr.Code, rr.Body.String())
			}

			var resp budgetResponseJSON
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode: %v", err)
			}

			// Country is always echoed back (even on miss).
			if resp.Country != tc.country {
				t.Errorf("expected country=%q, got %q", tc.country, resp.Country)
			}
			// Source URL is always present (the canonical Treasury
			// URL — even for non-KE countries, the response surfaces
			// the seed source so a citizen can verify the platform's
			// data provenance).
			if resp.SourceURL == "" {
				t.Errorf("expected source_url to be non-empty even for non-KE country")
			}
			if resp.Source != "seed" {
				t.Errorf("expected source=seed, got %q", resp.Source)
			}

			if len(resp.Items) != tc.wantItems {
				t.Errorf("expected %d items for country=%s, got %d", tc.wantItems, tc.country, len(resp.Items))
			}

			if tc.wantTotalGT && resp.Total <= 0 {
				t.Errorf("expected total > 0 for country=%s (has seed data), got %f", tc.country, resp.Total)
			}
			if !tc.wantTotalGT && resp.Total != 0 {
				t.Errorf("expected total=0 for country=%s (no seed data), got %f", tc.country, resp.Total)
			}
		})
	}
}

// TestBudgetHandler_DefaultCountryIsKE verifies the handler defaults
// to Kenya when the country query parameter is omitted. This guards
// the historical API contract — the Country middleware would resolve
// to "KE" in production, but the handler's own fallback (via
// middleware.CountryFromContext) is the production path the test
// exercises here.
func TestBudgetHandler_DefaultCountryIsKE(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/budget?year=2026", nil)
	rr := httptest.NewRecorder()
	handleBudget(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	var resp budgetResponseJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Country defaults to KE (the platform's historical default +
	// the first country adapter shipped). The handler resolves this
	// via middleware.CountryFromContext, which returns DefaultCountry
	// ("KE") when no country is stored on the context.
	if resp.Country != "KE" {
		t.Errorf("expected default country=KE, got %q", resp.Country)
	}
	if len(resp.Items) == 0 {
		t.Errorf("expected KE to return seed allocations, got 0 items")
	}
}

// TestBudgetHandler_FiltersByYear verifies the handler accepts both
// 2026 and 2027 as aliases for FY 2026/27 (Kenya's fiscal year
// straddles two calendar years) and returns the seed data for both.
// Other year values return 200 with an empty items list (NOT 404).
func TestBudgetHandler_FiltersByYear(t *testing.T) {
	cases := []struct {
		name      string
		year      string
		wantItems int
	}{
		{"year_2026_returns_seed_data", "2026", len(kenya_seed.SampleBudgetAllocations)},
		{"year_2027_returns_seed_data", "2027", len(kenya_seed.SampleBudgetAllocations)},
		{"year_2025_returns_empty", "2025", 0},
		{"year_2030_returns_empty", "2030", 0},
		{"year_omitted_defaults_to_2026", "", len(kenya_seed.SampleBudgetAllocations)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/api/v1/budget?country=KE"
			if tc.year != "" {
				url += "&year=" + tc.year
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()
			handleBudget(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200 for year=%q, got %d (body=%s)", tc.year, rr.Code, rr.Body.String())
			}

			var resp budgetResponseJSON
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if len(resp.Items) != tc.wantItems {
				t.Errorf("expected %d items for year=%q, got %d", tc.wantItems, tc.year, len(resp.Items))
			}

			// Year echoed back: for supported years (2026/2027),
			// the response surfaces the fiscal-year label
			// ("2026/27"). For unsupported years, the raw year is
			// echoed back so the caller can see what they asked
			// for even on miss.
			if tc.year == "" || tc.year == "2026" || tc.year == "2027" {
				if resp.Year != kenya_seed.BudgetFiscalYear() {
					t.Errorf("expected year=%q (fiscal-year label) for year=%q, got %q", kenya_seed.BudgetFiscalYear(), tc.year, resp.Year)
				}
			} else {
				if resp.Year != tc.year {
					t.Errorf("expected year=%q (raw echo) for unsupported year, got %q", tc.year, resp.Year)
				}
			}
		})
	}
}

// TestBudgetHandler_RejectsNonGet verifies the handler returns 405
// Method Not Allowed for non-GET methods. The endpoint is read-only
// (budget allocations are seeded, not user-mutable), so POST/PUT/
// DELETE/PATCH must be rejected.
func TestBudgetHandler_RejectsNonGet(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/budget?country=KE&year=2026", nil)
			rr := httptest.NewRecorder()
			handleBudget(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405 for %s, got %d (body=%s)", method, rr.Code, rr.Body.String())
			}

			// Assert the canonical error envelope shape (error +
			// message).
			var errResp struct {
				Error   string `json:"error"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if errResp.Error != "method_not_allowed" {
				t.Errorf("expected error=method_not_allowed, got %q", errResp.Error)
			}
			if errResp.Message == "" {
				t.Errorf("expected non-empty message, got empty")
			}
		})
	}
}
