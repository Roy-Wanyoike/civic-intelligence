-- 004_audit.down.sql
DROP FUNCTION IF EXISTS audit.tg_record() CASCADE;
DROP TABLE IF EXISTS audit.log_entries;
