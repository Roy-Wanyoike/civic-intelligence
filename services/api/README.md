# Civic Intelligence — API / BFF service

The citizen-facing HTTP gateway. Validates OIDC JWTs, enforces RBAC, rate-limits
(300 req/min per caller), emits Prometheus metrics, and calls domain services.
**Never touches the database directly** — reads go through the
`services/legislation` and `services/simulation` packages, AI work is proxied to
the Python FastAPI service, and writes are funneled through domain repositories.

The canonical machine-readable contract is
[docs/api/openapi.yaml](../../docs/api/openapi.yaml). This README is the
human-readable companion.

## Status

Implemented: all endpoints below (40+). The Kenya adapter, in-memory
repositories (ActRepository, DebtRepository), and the simulation service are
seeded with real / sample data so the frontend can render end-to-end flows
without a Postgres connection. Postgres-backed implementations drop in behind
the same `legislation.Wire*` boundaries — no HTTP contract change.

## Base URL

```
http://localhost:9000/api/v1    # local dev (BFF)
```

All paths in this README are relative to the base URL. Every response is JSON;
every error response is `{ "error": "...", "message": "...", "detail": {} }`.

## Authentication

| Token type | Header | Scopes |
| --- | --- | --- |
| OIDC JWT (Keycloak) | `Authorization: Bearer <jwt>` | `ai:ask`, `notification:write`, `user:admin`, `evidence:write` |

`DEV_MODE=true` (default) bypasses JWT signature verification — useful for
local development. **Never set `DEV_MODE=true` outside a sandbox.** Anonymous
calls are accepted on public-read endpoints; write endpoints return `401` if
the caller is anonymous.

A `X-Tenant-ID` header may be supplied to scope simulation calls; if omitted
the request is processed under `tenant-default`.

## Endpoints

### Legislation

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/bills` | List Bills (filter by `status`, `house`, `topic`, `q`; paginated) |
| `GET` | `/bills/{id}` | Bill detail (sourced from the Kenya Law adapter) |
| `GET` | `/bills/{id}/timeline` | Verified timeline events for a Bill |
| `GET` | `/bills/{id}/versions` | Immutable Bill versions (newest first) |
| `GET` | `/bills/{id}/summary` | AI-generated plain-language summary (validated) |
| `GET` | `/bills/{id}/changes` | Change log between Bill versions |
| `GET` | `/acts` | Acts of Parliament (issue #116) |
| `GET` | `/acts/{id}` | Act detail |
| `GET` | `/acts/{id}/audit` | Full lifecycle audit (assent, publication, commencement, regulations, amendments, court, repeal) — issue #193 |
| `GET` | `/acts/{id}/events` | Post-assent events (commencement, regulations, amendments, court challenges) |
| `POST` | `/acts/{id}/follow` | Follow a Law — creates a real subscription (issue #193 flagship) |
| `GET` | `/acts/{id}/lineage` | Full legal lineage — Bill → Parliamentary journey → Assent → Publication → Commencement → Regulations → Amendments → Court decisions → Current status. Missing steps are `NOT_VERIFIED`, never inferred. |
| `GET` | `/policies` | Government policy documents |
| `GET` | `/trending` | Trending Bills (recently enacted + approaching final stage) |
| `GET` | `/terminology/` | List all parliamentary terms |
| `GET` | `/terminology/{term}` | Plain-language explanation of a single parliamentary term |

### Constitution, Governments, Transitions

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/constitution` | Constitution metadata (`reality_layer: "FACT"` — never reinterpreted) |
| `GET` | `/governments` | List administrations (most recent first) |
| `GET` | `/governments/{id}` | Administration detail + president + terms |
| `GET` | `/governments/{id}/terms` | Presidential terms for an administration |
| `GET` | `/transitions` | Government transitions (outgoing → incoming president) |

### Scenarios (Phase 18)

