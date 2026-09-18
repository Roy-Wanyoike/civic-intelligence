-- 019_simulation_schema.up.sql
-- Phase 18 — Reality / Simulation separation.
--
-- Introduces the `simulation` bounded context. Every record in this schema is
-- explicitly tagged with a RealityLayer (HYPOTHETICAL | MODELED) — it can
-- reference observed reality (evidence), but it is NEVER itself an observed
-- civic fact. The boundary is enforced at the type level in the domain layer
-- and reflected here in the `reality_layer` columns + CHECK constraints.
--
-- Architectural contract reminders (Phase 18 spec; ADR-0005 AI-cannot-mutate-truth):
--   * Canonical civic truth lives in legislation/ingestion/documents.
--   * AI may propose candidate scenarios, assumptions, and model outputs, but
--     may NEVER write to canonical state directly. Every AI-proposed row must
--     declare its origin (created_from JSONB).
--   * Scenarios are versioned (scenario_versions); historical versions are
--     IMMUTABLE — never overwrite past truth (ADR-0011 immutable versions).
--   * Simulation results are IMMUTABLE once published (Gate I — canonical
--     truth protection; a published result cannot be silently edited).
--   * Audit log (scenario_audits) is append-only.
--
-- Tenant isolation: every table carries `tenant_id` with a btree index. The
-- application layer MUST scope every query by tenant_id; this schema enforces
-- the column presence + index so cross-tenant scans are detectable.
--
-- Schema created here (forward-only; 002_schemas.up.sql is immutable).

BEGIN;

CREATE SCHEMA IF NOT EXISTS simulation;
COMMENT ON SCHEMA simulation IS 'Simulation domain — hypothetical scenarios, assumptions, models, runs, results. Every row is tagged SIMULATION/HYPOTHETICAL; NEVER observed civic truth.';

-- ===== simulation.scenarios =================================================
-- The first-class hypothetical exploration object. Phase 18 §1, §3.
--
-- A Scenario is always tagged RealityLayer=HYPOTHETICAL. It can reference
-- observed reality via scenario_evidence, but the scenario itself is a
-- constructed exploration, not an observed fact. The baseline is OBSERVED
-- (the starting point of reality from which the hypothetical departs).
--
-- status follows the ScenarioStatus lifecycle: DRAFT → CONFIGURED →
-- VALIDATING → READY → RUNNING → COMPLETED → REVIEW_REQUIRED → ARCHIVED
-- (with FAILED at any validation/execution step).

CREATE TABLE IF NOT EXISTS simulation.scenarios (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id           TEXT NOT NULL,
    name                TEXT NOT NULL,
    description         TEXT,
    type                TEXT NOT NULL,                  -- POLICY | LEGISLATIVE | IMPLEMENTATION | COMPARATIVE | HISTORICAL_COUNTERFACTUAL | INSTITUTIONAL | ECONOMIC_SOCIAL | INFRASTRUCTURE
    jurisdiction        TEXT NOT NULL,
    reality_layer       TEXT NOT NULL DEFAULT 'HYPOTHETICAL',
    created_by          UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    baseline            JSONB NOT NULL,                 -- {description, as_of, source_refs, reality_layer=OBSERVED}
    time_horizon        JSONB NOT NULL,                 -- {start, end, duration}
    methodology         JSONB,                         -- {description, inputs, outputs, limitations, engine_name, engine_version, references}
    model_id            UUID,                           -- FK to simulation.scenario_models(id) added below
    model_version       TEXT,
    status              TEXT NOT NULL DEFAULT 'DRAFT',
    confidence_metadata JSONB,                         -- {evidence_coverage, assumption_clarity, model_validity, uncertainty, notes}
    created_from        JSONB,                         -- {source: HUMAN|AI_ASSISTED|IMPORTED|TEMPLATE, agent_id, agent_ver, prompt_ver, tool_ver}
    scenario_version    INT NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_scenarios_reality CHECK (reality_layer = 'HYPOTHETICAL'),
    CONSTRAINT chk_scenarios_type CHECK (
        type IN ('POLICY','LEGISLATIVE','IMPLEMENTATION','COMPARATIVE',
                 'HISTORICAL_COUNTERFACTUAL','INSTITUTIONAL',
                 'ECONOMIC_SOCIAL','INFRASTRUCTURE')
    ),
    CONSTRAINT chk_scenarios_status CHECK (
        status IN ('DRAFT','CONFIGURED','VALIDATING','READY','RUNNING',
                   'COMPLETED','REVIEW_REQUIRED','ARCHIVED','FAILED')
    )
);
CREATE INDEX idx_sim_scenarios_tenant       ON simulation.scenarios(tenant_id);
CREATE INDEX idx_sim_scenarios_status       ON simulation.scenarios(tenant_id, status);
CREATE INDEX idx_sim_scenarios_type         ON simulation.scenarios(tenant_id, type);
CREATE INDEX idx_sim_scenarios_model        ON simulation.scenarios(model_id);
CREATE INDEX idx_sim_scenarios_updated      ON simulation.scenarios(tenant_id, updated_at DESC);
COMMENT ON COLUMN simulation.scenarios.reality_layer IS 'Always HYPOTHETICAL for a scenario. Observed reality lives in legislation.*';

