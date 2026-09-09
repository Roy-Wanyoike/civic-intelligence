# ADR-0003: Single Postgres cluster with logical schemas

## Status

Accepted — 2026-09-09

## Context

The 9 bounded contexts need clear data ownership. Options:

1. **Database-per-service** (textbook microservices) — 9 Postgres clusters.
2. **Single cluster, schema-per-service** — one cluster, 9 schemas.
3. **Single cluster, single schema** — one big schema with prefix conventions.

## Decision

Adopt **option 2: single Postgres cluster, schema-per-bounded-context**.

Schemas: `legislation`, `ingestion`, `documents`, `evidence`, `intelligence`, `notifications`, `identity`, `search`, `audit`.

Cross-schema foreign keys are allowed (e.g., `evidence.citations.document_id` references `ingestion.documents(id)`) but only between explicitly-owned relations. No cross-service direct table writes — services use their own schema's tables.

## Consequences

- **Positive**: operational simplicity — one cluster to back up, monitor, upgrade.
- **Positive**: transactional integrity across schemas when needed (e.g., creating a Bill + its first version + its first event atomically).
- **Positive**: future split is straightforward — extract a schema to its own cluster via logical replication without changing application code (the application already treats each schema as a bounded context).
- **Negative**: a single misbehaving query can affect all services — mitigated by per-role statement timeouts, connection pool sizing, and slow-query logging.
- **Negative**: migration ordering matters across schemas — mitigated by sequential migration numbering + CI test that applies all migrations against a real Postgres container.

## References

- ARCHITECTURE.md §9 (Database strategy)
- `infrastructure/postgres/migrations/002_schemas.up.sql`
