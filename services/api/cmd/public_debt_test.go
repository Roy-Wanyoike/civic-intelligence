package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	kenya_seed "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// newTestDebtRepo constructs an in-memory DebtRepository seeded with the
// Kenya dataset. Test helper — production callers go through
// main.WireDebtRepository instead.
func newTestDebtRepo(t *testing.T) legislation.DebtRepository {
	t.Helper()
	repo, err := legislation.WireDebtRepository(kenya_seed.SeedDebt)
	if err != nil {
		t.Fatalf("seed debt repo: %v", err)
	}
	return repo
}

// TestDebtDashboard_ReturnsSummary verifies that the dashboard endpoint
// returns total/domestic/external debt with disclaimer.
func TestDebtDashboard_ReturnsSummary(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debt", nil)
	rec := httptest.NewRecorder()

	makeDebtRouter(newTestDebtRepo(t))(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		CountryCode    string   `json:"country_code"`
		TotalDebtStock *float64 `json:"total_debt_stock"`
		DomesticDebt   *float64 `json:"domestic_debt"`
		ExternalDebt   *float64 `json:"external_debt"`
		Disclaimer     string   `json:"disclaimer"`
		SourceURL      string   `json:"source_url"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if resp.CountryCode != "KE" {
		t.Errorf("expected KE; got %s", resp.CountryCode)
	}
	if resp.TotalDebtStock == nil {
		t.Error("expected total debt stock")
	}
	if resp.DomesticDebt == nil || resp.ExternalDebt == nil {
		t.Error("expected domestic + external breakdown")
	}
	if resp.Disclaimer == "" {
		t.Error("expected disclaimer")
	}
	if !containsStr(resp.Disclaimer, "does not") {
		t.Error("expected disclaimer to mention what the platform does NOT do")
	}
}

// TestDebtTimeline_ReturnsPoints verifies that the timeline endpoint
// returns observations.
func TestDebtTimeline_ReturnsPoints(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debt/timeline", nil)
	rec := httptest.NewRecorder()

	makeDebtRouter(newTestDebtRepo(t))(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		Points []map[string]any `json:"points"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Points) < 10 {
		t.Errorf("expected at least 10 trend points; got %d", len(resp.Points))
	}
}

// TestGovernmentDebt_ReturnsSummaryWithDisclaimer verifies the critical
// attribution rule: the platform never says "President X borrowed KSh X".
func TestGovernmentDebt_ReturnsSummaryWithDisclaimer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debt/governments/admin-uhuru-kenyatta", nil)
	rec := httptest.NewRecorder()

	makeDebtRouter(newTestDebtRepo(t))(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		AdministrationID string   `json:"administration_id"`
		Period           string   `json:"period"`
		DebtAtStart      *float64 `json:"debt_at_start"`
		DebtAtEnd        *float64 `json:"debt_at_end"`
		NewBorrowing     *float64 `json:"new_borrowing"`
		Disclaimer       string   `json:"disclaimer"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if resp.AdministrationID != "admin-uhuru-kenyatta" {
		t.Errorf("expected admin-uhuru-kenyatta; got %s", resp.AdministrationID)
	}
	if resp.DebtAtStart == nil || resp.DebtAtEnd == nil {
		t.Error("expected debt at start and end")
	}
	if resp.NewBorrowing == nil {
		t.Error("expected new borrowing")
	}
	// CRITICAL: disclaimer must NOT say "President X borrowed"
	if !containsStr(resp.Disclaimer, "NOT") {
		t.Error("expected disclaimer to explicitly NOT attribute to a person")
	}
	if containsStr(resp.Disclaimer, "Uhuru borrowed") {
		t.Error("disclaimer must not say 'Uhuru borrowed'")
	}
}

// TestGovernmentDebt_UnknownAdministration_Returns404 verifies that the
// handler does not fabricate a summary for an unknown administration.
func TestGovernmentDebt_UnknownAdministration_Returns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debt/governments/admin-does-not-exist", nil)
	rec := httptest.NewRecorder()

	makeDebtRouter(newTestDebtRepo(t))(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404; got %d", rec.Code)
	}
}

// TestDebtLoans_EmptyByDefault_ReturnsZero verifies that the loans endpoint
// returns an empty list (with disclaimer) when no borrowing agreements have
// been seeded. Loan-level ingestion is pending issue #93.
func TestDebtLoans_EmptyByDefault_ReturnsZero(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debt/loans", nil)
	rec := httptest.NewRecorder()

	makeDebtRouter(newTestDebtRepo(t))(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		Count               int              `json:"count"`
		BorrowingAgreements []map[string]any `json:"borrowing_agreements"`
		Disclaimer          string           `json:"disclaimer"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Count != 0 {
		t.Errorf("expected count=0; got %d", resp.Count)
	}
	if len(resp.BorrowingAgreements) != 0 {
		t.Errorf("expected empty agreements; got %d", len(resp.BorrowingAgreements))
	}
	if resp.Disclaimer == "" {
		t.Error("expected disclaimer")
	}
}

func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		(len(haystack) > 0 && indexOfStr(haystack, needle) >= 0))
}

func indexOfStr(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
