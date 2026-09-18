# ADR-0001: Modular monolith with workers (not 20 microservices)

## Status

Accepted — 2026-09-09

## Context

The Civic Intelligence Platform spec calls for 9 backend services + an AI service + a frontend + a country-adapter plugin system. The temptation is to deploy each as a separate microservice from day one. However:

- The domain is not yet proven — we don't know if "evidence" and "intelligence" deserve to be separate services or whether they'll be tightly coupled in practice.
- Operating 11+ microservices imposes a significant operational burden (deployment, monitoring, networking, secrets) that may exceed the team's capacity in Phase 1.
- The boundaries are clear in code but unclear in deployment until traffic patterns emerge.

## Decision

Adopt a **modular monolith + workers** deployment model:

- **Code boundaries** are strict from day one — each service has its own `internal/{domain,application,infrastructure}` package; no cross-service repository access.
- **Deployment boundaries** evolve based on scale — start with 5 deployables (Civic API, Ingestion Worker, Document Worker, AI Worker, Notification Worker).
- A service can be split into its own deployment later without redesigning the domain, because the code is already modular.

## Consequences

- **Positive**: lower operational cost in Phase 1; faster iteration; fewer networking hops; easier local development.
- **Positive**: the architecture is "split-ready" — when a service needs to scale independently, it can be extracted without changing its public API.
- **Negative**: shared database cluster means a bad migration could affect multiple services — mitigated by schema-per-bounded-context + audit triggers + forward-only migrations.
- **Negative**: a single service crash could affect others if they share a process — mitigated by deploying each worker as its own container.

## References

- [ARCHITECTURE.md §10 Deployment topology](../ARCHITECTURE.md)
- ADR-0003 (single Postgres cluster with logical schemas)
- ADR-0014 (monorepo with Go workspace + npm workspaces)
