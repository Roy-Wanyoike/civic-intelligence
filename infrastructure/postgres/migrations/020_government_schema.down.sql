-- 020_government_schema.down.sql
-- Reverse of 020_government_schema.up.sql. Drops triggers first, then tables
-- in reverse FK-dependency order, then the function, then the schema.
-- CASCADE on table drops removes dependent indexes / FKs automatically.
--
-- NOTE: The btree_gist extension is intentionally NOT dropped here — other
-- schemas may already depend on it, and CREATE EXTENSION IF NOT EXISTS is
-- idempotent so re-applying the up-migration is safe.

BEGIN;

DROP TRIGGER IF EXISTS tg_constitutions_touch_updated ON government.constitutions;
DROP FUNCTION IF EXISTS government.tg_touch_updated_at();

-- Drop tables in reverse FK-dependency order:
--   transitions / cabinet_members / government_periods → presidential_terms
--   → administrations → presidents
--   cross_references → articles → chapters → constitutions.
DROP TABLE IF EXISTS government.transitions              CASCADE;
DROP TABLE IF EXISTS government.cabinet_members          CASCADE;
DROP TABLE IF EXISTS government.government_periods       CASCADE;
DROP TABLE IF EXISTS government.presidential_terms       CASCADE;
DROP TABLE IF EXISTS government.administrations         CASCADE;
DROP TABLE IF EXISTS government.presidents              CASCADE;
DROP TABLE IF EXISTS government.constitution_cross_references CASCADE;
DROP TABLE IF EXISTS government.constitution_articles    CASCADE;
DROP TABLE IF EXISTS government.constitution_chapters    CASCADE;
DROP TABLE IF EXISTS government.constitutions            CASCADE;

DROP SCHEMA IF EXISTS government CASCADE;

COMMIT;
