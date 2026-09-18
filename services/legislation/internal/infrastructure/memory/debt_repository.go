// Package memory provides in-memory implementations of the legislation
// domain repositories. In production these are backed by PostgreSQL; the
// in-memory implementation is used by tests, the API service's local
// fallback, and the Kenya seed dataset.
//
// Architectural note: this package is internal to services/legislation/.
// Consumers obtain a *DebtRepository via legislation.WireDebtRepository,
// which re-exports the concrete type as the domain.DebtRepository
// interface.
package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// DebtRepository is an in-memory implementation of domain.DebtRepository.
//
// The implementation mirrors the simulation service's memory.go pattern:
// a sync.RWMutex guards every map, write methods take write locks, read
// methods take read locks.
//
// Immutability invariants (Spec section 9):
//   - RecordDebtSnapshot rejects re-writes by ID AND by (countryCode,
//     observationDate). Two snapshots for the same country + observation
//     date would silently overwrite each other; the implementation rejects
//     this with ErrSnapshotImmutable.
//   - RecordGovernmentDebtSummary rejects re-writes by AdministrationID.
//     Summaries are computed once and cached; if the underlying data
//     changes the caller must explicitly invalidate (not currently exposed
//     via the interface) and re-record.
type DebtRepository struct {
	mu sync.RWMutex

	// agreements is keyed by ID for O(1) lookup. A slice mirror preserves
	// insertion order for ListBorrowingAgreements.
	agreements     map[domain.ID]domain.BorrowingAgreement
	agreementOrder []domain.ID

	// disbursements and repayments are append-only slices keyed by
	// BorrowingAgreementID. Spec section 4 — disbursements and repayments
	// are distinct events that may span multiple presidential periods;
	// their temporal order MUST be preserved.
	disbursements map[domain.ID][]domain.DebtDisbursement
	repayments    map[domain.ID][]domain.DebtRepayment

	// services is the derived debt-service aggregate per fiscal period.
	services     map[domain.ID]domain.DebtService
	serviceOrder []domain.ID

	// snapshots is keyed by ID. The (countryCode, observationDate)
	// composite is also tracked in snapshotKeys to enforce immutability.
	snapshots     map[domain.ID]domain.PublicDebtSnapshot
	snapshotKeys  map[string]domain.ID // "CC|YYYY-MM-DD" -> snapshot ID
	snapshotOrder []domain.ID          // chronological insertion order

	// summaries is keyed by AdministrationID. Spec section 7, 37.
	summaries map[domain.ID]domain.GovernmentDebtSummary

	// legislatureSummaries is keyed by LegislatureID. Spec section 19, 37.
	// Each legislature/Parliamentary term carries its own debt summary so
	// the platform can surface per-legislature debt views without
	// re-attributing sovereign borrowing to the Parliament itself.
	legislatureSummaries map[domain.ID]domain.LegislatureDebtSummary

	// conflicts is an append-only slice. Spec section 28 — the platform
	// never silently resolves conflicts; they are surfaced for review.
	conflicts []domain.FiscalReconciliationConflict
}

// NewDebtRepository constructs an empty in-memory debt repository.
func NewDebtRepository() *DebtRepository {
	return &DebtRepository{
		agreements:    map[domain.ID]domain.BorrowingAgreement{},
		disbursements: map[domain.ID][]domain.DebtDisbursement{},
		repayments:    map[domain.ID][]domain.DebtRepayment{},
		services:      map[domain.ID]domain.DebtService{},
		snapshots:            map[domain.ID]domain.PublicDebtSnapshot{},
		snapshotKeys:         map[string]domain.ID{},
		summaries:            map[domain.ID]domain.GovernmentDebtSummary{},
		legislatureSummaries: map[domain.ID]domain.LegislatureDebtSummary{},
		conflicts:            []domain.FiscalReconciliationConflict{},
	}
}

// snapshotKey is the composite key used to enforce immutability per
// (countryCode, observationDate). Two snapshots for the same country on
// the same observation date would silently overwrite each other; the
// implementation rejects this.
func snapshotKey(countryCode string, observationDate time.Time) string {
	return fmt.Sprintf("%s|%s", countryCode, observationDate.UTC().Format("2006-01-02"))
}

// CreateBorrowingAgreement persists a new borrowing agreement.
// Returns domain.ErrBorrowingAgreementAlreadyExists if the ID is taken.
func (r *DebtRepository) CreateBorrowingAgreement(_ context.Context, agreement domain.BorrowingAgreement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.agreements[agreement.ID]; exists {
		return fmt.Errorf("%w: %s", domain.ErrBorrowingAgreementAlreadyExists, agreement.ID)
	}
	r.agreements[agreement.ID] = agreement
	r.agreementOrder = append(r.agreementOrder, agreement.ID)
	return nil
}

