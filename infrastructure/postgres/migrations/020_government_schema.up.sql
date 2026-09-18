-- 020_government_schema.up.sql
-- Phase: Constitution + Government spec (sections 1, 6, 7, 8, 9, 28, 30, 31).
--
-- Introduces the `government` bounded context. Owns:
--   * The Constitution as a first-class civic domain (not just another document).
--   * Presidents, administrations, presidential terms, government periods.
--   * Cabinet members and government transitions.
--
-- Architectural contract reminders (spec §30, §31):
--   1. No head of state is hard-coded in application logic. Presidents,
--      administrations, and terms are database entities.
--   2. The schema supports any country: Presidential, Parliamentary,
--      Semi-Presidential, Monarchical, Transitional.
--   3. Historical truth is NEVER overwritten. Government transitions are
--      first-class events (§28). Temporal validity (valid_from / valid_until)
--      is enforced with EXCLUDE GiST constraints to prevent overlapping
--      administrations / terms / periods for the same entity.
--
-- Temporal validity pattern:
--   * `valid_from` / `valid_until` are TIMESTAMPTZ (NOT the domain's
--     start_date / end_date, which describe the historical window; valid_*
--     describes the platform's knowledge window).
--   * EXCLUDE USING gist (... WITH =, valid_period WITH &&) prevents two
--     rows for the same entity (president / administration / period) from
--     having overlapping validity windows. Requires the btree_gist extension
--     to combine a scalar column (=) with a range column (&&).
--
-- Schema created here (forward-only; 002_schemas.up.sql is immutable).

BEGIN;

CREATE SCHEMA IF NOT EXISTS government;
COMMENT ON SCHEMA government IS 'Constitution + executive domain — constitutions, presidents, administrations, terms, government periods, cabinet, transitions. Multi-country.';

-- btree_gist is required for EXCLUDE constraints that combine a scalar
-- (e.g. president_id WITH =) with a tstzrange (valid_period WITH &&).
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- ===== government.constitutions ============================================
-- A country's constitution. A country may have multiple historical
-- constitutions (e.g. Kenya 1963, 2010); version distinguishes them.

CREATE TABLE IF NOT EXISTS government.constitutions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_code    CHAR(2) NOT NULL,                   -- ISO 3166-1 alpha-2
    title           TEXT NOT NULL,
    promulgated_at  TIMESTAMPTZ NOT NULL,
    assented_at     TIMESTAMPTZ,                        -- NULL when not applicable (e.g. some transitional constitutions)
    version         TEXT NOT NULL,                     -- e.g. "2010"
    source_url      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (country_code, version)
);
CREATE INDEX idx_gov_constitutions_country ON government.constitutions(country_code);

-- ===== government.constitution_chapters ====================================
-- A chapter groups related articles. Spec §1.

CREATE TABLE IF NOT EXISTS government.constitution_chapters (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    constitution_id UUID NOT NULL REFERENCES government.constitutions(id) ON DELETE CASCADE,
    number          INT NOT NULL CHECK (number > 0),
    title           TEXT NOT NULL,
    UNIQUE (constitution_id, number)
);
CREATE INDEX idx_gov_chapters_constitution ON government.constitution_chapters(constitution_id, number);

-- ===== government.constitution_articles ====================================
-- A single article of the constitution. Spec §1.

CREATE TABLE IF NOT EXISTS government.constitution_articles (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chapter_id  UUID NOT NULL REFERENCES government.constitution_chapters(id) ON DELETE CASCADE,
    number      TEXT NOT NULL,                          -- e.g. "Article 1", "Article 10"
    title       TEXT,
    text        TEXT NOT NULL,
    source_url  TEXT,
    UNIQUE (chapter_id, number)
);
CREATE INDEX idx_gov_articles_chapter ON government.constitution_articles(chapter_id, number);

-- ===== government.constitution_cross_references ============================
-- Links a constitutional article to another civic entity (Bill, Act,
-- Regulation, Court Decision, Institution). Spec §6.
-- reference_type: EXPLICIT | INFERRED | JUDICIAL | UNKNOWN

