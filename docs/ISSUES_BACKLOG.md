# Issues Backlog

**Source:** Consolidated from `audit-team-{1,2,3,4,5}-report.md` (spec §71-86 process layer).
**Worktree:** `/home/z/my-project/wt-observability` (branch `fix/wave7-observability`)
**Audit HEAD audited:** `dca1c78` — reports dated 2026-09-18.
**Current HEAD at backlog creation:** `f14a325` — Waves 5-7 remediation applied since.
**Date:** 2026-09-18 (Wave-7 docs pass).
**Cross-references:** Spec §71 (issue management), §77 (no-fake-completion), §78 (completion matrix), §81 (production gate), §82 (zero-pending-work).

---

## 1. How to read this backlog

Each entry follows the format mandated by spec §71 + §82:

```
### ISSUE-<n>: <title>
- Source: audit-team-N, section M, GAP-X-N
- Severity: blocker | major | minor
- Gate: <which of the 16 production gates this blocks>
- Description: <one paragraph>
- Acceptance criteria:
  - <criterion 1>
  - <criterion 2>
- Files affected: <list>
- Suggested fix: <one paragraph>
- Status: OPEN | IN_PROGRESS | RESOLVED
```

Severity legend (mirrors the audit reports):
- **blocker** — Spec requirement unmet AND no compensating implementation exists; downstream functionality cannot ship.
- **major** — Spec requirement partially implemented; the user-facing experience is degraded or production deployment is impossible.
- **minor** — Cosmetic, schema-only, documentation, or non-critical completeness gap.

Status values:
- **OPEN** — gap still present at HEAD `f14a325`.
- **IN_PROGRESS** — fix landed in Wave 5/6/7 but verification (Go test, runtime, QA sign-off) is incomplete.
- **RESOLVED** — fix landed AND verified by code inspection; awaiting independent QA per §73.

---

## 2. Summary count

| Severity | OPEN | IN_PROGRESS | RESOLVED | Total |
|----------|------|-------------|----------|-------|
| blocker  | 14   | 8           | 17       | 39    |
| major    | 33   | 14          | 22       | 69    |
| minor    | 18   | 5           | 11       | 34    |
| **Total** | **65** | **27** | **50** | **142** |

**142 total gaps** consolidated across all five audit reports. **65 remain OPEN** at HEAD `f14a325` and must close before the production gate can flip to PASS.

Blockers-by-gate distribution:

| Gate | Open blockers |
|------|----------------|
| 1 BUILD | 0 |
| 2 UNIT TESTS | 0 |
| 3 INTEGRATION | 1 |
| 4 CONTRACT | 1 |
| 5 E2E | 0 (Playwright installed; runtime blocked on Node ≤22) |
| 6 AI EVAL | 1 |
| 7 DATA QUALITY | 1 |
| 8 SECURITY | 2 |
| 9 ACCESSIBILITY | 1 |
| 10 PERFORMANCE | 1 |
| 11 CHAOS | 0 (artefacts exist; runtime not exercised) |
| 12 DR | 2 |
| 13 OBSERVABILITY | 2 |
| 14 DOCUMENTATION | 0 |
| 15 UX REVIEW | 0 (this task ENG-G4) |
| 16 PRODUCTION WORKFLOW | 2 |
| Cross-cutting (no single gate) | 1 |

---

## 3. BLOCKERS

### ISSUE-1: Bill detail page silently fell back to `mockBills` (GAP-13-1)
- Source: audit-team-1, §13, GAP-13-1
- Severity: blocker
- Gate: 16 PRODUCTION WORKFLOW (no fake-success in production paths — §77)
- Description: The Bill detail page was `force-static` with `generateStaticParams` returning only the mock IDs, and rendered `mockBills` exclusively — never calling `/api/v1/bills/{id}`. This was a §77 violation: the page presented fictional data as if it were real.
- Acceptance criteria:
  - Bill detail page is `force-dynamic` (no static param generation).
  - Page fetches `/api/v1/bills/{id}` at request time and renders the returned Bill.
  - When the API is unreachable, the page shows a friendly error (not mock data).
- Files affected: `apps/web/src/app/bills/[id]/page.tsx`
- Suggested fix: Switched to `force-dynamic` + `revalidate=60`; `fetchBill()` calls the Go BFF; on failure returns `null` and the page renders an `ErrorState`. Timeline preview keeps a mock fallback ONLY when the API is unreachable (clearly labelled `source: 'mock'`).
- Status: RESOLVED (commit `b731bf5` — homepage redesign + bills API). Verified by code inspection; runtime test pending Playwright on Node ≤22.

### ISSUE-2: No Postgres migration for debt domain (GAP-16-3)
- Source: audit-team-1, §16, GAP-16-3
- Severity: blocker
- Gate: 16 PRODUCTION WORKFLOW (production deployment cannot persist debt data)
- Description: `BorrowingAgreement`, `BorrowingSnapshot`, and the rest of the Phase-17+ debt domain had no SQL migrations — `WireDebtRepository` always returned the in-memory implementation, so production deployment could not persist debt data.
- Acceptance criteria:
  - `infrastructure/postgres/migrations/017_government_loans_grants.up.sql` (paired with `.down.sql`) creates `legislation.government_loans` and `legislation.government_grants` with FK + CHECK constraints and `tenant_id`.
  - Repository tests run against an in-memory Postgres clone and pass.
- Files affected: `infrastructure/postgres/migrations/017_*`, `services/legislation/internal/infrastructure/memory/debt_repository.go`
- Suggested fix: Migration 017 ships the schema; the in-memory repository remains the runtime implementation but is now backed by a real migration the operator can apply. Wire a real Postgres-backed `DebtRepository` once the integration tests land (Gate 3).
- Status: IN_PROGRESS (migration 017 landed; Postgres-backed repository still TODO; tracked in audit-team-2 GAP-28-4).

### ISSUE-3: No Government Selector UI (GAP-11-1)
- Source: audit-team-1, §11, GAP-11-1
- Severity: blocker
- Gate: 15 UX REVIEW (flagship experience missing)
- Description: The spec requires a data-driven Government Selector tree (Kenya → Government → President → Term 1/Term 2) that applies across Bills, Acts, Constitution, Executive actions, Institutions, Debt, Borrowing, Research, Timelines, Search, Graphs. No such component existed.
- Acceptance criteria:
  - `<GovernmentSelector>` component renders a tree rooted at the active country.
  - Selecting a government/term updates the `GovernmentContext` and re-fetches all context-aware sections.
  - Per-section context filtering applies on Bills, Acts, Constitution, Debt, Borrowing.
- Files affected: `apps/web/src/components/government-selector.tsx`, `apps/web/src/lib/government-context.tsx`, `apps/web/src/lib/government-defaults.ts`
- Suggested fix: Wave-5 frontend pass shipped the selector + context provider + defaults table; integrated into the layout header so every page observes the selected administration.
- Status: RESOLVED (commit `3d6ceb5`).

### ISSUE-4: No country / administration / term context provider (GAP-11-2)
- Source: audit-team-1, §11, GAP-11-2
- Severity: blocker
- Gate: 15 UX REVIEW
- Description: The middleware set `X-Civic-Country` only on outbound requests; no React context exposed the selected country/administration/term to client components, so cross-section context filtering was impossible.
- Acceptance criteria:
  - `<GovernmentContext.Provider>` wraps the app shell.
  - `useGovernment()` returns `{ country, administration, term }` to any client component.
  - Outbound API calls include `X-Civic-Country`, `X-Civic-Administration`, `X-Civic-Term` headers.
- Files affected: `apps/web/src/lib/government-context.tsx`, `apps/web/src/middleware.ts`
- Suggested fix: Wave-5 frontend pass shipped the context provider; selectors call `setGovernment(...)` which persists to a cookie + localStorage; the middleware reads the cookie and forwards as headers.
- Status: RESOLVED (commit `3d6ceb5`).

### ISSUE-5: Kenya institutions seed references non-existent columns (GAP-6-4)
- Source: audit-team-1, §6, GAP-6-4
- Severity: blocker
- Gate: 1 BUILD (fresh apply of seeds would fail)
- Description: `002_kenya_institutions.sql` inserted into `legislation.houses` columns `chamber` and `metadata`, and into `legislation.legislatures` a `metadata` column — but migration 008 does NOT define those columns, and no later migration ALTERs the tables. Seed application would fail at apply time.
- Acceptance criteria:
  - `migrations/008_legislative_core.up.sql` declares `chamber`, `metadata` on `legislation.houses` and `metadata` on `legislation.legislatures`.
  - `psql -f migrations/008_*.up.sql && psql -f seed/002_kenya_institutions.sql` succeeds against a fresh database.
- Files affected: `infrastructure/postgres/migrations/008_legislative_core.up.sql`, `infrastructure/postgres/seed/002_kenya_institutions.sql`
- Suggested fix: Migration 008 was amended (Wave 5 backend) to declare the columns the seed uses; seed apply succeeds against a fresh clone.
- Status: RESOLVED (commit `0cd73a1`).

### ISSUE-6: Legislature debt view not implemented (GAP-19-1)
- Source: audit-team-2, §19, GAP-19-1
- Severity: blocker
- Gate: 16 PRODUCTION WORKFLOW (spec §19 flagship experience absent)
- Description: There was no API endpoint, repository method, domain type, or UI surface for the per-legislature / per-legislative-period debt view that the spec requires (debt at beginning/end, new borrowing, domestic/external, disbursements, repayments, debt service, outstanding obligations, borrowing register).
- Acceptance criteria:
  - `/api/v1/debt/legislatures/{id}` returns the per-legislature debt summary.
  - The `/debt` page surfaces a per-legislature breakdown table.
  - At least one HTTP-level test exercises the endpoint.
- Files affected: `services/api/cmd/public_debt.go`, `apps/web/src/app/debt/page.tsx`
- Suggested fix: Wave-5 backend pass added the legislature-debt-view handler; the /debt page renders administration + legislature summaries from the API.
- Status: RESOLVED (commit `0cd73a1`).

### ISSUE-7: Homepage first viewport missing Civic Highlights + What Changed (GAP-32-1)
- Source: audit-team-2, §32, GAP-32-1
- Severity: blocker
- Gate: 15 UX REVIEW
- Description: The first viewport exposed only the search ("Ask") + Constitution Spotlight + "Try:" suggestion chips. The spec requires the first viewport to immediately expose "Ask Civic + Civic Highlights + What Changed" — only "Ask" was present.
- Acceptance criteria:
  - First viewport renders Ask search + Civic Highlights Carousel side-by-side (desktop) or stacked (mobile).
  - What Changed feed visible above the fold on mobile and within the first scroll-pause on desktop.
  - No unrelated sections occupy the hero.
- Files affected: `apps/web/src/app/page.tsx`, `apps/web/src/components/trending-carousel.tsx`
- Suggested fix: Wave-5 frontend pass restructured the homepage: hero now contains Ask + Civic Highlights Carousel; What Changed is the second section.
- Status: RESOLVED (commit `b731bf5`).

### ISSUE-8: Civic Highlights Carousel buried below unrelated sections (GAP-32-2)
- Source: audit-team-2, §32, GAP-32-2
- Severity: blocker
- Gate: 15 UX REVIEW
- Description: The Civic Highlights carousel (closest analogue: `TrendingCarousel`) was buried four sections deep in the homepage, violating "must not be hidden below several unrelated sections".
- Acceptance criteria:
  - Civic Highlights Carousel renders within the first viewport.
  - Spec-required slide fields (source, date, explanation, evidence, verification, significance, diversity, topic relevance, institution relevance, matter relevance) are present.
- Files affected: `apps/web/src/components/trending-carousel.tsx`, `apps/web/src/app/page.tsx`
- Suggested fix: Carousel moved to the hero; component renamed `CivicHighlightsCarousel` with the spec's required slide fields; `TrendingCarousel` kept as a legacy adapter shim for the `/trending` route.
- Status: RESOLVED (commit `b731bf5`).

### ISSUE-9: TrendingCarousel is NOT the Civic Highlights Carousel (GAP-33-1)
- Source: audit-team-2, §33, GAP-33-1
- Severity: blocker
- Gate: 15 UX REVIEW
- Description: The component was a generic "Trending Bills" carousel populated from `mockBills` — NOT the spec's Civic Highlights Carousel. The required slide fields (source, date, explanation, evidence, verification, significance, diversity, topic relevance, institution relevance, matter relevance) were absent.
- Acceptance criteria:
  - `CivicHighlightsCarousel` accepts `CivicHighlight[]` (not `Bill[]`).
  - Each `CivicHighlight` carries `source`, `date`, `explanation`, `evidence_url`, `verification_status`, `significance`, `diversity_score`, `topic_relevance`, `institution_relevance`, `matter_relevance`.
  - SSR renders the first slide (no hydration flash).
- Files affected: `apps/web/src/components/trending-carousel.tsx`
- Suggested fix: Wave-5 frontend pass rewrote the carousel: server-rendered initial slide, `CivicHighlight` type with all spec fields, `TrendingCarousel` retained as a legacy-shim adapter that converts `Bill[]` to `CivicHighlight[]`.
- Status: RESOLVED (commit `b731bf5`).

### ISSUE-10: 7 of 10 spec-required homepage components absent (GAP-34-1)
- Source: audit-team-2, §34, GAP-34-1
- Severity: blocker
- Gate: 15 UX REVIEW
- Description: Of the 10 components the spec asks to audit, only 3 were on the homepage (Global Navigation, partial Ask Civic, partial Civic Brief CTA). The remaining 7 (Civic Highlights, What Changed, Followed Topics, Followed Institutions, Recent Matters, Explore Countries, Research Updates) were missing.
- Acceptance criteria:
  - Homepage renders all 10 spec-required components.
  - Each component is data-driven (no hardcoded mock arrays in the production path).
- Files affected: `apps/web/src/app/page.tsx`
- Suggested fix: Wave-5 frontend pass restructured the page into 8 ordered sections: (1) Ask + Civic Highlights, (2) What Changed, (3) Constitution Spotlight, (4) Followed civic matters, (5) Trending Bills, (6) Explore countries, (7) Research updates + Civic Brief, (8) What If? (scenarios).
- Status: IN_PROGRESS (8/10 sections ship; Followed Topics + Followed Institutions require auth-gated personalisation — currently stubbed with non-personalised fallback).

### ISSUE-11: No Command Palette (GAP-37-1, GAP-37-2)
- Source: audit-team-3, §37, GAP-37-1 + GAP-37-2
- Severity: blocker
- Gate: 15 UX REVIEW
- Description: No command palette existed — spec requires Cmd/Ctrl+K access for search, navigation, matters, Constitution, institutions, governments, research, settings, followed topics, recent items. No keyboard access, no accessible labels, no empty/error states, no mobile usability.
- Acceptance criteria:
  - `Cmd/Ctrl+K` opens the palette.
  - Palette accepts free-text + arrow-key navigation.
  - Empty state, error state, and mobile responsive variants exist.
  - All palette actions are keyboard-accessible with `aria-activedescendant` + roving tabindex.
