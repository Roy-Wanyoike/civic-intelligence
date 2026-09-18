"""ResearchAgent — retrieves evidence for a question.

Given a ``Task``, this agent:

  1. Calls the evidence client (search service / hybrid retrieval) to find
     supporting chunks for the question.
  2. Optionally narrows the search to a specific bill (``task.context["bill_id"]``).
  3. Returns a :class:`Result` whose ``evidence_refs`` and ``citations`` list
     point to the supporting documents, and whose ``answer`` is a short
     summary of what evidence was found (NOT a synthesised answer — that's
     the :class:`ScenarioSynthesizer`'s job).

Hard contract: this agent never calls a model. It only retrieves evidence
and assembles references. That keeps it cheap and predictable.
"""
from __future__ import annotations

from typing import Any

from app.logging import get_logger

from .base import BaseAgent, Budget, Permission, Result, Task

log = get_logger("agents.researcher")


class ResearchAgent(BaseAgent):
    name = "researcher"
    version = "0.1.0"
    description = "Retrieves evidence for a question from the search/evidence service."
    permissions = Permission.READ_EVIDENCE | Permission.SEARCH_DOCUMENTS

    DEFAULT_TOP_K = 8

    async def _run(self, task: Task) -> Result:
        if self._evidence is None:
            log.warning(
                "researcher.no_evidence_client",
                extra={"correlation_id": task.correlation_id},
            )
            return Result(
                agent_name=self.name,
                agent_version=self.version,
                answer="No evidence client configured — returning empty result.",
                confidence=0.0,
            )

        bill_id = task.context.get("bill_id")
        top_k = int(task.context.get("top_k", self.DEFAULT_TOP_K))

        chunks: list[dict[str, Any]] = await self._evidence.retrieve(
            task.question, bill_id=bill_id, top_k=top_k
        )

        # No budget consumed — research doesn't call a model.
        citations: list[dict[str, Any]] = []
        evidence_refs: list[str] = []
        for ch in chunks:
            citations.append({
                "document_id": str(ch.get("document_id", "")),
                "source_url": str(ch.get("source_url", "")),
                "page_number": ch.get("page_number"),
                "section": ch.get("section"),
                "snippet": str(ch.get("text", ""))[:280],
                "score": float(ch.get("combined_score", 0.0)),
            })
            evidence_refs.append(str(ch.get("chunk_id", ch.get("document_id", ""))))

        confidence = _aggregate_confidence(chunks)

        answer = (
            f"Found {len(chunks)} supporting chunk(s) for the question. "
            f"Top score: {confidence:.2f}."
        )

        return Result(
            agent_name=self.name,
            agent_version=self.version,
            answer=answer,
            citations=citations,
            evidence_refs=evidence_refs,
            confidence=confidence,
            metadata={
                "chunks_returned": len(chunks),
                "bill_id": bill_id,
                "top_k_requested": top_k,
            },
        )


def _aggregate_confidence(chunks: list[dict[str, Any]]) -> float:
    """Map the top chunk's combined score onto [0, 1]. Empty → 0."""
    if not chunks:
        return 0.0
    top = max(float(c.get("combined_score", 0.0)) for c in chunks)
    # combined_score is already in [0, 1] from the search service.
    return max(0.0, min(1.0, top))