CREATE TABLE IF NOT EXISTS government.constitution_cross_references (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id      UUID NOT NULL REFERENCES government.constitution_articles(id) ON DELETE CASCADE,
    target_type     TEXT NOT NULL,                     -- BILL | ACT | REGULATION | COURT_DECISION | INSTITUTION
    target_id       UUID NOT NULL,                     -- opaque UUID of the target entity
    reference_type  TEXT NOT NULL DEFAULT 'UNKNOWN',
    source_url      TEXT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_cross_ref_target_type CHECK (
        target_type IN ('BILL','ACT','REGULATION','COURT_DECISION','INSTITUTION')
    ),
    CONSTRAINT chk_cross_ref_type CHECK (
        reference_type IN ('EXPLICIT','INFERRED','JUDICIAL','UNKNOWN')
    )
);
CREATE INDEX idx_gov_cross_refs_article ON government.constitution_cross_references(article_id);
CREATE INDEX idx_gov_cross_refs_target  ON government.constitution_cross_references(target_type, target_id);
CREATE INDEX idx_gov_cross_refs_type    ON government.constitution_cross_references(reference_type);

-- ===== government.presidents ================================================
-- Head of state (or head of government in some systems). Spec §7.

CREATE TABLE IF NOT EXISTS government.presidents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_code    CHAR(2) NOT NULL,
    full_name       TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    born_at         TIMESTAMPTZ,
    biography_url   TEXT,
    photo_url       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_gov_presidents_country ON government.presidents(country_code);

-- ===== government.administrations ===========================================
-- A presidential administration. Spec §7.
-- Temporal validity: an administration has at most one active validity window
-- per (country_code, president_id). The EXCLUDE constraint prevents overlap.

CREATE TABLE IF NOT EXISTS government.administrations (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_code      CHAR(2) NOT NULL,
    president_id      UUID NOT NULL REFERENCES government.presidents(id) ON DELETE RESTRICT,
    name              TEXT NOT NULL,
    start_date        TIMESTAMPTZ NOT NULL,
    end_date          TIMESTAMPTZ,
    government_system  TEXT NOT NULL DEFAULT 'PRESIDENTIAL',
    source_url        TEXT,
    valid_from        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_admin_system CHECK (
        government_system IN ('PRESIDENTIAL','PARLIAMENTARY','SEMI_PRESIDENTIAL',
                              'MONARCHICAL','TRANSITIONAL','OTHER')
    ),
    CONSTRAINT chk_admin_dates CHECK (
        end_date IS NULL OR end_date >= start_date
    ),
    CONSTRAINT chk_admin_valid_window CHECK (
        valid_until IS NULL OR valid_until >= valid_from
    ),
    CONSTRAINT excl_admin_validity EXCLUDE USING gist (
        country_code WITH =,
        president_id WITH =,
        (tstzrange(valid_from, valid_until)) WITH &&
    )
);
CREATE INDEX idx_gov_admin_country   ON government.administrations(country_code);
CREATE INDEX idx_gov_admin_president ON government.administrations(president_id);
CREATE INDEX idx_gov_admin_dates     ON government.administrations(start_date, end_date);

-- ===== government.presidential_terms =======================================
-- A single term of a president. A president may have multiple terms. Spec §8.
-- Temporal validity: at most one active term per (administration_id, term_number)
-- and at most one active term per (president_id) at any point in time.

CREATE TABLE IF NOT EXISTS government.presidential_terms (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    administration_id UUID NOT NULL REFERENCES government.administrations(id) ON DELETE CASCADE,
    president_id      UUID NOT NULL REFERENCES government.presidents(id) ON DELETE RESTRICT,
    term_number       INT NOT NULL CHECK (term_number > 0),
    election_date     TIMESTAMPTZ,
    swearing_in_date  TIMESTAMPTZ,
    start_date        TIMESTAMPTZ NOT NULL,
    end_date          TIMESTAMPTZ,
    status            TEXT NOT NULL DEFAULT 'active',   -- active | completed | vacant | revoked
    source_url        TEXT,
    valid_from        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_term_status CHECK (
        status IN ('active','completed','vacant','revoked')
    ),
    CONSTRAINT chk_term_dates CHECK (
        end_date IS NULL OR end_date >= start_date
    ),
    CONSTRAINT chk_term_valid_window CHECK (
        valid_until IS NULL OR valid_until >= valid_from
    ),
    CONSTRAINT excl_term_admin_validity EXCLUDE USING gist (
        administration_id WITH =,
        (tstzrange(valid_from, valid_until)) WITH &&
    ),
    CONSTRAINT excl_term_president_validity EXCLUDE USING gist (
        president_id WITH =,
        (tstzrange(valid_from, valid_until)) WITH &&
    )
);
CREATE INDEX idx_gov_terms_admin     ON government.presidential_terms(administration_id);
CREATE INDEX idx_gov_terms_president ON government.presidential_terms(president_id);
CREATE INDEX idx_gov_terms_dates     ON government.presidential_terms(start_date, end_date);
CREATE UNIQUE INDEX uq_gov_terms_admin_number
    ON government.presidential_terms(administration_id, term_number)
    WHERE valid_until IS NULL;

