# Civic Intelligence Platform — Flagship Features

**Repository:** https://github.com/Roy-Wanyoike/civic-intelligence
**Last updated:** 2026-09-18
**Commits:** 128
**Tests:** ~700+ (Go + Python + TypeScript)

---

## Overview

The Civic Intelligence Platform is an evidence-first civic intelligence system built for Africa, starting with Kenya and expanding to Uganda, Tanzania, Ghana, Nigeria, and South Africa. It turns scattered government data into a queryable, evidence-backed civic knowledge infrastructure.

### Core Principle

> Reality is observed. Scenarios are constructed. Assumptions are explicit. Results are simulated. Evidence remains traceable.

The platform **never** presents a simulated outcome as an actual civic event. Every claim is traceable to an authoritative source.

---

## 10 Flagship Features

### 1. Civic Knowledge Graph (`/graph`)

**What it is:** An interactive force-directed graph visualization showing how Bills, Acts, People, Institutions, Constitution Articles, and Government terms are connected.

**Why it's unique:** No civic intelligence platform in Africa offers a visual graph explorer where citizens can trace relationships between legislation, government, and constitutional provisions.

**Key capabilities:**
- 11 node types (bill, act, institution, person, constitution_article, administration, presidential_term, legislature, committee, creditor, borrowing_agreement)
- 12 edge types (ORIGINATES_FROM, SPONSORED_BY, CITES, ASSESSED_BY, ASSENTED_BY, BORROWED_BY, etc.)
- BFS shortest-path finder — "How is this Bill connected to this President?"
- Search by name, filter by type
- Depth selector (1, 2, 3 hops)
- Zoom/pan/drag, PNG export
- Accessible tabular alternative view
- Mobile list view

**API:** `GET /api/v1/graph/nodes`, `/graph/node/{id}`, `/graph/relationships`, `/graph/search`, `/graph/paths`

---

### 2. Personalized Civic Daily Brief (`/brief`)

**What it is:** An AI-powered daily summary of civic developments, personalized by the user's followed topics, institutions, and Bills.

**Why it's unique:** Existing civic dashboards show raw data. This generates a *personalized, AI-grounded, plain-language brief* with every claim backed by evidence.

**Key capabilities:**
- 4 sections: What Changed Today, Your Followed Topics, What to Watch, Constitutional Context
- AI summary clearly labelled with ASSUMPTION reality badge + disclaimer
- Every item carries an `evidence_url` — traceable to authoritative source
- Template fallback (never fakes AI output — labelled "Auto-generated summary")
- Archive of past briefs
- Share button, print-friendly, mobile-first

**API:** `POST /api/v1/brief/generate`, `GET /api/v1/brief/today`, `GET /api/v1/brief/archive`, `GET /api/v1/brief/{id}`

---

### 3. Cross-country Comparison (`/compare`)

**What it is:** Side-by-side comparison of legislation, government structure, public debt, and civic indicators across 6 African countries.

**Why it's unique:** No platform in Africa offers structured cross-jurisdictional comparison with the same schema. This enables research that was previously impossible.

**Key capabilities:**
- 6 countries: Kenya, Uganda, Tanzania, Ghana, Nigeria, South Africa
- Dimensions: government structure, legislation by topic, public debt, civic indicators
- Visual charts: bar chart (debt), line chart (legislation), chamber diagrams
- **NEVER ranks countries** — disclaimer on every response, verified by tests
- Civic Indicators Dashboard with CSV export
- Print-friendly + shareable

**API:** `GET /api/v1/compare/countries`, `/compare/legislation`, `/compare/debt`, `/compare/government-structure`, `/compare/indicators`

---

### 4. Civic Indicators Dashboard (`/indicators`)

**What it is:** A dashboard of key civic indicators per country with trend badges and CSV export.

**Key indicators:**
- Bills introduced (this year)
- Bills passed (this year)
- Acts commenced (this year)
- Public debt to GDP
- Parliament sessions held
- Committee meetings held
- Public participation opportunities

**Why it's unique:** Structured civic indicators that are comparable across countries, with every value carrying a source URL.

---

### 5. What If? Scenario Explorer (`/scenarios`)

**What it is:** A rigorous scenario and simulation layer for exploring possible outcomes of civic events and policy scenarios.

**Why it's unique:** The only civic platform that lets citizens explore "What if this Bill becomes law?" with Monte Carlo uncertainty, sensitivity analysis, and clear reality/simulation separation.

