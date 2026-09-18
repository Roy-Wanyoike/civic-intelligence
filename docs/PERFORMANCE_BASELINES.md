# Performance Baselines & SLOs

**Source:** Production Gate #10 (`docs/PRODUCTION_GATE.md`) — *"No benchmark, load, soak, or stress test files anywhere in the repo."*

**Scope:** This document fixes the platform's performance Service Level Objectives (SLOs) for the three hot paths covered by Production Gate #10:
1. **API read paths** — Bills, Acts, Constitution, Search, Debt dashboard.
2. **Scenario creation** — the `/api/v1/scenarios` POST path that runs the multi-agent simulation pipeline.
3. **Scenario run** — `/api/v1/scenarios/{id}/run`, the multi-agent Monte Carlo path.

The SLOs below are the *contract*. They are enforced by:
- Go benchmarks (`go test -bench=.`) — domain-level baselines.
- k6 load tests (`tests/load/`) — end-to-end sustained throughput.
- Grafana dashboards (`infrastructure/observability/grafana-dashboards/api-latency.json`) — production telemetry.

---

## 1. SLO catalogue

### 1.1 API reads — `/api/v1/{bills,acts,search,debt}`

| Endpoint                  | p50      | p95      | p99     | Error rate |
|---------------------------|----------|----------|---------|-----------|
| `GET /api/v1/bills`       | ≤ 200 ms | ≤ 500 ms | ≤ 1 s   | < 1 %      |
| `GET /api/v1/acts`        | ≤ 200 ms | ≤ 500 ms | ≤ 1 s   | < 1 %      |
| `GET /api/v1/search`      | ≤ 200 ms | ≤ 200 ms | ≤ 500 ms| < 1 %      |
| `GET /api/v1/debt`        | ≤ 200 ms | ≤ 200 ms | ≤ 500 ms| < 1 %      |
| `GET /api/v1/debt/timeline` | ≤ 200 ms | ≤ 500 ms | ≤ 1 s | < 1 %      |

The Bills + Acts + Debt-list endpoints share the broad 500 ms p95 ceiling; the Search + Debt-dashboard endpoints are tighter (200 ms p95) because:
- Search has a hand-tuned Postgres FTS projection (migration 022) and a 20-result cap, so it should never approach the broad ceiling.
- The debt dashboard returns a single aggregated snapshot, not a paginated list — its cost is dominated by the single `ListDebtSnapshots` call.

### 1.2 Scenario creation — `POST /api/v1/scenarios`

| Endpoint                          | Acknowledgement SLO          | Background SLO             |
|-----------------------------------|------------------------------|-----------------------------|
| `POST /api/v1/scenarios`          | ≤ 1 s (API ack)              | ≤ 30 s (status = READY)    |

The API acknowledges the scenario creation within 1 second — this is just the
validation + persistence round trip. The downstream multi-agent pipeline
(researcher → comparator → auditor → synthesizer) runs asynchronously and
flips the scenario's `status` from `DRAFT` → `CONFIGURED` → `VALIDATING` →
`READY`. The 30-second background SLO reflects the upper bound on that
pipeline in steady-state; a cold cache or AI-service degradation may push it
to 60 seconds before the platform surfaces a `REVIEW_REQUIRED` flag.

### 1.3 Scenario run — `POST /api/v1/scenarios/{id}/run`

| Endpoint                                  | p50      | p95       | p99    | Error rate |
|-------------------------------------------|----------|-----------|--------|-----------|
| `POST /api/v1/scenarios/{id}/run`         | ≤ 500 ms | ≤ 1.5 s   | ≤ 3 s  | < 2 %      |

The 2% error ceiling is intentionally looser than the read paths because
scenario runs compose three round trips that don't happen on the list
endpoints:
1. Evidence retrieval (Postgres + pgvector).
2. Comparable-case lookup (legislation service).
3. AI synthesis (Anthropic / StubProvider fallback).

A transient AI timeout is recoverable (the platform returns a
`validated=false` summary rather than a 500); the 2% ceiling allows for
this without false-positives in CI.

---

## 2. Baseline measurements

The baselines below were captured with `go test -bench=. -benchmem` against
the in-memory implementations (no Postgres, no Redis, no NATS). They are the
*floor* — the per-handler cost in the absence of I/O. Production latency is
the baseline + middleware + serialization + network + dependency I/O.

