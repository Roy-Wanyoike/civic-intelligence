# ADR-0004: Country adapter pattern (never hard-code Kenya in the domain)

## Status

Accepted — 2026-09-09

## Context

The platform's first country is Kenya. But the architecture must eventually support Uganda, Tanzania, Ghana, Nigeria, and South Africa without rebuilding the core. The biggest mistake would be hard-coding Kenya's specifics into the global domain model:

- Kenya has a bicameral legislature (National Assembly + Senate) — but not every country does.
- Kenya's Bill stages (First Reading, Second Reading, Committee Stage, Committee of the Whole House, Report Stage, Third Reading, Presidential Assent, Commencement) — not universal.
- Kenyan terminology ("Hansard", "Order Paper", "Votes and Proceedings", "Gazette Notice") — country-specific.

## Decision

Implement a **country adapter pattern**:

```go
type LegislativeSourceAdapter interface {
    Discover(ctx context.Context) ([]SourceItem, error)
    Fetch(ctx context.Context, item SourceItem) (*RawDocument, error)
    Parse(ctx context.Context, doc RawDocument) ([]ExtractedRecord, error)
    GetLegislativeStructure(ctx context.Context) (*LegislativeStructure, error)
    GetStages(ctx context.Context) ([]StageDefinition, error)
    GetTerminology(ctx context.Context) ([]TermDefinition, error)
}
```

All Kenya-specific code lives in `adapters/kenya/`. The global domain model in `services/legislation/` contains ZERO Kenya-specific strings. Stage names ("Second Reading") are values stored in `legislation.bill_stages` rows with `country_id='KE'`, never column names or Go constants.

## Consequences

- **Positive**: adding Uganda means writing `adapters/uganda/` + adding rows to `legislation.countries`, `legislation.bill_stages`, `legislation.institutions`. The legislation service code is unchanged.
- **Positive**: contract tests can assert that the adapter satisfies the interface — preventing silent drift.
- **Positive**: the domain team and the country-adapter team can work independently.
- **Negative**: the indirection adds complexity — the legislation service must always look up stage definitions via the adapter config, never inline them.
- **Negative**: Kenya-specific optimizations (e.g., special handling of the Senate's Mediation Committee) must be expressed as adapter data, not as code branches in the domain.

## Onboarding checklist for a new country

A country adapter MUST have all 7 before it can be merged:

1. Authoritative sources (SourceRegistry entries).
2. Legislative ontology (institutions, houses, committees).
3. Stage definitions (code, name, simple_explanation, allowed_next, is_terminal).
4. Document parsers (HTML/PDF/DOCX per source).
5. Terminology registry (≥25 terms with sources).
6. Tests (contract + stage validity + terminology completeness).
7. Data-quality rules (duplicate detection, date sanity, cross-source consistency).

## References

- ARCHITECTURE.md §6 (Country adapter pattern)
- `packages/contracts/country.go` (interface)
- `adapters/kenya/` (reference implementation)
