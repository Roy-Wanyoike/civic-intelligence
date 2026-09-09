# Postgres Migrations & Seed Data

This directory holds the schema, indexes, constraints, and seed data for the
Civic Intelligence Platform Postgres cluster (Postgres 16 + pgvector).

```
infrastructure/postgres/
├── migrations/    # forward-only SQL migrations (NNN_description.up/down.sql)
└── seed/          # idempotent seed data (countries, institutions, stages, terminology)
```

## Architectural rules (non-negotiable)

1. **One cluster, many schemas.** `legislation`, `ingestion`, `documents`,
   `evidence`, `intelligence`, `notifications`, `identity`, `search`, `audit`.
   Each schema is owned by exactly one service. Cross-schema writes are
   forbidden at the application layer and tracked by audit triggers.
2. **Migrations are immutable and forward-only.** Once a migration is merged
   to `main`, it is never edited. Fixes are new migrations.
3. **Bill versions are immutable.** `legislation.bill_versions` has a trigger
   that raises on any UPDATE. Re-fetches insert a new version row.
4. **AI never writes to canonical state directly.** `intelligence.candidate_facts`
   stores AI-proposed facts; the `validated` flag is the gate. A separate
   validating writer (in `services/legislation`) applies accepted candidates
   and emits audit log entries.
5. **Every source-originated entity has** `source_id`/`source_url`/`retrieved_at`/
   `content_hash` columns (see `ingestion.documents`, `legislation.bill_versions`,
   `legislation.bill_events`).
6. **Audit log is append-only.** A statement-level trigger on `audit.log_entries`
   blocks UPDATE / DELETE / TRUNCATE.

## Conventions

| Convention              | Rule                                                                                  |
| ----------------------- | ------------------------------------------------------------------------------------- |
| Primary keys            | UUID with default `uuid_generate_v4()`, EXCEPT lookup tables which use `BIGSERIAL`.   |
| Timestamps              | `TIMESTAMPTZ`, never `TIMESTAMP`. Defaults are `now()`.                              |
| Emails / identifiers    | `CITEXT`.                                                                             |
| Flexible metadata       | `JSONB` with a `COMMENT ON COLUMN` describing its shape.                            |
| Foreign keys            | Explicit `ON DELETE` chosen deliberately: `RESTRICT` for canonical entities, `CASCADE` for projections, `SET NULL` for soft links. |
| Vector embeddings       | `vector(1536)` + ivfflat cosine index.                                                |
| Fuzzy text              | `gin_trgm_ops` indexes on names/titles.                                             |
| Append-only timestamps  | BRIN indexes with `pages_per_range = 32|64`.                                          |

## File layout

```
001_extensions.up/down.sql           pgvector, pg_trgm, unaccent, citext, uuid-ossp, btree_gin
002_schemas.up/down.sql              the 9 bounded-context schemas
003_identity.up/down.sql             users, sessions, roles, permissions, preferences
004_audit.up/down.sql                audit.log_entries + reusable trigger function
005_ingestion_sources.up/down.sql    sources, endpoints, crawl_jobs, fetch_jobs
006_ingestion_documents.up/down.sql  IMMUTABLE raw documents + snapshots
007_documents.up/down.sql            parsed documents: pages, sections, chunks (vector(1536))
008_legislative_core.up/down.sql     countries, institutions, houses, committees, people, parties, counties
009_bills.up/down.sql                bills, IMMUTABLE bill_versions, stages lookup, events, amendments, clauses
010_acts.up/down.sql                 acts, regulations, policies, government_decisions
011_hansard_proceedings.up/down.sql  hansard, order papers, committee reports, votes, gazette notices, public participation
012_evidence.up/down.sql             claims, evidence docs, citations, evidence sets, source conflicts
013_intelligence.up/down.sql         summaries, explanations, Q&A, candidate_facts (gate), ai_responses
014_notifications.up/down.sql        follows, subscriptions, notifications, delivery attempts
015_search_projections.up/down.sql  FTS/vector projections + materialized views
016_indexes.up/down.sql             cross-cutting GIN/B-tree/BRIN/trigram indexes
017_constraints.up/down.sql         cross-cutting CHECK constraints + FK on-delete policy
```

## Applying migrations

Local dev (`docker compose up postgres`) applies nothing automatically —
you run the migrator explicitly. The standard tool is `migrate` (golang-migrate):

```bash
# Apply all pending up migrations
migrate -path infrastructure/postgres/migrations \
  -database "postgres://civic:civic@localhost:5432/civic?sslmode=disable" up

# Apply seeds (idempotent — safe to re-run)
for f in infrastructure/postgres/seed/*.sql; do
  psql "$DATABASE_URL" -f "$f"
done

# Roll back the last migration (DEV ONLY — see Rollback policy below)
migrate -path infrastructure/postgres/migrations -database "$DATABASE_URL" down 1
```

The `migrate.yml` GitHub Actions workflow applies migrations against staging
on every push to `main`, and requires a manual approval to deploy to prod.

## Rollback policy

**Migrations are forward-only in production.** The `.down.sql` files exist
only as a courtesy for local development and emergency teardowns; they are
**never** run against staging or prod automatically.

If a migration is broken:
1. Write a NEW forward migration that repairs the schema/data.
2. If a column was added wrongly, drop it in a new migration.
3. If data was inserted wrongly, write a corrective `UPDATE`/`DELETE` migration.

To rollback locally for development:

```bash
migrate -path infrastructure/postgres/migrations -database "$DATABASE_URL" down 1
```

## Branching model for migrations

* Migrations are numbered `NNN_description.up.sql` with three-digit zero-padded prefixes.
* When two PRs collide on the same number, rebase the later PR and renumber —
  never edit a merged migration.
* The CI `migrate` job runs migrations against a fresh throwaway database
  for every PR; if your migration fails to apply, the PR cannot merge.
* Migration filenames must be `kebab-case` or `snake_case` and must not contain
  spaces.

## Seed data

```
seed/001_countries.sql           KE, UG, TZ, GH, NG, ZA (only KE active)
seed/002_kenya_institutions.sql  Parliament of Kenya + bicameral houses + AG + Judiciary + 6 commissions + 47 counties
seed/003_kenya_stages.sql        14-stage Kenya bill lifecycle (DRAFTED → COMMENCEMENT)
seed/004_kenya_terminology.sql   30 Kenyan parliamentary terms (en + 1 sw)
```

Seeds are idempotent (`ON CONFLICT DO UPDATE`) and may be re-run after
migrations. They MUST NOT depend on transactional state — each statement
must succeed independently.

## Validation

* `psql -f infrastructure/postgres/migrations/*.up.sql` against a fresh
  Postgres 16 + pgvector container is the smoke test.
* The CI workflow in `.github/workflows/migrate.yml` applies migrations
  against a throwaway container before allowing merge.
* `EXPLAIN ANALYZE` the most expensive queries; tune ivfflat `lists` as
  the chunk table grows (rebuild with `REINDEX` in a maintenance migration).
