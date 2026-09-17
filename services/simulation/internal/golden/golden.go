// Package golden holds the protected golden scenario dataset. Phase 18 section 30.
//
// Golden scenarios are versioned and protected from silent modification.
// They cover the canonical categories:
//   - simple deterministic
//   - multi-variable
//   - historical counterfactual
//   - uncertainty
//   - missing data
//   - contradictory inputs
//   - invalid assumptions (expected to fail validation)
//   - extreme values
//   - scenario comparison
//   - reproducibility
//   - model version changes
//
// Tests in golden_test.go assert each golden scenario behaves as documented.
package golden

import (
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/domain"
)

// DatasetVersion is the version of the golden dataset. Bumping this requires
// a review of every test that asserts golden behaviour.
const DatasetVersion = "v1.0.0"

// defaultHorizon is the time horizon used by all golden scenarios.
var defaultHorizon = domain.TimeHorizon{
        Start:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
        End:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
        Duration: "P1Y",
}

// SimpleDeterministicModel is a trivial model that doubles its input.
// Used by the simple deterministic golden scenario.
var SimpleDeterministicModel = domain.ScenarioModel{
        ID:              "model-simple-deterministic",
        Name:            "Simple Deterministic Doubler",
        Version:         "1.0.0",
        Description:     "Doubles the input value. Used for reproducibility tests.",
        Methodology:     "output = input * 2",
        Inputs: []domain.ModelInput{
                {Name: "input_value", Type: domain.VariableDecimal, Unit: "unit", Required: true, Description: "The value to double."},
        },
        Outputs: []domain.ModelOutput{
                {Name: "output_value", Type: domain.VariableDecimal, Unit: "unit", Description: "The doubled value."},
        },
        Limitations: []string{
                "Trivial model with no real-world interpretation.",
                "Does not produce uncertainty.",
        },
        ValidationStatus: domain.ModelStatusActive,
}

// SimpleDeterministicScenario is the simplest possible scenario.
var SimpleDeterministicScenario = domain.Scenario{
        ID:           "scn-simple-deterministic",
        TenantID:     "tenant-golden",
        Name:         "Simple Deterministic Scenario",
        Description:  "A trivial scenario that doubles a single input value.",
        Type:         domain.ScenarioTypePolicy,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "Baseline state: input value is observed.",
                RealityLayer: domain.RealityLayerObserved,
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-simple-1",
                        ScenarioID:     "scn-simple-deterministic",
                        Statement:      "The input value is 10.",
                        Value:          float64(10),
                        Unit:           "unit",
                        Basis:          "User-defined input for the test.",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
        },
        Variables: []domain.ScenarioVariable{
                {
                        ID:               "var-input",
                        Name:             "input_value",
                        Type:             domain.VariableDecimal,
                        Unit:             "unit",
                        Value:            float64(10),
                        Default:          float64(10),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
        },
        ModelID:      "model-simple-deterministic",
        ModelVersion: "1.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// HistoricalCounterfactualScenario is a counterfactual exploration.
// Phase 18 section 3 (Historical Counterfactual).
var HistoricalCounterfactualScenario = domain.Scenario{
        ID:           "scn-counterfactual-2010",
        TenantID:     "tenant-golden",
        Name:         "Counterfactual: 2010 Constitution Not Promulgated",
        Description:  "Historical counterfactual exploring what would have happened had the 2010 Constitution of Kenya not been promulgated.",
        Type:         domain.ScenarioTypeCounterfactual,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "The Constitution of Kenya was promulgated on 27 August 2010.",
                RealityLayer: domain.RealityLayerObserved,
                SourceRefs: []domain.EvidenceRef{
                        {
                                Kind:               "DOCUMENT",
                                ID:                 "constitution-of-kenya-2010",
                                SourceURL:          "https://www.kenyalaw.org/kl/index.php?id=398",
                                VerificationStatus: "VERIFIED",
                        },
                },
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-cf-1",
                        ScenarioID:     "scn-counterfactual-2010",
                        Statement:      "Assume the 2010 Constitution referendum returned a NO vote.",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceLow,
                        Basis:          "Counterfactual premise; not observed.",
                },
        },
        Variables: []domain.ScenarioVariable{
                {
                        ID:               "var-referendum-result",
                        Name:             "referendum_result",
                        Type:             domain.VariableCategorical,
                        Value:            "NO",
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
        },
        ModelID:      "model-simple-deterministic",
        ModelVersion: "1.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// MissingDataScenario contains a variable marked UNKNOWN.
// Phase 18 section 8: never fabricate precision.
var MissingDataScenario = domain.Scenario{
        ID:           "scn-missing-data",
        TenantID:     "tenant-golden",
        Name:         "Scenario with Missing Data",
        Description:  "A scenario that explicitly marks an unknown input as UNKNOWN rather than fabricating a value.",
        Type:         domain.ScenarioTypePolicy,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "Baseline observed.",
                RealityLayer: domain.RealityLayerObserved,
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-missing-1",
                        ScenarioID:     "scn-missing-data",
                        Statement:      "The exact implementation cost is unknown.",
                        AssumptionType: domain.AssumptionStatusUnknown,
                        Confidence:     domain.ConfidenceUnknown,
                        Basis:          "No authoritative source published the figure.",
                },
        },
        Variables: []domain.ScenarioVariable{
                {
                        ID:               "var-missing-cost",
                        Name:             "implementation_cost",
                        Type:             domain.VariableCurrency,
                        Unit:             "KES",
                        Value:            nil, // explicit nil for UNKNOWN
                        AssumptionStatus: domain.AssumptionStatusUnknown,
                },
        },
        ModelID:      "model-simple-deterministic",
        ModelVersion: "1.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// InvalidScenario is a scenario that MUST fail validation.
