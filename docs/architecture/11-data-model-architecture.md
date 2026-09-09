# 11 — Data Model Architecture

This document describes the platform's data architecture: a single PostgreSQL cluster with logical schemas per bounded context, augmented with pgvector for vector search, JSONB for semi-structured data, and built-in full-text search for keyword queries. The decision to start with a single cluster (rather than one database per service) is documented in [ADR-0003](../adr/ADR-0003-postgres-single-cluster-logical-schemas.md); the decision to use pgvector before adding OpenSearch is in [ADR-0009](../adr/ADR-0009-pgvector-before-opensearch.md).

The data architecture serves the boundary rule: each service owns its schema, has a Postgres role scoped to its schema, and cannot write to another service's schema. The cluster is shared; the schemas are isolated. This gives us the operational simplicity of one cluster to back up, monitor, and upgrade, with the boundary discipline of separate databases.

## Single cluster, logical schemas

The platform runs one PostgreSQL cluster (a primary plus read replicas, with PgBouncer in front for connection pooling) per environment. Within that cluster, each bounded context has its own schema:

| Schema | Owned by | Contains |
| --- | --- | --- |
| `legislation` | `services/legislation/` | Country, Institution, Bill, BillVersion, BillStage, BillEvent, Amendment, Clause, Act, Regulation, Policy, HansardDocument, OrderPaper, CommitteeReport, Vote, VotesProceeding, GazetteNotice, PublicParticipation |
| `ingestion` | `services/ingestion/` | Source, SourceDocument, SourceSnapshot, CrawlJob, FetchLog |
| `documents` | `services/documents/` | Document, DocumentPage, DocumentSection, DocumentChunk, ParseLog |
| `evidence` | `services/evidence/` | Claim, Evidence, Citation, SourceConflict, CandidateFact (read-only mirror for validation) |
| `intelligence` | `services/intelligence/` + `services/ai/` | AIResponse, AIClaim, CandidateFact (write), PromptTemplate, EvalRun, EvalResult |
| `notifications` | `services/notifications/` | Follow, Notification, DeliveryAttempt, DigestSchedule |
| `identity` | `services/identity/` | User, UserPreference, APIKey, RoleBinding |
| `audit` | cross-cutting | AuditLog (append-only) |

Each schema is owned by a dedicated Postgres role (`legislation_owner`, `ingestion_owner`, etc.) that has full DDL and DML on its schema. Each service has its own runtime role (`legislation_app`, `ingestion_app`, etc.) that has DML but not DDL on its schema, and `SELECT` on the schemas it needs to read.

The AI runtime role (`intelligence_app`) is the special case: it has `SELECT` on `legislation.*` and `evidence.*`, and `INSERT` only on `intelligence.candidate_facts` (with a check constraint enforcing `status = 'proposed'`) and `intelligence.ai_responses`. It has no `UPDATE` or `DELETE` on any canonical table. This is the technical enforcement of [ADR-0005](../adr/ADR-0005-ai-cannot-mutate-truth.md).

## pgvector, JSONB, and full-text search

The cluster has three extensions installed and managed via migrations:

