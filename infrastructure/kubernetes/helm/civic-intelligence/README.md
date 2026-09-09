# Civic Intelligence Platform — Helm chart

Helm chart for the modular monolith + workers that make up the Civic
Intelligence Platform: 9 Go services, 1 Python AI service, 1 Next.js frontend,
plus their backing Postgres + NATS + Redis + OpenSearch + Temporal + Keycloak +
MinIO. Designed to be installed into a Kubernetes 1.27+ cluster.

## Quick install

```bash
# Add the Bitnami + NATS charts we depend on
helm repo add bitnami    https://charts.bitnami.com/bitnami
helm repo add nats       https://nats-io.github.io/k8s/helm/charts/
helm dependency update infrastructure/kubernetes/helm/civic-intelligence

# Install into the `civic` namespace using DEV overrides (lightest)
helm upgrade --install civic infrastructure/kubernetes/helm/civic-intelligence \
  --namespace civic --create-namespace \
  -f infrastructure/kubernetes/helm/civic-intelligence/values-dev.yaml
```

## Environments

| Environment | Values file            | Notes                                                        |
| ----------- | ----------------------- | ------------------------------------------------------------ |
| Dev (local cluster) | `values-dev.yaml` | Single replica per service, no HPA, no TLS, no ingress. |
| Staging     | `values-staging.yaml`  | HPA on, TLS via `letsencrypt-staging`, 1-2 replicas.        |
| Production  | `values-prod.yaml`     | HA replicas, prod TLS, ExternalSecrets backend, service mesh. |

```bash
helm upgrade --install civic ./civic-intelligence \
  --namespace civic-prod --create-namespace \
  -f values-prod.yaml \
  --set global.imageTag=$(git describe --tags --always)
```

## Values structure

The chart is driven by `services.<name>`, where `<name>` is one of:
`api, ingestion, documents, legislation, evidence, intelligence, ai, search,
notifications, identity, web`. Each service renders:

* `Deployment`
* `Service`
* `ConfigMap` (with shared connection strings + per-service env)
* `Secret` (or `ExternalSecret` if `externalSecrets.enabled=true`)
* `HorizontalPodAutoscaler` (if `services.<name>.autoscaling.enabled=true`)
* `NetworkPolicy` (if `services.<name>.networkPolicy.enabled=true`)
* `PodDisruptionBudget` (if `podDisruptionBudget.enabled=true`)

Plus one shared `Ingress` (`ingress.enabled=true`) and one shared
CA-bundle `ConfigMap`.

```yaml
services:
  api:
    enabled: true
    image: civic-intelligence-api
    replicas: 3
    resources:
      requests: { cpu: 250m, memory: 384Mi }
      limits:   { cpu: 1000m, memory: 768Mi }
    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 12
      cpuUtilization: 70
    env:
      PORT: "8080"
      LOG_LEVEL: info
    networkPolicy:
      enabled: true
      egress:
        - toService: postgresql
        - toService: nats
        - toFQDNs: ["api.anthropic.com", "api.openai.com"]
          ports: [443]
      ingress:
        - fromService: web
          ports: [8080]
```

## SSRF protection (crawler)

The `ingestion` service is the only pod permitted to egress to arbitrary HTTPS.
Its NetworkPolicy + an admission webhook (see `docs/adr/0008-ssrf-protection.md`)
restrict egress to FQDNs enumerated in the `EGRESS_ALLOWLIST` env var:

```yaml
services:
  ingestion:
    env:
      EGRESS_ALLOWLIST: "parliament.go.ke,kenyalaw.org,gazette.go.ke"
    networkPolicy:
      egress:
        - toFQDNsFromEnv: EGRESS_ALLOWLIST
          ports: [443, 80]
```

All other services are restricted to cluster-internal egress only.

## Secrets

* DEV / staging: inline `Secret` templates with placeholder values. Override
  per env with `--set services.api.secret.LLM_API_KEY=...`.
* PROD: set `externalSecrets.enabled=true` and provide an
`ExternalSecret`-compatible backend (AWS Secrets Manager by default,
configurable to Vault via `backend`).

## Backing services

* `postgresql.enabled=false` in dev (use docker compose). Enable in
  staging/prod with a real Postgres cluster (Bitnami chart) that loads the
  pgvector extension.
* `nats.enabled`, `redis.enabled`, `opensearch.enabled`, `keycloak.enabled`,
  `temporal.enabled` — flip per environment.

## Upgrades

* `helm upgrade` is the only supported path. The chart annotates each
  Deployment template with `checksum/config` so any ConfigMap change rolls
  the pods automatically.
* Migrations are run separately via `.github/workflows/migrate.yml`
  (golang-migrate). The chart does NOT apply migrations — that is intentional
  so the DB can migrate without rolling the application.

## Rollback

```bash
helm rollback civic <REVISION_NUMBER> -n civic-prod
```

Database rollbacks are forward-only — see
`infrastructure/postgres/README.md` for the rollback policy.

## Lint / template

```bash
helm lint infrastructure/kubernetes/helm/civic-intelligence \
  -f infrastructure/kubernetes/helm/civic-intelligence/values-dev.yaml

helm template civic infrastructure/kubernetes/helm/civic-intelligence \
  --namespace civic \
  -f infrastructure/kubernetes/helm/civic-intelligence/values-staging.yaml \
  > /tmp/civic-rendered.yaml
```
