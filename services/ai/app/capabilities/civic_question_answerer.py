"""CivicQuestionAnswerer — general citizen Q&A over the civic domain.

This is the headline capability: a citizen asks a natural-language question
about a Bill, an Act, or a topic, and receives an evidence-grounded answer.

The actual RAG flow lives in ``app/rag.py`` — this class is the public face
that the HTTP layer calls.
"""
from __future__ import annotations

from ..config import Settings
from ..domain import AIResponse, Question
from ..evidence_store import EvidenceStore
from ..gateway import ModelGateway
from ..rag import RAGPipeline


class CivicQuestionAnswererCapability:
    capability_name = "civic_question_answerer"

    def __init__(
        self,
        settings: Settings,
        gateway: ModelGateway,
        evidence_store: EvidenceStore,
    ) -> None:
        self._pipeline = RAGPipeline(settings, gateway, evidence_store)

    async def answer(self, question: Question) -> AIResponse:
        return await self._pipeline.answer(question)
