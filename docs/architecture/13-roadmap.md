# 13 — Roadmap

This document maps the platform's 8 phases to milestones. Each phase has a goal, a definition of done, the deliverables, and the architectural capabilities it unlocks. The phases are sequential — each builds on the previous — but the work within a phase is parallelizable across teams.

The roadmap is the contract between the engineering team and the rest of the organization. It is the source of truth for what we are building, in what order, and why. It is updated through PRs; a phase's scope does not change without an ADR and a maintainer review.

## Phase 1 — Foundation

**Goal:** Stand up the platform's skeleton and prove the architectural contract is enforceable.

**Definition of done:** A developer can clone the repo, run `docker compose up`, run migrations, and hit `GET /healthz` on the Civic API. The boundary rule is enforced in CI (Semgrep rule blocks AI-to-canonical writes). The eval harness runs against a stub capability and reports pass/fail. The observability stack (Prometheus, Grafana, Loki, Tempo) is wired up and the platform-overview dashboard renders.

**Deliverables:**

- Monorepo structure: `services/`, `adapters/`, `packages/`, `infrastructure/`, `docs/`, `web/`.
- The 10 service skeletons with `README.md`, `Dockerfile`, and a health endpoint each.
- `packages/events/` schemas for the initial event set.
- `packages/contracts/` Go interfaces for the service boundaries.
- PostgreSQL cluster with logical schemas and migration tooling.
- NATS JetStream cluster with initial streams.
- Temporal cluster with the worker registration scaffolding.
- Keycloak realm for local dev with a test user.
- The full documentation tree (this `docs/` set) and the 14 ADRs.
- CI with lint, unit, integration, contract, SAST, dependency-scan, container-scan gates.
- The custom Semgrep rule that enforces the boundary rule.

**Architectural capabilities unlocked:** None yet — this phase is scaffolding. The platform is deployable but does nothing useful for citizens yet.

**Estimated timeline:** 4-6 weeks.

## Phase 2 — Kenya Legislative Ingestion

**Goal:** Get real Kenya Bills into the canonical domain, end-to-end, with provenance.

**Definition of done:** A scheduled crawl of the Kenya parliament website fetches published bills, parses them into clauses, attaches evidence (citations to the source PDF and page), updates the search index, and the bills appear on the API at `GET /bills`. The `KenyaBillIngestionWorkflow` runs end-to-end and is idempotent. A citizen (via API) can fetch a bill's metadata, its versions, its timeline, and its parsed clauses.

**Deliverables:**

- `adapters/kenya/` with sources, ontology, parsers, terminology, quality rules, and golden documents for at least 5 bills.
- `services/ingestion/` with the fetcher, SSRF allowlist, source registry, and `SourceDocument` storage.
- `services/documents/` with the PDF parser, clause extractor, and embedding generator.
- `services/legislation/` with the `Bill`, `BillVersion`, `BillStage`, `BillEvent`, `Amendment`, `Clause` entities and their event emitters.
- `services/evidence/` with the `Claim`, `Evidence`, `Citation` entities and the candidate-fact validator.
- `services/search/` with the full-text search index and the keyword query path.
- The `KenyaBillIngestionWorkflow` in Temporal.
- The full event catalog from [`07-events-workflows.md`](./07-events-workflows.md) implemented.

**Architectural capabilities unlocked:** The platform has canonical truth. Citizens can search and read bills. The wedge's substrate is in place.

**Estimated timeline:** 8-10 weeks.

## Phase 3 — Bill Intelligence (the wedge)

**Goal:** Ship the Bill Summarizer and the citizen-facing Bill page. This is the first product.

**Definition of done:** A citizen can land on `/country/kenya`, see a list of bills, click one, read a plain-language summary with citations, see the bill's timeline, see its versions, and ask a question about it (via the SSE endpoint) with an evidence-backed answer. The AI capability ships with a 50-case eval dataset that passes at 95% pass rate, 95% citation precision, 1% hallucination rate.

**Deliverables:**

