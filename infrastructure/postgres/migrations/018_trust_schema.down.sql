-- 018_trust_schema.down.sql
-- Reverse of 018_trust_schema.up.sql. Drops triggers first, then tables, then
-- the audit function, then the schema. The order matters: audit triggers
-- reference trust.tg_record_audit() so they must be dropped before the
-- function, but they live ON the tables, so dropping the tables cascades
-- the triggers automatically. We still DROP TRIGGER IF EXISTS explicitly
-- to keep this idempotent even if a partial apply happened.

DROP TRIGGER IF EXISTS tg_trust_review_queue_audit_update ON trust.review_queue;
DROP TRIGGER IF EXISTS tg_trust_review_queue_audit_insert ON trust.review_queue;
DROP TRIGGER IF EXISTS tg_trust_corrections_audit_update ON trust.corrections;
DROP TRIGGER IF EXISTS tg_trust_corrections_audit_insert ON trust.corrections;
DROP TRIGGER IF EXISTS tg_trust_contradictions_audit_update ON trust.contradictions;
DROP TRIGGER IF EXISTS tg_trust_contradictions_audit_insert ON trust.contradictions;
DROP TRIGGER IF EXISTS tg_trust_claims_audit_update ON trust.claims;
DROP TRIGGER IF EXISTS tg_trust_claims_audit_insert ON trust.claims;

DROP TRIGGER IF EXISTS tg_audit_events_no_update ON trust.audit_events;
DROP FUNCTION IF EXISTS trust.tg_audit_events_immutable();
DROP FUNCTION IF EXISTS trust.tg_record_audit();

DROP TABLE IF EXISTS trust.review_queue;
DROP TABLE IF EXISTS trust.audit_events;
DROP TABLE IF EXISTS trust.corrections;
DROP TABLE IF EXISTS trust.contradictions;
DROP TABLE IF EXISTS trust.evidence;
DROP TABLE IF EXISTS trust.claims;
DROP TABLE IF EXISTS trust.source_checks;
DROP TABLE IF EXISTS trust.sources;

DROP SCHEMA IF EXISTS trust;
