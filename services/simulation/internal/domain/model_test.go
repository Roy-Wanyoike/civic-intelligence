package domain

import (
	"testing"
	"time"
)

// TestSimulationResult_Validate_RejectsObservedLayer verifies Gate I —
// Canonical Truth Protection. A simulation result MUST be tagged MODELED,
// never OBSERVED.
func TestSimulationResult_Validate_RejectsObservedLayer(t *testing.T) {
	r := SimulationResult{
		ID:           "res-1",
		RunID:        "run-1",
		ScenarioID:   "scn-1",
		RealityLayer: RealityLayerObserved, // WRONG
		ModelID:      "model-1",
		ModelVersion: "1.0.0",
		EngineVersion: "1.0.0",
		GeneratedAt:  time.Now(),
		Outputs: []ResultOutput{
			{
				Name:         "metric",
				Type:         VariableDecimal,
				Value:        float64(10),
				RealityLayer: RealityLayerModeled,
			},
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error when result is tagged OBSERVED; got nil")
	}
}

// TestSimulationResult_Validate_RejectsOutputTaggedObserved verifies that
// every output is tagged MODELED.
func TestSimulationResult_Validate_RejectsOutputTaggedObserved(t *testing.T) {
	r := SimulationResult{
		ID:           "res-1",
		RunID:        "run-1",
		ScenarioID:   "scn-1",
		RealityLayer: RealityLayerModeled,
		ModelID:      "model-1",
		ModelVersion: "1.0.0",
		EngineVersion: "1.0.0",
		GeneratedAt:  time.Now(),
		Outputs: []ResultOutput{
			{
				Name:         "metric",
				Type:         VariableDecimal,
				Value:        float64(10),
				RealityLayer: RealityLayerObserved, // WRONG
			},
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error when output is tagged OBSERVED; got nil")
	}
}

// TestSimulationResult_Validate_RequiresOutputs verifies that empty results
// are rejected.
func TestSimulationResult_Validate_RequiresOutputs(t *testing.T) {
	r := SimulationResult{
		ID:           "res-1",
		RunID:        "run-1",
		ScenarioID:   "scn-1",
		RealityLayer: RealityLayerModeled,
		ModelID:      "model-1",
		ModelVersion: "1.0.0",
		EngineVersion: "1.0.0",
		GeneratedAt:  time.Now(),
		Outputs:      nil,
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error when outputs are empty; got nil")
	}
}

// TestSimulationResult_Validate_RejectsUncertaintyWithoutMedian verifies
// Gate F — Uncertainty. Scenarios capable of producing uncertain results
// must expose uncertainty rather than unsupported point precision.
func TestSimulationResult_Validate_RejectsUncertaintyWithoutMedian(t *testing.T) {
	r := SimulationResult{
		ID:           "res-1",
		RunID:        "run-1",
		ScenarioID:   "scn-1",
		RealityLayer: RealityLayerModeled,
		ModelID:      "model-1",
		ModelVersion: "1.0.0",
		EngineVersion: "1.0.0",
		GeneratedAt:  time.Now(),
		Outputs: []ResultOutput{
			{
				Name:  "metric",
				Type:  VariableDecimal,
				Value: float64(10),
				Uncertainty: UncertaintySummary{
					HasUncertainty: true,
					// median is nil
				},
				RealityLayer: RealityLayerModeled,
			},
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error when uncertainty declared but no median; got nil")
	}
}

// TestScenarioModel_Validate_RequiresLimitations verifies Gate E — Model
// Transparency. 100% of published simulation results identify the
// model/methodology and limitations.
func TestScenarioModel_Validate_RequiresLimitations(t *testing.T) {
	m := ScenarioModel{
		ID:              "model-1",
		Name:            "Model 1",
		Version:         "1.0.0",
		Methodology:     "Deterministic",
		Inputs:          []ModelInput{{Name: "x", Type: VariableDecimal}},
		Outputs:         []ModelOutput{{Name: "y", Type: VariableDecimal}},
		Limitations:     nil,
		ValidationStatus: ModelStatusActive,
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected error when model has no limitations; got nil")
	}
}

// TestScenarioModel_CanBeUsed verifies that only VALIDATED or ACTIVE models
// can be used by scenarios.
func TestScenarioModel_CanBeUsed(t *testing.T) {
	cases := []struct {
		status ModelStatus
		want   bool
	}{
		{ModelStatusDraft, false},
		{ModelStatusTesting, false},
		{ModelStatusValidated, true},
		{ModelStatusActive, true},
		{ModelStatusDeprecated, false},
		{ModelStatusRevoked, false},
	}
	for _, c := range cases {
		m := ScenarioModel{ValidationStatus: c.status}
		if got := m.CanBeUsed(); got != c.want {
			t.Errorf("CanBeUsed(%s) = %v; want %v", c.status, got, c.want)
		}
	}
}

// TestCanTransitionRun verifies the run lifecycle state machine.
func TestCanTransitionRun(t *testing.T) {
	cases := []struct {
		from, to RunStatus
		want     bool
	}{
		{RunStatusCreated, RunStatusValidated, true},
		{RunStatusCreated, RunStatusStarted, false}, // cannot skip
		{RunStatusValidated, RunStatusStarted, true},
		{RunStatusStarted, RunStatusInputsLoaded, true},
		{RunStatusInputsLoaded, RunStatusModelLoaded, true},
		{RunStatusModelLoaded, RunStatusSimulationStarted, true},
		{RunStatusSimulationStarted, RunStatusSimulationCompleted, true},
		{RunStatusSimulationCompleted, RunStatusResultsValidated, true},
		{RunStatusResultsValidated, RunStatusPublished, true},
		{RunStatusPublished, RunStatusFailed, false}, // terminal
	}
	for _, c := range cases {
		got := CanTransitionRun(c.from, c.to)
		if got != c.want {
			t.Errorf("CanTransitionRun(%s, %s) = %v; want %v", c.from, c.to, got, c.want)
		}
	}
}

// TestValidatePipeline_RejectsMissingRequiredInput verifies that the
// validation pipeline rejects scenarios missing required model inputs.
func TestValidatePipeline_RejectsMissingRequiredInput(t *testing.T) {
	s := validScenario()
	m := ScenarioModel{
		ID:              s.ModelID,
		Name:            "Test Model",
		Version:         "1.0.0",
		Methodology:     "Test",
		Inputs: []ModelInput{
			{Name: "required_input", Type: VariableDecimal, Required: true},
		},
		Outputs: []ModelOutput{{Name: "y", Type: VariableDecimal}},
		Limitations: []string{"Test limitation."},
		ValidationStatus: ModelStatusActive,
	}
	v := ValidatePipeline{}
	if err := v.Validate(s, m); err == nil {
		t.Fatal("expected error when required input is missing; got nil")
	}
}

// TestValidatePipeline_RejectsIncompatibleUnits verifies unit compatibility.
func TestValidatePipeline_RejectsIncompatibleUnits(t *testing.T) {
	s := validScenario()
	m := ScenarioModel{
		ID:              s.ModelID,
		Name:            "Test Model",
		Version:         "1.0.0",
		Methodology:     "Test",
		Inputs: []ModelInput{
			{Name: "input_value", Type: VariableDecimal, Unit: "meters"},
		},
		Outputs: []ModelOutput{{Name: "y", Type: VariableDecimal}},
		Limitations: []string{"Test limitation."},
		ValidationStatus: ModelStatusActive,
	}
	// Scenario variable is "unit", model expects "meters".
	v := ValidatePipeline{}
	if err := v.Validate(s, m); err == nil {
		t.Fatal("expected error when units are incompatible; got nil")
	}
}

// TestValidatePipeline_RejectsUnusableModel verifies that DRAFT/REVOKED
// models are rejected.
func TestValidatePipeline_RejectsUnusableModel(t *testing.T) {
	s := validScenario()
	m := ScenarioModel{
		ID:              s.ModelID,
		Name:            "Test Model",
		Version:         "1.0.0",
		Methodology:     "Test",
		Inputs: []ModelInput{
			{Name: "input_value", Type: VariableDecimal, Unit: "unit"},
		},
		Outputs: []ModelOutput{{Name: "y", Type: VariableDecimal}},
		Limitations: []string{"Test limitation."},
		ValidationStatus: ModelStatusDraft, // not usable
	}
	v := ValidatePipeline{}
	if err := v.Validate(s, m); err == nil {
		t.Fatal("expected error when model is not usable; got nil")
	}
}

// TestValidatePipeline_AcceptsValidScenario verifies the happy path.
func TestValidatePipeline_AcceptsValidScenario(t *testing.T) {
	s := validScenario()
	m := ScenarioModel{
		ID:              s.ModelID,
		Name:            "Test Model",
		Version:         "1.0.0",
		Methodology:     "Test",
		Inputs: []ModelInput{
			{Name: "input_value", Type: VariableDecimal, Unit: "unit"},
		},
		Outputs: []ModelOutput{{Name: "y", Type: VariableDecimal}},
		Limitations: []string{"Test limitation."},
		ValidationStatus: ModelStatusActive,
	}
	v := ValidatePipeline{}
	if err := v.Validate(s, m); err != nil {
		t.Fatalf("expected valid; got error: %v", err)
	}
}
