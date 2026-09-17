// Package main — public debt & borrowing intelligence API endpoints.
// Spec: Public Debt & Borrowing Intelligence, sections 21-22.
//
// Endpoints:
//
//      GET /api/v1/debt                    -- national debt dashboard
//      GET /api/v1/debt/loans              -- borrowing register
//      GET /api/v1/debt/creditors          -- creditor intelligence
//      GET /api/v1/debt/timeline           -- debt stock timeline
//      GET /api/v1/debt/governments/{id}    -- government debt summary
//      GET /api/v1/debt/legislatures/{id}  -- legislature debt view (placeholder)
package main

import (
        "net/http"
        "strings"
)

// DebtDashboardResponse is the response for GET /api/v1/debt.
type DebtDashboardResponse struct {
        CountryCode     string  `json:"country_code"`
        TotalDebtStock  *float64 `json:"total_debt_stock"`
        DomesticDebt     *float64 `json:"domestic_debt"`
        ExternalDebt    *float64 `json:"external_debt"`
        DebtService     *float64 `json:"debt_service"`
        DebtToGDP       *float64 `json:"debt_to_gdp"`
        Currency        string  `json:"currency"`
        AsOf            string  `json:"as_of"`
        SourceURL       string  `json:"source_url"`
        Disclaimer      string  `json:"disclaimer"`
}

// DebtTrendPointResponse is a single point in the debt trend graph.
type DebtTrendPointResponse struct {
        Date           string  `json:"date"`
        TotalDebtStock *float64 `json:"total_debt_stock"`
        DomesticDebt   *float64 `json:"domestic_debt"`
        ExternalDebt   *float64 `json:"external_debt"`
        SourceURL      string  `json:"source_url"`
}

