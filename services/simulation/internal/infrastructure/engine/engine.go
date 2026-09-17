// Package engine implements concrete SimulationEngine providers. Phase 18 §10.
//
// The architecture deliberately keeps engines pluggable. New modeling
// methodologies (Monte Carlo, system dynamics, agent-based) can be added
// without touching the application layer.
package engine

import (
        "context"
        "fmt"
        "math/rand"

        "github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/domain"
)

// DeterministicRulesEngine applies a set of named rules to produce a single
// deterministic output. Phase 18 §10.
//
// It does NOT support uncertainty. For scenarios that require uncertainty,
// use MonteCarloEngine.
type DeterministicRulesEngine struct {
        rules []Rule
        ver   string
}

// Rule is a named computation that produces one output from inputs.
type Rule struct {
        OutputName string
        OutputType domain.VariableType
        Unit       string
        // Compute receives the scenario variables and returns the output value.
        Compute func(vars map[string]any) (any, error)
}

// NewDeterministicRulesEngine constructs an engine from a rule set.
func NewDeterministicRulesEngine(rules []Rule, version string) *DeterministicRulesEngine {
        return &DeterministicRulesEngine{rules: rules, ver: version}
}

// Spec returns the engine specification.
func (e *DeterministicRulesEngine) Spec() domain.EngineSpec {
        return domain.EngineSpec{
                Name:                "deterministic-rules",
                Version:             e.ver,
                Kind:                domain.EngineDeterministic,
                Description:         "Applies a fixed set of named rules to produce deterministic outputs. Does not produce uncertainty.",
                SupportsUncertainty: false,
        }
}

// Execute runs the rule set.
func (e *DeterministicRulesEngine) Execute(ctx context.Context, req domain.EngineRequest) (domain.EngineResponse, error) {
        // Prefer explicitly-passed variables; fall back to scenario variables.
        vars := map[string]any{}
        src := req.Variables
        if len(src) == 0 {
                src = req.Scenario.Variables
        }
        for _, v := range src {
                vars[v.Name] = v.Value
        }

        out := domain.EngineResponse{Diagnostics: map[string]any{"engine": e.Spec().Name, "rules": len(e.rules)}}

        for _, r := range e.rules {
                val, err := r.Compute(vars)
                if err != nil {
                        out.Errors = append(out.Errors, domain.SimulationError{
                                Code:    "RULE_FAILED",
                                Message: fmt.Sprintf("rule %s: %v", r.OutputName, err),
                                Stage:   "SIMULATION_STARTED",
                        })
                        continue
                }
                out.Outputs = append(out.Outputs, domain.ResultOutput{
                        Name:         r.OutputName,
                        Type:         r.OutputType,
                        Unit:         r.Unit,
                        Value:        val,
                        RealityLayer: domain.RealityLayerModeled,
                })
        }

        out.Timeline = []domain.TimelineEvent{
                {
                        Timestamp:   req.Scenario.TimeHorizon.Start,
                        Label:       domain.TimelineObserved,
                        Title:       "Baseline",
                        Description: req.Scenario.Baseline.Description,
                        Evidence:    req.Scenario.Baseline.SourceRefs,
                },
        }
        for _, a := range req.Assumptions {
                out.Timeline = append(out.Timeline, domain.TimelineEvent{
                        Timestamp:   req.Scenario.TimeHorizon.Start,
                        Label:       domain.TimelineAssumed,
                        Title:       a.Statement,
                        Description: "Assumption: " + a.Statement,
                })
        }
        out.Timeline = append(out.Timeline, domain.TimelineEvent{
                Timestamp:   req.Scenario.TimeHorizon.End,
                Label:       domain.TimelineModeled,
                Title:       "Modeled outcome",
                Description: "Simulated outcome under the stated assumptions.",
        })

        return out, nil
}

// MonteCarloEngine runs a stochastic simulation across N iterations and
// produces percentile-based uncertainty. Phase 18 §11.
//
// Every output is reported with median, p10, p90, min, max — never as a
// single point estimate. The UI must explain these are modeled outputs.
type MonteCarloEngine struct {
        simulate SimulationFn
        ver      string
}

// SimulationFn receives the variables + RNG and returns the per-iteration
// outputs as a map of name -> value.
type SimulationFn func(vars map[string]any, rng *rand.Rand) (map[string]any, error)