Every scenario response carries an explicit `reality_layer` tag
(`HYPOTHETICAL`, `MODELED`, `OBSERVED`, etc.) and a disclaimer. The platform
cannot leak a simulated output as an observed civic fact.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/scenarios` | List scenarios |
| `POST` | `/scenarios` | Create a scenario |
| `GET` | `/scenarios/{id}` | Scenario detail |
| `POST` | `/scenarios/{id}?action=validate` | Validate assumptions / constraints |
| `POST` | `/scenarios/{id}?action=run` | Run a scenario (deterministic / Monte Carlo) — supports `seed` query param for reproducibility |
| `POST` | `/scenarios/{id}?action=replay` | Replay the last run |
| `GET` | `/scenarios/{id}/assumptions` | Scenario assumptions |
| `GET` | `/scenarios/{id}/evidence` | Source evidence + baseline evidence + assumption evidence count |
| `GET` | `/scenarios/{id}/results` | Completed run results |
| `GET` | `/scenarios/{id}/timeline` | OBSERVED / ASSUMED / MODELED / UNKNOWN timeline |
| `GET` | `/scenarios/{id}/methodology` | Methodology + declared model limitations (Gate E) |
| `POST` | `/scenarios/compare` | Compare two or more scenarios |

### Public Debt, Loans, Grants

The platform **never** attributes sovereign borrowing personally to a
president. Every debt endpoint carries the canonical
`NO_POLITICAL_PERFORMANCE_SCORE` disclaimer.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/debt` | National debt dashboard (latest CBK snapshot + debt service + debt-to-GDP) |
| `GET` | `/debt/loans` | Borrowing register — each agreement carries an `attribution_warning` field when its `GovernmentAdministrationID` does not correspond to the administration in power on `ContractDate` (issue #221). Records are never rejected — uncertainty is surfaced for human review. |
| `GET` | `/debt/timeline` | Debt-stock observations (chronological; immutable per Spec §9) |
| `GET` | `/debt/governments/{id}` | Per-administration debt summary |
| `GET` | `/loans` | Sovereign loans tracker (list) |
| `GET` | `/loans/{id}` | Sovereign loan detail |
| `GET` | `/grants` | Grants received by the government (list) |
| `GET` | `/grants/{id}` | Grant detail |

### Search, Feed, Trending, What Changed

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/search?q=...` | Hybrid keyword + semantic search across Bills, Acts, regulations, Hansard, etc. |
| `GET` | `/feed` | Civic activity feed |
| `GET` | `/trending` | Trending Bills and topics |
| `GET` | `/what-changed` | Proactive civic change feed |
| `GET` | `/briefing` | Daily civic brief (optional `date` query param) |

### Trust, Evidence, Claims, Contradictions, Sources

The trust network (migration 018) owns the canonical tables; the API exposes
an in-memory store seeded with sample Kenyan sources, claims, evidence, and
one active contradiction so the `/trust` page has concrete data to render.

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/provenance/{entity_type}/{id}` | Full evidence chain for an entity |
| `GET` | `/evidence/{id}` | Evidence detail |
| `GET` | `/claims/{id}/evidence` | Claims with their evidence |
| `GET` | `/contradictions` | Active source conflicts |
| `GET` | `/sources` | List trust sources |
| `GET` | `/sources/{id}` | Source detail with health |

### Corrections, Subscriptions, Notifications

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| `POST` | `/corrections` | Submit a correction request (anonymous submitters must provide `submitted_email`) | public |
| `GET` | `/corrections` | List corrections | `user:admin` or `evidence:write` |
| `GET` | `/corrections/{id}` | Correction detail | `user:admin` or `evidence:write` |
| `POST` | `/subscriptions` | Follow an entity (`entity_type`: bill, committee, topic, institution, person) | required |
| `GET` | `/subscriptions` | List the caller's follows | required |
| `DELETE` | `/subscriptions/{id}` | Unfollow | required |
| `GET` | `/notifications` | List the caller's notifications (optional `unread=true`) | required |
| `POST` | `/notifications/{id}/read` | Mark a notification as read | required |

### AI Q&A (proxied to the Python service)

| Method | Path | Description | Auth |
| --- | --- | --- | --- |
| `POST` | `/questions` | Ask a civic question (non-streaming) | `ai:ask` |
| `POST` | `/questions/stream` | Ask a civic question (SSE streaming) | `ai:ask` |

### Health & Metrics

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe |
| `GET` | `/metrics` | Prometheus metrics (request counters, latency histograms, OIDC verification counters) |

### Sponsor (payments)

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/sponsor/mpesa` | Initiate an M-Pesa sponsorship |
| `POST` | `/sponsor/mpesa/callback` | M-Pesa STK push callback |
| `POST` | `/sponsor/card` | Initiate a card sponsorship (Stripe) |
| `POST` | `/sponsor/card/webhook` | Stripe webhook receiver |

### Operational

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/refresh` | Trigger adapter re-discovery (called by cron) |

## Examples

### List Bills

```bash
curl -s 'http://localhost:9000/api/v1/bills?status=in_progress&page=1&page_size=20' | jq
```

### Get a Bill with its AI summary

```bash
curl -s http://localhost:9000/api/v1/bills/00000000-0000-0000-0000-000000000001/summary | jq
```

### Get the full lifecycle audit for an Act

```bash
curl -s http://localhost:9000/api/v1/acts/act-public-finance-management-2012/audit | jq
```

### Follow a Law (requires auth)

```bash
curl -s -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  http://localhost:9000/api/v1/acts/act-public-finance-management-2012/follow | jq
```

### National debt dashboard

```bash
curl -s http://localhost:9000/api/v1/debt | jq
```

### Borrowing register (with attribution validation)

```bash
curl -s http://localhost:9000/api/v1/debt/loans | jq '.borrowing_agreements[0]'
```

### Per-administration debt summary

```bash
curl -s http://localhost:9000/api/v1/debt/governments/admin-uhuru-kenyatta | jq
```

### Government transitions

```bash
curl -s http://localhost:9000/api/v1/transitions | jq
```

### Run a hypothetical scenario

```bash
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"scenario_id":"scn-simple-deterministic","seed":42}' \
  'http://localhost:9000/api/v1/scenarios/scn-simple-deterministic?action=run&seed=42' | jq
```

### Compare two scenarios

```bash
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"scenario_ids":["scn-simple-deterministic","scn-multi-variable"]}' \
  http://localhost:9000/api/v1/scenarios/compare | jq
```

### Ask a civic question (streaming)

```bash
curl -N -X POST \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{"text":"What stage is the Housing Bill, 2024 at?","country":"KE"}' \
  http://localhost:9000/api/v1/questions/stream
```

### Submit a correction (anonymous)

```bash
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "target_type":"act",
    "target_id":"act-public-finance-management-2012",
    "category":"wrong_citation",
    "reason":"Section 12 citation should reference the 2020 amendment, not the 2012 principal Act.",
    "page_url":"https://civicintelligence.dev/acts/act-public-finance-management-2012",
    "submitted_email":"citizen@example.com"
  }' \
  http://localhost:9000/api/v1/corrections | jq
```

## Architectural rules

1. **Every factual claim carries citations.** Responses that could not be
   fully verified are flagged.
2. **The AI cannot mutate truth.** AI outputs are validated before reaching
   the client; AI cannot write to canonical state (ADR-0005).
3. **Missing records are `NOT_VERIFIED`, never "never happened."** The
   platform does not infer absence.
4. **Reality layers are explicit.** Scenario responses carry
   `reality_layer: "HYPOTHETICAL"` / `"MODELED"`; the Constitution carries
   `reality_layer: "FACT"`.
5. **Sovereign borrowing is never attributed personally to a president.**
   `NO_POLITICAL_PERFORMANCE_SCORE` is appended to every per-administration
   debt summary.
6. **Immutability where it matters.** Bill versions, Act versions, simulation
   results, scenario audits, and trust audit events are append-only at the
   Postgres layer (BEFORE UPDATE OR DELETE OR TRUNCATE triggers).

## Running

```bash
go run ./cmd/main.go
# Listening on :9000 (override with API_SERVICE_ADDR)
```

Environment variables (with defaults):

```
API_SERVICE_ADDR=:9000
OIDC_ISSUER=http://localhost:8081/realms/civic
OIDC_AUDIENCE=civic-intelligence
OIDC_JWKS_URL=http://localhost:8081/realms/civic/protocol/openid-connect/certs
DEV_MODE=true
SHUTDOWN_TIMEOUT=15s
AIService_URL=http://localhost:8000
```

## Tests

```bash
go test ./...
go test -race ./...
```

The `cmd` package has 130+ tests covering the Bills, Acts, post-assent,
government, debt, scenarios, trust, corrections, notifications, and
subscriptions endpoints.
