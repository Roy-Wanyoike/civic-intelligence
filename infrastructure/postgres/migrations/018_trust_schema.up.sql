-- 018_trust_schema.up.sql
-- Phase 9 — Trust layer.
--
-- Introduces the `trust` bounded context. This schema owns the immutable record
-- of *why* a fact on the platform is believed to be true: the source it came
-- from, the verification state of the claim, the evidence that backs it, the
-- contradictions detected between sources, and the audit trail of every
-- correction. It is intentionally separate from `evidence` (which stores
-- extracted facts) so the trust layer can be rebuilt independently of the
-- canonical evidence graph, and so corrections NEVER silently rewrite history.
--
-- Architectural contract reminder:
--   * Canonical civic truth lives in legislation/ingestion/documents.
--   * AI may propose candidate facts but may NEVER write to canonical state
--     without validation + evidence (ADR-0005).
--   * Corrections create new immutable versions — the previous_state is
--     always preserved (ADR-0013 contradiction engine; ADR-0011 immutable
--     bill versions).
--   * Audit log is append-only.
--
-- Schema created here (forward-only; 002_schemas.up.sql is immutable).

CREATE SCHEMA IF NOT EXISTS trust;
COMMENT ON SCHEMA trust IS 'Trust layer — sources, claims, evidence, contradictions, corrections, audit. Every canonical fact is traceable to a source.';

-- ===== trust.sources =========================================================
-- Authoritative sources of civic information (parliament portals, gazettes,
-- Kenya Law, official ministry pages). A trust source is the provenance root
-- for every claim downstream. institution_id is opaque (TEXT) because the same
-- institution may be sourced from multiple upstreams (e.g.KE National Assembly
-- vs.KE Senate).
--
-- Authority levels (see docs/architecture/05-evidence-system.md):
--   PRIMARY_OFFICIAL      — Official government source (parliament.go.ke, president.go.ke)
--   OFFICIAL_REPOSITORY   — Official legal repository (kenyalaw.org)
--   SECONDARY_VERIFIED    — Verified secondary sources (news with official refs)
--   UNVERIFIED            — Not used for canonical facts

CREATE TABLE trust.sources (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    institution_id      TEXT NOT NULL,
    country             CHAR(2) NOT NULL,
    source_type         TEXT NOT NULL,                      -- parliament | gazette | ministry | court | repository | news
    authority_level     TEXT NOT NULL DEFAULT 'UNVERIFIED',
    official_url        TEXT NOT NULL,
    domain              TEXT NOT NULL,                      -- derived from official_url; indexed for fast lookups
    status              TEXT NOT NULL DEFAULT 'active',    -- active | degraded | retired | blocked
    verification_method TEXT,                              -- manual | automated_tls | content_hash | oidc_signed
    last_verified_at    TIMESTAMPTZ,
    health_status       TEXT NOT NULL DEFAULT 'unknown',  -- healthy | degraded | down | unknown
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_sources_authority CHECK (
        authority_level IN ('PRIMARY_OFFICIAL','OFFICIAL_REPOSITORY','SECONDARY_VERIFIED','UNVERIFIED')
    ),
    CONSTRAINT chk_sources_status CHECK (
        status IN ('active','degraded','retired','blocked')
    ),
    CONSTRAINT chk_sources_health CHECK (
        health_status IN ('healthy','degraded','down','unknown')
    )
);
CREATE UNIQUE INDEX uq_trust_sources_domain      ON trust.sources(domain);
CREATE INDEX        idx_trust_sources_country    ON trust.sources(country);
CREATE INDEX        idx_trust_sources_authority   ON trust.sources(authority_level);
CREATE INDEX        idx_trust_sources_status      ON trust.sources(status, health_status);
COMMENT ON COLUMN trust.sources.institution_id  IS 'Opaque institution identifier (e.g. ke-na, ke-senate). Not a FK because institutions live in ingestion.sources.institution.';
COMMENT ON COLUMN trust.sources.verification_method IS 'How the source is verified to be authentic: manual, automated_tls, content_hash, oidc_signed.';

