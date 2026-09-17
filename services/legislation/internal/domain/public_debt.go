// Package legislation — public debt & borrowing intelligence domain.
//
// Public Debt & Borrowing Intelligence spec.
//
// CRITICAL ATTRIBUTION RULE (Spec section 35):
// The platform must NEVER automatically say:
//   "President X borrowed KSh X."
//
// Instead use precise formulations:
//   "The Government of Kenya recorded KSh X in borrowing during this period."
//   "Public debt increased from X to Y between these observation dates."
//
// Financial semantics (Spec section 3) — NEVER collapse:
//   - New borrowing      — a new borrowing agreement or issuance
//   - Commitment         — amount legally committed
//   - Disbursement       — money actually released
//   - Debt stock         — outstanding public debt at a particular date
//   - Repayment          — principal paid back
//   - Interest           — cost paid on the debt
//   - Debt service       — principal + interest + associated payments
//   - Refinancing         — new borrowing used to refinance existing obligations
//   - Debt restructuring — modification of existing debt terms
package domain

import "time"

// FiscalYear represents a government fiscal year.
type FiscalYear struct {
	ID         ID
	CountryCode string
	StartDate  time.Time
	EndDate    time.Time
	Label      string // e.g. "FY 2024/25"
	CreatedAt  time.Time
}

// CreditorCategory classifies a creditor. Spec section 18.
type CreditorCategory string

const (
	CreditorMultilateral   CreditorCategory = "MULTILATERAL"
	CreditorBilateral      CreditorCategory = "BILATERAL"
	CreditorCommercial     CreditorCategory = "COMMERCIAL"
	CreditorDomestic       CreditorCategory = "DOMESTIC_INVESTOR"
	CreditorInstitution    CreditorCategory = "FINANCIAL_INSTITUTION"
	CreditorOther          CreditorCategory = "OTHER"
)

// DomesticOrExternal classifies borrowing direction. Spec section 2.
type DomesticOrExternal string

const (
	BorrowingDomestic  DomesticOrExternal = "DOMESTIC"
	BorrowingExternal  DomesticOrExternal = "EXTERNAL"
)

// BorrowingAttribution is the temporal linkage between a borrowing event
// and a government period. Spec section 4.
//
// CRITICAL: do not attribute a loan simply because repayment occurred during
// a particular president's tenure. The platform distinguishes:
//   - CONTRACTED_DURING — when the agreement was signed
//   - DISBURSED_DURING  — when money was actually released
//   - REPAID_DURING     — when repayments occurred
//   - OUTSTANDING_DURING — when the debt was outstanding
//   - REFINANCED_DURING — when refinancing occurred
type BorrowingAttribution string

const (
	AttributionContracted  BorrowingAttribution = "CONTRACTED_DURING"
	AttributionDisbursed   BorrowingAttribution = "DISBURSED_DURING"
	AttributionRepaid      BorrowingAttribution = "REPAID_DURING"
	AttributionOutstanding BorrowingAttribution = "OUTSTANDING_DURING"
	AttributionRefinanced  BorrowingAttribution = "REFINANCED_DURING"
)

// BorrowingAgreement is a single borrowing agreement (loan, bond, etc.).
// Spec section 2.
//
// Some fields will legitimately be:
//   UNKNOWN, NOT_PUBLISHED, NOT_APPLICABLE
//
// NEVER manufacture missing financial terms.
type BorrowingAgreement struct {
	ID              ID
	CountryCode     string
	GovernmentAdministrationID ID
	PresidentialTermID *ID
	LegislatureID   *ID
	FiscalYearID    *ID

	Borrower        string // legal borrower (Republic of Kenya, parastatal, etc.)
	CreditorID      ID
	CreditorName    string
	CreditorCategory CreditorCategory
	InstrumentType   string // Treasury bill, Treasury bond, external loan, etc.
	DomesticOrExternal DomesticOrExternal

	OriginalAmount  float64
	OriginalCurrency string
	ExchangeRateAtContract float64 // if foreign currency
	ExchangeRateDate *time.Time
	ConvertedAmountReportingCurrency float64 // e.g. KES equivalent

	Purpose          string
	Sector           string

	ContractDate     *time.Time
	ApprovalDate     *time.Time
	SignatureDate    *time.Time
	DisbursementDate *time.Time
	MaturityDate     *time.Time

	InterestRate    *float64
	Fees            *float64
	GracePeriodMonths *int
	RepaymentSchedule string

	OutstandingBalance *float64
	AmountRepaid       *float64
	AmountDisbursed    *float64

	Status           string // CONTRACTED, DISBURSED, REPAID, RESTRUCTURED, DEFAULTED, UNKNOWN

	SourceURL       string
	SourceDocumentID *ID
	Evidence        []DebtEvidenceRef

	CreatedAt time.Time
	UpdatedAt time.Time
}

// DebtDisbursement records a single disbursement of a borrowing agreement.
type DebtDisbursement struct {
	ID               ID
	BorrowingAgreementID ID
	Date             time.Time
	Amount           float64
	Currency         string
	ExchangeRateAtDisbursement float64
	SourceURL        string
	CreatedAt        time.Time
}

