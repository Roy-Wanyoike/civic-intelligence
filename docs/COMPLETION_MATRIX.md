# Completion Matrix

**Source:** Spec §78 (`upload/Pasted Content_1789720147692.txt:2502-2543`)
**Cross-references:** Spec §77 (No-Fake-Completion Rule), §79 (Final Re-Audit), §81 (Final Production Gate), §85 (Definition of Done), §86 (The Final Rule)
**Audited by:** `audit-team-5-report.md` §3.6 — *"Completion matrix (§78) — does not exist"* (P1-2)
**Audited HEAD:** `dca1c78`
**Date:** 2026-09-18

---

## 1. How to read this matrix

**Allowed status values (spec §78):**

```
DISCOVERED      — spec only, no code exists yet
IN_PROGRESS     — partial implementation; at least one DONE criterion missing (see docs/NO_FAKE_COMPLETION.md §2)
IMPLEMENTED     — all 14 DONE criteria satisfied; QA has NOT yet signed off
TESTING         — implementation complete; tests written; QA NOT yet signed off
QA              — independent QA is actively verifying
BLOCKED         — cannot satisfy one or more criteria until an external dependency lands
VERIFIED        — independent QA has signed off; the ONLY status that counts as complete
```

> **Only `VERIFIED` counts as complete (§78).**

**Column legend:**
- ✅ = present and exercised end-to-end
- ⚠️ = present but partial (e.g. unit tests only, in-memory store only, falls back to hardcoded data)
- ❌ = absent or stub
- n/a = not applicable to this feature

**Evidence column** cites the file path or issue that proves the status. The full §78 columns are preserved: Feature | Requirement | Domain | Implementation | API | Database | UI | Integration | Tests | QA | Security | Performance | Observability | Documentation | Status | Evidence | Issue | PR.

---

## 2. Master matrix

