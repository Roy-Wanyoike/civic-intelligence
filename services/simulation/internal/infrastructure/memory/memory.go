// Package memory provides in-memory implementations of the simulation
// repositories. In production these are backed by PostgreSQL; the in-memory
// implementation is used by tests, the API service's local fallback, and
// the golden scenario dataset.
package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/Roy-Wanyoike/civic-intelligence/services/simulation/internal/domain"
)

// ScenarioRepo is an in-memory ScenarioRepository.
type ScenarioRepo struct {
	mu        sync.RWMutex
	scenarios map[domain.ID]domain.Scenario
	versions  map[domain.ID][]domain.ScenarioVersion
}

// NewScenarioRepo constructs an empty in-memory scenario repo.
func NewScenarioRepo() *ScenarioRepo {
	return &ScenarioRepo{
		scenarios: map[domain.ID]domain.Scenario{},
		versions:  map[domain.ID][]domain.ScenarioVersion{},
	}
}

// Create persists a scenario.
func (r *ScenarioRepo) Create(ctx context.Context, s domain.Scenario) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.scenarios[s.ID]; exists {
		return fmt.Errorf("scenario %s already exists", s.ID)
	}
	r.scenarios[s.ID] = s
	return nil
}

// Get retrieves a scenario with tenant isolation (Gate G).
func (r *ScenarioRepo) Get(ctx context.Context, tenantID domain.TenantID, id domain.ID) (*domain.Scenario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.scenarios[id]
	if !ok {
		return nil, fmt.Errorf("scenario %s not found", id)
	}
	if s.TenantID != tenantID {
		// Tenant isolation: do not leak existence to other tenants.
		return nil, fmt.Errorf("scenario %s not found", id)
	}
	sc := s
	return &sc, nil
}

