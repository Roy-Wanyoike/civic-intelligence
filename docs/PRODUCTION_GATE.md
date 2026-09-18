# Production Gate Checklist

**Source:** Spec §81 (`upload/Pasted Content_1789720147692.txt:2623-2646`)
**Cross-references:** Spec §77 (No-Fake-Completion Rule), §78 (Completion Matrix), §79 (Final Re-Audit), §82 (Zero-Pending-Work), §83 (Final Product Walkthrough), §85 (Definition of Done), §86 (The Final Rule)
**Audited by:** `audit-team-5-report.md` §3 — *"No production gate checklist (§81 — 16 gates mandated; 0 present)"* (P1-3, GAP-68-1)
**Audited HEAD:** `dca1c78`
**Date:** 2026-09-18

---

## 1. The gate

> Spec §81: *"Do not declare completion unless every gate below passes. Any failure means the audit continues."*

```text
BUILD                         PASS
UNIT TESTS                    PASS
INTEGRATION TESTS             PASS
CONTRACT TESTS                PASS
E2E TESTS                     PASS
AI EVALUATION                 PASS
DATA QUALITY                  PASS
SECURITY                      PASS
ACCESSIBILITY                 PASS
PERFORMANCE                   PASS
CHAOS                         PASS
DISASTER RECOVERY             PASS
OBSERVABILITY                 PASS
DOCUMENTATION                 PASS
UX REVIEW                     PASS
PRODUCTION WORKFLOW           PASS
```

Status values:
- ✅ **PASS** — gate satisfied; evidence cited.
- ⚠️ **PARTIAL** — some sub-gates pass, some fail; gate is NOT satisfied.
- ❌ **FAIL** — gate not satisfied; blocker cited.
- ⚪ **UNVERIFIED** — could not be assessed this audit cycle (toolchain unavailable, runtime not exercised).

> Spec §81: *"Any failure means the audit continues."*

---

## 2. Gate-by-gate status (as of HEAD `dca1c78`)

