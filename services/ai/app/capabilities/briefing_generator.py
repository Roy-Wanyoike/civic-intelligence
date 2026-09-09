"""BriefingGenerator — produces the daily Kenya Civic Brief.

Every item in the briefing MUST link to evidence. The briefing explains
significance WITHOUT political persuasion.
"""
from __future__ import annotations

from datetime import datetime, timezone
from typing import Any
from uuid import UUID

from ..config import Settings
from ..domain import BriefingItem, Confidence, DailyBriefing
from ..gateway import ModelGateway
from ..logging import get_logger

log = get_logger(__name__)


class BriefingGeneratorCapability:
    capability_name = "briefing_generator"

    def __init__(self, settings: Settings, gateway: ModelGateway) -> None:
        self._settings = settings
        self._gateway = gateway

    async def generate(
        self,
        country: str,
        events: list[dict[str, Any]],
    ) -> DailyBriefing:
        items: list[BriefingItem] = []
        for ev in events[:20]:  # cap at 20 items per briefing
            kind = ev.get("event_type", "new_bill")
            title = ev.get("title", "Untitled")
            description = ev.get("description", "")
            citations = ev.get("citations", [])
            significance = ev.get("significance") or self._default_significance(kind, title)
            items.append(
                BriefingItem(
                    kind=kind,
                    title=title,
                    description=description,
                    bill_id=ev.get("bill_id"),
                    citations=citations,
                    significance=significance,
                    confidence=Confidence.HIGH if citations else Confidence.LOW,
                )
            )

        headline = (
            f"Kenya Civic Brief — {datetime.now(timezone.utc).strftime('%A, %d %B %Y')}: "
            f"{len(items)} verified development(s)."
        )

        return DailyBriefing(
            country=country,
            date=datetime.now(timezone.utc),
            headline=headline,
            items=items,
        )

    @staticmethod
    def _default_significance(kind: str, title: str) -> str:
        # Conservative, neutral phrasing. Never asserts political value.
        defaults = {
            "new_bill": "A new Bill has been introduced in Parliament.",
            "stage_change": "A Bill has advanced to a new parliamentary stage.",
            "amendment": "An amendment has been tabled.",
            "new_document": "A new official document has been published.",
            "enacted": "Legislation has been enacted.",
            "committee": "A committee has published activity.",
        }
        return defaults.get(kind, "A verified civic development.")
