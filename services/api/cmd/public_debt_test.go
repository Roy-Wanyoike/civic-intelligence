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

// TestGovernmentDebtSummary_IncludesCanonicalDisclaimer verifies that the
// canonical NO_POLITICAL_PERFORMANCE_SCORE constant (Spec section 37) is
// appended to every per-administration summary. Issue #222.
//
// The canonical constant is appended verbatim — not used to replace the
// per-admin text. So the response disclaimer must contain BOTH the
// per-admin text (which mentions the administration by name) AND the
// canonical phrase about not computing political performance scores.
func TestGovernmentDebtSummary_IncludesCanonicalDisclaimer(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/debt/governments/admin-uhuru-kenyatta", nil)
        rec := httptest.NewRecorder()

        makeDebtRouter(newTestDebtRepo(t))(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                AdministrationID string `json:"administration_id"`
                Disclaimer       string `json:"disclaimer"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        // The canonical constant must appear in the response.
        if !containsStr(resp.Disclaimer, legislation.NO_POLITICAL_PERFORMANCE_SCORE) {
                t.Errorf("expected disclaimer to contain the canonical NO_POLITICAL_PERFORMANCE_SCORE constant; got: %q", resp.Disclaimer)
        }
        // Spot-check a unique phrase from the canonical constant that is NOT in
        // the per-admin seed text. This guards against the test passing by
        // accident if the per-admin text happened to overlap with the constant.
        if !containsStr(resp.Disclaimer, "best borrower") {
                t.Errorf("expected disclaimer to contain the canonical phrase 'best borrower'; got: %q", resp.Disclaimer)
        }
        // The per-admin text must ALSO still be present (we append, not replace).
        if !containsStr(resp.Disclaimer, "admin-uhuru-kenyatta") &&
                !containsStr(resp.Disclaimer, "Uhuru Kenyatta") {
                t.Errorf("expected disclaimer to retain per-administration context; got: %q", resp.Disclaimer)
        }
}

// TestGovernmentDebtSummary_IncludesCanonicalDisclaimer_Ruto verifies the
// canonical disclaimer is appended to EVERY administration summary, not just
// the Uhuru Kenyatta one.
func TestGovernmentDebtSummary_IncludesCanonicalDisclaimer_Ruto(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/debt/governments/admin-william-ruto", nil)
        rec := httptest.NewRecorder()

        makeDebtRouter(newTestDebtRepo(t))(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Disclaimer string `json:"disclaimer"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if !containsStr(resp.Disclaimer, legislation.NO_POLITICAL_PERFORMANCE_SCORE) {
                t.Errorf("expected canonical NO_POLITICAL_PERFORMANCE_SCORE to appear for admin-william-ruto; got: %q", resp.Disclaimer)
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

// TestDebtLoans_ReturnsSeededAgreements verifies that the loans endpoint
// returns the seeded borrowing agreements (sourced from public press releases:
// IMF, World Bank, AfDB, China Exim Bank, Eurobonds).
func TestDebtLoans_ReturnsSeededAgreements(t *testing.T) {
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

        if resp.Count == 0 {
                t.Errorf("expected seeded agreements; got count=0")
        }
        if len(resp.BorrowingAgreements) != resp.Count {
                t.Errorf("count mismatch: count=%d but agreements length=%d", resp.Count, len(resp.BorrowingAgreements))
        }
        if resp.Disclaimer == "" {
                t.Error("expected disclaimer")
        }
        // Each agreement must have a source URL (evidence-first).
        for i, a := range resp.BorrowingAgreements {
                if _, ok := a["source_url"]; !ok {
                        t.Errorf("agreement %d missing source_url", i)
                }
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
