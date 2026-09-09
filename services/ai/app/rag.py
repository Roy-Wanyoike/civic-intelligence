"""The RAG pipeline: question -> intent -> query expansion -> hybrid search ->
rerank -> evidence selection -> context -> LLM -> claim extraction -> citation
validation -> response.
"""
from __future__ import annotations

import json
import re
from datetime import datetime
from typing import Any

from .config import Settings
from .domain import (
    AIResponse,
    Claim,
    ClaimType,
    Citation,
    Confidence,
    ModelRequest,
    Question,
    ValidationStatus,
)
from .evidence_store import EvidenceStore
from .gateway import ModelGateway
from .citation_validator import validate_response
from .logging import get_logger

log = get_logger(__name__)


# ---------- intent classification ----------

_INTENT_KEYWORDS = {
    "summarize": ["summarize", "summary", "what is this bill", "explain this bill", "tldr"],
    "timeline": ["timeline", "history", "what happened", "when did", "stage"],
    "stage": ["what stage", "current stage", "where is it", "next stage"],
    "compare": ["compare", "diff", "what changed", "difference"],
    "impact": ["impact", "affect", "how does this affect", "what does it mean for"],
    "definition": ["what does", "what is", "explain", "mean", "definition"],
    "search": ["find", "search", "list", "show me"],
    "follow": ["follow", "subscribe", "notify"],
}


def classify_intent(question: Question) -> str:
    """Map a question to a capability name. Falls back to ``civic_question_answerer``."""
    text = question.text.lower()
    for intent, keywords in _INTENT_KEYWORDS.items():
        if any(kw in text for kw in keywords):
            return intent
    return "civic_question_answerer"


# ---------- query expansion ----------

_STOPWORDS = {"the", "a", "an", "is", "are", "of", "to", "in", "on", "this", "that", "with"}


def expand_query(question: Question) -> list[str]:
    """Produce multiple query variants for hybrid search.

    The current implementation is rule-based and conservative. A future
    iteration will use an LLM to expand synonyms and parliamentary terminology.
    """
    tokens = [t for t in re.findall(r"\b\w+\b", question.text.lower()) if t not in _STOPWORDS]
    variants = [question.text]
    if tokens:
        variants.append(" ".join(tokens[:6]))  # keyword-only variant
    if question.bill_id:
        variants.append(f"bill {question.bill_id}")
    if question.impact_lens:
        variants.append(f"impact {question.impact_lens}")
    return variants


# ---------- claim extraction ----------

def extract_claims(answer: str, citations: list[Citation]) -> list[Claim]:
    """Extract claims from a model response. Currently rule-based: split on
    sentences and attach citations heuristically. A future iteration will use
    a dedicated claim-extraction prompt.
    """
    if not answer:
        return []
    sentences = [s.strip() for s in re.split(r"(?<=[.!?])\s+", answer) if s.strip()]
    claims: list[Claim] = []
    for i, sent in enumerate(sentences):
        # Heuristic: cite the i-th citation for the i-th claim, cycling if needed.
        cite = citations[i % len(citations)] if citations else None
        claim = Claim(
            type=ClaimType.FACT,
            text=sent,
            citations=[cite] if cite else [],
            confidence=Confidence.MEDIUM if cite else Confidence.LOW,
        )
        claims.append(claim)
    return claims


# ---------- the pipeline ----------

class RAGPipeline:
    """Orchestrates the full RAG flow for a citizen question."""

    def __init__(
        self,
        settings: Settings,
        gateway: ModelGateway,
        evidence_store: EvidenceStore,
    ) -> None:
        self._settings = settings
        self._gateway = gateway
        self._evidence_store = evidence_store

    async def answer(self, question: Question) -> AIResponse:
        """Execute the full RAG pipeline."""
        t0 = datetime.utcnow()

        intent = classify_intent(question)
        log.info("rag_intent_classified", extra={"event": "rag_intent_classified", "intent": intent, "question_id": None})

        variants = expand_query(question)
        # Search using the most specific variant first.
        all_chunks = []
        seen_chunk_ids = set()
        for variant in variants:
            chunks = await self._evidence_store.retrieve(variant, bill_id=str(question.bill_id) if question.bill_id else None)
            for c in chunks:
                if c.chunk_id not in seen_chunk_ids:
                    all_chunks.append(c)
                    seen_chunk_ids.add(c.chunk_id)

        reranked = self._evidence_store.rerank(all_chunks)
        context = self._evidence_store.build_context(reranked)

        # Build the LLM prompt with strict instructions: never fabricate, cite everything.
        system_prompt = self._system_prompt()
        user_prompt = self._user_prompt(question, context)

        request = ModelRequest(
            capability=intent,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            response_format="text",
            max_tokens=1024,
            temperature=0.0,
        )

        model_response = await self._gateway.complete(request)
        claims = extract_claims(model_response.content, context.citations)

        response = AIResponse(
            answer=model_response.content,
            claims=claims,
            citations=context.citations,
            rag_context=context,
            validated=False,
            validation_status=ValidationStatus.PENDING,
            model=model_response.model,
            provider=model_response.provider,
            prompt_tokens=model_response.prompt_tokens,
            completion_tokens=model_response.completion_tokens,
            latency_ms=model_response.latency_ms,
            cost_cents=model_response.cost_cents,
        )

        if self._settings.citation_validation_required:
            validate_response(
                response,
                fail_on_unsupported=self._settings.citation_validation_fail_on_unsupported,
            )

        latency_total_ms = int((datetime.utcnow() - t0).total_seconds() * 1000)
        log.info(
            "rag_response_complete",
            extra={
                "event": "rag_response_complete",
                "intent": intent,
                "claims_count": len(claims),
                "citations_count": len(context.citations),
                "validated": response.validated,
                "latency_ms": latency_total_ms,
                "model": response.model,
                "provider": response.provider,
            },
        )
        return response

    @staticmethod
    def _system_prompt() -> str:
        return (
            "You are a civic information assistant for Kenya. Your job is to help "
            "citizens understand what their government is doing.\n\n"
            "HARD RULES:\n"
            "1. NEVER fabricate. Do not invent Bills, dates, votes, MPs, senators, "
            "committees, quotes, or stages.\n"
            "2. Every factual claim you make MUST be supported by a citation in "
            "the provided context.\n"
            "3. If the context does not contain the answer, say: 'Information could "
            "not be verified from available authoritative sources.'\n"
            "4. Distinguish FACT (from source), EXPLANATION (your plain-language "
            "interpretation), and UNKNOWN. Label each section.\n"
            "5. Never tell the user what political position to take. Present "
            "evidence, explain meaning, let them decide.\n"
            "6. Never explain WHY a change was made unless the source states the reason.\n"
            "7. Use plain language. Explain parliamentary terms inline.\n"
        )

    @staticmethod
    def _user_prompt(question: Question, context: Any) -> str:
        ctx_text = "\n\n".join(
            f"[{i+1}] (page {c.page_number or '?'}, section {c.section or '?'}) {c.snippet}"
            for i, c in enumerate(context.citations)
        ) or "[No evidence retrieved]"
        lens_hint = f"\nThe user is asking from the perspective of: {question.impact_lens}." if question.impact_lens else ""
        return (
            f"QUESTION: {question.text}\n"
            f"{lens_hint}\n\n"
            f"EVIDENCE (cite these by number):\n{ctx_text}\n\n"
            f"ANSWER:"
        )