**Key capabilities:**
- 8 scenario types (Policy, Legislative, Implementation, Comparative, Historical Counterfactual, Institutional, Economic/Social, Infrastructure)
- Monte Carlo engine with P10/P50/P90 percentiles (never false precision)
- Deterministic rules engine for simple scenarios
- Validation pipeline (inputs → evidence → assumptions → model → units → time → constraints)
- Reproducibility (input hash + random seed + version metadata)
- Scenario comparison (no political rankings)
- Timeline with OBSERVED/ASSUMED/MODELED/UNKNOWN labels
- 6-label visual language: FACT, EVIDENCE, ASSUMPTION, SIMULATION, UNKNOWN, LIMITATION

**API:** `POST /api/v1/scenarios`, `GET /api/v1/scenarios/{id}`, `POST /api/v1/scenarios/{id}/run`, `POST /api/v1/scenarios/compare`, `POST /api/v1/scenarios/{id}/replay`

---

### 6. Full Constitution Reader (`/constitution`)

**What it is:** The Constitution of Kenya, Article by Article, with connected legislation, institutions, and court decisions.

**Key capabilities:**
- Chapter → Article navigation
- Constitutional cross-reference engine (EXPLICIT, INFERRED, JUDICIAL, UNKNOWN)
- Each Article links to related Bills, Acts, institutions, court decisions
- Constitution Spotlight on homepage (DID YOU KNOW? / DO YOU KNOW?)
- Source: Kenya Law (kenyalaw.org)

**API:** `GET /api/v1/constitution`, `GET /api/v1/constitution/articles`

---

### 7. Government History (`/governments`)

**What it is:** Every presidential administration since independence, with terms, transitions, and temporal integrity.

**Key capabilities:**
- 5 Kenyan presidents (Jomo Kenyatta → William Ruto)
- N terms per president (Moi had 5, Jomo had 3 — data model supports any number)
- Government transitions as first-class events
- Government Selector (persistent context: Country → Administration → Term)
- No president hard-coded in application logic — all data-driven

**API:** `GET /api/v1/governments`, `GET /api/v1/governments/{id}`, `GET /api/v1/transitions`

---

### 8. Public Debt Dashboard (`/debt`)

**What it is:** Kenya's public debt and borrowing history with evidence-backed observations.

**Why it's unique:** The platform **never** attributes sovereign borrowing personally to a president. It says "The Government of Kenya recorded KSh X in borrowing during this period."

**Key capabilities:**
- 12 CBK observations (2013-2024) with source URLs
- Interactive SVG debt trend chart with accessible tabular equivalent
- Domestic vs External breakdown
- Government debt summaries (Uhuru Kenyatta, William Ruto administrations)
- Borrowing attribution: CONTRACTED_DURING, DISBURSED_DURING, REPAID_DURING, OUTSTANDING_DURING, REFINANCED_DURING
- 10 sample borrowing agreements (IMF, World Bank, AfDB, China Exim Bank, Eurobonds)
- Attribution validation enforced in code
- NO_POLITICAL_PERFORMANCE_SCORE disclaimer on every summary

**API:** `GET /api/v1/debt`, `/debt/loans`, `/debt/timeline`, `/debt/governments/{id}`, `/debt/legislatures/{id}`

---

### 9. Audit an Act (`/acts/[id]/audit`)

**What it is:** Full lifecycle audit of an Act of Parliament from assent through commencement, regulations, implementation, court challenges, and amendments.

**Why it's unique:** No other platform tracks what happens AFTER a Bill becomes law. This is the platform's strongest differentiating capability.

**Key capabilities:**
- Audit status: ASSENT_CONFIRMED, PUBLICATION_CONFIRMED, COMMENCEMENT_CONFIRMED, REGULATIONS_TRACKED, JUDICIAL_HISTORY_TRACKED, AMENDMENTS_TRACKED, REPEAL_STATUS_TRACKED, AUDIT_COMPLETE, PARTIALLY_TRACKED, DATA_GAP, CONFLICTING_SOURCES, NOT_VERIFIED
- "No commencement notice found" does NOT mean "never commenced" — reported as NOT_VERIFIED
- Follow-a-Law — creates a real subscription that monitors 8 event categories

**API:** `GET /api/v1/acts/{id}/audit`, `POST /api/v1/acts/{id}/follow`

---

### 10. Full Legal Lineage (`/acts/[id]/lineage`)

