# 04 — Country Adapters

This document describes the country-adapter pattern, the interface every adapter must implement, the seven things every adapter needs, and the "Senate is just another House" insight that shapes the entire domain. The first adapter is Kenya; the pattern is designed so that adding Uganda, Tanzania, Ghana, Nigeria, or South Africa touches no domain code.

The corresponding decision record is [ADR-0004](../adr/ADR-0004-country-adapter-pattern.md).

## The pattern

The domain model in `services/legislation/` is country-agnostic. A `Bill` has the same fields, the same relationships, and the same lifecycle regardless of where it was introduced. What differs between countries is:

- **Where** the bills come from (which parliament website, which API, which document format).
- **What** the stages are called (Kenya's "First Reading" is Uganda's "First Reading" but Ghana's "First Reading" is constitutionally distinct).
- **How** documents are structured (Kenya bills are PDFs with numbered clauses; some countries use HTML; some use DOCX).
- **What** the local terminology is (Kenya uses "memorandum of objects and reasons"; other jurisdictions use "explanatory memorandum" or "statement of compatibility").

Every country-specific concern lives in `adapters/<country>/`. The adapter is a self-contained Go module that plugs into two host services:

1. `services/ingestion/` — through the `SourceAdapter` interface (where to fetch, how to fetch, how to be polite).
2. `services/legislation/` — through the `OntologyAdapter` interface (what entities exist locally, how they map to canonical types).

Adapters never import each other. Adapters never import `services/search/`, `services/intelligence/`, or `services/ai/`. The dependency direction is strictly one-way: adapters depend on the domain ports; the domain does not depend on adapters.

## The interface

Every country adapter implements two interfaces, both defined in `packages/contracts/adapters/`:

```go
// SourceAdapter is the ingestion-side interface.
type SourceAdapter interface {
    Country() CountryCode                              // "KE", "UG", "TZ"
    Sources() []SourceDefinition                        // authoritative URLs and policies
    Fetch(ctx context.Context, src SourceDefinition)    // produces SourceDocuments
        ([]SourceDocument, error)
    ResolveURL(raw string) (string, error)              // SSRF-allowlist normalization
}

// OntologyAdapter is the domain-side interface.
type OntologyAdapter interface {
    Country() CountryCode
    Institutions() []Institution                        // houses, committees, ministries
    BillStages() []BillStageDefinition                  // ordered, with transition rules
    DocumentParsers() []DocumentParserRef               // pointers to parsers in adapters/<cc>/
    Terminology() TerminologyGlossary                   // local-term → canonical-term
    DataQualityRules() []DataQualityRule                // validation rules
}
```

A new adapter ships:

- `adapters/<cc>/sources.yaml` — the source registry.
- `adapters/<cc>/ontology.go` — the `OntologyAdapter` implementation.
- `adapters/<cc>/parsers/` — document parsers (PDF, HTML, DOCX → structured clauses).
- `adapters/<cc>/terminology.yaml` — the glossary.
- `adapters/<cc>/quality.go` — the data-quality rules.
- `adapters/<cc>/testdata/` — golden documents and expected parser outputs.
- `adapters/<cc>/adapter_test.go` — runs the parser against golden documents.

The adapter is wired into the platform by adding a single line to `services/ingestion/registry.go` and `services/legislation/registry.go`. Nothing else in the domain changes.

## The seven things every adapter needs

### 1. Authoritative sources

Every adapter declares the URLs and APIs it treats as authoritative: the parliament website, the gazette portal, the Hansard archive, the committee pages. Each source has a fetch policy (cadence, robots-respect, auth), a trust level (authoritative for official sources, third-party for news or aggregator sources), and an SSRF allowlist entry. For Kenya, the authoritative sources include `parliament.go.ke`, the Kenya Gazette e-registry, and the county assembly websites. Sources are registered in `adapters/<cc>/sources.yaml` and loaded by `services/ingestion/` at startup.

### 2. Legislative ontology

The adapter declares the country's institutions: the legislature (Parliament of Kenya), the houses (National Assembly, Senate), the committees (Departmental Committee on Finance, etc.), and the executive institutions (Presidency, Cabinet, ministries). Each maps onto the canonical `Institution`/`Legislature`/`House`/`Committee` types with country-specific identifiers preserved in a `country_code`/`country_id` pair. The ontology is static within a legislative term; it is updated when elections change the institution set.

### 3. Stage definitions

Every country has a different sequence of stages a bill passes through. Kenya's is: `FirstReading → SecondReading → CommitteeOfTheWholeHouse → ThirdReading → PresidentialAssent → Commencement`. The adapter declares this sequence, the rules for when a bill can transition between stages (a bill cannot be `Assented` without a `ThirdReading`), and the events that mark each transition. The domain enforces the rules; the adapter declares them.

### 4. Document parsers

The adapter ships parsers that turn the country's document formats into the canonical structured form (`Clause`, `Section`, `Page`). Kenya bills are PDFs with numbered clauses and schedules; the Kenya parser extracts clauses, sub-clauses, schedules, and the memorandum of objects and reasons. Parsers are tested against golden documents (see #6). Where a country reuses a common format (HTML bills), the adapter may reuse a shared parser and add only the country-specific layout handling.

### 5. Terminology glossary

The adapter ships a glossary that maps local terms to canonical concepts. "Memorandum of Objects and Reasons" → `ExplanatoryMemorandum`. "Hansard" → `VerbatimProceedings`. "Order Paper" → `DailyAgenda`. This glossary is used by the AI capabilities (so the LLM knows that "Hansard" in a Kenyan document means the same thing as "Congressional Record" in an American one) and by the search index (so a search for "memorandum" finds Kenyan bills). The glossary is bidirectional: local→canonical for ingestion, canonical→local for display.

### 6. Tests against golden documents

Every parser ships with golden documents in `adapters/<cc>/testdata/` — real PDFs, HTML pages, and API responses from the country's civic sources — and the expected parsed output. The test suite asserts that the parser reproduces the expected output exactly. This catches regressions when parsers are refactored and provides a concrete specification of what the parser does. Golden documents are curated, never auto-generated from the current parser output (that would make the tests tautological).

### 7. Data-quality rules

The adapter declares validation rules for the data it ingests: a bill must have a sponsor; a bill version must have a publication date; a stage transition must have a source citation; an amendment must target an existing clause; a vote must have a motion. Rules are declarative (written in `quality.go` as functions or as a YAML rules file) and run on every ingest. Failures are recorded as data-quality issues, not as fatal errors — the platform prefers "bill X is missing a sponsor (flagged)" over "bill X was rejected (lost)."

## The "Senate is just another House" insight

This insight shapes the entire domain. The Kenya Parliament is bicameral: the National Assembly and the Senate. A naive model would treat them as different entity types. The correct model treats them as two instances of the same type — `House` — that happen to have different compositions, different rules, and different roles, but the same lifecycle for a bill passing through them.

This is why a bill in Kenya can be introduced in either house, passed by both, and assented to by the President — the domain records the bill's stage transitions through each house uniformly. The same model handles Uganda's unicameral Parliament (one House) and Ghana's unicameral Parliament (one House). Adding a country with a tricameral structure (rare, but theoretically possible — some historical legislatures) would add a third `House`; the domain code would not change.

The insight generalizes: every country-specific structure (committee types, bill types, vote types, document types) is an instance of a canonical type with country-specific metadata, not a new type. This is what makes the adapter pattern work — the domain is a small set of types with rich relationships; the adapter supplies the instances.

## Adding a new country — checklist

When you add a new country adapter, work through this checklist in order. Each item corresponds to one of the seven things above plus the wiring.

1. **Open an ADR** documenting the country choice, the authoritative sources, and any unusual structural features of the legislature.
2. **Create `adapters/<cc>/`** with the seven files (sources, ontology, parsers, terminology, quality, testdata, tests).
3. **Implement `SourceAdapter`** with the country's authoritative sources and fetch policies.
4. **Implement `OntologyAdapter`** with the country's institutions, stages, parsers, terminology, and quality rules.
5. **Curate golden documents** — at least one bill PDF, one Hansard PDF, one gazette notice — and assert the parser output.
6. **Run the data-quality rules** against the first batch of ingested data and triage the failures.
7. **Add a row to the supported-countries table** in this document and in [`docs/architecture/01-overview.md`](./01-overview.md) and [`docs/architecture/13-roadmap.md`](./13-roadmap.md).
8. **Open a PR** with the maintainers as reviewers. The first adapter from a new country gets extra scrutiny.

## Currently supported countries

| Country | Code | Adapter directory | Phase | Status |
| --- | --- | --- | --- | --- |
| Kenya | KE | `adapters/kenya/` | Phase 2 | In development |
| Uganda | UG | `adapters/uganda/` | Phase 8 | Planned |
| Tanzania | TZ | `adapters/tanzania/` | Phase 8 | Planned |
| Ghana | GH | `adapters/ghana/` | Phase 8 | Planned |
| Nigeria | NG | `adapters/nigeria/` | Phase 8 | Planned |
| South Africa | ZA | `adapters/southafrica/` | Phase 8 | Planned |

Adding a row to this table is a PR; the actual adapter work follows the checklist above.
