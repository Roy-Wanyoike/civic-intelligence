// Package evidence owns the platform's claim/citation/evidence graph. The
// intelligence service produces claims (in explanations); the evidence
// service records what supports each claim and validates that AI responses
// only make claims backed by cited source text. When two sources disagree
// on a fact, the evidence service records a SourceConflict — it never
// silently resolves.
package domain

import "time"

// Claim is an atomic statement asserted by some actor (an AI model, a
// human, or another service) about a civic entity. Claims always carry a
// subject (the entity they're about) and a confidence.
type Claim struct {
	ID         string
	Subject    string // bill ID, act ID, etc.
	SubjectType string // "bill", "act", "amendment"
	Text       string
	Source     ClaimSource // "ai", "human", "ingestion_diff"
	Confidence float64
	CreatedAt  time.Time
}

// ClaimSource enumerates the kinds of actors that may produce a claim.
type ClaimSource string

const (
	ClaimSourceAI             ClaimSource = "ai"
	ClaimSourceHuman          ClaimSource = "human"
	ClaimSourceIngestionDiff   ClaimSource = "ingestion_diff"
	ClaimSourceOfficialRecord ClaimSource = "official_record"
)

// Citation is a pointer from a claim to a specific location in a document
// that supports (or refutes) it. The page_number + section + offset triple
// matches the chunk coordinates produced by the documents service, making
// citations precise.
type Citation struct {
	ID            string
	ClaimID       string
	DocumentID    string
	PageNumber    int
	SectionOffset int
	Quote         string
	Relationship  CitationRelationship
	Confidence    float64
	CreatedAt     time.Time
}

// CitationRelationship enumerates whether a citation supports, refutes or
// merely mentions the claim.
type CitationRelationship string

const (
	CitationSupports  CitationRelationship = "supports"
	CitationRefutes    CitationRelationship = "refutes"
	CitationMentions   CitationRelationship = "mentions"
)

// SourceReference is the provenance of a piece of evidence: which source
// the document came from, when it was published, what its content hash is.
type SourceReference struct {
	ID            string
	DocumentID    string
	SourceID      string
	CountryCode   string
	URL           string
	PublishedAt   *time.Time
	ContentHash   string
	IsAuthoritative bool // true if from an official source (e.g. Kenya Gazette)
}

// Evidence is the union of all citations supporting or refuting a single
// claim. The AI validation flow uses this to decide whether to accept or
// reject a candidate fact.
type Evidence struct {
	ClaimID    string
	Citations  []Citation
	Supports   int // count of supports
	Refutes    int // count of refutes
	Conflicts  []SourceConflict
}

// EvidenceSet is a collection of Evidence records produced when validating
// an AI response. It is the value object the intelligence service receives
// back from the CitationValidator.
type EvidenceSet struct {
	ID           string
	RequestID    string
	BillID       string
	Items        []Evidence
	ValidatedAt  time.Time
	ValidatorID  string
}

// SourceConflict records that two sources disagree on a fact. Conflicts are
// NEVER auto-resolved; they are surfaced for human review.
type SourceConflict struct {
	ID            string
	ClaimID       string
	ClaimText     string
	SourceARef    SourceReference
	SourceBRef    SourceReference
	SourceAValue  string
	SourceBValue  string
	DetectedAt    time.Time
	Resolution    ConflictResolution
	ResolvedBy    string
	ResolvedAt    *time.Time
}

// ConflictResolution enumerates the lifecycle of a conflict.
type ConflictResolution string

const (
	ConflictOpen       ConflictResolution = "open"
	ConflictResolved   ConflictResolution = "resolved"
	ConflictEscalated  ConflictResolution = "escalated"
	ConflictDismissed ConflictResolution = "dismissed"
)
