# Civic Intelligence — AI Service (services/ai)

A Python/FastAPI service that implements the **provider-agnostic AI gateway** and the
**evidence-grounded RAG pipeline** for the Civic Intelligence Platform.

> Architectural rule: **AI may PROPOSE candidate facts, but may NEVER directly write
> to canonical legislative state.** Every AI response passes through citation
> validation; responses with unsupported factual claims fail validation.

## Bounded context

Owns:

- AI explanations (Bill summaries, stage explanations, terminology explanations)
- Q&A grounded in evidence
- Comparison explanations
- Topic classification
- Impact analysis
- Briefing generation
- Candidate-fact proposals (forwarded to the Go `services/intelligence` for validation)
- AI evaluation (hallucination rate, citation correctness, terminology accuracy)

Does **NOT** own:

- Canonical legislative state (owned by `services/legislation`)
- Document storage (owned by `services/documents`)
- Evidence records (owned by `services/evidence`)
- Search indexes (owned by `services/search`)

## Capabilities (one module per capability — never a giant prompt)

```text
app/capabilities/
  bill_summarizer.py
  timeline_extractor.py
  stage_explainer.py
  terminology_explainer.py
  document_comparator.py
  citation_validator.py
  briefing_generator.py
  civic_question_answerer.py
  impact_analyzer.py
  topic_classifier.py
  entity_extractor.py
  contradiction_detector.py
```

## RAG pipeline

```text
question
  -> intent classification
  -> query expansion
  -> hybrid search (FTS + pgvector) against evidence/documents
  -> reranking
  -> evidence selection (top-k with confidence > threshold)
  -> context construction (with citations)
  -> LLM (provider-agnostic via ModelGateway)
  -> claim extraction
  -> citation validation
  -> response (with citations) or failure
```

## Local development

```bash
cd services/ai
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
cp .env.example .env  # configure model providers
uvicorn app.main:app --reload --port 8000
```

## Tests

```bash
pytest -v --cov=app --cov-report=term-missing
```

## Evaluation harness

```bash
pytest eval/ -v  # runs the permanent AI evaluation dataset
```

No AI prompt/model change reaches production without `eval/` passing.
