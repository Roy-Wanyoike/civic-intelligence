"""Civic AI capabilities. One module per capability — never a giant prompt."""
from __future__ import annotations

from .bill_summarizer import BillSummarizerCapability
from .timeline_extractor import TimelineExtractorCapability
from .stage_explainer import StageExplainerCapability
from .terminology_explainer import TerminologyExplainerCapability
from .document_comparator import DocumentComparatorCapability
from .entity_extractor import EntityExtractorCapability
from .topic_classifier import TopicClassifierCapability
from .impact_analyzer import ImpactAnalyzerCapability
from .civic_question_answerer import CivicQuestionAnswererCapability
from .contradiction_detector import ContradictionDetectorCapability
from .briefing_generator import BriefingGeneratorCapability

__all__ = [
    "BillSummarizerCapability",
    "TimelineExtractorCapability",
    "StageExplainerCapability",
    "TerminologyExplainerCapability",
    "DocumentComparatorCapability",
    "EntityExtractorCapability",
    "TopicClassifierCapability",
    "ImpactAnalyzerCapability",
    "CivicQuestionAnswererCapability",
    "ContradictionDetectorCapability",
    "BriefingGeneratorCapability",
]
