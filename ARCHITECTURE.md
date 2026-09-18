# Civic Intelligence Platform — Architecture

> The Civic Intelligence Platform has one canonical source of civic truth: the **Legislative (Civic) Domain**. Ingestion acquires information, Documents interpret file structure, Evidence establishes provenance, Intelligence generates explanations, Search creates query projections, and Notifications distribute verified changes. **No downstream service may silently mutate canonical legislative state.** AI may PROPOSE candidate facts but may NEVER directly write to canonical legislative state without validation + evidence.

This document is the canonical architecture contract. Every contribution MUST respect it. Violations are blocking review comments.

## 1. Mission

Government information is public, but public information is often difficult for ordinary citizens to understand. Parliamentary terminology, legislative stages, legal documents, amendments, committee reports, government notices, and policy documents can be difficult to interpret. The Civic Intelligence Platform builds the infrastructure that transforms authoritative government information into understandable explanations, searchable knowledge, timelines, evidence-backed answers, legislative monitoring, change detection, citizen alerts, research tools, and developer APIs.

The platform explains information without telling citizens what political position to take. The principle is: **Show people what happened. Explain what it means. Show them the evidence. Let them decide what they think.**

## 2. Service boundary map

```
                         ┌──────────────────────┐
                         │     API / BFF        │
                         │   Citizen Gateway    │
                         └──────────┬───────────┘
                                    │
             ┌──────────────────────┼──────────────────────┐
             │                      │                      │
             ▼                      ▼                      ▼
       Civic Service          Search Service        Identity Service
             │                      │
             │                      │
             ▼                      ▼
       Intelligence           Indexes / Vector
          Service                 Search
             │
      ┌──────┼────────┐
      │      │        │
      ▼      ▼        ▼
   Evidence  Civic   Notification
   Service   Data       Service
             │
             ▼
       Legislative
          Domain
             ▲
             │
      ┌──────┴─────────┐
      │                │
      ▼                ▼
 Ingestion Service   Workflow Engine
      │
      ▼
 Country Adapters
      │
      ▼
Official Sources
```

The important separation is:

- **Ingestion** discovers facts.
- **Civic Data** stores facts.
- **Evidence** proves facts.
- **Intelligence** explains facts.
- **Search** finds facts.
- **API** presents them.

## 3. Service ownership table

