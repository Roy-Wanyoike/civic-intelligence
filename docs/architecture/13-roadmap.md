# Roadmap

The Civic Intelligence Platform follows an 8-phase roadmap. Each phase builds on
the previous one and must be substantially complete before the next begins.

## Phase 1 — Foundation ✅ (substantially complete)

- Monorepo structure (Go workspace + npm workspaces)
- CI/CD pipeline (GitHub Actions: lint, test, build, security scan, migration apply)
- Database schema (16 forward-only migrations, 9 schemas, pgvector, audit triggers)
- Domain model (Bill, BillVersion, BillEvent, Institution, Person, Committee)
- Source registry schema
- Object storage configuration (MinIO for dev, S3/R2 for prod)
- Event infrastructure (NATS JetStream)
- Observability scaffolding (OTel collector, Prometheus, Grafana, Loki, Tempo)
- Kenya adapter interface + reference implementation

**Status:** 11/11 Go modules compile, 30 Go tests pass, 28 Python tests pass,
17/17 frontend routes return 200. Issues #19-#23 remain for full backend completion.

## Phase 2 — Kenya Legislative Ingestion (in progress)

Real crawlers for Parliament of Kenya sources:

- Parliament Bills adapter (issue #24)
- Hansard adapter (#25)
- Order Paper adapter (#26)
- Votes & Proceedings adapter (#27)
- Committee documents adapter (#28)
- Kenya Law (Acts + regulations) adapter (#29)
- Kenya Gazette adapter (#30)
- Source registry + health monitoring (#31)

**Exit criteria:** Real Kenyan Bills discoverable in the platform with verified
stages, timelines, and source documents.

## Phase 3 — Bill Intelligence

- Productionize BillSummarizer with real LLM, eval-regulated (#32)
- Verified timeline extractor (#33)
- Document comparator — clause-level diff (#34)
- Related-entities graph — Bills ↔ people ↔ committees ↔ topics (#35)

**Exit criteria:** A citizen can ask "What is happening with this Bill?" and
receive a verified, evidence-grounded answer with citations.

## Phase 4 — Citizen Experience

- WCAG 2.2 AA accessibility audit (#36)
- PWA capabilities — offline Bills cache, installable (#37)
- Conversational interface polish — multi-turn Bill chat (#38)

**Exit criteria:** The platform is usable by citizens on mobile devices,
accessible to screen readers, and works offline.

## Phase 5 — Monitoring

- Following + notifications (#39)
- Daily civic brief automation (#40)
- Stage-change + amendment detection (#41)

**Exit criteria:** Citizens can follow Bills and receive verified notifications
when changes occur. Daily brief generated automatically.

## Phase 6 — Civic Intelligence

- Acts + regulations intelligence — Kenya Law integration (#42)
- Gazette notice intelligence (#43)
- Public participation surfacing (#44)

**Exit criteria:** Platform covers the full legislative lifecycle from Bill
to enacted Act, plus regulations and gazette notices.

## Phase 7 — Research Platform

- Advanced search — filters, facets, semantic + keyword hybrid (#45)
- Developer API — rate-limited, documented, SDK plans (#46)
- Research workspaces + dataset export (#47)

**Exit criteria:** Researchers and developers can build on the platform via API,
export datasets, and save queries.

## Phase 8 — Global Expansion

- Uganda adapter (#48)
- Tanzania adapter (#49)
- Ghana adapter (#50)
- Nigeria adapter — federal + state (#51)
- South Africa adapter (#52)

**Exit criteria:** Platform supports 6 countries through the adapter pattern,
with zero changes to the core domain model per country.

## Cross-cutting work

- AI evaluation dataset expansion — ≥100 cases (#53)
- Security hardening — SSRF + prompt-injection defense (#54)
- Observability dashboards — Grafana (#55)
- OpenTelemetry instrumentation (#60)
- OIDC + RBAC enforcement (#59)
- Playwright E2E testing (#61)
- axe-core accessibility scanning (#62)

## Dependency upgrade backlog

Major version bumps that need dedicated upgrade sprints:

- React 18 → 19 (breaking: component lifecycle, hooks changes)
- Next.js 14 → 16 (breaking: app router changes, middleware)
- Tailwind CSS 3 → 4 (breaking: config format, plugin API rewrite)
- TypeScript 5.9 → 7.0 (breaking: type system changes)
- Docker: golang 1.22 → 1.27, node 22 → 26, python 3.12 → 3.14

These are tracked individually and will be tackled after Phase 2 is complete.
