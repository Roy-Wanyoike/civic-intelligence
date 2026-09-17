package domain

import (
	"errors"
	"fmt"
	"time"
)

// ModelStatus is the lifecycle of a scenario model. Phase 18 §9.
type ModelStatus string

const (
	ModelStatusDraft      ModelStatus = "DRAFT"
	ModelStatusTesting    ModelStatus = "TESTING"
	ModelStatusValidated  ModelStatus = "VALIDATED"
	ModelStatusActive     ModelStatus = "ACTIVE"
	ModelStatusDeprecated ModelStatus = "DEPRECATED"
	ModelStatusRevoked    ModelStatus = "REVOKED"
)

// ScenarioModel is a registered model that can be used by scenarios.
// Phase 18 §9.
type ScenarioModel struct {
	ID              ID
	Name            string
	Version         string
	Description     string
	Methodology     string
	Inputs          []ModelInput
	Outputs         []ModelOutput
	Limitations     []string
	Author          ID
	ValidationStatus ModelStatus
	CreatedAt       time.Time
}

// ModelInput declares an input the model requires.
type ModelInput struct {
	Name         string
	Type         VariableType
	Unit         string
	Required     bool
	Description  string
}

// ModelOutput declares an output the model produces.
type ModelOutput struct {
	Name         string
	Type         VariableType
	Unit         string
	Description  string
	UncertaintyCapable bool
}

// CanBeUsed returns true if the model is in a state where it can be used by
// active scenarios. Only VALIDATED and ACTIVE models are usable.
func (m ScenarioModel) CanBeUsed() bool {
	return m.ValidationStatus == ModelStatusValidated ||
		m.ValidationStatus == ModelStatusActive
}

// Validate enforces model invariants.
func (m ScenarioModel) Validate() error {
	if m.ID == "" {
		return errors.New("model: missing ID")
	}
	if m.Name == "" {
		return errors.New("model: missing Name")
	}
	if m.Version == "" {
		return errors.New("model: missing Version")
	}
	if m.Methodology == "" {
		return errors.New("model: missing Methodology")
	}
	if len(m.Inputs) == 0 {
		return errors.New("model: must declare at least one input")
	}
	if len(m.Outputs) == 0 {
		return errors.New("model: must declare at least one output")
	}
	// Limitations disclosure is mandatory (Gate E).
	if len(m.Limitations) == 0 {
		return errors.New("model: must declare at least one limitation")
	}
	return nil
}

// ModelRegistryVersion is an immutable snapshot of a model.
type ModelRegistryVersion struct {
	ModelID    ID
	Version    string
	Snapshot   ScenarioModel
	CreatedAt  time.Time
	CreatedBy  ID
}

// RunStatus is the lifecycle of a SimulationRun. Phase 18 §23.
type RunStatus string

const (
	RunStatusCreated           RunStatus = "CREATED"
	RunStatusValidated         RunStatus = "VALIDATED"
	RunStatusStarted          RunStatus = "STARTED"
	RunStatusInputsLoaded     RunStatus = "INPUTS_LOADED"
	RunStatusModelLoaded      RunStatus = "MODEL_LOADED"
	RunStatusSimulationStarted RunStatus = "SIMULATION_STARTED"
	RunStatusSimulationCompleted RunStatus = "SIMULATION_COMPLETED"
	RunStatusResultsValidated RunStatus = "RESULTS_VALIDATED"
	RunStatusPublished        RunStatus = "PUBLISHED"
	RunStatusFailed           RunStatus = "FAILED"
	RunStatusCancelled        RunStatus = "CANCELLED"
)

// CanTransitionRun is the lifecycle state machine for SimulationRun.
func CanTransitionRun(from, to RunStatus) bool {
	switch from {
	case RunStatusCreated:
		return to == RunStatusValidated || to == RunStatusFailed
	case RunStatusValidated:
		return to == RunStatusStarted || to == RunStatusFailed
	case RunStatusStarted:
		return to == RunStatusInputsLoaded || to == RunStatusFailed || to == RunStatusCancelled
	case RunStatusInputsLoaded:
		return to == RunStatusModelLoaded || to == RunStatusFailed
	case RunStatusModelLoaded:
		return to == RunStatusSimulationStarted || to == RunStatusFailed
	case RunStatusSimulationStarted:
		return to == RunStatusSimulationCompleted || to == RunStatusFailed || to == RunStatusCancelled
	case RunStatusSimulationCompleted:
		return to == RunStatusResultsValidated || to == RunStatusFailed
	case RunStatusResultsValidated:
		return to == RunStatusPublished || to == RunStatusFailed
	case RunStatusPublished:
		return false // terminal success
	case RunStatusFailed, RunStatusCancelled:
		return false // terminal failure
	}
	return false
}

