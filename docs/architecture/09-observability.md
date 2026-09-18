# 09 — Observability

This document describes the observability architecture: OpenTelemetry instrumentation points, the metrics catalog, dashboards, and the alerting philosophy. The platform's observability goal is that any engineer, on call or off, can answer "is the platform healthy?" in 30 seconds and "why is it unhealthy?" in 5 minutes — without SSHing into a box.

The three pillars are logs, metrics, and traces, all correlated through a shared trace context propagated from the API through workers to the database. The platform uses OpenTelemetry for instrumentation, Prometheus + Grafana for metrics and dashboards, Loki for structured logs, and Tempo for distributed traces. All four are open source and self-hosted; no observability vendor lock-in.

## OpenTelemetry instrumentation points

Every service in the platform is instrumented with the OpenTelemetry SDK (Go for the Go services, Python for the AI worker, Next.js for the web app). The instrumentation is layered:

1. **Inbound.** Every HTTP and gRPC handler creates a span. The span's attributes include the route, the method, the user ID (if authenticated), the request ID, and the status code.
2. **Outbound.** Every HTTP and gRPC call from one service to another creates a child span. The span's attributes include the target service, the target route, the latency, and the status.
3. **Database.** Every SQL query creates a child span (via the pgx instrumented driver in Go, via the asyncpg instrumentation in Python). The span's attributes include the statement (with parameters redacted), the rows affected, and the latency.
4. **Queue.** Every NATS publish and consume creates a span. The publish span links to the consume span through the message's trace context header, so a message's journey from publish to consume to handler-completion is visible as a single trace.
5. **Workflow.** Every Temporal workflow execution creates a top-level span; each activity is a child span. The workflow's trace is linked to the trigger (the event or the API call that started it).
6. **AI calls.** Every LLM call creates a span with the provider, the model, the prompt version, the input token count, the output token count, the latency, and the cost. These spans are tagged `ai` so they can be filtered and aggregated separately.
7. **Crawler calls.** Every fetch creates a span with the source, the URL (truncated), the HTTP status, the content length, and the SSRF-check result.

