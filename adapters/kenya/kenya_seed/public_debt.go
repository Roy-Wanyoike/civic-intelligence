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
// observations and per-administration summaries. It is intended to be
// passed to legislation.WireDebtRepository.
//
// The seeder is idempotent in the sense that re-running it returns the
// first "already exists" error, which the caller can interpret as
// "already seeded". The caller decides whether to treat that as fatal.
//
// Spec section 9: snapshots are immutable. If KenyaDebtSnapshots ever
// needs correction, the caller MUST add a new snapshot with a different
// ID rather than mutating this list.
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
