# ADR-0002: Go for backend, Python for AI

## Status

Accepted — 2026-09-09

## Context

The platform has two very different workload profiles:

1. **Civic domain + HTTP API + ingestion pipeline**: high concurrency, low latency, strict typing, long-lived processes, I/O-bound (database, NATS, object storage, HTTP).
2. **AI workloads**: heavy library ecosystem (transformers, embeddings, evaluation frameworks), integration with multiple LLM providers, rapid experimentation, structured output validation.

## Decision

- **Go** for all backend services (api, legislation, ingestion, documents, evidence, intelligence, search, notifications, identity) + the country-adapter plugin system.
- **Python (FastAPI)** for the AI service (`services/ai/`).
- The Python AI service is called by the Go `intelligence` service via HTTP; the Go service owns candidate-fact validation + promotion to canonical state.

## Consequences

- **Positive**: Go's static typing + goroutines fit the high-concurrency backend well; the Go services share a single language + toolchain.
- **Positive**: Python's AI ecosystem (Pydantic, FastAPI, transformers, eval frameworks) is unmatched — using Python for AI avoids re-implementing library functionality in Go.
- **Positive**: clear ownership — Go team owns canonical state + domain rules; AI team owns models + prompts + evaluation.
- **Negative**: two languages to deploy + monitor; mitigated by shared OpenTelemetry conventions.
- **Negative**: cross-language contract drift risk; mitigated by generating types from OpenAPI + a shared JSON schema package (`packages/contracts/` mirrors in Go).

## References

- ARCHITECTURE.md §6 (AI cannot mutate truth rule)
- ADR-0008 (evidence-first RAG)
