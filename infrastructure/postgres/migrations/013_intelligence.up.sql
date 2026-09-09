-- 013_intelligence.up.sql
CREATE TABLE intelligence.summaries (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_id         UUID NOT NULL REFERENCES legislation.bills(id) ON DELETE CASCADE,
    short_title     TEXT,
    one_line_summary TEXT,
    plain_language_explanation TEXT,
    current_stage_explained TEXT,
    what_it_would_do    JSONB NOT NULL DEFAULT '[]'::jsonb,
    who_it_affects      JSONB NOT NULL DEFAULT '[]'::jsonb,
    what_happens_next   JSONB NOT NULL DEFAULT '[]'::jsonb,
    confidence      TEXT NOT NULL DEFAULT 'unknown',
    validated       BOOLEAN NOT NULL DEFAULT FALSE,
    validation_failures JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_summaries_bill ON intelligence.summaries(bill_id, created_at DESC);

CREATE TABLE intelligence.questions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID REFERENCES identity.users(id),
    text            TEXT NOT NULL,
    bill_id         UUID REFERENCES legislation.bills(id),
    country         CHAR(2) NOT NULL DEFAULT 'KE',
    impact_lens     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE intelligence.answers (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    question_id     UUID NOT NULL REFERENCES intelligence.questions(id) ON DELETE CASCADE,
    answer          TEXT NOT NULL,
    claims          JSONB NOT NULL DEFAULT '[]'::jsonb,
    citations       JSONB NOT NULL DEFAULT '[]'::jsonb,
    validated       BOOLEAN NOT NULL DEFAULT FALSE,
    validation_status TEXT NOT NULL DEFAULT 'pending',
    validation_failures JSONB NOT NULL DEFAULT '[]'::jsonb,
    model           TEXT NOT NULL,
    provider        TEXT NOT NULL,
    prompt_tokens   INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    latency_ms      INT NOT NULL DEFAULT 0,
    cost_cents      NUMERIC(10,4) NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_answers_question ON intelligence.answers(question_id);

CREATE TABLE intelligence.candidate_facts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_id         UUID REFERENCES legislation.bills(id),
    payload         JSONB NOT NULL,
    evidence        JSONB NOT NULL DEFAULT '[]'::jsonb,
    proposed_by     TEXT NOT NULL,
    proposed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    validated       BOOLEAN NOT NULL DEFAULT FALSE,
    validation_evidence JSONB,
    accepted_at     TIMESTAMPTZ,
    CHECK ((validated = FALSE) = (accepted_at IS NULL))
);
CREATE INDEX idx_candidate_facts_bill ON intelligence.candidate_facts(bill_id, validated);

CREATE TABLE intelligence.topics (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code            TEXT NOT NULL UNIQUE,
    label           TEXT NOT NULL
);

CREATE TABLE intelligence.classifications (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type     TEXT NOT NULL,
    entity_id       UUID NOT NULL,
    topic_id        UUID NOT NULL REFERENCES intelligence.topics(id),
    confidence      REAL NOT NULL,
    classified_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE intelligence.impact_analyses (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bill_id         UUID NOT NULL REFERENCES legislation.bills(id) ON DELETE CASCADE,
    lens            TEXT NOT NULL,
    summary         TEXT NOT NULL,
    strength        TEXT NOT NULL,
    citations       JSONB NOT NULL DEFAULT '[]'::jsonb,
    caveats         JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE intelligence.ai_responses (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id      UUID NOT NULL,
    capability      TEXT NOT NULL,
    model           TEXT NOT NULL,
    provider        TEXT NOT NULL,
    prompt_tokens   INT NOT NULL DEFAULT 0,
    completion_tokens INT NOT NULL DEFAULT 0,
    latency_ms      INT NOT NULL DEFAULT 0,
    cost_cents      NUMERIC(10,4) NOT NULL DEFAULT 0,
    validated       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ai_responses_capability ON intelligence.ai_responses(capability, created_at DESC);
