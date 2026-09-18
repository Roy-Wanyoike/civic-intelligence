-- 007_documents.up.sql
CREATE TABLE documents.documents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ingestion_doc_id UUID NOT NULL REFERENCES ingestion.documents(id) ON DELETE RESTRICT,
    title           TEXT,
    text            TEXT,
    language        TEXT DEFAULT 'en',
    parsed_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    parser_version  TEXT NOT NULL
);

CREATE TABLE documents.pages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id     UUID NOT NULL REFERENCES documents.documents(id) ON DELETE CASCADE,
    page_number     INT NOT NULL,
    text            TEXT NOT NULL,
    bbox            JSONB,
    UNIQUE (document_id, page_number)
);

CREATE TABLE documents.sections (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id     UUID NOT NULL REFERENCES documents.documents(id) ON DELETE CASCADE,
    heading         TEXT NOT NULL,
    level           INT NOT NULL,
    path            TEXT NOT NULL,
    start_offset    INT,
    end_offset      INT
);
CREATE INDEX idx_sections_doc ON documents.sections(document_id);

CREATE TABLE documents.chunks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id     UUID NOT NULL REFERENCES documents.documents(id) ON DELETE CASCADE,
    page_id         UUID REFERENCES documents.pages(id) ON DELETE CASCADE,
    section_id      UUID REFERENCES documents.sections(id) ON DELETE SET NULL,
    text            TEXT NOT NULL,
    content_hash    TEXT NOT NULL,
    offset_start    INT NOT NULL,
    offset_end      INT NOT NULL,
    embedding       vector(1536),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_chunks_doc       ON documents.chunks(document_id);
CREATE INDEX idx_chunks_section   ON documents.chunks(section_id);
CREATE INDEX idx_chunks_embedding ON documents.chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_chunks_text_trgm ON documents.chunks USING gin (text gin_trgm_ops);
CREATE INDEX idx_chunks_text_fts  ON documents.chunks USING gin (to_tsvector('english', unaccent(text)));
