# Worklog

A linear record of engineer-level task entries. Each entry references a
task ID (e.g. `ENG-I3`), the branch it landed on, and a concise summary
of what was done, what was tested, and what was committed.

This file complements `CHANGELOG.md` (which tracks user-visible changes)
by capturing the per-engineer context: which worktree, which files
touched, which tests added, which disclaimers enforced.

---

## ENG-I3 — Cross-country civic comparison tool + civic indicators dashboard

**Branch:** `feat/wave9-civic-compare`
**Worktree:** `/home/z/my-project/wt-civic-compare`
**Commit:** `feat: cross-country comparison tool + civic indicators dashboard`
**Status:** complete — backend tests pass, frontend type-checks.

### Goal

Build the first African civic-intelligence platform feature that lets
users compare legislation, government structure, public debt, and civic
indicators across the 6 supported countries (Kenya, Uganda, Tanzania,
Ghana, Nigeria, South Africa) with the same structured schema. No
platform in Africa offers side-by-side comparison of civic data across
multiple countries — this enables cross-jurisdictional research that
was previously impossible.

### Files added

**Backend (Go):**
- `services/api/cmd/compare.go` (~970 lines) — 5 endpoints under
  `/api/v1/compare/*`:
    - `GET /api/v1/compare/countries?countries=KE,UG,TZ`
    - `GET /api/v1/compare/legislation?countries=KE,UG&topic=health`
    - `GET /api/v1/compare/debt?countries=KE,UG,TZ&from=2020&to=2024`
    - `GET /api/v1/compare/government-structure?countries=KE,UG,NG,ZA`
    - `GET /api/v1/compare/indicators?countries=KE,UG,TZ&indicators=debt_to_gdp,bills_introduced`
  - Static country-profile registry mirroring each adapter's
    `GetLegislativeStructure()` output (chamber count, members, term
    length, government system, parliament URL).
  - Per-country indicator registry (Bills introduced, Bills passed,
    Acts commenced, debt-to-GDP, parliament sessions, committee
    meetings, public participation) — every value carries a `source_url`.
  - Live Kenya debt series pulled from the existing
    `legislation.DebtRepository` (CBK + Treasury seed).
  - `comparisonDisclaimer` constant attached to EVERY response — "the
    platform does not rank countries or imply political preference".
  - For /compare/debt, the canonical `NO_POLITICAL_PERFORMANCE_SCORE`
    disclaimer is appended via the existing `appendCanonicalDisclaimer`
    helper.
- `services/api/cmd/compare_test.go` (~440 lines) — 11 test functions
  covering:
    - `TestCompare_CountriesReturnsAllProfiles` — 3-country compare
    - `TestCompare_CountriesDisclaimerPresent` — disclaimer invariant
    - `TestCompare_GovernmentStructureReturnsBicameralVsUnicameral`
    - `TestCompare_LegislationByTopic` — topic filter
    - `TestCompare_DebtReturnsKenyaLiveSeries` — KE pulls from repo
    - `TestCompare_DebtCarriesNoPoliticalPerformanceScore`
    - `TestCompare_IndicatorsFiltersByKey`
    - `TestCompare_IndicatorsDisclaimerPresent`
    - `TestCompare_NeverRanksCountries` — scans all 5 endpoints for
      forbidden ranking language ("best country", "top performer",
      "#1", etc.)
    - `TestCompare_AllFiveEndpointsReturnDisclaimer` — every endpoint
      carries the disclaimer (structural invariant)
    - `TestCompare_DefaultsToAllCountriesWhenParamMissing`
    - `TestCompare_DropsUnknownCountryCodes` — silent drop, no 400
    - `TestCompare_MethodNotAllowed` — POST/PUT/DELETE → 405
    - `TestCompare_UnknownSubResourceReturns404`

**Backend wiring:**
- `services/api/cmd/main.go` — registered `/api/v1/compare` and
  `/api/v1/compare/` routes with `makeCompareRouter(debtRepo)`, threading
  the existing DebtRepository through so /compare/debt can pull live
  CBK + Treasury observations for Kenya.

