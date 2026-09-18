"""HistoricalComparator — finds comparable past bills / scenarios.

Given a ``Task`` about a bill, this agent:

  1. Queries the simulation service for past scenarios tagged with the same
     bill_id OR overlapping topics.
  2. Queries the legislation service for prior bills on similar topics.
  3. Returns a :class:`Result` listing the comparable cases (with provenance
     pointers) and a short paragraph noting which past case is closest.

The comparator never calls a model directly — it only composes references.
If a similarity score is needed, callers can pass ``task.context["embeddings"]``
(list of float vectors) and the agent will compute cosine similarity in-process.
"""
from __future__ import annotations

import math
from typing import Any

from app.logging import get_logger

from .base import BaseAgent, Permission, Result, Task

log = get_logger("agents.comparator")


class HistoricalComparator(BaseAgent):
    name = "comparator"
    version = "0.1.0"
    description = (
        "Finds comparable past bills / scenarios by topic overlap and "
        "(optionally) embedding cosine similarity."
    )
    permissions = (
        Permission.READ_LEGISLATION | Permission.READ_SCENARIOS | Permission.SEARCH_DOCUMENTS
    )

    async def _run(self, task: Task) -> Result:
        bill_id = task.context.get("bill_id")
        topics: list[str] = list(task.context.get("topics", []))
        limit = int(task.context.get("limit", 10))

        comparable_scenarios: list[dict[str, Any]] = []
        comparable_bills: list[dict[str, Any]] = []

        if self._scenarios is not None:
            try:
                comparable_scenarios = await self._scenarios.list_scenarios(
                    bill_id=bill_id, limit=limit
                )
            except Exception as exc:  # noqa: BLE001
                log.warning(
                    "comparator.scenarios_failed",
                    extra={
                        "correlation_id": task.correlation_id,
                        "error": str(exc),
                    },
                )

        if self._legislation is not None and topics:
            try:
                # Reuse search_bills with a topic query.
                query = " ".join(topics)
                comparable_bills = await self._legislation.search_bills(
                    query=query, limit=limit
                )
            except Exception as exc:  # noqa: BLE001
                log.warning(
                    "comparator.legislation_failed",
                    extra={
                        "correlation_id": task.correlation_id,
                        "error": str(exc),
                    },
                )

        # Optionally rank by cosine similarity if embeddings were supplied.
        embeddings: list[list[float]] = list(
            task.context.get("embeddings", [])
        )
        query_vec: list[float] | None = task.context.get("query_embedding")
        ranked: list[dict[str, Any]] = []
        if query_vec and embeddings:
            ranked = _rank_by_cosine(query_vec, embeddings)

        answer = (
            f"Found {len(comparable_scenarios)} comparable scenario(s) and "
            f"{len(comparable_bills)} comparable bill(s)."
        )
        if ranked:
            top_score = ranked[0]["similarity"]
            answer += f" Top cosine similarity: {top_score:.3f}."

        citations: list[dict[str, Any]] = []
        for s in comparable_scenarios[:5]:
            citations.append({
                "kind": "scenario",
                "id": str(s.get("id", "")),
                "snippet": str(s.get("title", ""))[:200],
            })
        for b in comparable_bills[:5]:
            citations.append({
                "kind": "bill",
                "id": str(b.get("id", "")),
                "snippet": str(b.get("title", ""))[:200],
            })

        return Result(
            agent_name=self.name,
            agent_version=self.version,
            answer=answer,
            citations=citations,
            evidence_refs=[c["id"] for c in citations],
            confidence=min(1.0, (len(comparable_scenarios) + len(comparable_bills)) / 20.0),
            metadata={
                "comparable_scenarios": len(comparable_scenarios),
                "comparable_bills": len(comparable_bills),
                "ranked_by_embedding": bool(ranked),
            },
        )


def _cosine(a: list[float], b: list[float]) -> float:
    if not a or not b or len(a) != len(b):
        return 0.0
    dot = sum(x * y for x, y in zip(a, b, strict=True))
    na = math.sqrt(sum(x * x for x in a))
    nb = math.sqrt(sum(y * y for y in b))
    if na == 0.0 or nb == 0.0:
        return 0.0
    return dot / (na * nb)


def _rank_by_cosine(
    query: list[float], embeddings: list[list[float]]
) -> list[dict[str, Any]]:
    scored = [
        {"index": i, "similarity": _cosine(query, vec)}
        for i, vec in enumerate(embeddings)
    ]
    scored.sort(key=lambda r: r["similarity"], reverse=True)
    return scored
