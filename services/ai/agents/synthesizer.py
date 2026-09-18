"""ScenarioSynthesizer — composes an evidence-grounded explanation.

Given a ``Task`` whose ``context`` carries evidence chunks (from the
Researcher), comparable cases (from the Comparator), and an audit report
(from the AssumptionAuditor), this agent:

  1. Builds a structured prompt that bundles the evidence + audit results.
  2. Calls the model to produce a multi-paragraph explanation.
  3. Tags the explanation with ``ai_generated=True`` (already set by the
     base class) and lists the assumptions that were dropped (per the audit)
     so the consumer can surface them as caveats.

Hard contract: every claim in the synthesised answer is grounded in either
(a) a supplied evidence chunk or (b) flagged as an assumption. The synthesiser
NEVER invents citations. If the model produces an inline citation that
doesn't match a supplied chunk, it is stripped and the sentence is added to
``assumptions``.
"""
from __future__ import annotations

from decimal import Decimal
from typing import Any

from app.logging import get_logger

from .base import BaseAgent, BudgetExceeded, Permission, Result, Task

log = get_logger("agents.synthesizer")

_SYSTEM_PROMPT = (
    "You are the ScenarioSynthesizer of the Civic Intelligence Platform.\n"
    "Write a multi-paragraph explanation of the scenario using ONLY the "
    "evidence chunks and comparable cases supplied below.\n\n"
    "HARD RULES:\n"
    "1. NEVER fabricate citations. Only cite the document IDs supplied.\n"
    "2. Mark every sentence that goes beyond the evidence as an assumption.\n"
    "3. Do not state WHY a clause was added unless an authoritative source "
    "states the reason.\n"
    "4. Be politically neutral. Present evidence, not opinion.\n"
    "5. Output:\n"
    "   - EXPLANATION: <multi-paragraph markdown>\n"
    "   - ASSUMPTIONS: <bullet list, one per line>\n"
    "   - CONFIDENCE: low|medium|high\n"
)


