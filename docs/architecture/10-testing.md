# 10 — Testing

This document describes the platform's testing strategy: the layers (unit, integration, contract, e2e, AI-eval), the permanent AI evaluation dataset, and the rule that gates AI changes. The testing philosophy is: tests protect the architectural contract. A test that does not protect the contract is a test that can be deleted; a contract element without a test is a contract element that will break.

The platform has five test layers, each with a different purpose, different cost, and different cadence. The layers run in CI on every PR; the slower layers also run on a schedule against `main` to catch regressions that slip through.

## Test layers

### 1. Unit tests

Unit tests cover individual functions and types. They run in milliseconds, never touch the network or the database, and are the first line of defense against logic regressions. In Go they use the standard `testing` package; in Python they use `pytest`; in TypeScript they use Vitest.

Examples: a `BillStage` transition validator, a citation resolver's URL normalization, a JSONB column's marshal/unmarshal round-trip, a parser's clause-extraction function. Each unit test should cover one behavior; a function with three branches gets three tests.

Unit tests live next to the code they test (`foo_test.go` next to `foo.go`). They run on every PR and on every save locally (via `make test-watch`). The target is 90% line coverage for the domain packages; lower coverage is acceptable for plumbing (HTTP handlers, glue code) but is flagged for review.

### 2. Integration tests

Integration tests cover the interaction between a service and its real dependencies (Postgres, NATS, Temporal) but not other services. They run in a Docker Compose environment spun up by CI: a real Postgres cluster with migrations applied, a real NATS JetStream instance, a real Temporal cluster. The service under test connects to these; the test exercises the service's public API and asserts the observable behavior.

Examples: insert a `BillVersion` via the legislation service, query it back, assert the fields; publish a `BillVersionPublished` event, consume it via the search worker, assert the index was updated; run a `KenyaBillIngestionWorkflow` against a test fixture, assert the final state.

Integration tests live in `services/<service>/integration/`. They run on every PR (the Docker Compose spin-up is fast — under 30 seconds — because the images are cached). Slower integration tests (full-workflow runs) are tagged `slow` and run on PRs that touch the relevant code plus nightly on `main`.

### 3. Contract tests

Contract tests protect the boundaries between services and between the platform and its consumers. They come in two flavors:

- **Producer contract tests.** For each event published and each gRPC/HTTP endpoint exposed, a test asserts that the message matches the schema in `packages/events/` or `packages/contracts/`. A producer that drifts from the contract fails CI.
- **Consumer contract tests.** For each event consumed and each gRPC/HTTP endpoint called, a test asserts that the consumer handles the producer's schema correctly — including the producer's documented future schemas (additive changes that the consumer must tolerate).

Contract tests are generated from the schemas in `packages/events/schemas/` and `packages/contracts/`. Adding a field to an event schema requires a producer contract test update; removing a field requires a deprecation cycle. Contract tests are the technical enforcement of the schema-evolution policy described in [`07-events-workflows.md`](./07-events-workflows.md).

### 4. End-to-end (e2e) tests

E2e tests cover the full citizen-facing flow: a user opens the browser, navigates to a bill, reads the summary, asks a question, sees the answer with citations. They run against a fully-deployed environment (a Docker Compose cluster for PR e2e; a staging cluster for pre-release e2e) and exercise the real frontend, real API, real workers, real database.

E2e tests are written in Playwright (for the web app) and in Go (for API-level e2e). They are slow (minutes per test) and brittle (any change to the UI can break them), so they are scoped to the most important flows: homepage loads, bill page loads, search returns results, ask-a-question returns a cited answer, follow-bill triggers a notification on a change. The target is ~50 e2e tests covering the critical paths, not exhaustive coverage.

E2e tests run on PRs that touch the web app or the API, on every commit to `main` (nightly), and as a pre-release gate before any production deploy. A red e2e test blocks the release.

### 5. AI evaluation tests

AI evaluation tests are the most important and most distinctive layer. They are described in detail below.

## The permanent AI evaluation dataset

Every AI capability ships with a permanent evaluation dataset: a curated set of inputs and expected behaviors. The dataset is the contract between the AI subsystem and the rest of the platform — it defines what "good" looks like for each capability.

### What the dataset contains

For each capability, the dataset contains:

- **Inputs.** Real or realistic inputs: a real bill version, a real citizen question, a real pair of bill versions to diff. Inputs are anonymized where they contain PII; otherwise they are the real thing.
- **Expected behavior.** A structured description of what the output should look like: which claims should appear, which citations should be present (with the specific document/page/section), which key points should be covered, which hallucinations should not appear. Expected behavior is written by humans and reviewed.
- **Adversarial cases.** Inputs designed to test the capability's robustness: prompt-injection attempts, ambiguous questions, missing-data scenarios, contradictions in the source. The expected behavior for these is often "the capability should refuse / flag / ask for clarification."
- **Edge cases.** Inputs at the boundaries: very short bills, very long bills, bills with unusual structure, bills in a non-default language.

