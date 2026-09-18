# No-Fake-Completion Rule

**Source:** Spec §77 (`upload/Pasted Content_1789720147692.txt:2480-2499`)
**Cross-references:** Spec §78 (Completion Matrix), §79 (Final Re-Audit), §81 (Final Production Gate), §85 (Definition of Done), §86 (The Final Rule)
**Audited by:** `audit-team-5-report.md` §3.5 — *"No-fake-completion rule (§77) — not documented anywhere"* (P1-4)
**Status:** ACTIVE — applies to every commit, every PR, every release gate.

---

## 1. The rule

Never mark:

* "implemented"
* "complete"
* "production ready"
* "verified"
* "tested"

unless actual evidence supports that claim.

A passing unit test is not sufficient evidence for an end-to-end feature.
A passing E2E test is not sufficient evidence for production resilience.
A working UI is not sufficient evidence for data correctness.
A correct database schema is not sufficient evidence for user functionality.

---

## 2. When is a feature actually DONE?

A feature is only **DONE** when ALL of the following exist and have been demonstrated to work end-to-end:

1. **Architecture exists** — the feature has an architecture decision record (ADR) or architecture-doc section that names the components, the boundaries, the dependency direction, and the failure modes.
2. **Domain model exists** — Go/Python/TS types are declared with explicit fields, enums, and invariants; they are NOT free-text strings; they are NOT `interface{}` buckets.
3. **Backend logic exists** — a service / use case / handler actually executes the domain logic. No `// TODO` markers, no `return nil, nil` stubs, no "pending issue #N" placeholder responses.
4. **Database schema exists** — a forward-only Postgres migration creates the tables, indexes, FK constraints, CHECK constraints, and (where required) immutability triggers. The migration has a paired `down` script and is idempotent.
5. **API exists** — the endpoint is registered in the router (`mux.HandleFunc` / Next.js route / FastAPI route), is documented in `docs/api/openapi.yaml`, returns the documented status codes, and is exercised by at least one HTTP-level test.
6. **Frontend UX exists** — the route renders real data fetched from the API; no `mock-data.ts` fallback in production paths; no placeholder Bills / placeholder articles / placeholder loan records.
7. **Integrations exist** — every external dependency the feature touches (Parliament / Kenya Law / CBK / OpenAI / Anthropic / Stripe / Daraja / Postgres / NATS / Temporal / Redis / S3) is wired to a real client; no `StubProvider` in the production path; no `cs_test_placeholder_…` URLs.
8. **Evidence / provenance exists** — every factual claim the feature makes is traceable to an authoritative source via the evidence system (`/api/v1/evidence`, `/api/v1/provenance`). AI-proposed candidate facts cannot directly mutate canonical legislative state (ADR-0005).
9. **Tests exist** — at minimum: unit tests for the domain logic, integration tests for the service boundary, contract tests for any cross-service API, E2E test for the user-facing flow. A single passing unit test is not sufficient.
10. **QA verified** — independent QA (spec §73) has signed off. Engineering's "tests pass" is not QA verification.
11. **Security reviewed** — `SECURITY.md` threat model has been consulted; the feature's attack surface has been reviewed; OIDC / RBAC / rate-limiting / SSRF allowlist apply; no `TODO(issue #59): verify the signature` markers.
12. **Performance validated** — at least one benchmark / load test exercises the feature's hot path under expected production load. No N+1 queries; no missing indexes on frequently-filtered columns; no unbounded result sets.
13. **Observability exists** — the feature emits at least one metric from the catalog in `docs/architecture/09-observability.md`; traces propagate a correlation ID end-to-end; logs carry the structured fields required by `packages/observability/logger.go`.
14. **Documentation exists** — README / ARCHITECTURE / ADR / OpenAPI / CHANGELOG / `docs/architecture/*` are updated. The CHANGELOG entry explicitly states what was added, what changed, what is still in progress, and what is blocked.

If any of the 14 above is missing, the feature is `IN_PROGRESS` (or `BLOCKED`), **not** `IMPLEMENTED`, **not** `VERIFIED`, **not** "done".

---

## 3. Forbidden patterns

The following patterns, if present in a code path that has been claimed as "implemented", constitute a §77 violation:

### 3.1 Code markers

