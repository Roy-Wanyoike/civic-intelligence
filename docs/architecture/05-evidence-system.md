# 05 — Evidence System

This document describes the evidence system: the chain that connects every claim the platform makes to the specific page and section of the specific document that supports it, the citation validation flow, the "AI cannot mutate truth" rule in detail, and the contradiction engine.

The corresponding decision records are [ADR-0005](../adr/ADR-0005-ai-cannot-mutate-truth.md) (AI cannot mutate truth) and [ADR-0013](../adr/ADR-0013-contradiction-engine-do-not-silently-resolve.md) (contradiction engine does not silently resolve).

## The Claim → Evidence → Document → Page/Section → Source chain

Every factual assertion in the platform — "Bill X was introduced on date Y," "Senator Z voted aye on bill W," "this bill will increase the tax rate on category V from R1 to R2" — is a `Claim`. A claim without evidence is, by definition, not a claim the platform makes. The chain is:

```
Claim  ──has──▶  Evidence  ──cites──▶  Citation  ──resolves to──▶  DocumentPage/Section
                                                                    │
                                                                    │ belongs to
                                                                    ▼
                                                              SourceDocument
                                                                    │
                                                                    │ fetched from
                                                                    ▼
                                                                  Source
```

Walking the chain from any claim, a citizen (or a journalist, or a researcher, or an auditor) can arrive at the exact page of the exact document that backs the claim. This is the "evidence before AI" principle made operational: a click on a citation opens the cited page with the cited text highlighted. There is no claim without a click-through.

### Claim

A `Claim` is a single factual assertion: a typed statement with a subject, a predicate, and an object (in the RDF sense), or a structured record (in the relational sense). "Bill KE-2024-42 was introduced on 2024-03-15" is a claim. "Bill KE-2024-42 has sponsor Hon. Jane Doe" is a claim. Claims are immutable once accepted; if a claim is wrong, it is retracted (a `ClaimRetracted` event), not silently edited.

### Evidence

`Evidence` is the link between a `Claim` and one or more `Citation`s. A single claim may be backed by multiple pieces of evidence: a bill's introduction date is supported by the bill document itself (primary), the order paper for that date (primary), the Hansard for that date (primary), and the gazette notice of assent (primary, if it lists the introduction date as a back-reference). Multiple independent pieces of evidence strengthen the claim; a single weak piece weakens it.

### Citation

A `Citation` is a resolved pointer: `(document_id, page_number, section_id, quoted_text_range)`. At write time, the validator verifies that the document with that ID exists, has that page, has that section, and contains the quoted text. If any of those fail, the citation is rejected. If the citation passes, it is stored; if the underlying document is later re-parsed and the page/section shifts, the citation is flagged for re-validation.

### DocumentPage / DocumentSection