### How the dataset is curated

The dataset is curated by maintainers and domain experts. New cases are added when:

- A user reports a poor AI output (the case is added with the corrected expected behavior).
- A new capability is added (at least 10 initial cases).
- A new failure mode is discovered (an adversarial case is added to prevent regression).
- The platform expands to a new country or domain (cases for the new context are added).

Cases are never silently removed. A case that is found to be wrong is updated with a recorded reason; a case that is no longer relevant is marked deprecated (still runs, but a failure is not blocking) with a recorded reason. The dataset's growth is monotonic.

### The eval runner

The eval runner is a Temporal workflow (`EvaluationRunWorkflow`) that takes a capability name, a prompt version, a model version, and a dataset slice, and runs the capability against every input. For each input, it scores the output against the expected behavior:

- **Claims produced.** Did the output contain the expected claims?
- **Citations valid.** Did every produced citation resolve and support its claim?
- **Citations expected.** Did the output cite the expected sources?
- **Hallucinations.** Did the output contain any unsupported claims?
- **Key points covered.** For summary capabilities, did the output cover the expected key points?

The runner produces a per-case score and an aggregate pass rate. The aggregate is compared against the capability's threshold (typically 95% pass, 95% citation precision, 90% citation recall, 1% hallucination).

### When the eval runs

- **On every PR that touches `services/ai/`.** The full dataset runs in CI. A failure blocks merge.
- **On every PR that touches `packages/contracts/ai/`.** Schema changes can break capabilities; the eval catches it.
- **Nightly on `main`.** Catches drift caused by upstream model provider changes (a model quietly changing behavior after a provider-side update).
- **On demand.** A maintainer can trigger an eval run against a candidate prompt or model before merging.

## The rule: no AI change without evaluation

The platform's hard rule: **no AI model change, no prompt change, and no capability change reaches production without passing the eval suite.**

This rule is enforced in CI. A PR that modifies `services/ai/prompts/`, `services/ai/capabilities/`, or the model configuration triggers the eval run as a required check. A PR that modifies the eval dataset itself (because the expected behavior intentionally changed) requires the AI council's review of both the code change and the eval change; the eval change is treated as part of the diff, not as an afterthought.

This rule is the only thing standing between the platform and a quietly-degraded AI. An LLM provider can change a model's behavior with no notice (and has, repeatedly, in the industry); a prompt tweak that "read better" can quietly break citation behavior; a new capability can hallucinate in production. The eval suite catches all of these before they reach citizens.

## Performance and load tests

Beyond correctness, the platform runs performance and load tests on a schedule (nightly on `main`, weekly against staging). These tests:

- **API load.** Sustained 1000 RPS for 10 minutes; assert p99 latency stays under 1s and error rate stays under 0.1%.
- **Search load.** Sustained 500 queries/second; assert p99 under 500ms.
- **AI load.** Sustained 50 concurrent citizen questions; assert p99 under 30s, no model-provider rate-limit errors.
- **Ingestion backfill.** Replay a historical day's worth of fetches; assert the workflow completes within 2x the original wall-clock.

Performance regressions open an issue (not necessarily a merge block, unless the regression crosses a budget threshold).

## Test data

Test data is curated and versioned. The platform maintains:

- **Golden documents** in `adapters/<cc>/testdata/` for parser tests (see [`04-country-adapters.md`](./04-country-adapters.md)).
- **Synthetic bills** in `services/legislation/testdata/` for domain tests — small, hand-authored bills that exercise edge cases (a bill with no clauses, a bill with 500 clauses, a bill with unusual stage transitions).
- **Eval fixtures** in `services/ai/eval/datasets/` — the permanent AI evaluation dataset.
- **Anonymized real data** — a snapshot of real ingested data with PII removed, used for integration and e2e tests. Refreshed quarterly.

Test data never contains real citizen PII. Synthetic data is preferred over anonymized real data; anonymized real data is preferred over real data; real data is never used in tests.

## The testing culture

Tests are not optional and not afterthought. A PR that ships a feature without tests does not merge. A bug fix ships with a regression test that would have caught the bug. A refactor is gated on the existing tests passing without modification (if the refactor requires test changes, the tests were probably testing implementation, not behavior — fix both). The team reviews tests as carefully as it reviews code; a flaky test is treated as a production incident because it erodes trust in CI.
