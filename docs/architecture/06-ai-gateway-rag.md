# 06 — AI Gateway and RAG

This document describes the AI gateway (provider-agnostic, capability-based), the retrieval-augmented generation (RAG) pipeline, the 12 civic AI capabilities, and the evaluation framework that gates every prompt and model change. The deep version of the "AI cannot mutate truth" rule is in [`05-evidence-system.md`](./05-evidence-system.md); the corresponding decision record is [ADR-0008](../adr/ADR-0008-evidence-first-rag.md).

## The AI gateway

The AI gateway (`services/intelligence/`) is the single entry point for AI work in the platform. Workers, the API, and background jobs all call the gateway; the gateway routes to the right capability implementation, tracks the budget, enforces citation validation, and records the eval result. The gateway is provider-agnostic — it talks to OpenAI, Anthropic, Google, or self-hosted models through a uniform interface — and capability-based — a caller asks for a named capability (`BillSummarizer`, `Ask`), not a raw LLM call.

This design exists for three reasons. First, **isolation**: the AI workers (`services/ai/`, Python) are the only code that talks to model providers, so a compromised provider cannot reach the rest of the platform. Second, **eval-ability**: every AI call is logged with the prompt version, model version, inputs, outputs, and citations, so an eval run can reproduce any past call and check it against expected behavior. Third, **provider portability**: the gateway abstracts the provider, so switching from OpenAI to a self-hosted model for a given capability is a config change, not a code change.