- Files affected: `apps/web/src/components/command-palette.tsx`
- Suggested fix: Wave-5 frontend pass shipped the palette with navigation + search + recent items; integrated into the layout header.
- Status: RESOLVED (commit `3d6ceb5`).

### ISSUE-12: `/api/v1/search` returned empty results (GAP-38-1, GAP-38-2, GAP-49-1)
- Source: audit-team-3, §38 + §49, GAP-38-1, GAP-38-2, GAP-49-1
- Severity: blocker
- Gate: 16 PRODUCTION WORKFLOW
- Description: The search endpoint returned `{items: [], total: 0, note: "Search — pending issue #45"}`. No FTS query was issued against `search.bills_search` / `search.documents_search` / `search.entities_search`. No semantic/hybrid path; grep for `ts_query|tsquery|websearch_to_tsquery|plainto_tsquery|ts_rank|<=>|vector_distance` returned zero matches.
- Acceptance criteria:
  - `/api/v1/search?q=<term>` issues `websearch_to_tsquery` against `legislation.bills`, `legislation.acts`, `government.constitution_articles`.
  - Results are ranked by `ts_rank_cd` and include `headline` snippets.
  - At least one HTTP-level test exercises the endpoint with a known bill title.
- Files affected: `services/api/cmd/search.go`, `infrastructure/postgres/migrations/022_search_fts.up.sql`
- Suggested fix: Wave-6 backend pass added migration 022 (FTS columns + GIN indexes + 3 SQL search functions) and the `handleSearch` Go handler that calls them. Hybrid pgvector path remains TODO (ISSUE-46).
- Status: IN_PROGRESS (FTS path landed; semantic pgvector path still OPEN).

### ISSUE-13: Research routes don't exist (GAP-39-1, GAP-39-2)
- Source: audit-team-3, §39, GAP-39-1 + GAP-39-2
- Severity: blocker
- Gate: 15 UX REVIEW
- Description: None of the spec routes existed — `/research`, `/research/new`, `/research/[missionId]`, `/research/[missionId]/evidence`, `/timeline`, `/documents`, `/relationships`, `/contradictions`. No source discovery, document collection, evidence, claims, timelines, comparisons, legal lineage, entity mapping, contradictions, synthesis, citations, export, reproducibility, or replay capabilities in code.
- Acceptance criteria:
  - All 8 spec routes exist and render real (or clearly-labelled stub) UI.
  - At least `/research` and `/research/new` accept input and persist a Mission.
  - Mission evidence + contradictions pages render the evidence chain.
- Files affected: `apps/web/src/app/research/`, `services/ai/agents/`
- Suggested fix: Wave-6 added `services/ai/agents/` (BaseAgent + ResearchAgent + HistoricalComparator + AssumptionAuditor + ScenarioSynthesizer) as the backend foundation; the frontend `/research` page still says "coming soon" (GAP-65-1). Need full mission lifecycle UI.
- Status: IN_PROGRESS (backend agents landed; frontend routes still stubbed).

### ISSUE-14: Civic Agent Network unimplemented (GAP-40-1, GAP-40-2)
- Source: audit-team-3, §40, GAP-40-1 + GAP-40-2
- Severity: blocker
- Gate: 6 AI EVAL (no agent runtime = no AI eval surface)
- Description: Entire Civic Agent Network was unimplemented — no Agent Registry, Agent Identity, Agent Permissions, Agent Tools, Agent Runtime, Research Mission Engine, Planner, Orchestrator, Agent Memory, Collaboration, Research Artifacts, Claim Validation (as an agent), Citation Auditor, Contradiction Detection (as an agent), Reproducibility, Cost Controls, Safety Controls, Human Review, Observability, Evaluation. None of the 11 named agents existed.
- Acceptance criteria:
  - `AgentRegistry` with register/get/list/contains/default_budget_for.
  - `BaseAgent` with `Task`/`Result` envelopes, `Budget`, `Permission` flags (no WRITE_CIVIC_TRUTH).
  - At least 4 of 11 named agents shipped: Researcher, Comparator, Auditor, Synthesizer.
  - All agent `Result`s marked `ai_generated=True`.
- Files affected: `services/ai/agents/`
- Suggested fix: Wave-6 added `base.py`, `registry.py`, `researcher.py`, `comparator.py`, `auditor.py`, `synthesizer.py` (4 of 11 named agents). 23 unit tests pass.
- Status: IN_PROGRESS (4 of 11 agents; remaining 7 + Research Mission Engine + Orchestrator + Memory still TODO).

### ISSUE-15: No prompt-injection defense or per-mission caps (GAP-41-1, GAP-41-2, GAP-41-3, GAP-41-4, GAP-41-5)
- Source: audit-team-3, §41, GAP-41-1 + GAP-41-2 + GAP-41-3 + GAP-41-4 + GAP-41-5
- Severity: blocker
- Gate: 8 SECURITY
- Description: No prompt-injection defense (no input sanitization, no instruction-hierarchy enforcement, no jailbreak classifier); `RAGPipeline._user_prompt` concatenated user text + evidence verbatim. No per-mission caps on `max_duration`, `max_agents`, `max_steps`, `max_tool_calls`, `max_documents`, `max_tokens`, `max_model_cost`, `max_parallelism`, `max_depth`. No tool-abuse guard, no credential-theft defense, no data-exfiltration detector. No malicious-document sandbox (PDF/DOCX parsing in-process). No infinite-loop breaker per request.
- Acceptance criteria:
  - Input sanitization + instruction-hierarchy enforcement before LLM call.
  - Per-mission `Budget` with `max_tokens`, `max_cost_cents`, `max_steps`, `max_tool_calls` enforced pre-flight.
  - Outbound-content scanner rejects secret patterns in LLM output.
  - PDF/DOCX parsing runs in a sandboxed subprocess.
- Files affected: `services/ai/app/rag.py`, `services/ai/agents/base.py`, `services/documents/internal/infrastructure/extractors/`
- Suggested fix: Wave-6 added `Budget` to BaseAgent (max_tokens + max_cost_cents with `would_exceed()` pre-flight + `charge()` post-charge). Remaining controls (prompt-injection classifier, tool-abuse guard, sandbox, step-counter) still OPEN.
- Status: IN_PROGRESS (Budget landed; 5 of 6 controls OPEN).

### ISSUE-16: No AI-eval gating loop (GAP-42-1, GAP-42-2)
- Source: audit-team-3, §42, GAP-42-1 + GAP-42-2
- Severity: blocker
- Gate: 6 AI EVAL
- Description: No Observe→Evaluate→Detect Failure→Classify→Investigate→Improve→Test→Validate→Deploy→Monitor loop automated — eval runs on PR but does not gate merges (`go test ... || echo "::warning::tests failed"` soft-fail in CI line 71; ruff lint `|| true` line 25). Only 3 eval cases — spec requires 16 measured dimensions.
- Acceptance criteria:
  - CI fails on Go test failures (no `|| echo` soft-fail).
  - Eval suite covers all 16 dimensions: correctness, evidence coverage, citation correctness, source authority, temporal accuracy, entity resolution, timeline accuracy, retrieval quality, research completeness, uncertainty accuracy, terminology, neutrality, accessibility, latency, reliability, cost.
- Files affected: `.github/workflows/ci.yml`, `services/ai/eval/test_eval_dataset.py`
- Suggested fix: Remove the `|| echo "::warning::"` soft-fail; expand the eval dataset to cover the 16 dimensions. Blocked on a real LLM provider (StubProvider cannot exercise correctness).
- Status: OPEN.

### ISSUE-17: No developer org / API keys tables (GAP-43-1)
- Source: audit-team-3, §43, GAP-43-1
- Severity: blocker
- Gate: 16 PRODUCTION WORKFLOW
- Description: No Developer Organization / Project / Application / Credentials / API Keys / Environments / Roles / Entitlements / Usage / Webhooks endpoints — no `identity.developer_orgs` or `identity.api_keys` tables.
- Acceptance criteria:
  - Migration creates `identity.developer_orgs`, `identity.api_keys`, `identity.api_key_scopes`.
  - `/api/v1/dev/keys` (auth) issues + revokes keys.
  - OpenAPI documents the developer surface.
- Files affected: `infrastructure/postgres/migrations/`, `services/api/cmd/`, `docs/api/openapi.yaml`
- Suggested fix: New migration 023 + handler + OpenAPI section. Not yet started.
- Status: OPEN.

### ISSUE-18: documents.chunks schema/code drift (GAP-46-3)
- Source: audit-team-3, §46, GAP-46-3
- Severity: blocker
- Gate: 1 BUILD (Save() would fail at runtime)
- Description: Migration 007 defines columns `text, content_hash, offset_start, offset_end, embedding, created_at` on `documents.chunks`; the repository `Save` inserts `page_number, section_id, "offset", text, token_estimate` — columns `offset`, `token_estimate` don't exist; `content_hash`, `offset_start`, `offset_end`, `embedding`, `created_at` are never written. Either the migration or the repository is stale.
- Acceptance criteria:
  - `INSERT INTO documents.chunks (...)` in `document_repository.go` references only columns declared in migration 007.
  - All NOT NULL columns in migration 007 are populated by the Save path.
  - An integration test inserts a chunk and reads it back.
- Files affected: `services/documents/internal/infrastructure/postgres/document_repository.go`, `infrastructure/postgres/migrations/007_documents.up.sql`
- Suggested fix: Reconcile the column list — add `page_number`, `token_estimate` to the migration (with a paired down), and have `Save` populate `content_hash`, `offset_start`, `offset_end`, `embedding` (or downgrade the migration to drop those columns). Verified still OPEN at HEAD `f14a325`.
- Status: OPEN.

### ISSUE-19: No JetStream consumers anywhere (GAP-47-1, GAP-47-2)
- Source: audit-team-3, §47, GAP-47-1 + GAP-47-2
- Severity: blocker
- Gate: 13 OBSERVABILITY (events vanish into the void)
- Description: No JetStream consumers anywhere — grep for `js.Subscribe|js.AddStream|consumer.*durable|MaxDeliver|AckPolicy|dead.letter` returns only doc/ADR matches. Every published event vanishes into the void. No durable consumer config, no dead-letter stream, no replay, no consumer-recovery logic.
- Acceptance criteria:
  - At least one durable consumer exists per critical event (`bill.processed`, `bill.stage_changed`, `evidence.asserted`).
  - Dead-letter stream configured with `MaxDeliver=5`.
  - Replay tool can re-process events from a timestamp.
- Files affected: `services/ingestion/cmd/main.go`, `services/*/internal/infrastructure/nats/`
- Suggested fix: Wave-6 added the Redis + S3 + Temporal clients but no NATS JetStream consumers. Need a worker loop that subscribes to the event stream and dispatches to handlers. Verified still OPEN at HEAD `f14a325`.
- Status: OPEN.

### ISSUE-20: No Temporal workflows or workers (GAP-48-1, GAP-48-2, GAP-48-3)
- Source: audit-team-3, §48, GAP-48-1 + GAP-48-2 + GAP-48-3
- Severity: blocker
- Gate: 12 DR (cannot exercise kill-restart-recover)
- Description: No Temporal workflow implementations — ingestion, document processing, OCR, extraction, validation, embeddings, monitoring, research, fiscal processing, reconciliation, notifications, agents all unimplemented. No Temporal worker processes. The "kill workers during execution, restart them, verify workflows recover" requirement cannot be exercised because there are no workflows to recover.
- Acceptance criteria:
  - `KenyaBillSyncWorkflow` registered with a worker process.
  - 6 activities wired (Discover, Fetch, Parse, Extract, Validate, Publish).
  - Worker process `cmd/worker/main.go` runs and polls the `bill-processing` task queue.
- Files affected: `services/ingestion/internal/temporal/workflow.go`, `services/ingestion/cmd/worker/main.go`
- Suggested fix: Wave-6 added the workflow + 6 activities + worker; README documents the local runbook (`temporal server start-dev` + `go run ./cmd/worker`).
- Status: IN_PROGRESS (workflow registered; worker process compiles structurally; no Go toolchain available to verify runtime).

### ISSUE-21: No S3/MinIO client (GAP-50-1, GAP-50-2)
- Source: audit-team-3, §50, GAP-50-1 + GAP-50-2
- Severity: blocker
- Gate: 16 PRODUCTION WORKFLOW (no immutable document snapshot writes)
- Description: No S3/MinIO client code anywhere — `storage_key` was never written or read. No immutable document snapshot write path — `ingestion.document_snapshots` table was created but nothing populates it.
- Acceptance criteria:
  - `packages/storage/s3.go` wraps minio-go/v7.
  - Upload / Download / Delete / Exists / Presign / Ping methods.
  - At least 15 unit tests using a mockMinio in-memory implementation.
- Files affected: `packages/storage/s3.go`, `packages/storage/s3_test.go`
- Suggested fix: Wave-6 added the client (ObjectStorage interface + concrete `*s3Client`) + 15 tests using a hand-rolled mockMinio. Snapshot write path still needs wiring on the ingestion side.
- Status: IN_PROGRESS (client landed; ingestion-side snapshot writes still TODO).

### ISSUE-22: No Redis client (GAP-51-1, GAP-51-2, GAP-51-3)
- Source: audit-team-3, §51, GAP-51-1 + GAP-51-2 + GAP-51-3
- Severity: blocker
- Gate: 10 PERFORMANCE (no cache layer)
- Description: No Redis client in any service — the cost counter reset on every restart. No cache layer at all for API responses. No personalized cache isolation — the spec's headline rule "Never leak one user's personalized data through shared caching" was unenforceable because no caching layer existed.
- Acceptance criteria:
  - `packages/cache/redis.go` wraps go-redis/v9.
  - Get / Set (JSON) / Delete / Exists / Ping / Health methods.
  - At least 16 unit tests using miniredis.
- Files affected: `packages/cache/redis.go`, `packages/cache/redis_test.go`
- Suggested fix: Wave-6 added the client + 16 tests using miniredis (in-process; no Docker). Cache layer wiring into API handlers still TODO.
- Status: IN_PROGRESS (client landed; API cache middleware still TODO).

### ISSUE-23: End-to-end tracing NOT implemented (GAP-52-1, GAP-52-2, GAP-52-5)
- Source: audit-team-3, §52, GAP-52-1 + GAP-52-2 + GAP-52-5
- Severity: blocker
- Gate: 13 OBSERVABILITY
- Description: `packages/observability/tracer.go` was `NopTracer`/`NopSpan`; grep for `go.opentelemetry.io/otel` returns zero Go source matches. No correlation IDs propagated — `EventEnvelope.CorrelationID` was never populated by `NewEnvelope`; no `X-Request-ID` middleware; no W3C `traceparent` header propagation. Health-probe path mismatch — `templates/deployment.yaml` probes `/healthz/ready` and `/healthz/live`, but the API service registers only `/healthz` and `/readyz`.
- Acceptance criteria:
  - `OTel SDK` replaces `NopTracer` in production.
  - `X-Request-ID` middleware propagates IDs end-to-end through logs + audit.
  - K8s deployment probes match the API service's actual `/healthz` + `/readyz` paths.
