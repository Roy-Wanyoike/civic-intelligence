# Phase 1 — Foundation Audit

**Date:** 2026-09-09
**Auditor:** Principal Engineer (autonomous organization)
**Commit:** `5f4002a` (main, latest — includes OIDC + RBAC)
**Methodology:** Forensic inspection of actual repository state, not documentation claims.

---

## Executive Summary

Phase 1 Foundation is **READY WITH RISKS**. The architecture is sound, all
code compiles, 51 Go tests + 28 Python tests pass, and the deployment
infrastructure (Vercel multi-service + Docker Compose) is configured.
However, several gaps prevent declaring Phase 1 fully complete — primarily
that the Go API endpoints return placeholder responses instead of calling the
domain services, and the frontend uses mock data.

**Confidence:** Medium-High — the foundation is strong enough to build
Phase 2 on, but the vertical slice (real Bill → API → frontend) is not yet
demonstrated end-to-end.

---

## Scores

| Dimension | Score | Notes |
|-----------|-------|-------|
| Architecture | 8/10 | Clean bounded contexts, DDD, contracts package. Minor: some Go modules still reference root module. |
| Domain | 7/10 | Bill, BillVersion, BillEvent, Person, Institution modeled. Missing: Act, Regulation, Hansard entities in Go (DB schema has them). |
| Database | 9/10 | 16 migrations, 9 schemas, pgvector, audit triggers, CHECK constraint on candidate_facts. Solid. |
| Security | 6/10 | OIDC + RBAC middleware implemented (PR #90). But: DevVerifier doesn't check signatures; no SSRF allowlist yet; no secret scanning in CI. |
| Testing | 6/10 | 51 Go + 28 Python tests. But: no E2E, no integration tests against real DB, no contract tests for adapters. |
| Infrastructure | 6/10 | Docker Compose works; Vercel configured. But: Helm/Terraform are stubs; no Temporal workflows. |
| Observability | 3/10 | Structured logging exists. No OTel, no metrics, no dashboards. |
| Documentation | 8/10 | README, ARCHITECTURE, 14 ADRs, OpenAPI spec, roadmap doc. Strong. |

---

## Phase 1 Status

```
READY WITH RISKS

Confidence: Medium-High

Blocking Issues:
- #19: Go API endpoints return placeholders (not calling domain services)
- #20: Frontend uses mock data (not wired to real BFF)
- #60: No observability instrumentation

Completed Fixes (this session):
- #59: OIDC + RBAC implemented (PR #90)
- #91: Vercel FastAPI deployment fixed (PR #91)

Remaining Risks:
- DevVerifier doesn't verify JWT signatures (dev only — acceptable for now)
- No integration tests against real Postgres/NATS
- No E2E test for the critical Bill → API → frontend journey
- Helm/Terraform are stubs (deployment is Vercel-only for now)
```

---

## Detailed Findings

### Architecture (8/10) ✅ Strong

**What works:**
- 9 Go services + 1 Python AI service + Next.js frontend, each with `internal/{domain,application,infrastructure}` structure
- `packages/contracts/` is the shared type boundary — no service invents its own `Bill` type
- Country adapter pattern: Kenya-specific code isolated in `adapters/kenya/`
- Dependency direction enforced: domain packages have zero infrastructure imports
- 14 ADRs document major decisions

**Gaps:**
- 4 Go services (search, notifications, identity, intelligence) have domain code but no `cmd/main.go` — they can't start
- The API BFF returns placeholder JSON instead of calling the legislation service
- No inter-service HTTP clients defined

### Civic Domain (7/10) ✅ Adequate

**What works:**
- `Bill`, `BillVersion`, `BillEvent`, `Amendment`, `Clause` — fully modeled with tests
- `BillStateMachine` validates stage transitions using country-supplied definitions
- `Person`, `Institution`, `Legislature`, `House`, `Committee`, `PoliticalParty`, `Constituency`, `County` — modeled
- Kenya-specific strings (Senate, National Assembly, Second Reading) are NOT in the global domain — verified by tests

**Gaps:**
- `Act`, `Regulation`, `HansardDocument`, `OrderPaper`, `CommitteeReport`, `Vote`, `GazetteNotice`, `PublicParticipation` — exist in DB schema but NOT as Go domain types yet
- The domain model is Bill-focused; needs to expand for Phase 2 (Hansard, committees)

### Database (9/10) ✅ Excellent

**What works:**
- 16 forward-only migrations covering 9 schemas
- pgvector extension enabled
- ivfflat + GIN trigram + FTS indexes on document chunks
- Audit triggers on `legislation.bills`
- CHECK constraint on `intelligence.candidate_facts` enforcing "AI cannot mutate truth"
- `legislation.bill_versions` is immutable (INSERT only, partial unique index)
- CI job applies all migrations against a real Postgres container and verifies schemas

**Gaps:**
- No seed data for real Kenyan Bills
- No backup/restore tested
- No connection pool sizing validated

### Security (6/10) ⚠️ Needs work

**What works:**
- OIDC + RBAC middleware (PR #90): `OptionalAuth`, `RequireToken`, `RequireScope`, `RateLimit`
- Scope constants: `bill:read`, `ai:ask`, `notification:write`, `user:admin`
- Public reads (Bills, search, briefing) vs protected (Q&A, follow)
- Rate limiting: 300 req/min per IP
- `.gitignore` covers `.env`, secrets
- GitHub push protection active (blocked token leak)

**Gaps:**
- `DevVerifier` doesn't verify JWT signatures — dev only, must set `DEV_MODE=false` in prod
- No SSRF allowlist for crawlers (tracked #54)
- No `gitleaks` in CI (#54)
- No prompt-injection defense review (#54)

### Testing (6/10) ⚠️ Needs E2E

**What works:**
- 51 Go tests across 7 suites (legislation, kenya, evidence, ingestion, intelligence, documents, api)
- 28 Python tests (citation validator, gateway, capabilities, eval dataset)
- Next.js type-check + lint + build clean

**Gaps:**
- No E2E tests (#61)
- No integration tests against real Postgres/NATS
- No contract tests for the Kenya adapter's HTTP calls
- No accessibility scan (#62)

### Observability (3/10) ❌ Weak

**What works:**
- Python AI service has structured JSON logging
- Go services have `packages/observability/` with Logger interface + slog implementation
- `/healthz` + `/readyz` endpoints on API + legislation services

**Gaps:**
- No OpenTelemetry instrumentation (#60)
- No Prometheus `/metrics` endpoint
- No Grafana dashboards (#55)
- No distributed tracing
- No alerting

### Infrastructure (6/10) ⚠️ Partial

**What works:**
- Docker Compose for full dev stack (Postgres+pgvector, Redis, NATS, MinIO, OpenSearch, Temporal, Keycloak, MailHog, AI, Web)
- Vercel multi-service deployment configured (web + ai)
- 3 multi-stage Dockerfiles (Go, Python, Next.js)
- CI pipeline (lint, test, build, security scan, migration apply)

**Gaps:**
- Helm chart is a stub (#21)
- Terraform modules are stubs (#22)
- Temporal workflows not implemented (#23)
- No Go services in docker-compose

### Documentation (8/10) ✅ Strong

- README, ARCHITECTURE, CONTRIBUTING, SECURITY, CODE_OF_CONDUCT, CHANGELOG
- 14 ADRs in MADR format
- `docs/architecture/13-roadmap.md` (created this session)
- `docs/research/kenya-sources.md` (research document)
- Partial OpenAPI 3.1 spec
- `.github/CODEOWNERS`

---

## Critical Issues (P0/P1)

| # | Issue | Severity | Status |
|---|-------|----------|--------|
| #19 | Go API returns placeholders | P1 | Open — next priority |
| #20 | Frontend uses mock data | P1 | Open — depends on #19 |
| #60 | No OTel instrumentation | P1 | Open |
| #54 | No SSRF allowlist | P1 | Open |
| #61 | No E2E tests | P1 | Open |
| #59 | No OIDC/RBAC | P1 | ✅ CLOSED (PR #90) |
| #56 | File mode drift | P0 | ✅ CLOSED |
| #57 | No go.sum | P1 | ✅ CLOSED |
| #58 | Type divergence | P1 | ✅ CLOSED |

---

## Phase 1 Verdict

**READY WITH RISKS** — the foundation is strong enough to begin Phase 2,
but issues #19 and #20 should be resolved in parallel with Phase 2 work so
the vertical slice (real Bill → API → frontend) can be demonstrated.

The architecture, database schema, security middleware, and contracts
package are production-quality. The gaps are in wiring (API endpoints not
calling domain services) and observability — both addressable during
Phase 2 implementation.
