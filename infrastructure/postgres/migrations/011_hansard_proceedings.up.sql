-- 011_hansard_proceedings.up.sql
CREATE TABLE legislation.hansard_documents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    house_id        UUID REFERENCES legislation.houses(id),
    sitting_date    DATE,
    title           TEXT NOT NULL,
    source_url      TEXT,
    document_id     UUID NOT NULL REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_hansard_date ON legislation.hansard_documents(sitting_date DESC);

CREATE TABLE legislation.order_papers (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    house_id        UUID NOT NULL REFERENCES legislation.houses(id),
    sitting_date    DATE NOT NULL,
    source_url      TEXT,
    document_id     UUID NOT NULL REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (house_id, sitting_date)
);

CREATE TABLE legislation.committee_reports (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    committee_id    UUID NOT NULL REFERENCES legislation.committees(id),
    title           TEXT NOT NULL,
    report_type     TEXT,
    published_date  DATE,
    source_url      TEXT,
    document_id     UUID NOT NULL REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE legislation.votes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    house_id        UUID NOT NULL REFERENCES legislation.houses(id),
    bill_id         UUID REFERENCES legislation.bills(id),
    person_id       UUID REFERENCES legislation.people(id),
    vote            TEXT NOT NULL,
    vote_date       DATE NOT NULL,
    source_url      TEXT,
    document_id     UUID REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_votes_bill ON legislation.votes(bill_id);

CREATE TABLE legislation.votes_proceedings (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    house_id        UUID NOT NULL REFERENCES legislation.houses(id),
    sitting_date    DATE NOT NULL,
    source_url      TEXT,
    document_id     UUID NOT NULL REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE legislation.gazette_notices (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    gazette_no      TEXT,
    notice_no       TEXT,
    publication_date DATE,
    title           TEXT NOT NULL,
    source_url      TEXT,
    document_id     UUID NOT NULL REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_gazette_date ON legislation.gazette_notices(publication_date DESC);

CREATE TABLE legislation.public_participation (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_id         UUID NOT NULL REFERENCES legislation.bills(id) ON DELETE CASCADE,
    committee_id    UUID REFERENCES legislation.committees(id),
    title           TEXT NOT NULL,
    deadline        DATE,
    status          TEXT NOT NULL DEFAULT 'open',
    official_instructions TEXT,
    official_channel     TEXT,
    source_url      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
