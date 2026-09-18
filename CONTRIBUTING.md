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

## PR checklist

- [ ] Linked issue (`Closes #NNN` or `Refs #NNN`)
- [ ] Tests added / updated
- [ ] Docs updated if needed (README, ARCHITECTURE, ADR)
- [ ] If touching AI: eval dataset passes; citation validator flags 0 unsupported claims on sample set
- [ ] If touching `legislation.*`: audit trigger fires correctly
- [ ] No Kenya-specific strings outside `adapters/kenya/`
- [ ] No infrastructure imports in any `internal/domain/` package
- [ ] No direct writes from AI to `legislation.*`

## How to add a new country adapter

A country adapter MUST have all 7 of these before it can be merged:

1. **Authoritative sources** — a `SourceRegistry` entry per source URL with `authority_level` and crawl frequency.
2. **Legislative ontology** — the country's institutions, houses, committees mapped into the global `Institution`/`Legislature`/`House`/`Committee` model.
3. **Stage definitions** — every Bill stage with code, name, simple_explanation, allowed_next, is_terminal. Stored as `legislation.bill_stages` rows with `country_id`.
4. **Document parsers** — HTML/PDF/DOCX parsers for each source's document type.
5. **Terminology** — at least 25 parliamentary terms with plain-language explanations + source URLs.
6. **Tests** — contract tests proving the adapter satisfies `contracts.LegislativeSourceAdapter`; stage transition validity tests; terminology completeness tests.
7. **Data-quality rules** — duplicate detection, date sanity, stage-transition validity, cross-source consistency.

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
- If a PR adds Kenya-specific strings to global domain code, block it.
- If a PR's AI eval results show regression, block it.

## Local development

See [README.md](./README.md) Quickstart.

## Reporting security issues

See [SECURITY.md](./SECURITY.md). Do NOT open a public issue for security vulnerabilities.

## Code of Conduct

See [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md). Be kind. This is civic infrastructure.