These are the addressable units of a parsed document (owned by `services/documents/`). A page is a logical page (PDF page or HTML viewport); a section is a structural unit (a clause, a paragraph, a schedule item). Both have stable identifiers that survive re-parsing (a clause's ID is based on its number and text hash, not its byte offset). This stability is what makes citations robust.

### SourceDocument and Source

A `SourceDocument` is a fetched artifact (a specific PDF of bill number X, fetched at timestamp Y, with content hash Z). A `Source` is the registered external origin (the parliament website). The `SourceDocument` records *what* was fetched; the `Source` records *from where*. Together they establish the outermost provenance: a citizen can see not only which document supports a claim but where that document came from and when it was fetched.

## Citation validation flow

When a `CandidateFact` arrives from the AI subsystem (via the `CandidateFactProposed` event), the evidence worker runs the following flow:

```
CandidateFactProposed
        │
        ▼
   ┌────────────┐
   │ Extract    │   the AI supplied citations as (document_id, page, section, text)
   │ citations  │
   └─────┬──────┘
         │
         ▼
   ┌────────────┐
   │ Resolve    │   for each citation: does the document exist? the page? the section?
   │ citations  │   does it contain the quoted text?
   └─────┬──────┘
         │
         ▼
   ┌────────────┐
   │ Check      │   does the cited text actually support the claim?
   │ support    │   (LLM-based entailment check, with low temperature and a strict prompt)
   └─────┬──────┘
         │
         ▼
   ┌────────────┐
   │ Check      │   does the claim contradict an accepted canonical fact?
   │ conflict   │   if yes → record SourceConflict, do not accept
   └─────┬──────┘
         │
         ▼
   ┌────────────┐
   │ Accept or  │   accepted → CandidateFactValidated, becomes a Claim
   │ Reject     │   rejected → CandidateFactRejected, with reason
   └────────────┘
```

Every step is logged. The AI subsystem can see *why* its candidate was rejected — which citation failed to resolve, which support check failed, which conflict was found — and use that feedback to improve. Rejections are not failures; they are training signal.

## The "AI cannot mutate truth" rule in detail

The rule has five layers of enforcement, from policy down to the database role grant. Each layer is necessary; no single layer is sufficient.

### Layer 1: Policy

The rule is stated in `ARCHITECTURE.md`, in the worklog, in `CONTRIBUTING.md`, in this document, and in [ADR-0005](../adr/ADR-0005-ai-cannot-mutate-truth.md). Engineers are expected to know it and to refuse PRs that violate it. Policy alone is insufficient because it depends on every engineer reading and remembering the rule.

### Layer 2: Code structure

The AI subsystem is split into two services: `services/intelligence/` (Go, the gateway) and `services/ai/` (Python, the model workers). Neither service imports the domain's write path. `services/ai/` connects to Postgres with a role that has no `UPDATE` or `DELETE` on any canonical table — it can only `SELECT` from `legislation.*` and `evidence.*` and `INSERT` into `intelligence.candidate_facts` and `intelligence.ai_responses`. The code path for writing to canonical tables is not importable by the AI service.

### Layer 3: Candidate facts

Every AI-generated assertion that should become a canonical fact is written to `intelligence.candidate_facts` with a status of `proposed`. The candidate-fact table is in the `intelligence` schema, not the `legislation` schema, and the AI role has `INSERT` permission only on rows with `status = 'proposed'`. Moving a candidate to `accepted` requires a separate validation step (run by the evidence worker with a different role) that checks citations, support, and conflicts. The AI role cannot move a candidate to `accepted` even if it tried.

### Layer 4: Citation requirement

A candidate fact with zero citations is rejected at the schema level: the `candidate_facts` table has a check constraint requiring at least one citation, and the insert path validates that each citation resolves to a real `Evidence` row. A candidate whose citations do not resolve is rejected by the validator. The AI cannot bypass this; it has no path to write a `Claim` directly.

### Layer 5: Audit and review

Every accepted candidate fact is logged to the audit table with the candidate ID, the validator ID, the citations, and the timestamp. The audit table is append-only and reviewed weekly for anomalies — a sudden spike in accepted candidates from a single capability, a candidate accepted with only one weak citation, a candidate accepted by an unusual validator — these are signals to investigate. The audit log is the last line of defense: even if a candidate slips through validation, the audit trail shows how it got in.

## The contradiction engine

When a new claim (accepted from a candidate fact, or asserted by ingestion) is about to be written, the contradiction engine compares it against existing accepted claims. If two claims assert contradictory facts — "Bill X commences on 2024-01-01" vs. "Bill X commences on 2024-02-01" — the engine does not silently pick one. It writes a `SourceConflict` record that references both claims, both sets of evidence, and the nature of the contradiction, and emits a `SourceConflictDetected` event.

The conflict is then surfaced:

- To the citizen: the bill's page shows "two sources disagree on the commencement date; we are showing the more recent source and have flagged the discrepancy."
- To editors (maintainers): the conflict appears in a triage queue for human review.
- To the AI: the conflict is included in the context for future capabilities that touch this bill, so the AI knows there is a discrepancy and does not silently pick a side.

This rule — *record conflicts, do not silently resolve them* — is essential to the platform's trustworthiness. A platform that silently picks the "right" answer when sources disagree is a platform that can be wrong without anyone knowing. A platform that surfaces the conflict is a platform that admits its own uncertainty and invites the citizen to investigate. See [ADR-0013](../adr/ADR-0013-contradiction-engine-do-not-silently-resolve.md) for the full rationale.

The contradiction engine handles several classes of conflict:

- **Direct factual contradiction.** Two claims assert different values for the same fact (different commencement dates, different sponsors, different vote counts).
- **Structural contradiction.** Two claims assert different structures (a bill has 50 clauses per one source, 52 per another — usually a parsing error, but flagged until resolved).
- **Temporal contradiction.** A claim asserts a fact that contradicts an earlier-accepted claim about a state that should not have changed (e.g. a bill's sponsor changing after introduction without an recorded amendment).
- **Authority contradiction.** Two authoritative sources disagree (the parliament website says one thing, the gazette says another).

In all cases, the conflict is recorded; neither claim is silently deleted. The conflict stays open until a human resolves it (by marking one claim as authoritative and retracting the other, with a recorded reason) or until new evidence arrives that resolves it.
