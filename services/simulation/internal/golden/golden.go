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
//
// The model declares input_value as NOT required so it can be paired with
// scenarios whose variables do not happen to include input_value (e.g. the
// multi-variable, extreme-values, and contradictory-inputs scenarios). The
// doubler rule still produces a meaningful output for scenarios that DO
// supply input_value (the simple-deterministic and model-version-changes
// scenarios).
var SimpleDeterministicModel = domain.ScenarioModel{
        ID:              "model-simple-deterministic",
        Name:            "Simple Deterministic Doubler",
        Version:         "1.0.0",
        Description:     "Doubles the input value. Used for reproducibility tests.",
        Methodology:     "output = input * 2",
        Inputs: []domain.ModelInput{
                {Name: "input_value", Type: domain.VariableDecimal, Unit: "unit", Required: false, Description: "The value to double."},
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

// SimpleDeterministicTriplerModel is a v2 of the doubler that triples its
// input instead of doubling it. Used by the model version changes golden
// scenario — same scenario, two model versions, different outputs.
var SimpleDeterministicTriplerModel = domain.ScenarioModel{
        ID:              "model-simple-deterministic-v2",
        Name:            "Simple Deterministic Tripler",
        Version:         "2.0.0",
        Description:     "Triples the input value. v2 of the Simple Deterministic Doubler.",
        Methodology:     "output = input * 3",
        Inputs: []domain.ModelInput{
                {Name: "input_value", Type: domain.VariableDecimal, Unit: "unit", Required: false, Description: "The value to triple."},
        },
        Outputs: []domain.ModelOutput{
                {Name: "output_value", Type: domain.VariableDecimal, Unit: "unit", Description: "The tripled value."},
        },
        Limitations: []string{
                "Trivial model with no real-world interpretation.",
                "Does not produce uncertainty.",
                "v2 of the doubler; produces a different output for the same input.",
        },
        ValidationStatus: domain.ModelStatusActive,
}

// MonteCarloModel is a model that declares an uncertainty-capable output.
// Used by the uncertainty golden scenario. The actual Monte Carlo compute is
// performed by the MonteCarloEngine; this declaration advertises that the
// output should be reported with percentiles, not as a point estimate.
var MonteCarloModel = domain.ScenarioModel{
        ID:              "model-monte-carlo-demo",
        Name:            "Monte Carlo Demo",
        Version:         "1.0.0",
        Description:     "Demonstrates a Monte Carlo simulation that reports percentile-based uncertainty.",
        Methodology:     "output_value = input_value + uniform(0, 5) per iteration",
        Inputs: []domain.ModelInput{
                {Name: "input_value", Type: domain.VariableDecimal, Unit: "unit", Required: false, Description: "Base value around which the simulation varies."},
        },
        Outputs: []domain.ModelOutput{
                {Name: "output_value", Type: domain.VariableDecimal, Unit: "unit", Description: "Modeled output with percentile uncertainty.", UncertaintyCapable: true},
        },
        Limitations: []string{
                "Trivial model with no real-world interpretation.",
                "Random noise is uniformly distributed; not calibrated to any real dataset.",
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

// MultiVariableScenario exercises a scenario that declares three or more
// typed variables. Phase 18 §6 — a scenario is described by a vector of
// variables, not a single scalar.
//
// Variables:
//   - tax_rate (PERCENTAGE, 0–100)
//   - implementation_delay (INTEGER, months)
//   - funding_level (CURRENCY, KES)
//
// A constraint pins the variables together: tax_rate must be at most 30% AND
// implementation_delay must be under 6 months AND funding_level must be at
// least KES 1,000,000. The golden scenario satisfies all three.
var MultiVariableScenario = domain.Scenario{
        ID:           "scn-multi-variable",
        TenantID:     "tenant-golden",
        Name:         "Multi-Variable Fiscal Policy",
        Description:  "A scenario with three or more typed variables (tax_rate, implementation_delay, funding_level) joined by a constraint expression.",
        Type:         domain.ScenarioTypePolicy,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "Baseline observed.",
                RealityLayer: domain.RealityLayerObserved,
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-multi-1",
                        ScenarioID:     "scn-multi-variable",
                        Statement:      "Tax rate is set to 25%.",
                        Value:          float64(25),
                        Unit:           "%",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
                {
                        ID:             "asm-multi-2",
                        ScenarioID:     "scn-multi-variable",
                        Statement:      "Implementation delay is 3 months.",
                        Value:          float64(3),
                        Unit:           "month",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
                {
                        ID:             "asm-multi-3",
                        ScenarioID:     "scn-multi-variable",
                        Statement:      "Funding level is KES 2,000,000.",
                        Value:          float64(2_000_000),
                        Unit:           "KES",
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
                        Value:            float64(25),
                        Minimum:          float64Ptr(0),
                        Maximum:          float64Ptr(100),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
                {
                        ID:               "var-delay",
                        Name:             "implementation_delay",
                        Type:             domain.VariableInteger,
                        Unit:             "month",
                        Value:            3,
                        Minimum:          float64Ptr(0),
                        Maximum:          float64Ptr(24),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
                {
                        ID:               "var-funding",
                        Name:             "funding_level",
                        Type:             domain.VariableCurrency,
                        Unit:             "KES",
                        Value:            float64(2_000_000),
                        Minimum:          float64Ptr(0),
                        Maximum:          float64Ptr(10_000_000_000),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
        },
        Constraints: []domain.ScenarioConstraint{
                {
                        ID:          "cn-multi-budget-coherent",
                        Description: "Tax rate, delay, and funding must form a coherent fiscal plan.",
                        Expression:  "tax_rate <= 30 && implementation_delay < 6 && funding_level >= 1000000",
                        Kind:        "LOGICAL",
                },
        },
        ModelID:      "model-simple-deterministic",
        ModelVersion: "1.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// UncertaintyScenario pairs with the MonteCarloModel. The model declares an
// uncertainty-capable output; the test instantiates a MonteCarloEngine to
// verify the percentile-based uncertainty is disclosed (Phase 18 §11 — never
// report unsupported point precision for stochastic models).
var UncertaintyScenario = domain.Scenario{
        ID:           "scn-uncertainty",
        TenantID:     "tenant-golden",
        Name:         "Monte Carlo Uncertainty Disclosure",
        Description:  "A scenario that pairs with the MonteCarloModel to verify percentile-based uncertainty is reported rather than a point estimate.",
        Type:         domain.ScenarioTypePolicy,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "Baseline observed.",
                RealityLayer: domain.RealityLayerObserved,
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-unc-1",
                        ScenarioID:     "scn-uncertainty",
                        Statement:      "The input value is 10.",
                        Value:          float64(10),
                        Unit:           "unit",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
        },
        Variables: []domain.ScenarioVariable{
                {
                        ID:               "var-input-unc",
                        Name:             "input_value",
                        Type:             domain.VariableDecimal,
                        Unit:             "unit",
                        Value:            float64(10),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
        },
        ModelID:      "model-monte-carlo-demo",
        ModelVersion: "1.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// ContradictoryInputsScenario declares two assumptions that contradict each
// other: "Tax rate will rise" and "Tax rate will fall". The scenario MUST
// fail validation. The contradiction is captured by the constraint
// Expression `!(tax_rate_rise > 0 && tax_rate_fall > 0)` — both cannot be
// non-zero simultaneously.
var ContradictoryInputsScenario = domain.Scenario{
        ID:           "scn-contradictory-inputs",
        TenantID:     "tenant-golden",
        Name:         "Contradictory Inputs (must fail validation)",
        Description:  "Two assumptions that contradict each other; the constraint expression catches the contradiction and fails validation.",
        Type:         domain.ScenarioTypePolicy,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "Baseline observed.",
                RealityLayer: domain.RealityLayerObserved,
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-contradict-1",
                        ScenarioID:     "scn-contradictory-inputs",
                        Statement:      "Tax rate will rise by 5 percentage points.",
                        Value:          float64(5),
                        Unit:           "%",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
                {
                        ID:             "asm-contradict-2",
                        ScenarioID:     "scn-contradictory-inputs",
                        Statement:      "Tax rate will fall by 5 percentage points.",
                        Value:          float64(5),
                        Unit:           "%",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
        },
        Variables: []domain.ScenarioVariable{
                {
                        ID:               "var-rise",
                        Name:             "tax_rate_rise",
                        Type:             domain.VariablePercentage,
                        Unit:             "%",
                        Value:            float64(5),
                        Minimum:          float64Ptr(0),
                        Maximum:          float64Ptr(100),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
                {
                        ID:               "var-fall",
                        Name:             "tax_rate_fall",
                        Type:             domain.VariablePercentage,
                        Unit:             "%",
                        Value:            float64(5),
                        Minimum:          float64Ptr(0),
                        Maximum:          float64Ptr(100),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
        },
        Constraints: []domain.ScenarioConstraint{
                {
                        ID:          "cn-no-contradiction",
                        Description: "Tax rate cannot both rise AND fall in the same scenario.",
                        Expression:  "!(tax_rate_rise > 0 && tax_rate_fall > 0)",
                        Kind:        "LOGICAL",
                },
        },
        ModelID:      "model-simple-deterministic",
        ModelVersion: "1.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// ExtremeValuesScenario sets variables at the EDGE of their declared range.
// Phase 18 §21 — boundary values must validate (the range check is
// inclusive). The golden scenario pins tax_rate to its maximum of 100% and
// implementation_delay to its minimum of 0 months. Validation must succeed.
var ExtremeValuesScenario = domain.Scenario{
        ID:           "scn-extreme-values",
        TenantID:     "tenant-golden",
        Name:         "Extreme Values at Range Boundary",
        Description:  "Variables pinned to the edges of their declared range; the boundary is inclusive so validation must succeed.",
        Type:         domain.ScenarioTypePolicy,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "Baseline observed.",
                RealityLayer: domain.RealityLayerObserved,
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-extreme-1",
                        ScenarioID:     "scn-extreme-values",
                        Statement:      "Tax rate is set to the maximum of 100%.",
                        Value:          float64(100),
                        Unit:           "%",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
                {
                        ID:             "asm-extreme-2",
                        ScenarioID:     "scn-extreme-values",
                        Statement:      "Implementation delay is the minimum of 0 months.",
                        Value:          float64(0),
                        Unit:           "month",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
        },
        Variables: []domain.ScenarioVariable{
                {
                        ID:               "var-extreme-tax",
                        Name:             "tax_rate",
                        Type:             domain.VariablePercentage,
                        Unit:             "%",
                        Value:            float64(100),
                        Minimum:          float64Ptr(0),
                        Maximum:          float64Ptr(100),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
                {
                        ID:               "var-extreme-delay",
                        Name:             "implementation_delay",
                        Type:             domain.VariableInteger,
                        Unit:             "month",
                        Value:            0,
                        Minimum:          float64Ptr(0),
                        Maximum:          float64Ptr(24),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
        },
        ModelID:      "model-simple-deterministic",
        ModelVersion: "1.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// ModelVersionChangesScenarioV1 is the v1 variant of the model-version-changes
// golden scenario. It uses the SimpleDeterministicModel (v1.0.0 doubler).
// The v2 variant uses the SimpleDeterministicTriplerModel (v2.0.0 tripler).
// Running both produces different outputs for the same input — the test
// verifies this divergence so a silent model change is caught.
var ModelVersionChangesScenarioV1 = domain.Scenario{
        ID:           "scn-model-version-v1",
        TenantID:     "tenant-golden",
        Name:         "Model Version v1 (doubler)",
        Description:  "Identical to scn-model-version-v2 except ModelID/ModelVersion; used to verify that switching model versions produces different outputs.",
        Type:         domain.ScenarioTypePolicy,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "Baseline observed.",
                RealityLayer: domain.RealityLayerObserved,
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-mvc-v1-1",
                        ScenarioID:     "scn-model-version-v1",
                        Statement:      "The input value is 10.",
                        Value:          float64(10),
                        Unit:           "unit",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
        },
        Variables: []domain.ScenarioVariable{
                {
                        ID:               "var-input-v1",
                        Name:             "input_value",
                        Type:             domain.VariableDecimal,
                        Unit:             "unit",
                        Value:            float64(10),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
        },
        ModelID:      "model-simple-deterministic",
        ModelVersion: "1.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// ModelVersionChangesScenarioV2 is the v2 variant — identical to v1 except
// it points at the SimpleDeterministicTriplerModel (v2.0.0 tripler).
var ModelVersionChangesScenarioV2 = domain.Scenario{
        ID:           "scn-model-version-v2",
        TenantID:     "tenant-golden",
        Name:         "Model Version v2 (tripler)",
        Description:  "Identical to scn-model-version-v1 except ModelID/ModelVersion; used to verify that switching model versions produces different outputs.",
        Type:         domain.ScenarioTypePolicy,
        Jurisdiction: "KE",
        RealityLayer: domain.RealityLayerHypothetical,
        Baseline: domain.Baseline{
                Description:  "Baseline observed.",
                RealityLayer: domain.RealityLayerObserved,
        },
        Assumptions: []domain.ScenarioAssumption{
                {
                        ID:             "asm-mvc-v2-1",
                        ScenarioID:     "scn-model-version-v2",
                        Statement:      "The input value is 10.",
                        Value:          float64(10),
                        Unit:           "unit",
                        AssumptionType: domain.AssumptionStatusUser,
                        Confidence:     domain.ConfidenceHigh,
                },
        },
        Variables: []domain.ScenarioVariable{
                {
                        ID:               "var-input-v2",
                        Name:             "input_value",
                        Type:             domain.VariableDecimal,
                        Unit:             "unit",
                        Value:            float64(10),
                        AssumptionStatus: domain.AssumptionStatusUser,
                },
        },
        ModelID:      "model-simple-deterministic-v2",
        ModelVersion: "2.0.0",
        Status:       domain.ScenarioStatusReady,
        TimeHorizon:  defaultHorizon,
}

// AllScenarios returns every golden scenario.
func AllScenarios() []domain.Scenario {
        return []domain.Scenario{
                SimpleDeterministicScenario,
                HistoricalCounterfactualScenario,
                MissingDataScenario,
                ComparisonScenarios[0],
                ComparisonScenarios[1],
                MultiVariableScenario,
                UncertaintyScenario,
                ContradictoryInputsScenario,
                ExtremeValuesScenario,
                ModelVersionChangesScenarioV1,
                ModelVersionChangesScenarioV2,
                // InvalidScenario is intentionally included — the golden dataset
                // documents an explicit negative case so the validation pipeline
                // is exercised against an impossible (out-of-range) value.
                InvalidScenario,
        }
}

// AllModels returns every golden model.
func AllModels() []domain.ScenarioModel {
        return []domain.ScenarioModel{
                SimpleDeterministicModel,
                SimpleDeterministicTriplerModel,
                MonteCarloModel,
        }
}

func float64Ptr(f float64) *float64 {
        return &f
}