- Files affected: `packages/observability/tracer.go`, `services/api/internal/middleware/request_id.go`, `infrastructure/kubernetes/helm/civic-intelligence/templates/`
- Suggested fix: Wave-5 backend pass added `request_id.go` middleware (sets/reads `X-Request-Id`, propagates via context). Tracer is still `NopTracer` (ENG-G2 owns the OTel SDK swap). Health-probe path mismatch still OPEN.
- Status: IN_PROGRESS (request-ID landed; tracer + probe-path still OPEN — tracked by ENG-G2).

### ISSUE-24: OIDC JWT signatures not verified (GAP-53-1, P0-3)
- Source: audit-team-4, §53 + audit-team-5 P0-3, GAP-53-1
- Severity: blocker
- Gate: 8 SECURITY
- Description: OIDC JWT signatures were not verified in either `DevVerifier` (by design) or `KeycloakVerifier` (by TODO at `verifier.go:126`) — every protected endpoint accepted any well-formed JWT, enabling complete auth bypass.
- Acceptance criteria:
  - `KeycloakVerifier.parseAndVerify` rejects tokens whose signature does not verify against the JWKS key matching `header.Kid`.
  - `alg=none` and non-RS256 algorithms are rejected before JWKS lookup.
  - Unknown `kid` is a hard failure (no fallback).
  - Verifier test suite covers: valid token, expired token, wrong issuer, wrong audience, tampered signature, unknown kid, alg=none, malformed token.
- Files affected: `services/api/internal/oidc/verifier.go`, `services/api/internal/oidc/verifier_test.go`
- Suggested fix: Wave-5 backend pass rewrote `parseAndVerify` to use `go-jose/v3` for proper RS256 signature verification; added claim checks (issuer, audience, expiry, iat-not-in-future).
- Status: RESOLVED (commit `f2e5973`; verified by code inspection — `verifier.go:106` now calls `tok.Verify(key)`).

### ISSUE-25: DevMode defaults to true in production config (GAP-53-2)
- Source: audit-team-4, §53, GAP-53-2
- Severity: blocker
- Gate: 8 SECURITY
- Description: `DevMode=true` is the default in `Config` (`main.go:35`); deployment guides tell operators to set `DEV_MODE=false`, but nothing in CI or startup enforces it for non-dev environments.
- Acceptance criteria:
  - `DevMode` defaults to `false` in `Config`.
  - Startup fails fast when `DevMode=true && OIDC_ISSUER is a production host && NODE_ENV=production`.
  - CI runs a smoke test with `DevMode=false` to prove the production path works.
- Files affected: `services/api/cmd/main.go`
- Suggested fix: Flip the default to `false`; add a startup guard. Wave-5 left the default as `true` (still observed at `main.go:36`).
- Status: OPEN.

### ISSUE-26: Sponsorship endpoints return fake success (GAP-53-4)
- Source: audit-team-4, §53, GAP-53-4
- Severity: blocker
- Gate: 8 SECURITY (fake success on financial endpoints)
- Description: Sponsorship payment endpoints returned hardcoded "success" without calling M-Pesa Daraja or Stripe — fake success on financial endpoints. Stripe checkout returned `cs_test_placeholder_…` URL.
- Acceptance criteria:
  - Sponsor endpoints return `501 Not Implemented` with `{ "code": "not_implemented", "message": "..." }` until Daraja/Stripe are wired.
  - No `200 OK` from any sponsor endpoint in production.
  - OpenAPI documents the `501` response.
- Files affected: `services/api/cmd/main.go`
- Suggested fix: Wave-5 backend pass changed sponsor handlers to `501` with `paymentPendingMessage`; logged warnings carry the request_id.
- Status: RESOLVED (commit `f2e5973`; verified by code inspection — `main.go:1446, 1462, 1467` return `StatusNotImplemented`).

### ISSUE-27: No LCP/INP/CLS measurements (GAP-54-1)
- Source: audit-team-4, §54, GAP-54-1
- Severity: blocker
- Gate: 10 PERFORMANCE
- Description: No LCP / INP / CLS measurements exist anywhere — spec §54 hard-fails "Do not report theoretical performance."
- Acceptance criteria:
  - Playwright + Lighthouse CI step in `.github/workflows/ci.yml` asserts LCP ≤ 2.5s on `/`, `/bills`, `/debt`.
  - Baseline CLS ≤ 0.1 and INP ≤ 200ms.
- Files affected: `.github/workflows/ci.yml`, `tests/e2e/`
- Suggested fix: Add Lighthouse CI step; blocked on Node ≤22 for Playwright runtime.
- Status: OPEN.

### ISSUE-28: No load-test scripts (GAP-55-1)
- Source: audit-team-4, §55, GAP-55-1
- Severity: blocker
- Gate: 10 PERFORMANCE
- Description: Zero load-test scripts existed for any of the 12 traffic shapes spec §55 enumerates (100 → 10 000 users, 10× spike, 1/6/12/24-hour soaks).
- Acceptance criteria:
  - `tests/load/k6/` covers at least 4 user tiers + a 1-hour soak.
  - CI nightly GitHub Actions job runs the load suite against staging.
- Files affected: `tests/load/k6-bills.js`, `tests/load/k6-search.js`, `tests/load/k6-scenarios.js`, `tests/load/README.md`
- Suggested fix: Wave-6 added 3 k6 scripts (bills, search, scenarios) with documented SLOs and a CI integration snippet. 1-hour soak not yet scheduled.
- Status: IN_PROGRESS (3 scripts landed; soak schedule + degradation thresholds OPEN — tracked by ENG-G3).

### ISSUE-29: No chaos experiments (GAP-56-1)
- Source: audit-team-4, §56, GAP-56-1
- Severity: blocker
- Gate: 11 CHAOS
- Description: No chaos experiments for the 14 failure surfaces spec §56 lists (DB, Redis, NATS, Temporal, AI, search, object storage, external source, OCR, parser, workers, API, network, DNS).
- Acceptance criteria:
  - `tests/chaos/` covers at least DB, Redis, NATS, Temporal, AI failure modes.
  - Each experiment asserts "no panic, health flips ≤1s, recovers ≤5s, no data loss, no duplicate side effects".
- Files affected: `tests/chaos/db-failure.md`, `tests/chaos/redis-down.md`, `tests/chaos/nats-outage.md`, `tests/chaos/temporal-restart.md`, `tests/chaos/ai-timeout.md`, `tests/chaos/README.md`
- Suggested fix: Wave-6 added 5 chaos runbooks + universal SLOs + cadence (monthly + quarterly game-day). Automation roadmap (chaos-mesh / litmus) not yet started.
- Status: IN_PROGRESS (5 runbooks landed; automation OPEN — tracked by ENG-G3).

### ISSUE-30: No backup procedure or restore drill (GAP-57-1, GAP-57-2)
- Source: audit-team-4, §57, GAP-57-1 + GAP-57-2
- Severity: blocker
- Gate: 12 DR
- Description: No backup procedure defined (no script, no scheduled job, no WAL archive). No restore drill — RPO / RTO not measured or documented anywhere.
- Acceptance criteria:
  - `infrastructure/postgres/backup/` ships a WAL-G sidecar config + `backup.sh` invoked by cron.
  - `docs/operations/dr-runbook.md` declares RPO 5 min / RTO 30 min + quarterly restore-drill checklist.
- Files affected: `infrastructure/postgres/backup/`, `docs/operations/dr-runbook.md`
- Suggested fix: Wave-6 chaos runbooks reference DR but no backup script or runbook exists yet.
- Status: OPEN (tracked by ENG-G3).

### ISSUE-31: No service worker / no offline navigation (GAP-60-1)
- Source: audit-team-4, §60, GAP-60-1
- Severity: blocker
- Gate: 15 UX REVIEW
- Description: No service worker — spec §60 requires offline navigation to work.
- Acceptance criteria:
  - `public/sw.js` registered from the layout.
  - Cache-first for static assets; network-first for HTML navigations with `/offline` fallback.
  - Stale-while-revalidate for `/api/v1/*` GETs.
  - Background-sync queue for POST/PUT/DELETE to `/api/*`.
- Files affected: `apps/web/public/sw.js`, `apps/web/src/components/service-worker-register.tsx`, `apps/web/src/app/offline/page.tsx`
- Suggested fix: Wave-6 added the service worker + offline page + register component + manifest expansion (192 + 512 maskable icons + 3 shortcuts).
- Status: RESOLVED (commit `e544a34`).

### ISSUE-32: Mock data in production runtime (GAP-65-2, GAP-69-1)
- Source: audit-team-4, §65 + §69, GAP-65-2 + GAP-69-1
- Severity: blocker
- Gate: 16 PRODUCTION WORKFLOW
- Description: Mock data shipped as a runtime fallback (`bills/page.tsx:27-31`) — users see "Housing Bill, 2024" placeholder data with only a small "Showing sample data" warning. This is exactly the "fake success" the spec warns against. Five endpoint stubs (`/loans`, `/grants`, `/institutions`, `/people`, `/committees`) return canned JSON.
- Acceptance criteria:
  - `bills/page.tsx` returns a real `EmptyState` when the API is unreachable — not `mockBills`.
  - `/loans`, `/grants`, `/institutions`, `/people`, `/committees` return `501 Not Implemented` (not canned JSON).
  - Mock-data import gated behind `NODE_ENV=development`.
- Files affected: `apps/web/src/app/bills/page.tsx`, `services/api/cmd/main.go`
- Suggested fix: Wave-5 partially addressed: bills detail page no longer falls back to mockBills; bills list page still falls back when API unreachable. Sponsor endpoints switched to 501. Other endpoints still return canned stubs.
- Status: IN_PROGRESS (bills detail fixed; bills list + 5 stub endpoints OPEN).

### ISSUE-33: No i18n library (GAP-66-1)
- Source: audit-team-4, §66, GAP-66-1
- Severity: blocker
- Gate: 15 UX REVIEW
- Description: No i18n library and no message-extraction step — every user-facing string was hard-coded TSX. Spec §66 says "Do not hard-code strings throughout the UI."
- Acceptance criteria:
  - `next-intl` (or equivalent) adopted.
  - Strings extracted to `apps/web/src/messages/en.json`.
  - CI lint fails on hard-coded strings in TSX.
  - `sw.json` partial Swahili catalogue exists.
  - Language switcher in the header.
- Files affected: `apps/web/src/i18n/`, `apps/web/src/components/language-switcher.tsx`, `apps/web/package.json`
- Suggested fix: Wave-5 frontend pass shipped the i18n infrastructure (`i18n/request.ts`, `locales/en.json`, `locales/sw.json`, `LanguageSwitcher` component). String-extraction lint still TODO.
- Status: IN_PROGRESS (i18n library + locale files landed; lint rule + full Swahili coverage OPEN).

### ISSUE-34: No request-ID middleware (GAP-67-1)
- Source: audit-team-4, §67, GAP-67-1
- Severity: blocker
- Gate: 13 OBSERVABILITY
- Description: No request-ID middleware — spec §67 requires "request IDs" on every API.
- Acceptance criteria:
  - `RequestIDMiddleware` sets `X-Request-Id` (or reads an incoming one) and propagates through context + logs + audit log.
  - `writeError` echoes `X-Request-Id` in every error response.
- Files affected: `services/api/internal/middleware/request_id.go`, `services/api/internal/middleware/request_id_test.go`
- Suggested fix: Wave-5 backend pass added the middleware + tests; integrated into the mux chain.
- Status: RESOLVED (commit `f2e5973`).

### ISSUE-35: 0 of 8 golden user journey tests (P0-1)
- Source: audit-team-5, P0-1, GAP-68-1
- Severity: blocker
- Gate: 5 E2E
- Description: `tests/e2e/homepage.spec.ts` only checked `<h1>` text on 8 individual pages; 0 of 8 mandated §74 journeys implemented.
- Acceptance criteria:
  - 8 spec-mandated journey specs in `tests/e2e/golden-journeys/`:
    1. Citizen explores a Bill
    2. Citizen follows an Act
    3. Researcher explores a Government
    4. Journalist verifies a Loan
    5. Citizen explores a Scenario
    6. Citizen reads the Constitution
    7. Developer uses the API
    8. Citizen asks a question
  - Each journey clicks through every page in the spec's flow and asserts on real data.
- Files affected: `tests/e2e/golden-journeys/01-citizen-explores-bill.spec.ts` through `08-citizen-asks-question.spec.ts`
- Suggested fix: Wave-5 tests pass added all 8 journeys.
- Status: RESOLVED (commit `5273398`; runtime blocked on Node ≤22 — verified structurally).

### ISSUE-36: No critical failure test (P0-2)
- Source: audit-team-5, P0-2
- Severity: blocker
- Gate: 12 DR
- Description: No worker-crash + retry-chain test exists anywhere in the repo — `rg "critical.failure|CriticalFailure"` returns 0 hits.
- Acceptance criteria:
  - 5 critical-failure specs in `tests/e2e/critical-failures/`:
    1. API unavailable
    2. Malformed response
    3. Missing bill ID
    4. Rate limit
    5. DB connection lost
  - Each spec asserts the UI degrades gracefully (no panic, friendly error, retry).
- Files affected: `tests/e2e/critical-failures/01-api-unavailable.spec.ts` through `05-db-connection-lost.spec.ts`
- Suggested fix: Wave-5 tests pass added all 5 specs.
- Status: RESOLVED (commit `5273398`; runtime blocked on Node ≤22).

### ISSUE-37: Playwright devDependencies missing (P0-4)
- Source: audit-team-5, P0-4
- Severity: blocker
- Gate: 5 E2E
- Description: `@playwright/test` + `@axe-core/playwright` were NOT in `apps/web/package.json` devDependencies — `npm install` completed without installing them, so the 2 specs in `tests/e2e/` could not run.
- Acceptance criteria:
  - `@playwright/test` + `@axe-core/playwright` + `@axe-core/cli` in devDependencies.
  - `npm install` succeeds and installs them.
  - `npx playwright install` succeeds.
- Files affected: `apps/web/package.json`, `tests/e2e/playwright.config.ts`
- Suggested fix: Wave-5 tests pass added the devDependencies + `@axe-core/cli`; `playwright.config.ts` updated.
- Status: RESOLVED (commit `5273398`; runtime blocked on Node ≤22 because `@playwright/test` 1.49 requires Node ≤22 but the audit env has Node 24).

### ISSUE-38: OIDC verifier skipped signature verification (P0-3)
- Same as ISSUE-24. Listed separately because audit-team-5 raised it as P0-3.
- Status: RESOLVED (commit `f2e5973`).

### ISSUE-39: In-memory rate limiter not Redis-backed (GAP-43-5, GAP-53-3)
- Source: audit-team-3 §43 GAP-43-5 + audit-team-4 §53 GAP-53-3
- Severity: blocker
- Gate: 8 SECURITY
- Description: Rate limit is in-memory per-pod (`services/api/internal/middleware/auth.go:143-178`) — not the 300 req/min-per-API-key contract; won't survive multi-replica deployments. Single global `RateLimit(300, time.Minute)` — no per-tier, per-user, or AI-endpoint tightening (SECURITY.md documents `Anonymous 60/min`, `Authenticated 120/min`, `Partner 600/min`, AI 10/min).
- Acceptance criteria:
  - Three-tier Redis-backed limiter (anonymous / authenticated / partner / AI) per SECURITY.md.
  - Per-API-key counter survives pod restarts.
  - AI endpoints (`/v1/questions`, `/v1/bills/{id}/summarize`) capped at 10/min.
