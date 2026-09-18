package domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// stubClock returns a fixed time.
type stubClock struct{ t time.Time }

func (s stubClock) Now() time.Time { return s.t }

// stubIDGen returns sequential IDs.
type stubIDGen struct{ n int }

func (s *stubIDGen) New() string { s.n++; return idFromInt(s.n) }

func idFromInt(n int) string {
	switch n {
	case 1:
		return "id1"
	case 2:
		return "id2"
	default:
		return "idN"
	}
}

// TestCitationValidator_FlagsUnsupportedClaims is the headline test for the
// evidence service: when an AI response makes claims with no supporting
// citations, the validator MUST flag them in the Missing list. Claims with
// at least one "supports" citation are NOT flagged.
func TestCitationValidator_FlagsUnsupportedClaims(t *testing.T) {
	clock := stubClock{t: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)}
	idGen := &stubIDGen{}
	cites := &memCitationRepo{
		byClaim: map[string][]Citation{
			"c1": {{ID: "ct1", ClaimID: "c1", Relationship: CitationSupports, Quote: "x"}},
			"c2": {}, // no citations
			"c3": {{ID: "ct2", ClaimID: "c3", Relationship: CitationRefutes, Quote: "y"}},
		},
	}
	v := NewDefaultCitationValidator(&memClaimRepo{}, cites, clock, idGen)

	set, missing, err := v.Validate(context.Background(), ValidationRequest{
		RequestID: "r1",
		BillID:    "b1",
		Claims: []Claim{
			{ID: "c1", Text: "supported claim"},
			{ID: "c2", Text: "unsupported claim"},
			{ID: "c3", Text: "refuted claim"},
		},
	})
	assert.NoError(t, err)
	assert.Len(t, missing, 2, "c2 (no citations) and c3 (only refute) are unsupported")
	assert.Contains(t, missing, "c2")
	assert.Contains(t, missing, "c3")
	assert.Len(t, set.Items, 3)
	// Spot-check the supported claim.
	var evC1 *Evidence
	for i := range set.Items {
		if set.Items[i].ClaimID == "c1" {
			evC1 = &set.Items[i]
		}
	}
	if assert.NotNil(t, evC1) {
		assert.Equal(t, 1, evC1.Supports)
		assert.Equal(t, 0, evC1.Refutes)
	}
}

// TestSourceConflictDetector_NeverSilentlyResolves verifies that two
// citations disagreeing on a claim's quote always produce a SourceConflict
// record (rather than silently picking one). The conflict's resolution is
// "open" — human review required.
func TestSourceConflictDetector_NeverSilentlyResolves(t *testing.T) {
	clock := stubClock{t: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)}
	idGen := &stubIDGen{}
	repo := &memConflictRepo{}
	detector := NewDefaultSourceConflictDetector(repo, clock, idGen)

	e := Evidence{
		ClaimID: "c1",
		Citations: []Citation{
			{ID: "x1", ClaimID: "c1", Relationship: CitationSupports, Quote: "The bill is at second reading"},
			{ID: "x2", ClaimID: "c1", Relationship: CitationSupports, Quote: "The bill is at third reading"},
		},
	}
	conflicts, err := detector.Detect(context.Background(), e)
	assert.NoError(t, err)
	assert.Len(t, conflicts, 1, "disagreement must produce exactly one conflict")
	c := conflicts[0]
	assert.Equal(t, ConflictOpen, c.Resolution, "conflict must be open, never auto-resolved")
	assert.NotEmpty(t, c.ID)
	assert.Len(t, repo.saved, 1, "conflict must be persisted")
}

// TestAttachClaim_PersistsClaimAndCitations verifies the evidence service
// attaches citations to a claim and computes the support/refute counts.
func TestAttachClaim_PersistsClaimAndCitations(t *testing.T) {
	clock := stubClock{t: time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)}
	idGen := &stubIDGen{}
	claims := &memClaimRepo{}
	cites := &memCitationRepo{byClaim: map[string][]Citation{}}
	svc := NewDefaultEvidenceService(claims, cites, clock, idGen)

	ev, err := svc.AttachClaim(context.Background(), Claim{
		ID: "c1", Subject: "bill-1", SubjectType: "bill",
		Text: "Bill is at second reading", Source: ClaimSourceAI,
		Confidence: 0.9,
	}, []Citation{
		{DocumentID: "doc1", PageNumber: 1, SectionOffset: 0, Quote: "second reading", Relationship: CitationSupports},
		{DocumentID: "doc2", PageNumber: 3, SectionOffset: 12, Quote: "rejected at second reading", Relationship: CitationRefutes},
	})
	assert.NoError(t, err)
	assert.Equal(t, "c1", ev.ClaimID)
	assert.Equal(t, 1, ev.Supports)
	assert.Equal(t, 1, ev.Refutes)
	assert.Len(t, ev.Citations, 2)
	// Each citation should have been assigned an ID and a ClaimID.
	for _, c := range ev.Citations {
		assert.NotEmpty(t, c.ID)
		assert.Equal(t, "c1", c.ClaimID)
	}
}

// memClaimRepo is an in-memory ClaimRepository for tests.
type memClaimRepo struct{ saved []Claim }

func (m *memClaimRepo) Save(_ context.Context, c Claim) error { m.saved = append(m.saved, c); return nil }
func (m *memClaimRepo) Get(_ context.Context, id string) (*Claim, error) {
	for i := range m.saved {
		if m.saved[i].ID == id {
			return &m.saved[i], nil
		}
	}
	return nil, nil
}
func (m *memClaimRepo) ListBySubject(_ context.Context, _ string) ([]Claim, error) { return m.saved, nil }

// memCitationRepo is an in-memory CitationRepository for tests.
type memCitationRepo struct {
	saved   []Citation
	byClaim map[string][]Citation
}

func (m *memCitationRepo) Save(_ context.Context, c Citation) error {
	m.saved = append(m.saved, c)
	m.byClaim[c.ClaimID] = append(m.byClaim[c.ClaimID], c)
	return nil
}
func (m *memCitationRepo) ListByClaim(_ context.Context, claimID string) ([]Citation, error) {
	return m.byClaim[claimID], nil
}

// memConflictRepo is an in-memory SourceConflictRepository.
type memConflictRepo struct{ saved []SourceConflict }

func (m *memConflictRepo) Save(_ context.Context, c SourceConflict) error { m.saved = append(m.saved, c); return nil }
func (m *memConflictRepo) ListOpen(_ context.Context) ([]SourceConflict, error) {
	out := make([]SourceConflict, 0)
	for _, c := range m.saved {
		if c.Resolution == ConflictOpen {
			out = append(out, c)
		}
	}
	return out, nil
}
