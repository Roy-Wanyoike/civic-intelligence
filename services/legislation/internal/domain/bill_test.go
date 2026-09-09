package domain

import (
	"context"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
)

// makeTestStages returns a small, country-agnostic set of stages used by the
// legislation tests. The Kenya adapter's stages are tested in the adapter
// package; here we deliberately use synthetic stages to prove the Bill
// entity is country-agnostic.
func makeTestStages() []contracts.StageDefinition {
	return []contracts.StageDefinition{
		{Code: "DRAFT", Name: "Draft", Order: 1, AllowedTransitions: []string{"INTRODUCED", "WITHDRAWN"}},
		{Code: "INTRODUCED", Name: "Introduced", Order: 2, AllowedTransitions: []string{"DEBATED", "WITHDRAWN"}},
		{Code: "DEBATED", Name: "Debated", Order: 3, AllowedTransitions: []string{"VOTED", "WITHDRAWN"}},
		{Code: "VOTED", Name: "Voted", Order: 4, AllowedTransitions: []string{"PASSED", "REJECTED"}},
		{Code: "PASSED", Name: "Passed", Order: 5, AllowedTransitions: nil},
		{Code: "REJECTED", Name: "Rejected", Order: 6, AllowedTransitions: nil},
		{Code: "WITHDRAWN", Name: "Withdrawn", Order: 7, AllowedTransitions: nil},
	}
}

// TestBillStateMachine_AcceptsValidTransitions verifies the Bill aggregate
// accepts the legal transitions described by the (injected) validator.
func TestBillStateMachine_AcceptsValidTransitions(t *testing.T) {
	v, err := NewStageGraphValidator(makeTestStages())
	assert.NoError(t, err)

	now := time.Now().UTC()
	bill := NewBill("b1", "c1", "Test Bill", "Test", "h1", nil, "DRAFT", now)

	// DRAFT -> INTRODUCED -> DEBATED -> VOTED -> PASSED must all succeed.
	steps := []string{"INTRODUCED", "DEBATED", "VOTED", "PASSED"}
	for _, to := range steps {
		err := bill.ApplyTransition(v, to, "test", now.Add(time.Second), ID("evt-"+to))
		assert.NoError(t, err, "transition to %s should succeed", to)
	}
	assert.Equal(t, "PASSED", bill.CurrentStage())
	assert.Len(t, bill.Events(), len(steps), "each transition should append a BillEvent")
}

// TestBillStateMachine_RejectsInvalidTransitions is the headline test for
// the headline rule: invalid transitions must fail with a typed error. This
// guarantees that country-specific stage rules are enforced even though the
// Bill entity has no idea what the stages actually mean.
func TestBillStateMachine_RejectsInvalidTransitions(t *testing.T) {
	v, err := NewStageGraphValidator(makeTestStages())
	assert.NoError(t, err)

	now := time.Now().UTC()
	bill := NewBill("b1", "c1", "Test Bill", "Test", "h1", nil, "DRAFT", now)

	// DRAFT -> PASSED is illegal: must go DRAFT -> INTRODUCED -> ... -> PASSED.
	err = bill.ApplyTransition(v, "PASSED", "skipping stages", now, "evt-bad")
	assert.Error(t, err)
	var stageErr contracts.ErrStageTransition
	assert.ErrorAs(t, err, &stageErr, "must return ErrStageTransition")
	assert.Equal(t, "DRAFT", stageErr.From)
	assert.Equal(t, "PASSED", stageErr.To)
	assert.Equal(t, "DRAFT", bill.CurrentStage(), "bill stage must NOT change on failure")

	// INTRODUCED -> REJECTED is also illegal (REJECTED is only reachable from VOTED).
	_ = bill.ApplyTransition(v, "INTRODUCED", "", now, "e1")
	err = bill.ApplyTransition(v, "REJECTED", "", now, "e2")
	assert.Error(t, err)
	assert.Equal(t, "INTRODUCED", bill.CurrentStage())
}

// TestBillStateMachine_RejectsUnknownStages verifies that a transition to a
// stage the validator does not know is rejected, not silently accepted.
func TestBillStateMachine_RejectsUnknownStages(t *testing.T) {
	v, err := NewStageGraphValidator(makeTestStages())
	assert.NoError(t, err)
	now := time.Now().UTC()
	bill := NewBill("b1", "c1", "Test Bill", "Test", "h1", nil, "DRAFT", now)

	err = bill.ApplyTransition(v, "SECOND_READING", "", now, "e1")
	assert.Error(t, err)
	var ve contracts.ErrValidation
	assert.ErrorAs(t, err, &ve, "unknown stage must yield ErrValidation")
}

// TestStageGraphValidator_RejectsDuplicates verifies the validator cannot
// be constructed with two stages sharing the same code.
func TestStageGraphValidator_RejectsDuplicates(t *testing.T) {
	dup := makeTestStages()
	dup = append(dup, contracts.StageDefinition{Code: "DRAFT", Name: "Duplicate"})
	_, err := NewStageGraphValidator(dup)
	assert.Error(t, err)
}

// TestBillVersionAppendOnly verifies that adding a version preserves the
// previous version and makes the new one current.
func TestBillVersionAppendOnly(t *testing.T) {
	now := time.Now().UTC()
	bill := NewBill("b1", "c1", "Test Bill", "Test", "h1", nil, "DRAFT", now)
	v1 := BillVersion{ID: "v1", BillID: "b1", VersionNumber: 1, Title: "v1", PublishedAt: now}
	v2 := BillVersion{ID: "v2", BillID: "b1", VersionNumber: 2, Title: "v2", PublishedAt: now.Add(time.Minute)}
	bill.AddVersion(v1)
	bill.AddVersion(v2)
	assert.Len(t, bill.Versions(), 2)
	assert.Equal(t, "v2", bill.CurrentVersion().Title)
}

// Ensure unused imports don't break the build (context is used by callers
// of the repository interface defined in this package).
var _ = context.Background