- Files affected: `services/api/internal/middleware/auth.go`, `packages/cache/redis.go`
- Suggested fix: Wave-6 added the Redis client (ISSUE-22) but the rate-limit middleware still uses the in-memory implementation.
- Status: OPEN.

---

## 4. MAJOR

### ISSUE-40: Stub endpoints returning "pending (issue #19)" (GAP-1-1, P1-12, P1-13)
- Source: audit-team-1, §1, GAP-1-1 + audit-team-5 P1-12 + P1-13
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: Numerous stub endpoints (`/api/v1/people`, `/api/v1/committees`, `/api/v1/institutions`, `/api/v1/loans`, `/api/v1/grants`, `/api/v1/search`, `/api/v1/briefing`) returned `"pending (issue #19)"` strings. Search + Briefing have been replaced by real handlers (commit `0cd73a1`); the other 5 still return canned JSON.
- Acceptance criteria:
  - `/people`, `/committees`, `/institutions`, `/loans`, `/grants` return `501 Not Implemented` with `{ "code": "not_implemented", "message": "..." }`.
  - OpenAPI documents the `501` response on each.
- Files affected: `services/api/cmd/main.go`, `docs/api/openapi.yaml`
- Suggested fix: Convert each stub handler to `501` until a real repository backs it; remove from `services/api/README.md` claim of "all endpoints implemented".
- Status: IN_PROGRESS (Search + Briefing resolved; 5 remaining stubs OPEN).

### ISSUE-41: No `CivicMatter` shared interface (GAP-5-1, GAP-5-2, GAP-5-3)
- Source: audit-team-1, §5
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: There is NO `CivicMatter` Go interface or base struct — `Bill`, `Act`, `Regulation`, `Policy` are unrelated types with no shared shape. Spec-listed matter types with no domain representation: Court Decision / Judgment, Motion, Petition, Constitutional Instrument. `Amendment` exists but only against Bills; there is no `ActAmendment` for post-assent amendments.
- Acceptance criteria:
  - `contracts.CivicMatter` interface declares the shared shape (`ID`, `Country`, `Jurisdiction`, `Source`, `EvidenceRefs`).
  - `Bill`, `Act`, `Regulation`, `Policy`, `CourtDecision`, `Motion`, `Petition` all implement it.
  - `ActAmendment` type added for post-assent amendments.
- Files affected: `packages/contracts/country.go`, `services/legislation/internal/domain/`
- Suggested fix: Add the interface + the missing types + migrations. Not yet started.
- Status: OPEN.

### ISSUE-42: `Institution` struct missing fields (GAP-6-1, GAP-6-2, GAP-6-3, GAP-6-5)
- Source: audit-team-1, §6
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: The Go `Institution` struct is missing `Jurisdiction`, `OfficialSources`, `Responsibilities`, `HistoricalChanges`. `Institution.Type` is a free-text string with no enum constraint. No `historical_changes`/institution-history table. No `/api/v1/institutions/{id}` endpoint exposes institution detail.
- Acceptance criteria:
  - `Institution` struct carries all 5 spec fields.
  - `InstitutionType` enum enforces the closed set.
  - Migration creates `legislation.institution_history` table.
  - `/api/v1/institutions/{id}` returns the detail.
- Files affected: `services/legislation/internal/domain/civic_entities.go`, `infrastructure/postgres/migrations/`, `services/api/cmd/main.go`
- Suggested fix: Wave-5 backend pass added institutions seed drift fix; remaining struct-field + enum + history-table work OPEN.
- Status: IN_PROGRESS.

### ISSUE-43: Legislature / Motions / Petitions not modelled (GAP-7-1, GAP-7-2, GAP-7-3, GAP-7-4, GAP-7-5)
- Source: audit-team-1, §7
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: No SQL tables for Motions, Petitions, Parliamentary Questions, Sessions, Legislative Periods. No Go domain types for `Motion`, `Petition`, `ParliamentaryQuestion`, `Session`, `LegislativePeriod`. No API endpoints expose legislature/house/committee/member detail. Temporal validity (institution/person/role history) is not enforced on `legislation.people`. The web `/institutions` and `/committees` pages render hardcoded arrays — they do not call the API.
- Acceptance criteria:
  - Migration creates the 5 missing tables.
  - 5 Go domain types declared with explicit fields.
  - `/institutions` and `/committees` pages fetch from the API.
- Files affected: `infrastructure/postgres/migrations/`, `services/legislation/internal/domain/`, `apps/web/src/app/institutions/page.tsx`, `apps/web/src/app/committees/page.tsx`
- Suggested fix: Not yet started; track as a single Phase-N+1 epic.
- Status: OPEN.

### ISSUE-44: Constitution model incomplete (GAP-8-1, GAP-8-2, GAP-8-3, GAP-8-4, GAP-8-5)
- Source: audit-team-1, §8
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: The Constitution domain model has no `ConstitutionVersion` history, no `Clause` type, no `Schedule` type, no `Right`/`Duty`/`ConstitutionalPrinciple`/`ConstitutionalInstitution` types, no `ConstitutionAmendment` type. Required experiences `/constitution/chapter/[id]`, `/constitution/article/[id]`, `/constitution/search` do NOT exist. No `/api/v1/constitution/search` endpoint exists. The `ConstitutionCrossReference` table is not exposed via API. No SQL seed populates `government.constitution_chapters` / `government.constitution_articles`.
- Acceptance criteria:
  - All 6 missing types declared.
  - 3 missing web routes exist + fetch from API.
  - `/api/v1/constitution/search` issues FTS against `government.constitution_articles`.
  - Migration 022 (Wave-6 FTS) covers constitution articles.
- Files affected: `services/legislation/internal/domain/`, `apps/web/src/app/constitution/`, `infrastructure/postgres/migrations/022_search_fts.up.sql`
- Suggested fix: Wave-6 FTS migration now supports `search.search_constitution()`; remaining type + route + seed work OPEN.
- Status: IN_PROGRESS (FTS path landed; types + routes + seeds OPEN).

### ISSUE-45: Constitution Spotlight link target wrong (GAP-9-2, GAP-9-3, GAP-9-4)
- Source: audit-team-1, §9
- Severity: major
- Gate: 15 UX REVIEW
- Description: The "Read the Constitution" link points to `/about` instead of `/constitution` or `/constitution/article/[id]`. The spec says "Clicking the card should lead to the complete authoritative article/reader". No mechanism to avoid "repeated content" beyond excluding the currently-shown article. No e2e test asserts the spotlight renders.
- Acceptance criteria:
  - Link target is `/constitution/article/{id}` (or `/constitution` as fallback).
  - Daily-pick seed prevents article repetition within a session.
  - E2E test asserts the spotlight renders + click navigates.
- Files affected: `apps/web/src/components/constitution-spotlight.tsx`
- Suggested fix: Verified still OPEN at HEAD `f14a325` — `constitution-spotlight.tsx:72` still `href="/about"`.
- Status: OPEN.

### ISSUE-46: No semantic pgvector search path (GAP-49-3, GAP-49-4)
- Source: audit-team-3, §49
- Severity: major
- Gate: 5 E2E
- Description: `EvidenceStore.retrieve` is a stub — no real retrieval of evidence/semantic/temporal chunks; the RAG pipeline cannot ground responses. No ranking algorithm, no filter clauses, no evidence-retrieval API. The FTS path landed in Wave-6 but the semantic pgvector path is missing.
- Acceptance criteria:
  - `EvidenceStore.retrieve(query, k)` returns top-k chunks by cosine similarity on the `embedding` column.
  - Hybrid ranking combines `ts_rank_cd` + cosine similarity.
  - At least one benchmark query asserts p99 ≤ 500ms.
- Files affected: `services/ai/app/evidence_store.py`, `infrastructure/postgres/migrations/`
- Suggested fix: Implement the retrieval + hybrid ranker; blocked on embedding-generation (needs a real LLM provider).
- Status: OPEN.

### ISSUE-47: AI defaults to StubProvider (GAP-27-1, GAP-27-3, GAP-27-4, P1-14)
- Source: audit-team-2, §27 + audit-team-5 P1-14
- Severity: major
- Gate: 6 AI EVAL
- Description: AI defaults to `StubProvider` (`services/ai/app/config.py:model_gateway_default_provider = "stub"`); Anthropic provider falls back to stub (`gateway.py`: `anthropic_provider_not_implemented_falling_back_to_stub`). The Python `ClaimType` enum (`FACT, STAGE, DATE, ENTITY, DEFINITION, INTERPRETATION`) does not match the spec's four required categories (`FACT`, `EXPLANATION`, `INFERENCE`, `UNKNOWN`). Validation list ("citations, evidence, temporal correctness, source authority, contradictions, completeness, uncertainty") only partially enforced. The `gateway.complete` call does not enforce a "block unsupported claims" hard-stop.
- Acceptance criteria:
  - `model_gateway_default_provider = "openai"` or `"anthropic"` when API key present.
  - ClaimType enum aligned to spec's 4 categories.
  - All 7 validators enforced (citation, evidence, temporal, source authority, contradiction, completeness, uncertainty).
  - Unsupported claims hard-blocked (not just prepended warning).
- Files affected: `services/ai/app/config.py`, `services/ai/app/gateway.py`, `services/ai/app/rag.py`, `services/ai/app/domain.py`, `services/ai/app/citation_validator.py`
- Suggested fix: Wire the real OpenAI/Anthropic provider; align the ClaimType enum; expand validators; hard-block unsupported claims.
- Status: OPEN (tracked by AI council — no PR yet).

### ISSUE-48: PDFExtractor + DOCXExtractor are stubs (GAP-24-3, P1-15)
- Source: audit-team-2, §24 + audit-team-5 P1-15
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: `PDFExtractor is a stub; OCR may be needed`; `DOCXExtractor is a stub; production should use unioffice`. The pipeline exists only in unit tests.
- Acceptance criteria:
  - `PDFExtractor` uses `unidoc` or `pdfcpu` to extract text from PDFs.
  - `DOCXExtractor` uses `unioffice` to extract text from DOCX.
  - OCR fallback for scanned PDFs (`tesseract` or `aws-textract`).
- Files affected: `services/documents/internal/infrastructure/extractors/text.go`
- Suggested fix: Wire real extractors; not yet started.
- Status: OPEN.

### ISSUE-49: Ingestion service is a stub (GAP-24-1, GAP-24-2, GAP-24-4, GAP-24-5)
- Source: audit-team-2, §24
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: `services/ingestion/cmd/main.go:39-49` is a stub — registers only `/healthz` and `/readyz`. No scheduler that polls `Source.PollPeriod`, no FetchJob worker loop, no event subscription. Retries are not implemented. Source health monitoring is passive. The Temporal workflow (`services/ingestion/internal/temporal/workflow.go`) was an empty stub — fixed in Wave-6 (see ISSUE-20).
- Acceptance criteria:
  - `cmd/main.go` registers a FetchJob worker loop that polls sources.
  - Failed jobs re-queued with exponential backoff.
  - Source-health probe runs on a schedule.
- Files affected: `services/ingestion/cmd/main.go`, `services/ingestion/internal/temporal/workflow.go`
- Suggested fix: Wave-6 added the Temporal workflow + worker (ISSUE-20); the scheduler + retry + source-health probe still TODO.
- Status: IN_PROGRESS (Temporal landed; scheduler + retries OPEN).

### ISSUE-50: Evidence chain not surfaced in UI (GAP-25-1, GAP-25-2)
- Source: audit-team-2, §25
- Severity: major
- Gate: 15 UX REVIEW
- Description: The user-facing path "Answer → Claim → Evidence → Exact Passage → Source → Original Document" is not surfaced in the UI. The ask page and bill-ask-panel do not render the citation chain as a navigable trace. `services/evidence/cmd/main.go` is a stub (only `/healthz` + `/readyz`).
- Acceptance criteria:
  - `/ask` page renders the citation chain as a navigable trace.
  - `/api/v1/evidence/{id}` returns the chain.
  - `services/evidence/cmd/main.go` wires NATS + HTTP routes.
- Files affected: `apps/web/src/app/ask/page.tsx`, `apps/web/src/components/bill-ask-panel.tsx`, `services/evidence/cmd/main.go`
- Suggested fix: Surface the chain in the UI; not yet started.
- Status: OPEN.

### ISSUE-51: Provenance fields missing (GAP-26-1, GAP-26-4)
- Source: audit-team-2, §26
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: Spec-required provenance fields not stored: `extraction_method`, `transformation`, `generated explanation`. The provenance API path is `/api/v1/provenance/{entity_type}/{id}` but the frontend `/trust` page only lists sources; there is no per-claim provenance explorer.
- Acceptance criteria:
  - Migration adds `extraction_method`, `transformation`, `generated_explanation` columns to `trust.evidence`.
  - `/trust` page renders a per-claim provenance explorer.
- Files affected: `infrastructure/postgres/migrations/018_trust_schema.up.sql`, `apps/web/src/app/trust/page.tsx`
- Suggested fix: Add the columns + UI explorer; not yet started.
- Status: OPEN.

### ISSUE-52: Civic graph missing entity types + relationships table (GAP-28-1, GAP-28-2, GAP-28-3)
- Source: audit-team-2, §28
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: Several spec-required civic entity types are not modelled as domain entities: `Regulations`, `Court Decisions`, `Policies`, `Projects`, `Topics`. No general-purpose graph/edge table — only `government.constitution_cross_references`. `ConstitutionCrossReference` does not store provenance, temporal validity, or trust/confidence.
- Acceptance criteria:
  - 5 missing entity types declared + SQL tables created.
  - `civic.relationships(source_id, target_id, type, provenance, valid_from, valid_to, evidence, confidence)` table created.
- Files affected: `services/legislation/internal/domain/`, `infrastructure/postgres/migrations/`
- Suggested fix: Add the types + the relationships table; not yet started.
- Status: OPEN.

### ISSUE-53: No temporal query API (GAP-29-1, GAP-29-2)
- Source: audit-team-2, §29
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: There is no temporal query API. The `BillRepository`, `DebtRepository`, and `trust` store have no `AsOf(time)` or `valid_at` method. `legislation.bills`, `legislation.acts`, `legislation.bill_versions` do not have `valid_from`/`valid_to` columns. The platform cannot answer "what was the current_stage of Bill X on date Y".
- Acceptance criteria:
  - `AsOf(time)` method on the repositories.
  - `/api/v1/bills/{id}?as_of=YYYY-MM-DD` returns the state at that date.
  - `valid_from` / `valid_to` columns on bills + acts.
- Files affected: `services/legislation/internal/domain/repositories.go`, `infrastructure/postgres/migrations/`
- Suggested fix: Add the columns + the query method; not yet started.
- Status: OPEN.

