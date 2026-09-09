# Changelog

All notable changes to the Civic Intelligence Platform are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added — Phase 1 (Foundation)

- **Monorepo structure**: `apps/web`, `services/{api,legislation,ingestion,documents,evidence,intelligence,search,notifications,identity,ai}`, `adapters/kenya/{parliament,kenya_law,gazette,internal}`, `packages/{contracts,events,observability,auth,config}`, `infrastructure/{postgres,docker,kubernetes,terraform,observability,temporal,opensearch}`, `docs/{architecture,adr,api,routes}`, `tests/{integration,e2e,evaluation,contract}`.
- **Python AI service** (`services/ai/`): FastAPI app exposing 12 civic AI capabilities (BillSummarizer, TimelineExtractor, StageExplainer, TerminologyExplainer, DocumentComparator, EntityExtractor, TopicClassifier, ImpactAnalyzer, CivicQuestionAnswerer, CitationValidator, ContradictionDetector, BriefingGenerator). Provider-agnostic model gateway with Stub + OpenAI providers, daily budget enforcement, retry, fallback. Full RAG pipeline with hybrid search, reranking, evidence selection, context construction, claim extraction, and mandatory citation validation. 28 unit + integration tests, all passing. Permanent evaluation dataset (`eval/test_eval_dataset.py`) covering factual accuracy, citation correctness, and hallucination guardrails.
- **Next.js frontend** (`apps/web/`): Next.js 14 + Tailwind CSS + TanStack Query + lucide-react. 17 routes serving HTTP 200 — homepage, /bills, /bills/[id] (with timeline/versions/compare/documents/chat sub-routes), /search, /briefing, /about, /committees, /people, /institutions, /country, /country/kenya, /topics, and a custom 404. All pages type-check (`tsc --noEmit`), lint cleanly, and pass a production build. Accessibility: skip-link, ARIA landmarks, focus-visible outlines, reduced-motion support.
- **Database schema**: 16 forward-only Postgres migrations covering 9 logical schemas (legislation, ingestion, documents, evidence, intelligence, notifications, identity, search, audit). pgvector extension enabled. ivfflat + GIN trigram + FTS indexes on document chunks. Audit triggers on canonical tables. CHECK constraint on `intelligence.candidate_facts` enforcing "AI cannot mutate truth".
- **Kenya adapter** (`adapters/kenya/`): Reference implementation of `contracts.LegislativeSourceAdapter`. 11 Bill stages per the 2010 Constitution (FIRST_READING through COMMENCEMENT, plus REJECTED/WITHDRAWN/LAPSED terminals). 30 parliamentary terms with plain-language explanations + source URLs. Bicameral structure (National Assembly + Senate) and 14 committees. Contract tests verify interface compliance, stage transition validity, and terminology completeness. Source adapters (parliament, kenya_law, gazette) defined with polite HTTP client.
- **Go backend foundation**: Root `go.mod` + `packages/contracts/` (country adapter interface, events catalog, typed errors, pagination). `services/legislation/` with `BillStateMachine` (country-agnostic — takes `[]StageDefinition` as input), entities, repository interfaces, `CreateBill` use case, and unit tests verifying valid + invalid transitions. Compile-time assertion that `KenyaAdapter` satisfies the contract interface.
- **Infrastructure**: Docker Compose for the full local dev stack (Postgres+pgvector, Redis, NATS JetStream, MinIO, OpenSearch, Temporal + UI, Keycloak, MailHog, AI service, web). Multi-stage Dockerfiles for Go, Python, and Next.js. CI pipeline via GitHub Actions (Python tests, web lint+build, migration apply against real Postgres, Trivy security scan, Docker image build, CodeQL).
- **Documentation**: README, ARCHITECTURE.md (the canonical contract — the "AI cannot mutate truth" rule, service ownership table, dependency direction, country-adapter pattern, event catalog, Temporal workflows, deployment topology), CONTRIBUTING, SECURITY (threat model + controls + token rotation policy), CODE_OF_CONDUCT. 14 Architecture Decision Records (MADR format): modular monolith, Go+Python split, single Postgres cluster, country adapter pattern, AI cannot mutate truth, NATS JetStream, Temporal, evidence-first RAG, pgvector before OpenSearch, OIDC/Keycloak, immutable Bill versions, SSE for streaming, contradiction engine, monorepo. Partial OpenAPI 3.1 spec for the citizen API.
- **CI/CD**: GitHub Actions for `pull_request` and `push: main`. CodeQL weekly + on PRs. Dependabot weekly for pip, npm, Go modules, Docker, GitHub Actions. Issue templates (bug report + feature request) and a PR template enforcing the architectural checklist.

### Known limitations (Phase 1)

- Go backend services beyond `legislation` are directory-stubbed only. Full source for `api`, `ingestion`, `documents`, `evidence`, `intelligence`, `search`, `notifications`, `identity` is tracked in issue #CI-BE-001.
- Kenya adapter source adapters (parliament, kenya_law, gazette) have interface + fetch stubs but no real crawling/parsing yet. Tracked in issues #CI-AD-002..#CI-AD-008.
- Temporal workflows, Helm chart, and Terraform modules are file stubs. Tracked in issues #CI-INF-001..#CI-INF-003.
- The Next.js frontend ships with mock data. Real data requires the Go BFF + ingestion pipeline to be completed (Phase 2).
- The AI service uses the Stub provider by default. Configure `OPENAI_API_KEY` or `ANTHROPIC_API_KEY` in `.env` to use a real provider.

## [0.1.0] — 2026-09-09

Initial public release of the Civic Intelligence Platform repository structure.
