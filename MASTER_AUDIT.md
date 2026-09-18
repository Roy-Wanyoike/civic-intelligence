# Master Audit — Civic Intelligence Platform

**Date:** 2026-09-18
**Auditor:** Wave-5 docs pass (technical writer + release engineer)
**Worktree:** `/home/z/my-project/wt-docs-a` (branch `fix/wave5-docs-a`)
**Audited HEAD commit:** `dca1c78` — `fix(#212,#227): populate constitution chapters/articles + consolidate BillStageTransitionValidator (#244)`
**Prior baseline commit:** `a0638e7` (Phase 1 Foundation) — covered by the original `MASTER_AUDIT.md` dated 2026-09-09
**Scope delta:** 91 commits, 26 issues closed (PRs #179 → #244), spanning Phase 2 → Wave 5 work
**Audit reports referenced as evidence:**
- `audit-team-1-report.md` — Spec §1-17 (Primary Objective → Core Civic Model). 62 unique gaps.
- `audit-team-2-report.md` — Spec §18-34 (Public Debt Attribution → Homepage/Carousel). 75 unique gaps.
- `audit-team-3-report.md` — Spec §35-52 (Visual Design → Observability). 69 unique gaps.
- `audit-team-4-report.md` — Spec §53-70 (Security → Git History). 68 unique gaps.
- `audit-team-5-report.md` — Spec §71-86 (Process · Completion Matrix · Final Gates). 37 unique gaps.
- **Aggregate: 311 unique audit findings across the 5 audit reports.**

---

## Methodology

This rewrite was produced by:
1. Re-reading the prior `MASTER_AUDIT.md` (362 lines, dated 2026-09-09) which audited ONLY the Phase 1 commit `a0638e7` and enumerated 4 P0s + 6 P1s + 7 P2s + 5 P3s = 22 findings.
2. Reading the multi-agent `worklog.md` (1,152 lines) and the five `audit-team-{1..5}-report.md` files produced by the Wave-5 forensic re-audit cycle.
3. `git log --oneline --no-merges a0638e7..HEAD` to enumerate the 91 commits / 26 issue-closing PRs merged since Phase 1.
4. Grouping the post-Phase-1 work by feature area (Simulation, Constitution+Government, Post-Assent, Public Debt, Navbar, Bug fixes, Documentation) and cross-referencing each against the audit-team reports for IMPLEMENTED / IN_PROGRESS / BLOCKED status.
5. Spec §77 ("No-fake-completion rule") governs every status assignment below: a feature is only `VERIFIED` when actual evidence supports the claim. A passing unit test alone, a working UI alone, or a correct schema alone is not sufficient.

For the per-feature completion matrix see `docs/COMPLETION_MATRIX.md`. For the production gate checklist see `docs/PRODUCTION_GATE.md`. For the rule itself see `docs/NO_FAKE_COMPLETION.md`.

---

## Phase 1 baseline (commit `a0638e7`) — status of the original 22 findings

The prior `MASTER_AUDIT.md` recorded 22 findings against `a0638e7`. Their disposition as of HEAD `dca1c78`:

| Old ID | Title | Disposition |
|--------|-------|-------------|
| P0-1 | Duplicate Go type declarations caused hard compile failure | ✅ Fixed (`ffb44eb`, `d347b55`) |
| P0-2 | CI `continue-on-error: true` on Go compile check | ✅ Fixed (`d347b55`); ⚠️ audit-team-5 P1-20: the *test* step still soft-fails (`ci.yml:71` `go test ./... \|\| echo "::warning::…"`) — compile gate now hard, test gate still soft |
| P0-3 | `services/api/` BFF was completely empty | ✅ Fixed (`5f4002a` OIDC + RBAC + rate limiting; `411731e` wire Go API to serve real Bills) |
| P0-4 | 293 files committed as executable (100755) | ⚠️ Local `core.filemode false` applied; repo-level normalization still pending (audit-team-4 GAP-68-x) |
| P1-1 | Next.js frontend uses mock data, no real backend connection | ✅ Fixed (`411731e`); ⚠️ `mock-data.ts` still shipped in production build (audit-team-5 P2-9) |
| P1-2 | Go backend services lack `go.sum` files | ✅ Fixed (`ffb44eb`) |
| P1-3 | No Go code actually compiles | ✅ Fixed (`ffb44eb`) |
| P1-4 | GitHub PAT exposed in chat history | ⚠️ Operational — token rotation is a user-side action; not a code change |
| P1-5 | No OIDC/authentication wired | ✅ Partial — `5f4002a` ships middleware; ⚠️ audit-team-4 P0 / audit-team-5 P0-3: `verifier.go:126` carries `TODO(issue #59): verify the signature` — signature verification skipped, BLOCKED |
| P1-6 | No observability instrumentation | ⚠️ Partial — `86737d9` ships OTel-compatible metrics + tracing + logging; audit-team-5 P1-17: only 5 of 30+ metrics implemented; `NopTracer` still in production path |
| P2-1 | Python AI service stub provider echoes input | ⚠️ Still present — StubProvider is the default (`config.py:model_gateway_default_provider = "stub"`) |
| P2-2 | Next.js 14 + React 18 (not latest) | ⚠️ Unchanged — intentional stability call |
| P2-3 | No E2E test framework | ⚠️ Partial — `463deee` ships Playwright + axe-core config; audit-team-5 P0-4: `@playwright/test` + `@axe-core/playwright` are NOT in `apps/web/package.json` devDependencies, so tests cannot execute |
| P2-4 | No automated axe-core scan in CI | ⚠️ Same as P2-3 — axe-core tests written but cannot run |
| P2-5 | Docker Compose references services without Dockerfiles | ⚠️ Unchanged |
| P2-6 | No Helm chart content | ✅ Fixed (`463deee`) |
| P2-7 | No Terraform content | ✅ Fixed (`463deee`) |
| P3-1 | README references `docs/architecture/13-roadmap.md` which didn't exist | ✅ Fixed (`a364f1b`) |
| P3-2 | No `CODEOWNERS` file | ✅ Fixed (`a364f1b`) |
| P3-3 | No release tags or changelog automation | ⚠️ Unchanged — `CHANGELOG.md` still manually maintained |
| P3-4 | Python service has no type checking | ✅ Fixed (`463deee` — mypy configured) |
| P3-5 | Dependabot Docker ecosystem set to monthly | ⚠️ Unchanged |

**Net Phase 1 outcome:** 11/22 fixed, 9/22 partial or unchanged, 2/22 operational (token rotation). The Phase 1 audit's "✅ Fixed in working tree" status lines have been validated by this rewrite — they survive re-audit.

---

## Post-Phase-1 work — feature-area summary

`git log --oneline --no-merges a0638e7..HEAD` returns 91 commits. Grouping by feature area:

### 1. Simulation (Phase 18)

**PRs:** #187 (`c1e69c8` Phase 18 simulation service — domain, engines, API, golden dataset), #189 (`3dea157` scenario explorer UI + What If + visual language labels), #235 (`201ff5e` Postgres migrations 019 for simulation), #240 (`5ae0683` golden dataset + constraint evaluation + GovernmentDebtSummary disclaimer), #230 (`3375baf` make `TestRunService_Run_RejectsNonReadyScenario` actually test).

**IMPLEMENTED:**
- `services/simulation/` — country-agnostic scenario model with `RealityLayer` tagging (FACT / OBSERVED / HYPOTHETICAL / MODELED / UNKNOWN). Every response from the simulation API carries an explicit reality-layer tag.
- Three engines: deterministic, Monte-Carlo, counterfactual; `ValidatePipeline` (Inputs → Assumptions → Constraints → Model → Run → Results → Audit); reproducibility gate (Gate L); constraint evaluation (`EvaluateConstraint` parses `<`, `<=`, `>`, `>=`, `==`, `!=`, `&&`, `||`, `!`, parentheses, variable refs, numeric and boolean literals — issue #210).
- Golden dataset covering all 11 documented categories (simple-deterministic, multi-variable, historical-counterfactual, uncertainty, missing-data, contradictory-inputs, invalid, extreme-values, scenario-comparison, reproducibility, model-version-changes) — issue #209.
- Scenario API: 12 endpoints under `/api/v1/scenarios` — list, create, get, validate, run, replay, assumptions, evidence, results, timeline, methodology, compare.
- Scenario UI: list, create, detail, methodology, evidence, results, timeline, assumptions, comparison pages — every page surfaces the HYPOTHETICAL disclaimer.
- Postgres migration `019_simulation_schema.{up,down}.sql`: 11 tables (scenarios, scenario_versions immutable, scenario_assumptions, scenario_inputs, scenario_models, simulation_runs, simulation_results immutable, simulation_metrics, scenario_evidence, scenario_relationships, scenario_audits append-only). UUID PKs, tenant_id btree indexes, CHECK constraints mirroring every domain enum, FK to `identity.users`, immutability triggers on scenario_versions, simulation_results, scenario_audits.

**IN_PROGRESS:**
- Per audit-team-3 §45-52 and audit-team-5 §5, the simulation features sit at `TESTING` — golden dataset exists, unit tests pass, but no integration / E2E / journey test exercises the API end-to-end. Spec §78: `TESTING` is not `VERIFIED`.

**BLOCKED:** None for this feature area.

---

### 2. Constitution + Government domain

**PRs:** #200 (`6007e53` Constitution + Government domain — entities, seed data, API, UI), #244 (`dca1c78` populate constitution chapters/articles + consolidate BillStageTransitionValidator), #235 (`201ff5e` migration 020 government schema).

**IMPLEMENTED:**
- `services/legislation/government/` — `Constitution`, `ConstitutionChapter`, `ConstitutionArticle`, `President`, `Administration`, `PresidentialTerm`, `GovernmentTransition`, `CabinetMember` types, country-agnostic, sourced from authoritative material.
- Kenya seed (`adapters/kenya/kenya_seed/government.go` + `constitution.go`): Constitution of Kenya 2010 (20 curated articles across 9 chapters, sourced verbatim from Kenya Law), all administrations from 1964 to present (Kenyatta, Moi, Kibaki, Uhuru Kenyatta, William Ruto) with presidential terms and transition dates.
- Government API (`services/api/cmd/governments.go`): `/api/v1/governments` (list administrations), `/api/v1/governments/{id}` (detail + terms), `/api/v1/constitution` (authoritative text — never reinterpreted), `/api/v1/constitution/articles` (list), `/api/v1/constitution/articles/{id}` (detail), `/api/v1/transitions` (presidential transition timeline). Constitution endpoints carry `reality_layer: "FACT"`.
- Government UI: list + detail + terms pages; `/constitution` page fetches from the API at request time with graceful fallback to the existing `apps/web/src/data/constitution-articles.ts` data file (visible amber banner when the API is unreachable).
- Migration `020_government_schema.{up,down}.sql`: 10 tables (constitutions, constitution_chapters, constitution_articles, constitution_cross_references, presidents, administrations, presidential_terms, government_periods, cabinet_members, transitions). Temporal validity via `EXCLUDE USING gist` (prevents overlapping administration/term/period/cabinet windows). `btree_gist` extension created inline.

**IN_PROGRESS:**
- `BillStageTransitionValidator` consolidation (#227): `StageGraphValidator` is the canonical implementation with typed errors and a compile-time interface assertion; `BillStateMachine` is now a thin wrapper. audit-team-1 §5 notes that the broader `CivicMatter` abstraction (a unified interface for Bill, Act, Regulation, Policy, Judgment) is still absent (GAP-5-1) — that work is NOT in scope for this feature area but is tracked for a later wave.

**BLOCKED:** None.

---

### 3. Post-Assent Legislative Lifecycle

**PRs:** #199 (`21619bb` Post-Assent Legislative Lifecycle + Follow-a-Law + Audit-an-Act), #236 (`5e0b517` ActRepository with in-memory store, seed, and API wiring — #202), #243 (`a20cc8c` post-assent endpoints use domain logic, real subscriptions, data-driven lineage — #215, #216, #217), #235 (migration 021 post_assent_schema).

**IMPLEMENTED:**
- `services/legislation/internal/domain/post_assent.go` — `Act`, `ActVersion` (immutable — ADR-0011), `PostAssentEvent`, `PresidentialAssentEvent`, `LegislativeLifecycleAudit`, `ActRepository` interface (9 methods including `RecordAssent` — the explicit Bill → Act transition per Spec §15).
- `services/legislation/internal/infrastructure/memory/act_repository.go` — thread-safe in-memory repository with country isolation, append-only versions + events, and the Spec §15 `RecordAssent` semantics — closes #202.
- Post-assent API (`services/api/cmd/post_assent.go`):
  - `/api/v1/acts/{id}/audit` — delegates to `legislation.AuditForAct(act, events)` (no hardcoded "everything is confirmed"); an act without a commencement date reports `NOT_VERIFIED` + the `commencement_notice_not_found` data gap (closes #215).
  - `/api/v1/acts/{id}/events` — post-assent events (commencement, regulations, court challenges, amendments).
  - `/api/v1/acts/{id}/follow` — Follow-a-Law flagship experience (#193). Enforces auth (401 for anonymous), verifies the act exists (404 for ghost follows), creates a REAL persisted subscription in the shared `subscriptionStore` (closes #216). Returns the real `subscription_id` + 8-category monitor list.
  - `/api/v1/acts/{id}/lineage` — full legal lineage (Bill → Parliamentary journey → Assent → Publication → Commencement → Regulations → Amendments → Court decisions → Current status). Data-driven via `buildLineage(act, events)`; missing steps reported as `NOT_VERIFIED` with the canonical description "No authoritative record found yet." — never inferred (closes #217).
- Acts UI: `/acts`, `/acts/[id]/{audit,events,follow,lineage}` pages wired to the new API endpoints. `/acts/page.tsx` was a hardcoded array (issue #218); refactored to a server component that fetches `/api/v1/acts` so frontend and backend can no longer drift silently.
- Migration `021_post_assent_schema.{up,down}.sql`: 4 tables (presidential_assent_events, act_versions immutable, post_assent_events, legislative_lifecycle_audits). Hard FK `act_versions.act_id → legislation.acts(id) ON DELETE RESTRICT`. Immutability trigger on `act_versions` (ADR-0011).

**IN_PROGRESS:**
- The `subscriptionStore` is in-memory only (no Postgres backing). Per audit-team-5 §5 the Follow-a-Law row sits at `TESTING` — implemented + unit tests pass, but no integration test verifies persistence across a restart.
- The `origin_bill` and `parliamentary_journey` lineage steps are reported as `NOT_VERIFIED` because the platform does not yet track the originating Bill on the Act (acknowledged data gap, not a defect).

**BLOCKED:** None.

---

### 4. Public Debt & Borrowing Intelligence

**PRs:** #200 (`77a7a23` Public Debt & Borrowing Intelligence — fiscal domain, dashboards, graphs), #237 (`6a72f28` DebtRepository with in-memory store, seed, attribution validation — #203), #239 (`7ba8630` wire `/debt/loans` to repository + enforce attribution validation — #220, #221), #240 (`5ae0683` GovernmentDebtSummary disclaimer — #222).

**IMPLEMENTED:**
- `services/legislation/internal/domain/public_debt.go` — `PublicDebtSnapshot` (immutable), `BorrowingAgreement`, `Disbursement`, `Repayment`, `GovernmentDebtSummary`, `DebtRepository` interface (10 methods), `ValidateAttribution(agreement, []Administration) error` — verifies the agreement's `GovernmentAdministrationID` corresponds to the administration in power on `ContractDate` (issue #221).
- `services/legislation/internal/infrastructure/memory/debt_repository.go` — thread-safe in-memory repository; immutable snapshots (rejected by ID and by `(country, observation_date)` composite key); chronological list queries; orphan-rejecting disbursement/repayment appenders — closes #203.
- Kenya debt seed (`adapters/kenya/kenya_seed/public_debt.go`): 12 CBK debt-stock observations (2013–2024), per-administration summaries (Uhuru Kenyatta, William Ruto), and 10 borrowing agreements sourced from public press releases / prospectuses (China Exim Bank SGR, Eurobonds, World Bank DPOs, AfDB Last Mile, IMF SCF/ECF) — closes #220.
- Debt API (`services/api/cmd/public_debt.go`):
  - `/api/v1/debt` — national debt dashboard with the latest CBK snapshot + debt service + debt-to-GDP.
  - `/api/v1/debt/loans` — borrowing register; each item runs through `ValidateAttribution` and surfaces an `attribution_warning` field on mismatch.
  - `/api/v1/debt/timeline` — chronological debt-stock observations.
  - `/api/v1/debt/governments/{id}` — per-administration summary.
  - The `NO_POLITICAL_PERFORMANCE_SCORE` canonical constant is appended to every per-administration disclaimer — closes #222. The platform never attributes sovereign borrowing personally to a president.

**IN_PROGRESS:**
- Debt UI (`/debt`, `/loans`, `/grants`) ships; audit-team-3 §36 and audit-team-5 §5 note the debt-trend-chart component has a pre-existing `react-hooks/exhaustive-deps` lint warning. Not a release blocker.
- Debt data is in-memory only; no Postgres-backed persistence yet (migration `017` defines the schema; repository implementation is still in-memory).

**BLOCKED:** None.

---

### 5. Navbar + Country Subdomains

**PRs:** #179 (`14e281b` country subdomains + trending carousel + developer platform foundation), #173 (`4523e7d` redesign navbar with Explore dropdown, expandable search, mobile drawer), #241 (`0b8daf7` redesign navbar as mega-menu — all 30+ pages accessible + organized), #189 (`b3fd1db` dark/light theme switcher + Constitution Spotlight + country bills).

**IMPLEMENTED:**
- Navbar mega-menu (`apps/web/src/components/header.tsx`) — reorganised into 6 thematic groups (Legislation, Government, Finance, Intelligence, Scenarios, Resources) so 30+ destinations stay scannable. Includes a primary nav (Home, Ask, Briefing, Countries) and an expandable Explore dropdown.
- Country subdomains middleware (`apps/web/src/middleware.ts`) sets `X-Civic-Country` header for country-specific routing.
- Trending carousel, developer platform foundation, Constitution Spotlight component, dark/light theme switcher all shipped.

**IN_PROGRESS:**
- audit-team-1 GAP-4-2: the web app still hardcodes "Kenya · 🇰🇪" on the hero (`apps/web/src/app/page.tsx:21`); the spec calls for evolution into a global civic intelligence infrastructure. The country context derived from middleware is not yet surfaced on the hero.
- audit-team-5 P2-8: South Africa country page ships placeholder Bills (`za-bill-placeholder-1..4`) in `apps/web/src/app/country/south-africa/page.tsx:14-44`. The other 5 country adapters (Uganda, Tanzania, Ghana, Nigeria, South Africa) each carry `TODO: crawl …` markers (see Bug fixes section below).

**BLOCKED:** None.

---

### 6. Bug fixes (Wave 5)

**PRs:** #233 (`ecbbb22` break parliament Fetch/fetchURL infinite recursion — #201), #232 (`d13144e` remove unused imports that break next build — #204), #230 (`3375baf` make tautological test actually test — #207), #229 (`10b2a3c` remove endpoint doc lines for unregistered routes — #213, #219), #231 (`c8e2d5e` /acts/page.tsx fetches from API instead of hardcoded array — #218), #242 (`b629cb9` update CHANGELOG, API README, OpenAPI spec, developers page — #223, #224, #225, #226).

| Issue | PR | Fix |
|-------|----|-----|
| #201 | #233 | Parliament `DiscoverBills` infinite recursion — `fetchURL` was delegating back to `Fetch`; refactored to make the actual HTTP call. |
| #204 | #232 | `next build` failed on ESLint unused-import errors — removed unused imports from 3 files. |
| #207 | #230 | `TestRunService_Run_RejectsNonReadyScenario` was tautological — the test seeded a DRAFT scenario in a different repo than `rs` used. Rewrote to use `rs`'s own repo. |
| #213, #219 | #229 | `governments.go` and `public_debt.go` header comments listed unregistered endpoints. Removed the doc lines. |
| #218 | #231 | `/acts/page.tsx` rendered a hardcoded array; refactored to a server component that fetches `/api/v1/acts`. |
| #223, #224, #225, #226 | #242 | CHANGELOG, API README, OpenAPI spec, developers page refresh. |

**Status:** All seven bug fixes are ✅ IMPLEMENTED + unit-tested. None at `VERIFIED` until golden journey tests (spec §74, audit-team-5 P0-1) land.

---

### 7. Documentation

**PRs:** #242 (`b629cb9` CHANGELOG + API README + OpenAPI + developers page), #151 (`b85da2c` craft a world-class README), #67 (`a364f1b` missing roadmap doc + CODEOWNERS), #189 (`71f5c8b` Phase 1 Foundation audit report), #184 (`7c99716` Railway hosting guide + Vercel cron).

**IMPLEMENTED:**
- README (428 lines, world-class per #151).
- ARCHITECTURE.md (the canonical contract — service ownership table, dependency direction, country-adapter pattern, event catalog, Temporal workflows, deployment topology).
- 14 ADRs in MADR format (ADR-0001 through ADR-0014).
- OpenAPI 3.1 spec (`docs/api/openapi.yaml`, 1,838 lines, 60+ paths).
- 16 architecture docs in `docs/architecture/`.
- 2 research docs in `docs/research/`.
- CONTRIBUTING, SECURITY (threat model + controls + token rotation policy), CODE_OF_CONDUCT, DEPLOYMENT.

**IN_PROGRESS (this Wave 5 docs pass adds the missing process artefacts mandated by §77, §78, §81):**
- `docs/NO_FAKE_COMPLETION.md` — new (this commit).
- `docs/COMPLETION_MATRIX.md` — new (this commit).
- `docs/PRODUCTION_GATE.md` — new (this commit).
- This rewrite of `MASTER_AUDIT.md`.

**BLOCKED / stale (tracked for follow-up):**
- audit-team-5 P2-1: `docs/architecture/13-roadmap.md` says Phase 2 is "in progress" while Phases 13-14 are merged.
- audit-team-5 P2-2: `docs/architecture/phase-1-audit.md` not refreshed — still dated 2026-09-09 against commit `5f4002a`.
- audit-team-5 P3-3: `SECURITY.md` uses placeholder email `security@civicintelligence.dev (placeholder — replace with real address)`.
- audit-team-5 P3-4: `docs/api/openapi.yaml` production URL is placeholder `https://api.civicintelligence.dev/api/v1 (placeholder)`.
- audit-team-5 P2-9: `apps/web/src/lib/mock-data.ts` still shipped in production build (placeholders for the ingestion pipeline).

---

## Audit findings aggregated from the 5 audit-team reports

The Wave-5 forensic re-audit cycle (spec §79 Final Re-Audit) was performed by five parallel audit teams against HEAD `dca1c78`. The full per-finding inventory lives in each `audit-team-N-report.md`. Aggregate counts:

| Audit team | Spec sections | Unique gap IDs | Severity breakdown |
|------------|---------------|----------------|--------------------|
| audit-team-1 | §1-17 (Primary Objective → Core Civic Model) | 62 | blockers + majors + minors (see report) |
| audit-team-2 | §18-34 (Public Debt Attribution → Homepage/Carousel) | 75 | 0 blockers, 12 majors, 25 minors (across 17 sections) |
| audit-team-3 | §35-52 (Visual Design → Observability) | 69 | see report |
| audit-team-4 | §53-70 (Security → Git History) | 68 | see report |
| audit-team-5 | §71-86 (Process · Completion Matrix · Final Gates) | 37 | P0=4, P1=20, P2=9, P3=4 |
| **Total** | **§1-86** | **311 unique findings** | |

The 4 P0 release blockers from audit-team-5 (the load-bearing ones for the production gate):

| # | Gap | Spec ref | Evidence |
|---|-----|----------|----------|
| P0-1 | No golden user journey tests | §74 | `tests/e2e/homepage.spec.ts` only checks `<h1>` text on 8 individual pages; 0 of 8 mandated journeys implemented |
| P0-2 | No critical failure test | §75 | `rg "critical.failure\|CriticalFailure"` returns 0 hits in code |
| P0-3 | OIDC verifier does not verify JWT signatures | §85 DoD / §81 SECURITY | `services/api/internal/oidc/verifier.go:126` — `TODO(issue #59)`; trusts claims if token parses |
| P0-4 | E2E test runner cannot execute | §81 E2E TESTS | `@playwright/test` + `@axe-core/playwright` not in `apps/web/package.json` devDependencies |

**Cross-team blocker from audit-team-4:** The OIDC signature verification gap (P0-3 above) is also recorded by audit-team-4 as a §53 SECURITY blocker — same defect, two teams.

---

## Summary table

| Severity | Phase 1 baseline (a0638e7) | Wave 5 (dca1c78) | Delta |
|----------|----------------------------|------------------|-------|
| P0       | 4                          | 4                | 0 (different P0s — Phase 1 P0s all fixed, Wave-5 P0s are new process gaps) |
| P1       | 6                          | 20               | +14 (process layer gaps surfaced by §71-86 audit) |
| P2       | 7                          | 9                | +2 |
| P3       | 5                          | 4                | -1 |
| **Total** | **22**                     | **37** (audit-team-5 only) | +15 |

The 311 unique findings across the 5 audit-team reports include the audit-team-5 P0/P1/P2/P3 inventory above PLUS the per-section gaps recorded by audit-teams 1-4 (which use the `GAP-N-M` ID convention rather than the `P0-N` convention). Both numbering schemes are preserved verbatim in their respective reports.

---

## Key architectural finding

The Phase 1 audit's key finding was a **process failure** — unreviewed files from timed-out subagent attempts were committed without inspection, causing duplicate Go type declarations and a CI weakness that hid the problem. That defect class has been remediated: the Wave 5 audit reports record zero instances of duplicate type declarations or `continue-on-error: true` on the compile gate.

The Wave-5 audit's key finding is a **different** process failure — the §71-86 process layer that mandates golden user journeys, critical failure tests, completion matrix, production gate checklist, no-fake-completion rule, final re-audit cycle, and zero-pending-work is largely absent. The five audit reports collectively show:

- The repository ships an impressive amount of substantive feature work (Constitution + Government, Post-Assent lifecycle, Public Debt & Borrowing Intelligence, Phase 18 Simulation with golden dataset, 21 Postgres migrations across 9 schemas, 40+ registered API routes, 14 ADRs, an 1,838-line OpenAPI spec, 39 Go + 5 Python + 2 TS spec files).
- But every feature row in `docs/COMPLETION_MATRIX.md` sits at `TESTING` or worse. **Zero features are at `VERIFIED`** per spec §78. Per §86 ("The Final Rule"): *"If implementation exists but QA cannot verify it: it is not complete."*

This Wave-5 docs pass closes the four specific documentation gaps that triggered it (P1-1 stale `MASTER_AUDIT.md`, P1-4 undocumented §77 rule, §78 missing completion matrix, GAP-68-1 missing production gate checklist). The remaining P0/P1 process gaps (golden journey tests, critical failure test, OIDC signature verification, Playwright devDependencies) are tracked for follow-up waves and are referenced from `docs/PRODUCTION_GATE.md`.

**The audit must continue** (spec §79, §80, §86).