### ISSUE-54: Proactive intelligence not implemented (GAP-30-1, GAP-30-2, GAP-30-3, GAP-30-4, GAP-30-5)
- Source: audit-team-2, §30
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: Semantic diff is not implemented. Significance evaluation is hardcoded (string-matching the bill title). Impact mapping is not implemented. "Since Last Checked" is not implemented — the `/what-changed` feed returns the 20 most recent bills regardless of when the caller last visited. The Civic Brief handler is a stub returning `{"items": []}`.
- Acceptance criteria:
  - `document_comparator.py` uses `difflib` or a purpose-built legislative diff library.
  - Significance evaluated by impact analysis (not title matching).
  - Impact mapping maps changes to affected institutions/people/topics.
  - Per-user "last checked" cursor on `/what-changed`.
  - Civic Brief handler invokes the Python `briefing_generator.py` capability.
- Files affected: `services/ai/app/capabilities/document_comparator.py`, `services/ai/app/capabilities/briefing_generator.py`, `services/api/cmd/main.go`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-55: Fiscal reconciliation detector incomplete (GAP-22-1, GAP-22-2, GAP-22-3)
- Source: audit-team-2, §22
- Severity: major
- Gate: 7 DATA QUALITY
- Description: The conflict `ConflictType` enum covers only `AMOUNT_MISMATCH`, `DATE_MISMATCH`, `CURRENCY_MISMATCH`. The spec lists 8 detection categories — duplicate loans, commitment/disbursement confusion, debt-stock discrepancies, reporting-period differences, and conflicting official sources are not modelled. There is no active detection routine. `FiscalReconciliationConflict` is never persisted via the API.
- Acceptance criteria:
  - `ConflictType` enum has all 8 categories.
  - Scheduled job scans for conflicts.
  - `/api/v1/debt/conflicts` returns the conflicts.
- Files affected: `services/legislation/internal/domain/public_debt.go`, `services/evidence/internal/application/handlers.go`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-56: Debt charts missing 6 of 8 spec charts (GAP-21-1, GAP-21-2, GAP-21-3)
- Source: audit-team-2, §21
- Severity: major
- Gate: 15 UX REVIEW
- Description: Only chart #1 (debt stock over time) and the domestic-vs-external split are implemented. Missing: new borrowing by fiscal year, debt service over time, creditor composition, instrument composition, government-period comparison, debt-to-GDP over time. The chart has no zoom, no administration/term/legislature/fiscal-year selectors, no domestic/external toggle. The chart lacks "methodology" and "observation details" labels.
- Acceptance criteria:
  - All 8 spec charts render on `/debt`.
  - All interactive controls (zoom, filter, select administration, select term, switch domestic/external, switch debt stock/new borrowing, open evidence) work.
  - Each chart ships methodology + observation-details labels.
- Files affected: `apps/web/src/components/debt-trend-chart.tsx`, `apps/web/src/app/debt/page.tsx`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-57: Treasury/PDMO/CBK not seeded as trust sources (GAP-23-1)
- Source: audit-team-2, §23
- Severity: major
- Gate: 7 DATA QUALITY
- Description: The Kenya seed does not include rows for `National Treasury`, `Public Debt Management Office`, or `Central Bank of Kenya` as distinct trust sources, even though the debt dashboard cites them as sources. Spec sections 1 (Treasury), 2 (PDMO), 3 (CBK) are the top three priority sources.
- Acceptance criteria:
  - Migration seed inserts the 3 sources with the correct `authority_level`.
  - `/api/v1/sources` returns them.
- Files affected: `infrastructure/postgres/seed/`, `adapters/kenya/kenya_seed/`
- Suggested fix: Add the seed rows; not yet started.
- Status: OPEN.

### ISSUE-58: 5 of 6 country adapters are stubs (P1-11)
- Source: audit-team-5, P1-11
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: Uganda/Tanzania/Ghana/Nigeria/South Africa adapters each had `TODO: crawl …` in `parliament.go`. Wave-6 added parliament parsers + seed data for all 5, but the crawlers themselves still need real HTTP wiring.
- Acceptance criteria:
  - Each adapter's `parliament.go` issues a real HTTP GET to the country's parliament website.
  - Each adapter has at least 10 contract tests (Tanzania has 17, Ghana 19, Nigeria 23, South Africa 20, Uganda 15).
  - Each adapter exposes seed data via `internal/*_data.go`.
- Files affected: `adapters/{uganda,tanzania,ghana,nigeria,south_africa}/parliament/parliament.go`
- Suggested fix: Wave-6 added parsers + seed + contract tests (commit `09ede35`); the actual HTTP crawl + dedup loop still TODO.
- Status: IN_PROGRESS (parsers + tests + seeds landed; HTTP crawl OPEN).

### ISSUE-59: No HTTP/event contract tests (P1-5, GAP-67-7)
- Source: audit-team-5, P1-5 + audit-team-4 §67 GAP-67-7
- Severity: major
- Gate: 4 CONTRACT
- Description: No `services/*/integration/` directories exist. `docs/architecture/10-testing.md` claims they should. Zero event-contract tests. Zero HTTP producer-consumer contract tests. `packages/contracts/` has no test files. OpenAPI spec is not validated against actual handlers.
- Acceptance criteria:
  - At least one `services/<svc>/integration/` directory per service with at least one integration test.
  - Event-contract tests in `packages/events/`.
  - HTTP producer-consumer contract tests in `packages/contracts/`.
  - `oapi-codegen` or `restlayer` contract test that hits each documented path.
- Files affected: `services/*/integration/`, `packages/events/`, `packages/contracts/`
- Suggested fix: Not yet started (tracked by ENG-G1 in parallel).
- Status: OPEN.

### ISSUE-60: CI soft-fails Go tests (P1-20)
- Source: audit-team-5, P1-20
- Severity: major
- Gate: 2 UNIT TESTS
- Description: `.github/workflows/ci.yml:71` — `go test ./... || echo "::warning::tests failed in $d"`. Go test failures do not fail CI.
- Acceptance criteria:
  - CI fails on Go test failures (no `|| echo` soft-fail).
  - TS has a Vitest or Jest configuration.
- Files affected: `.github/workflows/ci.yml`, `apps/web/package.json`
- Suggested fix: Remove the soft-fail; add Vitest config. Not yet started (tracked by ENG-G1).
- Status: OPEN.

### ISSUE-61: Only 5 of 30+ metrics implemented (P1-17)
- Source: audit-team-5, P1-17
- Severity: major
- Gate: 13 OBSERVABILITY
- Description: `packages/observability/logger.go:229-235` defines only 6 metrics; the `09-observability.md` catalog lists 30+. `MetricsMiddleware` only records a single counter + single latency histogram — no per-route RED metrics, no p50/p95/p99 buckets, no error-rate counter.
- Acceptance criteria:
  - All 30+ metrics from the catalog emitted.
  - Per-route RED metrics (rate, errors, duration).
  - p50/p95/p99 latency histograms.
- Files affected: `packages/observability/metrics.go`, `packages/observability/logger.go`
- Suggested fix: Implement the remaining metrics + per-route RED. Tracked by ENG-G2.
- Status: OPEN.

### ISSUE-62: Grafana dashboards directory mismatch (P1-18)
- Source: audit-team-5, P1-18
- Severity: major
- Gate: 13 OBSERVABILITY
- Description: Doc says `infrastructure/observability/dashboards/`; actual is `grafana-dashboards/`.
- Acceptance criteria:
  - Either rename `grafana-dashboards/` to `dashboards/`, or update the doc.
- Files affected: `infrastructure/observability/grafana-dashboards/`, `docs/architecture/09-observability.md`
- Suggested fix: One-line rename OR one-line doc update. Tracked by ENG-G2.
- Status: OPEN.

### ISSUE-63: Adversarial test suite absent (GAP-53-6)
- Source: audit-team-4, §53, GAP-53-6
- Severity: major
- Gate: 8 SECURITY
- Description: No adversarial test suite — spec §53 enumerates IDOR, XSS, CSRF, SSRF, SQL injection, path traversal, command injection, prompt injection, tool abuse, replay, webhook attacks, privilege escalation, data exfiltration. Only `TestScenariosAPI_TenantIsolation` and SSRF allowlist unit tests exist.
- Acceptance criteria:
  - `tests/security/` directory with HTTP-level adversarial tests.
  - At least 13 test cases (one per spec attack category).
- Files affected: `tests/security/`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-64: Security tooling not in CI (GAP-53-5)
- Source: audit-team-4, §53, GAP-53-5
- Severity: major
- Gate: 8 SECURITY
- Description: `SECURITY.md` lists `gitleaks`, `osv-scanner`, `cosign`, SBOM, Semgrep boundary-rule enforcer as controls; none are wired into `.github/workflows/` (only Trivy filesystem scan with `exit-code: '0'` and CodeQL).
- Acceptance criteria:
  - `gitleaks-action`, `google/osv-scanner-action`, `sigstore/cosign-installer` + SBOM step in CI.
  - Trivy step `exit-code: '1'`.
- Files affected: `.github/workflows/ci.yml`, `SECURITY.md`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-65: API latency budget not measured (GAP-54-2)
- Source: audit-team-4, §54, GAP-54-2
- Severity: major
- Gate: 10 PERFORMANCE
- Description: No API latency budget test (p50 ≤ 200ms, p95 ≤ 500ms, p99 ≤ 1s) — the dashboards are present but the metric is never recorded.
- Acceptance criteria:
  - `apiHandler` wrapped with a Prometheus histogram tagged `route,method`.
  - `go test -bench` or k6 script asserts p99 ≤ 1s for `/api/v1/bills` and `/api/v1/search`.
- Files affected: `services/api/internal/middleware/`, `tests/load/`
- Suggested fix: Wave-6 added k6 scripts (ISSUE-28); per-route histogram still TODO. Tracked by ENG-G3.
- Status: IN_PROGRESS.

### ISSUE-66: Accessibility scan coverage low (GAP-58-1, GAP-58-2, GAP-58-3, GAP-58-4)
- Source: audit-team-4, §58
- Severity: major
- Gate: 9 ACCESSIBILITY
- Description: Only 4 pages scanned; platform has 30+ pages. No keyboard-traversal test. No screen-reader test. Accessibility job is not wired into `.github/workflows/ci.yml`.
- Acceptance criteria:
  - All 30+ routes axe-scanned.
  - Keyboard traversal test in `tests/e2e/keyboard.spec.ts`.
  - VoiceOver/NVDA smoke test (e.g. `guidepup`).
  - `a11y` job in `ci.yml` running `npx playwright test tests/e2e/accessibility.spec.ts`.
- Files affected: `tests/e2e/accessibility.spec.ts`, `.github/workflows/ci.yml`
- Suggested fix: Wave-5 added 4 axe specs; route expansion + keyboard + screen-reader + CI job OPEN. Blocked on Node ≤22.
- Status: IN_PROGRESS.

### ISSUE-67: Viewport matrix not tested (GAP-59-1, GAP-59-2, GAP-59-3)
- Source: audit-team-4, §59
- Severity: major
- Gate: 9 ACCESSIBILITY
- Description: No explicit tests for the 9 viewports spec §59 enumerates (320, 375, 390, 414, 768, 1024, 1280, 1440, 1920). No overflow / horizontal-scroll assertion. No responsive check for charts, modals, carousel, research workspace.
- Acceptance criteria:
  - Playwright matrix project covering the 9 spec widths.
  - `page.evaluate(() => document.documentElement.scrollWidth)` assertion ≤ viewport width.
  - Chart-resize + drawer-overflow assertions on `/debt` and `/scenarios/[id]`.
- Files affected: `tests/e2e/playwright.config.ts`, `tests/e2e/accessibility.spec.ts`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-68: No slow-network / stale-data indicator (GAP-60-2, GAP-60-3, GAP-60-4)
- Source: audit-team-4, §60
- Severity: major
- Gate: 15 UX REVIEW
- Description: No slow-3G / unstable-connection test; no `stale-data` indicator on the UI. No global "offline" / "stale cache" banner; failure paths use the React-error boundary instead of a network-state indicator. PWA manifest is incomplete.
- Acceptance criteria:
  - Playwright `.emulateNetworkConditions({ offline: true, latency: 2000 })` test.
  - `<NetworkStatusBanner/>` component reads `navigator.onLine` + React Query's `isOfflineError`.
  - Full PWA icon set (192, 512, maskable) + `shortcuts` (Wave-6 done — verify).
- Files affected: `apps/web/src/components/`, `apps/web/public/manifest.json`
- Suggested fix: Wave-6 expanded the manifest (192 + 512 + maskable + shortcuts); network-status banner + slow-network test OPEN.
- Status: IN_PROGRESS.

### ISSUE-69: Skeleton/empty/error not consistently applied (GAP-61-1, GAP-61-2, GAP-61-3, GAP-61-4)
- Source: audit-team-4, §61
- Severity: major
- Gate: 15 UX REVIEW
- Description: `/dashboard`, `/notifications`, `/following`, `/datasets` have no skeletons. No tooltips on truncated fields. No copy-to-clipboard for source links. No pagination controls on `/bills`. No sorting UI on `/acts`, `/loans`, `/grants`. No keyboard-behavior test for interactive controls.
- Acceptance criteria:
  - Every route ships a `LoadingSkeleton` / `EmptyState` / `ErrorBoundary` triple.
  - `title` attributes on all truncated strings.
  - `<CopyButton/>` on source-link rows.
  - `?page=` + `?sort=` on list endpoints + rendered pagination controls.
- Files affected: `apps/web/src/app/{dashboard,notifications,following,datasets,bills,acts,loans,grants}/page.tsx`
- Suggested fix: Wave-5 added partial pagination on `/bills`; remaining work OPEN.
- Status: IN_PROGRESS.

### ISSUE-70: Chart accessibility weak (GAP-63-1, GAP-63-2, GAP-63-3)
- Source: audit-team-4, §63
- Severity: major
- Gate: 9 ACCESSIBILITY
- Description: Chart has no keyboard interaction — points cannot be tab-focused and arrow-navigated. No touch interaction beyond native SVG `<title>`. Only one chart in the entire platform.
- Acceptance criteria:
  - `tabIndex={0}` + `onKeyDown` handlers on data points.
  - Tap handler updates an `aria-live` region.
  - Extract chart into `apps/web/src/components/chart.tsx`; every future chart ships its tabular equivalent.
- Files affected: `apps/web/src/components/debt-trend-chart.tsx`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-71: "Coming soon" copy in production UI (GAP-65-1)
- Source: audit-team-4, §65, GAP-65-1
- Severity: major
- Gate: 15 UX REVIEW
- Description: `/research` and `/developers` contain "coming soon" copy — spec §65 forbids this in production UI.
- Acceptance criteria:
  - `/research` replaced with a concrete call-to-action ("Create a workspace" — even if it requires sign-in).
  - `/developers` either hides the page or renders a real CTA.
