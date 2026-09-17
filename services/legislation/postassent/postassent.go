// Package postassent is the public façade for the legislation module's
// post-assent lifecycle domain logic.
//
// The canonical types and the AuditForAct algorithm live in the internal
// domain package (services/legislation/internal/domain) so they cannot be
// imported by other modules. This package re-exports the subset of those
// types + the AuditForAct function so consumers (the API service,
// simulations, exporters) can delegate to the single source of truth
// without duplicating the audit logic.
//
// Type aliases are used deliberately so that values constructed by callers
// are bit-for-bit identical to the internal domain types — no marshalling,
// no conversion, no drift. If the internal domain adds a field, the
// alias automatically picks it up.
//
// NOTE: This package does NOT depend on the ActRepository (issue ENG-B1).
// It only exposes the pure domain function and types; persistence is the
// caller's responsibility.
package postassent

import (
	domain "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// ID is the platform-wide identifier type for entities (alias of domain.ID).
type ID = domain.ID

// Act is a Bill that has received Presidential Assent and been published
// as law (alias of domain.Act).
type Act = domain.Act

// ActVersion is an immutable snapshot of an Act at a point in time
// (alias of domain.ActVersion).
type ActVersion = domain.ActVersion

// PostAssentEvent is a single event in the post-assent lifecycle of an Act
// (alias of domain.PostAssentEvent).
type PostAssentEvent = domain.PostAssentEvent

// PostAssentEventType enumerates the kinds of events tracked after a Bill
// becomes an Act (alias of domain.PostAssentEventType).
type PostAssentEventType = domain.PostAssentEventType

// PresidentialAssentEvent is a first-class event representing the
// president's action on a Bill (alias of domain.PresidentialAssentEvent).
type PresidentialAssentEvent = domain.PresidentialAssentEvent

// PresidentialAssentStatus records the outcome of a presidential assent
// event (alias of domain.PresidentialAssentStatus).
type PresidentialAssentStatus = domain.PresidentialAssentStatus

// ActStatus is the lifecycle state of an Act of Parliament
// (alias of domain.ActStatus).
type ActStatus = domain.ActStatus

// ActAuditStatus tracks the platform's coverage of an Act's post-assent
// lifecycle (alias of domain.ActAuditStatus).
type ActAuditStatus = domain.ActAuditStatus

// LegislativeLifecycleAudit aggregates the audit status of an Act across
// every post-assent dimension (alias of domain.LegislativeLifecycleAudit).
type LegislativeLifecycleAudit = domain.LegislativeLifecycleAudit

// EvidenceRef is a reference to a verified civic fact or document
// (alias of domain.EvidenceRef).
type EvidenceRef = domain.EvidenceRef

// AuditForAct inspects an Act and returns its lifecycle audit. This
// delegates to domain.AuditForAct — the single source of truth for
// post-assent audit logic.
//
// Spec section 16: "No commencement notice found" does NOT mean "the Act
// never commenced". Instead, the audit returns ActAuditNotVerified for
// commencement.
func AuditForAct(act Act, events []PostAssentEvent) LegislativeLifecycleAudit {
	return domain.AuditForAct(act, events)
}

// --- Constants re-exported so callers don't need to reach into the
// internal package to compare audit statuses or event types. ---

// Audit status constants (alias of the domain.ActAudit* constants).
const (
	AssentConfirmed        = domain.ActAuditAssentConfirmed
	PublicationConfirmed   = domain.ActAuditPublicationConfirmed
	CommencementConfirmed  = domain.ActAuditCommencementConfirmed
	ImplementationTracked  = domain.ActAuditImplementationTracked
	RegulationsTracked     = domain.ActAuditRegulationsTracked
	JudicialHistoryTracked = domain.ActAuditJudicialHistoryTracked
	AmendmentsTracked      = domain.ActAuditAmendmentsTracked
	RepealStatusTracked    = domain.ActAuditRepealStatusTracked
	AuditComplete          = domain.ActAuditComplete
	PartiallyTracked       = domain.ActAuditPartiallyTracked
	DataGap                = domain.ActAuditDataGap
	ConflictingSources     = domain.ActAuditConflictingSources
	NotVerified            = domain.ActAuditNotVerified
)

// Post-assent event type constants (alias of the domain.PostAssentEvent*
// constants).
const (
	EventAssent           = domain.PostAssentEventAssent
	EventPublication      = domain.PostAssentEventPublication
	EventCommencement     = domain.PostAssentEventCommencement
	EventRegulation       = domain.PostAssentEventRegulation
	EventImplementation  = domain.PostAssentEventImplementation
	EventCourtChallenge   = domain.PostAssentEventCourtChallenge
	EventJudicialDecision = domain.PostAssentEventJudicialDecision
	EventAmendment        = domain.PostAssentEventAmendment
	EventRepeal           = domain.PostAssentEventRepeal
)

// Act status constants (alias of the domain.ActStatus* constants).
const (
	StatusAssented  = domain.ActStatusAssented
	StatusPublished = domain.ActStatusPublished
	StatusCommenced = domain.ActStatusCommenced
	StatusAmended   = domain.ActStatusAmended
	StatusRepealed  = domain.ActStatusRepealed
	StatusUnknown   = domain.ActStatusUnknown
)
