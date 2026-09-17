package domain

import (
        "testing"
        "time"
)

// validScenario is a helper that builds a minimal valid scenario for tests.
func validScenario() Scenario {
        min := 0.0
        max := 100.0
        return Scenario{
                ID:           "scn-test",
                TenantID:     "tenant-test",
                Name:         "Test Scenario",
                Description:  "A test scenario.",
                Type:         ScenarioTypePolicy,
                Jurisdiction: "KE",
                RealityLayer: RealityLayerHypothetical,
                Baseline: Baseline{
                        Description:  "Baseline observed.",
                        RealityLayer: RealityLayerObserved,
                },
                Assumptions: []ScenarioAssumption{
                        {
                                ID:             "asm-1",
                                ScenarioID:     "scn-test",
                                Statement:      "The input value is 10.",
                                Value:          float64(10),
                                Unit:           "unit",
                                AssumptionType: AssumptionStatusUser,
                                Confidence:     ConfidenceHigh,
                        },
                },
                Variables: []ScenarioVariable{
                        {
                                ID:               "var-1",
                                Name:             "input_value",
                                Type:             VariableDecimal,
                                Unit:             "unit",
                                Value:            float64(10),
                                Minimum:          &min,
                                Maximum:          &max,
                                AssumptionStatus: AssumptionStatusUser,
                        },
                },
                TimeHorizon: TimeHorizon{
                        Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
                        End:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
                },
                ModelID:      "model-1",
                ModelVersion: "1.0.0",
                Status:       ScenarioStatusReady,
        }
}

// TestScenario_Validate_AcceptsValidScenario verifies a minimal valid scenario
// passes validation.
func TestScenario_Validate_AcceptsValidScenario(t *testing.T) {
        s := validScenario()
        if err := s.Validate(); err != nil {
                t.Fatalf("expected valid; got error: %v", err)
        }
}

// TestScenario_Validate_RejectsObservedRealityLayer verifies that a scenario
// can NEVER be tagged OBSERVED. Gate A — Reality Separation.
func TestScenario_Validate_RejectsObservedRealityLayer(t *testing.T) {
        s := validScenario()
        s.RealityLayer = RealityLayerObserved
        if err := s.Validate(); err == nil {
                t.Fatal("expected error when scenario is tagged OBSERVED; got nil")
        }
}

// TestScenario_Validate_RejectsObservedBaseline verifies that the baseline
// must be tagged OBSERVED (a scenario is hypothetical, but its baseline is
// observed reality).
func TestScenario_Validate_RejectsObservedBaseline(t *testing.T) {
        s := validScenario()
        s.Baseline.RealityLayer = RealityLayerHypothetical
        if err := s.Validate(); err == nil {
                t.Fatal("expected error when baseline is tagged HYPOTHETICAL; got nil")
        }
}

// TestScenario_Validate_RequiresAssumptions verifies Gate D — Assumption
// Transparency. 100% of published scenarios expose their assumptions.
func TestScenario_Validate_RequiresAssumptions(t *testing.T) {
        s := validScenario()
        s.Assumptions = nil
        if err := s.Validate(); err == nil {
                t.Fatal("expected error when no assumptions declared; got nil")
        }
}

// TestScenario_Validate_RejectsUnknownWithFabricatedValue verifies that
// UNKNOWN assumptions must not carry a fabricated value. Phase 18 section 8
// and section 24.
func TestScenario_Validate_RejectsUnknownWithFabricatedValue(t *testing.T) {
        s := validScenario()
        s.Assumptions = []ScenarioAssumption{
                {
                        ID:             "asm-unknown",
                        ScenarioID:     "scn-test",
                        Statement:      "Unknown assumption with fabricated value.",
                        Value:          float64(42),
                        AssumptionType: AssumptionStatusUnknown,
                        Confidence:     ConfidenceUnknown,
                },
        }
        if err := s.Validate(); err == nil {
                t.Fatal("expected error when UNKNOWN assumption carries a value; got nil")
        }
}

