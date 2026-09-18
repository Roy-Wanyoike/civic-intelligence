# Kenya adapter

The reference implementation of `contracts.LegislativeSourceAdapter`. All
Kenya-specific knowledge lives here.

## Sources covered

| Source                 | URL                                                                                 | Adapter file                | Status        |
| ---------------------- | ----------------------------------------------------------------------------------- | --------------------------- | ------------- |
| Parliament of Kenya    | https://www.parliament.go.ke                                                        | `parliament/parliament.go`  | Live (Bills)  |
| National Assembly       | https://www.parliament.go.ke/the-national-assembly/house-business/bills             | `parliament/parliament.go`  | Live (Bills)  |
| Senate                 | https://www.parliament.go.ke/the-senate/senate-bills                                | `parliament/parliament.go`  | Live (Bills)  |
| Bill Tracker           | https://www.parliament.go.ke/the-national-assembly/house-business/bill-tracker      | `parliament/parliament.go`  | Live (index)  |
| Kenya Law Reports      | https://new.kenyalaw.org/bills/                                                     | `kenya_law/kenya_law.go`    | Live (Bills)  |
| Kenya Gazette          | (via Kenya Law)                                                                     | `gazette/gazette.go`        | Stub          |

### Live-site reconnaissance (2026-09-09)

| URL                                                              | Status      | Notes                                                                          |
| ---------------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------ |
| `https://www.parliament.go.ke/`                                   | HTTP 200    | Drupal 8 root; lists Bills + Hansard + Order Papers + Votes & Proceedings.     |
| `https://www.parliament.go.ke/bills`                              | HTTP 404    | Use the per-house URLs below instead.                                          |
| `https://www.parliament.go.ke/the-national-assembly/house-business/bills` | HTTP 200 | National Assembly Bills listing; one `<div class="post-block">` per Bill.      |
| `https://www.parliament.go.ke/the-senate/senate-bills`           | HTTP 200    | Senate Bills listing; same `post-block` structure as NA.                       |
| `https://www.parliament.go.ke/the-national-assembly/house-business/bill-tracker` | HTTP 200 | Weekly "Bills Tracker as at <DATE>" PDF index; per-Bill stage info lives in those PDFs. |

Each Bill in the Bills listing has the following structure:

```html
<div class="post-block">
  <div class="post-content">
    <div class="post-title">
      <a href="…/sites/default/files/YYYY-MM/<BILL>.pdf" title="…">Bill Title</a>
    </div>
    <div class="post-meta">
      <span class="post-digest">Bill Digest: <a href="…">…</a></span>
      <span class="post-billtracker">Bill Tracker: <a href="…">…</a></span>
      <span class="post-petition">
        <a href="…/contact/<house>_petition?bill=…">Submit Comments</a>
      </span>
    </div>
  </div>
</div>
```

The `post-billtracker` link slot is currently empty on the live site for
most recently published Bills; per-Bill stage data is published only inside
the weekly tracker PDFs at `/the-national-assembly/house-business/bill-tracker`.
`Adapter.FetchBillTracker` therefore returns an empty stage with low
confidence when the per-Bill tracker slot is empty, and the documents service
is expected to schedule a follow-up PDF-parsing job to extract per-Bill
stage rows from the weekly tracker PDF.

## Document types supported

- Bills (parliament.go.ke Bills listing — `parliament.ParseBillsListing`)
- Bill Tracker (weekly PDF index — `parliament.ParseBillTrackerHTML` for the HTML wrapper; PDF parsing TODO #CI-AD-005)
- Hansard (debate transcripts — `parliament.ParseHansardHTML`)
- Order Papers (daily agenda — `parliament.ParseOrderPaperHTML`)
- Votes and Proceedings (`parliament.ParseVotesProceedingsHTML`)
- Committee reports (`parliament.ParseCommitteesHTML`)
- Acts of Parliament (Kenya Law — `kenya_law.ParseBillDetail`)
- Subsidiary legislation / regulations
- Legal notices
- Gazette notices

## Architectural rules

1. **All Kenya-specific strings live here.** The global domain model in `services/legislation/` contains ZERO of: "Senate", "National Assembly", "Second Reading", "Hansard", "Order Paper", "Gazette Notice", "Presidential Assent", etc.
2. **The adapter is a plugin**, loaded by the ingestion service. It is NOT a separately deployed service.
3. **Every claim extracted carries a source URL + content hash** — for evidence.
4. **Never fabricate.** If a Bill's stage cannot be determined from the source, return `Stage: ""` (empty), `Confidence: unknown`.
5. **Polite client.** Respects robots.txt, rate-limited to 1 req/sec per host, identifies itself with a contact email in the User-Agent.

## Stage tracking

The Parliament Bill Tracker is the canonical source of stage data for Kenyan
Bills. The flow:

1. `Adapter.DiscoverBills(ctx)` crawls the NA + Senate Bills listing pages
   and returns `[]BillCandidate` (URL, Title, House, PetitionURL,
   BillDigestURL, BillTrackerURL).
2. For each Bill, `Adapter.FetchBillTracker(ctx, billURL)` fetches the
   Bill's tracker page (if linked) or the weekly tracker PDF and returns
   a `*BillTracker` with Stage, Status, Date, SourceURL, RetrievedAt, and
   Confidence.
3. `Adapter.ParseStage(ctx, text)` maps Parliament's raw stage text
   (e.g., "Second Reading — 12 March 2026") to the canonical stage codes
   (FIRST_READING, SECOND_READING, …) defined in `internal/stages.go`.

Stage codes are imported (not duplicated) from `adapters/kenya/internal/stages.go`
to keep a single source of truth.

## Kenya-specific data

The following are values supplied to the global domain model:

### Bill stages (12 total)
`FIRST_READING`, `SECOND_READING`, `COMMITTEE_STAGE`, `COMMITTEE_OF_WHOLE_HOUSE`, `REPORT_STAGE`, `THIRD_READING`, `PRESIDENTIAL_ASSENT`, `COMMENCEMENT` (terminal), `MEDIATION`, `REJECTED` (terminal), `WITHDRAWN` (terminal), `LAPSED` (terminal).

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
go test ./parliament/... -v   # 51 tests (stage mapping, bill discovery, contract compliance)
go test ./...                 # full module
```

Test fixtures in `parliament/testdata/` were captured live on 2026-09-09
using `curl -sS -A "CivicIntelligence/0.1" <URL>` and stripped of
CSS/JS noise. The `bill_tracker_detail.html` fixture is a synthetic
representation of the per-Bill tracker page structure Parliament will
publish when the `post-billtracker` link slot is populated.

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
