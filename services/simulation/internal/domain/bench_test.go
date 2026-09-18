package domain_test

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/domain"
	"github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/infrastructure/engine"
)

// BenchmarkScenario_Validate measures the cost of the scenario invariant
// check (Phase 18 §21) on a fully populated scenario with one assumption
// and one variable. This is the per-request cost the API pays when a
// user creates or updates a scenario; the SLO documented in
// docs/PERFORMANCE_BASELINES.md is "Scenario creation ≤ 1s API
// acknowledgement", of which Validate must consume < 1%.
//
// Reported allocs/op must be 0; validation is a pure struct check.
func BenchmarkScenario_Validate(b *testing.B) {
	s := benchValidScenario()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := s.Validate(); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkValidatePipeline_FullValidation measures the full
// pre-execution validation pipeline (Validate → Evidence → Assumptions
// → Model → Units → TimeHorizon → Constraints). This is the upper bound
// on the cost the simulation service pays before kicking off a run.
// The benchmark uses a model whose inputs exactly match the scenario's
// variables so the happy path is exercised end-to-end.
func BenchmarkValidatePipeline_FullValidation(b *testing.B) {
	s := benchValidScenario()
	m := domain.ScenarioModel{
		ID:              s.ModelID,
		Name:            "Test Model",
		Version:         "1.0.0",
		Methodology:     "Test",
		Inputs:          []domain.ModelInput{{Name: "input_value", Type: domain.VariableDecimal, Unit: "unit"}},
		Outputs:         []domain.ModelOutput{{Name: "y", Type: domain.VariableDecimal}},
		Limitations:     []string{"Test limitation."},
		ValidationStatus: domain.ModelStatusActive,
	}
	v := domain.ValidatePipeline{}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := v.Validate(s, m); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkMonteCarloEngine_100Iterations measures a 100-iteration Monte
// Carlo run with a trivial single-output simulate function. This is the
// smallest meaningful Monte Carlo workload; it bounds the per-iteration
// cost so the API SLO for /scenarios/{id}/run (p95 ≤ 1.5s) is
// achievable even at 10× the iteration count.
//
// The engine is constructed via the real infrastructure/engine package
// so the benchmark measures the same code path the API uses in
// production (percentile computation, insertion sort, etc.).
func BenchmarkMonteCarloEngine_100Iterations(b *testing.B) {
	eng := engine.NewMonteCarloEngine(func(vars map[string]any, rng *rand.Rand) (map[string]any, error) {
		base, _ := vars["input_value"].(float64)
		noise := rng.Float64() * 10
		return map[string]any{"output_value": base + noise}, nil
	}, "1.0.0")

	s := benchValidScenario()
	req := domain.EngineRequest{
		Scenario:   s,
		Variables:  s.Variables,
		RandomSeed: 42,
		Iterations: 100,
	}
	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resp, err := eng.Execute(ctx, req)
		if err != nil {
			b.Fatalf("engine failed: %v", err)
		}
		if len(resp.Outputs) == 0 {
			b.Fatal("expected at least one output")
		}
		if !resp.Outputs[0].Uncertainty.HasUncertainty {
			b.Fatal("expected uncertainty to be reported for monte-carlo output")
		}
	}
}

// benchValidScenario mirrors the validScenario helper in
// scenario_test.go but is local to the bench file so the benchmark
// setup stays self-contained.
func benchValidScenario() domain.Scenario {
	min := 0.0
	max := 100.0
	return domain.Scenario{
		ID:           "scn-bench",
		TenantID:     "tenant-bench",
		Name:         "Bench Scenario",
		Description:  "A benchmark scenario.",
		Type:         domain.ScenarioTypePolicy,
		Jurisdiction: "KE",
		RealityLayer: domain.RealityLayerHypothetical,
		Baseline: domain.Baseline{
			Description:  "Baseline observed.",
			RealityLayer: domain.RealityLayerObserved,
		},
		Assumptions: []domain.ScenarioAssumption{
			{
				ID:             "asm-1",
				ScenarioID:     "scn-bench",
				Statement:      "The input value is 10.",
				Value:          float64(10),
				Unit:           "unit",
				AssumptionType: domain.AssumptionStatusUser,
				Confidence:     domain.ConfidenceHigh,
			},
		},
		Variables: []domain.ScenarioVariable{
			{
				ID:               "var-1",
				Name:             "input_value",
				Type:             domain.VariableDecimal,
				Unit:             "unit",
				Value:            float64(10),
				Minimum:          &min,
				Maximum:          &max,
				AssumptionStatus: domain.AssumptionStatusUser,
			},
		},
		TimeHorizon: domain.TimeHorizon{
			Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		ModelID:      "model-1",
		ModelVersion: "1.0.0",
		Status:       domain.ScenarioStatusReady,
	}
}
