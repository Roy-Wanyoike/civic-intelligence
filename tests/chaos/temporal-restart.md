# Chaos runbook: Temporal restart

**Hypothesis:** When the Temporal server restarts (or briefly becomes
unreachable), in-flight Bill Processing workflows PAUSE — they do not fail.
When Temporal returns, the worker reconnects and the workflows resume from
the last completed activity. No work is lost; no work is duplicated (Temporal
guarantees at-least-once activity execution; consumers must be idempotent).

**Blast radius:** The Bill Processing Pipeline. Read endpoints (bills list,
search, etc.) are unaffected — they don't go through Temporal.

## Preconditions

- Local Temporal dev server running: `temporal server start-dev`
- Worker running: `cd services/ingestion && go run ./cmd/worker`
- Temporal CLI: `temporal` (≥ 1.0)
- A workflow in-flight. Trigger one (it will hang at the Fetch step if the
  target URL is unreachable):

  ```bash
  temporal workflow execute \
    --task-queue bill-processing \
    --type BillProcessingWorkflow \
    --input '{"country_code":"KE","bill_id":"00000000-0000-0000-0000-000000000010","source_url":"https://www.parliament.go.ke/bills/14-2024","identifier":"Bill No. 14 of 2024","year":2024,"started_at":"2025-01-01T00:00:00Z"}'
  ```

## Steps

1. **Capture the workflow state.**

   ```bash
   temporal workflow list --query "TaskQueue='bill-processing'"
   # Note the WorkflowID + RunID of an in-flight workflow. Use the Temporal
   # UI (http://localhost:8080) to inspect which activities have completed.
   ```

2. **Restart Temporal.**

   ```bash
   # If using the dev server:
   pkill -f 'temporal server start-dev'
   sleep 2
   temporal server start-dev &

   # Or via docker:
   docker restart civic-temporal
   ```

3. **Verify the worker reconnects.**

   ```bash
   # Worker logs should show:
   #   "temporal client dial failed" (transient)
   #   ... followed by ...
   #   "reconnected to temporal" (or no error at all if the SDK retried fast)
   ```

4. **Verify the workflow is still there.**

   ```bash
   temporal workflow list --query "TaskQueue='bill-processing'"
   # Expect: the same WorkflowID + RunID as before the restart, status RUNNING.
   # The workflow should NOT be in TERMINATED or FAILED state.
   ```

5. **Verify the workflow resumes from the last completed activity.**

   ```bash
   temporal workflow describe --workflow-id <WF_ID> --run-id <RUN_ID>
   # Expect: the history shows the activities completed before the restart
   # (Discover, Fetch, ...) are NOT re-executed. The workflow picks up at
   # the next activity (e.g. Parse).
   ```

   In the Temporal UI, click the workflow → History tab. Look for an
   `ActivityTaskCompleted` event followed by an `ActivityTaskScheduled`
   event for the NEXT activity — there should be no `ActivityTaskFailed`
   events caused by the restart.

6. **Verify no duplicate side effects.**

   ```bash
   # Check the S3/MinIO bucket for the document ID — there should be
   # exactly one object, not two (Fetch activity should not have re-run).
   mc ls local/ingestion-KE/2025/<hash>.pdf

   # Check NATS — there should be exactly one bill.processed event for
   # the workflow's bill_id, not two.
   nats sub 'bill.processed' --count 1
   ```

## Pass criteria

- [ ] In-flight workflows remain in RUNNING state across the restart
- [ ] Worker reconnects automatically (no manual restart needed)
- [ ] Activities that completed before the restart are NOT re-executed
- [ ] No duplicate S3 objects / NATS events / DB rows
- [ ] No workflow transitions to FAILED due to the restart
- [ ] Worker logs show no `panic:` stack traces

## Failure modes that would fail this runbook

- Workflow transitions to FAILED → an activity is treating the Temporal
  reconnect as a hard failure instead of letting the SDK retry.
- Activity re-executes from scratch → the activity is not idempotent, OR
  the workflow code is using `workflow.Now()` / random / network inside the
  workflow (which breaks replay determinism — see ADR-0007).
- Worker doesn't reconnect → the Temporal client's ` DialOptions` are
  missing the auto-reconnect config (default in `go.temporal.io/sdk` ≥ 1.20).
- Duplicate side effects → the activity is not keyed on a stable ID. The
  Fetch activity uses the content hash as the S3 key — verify your activity
  implementation does the same.

## Related

- `services/ingestion/internal/temporal/workflow.go` — the workflow
  implementation; activities use `temporal.NewNonRetryableApplicationError`
  for permanent failures so the SDK knows NOT to replay them.
- `services/ingestion/internal/temporal/README.md` — local runbook for the
  Temporal dev server.
- ADR-0007 — why we chose Temporal over in-process orchestration.
