"""DocumentComparator — compares two Bill versions and explains what changed.

Architectural rule: NEVER claim WHY a change was made unless an authoritative
source states the reason. The ``stated_reason`` field is None unless the source
explicitly states one.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Any
from uuid import UUID

from ..config import Settings
from ..domain import BillDiff, Citation, Confidence
from ..gateway import ModelGateway


@dataclass
class BillVersionInput:
    bill_id: UUID
    version_label: str
    text: str
    clauses: list[str]


class DocumentComparatorCapability:
    capability_name = "document_comparator"

    def __init__(self, settings: Settings, gateway: ModelGateway) -> None:
        self._settings = settings
        self._gateway = gateway

    def diff(self, before: BillVersionInput, after: BillVersionInput, citations: list[Citation]) -> BillDiff:
        """Naive clause-level diff. A real implementation uses difflib or
        a purpose-built legislative diff library; this skeleton demonstrates
        the contract."""
        before_set = {c.strip(): c for c in before.clauses}
        after_set = {c.strip(): c for c in after.clauses}

        additions = [after_set[c] for c in after_set if c not in before_set]
        removals = [before_set[c] for c in before_set if c not in after_set]
        modifications: list[str] = []  # requires fuzzy match — omitted for brevity

        # Plain-language explanation (no speculation about WHY).
        explanation_parts = []
        if additions:
            explanation_parts.append(f"Version {after.version_label} adds {len(additions)} new clause(s).")
        if removals:
            explanation_parts.append(f"Version {after.version_label} removes {len(removals)} clause(s) present in {before.version_label}.")
        if not explanation_parts:
            explanation_parts.append("No substantive clause-level changes detected between the two versions.")
        explanation = " ".join(explanation_parts)

        return BillDiff(
            bill_id=before.bill_id,
            from_version=before.version_label,
            to_version=after.version_label,
            additions=additions,
            removals=removals,
            modifications=modifications,
            affected_clauses=[],
            plain_language_explanation=explanation,
            citations=citations,
            stated_reason=None,  # never invent a reason
            validated=bool(citations),
        )
