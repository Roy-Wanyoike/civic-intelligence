-- 021_post_assent_schema.up.sql
-- Phase: Post-Assent Legislative Lifecycle spec.
--
-- Extends the Bill lifecycle BEYOND Presidential Assent. The full lifecycle:
--
--   Bill → Introduction → First Reading → Committee → Public Participation →
--   Second Reading → Committee of Whole House → Third Reading →
--   Other House → Concurrence → Presidential Assent → ACT →
--   Commencement → Regulations → Administrative Actions →
--   Judicial Challenges → Judicial Decisions → Amendments → Repeal/Replacement
--
-- Tables added (all under the existing `legislation` schema):
--   * legislation.presidential_assent_events
--   * legislation.act_versions         (IMMUTABLE — ADR-0011)
--   * legislation.post_assent_events
--   * legislation.legislative_lifecycle_audits
--
-- Architectural contract reminders:
--   * Presidential assent is NOT the end of the lifecycle. Every post-assent
--     event is tracked with evidence.
--   * Act versions are IMMUTABLE. To "amend" an Act, a new version row is
--     inserted with effective_from = amendment date. Historical text is
--     preserved verbatim (ADR-0011 immutable bill/act versions).
--   * "No authoritative record found" is NOT the same as "nothing happened".
--     legislative_lifecycle_audits distinguishes NOT_VERIFIED from negative
--     findings via the audit_statuses + data_gaps arrays.
--
-- The `legislation` schema already exists (002_schemas.up.sql). The
-- `legislation.acts` table already exists (010_acts.up.sql) and is referenced
-- by act_versions via FK.

BEGIN;

-- ===== legislation.presidential_assent_events ==============================
-- A first-class event representing the president's action on a Bill. Spec §14.
--
-- This replaces the anti-pattern of mutating `bill.status = PASSED` without
-- preserving the historical event. assent_status is ASSENTED | RETURNED |
-- WITHHELD | UNKNOWN.
--
-- president_id / administration_id / presidential_term_id are opaque UUIDs
-- (soft FKs) referencing the government schema. We deliberately do NOT add
-- hard FK constraints here to keep the post-assent schema self-contained and
-- to avoid coupling legislation migrations to government migrations.

CREATE TABLE IF NOT EXISTS legislation.presidential_assent_events (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_id             UUID NOT NULL REFERENCES legislation.bills(id) ON DELETE RESTRICT,
    president_id        UUID,                          -- soft FK → government.presidents.id
    administration_id   UUID,                          -- soft FK → government.administrations.id
    presidential_term_id UUID,                          -- soft FK → government.presidential_terms.id
    event_date          TIMESTAMPTZ NOT NULL,
    official_source     TEXT NOT NULL,
    document_id         UUID REFERENCES ingestion.documents(id) ON DELETE SET NULL,
    evidence            JSONB NOT NULL DEFAULT '[]'::jsonb,  -- array of EvidenceRef
    assent_status       TEXT NOT NULL DEFAULT 'UNKNOWN',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_assent_event_status CHECK (
        assent_status IN ('ASSENTED','RETURNED','WITHHELD','UNKNOWN')
    )
);
CREATE INDEX idx_pa_events_bill   ON legislation.presidential_assent_events(bill_id, event_date DESC);
CREATE INDEX idx_pa_events_pres   ON legislation.presidential_assent_events(president_id)   WHERE president_id        IS NOT NULL;
CREATE INDEX idx_pa_events_admin  ON legislation.presidential_assent_events(administration_id) WHERE administration_id IS NOT NULL;
CREATE INDEX idx_pa_events_status ON legislation.presidential_assent_events(assent_status);

-- ===== legislation.act_versions =============================================
-- IMMUTABLE snapshot of an Act at a point in time. Spec §20.
--
-- Versions are append-only — never overwrite historical truth (ADR-0011).
-- To "amend" an Act, a new version row is inserted with effective_from set
-- and the previous version's effective_until closed out. The immutability
-- trigger (below) blocks UPDATE / DELETE / TRUNCATE.
--
-- FK: act_versions.act_id → legislation.acts(id) ON DELETE RESTRICT (the acts
-- table is owned by 010_acts.up.sql; we only add the version table here).

CREATE TABLE IF NOT EXISTS legislation.act_versions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    act_id          UUID NOT NULL REFERENCES legislation.acts(id) ON DELETE RESTRICT,
    version         INT NOT NULL CHECK (version > 0),
    text            TEXT NOT NULL,                     -- verbatim legal text of the act at this version
    amendment_id    UUID,                              -- soft FK to a legislation.amendments(id) row if this version is the result of an amendment; NULL for the original
    effective_from  TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,                        -- NULL = still in effect
    source_url      TEXT,
    content_hash    TEXT,                              -- sha256 of canonical text; immutability / dedup proof
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (act_id, version),
    CONSTRAINT chk_act_versions_window CHECK (
        effective_until IS NULL OR effective_until >= effective_from
    )
);
CREATE INDEX idx_act_versions_act   ON legislation.act_versions(act_id, version DESC);
CREATE INDEX idx_act_versions_eff   ON legislation.act_versions(effective_from, effective_until);
CREATE UNIQUE INDEX uq_act_versions_current ON legislation.act_versions(act_id) WHERE effective_until IS NULL;
COMMENT ON COLUMN legislation.act_versions.text IS 'Verbatim legal text. NEVER edited in place — to amend an Act, insert a new version with effective_from = amendment date.';
COMMENT ON COLUMN legislation.act_versions.content_hash IS 'sha256 of the canonical text. Two rows with the same content_hash are duplicates; the immutability trigger guarantees the hash never changes after insert.';

