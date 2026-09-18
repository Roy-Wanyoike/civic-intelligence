-- 017_government_loans_grants.up.sql
-- Government loans and grants tracker. Each row represents a sovereign loan
-- taken or grant received by the Kenyan government. Every row MUST link to
-- at least one credible source URL (evidence-first contract).

CREATE TABLE legislation.government_loans (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    lender          TEXT NOT NULL,         -- IMF, World Bank, China Exim Bank, etc.
    loan_type       TEXT NOT NULL,         -- bilateral, syndicated, eurobond, multilateral
    amount_usd      NUMERIC(15,2),
    amount_kes      NUMERIC(15,2),
    currency        TEXT NOT NULL DEFAULT 'USD',
    purpose         TEXT NOT NULL,
    sector          TEXT,                  -- infrastructure, health, education, etc.
    application_date DATE,
    approval_date   DATE,
    disbursement_date DATE,
    interest_rate   NUMERIC(5,2),         -- percentage
    repayment_period_years INT,
    status          TEXT NOT NULL DEFAULT 'applied', -- applied, approved, disbursed, repaid, defaulted
    source_url      TEXT,
    source_document_id UUID REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_loans_lender ON legislation.government_loans(lender);
CREATE INDEX idx_loans_status ON legislation.government_loans(status);
CREATE INDEX idx_loans_sector ON legislation.government_loans(sector);
CREATE INDEX idx_loans_date ON legislation.government_loans(application_date DESC);

CREATE TABLE legislation.government_grants (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    donor           TEXT NOT NULL,         -- USAID, EU, GIZ, JICA, etc.
    grant_type      TEXT NOT NULL,         -- bilateral, multilateral, foundation
    amount_usd      NUMERIC(15,2),
    amount_kes      NUMERIC(15,2),
    purpose         TEXT NOT NULL,
    sector          TEXT,
    announcement_date DATE,
    disbursement_date DATE,
    status          TEXT NOT NULL DEFAULT 'announced', -- announced, disbursed, pending
    source_url      TEXT,
    source_document_id UUID REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_grants_donor ON legislation.government_grants(donor);
CREATE INDEX idx_grants_status ON legislation.government_grants(status);
CREATE INDEX idx_grants_sector ON legislation.government_grants(sector);
CREATE INDEX idx_grants_date ON legislation.government_grants(announcement_date DESC);

-- Audit triggers (mirrors the pattern used by legislation.bills)
CREATE TRIGGER audit_loan_insert AFTER INSERT ON legislation.government_loans
    FOR EACH ROW EXECUTE FUNCTION audit.tg_record('loan.create');
CREATE TRIGGER audit_loan_update AFTER UPDATE ON legislation.government_loans
    FOR EACH ROW EXECUTE FUNCTION audit.tg_record('loan.update');
CREATE TRIGGER audit_grant_insert AFTER INSERT ON legislation.government_grants
    FOR EACH ROW EXECUTE FUNCTION audit.tg_record('grant.create');
CREATE TRIGGER audit_grant_update AFTER UPDATE ON legislation.government_grants
    FOR EACH ROW EXECUTE FUNCTION audit.tg_record('grant.update');