// DebtRepayment records a single principal repayment.
type DebtRepayment struct {
	ID               ID
	BorrowingAgreementID ID
	Date             time.Time
	Amount           float64
	Currency         string
	PrincipalPortion float64
	InterestPortion  float64
	SourceURL        string
	CreatedAt        time.Time
}

// DebtService is a derived aggregate of principal + interest for a period.
// Spec section 3.
type DebtService struct {
	ID              ID
	CountryCode     string
	FiscalYearID    *ID
	Period          string // e.g. "FY 2024/25"
	Principal       float64
	Interest        float64
	TotalService    float64
	Currency        string
	SourceURL       string
	CreatedAt       time.Time
}

// PublicDebtSnapshot is a point-in-time observation of total public debt.
// Spec section 3, 9.
//
// IMPORTANT: snapshots are immutable observations. Changes in debt stock can
// reflect exchange-rate movements, valuation changes, repayments,
// refinancing, arrears, adjustments, disbursement timing — NOT just new
// borrowing. NEVER infer new borrowing from (end - start).
type PublicDebtSnapshot struct {
	ID              ID
	CountryCode     string
	ObservationDate time.Time
	TotalDebtStock  float64
	DomesticDebt    float64
	ExternalDebt    float64
	Currency        string
	ExchangeRateAsOf time.Time
	SourceURL       string
	SourceDocumentID *ID
	CreatedAt       time.Time
}

// BorrowingAuthorization links a borrowing agreement to a parliamentary
// or budget authorization. Spec section 24.
type BorrowingAuthorization struct {
	ID              ID
	BorrowingAgreementID ID
	BudgetAllocationID *ID
	ParliamentaryAuthorizationID *ID
	AuthorizationDate time.Time
	AuthorizationType string // "BUDGET", "PARLIAMENTARY", "STATUTORY"
	SourceURL       string
	CreatedAt       time.Time
}

// DebtEvidenceRef is a reference to authoritative supporting material.
type DebtEvidenceRef struct {
	Kind     string // "TREASURY_REPORT", "CBK_REPORT", "PARLIAMENTARY_DOCUMENT", "GAZETTE", etc.
	ID       string
	SourceURL string
	RetrievedAt time.Time
}

// FiscalReconciliationConflict represents a detected conflict between
// official sources. Spec section 28.
type FiscalReconciliationConflict struct {
	ID              ID
	CountryCode     string
	Subject         string // e.g. "Public debt stock on 2023-06-30"
	ConflictType    string // "AMOUNT_MISMATCH", "DATE_MISMATCH", "CURRENCY_MISMATCH"
	SourceA         DebtEvidenceRef
	SourceB         DebtEvidenceRef
	ValueA          string
	ValueB          string
	Status          string // "OPEN", "INVESTIGATING", "RESOLVED", "UNRESOLVED", "SUPERSEDED"
	Resolution      string
	Reviewer        *ID
	ResolutionDate  *time.Time
	CreatedAt       time.Time
}

// DebtPurpose classifies documented borrowing purposes. Spec section 19.
type DebtPurpose string

const (
	PurposeInfrastructure  DebtPurpose = "INFRASTRUCTURE"
	PurposeBudgetSupport  DebtPurpose = "BUDGET_SUPPORT"
	PurposeHealth         DebtPurpose = "HEALTH"
	PurposeEducation      DebtPurpose = "EDUCATION"
	PurposeEnergy         DebtPurpose = "ENERGY"
	PurposeTransport      DebtPurpose = "TRANSPORT"
	PurposeWater           DebtPurpose = "WATER"
	PurposeAgriculture    DebtPurpose = "AGRICULTURE"
	PurposeICT            DebtPurpose = "ICT"
	PurposeSecurity       DebtPurpose = "SECURITY"
	PurposeRefinancing    DebtPurpose = "REFINANCING"
	PurposeGeneralGovt    DebtPurpose = "GENERAL_GOVERNMENT"
	PurposeOther          DebtPurpose = "OTHER"
	PurposeUnknown        DebtPurpose = "UNKNOWN"
)

// DebtTrendPoint is a single point in a debt time series.
type DebtTrendPoint struct {
	Date            time.Time
	TotalDebtStock  *float64
	DomesticDebt    *float64
	ExternalDebt    *float64
	SourceURL       string
}

// GovernmentDebtSummary aggregates debt data for a government period.
// Spec section 7.
type GovernmentDebtSummary struct {
	AdministrationID    ID
	PresidentialTermID   *ID
	Period              string // e.g. "2017-2022"
	DebtAtStart         *float64
	DebtAtEnd           *float64
	NewBorrowing        *float64
	Repayments          *float64
	DebtService         *float64
	ExternalDebt        *float64
	DomesticDebt         *float64
	Currency            string
	Methodology         string
	SourceURLs          []string
	Disclaimer          string
}

// NO_POLITICAL_PERFORMANCE_SCORE is the canonical disclaimer attached to
// every debt summary. Spec section 37.
const NO_POLITICAL_PERFORMANCE_SCORE = `This summary provides factual fiscal records only. The platform does not
calculate "best borrower", "worst borrower", "debt score", or any political
performance ranking. Users can interpret the underlying measurements
themselves.`
