// Package domain — DebtRepository interface and supporting types.
//
// This file defines the persistence boundary for the Public Debt &
// Borrowing Intelligence domain (Spec sections 1-37). The domain layer
// depends only on this interface — never on database/sql, net/http, or any
// storage driver.
//
// Architectural rules:
//  1. Snapshots are immutable observations. RecordDebtSnapshot MUST reject
//     any attempt to overwrite an existing snapshot (Spec section 9).
//  2. The platform NEVER attributes sovereign borrowing personally to a
//     president (Spec section 35). Methods return BorrowingAgreement
//     records; the caller is responsible for phrasing the attribution
//     correctly.
//  3. The platform distinguishes CONTRACTED_DURING, DISBURSED_DURING,
//     REPAID_DURING, OUTSTANDING_DURING, and REFINANCED_DURING (Spec
//     section 4). Append* methods preserve this distinction by recording
//     raw events; they never collapse semantics.
package domain

import (
	"context"
	"errors"
	"time"
)

// ErrSnapshotImmutable is returned by RecordDebtSnapshot when a snapshot
// with the same ID (or the same country + observation date) has already
// been recorded. Snapshots are immutable observations (Spec section 9) —
// they cannot be silently overwritten.
//
// If a correction is required, the caller MUST create a new snapshot with
// a different ID. The original observation stands as the historical record.
var ErrSnapshotImmutable = errors.New("debt snapshot is immutable and cannot be re-written")

// ErrGovernmentDebtSummaryAlreadyExists is returned by
// RecordGovernmentDebtSummary when a summary for the same administration
// has already been recorded. Summaries are computed once and cached; if
// the underlying data changes, the caller should recompute and record a
// new summary under a different administration ID (or the caller should
// delete and re-add — which is not exposed via this interface).
var ErrGovernmentDebtSummaryAlreadyExists = errors.New("government debt summary already exists; summaries are immutable cached values")

// ErrBorrowingAgreementAlreadyExists is returned by CreateBorrowingAgreement
// when an agreement with the same ID is already present.
var ErrBorrowingAgreementAlreadyExists = errors.New("borrowing agreement already exists")

// ErrBorrowingAgreementNotFound is returned by GetBorrowingAgreement when no
// agreement matches the requested ID.
var ErrBorrowingAgreementNotFound = errors.New("borrowing agreement not found")

// ErrGovernmentDebtSummaryNotFound is returned by GetGovernmentDebtSummary
// when no summary exists for the requested administration.
var ErrGovernmentDebtSummaryNotFound = errors.New("government debt summary not found")

// DebtFilter narrows ListBorrowingAgreements results. All fields are
// optional; a zero-value DebtFilter returns every agreement.
//
// Date filters apply to ContractDate. Spec section 4 — CONTRACTED_DURING
// is the canonical attribution axis.
type DebtFilter struct {
	CountryCode                string
	GovernmentAdministrationID *ID
	PresidentialTermID         *ID
	LegislatureID              *ID
	FiscalYearID               *ID
	CreditorID                 *ID
	CreditorCategory           *CreditorCategory
	DomesticOrExternal         *DomesticOrExternal
	Purpose                    *DebtPurpose
	Status                     string
	// From and To bound ContractDate. Inclusive on both ends.
	From *time.Time
	To   *time.Time
	// Limit caps the number of results. Zero or negative means unlimited.
	Limit int
}

// DebtRepository is the persistence interface for the public debt domain.
//
// The interface is split into write-side methods (Create*, Append*,
// Record*) and read-side methods (Get*, List*). Write methods that record
// immutable observations (RecordDebtSnapshot, RecordGovernmentDebtSummary)
// reject re-writes — Spec section 9.
//
// Implementations live in infrastructure/. The in-memory implementation is
// used by tests, the API service's local fallback, and the Kenya seed
// dataset. A Postgres-backed implementation will replace it in production
// (issue #203 tracks the migration).
type DebtRepository interface {
	// CreateBorrowingAgreement persists a new borrowing agreement.
	// Returns ErrBorrowingAgreementAlreadyExists if the ID is taken.
	CreateBorrowingAgreement(ctx context.Context, agreement BorrowingAgreement) error

	// GetBorrowingAgreement retrieves a single borrowing agreement by ID.
	// Returns ErrBorrowingAgreementNotFound if no agreement matches.
	GetBorrowingAgreement(ctx context.Context, id ID) (*BorrowingAgreement, error)

	// ListBorrowingAgreements returns agreements matching the filter.
	// A zero-value filter returns every agreement.
	ListBorrowingAgreements(ctx context.Context, filter DebtFilter) ([]BorrowingAgreement, error)

	// AppendDisbursement appends a disbursement event to a borrowing
	// agreement. Spec section 3 — disbursements are money actually released,
	// distinct from new borrowing (a new agreement) and from commitments.
	AppendDisbursement(ctx context.Context, disbursement DebtDisbursement) error

	// AppendRepayment appends a principal (and optional interest) repayment
	// event. Spec section 4 — repayments occurring during a particular
	// presidential period do NOT change the agreement's CONTRACTED_DURING
	// attribution.
	AppendRepayment(ctx context.Context, repayment DebtRepayment) error

	// AppendDebtService appends a derived debt-service aggregate for a
	// fiscal period. Spec section 3 — debt service = principal + interest
	// + associated payments.
	AppendDebtService(ctx context.Context, service DebtService) error

	// RecordDebtSnapshot records a point-in-time observation of total
	// public debt. Spec section 9 — snapshots are immutable; the same ID
	// (or country + observation date) MUST be rejected with
	// ErrSnapshotImmutable.
	//
	// IMPORTANT: changes in debt stock can reflect exchange-rate movements,
	// valuation changes, repayments, refinancing, arrears, adjustments, and
	// disbursement timing — NOT just new borrowing. NEVER infer new
	// borrowing from (end - start).
	RecordDebtSnapshot(ctx context.Context, snapshot PublicDebtSnapshot) error

	// ListDebtSnapshots returns snapshots for a country observed between
	// from and to (inclusive). Results are returned in chronological order.
	ListDebtSnapshots(ctx context.Context, countryCode string, from, to time.Time) ([]PublicDebtSnapshot, error)

	// GetGovernmentDebtSummary returns the cached summary for an
	// administration. Spec section 7 — summaries aggregate snapshots +
	// borrowing events for a presidential period; they are not political
	// performance scores.
	//
	// Returns ErrGovernmentDebtSummaryNotFound if no summary exists.
	GetGovernmentDebtSummary(ctx context.Context, adminID ID) (*GovernmentDebtSummary, error)

	// RecordGovernmentDebtSummary stores a computed administration summary.
	// Returns ErrGovernmentDebtSummaryAlreadyExists if a summary for the
	// same AdministrationID is already present — summaries are immutable
	// cached values (Spec section 7, 37).
	RecordGovernmentDebtSummary(ctx context.Context, summary GovernmentDebtSummary) error

	// AppendReconciliationConflict records a detected conflict between
	// official sources. Spec section 28 — the platform never silently
	// resolves conflicts; it surfaces them for human review.
	AppendReconciliationConflict(ctx context.Context, conflict FiscalReconciliationConflict) error
}
