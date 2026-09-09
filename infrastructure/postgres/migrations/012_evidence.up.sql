-- 012_evidence.up.sql
CREATE TABLE evidence.claims (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    claim_type      TEXT NOT NULL,
    claim_text      TEXT NOT NULL,
    made_by         TEXT,
    bill_id         UUID REFERENCES legislation.bills(id),
    entity_type     TEXT,
    entity_id       UUID,
    confidence      TEXT NOT NULL DEFAULT 'unknown',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_claims_entity ON evidence.claims(entity_type, entity_id);

CREATE TABLE evidence.citations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    claim_id        UUID NOT NULL REFERENCES evidence.claims(id) ON DELETE CASCADE,
    document_id     UUID NOT NULL REFERENCES ingestion.documents(id),
    page_number     INT,
    section         TEXT,
    snippet         TEXT NOT NULL,
    snippet_offset_start INT,
    snippet_offset_end   INT,
    source_url      TEXT NOT NULL,
    retrieved_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    validated       BOOLEAN NOT NULL DEFAULT FALSE,
    validated_at    TIMESTAMPTZ
);
CREATE INDEX idx_citations_claim ON evidence.citations(claim_id);

CREATE TABLE evidence.evidence_sets (
    claim_id        UUID NOT NULL REFERENCES evidence.claims(id) ON DELETE CASCADE,
    citation_id     UUID NOT NULL REFERENCES evidence.citations(id) ON DELETE CASCADE,
    weight          REAL NOT NULL DEFAULT 1.0,
    PRIMARY KEY (claim_id, citation_id)
);

CREATE TABLE evidence.source_conflicts (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    claim_id            UUID NOT NULL REFERENCES evidence.claims(id),
    source_a_id         UUID NOT NULL REFERENCES ingestion.sources(id),
    source_a_value      JSONB NOT NULL,
    source_a_retrieved_at TIMESTAMPTZ NOT NULL,
    source_b_id         UUID NOT NULL REFERENCES ingestion.sources(id),
    source_b_value      JSONB NOT NULL,
    source_b_retrieved_at TIMESTAMPTZ NOT NULL,
    resolution_strategy TEXT,
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
