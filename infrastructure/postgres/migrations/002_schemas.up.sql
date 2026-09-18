-- 002_schemas.up.sql
-- One logical schema per bounded context.

CREATE SCHEMA IF NOT EXISTS legislation;
CREATE SCHEMA IF NOT EXISTS ingestion;
CREATE SCHEMA IF NOT EXISTS documents;
CREATE SCHEMA IF NOT EXISTS evidence;
CREATE SCHEMA IF NOT EXISTS intelligence;
CREATE SCHEMA IF NOT EXISTS notifications;
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS search;
CREATE SCHEMA IF NOT EXISTS audit;

COMMENT ON SCHEMA legislation   IS 'Canonical civic entities — Bills, Acts, committees, people. The single source of civic truth.';
COMMENT ON SCHEMA ingestion     IS 'Source registry, crawl jobs, raw fetched documents.';
COMMENT ON SCHEMA documents     IS 'Parsed documents, pages, sections, chunks, embeddings.';
COMMENT ON SCHEMA evidence      IS 'Claims, citations, evidence sets, source conflicts.';
COMMENT ON SCHEMA intelligence  IS 'AI summaries, Q&A, candidate facts. AI NEVER writes to legislation.';
COMMENT ON SCHEMA notifications IS 'Follows, subscriptions, notifications.';
COMMENT ON SCHEMA identity      IS 'Users, sessions, preferences. Uses external OIDC for auth.';
COMMENT ON SCHEMA search        IS 'Search projections rebuilt from canonical data. Can be destroyed and rebuilt.';
COMMENT ON SCHEMA audit         IS 'Append-only audit log for canonical-table writes.';