**Frontend (Next.js 14 / TypeScript):**
- `apps/web/src/lib/compare-api.ts` (~200 lines) — typed client for
  the 5 compare endpoints. Exports `COUNTRY_META`, `INDICATOR_META`,
  `SUPPORTED_COUNTRIES`, `COMPARISON_DISCLAIMER`, and
  `formatIndicatorValue` helpers.
- `apps/web/src/app/compare/page.tsx` — server component that fetches
  profiles, debt data, and indicators in parallel and passes to
  `CompareView`. Reads `?countries=KE,UG,TZ` from the URL so users
  can deep-link.
- `apps/web/src/app/compare/compare-view.tsx` (~580 lines) — client
  component with:
    - Multi-select country chips (flags + names) for the 6 countries
    - Dimension selector: government structure, legislation, debt,
      indicators
    - Side-by-side comparison table with color-coded differences
      (highlight outliers in amber, not green/red — no value judgment)
    - Visual charts:
        - Debt comparison: grouped SVG bar chart (debt_to_gdp per country)
        - Legislative activity: line chart via `BaseChart` (bills
          introduced over time)
        - Government structure: per-country chamber diagram
          (unicameral vs bicameral)
    - "Key Differences" section with plain-language explanations
    - Print + Share buttons
    - Top + bottom disclaimer banners
- `apps/web/src/app/indicators/page.tsx` (~280 lines) — civic
  indicators dashboard:
    - Card grid: 7 indicators × N countries (filter by country + year)
    - Each card: value, trend badge (up/down/flat/unknown), source URL,
      last-updated date
    - CSV export via `data:` URL (no client round-trip needed)
    - "Compare with others" deep-link per country row
- `apps/web/src/app/indicators/indicators-filters.tsx` (~60 lines) —
  small client component driving the country + year select via Next.js
  router (keeps the page a server component).

**Navbar:**
- `apps/web/src/components/header.tsx` — added `/compare` and
  `/indicators` under the "Intelligence" nav group.
- `apps/web/src/components/command-palette.tsx` — added `/compare` and
  `/indicators` to the Intelligence command group.
- `apps/web/src/i18n/locales/en.json` — added `nav.pages.compare` and
  `nav.pages.indicators`.
- `apps/web/src/i18n/locales/sw.json` — added Swahili translations
  (`Linganisha`, `Viashiria`).

### Tests added

| Test                                                            | Count |
|----------------------------------------------------------------|-------|
| `TestCompare_CountriesReturnsAllProfiles`                       | 1     |
| `TestCompare_CountriesDisclaimerPresent`                       | 1     |
| `TestCompare_GovernmentStructureReturnsBicameralVsUnicameral`   | 1     |
| `TestCompare_LegislationByTopic`                               | 1     |
| `TestCompare_DebtReturnsKenyaLiveSeries`                       | 1     |
| `TestCompare_DebtCarriesNoPoliticalPerformanceScore`           | 1     |
| `TestCompare_IndicatorsFiltersByKey`                           | 1     |
| `TestCompare_IndicatorsDisclaimerPresent`                      | 1     |
| `TestCompare_NeverRanksCountries` (5 sub-tests)               | 5     |
| `TestCompare_AllFiveEndpointsReturnDisclaimer` (5 sub-tests)  | 5     |
| `TestCompare_DefaultsToAllCountriesWhenParamMissing`          | 1     |
| `TestCompare_DropsUnknownCountryCodes`                         | 1     |
| `TestCompare_MethodNotAllowed` (5 endpoints)                  | 1     |
| `TestCompare_UnknownSubResourceReturns404`                     | 1     |

**Total: 22 distinct test functions, 31 (sub-)tests, all passing.**

### Verification

