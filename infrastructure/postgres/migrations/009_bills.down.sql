-- 009_bills.down.sql
DROP TRIGGER IF EXISTS audit_bill_update ON legislation.bills;
DROP TRIGGER IF EXISTS audit_bill_insert ON legislation.bills;
DROP TABLE IF EXISTS legislation.clauses;
DROP TABLE IF EXISTS legislation.amendments;
DROP TABLE IF EXISTS legislation.bill_events;
DROP TABLE IF EXISTS legislation.bill_stages;
DROP TABLE IF EXISTS legislation.bill_versions;
DROP TABLE IF EXISTS legislation.bills;
