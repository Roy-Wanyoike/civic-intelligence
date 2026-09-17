package main

import (
        "context"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"
        "time"

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

// TestDebtLoans_ReturnsSeededAgreements verifies that the loans endpoint
// returns seeded borrowing agreements (issue #220). Previously the handler
// returned an empty list because no BorrowingAgreement records had been
// seeded; the Kenya seed now populates 10 sample agreements sourced from
// public press releases (IMF, World Bank, AfDB, China Exim Bank, Eurobonds).
func TestDebtLoans_ReturnsSeededAgreements(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/debt/loans", nil)
        rec := httptest.NewRecorder()

        makeDebtRouter(newTestDebtRepo(t))(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Count               int                          `json:"count"`
                BorrowingAgreements []BorrowingAgreementResponse `json:"borrowing_agreements"`
                Disclaimer          string                       `json:"disclaimer"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if resp.Count == 0 {
                t.Errorf("expected non-zero count; got %d (issue #220 regression: empty list)", resp.Count)
        }
        if len(resp.BorrowingAgreements) != resp.Count {
                t.Errorf("expected agreements slice length to match count; got %d agreements, count=%d",
                        len(resp.BorrowingAgreements), resp.Count)
        }
        if resp.Disclaimer == "" {
                t.Error("expected disclaimer")
        }
        if !containsStr(resp.Disclaimer, "CONTRACTED_DURING") {
                t.Error("expected disclaimer to mention CONTRACTED_DURING attribution axis")
        }

        // Verify each agreement has the required attribution fields populated.
        var sawUhuruAdmin, sawRutoAdmin bool
        for _, a := range resp.BorrowingAgreements {
                if a.ID == "" {
                        t.Error("expected agreement to have an ID")
                }
                if a.GovernmentAdministrationID == "" {
                        t.Errorf("agreement %s: expected government_administration_id; got empty", a.ID)
                }
                if a.Borrower != "Republic of Kenya" {
                        t.Errorf("agreement %s: expected borrower 'Republic of Kenya'; got %q", a.ID, a.Borrower)
                }
                if a.ContractDate == nil {
                        t.Errorf("agreement %s: expected contract_date; got nil", a.ID)
                }
                if a.Status == "" {
                        t.Errorf("agreement %s: expected status; got empty", a.ID)
                }
                if a.SourceURL == "" {
                        t.Errorf("agreement %s: expected source_url; got empty", a.ID)
                }
                // Seed agreements are all correctly attributed (ContractDate falls
                // within the attributed administration's window); issue #221's
                // attribution_warning must be absent on every seeded item.
                if a.AttributionWarning != "" {
                        t.Errorf("agreement %s: expected no attribution_warning on correctly-attributed seeded record; got %q",
                                a.ID, a.AttributionWarning)
                }
                switch a.GovernmentAdministrationID {
                case "admin-uhuru-kenyatta":
                        sawUhuruAdmin = true
                case "admin-william-ruto":
                        sawRutoAdmin = true
                }
        }
        if !sawUhuruAdmin {
                t.Error("expected at least one Uhuru-administration agreement")
        }
        if !sawRutoAdmin {
                t.Error("expected at least one Ruto-administration agreement")
        }
}

// TestDebtLoans_AttributionValidated verifies that the loans handler runs
// each agreement through legislation.ValidateAttribution and attaches an
// attribution_warning when the agreement's GovernmentAdministrationID does
// NOT correspond to the administration in power on ContractDate (issue #221).
//
// The test seeds a misattributed agreement — contracted in May 2014 (Uhuru's
// administration, 2013-04-09 .. 2022-09-13) but attributed to admin-william-ruto
// (whose term began 2022-09-13). The handler MUST return the record with an
// attribution_warning that names the correct administration; it must NOT
// hide the record. The platform reports uncertainty; it never silently
// discards data (Spec section 28).
func TestDebtLoans_AttributionValidated(t *testing.T) {
        repo := newTestDebtRepo(t)
        // Construct a deliberately misattributed agreement. ContractDate in
        // 2014 falls within admin-uhuru-kenyatta's window, but the agreement is
        // attributed to admin-william-ruto.
        contractDate := time.Date(2014, 5, 11, 0, 0, 0, 0, time.UTC)
        termID := legislation.ID("term-ruto-1")
        misattributed := legislation.BorrowingAgreement{
                ID:                          "loan-ke-test-misattributed",
                CountryCode:                 "KE",
                GovernmentAdministrationID:  "admin-william-ruto",
                PresidentialTermID:          &termID,
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-test",
                CreditorName:                "Test Creditor",
                CreditorCategory:            legislation.CreditorMultilateral,
                InstrumentType:              "Sovereign Loan",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              100_000_000,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeInfrastructure),
                ContractDate:                &contractDate,
                Status:                      "DISBURSED",
                SourceURL:                   "https://example.com/test",
        }
        if err := repo.CreateBorrowingAgreement(context.Background(), misattributed); err != nil {
                t.Fatalf("create misattributed agreement: %v", err)
        }

        req := httptest.NewRequest(http.MethodGet, "/api/v1/debt/loans", nil)
        rec := httptest.NewRecorder()
        makeDebtRouter(repo)(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Count               int                          `json:"count"`
                BorrowingAgreements []BorrowingAgreementResponse `json:"borrowing_agreements"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if resp.Count == 0 {
                t.Fatal("expected agreements; got 0")
        }

        var foundMisattributed bool
        var correctlyAttributedCount int
        for _, a := range resp.BorrowingAgreements {
                if a.ID == "loan-ke-test-misattributed" {
                        foundMisattributed = true
                        if a.AttributionWarning == "" {
                                t.Error("expected attribution_warning for misattributed agreement; got empty")
                        }
                        // The warning message produced by ValidateAttribution names the
                        // administration that SHOULD have been attributed. Verify the
                        // message points back to admin-uhuru-kenyatta (the in-power
                        // administration on 2014-05-11).
                        if !containsStr(a.AttributionWarning, "admin-uhuru-kenyatta") {
                                t.Errorf("expected attribution_warning to mention admin-uhuru-kenyatta; got %q",
                                        a.AttributionWarning)
                        }
                        continue
                }
                // Seeded agreements are correctly attributed; no warning expected.
                if a.AttributionWarning != "" {
                        t.Errorf("agreement %s: expected no attribution_warning on correctly-attributed record; got %q",
                                a.ID, a.AttributionWarning)
                }
                correctlyAttributedCount++
        }
        if !foundMisattributed {
                t.Error("expected to find misattributed agreement in response (issue #221: platform must not hide misattributed records)")
        }
        if correctlyAttributedCount == 0 {
                t.Error("expected at least one correctly-attributed agreement without warning")
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
