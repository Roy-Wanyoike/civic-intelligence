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
	assert.Contains(t, err.Error(), "not allowed")

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
}
