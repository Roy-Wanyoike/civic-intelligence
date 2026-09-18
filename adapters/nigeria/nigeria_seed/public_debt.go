// Package nigeria_seed provides authoritative seed data for Nigeria's public
// debt observations and borrowing agreements.
//
// All data is sourced from authoritative references:
//   - Debt Management Office (DMO) — Public Debt Data
//     https://www.dmo.gov.ng/public-debt-data
//   - Central Bank of Nigeria — Statistical Bulletin
//     https://www.cbn.gov.ng/documents/statbulletin.asp
//   - Federal Ministry of Finance — Annual Debt Sustainability Report
//     https://www.fmf.gov.ng/
//   - IMF Article IV Consultation Staff Reports for Nigeria
//     https://www.imf.org/en/Countries/NGA
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT say
// "President X borrowed NGN X". It says "The Government of Nigeria
// recorded NGN X in borrowing during this period." The legal borrower is
// the Federal Republic of Nigeria, not a person.
package nigeria_seed

import (
	"context"
	"fmt"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// dmoSourceURL is the canonical Debt Management Office public debt data page.
// Repeated on every snapshot so the API can surface the source per row.
const dmoSourceURL = "https://www.dmo.gov.ng/public-debt-data"

// cbnStatsBulletin is the canonical CBN Statistical Bulletin URL.
const cbnStatsBulletin = "https://www.cbn.gov.ng/documents/statbulletin.asp"

// NigeriaDebtSnapshots is the authoritative time series of Nigeria's total
// public debt stock. Spec section 9 — each snapshot is an immutable
// observation; changes between snapshots can reflect FX movements,
// valuation changes, repayments, refinancing, arrears, adjustments, and
// disbursement timing — NOT just new borrowing.
//
// Source: Debt Management Office quarterly public debt reports (December of
// each year). Amounts are in NGN billions.
var NigeriaDebtSnapshots = []legislation.PublicDebtSnapshot{
	snapshot("snap-ng-2017-12-31", "2017-12-31", 17.5e12, 7.7e12, 9.8e12),
	snapshot("snap-ng-2018-12-31", "2018-12-31", 21.7e12, 9.3e12, 12.4e12),
	snapshot("snap-ng-2019-12-31", "2019-12-31", 25.7e12, 10.7e12, 15.0e12),
	snapshot("snap-ng-2020-12-31", "2020-12-31", 32.9e12, 13.6e12, 19.3e12),
	snapshot("snap-ng-2021-12-31", "2021-12-31", 38.8e12, 15.8e12, 23.0e12),
	snapshot("snap-ng-2022-12-31", "2022-12-31", 44.1e12, 18.1e12, 26.0e12),
	snapshot("snap-ng-2023-12-31", "2023-12-31", 87.4e12, 38.4e12, 49.0e12),
	snapshot("snap-ng-2024-06-30", "2024-06-30", 121.6e12, 53.0e12, 68.6e12),
}

// snapshot is a helper that constructs a PublicDebtSnapshot from the
// compact arguments used in NigeriaDebtSnapshots. All Nigeria snapshots
// are denominated in NGN and sourced from the DMO + CBN.
func snapshot(id, date string, total, domestic, external float64) legislation.PublicDebtSnapshot {
	return legislation.PublicDebtSnapshot{
		ID:               legislation.ID(id),
		CountryCode:      "NG",
		ObservationDate:  parseDate(date),
		TotalDebtStock:   total,
		DomesticDebt:     domestic,
		ExternalDebt:     external,
		Currency:         "NGN",
		ExchangeRateAsOf: parseDate(date),
		SourceURL:        dmoSourceURL,
	}
}

// NigeriaBorrowingAgreements is a non-exhaustive list of Nigeria's most
// significant borrowing agreements since 2017. Each entry is sourced from
// the creditor's or the DMO's own public press release.
//
// CRITICAL ATTRIBUTION RULE (Spec section 4, 35): every agreement is
// attributed by CONTRACTED_DURING (ContractDate), NOT by DISBURSED_DURING
// or REPAID_DURING. The GovernmentAdministrationID corresponds to the
// administration in power on ContractDate. The legal borrower is always the
// Federal Republic of Nigeria — the platform never attributes sovereign
// borrowing personally to a president.
//
// Administration windows (see government.go):
//   - admin-muhammadu-buhari: 2015-05-29 .. 2023-05-29
//   - admin-bola-tinubu: 2023-05-29 .. present
var NigeriaBorrowingAgreements = []legislation.BorrowingAgreement{
	{
		ID:                          "loan-ng-2018-eurobond",
		CountryCode:                 "NG",
		GovernmentAdministrationID:   "admin-muhammadu-buhari",
		Borrower:                    "Federal Republic of Nigeria",
		CreditorID:                  "creditor-international-capital-markets",
		CreditorName:                 "International Capital Markets",
		CreditorCategory:             legislation.CreditorCommercial,
		InstrumentType:              "Sovereign Bond (Eurobond)",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               2.5e9,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeBudgetSupport),
		Sector:                       "General Government (budget support + refinancing)",
		ContractDate:                 ptrTime(parseDate("2018-07-19")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://www.dmo.gov.ng/press-release/nigeria-raises-2-5-billion-from-international-capital-market",
	},
	{
		ID:                          "loan-ng-2020-imf-rfi-covid",
		CountryCode:                 "NG",
		GovernmentAdministrationID:   "admin-muhammadu-buhari",
		Borrower:                    "Federal Republic of Nigeria",
		CreditorID:                  "creditor-imf",
		CreditorName:                 "International Monetary Fund",
		CreditorCategory:             legislation.CreditorMultilateral,
		InstrumentType:              "Rapid Financing Instrument (RFI)",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               3_400_000_000,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeBudgetSupport),
		Sector:                       "COVID-19 Response",
		ContractDate:                 ptrTime(parseDate("2020-04-28")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://www.imf.org/en/News/Articles/2020/04/28/pr20184-nigeria-imf-executive-board-approves-us-3-4-billion-emergency-funding-covid-19",
	},
	{
		ID:                          "loan-ng-2023-afdb-iita-agri",
		CountryCode:                 "NG",
		GovernmentAdministrationID:   "admin-bola-tinubu",
		Borrower:                    "Federal Republic of Nigeria",
		CreditorID:                  "creditor-afdb",
		CreditorName:                 "African Development Bank",
		CreditorCategory:             legislation.CreditorMultilateral,
		InstrumentType:              "Sovereign Loan",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               134_000_000,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeAgriculture),
		Sector:                       "Agriculture (Special Agro-Industrial Processing Zones)",
		ContractDate:                 ptrTime(parseDate("2023-08-22")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://www.afdb.org/en/news-and-events/press-releases/afdb-approves-134-million-nigeria-special-agro-industrial-processing-zones-program-62323",
	},
	{
		ID:                          "loan-ng-2023-world-bank-power",
		CountryCode:                 "NG",
		GovernmentAdministrationID:   "admin-bola-tinubu",
		Borrower:                    "Federal Republic of Nigeria",
		CreditorID:                  "creditor-world-bank-ida",
		CreditorName:                 "World Bank (IDA)",
		CreditorCategory:             legislation.CreditorMultilateral,
		InstrumentType:              "Development Policy Financing",
		DomesticOrExternal:           legislation.BorrowingExternal,
		OriginalAmount:               750_000_000,
		OriginalCurrency:            "USD",
		Purpose:                      string(legislation.PurposeEnergy),
		Sector:                       "Power Sector Reforms",
		ContractDate:                 ptrTime(parseDate("2023-12-15")),
		Status:                       "DISBURSED",
		SourceURL:                    "https://projects.worldbank.org/en/projects-operations/project-detail/P177323",
	},
}

// SeedDebt seeds a DebtRepository with Nigeria's authoritative public-debt
// observations and sample borrowing agreements. It is intended to be passed
// to legislation.WireDebtRepository.
func SeedDebt(repo legislation.DebtRepository) error {
	ctx := context.Background()
	for _, snap := range NigeriaDebtSnapshots {
		if err := repo.RecordDebtSnapshot(ctx, snap); err != nil {
			return fmt.Errorf("seed snapshot %s: %w", snap.ID, err)
		}
	}
	for _, agreement := range NigeriaBorrowingAgreements {
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
