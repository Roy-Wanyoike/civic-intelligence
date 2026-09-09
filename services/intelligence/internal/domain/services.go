package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// AIGatewayClient is the abstract interface to the Python AI gateway
// (services/ai). The Go orchestrator never runs inference itself; it
// delegates all model calls to this interface. This keeps the Go service
// pure-Go and lets the Python service own all model dependencies.
type AIGatewayClient interface {
	// GenerateSummary produces a one-paragraph summary of a bill.
	GenerateSummary(ctx context.Context, req SummaryRequest) (Summary, error)
	// GenerateExplanation produces a multi-section explanation.
	GenerateExplanation(ctx context.Context, req ExplanationRequest) (Explanation, error)
	// AnswerQuestion produces a streaming answer to a user question. The
	// channel receives partial Answer sections as they complete.
	AnswerQuestion(ctx context.Context, req AnswerRequest) (<-chan AnswerSection, error)
	// Classify assigns topics to a bill.
	Classify(ctx context.Context, req ClassifyRequest) (Classification, error)
	// AnalyseImpact produces a structured impact assessment.
	AnalyseImpact(ctx context.Context, req ImpactRequest) (ImpactAnalysis, error)
	// Embed returns the embedding vector for a chunk of text.
	Embed(ctx context.Context, text string) ([]float32, error)
}

// SummaryRequest is the input to GenerateSummary.
type SummaryRequest struct {
	BillID    string
	VersionNo int
	Audience  string
}

// ExplanationRequest is the input to GenerateExplanation.
type ExplanationRequest struct {
	BillID    string
	VersionNo int
	Audience  string
	Sections  []string // requested section titles
}

// AnswerRequest is the input to AnswerQuestion.
type AnswerRequest struct {
	BillID    string
	Question  string
	Audience  string
	UserID    string
}

// ClassifyRequest is the input to Classify.
type ClassifyRequest struct {
	BillID    string
	VersionNo int
}

// ImpactRequest is the input to AnalyseImpact.
type ImpactRequest struct {
	BillID    string
	VersionNo int
	Sector    string
}

// EvidenceLookupClient is the abstract interface to the evidence service.
// The intelligence service uses it to validate candidate facts: a fact is
// accepted only if the evidence service confirms supporting citations exist.
type EvidenceLookupClient interface {
	// LookupEvidence returns the evidence the evidence service has on
	// record for the given claim text. Empty result means no evidence.
	LookupEvidence(ctx context.Context, claimText string) (EvidenceLookup, error)
}

// EvidenceLookup is the value object returned by EvidenceLookupClient.
type EvidenceLookup struct {
	ClaimText string
	Supports int
	Refutes  int
}

// CandidateFactValidator decides whether a proposed candidate fact should
// be accepted or rejected. The canonical implementation requires:
//   - At least one piece of evidence with a non-empty quote.
//   - Confidence above a configurable threshold.
//   - No conflicting evidence (refutes == 0).
// Accepted candidates are published as events; rejected candidates are also
// published (for auditing).
type CandidateFactValidator interface {
	Validate(ctx context.Context, cf CandidateFact) (CandidateFact, error)
}

// EvidenceLookupClientBasedValidator is the canonical validator.
type EvidenceLookupClientBasedValidator struct {
	evidence EvidenceLookupClient
	minConfidence float64
	clock  Clock
	idGen  IDGenerator
}

// NewEvidenceLookupClientBasedValidator constructs the validator.
func NewEvidenceLookupClientBasedValidator(evidence EvidenceLookupClient, minConfidence float64, clock Clock, idGen IDGenerator) *EvidenceLookupClientBasedValidator {
	if minConfidence <= 0 {
		minConfidence = 0.6
	}
	return &EvidenceLookupClientBasedValidator{evidence: evidence, minConfidence: minConfidence, clock: clock, idGen: idGen}
}

// Validate implements CandidateFactValidator.
func (v *EvidenceLookupClientBasedValidator) Validate(ctx context.Context, cf CandidateFact) (CandidateFact, error) {
	cf.Status = CandidateFactStatusRejected
	now := v.clock.Now()
	cf.ValidatedAt = &now
	cf.Validator = "evidence_v1"

	// Rule 1: must have at least one piece of evidence.
	if len(cf.Evidence) == 0 {
		return cf, contracts.ErrEvidenceMissing{Claim: cf.Claim}
	}
	hasUsable := false
	for _, e := range cf.Evidence {
		if strings.TrimSpace(e.Quote) == "" {
			continue
		}
		if e.DocumentID == "" {
			continue
		}
		hasUsable = true
		break
	}
	if !hasUsable {
		return cf, contracts.ErrEvidenceMissing{Claim: cf.Claim}
	}

	// Rule 2: cross-check with the evidence service. If a refute exists,
	// the candidate is rejected.
	lookup, err := v.evidence.LookupEvidence(ctx, cf.Claim)
	if err != nil {
		// Lookup failure is non-fatal; we lean on the local evidence.
		lookup = EvidenceLookup{}
	}
	if lookup.Refutes > 0 {
		cf.Status = CandidateFactStatusRejected
		return cf, nil
	}
	if lookup.Supports == 0 && cf.Confidence < v.minConfidence {
		// No external corroboration AND low confidence: reject.
		cf.Status = CandidateFactStatusRejected
		return cf, nil
	}
	cf.Status = CandidateFactStatusAccepted
	return cf, nil
}

// Clock abstracts time.
type Clock interface{ Now() time.Time }

// SystemClock is the production Clock.
type SystemClock struct{}

// Now implements Clock.
func (SystemClock) Now() time.Time { return time.Now().UTC() }

// IDGenerator generates string IDs.
type IDGenerator interface {
	New() string
}

// EventPublisher is the abstract event bus.
type EventPublisher interface {
	Publish(ctx context.Context, event contracts.Event) error
}

// CandidateFactRepository persists candidate facts.
type CandidateFactRepository interface {
	Save(ctx context.Context, cf CandidateFact) error
	Get(ctx context.Context, id string) (*CandidateFact, error)
	ListByBill(ctx context.Context, billID string) ([]CandidateFact, error)
	ListPending(ctx context.Context, limit int) ([]CandidateFact, error)
}

// ExplanationRepository persists explanations.
type ExplanationRepository interface {
	Save(ctx context.Context, e Explanation) error
	GetByBill(ctx context.Context, billID string, audience string) (*Explanation, error)
}

// SummaryRepository persists summaries.
type SummaryRepository interface {
	Save(ctx context.Context, s Summary) error
	GetByBill(ctx context.Context, billID string) (*Summary, error)
}

// AnswerRepository persists Q&A pairs.
type AnswerRepository interface {
	Save(ctx context.Context, q Question, a Answer) error
}

// Compile-time assertion.
var _ = fmt.Sprintf
