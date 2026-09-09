"""Tests for the citation validator — the trust layer of the platform."""
from __future__ import annotations

from datetime import datetime
from uuid import uuid4

from app.citation_validator import (
    claim_requires_citation,
    validate_claim,
    validate_response,
)
from app.domain import (
    AIResponse,
    Claim,
    ClaimType,
    Citation,
    Confidence,
    SourceType,
    ValidationStatus,
)


def _make_citation(snippet: str = "The Bill was read a second time on 14 April 2024.") -> Citation:
    return Citation(
        document_id=uuid4(),
        source_url="https://parliament.go.ke/bills/123",
        page_number=3,
        section="Order Paper",
        snippet=snippet,
        source_type=SourceType.PARLIAMENT,
        retrieved_at=datetime.utcnow(),
    )


def test_claim_requires_citation_for_facts():
    fact = Claim(type=ClaimType.FACT, text="The Bill passed Second Reading.")
    interp = Claim(type=ClaimType.INTERPRETATION, text="This Bill is significant.")
    assert claim_requires_citation(fact) is True
    assert claim_requires_citation(interp) is False  # pure interpretation


def test_claim_with_no_citations_fails_validation():
    claim = Claim(type=ClaimType.FACT, text="The Bill passed Second Reading on 14 April 2024.")
    ok, reason = validate_claim(claim)
    assert ok is False
    assert reason == "claim_requires_evidence_but_has_no_citations"


def test_claim_with_valid_citation_passes():
    cite = _make_citation(snippet="The Bill passed Second Reading on 14 April 2024.")
    claim = Claim(
        type=ClaimType.FACT,
        text="The Bill passed Second Reading on 14 April 2024.",
        citations=[cite],
        confidence=Confidence.HIGH,
    )
    ok, reason = validate_claim(claim)
    assert ok is True
    assert reason is None


def test_claim_with_unsupported_snippet_fails():
    """A citation whose snippet shares no significant words with the claim
    should be rejected — this is the anti-hallucination gate."""
    cite = _make_citation(snippet="The weather in Nairobi was sunny today.")
    claim = Claim(
        type=ClaimType.FACT,
        text="The Bill passed Second Reading on 14 April 2024.",
        citations=[cite],
    )
    ok, reason = validate_claim(claim)
    assert ok is False
    assert reason == "citation_snippet_does_not_support_claim"


def test_response_with_unsupported_claims_is_flagged():
    bad_claim = Claim(type=ClaimType.FACT, text="The President assented on 1 January 2025.")
    response = AIResponse(
        answer="The President assented on 1 January 2025.",
        claims=[bad_claim],
        citations=[],
        model="stub-1",
        provider="stub",
    )
    validate_response(response)
    assert response.validated is False
    assert response.validation_status == ValidationStatus.FAILED
    assert len(response.validation_failures) >= 1


def test_response_with_supported_claims_passes():
    cite = _make_citation(snippet="The Bill passed Second Reading on 14 April 2024.")
    claim = Claim(
        type=ClaimType.FACT,
        text="The Bill passed Second Reading on 14 April 2024.",
        citations=[cite],
        confidence=Confidence.HIGH,
    )
    response = AIResponse(
        answer="The Bill passed Second Reading on 14 April 2024.",
        claims=[claim],
        citations=[cite],
        model="stub-1",
        provider="stub",
    )
    validate_response(response)
    assert response.validated is True
    assert response.validation_status == ValidationStatus.PASSED
    assert response.validation_failures == []
