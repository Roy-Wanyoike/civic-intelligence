<div align="center">

# Civic Intelligence Platform

### Understand what your government is doing. No legal or parliamentary jargon required.

Evidence-grounded civic intelligence for Kenya 🇰🇪 — and eventually all of Africa.

[![Tests](https://img.shields.io/badge/tests-176%20Go%20%2B%2028%20Python-brightgreen)]()
[![Countries](https://img.shields.io/badge/countries-Kenya%20%E2%9C%85%20%7C%20Uganda%20%E2%9C%85%20%7C%20Tanzania%20%F0%9F%93%8B-blue)]()
[![License](https://img.shields.io/badge/license-MIT-blue)]()
[![Vercel](https://img.shields.io/badge/deploy-Vercel%20ready-black)]()

</div>

---

## What is this?

Civic Intelligence is an open-source platform that transforms authoritative Kenyan government information into **plain-language explanations**, **verified timelines**, **evidence-backed answers**, and **citizen alerts**.

A citizen can:

> **Open a Bill** → read a simple explanation → see the verified timeline → understand what stage it's at → ask a question → get an evidence-grounded answer → inspect the original source document.

Every factual claim is traceable to an official source. AI explains evidence — it never replaces it.

### Why it matters

Government information is public, but often impenetrable. Parliamentary terminology, legislative stages, legal documents, and committee reports are difficult for ordinary citizens to understand. This platform bridges that gap — making civic information accessible to everyone, not just legal experts.

---

## Key Features

| Feature | Description |
|---------|-------------|
| **🔍 Bill Discovery** | 50+ real Kenyan Bills discovered live from [kenyalaw.org](https://new.kenyalaw.org/bills/) |
| **📊 Trending Bills** | Track bills signed into law, approaching final stage, and recently published |
| **📖 Terminology** | 10+ Kenyan parliamentary terms explained in plain language with sources |
| **🤖 AI Q&A** | Ask questions about any Bill — every answer is citation-validated (anti-hallucination) |
| **📅 Timeline** | Verified legislative timeline for each Bill with evidence links |
| **💰 Loans & Grants** | Government loans tracker (10 real loans + 5 grants since Sep 2022) |
| **🔔 Notifications** | Follow Bills, committees, topics — get verified alerts |
| **📰 Civic Feed** | Chronological feed of verified civic activity |
| **🏛️ Acts & Regulations** | Browse enacted Acts, regulations, and government policies |
| **🌍 Multi-Country** | Kenya ✅ + Uganda ✅ — adapter architecture for Tanzania, Ghana, Nigeria, South Africa |
| **💳 Sponsor** | M-Pesa STK Push + Stripe Card payments to support the platform |
| **📊 Datasets** | Download civic data as CSV/JSON |
| **🔒 Trust Layer** | Provenance explorer — trace every claim to its official source |

---

## Architecture

```
                     OFFICIAL SOURCES
                          │
            ┌─────────────┼─────────────┐
            │             │             │
        Parliament    Kenya Law    President
            │             │             │
            └─────────────┼─────────────┘
                          ▼
                   COUNTRY ADAPTERS
                    (Kenya, Uganda)
                          │
                          ▼
                     DISCOVERY
                          │
                          ▼
            FETCH → HASH → ARCHIVE → PARSE
                          │
                          ▼
                    NORMALIZER
                          │
                          ▼
                     VALIDATOR
                          │
                          ▼
              ┌───────────┴───────────┐
              ▼                       ▼
          EVIDENCE               CIVIC DOMAIN
              │                  (PostgreSQL)
              ▼                       │
         AI GATEWAY           ┌───────┴───────┐
         (Python)             ▼               ▼
              │            SEARCH        NOTIFICATIONS
              ▼               │               │
        CITATION              ▼               ▼
        VALIDATOR          CITIZEN          ALERTS
              │
              ▼
           CITIZEN
```

**The single most important rule:**

> AI may PROPOSE candidate facts, but may NEVER directly write to canonical legislative state. Every AI response passes citation validation; responses with unsupported claims fail validation.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| **Frontend** | Next.js 14, React 18, TypeScript, Tailwind CSS, TanStack Query |
| **Backend** | Go 1.22 (Chi router, DDD, clean architecture) |
| **AI** | Python 3.12, FastAPI, Pydantic v2, provider-agnostic gateway |
| **Database** | PostgreSQL 16 with pgvector, 9 schemas, 17 migrations |
| **Search** | PostgreSQL FTS + pgvector (OpenSearch ready) |
| **Events** | NATS JetStream |
| **Workflows** | Temporal |
| **Auth** | OIDC (Keycloak), RBAC with 4 roles + 9 permissions |
| **Observability** | OpenTelemetry, Prometheus (/metrics), Grafana dashboards |
| **Infrastructure** | Docker Compose, Helm, Terraform, Vercel |
| **Payments** | M-Pesa (Safaricom Daraja API), Stripe |

---

## Project Structure

```
civic-intelligence/
├── apps/web/                      # Next.js frontend (36 pages)
├── services/
│   ├── api/                       # Go BFF — 30+ REST endpoints, OIDC, RBAC
│   ├── ai/                        # Python FastAPI — 12 AI capabilities, RAG, citation validation
│   ├── legislation/               # Go — Bill domain, state machine, versioning
│   ├── ingestion/                 # Go — source registry, crawlers, dedup
│   ├── documents/                 # Go — PDF/HTML parsing, chunking
│   ├── evidence/                  # Go — claims, citations, source conflicts
│   ├── intelligence/              # Go — candidate-fact validation
│   ├── search/                    # Go — FTS + pgvector projections
│   ├── notifications/             # Go — follows, subscriptions, delivery
│   └── identity/                  # Go — users, sessions, RBAC
├── adapters/
│   ├── kenya/                     # 🇰🇪 Kenya adapter (84 tests, 50 real bills)
│   │   ├── parliament/            # Bills, Hansard, Order Papers, Votes
│   │   ├── kenya_law/             # Acts, Bills from kenyalaw.org
│   │   ├── president/             # Presidential assent events
│   │   └── gazette/               # Kenya Gazette notices
│   └── uganda/                    # 🇺🇬 Uganda adapter (11 tests, unicameral)
├── packages/                      # Shared Go packages
│   ├── contracts/                 # Adapter interface, events, typed errors
│   ├── auth/                      # Principal, scopes, OIDC verifier interface
│   ├── observability/             # Logger, metrics, tracer, SSRF allowlist
│   ├── config/                    # Env-tag struct loader
│   └── events/                    # Event helpers
├── infrastructure/
│   ├── postgres/migrations/       # 17 SQL migrations (9 schemas, pgvector)
│   ├── docker/                     # Docker Compose + 3 Dockerfiles
│   ├── kubernetes/helm/            # Helm chart with NetworkPolicy
│   ├── terraform/                  # RDS + S3 modules
│   └── observability/             # Prometheus, Grafana, OTel, Loki, Tempo
├── docs/
│   ├── architecture/              # 14 deep-dive docs + Phase 1 audit
│   ├── adr/                       # 14 Architecture Decision Records
│   ├── api/                       # OpenAPI 3.1 spec
│   └── research/                  # Kenya sources + global platform analysis
├── tests/e2e/                      # Playwright + axe-core accessibility
└── .github/                       # CI, CodeQL, Dependabot, CODEOWNERS
```

---

## Getting Started

### Prerequisites

- **Go** 1.22+
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

# 2. Run the Go API (serves real Bills from kenyalaw.org)
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

# Discover real Kenyan Bills (live from kenyalaw.org)
curl http://localhost:9000/api/v1/bills | jq '.total'
# → 50

# Get trending bills
curl http://localhost:9000/api/v1/trending | jq '.total_bills'
# → 50

# Check AI capabilities
curl http://localhost:8000/v1/capabilities
# → 12 capabilities
```

### Run Tests

```bash
# Go tests (176 tests across 9 suites)
cd services/api && go test ./...
cd adapters/kenya && go test ./...
cd adapters/uganda && go test ./...

# Python tests (28 tests — unit + eval)
cd services/ai && pytest tests/ eval/ -v

# Frontend
cd apps/web && npx tsc --noEmit && npx next lint
```

---

## Deploy to Vercel

The platform is configured for Vercel multi-service deployment:

1. Go to [vercel.com/new](https://vercel.com/new)
2. Import `Roy-Wanyoike/civic-intelligence`
3. Vercel detects `vercel.json` — configures two services:
   - **web** (Next.js) at `apps/web/`
   - **ai** (Python FastAPI) at `services/ai/`
4. Set environment variables:
   - `OPENAI_API_KEY` — for real AI summaries (optional — Stub provider works without it)
   - `DEV_MODE=false` — for production auth (optional — defaults to dev mode)
   - `DATABASE_URL` — your Postgres connection string (optional — API works without DB by calling adapters live)
5. Deploy

---

## API Reference

### Public Endpoints (no auth required)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/healthz` | Health check |
| GET | `/api/v1/bills` | List Bills (live from kenyalaw.org) |
| GET | `/api/v1/bills/{id}` | Bill detail (fetches + parses bill page) |
| GET | `/api/v1/bills/{id}/timeline` | Bill timeline events |
| GET | `/api/v1/bills/{id}/summary` | AI-generated plain-language summary |
| GET | `/api/v1/bills/{id}/changes` | Version comparison |
| GET | `/api/v1/bills/{id}/related` | Related entities (people, committees) |
| GET | `/api/v1/trending` | Trending bills (approaching final, hot, recent) |
| GET | `/api/v1/terminology/{term}` | Parliamentary term explanation |
| GET | `/api/v1/acts` | Acts of Parliament |
| GET | `/api/v1/loans` | Government loans tracker |
| GET | `/api/v1/grants` | Government grants tracker |
| GET | `/api/v1/feed` | Civic activity feed |
| GET | `/api/v1/search?q=...` | Search |
| GET | `/api/v1/briefing` | Daily civic brief |
| GET | `/api/v1/policies` | Government policies |
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

| Country | Status | Bills | Adapter | Tests |
|---------|--------|-------|---------|-------|
| 🇰🇪 Kenya | ✅ Active | 50+ real bills | ✅ Full (parliament, kenya_law, president, gazette) | 84 |
| 🇺🇬 Uganda | ✅ Ready | Skeleton | ✅ Full (unicameral) | 11 |
| 🇹🇿 Tanzania | 📋 Planned | — | — | — |
| 🇬🇭 Ghana | 📋 Planned | — | — | — |
| 🇳🇬 Nigeria | 📋 Planned | — | — | — |
| 🇿🇦 South Africa | 📋 Planned | — | — | — |

Adding a new country means implementing `contracts.LegislativeSourceAdapter` — the core domain stays unchanged.

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
Official Source (parliament.go.ke, kenyalaw.org, president.go.ke)
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
7. **Country independence** — Kenya-specific knowledge stays in `adapters/kenya/`.

---

## Documentation

- [ARCHITECTURE.md](./ARCHITECTURE.md) — Canonical architectural contract
- [CONTRIBUTING.md](./CONTRIBUTING.md) — How to contribute + add a country adapter
- [SECURITY.md](./SECURITY.md) — Security policy + threat model
- [docs/architecture/](./docs/architecture/) — 14 deep-dive docs
- [docs/adr/](./docs/adr/) — 14 Architecture Decision Records
- [docs/research/](./docs/research/) — Kenya sources + global platform analysis
- [docs/api/openapi.yaml](./docs/api/openapi.yaml) — OpenAPI 3.1 spec

---

## Roadmap

| Phase | Name | Status |
|-------|------|--------|
| 1 | Foundation | ✅ Complete |
| 2 | Kenya Legislative Ingestion | ✅ Complete (50 real bills) |
| 3 | Bill Intelligence | ✅ Complete (AI summaries, timelines, chat) |
| 4 | Civic Monitoring & Alerts | ✅ Complete (following, notifications, feed) |
| 5 | Multi-Organ Civic Intelligence | ✅ Complete (Acts, regulations, policies) |
| 6 | Civic Graph | ✅ Complete (relationship foundation) |
| 7 | Research Platform | ✅ Complete (datasets, developer API) |
| 8 | Civic Intelligence OS | ✅ Complete (Ask Kenya, dashboard, sponsor) |
| 9 | Trust & Verification | ✅ Complete (provenance, corrections) |
| 10 | Developer Platform | ✅ Complete (API docs, datasets) |
| 11 | Global Expansion | ✅ Complete (Uganda adapter + research) |

---

## Sponsors

Support the platform via **M-Pesa** or **Card** at [/sponsor](https://civic-intelligence.vercel.app/sponsor).

All funds go toward server costs, AI processing, and data sourcing. We do not accept sponsorship from political parties or politicians.

---

## Stats

- **372** files tracked
- **176** Go tests (9 suites)
- **28** Python tests
- **53** TypeScript/TSX files
- **36** frontend pages
- **30+** API endpoints
- **17** SQL migrations (9 schemas)
- **14** ADRs
- **2** country adapters (Kenya + Uganda)
- **50+** real Kenyan Bills

---

## License

MIT. See [LICENSE](./LICENSE).

---

## Disclaimer

Civic Intelligence is an independent civic-information project. It is not affiliated with the Government of Kenya. All claims are traceable to official sources. This platform does not provide legal advice.

<div align="center">

**Show people what happened. Explain what it means. Show them the evidence. Let them decide what they think.**

</div>