- Files affected: `apps/web/src/app/research/page.tsx`, `apps/web/src/app/developers/page.tsx`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-72: API pagination / sorting not implemented (GAP-67-2, GAP-67-3, GAP-67-4, GAP-67-5, GAP-67-6)
- Source: audit-team-4, §67
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: `/bills` returns the full list in one page; pagination params accepted and ignored. Sorting not implemented on any endpoint. Error response shapes are inconsistent (`{"error","message"}` vs `{"note": ...}` vs `{"code","message","detail"}`). No struct validation framework. No API-versioning policy (`Sunset` header, `Accept` content-type strategy).
- Acceptance criteria:
  - Real pagination on `/bills`, `/acts`, `/loans`, `/grants`, `/scenarios`, `/search`.
  - `sort` + `order` query params validated against an allow-list.
  - Centralised `writeError` in middleware; remove `{"note": ...}` stubs.
  - `github.com/go-playground/validator/v10` adopted.
- Files affected: `services/api/cmd/main.go`, `services/api/internal/middleware/`
- Suggested fix: Wave-5 added partial pagination on `/bills`; remaining work OPEN (tracked by ENG-G1).
- Status: IN_PROGRESS.

### ISSUE-73: Documentation drift (GAP-68-1, GAP-68-2, GAP-68-3, GAP-68-4, GAP-68-5, GAP-69-2, GAP-69-3, GAP-69-5)
- Source: audit-team-4, §68 + §69
- Severity: major
- Gate: 14 DOCUMENTATION
- Description: `README.md:9` claims "176 Go + 28 Python" tests; `MASTER_AUDIT.md` says "51 Go tests"; `services/api/README.md:324` says "130+ tests". `README.md:10` claims "Uganda ✅" but the adapter is a stub. `docs/architecture/12-deployment.md` describes Argo CD, Vault, sealed secrets, Kyverno — none implemented. Missing docs: operations guide, incident runbooks, troubleshooting, contributor guide, ingestion guide, AI guide, agent guide. `MASTER_AUDIT.md` and `phase-1-audit.md` reference stale commit `a0638e7`.
- Acceptance criteria:
  - CI-generated test-count badge (no hand-maintained numbers).
  - Uganda README badge downgraded to "📋 (in progress)" until adapter at parity.
  - Unimplemented deployment narratives moved to `docs/architecture/future/` or marked "Planned — not implemented."
  - Missing guides authored.
  - Audit docs refreshed against current HEAD `f14a325`.
- Files affected: `README.md`, `MASTER_AUDIT.md`, `docs/architecture/phase-1-audit.md`, `docs/architecture/12-deployment.md`, `services/api/README.md`
- Suggested fix: Wave-5 added `MASTER_AUDIT.md` rewrite (commit `3bda41a`); remaining doc-drift OPEN.
- Status: IN_PROGRESS.

### ISSUE-74: Branch protection + commit-message lint (GAP-70-1, GAP-70-2, GAP-70-3, GAP-70-4, GAP-70-5)
- Source: audit-team-4, §70
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: Two stale remote branches whose work was already merged. ≥13 commits pushed directly to `main` without a PR. No branch-protection rule documented or enforced (`.github/CODEOWNERS` is advisory only). PR template exists but the "Closes #N" required check is not enforced. Commit-message style is inconsistent.
- Acceptance criteria:
  - Stale remote branches deleted.
  - GitHub branch protection on `main` requiring PR review before merge.
  - `Require CODEOWNERS review` enabled.
  - `semantic-pull-request` Action + `Closes #N` required-PR-check.
  - `commitlint` Action with `@commitlint/config-conventional`.
- Files affected: `.github/workflows/`, `.github/CODEOWNERS`, `.github/PULL_REQUEST_TEMPLATE.md`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-75: File-mode drift (GAP-69-4)
- Source: audit-team-4, §69, GAP-69-4
- Severity: major
- Gate: 1 BUILD
- Description: File-mode drift (293 files at 100755) is still present in the committed tree — only `core.filemode false` masks it locally; a fresh clone on a filemode-enabled system shows the entire tree as modified.
- Acceptance criteria:
  - `git ls-files -z | xargs -0 chmod 644 && git add -A && git commit` normalizes modes.
  - CI check fails on `100755` blobs.
- Files affected: every file in the repo
- Suggested fix: One-shot chmod + commit; add a CI check. Not yet started.
- Status: OPEN.

### ISSUE-76: TS has no unit-test runner (P1-20, GAP-67-7)
- Source: audit-team-5, P1-20
- Severity: major
- Gate: 2 UNIT TESTS
- Description: TS has no unit-test runner configured (no Vitest, no Jest). TS unit tests cannot run.
- Acceptance criteria:
  - `vitest` (or `jest`) in devDependencies.
  - At least one TS unit test per critical component.
- Files affected: `apps/web/package.json`, `apps/web/vitest.config.ts`
- Suggested fix: Add Vitest config + a smoke test. Not yet started.
- Status: OPEN.

### ISSUE-77: No data-quality tests for bills/acts/constitution (P1-7 partial)
- Source: audit-team-5
- Severity: major
- Gate: 7 DATA QUALITY
- Description: Golden dataset exists for the simulation service; no data-quality tests for Bills/Acts/Constitution seed data; no schema-validation tests for the OpenAPI spec.
- Acceptance criteria:
  - Data-quality tests for Bills, Acts, Constitution seeds.
  - OpenAPI schema-validation test (spectral or oapi-codegen).
- Files affected: `services/legislation/`, `services/api/`, `docs/api/openapi.yaml`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-78: OpenAPI spec partial (GAP-43-4)
- Source: audit-team-3, §43, GAP-43-4
- Severity: major
- Gate: 14 DOCUMENTATION
- Description: OpenAPI spec is partial — `developers/page.tsx` lists `/api/v1/debt/governments/{id}`, `/api/v1/scenarios/{id}/methodology`, `/api/v1/terminology` etc. that are not in `openapi.yaml`.
- Acceptance criteria:
  - Every route registered in the Go BFF is documented in `openapi.yaml`.
  - CI lint fails on missing routes.
- Files affected: `docs/api/openapi.yaml`, `apps/web/src/app/developers/page.tsx`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-79: AI gateway endpoints not in OpenAPI (GAP-43-2)
- Source: audit-team-3, §43, GAP-43-2
- Severity: major
- Gate: 14 DOCUMENTATION
- Description: No research-API or intelligence-API surface distinct from the citizen API; the AI gateway endpoints (`/v1/bills/{id}/summarize`, `/v1/questions`) are internal-only and undocumented in OpenAPI.
- Acceptance criteria:
  - OpenAPI documents the AI gateway surface.
  - At least one HTTP-level test per documented AI endpoint.
- Files affected: `docs/api/openapi.yaml`, `services/ai/api/index.py`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-80: No SDK foundations (GAP-43-3)
- Source: audit-team-3, §43, GAP-43-3
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: No SDK foundations (no TypeScript/Python client package), no marketplace foundations, no embeddable widgets.
- Acceptance criteria:
  - `packages/sdk-ts/` and `packages/sdk-py/` ship a typed client.
  - At least one embeddable widget (e.g. Constitution Spotlight embed).
- Files affected: `packages/sdk-ts/`, `packages/sdk-py/`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-81: No country onboarding workflow (GAP-44-1, GAP-44-2, GAP-44-3)
- Source: audit-team-3, §44
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: No automated Country onboarding workflow. No Language / Currency / Timezone / Legal-System / Administrative-Region / Institution-Type / Government-Level lookup tables. No multilingual UI — i18n strings hardcoded in English (Wave-5 partially fixed).
- Acceptance criteria:
  - Lookup-table migration ships the 7 reference tables.
  - Country onboarding runbook documents the 13-step process.
  - Swahili locale catalogue at >50% coverage.
- Files affected: `infrastructure/postgres/migrations/`, `docs/operations/`
- Suggested fix: Wave-5 added i18n infra (ISSUE-33); lookup tables + onboarding runbook OPEN.
- Status: IN_PROGRESS.

### ISSUE-82: No transaction boundary / Idempotency-Key (GAP-45-1, GAP-45-2)
- Source: audit-team-3, §45
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: No transaction boundary abstraction — `document_repository.go:28-60` issues `INSERT INTO documents.documents`, then per-page + per-chunk INSERTs without wrapping them in `BEGIN ... COMMIT`; a partial save leaves orphan rows. No `Idempotency-Key` header middleware — POST endpoints rely on in-memory dedup.
- Acceptance criteria:
  - `Save` wraps multi-row inserts in a transaction.
  - `Idempotency-Key` middleware persists keys to Postgres.
- Files affected: `services/documents/internal/infrastructure/postgres/document_repository.go`, `services/api/internal/middleware/`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-83: Migrations' down rollback safety never tested (GAP-46-1, GAP-46-2, GAP-46-4)
- Source: audit-team-3, §46
- Severity: major
- Gate: 3 INTEGRATION
- Description: Only `.up.sql` migrations are applied in CI; `.down.sql` rollback safety is never tested. No orphan-record / duplicate-detection tests. No query-performance benchmarks (`EXPLAIN ANALYZE` suite).
- Acceptance criteria:
  - CI applies the latest migration, then rolls back to verify the down path.
  - Orphan-detection SQL job runs nightly.
  - `EXPLAIN ANALYZE` suite covers `/bills?status=in_progress`, `/debt`, `/search`.
- Files affected: `.github/workflows/ci.yml`, `infrastructure/postgres/migrations/`
- Suggested fix: Not yet started (tracked by ENG-G1).
- Status: OPEN.

### ISSUE-84: Event versioning + idempotency missing (GAP-47-3, GAP-47-4, GAP-47-5)
- Source: audit-team-3, §47
- Severity: major
- Gate: 4 CONTRACT
- Description: No event versioning — `EventEnvelope` has no `schema_version` field; `Payload map[string]any` provides no migration path. No idempotency key on events. `EventEnvelope.CorrelationID` was never populated (Wave-5 fixed — see ISSUE-34).
- Acceptance criteria:
  - `EventEnvelope.SchemaVersion` field added.
  - Idempotency key on every published event.
  - `CorrelationID` populated by `NewEnvelope`.
- Files affected: `packages/contracts/events.go`
- Suggested fix: Wave-5 added request-ID propagation; event versioning + idempotency OPEN.
- Status: IN_PROGRESS.

### ISSUE-85: No workflow/agent IDs in logs (GAP-52-3, GAP-52-4)
- Source: audit-team-3, §52
- Severity: major
- Gate: 13 OBSERVABILITY
- Description: No workflow IDs / event IDs / agent execution IDs surfaced in logs — Temporal didn't run, events lacked stable IDs, no agents existed. `MetricsMiddleware` only records a single counter + single latency histogram — no per-route RED metrics, no p50/p95/p99 buckets, no error-rate counter.
- Acceptance criteria:
  - Every log line carries `workflow_id`, `event_id`, `agent_execution_id` when applicable.
  - `MetricsMiddleware` records per-route RED metrics.
- Files affected: `packages/observability/logger.go`, `packages/observability/metrics.go`
- Suggested fix: Wave-6 added the agent base with correlation_id logging; per-route RED + workflow IDs OPEN. Tracked by ENG-G2.
- Status: IN_PROGRESS.

### ISSUE-86: No per-3rd-party payment integration (GAP-53-4 partial)
- Source: audit-team-4, §53
- Severity: major
- Gate: 8 SECURITY
- Description: M-Pesa Daraja API not wired (STK Push). Stripe Checkout Sessions API not wired. Sponsor endpoints return 501 (Wave-5 fix) until real integrations land.
- Acceptance criteria:
  - Daraja API wired with `consumer_key`, `consumer_secret`, `shortcode`, `passkey`, `callback_url`.
  - Stripe Checkout Sessions API wired with `secret_key`.
  - Sponsor endpoints return `200 OK` with a real checkout URL.
- Files affected: `services/api/cmd/main.go`
- Suggested fix: Not yet started; tracked by separate integration epic.
- Status: OPEN (501 placeholder shipped).

### ISSUE-87: Bills list page mock-data fallback in production (GAP-65-2 partial)
- Source: audit-team-4, §65
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: The `/bills` list page still returns `mockBills` when the API is unreachable. The bill detail page no longer does (ISSUE-1). This is inconsistent and a §77 violation if shipped as production default.
- Acceptance criteria:
  - `/bills` page returns a real `EmptyState` when the API is unreachable — not `mockBills`.
  - Mock-data import gated behind `NODE_ENV=development`.
- Files affected: `apps/web/src/app/bills/page.tsx`
- Suggested fix: Replace the catch with a real `EmptyState` and remove the `mockBills` import from the production path. Not yet done.
- Status: OPEN.

### ISSUE-88: No backup/restore runbook (GAP-57-3)
- Source: audit-team-4, §57, GAP-57-3
- Severity: major
- Gate: 12 DR
- Description: No event-replay / Temporal-recovery / search-rebuild test — spec §57 lists eight recovery surfaces; none are exercised.
- Acceptance criteria:
  - `tests/dr/` exercises Postgres PITR, NATS stream replay, OpenSearch/pgvector reindex.
  - Each test asserts RPO + RTO.
- Files affected: `tests/dr/`
- Suggested fix: Not yet started (tracked by ENG-G3).
- Status: OPEN.

### ISSUE-89: Homepage hierarchy + personalisation (GAP-34-2, GAP-34-3, GAP-34-4, GAP-34-5, GAP-34-6, GAP-34-7)
- Source: audit-team-2, §34
- Severity: major
- Gate: 15 UX REVIEW
- Description: Homepage stacks 6 sections with equal visual weight; no clear hierarchy. Followed Topics + Followed Institutions require per-user state — no auth-gated personalised section on the homepage. Recent Matters not modelled. Explore Countries not surfaced. Research Updates not surfaced. Mobile behavior not verified.
- Acceptance criteria:
  - Clear visual hierarchy with weighted sections.
  - Auth-gated personalised Followed Topics + Followed Institutions sections.
  - Mobile responsive verified across 9 viewports.
- Files affected: `apps/web/src/app/page.tsx`
- Suggested fix: Wave-5 restructured the homepage (ISSUE-7); personalisation + hierarchy polish OPEN.
- Status: IN_PROGRESS.

### ISSUE-90: Carousel SSR + a11y partial (GAP-33-2, GAP-33-3, GAP-33-4, GAP-33-5, GAP-33-6, GAP-33-7, GAP-33-8, GAP-33-9)
- Source: audit-team-2, §33
- Severity: major
- Gate: 9 ACCESSIBILITY
- Description: Server-rendered initial slide was missing — `useState(0)` + `useEffect` rendered the empty-skeleton branch on SSR. No swipe/touch handler. Reduced-motion not respected for autoplay. No focus handling. No keyboard arrow-key navigation. No lazy-loading. No caching. Small-payload requirement not addressed.
- Acceptance criteria:
  - SSR renders the first slide (no hydration flash).
  - `onTouchStart`/`onTouchMove`/`onTouchEnd` handlers.
  - `prefers-reduced-motion` respected for autoplay.
  - `aria-live` region for slide changes.
  - Arrow-key navigation.
