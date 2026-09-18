# Chaos runbook: Redis down

**Hypothesis:** When Redis is unavailable, the cache layer's `Health()`
flips to unhealthy, but individual `Get` calls return `redis.Nil` (or a
wrapped error). Consumers fall back to the canonical source (Postgres)
and return correct, if slower, responses. The API never returns 500 due
to a cache miss.

**Blast radius:** All cached read endpoints (bills list, bill detail,
search). Latency will increase (cache miss → DB) but responses remain
correct.

## Preconditions

- Local stack running with Redis at `127.0.0.1:6379`
- API reachable at `http://localhost:8080`
- A few cache entries already warm:
  ```bash
  for i in 1 2 3; do
    curl -s http://localhost:8080/api/v1/bills?page=$i >/dev/null
  done
  ```

## Steps

1. **Capture baseline latency.**

   ```bash
   for i in 1 2 3 4 5; do
     curl -s -o /dev/null -w '%{http_code} %{time_total}s\n' \
       http://localhost:8080/api/v1/bills?page=1
   done
   # Expect: 200 ~5-20ms (cache HIT)
   ```

2. **Kill Redis.**

   ```bash
   docker compose -f infrastructure/docker/docker-compose.yml stop redis
   # Or: docker kill civic-redis
   ```

3. **Verify `/healthz` reports Redis as down but API stays healthy.**

   ```bash
   curl -s http://localhost:8080/healthz | jq .
   # Expect:
   #   { "status": "degraded", "checks": { "db": "up", "redis": "down" } }
   # Note: status is "degraded" (not "unhealthy") — Redis is a performance
   # optimisation, not a hard dependency.
   ```

4. **Verify reads still work (with elevated latency).**

   ```bash
   for i in 1 2 3 4 5; do
     curl -s -o /dev/null -w '%{http_code} %{time_total}s\n' \
       http://localhost:8080/api/v1/bills?page=1
   done
   # Expect: 200, but latency is now ~50-200ms (DB hit on every request).
   # No request should return 500.
   ```

5. **Verify the X-Cache header reports MISS (not ERROR).**

   ```bash
   curl -s -i http://localhost:8080/api/v1/bills?page=1 | grep -i 'x-cache'
   # Expect: x-cache: MISS  (NOT: x-cache: ERROR)
   ```

6. **Verify writes still publish events (NATS is independent of Redis).**

   ```bash
   curl -s -X POST http://localhost:8080/api/v1/bills \
     -H 'Content-Type: application/json' \
     -d '{"country_code":"KE","identifier":"TEST-REDIS-DOWN","year":2024,"title":"Chaos"}'
   # Expect: 201 — the write path does not depend on Redis.
   ```

7. **Restore Redis.**

   ```bash
   docker compose -f infrastructure/docker/docker-compose.yml start redis
   ```

8. **Verify cache warms back up.**

   ```bash
   # First request: still a MISS (cache was empty on restart)
   curl -s -i http://localhost:8080/api/v1/bills?page=1 | grep -i 'x-cache'
   # Second request: HIT
   curl -s -i http://localhost:8080/api/v1/bills?page=1 | grep -i 'x-cache'
   # Expect: x-cache: HIT
   ```

## Pass criteria

- [ ] `/healthz` returns `degraded` with `redis: down` within 1s of outage
- [ ] No read returns 500 (cache miss is handled gracefully)
- [ ] `X-Cache: MISS` (not `ERROR`) on cache-unavailable responses
- [ ] Latency increases by < 10× during outage (DB hits, but no timeouts)
- [ ] `/healthz` flips back to `healthy` within 5s of Redis returning
- [ ] Cache warms back up automatically (no manual flush needed)

## Failure modes that would fail this runbook

- API returns 500 → the cache wrapper is not translating `redis.Nil` /
  connection errors into cache-miss semantics.
- API hangs → the Redis client's `ReadTimeout` is not set (or set too high).
  Should be ≤ 2s (see `packages/cache.DefaultConfig`).
- Cache stays empty after Redis returns → the client's connection pool
  isn't being re-established (verify with `redis-cli ping` from inside the
  API container).
- `/healthz` still reports `redis: down` after recovery → the health check
  is caching the failure; it should re-probe on every call.

## Related

- `packages/cache/redis.go` — the wrapper with `Health()` and the
  `wrapRedisErr` translation that makes `redis.Nil` a sentinel
- `tests/chaos/db-failure.md` — what happens when the cache fallback (DB)
  is also down