| Feature | Requirement | Domain | Implementation | API | Database | UI | Integration | Tests | QA | Security | Performance | Observability | Documentation | Status | Evidence | Issue | PR |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Bill discovery | §74 J1, §76 | ✅ `domain/bill.go` | ✅ | ✅ `/api/v1/bills` | ✅ migration 009 | ✅ `/bills` | ⚠️ in-memory store | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ 1 metric | ✅ OpenAPI | TESTING | `adapters/kenya/kenya_law/`, `services/api/cmd/main.go:94` | #19, #20 | #94 |
| Bill timeline | §74 J1 | ✅ `domain/services.go` | ✅ | ✅ `/bills/{id}/timeline` | ✅ migration 009 | ✅ `/bills/[id]/timeline` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/main_timeline_test.go` | #101 | #128 |
| Bill versions (immutable) | ADR-0011 | ✅ `domain/bill_version.go` | ✅ | ✅ `/bills/{id}/versions` | ✅ migration 009 + immutability trigger | ✅ `/bills/[id]/versions` | ⚠️ in-memory | ⚠️ unit only | ❌ | ✅ | ❌ | ⚠️ | ✅ ADR-0011 | TESTING | `services/legislation/internal/domain/bill_version.go` | ADR-0011 | #130 |
| Bill changes diff | §15 | ✅ `domain/services.go` | ✅ | ✅ `/bills/{id}/changes` | ✅ migration 009 | ✅ `/bills/[id]/compare` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/main_changes_test.go` | #102 | #130 |
| Bill AI summary | §76, §77 | ✅ Python | ⚠️ StubProvider default | ✅ `/bills/{id}/summary` | n/a | ✅ `bill-ask-panel.tsx` | ⚠️ AI svc unreachable → fallback | ⚠️ stub provider | ❌ | ❌ | ❌ | ⚠️ | ⚠️ | TESTING | `services/ai/app/capabilities/bill_summarizer.py` | #76 | #135 |
| Bill AI Q&A (Ask) | §74 J1 | ✅ Python | ⚠️ StubProvider default | ✅ `/questions` (auth) | n/a | ✅ `/ask`, `/bills/[id]/chat` | ⚠️ | ⚠️ stub provider | ❌ | ❌ | ❌ | ⚠️ | ⚠️ | TESTING | `services/ai/app/capabilities/civic_question_answerer.py` | #76 | #135 |
| Constitution | §74 J3 | ✅ `government/government.go` | ✅ | ✅ `/constitution`, `/constitution/articles` | ✅ migration 020 | ✅ `/constitution` | ⚠️ UI falls back to hardcoded data when API unreachable | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `adapters/kenya/kenya_seed/constitution.go` (20 curated articles / 9 chapters) | #212 | #244 |
| Constitution Spotlight | §74 J3 | ✅ | ✅ | ✅ | n/a | ✅ `components/constitution-spotlight.tsx` | ⚠️ | ❌ none | ❌ | n/a | ❌ | ⚠️ | ⚠️ | TESTING | No journey test | n/a | #189 |
| Government / administrations | §74 J4 | ✅ `government/government.go` | ✅ | ✅ `/governments` | ✅ migration 020 | ✅ `/governments/[id]` | ⚠️ UI falls back to hardcoded data | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/governments.go` | #192 | #200 |
| Presidential terms | §85 DoD | ✅ `government/government.go:PresidentialTerm` | ✅ | ✅ `/governments/{id}/terms` | ✅ migration 020 (EXCLUDE USING gist) | ✅ `/governments/[id]/terms/[term]` | ⚠️ | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `adapters/kenya/kenya_seed/government.go` (Kenyatta→Moi→Kibaki→Uhuru→Ruto) | #192 | #200 |
| Acts of Parliament | §15, §74 J5 | ✅ `domain/post_assent.go` | ✅ `act_repository.go` | ✅ `/acts` | ✅ migrations 010, 021 | ✅ `/acts`, `/acts/[id]` | ⚠️ in-memory | ⚠️ unit + 1 e2e page-load | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/legislation/internal/infrastructure/memory/act_repository.go` | #202 | #236 |
| Post-assent lifecycle audit | §15 | ✅ `domain/post_assent.go:AuditForAct` | ✅ delegates to domain logic | ✅ `/acts/{id}/audit` | ✅ migration 021 | ✅ `/acts/[id]/audit` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/post_assent.go` — NOT_VERIFIED reported for missing commencement date | #215 | #243 |
| Follow-a-Law | §74 J2 | ✅ `SubscriptionStore` | ✅ creates real persisted subscription | ✅ `/acts/{id}/follow` (auth) | ✅ migration 021 (events table) | ✅ `/acts/[id]/follow-button.tsx` | ⚠️ in-memory store (no Postgres) | ⚠️ unit only | ❌ | ⚠️ auth enforced (401 for anonymous) | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/subscriptions.go`; `subscriptionStore` shared with `/api/v1/subscriptions` | #216 | #243 |
| Act lineage | §15 | ✅ `buildLineage(act, events)` | ✅ data-driven | ✅ `/acts/{id}/lineage` | ✅ migration 021 | ✅ `/acts/[id]/lineage` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | Missing steps reported as `NOT_VERIFIED`, never inferred | #217 | #243 |
| Public Debt dashboard | §74 J6 | ✅ `domain/public_debt.go` | ✅ | ✅ `/debt` | ✅ migrations 017, 020 | ✅ `/debt`, `debt-trend-chart.tsx` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/legislation/internal/domain/public_debt.go` (12 CBK observations, 10 agreements) | #203 | #237 |
| Loans register | §74 J6 | ✅ `DebtRepository` | ✅ | ✅ `/debt/loans` | ✅ migration 017 | ✅ `/loans` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/public_debt.go` — `attribution_warning` field on mismatch | #220 | #239 |
| Grants register | §74 J6 | ✅ `DebtRepository` | ✅ | ✅ `/grants` | ✅ migration 017 | ✅ `/grants` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/public_debt.go` | n/a | #237 |
| Borrowing attribution validation | §85 DoD | ✅ `ValidateAttribution(agreement, []Administration)` | ✅ | ✅ `attribution_warning` field | n/a | ✅ surfaced on loan cards | n/a | ⚠️ unit only | ❌ | n/a | ❌ | ⚠️ | ✅ | TESTING | `domain/public_debt.go` — verifies agreement's administration matches ContractDate administration | #221 | #239 |
| Debt graphs | §85 DoD | ✅ | ✅ | ✅ `/debt/timeline` | ✅ migration 017 | ✅ `debt-trend-chart.tsx` | ⚠️ | ⚠️ unit only | ❌ | n/a | ❌ | ⚠️ | ✅ | TESTING | `apps/web/src/components/debt-trend-chart.tsx` (pre-existing exhaustive-deps lint warning) | n/a | #200 |
| Reconciliation | §85 DoD | ❌ no logic found | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | n/a | n/a | n/a | ❌ | DISCOVERED | No reconciliation logic exists in the repository | n/a | — |
| Trending Bills | §74 J1 | ✅ | ✅ | ✅ `/trending` | n/a | ✅ `/trending`, `trending-carousel.tsx` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/main.go:makeTrendingHandler` | n/a | #125 |
| Terminology | §74 J1 | ✅ `internal/terminology.go` | ✅ | ✅ `/terminology` | ✅ seed 004 | ✅ integrated | n/a | ⚠️ unit only | ❌ | n/a | ❌ | ⚠️ | ✅ | TESTING | `adapters/kenya/internal/terminology.go` | n/a | #125 |
| Search | §74 J1, §76 | ❌ stub | ❌ `TODO: call search service` | ⚠️ returns `{items: [], total: 0, note: "Search — pending issue #45"}` | ✅ migration 015 | ✅ `/search` | ❌ | ❌ none | ❌ | n/a | ❌ | ⚠️ | ⚠️ | IN_PROGRESS | `services/api/cmd/main.go:814` | #45 | — |
| Civic Feed (What Changed) | §74 J2 | ✅ What Changed engine | ✅ | ✅ `/feed`, `/what-changed` | n/a | ✅ `/feed`, `/what-changed` | ⚠️ in-memory | ⚠️ unit only | ❌ | n/a | ❌ | ⚠️ | ✅ | TESTING | Phase 14 engine | n/a | #187 |
| Briefing | §74 J1 | ❌ stub | ❌ | ⚠️ returns `{items: [], note: "Briefing — pending issue #40"}` | n/a | ✅ `/briefing` | ❌ | ❌ none | ❌ | n/a | ❌ | ⚠️ | ⚠️ | IN_PROGRESS | `services/api/cmd/main.go:830` | #40 | — |
| Notifications | §74 J2 | ⚠️ in-memory store | ⚠️ | ✅ `/notifications` (auth) | ✅ migration 014 | ✅ `/notifications` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ auth enforced | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/notifications.go` | n/a | #110 |
| Following / subscriptions | §74 J2 | ⚠️ in-memory store | ⚠️ | ✅ `/subscriptions`, `/follow` (auth) | ✅ migration 014 | ✅ `/following` | ⚠️ in-memory (shared with `/acts/{id}/follow`) | ⚠️ unit only | ❌ | ⚠️ auth enforced | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/subscriptions.go` | #110 | #133 |
| Trust / Provenance | §74 J3 | ⚠️ in-memory store | ⚠️ | ✅ `/provenance`, `/evidence`, `/claims`, `/contradictions`, `/sources` | ✅ migration 018 (immutability triggers) | ✅ `/trust` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/trust.go` | #165 | #176 |
| Corrections | §74 | ⚠️ in-memory store | ⚠️ | ✅ `/corrections` (public submit + admin review) | ✅ migration 018 | ✅ `/report` | ⚠️ in-memory | ⚠️ unit only | ❌ | ⚠️ public submit + admin review | ❌ | ⚠️ | ✅ | TESTING | `services/api/cmd/corrections.go` (audit trail) | #166 | #177 |
| Scenarios (Phase 18 simulation) | §33-34 | ✅ `services/simulation/` (RealityLayer tagging) | ✅ 3 engines (deterministic, Monte-Carlo, counterfactual) | ✅ 12 endpoints under `/scenarios` | ✅ migration 019 (11 tables, immutability triggers) | ✅ `/scenarios/…` (every page surfaces HYPOTHETICAL disclaimer) | ⚠️ in-memory | ✅ golden dataset (11 categories) + unit tests | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `services/simulation/internal/golden/golden.go`; reality-layer disclaimers enforced; constraint evaluation #210 | #190, #191, #209, #210 | #187, #189, #240 |
| Citation validation | §77, §86 | ✅ `services/ai/app/citation_validator.py` | ✅ | n/a (called inline by AI gateway) | n/a | n/a | n/a | ✅ unit | ❌ | n/a | n/a | ⚠️ | ✅ | TESTING | `services/ai/app/citation_validator.py` — responses with unsupported claims fail validation | n/a | #135 |
| Contradiction engine | ADR-0013 | ✅ `services/ai/app/capabilities/contradiction_detector.py` | ⚠️ | n/a | n/a | n/a | n/a | ⚠️ unit | ❌ | n/a | n/a | ⚠️ | ✅ ADR-0013 | TESTING | ADR-0013 — does not silently resolve contradictions | n/a | #135 |
| Country adapter — Kenya | §74 J1, §85 | ✅ full implementation | ✅ | ✅ 11 Bill stages per 2010 Constitution, 30 parliamentary terms, 14 committees | ✅ migration 009 + seed 002 | ✅ `/country/kenya` | ✅ real crawling of kenyalaw.org + parliament.go.ke | ✅ contract + parser tests | ❌ | ⚠️ | ❌ | ⚠️ | ✅ | TESTING | `adapters/kenya/` (parliament, kenya_law, gazette, president, kenya_seed) | n/a | #92, #97 |
| Country adapter — Uganda | §85 | ⚠️ stub | ⚠️ `TODO: crawl parliament.go.ug/bills` | n/a | n/a | ⚠️ | ❌ | ⚠️ contract | ❌ | n/a | n/a | n/a | ⚠️ | IN_PROGRESS | `adapters/uganda/parliament/parliament.go` | #48 | #149 |
| Country adapter — Tanzania | §85 | ⚠️ stub | ⚠️ `TODO(issue #154)` | n/a | n/a | ⚠️ | ❌ | ⚠️ contract | ❌ | n/a | n/a | n/a | ⚠️ | IN_PROGRESS | `adapters/tanzania/parliament/parliament.go` | #154 | #159 |
| Country adapter — Ghana | §85 | ⚠️ stub | ⚠️ `TODO: crawl parliament.gh/bills` | n/a | n/a | ⚠️ | ❌ | ⚠️ contract | ❌ | n/a | n/a | n/a | ⚠️ | IN_PROGRESS | `adapters/ghana/parliament/parliament.go` | #50 | #158 |
| Country adapter — Nigeria | §85 | ⚠️ stub | ⚠️ `TODO(issue #CI-AD-NG-001)` (placeholder ID) | n/a | n/a | ⚠️ | ❌ | ⚠️ contract | ❌ | n/a | n/a | n/a | ⚠️ | IN_PROGRESS | `adapters/nigeria/parliament/parliament.go` | (placeholder ID) | #160 |
| Country adapter — South Africa | §85 | ⚠️ stub | ⚠️ `TODO(issue #CI-AD-ZA-001)` (placeholder ID) | n/a | n/a | ⚠️ ships `za-bill-placeholder-1..4` | ❌ | ⚠️ contract | ❌ | n/a | n/a | n/a | ⚠️ | IN_PROGRESS | `adapters/south_africa/parliament/parliament.go`; `apps/web/src/app/country/south-africa/page.tsx:14-44` placeholder bills | (placeholder ID) | #161 |
| Authentication (OIDC) | §85 DoD | ⚠️ JWKS fetched but signature NOT verified | ⚠️ `DevVerifier` trusts claims if token parses | ✅ middleware (OptionalAuth, RequireToken, RequireScope, RateLimit) | ✅ migration 003 (users, sessions, roles, permissions) | n/a | ⚠️ `DevMode` default `true` in `cmd/main.go:35` → `DevVerifier` is runtime verifier | ⚠️ 8 unit tests pass | ❌ | ❌ BLOCKER | n/a | n/a | ⚠️ | ✅ | BLOCKED | `services/api/internal/oidc/verifier.go:126` — `TODO(issue #59): verify the signature using the JWKS key matching header.Kid` | #59 | #90 |
| Authorization (RBAC) | §85 DoD | ✅ 4 roles + 9 permissions | ✅ `RequireScope` middleware | ✅ | ✅ migration 016 (seed roles + permissions) | n/a | ⚠️ | ⚠️ unit only | ❌ | ✅ | ❌ | ⚠️ | ✅ | TESTING | `services/api/internal/middleware/auth.go` | n/a | #90 |
| Rate limiting | SECURITY.md | ⚠️ documented | ⚠️ single global `RateLimit(300, time.Minute)` | ⚠️ no per-tier, no per-user, no AI-endpoint tightening | n/a | n/a | ⚠️ no rate-limit middleware in mux chain | ❌ none | ❌ | n/a | ❌ | ⚠️ | ✅ | IN_PROGRESS | `services/api/cmd/main.go:220` — single global limit | n/a | #90 |
| Observability — metrics | §85 DoD | ⚠️ 5 of 30+ metrics implemented | ✅ `/metrics` (Prometheus) | n/a | n/a | n/a | ✅ unit | ❌ | n/a | n/a | ✅ self | ✅ | IN_PROGRESS | `packages/observability/logger.go:229-235` — only `civic_api_requests_total`, `civic_api_request_latency_seconds`, `civic_bills_discovered_total`, `civic_adapter_errors_total`, `civic_ai_questions_total`, `civic_ai_citation_failures_total` | n/a | #95 |
| Observability — tracing | §85 DoD | ⚠️ `NopTracer` only | ⚠️ "in production, replace with OTel SDK" | n/a | n/a | n/a | ⚠️ | ✅ unit | ❌ | n/a | n/a | ✅ self | ✅ | IN_PROGRESS | `packages/observability/tracer.go` | n/a | #95 |
| Grafana dashboards | §85 DoD | n/a | n/a | n/a | n/a | n/a | n/a | ❌ none | ❌ | n/a | n/a | ✅ self | ⚠️ | TESTING | 4 JSON files in `infrastructure/observability/grafana-dashboards/`; doc (`09-observability.md:109`) claims they live in `infrastructure/observability/dashboards/` — directory-name mismatch | #55 | n/a |
| Payments — M-Pesa | §85 | ⚠️ handler returns OK without calling Daraja API | ⚠️ | ✅ `/sponsor/mpesa` | n/a | ✅ `/sponsor` | ❌ Daraja API not wired | ❌ none | ❌ | ⚠️ | ❌ | ⚠️ | ⚠️ | IN_PROGRESS | `services/api/cmd/main.go:handleMpesaSponsor` | n/a | #148 |
| Payments — Stripe | §85 | ❌ stub | ❌ returns placeholder URL | ✅ `/sponsor/card` | n/a | ✅ | ❌ | ❌ none | ❌ | ⚠️ | ❌ | ⚠️ | ⚠️ | IN_PROGRESS | `main.go` — `checkout_url = "https://checkout.stripe.com/c/pay/cs_test_placeholder_…"` | n/a | #148 |
| PDF / DOCX extraction | §74 J1 | ❌ stubs | ❌ `PDFExtractor is a stub; OCR may be needed`; `DOCXExtractor is a stub; production should use unioffice` | n/a | n/a | n/a | ❌ | ❌ none | ❌ | n/a | n/a | n/a | ⚠️ | IN_PROGRESS | `services/documents/internal/infrastructure/extractors/text.go` — `pdf-v1-stub`, `docx-v1-stub` | n/a | — |
| Postgres migrations | §85 DoD | n/a | n/a | n/a | ✅ 21 migrations across 9 schemas | n/a | n/a | ❌ no migration-roundtrip tests | ❌ | n/a | n/a | n/a | ⚠️ | TESTING | `infrastructure/postgres/migrations/001-021` (forward-only, paired up+down, idempotent, immutability triggers, EXCLUDE USING gist, btree_gist extension) | #211, #214, #228 | #235 |
| Documentation — OpenAPI | §85 DoD | n/a | n/a | ✅ 60+ paths documented | n/a | n/a | n/a | ❌ no schema-validation test | n/a | n/a | n/a | n/a | ✅ | TESTING | `docs/api/openapi.yaml` (1,838 lines) | n/a | n/a |
| Documentation — ADRs | §85 DoD | n/a | n/a | n/a | n/a | n/a | n/a | n/a | n/a | n/a | n/a | n/a | ✅ | TESTING | 14 ADRs in `docs/adr/` (ADR-0001 through ADR-0014, MADR format) | n/a | n/a |
| Golden user journeys (8) | §74 | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ 0 of 8 journeys tested | ❌ | n/a | n/a | n/a | ❌ | DISCOVERED | `tests/e2e/homepage.spec.ts` only checks `<h1>` text on 8 individual pages; no journey test clicks through | P0-1 (audit-team-5) | — |
| Critical failure test | §75 | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | n/a | n/a | n/a | ❌ | DISCOVERED | No worker-crash + retry-chain test exists anywhere in the repo (`rg "critical.failure\|CriticalFailure"` returns 0 hits) | P0-2 (audit-team-5) | — |
| Integration tests | §81 | — | — | — | — | — | — | ❌ no `services/*/integration/` directories | ❌ | n/a | n/a | n/a | ⚠️ doc claims they exist | DISCOVERED | `docs/architecture/10-testing.md:23` claims `services/<svc>/integration/`; no such dirs exist | P1-5 (audit-team-5) | — |
| Contract tests (HTTP/events) | §81 | — | — | — | — | — | — | ❌ only 6 adapter contract_test.go files; no event/HTTP producer-consumer contract tests | ❌ | n/a | n/a | n/a | ⚠️ | DISCOVERED | `packages/contracts/` has no test files | n/a | — |
| E2E tests | §81 | — | — | — | — | ✅ minimal | — | ⚠️ 2 specs (homepage, accessibility); Playwright NOT installed (`@playwright/test` + `@axe-core/playwright` not in devDependencies) | ❌ | n/a | n/a | n/a | ⚠️ | TESTING | `tests/e2e/homepage.spec.ts`, `tests/e2e/accessibility.spec.ts`; 4 of 47 routes covered | P0-4 (audit-team-5) | #96 |
| Performance / load tests | §81 | — | — | — | — | — | — | ❌ no k6/Locust/Benchmark files anywhere in repo | ❌ | n/a | n/a | n/a | ❌ | DISCOVERED | `rg "benchmark\|k6\|locust\|stress\|soak"` returns 0 hits in code | P1-7 (audit-team-5) | — |
| Chaos tests | §81 | — | — | — | — | — | — | ❌ | ❌ | n/a | n/a | n/a | ❌ | DISCOVERED | None | P1-8 (audit-team-5) | — |
| Disaster recovery tests | §81 | — | — | — | — | — | — | ❌ no backup/restore runbook | ❌ | n/a | n/a | n/a | ❌ | DISCOVERED | None | P1-9 (audit-team-5) | — |
| AI evaluation dataset | §81 | ✅ | — | — | — | — | — | ✅ | ❌ | n/a | n/a | n/a | ⚠️ | TESTING | `services/ai/eval/test_eval_dataset.py` — but eval scores reflect StubProvider, not a real LLM | P1-14 (audit-team-5) | #135 |

---

## 3. Reading the matrix

- **Zero rows are at `VERIFIED`.** Per spec §78: *"Only VERIFIED counts as complete."* From the audit's perspective, **zero features are complete**.
- The "Tests" column almost universally says "⚠️ unit only" — confirming the §77 violation: a passing unit test is NOT sufficient evidence for an end-to-end feature.
- Three rows are `IN_PROGRESS` because the underlying implementation itself is a stub: **Search** (#45), **Briefing** (#40), **Stripe checkout** (placeholder URL). The repo currently claims in `services/api/README.md:15` that *"Implemented: all endpoints below (40+)"* — that claim violates §77 and must be qualified. See `docs/NO_FAKE_COMPLETION.md` §7.
- Six rows are `BLOCKED` or `DISCOVERED` because the implementation has a critical defect or is entirely missing:
  - **OIDC authentication** — BLOCKED on #59 (signature verification).
  - **Reconciliation** — DISCOVERED (no logic exists).
  - **Golden user journeys** — DISCOVERED (0 of 8 mandated §74 journeys tested).
  - **Critical failure test** — DISCOVERED (no §75 test exists).
  - **Integration / contract / performance / chaos / disaster-recovery tests** — DISCOVERED (no test files exist).
- Country adapters for Uganda, Tanzania, Ghana, Nigeria, South Africa are all `IN_PROGRESS` because each ships only contract tests + a `TODO: crawl …` stub. The Kenya adapter is the only `TESTING`-grade country adapter (real crawling + parser tests).

---

## 4. How to update this matrix

1. When a feature moves from `DISCOVERED` → `IN_PROGRESS`, update the row's Status column AND fill the Evidence column with the file path / issue that proves the partial implementation.
2. When a feature reaches `IMPLEMENTED`, all 14 DONE criteria in `docs/NO_FAKE_COMPLETION.md` §2 must be satisfied. Update every column to ✅ (or ⚠️ with a documented reason).
3. When a feature reaches `VERIFIED`, independent QA (spec §73) must have signed off. The Evidence column must cite the QA report.
4. When a feature is `BLOCKED`, the Issue column MUST cite a real GitHub issue (not a placeholder `#CI-AD-NNN` string).
5. This file is the source of truth for which features are actually done. README badges, CHANGELOG entries, and `services/api/README.md` claims must be consistent with this matrix (spec §77, §86).

---

## 5. Cross-references

- `MASTER_AUDIT.md` — top-level audit; references this matrix.
- `docs/NO_FAKE_COMPLETION.md` — the rule that governs every status assignment in this matrix.
- `docs/PRODUCTION_GATE.md` — the per-gate checklist that must pass before any row can move to `VERIFIED`.
- `audit-team-5-report.md` §5 — the audit-team-5 draft matrix that this document formalises.
- `audit-team-{1,2,3,4}-report.md` — per-section audits that ground each row's status.

---

*This matrix is governed by spec §78. **Only `VERIFIED` counts as complete.***