- Files affected: `apps/web/src/components/trending-carousel.tsx`
- Suggested fix: Wave-5 rewrote the carousel (ISSUE-9); reduced-motion + arrow-key + swipe still need verification.
- Status: IN_PROGRESS.

### ISSUE-91: Civic graph edge cases (GAP-28-4, GAP-28-5)
- Source: audit-team-2, §28
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: `BorrowingAgreement` domain type is not persisted in the DB (no `legislation.borrowing_agreements` table). The graph distinction "explicit / inferred / judicial / unknown" exists only for constitution cross-references, not for general bill-act-regulation-amendment relationships.
- Acceptance criteria:
  - Migration creates `legislation.borrowing_agreements`.
  - General relationships table carries an `evidence_kind` enum.
- Files affected: `infrastructure/postgres/migrations/`, `services/legislation/internal/domain/`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-92: Searchable types limited (GAP-38-3, GAP-38-4, GAP-38-5)
- Source: audit-team-3, §38
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: Searchable types cover only `bill` + briefing items; missing Acts, Constitution, Institutions, People, Governments, Legislatures, Loans, Debt, Documents, Evidence, Topics, Research. No filters for `jurisdiction`, `institution`, `matter`, `source authority`, `document type`. No temporal search filter.
- Acceptance criteria:
  - 13 searchable types indexed.
  - Filters documented in OpenAPI.
  - Temporal search filter (`from`, `to`).
- Files affected: `services/api/cmd/search.go`, `infrastructure/postgres/migrations/022_search_fts.up.sql`
- Suggested fix: Wave-6 FTS covers bills + acts + constitution; remaining 10 types + filters OPEN.
- Status: IN_PROGRESS.

### ISSUE-93: AI eval dataset too small (GAP-42-2, P1-14)
- Source: audit-team-3, §42 + audit-team-5 P1-14
- Severity: major
- Gate: 6 AI EVAL
- Description: Only 3 eval cases — spec requires 16 measured dimensions (correctness, evidence coverage, citation correctness, source authority, temporal accuracy, entity resolution, timeline accuracy, retrieval quality, research completeness, uncertainty accuracy, terminology, neutrality, accessibility, latency, reliability, cost).
- Acceptance criteria:
  - Eval dataset covers all 16 dimensions with at least 10 cases each.
  - Eval scores reflect a real LLM provider (not StubProvider).
- Files affected: `services/ai/eval/test_eval_dataset.py`, `services/ai/eval/datasets/`
- Suggested fix: Blocked on real LLM provider (ISSUE-47).
- Status: OPEN.

### ISSUE-94: No malicious-document sandbox (GAP-41-4)
- Source: audit-team-3, §41, GAP-41-4
- Severity: major
- Gate: 8 SECURITY
- Description: No malicious-document sandbox — PDF/DOCX parsing runs in-process; not isolated.
- Acceptance criteria:
  - PDF/DOCX parsing runs in a sandboxed subprocess.
  - Timeout + memory cap enforced.
- Files affected: `services/documents/internal/infrastructure/extractors/`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-95: No tool-abuse / data-exfiltration defense (GAP-41-3)
- Source: audit-team-3, §41, GAP-41-3
- Severity: major
- Gate: 8 SECURITY
- Description: No tool-abuse guard, no credential-theft defense, no data-exfiltration detector (no outbound-content scanning, no secret-pattern matcher on LLM output).
- Acceptance criteria:
  - Outbound-content scanner rejects secret patterns (AWS keys, JWTs, credit cards).
  - Tool-call rate limiter per mission.
- Files affected: `services/ai/agents/`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-96: No infinite-loop breaker (GAP-41-5)
- Source: audit-team-3, §41, GAP-41-5
- Severity: major
- Gate: 8 SECURITY
- Description: No infinite-loop breaker per request — no step counter on `RAGPipeline.answer`.
- Acceptance criteria:
  - Hard step counter (default 25) on every RAG + agent loop.
  - Counter exceeded → `429 Too Many Requests` with `Retry-After`.
- Files affected: `services/ai/app/rag.py`, `services/ai/agents/base.py`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-97: SSRF allowlist gap (GAP-41-6)
- Source: audit-team-3, §41, GAP-41-6
- Severity: minor (per audit-team-3)
- Gate: 8 SECURITY
- Description: `isBlockedIP` blocks RFC1918 but does not block cloud-metadata IPs (169.254.169.254 — covered by link-local) or 100.64.0.0/10 carrier-grade NAT.
- Acceptance criteria:
  - SSRF allowlist blocks 169.254.169.254 explicitly.
  - Blocks 100.64.0.0/10.
- Files affected: `packages/observability/ssrf.go`
- Suggested fix: One-line addition. Not yet started.
- Status: OPEN.

### ISSUE-98: API versioning policy missing (GAP-67-6)
- Source: audit-team-4, §67, GAP-67-6
- Severity: minor
- Gate: 14 DOCUMENTATION
- Description: No API-versioning policy — only the `/v1/` URL prefix; no `Sunset` header, no changelog of breaking changes, no `Accept: application/vnd.civic.v1+json` strategy.
- Acceptance criteria:
  - Versioning policy documented in `openapi.yaml` header.
  - `Sunset` header on any deprecated route.
- Files affected: `docs/api/openapi.yaml`, `services/api/internal/middleware/`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-99: Paginated debt view missing (GAP-19-2)
- Source: audit-team-2, §19, GAP-19-2
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: The borrowing register endpoint `/api/v1/debt/loans` returns agreements but does not include the per-agreement disbursement/repayment ledger required by the spec.
- Acceptance criteria:
  - `/api/v1/debt/loans/{id}/ledger` returns the per-agreement disbursement/repayment history.
- Files affected: `services/api/cmd/public_debt.go`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-100: Debt chart missing 6 controls (GAP-21-2)
- Same as ISSUE-56. Listed separately because it spans both UX REVIEW + PERFORMANCE.
- Status: OPEN.

### ISSUE-101: Bill `bill_events` schema missing fields (GAP-12-2, GAP-12-3, GAP-12-4)
- Source: audit-team-1, §12
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: The `legislation.bill_events` SQL schema does NOT preserve `actor`, `from_stage`, `to_stage`, `institution_id`, or `evidence` — only `event_type`, `event_date`, `house`, `description`, `source_url`, `source_document_id`, `confidence`, `note`. Persisting a `BillEvent` loses those fields. No `/api/v1/bills/{id}/events` endpoint. "Other House" + "Concurrence" only modelled as MEDIATION.
- Acceptance criteria:
  - Migration alters `bill_events` to add the missing columns.
  - `/api/v1/bills/{id}/events` endpoint registered.
- Files affected: `infrastructure/postgres/migrations/009_bills.up.sql`, `services/api/cmd/main.go`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-102: Bill intelligence fields missing (GAP-13-2, GAP-13-3, GAP-13-4, GAP-13-5)
- Source: audit-team-1, §13
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: `/api/v1/bills/{id}/versions` returns hardcoded empty array. `/api/v1/bills/{id}/documents` returns hardcoded empty array. No endpoints for amendments, votes, public participation, readings, presidential assent, Act relationship, commencement, regulations, court history, evidence, related Constitution Articles, related institutions, related topics. `SponsorID` not in `billResponse`. No `session`, `institution_id`, `committee_id`, `Act relationship` field.
- Acceptance criteria:
  - All listed fields populate from the domain.
  - At least one endpoint per missing surface.
- Files affected: `services/api/cmd/main.go`, `services/legislation/internal/domain/bill.go`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-103: Post-assent audit partial (GAP-14-1, GAP-14-2, GAP-14-3, GAP-14-4)
- Source: audit-team-1, §14
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: `PresidentialAssentEvent` is defined and the SQL table exists, but no API endpoint accepts or returns it. The audit handler does not query `presidential_assent_events`. Only 2 seed post-assent events exist. No SQL seed file populates `legislation.post_assent_events` or `legislation.presidential_assent_events`.
- Acceptance criteria:
  - `POST /api/v1/acts/{id}/assent` records a new assent event.
  - Audit handler queries `presidential_assent_events`.
  - Seed data covers all 4+ acts.
- Files affected: `services/api/cmd/post_assent.go`, `infrastructure/postgres/seed/`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-104: Legislative lineage incomplete (GAP-15-1, GAP-15-2, GAP-15-3, GAP-15-4, GAP-15-5)
- Source: audit-team-1, §15
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: Lineage collapses 11 spec nodes into 9; omits "Bill Version", "Passed Version", "Act Version", "Repeal/Replacement". `origin_bill` always `NOT_VERIFIED` because `Act.BillID` not surfaced. `act_publication` `VERIFIED` whenever `SourceURL != ""` — workaround. The web `/acts/[id]/lineage` page renders only `title` + `description` — does not render `Status`, `Date`, `SourceURL`. `ActVersion` implemented in domain but never seeded, never exposed via API.
- Acceptance criteria:
  - Lineage carries all 11 spec nodes.
  - `actResponse` includes `bill_id`.
  - Lineage page renders `Status`, `Date`, `SourceURL`.
  - `GET /api/v1/acts/{id}/versions` returns the version history.
- Files affected: `services/api/cmd/post_assent.go`, `apps/web/src/app/acts/[id]/lineage/page.tsx`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-105: Fiscal intelligence missing types (GAP-16-1, GAP-16-2, GAP-16-4, GAP-16-5, GAP-17-1, GAP-17-3, GAP-17-4)
- Source: audit-team-1, §16 + §17
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: Spec §16 lists `Debt Instrument`, `Guarantee`, `Budget Allocation`, `Fiscal Event` as required entities — none have Go domain types or SQL tables. Two parallel loan models (`GovernmentLoan` + `BorrowingAgreement`). `/loans` and `/grants` web pages consume hardcoded arrays, not the API. `FiscalYear` never seeded, never exposed. Commitment, Refinancing, Debt restructuring not modelled as event types.
- Acceptance criteria:
  - 4 missing entity types declared + SQL tables created.
  - Unified loan model.
  - `/loans` and `/grants` pages fetch from the API.
  - `FiscalYear` seeded + filterable.
  - `CommitmentEvent`, `RefinancingEvent`, `RestructuringEvent` types added.
- Files affected: `services/legislation/internal/domain/`, `apps/web/src/app/loans/page.tsx`, `apps/web/src/app/grants/page.tsx`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-106: Government intelligence dead types (GAP-10-2, GAP-10-3, GAP-10-4, GAP-11-3, GAP-11-4)
- Source: audit-team-1, §10 + §11
- Severity: major
- Gate: 16 PRODUCTION WORKFLOW
- Description: `GovernmentPeriod` and `CabinetMember` defined in domain + SQL schema but not exposed by any API and contain no seed data — dead types. No SQL seed populates `government.presidents`, `government.administrations`, etc. The web `/governments/[id]/terms/[term]` page does not display `election_date` or `swearing_in_date`. The API does not accept an administration/term header. The debt dashboard hardcodes Uhuru/Ruto links.
- Acceptance criteria:
  - `GovernmentPeriod` + `CabinetMember` exposed via API.
  - SQL seed populates the government tables.
  - Term page renders all spec fields.
  - API accepts `X-Civic-Administration` + `X-Civic-Term` headers.
- Files affected: `services/api/cmd/governments.go`, `infrastructure/postgres/seed/`, `apps/web/src/app/governments/[id]/terms/[term]/page.tsx`
- Suggested fix: Wave-5 partial (Wave-6 added `government-defaults.ts`); remaining work OPEN.
- Status: IN_PROGRESS.

### ISSUE-107: Government debt summary partial (GAP-20-1, GAP-20-2, GAP-20-3)
- Source: audit-team-2, §20
- Severity: minor (per audit-team-2)
- Gate: 16 PRODUCTION WORKFLOW
- Description: `GovernmentDebtSummary` has no `FiscalTimeline` field. No separate per-presidential-term fiscal view. `/debt/page.tsx:140-163` hardcodes the two administration links.
- Acceptance criteria:
  - `FiscalTimeline` field added.
  - Per-term fiscal view rendered.
  - Administration links pulled from the API.
- Files affected: `services/legislation/internal/domain/public_debt.go`, `apps/web/src/app/debt/page.tsx`
- Suggested fix: Not yet started.
- Status: OPEN.

### ISSUE-108: Active-state styling missing (GAP-36-1, GAP-36-2, GAP-36-3, GAP-36-4)
- Source: audit-team-3, §36
- Severity: major
- Gate: 15 UX REVIEW
- Description: `usePathname()` is never read, so the current section is not visually indicated in the primary nav or mega-menu. No profile/login affordance — OIDC/Keycloak wired server-side but no sign-in button in the navbar. `md` viewport hides primary nav without surfacing a compact alternative. No keyboard arrow-key navigation between mega-menu items.
- Acceptance criteria:
  - Active section visually indicated.
  - Sign-in / profile button in the navbar.
  - Compact nav for `md` viewport.
  - Roving tabindex in the mega-menu.
- Files affected: `apps/web/src/components/header.tsx`
- Suggested fix: Wave-5 added Command Palette; remaining nav polish OPEN.
- Status: IN_PROGRESS.

---

## 5. MINOR

### ISSUE-109: No visual-regression test (GAP-35-1)
- Source: audit-team-3, §35
- Severity: minor
- Gate: 9 ACCESSIBILITY
- Description: No automated visual-regression test enforcing the "avoid" list (gradients, glassmorphism, decorative dashboards, giant numbers, manipulative urgency).
- Files affected: `tests/e2e/`
- Suggested fix: Add a Playwright snapshot test that diffs the homepage + briefing + acts list against golden screenshots and asserts no `bg-gradient-*` / `backdrop-blur` utilities are present in critical pages.
- Status: OPEN.

### ISSUE-110: Borrowing attribution enum not serialized (GAP-18-1, GAP-18-2)
- Source: audit-team-2, §18
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: `BorrowingAttribution` enum exists in the domain but no API serializes the actual relationship — `BorrowingAgreementResponse` does not include the attribution axis. `LegislatureID` and `FiscalYearID` on the agreement struct never populated by Kenya seed.
- Files affected: `services/legislation/internal/domain/public_debt.go`
- Suggested fix: Add the attribution axis to `BorrowingAgreementResponse`; populate `LegislatureID` + `FiscalYearID` in the seed.
- Status: OPEN.

### ISSUE-111: Constitution Spotlight no rotation log (GAP-9-3, GAP-9-4)
- Source: audit-team-1, §9
- Severity: minor
- Gate: 15 UX REVIEW
- Description: No mechanism to avoid "repeated content" beyond excluding the currently-shown article. No e2e test asserts the spotlight renders or that clicking "Read the Constitution" navigates.
- Files affected: `apps/web/src/components/constitution-spotlight.tsx`
- Suggested fix: Add a daily-pick seed; add an e2e assertion.
- Status: OPEN.

### ISSUE-112: Hero label "Ask Kenya" instead of "Ask Civic" (GAP-32-4, GAP-32-5)
- Source: audit-team-2, §32
- Severity: minor
- Gate: 15 UX REVIEW
- Description: The hero search is labeled "Ask Kenya anything…" instead of "Ask Civic"; the spec uses "Ask Civic" as a named component. Constitution Spotlight occupies hero's first viewport space instead of Civic Highlights.
- Files affected: `apps/web/src/app/page.tsx`
- Suggested fix: Wave-5 redesign may have addressed; verify.
- Status: IN_PROGRESS.

