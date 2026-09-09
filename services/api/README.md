# Civic Intelligence — API / BFF service

The citizen-facing HTTP gateway. Validates OIDC JWTs, enforces RBAC, rate-limits,
and calls domain services. **Never touches the database directly.**

## Status

Phase 1: minimal health/readiness endpoints only. Full REST API tracked in
[issue #19](https://github.com/Roy-Wanyoike/civic-intelligence/issues/19).

## Endpoints (implemented)

- `GET /api/v1/healthz` — liveness probe
- `GET /api/v1/readyz` — readiness probe

## Endpoints (planned — issue #19)

Per [docs/api/openapi.yaml](../../docs/api/openapi.yaml):
- Bills: list, get, timeline, versions, summary, follow
- Search: hybrid keyword + semantic
- Briefing: daily civic brief
- Q&A: POST /questions (SSE streaming to AI service)
- Entities: people, committees, institutions

## Running

```bash
go run ./cmd/main.go
# Listening on :9000 (override with API_SERVICE_ADDR)
```
