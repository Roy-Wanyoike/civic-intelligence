// Package main — public debt & borrowing intelligence API endpoints.
// Spec: Public Debt & Borrowing Intelligence, sections 21-22.
//
// Endpoints (routed by makeDebtRouter via /api/v1/debt and /api/v1/debt/):
//
//	GET /api/v1/debt                    -- national debt dashboard
//	GET /api/v1/debt/loans              -- borrowing register
//	GET /api/v1/debt/timeline           -- debt stock timeline
//	GET /api/v1/debt/governments/{id}   -- government debt summary
//
// The following endpoints are NOT routed, even though earlier versions of this
// file documented them:
//
//	GET /api/v1/debt/creditors          -- not implemented
//	GET /api/v1/debt/legislatures/{id}  -- not implemented (placeholder)
//
// Issue #203: handlers consume a legislation.DebtRepository (constructed by
// legislation.WireDebtRepository and seeded by kenya_seed.SeedDebt). The
// hardcoded sampleDebtTrend + sampleGovernmentDebtSummaries that previously
// lived in this file have been moved to adapters/kenya/kenya_seed.
package main

import (
        "net/http"
        "strings"

        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// DebtDashboardResponse is the response for GET /api/v1/debt.
type DebtDashboardResponse struct {
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

// DebtTrendPointResponse is a single point in the debt trend graph.
type DebtTrendPointResponse struct {
        Date           string   `json:"date"`
        TotalDebtStock *float64 `json:"total_debt_stock"`
        DomesticDebt   *float64 `json:"domestic_debt"`
        ExternalDebt   *float64 `json:"external_debt"`
        SourceURL      string   `json:"source_url"`
}

// DebtTimelineResponse is the response for GET /api/v1/debt/timeline.
type DebtTimelineResponse struct {
        CountryCode string                   `json:"country_code"`
        Points      []DebtTrendPointResponse `json:"points"`
        Disclaimer  string                   `json:"disclaimer"`
}

// GovernmentDebtSummaryResponse is the response for GET /api/v1/debt/governments/{id}.
type GovernmentDebtSummaryResponse struct {
        AdministrationID string   `json:"administration_id"`
        Period           string   `json:"period"`
        DebtAtStart      *float64 `json:"debt_at_start"`
        DebtAtEnd        *float64 `json:"debt_at_end"`
        NewBorrowing     *float64 `json:"new_borrowing"`
        DebtService      *float64 `json:"debt_service"`
        ExternalDebt     *float64 `json:"external_debt"`
        DomesticDebt     *float64 `json:"domestic_debt"`
        Currency         string   `json:"currency"`
        SourceURLs       []string `json:"source_urls"`
        Disclaimer       string   `json:"disclaimer"`
}

// debtDisclaimerDashboard is the canonical disclaimer attached to the
// /api/v1/debt dashboard. Spec section 35, 37.
const debtDisclaimerDashboard = `Public debt figures are sourced from CBK Monthly Economic Indicators
and Treasury Annual Public Debt Reports. The platform does not calculate
"best borrower" or any political performance score. Every figure is an
immutable observation; changes in debt stock can reflect exchange-rate
movements, valuation changes, repayments, refinancing, arrears, adjustments,
and disbursement timing — NOT just new borrowing.`

// debtDisclaimerTimeline is attached to /api/v1/debt/timeline. Spec
// section 4 — the trend line is a series of immutable observations, not
// a causal attribution of borrowing to any individual president.
const debtDisclaimerTimeline = `Each point is an immutable observation. The trend line should NOT be
interpreted as a causal attribution of borrowing to any individual president.
Government transition dates overlay the chart; users can correlate visually
without the platform implying causation.`

// makeDebtDashboardHandler handles GET /api/v1/debt.
//
// The dashboard surfaces the most recent CBK snapshot for Kenya, plus
// dashboard-level derived figures (debt service + debt-to-GDP ratio)
// sourced from the Treasury Annual Public Debt Report. The derived
// figures are returned by kenya_seed.DebtDashboardFigures(); they are
// NOT derived from snapshot deltas (Spec section 9 — never infer new
// borrowing from end - start).
func makeDebtDashboardHandler(repo legislation.DebtRepository) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")

                // Pull all available Kenya snapshots and take the latest. ListDebtSnapshots
                // returns them in chronological order, so the last element is the most
                // recent observation. We deliberately do NOT compute (end - start) deltas
                // here — Spec section 9 forbids inferring new borrowing from snapshots.
                from := farPastDate()
                to := farFutureDate()
                snaps, err := repo.ListDebtSnapshots(r.Context(), "KE", from, to)
                if err != nil || len(snaps) == 0 {
                        writeError(w, http.StatusServiceUnavailable, "debt_unavailable",
                                "no debt snapshots available; the CBK seed data may not have loaded")
                        return
                }
                latest := snaps[len(snaps)-1]
                debtService, debtToGDP := kenyaDebtDashboardFigures()
                resp := DebtDashboardResponse{
                        CountryCode:    latest.CountryCode,
                        TotalDebtStock: ptrFloat(latest.TotalDebtStock),
                        DomesticDebt:   ptrFloat(latest.DomesticDebt),
                        ExternalDebt:   ptrFloat(latest.ExternalDebt),
                        DebtService:    ptrFloat(debtService),
                        DebtToGDP:      ptrFloat(debtToGDP),
                        Currency:       latest.Currency,
                        AsOf:           latest.ObservationDate.UTC().Format("2006-01-02"),
                        SourceURL:      latest.SourceURL,
                        Disclaimer:     debtDisclaimerDashboard,
                }
                writeJSON(w, http.StatusOK, resp)
        }
}