-- ===== simulation.scenario_models ===========================================
-- Registered models that can be used by scenarios. Phase 18 §9.
-- validation_status follows ModelStatus: DRAFT → TESTING → VALIDATED →
-- ACTIVE → DEPRECATED → REVOKED. Only VALIDATED/ACTIVE models are usable by
-- active scenarios (enforced in the application layer).

CREATE TABLE IF NOT EXISTS simulation.scenario_models (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id         TEXT NOT NULL,
    name              TEXT NOT NULL,
    version           TEXT NOT NULL,
    description       TEXT,
    methodology       TEXT NOT NULL,
    inputs            JSONB NOT NULL DEFAULT '[]'::jsonb,    -- [{name, type, unit, required, description}]
    outputs           JSONB NOT NULL DEFAULT '[]'::jsonb,    -- [{name, type, unit, description, uncertainty_capable}]
    limitations       JSONB NOT NULL DEFAULT '[]'::jsonb,    -- array of TEXT
    author            UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    validation_status TEXT NOT NULL DEFAULT 'DRAFT',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name, version),
    CONSTRAINT chk_sim_models_status CHECK (
        validation_status IN ('DRAFT','TESTING','VALIDATED','ACTIVE','DEPRECATED','REVOKED')
    )
);
CREATE INDEX idx_sim_models_tenant  ON simulation.scenario_models(tenant_id);
CREATE INDEX idx_sim_models_status ON simulation.scenario_models(tenant_id, validation_status);

-- Now add the FK from scenarios.model_id to scenario_models.id.
ALTER TABLE simulation.scenarios
    ADD CONSTRAINT fk_scenarios_model
    FOREIGN KEY (model_id) REFERENCES simulation.scenario_models(id) ON DELETE SET NULL;

-- ===== simulation.scenario_versions =========================================
-- IMMUTABLE snapshot of a scenario at a particular version. Phase 18 §1.
-- Versions are append-only — never overwrite historical truth (ADR-0011).
--
-- The immutability trigger (below) blocks UPDATE / DELETE / TRUNCATE so the
-- historical record cannot be silently rewritten. To "change" a scenario, a
-- new version row is inserted and scenarios.scenario_version is bumped.

CREATE TABLE IF NOT EXISTS simulation.scenario_versions (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id      TEXT NOT NULL,
    scenario_id    UUID NOT NULL REFERENCES simulation.scenarios(id) ON DELETE RESTRICT,
    version        INT NOT NULL CHECK (version > 0),
    snapshot       JSONB NOT NULL,                     -- full Scenario struct at this version
    change_summary TEXT,
    created_by     UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (scenario_id, version)
);
CREATE INDEX idx_sim_scenario_versions_tenant ON simulation.scenario_versions(tenant_id);
CREATE INDEX idx_sim_scenario_versions_scn   ON simulation.scenario_versions(scenario_id, version DESC);
COMMENT ON COLUMN simulation.scenario_versions.snapshot IS 'Immutable full-state JSONB snapshot of the scenario at this version. NEVER edited — to change a scenario, insert a new version.';