- `services/ai/` with the `BillSummarizer`, `TimelineExtractor`, `PlainLanguageExplainer`, `StakeholderExtractor`, and `Ask` capabilities.
- `services/intelligence/` with the gateway, capability catalog, prompt registry, and eval orchestrator.
- The full RAG pipeline (see [`06-ai-gateway-rag.md`](./06-ai-gateway-rag.md)).
- The citizen-facing API: `GET /bills`, `GET /bills/{id}`, `GET /bills/{id}/timeline`, `GET /bills/{id}/versions`, `POST /bills/{id}/questions` (SSE), `GET /search`.
- The Next.js frontend: homepage, country page, bills list, bill detail, bill timeline, bill versions, bill chat.
- The AI eval dataset and the eval runner.
- The observability dashboards for AI (cost, latency, hallucination rate, eval pass rate).

**Architectural capabilities unlocked:** The wedge is live. Citizens get value. The full pipeline (ingestion → domain → evidence → intelligence → search → API → web) is exercised end-to-end.

**Estimated timeline:** 8-10 weeks.

## Phase 4 — Citizen Experience

**Goal:** Make the platform usable and useful enough that citizens return.

**Definition of done:** Citizens can create accounts, follow bills and committees, receive notifications on changes, get a daily digest, use the search effectively, and navigate the platform on mobile. The notification worker is in production; the identity service is integrated with Keycloak; the daily digest workflow runs.

**Deliverables:**

- `services/identity/` with user profiles, preferences, API keys, role bindings, and the Keycloak integration.
- `services/notifications/` with follows, notification templating, delivery adapters (in-app, email, push), digest scheduling.
- The Next.js routes for auth, follows, notification preferences, search, briefing.
- The daily-digest workflow.
- Accessibility audit and fixes (WCAG 2.1 AA).
- Mobile responsiveness audit and fixes.
- Onboarding flow for first-time citizens.

**Architectural capabilities unlocked:** The platform is sticky. Citizens come back. The notification loop is closed.

**Estimated timeline:** 6-8 weeks.

## Phase 5 — Monitoring, Hardening, Scale

**Goal:** Make the platform production-grade: observable, secure, performant at scale.

**Definition of done:** The observability stack has the full metrics catalog from [`09-observability.md`](./09-observability.md). Alerts fire on user-visible degradation and have runbooks. The security controls from [`08-security.md`](./08-security.md) are all in place (SSRF, prompt-injection evals, audit log, image signing, dependency scanning). Load tests pass at the target RPS. The first public security review is complete.

**Deliverables:**

- The full metrics catalog instrumented and the dashboards built.
- The alert runbooks for every alert.
- The full security control matrix implemented and verified.
- The load tests running nightly on `main` and weekly on staging.
- The SLOs (API p99 < 1s, search p99 < 500ms, AI p99 < 30s, ingest workflow p99 < 10m) defined and monitored.
- The first public security review report and remediation.

**Architectural capabilities unlocked:** The platform is trustworthy. It can be relied upon.

**Estimated timeline:** 4-6 weeks (overlapping with Phase 4).

## Phase 6 — Civic Intelligence (Acts, Regulations, Gazette, Hansard)

**Goal:** Expand beyond Bills to the full civic record. The `CivicMatter` abstraction is realized.

**Definition of done:** Citizens can search and read Acts, Regulations, Gazette Notices, Hansard documents, and Committee Reports. The `Ask` capability can answer questions that draw evidence from across these source types. The contradiction engine catches conflicts between source types (e.g. a regulation that contradicts its parent Act).

**Deliverables:**

- `Act`, `Regulation`, `GazetteNotice`, `HansardDocument`, `CommitteeReport`, `OrderPaper` entities and their ingestion adapters.
- The `GazetteIngestionWorkflow` and `HansardIngestionWorkflow` in Temporal.
- The `CommitteeReportSummarizer`, `HansardQuoteExtractor`, `ContradictionDetector` capabilities.
- The cross-domain retrieval in the `Ask` capability's RAG pipeline.
- The expanded eval dataset covering cross-domain questions.
- The frontend routes for Acts, Regulations, Hansard, Committees.

