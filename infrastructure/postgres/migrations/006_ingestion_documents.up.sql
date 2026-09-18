-- 006_ingestion_documents.up.sql
CREATE TABLE ingestion.documents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id       UUID NOT NULL REFERENCES ingestion.sources(id),
    source_url      TEXT NOT NULL,
    retrieval_ts    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    content_hash    TEXT NOT NULL,
    mime_type       TEXT NOT NULL,
    doc_type         TEXT NOT NULL,
    country         CHAR(2) NOT NULL,
    institution     TEXT,
    publication_date DATE,
    parser_version  TEXT NOT NULL DEFAULT 'v1',
    extraction_status TEXT NOT NULL DEFAULT 'pending',
    raw_storage_key TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_url, content_hash)
);
CREATE INDEX idx_documents_country_type ON ingestion.documents(country, doc_type, publication_date DESC);
CREATE INDEX idx_documents_hash ON ingestion.documents(content_hash);

CREATE TABLE ingestion.document_snapshots (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id     UUID NOT NULL REFERENCES ingestion.documents(id) ON DELETE RESTRICT,
    snapshot_ts     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    storage_key     TEXT NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb
);
