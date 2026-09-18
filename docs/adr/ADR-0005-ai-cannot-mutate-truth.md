# ADR-0005: AI cannot mutate truth (candidate-fact validation boundary)

## Status

Accepted — 2026-09-09

## Context

The platform's headline feature is AI that explains government information. The biggest architectural risk is that AI silently mutates canonical civic state — for example, an LLM reads a Bill PDF and directly updates `legislation.bills.current_stage`. This would:

- Destroy the audit trail (no human decided the stage changed).
- Introduce hallucinated facts into canonical truth.
- Make the platform untrustable.

## Decision

Codify the rule at the schema, code, and process levels:

### Schema level
The `intelligence.candidate_facts` table has a CHECK constraint:
```sql
CHECK ((validated = FALSE) = (accepted_at IS NULL))
```
AI can INSERT into `intelligence.candidate_facts` but CANNOT set `validated=TRUE` or `accepted_at`. Only the `services/intelligence` (Go) service can, after validation.

### Code level
The Python AI service has NO database credentials on the `legislation` schema. It can read from `intelligence` + `documents` schemas only. It calls the Go `intelligence` service via HTTP to propose candidate facts.

### Process level
PR review checklist includes: "If touching AI: every claim is cited; eval dataset passes; no direct writes to `legislation.*`".

## Consequences

- **Positive**: AI hallucinations can never silently become canonical facts.
- **Positive**: every canonical change has a clear author (the validation step, which may be human or rule-based).
- **Positive**: the audit log captures the full chain: AI proposed → validated → promoted.
- **Negative**: latency — there's an extra hop for candidate facts to become canonical. Mitigated by: (a) auto-validation rules for high-confidence cases (e.g., stage from official Order Paper), (b) async processing via NATS.

## Flow

```
PDF (source)
   ↓
documents service: parse + chunk + embed
   ↓
AI service: extract candidate facts (with citations)
   ↓
intelligence service (Go):
   ↓ validate each candidate fact
   ↓ if validation passes:
   ↓     INSERT candidate_facts(validated=TRUE, accepted_at=NOW())
   ↓     emit bill.updated event
   ↓     legislation service consumes event → updates canonical state
   ↓ if validation fails:
   ↓     INSERT candidate_facts(validated=FALSE)
   ↓     emit ai.validation.failed event → observability + alerting
```

## References

- ARCHITECTURE.md §5 (The "AI cannot mutate truth" rule)
- `services/ai/app/citation_validator.py`
- `infrastructure/postgres/migrations/013_intelligence.up.sql` (CHECK constraint)
