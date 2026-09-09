-- 009_bills.up.sql
CREATE TABLE legislation.bills (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    house_id        UUID NOT NULL REFERENCES legislation.houses(id),
    identifier      TEXT NOT NULL,
    year             INT NOT NULL,
    title           TEXT NOT NULL,
    sponsor_id      UUID REFERENCES legislation.people(id),
    committee_id    UUID REFERENCES legislation.committees(id),
    status          TEXT NOT NULL DEFAULT 'in_progress',
    current_stage   TEXT,
    purpose         TEXT,
    description     TEXT,
    topics          JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (country_id, identifier, year)
);
CREATE INDEX idx_bills_status     ON legislation.bills(status);
CREATE INDEX idx_bills_year       ON legislation.bills(year DESC);
CREATE INDEX idx_bills_topics     ON legislation.bills USING gin (topics);
CREATE INDEX idx_bills_title_trgm ON legislation.bills USING gin (title gin_trgm_ops);

CREATE TABLE legislation.bill_versions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_id         UUID NOT NULL REFERENCES legislation.bills(id) ON DELETE RESTRICT,
    version_no      INT NOT NULL CHECK (version_no > 0),
    content_hash    TEXT NOT NULL,
    retrieved_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_url      TEXT NOT NULL,
    document_id     UUID REFERENCES ingestion.documents(id),
    is_current      BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (bill_id, version_no)
);
CREATE UNIQUE INDEX uq_bill_versions_current ON legislation.bill_versions(bill_id) WHERE is_current;

CREATE TABLE legislation.bill_stages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    code            TEXT NOT NULL,
    name            TEXT NOT NULL,
    simple_explanation TEXT NOT NULL,
    official_definition TEXT,
    allowed_next    JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_terminal     BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (country_id, code)
);

CREATE TABLE legislation.bill_events (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_id         UUID NOT NULL REFERENCES legislation.bills(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,
    event_date      DATE,
    date_is_approximate BOOLEAN NOT NULL DEFAULT FALSE,
    house           TEXT,
    description     TEXT NOT NULL,
    source_document_id UUID REFERENCES ingestion.documents(id),
    source_url      TEXT,
    confidence      TEXT NOT NULL DEFAULT 'unknown',
    note            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_bill_events_bill_date ON legislation.bill_events(bill_id, event_date);
CREATE INDEX idx_bill_events_type      ON legislation.bill_events(event_type);

CREATE TABLE legislation.amendments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_id         UUID NOT NULL REFERENCES legislation.bills(id) ON DELETE CASCADE,
    bill_version_id UUID REFERENCES legislation.bill_versions(id),
    mover_id        UUID REFERENCES legislation.people(id),
    title           TEXT NOT NULL,
    text            TEXT NOT NULL,
    proposed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status          TEXT NOT NULL DEFAULT 'proposed',
    source_url      TEXT
);

CREATE TABLE legislation.clauses (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_version_id UUID NOT NULL REFERENCES legislation.bill_versions(id) ON DELETE CASCADE,
    clause_no       INT NOT NULL,
    heading         TEXT,
    text            TEXT NOT NULL,
    page_number     INT,
    UNIQUE (bill_version_id, clause_no)
);

CREATE TRIGGER audit_bill_insert AFTER INSERT ON legislation.bills
    FOR EACH ROW EXECUTE FUNCTION audit.tg_record('bill.create');
CREATE TRIGGER audit_bill_update AFTER UPDATE ON legislation.bills
    FOR EACH ROW EXECUTE FUNCTION audit.tg_record('bill.update');
