"""BillSummarizer — generates a plain-language summary of a Bill.

Output structure (per spec):
  - short_title
  - one_line_summary
  - plain_language_explanation
  - current_stage_explained
  - what_it_would_do (list)
  - who_it_affects (list)
  - what_happens_next (list)
  - citations
  - confidence
  - validated
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Any
from uuid import UUID

from ..config import Settings
from ..domain import (
    BillSummary,
    Citation,
    Confidence,
    ModelRequest,
    ValidationStatus,
)
from ..citation_validator import validate_response
from ..evidence_store import EvidenceStore
from ..gateway import ModelGateway
from ..logging import get_logger

log = get_logger(__name__)


@dataclass
class BillContext:
    """Bill metadata + retrieved evidence chunks for summarization."""

    bill_id: UUID
    title: str
    identifier: str
    year: int
    sponsor: str | None
    house: str | None
    current_stage: str | None
    plain_text: str  # concatenated chunk text
    citations: list[Citation]


class BillSummarizerCapability:
    """Produces a validated, evidence-grounded Bill summary."""

    capability_name = "bill_summarizer"

    def __init__(self, settings: Settings, gateway: ModelGateway, evidence: EvidenceStore) -> None:
        self._settings = settings
        self._gateway = gateway
        self._evidence = evidence

    async def summarize(self, ctx: BillContext) -> BillSummary:
        system_prompt = (
            "You are the BillSummarizer capability of the Civic Intelligence Platform for Kenya.\n"
            "Produce a plain-language summary of the Bill below.\n\n"
            "HARD RULES:\n"
            "1. Use plain language a Form-4 student can understand.\n"
            "2. NEVER fabricate. If the source doesn't say something, don't say it.\n"
            "3. Distinguish FACT (from source), EXPLANATION (your interpretation), UNKNOWN.\n"
            "4. Do not explain WHY a clause was added unless the source states the reason.\n"
            "5. Be politically neutral. Present evidence, not opinion.\n"
            "6. Respond as strict JSON matching the requested schema.\n"
        )
        user_prompt = (
            f"BILL: {ctx.title} ({ctx.identifier}, {ctx.year})\n"
            f"SPONSOR: {ctx.sponsor or 'unknown'}\n"
            f"HOUSE: {ctx.house or 'unknown'}\n"
            f"CURRENT STAGE: {ctx.current_stage or 'unknown'}\n\n"
            f"SOURCE TEXT:\n{ctx.plain_text[:8000]}\n\n"
            "Respond as JSON with keys: short_title, one_line_summary, "
            "plain_language_explanation, current_stage_explained, "
            "what_it_would_do (array of strings), who_it_affects (array of strings), "
            "what_happens_next (array of strings, may be empty if unknown)."
        )

        request = ModelRequest(
            capability=self.capability_name,
            system_prompt=system_prompt,
            user_prompt=user_prompt,
            response_format="json",
            max_tokens=1024,
            temperature=0.0,
        )
        model_response = await self._gateway.complete(request)
        parsed = model_response.parsed_json or {}

        summary = BillSummary(
            bill_id=ctx.bill_id,
            short_title=str(parsed.get("short_title", ctx.title)),
            one_line_summary=str(parsed.get("one_line_summary", "")),
            plain_language_explanation=str(parsed.get("plain_language_explanation", "")),
            current_stage_explained=str(parsed.get("current_stage_explained", "")),
            what_it_would_do=list(parsed.get("what_it_would_do", [])),
            who_it_affects=list(parsed.get("who_it_affects", [])),
            what_happens_next=list(parsed.get("what_happens_next", [])),
            citations=ctx.citations,
            confidence=Confidence.HIGH if ctx.citations else Confidence.LOW,
            validated=False,
            validation_failures=[],
        )

        if self._settings.citation_validation_required:
            from ..domain import AIResponse, Claim, ClaimType
            response = AIResponse(
                answer=summary.plain_language_explanation,
                claims=[
                    Claim(type=ClaimType.FACT, text=summary.one_line_summary, citations=ctx.citations)
                ],
                citations=ctx.citations,
                model=model_response.model,
                provider=model_response.provider,
                prompt_tokens=model_response.prompt_tokens,
                completion_tokens=model_response.completion_tokens,
                latency_ms=model_response.latency_ms,
                cost_cents=model_response.cost_cents,
            )
            validate_response(
                response,
                fail_on_unsupported=self._settings.citation_validation_fail_on_unsupported,
            )
            summary.validated = response.validated
            summary.validation_failures = response.validation_failures
            if not response.validated:
                summary.confidence = Confidence.LOW
        return summary
