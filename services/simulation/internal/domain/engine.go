package domain

import (
	"context"
	"errors"
	"fmt"
)

// EngineKind enumerates the simulation methodologies. Phase 18 §10.
type EngineKind string

const (
	EngineDeterministic EngineKind = "DETERMINISTIC_RULES"
	EngineStatistical   EngineKind = "STATISTICAL"
	EngineAgentBased    EngineKind = "AGENT_BASED"
	EngineSystemDynamics EngineKind = "SYSTEM_DYNAMICS"
	EngineMonteCarlo    EngineKind = "MONTE_CARLO"
	EngineGraphProp     EngineKind = "GRAPH_PROPAGATION"
	EngineComparison    EngineKind = "SCENARIO_COMPARISON"
	EngineConstraint    EngineKind = "CONSTRAINT_SIMULATION"
)

// EngineSpec describes an engine implementation.
type EngineSpec struct {
	Name        string
	Version     string
	Kind        EngineKind
	Description string
	// SupportsUncertainty indicates whether the engine can produce uncertainty
	// ranges rather than point estimates. Phase 18 §11.
	SupportsUncertainty bool
}

// EngineRequest is the input bundle passed to a SimulationEngine.
type EngineRequest struct {
	Scenario      Scenario
	Model         ScenarioModel
	Variables     []ScenarioVariable
	Assumptions   []ScenarioAssumption
	RandomSeed    int64
	Iterations    int // for Monte Carlo engines
	Timeout       context.Context
}

// EngineResponse is the output bundle returned by a SimulationEngine.
type EngineResponse struct {
	Outputs     []ResultOutput
	Timeline    []TimelineEvent
	Impacts     []ModeledImpact
	Uncertainty UncertaintySummary
	Errors      []SimulationError
	Diagnostics map[string]any
}

// SimulationEngine is the provider-independent abstraction. Phase 18 §10.
//
// Implementations:
//   - DeterministicRulesEngine
//   - MonteCarloEngine
//   - SystemDynamicsEngine (future)
//   - GraphPropagationEngine (future)
//
// The architecture must not lock the platform to one modeling methodology.
type SimulationEngine interface {
	Spec() EngineSpec
	Execute(ctx context.Context, req EngineRequest) (EngineResponse, error)
}

// ValidatePipeline is the pre-execution validation chain. Phase 18 §21.
//
// Order is mandatory:
//
//	Validate Inputs
//	→ Validate Evidence
//	→ Validate Assumptions
//	→ Validate Model
//	→ Validate Units
//	→ Validate Time Horizon
//	→ Validate Constraints
//	→ Run
type ValidatePipeline struct{}

// Validate runs the full validation chain.
func (ValidatePipeline) Validate(s Scenario, m ScenarioModel) error {
	if err := ValidateInputs(s, m); err != nil {
		return fmt.Errorf("validation: inputs: %w", err)
	}
	if err := ValidateEvidence(s); err != nil {
		return fmt.Errorf("validation: evidence: %w", err)
	}
	if err := ValidateAssumptions(s); err != nil {
		return fmt.Errorf("validation: assumptions: %w", err)
	}
	if err := ValidateModel(s, m); err != nil {
		return fmt.Errorf("validation: model: %w", err)
	}
	if err := ValidateUnits(s, m); err != nil {
		return fmt.Errorf("validation: units: %w", err)
	}
	if err := ValidateTimeHorizon(s); err != nil {
		return fmt.Errorf("validation: time horizon: %w", err)
	}
	if err := ValidateConstraints(s); err != nil {
		return fmt.Errorf("validation: constraints: %w", err)
	}
	return nil
}

// ValidateInputs checks that every required model input is supplied.
func ValidateInputs(s Scenario, m ScenarioModel) error {
	provided := map[string]bool{}
	for _, v := range s.Variables {
		provided[v.Name] = true
	}
	for _, in := range m.Inputs {
		if in.Required && !provided[in.Name] {
			return fmt.Errorf("missing required input %q", in.Name)
		}
	}
	return nil
}

// ValidateEvidence ensures every externally sourced assumption has evidence.
func ValidateEvidence(s Scenario) error {
	for _, a := range s.Assumptions {
		switch a.AssumptionType {
		case AssumptionStatusObserved, AssumptionStatusHistorical, AssumptionStatusEstimate:
			if len(a.SourceEvidence) == 0 {
				return fmt.Errorf("assumption %q: externally sourced (%s) has no evidence", a.Statement, a.AssumptionType)
			}
		}
	}
	return nil
}

// ValidateAssumptions re-runs assumption validation.
func ValidateAssumptions(s Scenario) error {
	for i, a := range s.Assumptions {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("assumption[%d]: %w", i, err)
		}
	}
	return nil
}

// ValidateModel ensures the model is usable.
func ValidateModel(s Scenario, m ScenarioModel) error {
	if !m.CanBeUsed() {
		return fmt.Errorf("model %s/%s is not in a usable state (%s)", m.Name, m.Version, m.ValidationStatus)
	}
	if s.ModelID != m.ID {
		return fmt.Errorf("scenario ModelID %s does not match provided model %s", s.ModelID, m.ID)
	}
	if s.ModelVersion != "" && s.ModelVersion != m.Version {
		return fmt.Errorf("scenario ModelVersion %s does not match provided model version %s", s.ModelVersion, m.Version)
	}
	return nil
}

// ValidateUnits checks unit compatibility between variables and model inputs.
func ValidateUnits(s Scenario, m ScenarioModel) error {
	byName := map[string]ScenarioVariable{}
	for _, v := range s.Variables {
		byName[v.Name] = v
	}
	for _, in := range m.Inputs {
		v, ok := byName[in.Name]
		if !ok {
			continue // optional input not supplied
		}
		if in.Unit != "" && v.Unit != "" && in.Unit != v.Unit {
			return fmt.Errorf("incompatible units for %q: model expects %q, scenario provides %q", in.Name, in.Unit, v.Unit)
		}
		if in.Type != v.Type {
			return fmt.Errorf("incompatible types for %q: model expects %q, scenario provides %q", in.Name, in.Type, v.Type)
		}
	}
	return nil
}

// ValidateTimeHorizon checks the time horizon is sensible.
func ValidateTimeHorizon(s Scenario) error {
	if s.TimeHorizon.Start.IsZero() || s.TimeHorizon.End.IsZero() {
		return errors.New("time horizon must be set")
	}
	if !s.TimeHorizon.End.After(s.TimeHorizon.Start) {
		return errors.New("time horizon end must be after start")
	}
	return nil
}

// ValidateConstraints checks scenario constraints. Currently performs a
// best-effort range check on numeric variables.
func ValidateConstraints(s Scenario) error {
	for _, v := range s.Variables {
		if !v.isNumeric() {
			continue
		}
		f, ok := toFloat(v.Value)
		if !ok {
			continue // non-numeric value of a numeric variable is checked elsewhere
		}
		if v.Minimum != nil && f < *v.Minimum {
			return fmt.Errorf("variable %s: value %v below minimum %v", v.Name, f, *v.Minimum)
		}
		if v.Maximum != nil && f > *v.Maximum {
			return fmt.Errorf("variable %s: value %v above maximum %v", v.Name, f, *v.Maximum)
		}
	}
	return nil
}

// toFloat attempts to coerce any numeric value to float64.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}