-- ===== legislation.post_assent_events =======================================
-- A single event in the post-assent lifecycle of an Act. Spec §15.
-- event_type follows PostAssentEventType: ASSENT | PUBLICATION | COMMENCEMENT |
-- REGULATION | IMPLEMENTATION | COURT_CHALLENGE | JUDICIAL_DECISION |
-- AMENDMENT | REPEAL.

CREATE TABLE IF NOT EXISTS legislation.post_assent_events (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    act_id       UUID NOT NULL REFERENCES legislation.acts(id) ON DELETE CASCADE,
    bill_id      UUID REFERENCES legislation.bills(id) ON DELETE SET NULL,   -- set when the event ties back to a specific bill (e.g. ASSENT)
    event_type   TEXT NOT NULL,
    event_date   TIMESTAMPTZ NOT NULL,
    title        TEXT NOT NULL,
    description  TEXT,
    source_url   TEXT NOT NULL,
    document_id  UUID REFERENCES ingestion.documents(id) ON DELETE SET NULL,
    evidence     JSONB NOT NULL DEFAULT '[]'::jsonb,  -- array of EvidenceRef
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_post_assent_event_type CHECK (
        event_type IN ('ASSENT','PUBLICATION','COMMENCEMENT','REGULATION',
                       'IMPLEMENTATION','COURT_CHALLENGE','JUDICIAL_DECISION',
                       'AMENDMENT','REPEAL')
    )
);
CREATE INDEX idx_post_assent_events_act   ON legislation.post_assent_events(act_id, event_date DESC);
CREATE INDEX idx_post_assent_events_type   ON legislation.post_assent_events(event_type, event_date DESC);
CREATE INDEX idx_post_assent_events_bill   ON legislation.post_assent_events(bill_id) WHERE bill_id IS NOT NULL;

-- ===== legislation.legislative_lifecycle_audits =============================
-- Aggregated audit status of an Act across every post-assent dimension.
-- Spec §16. One row per Act (PK = act_id).
--
-- IMPORTANT: "No authoritative record found" is NOT the same as "nothing
-- happened". audit_statuses may include NOT_VERIFIED for dimensions where no
-- authoritative record was found; data_gaps lists the specific gaps.
--
-- current_status follows ActStatus: ASSENTED | PUBLISHED | COMMENCED |
-- AMENDED | REPEALED | UNKNOWN.

CREATE TABLE IF NOT EXISTS legislation.legislative_lifecycle_audits (
    act_id                  UUID PRIMARY KEY REFERENCES legislation.acts(id) ON DELETE CASCADE,
    assent_recorded         BOOLEAN NOT NULL DEFAULT FALSE,
    act_published           BOOLEAN NOT NULL DEFAULT FALSE,
    act_number              TEXT,
    commencement_notice     BOOLEAN NOT NULL DEFAULT FALSE,
    regulations_required    BOOLEAN NOT NULL DEFAULT FALSE,
    regulations_issued      BOOLEAN NOT NULL DEFAULT FALSE,
    institutional_action    BOOLEAN NOT NULL DEFAULT FALSE,
    implementation_docs     BOOLEAN NOT NULL DEFAULT FALSE,
    amended                 BOOLEAN NOT NULL DEFAULT FALSE,
    court_challenged        BOOLEAN NOT NULL DEFAULT FALSE,
    judicial_decision       BOOLEAN NOT NULL DEFAULT FALSE,
    repealed                 BOOLEAN NOT NULL DEFAULT FALSE,
    current_status          TEXT NOT NULL DEFAULT 'UNKNOWN',
    audit_statuses          JSONB NOT NULL DEFAULT '[]'::jsonb,  -- array of ActAuditStatus
    data_gaps               JSONB NOT NULL DEFAULT '[]'::jsonb,  -- array of TEXT
    last_reviewed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_lifecycle_audit_status CHECK (
        current_status IN ('ASSENTED','PUBLISHED','COMMENCED','AMENDED',
                           'REPEALED','UNKNOWN')
    )
);
CREATE INDEX idx_lifecycle_audit_status   ON legislation.legislative_lifecycle_audits(current_status);
CREATE INDEX idx_lifecycle_audit_reviewed ON legislation.legislative_lifecycle_audits(last_reviewed_at DESC);

-- ===== Immutability trigger on legislation.act_versions =====================
-- Block UPDATE / DELETE / TRUNCATE on act_versions so historical act text
-- can never be silently rewritten (ADR-0011). Pattern follows
-- 018_trust_schema.up.sql (trust.audit_events append-only trigger) and
-- 019_simulation_schema.up.sql (scenario_versions / simulation_results).

CREATE OR REPLACE FUNCTION legislation.tg_block_act_version_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'legislation.act_versions is immutable; UPDATE / DELETE / TRUNCATE is forbidden (ADR-0011). To amend an Act, insert a new version row.';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg_act_versions_immutable
    BEFORE UPDATE OR DELETE OR TRUNCATE ON legislation.act_versions
    FOR EACH STATEMENT EXECUTE FUNCTION legislation.tg_block_act_version_mutation();

COMMIT;
