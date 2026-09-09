-- 008_legislative_core.up.sql
CREATE TABLE legislation.countries (
    iso_code    CHAR(2) PRIMARY KEY,
    name        TEXT NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE legislation.institutions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    name            TEXT NOT NULL,
    type            TEXT NOT NULL,
    parent_id       UUID REFERENCES legislation.institutions(id),
    jurisdiction    TEXT,
    official_sources JSONB NOT NULL DEFAULT '[]'::jsonb,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_institutions_country ON legislation.institutions(country_id);

CREATE TABLE legislation.legislatures (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    institution_id  UUID NOT NULL REFERENCES legislation.institutions(id),
    name            TEXT NOT NULL,
    active          BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE legislation.houses (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    legislature_id  UUID NOT NULL REFERENCES legislation.legislatures(id),
    name            TEXT NOT NULL,
    sort_order      INT NOT NULL DEFAULT 0,
    active          BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE legislation.committees (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    house_id        UUID NOT NULL REFERENCES legislation.houses(id),
    name            TEXT NOT NULL,
    type            TEXT,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE legislation.people (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    name            TEXT NOT NULL,
    primary_role    TEXT,
    institution_id  UUID REFERENCES legislation.institutions(id),
    party_id        UUID,
    constituency_id UUID,
    county_id       UUID,
    start_date      DATE,
    end_date        DATE,
    official_url    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_people_country ON legislation.people(country_id);

CREATE TABLE legislation.political_parties (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    name            TEXT NOT NULL,
    abbreviation    TEXT,
    official_url    TEXT,
    active          BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE legislation.constituencies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    name            TEXT NOT NULL,
    county_id       UUID
);

CREATE TABLE legislation.counties (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country_id      CHAR(2) NOT NULL REFERENCES legislation.countries(iso_code),
    name            TEXT NOT NULL,
    code            TEXT NOT NULL
);