-- ===== simulation.scenario_assumptions =====================================
-- Explicit, traceable assumptions declared by a scenario. Phase 18 §7.
-- assumption_type follows AssumptionStatus: OBSERVED_INPUT | USER_DEFINED |
-- MODEL_ASSUMPTION | HISTORICAL_REFERENCE | ESTIMATE | UNKNOWN.
--
-- Gate B (Provenance): externally-sourced assumptions (OBSERVED_INPUT,
-- HISTORICAL_REFERENCE, ESTIMATE) MUST carry at least one source_evidence
-- entry. UNKNOWN assumptions MUST NOT carry a fabricated value — enforced in
-- the application layer.

CREATE TABLE IF NOT EXISTS simulation.scenario_assumptions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       TEXT NOT NULL,
    scenario_id     UUID NOT NULL REFERENCES simulation.scenarios(id) ON DELETE CASCADE,
    statement       TEXT NOT NULL,
    value           JSONB,                              -- NULL when assumption_type=UNKNOWN
    unit            TEXT,
    basis           TEXT,
    source_evidence JSONB NOT NULL DEFAULT '[]'::jsonb, -- array of EvidenceRef
    assumption_type TEXT NOT NULL,
    confidence      TEXT NOT NULL DEFAULT 'UNKNOWN',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sim_assumptions_type CHECK (
        assumption_type IN ('OBSERVED_INPUT','USER_DEFINED','MODEL_ASSUMPTION',
                            'HISTORICAL_REFERENCE','ESTIMATE','UNKNOWN')
    ),
    CONSTRAINT chk_sim_assumptions_confidence CHECK (
        confidence IN ('HIGH','MEDIUM','LOW','UNKNOWN')
    )
);
CREATE INDEX idx_sim_assumptions_tenant ON simulation.scenario_assumptions(tenant_id);
CREATE INDEX idx_sim_assumptions_scn    ON simulation.scenario_assumptions(scenario_id);
CREATE INDEX idx_sim_assumptions_type   ON simulation.scenario_assumptions(tenant_id, assumption_type);

-- ===== simulation.scenario_inputs ===========================================
-- Typed input variables for a scenario. Phase 18 §6.
-- type follows VariableType: INTEGER | DECIMAL | PERCENTAGE | CURRENCY | DATE
-- | DURATION | BOOLEAN | CATEGORICAL | GEOGRAPHIC | ENTITY_REFERENCE.
-- assumption_status mirrors AssumptionStatus (per-variable provenance tag).

