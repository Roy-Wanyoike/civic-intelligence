# ADR-0014: Monorepo (Go workspace + npm workspaces)

## Status

Accepted — 2026-09-09

## Context

The platform has Go services, a Python service, and a Next.js frontend. Options:

1. **Polyrepo** — one repo per service. Clean boundaries, but hard to refactor across services, hard to keep contracts in sync, hard to atomically test the full stack.
2. **Monorepo** — one repo for everything. Easier refactoring, atomic changes, shared CI, but needs tooling for multi-language builds.

## Decision

Adopt a **monorepo** with language-appropriate workspace tooling:

- **Go**: a root `go.mod` (module `github.com/Roy-Wanyoike/civic-intelligence`) + per-service `go.mod` with `replace` directives for shared `packages/*`.
- **Python**: standard `requirements.txt` per service (no workspace tooling needed).
- **Frontend**: npm workspaces (planned — currently a single `apps/web` package).

## Consequences

- **Positive**: atomic refactors — change a contract in `packages/contracts/` and update all consumers in one PR.
- **Positive**: shared CI — one workflow runs all tests.
- **Positive**: easier onboarding — clone one repo, run one command.
- **Positive**: ADRs and architecture docs live alongside code.
- **Negative**: larger repo = slower clones — mitigated by sparse checkout for contributors who only need one service.
- **Negative**: CI must be smart about what to test — mitigated by path filters in GitHub Actions (planned).

## References

- ARCHITECTURE.md (Repository layout)
- `go.mod` (root)
- `services/*/go.mod` (per-service modules)
