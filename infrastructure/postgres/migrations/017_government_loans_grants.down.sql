-- 017_government_loans_grants.down.sql
DROP TRIGGER IF EXISTS audit_grant_update ON legislation.government_grants;
DROP TRIGGER IF EXISTS audit_grant_insert ON legislation.government_grants;
DROP TRIGGER IF EXISTS audit_loan_update ON legislation.government_loans;
DROP TRIGGER IF EXISTS audit_loan_insert ON legislation.government_loans;

DROP TABLE IF EXISTS legislation.government_grants;
DROP TABLE IF EXISTS legislation.government_loans;
