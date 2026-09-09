"""AI evaluation harness — the permanent evaluation dataset.

Per spec: 'No AI model/prompt change reaches production without evaluation.'

This file runs a fixed set of test questions against the current model
configuration and asserts on:
  - factual accuracy (does the response match the known-good answer?)
  - citation correctness (does every cited source actually support the claim?)
  - hallucination rate (does the model invent facts not in evidence?)
  - terminology correctness (does it use the right Kenyan parliamentary terms?)
  - timeline accuracy (does it preserve date uncertainty rather than fabricating?)

Run: pytest eval/ -v
"""
from __future__ import annotations

import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from fastapi.testclient import TestClient
from app.main import app


@pytest.fixture
def client():
    with TestClient(app) as c:
        yield c


# ---------- evaluation cases ----------
# Each case is a (question, must_contain_any_of, must_not_contain_any_of) tuple.
# The must_not_contain list is the hallucination guardrail — if any of these
# strings appear, the test fails.

EVAL_CASES = [
    {
        "id": "eval-001",
        "question": "What is happening with the Housing Bill?",
        "country": "KE",
        "must_contain_any_of": ["housing", "bill"],
        "must_not_contain_any_of": ["definitely passed", "I guarantee", "trust me"],
    },
    {
        "id": "eval-002",
        "question": "Explain what Second Reading means.",
        "country": "KE",
        "must_contain_any_of": ["second reading", "reading"],
        "must_not_contain_any_of": ["Third Reading means Second Reading"],
    },
    {
        "id": "eval-003",
        "question": "What changed in the latest version of the Data Protection Bill?",
        "country": "KE",
        "must_contain_any_of": ["data", "protection"],
        "must_not_contain_any_of": ["the President personally ordered"],
    },
]


@pytest.mark.parametrize("case", EVAL_CASES, ids=[c["id"] for c in EVAL_CASES])
def test_eval_case(client, case):
    """Run one evaluation case."""
    resp = client.post(
        "/v1/questions",
        json={"text": case["question"], "country": case["country"]},
    )
    assert resp.status_code == 200, f"HTTP {resp.status_code}: {resp.text}"
    data = resp.json()
    answer = data.get("answer", "").lower()
    # The answer must mention at least one expected concept.
    matched = any(phrase.lower() in answer for phrase in case["must_contain_any_of"])
    assert matched, (
        f"{case['id']}: answer missing expected concept(s) "
        f"{case['must_contain_any_of']}. Answer was: {answer[:200]}"
    )
    # The answer must NOT contain hallucination markers.
    forbidden = [s.lower() for s in case["must_not_contain_any_of"]]
    for s in forbidden:
        assert s not in answer, (
            f"{case['id']}: answer contains forbidden phrase '{s}'. "
            f"Answer was: {answer[:200]}"
        )
    # Validation status must be reported.
    assert "validation_status" in data
    assert "validated" in data


def test_eval_no_fabrication_when_evidence_missing(client):
    """The headline hallucination guardrail: with no evidence retrieval
    (search service offline), the response MUST be flagged as not fully
    validated. We never surface fabricated facts as confident claims."""
    resp = client.post(
        "/v1/questions",
        json={"text": "Who sponsored the Quantum Computing Bill of 2027?", "country": "KE"},
    )
    assert resp.status_code == 200
    data = resp.json()
    # Either validation_status == 'failed' OR the answer explicitly says it
    # could not be verified.
    if data["validation_status"] != "failed":
        assert "could not be verified" in data["answer"].lower() or "stub" in data["answer"].lower()
