// Package application wires domain types into use cases. The application
// layer orchestrates the lifecycle of scenarios and runs.
package application

import (
        "context"
        "crypto/sha256"
        "encoding/hex"
        "encoding/json"
        "fmt"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/domain"
)

// ScenarioService is the application service for scenario lifecycle.
type ScenarioService struct {
        Repos   domain.ScenarioRepository
        Models  domain.ModelRepository
        Runs    domain.RunRepository
        Events  domain.EventPublisher
        Clock   domain.Clock
}

// NewScenarioService constructs a ScenarioService.
func NewScenarioService(r domain.ScenarioRepository, m domain.ModelRepository, runs domain.RunRepository, e domain.EventPublisher, c domain.Clock) *ScenarioService {
        if c == nil {
                c = domain.SystemClock{}
        }
        return &ScenarioService{Repos: r, Models: m, Runs: runs, Events: e, Clock: c}
}

// CreateScenario creates a new draft scenario. Phase 18 §2.
func (s *ScenarioService) CreateScenario(ctx context.Context, s2 *domain.Scenario) error {
        now := s.Clock.Now()
        s2.CreatedAt = now
        s2.UpdatedAt = now
        s2.RealityLayer = domain.RealityLayerHypothetical
        s2.Baseline.RealityLayer = domain.RealityLayerObserved
        if s2.Status == "" {
                s2.Status = domain.ScenarioStatusDraft
        }
        s2.ScenarioVersion = 1

        // Invariant: scenario must validate before persistence.
        if err := s2.Validate(); err != nil {
                return fmt.Errorf("create scenario: %w", err)
        }

        if err := s.Repos.Create(ctx, *s2); err != nil {
                return fmt.Errorf("create scenario: %w", err)
        }

        // Append the v1 snapshot.
        _ = s.Repos.AppendVersion(ctx, domain.ScenarioVersion{
                ScenarioID:    s2.ID,
                Version:       1,
                Snapshot:      *s2,
                CreatedAt:     now,
                CreatedBy:     s2.CreatedBy,
                ChangeSummary: "Initial creation",
        })

        // Publish lifecycle event. Phase 18 §25.
        s.publishEvent(ctx, "scenario.created", s2.ID, s2.TenantID)
        return nil
}

// UpdateScenario mutates a scenario and appends a new version snapshot.
func (s *ScenarioService) UpdateScenario(ctx context.Context, sc *domain.Scenario, changeSummary string) error {
        now := s.Clock.Now()
        sc.UpdatedAt = now
        sc.ScenarioVersion++

        if err := sc.Validate(); err != nil {
                return fmt.Errorf("update scenario: %w", err)
        }

        if err := s.Repos.Update(ctx, *sc); err != nil {
                return fmt.Errorf("update scenario: %w", err)
        }

        _ = s.Repos.AppendVersion(ctx, domain.ScenarioVersion{
                ScenarioID:    sc.ID,
                Version:       sc.ScenarioVersion,
                Snapshot:      *sc,
                CreatedAt:     now,
                CreatedBy:     sc.CreatedBy,
                ChangeSummary: changeSummary,
        })
        return nil
}

// Transition moves a scenario between lifecycle states. Phase 18 §2.
func (s *ScenarioService) Transition(ctx context.Context, tenantID domain.TenantID, id domain.ID, to domain.ScenarioStatus) error {
        sc, err := s.Repos.Get(ctx, tenantID, id)
        if err != nil {
                return fmt.Errorf("transition: %w", err)
        }
        if !domain.CanTransition(sc.Status, to) {
                return fmt.Errorf("transition: %s → %s not permitted", sc.Status, to)
        }
        sc.Status = to
        if err := s.UpdateScenario(ctx, sc, fmt.Sprintf("Lifecycle transition %s → %s", to, to)); err != nil {
                return err
        }

        switch to {
        case domain.ScenarioStatusValidating:
                s.publishEvent(ctx, "scenario.validated", id, tenantID)
        case domain.ScenarioStatusArchived:
                s.publishEvent(ctx, "scenario.archived", id, tenantID)
        case domain.ScenarioStatusReady:
                s.publishEvent(ctx, "scenario.published", id, tenantID)
        }
        return nil
}