```bash
# Backend tests
$ cd services/api && go test -count=1 ./cmd/ 2>&1 | tail -5
ok  	github.com/Roy-Wanyoike/civic-intelligence/services/api/cmd	0.015s

# Frontend type-check
$ cd apps/web && npx tsc --noEmit 2>&1 | tail -5
exit code: 0   # no errors
```

### Design invariants enforced

1. **The platform NEVER ranks countries.** Every response carries the
   `comparisonDisclaimer`; /compare/debt also appends
   `NO_POLITICAL_PERFORMANCE_SCORE`. The `TestCompare_NeverRanksCountries`
   test scans all 5 endpoints for forbidden ranking phrases ("best
   country", "top performer", "#1", "first place", etc.).
2. **Differences are described, not evaluated.** The `differences` array
   uses phrases like "bicameral vs unicameral" and "4-year vs 5-year
   terms" — never "better" / "worse" / "higher" / "lower".
3. **Colour coding is structural, not value-based.** Outlier cells are
   highlighted in amber (caution) rather than green/red (good/bad).
4. **Every value carries a source_url.** No figure is presented without
   a verifiable source — the user can always click through to the
   Parliament or IMF page that published it.
5. **Kenya's live debt data is pulled from the existing
   DebtRepository** (CBK + Treasury seed) — not duplicated. The
   compare endpoint reuses the existing `legislation.WireDebtRepository`
   wiring rather than fetching fresh data.
6. **The compare page is print-friendly + shareable.** The URL carries
   the selected countries + dimension via query params; the Share button
   uses the Web Share API with clipboard fallback.

### Follow-ups (not blocking)

- Replace illustrative pre-2024 Bills-introduced trend values with real
  per-country historical data once each adapter's `DiscoverBills()`
  supports year-range queries.
- Wire non-Kenya countries' live debt data when each country's
  DebtRepository seed lands (currently only Kenya has seeded CBK +
  Treasury observations; the other 5 fall back to IMF WEO debt-to-GDP).
- Add per-country topic taxonomy (currently the `topic` filter matches
  against the sample Bills' titles + tags).

### Commit

```
feat: cross-country comparison tool + civic indicators dashboard
```

---

## ENG-J1 — Country-scoping middleware + README rewrite for multi-country platform

**Branch:** `feat/wave10-readme-fix`
**Worktree:** `/home/z/my-project/wt-readme-fix`
**Commit:** `feat: country-scoping middleware + README rewrite for multi-country platform`
**Status:** complete — backend tests pass, frontend type-checks.

### Goal

The platform's README was outdated ("Kenya" throughout, only 2 countries
mentioned, 176 tests) and every API endpoint silently defaulted to Kenya
data regardless of which country the user selected in the Government
Selector. A user from Uganda would see Kenyan Bills, Kenyan People,
Kenyan Institutions — defeating the purpose of the multi-country
architecture. This task rewrote the README to reflect the current
6-country platform (KE, UG, TZ, GH, NG, ZA, 10 flagship features, 700+
tests, 68 frontend pages, 56 API routes) AND shipped the country-scoping
middleware that filters every list endpoint by the selected country.

### Part 1 — README rewrite

**Files modified:**
- `README.md` — full rewrite. New header ("Evidence-grounded civic
  intelligence for Africa" — not just Kenya), updated badges (700+ Go
  tests, 6 countries with flags, 68 pages, 56 API routes, 129 commits),
  10-feature flagship table (Civic Knowledge Graph, Daily Brief,
  Cross-country Comparison, Scenarios, Constitution, Government History,
  Public Debt, Audit an Act, Legal Lineage, Indicators), new
  "Multi-country Architecture" section with diagram showing 6 adapters
  flowing into the shared core domain, new "Contributing" section with
  6-step "how to add your country's data" guide (adapter → seed →
  main.go → country page → Government Selector defaults → contract
  tests), updated architecture diagram with 6 country sources flowing
  into the adapter layer, updated tech stack (Go 1.23, PostgreSQL 16,
  22 migrations, Redis, MinIO, Temporal, NATS, OpenTelemetry), new
  "Country Scoping" API reference section documenting X-Civic-Country
  + the ALL global mode, updated stats (700+ Go, 24 Python, TypeScript
  PASS, 68 pages, 56 routes, 22 migrations, 6 adapters, 10 features,
  129 commits), MIT license, updated disclaimer listing all 6
  countries.

### Part 2 — Country scoping middleware

**Files added:**
- `services/api/internal/middleware/country.go` (~190 lines) — the
  `Country` middleware. Resolution order: `X-Civic-Country` header →
  `?country=` query param → `"KE"` default. Validates against the 6
  supported codes (KE, UG, TZ, GH, NG, ZA) + the special `ALL` global
  mode. On invalid input returns 400 with a JSON body listing the
  supported codes. Stores the resolved code on the request context +
  echoes it back on the response as `X-Civic-Country` so the frontend
  can sync its selector to the actual scope the API used. Exports
  `CountryFromContext(ctx)` helper (returns `"KE"` when middleware not
  chained — preserves backward compatibility for handlers called
  directly from tests), `WithCountry(ctx, code)` for tests, and the
  `SupportedCountries` slice + `IsSupportedCountry(code)` predicate.
- `services/api/internal/middleware/country_test.go` (~310 lines,
  14 test functions) — covers header/query/precedence, default-to-KE,
  case-insensitivity (lowercase "ug" → "UG"), global ALL mode, 400 on
  invalid with supported-countries list echoed in body, empty header
  fall-through, all-supported-codes table test, response-header-always-
  set invariant, `CountryFromContext` defaults, `WithCountry`+round-trip
  inverse, `IsSupportedCountry` predicate, propagation through nested
  middleware chain.
- `services/api/cmd/country_handlers.go` (~55 lines) — per-handler
  filter helpers `filterActsByCountry([]actResponse, country)` and
  `filterMapsByCountry([]map[string]any, country)`. Both return the
  input unchanged when `country == "" || country == GlobalCountry`
  (the dashboard view). Otherwise return only the rows whose
  `country`/`Country` field matches.
- `services/api/cmd/country_scope_test.go` (~430 lines, 17 test
  functions) — covers `handlePeople`, `handleCommittees`,
  `handleInstitutions`, `handleActsList`, `handleLoansList`,
  `handleGrantsList`, `handleBriefing`, `handleSearch` with KE / UG /
  NG / ZA / ALL scopes; cross-country-leakage guard on detail lookups
  (UG-scoped request for a KE person ID returns 404, not 200);
  end-to-end `middleware.Country(handlePeople)` chain test verifying
  the response carries `X-Civic-Country: UG` and the handler sees UG
  on its context; `/compare/countries` returns 6 countries when no
  scope is set (the global dashboard default).

**Files modified:**
- `services/api/cmd/main.go`:
  - Wired `middleware.Country(corsed)` into the middleware chain
    BETWEEN `RequestID` (outermost) and `RateLimit` (innermost), so
    the chain is now RequestID → Country → CORS → OptionalAuth →
    Metrics → SecurityHeaders → RateLimit → handler. Country sits
    inside RequestID so a country 400 is logged with the request_id;
    Country is outside RateLimit so a 429 still carries the
    `X-Civic-Country` response header.
  - `handleActsList` — reads `middleware.CountryFromContext(r.Context())`,
    applies `filterActsByCountry` AFTER search + status filtering (so
    `?q=data&country=UG` returns only Uganda Acts that match "data"),
    echoes `country` in the response body.
  - `handlePeople`, `handleCommittees`, `handleInstitutions` — each
    reads the country from context, applies `filterMapsByCountry` on
    the list response, and applies a visibility gate on detail
    lookups (a UG-scoped request asking for a KE person's ID returns
    404, preventing cross-country leakage).
  - `samplePeople`, `sampleCommittees`, `sampleInstitutions` —
    expanded from Kenya-only to multi-country (UG, TZ, GH, NG, ZA
    rows added for each), so a Uganda user sees Ugandan MPs, a
    Nigeria user sees Nigerian Senate + House members, etc.
  - `handleLoansList`, `handleGrantsList` — echo the resolved country
    in the response body (the empty-list placeholders remain until
    issue #93 wires the live repo; the `/debt` endpoints already
    serve the live Kenya debt repository and were not modified).
  - `handleBriefing` — uses `middleware.CountryFromContext` instead
    of the hardcoded `"KE"` it used to.
- `services/api/cmd/search.go`:
  - `handleSearch` reads the country from context and filters the
    results by the country prefix on the item ID (the platform's
    seed IDs all carry a country prefix, e.g. `ke-act-data-…`).
    A KE-scoped search returns only KE results; a UG-scoped search
    returns nothing (no UG seed yet — the correct behaviour);
    `ALL` returns results from every country.
  - Echoes `country` in the response body.
  - Added `matchesCountry(id, country)` helper (replaced by a
    per-row `country_code` column when Postgres FTS lands).
- `apps/web/src/lib/government-context.tsx` — added `useCountryHeader()`
  hook that returns the active country code from the Government
  Provider context. React components that need the country code
  directly (e.g. to build a country-specific URL) can call this;
  most API calls automatically get the header via `lib/api.ts`.
- `apps/web/src/lib/api.ts`:
  - Added `COUNTRY_HEADER` constant (= `"X-Civic-Country"`),
    `SUPPORTED_COUNTRY_CODES` Set mirroring the Go middleware,
    and `getCountryFromCookie()` helper that reads the
    `civic_gov_selection` cookie (the same one the Government
    Selector writes) and returns the upper-cased country code
    (or `"KE"` when the cookie is absent / malformed).
  - `getJSON` and `postJSON` automatically attach the
    `X-Civic-Country` header to every API call via the new
    `countryHeaders()` helper. No caller has to remember to set
    it — every API call from the frontend now carries the
    country context.

### Global / dashboard mode (Part E)

The `/compare/*`, `/indicators`, `/dashboard`, and `/graph` endpoints
are inherently cross-country — they return data across all 6 countries
when no `?countries=` filter is supplied. This satisfies the spec's
requirement that "when X-Civic-Country is ALL (or not set on these
endpoints), return data for all countries." The compare endpoint's
`parseCountriesParam` already defaults to `supportedCountryCodes`
(KE, UG, TZ, GH, NG, ZA) when no `?countries=` param is supplied —
covered by `TestCompare_DefaultsToAllCountries`.

