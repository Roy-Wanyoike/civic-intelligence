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
