# 02 — Services

This document describes each service in the platform: what it owns, what it explicitly does not own, its key entities, the events it publishes and consumes, and its deployment shape. The service ownership boundaries are the architectural contract — they are enforced by code review, by CI (Semgrep rules), and by Postgres role grants. Cross the boundary and the PR will not merge.

The 10 services (9 Go + 1 Python) are:

1. [legislation](#legislation) — the canonical civic domain
2. [ingestion](#ingestion) — fetches external sources
3. [documents](#documents) — parses raw files into structured content
4. [evidence](#evidence) — claims, citations, contradictions
5. [intelligence](#intelligence-go) — AI gateway and orchestration
6. [ai](#ai-python) — Python AI workers (capabilities, prompts, evals)
7. [search](#search) — search projections and indexes
8. [notifications](#notifications) — follows and delivery
9. [identity](#identity) — users and preferences
10. [api](#api) — HTTP BFF

Each section is roughly 250 words.

---

## legislation

`services/legislation/` is the canonical civic domain. It owns the entities that everything else in the platform is a projection of: `Country`, `Institution`, `Legislature`, `House`, `Committee`, `Person`, `Bill`, `BillVersion`, `BillStage`, `BillEvent`, `Amendment`, `Clause`, `Act`, `Regulation`, `Policy`, `GovernmentDecision`, `HansardDocument`, `OrderPaper`, `CommitteeReport`, `Vote`, `VotesProceeding`, `GazetteNotice`, `PublicParticipation`. If a piece of civic truth exists in the platform, it lives in a `legislation` table.

What it owns: the schema for those entities, the business rules that govern their transitions (e.g. a `BillVersion` cannot move from `draft` to `assented` without passing through `introduced`), and the event emitters that notify the rest of the platform when canonical state changes. It also owns the audit log — every write to a canonical table is recorded immutably.

What it does NOT own: parsed document content (that is `documents`), citations and evidence (that is `evidence`), AI explanations (that is `intelligence`/`ai`), search indexes (`search`), or how citizens are notified (`notifications`). It does not call out to LLMs. It does not fetch from external sources. It is the innermost ring of the dependency graph.

Key events published: `BillIntroduced`, `BillVersionPublished`, `BillStageTransitioned`, `AmendmentTabled`, `ActGazetted`, `VoteRecorded`, `BillEventOccurred`. Key events consumed: `FactAccepted` (from `evidence`, to turn accepted candidate facts into canonical rows when appropriate — but only through an explicit validation step, never automatically from AI).

Deployment shape: runs in-process with the Civic API for synchronous reads; runs as a library inside workers that need to write canonical state (with a tight role grant). It is not a separate process — it is a Go package that other services import.

---

## ingestion

`services/ingestion/` acquires information from external civic sources. It owns the `Source` registry (the catalog of parliament sites, gazette portals, Hansard archives, and committee pages the platform pulls from), the `SourceDocument` records (a fetched-but-not-yet-parsed blob plus its metadata), and the `SourceSnapshot` records (a versioned capture of what a source page looked like at fetch time, used to detect changes and to establish provenance).

What it owns: the fetchers (HTTP clients with retry, backoff, robots-policy respect), the scheduler (cadence per source — daily, hourly, on-event), the raw document store (S3-compatible blob storage with content-addressed hashes), the SSRF allowlist (per-source host/scheme/port rules), and the fetch budget (per-source rate limiting to be a polite crawler). It owns the country-adapter plugin interface — Kenya's fetchers live in `adapters/kenya/` and plug in here.

What it does NOT own: parsing the fetched documents (that is `documents`), deciding what the document means (`evidence`/`intelligence`), or what the canonical bill/act/etc. is (`legislation`). It acquires bytes; it does not interpret them.

Key events published: `SourceFetched` (with a `SourceDocument` id and content hash), `SourceChanged` (when a snapshot differs from the prior one), `SourceFetchFailed` (with the failure reason). Key events consumed: none — ingestion is the start of the pipeline. It does consume Temporal signals for on-demand refetch.

Deployment shape: a standalone worker process with a dedicated network namespace that has no route to internal services except through explicit proxies. This is the SSRF isolation boundary. Runs at low concurrency (crawls are I/O-bound and polite) and scales horizontally by source partition.

---

## documents

`services/documents/` takes raw fetched blobs and turns them into structured, addressable content. It owns `Document`, `DocumentPage`, `DocumentSection`, `DocumentChunk` (the unit of retrieval for RAG), and the OCR/parsing pipeline that produces them. A document is parsed exactly once into its canonical structured form; subsequent fetches of the same source compare hashes and skip re-parsing if unchanged.

What it owns: the parsers (PDF text extraction with layout preservation, OCR via Tesseract for scanned PDFs, HTML sanitization and structuring, DOCX parsing), the chunking strategy (recursive character splitting with clause-aware boundaries for bills), the embedding generation (calls the embedding model and stores vectors in pgvector), and the parse-quality metrics (parse confidence, OCR failure rate, missing-page detection).

What it does NOT own: what the document *is* (a Bill, an Act, a Hansard — that is `legislation`'s call), what claims it makes (`evidence`), or what it means (`intelligence`/`ai`). It produces structured text and embeddings; it does not interpret.

Key events published: `DocumentParsed` (with the document id, sections, and chunk count), `DocumentParseFailed` (with the failure reason), `EmbeddingsGenerated` (with the vector count and model version). Key events consumed: `SourceFetched` (to start parsing a newly fetched document).

Deployment shape: a standalone worker process, CPU-heavy (OCR and parsing are compute-bound), runs on nodes with adequate CPU and memory. Scales horizontally by document partition. May use GPU nodes for OCR acceleration if the volume justifies it.

---

## evidence

`services/evidence/` establishes provenance. It owns the chain that connects every claim the platform makes to the specific page and section of the specific document that supports it. The core entities are `Claim` (a single factual assertion), `Evidence` (a piece of supporting material linked to a claim), `Citation` (the resolved pointer to a `DocumentPage`/`DocumentSection`), `SourceConflict` (a record of two pieces of evidence that disagree), and the validation logic that decides whether a `CandidateFact` from the AI becomes a `Claim` in the canonical domain.

What it owns: the citation resolution logic (turning a `(document_id, page, section)` tuple into a verified pointer), the contradiction engine (comparing new claims against existing ones and recording conflicts instead of resolving them silently), the candidate-fact validator (which checks that every citation resolves, that the cited text actually supports the claim, and that the claim does not contradict an accepted fact without explanation), and the citation quality scoring (how strong is the chain from claim to source).

What it does NOT own: the canonical legislative entities (`legislation`), the AI that produced the candidate fact (`intelligence`/`ai`), or the document content itself (`documents`). It is the connective tissue between claims and documents.

Key events published: `EvidenceAttached` (a claim now has backing evidence), `CandidateFactValidated` (a candidate passed validation, status: accepted), `CandidateFactRejected` (with rejection reason), `SourceConflictDetected` (two pieces of evidence disagree). Key events consumed: `DocumentParsed` (to start extracting claims), `CandidateFactProposed` (from `intelligence`, to validate).

Deployment shape: runs in-process with the Civic API for synchronous validation calls, and as a library inside the evidence worker for async validation of large candidate batches.

---

## intelligence (Go)

`services/intelligence/` is the AI gateway. It is provider-agnostic and capability-based: a citizen request (or a worker job) says "run the `BillSummarizer` capability on bill X," and the gateway routes it to the right implementation (in-process or in the Python `ai` worker), tracks the eval result, enforces citation validation on the output, and publishes the result. It owns the capability catalog, the rate limiting and budgeting for AI calls, the prompt template registry (versioned, with a diff between versions), and the eval harness triggers.

What it owns: the gateway HTTP/gRPC surface that workers and the API call, the capability registry, the prompt-template loader and versioner, the model-router (which provider/model handles which capability), the budget tracker (per-tenant per-day token spend), and the eval orchestrator (which triggers an eval run when a prompt or model changes).

What it does NOT own: the actual model execution (that is `ai`), the prompts themselves (they live in `services/ai/prompts/`, versioned with the capability), the citations storage (`evidence`), or any canonical truth (`legislation`). It is the orchestration layer; it does not call models directly except in trivial cases.

Key events published: `CandidateFactProposed` (an AI output that should become a canonical fact, sent to `evidence` for validation), `AIResponseGenerated` (an AI response ready to return to the user or store), `AIEvalCompleted` (an eval run finished, with pass/fail). Key events consumed: `FactAccepted`/`FactRejected` (from `evidence`), to update the AI response's citation status.

Deployment shape: runs in-process with the Civic API for synchronous citizen-facing AI calls (the `/bills/{id}/questions` SSE endpoint), and as a library inside the AI worker for async batch processing.

---

## ai (Python)

`services/ai/` is the Python AI worker. It owns the actual model calls, the prompt templates, the RAG retrieval logic, the 12 civic AI capabilities' implementations, and the permanent evaluation dataset. This is the one service where "fast" is not the primary goal — correctness, citation-ability, and eval-ability are.

What it owns: the prompt template files (Jinja2, versioned, with metadata for capability, model, and language), the retrieval pipeline (query expansion, hybrid search over pgvector + FTS, reranking, evidence selection), the LLM client (provider-agnostic with fallback), the capability implementations (`BillSummarizer`, `TimelineExtractor`, `AmendmentDiffExplainer`, `PlainLanguageExplainer`, `StakeholderExtractor`, `VoteExplainer`, `CommitteeReportSummarizer`, `HansardQuoteExtractor`, `ContradictionDetector`, `TopicClusterer`, `RelatedBillsFinder`, `Ask`), the eval dataset (golden inputs and expected behaviors), and the eval runner.

What it does NOT own: the canonical truth (`legislation` — the AI role has no `UPDATE` on any canonical table), the citation storage (`evidence`), the search indexes (`search`), or the user-facing API (`api`). It connects to Postgres with a role that has `SELECT` on `legislation.*` and `evidence.*` and `INSERT` only on `intelligence.candidate_facts` and `intelligence.ai_responses`.

Key events published: `CandidateFactProposed`, `AIResponseGenerated` (both via the `intelligence` gateway). Key events consumed: `BillVersionPublished`, `AmendmentTabled`, `HansardPublished` (to trigger background summarization jobs).

Deployment shape: a standalone Python worker process, optionally on GPU-capable nodes for in-house model hosting. Scales horizontally by capability partition. Connects to model providers through the same egress-isolated network as ingestion, so a compromised model provider cannot reach internal services.

---

## search

`services/search/` is the read-side projection store. It owns the indexes that power citizen-facing search: full-text search over Bills and Acts, vector search over document chunks for RAG retrieval, faceted search (by house, by stage, by topic, by date range), and the query planner that turns a citizen's search into the right combination of FTS + vector + facet lookups. It is a pure projection — it consumes events and rebuilds its indexes; it never publishes domain events.

What it owns: the index builders (Postgres FTS triggers, pgvector index updates, OpenSearch indexers for faceted search when needed), the query planner (decides which indexes to hit based on the query shape), the ranking model (BM25 for FTS, cosine similarity for vectors, learned reranker for the final order), and the search-result hydration (turning index hits into API-shaped objects by fetching from `legislation`).

What it does NOT own: the source data (`legislation`), the document content (`documents`), or the AI interpretation (`intelligence`/`ai`). It builds projections; it does not own the truth they project from.

Key events published: `SearchIndexUpdated` (an index was rebuilt for a particular entity). Key events consumed: `BillVersionPublished`, `BillStageTransitioned`, `EvidenceAttached`, `DocumentParsed` — essentially every canonical-change event triggers a projection update.

Deployment shape: runs in-process with the Civic API for query serving (low-latency reads), and as a library inside a search worker for async index rebuilds. OpenSearch, when used, runs as a separate cluster shared across environments.

---

## notifications

`services/notifications/` distributes verified changes to subscribers. A citizen follows a bill, a committee, a person, or a topic; when a canonical change occurs that matches their follow, they get a notification. It owns the `Follow` records (who follows what, with what filters), the `Notification` records (a unit of delivery — a subject, a body, a target), the delivery channels (in-app, email, push, webhook), and the retry/backoff logic for delivery failures.

What it owns: the follow registry, the notification templating (turning a canonical event into a human-readable notification), the delivery adapters (per-channel), the per-user digest scheduling (daily/weekly rollups), and the delivery metrics (success rate, latency, bounce rate).

What it does NOT own: what the notification is about (the canonical event came from `legislation`), who the user is (`identity`), or the search index that helps a user discover what to follow (`search`). It reacts to events; it does not originate them.

Key events published: `NotificationDispatched` (with channel, status, latency), `NotificationFailed` (with reason). Key events consumed: every canonical-change event — `BillIntroduced`, `BillStageTransitioned`, `VoteRecorded`, `ActGazetted`, `HansardPublished`, and so on.

Deployment shape: a standalone worker process. Scales horizontally by follow-partition. Delivery adapters may run as separate sub-processes for channel isolation (an email outage should not block push notifications).

---

## identity

`services/identity/` owns users and their preferences. It does not own authentication — that is delegated to Keycloak (OIDC). It owns the local user record (a profile that mirrors the OIDC identity, plus platform-specific fields), the user preferences (language, notification cadence, followed topics, accessibility settings), the API keys (for partner integrations), and the role bindings (citizen, editor, maintainer, admin — scoped per country, per institution, or globally).

What it owns: the `User` profile (local fields, not the OIDC identity), the `UserPreference` set, the `APIKey` records (hashed, scoped, with rotation policy), the `RoleBinding` records (user × role × scope), and the audit trail for identity changes.

What it does NOT own: authentication (Keycloak does), authorization policy (the policy is defined in `packages/contracts/` and enforced by middleware in `services/api/`), or any civic domain state. It is identity-adjacent, not identity-authoritative.

Key events published: `UserCreated`, `UserPreferenceChanged`, `APIKeyRotated`, `RoleBindingChanged`. Key events consumed: none — identity is the start of the user-side pipeline.

Deployment shape: runs in-process with the Civic API for synchronous profile reads and writes. No standalone worker; identity operations are low-volume and synchronous.

---

## api

`services/api/` is the HTTP Backend-for-Frontend (BFF). It is the only citizen-facing synchronous surface. It validates requests, authenticates via OIDC tokens (validated against Keycloak's JWKS), authorizes via the role bindings from `identity`, calls the domain services (in-process Go calls) and the search service, shapes the response, and applies rate limiting. It does **not** open a SQL connection to any schema — it calls services, which call the DB with their own roles.

What it owns: the HTTP routes (documented in [`docs/api/openapi.yaml`](../api/openapi.yaml)), the request validation (via generated validators from the OpenAPI spec), the response shaping (DTOs that hide internal structure), the rate limiter (per-IP and per-token tiers), the SSE handler for streaming AI responses, and the observability middleware (OpenTelemetry spans, structured logging, metrics).

What it does NOT own: business logic (that lives in the domain services), database access (none — it calls services), AI execution (it calls `intelligence`, which calls `ai`), or any canonical state. It is a thin BFF; if it grows business logic, that is a smell.

Key events published: none — the API is synchronous and does not publish events. (It may publish `APIRequestCompleted` metrics, but those are observability, not domain events.) Key events consumed: none — the API calls services directly.

Deployment shape: a standalone Go HTTP server, horizontally scalable behind a load balancer. The user-facing deployable. Runs with the `legislation`, `identity`, `search`, and `intelligence` packages compiled in for in-process calls.
