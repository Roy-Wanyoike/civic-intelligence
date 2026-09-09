// Package intelligence is the Go orchestrator for AI on the Civic Intelligence
// Platform. It owns: Summary, Explanation, Question, Answer, Topic,
// Classification, ImpactAnalysis and — most importantly — CandidateFact.
//
// The Go service does NOT run inference itself; it orchestrates calls to
// the Python AI gateway (services/ai, owned by the main agent) and the
// evidence service. The Python gateway may propose candidate facts; this Go
// service validates them via the CandidateFactValidator and publishes
// accepted facts as events. The legislation service consumes accepted facts
// and is the only entity allowed to mutate the canonical bill state.
package intelligence

import "time"

// Question is a user-asked question about a civic entity (typically a bill).
// Questions produce Answers; both are persisted for audit.
type Question struct {
	ID          string
	UserID      string
	BillID      string
	Text        string
	Audience    string // "general", "expert", "youth"
	CreatedAt   time.Time
}

// Answer is the AI's response to a Question. The response body is structured
// (sections, claims) so that the evidence service can validate citations.
type Answer struct {
	ID          string
	QuestionID  string
	Body        string
	Sections    []AnswerSection
	Claims      []ClaimRef
	ModelID     string
	LatencyMS   int64
	GeneratedAt time.Time
}

// AnswerSection is a titled chunk of an Answer (e.g. "Plain English summary",
// "Impact on counties", "Key stakeholders").
type AnswerSection struct {
	ID        string
	AnswerID  string
	Title     string
	Body      string
	Order     int
}

// ClaimRef is a pointer from an Answer to a Claim that must be validated by
// the evidence service. The Validator field is set after validation.
type ClaimRef struct {
	ID          string
	AnswerID    string
	Text        string
	Subject     string // bill ID
	SubjectType string // "bill"
	Confidence  float64
	Validator   string // "evidence_v1" once validated
	IsValidated bool
}

// Explanation is a precomputed, multi-paragraph explainer for a bill or
// amendment. Explanations are produced asynchronously and cached.
type Explanation struct {
	ID          string
	BillID      string
	VersionNo   int
	Audience    string
	Title       string
	SummaryText string
	BodyText    string
	SectionRefs []SectionRef
	ModelID     string
	Confidence  float64
	GeneratedAt time.Time
}

// SectionRef points to a section of the source document an explanation is
// grounded in. This is what makes the explanation citable.
type SectionRef struct {
	DocumentID    string
	PageNumber    int
	SectionOffset int
	Quote         string
}

// Summary is a one-paragraph TL;DR of a bill.
type Summary struct {
	ID         string
	BillID     string
	Text       string
	ModelID    string
	Confidence float64
	GeneratedAt time.Time
}

// Topic is a high-level subject tag (e.g. "health", "finance", "counties")
// assigned to a bill by classification.
type Topic struct {
	ID        string
	BillID    string
	Name      string
	Confidence float64
}

// Classification is the assignment of a bill to a Topic (or multiple
// topics) by an AI classifier.
type Classification struct {
	ID        string
	BillID    string
	Topics    []Topic
	ModelID   string
	Confidence float64
	CreatedAt time.Time
}

// ImpactAnalysis is a structured assessment of how a bill affects specific
// groups (counties, sectors, demographics). Always citable; never silently
// accepted as canonical truth.
type ImpactAnalysis struct {
	ID          string
	BillID      string
	Sector      string
	Summary     string
	AffectedGroups []string
	SectionRefs []SectionRef
	Confidence  float64
	GeneratedAt time.Time
}

// CandidateFact is a fact about a civic entity that AI has proposed. It is
// NOT canonical truth; it must be validated by the CandidateFactValidator
// and only after acceptance may it be materialised by the legislation
// service (which consumes the accepted event).
type CandidateFact struct {
	ID         string
	BillID     string
	Subject    string
	Claim      string
	Confidence float64
	Source     CandidateFactSource
	Evidence   []CandidateFactEvidence
	Status     CandidateFactStatus
	ProposedAt time.Time
	ValidatedAt *time.Time
	Validator   string
}

// CandidateFactSource enumerates the origins of a candidate fact.
type CandidateFactSource string

const (
	CandidateFactSourceLLM            CandidateFactSource = "llm"
	CandidateFactSourceIngestionDiff  CandidateFactSource = "ingestion_diff"
	CandidateFactSourceHuman          CandidateFactSource = "human"
)

// CandidateFactStatus enumerates the lifecycle states of a candidate fact.
type CandidateFactStatus string

const (
	CandidateFactStatusProposed  CandidateFactStatus = "proposed"
	CandidateFactStatusAccepted  CandidateFactStatus = "accepted"
	CandidateFactStatusRejected  CandidateFactStatus = "rejected"
	CandidateFactStatusNeedsReview CandidateFactStatus = "needs_review"
)

// CandidateFactEvidence is a pointer to the source text that supports the
// candidate fact. Without at least one piece of evidence the candidate is
// always rejected.
type CandidateFactEvidence struct {
	DocumentID    string
	PageNumber    int
	SectionOffset int
	Quote         string
	Confidence    float64
}
