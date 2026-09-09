# 12 — Deployment

This document describes the deployment topology: the initial five deployables, the Docker Compose setup for local development, the Helm chart structure for staging and production, the Argo CD gitops flow, and the dev/staging/prod environment split.

The platform ships as **five deployables** initially, backed by shared infrastructure (Postgres, NATS, Temporal, OpenSearch, Keycloak). The decision to start with five deployables rather than twenty microservices is documented in [ADR-0001](../adr/ADR-0001-modular-monolith-with-workers.md). The boundary between deployables is operational (independent scaling, independent release cadence, resource isolation), not architectural (the service boundaries are the same regardless of how they're packaged into deployables).

## The five deployables

| Deployable | Contains | Why it is its own deployable | Resource profile |
| --- | --- | --- | --- |
| **Civic API** | `services/api/` + `services/legislation/` (read) + `services/identity/` + `services/search/` (read) + `services/intelligence/` (gateway, sync) | User-facing latency; independent horizontal scaling; secrets for OIDC | CPU-bound, low memory, many replicas |
| **Ingestion Worker** | `services/ingestion/` + `adapters/kenya/` + `services/legislation/` (write, for canonical updates) | Network-egress isolation (SSRF); long-running polite crawls; dedicated network namespace | I/O-bound, low CPU, low memory, low replica count |
| **Document Worker** | `services/documents/` | CPU-heavy OCR and parsing; may use GPU for OCR acceleration; different scaling profile from API | CPU/GPU-bound, high memory, scales with document volume |
| **AI Worker** | `services/intelligence/` (async) + `services/ai/` | GPU-capable for in-house models; secrets for model providers; cost isolation | GPU-bound when self-hosting, I/O-bound when calling providers, scales with AI demand |
| **Notification Worker** | `services/notifications/` | Delivery SLAs; retry backoff; channel isolation (email outage should not block push) | I/O-bound, low CPU, low memory, low replica count |

All five deployables share the same codebase (the monorepo) and are built from the same Docker images, parameterized by entrypoint and config. The Civic API image starts the HTTP server; the worker images start the worker loop. The split is `command`-level in Kubernetes, not `image`-level — though we may produce separate images per worker later for image-size optimization.

## Docker Compose for local development

The local development environment is a single `docker compose up` that brings up the entire platform. The compose file lives at `infrastructure/docker-compose/docker-compose.yaml` and is the source of truth for what a "complete platform" looks like.

Services in the compose file:

- **postgres** — PostgreSQL 16 with `pgvector`, `pg_trgm` extensions; migrations applied on startup.
- **nats** — NATS server with JetStream enabled; streams and consumers created on startup.
- **temporal** — Temporal server (with Postgres as its backend); the web UI is exposed on a local port.
- **opensearch** — OpenSearch single-node; used for faceted search when pgvector/FTS is insufficient.
- **keycloak** — Keycloak with a pre-configured realm for local dev; the realm includes a test user with `citizen` role and a partner client.
- **minio** — S3-compatible blob storage for SourceDocuments.
- **api** — the Civic API, started with `air` for hot reload.
- **ingestion-worker**, **document-worker**, **ai-worker**, **notification-worker** — the four worker images, each started with `air` / `watchmedo` for hot reload.
- **web** — the Next.js dev server.

The compose file uses `depends_on` with healthchecks to bring services up in the right order (Postgres before Temporal before workers before API). Migrations run as an init container (well, an init service in compose) before any service that needs the schema.

Local development uses real infrastructure (real Postgres, real NATS, real Temporal) rather than mocks. This is intentional: mocks drift from reality, and the cost of running real infrastructure locally is low (Docker Compose handles it). A developer can `docker compose up`, wait 30 seconds, and have a fully-functional platform.

## Helm chart structure

The Helm charts live in `infrastructure/helm/`. The structure:

```
infrastructure/helm/
├── civic-platform/           # the umbrella chart
│   ├── Chart.yaml
│   ├── values.yaml            # default values (dev)
│   ├── values.staging.yaml    # staging overrides
│   ├── values.prod.yaml       # production overrides
│   └── charts/
│       ├── civic-api/         # the Civic API subchart
│       ├── ingestion-worker/  # the Ingestion Worker subchart
│       ├── document-worker/
│       ├── ai-worker/
│       ├── notification-worker/
│       ├── postgres/          # the Postgres subchart (Bitnami, customized)
│       ├── nats/
│       ├── temporal/
│       ├── opensearch/
│       ├── keycloak/
│       └── observability/     # Prometheus, Grafana, Loki, Tempo
```

Each subchart is independently versionable. The umbrella chart pins subchart versions; upgrading a subchart is a PR to the umbrella chart's `Chart.yaml`.

The values files override per-environment concerns: replica counts, resource requests/limits, secrets references (Vault paths in production, dummy values in dev), ingress configuration, autoscaling thresholds. The dev values file is checked in to the repo; staging and production values files are in a separate, access-controlled `infrastructure-helm-values` repo (because they contain secret references, even if not the secrets themselves).

Each worker subchart includes:

- A `Deployment` with the worker image, env vars from ConfigMaps and Secrets, resource requests/limits, and liveness/readiness probes.
- A `HorizontalPodAutoscaler` (CPU/memory for most; custom metric for queue depth on workers).
- A `PodDisruptionBudget` ensuring at least one replica stays available during node drains.
- A `Service` (headless, for worker discovery if needed) and a `ServiceAccount` with the minimum RBAC permissions.
- A `NetworkPolicy` restricting egress (especially strict for the ingestion worker; see [`08-security.md`](./08-security.md)).

The Civic API subchart additionally includes an `Ingress` and a `Service` of type `ClusterIP` exposed via the ingress.

## Argo CD gitops

The platform uses Argo CD for gitops. The flow:

1. A PR merges to `main`. CI builds the images, tags them with the commit SHA, pushes them to the registry, and updates the Helm values file in `infrastructure-helm-values` with the new image tag.
2. Argo CD detects the values-file change, renders the Helm chart, and syncs it to the staging cluster automatically (auto-sync on staging).
3. A maintainer promotes to production by manually creating an Argo CD `Application` that targets the production cluster with the production values file. Production uses auto-sync disabled; the maintainer clicks "sync" after review.

The gitops model means the cluster state is always derivable from the values files in the repo. There is no `kubectl apply` by humans; all changes go through PRs to the values files. A rollback is a revert of the values-file PR; Argo CD detects the revert and syncs the previous state.

Argo CD's sync waves control the order: secrets and ConfigMaps first, then infrastructure (Postgres, NATS, Temporal), then workers, then the API. This ensures that when the API starts, its dependencies are already running. Sync windows prevent production deploys during peak hours; off-hours emergency deploys require a manual sync-window override.

## Environments: dev, staging, prod

The platform runs three environments. Each has its own cluster (a kind cluster for local dev; managed Kubernetes clusters for staging and production), its own PostgreSQL cluster, its own NATS/Temporal/OpenSearch/Keycloak.

### dev

The dev environment is the developer's local machine. It uses Docker Compose (above). It is fully featured but small: one replica of each worker, a single-node Postgres, a single-node OpenSearch. It uses dummy secrets checked into the repo (under `infrastructure/docker-compose/secrets/`) and a Keycloak realm with a known test user. It is not internet-facing.

The dev environment is where a developer runs the e2e suite, the integration tests, and manually tests changes. It is rebuilt on every `docker compose up`; it has no persistent data across rebuilds (unless the developer explicitly preserves volumes).

### staging

The staging environment is a managed Kubernetes cluster that mirrors production's shape but at smaller scale: two replicas of the API, one of each worker, a single-node Postgres (with read replica), a single-node OpenSearch. It uses real secrets (from Vault, scoped to staging) and a Keycloak realm with test users at each role level.

Staging is the target of the CI pipeline's promotion step. Every merge to `main` deploys to staging automatically. Staging runs the e2e suite on every deploy; a failure blocks promotion to production.

Staging also hosts the pre-release eval runs: before a production release, the eval suite runs against staging's AI subsystem with the candidate prompt/model. A failure blocks the release.

Staging data is a scrubbed copy of production, refreshed weekly. The scrubbing removes PII (user data, free-text inputs) and reduces volume (e.g. only bills from the last 90 days) to keep the cluster small.

### prod

The production environment is a managed Kubernetes cluster with autoscaling node pools. It runs three replicas of the API, two of each worker (more for AI and ingestion during peak), a Postgres primary with two read replicas, and a three-node OpenSearch cluster. It uses real secrets (from Vault, scoped to production) and a Keycloak realm with real citizen users.

Production deploys are manual promotions from staging: a maintainer reviews the changelog, the eval results, the e2e results, and the staging metrics, then promotes. Argo CD syncs the values-file change to the production cluster.

Production has stricter controls: PodDisruptionBudgets on every service, NetworkPolicies on every namespace, image-signature verification on every pod, and audit-log replication to a WORM bucket. Production deploys happen during business hours (the platform's users are mostly in one time zone for now; this will change as we expand to more countries).

## Release cadence

The platform targets biweekly releases to production. Each release:

1. Cuts a release branch from `main` (`release/vX.Y.Z`).
2. Runs the full eval suite against the release branch.
3. Runs the full e2e suite against staging with the release branch deployed.
4. Is reviewed by the architecture council (for non-trivial changes).
5. Is promoted to production by a maintainer.
6. Is tagged in git; the changelog is updated; the release is announced.

Hotfixes follow the same flow but on a compressed timeline: a fix branch from the release branch, fast eval, fast e2e, immediate promotion. Hotfixes are rare; the platform's blast radius is small enough that most bugs can wait for the next scheduled release.

## Capacity planning

Capacity is reviewed monthly. The review covers: API request rate and p99 latency, worker queue depths, AI cost burn-down, Postgres CPU/disk, OpenSearch indexing lag. Scaling decisions (more replicas, larger instance types, additional read replicas) are made in the review and implemented as values-file PRs. Capacity is provisioned with headroom (typically 2x peak usage) to absorb spikes without alerting.

The AI worker is the most variable cost. Its scaling is driven by AI demand (citizen questions, batch summarization), not by general platform load. The cost dashboard (see [`09-observability.md`](./09-observability.md)) tracks spend per capability and per tenant; a capability that consistently overspends is flagged for optimization.
