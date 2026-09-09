# Docker images

Local dev stack lives in `docker-compose.yml`. Service Dockerfiles are:

| Dockerfile              | Build context | Build arg(s)                                           | Output                                     |
| ----------------------- | ------------- | ------------------------------------------------------ | ------------------------------------------ |
| `Dockerfile.go`         | repo root     | `SERVICE_PATH=services/api`, `BINARY_NAME=api`        | distroless static, ~10 MB                  |
| `Dockerfile.python-ai`  | repo root     | `PY_VERSION=3.12`                                       | python:3.12-slim + venv, ~120 MB          |
| `Dockerfile.web`        | repo root     | `NODE_VERSION=22`, `NEXT_PUBLIC_API_URL`                | distroless nodejs22 + standalone Next.js   |

## Building

```bash
# From repo root (so build context includes packages/, adapters/, services/)

# Go service
docker build -f infrastructure/docker/Dockerfile.go \
  --build-arg SERVICE_PATH=services/api \
  --build-arg BINARY_NAME=api \
  -t civic-intelligence/api:dev .

# Python AI service
docker build -f infrastructure/docker/Dockerfile.python-ai \
  -t civic-intelligence/ai:dev .

# Next.js web
docker build -f infrastructure/docker/Dockerfile.web \
  --build-arg NEXT_PUBLIC_API_URL=https://api.civic.example \
  -t civic-intelligence/web:dev .
```

## Local dev stack

```bash
cd infrastructure/docker
docker compose up -d
docker compose ps
```

Services exposed on host:

| Service            | Port(s)     | UI                                  |
| ------------------ | ----------- | ----------------------------------- |
| Postgres + pgvector | 5432        | —                                   |
| Redis              | 6379        | —                                   |
| NATS (JetStream)   | 4222, 8222  | http://localhost:8222               |
| MinIO              | 9000, 9001  | http://localhost:9001               |
| OpenSearch         | 9200, 9600  | http://localhost:9200               |
| Temporal           | 7233, 8233  | http://localhost:8233               |
| MailHog (dev SMTP) | 1025, 8025  | http://localhost:8025               |
| Keycloak (OIDC)    | 8080        | http://localhost:8080               |
| OTEL Collector     | 4317, 4318  | —                                   |
| Prometheus         | 9090        | http://localhost:9090               |
| Grafana            | 3000        | http://localhost:3000 (admin/admin) |
| Loki               | 3100        | —                                   |
| Tempo              | 3200        | http://localhost:3200               |

**All passwords are for LOCAL DEVELOPMENT ONLY.** Production secrets come from
Vault / Sealed Secrets / AWS SSM.
