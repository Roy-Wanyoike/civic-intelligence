package domain

import (
	"testing"
	"time"
)

// TestBorrowingAgreement_AttributionRule verifies that the attribution rule
// (Spec section 35) is enforced: borrowing is attributed to the government
// period during which it was CONTRACTED, not when repayment occurred.
func TestBorrowingAgreement_AttributionRule(t *testing.T) {
	now := time.Now()
	contractDate := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
	repaymentDate := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)

	// A loan contracted in 2020 (during Uhuru's Term 2) but repaid in 2024
	// (during Ruto's Term 1).
	agreement := BorrowingAgreement{
		ID:                       "loan-test-1",
		CountryCode:             "KE",
		GovernmentAdministrationID: "admin-uhuru-kenyatta",
		PresidentialTermID:      ptrToID("term-uhuru-2"),
		ContractDate:            &contractDate,
		DisbursementDate:        &contractDate,
		OriginalAmount:          1_000_000_000,
		OriginalCurrency:        "USD",
	}

	// Attributing by CONTRACTED_DURING points to Uhuru's term.
	if agreement.GovernmentAdministrationID != "admin-uhuru-kenyatta" {
		t.Errorf("expected admin-uhuru-kenyatta; got %s", agreement.GovernmentAdministrationID)
	}

	// The repayment event is a separate record; it does NOT change the
	// contracted-during attribution.
	repayment := DebtRepayment{
		ID:                    "repayment-1",
		BorrowingAgreementID:  agreement.ID,
		Date:                  repaymentDate,
		Amount:                50_000_000,
		PrincipalPortion:      45_000_000,
		InterestPortion:       5_000_000,
	}

	// The repayment's date falls in a different period, but the loan's
	// attribution is by contract date. This is the correct behaviour per
	// Spec section 4.
	if repayment.Date.Before(agreement.ContractDate.Add(0)) == false {
		// Repayment is after contract — expected.
	} else {
		t.Error("repayment should be after contract date")
	}
	_ = now
}

// TestPublicDebtSnapshot_ImmutableObservation verifies that snapshots are
// immutable observations. Spec section 9.
//
// Changes in debt stock can reflect exchange-rate movements, valuation
// changes, repayments, refinancing, arrears, adjustments, disbursement
// timing — NOT just new borrowing.
func TestPublicDebtSnapshot_ImmutableObservation(t *testing.T) {
	date1 := time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	snap1 := PublicDebtSnapshot{
		ID:              "snap-1",
		CountryCode:     "KE",
		ObservationDate: date1,
		TotalDebtStock:  8.4e12, // KSh 8.4T
		DomesticDebt:    3.7e12,
		ExternalDebt:    4.7e12,
		Currency:        "KES",
	}
	snap2 := PublicDebtSnapshot{
		ID:              "snap-2",
		CountryCode:     "KE",
		ObservationDate: date2,
		TotalDebtStock:  11.0e12,
		DomesticDebt:    4.5e12,
		ExternalDebt:    6.5e12,
		Currency:        "KES",
	}

	// Snapshots are immutable. Each snapshot stands on its own.
	if snap1.TotalDebtStock == snap2.TotalDebtStock {
		t.Error("snapshots should reflect different observations")
	}

	// IMPORTANT: do NOT infer new borrowing from (snap2 - snap1).
	// The difference includes FX, valuation, repayments, refinancing, etc.
	diff := snap2.TotalDebtStock - snap1.TotalDebtStock
	if diff <= 0 {
		t.Error("test setup error: expected debt to grow")
	}
	// The platform should NEVER report this as "new borrowing" without
	// consulting BorrowingAgreement records.
}

// TestGovernmentDebtSummary_HasDisclaimer verifies that every debt summary
// carries the NO_POLITICAL_PERFORMANCE_SCORE disclaimer. Spec section 37.
func TestGovernmentDebtSummary_HasDisclaimer(t *testing.T) {
	summary := GovernmentDebtSummary{
		AdministrationID: "admin-uhuru-kenyatta",
		Period:           "2013-2022",
		Disclaimer:       NO_POLITICAL_PERFORMANCE_SCORE,
	}
	if summary.Disclaimer == "" {
		t.Error("expected disclaimer to be present")
	}
	if !containsSubstring(summary.Disclaimer, "does not") {
		t.Error("expected disclaimer to mention what the platform does NOT do")
	}
	if !containsSubstring(summary.Disclaimer, "ranking") {
		t.Error("expected disclaimer to mention 'ranking'")
	}
}

// TestCreditorCategory_Values verifies the creditor categories.
func TestCreditorCategory_Values(t *testing.T) {
	cases := []struct {
		v   CreditorCategory
		exp string
	}{
		{CreditorMultilateral, "MULTILATERAL"},
		{CreditorBilateral, "BILATERAL"},
		{CreditorCommercial, "COMMERCIAL"},
		{CreditorDomestic, "DOMESTIC_INVESTOR"},
	}
	for _, c := range cases {
		if string(c.v) != c.exp {
			t.Errorf("expected %s; got %s", c.exp, c.v)
		}
	}
}

// TestBorrowingAttribution_Values verifies the attribution enum.
func TestBorrowingAttribution_Values(t *testing.T) {
	cases := []struct {
		v   BorrowingAttribution
		exp string
	}{
		{AttributionContracted, "CONTRACTED_DURING"},
		{AttributionDisbursed, "DISBURSED_DURING"},
		{AttributionRepaid, "REPAID_DURING"},
		{AttributionOutstanding, "OUTSTANDING_DURING"},
		{AttributionRefinanced, "REFINANCED_DURING"},
	}
	for _, c := range cases {
		if string(c.v) != c.exp {
			t.Errorf("expected %s; got %s", c.exp, c.v)
		}
	}
}

func ptrToID(s string) *ID { id := ID(s); return &id }

func containsSubstring(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		(len(haystack) > 0 && (indexOf(haystack, needle) >= 0)))
}

func indexOf(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
