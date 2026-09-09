## Summary
<!-- One paragraph: what & why. -->

## Linked issue
<!-- "Closes #NNN" / "Refs #NNN". Every meaningful PR must reference an issue. -->

## Architectural impact
<!-- Which bounded context does this touch? Which services? -->
- [ ] legislation
- [ ] ingestion
- [ ] documents
- [ ] evidence
- [ ] intelligence (Go)
- [ ] ai (Python)
- [ ] search
- [ ] notifications
- [ ] identity
- [ ] api (BFF)
- [ ] adapters/kenya
- [ ] apps/web
- [ ] infrastructure

## Did this AI output pass validation?
<!-- If this PR touches AI prompts, models, or RAG, paste the eval results. -->
- [ ] N/A (no AI change)
- [ ] Eval dataset passes locally
- [ ] Citation validator flags 0 unsupported claims on the sample set

## Checklist
- [ ] Tests added / updated
- [ ] Docs (README, ARCHITECTURE, ADR) updated if needed
- [ ] Migrations are forward-only (no editing merged migrations)
- [ ] No Kenya-specific strings introduced outside `adapters/kenya/`
- [ ] No `database/sql`, `net/http`, or NATS imports inside any `internal/domain` package
- [ ] No direct writes from AI to `legislation.*`
- [ ] No fabrications introduced (if AI, every claim is cited)

## Reviewer notes
<!-- Anything reviewers should focus on. -->