| # | Gate | Required | Status | Evidence | Blocker |
|---|------|----------|--------|----------|---------|
| 1 | BUILD | PASS | ⚠️ PARTIAL | `npx tsc --noEmit` exit 0; `npx next lint` exit 0 (1 pre-existing `react-hooks/exhaustive-deps` warning in `debt-trend-chart.tsx`); `npx next build` exit 0 (47 routes compiled, ~87 kB First Load JS shared). ⚠️ Go toolchain unavailable this audit session — Go build not exercised by audit-team-5. CI runs `go build ./...` per `.github/workflows/ci.yml` (P0-2 fix removed `continue-on-error: true`). | audit-team-5 §4: Go build not exercised in audit session; CI exercises it but test step still soft-fails (see gate #2). |
| 2 | UNIT TESTS | PASS | ⚠️ PARTIAL | TS has no unit-test runner configured (no Vitest, no Jest). Go tests run in CI but soft-fail: `.github/workflows/ci.yml:71` — `go test ./... \|\| echo "::warning::tests failed in $d"`. Python deps not installed this session (`pytest` not run). | audit-team-5 P1-20: CI does not enforce Go test pass rate; TS has zero unit tests. |
| 3 | INTEGRATION TESTS | PASS | ❌ FAIL | No `services/*/integration/` directories exist. `docs/architecture/10-testing.md:23` claims they should. The doc is the spec; the directories are missing. | audit-team-5 P1-5: integration-test directories claimed but absent. |
| 4 | CONTRACT TESTS | PASS | ❌ FAIL | Only 6 adapter `contract_test.go` files exist (Kenya, Uganda, Tanzania, Ghana, Nigeria, South Africa). Zero event-contract tests. Zero HTTP producer-consumer contract tests. `packages/contracts/` has no test files. | audit-team-5 §6 gate #4: only adapter-level contract tests present. |
| 5 | E2E TESTS | PASS | ❌ FAIL | `tests/e2e/homepage.spec.ts` + `tests/e2e/accessibility.spec.ts` exist but `@playwright/test` + `@axe-core/playwright` are NOT in `apps/web/package.json` devDependencies — `npm install` completes without installing them, so the 2 specs cannot execute. Only 4 of 47 routes are covered. | audit-team-5 P0-4: Playwright devDependencies missing. |
| 6 | AI EVALUATION | PASS | ⚠️ PARTIAL | `services/ai/eval/test_eval_dataset.py` exists and covers factual accuracy, citation correctness, hallucination guardrails. ⚠️ AI defaults to `StubProvider` (`services/ai/app/config.py:model_gateway_default_provider = "stub"`); Anthropic provider falls back to stub (`gateway.py`: `anthropic_provider_not_implemented_falling_back_to_stub`). Eval scores reflect a stub, not a real LLM. | audit-team-5 P1-14: AI evaluation runs against StubProvider. |
| 7 | DATA QUALITY | PASS | ⚠️ PARTIAL | Golden dataset exists for Phase 18 simulation service (`services/simulation/internal/golden/golden.go`, 11 categories, 760 lines, `golden_test.go`). No data-quality tests for Bills/Acts/Constitution seed data; no schema-validation tests for the OpenAPI spec. | audit-team-5 §6 gate #7: data-quality tests exist only for simulation. |
| 8 | SECURITY | PASS | ❌ FAIL | `SECURITY.md` exists with threat model + controls + token rotation policy. SSRF allowlist implemented (`packages/observability/ssrf.go`). ❌ OIDC verifier skips JWT signature verification (`services/api/internal/oidc/verifier.go:126` — `TODO(issue #59)`). ❌ Stripe checkout returns placeholder URL. ❌ M-Pesa handler does not call Daraja API. ❌ No rate-limit middleware in mux chain (`main.go:220` single global `RateLimit(300, time.Minute)`). ❌ `DevMode` default `true` in `cmd/main.go:35` → `DevVerifier` is runtime verifier. | audit-team-4 §53 + audit-team-5 P0-3: OIDC signature verification skipped; production security blocker. |
| 9 | ACCESSIBILITY | PASS | ❌ FAIL | skip-link, ARIA landmarks, sr-only labels, focus-visible outlines, reduced-motion support all present in code (`apps/web/src/app/globals.css:11-109`). 4 axe-core tests written. ❌ Tests cannot run (Playwright devDependencies missing — see gate #5). Only 4 of 47 routes covered. WCAG 2.2 AA compliance unverified. | audit-team-5 §6 gate #9: axe-core tests can't execute. |
| 10 | PERFORMANCE | PASS | ❌ FAIL | No benchmark, load, soak, or stress test files anywhere in the repo. `rg "benchmark\|k6\|locust\|stress\|soak"` returns 0 hits in code. No baseline measurements exist for `search_latency`, `crawl_success_rate`, `ai_failure_rate`, `citation_validation_failure_rate`. | audit-team-5 P1-7: no performance test suite exists. |
| 11 | CHAOS | PASS | ❌ FAIL | No chaos tests. No chaos-mesh / chaos-monkey / litmus configuration. | audit-team-5 P1-8: chaos engineering absent. |
| 12 | DISASTER RECOVERY | PASS | ❌ FAIL | No DR tests. No backup/restore runbook. Spec §75 critical-failure test (worker crash + retry chain) not implemented — `rg "critical.failure\|CriticalFailure"` returns 0 hits. | audit-team-5 P1-9 + P0-2: no DR test, no critical-failure test. |
| 13 | OBSERVABILITY | PASS | ❌ FAIL | ✅ Prometheus `/metrics` endpoint exists. ✅ Grafana dashboards (4 JSON files in `infrastructure/observability/grafana-dashboards/`). ❌ Only 5 of 30+ metrics from the catalog implemented (`packages/observability/logger.go:229-235`). ❌ `NopTracer` is the only tracer (`packages/observability/tracer.go` — comment says "in production, replace with OTel SDK"). ❌ Loki/Tempo configs present but not wired to any service. ❌ Grafana dashboards directory mismatch (doc says `dashboards/`, actual is `grafana-dashboards/`). | audit-team-5 P1-17 + P1-18: 25+ metrics missing; tracer is no-op; dashboards directory name mismatch. |
| 14 | DOCUMENTATION | PASS | ✅ PASS | README (428 lines), ARCHITECTURE.md (344 lines), 14 ADRs (MADR format), OpenAPI (1,838 lines, 60+ paths), CHANGELOG (184 lines, manually maintained through PR #244), SECURITY.md (93 lines with threat model), DEPLOYMENT.md, CONTRIBUTING.md, CODE_OF_CONDUCT.md, 16 architecture docs in `docs/architecture/`, 2 research docs in `docs/research/`. This Wave-5 docs pass adds: `docs/NO_FAKE_COMPLETION.md`, `docs/COMPLETION_MATRIX.md`, `docs/PRODUCTION_GATE.md`, refreshed `MASTER_AUDIT.md`. | — |
| 15 | UX REVIEW | PASS | ❌ FAIL | No UX review document. No usability test results. Spec §83 final-product-walkthrough (27 steps) not performed this session — requires a running environment (frontend + Go BFF + Postgres + Python AI service). | audit-team-5 P1-10: no UX review documented; §83 walkthrough not performed. |
| 16 | PRODUCTION WORKFLOW | PASS | ❌ FAIL | No end-to-end production-workflow test. Spec §83 walkthrough not performed. Spec §82 zero-pending-work NOT satisfied: 21 `TODO`/`FIXME` markers across 16 Go files; 2 in Python; 30+ `stub`/`placeholder` occurrences; 5+ placeholder issue IDs (`#CI-AD-NG-001`, `#CI-AD-ZA-001`, `#CI-AD-005`, `#CI-AD-007`, `#CI-AD-008`, `#CI-BE-001`, `#CI-INF-001..3`) that are not real GitHub issues. | audit-team-5 §7: zero-pending-work violated; 10+ silently-abandoned TODOs. |

---

## 3. Result

**Aggregate: 1 PASS · 4 PARTIAL · 11 FAIL · 0 UNVERIFIED**

```text
BUILD                         ⚠️ PARTIAL
UNIT TESTS                    ⚠️ PARTIAL
INTEGRATION TESTS             ❌ FAIL
CONTRACT TESTS                ❌ FAIL
E2E TESTS                     ❌ FAIL
AI EVALUATION                 ⚠️ PARTIAL
DATA QUALITY                  ⚠️ PARTIAL
SECURITY                      ❌ FAIL
ACCESSIBILITY                 ❌ FAIL
PERFORMANCE                   ❌ FAIL
CHAOS                         ❌ FAIL
DISASTER RECOVERY             ❌ FAIL
OBSERVABILITY                 ❌ FAIL
DOCUMENTATION                 ✅ PASS
UX REVIEW                     ❌ FAIL
PRODUCTION WORKFLOW           ❌ FAIL
```

> Spec §81: *"Any failure means the audit continues."*

**The audit must continue.** The platform is NOT production-ready. The completion matrix in `docs/COMPLETION_MATRIX.md` confirms: zero features are at `VERIFIED` (spec §78's only "complete" status).

---

## 4. Required remediations to pass each FAIL gate

The remediations are listed in priority order. Each must land before the corresponding gate can move from `FAIL` to `PASS`.

### Gate 8 — SECURITY (P0-3, blockers)
1. Implement OIDC JWT signature verification using the JWKS key matching `header.Kid` — closes `services/api/internal/oidc/verifier.go:126` TODO. (audit-team-5 P0-3, audit-team-4 §53 blocker)
2. Wire Stripe checkout to the real Stripe Checkout Sessions API — replace `cs_test_placeholder_…` URL.
3. Wire M-Pesa handler to the Daraja API (STK Push).
4. Add per-tier rate-limit middleware (per-user, per-role, AI-endpoint tightening).
5. Set `DevMode = false` in production config; gate `DevVerifier` behind the flag.

### Gate 5 — E2E TESTS (P0-4)
1. Add `@playwright/test` + `@axe-core/playwright` to `apps/web/package.json` devDependencies.
2. Write the 8 golden user journey tests (spec §74) in `tests/e2e/journeys/`. Each journey should click through every page in the spec's flow and assert on real data — not just `<h1>` text.
3. Add axe-core scans for every route (currently 4 of 47 covered).

### Gate 12 — DISASTER RECOVERY + Gate 11 — CHAOS (P0-2, P1-8, P1-9)
1. Write the §75 critical failure test as a Temporal workflow test that injects worker crashes, NATS delays, AI timeouts, and verifies the final canonical state.
2. Add chaos-mesh or chaos-monkey configuration; run chaos tests in CI.
3. Write a backup/restore runbook; test it.

### Gates 3 + 4 — INTEGRATION + CONTRACT TESTS (P1-5)
1. Create `services/<svc>/integration/` directories with at least one integration test per service.
2. Add event-contract tests (NATS producer ↔ consumer) in `packages/events/`.
3. Add HTTP producer-consumer contract tests in `packages/contracts/`.

### Gate 10 — PERFORMANCE (P1-7)
1. Add k6 load tests for the API BFF hot paths (`/bills`, `/search`, `/debt`, `/scenarios`).
2. Add Go benchmarks for the domain logic (`go test -bench=.`).
3. Add baseline measurements for `search_latency`, `crawl_success_rate`, `ai_failure_rate`, `citation_validation_failure_rate`.

### Gate 13 — OBSERVABILITY (P1-17, P1-18)
1. Implement the remaining 25+ metrics from the `09-observability.md` catalog.
2. Replace `NopTracer` with the real OTel SDK in production.
3. Wire Loki/Tempo to running services.
4. Rename `infrastructure/observability/grafana-dashboards/` to `dashboards/` to match the doc (or update the doc).

### Gate 15 — UX REVIEW (P1-10)
1. Conduct the §83 27-step final product walkthrough (Open homepage → inspect first viewport → use Civic Highlights → Ask Civic → Search → Open a Bill → Inspect timeline → Inspect evidence → Follow → Constitution Spotlight → Read Constitution → Select government → Select presidential term → Inspect legislature → Inspect debt → Open a borrowing record → Inspect graph sources → Start research → Inspect research evidence → Inspect contradictions → Test notifications → Test mobile → Test accessibility → Test slow network → Test errors → Test logout/login → Test permissions).
2. Record actual results; produce a UX review document.

### Gate 16 — PRODUCTION WORKFLOW (§82 zero-pending-work)
1. Convert each `TODO(issue #CI-AD-NNN)` placeholder to a real GitHub issue with owner + acceptance criteria (per §71, §82).
2. Convert the 21 Go + 2 Python `TODO`/`FIXME` markers to tracked issues OR resolve them.
3. Replace the 30+ `stub`/`placeholder` occurrences with real implementations OR explicitly mark them as `IN_PROGRESS` in `services/api/README.md` and `docs/COMPLETION_MATRIX.md`.

### Gate 2 — UNIT TESTS (P1-20)
1. Remove the `|| echo "::warning::tests failed in $d"` soft-fail from `.github/workflows/ci.yml:71`. Go test failures must fail CI.
2. Add a Vitest or Jest configuration for `apps/web/` so TS unit tests can run.

### Gate 1 — BUILD (no blocker cited; complete the audit)
1. Run `go build ./...` and `go test ./...` in the audit environment (Go toolchain was unavailable to audit-team-5 this session).
2. Run `pytest` in the audit environment (Python deps were not installed this session).

---

## 5. Cross-references

- `MASTER_AUDIT.md` — top-level audit; references this gate.
- `docs/NO_FAKE_COMPLETION.md` — the rule that governs every gate's pass/fail/partial status.
- `docs/COMPLETION_MATRIX.md` — the per-feature status matrix; only `VERIFIED` features can contribute to a `PASS` gate.
- `audit-team-5-report.md` §6 — the audit-team-5 gate-by-gate assessment that this document formalises.
- `audit-team-4-report.md` §53 — the SECURITY gate source.
- `SECURITY.md` — the threat model + controls that gate #8 references.
- `docs/architecture/09-observability.md` — the metrics catalog that gate #13 references.
- `docs/architecture/10-testing.md` — the test taxonomy that gates #2-#5 reference.

---

## 6. Sign-off

This gate is **NOT passing**. The audit must continue (spec §79, §80, §81, §86).

| Role | Name | Sign-off | Date |
|------|------|----------|------|
| Engineering lead | — | ❌ NOT signed off — 11 gates failing | 2026-09-18 |
| Independent QA (§73) | — | ❌ NOT signed off — 0 features at `VERIFIED` | 2026-09-18 |
| Security reviewer | — | ❌ NOT signed off — P0-3 OIDC signature verification skipped | 2026-09-18 |
| Release engineer | Wave-5 docs pass | ⚠️ PARTIAL — documentation gate passes; remaining 11 gates tracked for follow-up waves | 2026-09-18 |

When all 16 gates reach `PASS`, this section is updated with the names + dates of the four sign-offs above, the §83 final product walkthrough is performed, and `docs/COMPLETION_MATRIX.md` is reviewed for any remaining non-`VERIFIED` rows.

---

*This gate is governed by spec §81. **Any failure means the audit continues.***
