// Package golden holds the protected golden scenario dataset. The tests in
// this file exercise every documented golden category (Phase 18 §30) and
// assert each scenario behaves as documented:
//
//   - simple deterministic: validates + reproduces
//   - multi-variable: validates with a 3-variable constraint
//   - historical counterfactual: validates
//   - uncertainty: produces percentile-based uncertainty
//   - missing data: explicitly UNKNOWN, never fabricated
//   - contradictory inputs: FAILS validation
//   - invalid assumptions: FAILS validation
//   - extreme values: validates at the range boundary
//   - scenario comparison: two scenarios compare without ranking
//   - reproducibility: same seed + inputs → same output
//   - model version changes: same scenario, two versions, different outputs
package golden

import (
        "context"
        "math/rand"
        "strings"
        "testing"

        "github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/domain"
        "github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/infrastructure/engine"
)

// lookupModel finds a model in AllModels by ID. Test helper.
func lookupModel(t *testing.T, id domain.ID) domain.ScenarioModel {
        t.Helper()
        for _, m := range AllModels() {
                if m.ID == id {
                        return m
                }
        }
        t.Fatalf("model %s not in AllModels()", id)
        return domain.ScenarioModel{}
}

// validatePipeline runs the full ValidatePipeline against the scenario + its
// golden model. Test helper.
func validatePipeline(t *testing.T, s domain.Scenario) error {
        t.Helper()
        m := lookupModel(t, s.ModelID)
        return domain.ValidatePipeline{}.Validate(s, m)
}

// TestGoldenDataset_CoversAllDocumentedCategories verifies that the 11
// categories documented in the package docstring are all represented. This
// is the regression test for issue #209 — the dataset previously only
// surfaced 5 of 11 categories.
func TestGoldenDataset_CoversAllDocumentedCategories(t *testing.T) {
        all := AllScenarios()
        if len(all) < 11 {
                t.Fatalf("expected at least 11 golden scenarios; got %d", len(all))
        }
        // Verify every documented category has at least one scenario by checking
        // the dataset IDs that the per-category tests below rely on.
        wantIDs := []string{
                "scn-simple-deterministic",        // simple deterministic
                "scn-counterfactual-2010",        // historical counterfactual
                "scn-missing-data",               // missing data
                "scn-comparison-a", "scn-comparison-b", // scenario comparison
                "scn-multi-variable",             // multi-variable
                "scn-uncertainty",                // uncertainty
                "scn-contradictory-inputs",       // contradictory inputs
                "scn-extreme-values",             // extreme values
                "scn-model-version-v1", "scn-model-version-v2", // model version changes
                "scn-invalid-extreme",            // invalid assumptions
        }
        seen := map[string]bool{}
        for _, s := range all {
                seen[string(s.ID)] = true
        }
        for _, id := range wantIDs {
                if !seen[id] {
                        t.Errorf("expected golden scenario %q in AllScenarios(); missing", id)
                }
        }
        // Verify the corresponding models are exposed.
        modelSeen := map[domain.ID]bool{}
        for _, m := range AllModels() {
                modelSeen[m.ID] = true
        }
        for _, mid := range []domain.ID{"model-simple-deterministic", "model-simple-deterministic-v2", "model-monte-carlo-demo"} {
                if !modelSeen[mid] {
                        t.Errorf("expected golden model %q in AllModels(); missing", mid)
                }
        }
}

// TestGolden_SimpleDeterministic validates the scenario and reproduces the
// expected output (input * 2) under the doubler engine.
func TestGolden_SimpleDeterministic(t *testing.T) {
        s := SimpleDeterministicScenario
        if err := validatePipeline(t, s); err != nil {
                t.Fatalf("expected valid; got error: %v", err)
        }
        eng := engine.NewDeterministicRulesEngine([]engine.Rule{
                {
                        OutputName: "output_value",
                        OutputType: domain.VariableDecimal,
                        Unit:       "unit",
                        Compute: func(vars map[string]any) (any, error) {
                                v, _ := vars["input_value"].(float64)
                                return v * 2, nil
                        },
                },
        }, "1.0.0")
        resp, err := eng.Execute(context.Background(), domain.EngineRequest{Scenario: s})
        if err != nil {
                t.Fatalf("engine failed: %v", err)
        }
        if v, _ := resp.Outputs[0].Value.(float64); v != 20 {
                t.Errorf("expected output=20; got %v", v)
        }
}

// TestGolden_MultiVariable validates a scenario with 3+ variables joined by
// a multi-clause constraint Expression. The constraint must pass.
func TestGolden_MultiVariable(t *testing.T) {
        s := MultiVariableScenario
        if len(s.Variables) < 3 {
                t.Fatalf("expected at least 3 variables; got %d", len(s.Variables))
        }
        if err := validatePipeline(t, s); err != nil {
                t.Fatalf("expected valid; got error: %v", err)
        }
}