CREATE TABLE IF NOT EXISTS simulation.scenario_inputs (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id        TEXT NOT NULL,
    scenario_id      UUID NOT NULL REFERENCES simulation.scenarios(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    type             TEXT NOT NULL,
    unit             TEXT,
    value            JSONB,                             -- NULL when assumption_status=UNKNOWN
    minimum          NUMERIC,
    maximum          NUMERIC,
    default_value    JSONB,
    source           TEXT,
    assumption_status TEXT NOT NULL,
    constraints       JSONB NOT NULL DEFAULT '[]'::jsonb,-- array of ScenarioConstraint
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sim_inputs_type CHECK (
        type IN ('INTEGER','DECIMAL','PERCENTAGE','CURRENCY','DATE','DURATION',
                 'BOOLEAN','CATEGORICAL','GEOGRAPHIC','ENTITY_REFERENCE')
    ),
    CONSTRAINT chk_sim_inputs_assumption CHECK (
        assumption_status IN ('OBSERVED_INPUT','USER_DEFINED','MODEL_ASSUMPTION',
                              'HISTORICAL_REFERENCE','ESTIMATE','UNKNOWN')
    ),
    CONSTRAINT chk_sim_inputs_range CHECK (
        minimum IS NULL OR maximum IS NULL OR minimum <= maximum
    )
);
CREATE INDEX idx_sim_inputs_tenant ON simulation.scenario_inputs(tenant_id);
CREATE INDEX idx_sim_inputs_scn    ON simulation.scenario_inputs(scenario_id);

-- ===== simulation.simulation_runs ===========================================
-- Audit-trail record for a single execution of a scenario. Phase 18 §22, §23.
-- status follows RunStatus lifecycle: CREATED → VALIDATED → STARTED →
-- INPUTS_LOADED → MODEL_LOADED → SIMULATION_STARTED → SIMULATION_COMPLETED →
-- RESULTS_VALIDATED → PUBLISHED (terminal). FAILED / CANCELLED are terminal
-- failure states.

CREATE TABLE IF NOT EXISTS simulation.simulation_runs (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id           TEXT NOT NULL,
    scenario_id         UUID NOT NULL REFERENCES simulation.scenarios(id) ON DELETE RESTRICT,
    scenario_version    INT NOT NULL,
    model_id            UUID REFERENCES simulation.scenario_models(id) ON DELETE RESTRICT,
    model_version       TEXT,
    engine_version      TEXT,
    status              TEXT NOT NULL DEFAULT 'CREATED',
    input_values        JSONB NOT NULL DEFAULT '{}'::jsonb,
    assumption_versions JSONB NOT NULL DEFAULT '[]'::jsonb,  -- [{assumption_id, version}]
    dataset_versions    JSONB NOT NULL DEFAULT '[]'::jsonb,  -- [{dataset_id, version, hash}]
    random_seed         BIGINT,
    execution_id        TEXT,
    agent_versions      JSONB NOT NULL DEFAULT '[]'::jsonb,
    prompt_versions     JSONB NOT NULL DEFAULT '[]'::jsonb,
    tool_versions       JSONB NOT NULL DEFAULT '[]'::jsonb,
    environment         TEXT,
    workflow_id         TEXT,
    trace_id            TEXT,
    user_id             UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    agent               TEXT,
    system_ver          TEXT,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    duration_ms         BIGINT,
    resource_usage      JSONB,                            -- {cpu_millis, memory_mb, duration_seconds, network_bytes}
    errors              JSONB NOT NULL DEFAULT '[]'::jsonb, -- [{code, message, stage, timestamp}]
    result_summary      JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sim_runs_status CHECK (
        status IN ('CREATED','VALIDATED','STARTED','INPUTS_LOADED','MODEL_LOADED',
                   'SIMULATION_STARTED','SIMULATION_COMPLETED','RESULTS_VALIDATED',
                   'PUBLISHED','FAILED','CANCELLED')
    )
);
CREATE INDEX idx_sim_runs_tenant   ON simulation.simulation_runs(tenant_id);
CREATE INDEX idx_sim_runs_scn      ON simulation.simulation_runs(scenario_id, created_at DESC);
CREATE INDEX idx_sim_runs_status   ON simulation.simulation_runs(tenant_id, status);
CREATE INDEX idx_sim_runs_trace    ON simulation.simulation_runs(trace_id) WHERE trace_id IS NOT NULL;
COMMENT ON COLUMN simulation.simulation_runs.scenario_version IS 'Pinned to the immutable scenario_versions.version row that was executed — reproducibility anchor (Phase 18 §22).';

-- ===== simulation.simulation_results =======================================
-- IMMUTABLE full result payload for a completed run. Phase 18 §11.
--
-- Published results cannot be silently edited (Gate I — canonical truth
-- protection). The immutability trigger (below) blocks UPDATE/DELETE/TRUNCATE.
-- To re-run, insert a new run + new result. reality_layer is always MODELED.

CREATE TABLE IF NOT EXISTS simulation.simulation_results (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id          UUID NOT NULL REFERENCES simulation.simulation_runs(id) ON DELETE RESTRICT,
    tenant_id       TEXT NOT NULL,
    scenario_id     UUID NOT NULL REFERENCES simulation.scenarios(id) ON DELETE RESTRICT,
    reality_layer   TEXT NOT NULL DEFAULT 'MODELED',
    model_id        UUID,
    model_version   TEXT,
    engine_version  TEXT,
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    outputs         JSONB NOT NULL DEFAULT '[]'::jsonb,  -- array of ResultOutput (always MODELED)
    uncertainty     JSONB,                             -- UncertaintySummary
    timeline        JSONB NOT NULL DEFAULT '[]'::jsonb, -- array of TimelineEvent
    impacts         JSONB NOT NULL DEFAULT '[]'::jsonb, -- array of ModeledImpact
    comparison_refs JSONB NOT NULL DEFAULT '[]'::jsonb, -- array of UUIDs (sibling scenarios)
    random_seed     BIGINT,
    input_hash      TEXT,                               -- sha256 of input_values; reproducibility proof
    methodology     JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sim_results_reality CHECK (reality_layer = 'MODELED')
);
CREATE INDEX idx_sim_results_tenant   ON simulation.simulation_results(tenant_id);
CREATE INDEX idx_sim_results_run       ON simulation.simulation_results(run_id);
CREATE INDEX idx_sim_results_scn       ON simulation.simulation_results(scenario_id, generated_at DESC);
COMMENT ON COLUMN simulation.simulation_results.reality_layer IS 'Always MODELED. A result row tagged OBSERVED would be a canonical-truth violation (Gate I).';

-- ===== simulation.simulation_metrics ========================================
-- Denormalised per-metric rows derived from simulation_results.outputs. Phase 18 §11.
-- One row per (result, output_name) — supports fast metric comparison across
-- runs without parsing the JSONB outputs array.

CREATE TABLE IF NOT EXISTS simulation.simulation_metrics (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    result_id     UUID NOT NULL REFERENCES simulation.simulation_results(id) ON DELETE CASCADE,
    tenant_id     TEXT NOT NULL,
    scenario_id   UUID NOT NULL,
    name          TEXT NOT NULL,
    type          TEXT NOT NULL,                  -- VariableType
    unit          TEXT,
    value         JSONB NOT NULL,
    uncertainty   JSONB,
    reality_layer TEXT NOT NULL DEFAULT 'MODELED',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sim_metrics_reality CHECK (reality_layer = 'MODELED'),
    CONSTRAINT chk_sim_metrics_type CHECK (
        type IN ('INTEGER','DECIMAL','PERCENTAGE','CURRENCY','DATE','DURATION',
                 'BOOLEAN','CATEGORICAL','GEOGRAPHIC','ENTITY_REFERENCE')
    )
);
CREATE INDEX idx_sim_metrics_tenant   ON simulation.simulation_metrics(tenant_id);
CREATE INDEX idx_sim_metrics_result   ON simulation.simulation_metrics(result_id);
CREATE INDEX idx_sim_metrics_scn_name ON simulation.simulation_metrics(scenario_id, name);

-- ===== simulation.scenario_evidence =========================================
-- Evidence references linking a scenario back to observed civic facts. Phase 18 §8.
-- Each row is an EvidenceRef {kind, id, source_url, retrieved_at, verification_status}.

CREATE TABLE IF NOT EXISTS simulation.scenario_evidence (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id          TEXT NOT NULL,
    scenario_id        UUID NOT NULL REFERENCES simulation.scenarios(id) ON DELETE CASCADE,
    kind               TEXT NOT NULL,                  -- DOCUMENT | FACT | RELATIONSHIP | EXTERNAL
    evidence_id        TEXT NOT NULL,                  -- canonical ID of referenced evidence (opaque; not a hard FK)
    source_url         TEXT NOT NULL,
    retrieved_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verification_status TEXT NOT NULL DEFAULT 'UNKNOWN',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sim_evidence_kind CHECK (
        kind IN ('DOCUMENT','FACT','RELATIONSHIP','EXTERNAL')
    )
);
CREATE INDEX idx_sim_evidence_tenant ON simulation.scenario_evidence(tenant_id);
CREATE INDEX idx_sim_evidence_scn    ON simulation.scenario_evidence(scenario_id);
CREATE INDEX idx_sim_evidence_ev_id  ON simulation.scenario_evidence(evidence_id);

-- ===== simulation.scenario_relationships ===================================
-- Relates scenarios to each other (comparison, derivation, supersedence,
-- counterfactual pairings). Phase 18 §3 (Comparative) + §15 (Impacts).

CREATE TABLE IF NOT EXISTS simulation.scenario_relationships (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id            TEXT NOT NULL,
    source_scenario_id   UUID NOT NULL REFERENCES simulation.scenarios(id) ON DELETE CASCADE,
    target_scenario_id   UUID NOT NULL REFERENCES simulation.scenarios(id) ON DELETE CASCADE,
    relationship_type    TEXT NOT NULL,  -- COMPARISON | DERIVED_FROM | COUNTERFACTUAL_OF | SUPERSEDES | REPLICATES | VARIANT_OF
    notes                TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sim_rels_type CHECK (
        relationship_type IN ('COMPARISON','DERIVED_FROM','COUNTERFACTUAL_OF',
                              'SUPERSEDES','REPLICATES','VARIANT_OF')
    ),
    CONSTRAINT chk_sim_rels_different CHECK (source_scenario_id <> target_scenario_id)
);
CREATE INDEX idx_sim_rels_tenant   ON simulation.scenario_relationships(tenant_id);
CREATE INDEX idx_sim_rels_source   ON simulation.scenario_relationships(source_scenario_id);
CREATE INDEX idx_sim_rels_target   ON simulation.scenario_relationships(target_scenario_id);
CREATE INDEX idx_sim_rels_type     ON simulation.scenario_relationships(tenant_id, relationship_type);

-- ===== simulation.scenario_audits ===========================================
-- Append-only audit log scoped to the simulation bounded context. Mirrors the
-- shape of audit.log_entries but is owned by this schema so the simulation
-- layer can be rebuilt without losing its own history. The immutability
-- trigger (below) blocks UPDATE / DELETE / TRUNCATE.

CREATE TABLE IF NOT EXISTS simulation.scenario_audits (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       TEXT NOT NULL,
    event_type      TEXT NOT NULL,                       -- scenario.create | scenario.transition | run.start | run.complete | result.publish ...
    entity_type     TEXT NOT NULL,
    entity_id       UUID,
    actor_user_id   UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    before          JSONB,
    after           JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sim_audits_tenant  ON simulation.scenario_audits(tenant_id);
CREATE INDEX idx_sim_audits_entity  ON simulation.scenario_audits(tenant_id, entity_type, entity_id);
CREATE INDEX idx_sim_audits_event   ON simulation.scenario_audits(tenant_id, event_type, occurred_at DESC);
CREATE INDEX idx_sim_audits_time    ON simulation.scenario_audits(occurred_at DESC);

-- ===== Immutability triggers ================================================
-- Block UPDATE / DELETE / TRUNCATE on immutable tables. Pattern follows
-- 018_trust_schema.up.sql (trust.audit_events append-only trigger).
--
-- Tables protected:
--   * simulation.scenario_versions  — historical scenario snapshots (ADR-0011)
--   * simulation.simulation_results — published results (Gate I)
--   * simulation.scenario_audits    — append-only audit log

CREATE OR REPLACE FUNCTION simulation.tg_block_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION '%.% is immutable; UPDATE / DELETE / TRUNCATE is forbidden (ADR-0011 / Gate I)',
        TG_TABLE_SCHEMA, TG_TABLE_NAME;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg_scenario_versions_immutable
    BEFORE UPDATE OR DELETE OR TRUNCATE ON simulation.scenario_versions
    FOR EACH STATEMENT EXECUTE FUNCTION simulation.tg_block_mutation();

CREATE TRIGGER tg_simulation_results_immutable
    BEFORE UPDATE OR DELETE OR TRUNCATE ON simulation.simulation_results
    FOR EACH STATEMENT EXECUTE FUNCTION simulation.tg_block_mutation();

CREATE TRIGGER tg_scenario_audits_immutable
    BEFORE UPDATE OR DELETE OR TRUNCATE ON simulation.scenario_audits
    FOR EACH STATEMENT EXECUTE FUNCTION simulation.tg_block_mutation();

-- ===== updated_at maintenance trigger ======================================
-- Auto-touch updated_at on scenarios + scenario_models.

CREATE OR REPLACE FUNCTION simulation.tg_touch_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg_scenarios_touch_updated
    BEFORE UPDATE ON simulation.scenarios
    FOR EACH ROW EXECUTE FUNCTION simulation.tg_touch_updated_at();

CREATE TRIGGER tg_scenario_models_touch_updated
    BEFORE UPDATE ON simulation.scenario_models
    FOR EACH ROW EXECUTE FUNCTION simulation.tg_touch_updated_at();

COMMIT;
