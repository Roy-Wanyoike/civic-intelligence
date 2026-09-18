// Package simulation is the public entry point to the simulation service.
//
// The internal/ tree contains the implementation; this package exposes the
// minimal API surface needed by the API service (or any other consumer) to
// build and run scenarios.
//
// Architectural invariants (Phase 18 §1):
//   - Every type returned by this package carries an explicit RealityLayer
//     tag so simulation records can NEVER be confused with observed civic
//     facts.
//   - The package never mutates canonical civic truth.
//   - No political rankings.
package simulation

import (
	"github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/application"
	"github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/domain"
	"github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/golden"
	"github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/infrastructure/engine"
	"github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/infrastructure/memory"
)

// Re-export the types consumers need.
type (
	Scenario         = domain.Scenario
	ScenarioAssumption = domain.ScenarioAssumption
	ScenarioVariable = domain.ScenarioVariable
	ScenarioModel    = domain.ScenarioModel
	SimulationRun    = domain.SimulationRun
	SimulationResult = domain.SimulationResult
	ScenarioStatus   = domain.ScenarioStatus
	RealityLayer     = domain.RealityLayer
	TenantID         = domain.TenantID
	ID               = domain.ID
	ScenarioFilter   = domain.ScenarioFilter
	ScenarioComparison = application.ScenarioComparison

	ScenarioService = application.ScenarioService
	RunService      = application.RunService
)

// Convenience re-exports of enums.
const (
	RealityLayerHypothetical = domain.RealityLayerHypothetical
	RealityLayerModeled      = domain.RealityLayerModeled
	RealityLayerObserved     = domain.RealityLayerObserved
	RealityLayerUnknown      = domain.RealityLayerUnknown

	ScenarioStatusDraft          = domain.ScenarioStatusDraft
	ScenarioStatusConfigured     = domain.ScenarioStatusConfigured
	ScenarioStatusValidating     = domain.ScenarioStatusValidating
	ScenarioStatusReady          = domain.ScenarioStatusReady
	ScenarioStatusRunning        = domain.ScenarioStatusRunning
	ScenarioStatusCompleted      = domain.ScenarioStatusCompleted
	ScenarioStatusReviewRequired = domain.ScenarioStatusReviewRequired
	ScenarioStatusArchived       = domain.ScenarioStatusArchived
)

// Wire constructs the application services with an in-memory repository,
// seeds the golden dataset (Phase 18 §30), and returns the scenario + run
// services ready for use by the API layer.
//
// In production, callers replace the in-memory repositories with
// Postgres-backed implementations. The application services are agnostic to
// the storage backend.
func Wire() (*ScenarioService, *RunService) {
	scRepo := memory.NewScenarioRepo()
	mdRepo := memory.NewModelRepo()
	runRepo := memory.NewRunRepo()
	pub := memory.NoopEventPublisher{}
	clk := domain.SystemClock{}

	// Seed golden scenarios + models.
	for _, s := range golden.AllScenarios() {
		_ = scRepo.Create(nil, s)
	}
	for _, m := range golden.AllModels() {
		_ = mdRepo.Create(nil, m)
	}

	// Build a deterministic engine that doubles input_value.
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

	scenarios := application.NewScenarioService(scRepo, mdRepo, runRepo, pub, clk)
	runs := application.NewRunService(scRepo, mdRepo, runRepo, pub, clk, eng)
	return scenarios, runs
}