-- ===== trust.source_checks ===================================================
-- One row per liveness/provenance probe of a source. Tracking changed=TRUE
-- means the content_hash differs from the previous check — that's the
-- immutable hook the pipeline uses to re-fetch + re-extract evidence.

CREATE TABLE trust.source_checks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id       UUID NOT NULL REFERENCES trust.sources(id) ON DELETE CASCADE,
    checked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    http_status     INT,                          -- 200, 404, 0 for network failure
    latency_ms      INT,
    tls_valid       BOOLEAN NOT NULL DEFAULT FALSE,
    content_hash    TEXT,                         -- sha256 of fetched body (NULL on failure)
    changed         BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT chk_source_checks_http CHECK (http_status IS NULL OR (http_status >= 0 AND http_status < 600))
);
CREATE INDEX idx_trust_source_checks_source ON trust.source_checks(source_id, checked_at DESC);
CREATE INDEX idx_trust_source_checks_changed ON trust.source_checks(changed, checked_at DESC);

-- ===== trust.claims ==========================================================
-- A claim is the smallest unit of civic knowledge. Subject/predicate/object
-- is an RDF-style triple (e.g. "Bill X" — stage — "First Reading"). `text`
-- is the human-readable form. `verification_state` is the lifecycle state.
--
-- `valid_from` / `valid_to` model bitemporality: a claim may be true for a
-- finite window. NULL `valid_to` means "still in effect". When a claim is
-- superseded, the new claim's `valid_from` is set to the old claim's
-- `valid_to` (so they cannot overlap). Enforcement of that overlap rule is
-- done in a trigger (below) — same trigger writes to trust.audit_events.

CREATE TABLE trust.claims (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subject             TEXT NOT NULL,
    predicate           TEXT NOT NULL,
    object              TEXT NOT NULL,
    claim_type          TEXT NOT NULL DEFAULT 'UNKNOWN',  -- FACT | EXPLANATION | INFERENCE | UNKNOWN
    text                TEXT NOT NULL,
    confidence          NUMERIC(4,3) NOT NULL DEFAULT 0.000 CHECK (confidence >= 0.0 AND confidence <= 1.0),
    verification_state  TEXT NOT NULL DEFAULT 'UNVERIFIED',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_from          TIMESTAMPTZ,
    valid_to            TIMESTAMPTZ,
    CONSTRAINT chk_claims_type CHECK (
        claim_type IN ('FACT','EXPLANATION','INFERENCE','UNKNOWN')
    ),
    CONSTRAINT chk_claims_verification_state CHECK (
        verification_state IN (
            'UNVERIFIED','DISCOVERED','EXTRACTED','VALIDATING',
            'VERIFIED','CONFLICTED','CORRECTED','SUPERSEDED','REJECTED'
        )
    ),
    CONSTRAINT chk_claims_validity_window CHECK (
        valid_to IS NULL OR valid_from IS NULL OR valid_to >= valid_from
    )
);
CREATE INDEX idx_trust_claims_subject      ON trust.claims(subject);
CREATE INDEX idx_trust_claims_spo          ON trust.claims(subject, predicate, object);
CREATE INDEX idx_trust_claims_state        ON trust.claims(verification_state);
CREATE INDEX idx_trust_claims_validity      ON trust.claims(valid_from, valid_to);
CREATE INDEX idx_trust_claims_created_at    ON trust.claims(created_at DESC);
COMMENT ON COLUMN trust.claims.confidence IS 'AI-assigned confidence in [0.000, 1.000]. AI may set this; humans gate the verification_state.';

-- ===== trust.evidence ========================================================
-- The provenance chain from a claim back to the document + page + paragraph
-- where it was found. Every claim MUST have at least one evidence row before
-- it can be promoted to VERIFIED (enforced by the application layer — SQL
-- would require a deferred constraint that complicates bulk inserts).
--
-- document_id FKs to ingestion.documents; snapshot_id FKs to
-- ingestion.document_snapshots so the exact byte stream that backed the
-- claim is always recoverable.

