// Package integration — api_debt_test.go exercises the Public Debt &
// Borrowing Intelligence endpoints. The Kenya seed data (kenya_seed.SeedDebt)
// populates the in-memory DebtRepository at startup with CBK + Treasury
// observations, so every test asserts against deterministic seed values.
//
// CRITICAL ATTRIBUTION RULE (Spec §35, §37): the platform never attributes
// sovereign borrowing personally to a president or ranks administrations
// by "borrowing performance". Every response carries a disclaimer that
// explicitly states the platform does NOT compute political performance
// scores.
//
// Endpoints covered:
//
//	GET /api/v1/debt                       -- national debt dashboard
//	GET /api/v1/debt/loans                 -- borrowing register
//	GET /api/v1/debt/timeline              -- debt stock timeline
//	GET /api/v1/debt/governments/{id}      -- per-administration summary
//	GET /api/v1/debt/legislatures/{id}     -- per-legislature summary
package integration

import (
	"net/http"
	"strings"
	"testing"
)

// debtDashboardResponse is the documented /api/v1/debt success shape.
type debtDashboardResponse struct {
	CountryCode    string   `json:"country_code"`
	TotalDebtStock *float64 `json:"total_debt_stock"`
	DomesticDebt   *float64 `json:"domestic_debt"`
	ExternalDebt   *float64 `json:"external_debt"`
	DebtService    *float64 `json:"debt_service"`
	DebtToGDP      *float64 `json:"debt_to_gdp"`
	Currency       string   `json:"currency"`
	AsOf           string   `json:"as_of"`
	SourceURL      string   `json:"source_url"`
	Disclaimer     string   `json:"disclaimer"`
}

// TestDebt_DashboardReturnsSummary verifies the dashboard endpoint returns
// the latest CBK snapshot for Kenya with the documented fields.
func TestDebt_DashboardReturnsSummary(t *testing.T) {
	var resp debtDashboardResponse
	status := mustGet(t, apiURL("/debt"), &resp)
	assertStatus(t, "/debt", http.StatusOK, status)

	if resp.CountryCode != "KE" {
		t.Errorf("debt: expected country_code KE, got %q", resp.CountryCode)
	}
	if resp.TotalDebtStock == nil {
		t.Error("debt: expected total_debt_stock to be present")
	}
	if resp.DomesticDebt == nil || resp.ExternalDebt == nil {
		t.Error("debt: expected domestic + external debt breakdown")
	}
	if resp.Currency != "KES" {
		t.Errorf("debt: expected currency KES, got %q", resp.Currency)
	}
	if resp.AsOf == "" {
		t.Error("debt: expected non-empty as_of date")
	}
	if resp.SourceURL == "" {
		t.Error("debt: expected source_url (evidence-first)")
	}
	if resp.Disclaimer == "" {
		t.Error("debt: expected non-empty disclaimer")
	}
	// CRITICAL: disclaimer must mention what the platform does NOT do.
	if !strings.Contains(resp.Disclaimer, "does not") {
		t.Errorf("debt disclaimer must mention 'does not' (non-ranking promise); got %q", resp.Disclaimer)
	}
}

// TestDebt_TimelineReturnsObservations verifies the timeline endpoint
// returns a chronological series of immutable CBK observations.
func TestDebt_TimelineReturnsObservations(t *testing.T) {
	var resp struct {
		CountryCode string              `json:"country_code"`
		Points      []map[string]any    `json:"points"`
		Disclaimer  string              `json:"disclaimer"`
	}
	status := mustGet(t, apiURL("/debt/timeline"), &resp)
	assertStatus(t, "/debt/timeline", http.StatusOK, status)

	if resp.CountryCode != "KE" {
		t.Errorf("debt/timeline: expected country_code KE, got %q", resp.CountryCode)
	}
	if len(resp.Points) < 10 {
		t.Errorf("debt/timeline: expected at least 10 points, got %d", len(resp.Points))
	}
	// Each point must carry a date + source_url (immutable observation).
	for i, p := range resp.Points {
		if _, ok := p["date"]; !ok {
			t.Errorf("debt/timeline[%d]: missing date", i)
		}
		if _, ok := p["source_url"]; !ok {
			t.Errorf("debt/timeline[%d]: missing source_url", i)
		}
	}
	if !strings.Contains(resp.Disclaimer, "NOT") {
		t.Errorf("debt/timeline disclaimer should mention NOT (no causal attribution); got %q", resp.Disclaimer)
	}
}