// TestGolden_HistoricalCounterfactual validates the historical counterfactual
// scenario.
func TestGolden_HistoricalCounterfactual(t *testing.T) {
        s := HistoricalCounterfactualScenario
        if s.Type != domain.ScenarioTypeCounterfactual {
                t.Errorf("expected HISTORICAL_COUNTERFACTUAL type; got %s", s.Type)
        }
        if err := validatePipeline(t, s); err != nil {
                t.Fatalf("expected valid; got error: %v", err)
        }
}

// TestGolden_Uncertainty verifies the uncertainty scenario pairs with the
// MonteCarloModel and that a MonteCarloEngine produces percentile-based
// uncertainty rather than a point estimate.
func TestGolden_Uncertainty(t *testing.T) {
        s := UncertaintyScenario
        m := lookupModel(t, s.ModelID)
        // Pipeline must accept the scenario.
        if err := validatePipeline(t, s); err != nil {
                t.Fatalf("expected valid; got error: %v", err)
        }
        // At least one declared output must be uncertainty-capable.
        anyUncertaintyCapable := false
        for _, o := range m.Outputs {
                if o.UncertaintyCapable {
                        anyUncertaintyCapable = true
                        break
                }
        }
        if !anyUncertaintyCapable {
                t.Fatal("expected at least one uncertainty-capable output in MonteCarloModel")
        }
        // Run the Monte Carlo engine and verify uncertainty is disclosed.
        eng := engine.NewMonteCarloEngine(func(vars map[string]any, rng *rand.Rand) (map[string]any, error) {
                base, _ := vars["input_value"].(float64)
                noise := rng.Float64() * 5
                return map[string]any{"output_value": base + noise}, nil
        }, "1.0.0")
        resp, err := eng.Execute(context.Background(), domain.EngineRequest{
                Scenario:   s,
                RandomSeed: 7,
                Iterations: 50,
        })
        if err != nil {
                t.Fatalf("monte carlo failed: %v", err)
        }
        if !resp.Uncertainty.HasUncertainty {
                t.Fatal("expected uncertainty to be disclosed")
        }
        if resp.Uncertainty.P10 == nil || resp.Uncertainty.P90 == nil {
                t.Fatal("expected P10 and P90 to be set")
        }
}

// TestGolden_MissingData verifies the missing-data scenario explicitly
// declares UNKNOWN rather than fabricating a value.
func TestGolden_MissingData(t *testing.T) {
        s := MissingDataScenario
        if err := validatePipeline(t, s); err != nil {
                t.Fatalf("expected valid; got error: %v", err)
        }
        for _, v := range s.Variables {
                if v.AssumptionStatus != domain.AssumptionStatusUnknown {
                        t.Errorf("expected variable %s to be UNKNOWN; got %s", v.Name, v.AssumptionStatus)
                }
                if v.Value != nil {
                        t.Errorf("expected variable %s to have nil value; got %v", v.Name, v.Value)
                }
        }
}

// TestGolden_ContradictoryInputs verifies the contradictory-inputs scenario
// FAILS validation. The constraint catches that the tax_rate cannot both
// rise and fall.
func TestGolden_ContradictoryInputs(t *testing.T) {
        s := ContradictoryInputsScenario
        err := validatePipeline(t, s)
        if err == nil {
                t.Fatal("expected validation to FAIL for contradictory inputs; got nil")
        }
        // The error must reference the constraint that caught the contradiction.
        if !strings.Contains(err.Error(), "cn-no-contradiction") {
                t.Errorf("expected error to mention constraint cn-no-contradiction; got %v", err)
        }
}

// TestGolden_Invalid verifies the invalid scenario FAILS validation. The
// tax_rate value (150) exceeds its declared maximum (100).
func TestGolden_Invalid(t *testing.T) {
        s := InvalidScenario
        err := validatePipeline(t, s)
        if err == nil {
                t.Fatal("expected validation to FAIL for invalid scenario; got nil")
        }
        if !strings.Contains(err.Error(), "tax_rate") {
                t.Errorf("expected error to mention tax_rate; got %v", err)
        }
}

// TestGolden_ExtremeValues verifies that values pinned to the EDGE of the
// declared range still validate. The range check is inclusive.
func TestGolden_ExtremeValues(t *testing.T) {
        s := ExtremeValuesScenario
        if err := validatePipeline(t, s); err != nil {
                t.Fatalf("expected valid at range boundary; got error: %v", err)
        }
}

