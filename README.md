# Civic Intelligence Platform

**Understand what your government is doing. No legal or parliamentary jargon required.**

Civic Intelligence is a production-grade platform that transforms authoritative government information into understandable explanations, searchable knowledge, verified timelines, evidence-backed answers, and citizen alerts. The first country is **🇰🇪 Kenya**. The architecture is country-agnostic — adding a country means writing a new adapter, not rebuilding the platform.

> **The principle:** Show people what happened. Explain what it means. Show them the evidence. Let them decide what they think.

## The wedge and the platform

The first citizen product is **Bill Intelligence / Bill Summarizer** — but the system is not architected as a Bill-only application. Bills are the first domain. The platform is designed to eventually support Bills, Acts, regulations, gazette notices, Hansard, committee reports, motions, petitions, public participation, and government policies — and ultimately an **"Ask Kenya"** capability that constructs an evidence-backed intelligence map across the whole of government.

## Non-negotiable principles

1. **Evidence before AI** — Official sources are the foundation. AI explains evidence. AI does not replace evidence.
2. **Never fabricate** — Never invent legislation, votes, dates, MPs, senators, committees, quotes, or stages. If evidence is unavailable, we say so.
3. **Version everything** — Never overwrite historical legislative information. Every Bill version is preserved immutably.
4. **Source everything** — Every important factual claim is traceable to a source.
5. **Separate fact from interpretation** — The UI distinguishes FACT, EXPLANATION, INFERENCE, UNKNOWN.
6. **Political neutrality** — Never rank politicians, never encourage voting for or against a party, never manipulate sentiment.
7. **Country independence** — Kenya is a country adapter. Kenyan legislative stages never appear as global domain constants.

## Architecture in one diagram

```
Citizen
   ↓
Next.js (apps/web)
   ↓
API/BFF  (services/api, Go)
   ↓
Domain Services  (services/{legislation, ingestion, documents, evidence, intelligence, search, notifications, identity}, Go)
   ↓
Evidence + Civic Data  (PostgreSQL, pgvector, OpenSearch)
   ↓
AI Gateway  (services/ai, Python/FastAPI)
   ↓
Evidence-grounded AI  (RAG + citation validation)
```

## The single most important rule

> The Civic Intelligence Platform has one canonical source of civic truth: the **Legislative (Civic) Domain**. Ingestion acquires information, Documents interpret file structure, Evidence establishes provenance, Intelligence generates explanations, Search creates query projections, and Notifications distribute verified changes. **No downstream service may silently mutate canonical legislative state.** AI may PROPOSE candidate facts but may NEVER directly write to canonical legislative state without validation + evidence.

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the full architectural contract.

## Repository layout

```
civic-intelligence/
├── apps/web/                  # Next.js 14 + Tailwind + TanStack Query
├── services/
│   ├── api/                   # Go BFF (Chi router, OIDC, OpenAPI)
│   ├── legislation/           # Canonical civic entities (Go)
│   ├── ingestion/             # Source registry + crawlers (Go)
│   ├── documents/             # PDF/HTML/DOCX parsing + OCR (Go)
│   ├── evidence/              # Claims, citations, source conflicts (Go)
│   ├── intelligence/          # AI orchestration + candidate-fact validation (Go)
│   ├── ai/                    # Provider-agnostic AI gateway + RAG pipeline (Python/FastAPI)
│   ├── search/                # FTS + pgvector projections (Go)
│   ├── notifications/         # Follows + delivery (Go)
│   └── identity/              # OIDC + RBAC + preferences (Go)
├── adapters/kenya/            # Kenya-specific: stages, terminology, parliament/kenya_law/gazette
├── packages/                  # Shared Go: contracts, events, observability, auth, config
├── infrastructure/
│   ├── postgres/migrations/   # 16 forward-only migrations, all schemas
│   ├── docker/                # Docker Compose + Dockerfiles
│   ├── kubernetes/helm/       # Helm chart
│   ├── terraform/             # Cloud infra modules
│   └── observability/         # Prometheus, Grafana dashboards, Loki, Tempo, OTel collector
├── docs/
│   ├── architecture/          # Deep-dive docs
│   ├── adr/                   # 14 Architecture Decision Records
│   ├── api/                   # OpenAPI spec
│   └── routes/                # Per-route frontend docs
├── tests/                     # Integration, e2e, contract, AI evaluation
└── .github/                   # CI, CodeQL, Dependabot, issue/PR templates
```