// TestDebt_LoansReturnsAgreements verifies the borrowing register returns
// seeded agreements sourced from public press releases (IMF, World Bank,
// AfDB, China Exim Bank, Eurobond prospectuses).
func TestDebt_LoansReturnsAgreements(t *testing.T) {
	var resp struct {
		BorrowingAgreements []map[string]any `json:"borrowing_agreements"`
		Count               int             `json:"count"`
		Disclaimer          string          `json:"disclaimer"`
	}
	status := mustGet(t, apiURL("/debt/loans"), &resp)
	assertStatus(t, "/debt/loans", http.StatusOK, status)

	if resp.Count == 0 {
		t.Fatal("debt/loans: expected seeded agreements, got count=0")
	}
	if len(resp.BorrowingAgreements) != resp.Count {
		t.Errorf("debt/loans: count (%d) != len(borrowing_agreements) (%d)", resp.Count, len(resp.BorrowingAgreements))
	}
	if resp.Disclaimer == "" {
		t.Error("debt/loans: expected disclaimer")
	}
	// Every agreement must carry a source_url (evidence-first).
	for i, a := range resp.BorrowingAgreements {
		if _, ok := a["source_url"]; !ok {
			t.Errorf("debt/loans[%d]: missing source_url", i)
		}
		if _, ok := a["creditor_name"]; !ok {
			t.Errorf("debt/loans[%d]: missing creditor_name", i)
		}
		if _, ok := a["original_amount"]; !ok {
			t.Errorf("debt/loans[%d]: missing original_amount", i)
		}
	}
}

// TestDebt_GovernmentSummaryReturnsCanonicalDisclaimer verifies the
// per-administration summary includes the canonical
// NO_POLITICAL_PERFORMANCE_SCORE constant. The canonical constant must be
// appended verbatim so future spec edits propagate automatically.
func TestDebt_GovernmentSummaryReturnsCanonicalDisclaimer(t *testing.T) {
	var resp struct {
		AdministrationID string   `json:"administration_id"`
		Period           string   `json:"period"`
		DebtAtStart      *float64 `json:"debt_at_start"`
		DebtAtEnd        *float64 `json:"debt_at_end"`
		NewBorrowing     *float64 `json:"new_borrowing"`
		Disclaimer       string   `json:"disclaimer"`
	}
	status := mustGet(t, apiURL("/debt/governments/admin-uhuru-kenyatta"), &resp)
	assertStatus(t, "/debt/governments/admin-uhuru-kenyatta", http.StatusOK, status)

	if resp.AdministrationID != "admin-uhuru-kenyatta" {
		t.Errorf("debt/gov: expected admin-uhuru-kenyatta, got %q", resp.AdministrationID)
	}
	if resp.DebtAtStart == nil || resp.DebtAtEnd == nil {
		t.Error("debt/gov: expected debt_at_start + debt_at_end")
	}
	if resp.NewBorrowing == nil {
		t.Error("debt/gov: expected new_borrowing")
	}
	// CRITICAL: canonical disclaimer must mention "best borrower" (the
	// platform-wide promise to never compute political performance scores).
	if !strings.Contains(resp.Disclaimer, "best borrower") {
		t.Errorf("debt/gov disclaimer must contain canonical 'best borrower' phrase; got %q", resp.Disclaimer)
	}
	// Per-admin text must still be present (we append, not replace).
	if !strings.Contains(resp.Disclaimer, "admin-uhuru-kenyatta") &&
		!strings.Contains(resp.Disclaimer, "Uhuru Kenyatta") {
		t.Errorf("debt/gov disclaimer should retain per-admin context; got %q", resp.Disclaimer)
	}
}

// TestDebt_GovernmentSummaryUnknownReturns404 verifies the handler does
// NOT fabricate a summary for an unknown administration.
func TestDebt_GovernmentSummaryUnknownReturns404(t *testing.T) {
	var dummy map[string]any
	status := mustGet(t, apiURL("/debt/governments/admin-does-not-exist"), &dummy)
	assertStatus(t, "/debt/governments/unknown", http.StatusNotFound, status)
}

