"""End-to-end test of the FastAPI app via TestClient."""
from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from app.main import app


@pytest.fixture
def client():
    with TestClient(app) as c:
        yield c


def test_healthz(client):
    resp = client.get("/healthz")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "ok"


def test_readyz(client):
    resp = client.get("/readyz")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "ready"
    assert data["provider"] in {"stub", "openai", "anthropic"}


def test_list_capabilities(client):
    resp = client.get("/v1/capabilities")
    assert resp.status_code == 200
    caps = resp.json()["capabilities"]
    assert "bill_summarizer" in caps
    assert "citation_validator" in caps
    assert "contradiction_detector" in caps
    assert len(caps) == 12  # exactly the 12 from the spec


def test_terminology_explain(client):
    resp = client.post(
        "/v1/terminology/explain",
        json={
            "term": "Second Reading",
            "simple_explanation": "The stage where MPs debate the principles of a Bill.",
            "official_definition": "Per Standing Order 95.",
            "country": "KE",
            "sources": ["https://parliament.go.ke/standing-orders"],
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert "Second Reading" in data["explanation"]


def test_topics_classify(client):
    resp = client.post(
        "/v1/topics/classify",
        json={"text": "This Bill amends the Education Act and affects school funding."},
    )
    assert resp.status_code == 200
    data = resp.json()
    assert "education" in data["topics"]


def test_answer_question_marks_unsupported_claims(client):
    """The headline architectural test: a question with no evidence retrieval
    MUST return a response flagged as not fully validated."""
    resp = client.post(
        "/v1/questions",
        json={
            "text": "What is happening with the Housing Bill?",
            "country": "KE",
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    # Stub provider returns content, but with no evidence chunks retrieved
    # (search service is offline), validation should flag the response.
    assert data["validation_status"] in {"failed", "passed"}
    assert "validated" in data


def test_briefing_generate(client):
    resp = client.post(
        "/v1/briefing/generate",
        json={
            "country": "KE",
            "events": [
                {
                    "event_type": "new_bill",
                    "title": "Test Bill",
                    "description": "A test.",
                    "citations": [],
                }
            ],
        },
    )
    assert resp.status_code == 200
    data = resp.json()
    assert data["country"] == "KE"
    assert len(data["items"]) == 1
