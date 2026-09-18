// Package domain contains the pure domain model for scenarios, assumptions,
// models, simulation runs and results.
//
// Reality / Simulation separation (Phase 18 §1)
// ----------------------------------------------
// Every record in this domain carries an explicit RealityLayer tag so that
// no simulation record can ever be confused with an observed civic fact.
// The boundary is enforced at the type level.
package domain

import (
        "errors"
        "fmt"
        "time"
)

// ID is the platform-wide identifier type. Matches the legislation service.
type ID string

// TenantID scopes every scenario to a single tenant. The simulation service
// never allows cross-tenant access.
type TenantID string

// RealityLayer is the architectural boundary between observed reality and
// simulation. EVERY record in the simulation domain is tagged SIMULATION or
// HYPOTHETICAL — never OBSERVED. Observed records belong to the legislation
// service, not here.
type RealityLayer string

const (
        // RealityLayerObserved — owned by the legislation service, not this package.
        RealityLayerObserved RealityLayer = "OBSERVED"
        // RealityLayerHypothetical — a constructed scenario input that has not
        // been observed in reality.
        RealityLayerHypothetical RealityLayer = "HYPOTHETICAL"
        // RealityLayerModeled — a simulated output produced by a model.
        RealityLayerModeled RealityLayer = "MODELED"
        // RealityLayerUnknown — insufficient evidence to classify.
        RealityLayerUnknown RealityLayer = "UNKNOWN"
)

// ScenarioType categorises the kind of exploration. Phase 18 §3.
type ScenarioType string

const (
        ScenarioTypePolicy           ScenarioType = "POLICY"
        ScenarioTypeLegislative      ScenarioType = "LEGISLATIVE"
        ScenarioTypeImplementation   ScenarioType = "IMPLEMENTATION"
        ScenarioTypeComparative      ScenarioType = "COMPARATIVE"
        ScenarioTypeCounterfactual   ScenarioType = "HISTORICAL_COUNTERFACTUAL"
        ScenarioTypeInstitutional    ScenarioType = "INSTITUTIONAL"
        ScenarioTypeEconomicSocial   ScenarioType = "ECONOMIC_SOCIAL"
        ScenarioTypeInfrastructure   ScenarioType = "INFRASTRUCTURE"
)

// ScenarioStatus is the lifecycle state. Phase 18 §2.
type ScenarioStatus string

const (
        ScenarioStatusDraft          ScenarioStatus = "DRAFT"
        ScenarioStatusConfigured     ScenarioStatus = "CONFIGURED"
        ScenarioStatusValidating     ScenarioStatus = "VALIDATING"
        ScenarioStatusReady         ScenarioStatus = "READY"
        ScenarioStatusRunning        ScenarioStatus = "RUNNING"
        ScenarioStatusCompleted      ScenarioStatus = "COMPLETED"
        ScenarioStatusReviewRequired ScenarioStatus = "REVIEW_REQUIRED"
        ScenarioStatusArchived       ScenarioStatus = "ARCHIVED"
        ScenarioStatusFailed         ScenarioStatus = "FAILED"
)

// Scenario is the first-class hypothetical exploration object.
//
// INVARIANT: A Scenario is always tagged RealityLayerHypothetical. It can
// reference observed reality (source_evidence), but it is never itself an
// observed fact.
type Scenario struct {
        ID             ID             `json:"id"`
        TenantID       TenantID       `json:"tenant_id"`
        Name           string         `json:"name"`
        Description    string         `json:"description"`
        Type           ScenarioType   `json:"type"`
        Jurisdiction   string         `json:"jurisdiction"`
        RealityLayer   RealityLayer   `json:"reality_layer"`
        CreatedBy      ID             `json:"created_by"`
        CreatedAt      time.Time      `json:"created_at"`
        UpdatedAt      time.Time      `json:"updated_at"`

        Baseline       Baseline        `json:"baseline"`
        Assumptions    []ScenarioAssumption `json:"assumptions"`
        Variables      []ScenarioVariable  `json:"variables"`
        Constraints    []ScenarioConstraint `json:"constraints"`
        TimeHorizon    TimeHorizon    `json:"time_horizon"`

        SourceEvidence []EvidenceRef  `json:"source_evidence"`
        Methodology    Methodology    `json:"methodology"`
        ModelID        ID             `json:"model_id"`
        ModelVersion   string         `json:"model_version"`

        Status             ScenarioStatus     `json:"status"`
        ConfidenceMetadata ConfidenceMetadata `json:"confidence_metadata"`

        CreatedFrom     CreationOrigin `json:"created_from"`
        ScenarioVersion int            `json:"scenario_version"`
}

// Baseline is the observed starting point. Every scenario must declare its
// baseline explicitly so users can see where reality ends and the
// hypothetical begins.
type Baseline struct {
        Description  string         `json:"description"`
        AsOf          time.Time      `json:"as_of"`
        SourceRefs    []EvidenceRef  `json:"source_refs"`
        RealityLayer RealityLayer   `json:"reality_layer"`
}

// TimeHorizon is the period the scenario explores.
type TimeHorizon struct {
        Start    time.Time `json:"start"`
        End      time.Time `json:"end"`
        Duration  string    `json:"duration"`
}

// CreationOrigin records how the scenario was constructed (human, AI-assisted,
// imported). AI-generated components must be traceable per Phase 18 §19.
type CreationOrigin struct {
        Source    string // "HUMAN", "AI_ASSISTED", "IMPORTED", "TEMPLATE"
        AgentID   string // empty if human-only
        AgentVer  string
        PromptVer string
        ToolVer   string
}

