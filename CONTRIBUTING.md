# Contributing to Civic Intelligence

Thank you for contributing. Please read this document before opening a PR.

## Branch model

- Trunk-based development with short-lived feature branches.
- Default branch: `main`. **Never push directly to `main`.**
- Branch naming: `<type>/<scope>-<short-description>` (e.g., `feat/legislation-bill-stage-machine`, `fix/ai-citation-validator`).
- Rebase your branch on `main` before requesting review.

## Every meaningful change corresponds to an issue + PR

- **Never** close an incomplete issue.
- **Never** merge an incomplete feature.
- **Never** create fake API responses and call them production functionality.
- **Never** claim functionality that does not work.
- **Never** push directly to `main`.

## Commit format (Conventional Commits)

```
<type>(<scope>): <subject>

<body>

<footer>
```

- `type`: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`, `build`
- `scope`: e.g., `legislation`, `ai`, `kenya-adapter`, `web`, `infra`
- Subject in imperative mood, ≤ 72 chars.

## Architectural rules (NON-NEGOTIABLE)

Read [ARCHITECTURE.md](./ARCHITECTURE.md) first. The headline rules:

1. **AI never writes to canonical legislative state.** Only the legislation service writes to `legislation.*`. AI proposes candidate facts (in `intelligence.candidate_facts`); a validation step (manual or rule-based) must accept them before they become canonical.
2. **Country-specifics stay in the adapter.** No Kenya-specific strings (`"Senate"`, `"Second Reading"`, `"National Assembly"`) outside `adapters/kenya/`. Stage codes (`SECOND_READING`) are fine as values; stage names are adapter data.
3. **Domain packages are pure.** No `database/sql`, `net/http`, NATS, OpenAI, or any infrastructure import inside `services/*/internal/domain/`.
4. **Repository-per-bounded-context.** Don't import another service's repository. Services communicate via HTTP or events.
5. **Migrations are forward-only.** Never edit a merged migration. Rollback = new migration.
6. **BillVersion is immutable.** Never UPDATE a row in `legislation.bill_versions`. Only INSERT.
7. **Country data is isolated.** A contributor working in `adapters/tanzania/` CANNOT accidentally modify Kenyan data. The seed packages, stages, terminology, and structures live in physically separate Go modules per country. See "Adding a New Country" below.

## PR checklist

- [ ] Linked issue (`Closes #NNN` or `Refs #NNN`)
- [ ] Tests added / updated
- [ ] Docs updated if needed (README, ARCHITECTURE, ADR)
- [ ] If touching AI: eval dataset passes; citation validator flags 0 unsupported claims on sample set
- [ ] If touching `legislation.*`: audit trigger fires correctly
- [ ] No country-specific strings outside `adapters/{country}/`
- [ ] No infrastructure imports in any `internal/domain/` package
- [ ] No direct writes from AI to `legislation.*`
- [ ] If touching a country adapter: `cd tests/contract && go test -count=1 ./...` passes — the country-isolation tests guard against cross-country contamination

## Adding a New Country

The platform supports any country. Here's how to add yours:

### 1. Create the adapter directory

```
adapters/yourcountry/
├── adapter.go          # Implements contracts.LegislativeSourceAdapter
├── parliament/
│   ├── parliament.go   # Discover + Parse + Fetch
│   └── testdata/       # HTML fixtures for tests
├── internal/
│   └── yourcountry_data.go  # Bill stages, legislative structure, terminology
├── yourcountry_seed/
│   ├── government.go   # Presidents, administrations, terms
│   ├── acts.go         # Sample Acts
│   └── public_debt.go  # Debt snapshots (if applicable)
├── contract_test.go    # Verifies interface conformance
└── go.mod              # Module definition
```

### 2. Implement the adapter

- Copy `adapters/kenya/` (or `adapters/uganda/` for a simpler unicameral example) as a template.
- Replace all Kenya-specific data with your country's data.
- Implement `Discover(ctx)` to scrape your parliament's bills page.
- Implement `Parse(ctx, doc)` to extract bill fields.
- Implement `fetchURL` directly (do NOT delegate to `Fetch`).
- Update `go.mod` to require `services/legislation` if you add a seed package (see `adapters/kenya/go.mod` for the pattern).

### 3. Add seed data

- Add your country's presidents and administrations to `yourcountry_seed/government.go`. Use the `government.President`, `government.Administration`, and `government.PresidentialTerm` types from `services/legislation/government`.
- Add 3–5 sample Acts to `yourcountry_seed/acts.go` (mirror the `SeedAct` DTO pattern in `adapters/kenya/kenya_seed/acts.go`).
- Add debt snapshots to `yourcountry_seed/public_debt.go` if your country publishes them (mirror the `KenyaDebtSnapshots` and `KenyaBorrowingAgreements` patterns). Each snapshot is an immutable observation — see Spec §9.
- Source every value: every President, every Act, every snapshot MUST carry a `source_url` pointing at an authoritative reference (parliament website, official gazette, central bank, etc.).

### 4. Register the adapter

In `services/api/cmd/main.go`:

```go
import (
    "github.com/Roy-Wanyoike/civic-intelligence/adapters/registry"
    "github.com/Roy-Wanyoike/civic-intelligence/adapters/yourcountry"
)

func main() {
    // ... existing setup ...
    registry.MustRegisterDefault() // registers KE, UG, TZ, GH, NG, ZA
    // Register your country's adapter alongside the defaults:
    registry.Register(registry.NewYourCountryWrapper(yourcountry.NewYourCountryAdapter(yourcountry.Dependencies{})))
}
```

If your adapter's constructor signature matches the existing patterns (Tanzania / Uganda / Ghana take no args; Kenya / Nigeria / South Africa take `Dependencies{}`), you can also add a `NewYourCountryWrapper(...)` constructor to `adapters/registry/wrappers.go` and register it inside `MustRegisterDefault()` so the test suite picks it up automatically.

### 5. Add the country page

Create `apps/web/src/app/country/yourcountry/page.tsx`:

- Show your parliament info, recent Bills, government structure
- Use the existing pattern from `/country/kenya/page.tsx`
- The country code in the URL must match the `CountryCode()` returned by your adapter.

### 6. Add to Government Selector

In `apps/web/src/lib/government-defaults.ts`:

- Add your country to the `SUPPORTED_COUNTRIES` list exported from this module.
- The `GovernmentSelector` component reads this list to render the dropdown.

### 7. Write tests

- `contract_test.go` — verify interface conformance with `contracts.LegislativeSourceAdapter`.
- `parliament_test.go` — test `Discover` + `Parse` with HTML fixtures in `testdata/`.
- Add your country to `tests/contract/country_isolation_test.go`:
  - Append a row to the `expectedCountries` table.
  - Append a row to the `knownDomains` and `foreignDomains` maps in `TestRegistry_OfficialSourcesCountryIsolation`.
  - The rest of the suite picks up your country automatically (it is table-driven).

### Important: Data Isolation

Your country's data is completely isolated:

- Your adapter lives in `adapters/yourcountry/`
- Your seed data lives in `adapters/yourcountry/yourcountry_seed/`
- Your tests live in `adapters/yourcountry/`
- Your country page lives in `apps/web/src/app/country/yourcountry/`
- Changing your data CANNOT affect any other country — the packages are physically separate Go modules with their own `go.mod`.
- The API routes requests to your adapter only when the `X-Civic-Country` header (or the `civic_gov_selection` cookie) matches your country code.
- The `tests/contract/country_isolation_test.go` suite enforces this invariant in CI — if a contributor accidentally copies a stage slice from another country and forgets to swap the country code, the test fails.

### Worked examples in the repo

- **Kenya** (`adapters/kenya/`) — bicameral, full feature set (parliament + president + kenya_law + gazette + seed).
- **Uganda** (`adapters/uganda/`) — unicameral, minimal (parliament only).
- **Tanzania** (`adapters/tanzania/`) — unicameral, with seed data for government, Acts, and debt.
- **Ghana** (`adapters/ghana/`) — unicameral, with seed data.
- **Nigeria** (`adapters/nigeria/`) — bicameral (Senate + House of Representatives), with seed data.
- **South Africa** (`adapters/south_africa/`) — bicameral (National Assembly + National Council of Provinces), with seed data.

## How to add a new AI capability

1. Create `services/ai/app/capabilities/<name>.py` with a class implementing the capability.
2. Add the capability to `app/capabilities/__init__.py` and `app/main.py` (route + DI registration).
3. Add eval cases to `services/ai/eval/test_eval_dataset.py` covering the new capability.
4. Ensure the capability:
   - Accepts evidence as input (citations).
   - Produces output that passes citation validation.
   - Distinguishes FACT / EXPLANATION / INFERENCE / UNKNOWN.
   - Never fabricates — uses the "Information could not be verified" escape when evidence is insufficient.

## Code review expectations

- Reviewer should verify the architectural rules are respected, not just that tests pass.
- If a PR touches `legislation.*` writes, the reviewer must verify the writer is `services/legislation` (not `intelligence` or `ai`).
- If a PR adds country-specific strings to global domain code, block it.
- If a PR's AI eval results show regression, block it.
- If a PR adds a new country adapter, the reviewer MUST verify:
  - Every president / administration / Act / debt snapshot carries the new country's own code (not a copied neighbour's).
  - The adapter's `GetStages()` returns stages whose `Country` field is the new country code.
  - The `GetOfficialSources()` URLs all belong to the new country's own domains.
  - The `tests/contract/country_isolation_test.go` suite is updated with the new country in `expectedCountries`.

## Local development

See [README.md](./README.md) Quickstart.

### Useful verification commands

```bash
# Backend (API service):
cd services/api && go test -count=1 ./...

# Contract (all 6 country adapters + isolation tests):
cd tests/contract && go test -count=1 ./...

# Frontend type-check:
cd apps/web && npx tsc --noEmit

# Per-adapter tests:
cd adapters/kenya    && go test -count=1 ./...
cd adapters/uganda   && go test -count=1 ./...
cd adapters/tanzania && go test -count=1 ./...
cd adapters/ghana    && go test -count=1 ./...
cd adapters/nigeria  && go test -count=1 ./...
cd adapters/south_africa && go test -count=1 ./...
cd adapters/registry && go test -count=1 ./...
```

## Reporting security issues

See [SECURITY.md](./SECURITY.md). Do NOT open a public issue for security vulnerabilities.

## Code of Conduct

See [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md). Be kind. This is civic infrastructure.