// TestScenario_Validate_RejectsExternallySourcedWithoutEvidence verifies
// Gate B — Provenance. Every externally sourced assumption has traceable
// evidence.
func TestScenario_Validate_RejectsExternallySourcedWithoutEvidence(t *testing.T) {
        s := validScenario()
        s.Assumptions = []ScenarioAssumption{
                {
                        ID:             "asm-1",
                        ScenarioID:     "scn-test",
                        Statement:      "Externally observed value.",
                        Value:          float64(10),
                        AssumptionType: AssumptionStatusObserved,
                        Confidence:     ConfidenceHigh,
                },
        }
        if err := s.Validate(); err == nil {
                t.Fatal("expected error when externally sourced assumption has no evidence; got nil")
        }
}

// TestScenario_Validate_RejectsVariableOutOfRange verifies that variables
// outside their declared range are rejected by the validation pipeline.
// Phase 18 section 21. (Range checks live in ValidateConstraints, which is
// invoked by ValidatePipeline, not by Scenario.Validate itself.)
func TestScenario_Validate_RejectsVariableOutOfRange(t *testing.T) {
        s := validScenario()
        s.Variables[0].Value = float64(150) // exceeds max of 100
        m := ScenarioModel{
                ID:              s.ModelID,
                Name:            "Test Model",
                Version:         "1.0.0",
                Methodology:     "Test",
                Inputs:          []ModelInput{{Name: "input_value", Type: VariableDecimal, Unit: "unit"}},
                Outputs:         []ModelOutput{{Name: "y", Type: VariableDecimal}},
                Limitations:     []string{"Test limitation."},
                ValidationStatus: ModelStatusActive,
        }
        v := ValidatePipeline{}
        if err := v.Validate(s, m); err == nil {
                t.Fatal("expected error when variable value is out of range; got nil")
        }
}

// TestScenario_Validate_RejectsMinGreaterThanMax verifies min/max consistency.
func TestScenario_Validate_RejectsMinGreaterThanMax(t *testing.T) {
        s := validScenario()
        min := 100.0
        max := 0.0
        s.Variables[0].Minimum = &min
        s.Variables[0].Maximum = &max
        if err := s.Validate(); err == nil {
                t.Fatal("expected error when minimum > maximum; got nil")
        }
}

// TestScenario_CanTransition verifies the lifecycle state machine.
func TestScenario_CanTransition(t *testing.T) {
        cases := []struct {
                from, to ScenarioStatus
                want     bool
        }{
                {ScenarioStatusDraft, ScenarioStatusConfigured, true},
                {ScenarioStatusDraft, ScenarioStatusRunning, false}, // cannot skip
                {ScenarioStatusConfigured, ScenarioStatusValidating, true},
                {ScenarioStatusValidating, ScenarioStatusReady, true},
                {ScenarioStatusReady, ScenarioStatusRunning, true},
                {ScenarioStatusRunning, ScenarioStatusCompleted, true},
                {ScenarioStatusRunning, ScenarioStatusReviewRequired, true},
                {ScenarioStatusCompleted, ScenarioStatusArchived, true},
                {ScenarioStatusArchived, ScenarioStatusDraft, false}, // terminal
                {ScenarioStatusArchived, ScenarioStatusReady, false},
        }
        for _, c := range cases {
                got := CanTransition(c.from, c.to)
                if got != c.want {
                        t.Errorf("CanTransition(%s, %s) = %v; want %v", c.from, c.to, got, c.want)
                }
        }
}

// TestScenario_Validate_RejectsZeroTimeHorizon verifies time horizon sanity.
func TestScenario_Validate_RejectsZeroTimeHorizon(t *testing.T) {
        s := validScenario()
        s.TimeHorizon.Start = time.Time{}
        if err := s.Validate(); err == nil {
                t.Fatal("expected error when time horizon start is zero; got nil")
        }
}

// TestScenario_Validate_RejectsInvertedTimeHorizon verifies end > start.
func TestScenario_Validate_RejectsInvertedTimeHorizon(t *testing.T) {
        s := validScenario()
        s.TimeHorizon.Start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
        s.TimeHorizon.End = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
        if err := s.Validate(); err == nil {
                t.Fatal("expected error when time horizon end < start; got nil")
        }
}