// List lists scenarios for a tenant.
func (r *ScenarioRepo) List(ctx context.Context, tenantID domain.TenantID, filter domain.ScenarioFilter) ([]domain.Scenario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.Scenario{}
	for _, s := range r.scenarios {
		if s.TenantID != tenantID {
			continue
		}
		if !matchFilter(s, filter) {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// Update mutates a scenario.
func (r *ScenarioRepo) Update(ctx context.Context, s domain.Scenario) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.scenarios[s.ID]; !ok {
		return fmt.Errorf("scenario %s not found", s.ID)
	}
	r.scenarios[s.ID] = s
	return nil
}

// Archive marks a scenario as archived.
func (r *ScenarioRepo) Archive(ctx context.Context, tenantID domain.TenantID, id domain.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.scenarios[id]
	if !ok || s.TenantID != tenantID {
		return fmt.Errorf("scenario %s not found", id)
	}
	s.Status = domain.ScenarioStatusArchived
	r.scenarios[id] = s
	return nil
}

// AppendVersion appends an immutable snapshot. Phase 18 section 22.
func (r *ScenarioRepo) AppendVersion(ctx context.Context, v domain.ScenarioVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.versions[v.ScenarioID] = append(r.versions[v.ScenarioID], v)
	return nil
}

// GetVersion retrieves a specific version.
func (r *ScenarioRepo) GetVersion(ctx context.Context, id domain.ID, version int) (*domain.ScenarioVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.versions[id] {
		if v.Version == version {
			out := v
			return &out, nil
		}
	}
	return nil, fmt.Errorf("scenario %s version %d not found", id, version)
}

// ListVersions lists all versions.
func (r *ScenarioRepo) ListVersions(ctx context.Context, id domain.ID) ([]domain.ScenarioVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.ScenarioVersion, len(r.versions[id]))
	copy(out, r.versions[id])
	return out, nil
}

func matchFilter(s domain.Scenario, f domain.ScenarioFilter) bool {
	if len(f.Type) > 0 && !containsType(f.Type, s.Type) {
		return false
	}
	if len(f.Status) > 0 && !containsStatus(f.Status, s.Status) {
		return false
	}
	if f.Jurisdiction != "" && s.Jurisdiction != f.Jurisdiction {
		return false
	}
	if f.CreatedBy != nil && s.CreatedBy != *f.CreatedBy {
		return false
	}
	return true
}

func containsType(ts []domain.ScenarioType, t domain.ScenarioType) bool {
	for _, x := range ts {
		if x == t {
			return true
		}
	}
	return false
}

func containsStatus(ss []domain.ScenarioStatus, s domain.ScenarioStatus) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// ModelRepo is an in-memory ModelRepository.
type ModelRepo struct {
	mu       sync.RWMutex
	models   map[domain.ID]domain.ScenarioModel
	versions map[domain.ID][]domain.ModelRegistryVersion
}

// NewModelRepo constructs an empty in-memory model repo.
func NewModelRepo() *ModelRepo {
	return &ModelRepo{
		models:   map[domain.ID]domain.ScenarioModel{},
		versions: map[domain.ID][]domain.ModelRegistryVersion{},
	}
}

// Create persists a model.
func (r *ModelRepo) Create(ctx context.Context, m domain.ScenarioModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.models[m.ID]; exists {
		return fmt.Errorf("model %s already exists", m.ID)
	}
	r.models[m.ID] = m
	return nil
}

// Get retrieves a model.
func (r *ModelRepo) Get(ctx context.Context, id domain.ID) (*domain.ScenarioModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[id]
	if !ok {
		return nil, fmt.Errorf("model %s not found", id)
	}
	out := m
	return &out, nil
}

// List lists models.
func (r *ModelRepo) List(ctx context.Context, filter domain.ModelFilter) ([]domain.ScenarioModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.ScenarioModel{}
	for _, m := range r.models {
		if len(filter.Status) > 0 && !containsModelStatus(filter.Status, m.ValidationStatus) {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// Update mutates a model.
func (r *ModelRepo) Update(ctx context.Context, m domain.ScenarioModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.models[m.ID]; !ok {
		return fmt.Errorf("model %s not found", m.ID)
	}
	r.models[m.ID] = m
	return nil
}

// AppendVersion appends a model snapshot.
func (r *ModelRepo) AppendVersion(ctx context.Context, v domain.ModelRegistryVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.versions[v.ModelID] = append(r.versions[v.ModelID], v)
	return nil
}

func containsModelStatus(ss []domain.ModelStatus, s domain.ModelStatus) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// RunRepo is an in-memory RunRepository.
type RunRepo struct {
	mu      sync.RWMutex
	runs    map[domain.ID]domain.SimulationRun
	results map[domain.ID]domain.SimulationResult
}

// NewRunRepo constructs an empty in-memory run repo.
func NewRunRepo() *RunRepo {
	return &RunRepo{
		runs:    map[domain.ID]domain.SimulationRun{},
		results: map[domain.ID]domain.SimulationResult{},
	}
}

// Create persists a run.
func (r *RunRepo) Create(ctx context.Context, run domain.SimulationRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.runs[run.ID]; exists {
		return fmt.Errorf("run %s already exists", run.ID)
	}
	r.runs[run.ID] = run
	return nil
}

// Get retrieves a run with tenant isolation (Gate G).
func (r *RunRepo) Get(ctx context.Context, tenantID domain.TenantID, id domain.ID) (*domain.SimulationRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[id]
	if !ok {
		return nil, fmt.Errorf("run %s not found", id)
	}
	if run.TenantID != tenantID {
		return nil, fmt.Errorf("run %s not found", id)
	}
	out := run
	return &out, nil
}

// List lists runs for a scenario.
func (r *RunRepo) List(ctx context.Context, tenantID domain.TenantID, scenarioID *domain.ID) ([]domain.SimulationRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.SimulationRun{}
	for _, run := range r.runs {
		if run.TenantID != tenantID {
			continue
		}
		if scenarioID != nil && run.ScenarioID != *scenarioID {
			continue
		}
		out = append(out, run)
	}
	return out, nil
}

// Update mutates a run.
func (r *RunRepo) Update(ctx context.Context, run domain.SimulationRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.runs[run.ID]; !ok {
		return fmt.Errorf("run %s not found", run.ID)
	}
	r.runs[run.ID] = run
	return nil
}

// AppendResult appends a result.
func (r *RunRepo) AppendResult(ctx context.Context, res domain.SimulationResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results[res.RunID] = res
	return nil
}

// GetResult retrieves a result.
func (r *RunRepo) GetResult(ctx context.Context, runID domain.ID) (*domain.SimulationResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.results[runID]
	if !ok {
		return nil, fmt.Errorf("result for run %s not found", runID)
	}
	out := res
	return &out, nil
}

// NoopEventPublisher is a no-op EventPublisher for tests and local dev.
type NoopEventPublisher struct{}

// Publish discards the event.
func (NoopEventPublisher) Publish(ctx context.Context, subject string, payload []byte) error {
	return nil
}
