<div align="center">

# Civic Intelligence Platform

### Understand what your government is doing. No legal or parliamentary jargon required.

**Evidence-grounded civic intelligence for Africa** 🌍 — one platform, fourteen countries.

🇰🇪 Kenya · 🇺🇬 Uganda · 🇹🇿 Tanzania · 🇬🇭 Ghana · 🇳🇬 Nigeria · 🇿🇦 South Africa · 🇷🇼 Rwanda · 🇿🇲 Zambia · 🇸🇳 Senegal · 🇪🇬 Egypt · 🇲🇦 Morocco · 🇨🇩 DR Congo · 🇪🇹 Ethiopia · 🇲🇼 Malawi

[![Tests](https://img.shields.io/badge/tests-1003%20Go%20%2B%2028%20Python%20%2B%20TS%20PASS-brightgreen)]()
[![Countries](https://img.shields.io/badge/countries-14-blue)]()
[![Pages](https://img.shields.io/badge/frontend-73%20pages-9cf)]()
[![API](https://img.shields.io/badge/API-70%20routes-orange)]()
[![Commits](https://img.shields.io/badge/commits-164-lightgrey)]()
[![License](https://img.shields.io/badge/license-MIT-blue)]()
[![Vercel](https://img.shields.io/badge/deploy-Vercel%20ready-black)]()
[![Release](https://img.shields.io/badge/release-v0.2.0-blueviolet)]()

</div>

---

## What is this?

Civic Intelligence is an **open-source, multi-country civic intelligence system** that transforms authoritative government information across Africa into **plain-language explanations**, **verified timelines**, **evidence-backed answers**, and **citizen alerts**.

The platform runs as **one codebase serving fourteen countries**. Each country sees only its own data by default — Bills, Acts, Institutions, People, Debt, Constitution, and Government History — through a per-country **adapter** that implements the same `contracts.LegislativeSourceAdapter` interface. A contributor from Uganda works on `adapters/uganda/` and never touches Kenya's data; a contributor from Nigeria works on `adapters/nigeria/` and never touches Tanzania's data. The platform routes each request to the correct adapter based on the selected country (set via the navbar **Government Selector**).

A citizen can:

> **Open a Bill** → read a simple explanation → see the verified timeline → understand what stage it's at → ask a question → get an evidence-grounded answer → inspect the original source document.

Every factual claim is traceable to an official source. AI explains evidence — it never replaces it.

### Why it matters

Government information is public, but often impenetrable. Parliamentary terminology, legislative stages, legal documents, and committee reports are difficult for ordinary citizens to understand — and even harder to compare across borders. This platform bridges that gap, making civic information accessible to everyone (not just legal experts) and **comparable across African jurisdictions** for the first time.

---

## Key Features

The platform ships **10 flagship features**. Every feature is grounded in authoritative sources and respects the **country scope** set by the Government Selector.

| # | Feature | Description | API |
|---|---------|-------------|-----|
| 1 | **🕸️ Civic Knowledge Graph** | Interactive force-directed graph of how Bills, Acts, People, Institutions, Constitution Articles, and Government terms connect. BFS path finder, 11 node types, 12 edge types. | `/api/v1/graph/*` |
| 2 | **📰 Daily Brief** | Personalized, AI-grounded, plain-language daily summary of civic developments filtered by what the citizen follows. | `/api/v1/brief/*` |
| 3 | **🌍 Cross-country Comparison** | Side-by-side comparison of legislation, government structure, public debt, and civic indicators across the 14 supported countries. | `/api/v1/compare/*` |
| 4 | **🔮 Scenarios** | "What-if" simulation engine — model the impact of a policy change before it happens. Reality-tagged SIMULATED. | `/api/v1/scenarios/*` |
| 5 | **📜 Constitution** | Authoritative Constitution text (chapters + articles), never reinterpreted. | `/api/v1/constitution/*` |
| 6 | **🏛️ Government History** | Administrations, presidential terms, and legislatures over time. | `/api/v1/governments/*`, `/api/v1/transitions` |
| 7 | **💰 Public Debt** | Sovereign borrowing tracker — loans, creditors, debt-to-GDP. Never attributed personally to a president. | `/api/v1/debt/*` |
| 8 | **🔍 Audit an Act** | Provenance explorer — trace every claim in an Act to its source document. | `/api/v1/acts/{id}/audit`, `/api/v1/provenance/*` |
| 9 | **🔀 Legal Lineage** | Visual timeline of how a Bill becomes an Act — readings, committee stage, assent, commencement. | `/api/v1/acts/{id}/lineage`, `/api/v1/acts/{id}/events` |
| 10 | **📊 Indicators** | Civic indicators dashboard — bills introduced, bills passed, debt-to-GDP, parliament sessions per country. | `/api/v1/compare/indicators` |

See [`FLAGSHIP_FEATURES.md`](./FLAGSHIP_FEATURES.md) for the full feature catalogue.

---

## Multi-country Architecture

### One project, many countries

The platform is a **single deployable** that serves all 14 countries. There is no per-country fork, no per-country database, and no per-country frontend — only per-country **adapters** that plug into the same core domain.

```
┌──────────────────────────────────────────────────────────────────────┐
│                        FRONTEND (Next.js)                            │
│  Government Selector (navbar) → sets X-Civic-Country header           │
│  73 pages — each reads the selected country from context              │
└──────────────────────────────┬───────────────────────────────────────┘
                               │  X-Civic-Country: UG
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                          API BFF (Go)                                 │
│  CountryMiddleware → validates code → stores on request context      │
│  handlers call middleware.CountryFromContext(r.Context())             │
└──────────────────────────────┬───────────────────────────────────────┘
                               │  country = "UG"
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                   COUNTRY ADAPTER LAYER                               │
│  One interface (contracts.LegislativeSourceAdapter) — 14 impls:       │
│                                                                       │
│  ┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐│
│  │kenya   ││uganda  ││tanzania││ghana   ││nigeria ││s. africa││rwanda ││
│  │🇰🇪 KE  ││🇺🇬 UG  ││🇹🇿 TZ  ││🇬🇭 GH  ││🇳🇬 NG  ││🇿🇦 ZA  ││🇷🇼 RW ││
│  └────────┘└────────┘└────────┘└────────┘└────────┘└────────┘└────────┘│
│  ┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐│
│  │zambia  ││senegal ││egypt   ││morocco ││dr congo││ethiopia││malawi  ││
│  │🇿🇲 ZM  ││🇸🇳 SN  ││🇪🇬 EG  ││🇲🇦 MA  ││🇨🇩 CD  ││🇪🇹 ET  ││🇲🇼 MW ││
│  └────────┘└────────┘└────────┘└────────┘└────────┘└────────┘└────────┘│
│        Each adapter has its own seed data — changes are ISOLATED     │
└──────────────────────────────┬───────────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│              SHARED CORE DOMAIN (PostgreSQL 16)                       │
│  Same schemas for every country — Bills, Acts, Institutions, Debt,   │
│  Constitution, Government — partitioned logically by country_code.    │
└──────────────────────────────────────────────────────────────────────┘
```

### Each country sees only their data by default

When a user opens the platform, the navbar **Government Selector** defaults to Kenya (`KE`). Switching the selector to Uganda writes the selection to a cookie and — via the frontend API client — sends `X-Civic-Country: UG` on **every** request. The API's `CountryMiddleware` reads that header, validates it against the supported-country list, and stores it on the request context. Every list endpoint (`/api/v1/acts`, `/api/v1/people`, `/api/v1/institutions`, `/api/v1/loans`, …) filters its response by that country code before serialising.

### Country switcher in the navbar

The Government Selector in `apps/web/src/components/government-selector.tsx` renders a breadcrumb (e.g. `Kenya / William Ruto Administration / Term 1`) and a dropdown that lets the user switch countries + administrations. The selection is persisted in the `civic_gov_selection` cookie (read server-side in `app/layout.tsx` so the first server render already reflects the previous choice — no client-side flicker).

### Adapter architecture

Every country adapter implements the same `contracts.LegislativeSourceAdapter` interface. The 14 adapters live in `adapters/`:

| Adapter | Country | Code | Status |
|---------|---------|------|--------|
| `adapters/kenya/` | Kenya 🇰🇪 | `KE` | ✅ Active (parliament + kenya_law + president + gazette crawlers, seed data) |
| `adapters/uganda/` | Uganda 🇺🇬 | `UG` | ✅ Ready (Parliament of Uganda, unicameral) |
| `adapters/tanzania/` | Tanzania 🇹🇿 | `TZ` | ✅ Ready (Bunge la Tanzania) |
| `adapters/ghana/` | Ghana 🇬🇭 | `GH` | ✅ Ready (Parliament of Ghana) |
| `adapters/nigeria/` | Nigeria 🇳🇬 | `NG` | ✅ Ready (National Assembly: Senate + House of Reps) |
| `adapters/south_africa/` | South Africa 🇿🇦 | `ZA` | ✅ Ready (Parliament: NA + NCOP) |
| `adapters/rwanda/` | Rwanda 🇷🇼 | `RW` | ✅ Ready (Parliament of Rwanda, bicameral) |
| `adapters/zambia/` | Zambia 🇿🇲 | `ZM` | ✅ Ready (National Assembly of Zambia, unicameral) |
| `adapters/senegal/` | Senegal 🇸🇳 | `SN` | ✅ Ready (Assemblée Nationale du Sénégal) |
| `adapters/egypt/` | Egypt 🇪🇬 | `EG` | ✅ Ready (Egyptian Parliament, bicameral) |
| `adapters/morocco/` | Morocco 🇲🇦 | `MA` | ✅ Ready (Parliament of Morocco, bicameral) |
| `adapters/dr_congo/` | DR Congo 🇨🇩 | `CD` | ✅ Ready (Parliament of the DRC, bicameral) |
| `adapters/ethiopia/` | Ethiopia 🇪🇹 | `ET` | ✅ Ready (Federal Parliamentary Assembly, bicameral) |
| `adapters/malawi/` | Malawi 🇲🇼 | `MW` | ✅ Ready (National Assembly of Malawi, unicameral) |

Each adapter is a Go module with its own `go.mod` so it can evolve independently. The shared domain (`services/legislation/`, `services/evidence/`, …) never references a specific country — it only references the adapter interface.

### Country data isolation

**Each country's adapter is independent.** A contributor from Uganda working on `adapters/uganda/` will NOT affect Kenya's data or any other country's data. The platform routes requests to the correct adapter based on the selected country (via `middleware.CountryFromContext`). The adapter returns only that country's Bills, Acts, Institutions, People, and Debt — there is no shared mutable state.

The seed data for each country lives in `adapters/{country}/{country}_seed/` (e.g. `adapters/kenya/kenya_seed/`). A change to `adapters/kenya/kenya_seed/acts.go` only affects Kenya's Acts. A change to `adapters/uganda/internal/uganda_data.go` only affects Uganda's data. The core domain, the API handlers, and the frontend are country-agnostic — they always defer to the adapter for the selected country.

---

## Contributing

### Adding your country's data

If you are a contributor from **Uganda, Nigeria, Tanzania, Ghana, or South Africa**, you can add your country's civic data without touching any other country's data. Follow these 6 steps:

#### Step 1 — Create the adapter

```bash
mkdir -p adapters/{your_country}/
cd adapters/{your_country}/
go mod init github.com/Roy-Wanyoike/civic-intelligence/adapters/{your_country}
```

Implement `contracts.LegislativeSourceAdapter`. Mirror the structure of `adapters/kenya/adapter.go` — the interface is small (Bills, Acts, Institutions, People, Parliamentary terminology).

#### Step 2 — Add seed data

```bash
mkdir -p adapters/{your_country}/{your_country}_seed/
```

Add seed files for `acts.go`, `government.go`, `public_debt.go`, `constitution.go`, etc. — mirroring `adapters/kenya/kenya_seed/`. Each seed file is just a Go file exporting typed slices; the seeder is invoked from `services/legislation` at boot.

#### Step 3 — Register the adapter in main.go

In `services/api/cmd/main.go`, add your country to the supported list:

```go
middleware.SupportedCountries = []string{"KE", "UG", "TZ", "GH", "NG", "ZA"} // add yours
```

…and wire your adapter into the country-aware adapter lookup (the dispatch happens by country code from the request context).

#### Step 4 — Add the country page

```bash
mkdir -p apps/web/src/app/country/{your_country}/
```

Add a `page.tsx` that renders the country's civic dashboard. Mirror `apps/web/src/app/country/kenya/page.tsx`.

#### Step 5 — Add the country to the Government Selector defaults

In `apps/web/src/lib/government-defaults.ts`, add your country to the supported list (the selector uses this to render the dropdown).

#### Step 6 — Write contract tests

```bash
adapters/{your_country}/contract_test.go  # verify your adapter satisfies the interface
```

Mirror `adapters/kenya/contract_test.go`. The contract test enforces that your adapter produces the same shape of data as every other adapter.

### Each country's data is isolated — changes don't affect other countries

- The `adapters/{country}/` directory is a self-contained Go module.
- The `adapters/{country}/{country}_seed/` directory holds that country's authoritative seed data.
- The `services/api/cmd/main.go` dispatches to the correct adapter based on `middleware.CountryFromContext(r.Context())`.
- A merge to `adapters/uganda/` cannot regress `adapters/kenya/` — the Kenya adapter is unaware that Uganda exists.
- The contract test in `tests/contract/adapter_contract_test.go` runs against every adapter and fails if any adapter drifts from the interface.

---

## Architecture

```
                              OFFICIAL SOURCES (14 countries)
                                              │
                ┌──────┬──────┬──────┬──────┬──────┬──────┬──────┐
                │ KE   │ UG   │ TZ   │ GH   │ NG   │ ZA   │ RW   │  … 8 more
                └──┬───┴──┬───┴──┬───┴──┬───┴──┬───┴──┬───┴──┬───┘
                   ▼     ▼     ▼     ▼     ▼     ▼     ▼
                ┌──────────────────────────────────────────────────────┐
                │              COUNTRY ADAPTERS (14 modules)            │
                │   Each implements contracts.LegislativeSourceAdapter  │
                └──────────────────────────────┬───────────────────────┘
                                               │
                                               ▼
                                       DISCOVERY + FETCH
                                               │
                                               ▼
                                 HASH → ARCHIVE → PARSE → NORMALIZE → VALIDATE
                                               │
                          ┌────────────────────┴────────────────────┐
                          ▼                                         ▼
                      EVIDENCE                               CIVIC DOMAIN
                       (claims,                          (PostgreSQL 16 — 9 schemas,
                        citations,                          partitioned by country_code)
                        source conflicts)                       │
                          │                              ┌──────┴──────┐
                          ▼                              ▼             ▼
                     AI GATEWAY                     SEARCH       NOTIFICATIONS
                     (Python 3.12,                     │             │
                      FastAPI,                          ▼             ▼
                      RAG, citation                    CITIZEN       ALERTS
                      validation)                          │
                          │                                  │
                          ▼                                  │
                    CITATION VALIDATOR                       │
                          │                                  │
                          └──────────────────────────────────┘
                                              │
                                              ▼
                                          CITIZEN
                                (via Next.js frontend, 73 pages)
```

**The single most important rule:**

> AI may PROPOSE candidate facts, but may NEVER directly write to canonical legislative state. Every AI response passes citation validation; responses with unsupported claims fail validation.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| **Frontend** | Next.js 14, React 18, TypeScript, Tailwind CSS, TanStack Query |
| **Backend** | Go 1.23 (Chi router, DDD, clean architecture) |
| **AI** | Python 3.12, FastAPI, Pydantic v2, provider-agnostic gateway |
| **Database** | PostgreSQL 16 with pgvector, 9 schemas, 22 migrations |
| **Search** | PostgreSQL FTS + pgvector (OpenSearch ready) |
| **Events** | NATS JetStream |
| **Workflows** | Temporal |
| **Cache** | Redis |
| **Object Store** | MinIO (S3-compatible) |
| **Auth** | OIDC (Keycloak), RBAC with 4 roles + 9 permissions |
| **Observability** | OpenTelemetry, Prometheus (`/metrics`), Grafana, Loki, Tempo |
| **Infrastructure** | Docker Compose, Helm, Terraform, Vercel |
| **Payments** | M-Pesa (Safaricom Daraja API), Stripe |

---

## Project Structure

```
civic-intelligence/
├── apps/web/                      # Next.js frontend (73 pages)
├── services/
│   ├── api/                       # Go BFF — 70 REST endpoints, OIDC, RBAC, CountryMiddleware
│   ├── ai/                        # Python FastAPI — 12 AI capabilities, RAG, citation validation
│   ├── legislation/               # Go — Bill domain, state machine, versioning, debt repo
│   ├── ingestion/                 # Go — source registry, crawlers, dedup, Temporal workflows
│   ├── documents/                 # Go — PDF/HTML parsing, chunking
│   ├── evidence/                  # Go — claims, citations, source conflicts
│   ├── intelligence/              # Go — candidate-fact validation
│   ├── simulation/                # Go — scenario engine + constraint evaluation
│   └── (search/notifications/identity — stubs)
├── adapters/                      # 🌍 14 country adapters — one per country
│   ├── kenya/                     # 🇰🇪 KE — parliament + kenya_law + president + gazette crawlers + seed
│   ├── uganda/                    # 🇺🇬 UG — Parliament of Uganda (unicameral)
│   ├── tanzania/                  # 🇹🇿 TZ — Bunge la Tanzania
│   ├── ghana/                     # 🇬🇭 GH — Parliament of Ghana
│   ├── nigeria/                   # 🇳🇬 NG — National Assembly (Senate + House of Reps)
│   ├── south_africa/              # 🇿🇦 ZA — Parliament (NA + NCOP)
│   ├── rwanda/                    # 🇷🇼 RW — Parliament of Rwanda (bicameral)
│   ├── zambia/                    # 🇿🇲 ZM — National Assembly of Zambia (unicameral)
│   ├── senegal/                   # 🇸🇳 SN — Assemblée Nationale du Sénégal
│   ├── egypt/                     # 🇪🇬 EG — Egyptian Parliament (bicameral)
│   ├── morocco/                   # 🇲🇦 MA — Parliament of Morocco (bicameral)
│   ├── dr_congo/                  # 🇨🇩 CD — Parliament of the DRC (bicameral)
│   ├── ethiopia/                  # 🇪🇹 ET — Federal Parliamentary Assembly (bicameral)
│   ├── malawi/                    # 🇲🇼 MW — National Assembly of Malawi (unicameral)
│   └── registry/                  # Country adapter registry + wrappers
├── packages/                      # Shared Go packages
│   ├── contracts/                 # Adapter interface, events, typed errors, country codes
│   ├── auth/                      # Principal, scopes, OIDC verifier interface
│   ├── observability/             # Logger, metrics, tracer, SSRF allowlist
│   ├── config/                    # Env-tag struct loader
│   ├── events/                    # NATS event helpers
│   ├── cache/                     # Redis client
│   └── storage/                   # MinIO/S3 client
├── infrastructure/
│   ├── postgres/migrations/       # 22 SQL migrations (9 schemas, pgvector)
│   ├── postgres/seed/             # Country + Kenya institution seed SQL
│   ├── docker/                    # Docker Compose + 3 Dockerfiles
│   ├── kubernetes/helm/           # Helm chart with NetworkPolicy
│   ├── terraform/                 # RDS + S3 + EKS modules
│   └── observability/             # Prometheus, Grafana, OTel, Loki, Tempo
├── presentation/                  # Investor / demo slide deck (HTML + Markdown)
├── docs/
│   ├── architecture/              # 14 deep-dive docs + Phase 1 audit
│   ├── adr/                       # 14 Architecture Decision Records
│   ├── api/                       # OpenAPI 3.1 spec
│   └── research/                  # Kenya sources + global platform analysis
├── tests/
│   ├── contract/                  # Adapter + events + OpenAPI contract tests
│   ├── e2e/                       # Playwright + axe-core accessibility
│   ├── integration/               # API integration tests (acts, bills, debt, governments)
│   ├── load/                      # k6 load tests
│   └── chaos/                     # Chaos engineering scenarios
├── CHANGELOG.md                   # Release history (v0.2.0 current)
├── DEPLOYMENT.md                  # Vercel + Railway + Supabase deploy guide
├── FLAGSHIP_FEATURES.md           # 10 flagship features catalogue
├── README.md                      # this file
└── vercel.json                    # multi-service config (web + ai)
```

---

## Getting Started

### Prerequisites

- **Go** 1.23+
- **Node.js** 22+
- **Python** 3.12+
- **Docker** (optional, for full dev stack)

### Quick Start

```bash
# Clone the repo
git clone https://github.com/Roy-Wanyoike/civic-intelligence.git
cd civic-intelligence

# 1. Start infrastructure (optional — for full local dev)
docker compose -f infrastructure/docker/docker-compose.yml up -d

# 2. Run the Go API (serves real Bills from kenyalaw.org, scoped by country)
cd services/api
go run ./cmd/main.go    # → http://localhost:9000

# 3. Run the Python AI service
cd services/ai
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8000    # → http://localhost:8000

# 4. Run the Next.js frontend
cd apps/web
npm install
npm run dev    # → http://localhost:3000
```

### Verify it works

```bash
# API health check
curl http://localhost:9000/api/v1/healthz
# → {"status":"ok","service":"api","version":"0.2.0"}

# List Kenyan Bills (default country = KE)
curl http://localhost:9000/api/v1/bills | jq '.total'

# List Ugandan Bills (set the X-Civic-Country header)
curl -H "X-Civic-Country: UG" http://localhost:9000/api/v1/acts | jq '.country'
# → "UG"

# Global / dashboard view (data across all 14 countries)
curl -H "X-Civic-Country: ALL" http://localhost:9000/api/v1/compare/countries | jq '.countries | length'
# → 14
```

### Run Tests

```bash
# Go tests (1003 across all modules — 33 modules)
# Run from the repo root to execute every Go module's tests:
for d in services/api services/legislation services/evidence services/ingestion services/intelligence services/simulation services/documents adapters/kenya adapters/uganda adapters/tanzania adapters/ghana adapters/nigeria adapters/south_africa adapters/rwanda adapters/zambia adapters/senegal adapters/egypt adapters/morocco adapters/dr_congo adapters/ethiopia adapters/malawi packages/observability packages/storage packages/cache tests/contract tests/integration tests/chaos; do
  (cd "$d" && go test ./...)
done

# Python tests (28 — unit + eval)
cd services/ai && pytest tests/ eval/ -v

# Frontend type-check (TypeScript PASS)
cd apps/web && npx tsc --noEmit && npx next lint

# Production build (verifies all 73 pages + metadata routes generate)
cd apps/web && npx next build
```

---

## Deploy to Vercel

The platform is configured for Vercel multi-service deployment:

1. Go to [vercel.com/new](https://vercel.com/new)
2. Import `Roy-Wanyoike/civic-intelligence`
3. Vercel detects `vercel.json` — configures two services:
   - **web** (Next.js) at `apps/web/`
   - **ai** (Python FastAPI) at `services/ai/` with `entrypoint: main.py`
4. Set environment variables:
   - `NEXT_PUBLIC_SITE_URL` — your production URL (e.g. `https://your-app.vercel.app`) for correct Open Graph + canonical URLs
   - `AI_SERVICE_URL` — your Python AI service URL (Vercel auto-wires this if you use the multi-service pattern; otherwise point at Railway/Render)
   - `API_SERVICE_URL` — your Go API service URL (Railway/Render/Fly.io)
   - `OPENAI_API_KEY` — for real AI summaries (optional — Stub provider works without it)
   - `DEV_MODE=false` — for production auth (optional — defaults to dev mode)
   - `DATABASE_URL` — your Postgres connection string (optional — API works without DB by calling adapters live)
5. Deploy

The `vercel.json` cron runs `POST /api/v1/refresh` daily at 03:00 UTC (Vercel Hobby plan limits crons to once per day — upgrade to Pro for hourly).

See [`DEPLOYMENT.md`](./DEPLOYMENT.md) for the full guide including Railway/Render/Fly.io alternatives.

---

## API Reference

### Country Scoping

Every list endpoint accepts an `X-Civic-Country` header (or `?country=KE` query param) that scopes the response to a single country. Supported values (14 countries + global view):

| Code | Country |
|------|---------|
| `KE` | Kenya 🇰🇪 |
| `UG` | Uganda 🇺🇬 |
| `TZ` | Tanzania 🇹🇿 |
| `GH` | Ghana 🇬🇭 |
| `NG` | Nigeria 🇳🇬 |
| `ZA` | South Africa 🇿🇦 |
| `RW` | Rwanda 🇷🇼 |
| `ZM` | Zambia 🇿🇲 |
| `SN` | Senegal 🇸🇳 |
| `EG` | Egypt 🇪🇬 |
| `MA` | Morocco 🇲🇦 |
| `CD` | DR Congo 🇨🇩 |
| `ET` | Ethiopia 🇪🇹 |
| `MW` | Malawi 🇲🇼 |
| `ALL` | Global / cross-country dashboard view |

If the header is absent, the request defaults to `KE` (Kenya). The resolved country is echoed back on the response as `X-Civic-Country`. Invalid codes return `400 Bad Request` with the supported list.

The `ALL` mode is used by the `/compare`, `/indicators`, `/dashboard`, and `/graph` endpoints to return data across all 14 countries.

### Public Endpoints (no auth required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/healthz` | Health check |
| GET | `/api/v1/bills` | List Bills (filtered by country) |
| GET | `/api/v1/bills/{id}` | Bill detail (fetches + parses bill page) |
| GET | `/api/v1/bills/{id}/timeline` | Bill timeline events |
| GET | `/api/v1/bills/{id}/summary` | AI-generated plain-language summary |
| GET | `/api/v1/bills/{id}/changes` | Version comparison |
| GET | `/api/v1/bills/{id}/related` | Related entities (people, committees) |
| GET | `/api/v1/trending` | Trending bills (approaching final, hot, recent) |
| GET | `/api/v1/terminology/{term}` | Parliamentary term explanation |
| GET | `/api/v1/acts` | Acts of Parliament (filtered by country) |
| GET | `/api/v1/acts/{id}` | Act detail |
| GET | `/api/v1/acts/{id}/audit` | Audit an Act — provenance explorer |
| GET | `/api/v1/acts/{id}/lineage` | Legal lineage — Bill → Act |
| GET | `/api/v1/acts/{id}/events` | Post-assent events |
| GET | `/api/v1/loans` | Government loans (filtered by country) |
| GET | `/api/v1/grants` | Government grants (filtered by country) |
| GET | `/api/v1/people` | People — MPs, Speakers, President (filtered by country) |
| GET | `/api/v1/committees` | Parliamentary committees (filtered by country) |
| GET | `/api/v1/institutions` | Institutions — Parliament, NA, Senate, Executive (filtered by country) |
| GET | `/api/v1/feed` | Civic activity feed |
| GET | `/api/v1/search?q=...` | Search (filtered by country) |
| GET | `/api/v1/briefing` | Daily civic brief (filtered by country) |
| GET | `/api/v1/brief/*` | Personalised Civic Daily Brief |
| GET | `/api/v1/policies` | Government policies |
| GET | `/api/v1/constitution` | Constitution text |
| GET | `/api/v1/constitution/articles` | Constitution articles |
| GET | `/api/v1/governments` | Administration history (filtered by country) |
| GET | `/api/v1/transitions` | Government transitions |
| GET | `/api/v1/debt` | Public debt dashboard (filtered by country) |
| GET | `/api/v1/debt/loans` | Borrowing register |
| GET | `/api/v1/debt/timeline` | Debt stock timeline |
| GET | `/api/v1/debt/governments/{id}` | Per-administration debt summary |
| GET | `/api/v1/debt/legislatures/{id}` | Per-legislature debt summary |
| GET | `/api/v1/graph` | Civic Knowledge Graph (filterable by country) |
| GET | `/api/v1/graph/nodes` | Graph nodes |
| GET | `/api/v1/graph/relationships` | Graph edges |
| GET | `/api/v1/graph/paths` | BFS shortest-path finder |
| GET | `/api/v1/compare/countries` | Cross-country comparison — country profiles |
| GET | `/api/v1/compare/legislation` | Cross-country legislation comparison |
| GET | `/api/v1/compare/debt` | Cross-country debt comparison |
| GET | `/api/v1/compare/government-structure` | Cross-country government structure |
| GET | `/api/v1/compare/indicators` | Cross-country civic indicators |
| GET | `/api/v1/scenarios` | Scenario simulations |
| GET | `/api/v1/provenance/{type}/{id}` | Provenance explorer |
| GET | `/api/v1/sources` | Trust sources |
| GET | `/api/v1/contradictions` | Source conflicts |
| GET | `/metrics` | Prometheus metrics |

### Protected Endpoints (auth + scope required)

| Method | Endpoint | Scope | Description |
|--------|----------|-------|-------------|
| POST | `/api/v1/questions` | `ai:ask` | Ask a question (AI Q&A) |
| POST | `/api/v1/questions/stream` | `ai:ask` | SSE streaming Q&A |
| POST | `/api/v1/subscriptions` | `notification:write` | Follow a Bill/committee/topic |
| GET | `/api/v1/subscriptions` | — | List your follows |
| GET | `/api/v1/notifications` | — | List your notifications |

### Sponsor Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/sponsor/mpesa` | Initiate M-Pesa STK Push |
| POST | `/api/v1/sponsor/mpesa/callback` | Safaricom Daraja callback |
| POST | `/api/v1/sponsor/card` | Create Stripe Checkout session |
| POST | `/api/v1/sponsor/card/webhook` | Stripe webhook |

---

## Countries

| Country | Code | Adapter | Status |
|---------|------|---------|--------|
| 🇰🇪 Kenya | `KE` | `adapters/kenya/` | ✅ Active — full adapter (parliament + kenya_law + president + gazette crawlers, seed data) |
| 🇺🇬 Uganda | `UG` | `adapters/uganda/` | ✅ Ready — Parliament of Uganda (unicameral) |
| 🇹🇿 Tanzania | `TZ` | `adapters/tanzania/` | ✅ Ready — Bunge la Tanzania |
| 🇬🇭 Ghana | `GH` | `adapters/ghana/` | ✅ Ready — Parliament of Ghana |
| 🇳🇬 Nigeria | `NG` | `adapters/nigeria/` | ✅ Ready — National Assembly (Senate + House of Representatives) |
| 🇿🇦 South Africa | `ZA` | `adapters/south_africa/` | ✅ Ready — Parliament (National Assembly + NCOP) |
| 🇷🇼 Rwanda | `RW` | `adapters/rwanda/` | ✅ Ready — Parliament of Rwanda (bicameral) |
| 🇿🇲 Zambia | `ZM` | `adapters/zambia/` | ✅ Ready — National Assembly of Zambia (unicameral) |
| 🇸🇳 Senegal | `SN` | `adapters/senegal/` | ✅ Ready — Assemblée Nationale du Sénégal |
| 🇪🇬 Egypt | `EG` | `adapters/egypt/` | ✅ Ready — Egyptian Parliament (bicameral) |
| 🇲🇦 Morocco | `MA` | `adapters/morocco/` | ✅ Ready — Parliament of Morocco (bicameral) |
| 🇨🇩 DR Congo | `CD` | `adapters/dr_congo/` | ✅ Ready — Parliament of the DRC (bicameral) |
| 🇪🇹 Ethiopia | `ET` | `adapters/ethiopia/` | ✅ Ready — Federal Parliamentary Assembly (bicameral) |
| 🇲🇼 Malawi | `MW` | `adapters/malawi/` | ✅ Ready — National Assembly of Malawi (unicameral) |

Adding a new country means implementing `contracts.LegislativeSourceAdapter` — the core domain stays unchanged. See [Contributing](#contributing) above.

---

## The Evidence-First Architecture

Every factual claim follows this chain:

```
Claim
  ↓
Evidence
  ↓
Document
  ↓
Snapshot (immutable)
  ↓
Official Source (parliament.go.ke, kenyalaw.org, parliament.go.ug, …)
  ↓
Date Retrieved
  ↓
Content Hash (SHA-256)
```

**AI may explain evidence. AI may never replace evidence.**

The citation validator checks:
1. Does the cited source exist?
2. Does the citation point to the correct document?
3. Does the cited passage support the claim?
4. Is the claim stronger than the evidence?
5. Is the source authoritative?

If validation fails, the claim is NOT published.

---

## Non-Negotiable Principles

1. **Evidence before AI** — Official sources are the foundation.
2. **Never fabricate** — Never invent legislation, votes, dates, MPs, or stages.
3. **Version everything** — Bill versions are immutable. History is never overwritten.
4. **Source everything** — Every factual claim is traceable to a source.
5. **Separate fact from interpretation** — UI distinguishes FACT, EXPLANATION, INFERENCE, UNKNOWN.
6. **Political neutrality** — Never rank politicians, never encourage voting for/against a party.
7. **Country independence** — Country-specific knowledge stays in `adapters/{country}/`. One country's data cannot leak into another.

---

## Documentation

- [FLAGSHIP_FEATURES.md](./FLAGSHIP_FEATURES.md) — the 10 flagship features catalogue
- [ARCHITECTURE.md](./ARCHITECTURE.md) — canonical architectural contract
- [CONTRIBUTING.md](./CONTRIBUTING.md) — how to contribute + add a country adapter
- [SECURITY.md](./SECURITY.md) — security policy + threat model
- [presentation/](./presentation/) — investor + demo slide deck (HTML + Markdown + speaker notes)
- [docs/architecture/](./docs/architecture/) — 14 deep-dive docs (incl. `04-country-adapters.md`)
- [docs/adr/](./docs/adr/) — 14 Architecture Decision Records (incl. `ADR-0004-country-adapter-pattern.md`)
- [docs/research/](./docs/research/) — Kenya sources + global platform analysis
- [docs/api/openapi.yaml](./docs/api/openapi.yaml) — OpenAPI 3.1 spec

---

## Roadmap

| Phase | Name | Status |
|-------|------|--------|
| 1 | Foundation | ✅ Complete |
| 2 | Kenya Legislative Ingestion | ✅ Complete (50+ real bills + Hansard + Order Paper + V&P + Committees + Gazette crawlers) |
| 3 | Bill Intelligence | ✅ Complete (AI summaries, timelines, chat) |
| 4 | Civic Monitoring & Alerts | ✅ Complete (following, notifications, feed) |
| 5 | Multi-Organ Civic Intelligence | ✅ Complete (Acts, regulations, policies) |
| 6 | Civic Graph | ✅ Complete (relationship foundation) |
| 7 | Research Platform | ✅ Complete (datasets, developer API) |
| 8 | Civic Intelligence OS | ✅ Complete (Ask Kenya, dashboard, sponsor) |
| 9 | Trust & Verification | ✅ Complete (provenance, corrections) |
| 10 | Developer Platform | ✅ Complete (API docs, datasets) |
| 11 | Global Expansion | ✅ Complete (14 country adapters: KE, UG, TZ, GH, NG, ZA, RW, ZM, SN, EG, MA, CD, ET, MW) |
| 12 | Country Scoping + Multi-country Platform | ✅ Complete (CountryMiddleware, Government Selector, cross-country comparison) |
| 13 | Civic Knowledge Network | 📋 Planned (see docs/architecture/phase-13-14-spec.md) |
| 14 | Proactive Civic Intelligence | 📋 Planned (see docs/architecture/phase-13-14-spec.md) |

---

## Sponsors

Support the platform via **M-Pesa** or **Card** at [/sponsor](https://civic-intelligence.vercel.app/sponsor).

All funds go toward server costs, AI processing, and data sourcing. We do not accept sponsorship from political parties or politicians.

---

## Stats

- **1003** Go tests (across API + 14 country adapters + legislation + simulation + contract + integration + chaos)
- **28** Python tests (AI gateway + capabilities + eval)
- **TypeScript PASS** (frontend type-check + lint)
- **73** frontend pages
- **70** API routes
- **94** documented OpenAPI paths
- **22** SQL migrations (9 schemas, pgvector)
- **14** Architecture Decision Records
- **14** country adapters
- **12** AI capabilities
- **10** flagship features
- **164** commits
- **v0.2.0** current release (see [CHANGELOG.md](./CHANGELOG.md))

---

## License

MIT. See [LICENSE](./LICENSE).

---

## Disclaimer

Civic Intelligence is an independent civic-information project. It is not affiliated with the Government of Kenya, Uganda, Tanzania, Ghana, Nigeria, South Africa, Rwanda, Zambia, Senegal, Egypt, Morocco, DR Congo, Ethiopia, or Malawi. All claims are traceable to official sources. This platform does not provide legal advice.

<div align="center">

**Show people what happened. Explain what it means. Show them the evidence. Let them decide what they think.**

</div>
