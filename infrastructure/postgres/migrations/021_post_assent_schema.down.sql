-- 021_post_assent_schema.down.sql
-- Reverse of 021_post_assent_schema.up.sql. Drops the immutability trigger
-- first, then the function, then the tables. CASCADE on table drops removes
-- dependent indexes / FKs automatically.
--
-- NOTE: This does NOT drop legislation.acts — that table is owned by
-- 010_acts.up.sql. Only the post-assent tables added by 021 are dropped.

BEGIN;

DROP TRIGGER IF EXISTS tg_act_versions_immutable ON legislation.act_versions;
DROP FUNCTION IF EXISTS legislation.tg_block_act_version_mutation();

DROP TABLE IF EXISTS legislation.legislative_lifecycle_audits CASCADE;
DROP TABLE IF EXISTS legislation.post_assent_events          CASCADE;
DROP TABLE IF EXISTS legislation.act_versions                 CASCADE;
DROP TABLE IF EXISTS legislation.presidential_assent_events  CASCADE;

COMMIT;
