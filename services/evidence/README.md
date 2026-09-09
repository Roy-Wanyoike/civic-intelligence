# evidence service

The **evidence** service owns the platform's claim/citation/evidence graph.
It records what supports each claim made by AI (or humans), validates that
AI responses only make claims backed by cited source text, and surfaces
source conflicts for human review.

## Bounded context

Owns:
- Claim, Citation, Evidence, EvidenceSet
- SourceReference (provenance of a piece of evidence)
- SourceConflict (when two sources disagree)

Does NOT own:
- Documents (→ documents)
- AI explanations (→ intelligence)
- Bills / Acts (→ legislation)
- Notifications (→ notifications)

## Architectural rules enforced by this service

1. **CitationValidator never silently accepts a claim with no evidence.**
   If an AI response makes a claim with no supporting citation, the claim
   is flagged in the Missing list and an `ai.validation.failed` event is
   published.
2. **SourceConflictDetector never silently resolves.** When two citations
   disagree on a claim's value, a SourceConflict record is created with
   resolution "open" and surfaced for human review.
3. **The service has no DB drivers in domain.** All SQL lives in
   internal/infrastructure/postgres.
4. **EvidenceService.AttachClaim is the only way to persist a claim+citation
   pair.** It atomically records the claim, persists every citation, and
   computes support/refute counts.
