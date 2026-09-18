-- 022_search_fts.up.sql
-- Phase: Full-text search on canonical legislation + constitution tables.
--
-- Adds tsvector columns, GIN indexes, and auto-update triggers to:
--   * legislation.bills              (title + identifier + purpose + description)
--   * legislation.acts               (title + citation)
--   * government.constitution_articles (number + title + text)
--
-- Adds three SQL functions under the `search` schema that the search service
-- (and any read API) calls to run websearch-style queries:
--   * search.search_bills(query text, limit int)
--   * search.search_acts(query text, limit int)
--   * search.search_constitution(query text, limit int)
--
-- Each function returns the entity UUID plus a ts_rank_cd score and a snippet
-- (headline). The `search` schema is documented as "search projections rebuilt
-- from canonical data" (002_schemas.up.sql); these functions READ from the
-- canonical tables directly (no projection table needed) — the tsvector is a
-- STORED GENERATED column on the canonical table, so it is always in sync.
--
-- Dependencies:
--   * 001_extensions (unaccent, pg_trgm) — already created.
--   * 002_schemas (legislation, government, search) — already created.
--   * 009_bills (legislation.bills) — already created.
--   * 010_acts (legislation.acts) — already created.
--   * 020_government_schema (government.constitution_articles) — already created.
--
-- All statements are idempotent (CREATE ... IF NOT EXISTS) so the migration
-- is safe to re-apply.

BEGIN;

-- ============================================================================
-- legislation.bills — tsvector for title + identifier + purpose + description
-- ============================================================================

ALTER TABLE legislation.bills
    ADD COLUMN IF NOT EXISTS fts_search tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', unaccent(
            coalesce(title, '')       || ' ' ||
            coalesce(identifier, '')  || ' ' ||
            coalesce(purpose, '')      || ' ' ||
            coalesce(description, '')
        ))
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_bills_fts_search
    ON legislation.bills USING gin (fts_search);

-- The trigger keeps the fts_search column's ts_rank headline snippet available
-- without recomputation. (Generated columns auto-update, so this trigger is
-- a no-op marker — included for symmetry with acts/constitution where the
-- column is NOT generated and we DO need a manual update.)
CREATE OR REPLACE FUNCTION legislation.fn_bills_fts_touch() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    -- No-op: fts_search is GENERATED ALWAYS, so Postgres maintains it.
    RETURN NEW;
END;
$$;
COMMENT ON FUNCTION legislation.fn_bills_fts_touch IS
    'Placeholder trigger function — bills.fts_search is GENERATED ALWAYS so no manual update is needed. Kept for symmetry with acts/constitution where manual updates are required.';

DROP TRIGGER IF EXISTS tg_bills_fts_touch ON legislation.bills;
CREATE TRIGGER tg_bills_fts_touch
    AFTER INSERT OR UPDATE OF title, identifier, purpose, description
    ON legislation.bills
    FOR EACH ROW EXECUTE FUNCTION legislation.fn_bills_fts_touch();

-- ============================================================================
-- legislation.acts — tsvector for title + citation (acts have no purpose/desc)
-- ============================================================================

ALTER TABLE legislation.acts
    ADD COLUMN IF NOT EXISTS fts_search tsvector;

-- Acts don't change often, but the title/citation are mutable (a correction
-- to a citation is a legitimate edit). A trigger maintains fts_search.
CREATE OR REPLACE FUNCTION legislation.fn_acts_fts_update() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    NEW.fts_search := to_tsvector('english', unaccent(
        coalesce(NEW.title, '')     || ' ' ||
        coalesce(NEW.citation, '')
    ));
    RETURN NEW;
END;
$$;
COMMENT ON FUNCTION legislation.fn_acts_fts_update IS
    'Maintains legislation.acts.fts_search on every INSERT/UPDATE of title or citation.';

DROP TRIGGER IF EXISTS tg_acts_fts_update ON legislation.acts;
CREATE TRIGGER tg_acts_fts_update
    BEFORE INSERT OR UPDATE OF title, citation
    ON legislation.acts
    FOR EACH ROW EXECUTE FUNCTION legislation.fn_acts_fts_update();

-- Backfill any pre-existing rows (their fts_search is currently NULL).
UPDATE legislation.acts
   SET fts_search = to_tsvector('english', unaccent(
       coalesce(title, '') || ' ' || coalesce(citation, '')
   ))
 WHERE fts_search IS NULL;

CREATE INDEX IF NOT EXISTS idx_acts_fts_search
    ON legislation.acts USING gin (fts_search);

-- ============================================================================
-- government.constitution_articles — tsvector for number + title + text
-- ============================================================================

ALTER TABLE government.constitution_articles
    ADD COLUMN IF NOT EXISTS fts_search tsvector;

CREATE OR REPLACE FUNCTION government.fn_constitution_articles_fts_update() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    NEW.fts_search := to_tsvector('english', unaccent(
        coalesce(NEW.number, '') || ' ' ||
        coalesce(NEW.title, '')   || ' ' ||
        coalesce(NEW.text, '')
    ));
    RETURN NEW;