| Pattern | Meaning | Action |
|---------|---------|--------|
| `TODO` / `FIXME` / `XXX` / `HACK` | Implementation deferred | Convert to a tracked GitHub issue per §82; mark the feature `IN_PROGRESS` in `docs/COMPLETION_MATRIX.md` |
| `placeholder` / `stub` / `mock` / `fake` (in production code paths) | Not real data | Replace with real implementation OR gate behind a `DevMode` flag that is `false` in production |
| `hardcoded production value` | e.g. `checkout_url = "https://checkout.stripe.com/c/pay/cs_test_placeholder_…"` | Replace with real client call |
| `commented-out implementation` | The implementation exists but is disabled | Either delete the comment and ship the code, or delete the comment and create an issue |
| `return nil, nil` / `return []` / `return ""` in a handler | Empty success response masquerading as real data | Wire to a real repository; if there is genuinely no data yet, return a structured `404` or `200 + count=0 + note` |
| `// pending issue #N` | Documented deferral | Convert to a real GitHub issue (some `#CI-AD-NNN` IDs in the repo are placeholder strings, not real issues — see audit-team-5 §2) |
| `DevMode = true` default in production config | Bypasses security controls | Set `DevMode = false` in production; gate dev-only code paths behind the flag |

### 3.2 Documentation patterns

| Pattern | Meaning | Action |
|---------|---------|--------|
| `services/api/README.md:15` says *"Implemented: all endpoints below (40+)"* without distinguishing stubbed ones | §77 violation — does not distinguish `IMPLEMENTED` from `IN_PROGRESS` | Update to enumerate which endpoints are at which status; reference `docs/COMPLETION_MATRIX.md` |
| `README.md` badge says *"tests: 176 Go + 28 Python"* without evidence | Unverified claim | Reference the actual `go test ./...` and `pytest` output; link to CI artifacts |
| `CHANGELOG.md` lists a feature as "Added" without mentioning it is `IN_PROGRESS` | §77 violation | Add a "Known limitations" subsection or an `IN_PROGRESS` tag |
| `docs/architecture/13-roadmap.md` says Phase N is "in progress" while Phase N+1 has merged | Documentation lags implementation | Refresh the roadmap |

### 3.3 Test patterns