**What it is:** Complete traceability from Bill → Act → Amendment → Regulation → Court Decision.

**Key capabilities:**
- 9-step lineage: Origin Bill → Parliamentary Journey → Presidential Assent → Publication → Commencement → Regulations → Amendments → Court Decisions → Current Status
- Each step sourced from actual data (not hardcoded)
- Missing steps marked NOT_VERIFIED with "No authoritative record found yet."
- Never infers that nothing happened

**API:** `GET /api/v1/acts/{id}/lineage`

---

## Technology Stack

| Layer | Technology |
|---|---|
| Frontend | Next.js 16 (App Router), TypeScript, Tailwind CSS, D3.js, reveal.js |
| Backend (BFF) | Go 1.23, HTTP mux, OIDC (Keycloak), JWT (RS256 via go-jose) |
| AI Service | Python 3.12, FastAPI, tenacity, provider-agnostic LLM gateway |
| Domain | Go — pure domain packages (legislation, simulation, government) |
| Database | PostgreSQL 16 with pgvector, 22 migrations, 9 schemas |
| Infrastructure | Temporal, NATS JetStream, Redis, MinIO/S3, OpenTelemetry |
| Observability | Prometheus (31 metrics), Grafana (4 dashboards), OTel tracer |
| Testing | Go testing, Playwright (8 golden journeys + 5 critical failure tests), k6 (load), chaos tests |
| CI | GitHub Actions (build, test, migrations, lint) |
| Deployment | Vercel (frontend), Railway (backend), Helm (Kubernetes) |

---

## Multi-country Architecture

| Country | Code | Legislature | Houses | Adapter Status |
|---|---|---|---|---|
| Kenya | KE | Parliament of Kenya | 2 (NA + Senate) | ✅ Full (Discover + Parse + seed data) |
| Uganda | UG | Parliament of Uganda | 1 (unicameral) | ✅ Full |
| Tanzania | TZ | Bunge | 1 (unicameral) | ✅ Full |
| Ghana | GH | Parliament of Ghana | 1 (unicameral) | ✅ Full |
| Nigeria | NG | National Assembly | 2 (HoR + Senate) | ✅ Full |
| South Africa | ZA | Parliament | 2 (NA + NCOP) | ✅ Full |

---

## Evidence-First AI

The platform uses a 6-label visual language to separate reality from simulation:

| Label | Meaning | Color |
|---|---|---|
| **FACT** | Verified civic information from authoritative source | Emerald |
| **EVIDENCE** | Authoritative supporting material | Sky |
| **ASSUMPTION** | An explicit scenario input | Amber |
| **SIMULATION** | A modeled outcome (NOT an observed fact) | Violet |
| **UNKNOWN** | Insufficient evidence (never fabricates precision) | Stone |
| **LIMITATION** | A known limitation of the model | Rose |

Every AI-generated output is labelled. Every simulation result carries the model version, engine version, input hash, and random seed for reproducibility.

---

## Test Coverage

| Suite | Count | Status |
|---|---|---|
| Go unit tests | ~500+ | ✅ ALL PASS |
| Go integration tests | 45 | ✅ PASS |
| Go contract tests | 20 | ✅ PASS |
| Go chaos tests | 11 + 8 subtests | ✅ PASS |
| Go benchmarks | 9 | ✅ PASS |
| Python AI tests | 24 | ✅ PASS |
| TypeScript | tsc --noEmit | ✅ PASS |
| Playwright golden journeys | 8 | ✅ Written |
| Playwright critical failure tests | 5 | ✅ Written |
| k6 load tests | 3 | ✅ Written |

---

## Conference Talk

**Title:** "Building Civic Intelligence with Next.js + AI: Lessons from Kenya's Parliament"

**Format:** droidcon Uganda 2026 Session (30 min)

**Presentation:** `presentation/` folder with:
- `PROPOSAL.md` — submission abstract + 3 learning takeaways
- `slides.html` — self-contained reveal.js deck (14 slides)
- `SPEAKER_NOTES.md` — per-slide talking points
- `README.md` — how to present + demo checklist

---

## Links

- **Repository:** https://github.com/Roy-Wanyoike/civic-intelligence
- **Live demo:** (deploy to Vercel + Railway)
- **API docs:** `docs/api/openapi.yaml` (83 paths)
- **Architecture:** `ARCHITECTURE.md` (344 lines)
- **ADRs:** `docs/adr/` (14 Architecture Decision Records)
- **Presentation:** `presentation/slides.html`