// GetScenario retrieves a scenario (with tenant isolation).
func (s *ScenarioService) GetScenario(ctx context.Context, tenantID domain.TenantID, id domain.ID) (*domain.Scenario, error) {
        return s.Repos.Get(ctx, tenantID, id)
}

// ListScenarios lists scenarios for a tenant.
func (s *ScenarioService) ListScenarios(ctx context.Context, tenantID domain.TenantID, filter domain.ScenarioFilter) ([]domain.Scenario, error) {
        return s.Repos.List(ctx, tenantID, filter)
}

// CompareScenarios performs a side-by-side comparison. Phase 18 §13.
// CRITICAL: this method NEVER produces political rankings. It only describes
// differences.
func (s *ScenarioService) CompareScenarios(ctx context.Context, tenantID domain.TenantID, ids []domain.ID) (*ScenarioComparison, error) {
        out := &ScenarioComparison{}
        for _, id := range ids {
                sc, err := s.Repos.Get(ctx, tenantID, id)
                if err != nil {
                        return nil, fmt.Errorf("compare: get %s: %w", id, err)
                }
                out.Scenarios = append(out.Scenarios, *sc)
        }
        out.Dimensions = defaultComparisonDimensions()
        out.Disclaimer = "This comparison describes differences only. The platform does not rank scenarios, recommend choices, or imply political preference."
        return out, nil
}

// ScenarioComparison is the comparison result. Phase 18 §13.
type ScenarioComparison struct {
        Scenarios  []domain.Scenario    `json:"scenarios"`
        Dimensions []ComparisonDimension `json:"dimensions"`
        Disclaimer string               `json:"disclaimer"`
}

// ComparisonDimension is one dimension along which scenarios are compared.
type ComparisonDimension struct {
        Name        string `json:"name"`
        Description string `json:"description"`
}

func defaultComparisonDimensions() []ComparisonDimension {
        return []ComparisonDimension{
                {Name: "Assumptions", Description: "The set of explicit assumptions each scenario makes."},
                {Name: "Inputs", Description: "The typed variables and their values."},
                {Name: "Methodology", Description: "The model and methodology used."},
                {Name: "Time horizon", Description: "Start, end, and duration of the exploration."},
                {Name: "Affected entities", Description: "Institutions, populations, or regions potentially affected."},
                {Name: "Modeled outputs", Description: "The outputs the model produced."},
                {Name: "Uncertainty", Description: "How uncertainty was disclosed."},
                {Name: "Evidence", Description: "Source evidence for each assumption."},
                {Name: "Limitations", Description: "Known limitations of the model and scenario."},
        }
}

// publishEvent publishes a lifecycle event.
func (s *ScenarioService) publishEvent(ctx context.Context, subject string, id domain.ID, tenantID domain.TenantID) {
        if s.Events == nil {
                return
        }
        payload, _ := json.Marshal(map[string]any{
                "scenario_id": id,
                "tenant_id":   tenantID,
                "timestamp":   s.Clock.Now(),
        })
        _ = s.Events.Publish(ctx, subject, payload)
}

// RunService is the application service for simulation runs.
type RunService struct {
        Repos   domain.ScenarioRepository
        Models  domain.ModelRepository
        Runs    domain.RunRepository
        Events  domain.EventPublisher
        Clock   domain.Clock
        Engine  domain.SimulationEngine
}

// NewRunService constructs a RunService.
func NewRunService(r domain.ScenarioRepository, m domain.ModelRepository, runs domain.RunRepository, e domain.EventPublisher, c domain.Clock, eng domain.SimulationEngine) *RunService {
        if c == nil {
                c = domain.SystemClock{}
        }
        return &RunService{Repos: r, Models: m, Runs: runs, Events: e, Clock: c, Engine: eng}
}

// Run executes a scenario. Phase 18 §10, §21–§24.
//
// This is the canonical execution path. The API never holds the request
// open; long-running simulations execute asynchronously via Temporal in
// production. Here we execute synchronously for the deterministic engine.
func (s *RunService) Run(ctx context.Context, tenantID domain.TenantID, scenarioID domain.ID, userID domain.ID, seed int64) (*domain.SimulationRun, error) {
        return s.runWithSuffix(ctx, tenantID, scenarioID, userID, seed, "")
}

