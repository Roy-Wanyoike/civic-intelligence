# Changelog

All notable changes to the Civic Intelligence Platform are documented here.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] — 2026-09-21

Patch release fixing the Vercel AI service deploy error, adding
comprehensive SEO/metadata, and refreshing the README to reflect the
v0.2.0 state.

### Fixed — Vercel AI service entrypoint (PR #258)

Vercel multi-service build failed with:

  Error: Service "ai" detected framework "fastapi" in "services/ai"
  and must specify an "entrypoint" for runtime "python".

Fix: added `"entrypoint": "main.py"` to the AI service config in
`vercel.json`. The existing `services/ai/main.py` already exports
the FastAPI ASGI app via `from app.main import app` — Vercel's Python
runtime auto-detects the `app` variable at module scope.

Also added `services/ai/runtime.txt` pinning Python 3.12 so Vercel's
runtime matches the local dev virtualenv and the docker-compose env.

### Added — SEO + metadata improvements (PR #259)

- `/robots.txt` — explicit allow for GPTBot, ClaudeBot, Google-Extended
  (so AI crawlers can ground their answers with the platform's content);
  disallow for `/api/`, `/auth/`, `/offline`, `/report`.
- `/sitemap.xml` — 50 URLs across primary, secondary, utility, country,
  and per-record index pages. Each URL carries priority + changeFrequency.
- `/icon` (favicon, 32×32 PNG) — generated from `icon.tsx` via next/og.
- `/apple-icon` (Apple Touch Icon, 180×180) — generated from `apple-icon.tsx`.
- `/opengraph-image` (1200×630) — generated from `opengraph-image.tsx`.
- `/twitter-image` (1200×630) — generated from `twitter-image.tsx`.
- JSON-LD structured data — `PlatformJsonLd` component emitting a `@graph`
  with WebSite (SearchAction sitelinks), Organization, WebApplication
  (applicationCategory=GovernmentApplication, 10-item featureList).
- Expanded root metadata: 24 keywords covering all 14 country parliaments,
  OG + Twitter cards, robots directives, manifest, icons.
- Per-page metadata added to 8 pages that were missing it (homepage,
  /report, 6 scenario sub-pages). The /report page + scenario pages are
  `noindex` (user-generated, not durable content for indexing).
- `NEXT_PUBLIC_SITE_URL` env var support so OG image URLs + canonical
  URLs point at the right host in production.

### Updated — README for v0.2.0 (PR #260)

Comprehensive README refresh: country count 6 → 14, frontend pages 68 → 73,
API routes 56 → 70, Go tests 700+ → 1003, Python tests 24 → 28, commits
129 → 164. All 14 country adapters listed in both the adapter architecture
section AND the Countries section. Deploy instructions now mention
NEXT_PUBLIC_SITE_URL, AI_SERVICE_URL, API_SERVICE_URL env vars + the daily
cron at 03:00 UTC. Roadmap extended with Phases 13 + 14 as Planned.

## [0.2.0] — 2026-09-21

The Wave 12/13 release — adds 8 new country adapters (Rwanda, Zambia,
Senegal, Egypt, Morocco, DR Congo, Ethiopia, Malawi), bringing the
platform from 6 to 14 supported African countries. Also implements the
5 long-standing Kenya crawler TODOs and fixes the Vercel Hobby cron
limitation.

### Migration notes

- **Vercel Hobby plan users**: pull the latest `vercel.json` — the cron
  schedule changed from `0 * * * *` (hourly, rejected by Vercel Hobby)
  to `0 3 * * *` (daily at 03:00 UTC).
- **API clients reading `supported_countries`** from the 400-error body:
  the list now contains 14 unique codes (was 18 with duplicates in the
  broken wave-13 release). Each code now appears exactly once.
- **Frontend Government Selector**: now offers all 14 countries. Users
  who previously selected Rwanda/Zambia/Senegal/Egypt/Morocco/DR Congo/
  Ethiopia/Malawi in the dropdown will see the correct country's data
  (was silently downgraded to Kenya).

### Fixed — Vercel Hobby cron schedule (PR #254)

Vercel Hobby accounts only allow cron jobs that run at most once per day.
The previous schedule `0 * * * *` (every hour at minute 0) was rejected
on deploy. Changed to `0 3 * * *` — runs once a day at 03:00 UTC.

### Added — Kenya live crawlers (PR #255)

Implemented the five long-standing `TODO: crawl` markers in the Kenya
adapter package. Each `Discover*` method previously returned an empty
slice; they now perform real HTTP crawls against verified URLs on
parliament.go.ke and new.kenyalaw.org.

The crawlers cover all 12 official Kenyan sources:

| Source | URL |
|--------|-----|
| NA Bills | `parliament.go.ke/the-national-assembly/house-business/bills` |
| Senate Bills | `parliament.go.ke/the-senate/senate-bills` |
| NA Bill Tracker | `parliament.go.ke/the-national-assembly/house-business/bill-tracker` |
| NA Hansard | `parliament.go.ke/the-national-assembly/house-business/hansard` |
| Senate Hansard | `parliament.go.ke/the-senate/Hansard` (capital H) |
| NA Order Paper | `parliament.go.ke/the-national-assembly/house-business/order-paper` |
| Senate Order Paper | `parliament.go.ke/the-senate/house-business/order-paper` |
| NA Votes & Proceedings | `parliament.go.ke/the-national-assembly/house-business/votes-proceeding` (singular) |
| Senate Votes & Proceedings | `parliament.go.ke/the-senate/house-business/votes-proceeding` |
| NA Committees | `parliament.go.ke/the-national-assembly/committees` |
| Senate Committees | `parliament.go.ke/the-senate/committees/senate-committees` |
| Kenya Gazette | `new.kenyalaw.org/kenya_law/gazette/` |

All crawls go through the polite `PoliteClient` (1 request/sec/host).
Each candidate carries a deterministic source ID derived from house +
sitting date (or URL hash fallback) so downstream consumers can dedupe
across discovery runs. PDF text extraction + per-notice parsing remain
delegated to the documents service.

The PR ships 20 new tests + 6 HTML fixtures mirroring the actual site
structure. Test count went from 983 → 1003.

### Fixed — Wave 13 audit (PRs #250, #251, #252)

Wave 13 (commit `d4dd0ef`, 2026) shipped four new country adapters — Morocco,
DR Congo, Ethiopia, Malawi — but the integration was incomplete in several
places. A post-merge audit (PR #250) found and fixed five independent bugs:

- **`SupportedCountries` had duplicates** (services/api/internal/middleware/country.go).
  The wave-12 codes (RW, ZM, SN, EG) were accidentally re-appended after the
  wave-13 codes, producing a list of 18 entries with 4 duplicates. Every
  400-error body sent to API clients listed each wave-12 code twice. Fixed by
  deduping to the canonical 14-entry list; new `TestCountry_SupportedCountriesNoDuplicates`
  guards against this regression.
- **Wave-13 adapters were never registered in the runtime** (adapters/registry/wrappers.go).
  `MustRegisterDefault` only registered 10 adapters. Morocco/DR Congo/Ethiopia/Malawi
  had working code but were unreachable: `registry.GetAdapter("MA")` returned
  `ErrUnsupportedCountry`, and `GET /api/v1/countries` omitted them. Fixed by
  adding the four wrappers + constructors + registration entries.
- **`SUPPORTED_COUNTRY_CODES` (frontend)** (apps/web/src/lib/api.ts) was
  missing the 8 newer codes. The cookie-validation logic silently downgraded
  any of those selections back to `KE` — a user picking Rwanda in the
  dropdown would still see Kenya data. Fixed by adding all 8 missing codes.
- **Next.js production build failed** (apps/web):
  - `apps/web/src/app/people/[id]/scorecard/page.tsx`,
    `apps/web/src/app/constituencies/[id]/page.tsx`: both exported both
    `metadata` and `generateMetadata` — Next.js 14 forbids exporting both.
    Merged the static description into the dynamic `generateMetadata`.
  - `/offline` page was a Server Component rendering a `<button onClick>` —
    event handlers cannot cross the server/client boundary, so static page
    generation for `/offline` timed out (3 retries × 60s, then the whole
    build failed). Extracted the button into a new `'use client'` island
    component (`apps/web/src/app/offline/retry-button.tsx`).
  - 6 ESLint fail-the-build errors: unused imports (`EyeOff`, `Search`,
    `Share2`, `GraphResponse`, `_year`); unescaped apostrophe in calendar
    caption; `let` → `const` for non-reassigned bindings; missing
    `aria-selected` on `role="option"`.
- **`packages/cache/go.mod`** pinned `gopher-json` to pseudo-version
  `v0.0.0-20230918194607-3b0d2c1b556d` referencing a non-existent commit,
  breaking `go mod download` for the whole cache module. Re-pinned to
  `v0.0.0-20230218143504-906a9b012302` (the real HEAD commit + timestamp).

PR #251 removed stale `FIXME: verify with go build when Go available` markers
that were added in earlier sessions when the Go toolchain was unavailable.
Go 1.23.4 is now available; every module builds; the full test suite passes.

PR #252 removed two unused npm dependencies from `apps/web`:
- `date-fns` — every date format uses `Intl.DateTimeFormat`; no file imports it.
- `class-variance-authority` — the project does not use shadcn/ui variants; no
  file imports `cva`.

### Added — Wave 12 (commit `2065bfe`, 2026) — Rwanda, Zambia, Senegal, Egypt

Four new country adapters, bringing the platform from 6 to 10 supported
countries:

- **Rwanda (RW)** — Parliament of Rwanda (bicameral: Chamber of Deputies +
  Senate). Source: parliament.gov.rw.
- **Zambia (ZM)** — National Assembly of Zambia (unicameral). Source:
  parliament.gov.zm.
- **Senegal (SN)** — Assemblée Nationale du Sénégal (unicameral). Source:
  assemblee-nationale.sn.
- **Egypt (EG)** — Egyptian Parliament (bicameral: House of Representatives +
  Senate). Source: parliament.eg.

Each adapter ships its full country data (legislative structure, stages,
glossary, sample Bills, official sources) in `adapters/{country}/internal/`,
with a parser that handles the parliament's actual HTML structure
(`testdata/bills.html`). Contract tests verify each adapter's isolation
(its `DiscoverBills` never returns foreign Bills).

### Added — Wave 13 (commit `d4dd0ef`, 2026) — Morocco, DR Congo, Ethiopia, Malawi

Four new country adapters, bringing the platform from 10 to 14 supported
countries:

- **Morocco (MA)** — Parliament of Morocco (bicameral: House of Representatives +
  House of Councillors). Source: parlement.ma.
- **DR Congo (CD)** — Parliament of the Democratic Republic of the Congo
  (bicameral: National Assembly + Senate). Sources: assemblee-nationale.cd,
  senat.cd.
- **Ethiopia (ET)** — Federal Parliamentary Assembly of Ethiopia (bicameral:
  House of Peoples' Representatives + House of Federation). Source:
  parliament.gov.et.
- **Malawi (MW)** — National Assembly of Malawi (unicameral). Source:
  parliament.gov.mw.

Each adapter ships its full country data and parser, matching the wave-12
adapter shape. The platform now covers 14 African countries.

### Added — Civic Daily Brief + AI Summary (task ENG-I2, branch `feat/wave9-civic-brief`)

Personalised, AI-grounded, plain-language daily summary of civic developments
— the first surface on the platform that tells each citizen what matters to
*them* based on their followed topics, institutions, and Bills, with every
claim backed by a primary-source evidence URL.

**Backend — `services/api/cmd/brief.go`** (new file, ~620 LOC):
- `POST /api/v1/brief/generate` — generate + persist a personalised brief
- `GET  /api/v1/brief/today`     — today's brief (on-demand if missing)
- `GET  /api/v1/brief/archive`   — list previously generated briefs
- `GET  /api/v1/brief/{id}`      — single brief by ID
- Brief shape: `{ id, date, headline, sections[], ai_summary, ai_disclaimer,
  ai_source, evidence_count, generated_at, country, user_id }`
- Four sections: "What Changed Today", "Your Followed Topics", "What to
  Watch", "Constitutional Context" — every item carries an `evidence_url`.
- AI summary: calls the Python AI service `/v1/briefing/generate` (existing
  capability `BriefingGeneratorCapability`). When the AI service is
  unreachable, falls back to a deterministic template summary explicitly
  labelled "Auto-generated summary." — NEVER presented as AI output (the
  no-fake-AI rule from `docs/NO_FAKE_COMPLETION.md`). `ai_source` field
  distinguishes `"ai-service"` from `"template-fallback"`.
- `ai_disclaimer` constant: `"AI-generated summary. Verify against primary
  sources."` — surfaced verbatim on every brief + rendered next to the
  ASSUMPTION reality badge on the frontend.
- Filtering: followed_topics matched against Bill titles (case-insensitive
  substring); followed_institutions against house + title; followed_bills
  against source ID. An empty follow set returns ALL Bills (the brief is
  still useful for a citizen who follows nothing yet).
- Constitutional Context section: curated mapping of topics → Constitution
  of Kenya 2010 articles (Article 10 default; Article 43 health/education/
  housing/water; Article 201 public finance; Article 31 data protection;
  Article 42 environment; Article 60 land; Article 53 children; Article 41
  labour; Article 238 security). Every article links to kenyalaw.org.
- In-memory `briefStore` (sync.RWMutex + `map[string]brief`). The /today
  endpoint caches the on-demand brief; subsequent calls return the cached
  copy (same `generated_at` timestamp).
- When the adapter fails (HTTP 500 / network error), the handler returns
  200 with an empty Bill list + the template fallback summary — the brief
  surfaces the failure honestly rather than returning 5xx.

**Backend — `services/api/cmd/main.go`**: registered four new routes on the
existing `apiHandler` mux. The legacy `/api/v1/briefing` endpoint is kept for
backward compat with `api.ts:getBriefing` (the old empty-placeholder handler).

**Backend — `services/api/cmd/brief_test.go`** (new file, 26 tests):
- `TestGenerateBrief_BasicShape` — id, date, headline, 4 sections, AI disclaimer.
- `TestGenerateBrief_EveryItemHasEvidenceURL` — the brief's core contract:
  every item in every section carries a non-empty `evidence_url`.
- `TestGenerateBrief_AIDisclaimerPresent` — the exact disclaimer string.
- `TestGenerateBrief_TemplateFallbackNotLabelledAsAI` — template fallback
  is prefixed "Auto-generated summary." (the no-fake-AI gate).
- `TestGenerateBrief_EmptyBills` — honest empty state when no Bills found.
- `TestGenerateBrief_FollowedTopicsFilters` — followed_topics "health"
  surfaces the Public Health Bill.
- `TestGenerateBrief_FollowedTopicsNoMatch` — surfaces the templated
  "no matches" item (with its own evidence_url) when nothing matched.
- `TestGenerateBrief_ConstitutionalContextForHealth` — followed topic
  "health" → Article 43 surfaces with kenyalaw.org evidence_url.
- `TestGenerateBrief_HeadlineCounts` — headline counts Bills into
  published / amendment / finance buckets.
- `TestBriefGenerateHandler_Post`, `TestBriefGenerateHandler_GetRejectsMethod`,
  `TestBriefTodayHandler_OnDemand`, `TestBriefTodayHandler_AdapterError`,
  `TestBriefArchiveHandler_ListsStoredBriefs`, `TestBriefDetailHandler_Found`,
  `TestBriefDetailHandler_NotFound`, `TestBriefDetailHandler_RejectsReservedNames`,
  `TestBriefEndpoints_RegisteredOnMux` — full HTTP + routing coverage.
- `TestCallAIBriefingGenerator_Reachable` + `TestCallAIBriefingGenerator_Unreachable`
  + `TestGenerateAISummary_FallsBackToTemplate` — AI integration.
- `TestComposeAISummaryFromResponse_NeverEchoesAIHeadline` — the BFF composes
  its own summary from observed Bills; it never echoes the AI service's
  headline verbatim (single-source-of-truth rule).
- `TestBrief_JSONShape`, `TestBriefStore_PutGet`, `TestBriefArchiveEntry_JSONShape`,
  `TestBriefAISummaryDisclaimer_Constant` — contract + store + JSON shape.

**Frontend — `apps/web/src/app/briefing/page.tsx`** (redesigned server component):
- Hero: date + headline + meta (country, evidence citation count, generated_at).
- "3 things you need to know" — bulleted top-3 from "What Changed Today".
- "What Changed Today" timeline of changes (every item links to evidence).
- "Your Followed Topics" personalised section (empty-state copy when no follows).
- "What to Watch" upcoming / approaching-final Bills.
- "Constitutional Context" — relevant articles with article text + connection.
- "AI Summary" — clearly labelled with `RealityBadge kind="ASSUMPTION"` +
  the platform-wide `ai_disclaimer` string + (when fallback) an explicit
  "(AI service unavailable — this is an auto-generated summary, not a
  model output.)" caveat.
- "Evidence" footer — every source URL cited in the brief, deduped.
- Share button (Web Share API → clipboard fallback) + "Subscribe to daily
  email" button (placeholder → `/notifications`) + "Personalise" button
  (→ `/following`).
- Mobile-first: collapsible sections (tap to expand/collapse), horizontal
  swipe-between-sections via scroll-snap on the section-nav strip, 44px+
  touch targets, `print:` Tailwind variants for print-friendly output.
- Honest fallback: when the BFF is unreachable, the page renders an empty
  brief with an alert banner — it never fabricates a brief.

**Frontend — `apps/web/src/app/briefing/brief-reader.tsx`** (new client
component, ~330 LOC): owns the interactive bits — collapsible sections,
swipe nav, share button, evidence footer.

**Frontend — `apps/web/src/app/briefing/archive/page.tsx`** (new): archive
list of past briefs; each entry links to `/briefing?id={brief-id}` which
the brief page reads via `searchParams` and uses to fetch the specific
brief from `/api/v1/brief/{id}`.

**Frontend — `apps/web/src/lib/types.ts`**: added `BriefSectionItem`,
`BriefSection`, `CivicBrief`, `BriefArchiveEntry` types mirroring the Go
structs in `services/api/cmd/brief.go`.

**Frontend — `apps/web/src/lib/api.ts`**: added `generateBrief()`,
`getTodaysBrief()`, `listBriefArchive()`, `getBrief(id)` typed clients.
Kept the legacy `getBriefing()` for backward compat.

**Verification**:
- `cd services/api && go test -count=1 ./cmd/ 2>&1 | tail -5` → `ok … 0.015s`
  (170 tests passing, +26 new brief tests).
- `cd apps/web && ./node_modules/.bin/tsc --noEmit 2>&1 | tail -5` → clean.

### Added — Wave 5 Documentation Pass (this commit, branch `fix/wave5-docs-a`)

#### Wave 5 forensic re-audit (5 audit teams, spec §79)

Wave 5 performed the mandated Final Re-Audit cycle (spec §79) by dispatching
five parallel audit teams against HEAD `dca1c78` (commit
`fix(#212,#227): populate constitution chapters/articles + consolidate
BillStageTransitionValidator (#244)`). Each team audited a disjoint spec slice:

- `audit-team-1-report.md` — Spec §1-17 (Primary Objective → Core Civic Model). 62 unique gaps.
- `audit-team-2-report.md` — Spec §18-34 (Public Debt Attribution → Homepage/Carousel). 75 unique gaps.
- `audit-team-3-report.md` — Spec §35-52 (Visual Design → Observability). 69 unique gaps.
- `audit-team-4-report.md` — Spec §53-70 (Security → Git History). 68 unique gaps.
- `audit-team-5-report.md` — Spec §71-86 (Process · Completion Matrix · Final Gates). 37 unique gaps.
- **Aggregate: 311 unique audit findings across the 5 audit-team reports.**

The four P0 release blockers from audit-team-5:
1. **P0-1** — No golden user journey tests (spec §74; 0 of 8 mandated journeys implemented).
2. **P0-2** — No critical failure test (spec §75; `rg "critical.failure|CriticalFailure"` returns 0 hits in code).
3. **P0-3** — OIDC verifier does not verify JWT signatures
   (`services/api/internal/oidc/verifier.go:126` — `TODO(issue #59): verify the signature using the JWKS key matching header.Kid`).
4. **P0-4** — E2E test runner cannot execute (`@playwright/test` + `@axe-core/playwright` not in `apps/web/package.json` devDependencies).

#### Wave 5 documentation remediations (this commit, task ENG-E5)

This commit closes the four documentation gaps that triggered Wave 5:

- **#P1-1 / `MASTER_AUDIT.md` stale** — rewrote `MASTER_AUDIT.md` against
  HEAD `dca1c78` (was auditing Phase 1 commit `a0638e7`; 91 commits / 26
  issue-closing PRs merged since). The new audit groups post-Phase-1 work by
  feature area (Simulation, Constitution+Government, Post-Assent, Public Debt,
  Navbar, Bug fixes, Documentation) and cross-references all 5 audit-team
  reports as evidence. Each feature area lists what's IMPLEMENTED, what's
  IN_PROGRESS, what's BLOCKED. The Phase 1 baseline 22 findings are tracked
  through to their current disposition (11 fixed, 9 partial, 2 operational).
- **#P1-4 / §77 No-Fake-Completion rule not documented** — created
  `docs/NO_FAKE_COMPLETION.md`. Documents the rule (a feature is only DONE when
  all 14 criteria are met: Architecture, Domain model, Backend logic, Database
  schema, API, Frontend UX, Integrations, Evidence/provenance, Tests, QA
  verified, Security reviewed, Performance validated, Observability,
  Documentation). Lists forbidden patterns (TODO, placeholder, mock, stub,
  fake, hardcoded production value, commented-out implementation). Provides an
  engineer sign-off checklist for PR descriptions.
- **#§78 / Completion matrix missing** — created `docs/COMPLETION_MATRIX.md`.
  Master matrix with all 17 spec-mandated columns (Feature | Requirement |
  Domain | Implementation | API | Database | UI | Integration | Tests | QA |
  Security | Performance | Observability | Documentation | Status | Evidence |
  Issue | PR). Covers 45+ feature rows: Bill discovery, timeline, versions,
  AI summary, Constitution, Government, Presidential terms, Acts, Post-assent
  audit, Follow-a-Law, lineage, Public Debt, Loans, Grants, attribution,
  Trending, Terminology, Search, Civic Feed, Briefing, Notifications, Trust,
  Corrections, Scenarios, Citation validation, Contradiction engine, 6 country
  adapters, OIDC, RBAC, rate limiting, observability (metrics + tracing +
  Grafana), payments (M-Pesa + Stripe), PDF/DOCX extraction, Postgres
  migrations, OpenAPI, ADRs, Golden user journeys, Critical failure test,
  Integration tests, Contract tests, E2E tests, Performance tests, Chaos
  tests, Disaster recovery tests, AI evaluation dataset. Only `VERIFIED`
  counts as complete (§78). **Zero rows are at `VERIFIED`** — the platform is
  not complete from the audit's perspective.
- **#GAP-68-1 / Production gate checklist missing** — created
  `docs/PRODUCTION_GATE.md`. Checklist with all 16 gates from §81 (BUILD,
  UNIT TESTS, INTEGRATION TESTS, CONTRACT TESTS, E2E TESTS, AI EVALUATION,
  DATA QUALITY, SECURITY, ACCESSIBILITY, PERFORMANCE, CHAOS, DISASTER
  RECOVERY, OBSERVABILITY, DOCUMENTATION, UX REVIEW, PRODUCTION WORKFLOW).
  For each gate: pass/fail/partial status, evidence cited, blocker reference.
  **Aggregate: 1 PASS (DOCUMENTATION) · 4 PARTIAL · 11 FAIL · 0 UNVERIFIED.**
  The audit must continue (spec §81).

#### Wave 5 engineering remediations (other commits this session, separate worktrees)

The following Wave 5 fixes were implemented in parallel worktrees
(`wt-backend-a`, `wt-backend-b`, `wt-frontend-a`, `wt-frontend-b`, `wt-tests-a`)
and are recorded in `worklog.md` entries ENG-A1 through ENG-D2:

- **#201** (PR #233) — Parliament `DiscoverBills` infinite recursion (`fetchURL` delegating back to `Fetch`).
- **#204** (PR #232) — `next build` failed on ESLint unused-import errors.
- **#207** (PR #230) — `TestRunService_Run_RejectsNonReadyScenario` was tautological (seeded a DRAFT scenario in a different repo than `rs` used).
- **#211, #214, #228** (PR #235) — Postgres migrations 019 (simulation), 020 (government), 021 (post-assent) — 25 new tables across 3 schemas, with immutability triggers, EXCLUDE USING gist temporal validity, and btree_gist extension.
- **#213, #219** (PR #229) — `governments.go` and `public_debt.go` header comments listed unregistered endpoints.
- **#215, #216, #217** (PR #243) — Post-assent endpoints now use domain logic (`AuditForAct`), real persisted subscriptions (Follow-a-Law creates a real `subscription_id`), and data-driven lineage (missing steps reported as `NOT_VERIFIED`, never inferred).
- **#218** (PR #231) — `/acts/page.tsx` fetches from API instead of hardcoded array.
- **#220, #221** (PR #239) — `/debt/loans` wired to repository + `ValidateAttribution` enforced.
- **#222** (PR #240) — `GovernmentDebtSummary` carries the `NO_POLITICAL_PERFORMANCE_SCORE` canonical disclaimer.
- **#202** (PR #236) — `ActRepository` in-memory implementation with seed + API wiring.
- **#203** (PR #237) — `DebtRepository` in-memory implementation with seed + attribution validation.
- **#209, #210** (PR #240) — Golden dataset (11 categories) + constraint evaluation grammar.
- **#212, #227** (PR #244) — Constitution chapters/articles populated (20 curated articles / 9 chapters from Kenya Law); `BillStageTransitionValidator` consolidated to a single canonical `StageGraphValidator` implementation (`BillStateMachine` is now a thin wrapper).
- **#223, #224, #225, #226** (PR #242) — CHANGELOG, API README, OpenAPI spec, developers page refreshed.

### Added — Post-Phase-1 (PRs #196–#241)

#### Phase 18 — Simulation Infrastructure

- **Scenarios domain** (`services/simulation/`): country-agnostic scenario model
  (`Scenario`, `ScenarioAssumption`, `ScenarioModel`, `ScenarioConstraint`,
  `ScenarioVariable`) with `RealityLayer` tagging (FACT / OBSERVED / HYPOTHETICAL /
  MODELED / UNKNOWN). Every response from the simulation API carries an explicit
  reality-layer tag so HYPOTHETICAL outputs cannot be confused with observed
  civic facts.
- **Simulation engines**: deterministic, Monte-Carlo, and counterfactual engines
  with `ValidatePipeline` (Inputs → Assumptions → Constraints → Model → Run →
  Results → Audit), reproducibility gate (Gate L), and constraint evaluation
  (`EvaluateConstraint` parses a small expression grammar: `<`, `<=`, `>`, `>=`,
  `==`, `!=`, `&&`, `||`, `!`, parentheses, variable references, numeric and
  boolean literals — issue #210).
- **Golden dataset** (`services/simulation/internal/golden/golden.go`): covers
  all 11 documented categories (simple-deterministic, multi-variable,
  historical-counterfactual, uncertainty, missing-data, contradictory-inputs,
  invalid, extreme-values, scenario-comparison, reproducibility,
  model-version-changes). Backed by `golden_test.go` with one test function per
  category — issue #209.
- **Scenario API** (`services/api/cmd/scenarios.go`): 12 endpoints under
  `/api/v1/scenarios` — list, create, get, validate, run, replay, assumptions,
  evidence, results, timeline, methodology, compare. Each response carries a
  HYPOTHETICAL / SIMULATED disclaimer (Phase 18 section 33).

#### Constitution + Government domain

- **Government types** (`services/legislation/government/`): `Constitution`,
  `President`, `Administration`, `PresidentialTerm`, `GovernmentTransition`,
  `CabinetMember` — country-agnostic, sourced from authoritative material.
- **Kenya seed** (`adapters/kenya/kenya_seed/government.go`): Constitution of
  Kenya 2010, all administrations from 1964 to present (Kenyatta, Moi, Kibaki,
  Uhuru Kenyatta, William Ruto) with presidential terms and transition dates.
- **Government API** (`services/api/cmd/governments.go`): `/api/v1/governments`
  (list administrations), `/api/v1/governments/{id}` (detail + terms),
  `/api/v1/constitution` (authoritative text — never reinterpreted),
  `/api/v1/transitions` (presidential transition timeline). The Constitution
  endpoint carries `reality_layer: "FACT"`.

#### Post-Assent Legislative Lifecycle

- **Post-assent domain** (`services/legislation/internal/domain/post_assent.go`):
  `Act`, `ActVersion` (immutable — ADR-0011), `PostAssentEvent`,
  `PresidentialAssentEvent`, `LegislativeLifecycleAudit`,
  `ActRepository` interface (9 methods including `RecordAssent` — the explicit
  Bill → Act transition per Spec §15).
- **ActRepository implementation**
  (`services/legislation/internal/infrastructure/memory/act_repository.go`):
  thread-safe in-memory repository with country isolation, append-only versions
  + events, and the Spec §15 `RecordAssent` semantics — fixes #202.
- **Post-assent API** (`services/api/cmd/post_assent.go`):
  `/api/v1/acts/{id}/audit` (full lifecycle audit),
  `/api/v1/acts/{id}/events` (post-assent events: commencement, regulations,
  court challenges, amendments),
  `/api/v1/acts/{id}/follow` (Follow-a-Law flagship experience — issue #193,
  creates a real persisted subscription with eight monitoring domains),
  `/api/v1/acts/{id}/lineage` (full legal lineage — Bill → Parliamentary
  journey → Assent → Publication → Commencement → Regulations → Amendments →
  Court decisions → Current status). Missing steps are reported as
  `NOT_VERIFIED`, never inferred.

#### Public Debt & Borrowing Intelligence

- **Public-debt domain** (`services/legislation/internal/domain/public_debt.go`):
  `PublicDebtSnapshot` (immutable), `BorrowingAgreement`,
  `Disbursement`, `Repayment`, `GovernmentDebtSummary`,
  `DebtRepository` interface (10 methods),
  `ValidateAttribution(agreement, []Administration) error` — verifies the
  agreement's `GovernmentAdministrationID` corresponds to the administration
  in power on `ContractDate` (issue #221).
- **DebtRepository implementation**
  (`services/legislation/internal/infrastructure/memory/debt_repository.go`):
  thread-safe in-memory repository; immutable snapshots (rejected by ID and by
  `(country, observation_date)` composite key); chronological list queries;
  orphan-rejecting disbursement/repayment appenders — fixes #203.
- **Kenya debt seed** (`adapters/kenya/kenya_seed/public_debt.go`): 12 CBK
  debt-stock observations (2013–2024), per-administration summaries
  (Uhuru Kenyatta, William Ruto), and 10 borrowing agreements sourced from
  public press releases / prospectuses (China Exim Bank SGR, Eurobonds, World
  Bank DPOs, AfDB Last Mile, IMF SCF/ECF) — fixes #220.
- **Debt API** (`services/api/cmd/public_debt.go`): `/api/v1/debt` (national
  debt dashboard with the latest CBK snapshot + debt service + debt-to-GDP),
  `/api/v1/debt/loans` (borrowing register — each item runs through
  `ValidateAttribution` and surfaces an `attribution_warning` field on
  mismatch), `/api/v1/debt/timeline` (chronological debt-stock observations),
  `/api/v1/debt/governments/{id}` (per-administration summary). The
  `NO_POLITICAL_PERFORMANCE_SCORE` canonical constant is appended to every
  per-administration disclaimer — fixes #222. The platform never attributes
  sovereign borrowing personally to a president.

#### Postgres migrations (Phase 18 + government + post-assent schemas)

- **019_simulation_schema.{up,down}.sql**: 11 tables (`scenarios`,
  `scenario_versions` (immutable), `scenario_assumptions`, `scenario_inputs`,
  `scenario_models`, `simulation_runs`, `simulation_results` (immutable),
  `simulation_metrics`, `scenario_evidence`, `scenario_relationships`,
  `scenario_audits` (append-only)). UUID PKs, `tenant_id` btree indexes,
  CHECK constraints mirroring every domain enum, FK to `identity.users`,
  immutability triggers (BEFORE UPDATE OR DELETE OR TRUNCATE) on
  `scenario_versions`, `simulation_results`, `scenario_audits` — fixes #211.
- **020_government_schema.{up,down}.sql**: 10 tables (`constitutions`,
  `constitution_chapters`, `constitution_articles`,
  `constitution_cross_references`, `presidents`, `administrations`,
  `presidential_terms`, `government_periods`, `cabinet_members`,
  `transitions`). Temporal validity via `EXCLUDE USING gist` (prevents
  overlapping administration/term/period/cabinet windows). `btree_gist`
  extension created inline — fixes #214.
- **021_post_assent_schema.{up,down}.sql**: 4 tables
  (`presidential_assent_events`, `act_versions` (immutable),
  `post_assent_events`, `legislative_lifecycle_audits`). Hard FK
  `act_versions.act_id → legislation.acts(id) ON DELETE RESTRICT`.
  Immutability trigger on `act_versions` (ADR-0011) — fixes #228.

#### Frontend

- **Navbar mega-menu redesign** (`apps/web/src/components/header.tsx`):
  reorganised into 6 thematic groups (Legislation, Government, Finance,
  Intelligence, Scenarios, Resources) so 30+ destinations stay scannable.
  Includes a primary nav (Home, Ask, Briefing, Countries) and an expandable
  Explore dropdown.
- **Scenarios UI** (`apps/web/src/app/scenarios/`): list, create, detail,
  methodology, evidence, results, timeline, assumptions, and comparison
  pages — every page surfaces the HYPOTHETICAL disclaimer.
- **Acts UI** (`apps/web/src/app/acts/[id]/{audit,events,follow,lineage}/`):
  post-assent lifecycle pages wired to the new API endpoints.
- **Government UI** (`apps/web/src/app/governments/`): list + detail + terms
  pages.
- **Debt UI** (`apps/web/src/app/debt/`, `/loans`, `/grants`): public debt
  dashboard with trend chart, sovereign loans tracker, grants page.
- **Trust + Provenance UI** (`/trust`), **Notifications** (`/notifications`),
  **Following** (`/following`), **What Changed** (`/what-changed`),
  **Feed** (`/feed`), **Trending** (`/trending`), **Sponsor** (`/sponsor`).

### Fixed — Post-Phase-1

- **#201** — Parliament `DiscoverBills` infinite recursion (`fetchURL` was
  delegating back to `Fetch`; refactored to make the actual HTTP call and let
  `Fetch` build the `RawDocument`).
- **#204** — `next build` failed on ESLint unused-import errors (removed
  unused imports from 3 files).
- **#207** — `TestRunService_Run_RejectsNonReadyScenario` was tautological:
  the test seeded a DRAFT scenario in a different repo than `rs` used. Rewrote
  the test to use `rs`'s own repo and assert the error mentions READY.
- **#213 / #219** — `governments.go` and `public_debt.go` header comments
  listed unregistered endpoints. Removed the doc lines for routes that are
  not actually registered in `main.go`.
- **#218** — `/acts/page.tsx` rendered a hardcoded array of acts; refactored
  to a server component that fetches `/api/v1/acts` so frontend and backend
  can no longer drift silently.

### Added — Phase 1 (Foundation)

- **Monorepo structure**: `apps/web`, `services/{api,legislation,ingestion,documents,evidence,intelligence,search,notifications,identity,ai}`, `adapters/kenya/{parliament,kenya_law,gazette,internal}`, `packages/{contracts,events,observability,auth,config}`, `infrastructure/{postgres,docker,kubernetes,terraform,observability,temporal,opensearch}`, `docs/{architecture,adr,api,routes}`, `tests/{integration,e2e,evaluation,contract}`.
- **Python AI service** (`services/ai/`): FastAPI app exposing 12 civic AI capabilities (BillSummarizer, TimelineExtractor, StageExplainer, TerminologyExplainer, DocumentComparator, EntityExtractor, TopicClassifier, ImpactAnalyzer, CivicQuestionAnswerer, CitationValidator, ContradictionDetector, BriefingGenerator). Provider-agnostic model gateway with Stub + OpenAI providers, daily budget enforcement, retry, fallback. Full RAG pipeline with hybrid search, reranking, evidence selection, context construction, claim extraction, and mandatory citation validation. 28 unit + integration tests, all passing. Permanent evaluation dataset (`eval/test_eval_dataset.py`) covering factual accuracy, citation correctness, and hallucination guardrails.
- **Next.js frontend** (`apps/web/`): Next.js 14 + Tailwind CSS + TanStack Query + lucide-react. 17 routes serving HTTP 200 — homepage, /bills, /bills/[id] (with timeline/versions/compare/documents/chat sub-routes), /search, /briefing, /about, /committees, /people, /institutions, /country, /country/kenya, /topics, and a custom 404. All pages type-check (`tsc --noEmit`), lint cleanly, and pass a production build. Accessibility: skip-link, ARIA landmarks, focus-visible outlines, reduced-motion support.
- **Database schema**: 16 forward-only Postgres migrations covering 9 logical schemas (legislation, ingestion, documents, evidence, intelligence, notifications, identity, search, audit). pgvector extension enabled. ivfflat + GIN trigram + FTS indexes on document chunks. Audit triggers on canonical tables. CHECK constraint on `intelligence.candidate_facts` enforcing "AI cannot mutate truth".
- **Kenya adapter** (`adapters/kenya/`): Reference implementation of `contracts.LegislativeSourceAdapter`. 11 Bill stages per the 2010 Constitution (FIRST_READING through COMMENCEMENT, plus REJECTED/WITHDRAWN/LAPSED terminals). 30 parliamentary terms with plain-language explanations + source URLs. Bicameral structure (National Assembly + Senate) and 14 committees. Contract tests verify interface compliance, stage transition validity, and terminology completeness. Source adapters (parliament, kenya_law, gazette) defined with polite HTTP client.
- **Go backend foundation**: Root `go.mod` + `packages/contracts/` (country adapter interface, events catalog, typed errors, pagination). `services/legislation/` with `BillStateMachine` (country-agnostic — takes `[]StageDefinition` as input), entities, repository interfaces, `CreateBill` use case, and unit tests verifying valid + invalid transitions. Compile-time assertion that `KenyaAdapter` satisfies the contract interface.
- **Infrastructure**: Docker Compose for the full local dev stack (Postgres+pgvector, Redis, NATS JetStream, MinIO, OpenSearch, Temporal + UI, Keycloak, MailHog, AI service, web). Multi-stage Dockerfiles for Go, Python, and Next.js. CI pipeline via GitHub Actions (Python tests, web lint+build, migration apply against real Postgres, Trivy security scan, Docker image build, CodeQL).
- **Documentation**: README, ARCHITECTURE.md (the canonical contract — the "AI cannot mutate truth" rule, service ownership table, dependency direction, country-adapter pattern, event catalog, Temporal workflows, deployment topology), CONTRIBUTING, SECURITY (threat model + controls + token rotation policy), CODE_OF_CONDUCT. 14 Architecture Decision Records (MADR format): modular monolith, Go+Python split, single Postgres cluster, country adapter pattern, AI cannot mutate truth, NATS JetStream, Temporal, evidence-first RAG, pgvector before OpenSearch, OIDC/Keycloak, immutable Bill versions, SSE for streaming, contradiction engine, monorepo. Partial OpenAPI 3.1 spec for the citizen API.
- **CI/CD**: GitHub Actions for `pull_request` and `push: main`. CodeQL weekly + on PRs. Dependabot weekly for pip, npm, Go modules, Docker, GitHub Actions. Issue templates (bug report + feature request) and a PR template enforcing the architectural checklist.

### Known limitations (Phase 1)

- Go backend services beyond `legislation` are directory-stubbed only. Full source for `api`, `ingestion`, `documents`, `evidence`, `intelligence`, `search`, `notifications`, `identity` is tracked in issue #CI-BE-001.
- Kenya adapter source adapters (parliament, kenya_law, gazette) have interface + fetch stubs but no real crawling/parsing yet. Tracked in issues #CI-AD-002..#CI-AD-008.
- Temporal workflows, Helm chart, and Terraform modules are file stubs. Tracked in issues #CI-INF-001..#CI-INF-003.
- The Next.js frontend ships with mock data. Real data requires the Go BFF + ingestion pipeline to be completed (Phase 2).
- The AI service uses the Stub provider by default. Configure `OPENAI_API_KEY` or `ANTHROPIC_API_KEY` in `.env` to use a real provider.

## [0.1.0] — 2026-09-09

Initial public release of the Civic Intelligence Platform repository structure.
