"""AssumptionAuditor — identifies unsupported assertions in a piece of text.

Given a ``Task`` whose ``context["text"]`` is an AI-generated draft answer
(or an article, or a Q&A response), this agent:

  1. Optionally uses the model to extract candidate assertions (sentences
     that make a factual claim about the world).
  2. For each candidate assertion, calls the evidence client to check for
     supporting citations.
  3. Returns a :class:`Result`` whose ``assumptions`` field lists the
     assertions WITHOUT supporting evidence — these are the unsupported
     assumptions that downstream consumers must flag or drop.

Hard contract: the auditor only READS. It never edits the input text. The
``Result.assumptions`` list is for the synthesiser / human reviewer to act on.
"""
from __future__ import annotations

import re
from typing import Any

from app.logging import get_logger

from .base import BaseAgent, Permission, Result, Task

log = get_logger("agents.auditor")

# Cheap sentence splitter — good enough for the audit step. The model would
# be more accurate but we want to be able to run this agent WITHOUT a model
# (so the auditor is still useful when the gateway is down).
_SENTENCE_SPLIT = re.compile(r"(?<=[.!?])\s+")

# Heuristic assertion markers — sentences containing these verbs are more
# likely to be factual claims worth auditing.
_ASSERTION_HINTS = (
    "will", "shall", "must", "is expected", "are expected",
    "increases", "decreases", "reduces", "raises", "lowers",
    "affects", "impacts", "results in", "causes",
    "means", "requires", "allows", "prohibits",
)


class AssumptionAuditor(BaseAgent):
    name = "auditor"
    version = "0.1.0"
    description = (
        "Identifies unsupported factual assertions in a piece of AI-generated "
        "text by cross-checking each candidate claim against the evidence service."
    )
    permissions = Permission.READ_EVIDENCE | Permission.SEARCH_DOCUMENTS | Permission.CALL_MODEL

    #: When True, the auditor will use the model (if available) to extract
    #: candidate assertions. When False, it uses the regex splitter only.
    #: Default False so the auditor works offline.
    use_model_for_extraction: bool = False

    async def _run(self, task: Task) -> Result:
        text: str = task.context.get("text", "")
        if not text:
            return Result(
                agent_name=self.name,
                agent_version=self.version,
                answer="Nothing to audit — context.text is empty.",
                confidence=1.0,
                metadata={"sentences_audited": 0},
            )

        # 1. Extract candidate assertions.
        candidates = _extract_candidates(text, use_model=False)
        if self.use_model_for_extraction and self._model is not None:
            try:
                content, pt, ct, cost = await self._model.complete(
                    system_prompt=(
                        "You are the AssumptionAuditor. Extract every factual "
                        "claim from the user text. Return a JSON array of "
                        "strings, one per claim, no commentary."
                    ),
                    user_prompt=text[:4000],
                    max_tokens=512,
                    temperature=0.0,
                )
                # Don't try to JSON-parse — just split by lines; the synthesiser
                # will re-check. Charge the budget regardless.
                self._charge_model(task, pt, ct, cost)
                model_lines = [ln.strip().strip('"-').strip() for ln in content.splitlines() if ln.strip()]
                if model_lines:
                    candidates = model_lines
            except Exception as exc:  # noqa: BLE001
                log.warning(
                    "auditor.model_extraction_failed",
                    extra={"correlation_id": task.correlation_id, "error": str(exc)},
                )

        # 2. Cross-check each candidate against the evidence service.
        unsupported: list[str] = []
        supported: list[dict[str, Any]] = []
        if self._evidence is None:
            # No evidence client → treat ALL candidates as unsupported.
            unsupported.extend(candidates)
        else:
            for claim in candidates:
                try:
                    chunks = await self._evidence.retrieve(claim, top_k=3)
                except Exception as exc:  # noqa: BLE001
                    log.warning(
                        "auditor.evidence_lookup_failed",
                        extra={"correlation_id": task.correlation_id, "claim": claim, "error": str(exc)},
                    )
                    unsupported.append(claim)
                    continue
                if chunks:
                    supported.append({"claim": claim, "evidence_count": len(chunks)})
                else:
                    unsupported.append(claim)

        confidence = (
            1.0 - (len(unsupported) / max(1, len(candidates)))
            if candidates
            else 1.0
        )
        answer = (
            f"Audited {len(candidates)} candidate assertion(s): "
            f"{len(supported)} supported, {len(unsupported)} unsupported."
        )

        return Result(
            agent_name=self.name,
            agent_version=self.version,
            answer=answer,
            citations=[
                {"kind": "supported_claim", "claim": s["claim"], "evidence_count": s["evidence_count"]}
                for s in supported
            ],
            assumptions=unsupported,
            confidence=confidence,
            metadata={
                "candidates_total": len(candidates),
                "supported": len(supported),
                "unsupported": len(unsupported),
                "used_model": self.use_model_for_extraction and self._model is not None,
            },
        )


def _extract_candidates(text: str, *, use_model: bool = False) -> list[str]:
    """Cheap sentence splitter + assertion-marker filter."""
    sentences = [s.strip() for s in _SENTENCE_SPLIT.split(text) if s.strip()]
    if not sentences:
        return []
    # Keep sentences that look like assertions. Drop pure questions / commands
    # / headings (very short, end in '?').
    out: list[str] = []
    for s in sentences:
        if s.endswith("?"):
            continue
        if len(s) < 12:
            continue
        if any(hint in s.lower() for hint in _ASSERTION_HINTS):
            out.append(s)
    return out or sentences  # fall back to all sentences if no markers matched
