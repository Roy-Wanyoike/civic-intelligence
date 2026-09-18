package domain

import "time"

// Bill is the central aggregate root of the legislation bounded context. It
// represents a piece of proposed legislation and owns its lifecycle (the
// current stage, the sequence of BillEvents, and the set of BillVersions).
//
// The Bill entity is country-agnostic. The country-specific behaviour — what
// stages exist and which transitions between them are legal — is supplied
// by an injected BillStageTransitionValidator built from the country's
// adapter. This means the same Bill type works for Kenya today and for any
// future country we onboard.
type Bill struct {
	id           ID
	countryID    ID
	title        string
	shortTitle   string
	houseID      ID
	sponsorID    *ID
	currentStage string
	introducedAt time.Time
	updatedAt    time.Time
	versions     []BillVersion
	events       []BillEvent
	amendments   []Amendment
}

// NewBill constructs a Bill at its initial stage. The initial stage code is
// provided by the caller (typically the country adapter); it must be the
// first stage of the country's lifecycle.
func NewBill(id, countryID ID, title, shortTitle string, houseID ID, sponsor *ID, initialStage string, now time.Time) *Bill {
	return &Bill{
		id:           id,
		countryID:    countryID,
		title:        title,
		shortTitle:   shortTitle,
		houseID:      houseID,
		sponsorID:    sponsor,
		currentStage: initialStage,
		introducedAt: now,
		updatedAt:    now,
	}
}

// ID returns the bill's identifier.
func (b *Bill) ID() ID { return b.id }

// CountryID returns the country the bill belongs to.
func (b *Bill) CountryID() ID { return b.countryID }

// Title returns the bill's full title.
func (b *Bill) Title() string { return b.title }

// ShortTitle returns the bill's short title.
func (b *Bill) ShortTitle() string { return b.shortTitle }

// HouseID returns the house the bill originated in.
func (b *Bill) HouseID() ID { return b.houseID }

// SponsorID returns the bill's sponsor (may be nil).
func (b *Bill) SponsorID() *ID { return b.sponsorID }

// CurrentStage returns the bill's current stage code.
func (b *Bill) CurrentStage() string { return b.currentStage }

// IntroducedAt returns when the bill was introduced.
func (b *Bill) IntroducedAt() time.Time { return b.introducedAt }

// UpdatedAt returns when the bill was last updated.
func (b *Bill) UpdatedAt() time.Time { return b.updatedAt }

// Versions returns the bill's version history (oldest first).
func (b *Bill) Versions() []BillVersion { return b.versions }

// Events returns the bill's event history (oldest first).
func (b *Bill) Events() []BillEvent { return b.events }

// Amendments returns the bill's amendments.
func (b *Bill) Amendments() []Amendment { return b.amendments }

// ApplyTransition mutates the bill by transitioning it to a new stage. The
// validator parameter is responsible for ensuring the transition is legal;
// the Bill entity itself does not know any country-specific stage rules.
// On success the bill records a new BillEvent of type "stage_change" and
// updates its updatedAt timestamp.
func (b *Bill) ApplyTransition(validator BillStageTransitionValidator, newStage, reason string, occurredAt time.Time, eventID ID) error {
	if validator == nil {
		return ErrNoValidator
	}
	if err := validator.ValidateTransition(b.currentStage, newStage); err != nil {
		return err
	}
	old := b.currentStage
	b.currentStage = newStage
	b.updatedAt = occurredAt
	b.events = append(b.events, BillEvent{
		id:          eventID,
		billID:      b.id,
		kind:        BillEventKindStageChange,
		fromStage:   old,
		toStage:     newStage,
		reason:      reason,
		occurredAt:  occurredAt,
	})
	return nil
}

// AddVersion records a new version of the bill's text. Versions are
// append-only; the new version becomes the current version.
func (b *Bill) AddVersion(v BillVersion) {
	b.versions = append(b.versions, v)
	b.updatedAt = v.PublishedAt
}

// CurrentVersion returns the latest published version, or nil if none.
func (b *Bill) CurrentVersion() *BillVersion {
	if len(b.versions) == 0 {
		return nil
	}
	return &b.versions[len(b.versions)-1]
}

// AddAmendment records an amendment against this bill.
func (b *Bill) AddAmendment(a Amendment) {
	b.amendments = append(b.amendments, a)
	b.updatedAt = a.CreatedAt
}

// RecordEvent appends a free-form bill event (e.g. a vote, a publication, a
// referral to committee) without changing the bill's stage.
func (b *Bill) RecordEvent(e BillEvent) {
	b.events = append(b.events, e)
	b.updatedAt = e.occurredAt
}

// BillEventKind enumerates the kinds of events a bill may record.
type BillEventKind string

const (
	BillEventKindStageChange BillEventKind = "stage_change"
	BillEventKindPublication BillEventKind = "publication"
	BillEventKindVote        BillEventKind = "vote"
	BillEventKindReferral    BillEventKind = "referral"
	BillEventKindAssent      BillEventKind = "assent"
	BillEventKindCommencement BillEventKind = "commencement"
	BillEventKindWithdrawal  BillEventKind = "withdrawal"
	BillEventKindOther       BillEventKind = "other"
)

// BillEvent is an immutable record of something that happened to a bill. The
// union of fromStage/toStage captures the before/after when the event is a
// stage change; for other kinds they may both be the current stage.
type BillEvent struct {
	id         ID
	billID     ID
	kind       BillEventKind
	fromStage  string
	toStage    string
	reason     string
	occurredAt time.Time
	actor      string
	metadata   map[string]string
}

// NewBillEvent constructs a BillEvent.
func NewBillEvent(id, billID ID, kind BillEventKind, from, to, reason, actor string, occurredAt time.Time) BillEvent {
	return BillEvent{
		id: id, billID: billID, kind: kind,
		fromStage: from, toStage: to, reason: reason,
		actor: actor, occurredAt: occurredAt,
	}
}

// ID returns the event's identifier.
func (e BillEvent) ID() ID { return e.id }

// BillID returns the bill the event belongs to.
func (e BillEvent) BillID() ID { return e.billID }

// Kind returns the event kind.
func (e BillEvent) Kind() BillEventKind { return e.kind }

// FromStage returns the stage before the event.
func (e BillEvent) FromStage() string { return e.fromStage }

// ToStage returns the stage after the event.
func (e BillEvent) ToStage() string { return e.toStage }

// Reason returns the human-readable reason for the event.
func (e BillEvent) Reason() string { return e.reason }

// OccurredAt returns the event's timestamp.
func (e BillEvent) OccurredAt() time.Time { return e.occurredAt }

// Actor returns the principal ID (or "" for system events).
func (e BillEvent) Actor() string { return e.actor }

// Metadata returns the event's metadata map (may be nil).
func (e BillEvent) Metadata() map[string]string { return e.metadata }