// SimulationRun is the audit-trail record for a single execution. Phase 18 §22, §23.
type SimulationRun struct {
	ID             ID
	TenantID       TenantID
	ScenarioID     ID
	ScenarioVersion int
	ModelID        ID
	ModelVersion   string
	EngineVersion  string
	Status         RunStatus

	// Reproducibility bundle. Phase 18 §22.
	InputValues      map[string]any
	AssumptionVersions []AssumptionVersionRef
	DatasetVersions  []DatasetVersionRef
	RandomSeed       int64
	ExecutionID      string
	AgentVersions    []AgentVersionRef
	PromptVersions   []string
	ToolVersions     []string
	Environment      string

	// Audit fields.
	WorkflowID  string
	TraceID     string
	User        ID
	Agent       string
	SystemVer   string
	StartedAt   time.Time
	CompletedAt *time.Time
	Duration    time.Duration

	// Resource consumption.
	ResourceUsage ResourceUsage

	// Errors collected during execution.
	Errors []SimulationError

	// Result summary; details live in SimulationResult.
	ResultSummary *ResultSummary
}

// AssumptionVersionRef is a pointer to the version of an assumption used in a
// run. Phase 18 §22.
type AssumptionVersionRef struct {
	AssumptionID string
	Version      int
}

// DatasetVersionRef is a pointer to a dataset version used in a run.
type DatasetVersionRef struct {
	DatasetID string
	Version   string
	Hash      string // content hash for integrity
}

// AgentVersionRef is a pointer to the version of an AI agent involved.
type AgentVersionRef struct {
	AgentID string
	Version string
}

// ResourceUsage records resource consumption for the run.
type ResourceUsage struct {
	CPUMillis       int64
	MemoryMB        int64
	DurationSeconds int64
	NetworkBytes    int64
}

// SimulationError is a structured error.
type SimulationError struct {
	Code      string
	Message   string
	Stage     string // which stage of the pipeline failed
	Timestamp time.Time
}

// ResultSummary is the high-level result; full results are in SimulationResult.
type ResultSummary struct {
	Status       string
	HeadlineMetric string
	HeadlineValue  any
	Uncertainty    UncertaintySummary
	Limitations     []string
	GeneratedAt    time.Time
}

// UncertaintySummary is the high-level uncertainty disclosure. Phase 18 §11.
type UncertaintySummary struct {
	HasUncertainty bool
	Median         *float64
	P10            *float64
	P90            *float64
	Min            *float64
	Max            *float64
	Notes          string
}

// SimulationResult is the full result payload. Phase 18 §11.
type SimulationResult struct {
	ID              ID
	RunID           ID
	TenantID        TenantID
	ScenarioID      ID
	RealityLayer    RealityLayer // always MODELED
	ModelID         ID
	ModelVersion    string
	EngineVersion   string
	GeneratedAt     time.Time

	Outputs         []ResultOutput
	Uncertainty     UncertaintySummary
	Timeline        []TimelineEvent
	Impacts         []ModeledImpact
	ComparisonRefs  []ID // sibling scenario IDs if this is part of a comparison

	// Reproducibility metadata.
	RandomSeed      int64
	InputHash        string
	Methodology      Methodology
}

// ResultOutput is a single modeled output value. Always tagged SIMULATION.
type ResultOutput struct {
	Name         string
	Type         VariableType
	Unit         string
	Value        any
	Uncertainty  UncertaintySummary
	RealityLayer RealityLayer // always MODELED
}

// Validate enforces result invariants.
func (r SimulationResult) Validate() error {
	if r.RealityLayer != RealityLayerModeled {
		return fmt.Errorf("result: reality layer must be MODELED, got %s (canonical truth protection — Gate I)", r.RealityLayer)
	}
	if len(r.Outputs) == 0 {
		return errors.New("result: must declare at least one output")
	}
	for i, o := range r.Outputs {
		if o.RealityLayer != RealityLayerModeled {
			return fmt.Errorf("result: output[%d] must be tagged MODELED", i)
		}
	}
	// Gate F: scenarios capable of uncertainty must expose it, not point precision.
	for _, o := range r.Outputs {
		if o.Uncertainty.HasUncertainty && o.Uncertainty.Median == nil {
			return fmt.Errorf("result output %s: declared uncertainty but no median provided", o.Name)
		}
	}
	return nil
}

// ModeledImpact is a downstream effect predicted by the model. Phase 18 §15.
type ModeledImpact struct {
	FromEntity  string
	ToEntity    string
	RelationType string // OBSERVED_RELATIONSHIP / HYPOTHETICAL_RELATIONSHIP / MODELED_RELATIONSHIP
	Strength    float64
	Uncertainty UncertaintySummary
	Evidence    []EvidenceRef
}

// TimelineEventLabel classifies a scenario timeline event. Phase 18 §14.
type TimelineEventLabel string

const (
	TimelineObserved TimelineEventLabel = "OBSERVED"
	TimelineAssumed  TimelineEventLabel = "ASSUMED"
	TimelineModeled TimelineEventLabel = "MODELED"
	TimelineUnknown TimelineEventLabel = "UNKNOWN"
)

// TimelineEvent is a single point on the scenario timeline. Phase 18 §14.
type TimelineEvent struct {
	Timestamp time.Time
	Label     TimelineEventLabel
	Title     string
	Description string
	Evidence  []EvidenceRef
}