// GetBorrowingAgreement retrieves a single borrowing agreement by ID.
// Returns domain.ErrBorrowingAgreementNotFound if no agreement matches.
func (r *DebtRepository) GetBorrowingAgreement(_ context.Context, id domain.ID) (*domain.BorrowingAgreement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.agreements[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrBorrowingAgreementNotFound, id)
	}
	out := a
	return &out, nil
}

// ListBorrowingAgreements returns agreements matching the filter.
// A zero-value filter returns every agreement in insertion order.
func (r *DebtRepository) ListBorrowingAgreements(_ context.Context, filter domain.DebtFilter) ([]domain.BorrowingAgreement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := []domain.BorrowingAgreement{}
	for _, id := range r.agreementOrder {
		a := r.agreements[id]
		if !debtFilterMatches(a, filter) {
			continue
		}
		out = append(out, a)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

// AppendDisbursement appends a disbursement event to a borrowing
// agreement. Spec section 3 — disbursements are money actually released,
// distinct from new borrowing and from commitments.
//
// The borrowing agreement MUST already exist; otherwise the disbursement
// is orphaned and the call returns an error.
func (r *DebtRepository) AppendDisbursement(_ context.Context, d domain.DebtDisbursement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.agreements[d.BorrowingAgreementID]; !ok {
		return fmt.Errorf("%w: %s", domain.ErrBorrowingAgreementNotFound, d.BorrowingAgreementID)
	}
	r.disbursements[d.BorrowingAgreementID] = append(r.disbursements[d.BorrowingAgreementID], d)
	return nil
}

// AppendRepayment appends a principal (and optional interest) repayment
// event. Spec section 4 — repayments occurring during a particular
// presidential period do NOT change the agreement's CONTRACTED_DURING
// attribution.
func (r *DebtRepository) AppendRepayment(_ context.Context, p domain.DebtRepayment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.agreements[p.BorrowingAgreementID]; !ok {
		return fmt.Errorf("%w: %s", domain.ErrBorrowingAgreementNotFound, p.BorrowingAgreementID)
	}
	r.repayments[p.BorrowingAgreementID] = append(r.repayments[p.BorrowingAgreementID], p)
	return nil
}

// AppendDebtService appends a derived debt-service aggregate for a fiscal
// period. Spec section 3 — debt service = principal + interest + associated
// payments.
//
// If a DebtService with the same ID already exists, the new record
// replaces it (debt service is a derived aggregate, not an immutable
// observation; re-computation is allowed).
func (r *DebtRepository) AppendDebtService(_ context.Context, s domain.DebtService) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.services[s.ID]; !exists {
		r.serviceOrder = append(r.serviceOrder, s.ID)
	}
	r.services[s.ID] = s
	return nil
}

// RecordDebtSnapshot records a point-in-time observation of total public
// debt. Spec section 9 — snapshots are immutable; the same ID or the same
// (countryCode, observationDate) MUST be rejected with
// domain.ErrSnapshotImmutable.
func (r *DebtRepository) RecordDebtSnapshot(_ context.Context, snap domain.PublicDebtSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.snapshots[snap.ID]; exists {
		return fmt.Errorf("%w: snapshot ID %s already recorded", domain.ErrSnapshotImmutable, snap.ID)
	}
	key := snapshotKey(snap.CountryCode, snap.ObservationDate)
	if existingID, exists := r.snapshotKeys[key]; exists {
		return fmt.Errorf("%w: snapshot %s already records (%s, %s)",
			domain.ErrSnapshotImmutable, existingID, snap.CountryCode,
			snap.ObservationDate.UTC().Format("2006-01-02"))
	}
	r.snapshots[snap.ID] = snap
	r.snapshotKeys[key] = snap.ID
	r.snapshotOrder = append(r.snapshotOrder, snap.ID)
	return nil
}

// ListDebtSnapshots returns snapshots for a country observed between
// from and to (inclusive on both ends). Results are returned in
// chronological order by ObservationDate.
func (r *DebtRepository) ListDebtSnapshots(_ context.Context, countryCode string, from, to time.Time) ([]domain.PublicDebtSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := []domain.PublicDebtSnapshot{}
	for _, id := range r.snapshotOrder {
		s := r.snapshots[id]
		if s.CountryCode != countryCode {
			continue
		}
		// Inclusive on both ends. Compare on date-only precision (UTC
		// midnight) to match how snapshots are typically keyed.
		od := s.ObservationDate.UTC()
		if od.Before(from.UTC()) || od.After(to.UTC()) {
			continue
		}
		out = append(out, s)
	}
	// Defensive sort by ObservationDate — insertion order may not match
	// chronological order if the seeder inserted out of order.
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ObservationDate.Before(out[j].ObservationDate)
	})
	return out, nil
}

// GetGovernmentDebtSummary returns the cached summary for an
// administration. Returns domain.ErrGovernmentDebtSummaryNotFound if no
// summary exists.
func (r *DebtRepository) GetGovernmentDebtSummary(_ context.Context, adminID domain.ID) (*domain.GovernmentDebtSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.summaries[adminID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrGovernmentDebtSummaryNotFound, adminID)
	}
	out := s
	return &out, nil
}

