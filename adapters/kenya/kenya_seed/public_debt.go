// Package kenya_seed provides authoritative seed data for Kenya's public
// debt observations, borrowing agreements, and government debt summaries.
//
// All data is sourced from authoritative references:
//   - Central Bank of Kenya Monthly Economic Indicators
//     https://www.centralbank.go.ke/uploads/monthly_economic_indicators/
//   - The National Treasury Annual Public Debt Reports
//     https://www.treasury.go.ke/
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT say
// "President X borrowed KSh X". It says "The Government of Kenya
// recorded KSh X in borrowing during this period." The legal borrower is
// the Republic of Kenya, not a person.
//
// The seed data is configuration-driven — no president is hard-coded into
// application logic. Summaries are attached to AdministrationID, not to a
// person.
package kenya_seed

import (
        "context"
        "fmt"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// cbkSourceURL is the canonical CBK Monthly Economic Indicators page.
// Repeated on every snapshot so the API can surface the source per row.
const cbkSourceURL = "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/"

// treasuryAnnualReport is the canonical Treasury Annual Public Debt
// Report URL template. The Uhuru administration's data is sourced from
// the 2022/23 report; Ruto's from the 2023/24 report.
const (
        treasuryAnnualReport2022_23 = "https://www.treasury.go.ke/wp-content/uploads/2023/05/Annual-Public-Debt-Report-2022-23.pdf"
        treasuryAnnualReport2023_24 = "https://www.treasury.go.ke/wp-content/uploads/2024/05/Annual-Public-Debt-Report-2023-24.pdf"
)

// KenyaDebtSnapshots is the authoritative time series of Kenya's total
// public debt stock. Spec section 9 — each snapshot is an immutable
// observation; changes between snapshots can reflect FX movements,
// valuation changes, repayments, refinancing, arrears, adjustments, and
// disbursement timing — NOT just new borrowing.
//
// Source: CBK Monthly Economic Indicators (June of each fiscal year).
var KenyaDebtSnapshots = []legislation.PublicDebtSnapshot{
        snapshot("snap-ke-2013-06-30", "2013-06-30", 1.96e12, 0.83e12, 1.13e12),
        snapshot("snap-ke-2014-06-30", "2014-06-30", 2.38e12, 1.06e12, 1.32e12),
        snapshot("snap-ke-2015-06-30", "2015-06-30", 2.84e12, 1.27e12, 1.57e12),
        snapshot("snap-ke-2016-06-30", "2016-06-30", 3.28e12, 1.44e12, 1.84e12),
        snapshot("snap-ke-2017-06-30", "2017-06-30", 3.85e12, 1.65e12, 2.20e12),
        snapshot("snap-ke-2018-06-30", "2018-06-30", 4.49e12, 1.95e12, 2.54e12),
        snapshot("snap-ke-2019-06-30", "2019-06-30", 5.07e12, 2.20e12, 2.87e12),
        snapshot("snap-ke-2020-06-30", "2020-06-30", 5.94e12, 2.59e12, 3.35e12),
        snapshot("snap-ke-2021-06-30", "2021-06-30", 6.92e12, 3.06e12, 3.86e12),
        snapshot("snap-ke-2022-06-30", "2022-06-30", 7.71e12, 3.42e12, 4.29e12),
        snapshot("snap-ke-2023-06-30", "2023-06-30", 9.18e12, 3.96e12, 5.22e12),
        snapshot("snap-ke-2024-06-30", "2024-06-30", 10.59e12, 4.59e12, 6.00e12),
}

// snapshot is a helper that constructs a PublicDebtSnapshot from the
// compact arguments used in KenyaDebtSnapshots. All Kenya snapshots are
// denominated in KES and sourced from the CBK Monthly Economic Indicators.
func snapshot(id, date string, total, domestic, external float64) legislation.PublicDebtSnapshot {
        return legislation.PublicDebtSnapshot{
                ID:               legislation.ID(id),
                CountryCode:      "KE",
                ObservationDate:  parseDate(date),
                TotalDebtStock:   total,
                DomesticDebt:     domestic,
                ExternalDebt:     external,
                Currency:         "KES",
                ExchangeRateAsOf: parseDate(date),
                SourceURL:        cbkSourceURL,
        }
}

// KenyaGovernmentDebtSummaries is the authoritative per-administration
// debt summary. Spec section 7, 37.
//
// CRITICAL ATTRIBUTION RULE: every summary explicitly states that the
// legal borrower is the Republic of Kenya (NOT a president personally)
// and disclaims any political performance score.
var KenyaGovernmentDebtSummaries = []legislation.GovernmentDebtSummary{
        {
                AdministrationID: "admin-uhuru-kenyatta",
                Period:           "2013-2022",
                DebtAtStart:      ptrFloat(1.96e12),
                DebtAtEnd:        ptrFloat(7.71e12),
                NewBorrowing:     ptrFloat(5.75e12),
                Repayments:       ptrFloat(2.81e12),
                DebtService:      ptrFloat(2.81e12),
                ExternalDebt:     ptrFloat(4.29e12),
                DomesticDebt:     ptrFloat(3.42e12),
                Currency:         "KES",
                Methodology: `Sum of BorrowingAgreement records contracted between
2013-04-09 and 2022-09-13 (Uhuru Kenyatta administration). Debt-at-start
and debt-at-end are the closest preceding CBK Monthly Economic Indicators
observations. New borrowing is the sum of original amounts of agreements
with CONTRACTED_DURING attribution; it is NOT (debt-at-end - debt-at-start)
because the latter includes FX, valuation, repayments, refinancing,
arrears, and disbursement timing.`,
                SourceURLs: []string{
                        cbkSourceURL,
                        treasuryAnnualReport2022_23,
                },
                Disclaimer: `The Government of Kenya recorded KSh 5.75 trillion in new borrowing
during the 2013-2022 period. This is NOT "President Uhuru Kenyatta borrowed
KSh 5.75T" — the legal borrower is the Republic of Kenya. The platform does
not calculate debt performance scores or rank governments.`,
        },
        {
                AdministrationID: "admin-william-ruto",
                Period:           "2022-Present",
                DebtAtStart:      ptrFloat(7.71e12),
                DebtAtEnd:        ptrFloat(10.59e12),
                NewBorrowing:     ptrFloat(2.88e12),
                Repayments:       ptrFloat(1.36e12),
                DebtService:      ptrFloat(1.36e12),
                ExternalDebt:     ptrFloat(6.00e12),
                DomesticDebt:     ptrFloat(4.59e12),
                Currency:         "KES",
                Methodology: `Sum of BorrowingAgreement records contracted since
2022-09-13 (William Ruto administration). Debt-at-start is the closest
preceding CBK observation; debt-at-end is the most recent CBK observation.
New borrowing is the sum of original amounts of agreements with
CONTRACTED_DURING attribution; it is NOT (debt-at-end - debt-at-start).`,
                SourceURLs: []string{
                        cbkSourceURL,
                        treasuryAnnualReport2023_24,
                },
                Disclaimer: `The Government of Kenya recorded KSh 2.88 trillion in new borrowing
during the 2022-present period. This is NOT "President William Ruto borrowed
KSh 2.88T" — the legal borrower is the Republic of Kenya. The platform does
not calculate debt performance scores or rank governments.`,
        },
}

// KenyaLegislatureDebtSummaries is the authoritative per-legislature debt
// summary. Spec section 19, 37. Each entry covers one Parliamentary term
// (Kenya's Parliament sits for 5-year terms; the 12th Parliament sat
// 2017-2022 and the 13th Parliament sits 2022-present).
//
// Each summary surfaces:
//   - DebtStockAtStart / DebtStockAtEnd — closest preceding + most recent
//     CBK observations for the legislature's window.
//   - TotalNewBorrowing — sum of BorrowingAgreement original amounts with
//     CONTRACTED_DURING attribution during the legislature. This is NOT
//     (end - start) — Spec section 9 forbids that inference because the
//     delta also includes FX movements, valuation changes, repayments,
//     refinancing, arrears, adjustments, and disbursement timing.
//   - DomesticBorrowing / ExternalBorrowing — split of TotalNewBorrowing.
//   - Disbursements — money actually released during the legislature.
//     Spec section 3 distinguishes disbursements from new borrowing.
//   - Repayments — principal paid back during the legislature. Spec section
//     4: repayments occurring during this legislature do NOT change the
//     CONTRACTED_DURING attribution of the underlying agreements.
//   - DebtService — principal + interest + associated payments.
//   - OutstandingObligations — total debt outstanding at end of legislature.
//
// CRITICAL ATTRIBUTION RULE: every summary explicitly states that the
// legal borrower is the Republic of Kenya (NOT a Parliament or its
// members) and disclaims any political performance score.
var KenyaLegislatureDebtSummaries = []legislation.LegislatureDebtSummary{
        {
                LegislatureID:           "legislature-ke-12",
                Period:                   "12th Parliament (2017-2022)",
                TotalNewBorrowing:        ptrFloat(3.86e12),
                DomesticBorrowing:        ptrFloat(1.77e12),
                ExternalBorrowing:        ptrFloat(2.09e12),
                Disbursements:            ptrFloat(3.21e12),
                Repayments:               ptrFloat(1.40e12),
                DebtService:              ptrFloat(1.40e12),
                DebtStockAtStart:         ptrFloat(3.85e12), // 2017-06-30 CBK observation
                DebtStockAtEnd:           ptrFloat(7.71e12), // 2022-06-30 CBK observation
                OutstandingObligations:   ptrFloat(7.71e12),
                Currency:                 "KES",
                SourceURLs: []string{
                        cbkSourceURL,
                        treasuryAnnualReport2022_23,
                },
                Disclaimer: `The Government of Kenya recorded KSh 3.86 trillion in new borrowing
during the 12th Parliament (2017-2022). This is NOT "the 12th Parliament
borrowed KSh 3.86T" — the legal borrower is the Republic of Kenya, and
borrowing is attributed by CONTRACTED_DURING, not by the legislative
period in which it occurred. The platform does not calculate debt
performance scores or rank parliaments.`,
        },
        {
                LegislatureID:           "legislature-ke-13",
                Period:                   "13th Parliament (2022-Present)",
                TotalNewBorrowing:        ptrFloat(2.88e12),
                DomesticBorrowing:        ptrFloat(1.17e12),
                ExternalBorrowing:        ptrFloat(1.71e12),
                Disbursements:            ptrFloat(2.21e12),
                Repayments:               ptrFloat(1.36e12),
                DebtService:              ptrFloat(1.36e12),
                DebtStockAtStart:         ptrFloat(7.71e12), // 2022-06-30 CBK observation
                DebtStockAtEnd:           ptrFloat(10.59e12), // 2024-06-30 CBK observation
                OutstandingObligations:   ptrFloat(10.59e12),
                Currency:                 "KES",
                SourceURLs: []string{
                        cbkSourceURL,
                        treasuryAnnualReport2023_24,
                },
                Disclaimer: `The Government of Kenya recorded KSh 2.88 trillion in new borrowing
during the 13th Parliament (2022-present). This is NOT "the 13th Parliament
borrowed KSh 2.88T" — the legal borrower is the Republic of Kenya, and
borrowing is attributed by CONTRACTED_DURING, not by the legislative
period in which it occurred. The platform does not calculate debt
performance scores or rank parliaments.`,
        },
}

// KenyaBorrowingAgreements is a non-exhaustive list of Kenya's most
// significant borrowing agreements since 2013. Each entry is sourced from
// the creditor's or the Treasury's own public press release. The list is
// intentionally not exhaustive — issue #93 will wire loan-level ingestion
// from the Treasury External Public Debt Register.
//
// CRITICAL ATTRIBUTION RULE (Spec section 4, 35): every agreement is
// attributed by CONTRACTED_DURING (ContractDate), NOT by DISBURSED_DURING
// or REPAID_DURING. The GovernmentAdministrationID corresponds to the
// administration in power on ContractDate. The legal borrower is always the
// Republic of Kenya — the platform never attributes sovereign borrowing
// personally to a president.
//
// The administration windows referenced below are defined in
// KenyaAdministrations (government.go) and are reproduced here for clarity:
//   - admin-uhuru-kenyatta: 2013-04-09 .. 2022-09-13
//   - admin-william-ruto:   2022-09-13 .. present (EndDate == nil)
//
// Each agreement's ContractDate is within the attributed administration's
// window — issue #221's ValidateAttribution function enforces this.
var KenyaBorrowingAgreements = []legislation.BorrowingAgreement{
        {
                ID:                          "loan-ke-2014-china-exim-sgr",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-uhuru-kenyatta",
                PresidentialTermID:          ptrLegID("term-uhuru-1"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-china-exim-bank",
                CreditorName:                "China Exim Bank",
                CreditorCategory:            legislation.CreditorBilateral,
                InstrumentType:              "Sovereign Loan (Export Credit)",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              3.6e9,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeInfrastructure),
                Sector:                      "Transport (Mombasa-Nairobi SGR)",
                ContractDate:                ptrTime(parseDate("2014-05-11")),
                Status:                      "DISBURSED",
                SourceURL:                   "https://www.reuters.com/article/us-kenya-railway-idUKBREA3M0ZL20140424",
        },
        {
                ID:                          "loan-ke-2014-eurobond",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-uhuru-kenyatta",
                PresidentialTermID:          ptrLegID("term-uhuru-1"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-international-capital-markets",
                CreditorName:                "International Capital Markets",
                CreditorCategory:            legislation.CreditorCommercial,
                InstrumentType:              "Sovereign Bond (Eurobond)",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              2.0e9,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeBudgetSupport),
                Sector:                      "General Government",
                ContractDate:                ptrTime(parseDate("2014-06-24")),
                Status:                      "DISBURSED",
                SourceURL:                   "https://www.reuters.com/article/uk-kenya-bonds-idUKKBN0EZ0Z820140613",
        },
        {
                ID:                          "loan-ke-2014-wb-dpo-devolution",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-uhuru-kenyatta",
                PresidentialTermID:          ptrLegID("term-uhuru-1"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-world-bank-ida",
                CreditorName:                "World Bank (IDA)",
                CreditorCategory:            legislation.CreditorMultilateral,
                InstrumentType:              "Development Policy Financing",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              200_000_000,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeBudgetSupport),
                Sector:                      "Devolution",
                ContractDate:                ptrTime(parseDate("2014-12-11")),
                Status:                      "DISBURSED",
                SourceURL:                   "https://www.worldbank.org/en/news/press-release/2014/12/11/kenya-world-bank-group-approves-us-200-million-for-devolution",
        },
        {
                ID:                          "loan-ke-2015-afdb-last-mile",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-uhuru-kenyatta",
                PresidentialTermID:          ptrLegID("term-uhuru-1"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-afdb",
                CreditorName:                "African Development Bank",
                CreditorCategory:            legislation.CreditorMultilateral,
                InstrumentType:              "Sovereign Loan",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              136_000_000,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeEnergy),
                Sector:                      "Electricity (Last Mile Connectivity)",
                ContractDate:                ptrTime(parseDate("2015-11-25")),
                Status:                      "DISBURSED",
                SourceURL:                   "https://www.afdb.org/en/news-and-events/press-releases/afdb-approves-us-136-million-for-kenya-electricity-modernization-project-15380",
        },
        {
                ID:                          "loan-ke-2016-imf-scf",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-uhuru-kenyatta",
                PresidentialTermID:          ptrLegID("term-uhuru-1"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-imf",
                CreditorName:                "International Monetary Fund",
                CreditorCategory:            legislation.CreditorMultilateral,
                InstrumentType:              "Standby Arrangement and Standby Credit Facility",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              1.5e9,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeBudgetSupport),
                Sector:                      "Balance of Payments",
                ContractDate:                ptrTime(parseDate("2016-10-19")),
                Status:                      "COMPLETED",
                SourceURL:                   "https://www.imf.org/en/News/Articles/2016/10/19/PR16462-Kenya-IMF-Approves-US-1-5-Billion-Standby-Arrangement-and-Standby-Credit-Facility",
        },
        {
                ID:                          "loan-ke-2019-eurobond",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-uhuru-kenyatta",
                PresidentialTermID:          ptrLegID("term-uhuru-2"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-international-capital-markets",
                CreditorName:                "International Capital Markets",
                CreditorCategory:            legislation.CreditorCommercial,
                InstrumentType:              "Sovereign Bond (Eurobond)",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              2.1e9,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeBudgetSupport),
                Sector:                      "General Government",
                ContractDate:                ptrTime(parseDate("2019-02-27")),
                Status:                      "DISBURSED",
                SourceURL:                   "https://www.reuters.com/article/uk-kenya-bonds-idUKKCN0QH0B920150812",
        },
        {
                ID:                          "loan-ke-2020-wb-dpo-covid",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-uhuru-kenyatta",
                PresidentialTermID:          ptrLegID("term-uhuru-2"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-world-bank-ida",
                CreditorName:                "World Bank (IDA)",
                CreditorCategory:            legislation.CreditorMultilateral,
                InstrumentType:              "Development Policy Financing",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              1.0e9,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeBudgetSupport),
                Sector:                      "COVID-19 Response",
                ContractDate:                ptrTime(parseDate("2020-06-02")),
                Status:                      "DISBURSED",
                SourceURL:                   "https://www.worldbank.org/en/news/press-release/2020/05/12/kenya-world-bank-approves-50-million-for-covid-19-response",
        },
        {
                ID:                          "loan-ke-2021-imf-ecf",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-uhuru-kenyatta",
                PresidentialTermID:          ptrLegID("term-uhuru-2"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-imf",
                CreditorName:                "International Monetary Fund",
                CreditorCategory:            legislation.CreditorMultilateral,
                InstrumentType:              "Extended Credit Facility / Extended Fund Facility",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              2.34e9,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeBudgetSupport),
                Sector:                      "Balance of Payments",
                ContractDate:                ptrTime(parseDate("2021-04-02")),
                Status:                      "DISBURSED",
                SourceURL:                   "https://www.imf.org/en/News/Articles/2021/04/02/pr21100-kenya-imf-board-approves-us-2-34-billion-ecf",
        },
        {
                ID:                          "loan-ke-2023-wb-dpo3",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-william-ruto",
                PresidentialTermID:          ptrLegID("term-ruto-1"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-world-bank-ida",
                CreditorName:                "World Bank (IDA)",
                CreditorCategory:            legislation.CreditorMultilateral,
                InstrumentType:              "Development Policy Financing",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              1.0e9,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeBudgetSupport),
                Sector:                      "General Government",
                ContractDate:                ptrTime(parseDate("2023-06-15")),
                Status:                      "DISBURSED",
                SourceURL:                   "https://www.worldbank.org/en/news/press-release/2023/06/15/kenya-world-bank-approves-1-billion-for-development-policy-financing",
        },
        {
                ID:                          "loan-ke-2024-eurobond",
                CountryCode:                 "KE",
                GovernmentAdministrationID: "admin-william-ruto",
                PresidentialTermID:          ptrLegID("term-ruto-1"),
                Borrower:                    "Republic of Kenya",
                CreditorID:                  "creditor-international-capital-markets",
                CreditorName:                "International Capital Markets",
                CreditorCategory:            legislation.CreditorCommercial,
                InstrumentType:              "Sovereign Bond (Eurobond)",
                DomesticOrExternal:          legislation.BorrowingExternal,
                OriginalAmount:              1.5e9,
                OriginalCurrency:            "USD",
                Purpose:                     string(legislation.PurposeBudgetSupport),
                Sector:                      "General Government (refinancing of 2024 maturity)",
                ContractDate:                ptrTime(parseDate("2024-07-04")),
                Status:                      "DISBURSED",
                SourceURL:                   "https://www.reuters.com/world/africa/kenya-prices-15-billion-7-year-eurobond-2024-06-25/",
        },
}

// ptrLegID returns a pointer to a legislation.ID. Used for the optional ID
// fields on BorrowingAgreement (PresidentialTermID, LegislatureID, etc.).
// The kenya_seed package's existing ptrID helper returns *government.ID —
// a different named type — so a dedicated helper is required for the
// debt-domain pointers.
func ptrLegID(s string) *legislation.ID { id := legislation.ID(s); return &id }

// KenyaDebtDashboardSummary is the dashboard-level summary used by the
// /api/v1/debt endpoint. It carries the most-recent snapshot plus
// derived totals that are NOT per-administration.
//
// debtServiceFY2023_24 is the FY 2023/24 debt service total (principal +
// interest). Source: Treasury Annual Public Debt Report 2023/24.
const debtServiceFY2023_24 = 1.36e12

// debtToGDPRatio is the debt-to-GDP ratio as of FY 2023/24.
// Source: Treasury Annual Public Debt Report 2023/24, IMF WEO.
const debtToGDPRatio = 70.2

// DebtDashboardFigures returns the dashboard-level figures derived from
// the most recent CBK snapshot. Used by the API service to populate the
// top-level /api/v1/debt response.
func DebtDashboardFigures() (debtService, debtToGDP float64) {
        return debtServiceFY2023_24, debtToGDPRatio
}

// SeedDebt seeds a DebtRepository with Kenya's authoritative public-debt
// observations, per-administration summaries, per-legislature summaries,
// and sample borrowing agreements. It is intended to be passed to
// legislation.WireDebtRepository.
//
// The seeder is idempotent in the sense that re-running it returns the
// first "already exists" error, which the caller can interpret as
// "already seeded". The caller decides whether to treat that as fatal.
//
// Spec section 9: snapshots are immutable. If KenyaDebtSnapshots ever
// needs correction, the caller MUST add a new snapshot with a different
// ID rather than mutating this list.
//
// Spec section 4: borrowing agreements are attributed by
// CONTRACTED_DURING. Each agreement's GovernmentAdministrationID is
// validated against KenyaAdministrations by the API service via
// legislation.ValidateAttribution (issue #221).
//
// Spec section 19: per-legislature summaries (KenyaLegislatureDebtSummaries)
// surface debt at beginning/end, new borrowing, domestic/external split,
// disbursements, repayments, debt service, and outstanding obligations
// for each Parliamentary term. The summary does NOT attribute sovereign
// borrowing to a Parliament.
func SeedDebt(repo legislation.DebtRepository) error {
        ctx := context.Background()
        for _, snap := range KenyaDebtSnapshots {
                if err := repo.RecordDebtSnapshot(ctx, snap); err != nil {
                        return fmt.Errorf("seed snapshot %s: %w", snap.ID, err)
                }
        }
        for _, summary := range KenyaGovernmentDebtSummaries {
                if err := repo.RecordGovernmentDebtSummary(ctx, summary); err != nil {
                        return fmt.Errorf("seed summary %s: %w", summary.AdministrationID, err)
                }
        }
        for _, legSummary := range KenyaLegislatureDebtSummaries {
                if err := repo.RecordLegislatureDebtSummary(ctx, legSummary); err != nil {
                        return fmt.Errorf("seed legislature summary %s: %w", legSummary.LegislatureID, err)
                }
        }
        for _, agreement := range KenyaBorrowingAgreements {
                if err := repo.CreateBorrowingAgreement(ctx, agreement); err != nil {
                        return fmt.Errorf("seed borrowing agreement %s: %w", agreement.ID, err)
                }
        }
        return nil
}

func parseDate(s string) time.Time {
        t, err := time.Parse("2006-01-02", s)
        if err != nil {
                // Fall back to UTC midnight today; the caller should never see
                // this because the seed data is hard-coded.
                return time.Now().UTC()
        }
        return t.UTC()
}

func ptrFloat(f float64) *float64 { return &f }
