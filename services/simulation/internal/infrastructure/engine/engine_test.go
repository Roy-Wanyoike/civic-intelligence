package engine

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/domain"
)

func testScenario() domain.Scenario {
	return domain.Scenario{
		ID:           "scn-test",
		TenantID:     "tenant-test",
		Name:         "Test",
		Jurisdiction: "KE",
		RealityLayer: domain.RealityLayerHypothetical,
		Baseline: domain.Baseline{
			Description:  "Baseline observed.",
			RealityLayer: domain.RealityLayerObserved,
		},
		Assumptions: []domain.ScenarioAssumption{
			{
				ID:             "asm-1",
				Statement:      "input is 10",
				Value:          float64(10),
				AssumptionType: domain.AssumptionStatusUser,
				Confidence:     domain.ConfidenceHigh,
			},
		},
		Variables: []domain.ScenarioVariable{
			{
				Name:             "input_value",
				Type:             domain.VariableDecimal,
				Unit:             "unit",
				Value:            float64(10),
				AssumptionStatus: domain.AssumptionStatusUser,
			},
		},
		TimeHorizon: domain.TimeHorizon{
			Start: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

// TestDeterministicRulesEngine_DoublesInput verifies the deterministic engine
// produces the expected output and tags it MODELED.
func TestDeterministicRulesEngine_DoublesInput(t *testing.T) {
	eng := NewDeterministicRulesEngine([]Rule{
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

	resp, err := eng.Execute(context.Background(), domain.EngineRequest{
		Scenario: testScenario(),
	})
	if err != nil {
		t.Fatalf("engine failed: %v", err)
	}
	if len(resp.Outputs) != 1 {
		t.Fatalf("expected 1 output; got %d", len(resp.Outputs))
	}
	v, ok := resp.Outputs[0].Value.(float64)
	if !ok {
		t.Fatalf("expected float64; got %T", resp.Outputs[0].Value)
	}
	if v != 20 {
		t.Errorf("expected output=20; got %v", v)
	}
	if resp.Outputs[0].RealityLayer != domain.RealityLayerModeled {
		t.Errorf("expected output reality layer MODELED; got %s", resp.Outputs[0].RealityLayer)
	}
}

// TestDeterministicRulesEngine_DoesNotProduceUncertainty verifies the engine
// spec honestly reports it cannot produce uncertainty.
func TestDeterministicRulesEngine_DoesNotProduceUncertainty(t *testing.T) {
	eng := NewDeterministicRulesEngine(nil, "1.0.0")
	if eng.Spec().SupportsUncertainty {
		t.Fatal("deterministic engine must NOT claim to support uncertainty")
	}
}

// TestMonteCarloEngine_Reproducible verifies Gate L — Reproducibility.
// Repeated execution with identical inputs and random seed produces
// identical outputs.
func TestMonteCarloEngine_Reproducible(t *testing.T) {
	mkEngine := func() *MonteCarloEngine {
		return NewMonteCarloEngine(func(vars map[string]any, rng *rand.Rand) (map[string]any, error) {
			base, _ := vars["input_value"].(float64)
			noise := rng.Float64() * 10
			return map[string]any{"output_value": base + noise}, nil
		}, "1.0.0")
	}
	req := domain.EngineRequest{
		Scenario:   testScenario(),
		RandomSeed: 42,
		Iterations: 50,
	}
	r1, err := mkEngine().Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}
	r2, err := mkEngine().Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}
	if len(r1.Outputs) != 1 || len(r2.Outputs) != 1 {
		t.Fatalf("expected 1 output each; got %d and %d", len(r1.Outputs), len(r2.Outputs))
	}
	v1, _ := r1.Outputs[0].Value.(float64)
	v2, _ := r2.Outputs[0].Value.(float64)
	if v1 != v2 {
		t.Errorf("expected reproducible output; got %v and %v", v1, v2)
	}
	if !mkEngine().Spec().SupportsUncertainty {
		t.Fatal("Monte Carlo engine must support uncertainty")
	}
	if !r1.Outputs[0].Uncertainty.HasUncertainty {
		t.Fatal("expected uncertainty to be disclosed")
	}
	if r1.Outputs[0].Uncertainty.P10 == nil || r1.Outputs[0].Uncertainty.P90 == nil {
		t.Fatal("expected P10 and P90 to be set")
	}
	// P10 <= median <= P90.
	p10 := *r1.Outputs[0].Uncertainty.P10
	med := r1.Outputs[0].Value.(float64)
	p90 := *r1.Outputs[0].Uncertainty.P90
	if !(p10 <= med && med <= p90) {
		t.Errorf("expected p10 <= median <= p90; got p10=%v median=%v p90=%v", p10, med, p90)
	}
}
