# Chaos engineering — Civic Intelligence Platform

Chaos runbooks for verifying graceful degradation under infrastructure
failures. Each runbook is a single, executable scenario: preconditions,
steps, pass criteria, and the failure modes that would fail the run.

> **Why runbooks instead of automated chaos tooling?** The platform is
> pre-production. Manual runbooks let us learn the failure shapes before
> we automate them away. Once a runbook has been executed successfully
> ≥ 5 times, automate it with `chaos-mesh` / `litmus` / `pumba` (see
> "Automation roadmap" below).

## Runbooks

| Runbook                  | Component killed  | What we verify                                                |
|--------------------------|-------------------|---------------------------------------------------------------|
| `db-failure.md`          | Postgres          | API returns 503, never panics; cache fallback works          |
| `nats-outage.md`         | NATS              | Reads unaffected; writes buffer + replay; no event loss      |
| `redis-down.md`          | Redis             | Cache miss → DB fallback; latency rises but no 500s            |
| `temporal-restart.md`    | Temporal          | In-flight workflows resume from last activity; no duplicates |
| `ai-timeout.md`          | AI service        | 504 with JSON body; cached AI outputs still serve             |

## SLOs (the universal pass bar)

Every runbook checks against the same SLOs:

| SLO                                   | Threshold                                  |
|---------------------------------------|--------------------------------------------|
| Availability during partial outage    | ≥ 99% (degraded is OK; 500s are not)      |
| No `panic:` stack traces in logs       | 0                                          |
| `/healthz` flips state within         | 1 s of the outage                          |
| `/healthz` recovers within            | 5 s of the dependency returning            |
| No data loss                          | 0 events / rows / objects missing          |
| No duplicate side effects             | 0 (idempotency keys must dedup replays)   |

## Running a chaos drill

### Local (manual)

```bash
# 1. Start the full stack
docker compose -f infrastructure/docker/docker-compose.yml up -d

# 2. Pick a runbook, follow its steps verbatim
# e.g. open tests/chaos/db-failure.md in another pane

# 3. Record results (pass/fail per criterion) in the runbook's "Results"
#    section, or in your team's incident-tracking tool.

# 4. If any criterion FAILED: file an issue, link it from the runbook,
#    and don't ship the next release until it's fixed.
```

### CI (smoke versions)

A subset of each runbook is automated in CI as a smoke test. The smoke
version uses a shorter timeout and only checks the pass criteria — not
the full recovery flow. See `.github/workflows/chaos-smoke.yml` (TODO).

## Cadence

| Drill                  | Frequency                          | Owner               |
|------------------------|------------------------------------|---------------------|
| Full chaos drill       | Monthly (first Monday)             | SRE on-call         |
| Game-day (all runbooks)| Quarterly                          | Platform team       |
| CI smoke drill        | Every PR touching infra code       | Automated           |

## Automation roadmap

Replace manual `docker stop` with declarative chaos experiments:

1. **Phase 1 (now):** manual runbooks (this directory).
2. **Phase 2:** `chaos-mesh` experiments in a dedicated `chaos` namespace
   (kills pods via NetworkChaos / PodChaos CRDs).
3. **Phase 3:** `litmus` chaos workflows scheduled via the chaosengine
   against staging, with results piped to Grafana.
4. **Phase 4:** production chaos — kill a single pod, then a zone, then
   a region. With multi-AZ HA verified, expand to game-days that kill
   entire services briefly.

## When a drill fails

1. Don't restore the failed component yet — capture the failure mode:
   ```bash
   docker compose logs api --since=5m > /tmp/chaos-failure.log
   docker compose logs postgres --since=5m >> /tmp/chaos-failure.log
   ```
2. File an issue: title "Chaos: <runbook> — <failure mode>".
3. Tag the runbook with `Status: FAILING — see #<issue>` until fixed.
4. Only then restore the component and re-run the runbook to confirm the
   fix.

## Related

- `tests/load/README.md` — performance SLOs (the chaos runbooks assume
  these baselines)
- `infrastructure/docker/docker-compose.yml` — the local stack the
  runbooks assume
- `docs/adr/ADR-0007-temporal-for-durable-workflows.md` — why Temporal
  survives restarts
- `MASTER_AUDIT.md` — known failure-mode inventory
