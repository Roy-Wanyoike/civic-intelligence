package domain

import (
	"fmt"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// BillVersion is a snapshot of the bill's text at a point in time. Versions
// are append-only; the canonical text is whatever the most recent version
// contains. The SourceDocumentID points to the raw document fetched by
// ingestion and parsed by documents; legislation never owns the bytes.
type BillVersion struct {
	ID              ID
	BillID          ID
	VersionNumber   int
	Title           string
	TextHash        string
	SourceDocumentID string // documents service ID
	PublishedAt     time.Time
	ChangesSummary string
	ClauseRefs      []ClauseRef
}

// ClauseRef is a reference to a specific clause in a bill version, including
// the page and section of the source document it came from. This enables
// precise citations in AI explanations.
type ClauseRef struct {
	ClauseNumber int
	Title        string
	PageNumber   int
	SectionOffset int
	DocumentID   string
}

// Amendment is a proposed change to a bill's text, proposed by a sponsor.
type Amendment struct {
	ID          ID
	BillID      ID
	VersionNo   int
	SponsorID   ID
	ClauseRef   string // e.g. "clause 12(3)(b)"
	Type        string // "insertion", "deletion", "substitution"
	Text        string
	Justification string
	Status      AmendmentStatus
	CreatedAt   time.Time
}

// AmendmentStatus enumerates amendment lifecycle states.
type AmendmentStatus string

const (
	AmendmentStatusProposed  AmendmentStatus = "proposed"
	AmendmentStatusAccepted  AmendmentStatus = "accepted"
	AmendmentStatusRejected  AmendmentStatus = "rejected"
	AmendmentStatusWithdrawn AmendmentStatus = "withdrawn"
)

// Act is a Bill that has received Presidential Assent and been published as
// law. It carries a gazette reference and a commencement date.
type Act struct {
	ID               ID
	BillID           ID
	CountryID        ID
	ActNumber        string
	ActName          string
	GazetteRef       string
	CommencementDate *time.Time
	AssentedAt       time.Time
	SourceDocumentID string
}

// Regulation is a subordinate legislation made under the authority of an
// Act (e.g. a Legal Notice in the Kenya Gazette).
type Regulation struct {
	ID               ID
	ActID            ID
	CountryID        ID
	Number           string
	Title            string
	GazetteRef       string
	EffectiveDate    time.Time
	SourceDocumentID string
}

// Policy is a government policy document (not legally binding but tracked
// because it shapes legislative priorities).
type Policy struct {
	ID             ID
	CountryID      ID
	Title          string
	DepartmentID   *ID
	PublishedAt    time.Time
	SourceDocumentID string
}

// ErrNoValidator is returned when an operation requires a stage validator
// but none was provided.
var ErrNoValidator = fmt.Errorf("bill stage validator not provided")

// BillStageTransitionValidator is the country-agnostic interface the Bill
// aggregate uses to validate stage transitions. The implementation is built
// from the country adapter's StageDefinitions; the Bill entity never
// hard-codes any stage name.
type BillStageTransitionValidator interface {
	// ValidateTransition returns nil if transitioning from -> to is
	// permitted by the country's stage definitions.
	ValidateTransition(from, to string) error
}

// StageGraphValidator is the canonical implementation of
// BillStageTransitionValidator. It is constructed from a slice of
// contracts.StageDefinition (typically produced by the country adapter).
// It NEVER hard-codes any stage name; whatever the adapter supplies is what
// the validator enforces.
type StageGraphValidator struct {
	stages map[string]contracts.StageDefinition
}

// NewStageGraphValidator constructs a StageGraphValidator from the given
// stage definitions. Duplicate codes are an error.
func NewStageGraphValidator(stages []contracts.StageDefinition) (*StageGraphValidator, error) {
	if len(stages) == 0 {
		return nil, fmt.Errorf("stage validator: no stages provided")
	}
	out := &StageGraphValidator{stages: make(map[string]contracts.StageDefinition, len(stages))}
	for _, s := range stages {
		if _, dup := out.stages[s.Code]; dup {
			return nil, fmt.Errorf("stage validator: duplicate stage code %q", s.Code)
		}
		out.stages[s.Code] = s
	}
	return out, nil
}

// ValidateTransition implements BillStageTransitionValidator.
func (v *StageGraphValidator) ValidateTransition(from, to string) error {
	src, ok := v.stages[from]
	if !ok {
		return contracts.ErrValidation{
			Kind:   "bill_stage",
			Field:  "from",
			Reason: fmt.Sprintf("unknown source stage %q", from),
		}
	}
	if _, ok := v.stages[to]; !ok {
		return contracts.ErrValidation{
			Kind:   "bill_stage",
			Field:  "to",
			Reason: fmt.Sprintf("unknown target stage %q", to),
		}
	}
	for _, allowed := range src.AllowedTransitions {
		if allowed == to {
			return nil
		}
	}
	return contracts.ErrStageTransition{
		From:    from,
		To:      to,
		Allowed: src.AllowedTransitions,
	}
}

// Stages returns the underlying stage definitions (for inspection/debugging).
func (v *StageGraphValidator) Stages() []contracts.StageDefinition {
	out := make([]contracts.StageDefinition, 0, len(v.stages))
	for _, s := range v.stages {
		out = append(out, s)
	}
	return out
}

// InitialStage returns the stage with the lowest Order value, or "" if none.
func (v *StageGraphValidator) InitialStage() string {
	var best *contracts.StageDefinition
	for code := range v.stages {
		s := v.stages[code]
		if best == nil || s.Order < best.Order {
			best = &s
		}
	}
	if best == nil {
		return ""
	}
	return best.Code
}

// Compile-time assertion that StageGraphValidator satisfies the interface.
var _ BillStageTransitionValidator = (*StageGraphValidator)(nil)
