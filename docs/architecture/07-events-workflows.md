# 07 — Events and Workflows

This document is the canonical event catalog and workflow reference. Every event published on the NATS JetStream cluster is listed here with its publisher, subscribers, and payload schema. Every durable workflow orchestrated by Temporal is listed here with its triggers, steps, and failure modes.

The corresponding decision records are [ADR-0006](../adr/ADR-0006-nats-jetstream-for-events.md) (NATS) and [ADR-0007](../adr/ADR-0007-temporal-for-durable-workflows.md) (Temporal).

## Command vs. event distinction

The platform distinguishes **commands** from **events**. A command is a directed request to do something ("fetch this source," "validate this candidate fact"). An event is a fact about something that has already happened ("source fetched," "candidate fact validated"). Commands are sent over gRPC (synchronous) or via a work queue (async); events are published to NATS JetStream (async, broadcast).

The rule: services publish events; they consume commands (from the API, from Temporal workflows, from other services' gRPC). A service never publishes a command — commands come from outside. A service never consumes an event by sending a command back — it consumes the event and performs its own work, which may publish further events.

This separation prevents the "event-callback soup" where every event triggers a command that triggers another event ad infinitum. Each event has at most a handful of consumers; each consumer does one thing; the chain terminates.

## Event catalog

Events are grouped by publishing service. Each event has a name (in `PascalCase`), a publisher, a list of subscribers, and a payload schema (in Avro or JSON Schema, stored in `packages/events/schemas/`).

### Ingestion events

| Event | Publisher | Subscribers | Payload | Notes |
| --- | --- | --- | --- | --- |
| `SourceFetched` | ingestion worker | documents worker, evidence worker | `source_id`, `source_document_id`, `content_hash`, `fetched_at`, `mime_type` | One per successful fetch. Triggers parsing. |
| `SourceFetchFailed` | ingestion worker | notifications (for editors), observability | `source_id`, `error`, `attempt_count`, `next_retry_at` | Polite backoff; alerts on repeated failure. |
| `SourceChanged` | ingestion worker | notifications, search | `source_id`, `old_hash`, `new_hash`, `diff_summary` | Snapshot comparison detected a change. |

### Documents events

| Event | Publisher | Subscribers | Payload | Notes |
| --- | --- | --- | --- | --- |
| `DocumentParsed` | documents worker | evidence worker, search, intelligence | `document_id`, `sections`, `chunks`, `parse_confidence` | Sections and chunks are addressable; downstream services resolve them. |
| `DocumentParseFailed` | documents worker | notifications (editors), observability | `document_id`, `error`, `stage` | Stage = `extraction` / `ocr` / `chunking`. |
| `EmbeddingsGenerated` | documents worker | search | `document_id`, `chunk_count`, `model_version`, `vector_dim` | Embeddings stored in pgvector. |

### Legislation events

| Event | Publisher | Subscribers | Payload | Notes |
| --- | --- | --- | --- | --- |
| `BillIntroduced` | legislation (via ingestion adapter) | search, notifications, intelligence | `bill_id`, `country`, `house_id`, `sponsor_id`, `introduced_at` | The canonical record is created. |
| `BillVersionPublished` | legislation | documents (parse new version), evidence, search, intelligence, notifications | `bill_id`, `bill_version_id`, `version_number`, `published_at` | New immutable version. |
| `BillStageTransitioned` | legislation | search, notifications, intelligence | `bill_id`, `from_stage`, `to_stage`, `transitioned_at`, `source_citation_id` | Must have a citation (OrderPaper, Hansard, or Gazette). |
| `AmendmentTabled` | legislation | documents (parse amendment), evidence, intelligence, notifications | `bill_id`, `amendment_id`, `sponsor_id`, `target_clause`, `tabled_at` | |
| `ActGazetted` | legislation | search, notifications, intelligence | `act_id`, `bill_id`, `gazette_notice_id`, `gazetted_at` | Bill graduates to Act. |
| `VoteRecorded` | legislation | search, notifications, intelligence | `bill_id`, `vote_id`, `house_id`, `outcome`, `votes_for`, `votes_against`, `recorded_at` | |
| `BillEventOccurred` | legislation | search, notifications | `bill_id`, `event_type`, `occurred_at`, `source_citation_id` | Generic timeline event. |

### Evidence events

| Event | Publisher | Subscribers | Payload | Notes |
| --- | --- | --- | --- | --- |
| `EvidenceAttached` | evidence worker | search, intelligence, notifications | `claim_id`, `evidence_id`, `citation_ids` | A claim now has backing. |
| `CandidateFactProposed` | intelligence (gateway) | evidence worker | `candidate_fact_id`, `capability`, `claim_text`, `citations` | AI proposes a fact. |
| `CandidateFactValidated` | evidence worker | intelligence, legislation, notifications | `candidate_fact_id`, `status: accepted`, `claim_id` | Validated; becomes a Claim. |
| `CandidateFactRejected` | evidence worker | intelligence, observability | `candidate_fact_id`, `status: rejected`, `reason`, `failed_citations` | Rejection is feedback. |
| `SourceConflictDetected` | evidence worker | notifications (editors), intelligence, observability | `conflict_id`, `claim_a_id`, `claim_b_id`, `conflict_type`, `description` | See [ADR-0013](../adr/ADR-0013-contradiction-engine-do-not-silently-resolve.md). |
| `FactAccepted` | evidence worker (validator) | legislation (for canonical writes), notifications | `claim_id`, `evidence_ids`, `accepted_by`, `accepted_at` | A fact entered canonical truth. |
| `FactRetracted` | legislation | evidence, search, notifications, intelligence | `claim_id`, `reason`, `retracted_by`, `retracted_at` | A fact was wrong; retracted with reason. |

### Intelligence events

| Event | Publisher | Subscribers | Payload | Notes |
| --- | --- | --- | --- | --- |
| `AIResponseGenerated` | intelligence (gateway) | notifications (for the user, if async), observability | `ai_response_id`, `capability`, `model_version`, `prompt_version`, `claim_count`, `citation_count` | |
| `AIEvalCompleted` | intelligence (eval runner) | observability, maintainers (via notifications) | `eval_run_id`, `capability`, `pass_rate`, `hallucination_rate`, `latency_p50`, `cost_per_call` | Triggers alert if pass rate drops. |

### Search events

| Event | Publisher | Subscribers | Payload | Notes |
| --- | --- | --- | --- | --- |
| `SearchIndexUpdated` | search worker | observability | `index_name`, `entity_type`, `entity_id`, `updated_at` | Projection updated. |

### Notifications events

| Event | Publisher | Subscribers | Payload | Notes |
| --- | --- | --- | --- | --- |
| `NotificationDispatched` | notifications worker | observability | `notification_id`, `user_id`, `channel`, `latency_ms`, `status` | |
| `NotificationFailed` | notifications worker | observability | `notification_id`, `user_id`, `channel`, `error`, `retry_count` | Retries with exponential backoff. |

### Identity events

| Event | Publisher | Subscribers | Payload | Notes |
| --- | --- | --- | --- | --- |
| `UserCreated` | identity | notifications (welcome), observability | `user_id`, `created_at` | |
| `UserPreferenceChanged` | identity | notifications (re-schedule digests), search (re-personalize) | `user_id`, `preference_key`, `old_value`, `new_value` | |
| `APIKeyRotated` | identity | observability, security audit | `api_key_id_prefix`, `rotated_by`, `rotated_at` | Never logs the key itself. |
| `RoleBindingChanged` | identity | api (invalidate auth cache), observability, security audit | `user_id`, `role`, `scope`, `action: granted/revoked` | |

## Temporal workflows

Temporal orchestrates the long-running, multi-step flows that span services. A workflow is durable: if a worker crashes mid-execution, Temporal resumes from the last completed step. Workflows are written in Go in `services/<service>/workflows/` and registered with the Temporal cluster at worker startup.

The platform's first workflow is `KenyaBillIngestionWorkflow`, which orchestrates the full pipeline from fetch to indexed-and-summarized for a single bill. Subsequent countries will have `<Country>BillIngestionWorkflow` with the same shape but country-specific adapters plugged in.

### KenyaBillIngestionWorkflow

**Trigger:** a scheduled crawl, an on-demand refetch, or a `SourceChanged` event for a bill's source page.

**Steps:**

1. **Fetch.** Call the Kenya adapter's `SourceAdapter.Fetch` for the bill's source. Produces a `SourceDocument`.
2. **Parse.** Call the documents worker to parse the `SourceDocument` into sections and chunks. Produces a `Document` and publishes `DocumentParsed`.
3. **Identify.** Call the legislation service to identify the bill and version: is this a new bill, a new version of an existing bill, or an unchanged re-fetch? Produces (or updates) a `Bill` and `BillVersion`.
4. **Extract evidence.** Call the evidence worker to extract claims and citations from the parsed document. Publishes `EvidenceAttached`.
5. **Index.** Call the search worker to update the search index for this bill. Publishes `SearchIndexUpdated`.
6. **Summarize.** Call the intelligence gateway's `BillSummarizer` capability to generate (or update) the bill's plain-language summary. Publishes `AIResponseGenerated` and `CandidateFactProposed` (for any candidate facts the summary asserts).
7. **Notify.** Call the notifications worker to dispatch notifications to citizens who follow this bill. Publishes `NotificationDispatched`.

**Failure modes:**

- **Fetch failure (network, robots, source down).** Retry with backoff up to 5 attempts; if all fail, publish `SourceFetchFailed` and surface to editors.
- **Parse failure (corrupt PDF, OCR failure).** Retry once with a different parser; if that fails, publish `DocumentParseFailed` and surface. The bill record exists but with no parsed content.
- **Identification failure (ambiguous bill, missing metadata).** Surface to editors for manual identification. Do not guess.
- **Evidence failure (citations don't resolve, contradictions detected).** The bill is canonical but candidate facts are rejected. Publish `CandidateFactRejected` and `SourceConflictDetected` as appropriate.
- **Summarize failure (LLM timeout, hallucination).** Retry once; if that fails, the bill is searchable but the summary is missing. Surface to maintainers; do not block the rest of the pipeline.

**Idempotency:** every step is idempotent. Re-running the workflow for the same bill version produces the same end state (modulo timestamps and any new evidence that arrived between runs). The content hash on the `SourceDocument` short-circuits re-parsing if the source is unchanged.

### Other workflows (planned)

| Workflow | Service | Status | Purpose |
| --- | --- | --- | --- |
| `KenyaBillIngestionWorkflow` | ingestion | Phase 2 | Full pipeline for a Kenya bill. |
| `GazetteIngestionWorkflow` | ingestion | Phase 6 | Parse a gazette notice into acts, regulations, appointments. |
| `HansardIngestionWorkflow` | documents | Phase 6 | Parse a Hansard PDF into speeches, votes, motions. |
| `EvaluationRunWorkflow` | intelligence | Phase 1 | Run the eval suite against a candidate prompt/model change. |
| `DailyDigestWorkflow` | notifications | Phase 4 | Roll up the day's notifications into a per-user digest. |
| `BackfillWorkflow` | ingestion | Phase 2 | Re-run the pipeline for a historical date range. |

## Work queue vs. event stream

Some work is triggered by events (a `BillVersionPublished` event triggers the search worker to re-index). Other work is triggered by a work queue (a citizen's `Ask` request is enqueued and the AI worker picks it up). The distinction:

- **Events** are broadcast. Many subscribers may respond. The publisher does not wait.
- **Work queue items** are point-to-point. Exactly one worker picks up each item. The caller may wait (synchronous AI calls) or may not (async batch jobs).

NATS JetStream supports both patterns: the `pub-sub` mode for events and the `queue-group` mode for work distribution. Both use the same infrastructure; the difference is the consumer's subscription mode.

## Schema evolution

Event schemas evolve. New fields are added (with defaults, marked optional) without breaking older consumers. Removed fields are first deprecated (still present, marked deprecated) for one release cycle, then removed. Major schema changes require a new event name (`BillVersionPublished` → `BillVersionPublishedV2`) so old and new consumers can coexist during migration. Schemas are versioned in `packages/events/schemas/` with SemVer; the registry enforces backward compatibility on every PR.
