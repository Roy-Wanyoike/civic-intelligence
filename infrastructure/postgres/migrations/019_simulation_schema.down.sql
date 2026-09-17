-- 019_simulation_schema.down.sql
-- Reverse of 019_simulation_schema.up.sql. Drops triggers first, then tables,
-- then the functions, then the schema. CASCADE on table drops removes
-- dependent indexes / FKs automatically.

BEGIN;

-- Drop triggers explicitly so the down-migration is idempotent even if a
-- partial apply happened (matches 018_trust_schema.down.sql pattern).
DROP TRIGGER IF EXISTS tg_scenario_models_touch_updated ON simulation.scenario_models;
DROP TRIGGER IF EXISTS tg_scenarios_touch_updated       ON simulation.scenarios;
DROP TRIGGER IF EXISTS tg_scenario_audits_immutable     ON simulation.scenario_audits;
DROP TRIGGER IF EXISTS tg_simulation_results_immutable  ON simulation.simulation_results;
DROP TRIGGER IF EXISTS tg_scenario_versions_immutable   ON simulation.scenario_versions;

DROP FUNCTION IF EXISTS simulation.tg_touch_updated_at();
DROP FUNCTION IF EXISTS simulation.tg_block_mutation();

-- Drop tables in reverse FK-dependency order:
--   metrics → results → runs → inputs / assumptions / evidence / relationships
--   → scenario_versions → scenarios → scenario_models.
DROP TABLE IF EXISTS simulation.simulation_metrics      CASCADE;
DROP TABLE IF EXISTS simulation.simulation_results      CASCADE;
DROP TABLE IF EXISTS simulation.simulation_runs         CASCADE;
DROP TABLE IF EXISTS simulation.scenario_inputs         CASCADE;
DROP TABLE IF EXISTS simulation.scenario_assumptions    CASCADE;
DROP TABLE IF EXISTS simulation.scenario_evidence       CASCADE;
DROP TABLE IF EXISTS simulation.scenario_relationships  CASCADE;
DROP TABLE IF EXISTS simulation.scenario_versions        CASCADE;
DROP TABLE IF EXISTS simulation.scenario_audits         CASCADE;
DROP TABLE IF EXISTS simulation.scenarios               CASCADE;
DROP TABLE IF EXISTS simulation.scenario_models         CASCADE;

DROP SCHEMA IF EXISTS simulation CASCADE;

COMMIT;
