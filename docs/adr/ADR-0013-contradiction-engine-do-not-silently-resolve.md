# ADR-0013: Contradiction engine — do not silently resolve

## Status

Accepted — 2026-09-09

## Context

Authoritative sources sometimes conflict. Example: the Parliament Bill Tracker says a Bill is at "Committee Stage", but the official Order Paper lists it for "Second Reading" the next day. Options:

1. **Silently pick the more authoritative source** — fast, but destroys information and may be wrong.
2. **Show the conflict to the user** — transparent, but may be confusing.
3. **Record the conflict, apply a resolution strategy, expose the conflict to editors** — balanced.

## Decision

Adopt **option 3**: never silently resolve. Always record a `SourceConflict`:

- `claim_id` — what's being asserted.
- `source_a_id` + `source_a_value` + `source_a_retrieved_at`.
- `source_b_id` + `source_b_value` + `source_b_retrieved_at`.
- `resolution_strategy` — one of: `prefer_authoritative`, `prefer_more_recent`, `prefer_event_date_over_publication_date`, `prefer_committee_report_over_tracker`, `manual_review_required`, or NULL.
- `resolved_at` — when (if) the conflict was resolved.

The platform surfaces the conflict to editors via the admin UI; citizens see the most likely value with a "verified at" timestamp.

## Consequences

- **Positive**: nothing is silently lost — every conflict is auditable.
- **Positive**: editors can review + correct, improving data quality over time.
- **Positive**: citizens are not misled by a single source's mistake.
- **Negative**: more work for editors when conflicts arise — mitigated by sensible default resolution strategies for high-frequency conflict types (e.g., Parliament vs. Senate trackers on the same Bill).

## References

- ARCHITECTURE.md §12 (Evidence system)
- `services/ai/app/capabilities/contradiction_detector.py`
- `infrastructure/postgres/migrations/012_evidence.up.sql` (`evidence.source_conflicts` table)
