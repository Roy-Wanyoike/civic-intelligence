# ADR-0009: pgvector before OpenSearch

## Status

Accepted — 2026-09-09

## Context

The platform needs both keyword (FTS) and semantic (vector) search. Options:

1. **PostgreSQL FTS + pgvector** — single datastore, no operational overhead, ACID.
2. **OpenSearch from day one** — purpose-built, but extra cluster to operate.
3. **Postgres now, OpenSearch later** — start simple, add when justified.

## Decision

Adopt **option 3**: start with PostgreSQL FTS + pgvector. Introduce OpenSearch only when corpus size or search requirements justify it.

- The `documents.chunks` table has both a GIN trigram index for fuzzy keyword search and an `ivfflat` index on the `embedding` column (pgvector) for semantic search.
- The `search` schema holds derived projections (rebuilt from canonical data) — these can be moved to OpenSearch later without changing the application code.
- OpenSearch is included in `docker-compose.yml` for local development so the migration path is tested.

## Consequences

- **Positive**: one fewer cluster to operate in Phase 1 — Postgres is already required.
- **Positive**: ACID transactions across canonical data + search indexes (when needed).
- **Positive**: pgvector is sufficient for our scale (estimated ~100k–1M chunks in Phase 1–3).
- **Positive**: the `search` schema is explicitly a projection — if OpenSearch is introduced later, the projection logic moves, but the canonical data doesn't.
- **Negative**: at very large corpus sizes (>10M chunks), pgvector's ivfflat index will need tuning or replacement. Mitigated by monitoring `search_latency` and having OpenSearch pre-configured in the dev stack.

## When to migrate to OpenSearch

Trigger the migration when ANY of these is true:
- `search_latency` p95 > 500ms for sustained periods.
- Corpus > 5M chunks.
- Need for faceted search, complex aggregations, or relevance tuning beyond Postgres FTS.
- Multi-tenant search isolation requirements.

## References

- ARCHITECTURE.md §9 (Database strategy)
- `infrastructure/postgres/migrations/007_documents.up.sql` (ivfflat + GIN trigram + FTS indexes)
- `infrastructure/docker/docker-compose.yml` (OpenSearch included for dev)
