-- 010_acts.up.sql
CREATE TABLE legislation.acts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    citation        TEXT NOT NULL,
    title           TEXT NOT NULL,
    assent_date     DATE,
    commencement_date DATE,
    repeals         JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_url      TEXT,
    document_id     UUID REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (country_id, citation)
);

CREATE TABLE legislation.regulations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    parent_act_id   UUID REFERENCES legislation.acts(id),
    citation        TEXT NOT NULL,
    title           TEXT NOT NULL,
    gazette_notice  TEXT,
    effective_date  DATE,
    status          TEXT NOT NULL DEFAULT 'in_force',
    source_url      TEXT,
    document_id     UUID REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE legislation.policies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    institution_id  UUID REFERENCES legislation.institutions(id),
    title           TEXT NOT NULL,
    summary         TEXT,
    published_date  DATE,
    status          TEXT NOT NULL DEFAULT 'active',
    source_url      TEXT,
    document_id     UUID REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE legislation.government_decisions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    institution_id  UUID REFERENCES legislation.institutions(id),
    title           TEXT NOT NULL,
    description     TEXT,
    decided_at      DATE,
    source_url      TEXT,
    document_id     UUID REFERENCES ingestion.documents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
