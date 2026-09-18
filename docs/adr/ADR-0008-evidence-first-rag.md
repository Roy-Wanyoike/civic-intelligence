# ADR-0008: Evidence-first RAG with citation validation

## Status

Accepted — 2026-09-09

## Context

The platform's headline feature is "Ask about this Bill" — a citizen asks a question and gets an evidence-grounded answer. The temptation is to use a generic LLM (e.g., "Ask ChatGPT about Kenya Bills"). This fails because:

- LLMs hallucinate facts (invented dates, invented MPs, invented votes).
- LLMs cannot cite their sources.
- LLMs cannot distinguish "I don't know" from "here's a confident-sounding wrong answer."
- For civic information, an ungrounded claim is worse than no answer.

## Decision

Adopt an **evidence-first RAG pipeline with mandatory citation validation**:

```
question → intent classification → query expansion → hybrid search (FTS + pgvector)
       → reranking → evidence selection (top-k with confidence > threshold)
       → context construction (with citations)
       → LLM (provider-agnostic) → claim extraction
       → citation validation → response (with citations) OR flagged failure
```

### Hard rules (enforced in code)

1. The system prompt explicitly says: "NEVER fabricate. Do not invent Bills, dates, votes, MPs, senators, committees, quotes, or stages. If the context does not contain the answer, say: 'Information could not be verified from available authoritative sources.'"
2. Every factual claim extracted from the response MUST have at least one citation.
3. A citation's snippet MUST share at least one significant word with the claim (anti-hallucination heuristic).
4. A response with unsupported factual claims has `validation_status = 'failed'` and is surfaced to the user with a clear "[This response could not be fully verified]" notice.

### Permanent evaluation dataset

`services/ai/eval/test_eval_dataset.py` runs a fixed set of test questions and asserts on:
- Factual accuracy (response matches known-good answer).
- Citation correctness (cited sources actually support the claim).
- Hallucination rate (response does not contain forbidden markers like "I guarantee", "definitely passed").
- Terminology correctness (uses the right Kenyan parliamentary terms).
- Timeline accuracy (preserves date uncertainty rather than fabricating).

> No AI prompt/model change reaches production without the eval dataset passing.

## Consequences

- **Positive**: citizens can trust the answers — every claim links to evidence.
- **Positive**: hallucinations are caught at the validation gate, not in production.
- **Positive**: the eval dataset provides a measurable signal for AI improvements.
- **Negative**: some legitimate answers will be flagged as "could not be fully verified" — that's a feature, not a bug. The platform's credibility depends on under-claiming, not over-claiming.
- **Negative**: the citation validation heuristic (snippet word overlap) will occasionally produce false negatives. Mitigated by: (a) showing the user the unsupported answer with a clear flag, (b) ongoing tuning of the heuristic based on real-world usage.

## References

- ARCHITECTURE.md §12 (Evidence system)
- `services/ai/app/citation_validator.py`
- `services/ai/app/rag.py`
- `services/ai/eval/test_eval_dataset.py`
- ADR-0005 (AI cannot mutate truth)
