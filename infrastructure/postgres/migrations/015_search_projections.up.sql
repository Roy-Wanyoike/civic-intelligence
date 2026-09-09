-- 015_search_projections.up.sql
CREATE TABLE search.bills_search (
    bill_id         UUID PRIMARY KEY REFERENCES legislation.bills(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    identifier      TEXT NOT NULL,
    year             INT,
    status          TEXT,
    current_stage   TEXT,
    topics          JSONB,
    search_text     tsvector GENERATED ALWAYS AS (
        to_tsvector('english', unaccent(
            coalesce(title, '') || ' ' || coalesce(identifier, '') || ' ' ||
            coalesce(purpose, '') || ' ' || coalesce(description, '')
        ))
    ) STORED
);
CREATE INDEX idx_bills_search_fts ON search.bills_search USING gin (search_text);

CREATE TABLE search.documents_search (
    document_id     UUID PRIMARY KEY REFERENCES documents.documents(id) ON DELETE CASCADE,
    doc_type        TEXT,
    title           TEXT,
    country         CHAR(2),
    search_text     tsvector
);
CREATE INDEX idx_documents_search_fts ON search.documents_search USING gin (search_text);

CREATE TABLE search.entities_search (
    entity_type     TEXT NOT NULL,
    entity_id       UUID NOT NULL,
    label           TEXT NOT NULL,
    PRIMARY KEY (entity_type, entity_id)
);
