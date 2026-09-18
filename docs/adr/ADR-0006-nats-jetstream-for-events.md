# ADR-0006: NATS JetStream for events

## Status

Accepted — 2026-09-09

## Context

The platform needs an event backbone for asynchronous workflows: ingestion publishes `source.discovered`, documents publishes `document.parsed`, intelligence publishes `ai.explanation.generated`, etc. Options:

1. **Kafka** — battle-tested, huge throughput, but operationally heavy (Zookeeper/KRaft, partitions, rebalancing).
2. **RabbitMQ** — solid, simpler, but no native streaming semantics.
3. **NATS JetStream** — lightweight, native streaming, persisted, simple to operate, supports subjects + queues + durable consumers.

## Decision

Adopt **NATS JetStream** as the event backbone.

## Consequences

- **Positive**: single binary, low memory footprint, simple to operate in Kubernetes.
- **Positive**: native support for durable consumers, at-least-once delivery, subject hierarchies (`bill.updated`, `bill.stage_changed`).
- **Positive**: built-in monitoring endpoint (`http://nats:8222/healthz`).
- **Positive**: Go client is mature; Python client available for the AI service.
- **Negative**: smaller ecosystem than Kafka; fewer managed offerings; mitigated by self-hosting.
- **Negative**: less suitable for very high-throughput log-style workloads — but our volumes (Bills × stage changes × documents) are modest.

## Convention

- Event subjects use dotted notation: `bill.updated`, `bill.stage_changed`, `ai.explanation.generated`.
- Every event has a `correlation_id` (UUID) that propagates across services via OpenTelemetry baggage.
- Consumers are durable + named (e.g., `intelligence-service`, `notifications-service`).

## References

- ARCHITECTURE.md §7 (Event architecture)
- `infrastructure/docker/docker-compose.yml` (NATS service)
- ADR-0012 (SSE for streaming AI responses to clients — different problem; NATS handles internal events)