## Quickstart (local development)

```bash
# 1. Start the dev infrastructure
docker compose -f infrastructure/docker/docker-compose.yml up -d

# 2. Apply migrations (forward-only)
for f in $(ls infrastructure/postgres/migrations/*.up.sql | sort); do
  PGPASSWORD=civic psql -h localhost -U civic -d civic_intelligence -1 -f "$f"
done

# 3. Run the AI service
cd services/ai
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8000

# 4. Run the frontend
cd apps/web
npm install
npm run dev    # http://localhost:3000

# 5. Run tests
cd services/ai && pytest tests/ eval/ -v
cd apps/web && npm run lint && npx tsc --noEmit
```

## Status (Phase 1 — Foundation)

**What works today:**
- ✅ Next.js frontend — all 17 routes return HTTP 200; production build clean; type-check + lint pass
- ✅ Python AI service — 28/28 tests pass; FastAPI app with 12 civic AI capabilities; provider-agnostic gateway (Stub + OpenAI wired); RAG pipeline; citation validator with anti-hallucination guards; permanent evaluation dataset
- ✅ Database schema — 16 forward-only migrations covering all 9 schemas + pgvector + audit triggers + RBAC seed
- ✅ Docker Compose for the full local dev stack (Postgres+pgvector, Redis, NATS, MinIO, OpenSearch, Temporal, Keycloak, MailHog, AI service, web)
- ✅ CI pipeline — Python tests, web lint+build, migration apply against a real Postgres service container, Trivy security scan, Docker image build, CodeQL
- ✅ Documentation — README, ARCHITECTURE.md, CONTRIBUTING, SECURITY, 14 ADRs, OpenAPI, route docs

**What's scaffolded but not yet runnable in this environment (real source code):**
- ⚠️ Go backend services — directory structure + contracts in place; full source is being written in parallel (issue #CI-BE-001)
- ⚠️ Kenya adapter — interface defined; parliament/kenya_law/gazette adapters stubbed (issue #CI-AD-001)
- ⚠️ Temporal workflows, Helm chart, Terraform modules — file templates exist; full implementation tracked in issues #CI-INF-001..003

**What's tracked as next-phase work (GitHub issues):**
- Phase 2 — Kenya legislative ingestion (real crawlers against parliament.go.ke, kenyalaw.org, the gazette)
- Phase 3 — Bill Intelligence production-quality (verified timelines, document comparator, related entities)
- Phase 4 — Citizen experience polish (PWA, mobile, accessibility audit)
- Phase 5 — Monitoring (following, notifications, stage-change detection, daily briefing automation)
- Phase 6 — Civic Intelligence expansion (Acts, regulations, gazettes, policies, public participation)
- Phase 7 — Research platform (advanced search, entity graph, workspaces, export, public API)
- Phase 8 — Global expansion (Uganda, Tanzania, Ghana, Nigeria, South Africa adapters)

## Documentation

- [ARCHITECTURE.md](./ARCHITECTURE.md) — the canonical architectural contract
- [CONTRIBUTING.md](./CONTRIBUTING.md) — branch model, PR rules, commit format, how to add a country adapter
- [SECURITY.md](./SECURITY.md) — security policy + threat model
- [docs/architecture/](./docs/architecture/) — deep-dive docs (services, domain model, evidence system, AI gateway, events, security, observability, testing, deployment, roadmap, product vision)
- [docs/adr/](./docs/adr/) — 14 Architecture Decision Records (MADR format)
- [docs/api/openapi.yaml](./docs/api/openapi.yaml) — partial OpenAPI 3.1 spec for the citizen API
- [docs/routes/](./docs/routes/) — per-route frontend documentation

## License

MIT. See [LICENSE](./LICENSE).

## Principles for contributors

- Every meaningful change corresponds to an issue + PR — never push directly to `main`.
- Never close an incomplete issue. Never merge an incomplete feature.
- Never create fake API responses and call them production functionality.
- No AI prompt/model change reaches production without the eval dataset passing.
- No Kenya-specific strings outside `adapters/kenya/`.
- No `database/sql`, `net/http`, or NATS imports inside any `internal/domain` package.
- No direct writes from AI to `legislation.*` — only via validated candidate facts.

## Disclaimer

Civic Intelligence is an independent civic-information project. It is not affiliated with the Government of Kenya. All claims are traceable to official sources. This platform does not provide legal advice.