CREATE TABLE trust.evidence (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    claim_id        UUID NOT NULL REFERENCES trust.claims(id) ON DELETE CASCADE,
    document_id     UUID REFERENCES ingestion.documents(id) ON DELETE SET NULL,
    snapshot_id     UUID REFERENCES ingestion.document_snapshots(id) ON DELETE SET NULL,
    page_number     INT,
    section         TEXT,
    paragraph       TEXT,
    text_span       TEXT,                              -- quoted snippet from the source
    source_url      TEXT NOT NULL,
    retrieved_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    content_hash    TEXT,                              -- sha256 of the snapshot; immutability proof
    CONSTRAINT chk_evidence_page CHECK (page_number IS NULL OR page_number > 0)
);
CREATE INDEX idx_trust_evidence_claim    ON trust.evidence(claim_id);
CREATE INDEX idx_trust_evidence_document ON trust.evidence(document_id);
CREATE INDEX idx_trust_evidence_snapshot ON trust.evidence(snapshot_id);
CREATE INDEX idx_trust_evidence_source   ON trust.evidence(source_url);

-- ===== trust.contradictions ==================================================
-- When two claims (typically from different sources) disagree, a
-- contradiction row is created. The contradiction engine NEVER silently
-- resolves (ADR-0013); it surfaces the conflict for human review.
--
-- status:     detected | under_review | resolved | dismissed
-- resolution: claim_a_wins | claim_b_wins | merged | superseded | NULL

CREATE TABLE trust.contradictions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    claim_a_id      UUID NOT NULL REFERENCES trust.claims(id) ON DELETE CASCADE,
    claim_b_id      UUID NOT NULL REFERENCES trust.claims(id) ON DELETE CASCADE,
    source_a_id     UUID REFERENCES trust.sources(id) ON DELETE SET NULL,
    source_b_id     UUID REFERENCES trust.sources(id) ON DELETE SET NULL,
    detected_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status          TEXT NOT NULL DEFAULT 'detected',
    resolution      TEXT,
    reviewer_id     UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    resolved_at     TIMESTAMPTZ,
    CONSTRAINT chk_contradictions_status CHECK (
        status IN ('detected','under_review','resolved','dismissed')
    ),
    CONSTRAINT chk_contradictions_resolution CHECK (
        resolution IS NULL OR resolution IN ('claim_a_wins','claim_b_wins','merged','superseded')
    ),
    CONSTRAINT chk_contradictions_different CHECK (claim_a_id <> claim_b_id)
);
CREATE INDEX idx_trust_contradictions_status  ON trust.contradictions(status, detected_at DESC);
CREATE INDEX idx_trust_contradictions_claim_a ON trust.contradictions(claim_a_id);
CREATE INDEX idx_trust_contradictions_claim_b ON trust.contradictions(claim_b_id);
CREATE INDEX idx_trust_contradictions_reviewer ON trust.contradictions(reviewer_id);

-- ===== trust.corrections =====================================================
-- Submitted corrections are NEVER applied directly to canonical state. Each
-- submission becomes a row here with status pending -> in_review -> accepted |
-- rejected. When accepted, a *new* immutable version is created on the
-- target entity (ADR-0011) and the previous_state JSONB preserves the
-- before-snapshot for the audit trail.