// ConfidenceMetadata captures the overall confidence of the scenario. It is
// NEVER a single number — it is decomposed per Phase 18 §11 and §24.
type ConfidenceMetadata struct {
        EvidenceCoverage  ConfidenceLevel // how well-evidenced are the inputs?
        AssumptionClarity ConfidenceLevel // how explicit are the assumptions?
        ModelValidity     ConfidenceLevel // is the model fit for purpose?
        Uncertainty       ConfidenceLevel // is uncertainty disclosed?
        Notes             string
}

// ConfidenceLevel is a coarse 4-level confidence rating. We deliberately do
// NOT use a 0–100 score to avoid false precision.
type ConfidenceLevel string

const (
        ConfidenceHigh    ConfidenceLevel = "HIGH"
        ConfidenceMedium  ConfidenceLevel = "MEDIUM"
        ConfidenceLow     ConfidenceLevel = "LOW"
        ConfidenceUnknown ConfidenceLevel = "UNKNOWN"
)

// Validate enforces the scenario invariants. Phase 18 §21.
func (s *Scenario) Validate() error {
        if s.ID == "" {
                return errors.New("scenario: missing ID")
        }
        if s.TenantID == "" {
                return errors.New("scenario: missing TenantID")
        }
        if s.Name == "" {
                return errors.New("scenario: missing Name")
        }
        if s.Jurisdiction == "" {
                return errors.New("scenario: missing Jurisdiction")
        }
        // Reality separation: a scenario must be tagged HYPOTHETICAL.
        if s.RealityLayer != RealityLayerHypothetical {
                return fmt.Errorf("scenario: reality layer must be HYPOTHETICAL, got %s", s.RealityLayer)
        }
        // Baseline reality separation: the baseline itself is OBSERVED.
        if s.Baseline.RealityLayer != RealityLayerObserved {
                return errors.New("scenario: baseline must be tagged OBSERVED")
        }
        if s.TimeHorizon.Start.IsZero() || s.TimeHorizon.End.IsZero() {
                return errors.New("scenario: time horizon must be set")
        }
        if !s.TimeHorizon.End.After(s.TimeHorizon.Start) {
                return errors.New("scenario: time horizon end must be after start")
        }
        if s.ModelID == "" {
                return errors.New("scenario: missing ModelID")
        }
        // Every scenario must expose its assumptions (Gate D).
        if len(s.Assumptions) == 0 {
                return errors.New("scenario: must declare at least one assumption")
        }
        // Validate every assumption.
        for i, a := range s.Assumptions {
                if err := a.Validate(); err != nil {
                        return fmt.Errorf("scenario: assumption[%d]: %w", i, err)
                }
        }
        // Validate every variable.
        for i, v := range s.Variables {
                if err := v.Validate(); err != nil {
                        return fmt.Errorf("scenario: variable[%d]: %w", i, err)
                }
        }
        return nil
}

// CanTransition returns true if the lifecycle transition is permitted.
// Phase 18 §2.
func CanTransition(from, to ScenarioStatus) bool {
        switch from {
        case ScenarioStatusDraft:
                return to == ScenarioStatusConfigured || to == ScenarioStatusArchived
        case ScenarioStatusConfigured:
                return to == ScenarioStatusValidating || to == ScenarioStatusDraft || to == ScenarioStatusArchived
        case ScenarioStatusValidating:
                return to == ScenarioStatusReady || to == ScenarioStatusConfigured || to == ScenarioStatusFailed
        case ScenarioStatusReady:
                return to == ScenarioStatusRunning || to == ScenarioStatusValidating || to == ScenarioStatusArchived
        case ScenarioStatusRunning:
                return to == ScenarioStatusCompleted || to == ScenarioStatusFailed || to == ScenarioStatusReviewRequired
        case ScenarioStatusCompleted:
                return to == ScenarioStatusReviewRequired || to == ScenarioStatusArchived || to == ScenarioStatusReady
        case ScenarioStatusReviewRequired:
                return to == ScenarioStatusReady || to == ScenarioStatusArchived || to == ScenarioStatusCompleted
        case ScenarioStatusFailed:
                return to == ScenarioStatusDraft || to == ScenarioStatusArchived
        case ScenarioStatusArchived:
                return false // terminal
        }
        return false
}

// ScenarioVersion is an immutable snapshot of a scenario at a particular
// version. Versions are append-only — never overwrite historical truth.
type ScenarioVersion struct {
        ScenarioID      ID
        Version         int
        Snapshot        Scenario
        CreatedAt       time.Time
        CreatedBy       ID
        ChangeSummary   string
}

// EvidenceRef points at a verified civic fact or document. Phase 18 §8.
type EvidenceRef struct {
        // Kind is "DOCUMENT", "FACT", "RELATIONSHIP", or "EXTERNAL".
        Kind string `json:"kind"`
        // ID is the canonical ID of the referenced evidence.
        ID string `json:"id"`
        // SourceURL is the authoritative source URL.
        SourceURL string `json:"source_url"`
        // RetrievedAt is when the evidence was retrieved.
        RetrievedAt time.Time `json:"retrieved_at"`
        // VerificationStatus of the referenced evidence.
        VerificationStatus string `json:"verification_status"`
}

// Methodology describes how the scenario was produced. Phase 18 §9, §22.
type Methodology struct {
        Description  string        `json:"description"`
        Inputs       []string      `json:"inputs"`
        Outputs      []string      `json:"outputs"`
        Limitations  []string      `json:"limitations"`
        EngineName   string        `json:"engine_name"`
        EngineVer    string        `json:"engine_version"`
        References   []EvidenceRef `json:"references"`
}
