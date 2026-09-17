// Package memory provides in-memory implementations of the legislation
// repositories. In production these are backed by PostgreSQL; the in-memory
// implementation is used by tests, the API service's local fallback, and the
// Kenya seed dataset.
//
// This file implements ActRepository (issue #202). The structure mirrors
// services/simulation/internal/infrastructure/memory/memory.go: a single
// sync.RWMutex guards three append-only maps (acts, versions, events).
// Tenant isolation is enforced via Act.CountryID — every query is scoped
// to the requested country so two countries' acts never leak into each
// other's listings.
package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// ActRepo is an in-memory ActRepository.
//
// The store is concurrency-safe: a single sync.RWMutex guards all three
// maps. Reads take the read lock; writes take the write lock. Append-only
// collections (versions, events) are never mutated in place — slices are
// copied on read so callers cannot mutate the underlying store.
type ActRepo struct {
	mu       sync.RWMutex
	acts     map[domain.ID]domain.Act
	versions map[domain.ID][]domain.ActVersion
	events   map[domain.ID][]domain.PostAssentEvent
}

// NewActRepo constructs an empty in-memory ActRepository.
func NewActRepo() *ActRepo {
	return &ActRepo{
		acts:     map[domain.ID]domain.Act{},
		versions: map[domain.ID][]domain.ActVersion{},
		events:   map[domain.ID][]domain.PostAssentEvent{},
	}
}

// CreateAct persists a new Act. Returns an error if an Act with the same
// ID already exists (idempotency is the caller's responsibility — the
// platform never silently overwrites canonical civic truth).
func (r *ActRepo) CreateAct(ctx context.Context, a domain.Act) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.acts[a.ID]; exists {
		return fmt.Errorf("act %s already exists", a.ID)
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = a.CreatedAt
	}
	r.acts[a.ID] = a
	return nil
}

// GetAct retrieves an Act by ID. Returns an error if not found.
//
// Tenant isolation note: the ActRepository interface does not carry a
// tenant parameter (the civic domain is country-scoped, not multi-tenant
// in the simulation sense). Tenant isolation is enforced at the ListActs
// layer via ActFilter.CountryID.
func (r *ActRepo) GetAct(ctx context.Context, id domain.ID) (*domain.Act, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.acts[id]
	if !ok {
		return nil, fmt.Errorf("act %s not found", id)
	}
	out := a
	return &out, nil
}

// ListActs lists Acts matching the filter. The filter enforces country
// isolation (only Acts whose CountryID matches filter.CountryID are
// returned). Limit/Offset are applied after filtering.
func (r *ActRepo) ListActs(ctx context.Context, filter domain.ActFilter) ([]domain.Act, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.Act{}
	for _, a := range r.acts {
		// Country isolation: skip acts that don't match the requested country.
		if filter.CountryID != nil && a.CountryID != *filter.CountryID {
			continue
		}
		if len(filter.Status) > 0 && !containsActStatus(filter.Status, a.Status) {
			continue
		}
		out = append(out, a)
	}
	// Apply offset/limit.
	if filter.Offset > 0 {
		if filter.Offset >= len(out) {
			return []domain.Act{}, nil
		}
		out = out[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(out) {
		out = out[:filter.Limit]
	}
	return out, nil
}

// UpdateAct mutates an existing Act. The UpdatedAt field is bumped to now.
func (r *ActRepo) UpdateAct(ctx context.Context, a domain.Act) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.acts[a.ID]; !ok {
		return fmt.Errorf("act %s not found", a.ID)
	}
	a.UpdatedAt = time.Now().UTC()
	r.acts[a.ID] = a
	return nil
}

// AppendVersion appends an immutable ActVersion snapshot. Versions are
// append-only — the historical text is preserved forever (Spec §20).
func (r *ActRepo) AppendVersion(ctx context.Context, v domain.ActVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	r.versions[v.ActID] = append(r.versions[v.ActID], v)
	return nil
}

// ListVersions lists all versions for an Act, oldest-first. Returns an
// empty slice (never nil) if none exist.
func (r *ActRepo) ListVersions(ctx context.Context, actID domain.ID) ([]domain.ActVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.ActVersion, len(r.versions[actID]))
	copy(out, r.versions[actID])
	return out, nil
}

// AppendPostAssentEvent appends an immutable post-assent event (Spec §15).
// Events are append-only — history is never rewritten.
func (r *ActRepo) AppendPostAssentEvent(ctx context.Context, e domain.PostAssentEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	r.events[e.ActID] = append(r.events[e.ActID], e)
	return nil
}

// ListPostAssentEvents lists all post-assent events for an Act. Returns
// an empty slice (never nil) if none exist.
func (r *ActRepo) ListPostAssentEvents(ctx context.Context, actID domain.ID) ([]domain.PostAssentEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.PostAssentEvent, len(r.events[actID]))
	copy(out, r.events[actID])
	return out, nil
}

// RecordAssent implements Spec §15 — the Bill→Act transition is EXPLICIT.
// A PresidentialAssentEvent with status ASSENTED produces an Act; the
// returned Act is persisted in the store. Non-assent statuses (RETURNED,
// WITHHELD, UNKNOWN) do NOT produce an Act and return an error.
//
// The Act's ID is derived deterministically from the Bill's ID by prefixing
// "act-". This makes RecordAssent idempotent on the same Bill — calling it
// twice for the same Bill returns an "act already exists" error from
// CreateAct, which is the desired behavior (the caller must explicitly
// handle this case).
//
// An ASSENT post-assent event is also recorded so the audit trail begins
// at the assent moment.
func (r *ActRepo) RecordAssent(ctx context.Context, ev domain.PresidentialAssentEvent) (*domain.Act, error) {
	if ev.AssentStatus != domain.AssentStatusAssented {
		return nil, fmt.Errorf("RecordAssent: only ASSENTED events produce an Act (got %s)", ev.AssentStatus)
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	act := domain.Act{
		ID:         domain.ID("act-" + string(ev.BillID)),
		BillID:     ev.BillID,
		AssentedAt: ev.Date,
		Status:     domain.ActStatusAssented,
		CreatedAt:  ev.CreatedAt,
		UpdatedAt:  ev.CreatedAt,
	}
	if err := r.CreateAct(ctx, act); err != nil {
		return nil, err
	}
	// Append the ASSENT post-assent event so the audit trail starts at
	// the moment of assent. Failure here is non-fatal — the Act exists.
	assentEvent := domain.PostAssentEvent{
		ID:        domain.ID("ev-assent-" + string(ev.ID)),
		ActID:     act.ID,
		BillID:    &ev.BillID,
		EventType: domain.PostAssentEventAssent,
		EventDate: ev.Date,
		Title:     "Presidential Assent",
		SourceURL: ev.OfficialSource,
		CreatedAt: ev.CreatedAt,
	}
	_ = r.AppendPostAssentEvent(ctx, assentEvent)
	out := act
	return &out, nil
}

func containsActStatus(s []domain.ActStatus, v domain.ActStatus) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// Compile-time assertion that ActRepo implements the domain.ActRepository
// interface. If the interface evolves and ActRepo drifts, this fails the
// build immediately.
var _ domain.ActRepository = (*ActRepo)(nil)
