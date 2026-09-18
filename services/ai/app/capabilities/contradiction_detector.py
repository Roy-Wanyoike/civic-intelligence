"""ContradictionDetector — detects when two authoritative sources conflict.

Per spec: when sources A and B disagree (e.g., one says stage=Committee,
another says stage=Second Reading), DO NOT silently choose. Record a
``SourceConflict`` with metadata about each source so a human (or later
heuristic) can decide which to prefer.
"""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Any
from uuid import UUID, uuid4

from ..config import Settings
from ..gateway import ModelGateway


@dataclass
class SourceConflict:
    id: UUID
    claim: str
    source_a_id: UUID
    source_a_value: Any
    source_a_url: str
    source_a_retrieved_at: datetime
    source_b_id: UUID
    source_b_value: Any
    source_b_url: str
    source_b_retrieved_at: datetime
    resolution_strategy: str | None  # None means unresolved
    resolved_at: datetime | None


class ContradictionDetectorCapability:
    capability_name = "contradiction_detector"

    RESOLUTION_STRATEGIES = [
        "prefer_authoritative",
        "prefer_more_recent",
        "prefer_event_date_over_publication_date",
        "prefer_committee_report_over_tracker",
        "manual_review_required",
    ]

    def __init__(self, settings: Settings, gateway: ModelGateway) -> None:
        self._settings = settings
        self._gateway = gateway

    def detect(
        self,
        claim: str,
        source_a: dict[str, Any],
        source_b: dict[str, Any],
    ) -> SourceConflict | None:
        """Return a SourceConflict if values differ; else None."""
        if source_a.get("value") == source_b.get("value"):
            return None
        return SourceConflict(
            id=uuid4(),
            claim=claim,
            source_a_id=source_a["source_id"],
            source_a_value=source_a["value"],
            source_a_url=source_a["url"],
            source_a_retrieved_at=source_a["retrieved_at"],
            source_b_id=source_b["source_id"],
            source_b_value=source_b["value"],
            source_b_url=source_b["url"],
            source_b_retrieved_at=source_b["retrieved_at"],
            resolution_strategy=None,
            resolved_at=None,
        )