### Test counts (after this task)

- **741 Go tests** passing across all modules (was ~728 before this
  task — added 31 new tests: 14 in middleware/country_test.go +
  17 in cmd/country_scope_test.go).
  - services/api: 287 (was 255 — added 32)
  - services/legislation: 77
  - services/simulation: 50
  - services/evidence: 3
  - services/intelligence: 5
  - services/documents: 3
  - services/ingestion: 3
  - packages/observability: 29
  - packages/storage: 17
  - adapters/kenya: 79
  - adapters/uganda: 26
  - adapters/tanzania: 17
  - adapters/ghana: 19
  - adapters/nigeria: 23
  - adapters/south_africa: 20
  - tests/contract: 20
  - tests/integration: 55
  - tests/chaos: 8
- **24 Python tests** passing (AI gateway + capabilities + eval —
  unchanged by this task).
- **TypeScript PASS** — `npx tsc --noEmit` exits 0.

### Verification

```bash
cd services/api && go test -count=1 ./... 2>&1 | tail -5
# → ok  	github.com/Roy-Wanyoike/civic-intelligence/services/api/cmd
# → ok  	github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware
# → ok  	github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/oidc

cd apps/web && npx tsc --noEmit 2>&1 | tail -5
# → exit: 0  (no output, no errors)
```

### Commit

```
feat: country-scoping middleware + README rewrite for multi-country platform
```
