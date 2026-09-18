# Chaos runbook: NATS outage

**Hypothesis:** When the NATS bus is unavailable, the API continues serving
reads. Async writes (event publication) are buffered in an in-process queue
and retried; if the outage exceeds the buffer window, writes return 202
(accepted, deferred) rather than failing.

**Blast radius:** All endpoints that publish events (`POST /api/v1/bills`,
scenario runs, ingestion webhooks). Read endpoints should be unaffected.

## Preconditions

- Local stack running with NATS at `nats://localhost:4222`
- API reachable at `http://localhost:8080`
- `nats` CLI: `brew install nats-io/nats-tools/nats`

## Steps

1. **Baseline: subscribe to `bill.processed`.**

   ```bash
   nats sub 'bill.processed' &
   ```

2. **Trigger an event.**

   ```bash
   curl -s -X POST http://localhost:8080/api/v1/bills \
     -H 'Content-Type: application/json' \
     -d '{"country_code":"KE","identifier":"TEST-CHAOS-1","year":2024,"title":"Chaos test bill"}'
   # Expect: 201 Created; the nats sub terminal prints the event.
   ```

3. **Kill NATS.**

   ```bash
   docker compose -f infrastructure/docker/docker-compose.yml stop nats
   ```

4. **Verify reads still work (NATS is not on the read path).**

   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/api/v1/bills
   # Expect: 200.
   ```

5. **Verify write publishes to a dead-letter queue + retries.**

   ```bash
   # The publisher should buffer the event and retry with exponential
   # backoff. Watch the API logs:
   docker compose logs api --tail=100 -f | grep -i 'nats.*fail\|retry\|buffered'

   # POST another bill during the outage:
   curl -s -X POST http://localhost:8080/api/v1/bills \
     -H 'Content-Type: application/json' \
     -d '{"country_code":"KE","identifier":"TEST-CHAOS-2","year":2024,"title":"Chaos test bill 2"}'
   # Expect: 202 Accepted (deferred), NOT 500.
   ```

6. **Restore NATS.**

   ```bash
   docker compose -f infrastructure/docker/docker-compose.yml start nats
   ```

7. **Verify the buffered event is replayed.**

   ```bash
   # Within ~10s of NATS returning, the publisher should flush its buffer.
   # The nats sub terminal from step 1 should print the TEST-CHAOS-2 event.
   docker compose logs api --tail=50 | grep -i 'nats.*recovered\|flush'
   ```

## Pass criteria

- [ ] Reads return 200 throughout the outage
- [ ] Writes during the outage return 202 (deferred), not 500
- [ ] No `panic:` stack traces in API logs
- [ ] On NATS recovery, buffered events are flushed within 30s
- [ ] No events are duplicated (check by comparing `event_id` of the
      original and the replayed message — they must match)

## Failure modes that would fail this runbook

- Writes return 500 → the publisher is not catching the connection error
  and translating it to a deferred publish.
- Buffered events are lost on restart → the in-process buffer should be
  backed by a spill-to-disk mechanism (or at minimum, survive the duration
  of a NATS restart).
- Events are duplicated on replay → the publisher is using at-least-once
  semantics without dedup; subscribers must be idempotent on `event_id`.

## Related

- `packages/events/nats.go` (TODO once wired) — the publisher with the
  buffer + retry policy
- `tests/chaos/temporal-restart.md` — NATS is also the transport Temporal
  workers use for some signals; verify both stay healthy.