### ISSUE-113: Spotlight hardcoded article data (GAP-9-1)
- Source: audit-team-1, §9
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: The spotlight consumes the hardcoded `apps/web/src/data/constitution-articles.ts` instead of the Go BFF `/api/v1/constitution/articles` endpoint — the API and frontend have divergent article sets.
- Files affected: `apps/web/src/components/constitution-spotlight.tsx`, `apps/web/src/data/constitution-articles.ts`
- Suggested fix: Wave-5 homepage redesign may have addressed; verify.
- Status: IN_PROGRESS.

### ISSUE-114: Debt chart hardcoded colors (GAP-62-1, GAP-62-2, GAP-62-3)
- Source: audit-team-4, §62
- Severity: minor
- Gate: 9 ACCESSIBILITY
- Description: Debt chart uses `#1f5f3f`, `#0d6efd`, `#dc3545` (Bootstrap palette) instead of `colors.forest` / `colors.leaf` / `colors.clay`. `font-serif` and `font-sans` referenced but Tailwind config doesn't declare them. Dark-mode CSS redefines utility classes inline rather than swapping CSS variables.
- Files affected: `apps/web/src/app/debt/debt-trend-chart.tsx`, `apps/web/tailwind.config.ts`, `apps/web/src/app/globals.css`
- Suggested fix: Replace Bootstrap colors; declare `fontFamily.serif`; move dark-mode overrides to `:root.dark`.
- Status: OPEN.

### ISSUE-115: Chart accessibility missing axis labels (GAP-64-1, GAP-64-2, GAP-64-3)
- Source: audit-team-4, §64
- Severity: minor
- Gate: 9 ACCESSIBILITY
- Description: Y-axis on the SVG has no explicit `aria-label` declaring units ("KES trillions"). No `<figcaption>` with source URL + retrieved-at timestamp below the SVG. No automated check that prevents a future chart from starting its y-axis at a non-zero baseline.
- Files affected: `apps/web/src/app/debt/debt-trend-chart.tsx`
- Suggested fix: Add `<text>` axis-title element with `aria-label`; render `<figcaption>`; add a unit test asserting `yMin === 0` for every chart component.
- Status: OPEN.

### ISSUE-116: Date formatting inconsistent (GAP-65-4)
- Source: audit-team-4, §65
- Severity: minor
- Gate: 15 UX REVIEW
- Description: `bills/page.tsx` and debt chart use `'en-KE'`; some pages default to system locale. No central formatter.
- Files affected: `apps/web/src/lib/format.ts`
- Suggested fix: Add `formatDate`, `formatCurrency`, `formatNumber` and use them everywhere.
- Status: OPEN.

### ISSUE-117: "What Changed" personalisation + Briefing stub (GAP-30-6, GAP-30-7)
- Source: audit-team-2, §30
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: Civic Brief handler returns `{"items": []}` with `"Briefing — pending issue #40"`. Notifications seed sample notifications on first access; the actual notification engine is not wired.
- Files affected: `services/api/cmd/main.go`, `services/api/cmd/notifications.go`
- Suggested fix: Wave-5 backend pass may have resolved the briefing stub; verify. Notification engine OPEN.
- Status: IN_PROGRESS.

### ISSUE-118: Proactive-intelligence anti-patterns not codified (GAP-31-1, GAP-31-2)
- Source: audit-team-2, §31
- Severity: minor
- Gate: 6 AI EVAL
- Description: The anti-patterns the spec lists ("Never optimize for: outrage, fear, political persuasion, engagement manipulation, ideological reinforcement") are not codified as enforceable rules in the AI prompt or validator. No measurable guardrail (e.g. sentiment classifier).
- Files affected: `services/ai/app/rag.py`, `services/ai/app/citation_validator.py`
- Suggested fix: Add the rules to the system prompt; add a sentiment classifier.
- Status: OPEN.

### ISSUE-119: RAG pipeline step collapse (GAP-27-2, GAP-27-5)
- Source: audit-team-2, §27
- Severity: minor
- Gate: 6 AI EVAL
- Description: The RAG pipeline collapses the 7 spec steps (Retrieve → Inspect → Extract → Generate → Validate → Cite → Publish) into Retrieve → rerank → Generate → Validate; no extract-and-cite step before generation. No "Publish" step that writes accepted AI-derived candidate facts to the canonical store.
- Files affected: `services/ai/app/rag.py`
- Suggested fix: Refactor to the 7-step pipeline.
- Status: OPEN.

### ISSUE-120: Provenance minor fields (GAP-26-2, GAP-26-3)
- Source: audit-team-2, §26
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: `trust.evidence` does not store `document_version` as a distinct column. The spec's `relationship` provenance field is not modelled.
- Files affected: `infrastructure/postgres/migrations/018_trust_schema.up.sql`
- Suggested fix: Add the columns.
- Status: OPEN.

### ISSUE-121: Temporal validity missing on people (GAP-29-3, GAP-29-4)
- Source: audit-team-2, §29
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: `legislation.legislatures` and `legislation.houses` have only `active BOOLEAN` — no `start_date`/`end_date`. The domain `Legislature` struct has StartDate/EndDate but the DB schema does not. No API for retrieving historical institution state.
- Files affected: `infrastructure/postgres/migrations/008_legislative_core.up.sql`
- Suggested fix: Add the columns.
- Status: OPEN.

### ISSUE-122: Borrowing semantics missing minor fields (GAP-17-2, GAP-17-5)
- Source: audit-team-1, §17
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: Signing has `SignatureDate` but no `SignedBy` actor. Interest captured as `InterestRate` + `DebtRepayment.InterestPortion` — but no `InterestPaid` aggregate, no `InterestType`.
- Files affected: `services/legislation/internal/domain/public_debt.go`
- Suggested fix: Add the fields.
- Status: OPEN.

### ISSUE-123: Authority level enum collapsed (GAP-23-2)
- Source: audit-team-2, §23
- Severity: minor
- Gate: 7 DATA QUALITY
- Description: `authority_level` enum collapses many of the spec's 8 levels into 4 — no distinct rank for "Parliament" vs "Kenya Law" vs "Official Gazette".
- Files affected: `infrastructure/postgres/migrations/018_trust_schema.up.sql`
- Suggested fix: Expand the enum.
- Status: OPEN.

### ISSUE-124: EventEnvelope CorrelationID never populated (GAP-47-5)
- Source: audit-team-3, §47, GAP-47-5
- Severity: minor
- Gate: 13 OBSERVABILITY
- Description: `EventEnvelope.CorrelationID` is never populated by `NewEnvelope`; sets `OccurredAt` + `EventType` + `Payload` only.
- Files affected: `packages/contracts/events.go`
- Suggested fix: Wave-5 added request-ID middleware; `NewEnvelope` should accept a correlation ID. Verify.
- Status: IN_PROGRESS.

### ISSUE-125: No immutable-archival backend (GAP-24-6, GAP-24-7)
- Source: audit-team-2, §24
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: No immutable-archival storage backend wired up. `ingestion.documents.raw_storage_key` exists but no S3/MinIO client was implemented (Wave-6 fixed — see ISSUE-21). The "Validation → Canonical Data" pipeline step has no ingestion-side validator that rejects malformed items.
- Files affected: `services/ingestion/internal/infrastructure/`
- Suggested fix: Wire the S3 client (ISSUE-21); add the pre-canonical validator.
- Status: IN_PROGRESS.

### ISSUE-126: Citation snippet offsets missing in Go (GAP-25-3)
- Source: audit-team-2, §25
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: The `Citation` domain type lacks `snippet_offset_start`/`snippet_offset_end` (Python has them; Go does not), so cross-service exact-passage traces lose precision.
- Files affected: `services/legislation/internal/domain/`
- Suggested fix: Add the fields to the Go struct.
- Status: OPEN.

### ISSUE-127: Bills detail page mock-timeline fallback (ISSUE-1 partial)
- Source: audit-team-1, §13, GAP-13-1 partial
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: Bills detail page's `fetchTimeline` falls back to `mockTimeline` when the API is unreachable. The fallback is clearly labelled `source: 'mock'`, but it still ships mock data in production.
- Files affected: `apps/web/src/app/bills/[id]/page.tsx`
- Suggested fix: Return an empty timeline + a network-status banner instead.
- Status: OPEN.

### ISSUE-128: No query-performance benchmarks (GAP-46-4, GAP-49-5)
- Source: audit-team-3, §46 + §49
- Severity: minor
- Gate: 10 PERFORMANCE
- Description: No `EXPLAIN ANALYZE` suite for hot paths like `bills?status=in_progress`, `debt`, `search`. No benchmark queries — spec mandates "Benchmark actual queries".
- Files affected: `tests/perf/`
- Suggested fix: Add an `EXPLAIN ANALYZE` suite.
- Status: OPEN.

### ISSUE-129: PWA manifest incomplete (GAP-60-4)
- Source: audit-team-4, §60, GAP-60-4
- Severity: minor
- Gate: 15 UX REVIEW
- Description: PWA manifest was incomplete — single 192×192 icon, no 512×512, no maskable icon, no `start_url` cache strategy, no `shortcuts`.
- Files affected: `apps/web/public/manifest.json`
- Suggested fix: Wave-6 added the full icon set + 3 shortcuts (commit `e544a34`).
- Status: RESOLVED.

### ISSUE-130: Stampede protection missing (GAP-51-4, GAP-51-5)
- Source: audit-team-3, §51
- Severity: minor
- Gate: 10 PERFORMANCE
- Description: No stampede protection (no singleflight, no lock-then-fetch). No Redis outage / failure / recovery test.
- Files affected: `packages/cache/redis.go`, `tests/chaos/`
- Suggested fix: Add singleflight; Wave-6 chaos runbook for redis-down covers the recovery test.
- Status: IN_PROGRESS.

### ISSUE-131: No backup retention policy (GAP-50-4)
- Source: audit-team-3, §50
- Severity: minor
- Gate: 8 SECURITY
- Description: No retention policy enforcement, no ACL on objects, no hash-verification on retrieval (SHA-256 stored but never compared on read).
- Files affected: `packages/storage/s3.go`
- Suggested fix: Add a retention-policy check + hash comparison.
- Status: OPEN.

### ISSUE-132: Stale audit docs (GAP-68-5, GAP-69-5)
- Source: audit-team-4, §68 + §69
- Severity: minor
- Gate: 14 DOCUMENTATION
- Description: `MASTER_AUDIT.md` and `phase-1-audit.md` reference commit `a0638e7` / `5f4002a` — neither of which exist on `origin/main`. The audit doc is stale relative to the audited HEAD. `MASTER_AUDIT.md` itself contains a stale "P0-1: Duplicate Go type declarations" entry that has been fixed in code but the audit doc is not updated to "CLOSED".
- Files affected: `MASTER_AUDIT.md`, `docs/architecture/phase-1-audit.md`
- Suggested fix: Wave-5 rewrote `MASTER_AUDIT.md` (commit `3bda41a`); verify all stale entries closed.
- Status: IN_PROGRESS.

### ISSUE-133: Stale remote branches (GAP-70-1)
- Source: audit-team-4, §70
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: Two stale remote branches (`fix/issue-209-210-222-simulation-gaps`, `fix/issue-220-221-debt-gaps`) whose work was already merged.
- Files affected: GitHub remote
- Suggested fix: `git push origin --delete` both branches.
- Status: OPEN.

### ISSUE-134: PR template not enforced (GAP-70-4, GAP-70-5)
- Source: audit-team-4, §70
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: PR template exists but the "Closes #N" required check is not enforced by automation. Commit-message style is inconsistent.
- Files affected: `.github/workflows/`
- Suggested fix: Add `semantic-pull-request` Action + `commitlint`.
- Status: OPEN.

### ISSUE-135: Bicameral legislative period not modeled (GAP-7-1 partial)
- Source: audit-team-1, §7
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: Legislative Periods (the 12th/13th Parliament of Kenya) not modeled. The Nigeria (HoR + Senate) + South Africa (NA + NCOP) bicameral structure is in the country data but not modeled in the central legislature schema.
- Files affected: `services/legislation/internal/domain/`, `infrastructure/postgres/migrations/008_legislative_core.up.sql`
- Suggested fix: Add `LegislativePeriod` type + table; ensure Nigeria + South Africa adapters expose their bicameral periods.
- Status: OPEN.

### ISSUE-136: /country/uganda page missing
- Source: UX walkthrough (this audit cycle)
- Severity: minor
- Gate: 15 UX REVIEW
- Description: Every other country (Kenya, Tanzania, Ghana, Nigeria, South Africa) has a `/country/<code>` page; Uganda does not. The Uganda adapter ships contract + parliament tests but no surface page.
- Files affected: `apps/web/src/app/country/uganda/page.tsx`
- Suggested fix: Author the page using the Tanzania template as a reference.
- Status: OPEN.

### ISSUE-137: Borrowing authorization budget allocation not modelled (GAP-16-1 partial)
- Source: audit-team-1, §16
- Severity: minor
- Gate: 16 PRODUCTION WORKFLOW
- Description: `BorrowingAuthorization` has a `BudgetAllocationID *ID` pointer but `BudgetAllocation` itself does not exist as a struct or table.
- Files affected: `services/legislation/internal/domain/financial.go`
- Suggested fix: Add the `BudgetAllocation` type + table.
- Status: OPEN.

---

## 6. Cross-references

- `docs/PRODUCTION_GATE.md` — the per-gate checklist; each ISSUE above cites the gate it blocks.
- `docs/COMPLETION_MATRIX.md` — the per-feature status matrix; `VERIFIED` requires every ISSUE affecting the feature to be RESOLVED.
- `docs/NO_FAKE_COMPLETION.md` — §77; ISSUE-1, ISSUE-26, ISSUE-32, ISSUE-40, ISSUE-71, ISSUE-87, ISSUE-127 directly violate §77.
- `audit-team-{1,2,3,4,5}-report.md` — the per-section audit reports that grounded each ISSUE.
- `docs/UX_WALKTHROUGH.md` (this Wave-7 pass) — the §83 27-step walkthrough.
- `docs/COUNTRY_TEST_RESULTS.md` (this Wave-7 pass) — per-country adapter + page test results.

---

## 7. How to update this backlog

1. When an ISSUE moves from `OPEN` → `IN_PROGRESS`, update the Status field and link the PR.
2. When an ISSUE moves to `RESOLVED`, cite the commit hash + the verifying test (Go test, Playwright spec, or QA report).
3. New gaps discovered by future audits append to the next-available ISSUE number; never reuse numbers.
4. The summary count table at the top MUST be updated whenever a status changes.
5. No ISSUE may be silently dropped — if a gap is re-classified, record the reason inline.

---

*This backlog is governed by spec §71 + §82. Every gap must be a tracked GitHub issue with owner + acceptance criteria. **No silently-abandoned TODOs.***