// NewMonteCarloEngine constructs an engine.
func NewMonteCarloEngine(simulate SimulationFn, version string) *MonteCarloEngine {
        return &MonteCarloEngine{simulate: simulate, ver: version}
}

// Spec returns the engine specification.
func (e *MonteCarloEngine) Spec() domain.EngineSpec {
        return domain.EngineSpec{
                Name:                "monte-carlo",
                Version:             e.ver,
                Kind:                domain.EngineMonteCarlo,
                Description:         "Stochastic Monte Carlo simulation producing percentile-based uncertainty.",
                SupportsUncertainty: true,
        }
}

// Execute runs the simulation.
func (e *MonteCarloEngine) Execute(ctx context.Context, req domain.EngineRequest) (domain.EngineResponse, error) {
        if req.Iterations < 1 {
                req.Iterations = 100
        }
        rng := rand.New(rand.NewSource(req.RandomSeed))
        vars := map[string]any{}
        src := req.Variables
        if len(src) == 0 {
                src = req.Scenario.Variables
        }
        for _, v := range src {
                vars[v.Name] = v.Value
        }

        // Collect per-output samples.
        samples := map[string][]float64{}
        outputOrder := []string{}

        for i := 0; i < req.Iterations; i++ {
                out, err := e.simulate(vars, rng)
                if err != nil {
                        return domain.EngineResponse{}, fmt.Errorf("iteration %d: %w", i, err)
                }
                for name, val := range out {
                        f, ok := toFloat(val)
                        if !ok {
                                continue
                        }
                        if _, exists := samples[name]; !exists {
                                outputOrder = append(outputOrder, name)
                        }
                        samples[name] = append(samples[name], f)
                }
        }

        resp := domain.EngineResponse{
                Diagnostics: map[string]any{
                        "engine":     e.Spec().Name,
                        "iterations": req.Iterations,
                        "seed":       req.RandomSeed,
                },
        }

        for _, name := range outputOrder {
                s := samples[name]
                if len(s) == 0 {
                        continue
                }
                sorted := sortedCopy(s)
                median := percentile(sorted, 50)
                p10 := percentile(sorted, 10)
                p90 := percentile(sorted, 90)
                minV := sorted[0]
                maxV := sorted[len(sorted)-1]
                resp.Outputs = append(resp.Outputs, domain.ResultOutput{
                        Name:  name,
                        Type:  domain.VariableDecimal,
                        Value: median,
                        Uncertainty: domain.UncertaintySummary{
                                HasUncertainty: true,
                                Median:         &median,
                                P10:            &p10,
                                P90:            &p90,
                                Min:            &minV,
                                Max:            &maxV,
                                Notes:          "Modeled output. Values are simulated percentiles, NOT observed facts.",
                        },
                        RealityLayer: domain.RealityLayerModeled,
                })
        }

        // Top-level uncertainty is the headline output's uncertainty.
        if len(resp.Outputs) > 0 {
                resp.Uncertainty = resp.Outputs[0].Uncertainty
        }

        resp.Timeline = []domain.TimelineEvent{
                {
                        Timestamp:   req.Scenario.TimeHorizon.Start,
                        Label:       domain.TimelineObserved,
                        Title:       "Baseline",
                        Description: req.Scenario.Baseline.Description,
                },
                {
                        Timestamp:   req.Scenario.TimeHorizon.End,
                        Label:       domain.TimelineModeled,
                        Title:       "Modeled outcome distribution",
                        Description: fmt.Sprintf("Monte Carlo distribution across %d iterations.", req.Iterations),
                },
        }

        return resp, nil
}

// toFloat coerces any numeric value to float64.
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

// sortedCopy returns a sorted copy of the slice without mutating the input.
func sortedCopy(s []float64) []float64 {
        out := make([]float64, len(s))
        copy(out, s)
        // Insertion sort -- fine for the small N we expect per output.
        for i := 1; i < len(out); i++ {
                j := i
                for j > 0 && out[j-1] > out[j] {
                        out[j-1], out[j] = out[j], out[j-1]
                        j--
                }
        }
        return out
}

// percentile returns the p-th percentile of a sorted slice.
func percentile(sorted []float64, p float64) float64 {
        if len(sorted) == 0 {
                return 0
        }
        if len(sorted) == 1 {
                return sorted[0]
        }
        idx := int(float64(len(sorted)-1) * p / 100)
        return sorted[idx]
}