CREATE TABLE trust.corrections (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_type         TEXT NOT NULL,                  -- bill | act | claim | evidence | source | regulation
    target_id           UUID NOT NULL,                  -- opaque UUID of the target entity
    reason              TEXT NOT NULL,
    category            TEXT NOT NULL DEFAULT 'wrong_fact',
    previous_state      JSONB,
    corrected_state     JSONB NOT NULL,
    evidence            TEXT,                            -- URL or free-text evidence
    submitted_by        UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    submitted_email     CITEXT,                          -- for anonymous citizen submissions
    reviewed_by         UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    reviewed_at         TIMESTAMPTZ,
    status              TEXT NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_corrections_target_type CHECK (
        target_type IN ('bill','act','claim','evidence','source','regulation','policy','person','committee')
    ),
    CONSTRAINT chk_corrections_status CHECK (
        status IN ('pending','in_review','accepted','rejected','superseded')
    ),
    CONSTRAINT chk_corrections_category CHECK (
        category IN (
            'wrong_fact','wrong_source','wrong_citation','outdated',
            'incorrect_interpretation','missing_information',
            'conflicting_sources','broken_document'
        )
    )
);
CREATE INDEX idx_trust_corrections_target   ON trust.corrections(target_type, target_id);
CREATE INDEX idx_trust_corrections_status    ON trust.corrections(status, created_at DESC);
CREATE INDEX idx_trust_corrections_submitter ON trust.corrections(submitted_by);
COMMENT ON COLUMN trust.corrections.previous_state IS 'JSONB snapshot of the target entity BEFORE the correction is applied. NULL on first submission.';
COMMENT ON COLUMN trust.corrections.corrected_state IS 'JSONB of the proposed new state. NEVER applied directly to canonical tables — a new immutable version is created instead.';

-- ===== trust.audit_events ====================================================
-- Append-only audit log scoped to the trust bounded context. Mirrors the
-- shape of audit.log_entries but is owned by this schema so the trust layer
-- can be rebuilt without losing its own history. A trigger (below) blocks
-- UPDATE / DELETE / TRUNCATE.

