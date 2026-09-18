"""TerminologyExplainer — explains Kenyan parliamentary terms in plain language.

Term definitions come from the country adapter's terminology registry. The
explanation is grounded in the term's ``simple_explanation`` and
``official_definition`` (where available).
"""
from __future__ import annotations

from dataclasses import dataclass

from ..config import Settings
from ..gateway import ModelGateway


@dataclass
class TermDefinition:
    term: str
    simple_explanation: str
    official_definition: str | None
    stage_code: str | None
    country: str
    sources: list[str]


class TerminologyExplainerCapability:
    capability_name = "terminology_explainer"

    def __init__(self, settings: Settings, gateway: ModelGateway) -> None:
        self._settings = settings
        self._gateway = gateway

    async def explain(self, term: TermDefinition) -> str:
        out = f"{term.term} — {term.simple_explanation}"
        if term.official_definition:
            out += f"\n\nOfficial definition: {term.official_definition}"
        if term.sources:
            out += "\n\nSource: " + ", ".join(term.sources)
        return out