// Replay re-executes a previous run using its stored configuration. Phase 18 §22, Gate C, Gate L.
func (s *RunService) Replay(ctx context.Context, tenantID domain.TenantID, runID domain.ID) (*domain.SimulationRun, error) {
        prev, err := s.Runs.Get(ctx, tenantID, runID)
        if err != nil {
                return nil, fmt.Errorf("replay: get run: %w", err)
        }
        return s.runWithSuffix(ctx, tenantID, prev.ScenarioID, prev.User, prev.RandomSeed, "-replay")
}

// runWithSuffix is the shared implementation. The suffix differentiates
// replays from the original run to avoid ID collisions.
func (s *RunService) runWithSuffix(ctx context.Context, tenantID domain.TenantID, scenarioID domain.ID, userID domain.ID, seed int64, runIDSuffix string) (*domain.SimulationRun, error) {
        sc, err := s.Repos.Get(ctx, tenantID, scenarioID)
        if err != nil {
                return nil, fmt.Errorf("run: get scenario: %w", err)
        }
        if sc.Status != domain.ScenarioStatusReady {
                return nil, fmt.Errorf("run: scenario must be READY, is %s", sc.Status)
        }

        m, err := s.Models.Get(ctx, sc.ModelID)
        if err != nil {
                return nil, fmt.Errorf("run: get model: %w", err)
        }

        // Validation pipeline. Phase 18 §21.
        validator := domain.ValidatePipeline{}
        if err := validator.Validate(*sc, *m); err != nil {
                return nil, fmt.Errorf("run: validation: %w", err)
        }

        // Construct the run record.
        runID := domain.ID(fmt.Sprintf("run-%s-%d%s", scenarioID, s.Clock.Now().UnixNano(), runIDSuffix))
        run := domain.SimulationRun{
                ID:              runID,
                TenantID:        tenantID,
                ScenarioID:      scenarioID,
                ScenarioVersion: sc.ScenarioVersion,
                ModelID:         m.ID,
                ModelVersion:    m.Version,
                EngineVersion:   s.Engine.Spec().Version,
                Status:          domain.RunStatusCreated,
                InputValues:     variablesToMap(sc.Variables),
                RandomSeed:      seed,
                ExecutionID:      string(runID),
                User:             userID,
                StartedAt:       s.Clock.Now(),
                Environment:     "local",
        }
        if err := s.Runs.Create(ctx, run); err != nil {
                return nil, fmt.Errorf("run: persist: %w", err)
        }

        s.publishEvent(ctx, "simulation.started", runID, tenantID)

        // Transition CREATED → VALIDATED.
        run.Status = domain.RunStatusValidated
        _ = s.Runs.Update(ctx, run)

        // Transition VALIDATED → STARTED → INPUTS_LOADED → MODEL_LOADED.
        run.Status = domain.RunStatusStarted
        _ = s.Runs.Update(ctx, run)
        run.Status = domain.RunStatusInputsLoaded
        _ = s.Runs.Update(ctx, run)
        run.Status = domain.RunStatusModelLoaded
        _ = s.Runs.Update(ctx, run)
        run.Status = domain.RunStatusSimulationStarted
        _ = s.Runs.Update(ctx, run)

        // Execute the engine.
        resp, err := s.Engine.Execute(ctx, domain.EngineRequest{
                Scenario:    *sc,
                Model:       *m,
                Variables:   sc.Variables,
                Assumptions: sc.Assumptions,
                RandomSeed:  seed,
                Timeout:     ctx,
        })
        if err != nil {
                run.Status = domain.RunStatusFailed
                run.Errors = append(run.Errors, domain.SimulationError{
                        Code:      "ENGINE_FAILED",
                        Message:   err.Error(),
                        Stage:     "SIMULATION_STARTED",
                        Timestamp: s.Clock.Now(),
                })
                _ = s.Runs.Update(ctx, run)
                s.publishEvent(ctx, "simulation.failed", runID, tenantID)
                return &run, fmt.Errorf("run: engine: %w", err)
        }

        // Transition SIMULATION_STARTED → SIMULATION_COMPLETED.
        run.Status = domain.RunStatusSimulationCompleted
        _ = s.Runs.Update(ctx, run)

        // Build the result.
        result := domain.SimulationResult{
                ID:           domain.ID(fmt.Sprintf("result-%s", runID)),
                RunID:        runID,
                TenantID:     tenantID,
                ScenarioID:   scenarioID,
                RealityLayer: domain.RealityLayerModeled,
                ModelID:      m.ID,
                ModelVersion: m.Version,
                EngineVersion: s.Engine.Spec().Version,
                GeneratedAt:  s.Clock.Now(),
                Outputs:      resp.Outputs,
                Uncertainty:  resp.Uncertainty,
                Timeline:     resp.Timeline,
                Impacts:      resp.Impacts,
                RandomSeed:   seed,
                Methodology: domain.Methodology{
                        Description: m.Methodology,
                        Inputs:      inputsToStrings(m.Inputs),
                        Outputs:     outputsToStrings(m.Outputs),
                        Limitations: m.Limitations,
                        EngineName:  s.Engine.Spec().Name,
                        EngineVer:   s.Engine.Spec().Version,
                },
        }
        // Compute input hash for reproducibility (Gate C, Gate L).
        result.InputHash = hashInputs(sc.Variables, sc.Assumptions, seed)

        if err := result.Validate(); err != nil {
                run.Status = domain.RunStatusFailed
                run.Errors = append(run.Errors, domain.SimulationError{
                        Code:      "RESULT_INVALID",
                        Message:   err.Error(),
                        Stage:     "RESULTS_VALIDATED",
                        Timestamp: s.Clock.Now(),
                })
                _ = s.Runs.Update(ctx, run)
                return &run, fmt.Errorf("run: result validation: %w", err)
        }

        run.Status = domain.RunStatusResultsValidated
        run.ResultSummary = &domain.ResultSummary{
                Status:        "OK",
                HeadlineMetric: headlineMetric(resp.Outputs),
                HeadlineValue:  headlineValue(resp.Outputs),
                Uncertainty:    resp.Uncertainty,
                Limitations:    m.Limitations,
                GeneratedAt:    s.Clock.Now(),
        }
        now := s.Clock.Now()
        run.CompletedAt = &now
        run.Duration = now.Sub(run.StartedAt)
        run.Status = domain.RunStatusPublished
        _ = s.Runs.Update(ctx, run)
        _ = s.Runs.AppendResult(ctx, result)

        s.publishEvent(ctx, "simulation.completed", runID, tenantID)
        return &run, nil
}

