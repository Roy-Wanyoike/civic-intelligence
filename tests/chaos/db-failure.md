# Chaos runbook: Postgres failure

**Hypothesis:** When the Postgres primary is unavailable, the API degrades
gracefully — it returns HTTP 503 with a JSON error envelope, never a panic
or a stack trace.

**Blast radius:** All read endpoints that touch canonical state (bills,
acts, constitution, search). Cached responses (Redis) should still serve
for the cache TTL window.

## Preconditions

- Local stack running: `docker compose -f infrastructure/docker/docker-compose.yml up`
- API reachable at `http://localhost:8080/healthz` → 200 OK
- `psql` or `docker exec` access to the Postgres container

## Steps

1. **Capture the baseline.**

   ```bash
   curl -s -o /dev/null -w '%{http_code} %{time_total}s\n' \
     http://localhost:8080/api/v1/bills?limit=1
   # Expect: 200 ~50ms
   ```

2. **Kill Postgres.**

   ```bash
   docker compose -f infrastructure/docker/docker-compose.yml stop postgres
   # Or: docker kill civic-postgres
   ```

3. **Verify the API returns 503 (not 500, not panic).**

   ```bash
   curl -s -i http://localhost:8080/api/v1/bills?limit=1 | head -20
   # Expect:
   #   HTTP/1.1 503 Service Unavailable
   #   Content-Type: application/json
   #   {"error":"database unavailable","retry_after":30}
   ```

4. **Verify the API did NOT panic.**

   ```bash
   docker compose logs api --tail=50 | grep -i 'panic\|goroutine'
   # Expect: no matches. You should see "db.ping failed" / "db query failed"
   # error logs but no stack trace.
   ```

5. **Verify `/healthz` reports unhealthy.**

   ```bash
   curl -s http://localhost:8080/healthz
   # Expect: 503 with {"status":"unhealthy","checks":{"db":"down"}}
   ```

6. **Verify cache fallback (if a cached copy exists).**

   ```bash
   # If a bill was fetched before the outage, the Redis cache should still
   # serve it for the TTL window. The cache layer's Health() will be failing
   # but individual GETs against existing keys still work.
   curl -s http://localhost:8080/api/v1/bills/<previously-fetched-id>
   # Expect: 200 from cache, with an X-Cache header showing HIT
   ```

7. **Restore Postgres.**

   ```bash
   docker compose -f infrastructure/docker/docker-compose.yml start postgres
   ```

8. **Verify recovery.**

   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/api/v1/bills?limit=1
   # Expect: 200 within ~5 seconds of Postgres being ready.
   curl -s http://localhost:8080/healthz
   # Expect: 200 {"status":"healthy","checks":{"db":"up","redis":"up"}}
   ```

## Pass criteria

- [ ] API returns 503 (NOT 500, NOT 502, NOT panic)
- [ ] No `panic:` or `goroutine` stack traces in API logs
- [ ] `/healthz` flips to unhealthy within 1s of the outage
- [ ] `/healthz` flips back to healthy within 5s of recovery
- [ ] Cached reads continue to serve during the outage

## Failure modes that would fail this runbook

- API returns 500 with a stack trace → middleware is leaking exceptions
  instead of translating them to 503.
- API hangs indefinitely → DB driver timeout not configured (should be
  ≤ 5s for read queries).
- API crashes the process → not caught by the recovery middleware.
- `/healthz` still reports healthy after outage → health check is not
  actually pinging the DB.

## Related

- `tests/chaos/redis-down.md` — what happens when the cache layer fails
- `infrastructure/postgres/migrations/003_identity.up.sql` — connection
  pool settings that determine the timeout behaviour seen here
- ADR-0009 (if/when written) — graceful degradation policy