The trace context (W3C `traceparent` header) is propagated across process boundaries: API → worker (via NATS message headers), worker → worker (via Temporal workflow context), worker → database (via pgx's trace context). A single citizen request — say, "ask a question about bill X" — produces a single trace that spans the API, the search service, the intelligence gateway, the AI worker, the LLM provider (visible as a child span of the AI call), the evidence validator, and the database queries each one makes. This end-to-end trace is the primary debugging tool.

## Metrics catalog

The platform exposes Prometheus metrics on a `/metrics` endpoint per service. The catalog below is the canonical list — every metric the platform defines, with its type, dimensions, and the alert it triggers. Adding a metric is a PR; removing or renaming one is a PR with a deprecation cycle.

### Ingestion

| Metric | Type | Dimensions | Alert |
| --- | --- | --- | --- |
| `crawl_success_rate` | gauge | source, country | Alert if < 95% over 1h rolling window |
| `crawl_latency_p50` / `p99` | summary | source | Alert if p99 > 30s |
| `crawl_bytes_fetched` | counter | source | Capacity planning |
| `ssrf_block_count` | counter | source, reason | Alert on any non-zero (indicates a misconfigured source) |

### Documents

| Metric | Type | Dimensions | Alert |
| --- | --- | --- | --- |
| `document_parse_failure_rate` | gauge | parser_type | Alert if > 5% over 1h |
| `ocr_failure_rate` | gauge | source | Alert if > 10% (likely a source-format change) |
| `parse_latency_p50` / `p99` | summary | parser_type | Alert if p99 > 60s |
| `embeddings_generated_total` | counter | model_version | Capacity and cost tracking |

### Evidence

| Metric | Type | Dimensions | Alert |
| --- | --- | --- | --- |
| `citation_validation_failure_rate` | gauge | capability | Alert if > 5% (the AI is producing bad citations) |
| `candidate_fact_acceptance_rate` | gauge | capability | Alert if < 50% (the AI is producing low-quality facts) |
| `source_conflict_count` | counter | conflict_type | Alert on spike (sources changing in unusual ways) |
| `contradiction_engine_latency_p99` | summary | — | Alert if > 5s |

### Intelligence / AI

| Metric | Type | Dimensions | Alert |
| --- | --- | --- | --- |
| `ai_failure_rate` | gauge | capability, provider | Alert if > 5% over 5m |
| `llm_latency_p50` / `p99` | summary | capability, provider, model | Alert if p99 > 30s for citizen-facing, 5m for batch |
| `llm_cost_usd_total` | counter | capability, provider | Alert on daily-budget breach (per-tier) |
| `llm_token_count` | counter | capability, provider, direction (in/out) | Capacity and cost |
| `eval_pass_rate` | gauge | capability | Alert if < 95% (a release would be blocked anyway) |
| `ai_hallucination_rate` | gauge | capability | Alert if > 1% |

### Search

| Metric | Type | Dimensions | Alert |
| --- | --- | --- | --- |
| `search_latency_p50` / `p99` | summary | query_type | Alert if p99 > 500ms |
| `search_result_count_avg` | gauge | query_type | Capacity and UX |
| `search_index_lag_seconds` | gauge | index_name | Alert if > 60s (projections are falling behind) |
| `search_zero_result_rate` | gauge | query_type | Alert if rising (possibly a parser regression) |

### API

| Metric | Type | Dimensions | Alert |
| --- | --- | --- | --- |
| `api_latency_p50` / `p99` | summary | route, method | Alert if p99 > 1s for read, 3s for AI |
| `api_request_count` | counter | route, status_code | Capacity |
| `api_error_rate` | gauge | route | Alert if > 1% 5xx over 5m |
| `api_rate_limit_hits` | counter | tier, route | Capacity and abuse signal |
| `sse_active_connections` | gauge | — | Capacity |
| `sse_message_latency_p99` | summary | — | Alert if > 200ms (streaming UX) |

### Workflows

| Metric | Type | Dimensions | Alert |
| --- | --- | --- | --- |
| `workflow_failure_rate` | gauge | workflow_name | Alert if > 2% over 1h |
| `workflow_duration_p50` / `p99` | summary | workflow_name | Alert if p99 > 10m for ingestion |
| `workflow_retry_count` | counter | workflow_name, activity | Alert on > 3 retries (something is wrong) |
| `queue_depth` | gauge | queue_name | Alert if > 1000 (backpressure) |

### Identity

| Metric | Type | Dimensions | Alert |
| --- | --- | --- | --- |
| `auth_failure_rate` | gauge | method | Alert if > 10% (possibly an attack, possibly a Keycloak outage) |
| `api_key_rotation_lag_days` | gauge | api_key_id | Alert if > 80 days (rotation policy) |

### Notifications

| Metric | Type | Dimensions | Alert |
| --- | --- | --- | --- |
| `notification_delivery_success_rate` | gauge | channel | Alert if < 95% |
| `notification_delivery_latency_p99` | summary | channel | Alert if > 30s for push, 5m for email |
| `notification_bounce_rate` | gauge | channel | Alert if > 2% (list hygiene) |

## Dashboards

Grafana dashboards are versioned in `infrastructure/observability/dashboards/` and provisioned automatically. The dashboard set:

- **Platform Overview.** Top-line health: request rate, error rate, p99 latency, queue depths, AI cost burn-down. The dashboard an on-call engineer looks at first.
- **Ingestion.** Per-source success rate, latency, byte volume, SSRF blocks.
- **Documents.** Parse success rate, OCR failures, parse latency by parser type.
- **Evidence.** Citation validation rate, candidate fact acceptance, contradiction count.
- **AI.** Per-capability latency, cost, hallucination rate, eval pass rate, LLM token counts.
- **Search.** Per-query-type latency, zero-result rate, index lag.
- **API.** Per-route latency, error rate, rate-limit hits, SSE connections.
- **Workflows.** Per-workflow duration, failure rate, retry count, queue depth.
- **Security.** Auth failure rate, role binding changes, API key rotation lag, audit-log write rate.
- **Cost.** Daily and monthly cost by service (LLM cost dominates; the dashboard shows per-capability and per-tenant breakdowns).

Each dashboard links to the relevant alert runbook in the on-call handbook.

## Alerting philosophy

The platform's alerting philosophy is: **alert on user-visible degradation, not on infrastructure noise.** A CPU spike on a worker pod is not an alert; a 99th-percentile API latency spike that affects citizens is. The rules:

1. **Actionable.** Every alert has a runbook with a concrete first step. Alerts without runbooks are noise; remove them.
2. **User-impact-aligned.** Alerts fire when the platform's behavior degrades for users — slower API, more errors, more hallucinations — not when internal metrics cross arbitrary thresholds. A 90% CPU alert is irrelevant if the API is responding in 50ms; an API p99 alert at 2s is critical even if CPU is 30%.
3. **Severity-tiered.** Two severities: `page` (wake someone up) and `ticket` (handle during business hours). The bar for `page` is "the platform is broken for citizens right now." Everything else is a ticket.
4. **Deduplicated.** Related alerts (e.g. "ingestion failure for source X" and "workflow failure for KenyaBillIngestionWorkflow") are deduplicated into a single incident where possible.
5. **Auto-resolving.** Alerts that auto-resolve when the metric returns to normal. No stale alerts in the queue.

The on-call rotation is one primary and one secondary engineer, weekly shifts. The runbook for every alert is in `infrastructure/observability/runbooks/` and is linked from the alert's Grafana panel. A page that does not have a runbook is a bug; file an issue.

## Log strategy

Logs are structured JSON, shipped to Loki, queried via LogQL. The log level defaults to `INFO` in production; `DEBUG` is enabled per-service on demand (via a runtime config, not a redeploy) for investigation. Logs include the trace ID and span ID from OpenTelemetry, so a log query can be scoped to a single request's trace.

PII is scrubbed at the logger level (see [SECURITY.md](../../SECURITY.md) for the policy). The field allowlist permits request IDs, user IDs (hashed), route names, and latency; it forbids request bodies, response bodies, tokens, and free-text user input. PII that needs to be logged for debugging (e.g. a citizen question that triggered a bug) is logged to a separate, access-gated debug log with a 7-day retention.

## Cost observability

AI cost is the platform's dominant variable cost. The cost dashboard tracks daily and monthly spend by capability, by provider, by tenant (for partner API keys), and by model version. Budget alerts fire at 50%, 80%, and 100% of the daily budget per tier. A capability that consistently overspends its budget is flagged for optimization (caching, smaller context, cheaper model) — not silently given more budget.
