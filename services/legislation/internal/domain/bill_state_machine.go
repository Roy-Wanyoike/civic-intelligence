package domain

import (
        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// BillStateMachine validates stage transitions for a Bill. The allowed
// transitions are COUNTRY-SPECIFIC — supplied by the adapter via
// StageDefinition.AllowedNext / AllowedTransitions. This struct contains
// ZERO Kenya-specific knowledge. If you find the string "Second Reading"
// here, that's a bug.
//
// Issue #227: this struct is now a thin wrapper around the canonical
// StageGraphValidator (in bill_version.go). The duplicate validation logic
// that previously lived here has been removed — BillStateMachine simply
// delegates ValidateTransition / IsTerminal / Stages to the canonical
// implementation. New callers should prefer NewStageGraphValidator
// directly (it returns typed errors and rejects duplicate stage codes);
// BillStateMachine is retained for backward compatibility with existing
// callers that pass duplicate codes and expect no error from the
// constructor.
type BillStateMachine struct {
        inner *StageGraphValidator
}

// NewBillStateMachine builds a state machine from a country's stage
// definitions. The legislation service constructs one per country at
// startup, using data supplied by the country adapter.
//
// Unlike NewStageGraphValidator, this constructor DOES NOT return an error
// on duplicate stage codes — it silently keeps the last definition with
// each code. This preserves the historical behavior expected by existing
// callers. New callers should prefer NewStageGraphValidator, which rejects
// duplicates.
func NewBillStateMachine(stages []contracts.StageDefinition) *BillStateMachine {
        m := make(map[string]contracts.StageDefinition, len(stages))
        for _, s := range stages {
                m[s.Code] = s
        }
        return &BillStateMachine{inner: &StageGraphValidator{stages: m}}
}

// ValidateTransition delegates to the canonical StageGraphValidator. See
// that type's docstring for the full validation rules.
func (sm *BillStateMachine) ValidateTransition(from, to string) error {
        return sm.inner.ValidateTransition(from, to)
}

// IsTerminal reports whether the given stage code is terminal for this
// country. Delegates to the canonical StageGraphValidator.
func (sm *BillStateMachine) IsTerminal(stage string) bool {
        s, ok := sm.inner.stages[stage]
        return ok && s.IsTerminal
}

// Stages returns all stage definitions for this country. Used to seed the
// bill_stages table at startup. Delegates to the canonical
// StageGraphValidator.
func (sm *BillStateMachine) Stages() []contracts.StageDefinition {
        return sm.inner.Stages()
}

// Compile-time assertion that BillStateMachine still satisfies the
// BillStageTransitionValidator interface even after delegation.
var _ BillStageTransitionValidator = (*BillStateMachine)(nil)
