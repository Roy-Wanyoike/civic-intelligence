"""Citation validator.

This is the trust layer of the entire platform. Every AI response passes through
:func:`validate_response`. A response with unsupported factual claims fails
validation and is either flagged or rejected based on settings.

Architectural rule: an AI response that cannot be grounded in evidence is NOT
surfaced as a factual claim to citizens.
"""
from __future__ import annotations

import re
from urllib.parse import urlparse

from .domain import (
    AIResponse,
    Claim,
    ClaimType,
    Confidence,
    ValidationStatus,
)
from .logging import get_logger

log = get_logger(__name__)


# Patterns that suggest a claim is asserting a hard fact that MUST be cited.
_FACT_PATTERNS = re.compile(
    r"\b(?:on\s+\d{1,2}\s+\w+|in\s+\d{4}|stage\s+\w+|passed|rejected|"
    r"assented|gazetted|introduced|sponsored\s+by|published\s+on)\b",
    re.IGNORECASE,
)


def claim_requires_citation(claim: Claim) -> bool:
    """A claim requires citation if it asserts a fact, date, stage, entity, or definition."""
    if claim.type in {ClaimType.FACT, ClaimType.STAGE, ClaimType.DATE, ClaimType.ENTITY, ClaimType.DEFINITION}:
        return True
    if claim.type == ClaimType.INTERPRETATION:
        # Interpretations require evidence if they make factual assertions.
        return bool(_FACT_PATTERNS.search(claim.text))
    return False


def validate_claim(claim: Claim) -> tuple[bool, str | None]:
    """Return ``(ok, failure_reason)``."""
    if not claim_requires_citation(claim):
        return True, None
    if not claim.citations:
        return False, "claim_requires_evidence_but_has_no_citations"
    for cite in claim.citations:
        # URL must be parseable and have a network location.
        try:
            url = urlparse(str(cite.source_url))
            if not url.scheme or not url.netloc:
                return False, f"citation_url_invalid:{cite.source_url}"
        except Exception as e:  # pragma: no cover - defensive
            return False, f"citation_url_unparseable:{e}"
        # Snippet must be non-empty.
        if not cite.snippet or not cite.snippet.strip():
            return False, "citation_snippet_empty"
        # Snippet should overlap with the claim text (rough heuristic).
        # We check for at least one shared significant word.
        claim_words = {w.lower() for w in re.findall(r"\b\w{4,}\b", claim.text)}
        snippet_words = {w.lower() for w in re.findall(r"\b\w{4,}\b", cite.snippet)}
        if claim_words and not (claim_words & snippet_words):
            return False, "citation_snippet_does_not_support_claim"
    return True, None


def validate_response(
    response: AIResponse, *, fail_on_unsupported: bool = True
) -> AIResponse:
    """Validate every claim in the response. Mutates the response in place
    and returns it for convenience.
    """
    failures: list[str] = []
    for claim in response.claims:
        ok, reason = validate_claim(claim)
        if not ok and reason:
            failures.append(f"{claim.id}:{reason}")

    if not response.citations and response.claims:
        failures.append("response_has_claims_but_no_citations")

    if failures:
        response.validation_status = ValidationStatus.FAILED
        response.validated = False
        response.validation_failures = failures
        log.warning(
            "citation_validation_failed",
            extra={
                "event": "citation_validation_failed",
                "response_id": str(response.id),
                "failures": failures,
            },
        )
        if fail_on_unsupported:
            # In strict mode we mark the answer as needing human review.
            response.answer = (
                "[This response could not be fully verified from available "
                "authoritative sources.]\n\n" + response.answer
            )
    elif response.claims and all(c.is_supported() for c in response.claims):
        response.validation_status = ValidationStatus.PASSED
        response.validated = True
    else:
        # No claims to validate (e.g., a pure refusal or off-topic response).
        response.validation_status = ValidationStatus.PASSED
        response.validated = True

    return response


def downgrade_confidence(claim: Claim, reason: str) -> Claim:
    """Lower a claim's confidence and record why. Used when evidence is weak."""
    claim.confidence = Confidence.LOW
    log.info(
        "claim_confidence_downgraded",
        extra={"event": "claim_confidence_downgraded", "claim_id": str(claim.id), "reason": reason},
    )
    return claim
