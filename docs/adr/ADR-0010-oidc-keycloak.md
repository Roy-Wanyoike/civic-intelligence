# ADR-0010: OIDC via Keycloak (don't build auth)

## Status

Accepted — 2026-09-09

## Context

The platform needs authentication (citizens, researchers, editors, admins) and authorization (RBAC). Options:

1. **Build our own auth** — bad idea. Auth is hard, security-critical, and a distraction.
2. **OIDC via external provider** (Auth0, Okta, Google) — fast, but vendor lock-in + cost + data residency concerns for civic data.
3. **OIDC via self-hosted Keycloak** — open source, standards-compliant, full control.

## Decision

Adopt **OIDC via self-hosted Keycloak** for authentication. The `services/identity` service handles only the application-level concerns: users, sessions (cached), roles, permissions, preferences.

- Keycloak issues JWTs.
- The Go BFF validates JWTs using Keycloak's public keys (JWKS).
- The `identity` service stores user metadata (display name, locale, preferences) keyed by the OIDC `sub` claim.

## Consequences

- **Positive**: no auth code to maintain — Keycloak handles password resets, MFA, social login, etc.
- **Positive**: standards-compliant — any OIDC provider can replace Keycloak without changing application code.
- **Positive**: full data sovereignty (Keycloak runs in our cluster).
- **Negative**: extra service to operate (Keycloak + its own Postgres). Mitigated by including it in `docker-compose.yml` and the Helm chart.
- **Negative**: Keycloak's UX is dated; mitigated by using it as a backend only, with our own login UI.

## RBAC

RBAC lives in the `identity` schema (not Keycloak):
- Roles: `citizen`, `researcher`, `editor`, `admin`.
- Permissions: `bill:read`, `bill:follow`, `bill:question`, `ai:stream`, `search:query`, `briefing:read`, `editor:review`, `admin:users`, `admin:sources`.
- Every protected endpoint asserts the required permission.

## References

- ARCHITECTURE.md §14 (Security)
- `infrastructure/docker/docker-compose.yml` (Keycloak service)
- `infrastructure/postgres/migrations/003_identity.up.sql` (RBAC tables)
- `infrastructure/postgres/migrations/016_seed_roles.up.sql` (seed data)
