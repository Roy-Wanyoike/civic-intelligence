package domain_test

import (
        "testing"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
        "github.com/stretchr/testify/assert"
)

// These stages are deliberately GENERIC — they don't mention Kenya or any
// country-specific terminology. The Kenya adapter supplies the actual values.
func testStages() []contracts.StageDefinition {
        return []contracts.StageDefinition{
                // Non-terminal stages explicitly list which terminals are reachable.
                {Code: "STAGE_A", Name: "Stage A", AllowedNext: []string{"STAGE_B", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                {Code: "STAGE_B", Name: "Stage B", AllowedNext: []string{"STAGE_C", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                // Terminal stages: no outgoing transitions.
                {Code: "STAGE_C", Name: "Stage C", AllowedNext: nil, IsTerminal: true, Country: "TE"},
                {Code: "REJECTED", Name: "Rejected", AllowedNext: nil, IsTerminal: true, Country: "TE"},
                {Code: "WITHDRAWN", Name: "Withdrawn", AllowedNext: nil, IsTerminal: true, Country: "TE"},
        }
}

func TestBillStateMachine_ValidTransition(t *testing.T) {
        sm := domain.NewBillStateMachine(testStages())
        assert.NoError(t, sm.ValidateTransition("", "STAGE_A"))    // brand-new -> A
        assert.NoError(t, sm.ValidateTransition("STAGE_A", "STAGE_B"))
        assert.NoError(t, sm.ValidateTransition("STAGE_B", "STAGE_C"))
}

func TestBillStateMachine_InvalidTransition(t *testing.T) {
        sm := domain.NewBillStateMachine(testStages())
        err := sm.ValidateTransition("STAGE_A", "STAGE_C")
        assert.Error(t, err)
        // Issue #227: the canonical StageGraphValidator returns a typed
        // contracts.ErrStageTransition whose message reads
        // "invalid stage transition %q -> %q (allowed: %v)". The wrapper
        // surfaces this verbatim.
        var stageErr contracts.ErrStageTransition
        assert.ErrorAs(t, err, &stageErr, "must return ErrStageTransition")
        assert.Equal(t, "STAGE_A", stageErr.From)
        assert.Equal(t, "STAGE_C", stageErr.To)
        assert.Contains(t, stageErr.Allowed, "STAGE_B")

        // Cannot skip stages.
        err = sm.ValidateTransition("", "STAGE_C")
        assert.Error(t, err)

        // Cannot start at terminal.
        err = sm.ValidateTransition("", "REJECTED")
        assert.Error(t, err)
}

func TestBillStateMachine_TerminalIsFinal(t *testing.T) {
        sm := domain.NewBillStateMachine(testStages())
        err := sm.ValidateTransition("STAGE_C", "STAGE_A")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "terminal")

        // But terminal stages are always reachable from any non-terminal stage.
        assert.NoError(t, sm.ValidateTransition("STAGE_A", "REJECTED"))
        assert.NoError(t, sm.ValidateTransition("STAGE_B", "WITHDRAWN"))
}

func TestBillStateMachine_UnknownStage(t *testing.T) {
        sm := domain.NewBillStateMachine(testStages())
        err := sm.ValidateTransition("STAGE_A", "STAGE_X")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "unknown target stage")

        err = sm.ValidateTransition("STAGE_X", "STAGE_A")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "unknown source stage")
}

func TestBillStateMachine_EmptyTarget(t *testing.T) {
        sm := domain.NewBillStateMachine(testStages())
        err := sm.ValidateTransition("STAGE_A", "")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "target stage is empty")
}

// TestBillStateMachine_DelegatesToCanonical verifies that
// BillStateMachine delegates ValidateTransition to the canonical
// StageGraphValidator (issue #227) by asserting the typed error is the
// same type the canonical validator returns.
func TestBillStateMachine_DelegatesToCanonical(t *testing.T) {
        stages := testStages()

        // Build both validators from the same stages.
        sm := domain.NewBillStateMachine(stages)
        canonical, err := domain.NewStageGraphValidator(stages)
        assert.NoError(t, err)

        // Same input → same typed error. We don't compare errors directly
        // because Go's errors.Is requires the same instance; instead we
        // compare the projected fields.
        smErr := sm.ValidateTransition("STAGE_A", "STAGE_C")
        canonicalErr := canonical.ValidateTransition("STAGE_A", "STAGE_C")

        var smStage, canonicalStage contracts.ErrStageTransition
        assert.ErrorAs(t, smErr, &smStage, "wrapper must return ErrStageTransition")
        assert.ErrorAs(t, canonicalErr, &canonicalStage, "canonical must return ErrStageTransition")
        assert.Equal(t, canonicalStage.From, smStage.From)
        assert.Equal(t, canonicalStage.To, smStage.To)
        assert.ElementsMatch(t, canonicalStage.Allowed, smStage.Allowed)
}
