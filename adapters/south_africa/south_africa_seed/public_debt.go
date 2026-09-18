// Package south_africa_seed provides authoritative seed data for South Africa's
// public debt observations and borrowing agreements.
//
// All data is sourced from authoritative references:
//   - South African Reserve Bank — Quarterly Bulletin
//     https://www.resbank.co.za/publications/quarterly-bulletins
//   - National Treasury — Budget Review + Annual Debt Report
//     https://www.treasury.gov.za/publications/
//   - IMF Article IV Consultation Staff Reports for South Africa
//     https://www.imf.org/en/Countries/ZAF
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT say
// "President X borrowed ZAR X". It says "The Government of South Africa
// recorded ZAR X in borrowing during this period." The legal borrower is
// the Republic of South Africa, not a person.
package south_africa_seed

import (
	"context"
	"fmt"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// sarbSourceURL is the canonical SARB Quarterly Bulletin page. Repeated on
// every snapshot so the API can surface the source per row.
const sarbSourceURL = "https://www.resbank.co.za/publications/quarterly-bulletins"

// treasuryBudgetReview is the canonical National Treasury Budget Review URL.
const treasuryBudgetReview = "https://www.treasury.gov.za/publications/budget/"

// SouthAfricaDebtSnapshots is the authoritative time series of South Africa's
// total public debt stock. Spec section 9 — each snapshot is an immutable
// observation; changes between snapshots can reflect FX movements,
// valuation changes, repayments, refinancing, arrears, adjustments, and
// disbursement timing — NOT just new borrowing.
//
// Source: South African Reserve Bank Quarterly Bulletin (March of each year,
// end-of-fiscal-year reporting). Amounts are in ZAR billions.
var SouthAfricaDebtSnapshots = []legislation.PublicDebtSnapshot{
	snapshot("snap-za-2017-03-31", "2017-03-31", 2.21e12, 1.07e12, 1.14e12),
	snapshot("snap-za-2018-03-31", "2018-03-31", 2.46e12, 1.20e12, 1.26e12),
	snapshot("snap-za-2019-03-31", "2019-03-31", 2.75e12, 1.36e12, 1.39e12),
	snapshot("snap-za-2020-03-31", "2020-03-31", 3.18e12, 1.56e12, 1.62e12),
	snapshot("snap-za-2021-03-31", "2021-03-31", 3.95e12, 2.10e12, 1.85e12),
	snapshot("snap-za-2022-03-31", "2022-03-31", 4.74e12, 2.57e12, 2.17e12),
	snapshot("snap-za-2023-03-31", "2023-03-31", 5.24e12, 2.85e12, 2.39e12),
	snapshot("snap-za-2024-03-31", "2024-03-31", 5.92e12, 3.30e12, 2.62e12),
}

// snapshot is a helper that constructs a PublicDebtSnapshot from the
// compact arguments used in SouthAfricaDebtSnapshots. All South Africa
// snapshots are denominated in ZAR and sourced from the SARB.
func snapshot(id, date string, total, domestic, external float64) legislation.PublicDebtSnapshot {
	return legislation.PublicDebtSnapshot{
		ID:               legislation.ID(id),
		CountryCode:      "ZA",
		ObservationDate:  parseDate(date),
		TotalDebtStock:   total,
		DomesticDebt:     domestic,
		ExternalDebt:     external,
		Currency:         "ZAR",
		ExchangeRateAsOf: parseDate(date),
		SourceURL:        sarbSourceURL,
	}
}

// SouthAfricaBorrowingAgreements is a non-exhaustive list of South Africa's
// most significant borrowing agreements since 2018. Each entry is sourced
// from the creditor's or the National Treasury's own public press release.
//
// CRITICAL ATTRIBUTION RULE (Spec section 4, 35): every agreement is
// attributed by CONTRACTED_DURING (ContractDate), NOT by DISBURSED_DURING
// or REPAID_DURING. The GovernmentAdministrationID corresponds to the
// administration in power on ContractDate. The legal borrower is always the
// Republic of South Africa — the platform never attributes sovereign
// borrowing personally to a president.
//
// Administration windows (see government.go):
//   - admin-jacob-zuma: 2009-05-09 .. 2018-02-15
//   - admin-cyril-ramaphosa: 2018-02-15 .. present
var SouthAfricaBorrowingAgreements = []legislation.BorrowingAgreement{
	{
		ID:                          "loan-za-2018-treasury-bond-2048",
		CountryCode:                 "ZA",
		GovernmentAdministrationID:   "admin-cyril-ramaphosa",
		Borrower:                    "Republic of South Africa",
		CreditorID:                  "creditor-domestic-investors",
		CreditorName:                 "Domestic Investors (Primary Dealer Auction)",
		CreditorCategory:             legislation.CreditorDomestic,
		InstrumentType:              "Treasury Bond (R2048)",
		DomesticOrExternal:           legislation.BorrowingDomestic,
		OriginalAmount:               5.0e9,
		OriginalCurrency:            "ZAR",
		Purpose:                      string(legislation.PurposeBudgetSupport),
		Sector:                       "General Government",
		ContractDate:                 ptrTime(parseDate("2018-07-20")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://www.treasury.gov.za/comm_media/press/2018/20180720%20Media%20Statement%20Bond%20Switch%20Auction.pdf",
	},
	{
		ID:                          "loan-za-2020-imf-rfi-covid",
		CountryCode:                 "ZA",
		GovernmentAdministrationID:   "admin-cyril-ramaphosa",
		Borrower:                    "Republic of South Africa",
		CreditorID:                  "creditor-imf",
		CreditorName:                 "International Monetary Fund",
		CreditorCategory:             legislation.CreditorMultilateral,
		InstrumentType:              "Rapid Financing Instrument (RFI)",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               4_300_000_000,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeBudgetSupport),
		Sector:                       "COVID-19 Response + Balance of Payments",
		ContractDate:                 ptrTime(parseDate("2020-07-27")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://www.imf.org/en/News/Articles/2020/07/27/pr20275-south-africa-imf-executive-board-approves-us-4-3-billion-emergency-covid-19-support",
	},
	{
		ID:                          "loan-za-2022-afdb-just-energy-transition",
		CountryCode:                 "ZA",
		GovernmentAdministrationID:   "admin-cyril-ramaphosa",
		Borrower:                    "Republic of South Africa",
		CreditorID:                  "creditor-afdb",
		CreditorName:                 "African Development Bank",
		CreditorCategory:             legislation.CreditorMultilateral,
		InstrumentType:              "Policy-Based Loan",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               300_000_000,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeEnergy),
		Sector:                       "Just Energy Transition",
		ContractDate:                 ptrTime(parseDate("2022-11-30")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://www.afdb.org/en/news-and-events/press-releases/afdb-approves-300-million-support-south-africas-just-energy-transition-57689",
	},
}

// SeedDebt seeds a DebtRepository with South Africa's authoritative
// public-debt observations and sample borrowing agreements. It is intended
// to be passed to legislation.WireDebtRepository.
func SeedDebt(repo legislation.DebtRepository) error {
	ctx := context.Background()
	for _, snap := range SouthAfricaDebtSnapshots {
		if err := repo.RecordDebtSnapshot(ctx, snap); err != nil {
			return fmt.Errorf("seed snapshot %s: %w", snap.ID, err)
		}
	}
	for _, agreement := range SouthAfricaBorrowingAgreements {
		if err := repo.CreateBorrowingAgreement(ctx, agreement); err != nil {
			return fmt.Errorf("seed borrowing agreement %s: %w", agreement.ID, err)
		}
	}
	return nil
}

func parseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Now().UTC()
	}
	return t.UTC()
}

func ptrFloat(f float64) *float64 { return &f }