The gateway exposes two surfaces: a synchronous gRPC interface (for the API's `/bills/{id}/questions` SSE endpoint and other citizen-facing calls) and an async work-queue interface (for background batch jobs like summarizing every new bill version). Both produce the same artifacts: an `AIResponse` with one or more `AIClaim`s, each with citations.

## The RAG pipeline

The retrieval-augmented generation pipeline is the heart of the platform's AI. A citizen question (or a worker job) enters as text; the pipeline retrieves relevant evidence, builds a context window, calls the LLM, extracts claims, validates citations, and returns the response. The pipeline is nine stages:

```
1. Intent      →  classify the question (factual lookup, summary, comparison, open-ended)
2. Query       →  expand the question into retrieval queries (rewrites, synonyms, terminology)
   Expansion
3. Hybrid      →  parallel: BM25 full-text search over Bills/Acts + vector search over chunks
   Search
4. Reranking   →  cross-encoder reranks the merged results by relevance to the original intent
5. Evidence    →  select the top-N evidence items that fit the context window, with diversity
   Selection    (don't pick 10 chunks from the same page)
6. Context     →  render the evidence as quoted, attributed context (not as instructions)
   Build
7. LLM         →  call the model with the system prompt + context + question
8. Claim       →  parse the LLM output into structured AIClaims, each with its citations
   Extraction
9. Citation    →  run the evidence validator (see 05-evidence-system.md) on each claim;
   Validation   reject claims with unresolved or unsupported citations
10. Response   →  return the validated AIResponse (claims + citations) to the caller
```

Every stage is instrumented and logged. The pipeline is deterministic given the same inputs and the same model/prompt version (modulo the LLM's own non-determinism, which we minimize with low temperature and seeded sampling where the provider supports it). The eval framework can replay any past call and check it against the expected output.

### Intent classification

The intent stage decides what kind of question this is. "When was bill KE-2024-42 introduced?" is a factual lookup — it should hit the canonical record, not call an LLM at all. "What does this bill do?" is a summary — it should call `BillSummarizer`. "How does this bill compare to the 2023 version?" is a comparison — it should call a comparison capability. "Why is the government regulating this?" is open-ended — it should call the `Ask` capability with cross-domain retrieval.

Classifying intent lets the gateway route to the right capability without burning an LLM call on every question. Simple factual lookups return instantly from the canonical record; only the harder questions go to the model.

### Query expansion

A citizen asks "How will this bill affect my business?" The retrieval pipeline needs to find the clauses that mention business impact, taxation, regulation, and compliance — none of which appear in the literal question. Query expansion takes the citizen question and produces a set of retrieval queries: the original, a rewritten version, a version with synonyms, a version that adds terminology from the country glossary, and a version that adds the bill's title and topic tags. Each query runs against the hybrid search; results are merged and reranked.

### Hybrid search and reranking

Hybrid search runs BM25 (full-text over Bills, Acts, Hansard) and vector search (cosine similarity over document chunks in pgvector) in parallel and merges the results. BM25 catches exact-keyword matches; vector search catches semantic matches. Reranking with a cross-encoder (a small, fast model trained on relevance) reorders the merged results by actual relevance to the original question. The top-N results (typically 10-20 chunks, depending on context window size) become the evidence pool.

### Citation validation

Every claim the LLM emits is parsed into a structured `AIClaim` with citations. The citation validator (described in [`05-evidence-system.md`](./05-evidence-system.md)) checks that each citation resolves to a real document, page, and section, that the cited text supports the claim (via an entailment check), and that the claim does not contradict an accepted canonical fact. Claims that fail validation are stripped from the response and the user sees a warning that the AI's answer was partially unsupported. This is the technical enforcement of the "evidence before AI" principle.

## The 12 civic AI capabilities

A "capability" is a named AI task with a defined input contract, output contract, and evidence requirements. Capabilities are registered in the gateway catalog and implemented either in `services/ai/` (Python) or — for pure orchestration — in `services/intelligence/` (Go). New capabilities require an ADR and an eval dataset.

| # | Capability | Input | Output | Evidence req. |
| --- | --- | --- | --- | --- |
| 1 | `BillSummarizer` | BillVersion | Plain-language summary + key points | One citation per key point |
| 2 | `TimelineExtractor` | Bill | Ordered list of events with dates and descriptions | One citation per event |
| 3 | `AmendmentDiffExplainer` | BillVersion A, BillVersion B | Plain-language explanation of what changed | One citation per change |
| 4 | `PlainLanguageExplainer` | Clause / Section | Plain-language rewrite preserving legal meaning | One citation to the source clause |
| 5 | `StakeholderExtractor` | Bill | List of stakeholders (sponsors, committees, affected parties) | One citation per stakeholder |
| 6 | `VoteExplainer` | Vote | Explanation of what was voted on and why it mattered | Citations to the motion and Hansard |
| 7 | `CommitteeReportSummarizer` | CommitteeReport | Summary of the committee's findings and recommendations | One citation per finding |
| 8 | `HansardQuoteExtractor` | Topic / Bill | Relevant quotes from Hansard with attribution | One citation per quote |
| 9 | `ContradictionDetector` | Bill (or pair of CivicMatters) | List of contradictions found | One citation per side of each contradiction |
| 10 | `TopicClusterer` | Set of CivicMatters | Clustered topics with representative members | Citations to representative members |
| 11 | `RelatedBillsFinder` | Bill | List of related bills with relationship type | Citations to the related bill and the relationship evidence |
| 12 | `Ask` | Citizen question (free text) | Evidence-backed answer with citations | One citation per claim in the answer |

The `Ask` capability is the north-star capability — it is the one that powers "Ask Kenya." It is also the hardest, because it requires cross-domain retrieval (Bills + Acts + Hansard + Committees + Policies) and the most rigorous citation validation. The other 11 capabilities are scoped to a single document or pair of documents and are therefore more tractable; they are the building blocks the `Ask` capability composes.

## Eval framework

Every capability ships with a permanent evaluation dataset: golden inputs (real citizen questions, real bill versions, real Hansard pages) and expected behaviors (the answer should contain these claims, these citations, these key points; it should not contain these hallucinations). The eval framework runs the capability against every golden input, scores the output against the expected behavior, and produces a pass/fail with a score.

The rule that gates releases: **no AI model change, no prompt change, and no capability change reaches production without passing the eval suite.** This is enforced in CI: a PR that touches `services/ai/prompts/`, `services/ai/capabilities/`, or the model config blocks merge until the eval suite passes. If the eval suite itself needs updating (because the expected behavior changed intentionally), the PR includes the eval update and a justification, and the AI council reviews both the code and the eval change together.

The eval dataset is curated. Golden inputs are selected to cover the capability's surface area — easy cases, hard cases, edge cases, adversarial cases (prompt injection attempts, ambiguous questions, missing-data scenarios). Expected behaviors are written by humans (maintainers, domain experts) and reviewed. The dataset grows monotonically — we add cases, we do not remove them (except when a case is found to be wrong, with a recorded reason). A capability that has been in production for a year has a dataset of hundreds of cases; a new capability starts with at least ten.

### Eval metrics

For each capability, the eval framework reports:

- **Pass rate.** What fraction of golden inputs produced an output that met the expected behavior.
- **Citation precision.** Of the citations the AI produced, how many were valid and supporting.
- **Citation recall.** Of the citations the expected behavior required, how many the AI produced.
- **Hallucination rate.** What fraction of the AI's claims were unsupported (had zero valid citations after validation).
- **Latency.** P50 and P99 wall-clock time per call.
- **Cost.** Token spend per call, in USD at current provider rates.

A capability passes if its pass rate is above the threshold (typically 95% for production), its citation precision is above 95%, its citation recall is above 90%, and its hallucination rate is below 1%. Latency and cost are reported but not gated (they are budgets, not hard limits).

## Streaming responses

Citizen-facing AI calls (`/bills/{id}/questions` SSE) stream the response token-by-token. The LLM output is rendered to the user as it arrives, but claims are not marked as "cited" until the citation validator runs at the end. The UX is: the user sees the answer appear in real time, and each claim gets a citation badge once validated. Claims that fail validation are visually marked as "unsupported" in the UI. This is the tradeoff [ADR-0012](../adr/ADR-0012-sse-for-streaming-ai-responses.md) makes — we accept that the user sees claims before they are validated, in exchange for the responsiveness of streaming.