- **`pgvector`** for vector columns. Document chunks have an `embedding vector(1536)` column (OpenAI `text-embedding-3-small` default; configurable per capability). Queries use the `<=>` (cosine distance) operator with an IVFFlat or HNSW index depending on scale. pgvector lets us keep vectors in the same database as the entities they describe, which simplifies the RAG retrieval pipeline (no separate vector store to keep in sync).
- **JSONB** for semi-structured data. Bill metadata (the country-specific extras that don't fit the canonical schema), AI response metadata (model version, prompt version, token counts), and source fetch headers are stored as JSONB columns. JSONB gives us schema flexibility for fields that vary by country or by provider without proliferating columns.
- **`pg_trgm`** and built-in **full-text search** for keyword queries. Bills and Acts have a `tsvector` column maintained by a trigger, indexed with GIN. Keyword search uses `ts_rank_cd` for ranking. This is the BM25-equivalent in our hybrid search pipeline (see [`06-ai-gateway-rag.md`](./06-ai-gateway-rag.md)); it is sufficient for the platform's scale through Phase 5 and is augmented by OpenSearch only when query volume justifies it.

## Migration strategy

Migrations are SQL files in `infrastructure/postgres/migrations/`, named `YYYYMMDDHHMMSS_<slug>.up.sql` and `YYYYMMDDHHMMSS_<slug>.down.sql`. They are applied by `migrate` (the golang-migrate CLI) at service startup and in CI. The rules:

1. **Forward-only.** Migrations are applied in order; `down` migrations exist for development convenience but are never run in staging or production. A botched migration in production is fixed by a new forward migration, not by rolling back.
2. **Idempotent where possible.** A migration that adds a column should check if the column exists before adding it (via `IF NOT EXISTS`). This makes re-running migrations safe and makes failed runs recoverable.
3. **Backward-compatible.** A migration that renames a column does it in two steps: add the new column, dual-write to both, backfill the new column from the old, switch readers to the new column, drop the old column in a later migration. Each step is a separate migration; each step is independently deployable. This lets us deploy code that uses the new column before the old column is dropped, and roll back the code without rolling back the data.
4. **No destructive operations in the same release as the code that depends on the old shape.** Dropping a column is a multi-PR, multi-release operation: deploy code that stops reading the column; deploy the migration that drops the column. Reversing them risks a deploy where the code expects the old column and the database has dropped it.
5. **Reviewed by the data team.** Every migration PR is reviewed by a maintainer with database experience. Performance-impacting migrations (adding an index to a large table, restructuring a wide table) are benchmarked against a production-sized dataset before merge.

The migration tooling is the same in dev, staging, and production; the only difference is the target cluster. Local dev applies migrations on `docker compose up`; CI applies them before the integration tests; production applies them as part of the deploy pipeline, with a pre-deploy dry run that reports the operations and the estimated duration.

## Versioning strategy for legislative data

Legislative data is versioned immutably. The core invariant is in [`BillVersion`](./03-domain-model.md#billversion): once a `BillVersion` is published, it never changes. This invariant is enforced at the schema level:

- The `legislation.bill_versions` table has a trigger that prevents `UPDATE` and `DELETE` on any row with `status = 'published'`. Draft versions can be updated (and are, during the bill's drafting); published versions are immutable.
- Each `BillVersion` has a `content_hash` (the hash of its canonical serialized form) that is computed at publish time and stored. Re-publishing the same content produces the same hash, which is how the platform detects that a "new" version is actually the same as the prior one.
- Amendments reference a specific `BillVersion` and a specific `Clause` within that version; the reference is immutable. An amendment to version 3 of bill X cannot silently become an amendment to version 4.
- AI explanations and citations reference specific `BillVersion`s; if a new version is published, the AI must regenerate its explanation against the new version (and the old explanation is retained for the old version). This is why the "what the bill said on day X" query is trivial: it just reads version X.

The immutability invariant extends to other legislative entities: `Act` versions, `Regulation` versions, `HansardDocument` versions. Anything that is a published record of civic truth at a point in time is immutable. Things that are not records of civic truth (an `AIExplanation`, a `Follow`, a `UserPreference`) are mutable.

See [ADR-0011](../adr/ADR-0011-immutable-bill-versions.md) for the rationale.

## Read replicas and read/write split

The cluster has read replicas that lag the primary by seconds. Read-heavy services (search, API GET endpoints, AI retrieval) read from replicas; write services write to the primary. The split is configured at the connection-pool level (PgBouncer routes based on the query type or the connection's read/write hint).

The trade-off is eventual consistency: a citizen who follows a bill and gets a notification, then immediately loads the bill page, may briefly see the pre-update state (the notification was generated from the primary; the page read from a replica that hasn't replicated yet). For most user flows this is invisible (the replication lag is under a second); for flows where it matters (an editor triaging a conflict needs to see the latest state), the read goes to the primary.

## Backup and recovery

The cluster is backed up continuously via WAL archiving to object storage, with daily base backups. Point-in-time recovery is available to any second within the retention window (30 days for production, 7 days for staging). The recovery procedure is tested quarterly by restoring to a staging cluster and running the e2e suite against it.

The audit log is replicated to a separate, access-gated object-storage bucket with object lock (WORM) for compliance retention beyond the operational cluster's retention. The audit log survives even if the cluster is destroyed.

## When we outgrow the single cluster

The single-cluster decision is revisable. The triggers to revisit it:

- **Write throughput.** If the primary cannot keep up with the write rate (estimated: 1000 writes/second sustained is the rough threshold where vertical scaling stops being cost-effective), we split write-heavy schemas to their own clusters.
- **Isolation requirements.** If a schema needs independent backup/restore cadence (e.g. the audit log needs longer retention than the operational data), it moves to its own cluster.
- **Blast radius.** If a schema's misconfiguration (a runaway query, a bloated index) regularly impacts other schemas, it moves to its own cluster.

The first likely split is `intelligence` (high write rate from AI calls, disposable if needed — AI responses can be regenerated) and `documents` (high storage growth from parsed content). The `legislation` schema stays on the primary cluster indefinitely because it is the canonical truth and the most-read schema.

The split, when it happens, is a schema relocation with a clear migration path — the service boundary is already in place (separate schema, separate role, separate service), so the operational split is a connection-string change plus a data migration. The architecture is designed for this split; the default is just to not do it prematurely.

## Data retention

Retention rules per schema:

| Schema | Retention | Notes |
| --- | --- | --- |
| `legislation` | forever (canonical) | Versioned immutably; no deletes. |
| `ingestion` | SourceDocuments 90 days raw, snapshots 1 year | Re-parsing needs the raw; long-term needs the snapshot. |
| `documents` | parsed content forever; embeddings versioned | Embeddings may be regenerated when models change. |
| `evidence` | forever | Claims and citations are canonical once accepted. |
| `intelligence` | AIResponse 90 days active, 1 year cold | Regenerable; not canonical. |
| `notifications` | delivery log 30 days; Follows forever | Follows are user data; delivery is operational. |
| `identity` | user data forever; API key hashes rotated per policy | PII; access-gated. |
| `audit` | 7 years (compliance) | WORM in object storage. |

Retention is enforced by scheduled jobs that move expired data to cold storage (object storage with lifecycle policies) and delete from the operational cluster. Deletes are logged to the audit table.