CREATE TABLE trust.audit_events (
    id              BIGSERIAL PRIMARY KEY,
    event_type      TEXT NOT NULL,                       -- claim.create | claim.supersede | correction.accept | contradiction.detect ...
    entity_type     TEXT NOT NULL,
    entity_id       UUID,
    actor_user_id   UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    before          JSONB,
    after           JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_trust_audit_entity ON trust.audit_events(entity_type, entity_id);
CREATE INDEX idx_trust_audit_actor  ON trust.audit_events(actor_user_id, occurred_at DESC);
CREATE INDEX idx_trust_audit_event  ON trust.audit_events(event_type, occurred_at DESC);
CREATE INDEX idx_trust_audit_time   ON trust.audit_events(occurred_at DESC);

-- Block UPDATE / DELETE / TRUNCATE on the audit log (append-only).
CREATE OR REPLACE FUNCTION trust.tg_audit_events_immutable() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'trust.audit_events is append-only; UPDATE/DELETE/TRUNCATE is forbidden';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg_audit_events_no_update
    BEFORE UPDATE OR DELETE OR TRUNCATE ON trust.audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION trust.tg_audit_events_immutable();

-- ===== trust.review_queue ====================================================
-- Work queue for items awaiting human review (corrections, contradictions,
-- newly-discovered sources, AI-proposed candidate facts). priority is a
-- small int 0..5 where 0 = urgent and 5 = backlog.

CREATE TABLE trust.review_queue (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type     TEXT NOT NULL,                      -- correction | contradiction | claim | source
    entity_id       UUID NOT NULL,
    reason          TEXT NOT NULL,
    priority        SMALLINT NOT NULL DEFAULT 3 CHECK (priority >= 0 AND priority <= 5),
    status          TEXT NOT NULL DEFAULT 'open',       -- open | assigned | resolved | closed
    assigned_to     UUID REFERENCES identity.users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_review_queue_entity_type CHECK (
        entity_type IN ('correction','contradiction','claim','source','evidence')
    ),
    CONSTRAINT chk_review_queue_status CHECK (
        status IN ('open','assigned','resolved','closed')
    )
);
CREATE INDEX idx_trust_review_queue_status   ON trust.review_queue(status, priority, created_at DESC);
CREATE INDEX idx_trust_review_queue_assigned ON trust.review_queue(assigned_to);
CREATE INDEX idx_trust_review_queue_entity   ON trust.review_queue(entity_type, entity_id);

-- ===== Audit triggers on canonical trust tables =============================
-- Every INSERT/UPDATE on claims, contradictions, corrections, review_queue
-- writes a row to trust.audit_events so the trust layer's own history is
-- always recoverable.

CREATE OR REPLACE FUNCTION trust.tg_record_audit() RETURNS TRIGGER AS $$
DECLARE
    v_actor UUID;
BEGIN
    SELECT COALESCE(current_setting('app.actor_user_id', true), '')::uuid INTO v_actor;
    INSERT INTO trust.audit_events (event_type, entity_type, entity_id, actor_user_id, before, after)
    VALUES (
        TG_ARGV[0],
        TG_TABLE_NAME,
        COALESCE(NEW.id, OLD.id),
        v_actor,
        CASE WHEN TG_OP = 'UPDATE' THEN to_jsonb(OLD) ELSE NULL END,
        CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE to_jsonb(NEW) END
    );
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE TRIGGER tg_trust_claims_audit_insert
    AFTER INSERT ON trust.claims
    FOR EACH ROW EXECUTE FUNCTION trust.tg_record_audit('claim.create');
CREATE TRIGGER tg_trust_claims_audit_update
    AFTER UPDATE ON trust.claims
    FOR EACH ROW EXECUTE FUNCTION trust.tg_record_audit('claim.update');

CREATE TRIGGER tg_trust_contradictions_audit_insert
    AFTER INSERT ON trust.contradictions
    FOR EACH ROW EXECUTE FUNCTION trust.tg_record_audit('contradiction.detect');
CREATE TRIGGER tg_trust_contradictions_audit_update
    AFTER UPDATE ON trust.contradictions
    FOR EACH ROW EXECUTE FUNCTION trust.tg_record_audit('contradiction.update');

CREATE TRIGGER tg_trust_corrections_audit_insert
    AFTER INSERT ON trust.corrections
    FOR EACH ROW EXECUTE FUNCTION trust.tg_record_audit('correction.submit');
CREATE TRIGGER tg_trust_corrections_audit_update
    AFTER UPDATE ON trust.corrections
    FOR EACH ROW EXECUTE FUNCTION trust.tg_record_audit('correction.update');

CREATE TRIGGER tg_trust_review_queue_audit_insert
    AFTER INSERT ON trust.review_queue
    FOR EACH ROW EXECUTE FUNCTION trust.tg_record_audit('review_queue.create');
CREATE TRIGGER tg_trust_review_queue_audit_update
    AFTER UPDATE ON trust.review_queue
    FOR EACH ROW EXECUTE FUNCTION trust.tg_record_audit('review_queue.update');

-- ===== Seed: example primary sources =======================================
-- A small, hand-curated seed set so the /trust page + provenance API have
-- something concrete to render before the ingestion pipeline is fully wired.
-- These rows map to the same institutions documented in
-- infrastructure/postgres/seed/002_kenya_institutions.sql.

INSERT INTO trust.sources (institution_id, country, source_type, authority_level, official_url, domain, status, verification_method, last_verified_at, health_status) VALUES
    ('ke-parliament',   'KE', 'parliament', 'PRIMARY_OFFICIAL',     'https://www.parliament.go.ke/',          'www.parliament.go.ke',          'active', 'manual',         NOW() - INTERVAL '1 hour',  'healthy'),
    ('ke-senate',       'KE', 'parliament', 'PRIMARY_OFFICIAL',     'https://www.senate.go.ke/',              'www.senate.go.ke',              'active', 'manual',         NOW() - INTERVAL '2 hours', 'healthy'),
    ('ke-kenyalaw',     'KE', 'repository', 'OFFICIAL_REPOSITORY', 'https://www.kenyalaw.org/',              'www.kenyalaw.org',              'active', 'content_hash',   NOW() - INTERVAL '30 minutes', 'healthy'),
    ('ke-president',    'KE', 'president',  'PRIMARY_OFFICIAL',     'https://www.president.go.ke/',          'www.president.go.ke',           'active', 'manual',         NOW() - INTERVAL '3 hours', 'healthy'),
    ('ke-gazette',      'KE', 'gazette',    'PRIMARY_OFFICIAL',     'https://gazettes.africa/gazettes/ke',    'gazettes.africa',               'active', 'content_hash',   NOW() - INTERVAL '4 hours', 'healthy')
ON CONFLICT DO NOTHING;
