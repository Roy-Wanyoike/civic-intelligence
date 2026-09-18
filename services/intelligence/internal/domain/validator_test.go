package domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type stubClock struct{ t time.Time }

func (s stubClock) Now() time.Time { return s.t }

type stubIDGen struct{ n int }

func (s *stubIDGen) New() string { s.n++; return "id" }

// stubEvidenceClient returns a configurable evidence lookup.
type stubEvidenceClient struct {
	supports int
	refutes  int
	err      error
}

func (s stubEvidenceClient) LookupEvidence(_ context.Context, _ string) (EvidenceLookup, error) {
	if s.err != nil {
		return EvidenceLookup{}, s.err
	}
	return EvidenceLookup{Supports: s.supports, Refutes: s.refutes}, nil
}

// TestCandidateFactValidator_RejectsWithoutEvidence is the headline test:
// a candidate fact with NO evidence at all must be rejected with a typed
// ErrEvidenceMissing. This protects the canonical civic domain from AI
// hallucinations.
func TestCandidateFactValidator_RejectsWithoutEvidence(t *testing.T) {
	v := NewEvidenceLookupClientBasedValidator(stubEvidenceClient{}, 0.6, stubClock{t: time.Now()}, &stubIDGen{})
	cf := CandidateFact{
		ID:        "cf1",
		BillID:    "b1",
		Subject:   "bill-1",
		Claim:     "Bill passed second reading",
		Confidence: 0.9,
		Source:    CandidateFactSourceLLM,
	}
	out, err := v.Validate(context.Background(), cf)
	assert.Error(t, err)
	assert.Equal(t, CandidateFactStatusRejected, out.Status)
	assert.NotNil(t, out.ValidatedAt)
}

// TestCandidateFactValidator_RejectsLowConfidenceWithoutExternalSupport
// verifies that a candidate fact with local evidence but no external
// corroboration AND confidence below the threshold is rejected. The
// external lookup returning 0 supports / 0 refutes is the "no signal" case.
func TestCandidateFactValidator_RejectsLowConfidenceWithoutExternalSupport(t *testing.T) {
	v := NewEvidenceLookupClientBasedValidator(stubEvidenceClient{supports: 0, refutes: 0}, 0.9, stubClock{t: time.Now()}, &stubIDGen{})
	cf := CandidateFact{
		ID:        "cf2",
		BillID:    "b1",
		Claim:     "Bill passed second reading",
		Confidence: 0.7, // below 0.9 threshold
		Source:    CandidateFactSourceLLM,
		Evidence: []CandidateFactEvidence{
			{DocumentID: "doc1", PageNumber: 1, Quote: "second reading passed"},
		},
	}
	out, err := v.Validate(context.Background(), cf)
	assert.NoError(t, err)
	assert.Equal(t, CandidateFactStatusRejected, out.Status, "low confidence + no external support must reject")
}

// TestCandidateFactValidator_AcceptsHighConfidenceWithEvidence verifies the
// happy path: high confidence + at least one piece of local evidence +
// no refutes from the evidence service => accepted.
func TestCandidateFactValidator_AcceptsHighConfidenceWithEvidence(t *testing.T) {
	v := NewEvidenceLookupClientBasedValidator(stubEvidenceClient{supports: 1, refutes: 0}, 0.6, stubClock{t: time.Now()}, &stubIDGen{})
	cf := CandidateFact{
		ID:        "cf3",
		BillID:    "b1",
		Claim:     "Bill passed second reading",
		Confidence: 0.9,
		Source:    CandidateFactSourceLLM,
		Evidence: []CandidateFactEvidence{
			{DocumentID: "doc1", PageNumber: 1, Quote: "second reading passed"},
		},
	}
	out, err := v.Validate(context.Background(), cf)
	assert.NoError(t, err)
	assert.Equal(t, CandidateFactStatusAccepted, out.Status)
	assert.NotNil(t, out.ValidatedAt)
	assert.Equal(t, "evidence_v1", out.Validator)
}

// TestCandidateFactValidator_RejectsWhenEvidenceRefutes verifies that a
// refute from the evidence service always rejects, even if local evidence
// supports.
func TestCandidateFactValidator_RejectsWhenEvidenceRefutes(t *testing.T) {
	v := NewEvidenceLookupClientBasedValidator(stubEvidenceClient{supports: 2, refutes: 1}, 0.5, stubClock{t: time.Now()}, &stubIDGen{})
	cf := CandidateFact{
		ID:        "cf4",
		BillID:    "b1",
		Claim:     "Bill passed second reading",
		Confidence: 0.99,
		Source:    CandidateFactSourceLLM,
		Evidence: []CandidateFactEvidence{
			{DocumentID: "doc1", PageNumber: 1, Quote: "second reading passed"},
		},
	}
	out, err := v.Validate(context.Background(), cf)
	assert.NoError(t, err)
	assert.Equal(t, CandidateFactStatusRejected, out.Status)
}

// TestCandidateFactValidator_RejectsEmptyQuote verifies that evidence with
// an empty quote does not count as "usable evidence".
func TestCandidateFactValidator_RejectsEmptyQuote(t *testing.T) {
	v := NewEvidenceLookupClientBasedValidator(stubEvidenceClient{}, 0.5, stubClock{t: time.Now()}, &stubIDGen{})
	cf := CandidateFact{
		ID:        "cf5",
		BillID:    "b1",
		Claim:     "Bill passed second reading",
		Confidence: 0.99,
		Source:    CandidateFactSourceLLM,
		Evidence: []CandidateFactEvidence{
			{DocumentID: "doc1", PageNumber: 1, Quote: ""},
		},
	}
	out, err := v.Validate(context.Background(), cf)
	assert.Error(t, err)
	assert.Equal(t, CandidateFactStatusRejected, out.Status)
}
