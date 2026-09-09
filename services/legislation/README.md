# Civic Intelligence — legislation service

The **canonical civic-domain service**. Owns the single source of civic truth:
Bills, Acts, stages, events, committees, people.

## Bounded context

**Owns:**
- Bills, Bill versions (immutable), Bill events, amendments, clauses
- Acts, regulations, policies, government decisions
- Hansard, Order Papers, committee reports, votes, gazette notices
- People, institutions, legislatures, houses, committees, parties, constituencies, counties
- Bill state machine (validates stage transitions using country-supplied definitions)

**Does NOT own:**
- How a Bill's stage is spelled — that's adapter data (`adapters/kenya/`)
- Where a document came from — that's `services/ingestion/`
- What the Bill text means in plain language — that's `services/intelligence/` (Go) + `services/ai/` (Python)
- How citizens find Bills — that's `services/search/`
- Whether a Bill actually changed stage — that's this service, but only via validated candidate facts

## Architectural rules

1. **No infrastructure imports in `internal/domain/`** — no `database/sql`, `net/http`, NATS, OpenAI.
2. **No country-specific strings** — no "Senate", "National Assembly", "Second Reading". Those are values supplied by the country adapter.
3. **AI never writes here directly** — only via validated candidate facts from `services/intelligence/`.

## Bill state machine

The `BillStateMachine` takes a `[]StageDefinition` (country-supplied) and validates
transitions. Terminal stages (REJECTED, WITHDRAWN, LAPSED) are always reachable
from any non-terminal stage. The state machine contains ZERO country-specific
strings.

```go
sm := domain.NewBillStateMachine(kenyaStages)  // kenyaStages supplied by the Kenya adapter
err := sm.ValidateTransition("FIRST_READING", "SECOND_READING")  // nil = valid
err = sm.ValidateTransition("FIRST_READING", "PRESIDENTIAL_ASSENT")  // error = invalid
```

## Running

```bash
go run ./cmd/main.go
# Listening on :9001 (override with LEGISLATION_SERVICE_ADDR)
```

## Tests

```bash
go test ./internal/domain/... -v
```
