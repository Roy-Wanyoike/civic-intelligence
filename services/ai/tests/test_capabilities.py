"""Tests for the capabilities: BillSummarizer, ImpactAnalyzer, TopicClassifier, etc."""
from __future__ import annotations

from uuid import uuid4

import pytest

from app.capabilities.briefing_generator import BriefingGeneratorCapability
from app.capabilities.contradiction_detector import ContradictionDetectorCapability
from app.capabilities.document_comparator import (
    BillVersionInput,
    DocumentComparatorCapability,
)
from app.capabilities.impact_analyzer import EvidenceStrength, ImpactAnalyzerCapability
from app.capabilities.topic_classifier import TopicClassifierCapability
from app.config import get_settings
from app.evidence_store import EvidenceStore
from app.gateway import StubProvider, build_gateway


@pytest.fixture
def settings():
    return get_settings()


@pytest.fixture
def gateway(settings):
    # Force the stub provider for tests.
    from app.gateway import ModelGateway
    return ModelGateway(primary=StubProvider(), fallbacks=[], budget_usd_per_day=10.0, max_retries=0)


@pytest.fixture
def evidence(settings):
    return EvidenceStore(settings)


# ---------- topic classifier ----------

def test_topic_classifier_matches_known_topics():
    cap = TopicClassifierCapability(None, None)  # type: ignore[arg-type]
    result = cap.classify("This Bill amends the Education Act and affects school funding and university admissions.")
    assert "education" in result.topics
    assert result.confidence > 0.4


def test_topic_classifier_returns_empty_for_unrelated_text():
    cap = TopicClassifierCapability(None, None)  # type: ignore[arg-type]
    result = cap.classify("The cat sat on the mat.")
    assert result.topics == []


# ---------- impact analyzer ----------

def test_impact_analyzer_marks_unknown_when_no_relevance():
    cap = ImpactAnalyzerCapability(None, None)  # type: ignore[arg-type]
    assessment = cap.analyze(
        lens="farmer",
        bill_text="This Bill regulates digital identity verification.",
        citations=[],
    )
    assert assessment.strength == EvidenceStrength.UNKNOWN
    assert "could not be verified" in assessment.summary.lower()


def test_impact_analyzer_finds_direct_support_with_citations():
    cap = ImpactAnalyzerCapability(None, None)  # type: ignore[arg-type]
    assessment = cap.analyze(
        lens="farmer",
        bill_text="This Bill amends the Agriculture Act and provides for crop insurance, livestock traceability, and farmer cooperatives.",
        citations=[type("C", (), {"source_url": "https://parliament.go.ke"})()],  # stub citation
    )
    assert assessment.strength in {EvidenceStrength.POTENTIAL, EvidenceStrength.DIRECTLY_SUPPORTED}


# ---------- document comparator ----------

def test_document_comparator_detects_additions_and_removals():
    cap = DocumentComparatorCapability(None, None)  # type: ignore[arg-type]
    bill_id = uuid4()
    before = BillVersionInput(
        bill_id=bill_id,
        version_label="v1",
        text="",
        clauses=["Clause 1: Short title", "Clause 2: Interpretation"],
    )
    after = BillVersionInput(
        bill_id=bill_id,
        version_label="v2",
        text="",
        clauses=["Clause 1: Short title", "Clause 3: Commencement"],
    )
    diff = cap.diff(before, after, citations=[])
    assert "v2" in diff.to_version
    assert len(diff.additions) == 1
    assert len(diff.removals) == 1
    assert diff.stated_reason is None  # never invents a reason
    assert diff.validated is False  # no citations = not validated


# ---------- contradiction detector ----------

def test_contradiction_detector_records_conflict():
    cap = ContradictionDetectorCapability(None, None)  # type: ignore[arg-type]
    from datetime import datetime
    from uuid import uuid4
    conflict = cap.detect(
        claim="Bill stage",
        source_a={
            "source_id": uuid4(),
            "value": "Committee",
            "url": "https://parliament.go.ke/a",
            "retrieved_at": datetime.utcnow(),
        },
        source_b={
            "source_id": uuid4(),
            "value": "Second Reading",
            "url": "https://senate.go.ke/b",
            "retrieved_at": datetime.utcnow(),
        },
    )
    assert conflict is not None
    assert conflict.source_a_value == "Committee"
    assert conflict.source_b_value == "Second Reading"
    assert conflict.resolution_strategy is None  # not auto-resolved


def test_contradiction_detector_returns_none_when_values_agree():
    cap = ContradictionDetectorCapability(None, None)  # type: ignore[arg-type]
    from datetime import datetime
    from uuid import uuid4
    same_value = "Second Reading"
    conflict = cap.detect(
        claim="Bill stage",
        source_a={
            "source_id": uuid4(),
            "value": same_value,
            "url": "https://parliament.go.ke/a",
            "retrieved_at": datetime.utcnow(),
        },
        source_b={
            "source_id": uuid4(),
            "value": same_value,
            "url": "https://senate.go.ke/b",
            "retrieved_at": datetime.utcnow(),
        },
    )
    assert conflict is None


# ---------- briefing generator ----------

@pytest.mark.asyncio
async def test_briefing_generator_produces_neutral_items(gateway):
    settings = get_settings()
    cap = BriefingGeneratorCapability(settings, gateway)
    briefing = await cap.generate(
        country="KE",
        events=[
            {
                "event_type": "new_bill",
                "title": "Housing Bill 2024",
                "description": "A Bill for housing.",
                "citations": [],
            },
            {
                "event_type": "stage_change",
                "title": "Data Protection Bill",
                "description": "Advanced to Second Reading.",
                "citations": [],
            },
        ],
    )
    assert briefing.country == "KE"
    assert len(briefing.items) == 2
    # Significance text must be politically neutral.
    for item in briefing.items:
        assert "good" not in item.significance.lower()
        assert "bad" not in item.significance.lower()
