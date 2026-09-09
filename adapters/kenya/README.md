# Kenya adapter

The reference implementation of `contracts.LegislativeSourceAdapter`. All
Kenya-specific knowledge lives here.

## Sources covered

| Source                 | URL                              | Adapter file                | Status        |
| ---------------------- | -------------------------------- | --------------------------- | ------------- |
| Parliament of Kenya    | https://parliament.go.ke         | `parliament/parliament.go`  | Stub          |
| National Assembly       | https://nationalassembly.go.ke   | `parliament/parliament.go`  | Stub          |
| Senate                 | https://senate.go.ke             | `parliament/parliament.go`  | Stub          |
| Kenya Law Reports      | https://kenyalaw.org             | `kenya_law/kenya_law.go`    | Stub          |
| Kenya Gazette          | (via Kenya Law)                  | `gazette/gazette.go`        | Stub          |

## Document types supported

- Bills (parliament.go.ke Bill Tracker)
- Hansard (debate transcripts)
- Order Papers (daily agenda)
- Votes and Proceedings
- Committee reports
- Acts of Parliament (Kenya Law)
- Subsidiary legislation / regulations
- Legal notices
- Gazette notices

## Architectural rules

1. **All Kenya-specific strings live here.** The global domain model in `services/legislation/` contains ZERO of: "Senate", "National Assembly", "Second Reading", "Hansard", "Order Paper", "Gazette Notice", "Presidential Assent", etc.
2. **The adapter is a plugin**, loaded by the ingestion service. It is NOT a separately deployed service.
3. **Every claim extracted carries a source URL + content hash** — for evidence.
4. **Never fabricate.** If a Bill's stage cannot be determined from the source, return `Stage: ""` (empty), `Confidence: unknown`.
5. **Polite client.** Respects robots.txt, rate-limited to 1 req/sec per host, identifies itself with a contact email in the User-Agent.

## Kenya-specific data

The following are values supplied to the global domain model:

### Bill stages (11 total)
`FIRST_READING`, `SECOND_READING`, `COMMITTEE_STAGE`, `COMMITTEE_OF_WHOLE_HOUSE`, `REPORT_STAGE`, `THIRD_READING`, `PRESIDENTIAL_ASSENT`, `COMMENCEMENT` (terminal), `REJECTED` (terminal), `WITHDRAWN` (terminal), `LAPSED` (terminal).

### Terminology (30 terms)
See `internal/kenya_data.go` for the full registry. Includes: First/Second/Third Reading, Committee Stage, Committee of the Whole House, Report Stage, Presidential Assent, Commencement, Hansard, Order Paper, Votes and Proceedings, Gazette Notice, Government Bill, Private Member's Bill, Money Bill, Mediation Committee, County Government, CDF, Bicameral, Prorogation, Sine Die, Quorum, Division, Mover, Seconder, Clerk of the Senate, Speaker of the National Assembly, Attorney General, Solicitor General, Public Participation.

### Institutions
Parliament of Kenya (legislature), Office of the Attorney General (constitutional body), Judiciary, IEBC, EACC, Controller of Budget, Auditor General.

### Legislature + Houses
- Legislature: Parliament of Kenya
- Houses: National Assembly (sort_order=1), Senate (sort_order=2)

### Committees (sample)
- National Assembly: Departmental Committee on Lands, on Finance and National Planning, on Education and Research, on Health, on Transport, on Justice and Legal Affairs; Public Accounts Committee; Public Investments Committee.
- Senate: Standing Committee on Information/Communication/ICT, on Finance and Budget, on Justice/Legal Affairs/Human Rights, on Health, on Education; County Public Accounts and Investments Committee.

## Running tests

```bash
cd adapters/kenya
go test ./... -v
```

## Onboarding a new country

Use `adapters/kenya/` as the template. A new country adapter MUST have all 7:

1. Authoritative sources (SourceRegistry entries)
2. Legislative ontology (institutions, houses, committees)
3. Stage definitions (code, name, simple_explanation, allowed_next, is_terminal)
4. Document parsers (HTML/PDF/DOCX per source)
5. Terminology registry (≥25 terms with sources)
6. Tests (contract + stage validity + terminology completeness)
7. Data-quality rules (duplicate detection, date sanity, cross-source consistency)

See [ADR-0004](../../docs/adr/ADR-0004-country-adapter-pattern.md) for the full rationale.
