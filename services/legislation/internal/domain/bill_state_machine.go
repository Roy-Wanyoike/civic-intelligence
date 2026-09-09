package domain

import (
	"fmt"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// BillStateMachine validates stage transitions for a Bill. The allowed
// transitions are COUNTRY-SPECIFIC — supplied by the adapter via
// StageDefinition.AllowedNext. This struct contains ZERO Kenya-specific
// knowledge. If you find the string "Second Reading" here, that's a bug.
type BillStateMachine struct {
	stages map[string]contracts.StageDefinition // code -> definition
}

// NewBillStateMachine builds a state machine from a country's stage definitions.
// The legislation service constructs one per country at startup, using data
// supplied by the country adapter.
func NewBillStateMachine(stages []contracts.StageDefinition) *BillStateMachine {
	m := make(map[string]contracts.StageDefinition, len(stages))
	for _, s := range stages {
		m[s.Code] = s
	}
	return &BillStateMachine{stages: m}
}

// ValidateTransition returns nil if from → to is a valid transition for this
// country, or an error explaining why it is not.
//
// Rules:
//   - "from" must be a known stage (unless it's the empty string, meaning the
//     Bill has no stage yet — i.e., it's brand new).
//   - "to" must be a known stage.
//   - "to" must be in "from".AllowedNext, OR "to" must be a terminal stage
//     (REJECTED/WITHDRAWN/LAPSED are always reachable from any non-terminal stage).
//   - If "from" is terminal, no transitions are allowed.
func (sm *BillStateMachine) ValidateTransition(from, to string) error {
	if to == "" {
		return fmt.Errorf("invalid transition: target stage is empty")
	}
	target, ok := sm.stages[to]
	if !ok {
		return fmt.Errorf("invalid transition: unknown target stage %q", to)
	}

	// Brand-new Bill: only non-terminal stages are valid initial stages.
	if from == "" {
		if target.IsTerminal {
			return fmt.Errorf("invalid transition: cannot start a Bill at terminal stage %q", to)
		}
		return nil
	}

	source, ok := sm.stages[from]
	if !ok {
		return fmt.Errorf("invalid transition: unknown source stage %q", from)
	}
	if source.IsTerminal {
		return fmt.Errorf("invalid transition: source stage %q is terminal — no transitions allowed", from)
	}

	// Allowed if explicitly listed, or if target is terminal (REJECTED/WITHDRAWN/LAPSED).
	for _, next := range source.AllowedNext {
		if next == to {
			return nil
		}
	}
	if target.IsTerminal {
		return nil
	}
	return fmt.Errorf("invalid transition: %q -> %q is not allowed (allowed: %v)", from, to, source.AllowedNext)
}

// IsTerminal reports whether the given stage code is terminal for this country.
func (sm *BillStateMachine) IsTerminal(stage string) bool {
	s, ok := sm.stages[stage]
	return ok && s.IsTerminal
}

// Stages returns all stage definitions for this country. Used to seed the
// bill_stages table at startup.
func (sm *BillStateMachine) Stages() []contracts.StageDefinition {
	out := make([]contracts.StageDefinition, 0, len(sm.stages))
	for _, s := range sm.stages {
		out = append(out, s)
	}
	return out
}