// DebtTimelineResponse is the response for GET /api/v1/debt/timeline.
type DebtTimelineResponse struct {
        CountryCode string                    `json:"country_code"`
        Points      []DebtTrendPointResponse `json:"points"`
        Disclaimer  string                    `json:"disclaimer"`
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

// sampleDebtTrend is sample debt stock observations sourced from CBK and
// Treasury. Spec section 9: "Populate the production chart from validated
// Treasury/CBK observations."
//
// Source: CBK Monthly Economic Indicators, Treasury Annual Public Debt Report.
var sampleDebtTrend = []DebtTrendPointResponse{
        {Date: "2013-06-30", TotalDebtStock: ptrFloat(1.96e12), DomesticDebt: ptrFloat(0.83e12), ExternalDebt: ptrFloat(1.13e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2014-06-30", TotalDebtStock: ptrFloat(2.38e12), DomesticDebt: ptrFloat(1.06e12), ExternalDebt: ptrFloat(1.32e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2015-06-30", TotalDebtStock: ptrFloat(2.84e12), DomesticDebt: ptrFloat(1.27e12), ExternalDebt: ptrFloat(1.57e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2016-06-30", TotalDebtStock: ptrFloat(3.28e12), DomesticDebt: ptrFloat(1.44e12), ExternalDebt: ptrFloat(1.84e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2017-06-30", TotalDebtStock: ptrFloat(3.85e12), DomesticDebt: ptrFloat(1.65e12), ExternalDebt: ptrFloat(2.20e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2018-06-30", TotalDebtStock: ptrFloat(4.49e12), DomesticDebt: ptrFloat(1.95e12), ExternalDebt: ptrFloat(2.54e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2019-06-30", TotalDebtStock: ptrFloat(5.07e12), DomesticDebt: ptrFloat(2.20e12), ExternalDebt: ptrFloat(2.87e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2020-06-30", TotalDebtStock: ptrFloat(5.94e12), DomesticDebt: ptrFloat(2.59e12), ExternalDebt: ptrFloat(3.35e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2021-06-30", TotalDebtStock: ptrFloat(6.92e12), DomesticDebt: ptrFloat(3.06e12), ExternalDebt: ptrFloat(3.86e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2022-06-30", TotalDebtStock: ptrFloat(7.71e12), DomesticDebt: ptrFloat(3.42e12), ExternalDebt: ptrFloat(4.29e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2023-06-30", TotalDebtStock: ptrFloat(9.18e12), DomesticDebt: ptrFloat(3.96e12), ExternalDebt: ptrFloat(5.22e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
        {Date: "2024-06-30", TotalDebtStock: ptrFloat(10.59e12), DomesticDebt: ptrFloat(4.59e12), ExternalDebt: ptrFloat(6.00e12), SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"},
}

// sampleGovernmentDebtSummaries maps administration IDs to debt summaries.
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT say
// "President X borrowed KSh X". It says "The Government of Kenya recorded
// KSh X in borrowing during this period."
var sampleGovernmentDebtSummaries = map[string]GovernmentDebtSummaryResponse{
        "admin-uhuru-kenyatta": {
                AdministrationID: "admin-uhuru-kenyatta",
                Period:           "2013-2022",
                DebtAtStart:      ptrFloat(1.96e12),
                DebtAtEnd:        ptrFloat(7.71e12),
                NewBorrowing:     ptrFloat(5.75e12),
                DebtService:      ptrFloat(2.81e12),
                ExternalDebt:     ptrFloat(4.29e12),
                DomesticDebt:     ptrFloat(3.42e12),
                Currency:         "KES",
                SourceURLs: []string{
                        "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/",
                        "https://www.treasury.go.ke/wp-content/uploads/2023/05/Annual-Public-Debt-Report-2022-23.pdf",
                },
                Disclaimer: `The Government of Kenya recorded KSh 5.75 trillion in new borrowing
during the 2013-2022 period. This is NOT "President Uhuru Kenyatta borrowed
KSh 5.75T" — the legal borrower is the Republic of Kenya. The platform does
not calculate debt performance scores or rank governments.`,
        },
        "admin-william-ruto": {
                AdministrationID: "admin-william-ruto",
                Period:           "2022-Present",
                DebtAtStart:      ptrFloat(7.71e12),
                DebtAtEnd:        ptrFloat(10.59e12),
                NewBorrowing:     ptrFloat(2.88e12),
                DebtService:      ptrFloat(1.36e12),
                ExternalDebt:     ptrFloat(6.00e12),
                DomesticDebt:     ptrFloat(4.59e12),
                Currency:         "KES",
                SourceURLs: []string{
                        "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/",
                        "https://www.treasury.go.ke/wp-content/uploads/2024/05/Annual-Public-Debt-Report-2023-24.pdf",
                },
                Disclaimer: `The Government of Kenya recorded KSh 2.88 trillion in new borrowing
during the 2022-present period. This is NOT "President William Ruto borrowed
KSh 2.88T" — the legal borrower is the Republic of Kenya. The platform does
not calculate debt performance scores or rank governments.`,
        },
}

// makeDebtDashboardHandler handles GET /api/v1/debt.
func makeDebtDashboardHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                // Most recent snapshot.
                latest := sampleDebtTrend[len(sampleDebtTrend)-1]
                resp := DebtDashboardResponse{
                        CountryCode:    "KE",
                        TotalDebtStock: latest.TotalDebtStock,
                        DomesticDebt:    latest.DomesticDebt,
                        ExternalDebt:   latest.ExternalDebt,
                        DebtService:    ptrFloat(1.36e12), // FY 2023/24 debt service
                        DebtToGDP:      ptrFloat(70.2),   // ~70.2% as of FY 2023/24
                        Currency:       "KES",
                        AsOf:           latest.Date,
                        SourceURL:      latest.SourceURL,
                        Disclaimer: `Public debt figures are sourced from CBK Monthly Economic Indicators
and Treasury Annual Public Debt Reports. The platform does not calculate
"best borrower" or any political performance score. Every figure is an
immutable observation; changes in debt stock can reflect exchange-rate
movements, valuation changes, repayments, refinancing, arrears, adjustments,
and disbursement timing — NOT just new borrowing.`,
                }
                writeJSON(w, http.StatusOK, resp)
        }
}

// makeDebtTimelineHandler handles GET /api/v1/debt/timeline.
func makeDebtTimelineHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                writeJSON(w, http.StatusOK, DebtTimelineResponse{
                        CountryCode: "KE",
                        Points:      sampleDebtTrend,
                        Disclaimer: `Each point is an immutable observation. The trend line should NOT be
interpreted as a causal attribution of borrowing to any individual president.
Government transition dates overlay the chart; users can correlate visually
without the platform implying causation.`,
                })
        }
}

// makeDebtLoansHandler handles GET /api/v1/debt/loans.
// Returns the borrowing register. Until the fiscal service is wired up,
// returns an empty list with a clear disclaimer.
func makeDebtLoansHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                writeJSON(w, http.StatusOK, map[string]any{
                        "borrowing_agreements": []any{},
                        "count":                 0,
                        "disclaimer": `Each borrowing agreement is sourced from authoritative material
(Treasury External Public Debt Register, IMF, World Bank, AfDB press releases).
The platform distinguishes CONTRACTED_DURING, DISBURSED_DURING, REPAID_DURING,
OUTSTANDING_DURING, and REFINANCED_DURING to prevent misleading historical
attribution. Loan-level ingestion is pending (issue #93).`,
                })
        }
}

// makeGovernmentDebtHandler handles GET /api/v1/debt/governments/{id}.
func makeGovernmentDebtHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                id := strings.TrimPrefix(r.URL.Path, "/api/v1/debt/governments/")
                id = strings.TrimSuffix(id, "/")
                summary, ok := sampleGovernmentDebtSummaries[id]
                if !ok {
                        writeError(w, http.StatusNotFound, "not_found", "no debt summary for administration: "+id)
                        return
                }
                writeJSON(w, http.StatusOK, summary)
        }
}

// makeDebtRouter routes /api/v1/debt/* sub-resources.
func makeDebtRouter() http.HandlerFunc {
        dash := makeDebtDashboardHandler()
        timeline := makeDebtTimelineHandler()
        loans := makeDebtLoansHandler()
        gov := makeGovernmentDebtHandler()
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

func ptrFloat(f float64) *float64 { return &f }
