// Package legislation is the canonical civic-domain service. This file
// adds the Wire() constructor (issue #202) that builds an in-memory
// ActRepository and seeds it with country-supplied data.
//
// Architectural note: the legislation service is country-agnostic. Country-
// specific seed data (e.g. Kenya Acts) lives in adapters/kenya/kenya_seed.
// Because adapters/kenya already depends on services/legislation, the
// legislation service CANNOT import adapters/kenya back (that would be a
// circular module dependency). Wire() therefore accepts the seed data as
// parameters — the caller (typically the API service in main.go) sources
// the data from the country adapter and passes it in.
//
// This mirrors services/simulation/wire.go in spirit: a single entry point
// that constructs the in-memory repository, seeds it, and returns it.
package legislation

import (
	"context"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/infrastructure/memory"
)

// Re-export the domain types consumers need. Using type aliases (not
// type definitions) so a legislation.Act IS a domain.Act — no conversion
// needed at the call site.
type (
	ID                     = domain.ID
	Act                    = domain.Act
	ActVersion             = domain.ActVersion
	ActStatus              = domain.ActStatus
	ActFilter              = domain.ActFilter
	ActRepository          = domain.ActRepository
	PostAssentEvent        = domain.PostAssentEvent
	PostAssentEventType    = domain.PostAssentEventType
	PresidentialAssentEvent = domain.PresidentialAssentEvent
	PresidentialAssentStatus = domain.PresidentialAssentStatus
)

// Re-export enums for caller convenience.
const (
	ActStatusAssented  = domain.ActStatusAssented
	ActStatusPublished = domain.ActStatusPublished
	ActStatusCommenced = domain.ActStatusCommenced
	ActStatusAmended   = domain.ActStatusAmended
	ActStatusRepealed  = domain.ActStatusRepealed
	ActStatusUnknown   = domain.ActStatusUnknown

	AssentStatusAssented = domain.AssentStatusAssented
	AssentStatusReturned = domain.AssentStatusReturned
	AssentStatusWithheld = domain.AssentStatusWithheld
	AssentStatusUnknown  = domain.AssentStatusUnknown

	PostAssentEventAssent           = domain.PostAssentEventAssent
	PostAssentEventPublication      = domain.PostAssentEventPublication
	PostAssentEventCommencement     = domain.PostAssentEventCommencement
	PostAssentEventRegulation       = domain.PostAssentEventRegulation
	PostAssentEventImplementation   = domain.PostAssentEventImplementation
	PostAssentEventCourtChallenge  = domain.PostAssentEventCourtChallenge
	PostAssentEventJudicialDecision = domain.PostAssentEventJudicialDecision
	PostAssentEventAmendment        = domain.PostAssentEventAmendment
	PostAssentEventRepeal            = domain.PostAssentEventRepeal
)

// Wire constructs an in-memory ActRepository and seeds it with the supplied
// Acts and post-assent events. The caller is responsible for sourcing the
// seed data — typically from a country adapter's seed package (e.g.
// adapters/kenya/kenya_seed).
//
// Example (from services/api/cmd/main.go):
//
//	acts := kenya_seed.MapActs(kenya_seed.KenyaActs, now)
//	events := kenya_seed.MapPostAssentEvents(kenya_seed.KenyaPostAssentEvents, now)
//	repo := legislation.Wire(acts, events)
//
// In production, callers replace the in-memory repository with a
// Postgres-backed implementation. The application services are agnostic
// to the storage backend.
func Wire(seedActs []Act, seedEvents []PostAssentEvent) ActRepository {
	repo := memory.NewActRepo()
	ctx := context.Background()
	for _, a := range seedActs {
		_ = repo.CreateAct(ctx, a)
	}
	for _, e := range seedEvents {
		_ = repo.AppendPostAssentEvent(ctx, e)
	}
	return repo
}

// WireEmpty constructs an empty in-memory ActRepository with no seed data.
// Useful for tests that want to start from a blank slate.
func WireEmpty() ActRepository {
	return memory.NewActRepo()
}

// Now is re-exported here so callers that build seed data don't have to
// import time directly. It returns the current UTC time.
func Now() time.Time { return time.Now().UTC() }