// publishEvent publishes a run lifecycle event.
func (s *RunService) publishEvent(ctx context.Context, subject string, id domain.ID, tenantID domain.TenantID) {
        if s.Events == nil {
                return
        }
        payload, _ := json.Marshal(map[string]any{
                "run_id":     id,
                "tenant_id":  tenantID,
                "timestamp":  s.Clock.Now(),
        })
        _ = s.Events.Publish(ctx, subject, payload)
}

func variablesToMap(vars []domain.ScenarioVariable) map[string]any {
        out := map[string]any{}
        for _, v := range vars {
                out[v.Name] = v.Value
        }
        return out
}

func inputsToStrings(in []domain.ModelInput) []string {
        out := make([]string, len(in))
        for i, v := range in {
                out[i] = fmt.Sprintf("%s (%s, %s)", v.Name, v.Type, v.Unit)
        }
        return out
}

func outputsToStrings(in []domain.ModelOutput) []string {
        out := make([]string, len(in))
        for i, v := range in {
                out[i] = fmt.Sprintf("%s (%s, %s)", v.Name, v.Type, v.Unit)
        }
        return out
}

func headlineMetric(outs []domain.ResultOutput) string {
        if len(outs) == 0 {
                return ""
        }
        return outs[0].Name
}

func headlineValue(outs []domain.ResultOutput) any {
        if len(outs) == 0 {
                return nil
        }
        return outs[0].Value
}

// hashInputs returns a deterministic hash of the run's input bundle. Two
// runs with the same hash AND same engine version MUST produce the same
// output (Gate L — Reproducibility).
func hashInputs(vars []domain.ScenarioVariable, asms []domain.ScenarioAssumption, seed int64) string {
        h := sha256.New()
        enc := json.NewEncoder(h)
        enc.SetEscapeHTML(false)
        _ = enc.Encode(vars)
        _ = enc.Encode(asms)
        _ = enc.Encode(seed)
        return hex.EncodeToString(h.Sum(nil))
}

// timeNow is a convenience for tests that want to inject a clock.
var timeNow = time.Now
