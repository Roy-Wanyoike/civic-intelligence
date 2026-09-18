# ADR-0011: Immutable Bill versions

## Status

Accepted — 2026-09-09

## Context

Bills change over time. A Bill introduced in March may have a different text by September (committee amendments, public participation feedback,Report Stage changes). The platform MUST preserve every version immutably so:

- Citizens can see what changed between versions.
- Researchers can cite a specific version.
- The platform never silently overwrites history.

## Decision

`legislation.bill_versions` is **IMMUTABLE**:

- INSERT only — never UPDATE.
- Each row has a unique `content_hash` (SHA-256 of the source document's raw bytes).
- Each row has a `version_no` (monotonically increasing per Bill).
- One row per Bill has `is_current = TRUE` (enforced by a partial unique index).
- Promoting a new current version = transaction that sets the old row's `is_current = FALSE` and inserts a new row with `is_current = TRUE`.
- The original raw document is stored immutably in object storage (S3/MinIO) keyed by content hash.

## Consequences

- **Positive**: full legislative history is preserved — every version can be cited, compared, audited.
- **Positive**: comparing versions is exact (content-hash based) — no fuzzy text diffing needed for the "what changed" view.
- **Positive**: the audit log captures who promoted which version and when.
- **Negative**: storage cost grows linearly with version count — acceptable, since Bill text is small (~10s of KB per version).
- **Negative**: the "current version" pointer must be transactionally consistent — mitigated by the partial unique index `uq_bill_versions_current`.

## Compare versions

The `DocumentComparator` capability produces a diff between two versions:
- Additions: clauses in the new version not in the old.
- Removals: clauses in the old version not in the new.
- Modifications: clauses whose text changed.
- A plain-language explanation of the changes.
- **Never** an explanation of *why* a change was made unless the source explicitly states the reason.

## References

- ARCHITECTURE.md §1 (Non-negotiable principles — Version everything)
- `infrastructure/postgres/migrations/009_bills.up.sql` (bill_versions table + partial unique index)
- `services/ai/app/capabilities/document_comparator.py`
