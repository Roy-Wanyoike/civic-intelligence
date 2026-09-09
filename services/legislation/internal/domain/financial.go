package domain

import "time"

// GovernmentLoan represents a sovereign loan taken by the Kenyan government
// (or any tracked country). Every loan MUST link to at least one credible
// source URL (evidence-first contract — no fabricated amounts).
//
// LoanType semantics:
//   - "bilateral":     lender is a single sovereign state or its EXIM bank
//   - "syndicated":    pool of commercial banks under one facility agent
//   - "eurobond":      sovereign bond issued on international capital markets
//   - "multilateral":  IMF, World Bank, AfDB, ADB, etc.
//
// Status lifecycle:
//
//	applied -> approved -> disbursed -> repaid
//	                                -> defaulted
//
// The GovernmentLoan entity is intentionally anemic at the domain layer:
// validation of state transitions, currency conversion and aggregate totals
// is performed by an application-layer service (to be added when the loans
// service is wired up). Keeping the entity simple lets the schema evolve
// without churning callers.
type GovernmentLoan struct {
	ID                   ID
	CountryID            string
	Lender               string
	LoanType             string // bilateral, syndicated, eurobond, multilateral
	AmountUSD            float64
	AmountKES            float64
	Currency             string
	Purpose              string
	Sector               string
	ApplicationDate      *time.Time
	ApprovalDate         *time.Time
	DisbursementDate     *time.Time
	InterestRate         float64
	RepaymentPeriodYears int
	Status               string // applied, approved, disbursed, repaid, defaulted
	SourceURL            string
	SourceDocumentID     *ID
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// GovernmentGrant represents a grant (non-repayable financial transfer) received
// by the Kenyan government. Like GovernmentLoan, every row MUST link to a
// credible source URL.
//
// GrantType semantics:
//   - "bilateral":     single donor sovereign state (e.g. Germany via GIZ)
//   - "multilateral":  EU, World Bank grant window, UN agency, etc.
//   - "foundation":    private philanthropic foundation
//
// Status lifecycle:
//
//	announced -> pending -> disbursed
type GovernmentGrant struct {
	ID               ID
	CountryID        string
	Donor            string
	GrantType        string
	AmountUSD        float64
	AmountKES        float64
	Purpose          string
	Sector           string
	AnnouncementDate *time.Time
	DisbursementDate *time.Time
	Status           string // announced, disbursed, pending
	SourceURL        string
	SourceDocumentID *ID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// LoanStatus constants — single source of truth for loan lifecycle states.
// Mirrors the DEFAULT + CHECK-friendly vocabulary used by the migration.
const (
	LoanStatusApplied   = "applied"
	LoanStatusApproved  = "approved"
	LoanStatusDisbursed = "disbursed"
	LoanStatusRepaid    = "repaid"
	LoanStatusDefaulted = "defaulted"
)

// GrantStatus constants — single source of truth for grant lifecycle states.
const (
	GrantStatusAnnounced = "announced"
	GrantStatusPending   = "pending"
	GrantStatusDisbursed = "disbursed"
)

// LoanType constants.
const (
	LoanTypeBilateral    = "bilateral"
	LoanTypeSyndicated   = "syndicated"
	LoanTypeEurobond     = "eurobond"
	LoanTypeMultilateral = "multilateral"
)

// GrantType constants.
const (
	GrantTypeBilateral    = "bilateral"
	GrantTypeMultilateral = "multilateral"
	GrantTypeFoundation   = "foundation"
)

// IsValidLoanStatus reports whether s is a known loan status.
func IsValidLoanStatus(s string) bool {
	switch s {
	case LoanStatusApplied, LoanStatusApproved, LoanStatusDisbursed,
		LoanStatusRepaid, LoanStatusDefaulted:
		return true
	}
	return false
}

// IsValidGrantStatus reports whether s is a known grant status.
func IsValidGrantStatus(s string) bool {
	switch s {
	case GrantStatusAnnounced, GrantStatusPending, GrantStatusDisbursed:
		return true
	}
	return false
}

// IsValidLoanType reports whether s is a known loan type.
func IsValidLoanType(s string) bool {
	switch s {
	case LoanTypeBilateral, LoanTypeSyndicated, LoanTypeEurobond, LoanTypeMultilateral:
		return true
	}
	return false
}

// IsValidGrantType reports whether s is a known grant type.
func IsValidGrantType(s string) bool {
	switch s {
	case GrantTypeBilateral, GrantTypeMultilateral, GrantTypeFoundation:
		return true
	}
	return false
}
