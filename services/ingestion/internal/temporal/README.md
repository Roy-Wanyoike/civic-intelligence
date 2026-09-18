# Bill Processing Pipeline — Temporal workflow

This package implements the durable **Bill Processing Pipeline** as a Temporal
workflow + activities. See [ADR-0007](../../../../docs/adr/ADR-0007-temporal-for-durable-workflows.md)
for the architectural rationale.

## Pipeline

```
Discover → Fetch → Parse → Extract → Validate → Publish
```

| Step       | Activity                | Side effects                                            | Timeout    | Max retries |
|------------|-------------------------|---------------------------------------------------------|------------|-------------|
| Discover   | `DiscoverActivity`      | HTTP crawl of source endpoint                           | 2 min      | 5           |
| Fetch      | `FetchActivity`         | HTTP download → object storage (S3/MinIO)               | 90 s       | 5           |
| Parse      | `ParseActivity`         | documents service parser (may OCR)                      | 5 min      | 5           |
| Extract    | `ExtractActivity`       | AI gateway (summary, topics, stage) — **candidate facts** only | 60 s | 3 |
| Validate   | `ValidateActivity`      | evidence service cross-check (drops unsupported claims) | 30 s      | 5           |
| Publish    | `PublishActivity`       | legislation upsert + NATS `bill.processed` event       | 2 min      | 10          |

### Architectural contracts enforced here

1. **AI proposes, evidence disposes.** The Extract activity produces candidate
   facts (summary, topics, stage). The Validate activity gates them: any
   topic whose claim has no supporting citation is dropped. The summary is
   kept but flagged `validated=false` if uncorroborated. See ADR-0011.
2. **Temporal never writes to `legislation.*` directly.** Publish calls the
   legislation service's application layer (`LegislationClient.UpsertBill`).
3. **Workflow code is deterministic.** All `time.Now()` calls are inside
   activities; the workflow uses `workflow.Now(ctx)`. No `math/rand` in
   workflow code.
4. **Idempotent on RunID.** `UpsertBill` is keyed on
   `(country_code, identifier, year)`; a workflow replay re-upserts the same
   row and re-publishes the same event. Subscribers must be idempotent on
   `bill_id`.

## Local runbook

### Prerequisites

- Go 1.25+
- Temporal CLI (`temporal`) — install via `brew install temporal` or
  <https://docs.temporal.io/cli#install>
- (Optional) Temporalite — alternative to the CLI dev server, same UX

### 1. Start Temporal

```bash
temporal server start-dev
# Web UI: http://localhost:8080
# gRPC:   127.0.0.1:7233
```

### 2. Start the worker

```bash
cd services/ingestion
go run ./cmd/worker
# Logs JSON to stdout. Ctrl-C to stop.
```

Environment variables:

| Var                       | Default              | Meaning                              |
|---------------------------|----------------------|--------------------------------------|
| `TEMPORAL_ADDRESS`        | `127.0.0.1:7233`     | Temporal gRPC host:port              |
| `TEMPORAL_NAMESPACE`      | `default`            | Temporal namespace                   |
| `TEMPORAL_TASK_QUEUE`     | `bill-processing`    | Task queue the worker polls         |
| `TEMPORAL_WORKER_CONCURRENT` | `50`              | Max concurrent activity executions  |
| `LOG_LEVEL`               | `info`               | debug \| info \| warn \| error      |

### 3. Trigger a workflow

Using the Temporal CLI:

```bash
temporal workflow execute \
  --task-queue bill-processing \
  --type BillProcessingWorkflow \
  --input '{
    "country_code":"KE",
    "bill_id":"00000000-0000-0000-0000-000000000001",
    "source_url":"https://www.parliament.go.ke/bills/14-2024",
    "identifier":"Bill No. 14 of 2024",
    "year":2024,
    "started_at":"2025-01-01T00:00:00Z"
  }'
```

Or from Go code (e.g. an API handler):

```go
c, _ := client.Dial(client.Options{HostPort: "127.0.0.1:7233"})
run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
    TaskQueue: "bill-processing",
    ID:        fmt.Sprintf("bill:%s:%s:%d", countryCode, identifier, year),
}, "BillProcessingWorkflow", temporal.BillProcessingWorkflowInput{
    CountryCode: countryCode, Identifier: identifier, Year: year,
    SourceURL: sourceURL, BillID: billID, StartedAt: time.Now().UTC(),
})
```

The `ID` is derived from the bill tuple so a duplicate trigger collapses on
the server (Temporal dedups by WorkflowID when `IDReusePolicy` rejects dupes).

### 4. Observe

Open <http://localhost:8080> → click the workflow → see every activity
input, output, retry, and failure. This is the durability win: if the worker
crashes mid-pipeline, restarting it replays from the last completed activity.

## Wiring dependencies

`cmd/worker/main.go` constructs the activity structs with `nil` dependencies
so the worker compiles out of the box. In a real deployment, replace the
`nil`s with concrete clients constructed from environment:

| Activity field  | Concrete impl                                                 |
|-----------------|---------------------------------------------------------------|
| `Registry`      | `domain.AdapterRegistry` (already in `internal/domain`)      |
| `Fetcher`       | `internal/infrastructure/fetch.HTTPFetcher`                   |
| `Storage`       | `packages/storage.S3` (S3/MinIO client — see ENG-F3)         |
| `IDGen`         | `github.com/google/uuid`                                      |
| `Documents`     | HTTP client → documents service                               |
| `AI`            | HTTP client → `services/ai` gateway                           |
| `Validator`     | `intelligence.domain.EvidenceLookupClientBasedValidator`     |
| `Evidence`      | HTTP client → evidence service                                |
| `Publisher`     | `packages/events.NATS` publisher                             |
| `Legislation`   | HTTP client → legislation service `/internal/bills` endpoint |

## Failure modes

| Failure                       | Behavior                                                       |
|-------------------------------|----------------------------------------------------------------|
| Source URL 404                | Discover returns no items → workflow fails non-retryable      |
| Network blip during Fetch     | Temporal retries up to 5× with exponential backoff             |
| AI gateway timeout            | Extract retries 3×, then workflow fails (replayable)           |
| All topics rejected           | Workflow continues; publishes bill with empty topics           |
| Legislation upsert conflict   | Publish retries 10× (idempotent on country+identifier+year)    |
| Worker crash mid-workflow     | Restart worker; Temporal replays from last completed activity  |

## Tests

Unit tests live in `workflow_test.go` (TODO: add once Go is available —
will use `go.temporal.io/sdk/testsuite` to replay the workflow against
stub activities).