-- ===== government.government_periods =======================================
-- Operational period of a government: administration + (optional) parliament +
-- cabinet. Spec §9. Temporal validity: at most one active period per
-- administration at any point in time.

CREATE TABLE IF NOT EXISTS government.government_periods (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    administration_id     UUID NOT NULL REFERENCES government.administrations(id) ON DELETE CASCADE,
    presidential_term_id  UUID REFERENCES government.presidential_terms(id) ON DELETE SET NULL,
    parliament_id         UUID,                          -- opaque; not a hard FK (parliament lives in legislation.houses)
    start_date            TIMESTAMPTZ NOT NULL,
    end_date              TIMESTAMPTZ,
    valid_from            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until           TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_period_dates CHECK (
        end_date IS NULL OR end_date >= start_date
    ),
    CONSTRAINT chk_period_valid_window CHECK (
        valid_until IS NULL OR valid_until >= valid_from
    ),
    CONSTRAINT excl_period_admin_validity EXCLUDE USING gist (
        administration_id WITH =,
        (tstzrange(valid_from, valid_until)) WITH &&
    )
);
CREATE INDEX idx_gov_periods_admin   ON government.government_periods(administration_id);
CREATE INDEX idx_gov_periods_term    ON government.government_periods(presidential_term_id);
CREATE INDEX idx_gov_periods_dates   ON government.government_periods(start_date, end_date);

-- ===== government.cabinet_members ==========================================
-- A member of the cabinet for a government period. Spec §9.
-- Temporal validity: at most one active membership per (government_period_id,
-- person_id) at any point in time.

CREATE TABLE IF NOT EXISTS government.cabinet_members (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    government_period_id  UUID NOT NULL REFERENCES government.government_periods(id) ON DELETE CASCADE,
    person_id             UUID NOT NULL,               -- opaque; references legislation.people(id)
    role                  TEXT NOT NULL,               -- e.g. "Cabinet Secretary for Finance"
    ministry              TEXT NOT NULL,
    start_date            TIMESTAMPTZ NOT NULL,
    end_date              TIMESTAMPTZ,
    source_url            TEXT,
    valid_from            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until           TIMESTAMPTZ,
    CONSTRAINT chk_cabinet_dates CHECK (
        end_date IS NULL OR end_date >= start_date
    ),
    CONSTRAINT chk_cabinet_valid_window CHECK (
        valid_until IS NULL OR valid_until >= valid_from
    ),
    CONSTRAINT excl_cabinet_person_validity EXCLUDE USING gist (
        government_period_id WITH =,
        person_id WITH =,
        (tstzrange(valid_from, valid_until)) WITH &&
    )
);
CREATE INDEX idx_gov_cabinet_period  ON government.cabinet_members(government_period_id);
CREATE INDEX idx_gov_cabinet_person  ON government.cabinet_members(person_id);
CREATE INDEX idx_gov_cabinet_dates   ON government.cabinet_members(start_date, end_date);

-- ===== government.transitions ===============================================
-- A first-class civic event marking the handover from one administration to
-- another. Spec §28. Historical truth is never overwritten — a transition is
-- append-only (the historical record of transitions cannot be edited).

CREATE TABLE IF NOT EXISTS government.transitions (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_code            CHAR(2) NOT NULL,
    outgoing_admin_id       UUID REFERENCES government.administrations(id) ON DELETE SET NULL,
    incoming_admin_id       UUID NOT NULL REFERENCES government.administrations(id) ON DELETE RESTRICT,
    transition_date         TIMESTAMPTZ NOT NULL,
    outgoing_president_id   UUID REFERENCES government.presidents(id) ON DELETE SET NULL,
    incoming_president_id   UUID NOT NULL REFERENCES government.presidents(id) ON DELETE RESTRICT,
    source_url              TEXT,
    notes                   TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_gov_transitions_country    ON government.transitions(country_code, transition_date DESC);
CREATE INDEX idx_gov_transitions_outgoing   ON government.transitions(outgoing_admin_id);
CREATE INDEX idx_gov_transitions_incoming   ON government.transitions(incoming_admin_id);
CREATE INDEX idx_gov_transitions_in_pres    ON government.transitions(incoming_president_id);

-- ===== updated_at maintenance trigger ======================================
-- Auto-touch updated_at on constitutions.

CREATE OR REPLACE FUNCTION government.tg_touch_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg_constitutions_touch_updated
    BEFORE UPDATE ON government.constitutions
    FOR EACH ROW EXECUTE FUNCTION government.tg_touch_updated_at();

COMMIT;
