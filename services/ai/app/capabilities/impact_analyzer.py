"""ImpactAnalyzer — analyzes potential relevance of a Bill to a user's lens.

Hard rule: distinguish DIRECTLY_SUPPORTED, POTENTIAL, INFERENCE, UNKNOWN.
Never present speculative impact as established fact.
"""
from __future__ import annotations

from dataclasses import dataclass
from enum import Enum
from typing import Literal

from ..config import Settings
from ..domain import Citation, Confidence
from ..gateway import ModelGateway


class EvidenceStrength(str, Enum):
    DIRECTLY_SUPPORTED = "directly_supported"
    POTENTIAL = "potential"
    INFERENCE = "inference"
    UNKNOWN = "unknown"


Lens = Literal[
    "citizen", "student", "small_business", "employee", "farmer",
    "driver", "tenant", "homeowner", "employer", "digital_worker",
]


@dataclass
class ImpactAssessment:
    lens: Lens
    summary: str
    strength: EvidenceStrength
    citations: list[Citation]
    caveats: list[str]


class ImpactAnalyzerCapability:
    capability_name = "impact_analyzer"

    LENS_KEYWORDS = {
        "small_business": ["business", "enterprise", " sme", "trader", "company", "licens"],
        "farmer": ["agricultur", "farmer", "crop", "livestock", "land"],
        "tenant": ["tenant", "rent", "landlord", "lease"],
        "homeowner": ["homeowner", "property owner", "mortgage"],
        "employer": ["employer", "employment", "labour", "worker"],
        "digital_worker": ["digital", "data", "online", "platform", "ict"],
        "driver": ["transport", "vehicle", "driver", "traffic", "ntsa"],
        "student": ["education", "school", "university", "student"],
        "employee": ["employee", "employment", "workplace", "labour"],
        "citizen": [],  # catch-all
    }

    def __init__(self, settings: Settings, gateway: ModelGateway) -> None:
        self._settings = settings
        self._gateway = gateway

    def analyze(self, lens: Lens, bill_text: str, citations: list[Citation]) -> ImpactAssessment:
        text_lower = bill_text.lower()
        keywords = self.LENS_KEYWORDS.get(lens, [])
        hits = [kw for kw in keywords if kw in text_lower]
        if not hits:
            return ImpactAssessment(
                lens=lens,
                summary="This Bill does not appear to directly address this group based on a keyword scan. Information could not be verified from available authoritative sources.",
                strength=EvidenceStrength.UNKNOWN,
                citations=[],
                caveats=["Keyword-based scan only. A human review is recommended before publishing."],
            )
        strength = EvidenceStrength.DIRECTLY_SUPPORTED if len(hits) >= 3 and citations else EvidenceStrength.POTENTIAL
        return ImpactAssessment(
            lens=lens,
            summary=f"This Bill references {len(hits)} concept(s) relevant to {lens.replace('_', ' ')}: {', '.join(hits[:5])}.",
            strength=strength,
            citations=citations,
            caveats=["This is a structural relevance scan, not a legal interpretation."] if strength != EvidenceStrength.DIRECTLY_SUPPORTED else [],
        )
