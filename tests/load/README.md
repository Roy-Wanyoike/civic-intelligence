# Load tests (k6)

k6 scripts that exercise the three heaviest read paths against the API.

## Scripts

| Script              | Endpoint                          | Default RPS | Default duration | SLO (latency / errors)                  |
|---------------------|-----------------------------------|------------|------------------|-----------------------------------------|
| `k6-bills.js`       | `GET  /api/v1/bills`               | 100/s      | 5 min            | p50 < 200 ms, p95 < 500 ms, p99 < 1 s; err < 1% |
| `k6-search.js`      | `GET  /api/v1/search`              |  50/s      | 5 min            | p50 < 200 ms, p95 < 500 ms, p99 < 1 s; err < 1% |
| `k6-scenarios.js`   | `POST /api/v1/scenarios/{id}/run`  |  10/s      | 5 min            | p50 < 500 ms, p95 < 1.5 s, p99 < 3 s; err < 2% |

The `k6-scenarios.js` SLOs are looser at the tail because each run composes
evidence retrieval, comparable-case lookup, and AI synthesis — three round
trips that don't happen on the list endpoints.

## Prerequisites

- k6 ≥ 0.50 — install via `brew install k6` or
  <https://grafana.com/docs/k6/latest/set-up/install-k6/>
- A running API. Local dev: `docker compose -f infrastructure/docker/docker-compose.yml up`
  (services API at `http://localhost:8080`).

## Run

```bash
# Default — 5 minutes against localhost
k6 run tests/load/k6-bills.js

# Custom target + duration + rate
BASE_URL=http://staging.civic.internal K6_DURATION=10m K6_RATE=200 \
  k6 run tests/load/k6-bills.js

# Authenticated run (private beta)
K6_AUTH_TOKEN=$(cat ./token.txt) k6 run tests/load/k6-bills.js
```

k6 prints a summary table at the end. The `k6-bills.js` and `k6-scenarios.js`
scripts also print an SLO pass/fail block and write a JSON summary to
`tests/load/results/`.

## Interpreting results

- `http_req_failed` — k6's own failure counter (network errors, 5xx). Should
  be < 1% (or < 2% for scenarios).
- `bills_errors` / `search_errors` / `scenario_run_errors` — application-
  level checks we defined per script. These catch "200 OK but body shape is
  wrong" cases that `http_req_failed` misses.
- `p(50)` / `p(95)` / `p(99)` — the SLO percentiles. The `thresholds`
  block in each script causes k6 to exit non-zero when any SLO is breached,
  so you can wire these into CI: `k6 run ... --out json=results.json || exit 1`.

## CI integration

```yaml
# .github/workflows/load.yml
- name: Run bills load test
  run: k6 run tests/load/k6-bills.js
  env:
    BASE_URL: http://localhost:8080
    K6_DURATION: 1m   # keep CI fast
    K6_RATE: 50
```

## When SLOs are breached

1. Check the API's p99 latency in Grafana (the k6 numbers should match).
2. If `http_req_failed` is high → infra problem (DB down, NATS full, etc.).
3. If only `bills_errors` is high → the response shape regressed.
4. If p95/p99 drift upward over time → look for missing indexes / N+1
   queries; check the Postgres slow query log.

See `tests/chaos/README.md` for related failure-mode runbooks.