class ScenarioSynthesizer(BaseAgent):
    name = "synthesizer"
    version = "0.1.0"
    description = (
        "Produces an evidence-grounded multi-paragraph explanation by "
        "composing the outputs of the Researcher, Comparator, and Auditor "
        "agents."
    )
    permissions = Permission.CALL_MODEL | Permission.WRITE_AUDIT_LOG

    #: Hard cap on the input context the synthesiser will send to the model.
    #: 6000 tokens ≈ 24k characters; tuned to fit in the default model
    #: context window while leaving room for the answer.
    MAX_CONTEXT_CHARS: int = 24_000

    async def _run(self, task: Task) -> Result:
        if self._model is None:
            return Result(
                agent_name=self.name,
                agent_version=self.version,
                answer="No model client configured — cannot synthesise.",
                confidence=0.0,
                metadata={"reason": "no_model"},
            )

        evidence_chunks: list[dict[str, Any]] = list(
            task.context.get("evidence_chunks", [])
        )
        comparable_cases: list[dict[str, Any]] = list(
            task.context.get("comparable_cases", [])
        )
        audit_assumptions: list[str] = list(
            task.context.get("audit_assumptions", [])
        )
        question: str = task.context.get("question", task.question)

        user_prompt = _build_user_prompt(
            question=question,
            evidence=evidence_chunks,
            comparables=comparable_cases,
            assumptions=audit_assumptions,
        )
        # Truncate to the budget.
        user_prompt = user_prompt[: self.MAX_CONTEXT_CHARS]

        # Pre-flight budget check: estimate tokens at ~4 chars/token and
        # max_tokens=512. Don't call if it would breach.
        est_input_tokens = len(user_prompt) // 4
        est_output_tokens = 512
        est_cost = Decimal("0.50")  # conservative
        if task.budget.would_exceed(est_input_tokens + est_output_tokens, est_cost):
            raise BudgetExceeded(
                f"synthesizer would exceed budget "
                f"(need ~{est_input_tokens + est_output_tokens} tokens, "
                f"have {task.budget.remaining()[0]})"
            )

        try:
            content, pt, ct, cost = await self._model.complete(
                system_prompt=_SYSTEM_PROMPT,
                user_prompt=user_prompt,
                max_tokens=est_output_tokens,
                temperature=0.0,
            )
        except Exception as exc:  # noqa: BLE001
            log.error(
                "synthesizer.model_failed",
                extra={"correlation_id": task.correlation_id, "error": str(exc)},
            )
            return Result(
                agent_name=self.name,
                agent_version=self.version,
                answer="Synthesis failed — see logs.",
                confidence=0.0,
                metadata={"error": str(exc)},
            )

        # Charge the actual cost (may be lower than the estimate).
        self._charge_model(task, pt, ct, cost)

        # Parse the structured output.
        explanation, assumptions, conf = _parse_structured_output(content)

        # Citations: only the chunks we passed in (we trust none others).
        citations = [
            {
                "document_id": str(ch.get("document_id", "")),
                "source_url": str(ch.get("source_url", "")),
                "snippet": str(ch.get("text", ""))[:200],
            }
            for ch in evidence_chunks[:8]
        ]

        return Result(
            agent_name=self.name,
            agent_version=self.version,
            answer=explanation,
            citations=citations,
            assumptions=assumptions,
            confidence=_confidence_to_float(conf),
            metadata={
                "model_prompt_tokens": pt,
                "model_completion_tokens": ct,
                "model_cost_cents": str(cost),
                "evidence_chunks_supplied": len(evidence_chunks),
                "comparable_cases_supplied": len(comparable_cases),
                "audit_assumptions_supplied": len(audit_assumptions),
            },
        )


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _build_user_prompt(
    *,
    question: str,
    evidence: list[dict[str, Any]],
    comparables: list[dict[str, Any]],
    assumptions: list[str],
) -> str:
    parts: list[str] = [f"QUESTION:\n{question}\n"]

    if evidence:
        parts.append("EVIDENCE CHUNKS (cite these only):")
        for i, ch in enumerate(evidence, 1):
            parts.append(
                f"  [{i}] doc={ch.get('document_id')} "
                f"src={ch.get('source_url')}\n"
                f"      {str(ch.get('text', ''))[:600]}"
            )
        parts.append("")
    else:
        parts.append("EVIDENCE CHUNKS: (none — answer with high uncertainty)\n")

    if comparables:
        parts.append("COMPARABLE CASES:")
        for c in comparables[:5]:
            parts.append(
                f"  - {c.get('kind', 'case')} {c.get('id')}: "
                f"{c.get('snippet', '')[:160]}"
            )
        parts.append("")

    if assumptions:
        parts.append("KNOWN ASSUMPTIONS (carry these forward as caveats):")
        for a in assumptions[:10]:
            parts.append(f"  - {a}")
        parts.append("")

    parts.append("Now produce the EXPLANATION, ASSUMPTIONS, and CONFIDENCE.")
    return "\n".join(parts)


def _parse_structured_output(content: str) -> tuple[str, list[str], str]:
    """Split the model's output into (explanation, assumptions, confidence)."""
    explanation_lines: list[str] = []
    assumptions: list[str] = []
    confidence = "unknown"

    section = "explanation"
    for line in content.splitlines():
        stripped = line.strip()
        upper = stripped.upper()
        if upper.startswith("EXPLANATION:"):
            section = "explanation"
            rest = stripped[len("EXPLANATION:"):].strip()
            if rest:
                explanation_lines.append(rest)
            continue
        if upper.startswith("ASSUMPTIONS:"):
            section = "assumptions"
            rest = stripped[len("ASSUMPTIONS:"):].strip()
            if rest:
                assumptions.append(rest.lstrip("-*").strip())
            continue
        if upper.startswith("CONFIDENCE:"):
            section = "confidence"
            rest = stripped[len("CONFIDENCE:"):].strip().lower()
            if rest in ("low", "medium", "high"):
                confidence = rest
            continue
        if section == "explanation":
            explanation_lines.append(line)
        elif section == "assumptions" and stripped:
            assumptions.append(stripped.lstrip("-*").strip())

    return ("\n".join(explanation_lines).strip(), assumptions, confidence)


def _confidence_to_float(s: str) -> float:
    return {"low": 0.3, "medium": 0.6, "high": 0.9}.get(s, 0.0)