| Benchmark                                   | ns/op   | B/op   | allocs/op | Notes                                                   |
|---------------------------------------------|---------|--------|-----------|---------------------------------------------------------|
| `BenchmarkBillStateMachine_ValidTransition` | ~32     | 0      | 0         | Map lookup; 0 allocs is the contract.                   |
| `BenchmarkAuditForAct_CompleteAct`          | ~253    | 240    | 4         | Worst-case audit (1 of every event kind).                |
| `BenchmarkValidateAttribution`             | ~18     | 0      | 0         | 5-admin linear scan; 0 allocs is the contract.          |
| `BenchmarkScenario_Validate`                | ~23     | 0      | 0         | Pure struct validation; 0 allocs is the contract.       |
| `BenchmarkValidatePipeline_FullValidation` | ~268    | 144    | 1         | Full 7-stage pipeline.                                   |
| `BenchmarkMonteCarloEngine_100Iterations`  | ~40000  | 43900  | 326       | 100-iter Monte Carlo with insertion-sort percentiles.   |
| `BenchmarkHandleActsList`                   | ~8900   | 9950   | 46        | In-memory actRepo + JSON serialization.                 |
| `BenchmarkHandleSearch`                     | ~22900  | 19933  | 113       | In-memory substring search across seed data.            |
| `BenchmarkHandleDebtDashboard`              | ~4200   | 7874   | 25        | Latest-snapshot lookup + dashboard figures.             |

**Reading the table:**
- `ns/op` is the per-operation wall-clock cost in nanoseconds. Convert to
  milliseconds by dividing by 1,000,000.
- `B/op` and `allocs/op` are heap-allocation counts. Zero-alloc benchmarks
  (`BillStateMachine`, `ValidateAttribution`, `Scenario.Validate`) are
  pinned at 0 by the validation contract — any regression is a per-event
  tax on the hot path.

---

## 3. How to run the benchmarks

```bash
# Per-service benchmarks (recommended — each service has its own go.mod)
cd services/legislation && go test -bench=. -benchmem -run=^$ ./...
cd services/simulation  && go test -bench=. -benchmem -run=^$ ./...
cd services/api          && go test -bench=. -benchmem -run=^$ ./...

# Or, against every module in the monorepo:
for m in services/legislation services/simulation services/api; do
  (cd "$m" && go test -bench=. -benchmem -run=^$ ./...)
done
```

The `-run=^$` flag tells Go to skip all unit tests (we want the benchmark
binary, not the test binary). The `-benchmem` flag reports allocations.

---

## 4. How to run the k6 load tests

```bash
# Default — 5 minutes against localhost
k6 run tests/load/k6-bills.js
k6 run tests/load/k6-search.js
k6 run tests/load/k6-scenarios.js

# Custom target + duration + rate
BASE_URL=http://staging.civic.internal K6_DURATION=10m K6_RATE=200 \
  k6 run tests/load/k6-bills.js
```

Each script writes a JSON summary to `tests/load/results/` and prints an SLO
pass/fail block to stdout. The `thresholds` block in each script causes k6
to exit non-zero when any SLO is breached, so they can be wired into CI:

```yaml
# .github/workflows/load.yml
- name: Run bills load test
  run: k6 run tests/load/k6-bills.js
  env:
    BASE_URL: http://localhost:8080
    K6_DURATION: 1m   # keep CI fast
    K6_RATE: 50
```

See `tests/load/README.md` for the full script table + troubleshooting guide.

---

## 5. Regression policy

| Signal                                | Action                                                            |
|--------------------------------------|------------------------------------------------------------------|
| Any benchmark regresses > 20 %        | Open an issue; tag with `perf-regression`. Investigate before merge. |
| Any benchmark regresses > 2×         | Block merge. Roll back the offending change.                     |
| A 0-alloc benchmark grows allocs/op  | Block merge — the contract is violated.                          |
| k6 SLO breach in CI                  | Block merge unless the breach is documented as expected (e.g. cold cache). |
| p99 in Grafana > SLO for > 5 min     | Page the SRE on-call.                                            |

---

## 6. Cross-references

- `docs/PRODUCTION_GATE.md` §2 row 10 — the gate this document closes.
- `tests/load/README.md` — k6 script catalog + CI integration snippet.
- `tests/chaos/README.md` — chaos runbooks (Gate #11) that assume these
  SLOs as the steady-state baseline.
- `docs/DISASTER_RECOVERY.md` — DR runbook (Gate #12) that assumes the
  same SLOs as the recovery target.
- `infrastructure/observability/grafana-dashboards/api-latency.json` —
  the production dashboard that surfaces these percentiles in real time.
- `infrastructure/observability/grafana-dashboards/api-overview.json` —
  the rollup dashboard (availability + error rate).
