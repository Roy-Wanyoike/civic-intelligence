# Chaos runbook: AI service timeout

**Hypothesis:** When the AI service (`services/ai`) is slow or unreachable,
the platform degrades gracefully:

1. Synchronous endpoints that need AI (e.g. `POST /api/v1/questions`)
   return 504 Gateway Timeout after a short, hard cap (≤ 30s) with a JSON
   body that offers the citizen a "try again later" CTA.
2. Background pipelines (Bill Processing) flag the AI-dependent step as
   `validated=false` and continue — the bill is still published, just
   without an AI summary.
3. Cached AI outputs (summaries, explanations) continue to serve.
4. No request hangs forever; no request returns a 500 with a stack trace.

**Blast radius:** Q&A, summarisation, scenario synthesis, classification.
Reads of canonical legislation (bills, acts, constitution) are unaffected.

## Preconditions

- Local stack running with the AI service at `http://localhost:8000`
- API reachable at `http://localhost:8080`
- A pre-warmed cache entry for at least one bill summary
- `ab` or `hey` for load (optional)

## Steps

1. **Baseline: ask a question.**

   ```bash
   curl -s -X POST http://localhost:8080/api/v1/questions \
     -H 'Content-Type: application/json' \
     -d '{"bill_id":"<some-bill-id>","question":"What does this bill do?","audience":"general"}'
   # Expect: 200 with an AI-generated answer in < 10s.
   ```

2. **Make the AI service slow.**

   Option A — add a delay:

   ```bash
   # If the AI service has a debug env var:
   AI_DEBUG_LATENCY_MS=60000 docker compose restart ai
   ```

   Option B — kill the AI service entirely:

   ```bash
   docker compose -f infrastructure/docker/docker-compose.yml stop ai
   ```

   Option C — block at the network layer (more realistic):

   ```bash
   # Insert a packet-drop rule between API and AI:
   sudo iptables -A OUTPUT -p tcp --dport 8000 -j DROP
   ```

3. **Verify synchronous Q&A returns 504 (NOT 500, NOT hang).**

   ```bash
   curl -s -i --max-time 35 -X POST http://localhost:8080/api/v1/questions \
     -H 'Content-Type: application/json' \
     -d '{"bill_id":"<some-bill-id>","question":"What does this bill do?","audience":"general"}' \
     | head -20
   # Expect (option A — slow AI):
   #   HTTP/1.1 504 Gateway Timeout
   #   Content-Type: application/json
   #   {"error":"ai service timed out","retry_after":60}
   # Expect (option B — AI down):
   #   HTTP/1.1 503 Service Unavailable
   #   {"error":"ai service unavailable","retry_after":60}
   ```

4. **Verify cached AI outputs still serve.**

   ```bash
   curl -s -i http://localhost:8080/api/v1/bills/<pre-warmed-bill-id>/summary
   # Expect: 200, X-AI-Generated: true, X-Cache: HIT
   # The cached summary should be served from Redis, not recomputed.
   ```

5. **Verify the Bill Processing Pipeline degrades, not fails.**

   ```bash
   # Trigger a workflow that will hit the (down) AI service at the Extract step:
   temporal workflow execute \
     --task-queue bill-processing \
     --type BillProcessingWorkflow \
     --input '{"country_code":"KE","bill_id":"00000000-0000-0000-0000-000000000011","source_url":"https://example.invalid/bill.pdf","identifier":"TEST-AI-TIMEOUT","year":2024}'

   # Expect: the workflow completes (status COMPLETED) but the result has
   # validated=false and an empty topics list. The bill is still published
   # to legislation.bills — just without AI enrichment.
   ```

   In the Temporal UI, the Extract activity will show as failed (max
   retries hit), but the Validate + Publish activities still run with the
   degraded state.

6. **Restore the AI service.**

   ```bash
   docker compose start ai
   # Or: sudo iptables -D OUTPUT -p tcp --dport 8000 -j DROP
   ```

7. **Verify recovery.**

   ```bash
   curl -s -o /dev/null -w '%{http_code} %{time_total}s\n' \
     -X POST http://localhost:8080/api/v1/questions \
     -H 'Content-Type: application/json' \
     -d '{"bill_id":"<some-bill-id>","question":"What does this bill do?"}'
   # Expect: 200 in < 10s, AI service restored.
   ```

## Pass criteria

- [ ] Synchronous AI-dependent requests return 504/503 (NOT 500, NOT hang)
  within 30s
- [ ] No `panic:` stack traces in API logs
- [ ] Cached AI outputs (summaries, explanations) continue to serve
- [ ] Bill Processing Pipeline completes with `validated=false` rather than
  transitioning to FAILED
- [ ] `/healthz` reports `ai: down` (or `ai: degraded`)
- [ ] After recovery, new AI requests succeed without manual intervention

## Failure modes that would fail this runbook

- API hangs forever → no per-request timeout in the AI gateway client.
  Should be ≤ 30s (see `services/ai/app/gateway.py` `model_gateway_timeout_seconds`).
- API returns 500 with a stack trace → the AI gateway error isn't being
  caught and translated to 504.
- Bill workflow transitions to FAILED → the Extract activity's retry
  policy is too aggressive (MaximumAttempts > 3); a single AI outage
  kills the whole pipeline.
- Cached outputs stop serving → the cache layer is depending on the AI
  service to validate cached entries (it shouldn't — cache is independent).
- After recovery, AI still fails → the gateway's circuit breaker hasn't
  reset; verify `GatewayAllProvidersFailedError` clears on next success.

## Related

- `services/ai/app/gateway.py` — the provider-agnostic gateway with
  retries, timeouts, and budget enforcement.
- `services/ingestion/internal/temporal/activities.go` — ExtractActivity
  uses a 60s timeout + 3 retries (see `ActivityOptionsFor("extract")`).
- `tests/chaos/db-failure.md` — the AI service itself depends on Postgres;
  a DB outage will look like an AI outage from the API's perspective.
