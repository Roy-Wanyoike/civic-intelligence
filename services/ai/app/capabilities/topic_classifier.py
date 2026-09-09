"""TopicClassifier — assigns topic tags to a Bill/document.

Uses a controlled vocabulary (housing, health, education, finance, etc.) so
search and recommendations stay consistent.
"""
from __future__ import annotations

from dataclasses import dataclass

from ..config import Settings
from ..gateway import ModelGateway


TOPIC_VOCABULARY = [
    "agriculture", "constitution", "county-government", "data-protection",
    "education", "elections", "energy", "environment", "finance", "fisheries",
    "gender", "health", "housing", "ict", "infrastructure", "judiciary",
    "labour", "land", "mining", "public-finance", "security", "taxation",
    "trade", "transport", "water", "wildlife", "youth",
]


@dataclass
class TopicClassification:
    topics: list[str]
    confidence: float


class TopicClassifierCapability:
    capability_name = "topic_classifier"

    def __init__(self, settings: Settings, gateway: ModelGateway) -> None:
        self._settings = settings
        self._gateway = gateway

    def classify(self, text: str) -> TopicClassification:
        """Keyword-based classifier. A real implementation uses embeddings + threshold."""
        text_lower = text.lower()
        matched = [t for t in TOPIC_VOCABULARY if t.replace("-", " ") in text_lower or t in text_lower]
        confidence = min(1.0, 0.4 + 0.15 * len(matched))
        return TopicClassification(topics=matched[:5], confidence=confidence)