// TestDebt_LegislatureSummaryReturnsFields verifies the per-legislature
// debt endpoint surfaces the documented fields: debt at beginning/end,
// new borrowing, domestic/external split, disbursements, repayments, debt
// service, outstanding obligations, currency, source URLs, disclaimer.
//
// CRITICAL ATTRIBUTION RULE (Spec §35): the disclaimer must NOT say
// "the 13th Parliament borrowed KSh X".
func TestDebt_LegislatureSummaryReturnsFields(t *testing.T) {
	var resp struct {
		LegislatureID         string   `json:"legislature_id"`
		Period                 string   `json:"period"`
		TotalNewBorrowing      *float64 `json:"total_new_borrowing"`
		DomesticBorrowing      *float64 `json:"domestic_borrowing"`
		ExternalBorrowing      *float64 `json:"external_borrowing"`
		Disbursements          *float64 `json:"disbursements"`
		Repayments             *float64 `json:"repayments"`
		DebtService            *float64 `json:"debt_service"`
		DebtStockAtStart       *float64 `json:"debt_stock_at_start"`
		DebtStockAtEnd         *float64 `json:"debt_stock_at_end"`
		OutstandingObligations *float64 `json:"outstanding_obligations"`
		Currency               string   `json:"currency"`
		SourceURLs             []string `json:"source_urls"`
		Disclaimer             string   `json:"disclaimer"`
	}
	status := mustGet(t, apiURL("/debt/legislatures/legislature-ke-13"), &resp)
	assertStatus(t, "/debt/legislatures/legislature-ke-13", http.StatusOK, status)

	if resp.LegislatureID != "legislature-ke-13" {
		t.Errorf("debt/leg: expected legislature-ke-13, got %q", resp.LegislatureID)
	}
	if resp.DebtStockAtStart == nil || resp.DebtStockAtEnd == nil {
		t.Error("debt/leg: expected debt_stock_at_start + debt_stock_at_end")
	}
	if resp.DomesticBorrowing == nil || resp.ExternalBorrowing == nil {
		t.Error("debt/leg: expected domestic + external borrowing split")
	}
	if resp.Disbursements == nil || resp.Repayments == nil || resp.DebtService == nil {
		t.Error("debt/leg: expected disbursements + repayments + debt_service")
	}
	if resp.OutstandingObligations == nil {
		t.Error("debt/leg: expected outstanding_obligations")
	}
	if resp.Currency != "KES" {
		t.Errorf("debt/leg: expected currency KES, got %q", resp.Currency)
	}
	if len(resp.SourceURLs) == 0 {
		t.Error("debt/leg: expected at least one source URL (evidence-first)")
	}
	// CRITICAL: disclaimer must NOT say "the 13th Parliament borrowed".
	if strings.Contains(resp.Disclaimer, "13th Parliament borrowed") {
		t.Errorf("debt/leg disclaimer must not attribute borrowing to a Parliament; got %q", resp.Disclaimer)
	}
	// Canonical constant must be present.
	if !strings.Contains(resp.Disclaimer, "best borrower") {
		t.Errorf("debt/leg disclaimer must contain canonical 'best borrower' phrase; got %q", resp.Disclaimer)
	}
}

// TestDebt_LegislatureUnknownReturns404 verifies the handler surfaces 404
// for an unknown legislature (does NOT fabricate).
func TestDebt_LegislatureUnknownReturns404(t *testing.T) {
	var dummy map[string]any
	status := mustGet(t, apiURL("/debt/legislatures/legislature-does-not-exist"), &dummy)
	assertStatus(t, "/debt/legislatures/unknown", http.StatusNotFound, status)
}

// TestDebt_UnknownSubResourceReturns404 verifies /debt/<unknown> returns
// 404 not_found (not 500).
func TestDebt_UnknownSubResourceReturns404(t *testing.T) {
	var dummy map[string]any
	status := mustGet(t, apiURL("/debt/unknown-sub-resource"), &dummy)
	assertStatus(t, "/debt/unknown", http.StatusNotFound, status)
}
