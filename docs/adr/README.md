# Architecture Decision Records (ADRs)

This directory contains the Architecture Decision Records for the Civic Intelligence Platform. We use the [MADR](https://adr.github.io/madr/) (Markdown ADR) format. Each ADR is a numbered Markdown file with the structure: Status, Context, Decision, Consequences, Alternatives Considered, Notes.

## What an ADR is, and is not

An ADR is a record of a decision that future maintainers will need to understand. It captures *why* a choice was made, not just *what* was chosen. The code shows the *what*; the ADR shows the *why*.

An ADR is **not** a specification (the spec is in `docs/architecture/`). It is **not** a change log (the change log is in each service's `CHANGELOG.md` and the root `CHANGELOG.md`). It is **not** a request for comments (comments happen on the PR that adds the ADR; once merged, the ADR is the decision).

## When to write an ADR

Write an ADR when you make a decision that:

- **Has consequences.** A choice that affects future work in a way the next engineer would not predict from reading the code.
- **Has alternatives.** A choice where a reasonable engineer would have picked differently, and the rationale is not obvious.
- **Is hard to reverse.** A choice that, once made, is expensive to change (because code, data, or operational practice has built up around it).
- **Crosses a boundary.** A choice that affects the architectural contract in [`ARCHITECTURE.md`](../../ARCHITECTURE.md) or [`docs/architecture/`](../architecture/).

Do not write an ADR for trivial decisions (variable names, function signatures, library versions within a minor range). Those are code-review decisions, not architectural ones.

## How to write an ADR

1. Copy `ADR-0000-template.md` (or the most recent ADR's structure) to `ADR-XXXX-<short-slug>.md`, where `XXXX` is the next available number.
2. Fill in the sections: Status (Proposed or Accepted), Date, Deciders, Related (links to other ADRs and to docs), Context, Decision, Consequences, Alternatives Considered, Notes.
3. Be specific. "We chose X because Y" is good. "We chose X" is not.
4. Be honest about negatives. A decision with no downsides is a decision you have not thought about hard enough.
5. Open a PR. Tag the architecture council. The PR's review is the comment period.
6. Once merged, the ADR is Accepted. To change or supersede it, write a new ADR that links to the old one and sets the old one's Status to `Superseded by ADR-YYYY`.

## Index

| # | Title | Status |
| --- | --- | --- |
| [0001](./ADR-0001-modular-monolith-with-workers.md) | Modular Monolith with Workers | Accepted |
| [0002](./ADR-0002-go-for-backend-python-for-ai.md) | Go for Backend, Python for AI | Accepted |
| [0003](./ADR-0003-postgres-single-cluster-logical-schemas.md) | PostgreSQL Single Cluster with Logical Schemas | Accepted |
| [0004](./ADR-0004-country-adapter-pattern.md) | Country Adapter Pattern | Accepted |
| [0005](./ADR-0005-ai-cannot-mutate-truth.md) | AI Cannot Mutate Truth | Accepted |
| [0006](./ADR-0006-nats-jetstream-for-events.md) | NATS JetStream for Events | Accepted |
| [0007](./ADR-0007-temporal-for-durable-workflows.md) | Temporal for Durable Workflows | Accepted |
| [0008](./ADR-0008-evidence-first-rag.md) | Evidence-First RAG | Accepted |
| [0009](./ADR-0009-pgvector-before-opensearch.md) | pgvector Before OpenSearch | Accepted |
| [0010](./ADR-0010-oidc-keycloak.md) | OIDC via Keycloak | Accepted |
| [0011](./ADR-0011-immutable-bill-versions.md) | Immutable Bill Versions | Accepted |
| [0012](./ADR-0012-sse-for-streaming-ai-responses.md) | SSE for Streaming AI Responses | Accepted |
| [0013](./ADR-0013-contradiction-engine-do-not-silently-resolve.md) | Contradiction Engine — Do Not Silently Resolve | Accepted |
| [0014](./ADR-0014-monorepo-vs-polyrepo.md) | Monorepo with Go Workspace and npm Workspaces | Accepted |

## Relationship to the architecture docs

The ADRs capture *decisions*; the [architecture docs](../architecture/) capture *the system as it currently is*. The two should be consistent: an ADR that says "we chose X" should correspond to architecture docs that describe X. When the architecture changes in a way that contradicts an ADR, either the ADR is amended (with a new ADR that supersedes it) or the architecture docs are updated to match the ADR's reality.

When in doubt about whether something is an ADR or an architecture doc, ask: "is this a decision someone made, or a description of how the system works?" Decisions go in ADRs; descriptions go in architecture docs. The same topic may appear in both: the country-adapter pattern is described in [`docs/architecture/04-country-adapters.md`](../architecture/04-country-adapters.md) and decided in [ADR-0004](./ADR-0004-country-adapter-pattern.md).