| Bounded context | Directory                              | Owns                                            | Does NOT own                                                        |
| --------------- | -------------------------------------- | ----------------------------------------------- | ------------------------------------------------------------------- |
| legislation     | `services/legislation/`                | Bills, Acts, stages, events, committees, people | How Kenya's stages are spelled (that's the adapter)                 |
| ingestion       | `services/ingestion/`                  | Sources, crawl jobs, raw documents              | What the Bill's stage is (that's legislation)                       |
| documents       | `services/documents/`                  | parsed documents, pages, sections, chunks       | What the chunks mean (that's intelligence)                          |
| evidence        | `services/evidence/`                    | claims, evidence, citations                     | What the claims mean (that's intelligence)                          |
| intelligence (Go) | `services/intelligence/`              | candidate-fact validation, orchestration        | Canonical state (it proposes; legislation accepts)                  |
| ai (Python)     | `services/ai/`                         | AI explanations, Q&A, RAG, evaluation            | Writing to canonical state — never                                  |
| search          | `services/search/`                     | search indexes/projections                      | Authoritative entity data — owns only projections                   |
| notifications   | `services/notifications/`              | follows and notifications                       | Whether a Bill actually changed stage (that's legislation)          |
| identity        | `services/identity/`                   | users/preferences                                | Civic information                                                   |
| api             | `services/api/`                        | HTTP BFF — calls domain services, no direct DB  | Business logic                                                      |
| kenya adapter   | `adapters/kenya/`                      | All Kenya-specific source code                  | Anything that should work for any other country                     |

## 4. Dependency direction

```
                    API
                     ↓
                Application
                     ↓
                  Domain
                     ↑
                Repository
                     ↑
              Infrastructure
```

**NEVER:** `Domain → PostgreSQL`, `Domain → HTTP`, `Domain → NATS`, `Domain → OpenAI`.

The domain package imports only its own types and shared contract types. All infrastructure (Postgres, NATS, HTTP, AI providers) is implemented as adapters that the domain's interfaces are satisfied by.

## 5. The "AI cannot mutate truth" rule

This rule is written into the architecture:

```
                 AI
                  │
                  │ proposes
                  ▼
             Candidate Fact
                  │
                  ▼
             Validation
                  │
            ┌─────┴─────┐
            │           │
         Accepted     Rejected
            │
            ▼
       Civic Domain
```

Never: `PDF → LLM → UPDATE bills SET status=...`

Instead: `PDF → Extraction → Candidate facts → Validation → Evidence → Domain command → Canonical state`

The `intelligence.candidate_facts` table has a CHECK constraint: `(validated = FALSE) = (accepted_at IS NULL)`. AI cannot promote a candidate fact to canonical state without going through validation. This is enforced at the schema level.

## 6. Country adapter pattern

The adapter is a plugin, not a service. It is loaded by the ingestion service.

```go
type LegislativeSourceAdapter interface {
    Discover(ctx context.Context) ([]SourceItem, error)
    Fetch(ctx context.Context, item SourceItem) (*RawDocument, error)
    Parse(ctx context.Context, doc RawDocument) ([]ExtractedRecord, error)
    GetLegislativeStructure(ctx context.Context) (*LegislativeStructure, error)
    GetStages(ctx context.Context) ([]StageDefinition, error)
    GetTerminology(ctx context.Context) ([]TermDefinition, error)
}
```

Kenya-specific code stays in `adapters/kenya/`. The global domain model in `services/legislation/` contains ZERO Kenya-specific strings — no "Senate", no "National Assembly", no "Second Reading". Those are all values supplied by the adapter.

This means: when we add Uganda, we write `adapters/uganda/` and everything works without touching the legislation domain. See [ADR-0004](./docs/adr/ADR-0004-country-adapter-pattern.md).

## 7. Event architecture

Services communicate primarily through events (NATS JetStream) for asynchronous workflows.

**Commands** say "do something." **Events** say "something happened." Don't use events as hidden commands.

Event catalog (publisher → subscribers):

| Event                         | Publisher              | Subscribers                                |
| ----------------------------- | ---------------------- | ------------------------------------------ |
| `source.discovered`           | ingestion              | documents, search                          |
| `source.changed`              | ingestion              | documents, notifications                   |
| `document.discovered`         | ingestion              | documents, search                          |
| `document.downloaded`          | ingestion              | documents                                  |
| `document.parsed`              | documents              | intelligence, evidence, search             |
| `document.ocr_completed`      | documents              | intelligence, evidence                     |
| `bill.discovered`              | ingestion              | legislation, search, notifications          |
| `bill.updated`                 | legislation            | search, notifications                      |
| `bill.version_created`         | legislation            | search, notifications, intelligence        |
| `bill.stage_changed`           | legislation            | search, notifications, intelligence         |
| `amendment.discovered`        | ingestion              | legislation, notifications                 |
| `committee.updated`           | ingestion              | legislation, search                        |
| `hansard.published`            | ingestion              | documents, search                          |
| `citation.created`             | evidence               | intelligence                               |
| `ai.explanation.generated`    | intelligence           | search, notifications                     |
| `ai.validation.failed`        | intelligence           | (observability + alerting)                |
| `notification.created`        | legislation            | notifications                              |

## 8. Workflow orchestration (Temporal)

Temporal orchestrates long-running processes. It knows **what workflow is happening**; the Legislative service knows **what a valid Bill state is**. Keep those responsibilities separate.

```
KenyaBillSyncWorkflow
   ├── Discover   (adapter.Discover)
   ├── Fetch      (adapter.Fetch)
   ├── Archive    (object storage, content hash)
   ├── Parse      (documents service)
   ├── Extract    (intelligence + evidence)
   ├── Validate   (candidate-fact validation)
   ├── Update     (domain command to legislation)
   ├── Embed      (documents service → pgvector)
   ├── Generate   (intelligence → AI explanation)
   ├── Validate citations
   └── Publish     (search projection + notifications)
```

Every step is retryable and observable.

## 9. Database strategy

Single Postgres cluster with logical schemas per bounded context (operational simplicity, future split possible without redesign):

```
PostgreSQL
├── legislation schema
├── ingestion schema
├── documents schema
├── evidence schema
├── intelligence schema
├── notifications schema
├── identity schema
├── search schema
└── audit schema
```

Extensions: `pgvector` (embeddings), `pg_trgm` (fuzzy text), `unaccent` (FTS normalization), `citext` (case-insensitive emails), `uuid-ossp` + `pgcrypto` (PKs).

`legislation.bill_versions` is IMMUTABLE — never UPDATE, only INSERT. Historical snapshots are preserved forever.

## 10. Deployment topology

The **code boundaries are established now**, but deployment boundaries evolve based on scale. Initial deployment is **modular monolith + workers**, not 15 microservices.

Five deployables for the first serious production iteration:

1. **Civic API** — API + legislative application layer + search queries + user-facing operations
2. **Ingestion Worker** — crawlers + Kenya adapters + document acquisition + Temporal workflows
3. **Document Worker** — extraction + OCR + chunking + embeddings
4. **AI Worker** — summarization + RAG + comparison + terminology + validation (Python)
5. **Notification Worker** — subscriptions + notifications + delivery

## 11. AI gateway + RAG pipeline

The application requests **capabilities** (e.g., `bill_summarizer`), not providers. The gateway routes to the configured provider, handles retries, timeouts, token budgets, cost tracking, model selection, structured output, safety validation, and fallback.

RAG pipeline:
```
question → intent classification → query expansion → hybrid search (FTS + pgvector)
       → reranking → evidence selection → context construction
       → LLM (provider-agnostic) → claim extraction → citation validation
       → response (with citations) or failure
```

A response with unsupported factual claims FAILS validation. The UI surfaces this honestly: "[This response could not be fully verified from available authoritative sources.]"

Twelve civic AI capabilities — one module per capability, never a giant prompt:
`BillSummarizer`, `TimelineExtractor`, `StageExplainer`, `TerminologyExplainer`, `DocumentComparator`, `EntityExtractor`, `TopicClassifier`, `ImpactAnalyzer`, `CivicQuestionAnswerer`, `CitationValidator`, `ContradictionDetector`, `BriefingGenerator`.

## 12. Evidence system

```
Claim → Evidence → Document → Page/Section → Source
```

AI responses MUST reference evidence. Citation validation is the trust layer:
- A claim without citations fails validation if it asserts a fact.
- A citation whose snippet shares no significant words with the claim fails validation (anti-hallucination).
- A citation with an unparseable URL fails validation.

When two authoritative sources conflict (e.g., source A says stage=Committee, source B says stage=Second Reading), the system DOES NOT silently resolve. It records a `SourceConflict` with both sources and metadata, and flags it for manual review. See [ADR-0013](./docs/adr/ADR-0013-contradiction-engine-do-not-silently-resolve.md).

## 13. Observability

OpenTelemetry instruments every service. Metrics, logs, and traces flow into Prometheus + Loki + Tempo.

Key metrics (the spec's required catalog):

| Metric                              | Source service         |
| ----------------------------------- | ---------------------- |
| `crawl_success_rate`                | ingestion              |
| `document_parse_failure_rate`       | documents              |
| `ocr_failure_rate`                  | documents              |
| `ai_failure_rate`                   | intelligence + ai      |
| `citation_validation_failure_rate`  | intelligence + ai      |
| `search_latency`                    | search                 |
| `api_latency`                       | api                    |
| `workflow_failure_rate`             | temporal               |
| `queue_depth`                       | nats                   |
| `llm_latency`                       | ai                     |
| `llm_cost`                          | ai                     |

## 14. Security

- OIDC authentication (Keycloak as self-hosted identity provider where appropriate)
- RBAC with role/permission tables; citizen / researcher / editor / admin
- Per-endpoint rate limiting
- API authorization on every protected route
- Input validation (Pydantic on Python; chi middleware + validator on Go)
- **SSRF protection**: crawler egress restricted to an allowlist; user-controlled URLs never directly drive crawler infrastructure
- Crawler network isolation (separate network namespace, egress firewall)
- Secret management via sealed secrets / Vault
- Audit logging on all canonical-table writes (audit schema + triggers)
- Encryption in transit (TLS) and at rest (disk + DB)
- Dependency scanning (Dependabot, pip-audit, npm audit, Trivy)
- Container scanning (Trivy)
- SAST (CodeQL)
- DAST where appropriate

See [SECURITY.md](./SECURITY.md) for the full policy.

## 15. Testing

| Layer        | What                                                  | Tooling                          |
| ------------ | ----------------------------------------------------- | -------------------------------- |
| Unit         | Domain logic, parsing, validation                     | Go `testing`, pytest             |
| Integration  | Database, queues, storage, APIs                       | testcontainers-go, pytest-httpx  |
| Contract     | Country source adapters                               | Go interface assertions           |
| E2E          | Source → ingestion → data → AI → API → UI             | Playwright                       |
| AI evaluation | Factual accuracy, citation correctness, hallucination | Permanent eval dataset in `eval/` |

> No AI model/prompt change reaches production without evaluation.

## 16. Definition of Done

A feature is complete only when:
- ✅ Requirements are understood
- ✅ Issue exists
- ✅ Implementation exists
- ✅ Tests exist
- ✅ Integration works
- ✅ Security is considered
- ✅ Observability exists
- ✅ Documentation exists
- ✅ API contracts are documented
- ✅ Frontend works responsively
- ✅ Accessibility is considered (WCAG 2.2 AA target)
- ✅ PR is reviewed
- ✅ PR is merged
- ✅ Issue is closed

## 17. Roadmap (8 phases)

See [docs/architecture/13-roadmap.md](./docs/architecture/13-roadmap.md) for the full phase plan. Summary:

1. **Foundation** — monorepo, CI/CD, database, migrations, domain model, source registry, object storage, event infrastructure, Temporal, observability, Kenya adapter interface. *(in progress — see QA report)*
2. **Kenya Legislative Ingestion** — Parliament source discovery, Bills, Bill Trackers, Hansard, Order Papers, Votes & Proceedings, committees, Acts, Kenya Law integration, source archive, document versioning.
3. **Bill Intelligence** — summaries, stage explanations, timelines, citations, document comparison, related entities, Bill chat.
4. **Citizen Experience** — homepage, country selection, search, Bill discovery, Bill pages, timeline, document viewer, conversational interface.
5. **Monitoring** — following, notifications, stage-change detection, amendment detection, new-document detection, daily briefing.
6. **Civic Intelligence** — Acts, regulations, Gazette notices, policies, government publications, committee intelligence, public participation.
7. **Research Platform** — advanced search, historical analysis, cross-document comparison, entity graph, research workspaces, export, datasets, API.
8. **Global Expansion** — Uganda, Tanzania, Ghana, Nigeria, South Africa via the adapter pattern. Never add a country until its adapter has: authoritative sources, legislative ontology, stage definitions, document parsers, terminology, tests, data-quality rules.

## 18. The long-term north star

> A citizen types: **"What is my government doing about the cost of housing?"**
>
> The platform investigates across authorized sources — National Assembly Bills, Senate proceedings, executive policies, ministry programmes, regulator regulations, county initiatives, relevant judgments — and returns: a simple explanation, a timeline, and the evidence.

The platform should eventually allow a citizen to ask **"What's happening?"** and receive **a simple explanation, a timeline, and the evidence** — without needing to understand the architecture underneath. Build it like global civic infrastructure.