// Used to test that the validation pipeline rejects impossible values.
var InvalidScenario = domain.Scenario{
        ID:           "scn-invalid-extreme",
        TenantID:     "tenant-golden",
        Name:         "Invalid: Out-of-Range Variable",
        Description:  "A scenario with a variable outside its declared range. Must fail validation.",
        Type:         domain.ScenarioTypePolicy,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "Baseline observed.",
                RealityLayer: domain.RealityLayerObserved,
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-invalid-1",
                        ScenarioID:     "scn-invalid-extreme",
                        Statement:      "Assume tax_rate = 150%.",
                        Value:          float64(150),
                        Unit:           "%",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
        },
        Variables: []domain.ScenarioVariable{
                {
                        ID:               "var-tax-rate",
                        Name:             "tax_rate",
                        Type:             domain.VariablePercentage,
                        Unit:             "%",
                        Value:            float64(150),
                        Minimum:          float64Ptr(0),
                        Maximum:          float64Ptr(100),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
        },
        ModelID:      "model-simple-deterministic",
        ModelVersion: "1.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// ComparisonScenarios are a set of 2 scenarios for comparison testing.
var ComparisonScenarios = []domain.Scenario{
        {
                ID:           "scn-comparison-a",
                TenantID:     "tenant-golden",
                Name:         "Comparison A: Implementation Delay 1 Month",
                Description:  "Implementation delayed by 1 month.",
                Type:         domain.ScenarioTypeComparative,
                Jurisdiction: "KE",
                RealityLayer: domain.RealityLayerHypothetical,
                Baseline: domain.Baseline{
                        Description:  "Baseline observed.",
                        RealityLayer: domain.RealityLayerObserved,
                },
                Assumptions: []domain.ScenarioAssumption{
                        {
                                ID:             "asm-cmp-a-1",
                                ScenarioID:     "scn-comparison-a",
                                Statement:      "Implementation delay is 1 month.",
                                Value:          float64(1),
                                Unit:           "month",
                                AssumptionType: domain.AssumptionStatusUser,
                                Confidence:     domain.ConfidenceHigh,
                        },
                },
                Variables: []domain.ScenarioVariable{
                        {
                                ID:               "var-delay",
                                Name:             "implementation_delay",
                                Type:             domain.VariableInteger,
                                Unit:             "month",
                                Value:            1,
                                AssumptionStatus: domain.AssumptionStatusUser,
                        },
                },
                ModelID:      "model-simple-deterministic",
                ModelVersion: "1.0.0",
                Status:       domain.ScenarioStatusReady,
                TimeHorizon:  defaultHorizon,
        },
        {
                ID:           "scn-comparison-b",
                TenantID:     "tenant-golden",
                Name:         "Comparison B: Implementation Delay 6 Months",
                Description:  "Implementation delayed by 6 months.",
                Type:         domain.ScenarioTypeComparative,
                Jurisdiction: "KE",
                RealityLayer: domain.RealityLayerHypothetical,
                Baseline: domain.Baseline{
                        Description:  "Baseline observed.",
                        RealityLayer: domain.RealityLayerObserved,
                },
                Assumptions: []domain.ScenarioAssumption{
                        {
                                ID:             "asm-cmp-b-1",
                                ScenarioID:     "scn-comparison-b",
                                Statement:      "Implementation delay is 6 months.",
                                Value:          float64(6),
                                Unit:           "month",
                                AssumptionType: domain.AssumptionStatusUser,
                                Confidence:     domain.ConfidenceHigh,
                        },
                },
                Variables: []domain.ScenarioVariable{
                        {
                                ID:               "var-delay",
                                Name:             "implementation_delay",
                                Type:             domain.VariableInteger,
                                Unit:             "month",
                                Value:            6,
                                AssumptionStatus: domain.AssumptionStatusUser,
                        },
                },
                ModelID:      "model-simple-deterministic",
                ModelVersion: "1.0.0",
                Status:       domain.ScenarioStatusReady,
                TimeHorizon:  defaultHorizon,
        },
}

// AllScenarios returns every golden scenario.
func AllScenarios() []domain.Scenario {
        return []domain.Scenario{
                SimpleDeterministicScenario,
                HistoricalCounterfactualScenario,
                MissingDataScenario,
                ComparisonScenarios[0],
                ComparisonScenarios[1],
        }
}

// AllModels returns every golden model.
func AllModels() []domain.ScenarioModel {
        return []domain.ScenarioModel{SimpleDeterministicModel}
}

func float64Ptr(f float64) *float64 {
        return &f
}