| Pattern | Meaning | Action |
|---------|---------|--------|
| `t.Run("RejectsNonReadyScenario", func(t *testing.T) { /* always passes */ })` | Tautological test (audit-team-5 #207) | Rewrite to assert the actual rejection condition |
| Unit test exists but no integration / E2E test | §77 violation — unit test alone is not sufficient | Add integration + E2E tests |
| E2E test exists but does not click through a journey | §74 violation — page-load assertions are not journey tests | Add journey tests (spec §74 mandates 8) |
| Tests pass but no failure-injection test exists | §75 violation | Add critical failure test (worker crash + retry chain) |

---

## 4. Engineer sign-off checklist

Before declaring a feature DONE, the engineer responsible must sign off the following in the PR description (the PR template at `.github/PULL_REQUEST_TEMPLATE.md` is the place for this):

```
- [ ] Architecture: <ADR-XXXX or docs/architecture/NN-name.md#section>
- [ ] Domain model: <file:line>
- [ ] Backend logic: <file:line> — no TODO/FIXME/stub markers in the changed code
- [ ] Database schema: <migration file> — paired up+down, idempotent, FK + CHECK constraints
- [ ] API: <route path> — registered in router, documented in openapi.yaml, has HTTP-level test
- [ ] Frontend UX: <route path> — fetches from API, no mock-data.ts fallback in production path
- [ ] Integrations: <list of external clients> — all wired to real clients, no stub providers
- [ ] Evidence/provenance: <evidence system path> — every claim traceable to authoritative source
- [ ] Tests: unit <count> + integration <count> + contract <count> + E2E <count>
- [ ] QA verified: <independent QA owner> signed off on <date>
- [ ] Security reviewed: <security-reviewer> signed off; threat-model consulted
- [ ] Performance validated: <benchmark file> or <load test result>
- [ ] Observability: <metric name(s) added> from the catalog in 09-observability.md
- [ ] Documentation: README / ARCHITECTURE / ADR / OpenAPI / CHANGELOG updated
- [ ] COMPLETION_MATRIX.md row updated to status = IMPLEMENTED (or VERIFIED if QA passed)
```

If any box cannot be checked honestly, the feature is `IN_PROGRESS` — not `IMPLEMENTED`, not `VERIFIED`, not "done". Mark it as such in `docs/COMPLETION_MATRIX.md` and create a follow-up issue per §82.

---

## 5. Status vocabulary (spec §78)

The only allowed status values for a feature (in `docs/COMPLETION_MATRIX.md` and in any internal claim):

```
DISCOVERED      — spec only, no code exists yet
IN_PROGRESS     — partial implementation; at least one of the 14 DONE criteria is missing
IMPLEMENTED     — all 14 DONE criteria are satisfied; QA has NOT yet signed off
TESTING         — implementation complete; tests written; QA NOT yet signed off
QA              — independent QA is actively verifying
BLOCKED         — cannot satisfy one or more criteria until an external dependency lands
VERIFIED        — independent QA has signed off; the only status that counts as "complete" (§78)
```

**Only `VERIFIED` counts as complete.** A row at `IMPLEMENTED` is NOT complete. A row at `TESTING` is NOT complete. The matrix in `docs/COMPLETION_MATRIX.md` is the source of truth for which features are actually done.

---

## 6. Audit enforcement

The audit cycle (spec §71-86) enforces this rule as follows:

1. **§71 Issue management** — every `TODO`/`FIXME`/`stub`/`placeholder` discovered by the audit becomes a tracked GitHub issue (no placeholder `#CI-AD-NNN` strings).
2. **§73 Independent QA** — engineering's claim of `IMPLEMENTED` is verified by independent QA; only QA's sign-off promotes a row to `VERIFIED`.
3. **§77 This rule** — the rule itself (this document).
4. **§78 Completion matrix** — `docs/COMPLETION_MATRIX.md` records the status of every feature; `VERIFIED` is the only terminal status.
5. **§79 Final re-audit** — the audit cycle re-runs after every implementation wave; the four comparison axes (Requirement ↔ Repository ↔ Runtime ↔ Automated Tests ↔ Independent QA ↔ Production Failure Tests) must all agree.
6. **§81 Final production gate** — `docs/PRODUCTION_GATE.md` records the pass/fail/partial status of every gate; any failure means the audit continues.
7. **§82 Zero-pending-work** — no `TODO`/`stub`/`placeholder` may be silently abandoned; each must be a tracked issue with owner + acceptance criteria.
8. **§86 The Final Rule** — *"Do not optimize for declaring the project finished. Optimize for proving that the project works."*

---

## 7. Wave 5 violations identified by audit-team-5

The audit-team-5 report (§3.5, §7) enumerates the following §77 violations found in the repository at HEAD `dca1c78`. These are tracked for remediation in subsequent waves:

| Violation | File:line | Issue |
|-----------|-----------|-------|
| `TODO(issue #59): verify the signature using the JWKS key` | `services/api/internal/oidc/verifier.go:126` | #59 |
| `TODO: call search service. For now, search through discovered bills.` | `services/api/cmd/main.go:814` | #45 |
| `Briefing — pending issue #40` | `services/api/cmd/main.go:830` | #40 |
| `Anthropic provider not implemented in this iteration — fall back to stub` | `services/ai/app/gateway.py` | (no issue) |
| `PDFExtractor is a stub; OCR may be needed` | `services/documents/internal/infrastructure/extractors/text.go` | (no issue) |
| `DOCXExtractor is a stub; production should use unioffice` | `services/documents/internal/infrastructure/extractors/text.go` | (no issue) |
| `TODO(issue #CI-AD-NG-001)` Nigeria crawl | `adapters/nigeria/parliament/parliament.go` | (placeholder ID — not a real GitHub issue) |
| `TODO(issue #CI-AD-ZA-001)` South Africa HTML parser | `adapters/south_africa/parliament/parliament.go` | (placeholder ID) |
| `TODO: crawl parliament.go.ug/bills` | `adapters/uganda/parliament/parliament.go` | #48 |
| `TODO(issue #154)` Tanzania crawl | `adapters/tanzania/parliament/parliament.go` | #154 |
| `TODO: crawl parliament.gh/bills` | `adapters/ghana/parliament/parliament.go` | #50 |
| `Stripe checkout_url = "https://checkout.stripe.com/c/pay/cs_test_placeholder_…"` | `services/api/cmd/main.go` | (no issue) |

`services/api/README.md:15` claim *"Implemented: all endpoints below (40+)"* is also a §77 violation until each of the stubbed endpoints above is either implemented or marked `IN_PROGRESS` in the README + the completion matrix.

---

## 8. Cross-references

- `MASTER_AUDIT.md` — top-level audit; references this rule.
- `docs/COMPLETION_MATRIX.md` — the per-feature status matrix; only `VERIFIED` is terminal.
- `docs/PRODUCTION_GATE.md` — the per-gate checklist; any failure means the audit continues.
- `docs/architecture/10-testing.md` — test taxonomy (unit / integration / contract / E2E / golden journey / critical failure / AI eval).
- `docs/architecture/09-observability.md` — metrics catalog; every feature must emit at least one metric from this catalog.
- `docs/adr/ADR-0005-ai-cannot-mutate-truth.md` — the "AI may PROPOSE candidate facts but may NEVER directly write to canonical legislative state" rule, which this §77 rule operationalises.
- `audit-team-5-report.md` §3.5 — the audit finding that triggered this document (P1-4).
- `CONTRIBUTING.md` — the PR checklist engineers must follow.

---

*This document is governed by spec §77. It is the rule, not a suggestion.*
