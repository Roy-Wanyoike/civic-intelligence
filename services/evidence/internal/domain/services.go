package domain

import (
        "context"
        "fmt"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// EvidenceService is the domain service responsible for attaching
// citations to claims and computing evidence sets. The application layer
// drives it; the domain service itself has no I/O dependencies.
type EvidenceService interface {
        // AttachClaim records a claim and the evidence supporting it. The
        // returned Evidence is what downstream validation uses.
        AttachClaim(ctx context.Context, claim Claim, citations []Citation) (Evidence, error)
}

// CitationValidator inspects an AI response containing claims and returns
// a verdict: which claims are supported, which lack evidence, and which
// are refuted. The application layer consumes the result to decide whether
// to publish CandidateFactAccepted or CandidateFactRejected.
type CitationValidator interface {
        // Validate returns an EvidenceSet describing every claim in the
        // response. The Missing slice contains the claim IDs that had no
        // supporting citation.
        Validate(ctx context.Context, req ValidationRequest) (EvidenceSet, []string, error)
}

// ValidationRequest is the input to the CitationValidator.
type ValidationRequest struct {
        RequestID string
        BillID    string
        Claims    []Claim
}

// SourceConflictDetector inspects evidence for disagreements and produces
// SourceConflict records. The detector never silently resolves; it always
// surfaces conflicts for human review.
type SourceConflictDetector interface {
        Detect(ctx context.Context, e Evidence) ([]SourceConflict, error)
}

// ClaimRepository persists claims.
type ClaimRepository interface {
        Save(ctx context.Context, c Claim) error
        Get(ctx context.Context, id string) (*Claim, error)
        ListBySubject(ctx context.Context, subject string) ([]Claim, error)
}

// CitationRepository persists citations.
type CitationRepository interface {
        Save(ctx context.Context, c Citation) error
        ListByClaim(ctx context.Context, claimID string) ([]Citation, error)
}

// EvidenceRepository persists evidence sets.
type EvidenceRepository interface {
        Save(ctx context.Context, e EvidenceSet) error
        GetByRequest(ctx context.Context, requestID string) (*EvidenceSet, error)
}

// SourceConflictRepository persists conflicts.
type SourceConflictRepository interface {
        Save(ctx context.Context, c SourceConflict) error
        ListOpen(ctx context.Context) ([]SourceConflict, error)
}

// SourceReferenceRepository persists source references.
type SourceReferenceRepository interface {
        Save(ctx context.Context, r SourceReference) error
        GetByDocument(ctx context.Context, documentID string) (*SourceReference, error)
}

// EventPublisher is the abstract event bus.
type EventPublisher interface {
        Publish(ctx context.Context, event contracts.Event) error
}

// Clock abstracts time.
type Clock interface{ Now() time.Time }

// SystemClock is the production Clock.
type SystemClock struct{}

// Now implements Clock.
func (SystemClock) Now() time.Time { return time.Now().UTC() }

// DefaultEvidenceService is the canonical implementation of EvidenceService.
type DefaultEvidenceService struct {
        claims  ClaimRepository
        cites   CitationRepository
        clock   Clock
        idGen   IDGenerator
}

// IDGenerator generates string IDs.
type IDGenerator interface {
        New() string
}

// NewDefaultEvidenceService constructs the service.
func NewDefaultEvidenceService(claims ClaimRepository, cites CitationRepository, clock Clock, idGen IDGenerator) *DefaultEvidenceService {
        return &DefaultEvidenceService{claims: claims, cites: cites, clock: clock, idGen: idGen}
}

// AttachClaim implements EvidenceService. It persists the claim and each
// citation, then computes the support/refute counts and returns the
// resulting Evidence.
func (s *DefaultEvidenceService) AttachClaim(ctx context.Context, claim Claim, citations []Citation) (Evidence, error) {
        if claim.Text == "" {
                return Evidence{}, contracts.ErrValidation{Kind: "claim", Field: "text", Reason: "must not be empty"}
        }
        if claim.ID == "" {
                claim.ID = s.idGen.New()
        }
        now := s.clock.Now()
        if claim.CreatedAt.IsZero() {
                claim.CreatedAt = now
        }
        if err := s.claims.Save(ctx, claim); err != nil {
                return Evidence{}, err
        }
        // Build the Evidence with the MODIFIED citations (IDs + ClaimIDs assigned).
        // Do NOT pre-populate ev.Citations with the originals — that would duplicate them.
        ev := Evidence{ClaimID: claim.ID, Citations: make([]Citation, 0, len(citations))}
        for i := range citations {
                citations[i].ClaimID = claim.ID
                if citations[i].ID == "" {
                        citations[i].ID = s.idGen.New()
                }
                if citations[i].CreatedAt.IsZero() {
                        citations[i].CreatedAt = now
                }
                if err := s.cites.Save(ctx, citations[i]); err != nil {
                        return Evidence{}, err
                }
                switch citations[i].Relationship {
                case CitationSupports:
                        ev.Supports++
                case CitationRefutes:
                        ev.Refutes++
                }
                ev.Citations = append(ev.Citations, citations[i])
        }
        return ev, nil
}

// DefaultCitationValidator is the canonical CitationValidator implementation.
// A claim is considered supported if at least one citation with relationship
// "supports" is present. A claim is considered refuted if any citation
// refutes it. A claim with no citations is missing.
type DefaultCitationValidator struct {
        claims ClaimRepository
        cites  CitationRepository
        clock  Clock
        idGen  IDGenerator
}

// NewDefaultCitationValidator constructs the validator.
func NewDefaultCitationValidator(claims ClaimRepository, cites CitationRepository, clock Clock, idGen IDGenerator) *DefaultCitationValidator {
        return &DefaultCitationValidator{claims: claims, cites: cites, clock: clock, idGen: idGen}
}

// Validate implements CitationValidator.
func (v *DefaultCitationValidator) Validate(ctx context.Context, req ValidationRequest) (EvidenceSet, []string, error) {
        if req.RequestID == "" {
                return EvidenceSet{}, nil, contracts.ErrValidation{Kind: "validation", Field: "request_id", Reason: "must not be empty"}
        }
        set := EvidenceSet{
                ID:          v.idGen.New(),
                RequestID:   req.RequestID,
                BillID:      req.BillID,
                ValidatorID: "evidence_v1",
                ValidatedAt: v.clock.Now(),
        }
        missing := make([]string, 0)
        for _, claim := range req.Claims {
                cites, err := v.cites.ListByClaim(ctx, claim.ID)
                if err != nil {
                        return EvidenceSet{}, nil, err
                }
                ev := Evidence{ClaimID: claim.ID, Citations: cites}
                for _, c := range cites {
                        switch c.Relationship {
                        case CitationSupports:
                                ev.Supports++
                        case CitationRefutes:
                                ev.Refutes++
                        }
                }
                set.Items = append(set.Items, ev)
                if ev.Supports == 0 {
                        missing = append(missing, claim.ID)
                }
        }
        return set, missing, nil
}

// DefaultSourceConflictDetector detects when two citations to the same claim
// disagree on the quoted value. It always returns a SourceConflict record;
// it never silently resolves.
type DefaultSourceConflictDetector struct {
        repo SourceConflictRepository
        clock Clock
        idGen  IDGenerator
}

// NewDefaultSourceConflictDetector constructs the detector.
func NewDefaultSourceConflictDetector(repo SourceConflictRepository, clock Clock, idGen IDGenerator) *DefaultSourceConflictDetector {
        return &DefaultSourceConflictDetector{repo: repo, clock: clock, idGen: idGen}
}

// Detect implements SourceConflictDetector. A conflict exists when two
// citations for the same claim have quotes that differ on a key fact. The
// heuristic here compares normalised quotes; production would use semantic
// similarity.
func (d *DefaultSourceConflictDetector) Detect(ctx context.Context, e Evidence) ([]SourceConflict, error) {
        // Group all "supports" citations for this claim together. If two
        // supporting citations disagree on the quote, that's a conflict.
        supports := make([]Citation, 0, len(e.Citations))
        for _, c := range e.Citations {
                if c.Relationship != CitationSupports {
                        continue
                }
                if normalise(c.Quote) == "" {
                        continue
                }
                supports = append(supports, c)
        }
        if len(supports) < 2 {
                return nil, nil
        }
        conflicts := make([]SourceConflict, 0)
        now := d.clock.Now()
        // Compare each pair of supporting citations; if their normalised
        // quotes differ, record a conflict.
        for i := 0; i < len(supports); i++ {
                for j := i + 1; j < len(supports); j++ {
                        if normalisedEqual(supports[i].Quote, supports[j].Quote) {
                                continue
                        }
                        c := SourceConflict{
                                ID:           d.idGen.New(),
                                ClaimID:      e.ClaimID,
                                SourceAValue: supports[i].Quote,
                                SourceBValue: supports[j].Quote,
                                DetectedAt:   now,
                                Resolution:   ConflictOpen,
                        }
                        if err := d.repo.Save(ctx, c); err != nil {
                                return nil, err
                        }
                        conflicts = append(conflicts, c)
                }
        }
        return conflicts, nil
}

// normalise strips whitespace and lowercases a string for comparison.
func normalise(s string) string {
        return strings.ToLower(strings.TrimSpace(s))
}

// normalisedEqual reports whether two strings are equal after normalisation.
func normalisedEqual(a, b string) bool {
        return normalise(a) == normalise(b)
}

// Compile-time assertions.
var _ = fmt.Sprintf