END;
$$;
COMMENT ON FUNCTION government.fn_constitution_articles_fts_update IS
    'Maintains government.constitution_articles.fts_search on every INSERT/UPDATE of number, title, or text.';

DROP TRIGGER IF EXISTS tg_constitution_articles_fts_update ON government.constitution_articles;
CREATE TRIGGER tg_constitution_articles_fts_update
    BEFORE INSERT OR UPDATE OF number, title, text
    ON government.constitution_articles
    FOR EACH ROW EXECUTE FUNCTION government.fn_constitution_articles_fts_update();

-- Backfill any pre-existing rows.
UPDATE government.constitution_articles
   SET fts_search = to_tsvector('english', unaccent(
       coalesce(number, '') || ' ' || coalesce(title, '') || ' ' || coalesce(text, '')
   ))
 WHERE fts_search IS NULL;

CREATE INDEX IF NOT EXISTS idx_constitution_articles_fts_search
    ON government.constitution_articles USING gin (fts_search);

-- ============================================================================
-- Search functions (websearch_to_tsquery supports "quoted phrases" + AND/OR)
-- ============================================================================

-- --- search.search_bills ----------------------------------------------------
-- Returns: (id, title, identifier, year, status, rank, snippet)
CREATE OR REPLACE FUNCTION search.search_bills(
    query text,
    limit int DEFAULT 20
) RETURNS TABLE (
    id          uuid,
    title       text,
    identifier  text,
    year        int,
    status      text,
    rank        real,
    snippet     text
)
LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT
        b.id,
        b.title,
        b.identifier,
        b.year,
        b.status,
        ts_rank_cd(b.fts_search, q) AS rank,
        ts_headline('english',
            coalesce(b.title, '')     || E'\n' ||
            coalesce(b.identifier, '') || E'\n' ||
            coalesce(b.purpose, '')    || E'\n' ||
            coalesce(b.description, ''),
            q,
            'StartSel=<mark>, StopSel=</mark>, MaxWords=35, MinWords=15'
        ) AS snippet
    FROM legislation.bills b,
         websearch_to_tsquery('english', unaccent(query)) AS q
    WHERE b.fts_search @@ q
    ORDER BY rank DESC, b.year DESC
    LIMIT GREATEST(LEAST(limit, 200), 1);  -- clamp [1, 200]
$$;
COMMENT ON FUNCTION search.search_bills IS
    'Websearch-style FTS over legislation.bills. Clamps limit to [1, 200]. Returns ts_rank_cd score and a <mark>-highlighted snippet.';

-- --- search.search_acts -----------------------------------------------------
-- Returns: (id, citation, title, assent_date, rank, snippet)
CREATE OR REPLACE FUNCTION search.search_acts(
    query text,
    limit int DEFAULT 20
) RETURNS TABLE (
    id           uuid,
    citation     text,
    title        text,
    assent_date  date,
    rank         real,
    snippet      text
)
LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT
        a.id,
        a.citation,
        a.title,
        a.assent_date,
        ts_rank_cd(a.fts_search, q) AS rank,
        ts_headline('english',
            coalesce(a.title, '') || E'\n' || coalesce(a.citation, ''),
            q,
            'StartSel=<mark>, StopSel=</mark>, MaxWords=35, MinWords=15'
        ) AS snippet
    FROM legislation.acts a,
         websearch_to_tsquery('english', unaccent(query)) AS q
    WHERE a.fts_search @@ q
    ORDER BY rank DESC, a.assent_date DESC NULLS LAST
    LIMIT GREATEST(LEAST(limit, 200), 1);
$$;
COMMENT ON FUNCTION search.search_acts IS
    'Websearch-style FTS over legislation.acts. Clamps limit to [1, 200].';

-- --- search.search_constitution --------------------------------------------
-- Returns: (id, number, title, snippet, rank, chapter_id)
CREATE OR REPLACE FUNCTION search.search_constitution(
    query text,
    limit int DEFAULT 20
) RETURNS TABLE (
    id          uuid,
    number      text,
    title       text,
    snippet     text,
    rank        real,
    chapter_id  uuid
)
LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT
        c.id,
        c.number,
        c.title,
        ts_headline('english',
            coalesce(c.number, '') || E'\n' || coalesce(c.title, '') || E'\n' || coalesce(c.text, ''),
            q,
            'StartSel=<mark>, StopSel=</mark>, MaxWords=35, MinWords=15'
        ) AS snippet,
        ts_rank_cd(c.fts_search, q) AS rank,
        c.chapter_id
    FROM government.constitution_articles c,
         websearch_to_tsquery('english', unaccent(query)) AS q
    WHERE c.fts_search @@ q
    ORDER BY rank DESC, c.number ASC
    LIMIT GREATEST(LEAST(limit, 200), 1);
$$;
COMMENT ON FUNCTION search.search_constitution IS
    'Websearch-style FTS over government.constitution_articles. Clamps limit to [1, 200].';

COMMIT;