// RecordGovernmentDebtSummary stores a computed administration summary.
// Returns domain.ErrGovernmentDebtSummaryAlreadyExists if a summary for the
// same AdministrationID is already present.
func (r *DebtRepository) RecordGovernmentDebtSummary(_ context.Context, summary domain.GovernmentDebtSummary) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.summaries[summary.AdministrationID]; exists {
		return fmt.Errorf("%w: %s", domain.ErrGovernmentDebtSummaryAlreadyExists, summary.AdministrationID)
	}
	r.summaries[summary.AdministrationID] = summary
	return nil
}

// GetLegislatureDebtSummary returns the cached summary for a legislature
// (Parliamentary term). Returns domain.ErrLegislatureDebtSummaryNotFound if
// no summary exists.
func (r *DebtRepository) GetLegislatureDebtSummary(_ context.Context, legislatureID domain.ID) (*domain.LegislatureDebtSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.legislatureSummaries[legislatureID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", domain.ErrLegislatureDebtSummaryNotFound, legislatureID)
	}
	out := s
	return &out, nil
}

// RecordLegislatureDebtSummary stores a computed legislature summary.
// Returns domain.ErrLegislatureDebtSummaryAlreadyExists if a summary for the
// same LegislatureID is already present.
func (r *DebtRepository) RecordLegislatureDebtSummary(_ context.Context, summary domain.LegislatureDebtSummary) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.legislatureSummaries[summary.LegislatureID]; exists {
		return fmt.Errorf("%w: %s", domain.ErrLegislatureDebtSummaryAlreadyExists, summary.LegislatureID)
	}
	r.legislatureSummaries[summary.LegislatureID] = summary
	return nil
}

// AppendReconciliationConflict records a detected conflict between official
// sources. Spec section 28 — conflicts are append-only; the platform never
// silently resolves them.
func (r *DebtRepository) AppendReconciliationConflict(_ context.Context, c domain.FiscalReconciliationConflict) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conflicts = append(r.conflicts, c)
	return nil
}

// ListReconciliationConflicts returns all recorded conflicts in insertion
// order. This is an extension method on the concrete type — not part of
// the domain.DebtRepository interface — used by tests and the
// reconciliation UI to surface open conflicts.
func (r *DebtRepository) ListReconciliationConflicts() []domain.FiscalReconciliationConflict {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.FiscalReconciliationConflict, len(r.conflicts))
	copy(out, r.conflicts)
	return out
}

// ListDisbursements returns the disbursements recorded against a
// borrowing agreement, in chronological order. Concrete-only extension
// used by tests.
func (r *DebtRepository) ListDisbursements(agreementID domain.ID) []domain.DebtDisbursement {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src := r.disbursements[agreementID]
	out := make([]domain.DebtDisbursement, len(src))
	copy(out, src)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}

// ListRepayments returns the repayments recorded against a borrowing
// agreement, in chronological order. Concrete-only extension used by
// tests.
func (r *DebtRepository) ListRepayments(agreementID domain.ID) []domain.DebtRepayment {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src := r.repayments[agreementID]
	out := make([]domain.DebtRepayment, len(src))
	copy(out, src)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}

// debtFilterMatches applies DebtFilter to a BorrowingAgreement. The filter
// is AND-semantic across all populated fields.
func debtFilterMatches(a domain.BorrowingAgreement, f domain.DebtFilter) bool {
	if f.CountryCode != "" && a.CountryCode != f.CountryCode {
		return false
	}
	if f.GovernmentAdministrationID != nil && a.GovernmentAdministrationID != *f.GovernmentAdministrationID {
		return false
	}
	if f.PresidentialTermID != nil {
		if a.PresidentialTermID == nil || *a.PresidentialTermID != *f.PresidentialTermID {
			return false
		}
	}
	if f.LegislatureID != nil {
		if a.LegislatureID == nil || *a.LegislatureID != *f.LegislatureID {
			return false
		}
	}
	if f.FiscalYearID != nil {
		if a.FiscalYearID == nil || *a.FiscalYearID != *f.FiscalYearID {
			return false
		}
	}
	if f.CreditorID != nil && a.CreditorID != *f.CreditorID {
		return false
	}
	if f.CreditorCategory != nil && a.CreditorCategory != *f.CreditorCategory {
		return false
	}
	if f.DomesticOrExternal != nil && a.DomesticOrExternal != *f.DomesticOrExternal {
		return false
	}
	if f.Purpose != nil && a.Purpose != string(*f.Purpose) {
		return false
	}
	if f.Status != "" && a.Status != f.Status {
		return false
	}
	if f.From != nil {
		if a.ContractDate == nil || a.ContractDate.Before(*f.From) {
			return false
		}
	}
	if f.To != nil {
		if a.ContractDate == nil || a.ContractDate.After(*f.To) {
			return false
		}
	}
	return true
}