// TestGolden_ScenarioComparison verifies the two comparison scenarios are
// mutually comparable (same ModelID, same Type, distinct Variables).
func TestGolden_ScenarioComparison(t *testing.T) {
        a, b := ComparisonScenarios[0], ComparisonScenarios[1]
        if a.Type != domain.ScenarioTypeComparative || b.Type != domain.ScenarioTypeComparative {
                t.Errorf("expected COMPARATIVE type for both scenarios")
        }
        if a.ModelID != b.ModelID {
                t.Errorf("expected same ModelID for comparison; got %s vs %s", a.ModelID, b.ModelID)
        }
        if err := validatePipeline(t, a); err != nil {
                t.Fatalf("comparison-a invalid: %v", err)
        }
        if err := validatePipeline(t, b); err != nil {
                t.Fatalf("comparison-b invalid: %v", err)
        }
}

// TestGolden_Reproducibility verifies Gate L — same seed + inputs must
// produce the same output across repeated executions.
func TestGolden_Reproducibility(t *testing.T) {
        s := SimpleDeterministicScenario
        mkEngine := func() *engine.DeterministicRulesEngine {
                return engine.NewDeterministicRulesEngine([]engine.Rule{
                        {
                                OutputName: "output_value",
                                OutputType: domain.VariableDecimal,
                                Unit:       "unit",
                                Compute: func(vars map[string]any) (any, error) {
                                        v, _ := vars["input_value"].(float64)
                                        return v * 2, nil
                                },
                        },
                }, "1.0.0")
        }
        req := domain.EngineRequest{Scenario: s, RandomSeed: 42}
        r1, err := mkEngine().Execute(context.Background(), req)
        if err != nil {
                t.Fatalf("run 1 failed: %v", err)
        }
        r2, err := mkEngine().Execute(context.Background(), req)
        if err != nil {
                t.Fatalf("run 2 failed: %v", err)
        }
        v1, _ := r1.Outputs[0].Value.(float64)
        v2, _ := r2.Outputs[0].Value.(float64)
        if v1 != v2 {
                t.Errorf("expected reproducible output; got %v vs %v", v1, v2)
        }
}

// TestGolden_ModelVersionChanges verifies that the SAME scenario (identical
// variables) produces DIFFERENT outputs when run under two different model
// versions. This catches silent model regressions.
func TestGolden_ModelVersionChanges(t *testing.T) {
        v1 := ModelVersionChangesScenarioV1
        v2 := ModelVersionChangesScenarioV2
        // Same input_value, different ModelID/ModelVersion.
        if v1.Variables[0].Value != v2.Variables[0].Value {
                t.Fatalf("expected identical input_value across versions; got %v vs %v",
                        v1.Variables[0].Value, v2.Variables[0].Value)
        }
        if v1.ModelID == v2.ModelID {
                t.Fatal("expected different ModelID across versions")
        }
        if v1.ModelVersion == v2.ModelVersion {
                t.Fatal("expected different ModelVersion across versions")
        }
        // Both must validate against their respective models.
        if err := validatePipeline(t, v1); err != nil {
                t.Fatalf("v1 invalid: %v", err)
        }
        if err := validatePipeline(t, v2); err != nil {
                t.Fatalf("v2 invalid: %v", err)
        }
        // v1 doubles; v2 triples — outputs MUST differ.
        engV1 := engine.NewDeterministicRulesEngine([]engine.Rule{
                {
                        OutputName: "output_value",
                        OutputType: domain.VariableDecimal,
                        Unit:       "unit",
                        Compute: func(vars map[string]any) (any, error) {
                                v, _ := vars["input_value"].(float64)
                                return v * 2, nil
                        },
                },
        }, "1.0.0")
        engV2 := engine.NewDeterministicRulesEngine([]engine.Rule{
                {
                        OutputName: "output_value",
                        OutputType: domain.VariableDecimal,
                        Unit:       "unit",
                        Compute: func(vars map[string]any) (any, error) {
                                v, _ := vars["input_value"].(float64)
                                return v * 3, nil
                        },
                },
        }, "2.0.0")
        resp1, err := engV1.Execute(context.Background(), domain.EngineRequest{Scenario: v1})
        if err != nil {
                t.Fatalf("v1 engine failed: %v", err)
        }
        resp2, err := engV2.Execute(context.Background(), domain.EngineRequest{Scenario: v2})
        if err != nil {
                t.Fatalf("v2 engine failed: %v", err)
        }
        out1, _ := resp1.Outputs[0].Value.(float64)
        out2, _ := resp2.Outputs[0].Value.(float64)
        if out1 == out2 {
                t.Errorf("expected different outputs across model versions; both were %v", out1)
        }
}
