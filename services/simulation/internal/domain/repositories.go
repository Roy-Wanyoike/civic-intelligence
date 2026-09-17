package domain

import (
	"context"
	"time"
)

// ScenarioRepository persists scenarios and their versions.
type ScenarioRepository interface {
	Create(ctx context.Context, s Scenario) error
	Get(ctx context.Context, tenantID TenantID, id ID) (*Scenario, error)
	List(ctx context.Context, tenantID TenantID, filter ScenarioFilter) ([]Scenario, error)
	Update(ctx context.Context, s Scenario) error
	Archive(ctx context.Context, tenantID TenantID, id ID) error

	// Versions are append-only. Phase 18 §22.
	AppendVersion(ctx context.Context, v ScenarioVersion) error
	GetVersion(ctx context.Context, id ID, version int) (*ScenarioVersion, error)
	ListVersions(ctx context.Context, id ID) ([]ScenarioVersion, error)
}

// ScenarioFilter is a query filter for scenarios.
type ScenarioFilter struct {
	Type         []ScenarioType
	Status       []ScenarioStatus
	Jurisdiction string
	CreatedBy    *ID
	Limit        int
	Offset       int
}

// ModelRepository persists scenario models.
type ModelRepository interface {
	Create(ctx context.Context, m ScenarioModel) error
	Get(ctx context.Context, id ID) (*ScenarioModel, error)
	List(ctx context.Context, filter ModelFilter) ([]ScenarioModel, error)
	Update(ctx context.Context, m ScenarioModel) error
	AppendVersion(ctx context.Context, v ModelRegistryVersion) error
}

// ModelFilter is a query filter for models.
type ModelFilter struct {
	Status       []ModelStatus
	Limit        int
	Offset       int
}

// RunRepository persists simulation runs and their results.
type RunRepository interface {
	Create(ctx context.Context, r SimulationRun) error
	Get(ctx context.Context, tenantID TenantID, id ID) (*SimulationRun, error)
	List(ctx context.Context, tenantID TenantID, scenarioID *ID) ([]SimulationRun, error)
	Update(ctx context.Context, r SimulationRun) error
	AppendResult(ctx context.Context, r SimulationResult) error
	GetResult(ctx context.Context, runID ID) (*SimulationResult, error)
}

// EventPublisher publishes simulation lifecycle events. Phase 18 §25.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}

// Clock is the time source, injected for testability.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production implementation.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
