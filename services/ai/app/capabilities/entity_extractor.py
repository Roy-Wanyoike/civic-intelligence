"""EntityExtractor — extracts People, Committees, Institutions, Topics from text."""
from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID

from ..config import Settings
from ..gateway import ModelGateway


@dataclass
class ExtractedEntity:
    kind: str  # person | committee | institution | topic
    name: str
    confidence: float
    source_snippet: str


class EntityExtractorCapability:
    capability_name = "entity_extractor"

    def __init__(self, settings: Settings, gateway: ModelGateway) -> None:
        self._settings = settings
        self._gateway = gateway

    async def extract(self, text: str) -> list[ExtractedEntity]:
        # Stub: regex-based person name detection. Real implementation uses an
        # LLM with a constrained JSON schema + a gazetteer of known MPs/Senators.
        import re
        results: list[ExtractedEntity] = []
        # Match "Hon. <Name>" / "Senator <Name>" / "MP <Name>" patterns.
        for m in re.finditer(r"\b(?:Hon\.?|Honourable|Senator|MP|Cabinet Secretary)\s+([A-Z][a-z]+(?:\s+[A-Z][a-z]+){0,2})", text):
            results.append(
                ExtractedEntity(
                    kind="person",
                    name=m.group(1),
                    confidence=0.6,
                    source_snippet=text[max(0, m.start()-20):m.end()+20],
                )
            )
        return results
