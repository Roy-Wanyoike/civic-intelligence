-- 005_ingestion_sources.up.sql
CREATE TABLE ingestion.sources (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    country             CHAR(2) NOT NULL,
    institution         TEXT NOT NULL,
    authority_level     TEXT NOT NULL,
    url                 TEXT NOT NULL,
    adapter             TEXT NOT NULL,
    supported_doc_types JSONB NOT NULL DEFAULT '[]'::jsonb,
    crawl_frequency     INTERVAL NOT NULL DEFAULT '1 hour',
    health              TEXT NOT NULL DEFAULT 'unknown',
    last_success_at     TIMESTAMPTZ,
    last_failure_at     TIMESTAMPTZ,
    robots_compliance   JSONB NOT NULL DEFAULT '{}'::jsonb,
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_sources_adapter ON ingestion.sources(adapter);

CREATE TABLE ingestion.source_endpoints (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id       UUID NOT NULL REFERENCES ingestion.sources(id) ON DELETE CASCADE,
    doc_type        TEXT NOT NULL,
    url_pattern     TEXT NOT NULL,
    parser          TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (source_id, doc_type)
);

CREATE TABLE ingestion.crawl_jobs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_id       UUID NOT NULL REFERENCES ingestion.sources(id) ON DELETE CASCADE,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'pending',
    items_discovered INT NOT NULL DEFAULT 0,
    items_new       INT NOT NULL DEFAULT 0,
    items_changed   INT NOT NULL DEFAULT 0,
    error           TEXT,
    workflow_id     TEXT
);
CREATE INDEX idx_crawl_jobs_status ON ingestion.crawl_jobs(status, started_at DESC);

CREATE TABLE ingestion.fetch_jobs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    crawl_job_id    UUID REFERENCES ingestion.crawl_jobs(id) ON DELETE CASCADE,
    source_url      TEXT NOT NULL,
    content_hash    TEXT,
    storage_key     TEXT,
    mime_type       TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    fetched_at      TIMESTAMPTZ,
    error           TEXT,
    retries         INT NOT NULL DEFAULT 0
);
CREATE INDEX idx_fetch_jobs_status ON ingestion.fetch_jobs(status);
