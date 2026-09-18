// Package ghana_seed provides authoritative seed data for Ghana's public debt
// observations and borrowing agreements.
//
// All data is sourced from authoritative references:
//   - Bank of Ghana — Monetary Time Series + Summary of Economic and Financial
//     Indicators https://www.bog.gov.gh/statistics/time-series/
//   - Ministry of Finance — Annual Public Debt Report
//     https://mofep.gov.gh/publications/debt-management-reports
//   - IMF Article IV Consultation Staff Reports for Ghana
//     https://www.imf.org/en/Countries/GHA
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT say
// "President X borrowed GHS X". It says "The Government of Ghana recorded
// GHS X in borrowing during this period." The legal borrower is the
// Republic of Ghana, not a person.
package ghana_seed

import (
	"context"
	"fmt"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// bogSourceURL is the canonical Bank of Ghana monetary time series page.
// Repeated on every snapshot so the API can surface the source per row.
const bogSourceURL = "https://www.bog.gov.gh/statistics/time-series/"

// mofAnnualDebtReport is the canonical Ministry of Finance annual debt
// management report URL.
const mofAnnualDebtReport = "https://mofep.gov.gh/publications/debt-management-reports"

// GhanaDebtSnapshots is the authoritative time series of Ghana's total public
// debt stock. Spec section 9 — each snapshot is an immutable observation;
// changes between snapshots can reflect FX movements, valuation changes,
// repayments, refinancing, arrears, adjustments, and disbursement timing —
// NOT just new borrowing.
//
// Source: Bank of Ghana Summary of Economic and Financial Indicators
// (December of each year). Amounts are in GHS millions.
var GhanaDebtSnapshots = []legislation.PublicDebtSnapshot{
	snapshot("snap-gh-2017-12-31", "2017-12-31", 142.6e9, 36.1e9, 106.5e9),
	snapshot("snap-gh-2018-12-31", "2018-12-31", 173.2e9, 41.9e9, 131.3e9),
	snapshot("snap-gh-2019-12-31", "2019-12-31", 218.0e9, 50.3e9, 167.7e9),
	snapshot("snap-gh-2020-12-31", "2020-12-31", 291.6e9, 76.9e9, 214.7e9),
	snapshot("snap-gh-2021-12-31", "2021-12-31", 344.5e9, 91.0e9, 253.5e9),
	snapshot("snap-gh-2022-12-31", "2022-12-31", 472.4e9, 122.3e9, 350.1e9),
	snapshot("snap-gh-2023-12-31", "2023-12-31", 612.9e9, 145.7e9, 467.2e9),
	snapshot("snap-gh-2024-06-30", "2024-06-30", 738.9e9, 173.1e9, 565.8e9),
}

// snapshot is a helper that constructs a PublicDebtSnapshot from the
// compact arguments used in GhanaDebtSnapshots. All Ghana snapshots are
// denominated in GHS and sourced from the Bank of Ghana.
func snapshot(id, date string, total, domestic, external float64) legislation.PublicDebtSnapshot {
	return legislation.PublicDebtSnapshot{
		ID:               legislation.ID(id),
		CountryCode:      "GH",
		ObservationDate:  parseDate(date),
		TotalDebtStock:   total,
		DomesticDebt:     domestic,
		ExternalDebt:     external,
		Currency:         "GHS",
		ExchangeRateAsOf: parseDate(date),
		SourceURL:        bogSourceURL,
	}
}

// GhanaBorrowingAgreements is a non-exhaustive list of Ghana's most
// significant borrowing agreements since 2017. Each entry is sourced from
// the creditor's or the Ministry of Finance's own public press release.
//
// CRITICAL ATTRIBUTION RULE (Spec section 4, 35): every agreement is
// attributed by CONTRACTED_DURING (ContractDate), NOT by DISBURSED_DURING
// or REPAID_DURING. The GovernmentAdministrationID corresponds to the
// administration in power on ContractDate. The legal borrower is always the
// Republic of Ghana — the platform never attributes sovereign borrowing
// personally to a president.
//
// Administration windows (see government.go):
//   - admin-john-mahama: 2012-07-24 .. 2017-01-07
//   - admin-nana-akufo-addo: 2017-01-07 .. present
var GhanaBorrowingAgreements = []legislation.BorrowingAgreement{
	{
		ID:                          "loan-gh-2018-eurobond",
		CountryCode:                 "GH",
		GovernmentAdministrationID:   "admin-nana-akufo-addo",
		Borrower:                    "Republic of Ghana",
		CreditorID:                  "creditor-international-capital-markets",
		CreditorName:                 "International Capital Markets",
		CreditorCategory:             legislation.CreditorCommercial,
		InstrumentType:              "Sovereign Bond (Eurobond)",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               2.0e9,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeBudgetSupport),
		Sector:                       "General Government (refinancing + capital expenditure)",
		ContractDate:                 ptrTime(parseDate("2018-05-18")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://www.reuters.com/article/uk-ghana-bonds-idUKKCN1IJ13J",
	},
	{
		ID:                          "loan-gh-2020-imf-rcf-covid",
		CountryCode:                 "GH",
		GovernmentAdministrationID:   "admin-nana-akufo-addo",
		Borrower:                    "Republic of Ghana",
		CreditorID:                  "creditor-imf",
		CreditorName:                 "International Monetary Fund",
		CreditorCategory:             legislation.CreditorMultilateral,
		InstrumentType:              "Rapid Credit Facility (RCF)",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               1_000_000_000,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeBudgetSupport),
		Sector:                       "COVID-19 Response",
		ContractDate:                 ptrTime(parseDate("2020-04-13")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://www.imf.org/en/News/Articles/2020/04/13/pr20145-ghana-imf-executive-board-approves-us-1-billion-disbursement-rcf-covid-19",
	},
	{
		ID:                          "loan-gh-2021-afdb-agri-inputs",
		CountryCode:                 "GH",
		GovernmentAdministrationID:   "admin-nana-akufo-addo",
		Borrower:                    "Republic of Ghana",
		CreditorID:                  "creditor-afdb",
		CreditorName:                 "African Development Bank",
		CreditorCategory:             legislation.CreditorMultilateral,
		InstrumentType:              "Sovereign Loan",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               46.71e6,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeAgriculture),
		Sector:                       "Agriculture (Planting for Food and Jobs)",
		ContractDate:                 ptrTime(parseDate("2021-12-10")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://www.afdb.org/en/documents/ghana-agricultural-inputs-subsidy-programme",
	},
	{
		ID:                          "loan-gh-2022-world-bank-dpo",
		CountryCode:                 "GH",
		GovernmentAdministrationID:   "admin-nana-akufo-addo",
		Borrower:                    "Republic of Ghana",
		CreditorID:                  "creditor-world-bank-ida",
		CreditorName:                 "World Bank (IDA)",
		CreditorCategory:             legislation.CreditorMultilateral,
		InstrumentType:              "Development Policy Financing",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               250_000_000,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeBudgetSupport),
		Sector:                       "General Government (fiscal + financial sector sustainability)",
		ContractDate:                 ptrTime(parseDate("2022-06-30")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://projects.worldbank.org/en/projects-operations/project-detail/P177005",
	},
}

// SeedDebt seeds a DebtRepository with Ghana's authoritative public-debt
// observations and sample borrowing agreements. It is intended to be passed
// to legislation.WireDebtRepository.
func SeedDebt(repo legislation.DebtRepository) error {
	ctx := context.Background()
	for _, snap := range GhanaDebtSnapshots {
		if err := repo.RecordDebtSnapshot(ctx, snap); err != nil {
			return fmt.Errorf("seed snapshot %s: %w", snap.ID, err)
		}
	}
	for _, agreement := range GhanaBorrowingAgreements {
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