// makeDebtTimelineHandler handles GET /api/v1/debt/timeline.
//
// The timeline returns every CBK observation for Kenya in chronological
// order. Each point is an immutable observation (Spec section 9).
func makeDebtTimelineHandler(repo legislation.DebtRepository) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                snaps, err := repo.ListDebtSnapshots(r.Context(), "KE", farPastDate(), farFutureDate())
                if err != nil {
                        writeError(w, http.StatusServiceUnavailable, "debt_unavailable",
                                "debt snapshots could not be loaded")
                        return
                }
                points := make([]DebtTrendPointResponse, 0, len(snaps))
                for _, s := range snaps {
                        points = append(points, snapshotToTrendPoint(s))
                }
                writeJSON(w, http.StatusOK, DebtTimelineResponse{
                        CountryCode: "KE",
                        Points:      points,
                        Disclaimer:  debtDisclaimerTimeline,
                })
        }
}

// makeDebtLoansHandler handles GET /api/v1/debt/loans.
// Returns the borrowing register. Until the fiscal service is wired up,
// returns an empty list with a clear disclaimer.
//
// Issue #203: the handler now consults the repository via
// ListBorrowingAgreements. If the seeder has not populated any agreements
// (which is the current state — loan-level ingestion is pending issue
// #93), the response is still well-formed: count=0, agreements=[].
func makeDebtLoansHandler(repo legislation.DebtRepository) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                agreements, err := repo.ListBorrowingAgreements(r.Context(), legislation.DebtFilter{
                        CountryCode: "KE",
                })
                if err != nil {
                        writeError(w, http.StatusServiceUnavailable, "debt_unavailable",
                                "borrowing agreements could not be loaded")
                        return
                }
                writeJSON(w, http.StatusOK, map[string]any{
                        "borrowing_agreements": agreements,
                        "count":                len(agreements),
                        "disclaimer": `Each borrowing agreement is sourced from authoritative material
(Treasury External Public Debt Register, IMF, World Bank, AfDB press releases).
The platform distinguishes CONTRACTED_DURING, DISBURSED_DURING, REPAID_DURING,
OUTSTANDING_DURING, and REFINANCED_DURING to prevent misleading historical
attribution. Loan-level ingestion is pending (issue #93).`,
                })
        }
}

// makeGovernmentDebtHandler handles GET /api/v1/debt/governments/{id}.
//
// The handler consults the repository for the cached summary. If no
// summary exists for the requested administration, the response is 404
// with a clear error — the platform does not fabricate summaries.
func makeGovernmentDebtHandler(repo legislation.DebtRepository) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                id := strings.TrimPrefix(r.URL.Path, "/api/v1/debt/governments/")
                id = strings.TrimSuffix(id, "/")
                summary, err := repo.GetGovernmentDebtSummary(r.Context(), legislation.ID(id))
                if err != nil {
                        writeError(w, http.StatusNotFound, "not_found", "no debt summary for administration: "+id)
                        return
                }
                writeJSON(w, http.StatusOK, summaryFromDomain(*summary))
        }
}

// makeDebtRouter routes /api/v1/debt/* sub-resources. The repository is
// threaded through every handler so they all read from the same in-memory
// store (or, in production, the same Postgres-backed store).
func makeDebtRouter(repo legislation.DebtRepository) http.HandlerFunc {
        dash := makeDebtDashboardHandler(repo)
        timeline := makeDebtTimelineHandler(repo)
        loans := makeDebtLoansHandler(repo)
        gov := makeGovernmentDebtHandler(repo)
        return func(w http.ResponseWriter, r *http.Request) {
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/debt")
                path = strings.TrimPrefix(path, "/")
                switch {
                case path == "" || path == "/":
                        dash(w, r)
                case strings.HasPrefix(path, "timeline"):
                        timeline(w, r)
                case strings.HasPrefix(path, "loans"):
                        loans(w, r)
                case strings.HasPrefix(path, "governments/"):
                        gov(w, r)
                default:
                        writeError(w, http.StatusNotFound, "not_found", "unknown debt sub-resource: "+path)
                }
        }
}

// snapshotToTrendPoint converts a domain.PublicDebtSnapshot to the JSON
// response shape. The conversion is lossless — the response type is a
// strict subset of the domain type.
func snapshotToTrendPoint(s legislation.PublicDebtSnapshot) DebtTrendPointResponse {
        return DebtTrendPointResponse{
                Date:           s.ObservationDate.UTC().Format("2006-01-02"),
                TotalDebtStock: ptrFloat(s.TotalDebtStock),
                DomesticDebt:   ptrFloat(s.DomesticDebt),
                ExternalDebt:   ptrFloat(s.ExternalDebt),
                SourceURL:      s.SourceURL,
        }
}

// summaryFromDomain converts a domain.GovernmentDebtSummary to the JSON
// response shape.
func summaryFromDomain(s legislation.GovernmentDebtSummary) GovernmentDebtSummaryResponse {
        return GovernmentDebtSummaryResponse{
                AdministrationID: string(s.AdministrationID),
                Period:           s.Period,
                DebtAtStart:      s.DebtAtStart,
                DebtAtEnd:        s.DebtAtEnd,
                NewBorrowing:     s.NewBorrowing,
                DebtService:      s.DebtService,
                ExternalDebt:     s.ExternalDebt,
                DomesticDebt:     s.DomesticDebt,
                Currency:         s.Currency,
                SourceURLs:       s.SourceURLs,
                Disclaimer:       s.Disclaimer,
        }
}

func ptrFloat(f float64) *float64 { return &f }
