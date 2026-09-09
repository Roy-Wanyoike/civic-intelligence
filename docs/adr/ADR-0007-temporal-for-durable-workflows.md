# ADR-0007: Temporal for durable workflows

## Status

Accepted — 2026-09-09

## Context

The Bill ingestion pipeline is a long-running, multi-step process: discover → fetch → archive → parse → extract → validate → version → index → embed → generate explanation → validate citations → publish. Each step can fail and must be retryable. Options:

1. **In-process orchestration** — simple but no durability, no visibility, hard to recover from crashes.
2. **Temporal** — durable workflows as code, retries, timeouts, observability, replay.
3. **Airflow / Prefect / Dagster** — data-pipeline focused, heavier, less suitable for service-oriented workflows.

## Decision

Adopt **Temporal** for durable workflows.

The Kenya Bill ingestion pipeline becomes a `KenyaBillIngestionWorkflow` in code. Each step is an activity. Temporal handles retries, timeouts, and recovery.

## Consequences

- **Positive**: durable — if the worker crashes mid-workflow, Temporal replays from the last completed activity.
- **Positive**: observable — Temporal UI shows every workflow's state, inputs, outputs, failures.
- **Positive**: programmable in Go (native SDK) — workflows + activities are plain Go code.
- **Positive**: versioning — workflow changes are versioned via `WorkflowID` + `RunID`.
- **Negative**: extra infrastructure to operate (Temporal server + its own Postgres). Mitigated by the `temporalio/auto-setup` image which handles schema setup.
- **Negative**: workflow code must be deterministic — no random, no time.Now(), no network calls inside workflow code (those go in activities).

## Boundary

Temporal knows **what workflow is happening**. The Legislative service knows **what a valid Bill state is**. These responsibilities stay separate. Temporal never directly updates `legislation.*` — it calls the legislation service's application layer.

## References

- ARCHITECTURE.md §8 (Workflow orchestration)
- `infrastructure/docker/docker-compose.yml` (Temporal service + UI)
