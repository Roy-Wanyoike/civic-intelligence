package domain

import (
	"errors"
	"fmt"
	"time"
)

// VariableType is the type system for scenario variables. Phase 18 §6.
type VariableType string

const (
	VariableInteger      VariableType = "INTEGER"
	VariableDecimal      VariableType = "DECIMAL"
	VariablePercentage   VariableType = "PERCENTAGE"
	VariableCurrency     VariableType = "CURRENCY"
	VariableDate         VariableType = "DATE"
	VariableDuration     VariableType = "DURATION"
	VariableBoolean      VariableType = "BOOLEAN"
	VariableCategorical  VariableType = "CATEGORICAL"
	VariableGeographic   VariableType = "GEOGRAPHIC"
	VariableEntityRef    VariableType = "ENTITY_REFERENCE"
)

// AssumptionStatus of a variable — distinguishes observed inputs from
// hypothetical ones. Phase 18 §6, §7.
type AssumptionStatus string

const (
	AssumptionStatusObserved  AssumptionStatus = "OBSERVED_INPUT"
	AssumptionStatusUser       AssumptionStatus = "USER_DEFINED"
	AssumptionStatusModel      AssumptionStatus = "MODEL_ASSUMPTION"
	AssumptionStatusHistorical AssumptionStatus = "HISTORICAL_REFERENCE"
	AssumptionStatusEstimate   AssumptionStatus = "ESTIMATE"
	AssumptionStatusUnknown    AssumptionStatus = "UNKNOWN"
)

// ScenarioVariable is a typed input to a scenario. Phase 18 §6.
type ScenarioVariable struct {
	ID              string
	Name            string
	Type            VariableType
	Unit            string
	Value           any
	Minimum         *float64
	Maximum         *float64
	Default         any
	Source          string
	AssumptionStatus AssumptionStatus
}

// Validate enforces variable invariants.
func (v ScenarioVariable) Validate() error {
	if v.Name == "" {
		return errors.New("variable: missing Name")
	}
	if v.Type == "" {
		return errors.New("variable: missing Type")
	}
	if v.AssumptionStatus == "" {
		return errors.New("variable: missing AssumptionStatus")
	}
	// Min/Max consistency: if both set, min <= max.
	if v.Minimum != nil && v.Maximum != nil && *v.Minimum > *v.Maximum {
		return fmt.Errorf("variable %s: minimum (%v) > maximum (%v)", v.Name, *v.Minimum, *v.Maximum)
	}
	// Range validation only applies to numeric types.
	if !v.isNumeric() {
		if v.Minimum != nil || v.Maximum != nil {
			return fmt.Errorf("variable %s: min/max not supported for type %s", v.Name, v.Type)
		}
	}
	// Unknown assumption status must NEVER carry a fabricated value (§8).
	if v.AssumptionStatus == AssumptionStatusUnknown {
		if v.Value != nil {
			return fmt.Errorf("variable %s: UNKNOWN assumption must not carry a value (would be fabricated precision)", v.Name)
		}
	}
	return nil
}

func (v ScenarioVariable) isNumeric() bool {
	switch v.Type {
	case VariableInteger, VariableDecimal, VariablePercentage, VariableCurrency:
		return true
	}
	return false
}

// ScenarioConstraint expresses a constraint on the scenario. Phase 18 §21.
type ScenarioConstraint struct {
	ID          string
	Description string
	Expression  string // serialised constraint expression
	Kind        string // "RANGE", "RELATIONSHIP", "TEMPORAL", "LOGICAL"
}

// ScenarioAssumption is an explicit, traceable assumption. Phase 18 §7.
type ScenarioAssumption struct {
	ID             string
	ScenarioID      ID
	Statement      string
	Value           any
	Unit            string
	Basis           string
	SourceEvidence  []EvidenceRef
	AssumptionType  AssumptionStatus
	Confidence      ConfidenceLevel
	CreatedAt       time.Time
}

// Validate enforces assumption invariants.
func (a ScenarioAssumption) Validate() error {
	if a.Statement == "" {
		return errors.New("assumption: missing Statement")
	}
	if a.AssumptionType == "" {
		return errors.New("assumption: missing AssumptionType")
	}
	// Gate B (Provenance): externally sourced assumptions must have at least
	// one evidence reference, OR be explicitly marked UNKNOWN / USER_DEFINED.
	if a.AssumptionType == AssumptionStatusObserved ||
		a.AssumptionType == AssumptionStatusHistorical ||
		a.AssumptionType == AssumptionStatusEstimate {
		if len(a.SourceEvidence) == 0 {
			return fmt.Errorf("assumption %q: externally sourced (%s) must have at least one evidence reference", a.Statement, a.AssumptionType)
		}
	}
	// Unknown assumptions must NOT carry a fabricated value (§8, §24).
	if a.AssumptionType == AssumptionStatusUnknown && a.Value != nil {
		return fmt.Errorf("assumption %q: UNKNOWN must not carry a value", a.Statement)
	}
	if a.Confidence == "" {
		return errors.New("assumption: missing Confidence")
	}
	return nil
}
