-- 022_search_fts.down.sql
-- Reverses 022_search_fts.up.sql exactly:
--   1. Drop the three search.* functions.
--   2. Drop the triggers on legislation.acts and government.constitution_articles.
--   3. Drop the trigger on legislation.bills (no-op marker).
--   4. Drop the trigger functions.
--   5. Drop the GIN indexes.
--   6. Drop the fts_search columns (which also removes the GENERATED expression).
--
-- All DROPs are IF EXISTS so the down migration is safe to re-run.
-- Wrapped in a single transaction so a partial failure rolls back cleanly.

BEGIN;

-- ============================================================================
-- 1. Search functions
-- ============================================================================
DROP FUNCTION IF EXISTS search.search_bills(text, int);
DROP FUNCTION IF EXISTS search.search_acts(text, int);
DROP FUNCTION IF EXISTS search.search_constitution(text, int);

-- ============================================================================
-- 2. Triggers
-- ============================================================================
DROP TRIGGER IF EXISTS tg_bills_fts_touch                ON legislation.bills;
DROP TRIGGER IF EXISTS tg_acts_fts_update                 ON legislation.acts;
DROP TRIGGER IF EXISTS tg_constitution_articles_fts_update ON government.constitution_articles;

-- ============================================================================
-- 3. Trigger functions
-- ============================================================================
DROP FUNCTION IF EXISTS legislation.fn_bills_fts_touch();
DROP FUNCTION IF EXISTS legislation.fn_acts_fts_update();
DROP FUNCTION IF EXISTS government.fn_constitution_articles_fts_update();

-- ============================================================================
-- 4. GIN indexes
-- ============================================================================
DROP INDEX IF EXISTS legislation.idx_bills_fts_search;
DROP INDEX IF EXISTS legislation.idx_acts_fts_search;
DROP INDEX IF EXISTS government.idx_constitution_articles_fts_search;

-- ============================================================================
-- 5. tsvector columns
--    Dropping the column drops the GENERATED expression with it, so no
--    separate cleanup is needed for the GENERATED ALWAYS AS (...) clause.
-- ============================================================================
ALTER TABLE legislation.bills
    DROP COLUMN IF EXISTS fts_search;
ALTER TABLE legislation.acts
    DROP COLUMN IF EXISTS fts_search;
ALTER TABLE government.constitution_articles
    DROP COLUMN IF EXISTS fts_search;

COMMIT;
