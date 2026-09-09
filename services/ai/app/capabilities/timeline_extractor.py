"""TimelineExtractor — builds a verified timeline of Bill events.

Architectural rule: NEVER fabricate missing dates. If the exact date is
unknown, preserve that uncertainty in the output.
"""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Any
from uuid import UUID

from ..config import Settings
from ..domain import Citation, Confidence
from ..gateway import ModelGateway
from ..logging import get_logger

log = get_logger(__name__)


@dataclass
class TimelineEvent:
    event_type: str
    date: datetime | None  # None means "date unknown — preserved as uncertainty"
    date_is_approximate: bool
    house: str | None
    description: str
    source_document_id: UUID | None
    source_url: str
    confidence: Confidence
    note: str | None = None  # e.g., "Date inferred from Order Paper; exact date not stated."


@dataclass
class BillTimeline:
    bill_id: UUID
    events: list[TimelineEvent]

    def ordered(self) -> list[TimelineEvent]:
        """Return events ordered by date where known; undated events appended last."""
        dated = [e for e in self.events if e.date is not None]
        undated = [e for e in self.events if e.date is None]
        dated.sort(key=lambda e: e.date)  # type: ignore[arg-type]
        return dated + undated


class TimelineExtractorCapability:
    """Extracts a verified timeline from Bill documents and events."""

    capability_name = "timeline_extractor"

    def __init__(self, settings: Settings, gateway: ModelGateway) -> None:
        self._settings = settings
        self._gateway = gateway

    async def extract(
        self,
        bill_id: UUID,
        events: list[dict[str, Any]],
        citations: list[Citation],
    ) -> BillTimeline:
        timeline_events: list[TimelineEvent] = []
        for ev in events:
            date_raw = ev.get("date")
            date = datetime.fromisoformat(date_raw) if isinstance(date_raw, str) else None
            timeline_events.append(
                TimelineEvent(
                    event_type=ev.get("event_type", "unknown"),
                    date=date,
                    date_is_approximate=bool(ev.get("date_is_approximate", False)),
                    house=ev.get("house"),
                    description=ev.get("description", ""),
                    source_document_id=ev.get("source_document_id"),
                    source_url=ev.get("source_url", ""),
                    confidence=Confidence.HIGH if date and ev.get("source_url") else Confidence.LOW,
                    note=ev.get("note"),
                )
            )
        return BillTimeline(bill_id=bill_id, events=timeline_events)