**Architectural capabilities unlocked:** The platform is a civic intelligence platform, not a bill platform. The north star ("Ask Kenya") becomes plausible.

**Estimated timeline:** 10-12 weeks.

## Phase 7 — Research Platform

**Goal:** Open the platform to researchers and developers via the API and open data.

**Definition of done:** External developers can register for API keys, hit the documented OpenAPI surface, build on top of the platform, and access a stable, versioned dataset. Researchers can download bulk data exports (Bills, Acts, votes) in CSV and Parquet. The API has SLA monitoring and rate-limit tiers in production.

**Deliverables:**

- The partner API tier with API keys, rate limiting, and usage analytics.
- The bulk data export pipeline (nightly snapshots to object storage).
- The SDK plans (a TypeScript SDK for the web, a Python SDK for researchers).
- The API versioning policy (URL prefix `/api/v1/`, deprecation policy) in production.
- The developer documentation and the API consumer guide.

**Architectural capabilities unlocked:** The platform is infrastructure. Civic-tech NGOs, newsroom dev teams, and government digital services build on top of it.

**Estimated timeline:** 6-8 weeks.

## Phase 8 — Global Expansion

**Goal:** Take the platform beyond Kenya. Add Uganda, Tanzania, Ghana, Nigeria, South Africa.

**Definition of done:** Each new country has a working adapter, a country landing page, and ingested bills. The platform's domain model is proven country-agnostic by the fact that adding a country touched no domain code. The cross-country `CivicMatter` view lets a researcher compare bills across countries on the same topic.

**Deliverables:**

- `adapters/uganda/`, `adapters/tanzania/`, `adapters/ghana/`, `adapters/nigeria/`, `adapters/southafrica/` following the seven-thing checklist from [`04-country-adapters.md`](./04-country-adapters.md).
- The country landing pages and the country switcher in the frontend.
- The cross-country `CivicMatter` search and comparison.
- The `Judgment` entity (court decisions) for countries where that data is available.
- Localized UIs (Swahili for Tanzania, etc.) where the language differs from English.

**Architectural capabilities unlocked:** The platform is a global civic intelligence infrastructure. The "Ask [Country]" vision is realized across multiple countries.

**Estimated timeline:** 12+ weeks (one country at a time, parallelizable across adapter teams).

## Milestone summary

| Phase | Name | Est. weeks | Cumulative |
| --- | --- | --- | --- |
| 1 | Foundation | 4-6 | 4-6 |
| 2 | Kenya Legislative Ingestion | 8-10 | 12-16 |
| 3 | Bill Intelligence | 8-10 | 20-26 |
| 4 | Citizen Experience | 6-8 | 26-34 |
| 5 | Monitoring, Hardening, Scale | 4-6 (parallel) | 26-34 |
| 6 | Civic Intelligence | 10-12 | 36-46 |
| 7 | Research Platform | 6-8 | 42-54 |
| 8 | Global Expansion | 12+ | 54+ |

The timeline is indicative, not a commitment. Phase boundaries may shift based on learnings; the phases themselves are stable. The first tagged release (`v0.1.0`) corresponds to the end of Phase 1; `v1.0.0` corresponds to the end of Phase 3 (the wedge in production with the eval suite passing).

## What is deliberately not on the roadmap

- **Mobile apps.** The web app is mobile-responsive; native apps are not planned until there is evidence that citizens want them. The PWA path is a possible intermediate.
- **A vote-decider or polling feature.** Out of scope forever (see [`01-overview.md`](./01-overview.md)).
- **Monetization.** The platform is open source and free for citizens. Partner API tiers may eventually charge for high-volume usage; the pricing model is TBD.
- **Other countries' data without their adapters.** We do not ingest a country's data without a maintained adapter; "scraper-only" countries are not on the roadmap.
