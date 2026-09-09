"""FastAPI application for the AI service.

Endpoints:
  GET  /healthz
  GET  /readyz
  POST /v1/bills/{id}/summarize
  POST /v1/bills/{id}/diff
  POST /v1/bills/{id}/timeline
  POST /v1/questions               (general citizen Q&A)
  POST /v1/stages/explain
  POST /v1/terminology/explain
  POST /v1/impact/analyze
  POST /v1/briefing/generate
  POST /v1/entities/extract
  POST /v1/topics/classify
  POST /v1/contradictions/detect
  GET  /v1/capabilities            (lists the 12 capabilities)
"""
from __future__ import annotations

from contextlib import asynccontextmanager
from datetime import datetime
from typing import Any
from uuid import UUID

from fastapi import FastAPI, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field

from .capabilities.briefing_generator import BriefingGeneratorCapability
from .capabilities.contradiction_detector import ContradictionDetectorCapability
from .capabilities.civic_question_answerer import CivicQuestionAnswererCapability
from .capabilities.bill_summarizer import BillContext, BillSummarizerCapability
from .capabilities.document_comparator import (
    BillVersionInput,
    DocumentComparatorCapability,
)
from .capabilities.entity_extractor import EntityExtractorCapability
from .capabilities.impact_analyzer import (
    EvidenceStrength,
    ImpactAnalyzerCapability,
    ImpactAssessment,
    Lens,
)
from .capabilities.stage_explainer import StageDefinition, StageExplainerCapability
from .capabilities.terminology_explainer import (
    TermDefinition,
    TerminologyExplainerCapability,
)
from .capabilities.timeline_extractor import TimelineExtractorCapability
from .capabilities.topic_classifier import TopicClassifierCapability
from .config import get_settings
from .domain import Question
from .evidence_store import EvidenceStore
from .gateway import ModelGateway, build_gateway
from .logging import configure_logging, get_logger, log_event
from .rag import RAGPipeline


# ---------- lifespan ----------

@asynccontextmanager
async def lifespan(app: FastAPI):
    settings = get_settings()
    configure_logging()
    log = get_logger(__name__)
    log_event(log, "ai_service_starting", env=settings.env, port=settings.service_port)
    gateway = build_gateway(settings)
    evidence = EvidenceStore(settings)
    app.state.gateway = gateway
    app.state.evidence = evidence
    app.state.settings = settings
    # capabilities
    app.state.bill_summarizer = BillSummarizerCapability(settings, gateway, evidence)
    app.state.timeline_extractor = TimelineExtractorCapability(settings, gateway)
    app.state.stage_explainer = StageExplainerCapability(settings, gateway)
    app.state.terminology_explainer = TerminologyExplainerCapability(settings, gateway)
    app.state.document_comparator = DocumentComparatorCapability(settings, gateway)
    app.state.entity_extractor = EntityExtractorCapability(settings, gateway)
    app.state.topic_classifier = TopicClassifierCapability(settings, gateway)
    app.state.impact_analyzer = ImpactAnalyzerCapability(settings, gateway)
    app.state.contradiction_detector = ContradictionDetectorCapability(settings, gateway)
    app.state.briefing_generator = BriefingGeneratorCapability(settings, gateway)
    app.state.qa = CivicQuestionAnswererCapability(settings, gateway, evidence)
    app.state.rag = RAGPipeline(settings, gateway, evidence)
    log_event(log, "ai_service_ready", provider=settings.model_gateway_default_provider)
    yield
    log_event(log, "ai_service_stopping")


app = FastAPI(
    title="Civic Intelligence — AI Service",
    description="Provider-agnostic AI gateway + evidence-grounded RAG pipeline.",
    version="0.1.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000", "http://localhost:3001"],
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["*"],
    allow_credentials=False,
)


# ---------- request/response schemas ----------

class HealthResponse(BaseModel):
    status: str
    version: str
    provider: str


class BillSummaryRequest(BaseModel):
    title: str
    identifier: str
    year: int
    sponsor: str | None = None
    house: str | None = None
    current_stage: str | None = None
    plain_text: str
    citations: list[dict[str, Any]] = Field(default_factory=list)


class BillDiffRequest(BaseModel):
    before: dict[str, Any]
    after: dict[str, Any]
    citations: list[dict[str, Any]] = Field(default_factory=list)


class TimelineRequest(BaseModel):
    events: list[dict[str, Any]]
    citations: list[dict[str, Any]] = Field(default_factory=list)


class StageExplainRequest(BaseModel):
    code: str
    name: str
    simple_explanation: str
    official_definition: str | None = None
    country: str = "KE"
    allowed_next: list[str] = Field(default_factory=list)
    sources: list[str] = Field(default_factory=list)
    bill_title: str | None = None


class TerminologyExplainRequest(BaseModel):
    term: str
    simple_explanation: str
    official_definition: str | None = None
    stage_code: str | None = None
    country: str = "KE"
    sources: list[str] = Field(default_factory=list)


class QuestionRequest(BaseModel):
    text: str
    bill_id: UUID | None = None
    country: str = "KE"
    impact_lens: Lens | None = None


class ImpactRequest(BaseModel):
    lens: Lens
    bill_text: str
    citations: list[dict[str, Any]] = Field(default_factory=list)


class BriefingRequest(BaseModel):
    country: str = "KE"
    events: list[dict[str, Any]]


class EntityRequest(BaseModel):
    text: str


class TopicRequest(BaseModel):
    text: str


class ContradictionRequest(BaseModel):
    claim: str
    source_a: dict[str, Any]
    source_b: dict[str, Any]


# ---------- routes ----------

@app.get("/healthz", response_model=HealthResponse, tags=["meta"])
async def healthz() -> HealthResponse:
    return HealthResponse(status="ok", version="0.1.0", provider="unknown")


@app.get("/readyz", response_model=HealthResponse, tags=["meta"])
async def readyz() -> HealthResponse:
    settings = get_settings()
    return HealthResponse(
        status="ready",
        version="0.1.0",
        provider=settings.model_gateway_default_provider,
    )


@app.get("/v1/capabilities", tags=["meta"])
async def list_capabilities() -> dict[str, Any]:
    """List the 12 civic AI capabilities. One module per capability — never a giant prompt."""
    return {
        "capabilities": [
            "bill_summarizer",
            "timeline_extractor",
            "stage_explainer",
            "terminology_explainer",
            "document_comparator",
            "entity_extractor",
            "topic_classifier",
            "impact_analyzer",
            "civic_question_answerer",
            "citation_validator",
            "contradiction_detector",
            "briefing_generator",
        ]
    }


@app.post("/v1/bills/{bill_id}/summarize", tags=["bills"])
async def summarize_bill(bill_id: UUID, req: BillSummaryRequest) -> dict[str, Any]:
    cap: BillSummarizerCapability = app.state.bill_summarizer
    from .domain import Citation
    citations = [Citation(**c) for c in req.citations]
    ctx = BillContext(
        bill_id=bill_id,
        title=req.title,
        identifier=req.identifier,
        year=req.year,
        sponsor=req.sponsor,
        house=req.house,
        current_stage=req.current_stage,
        plain_text=req.plain_text,
        citations=citations,
    )
    summary = await cap.summarize(ctx)
    return summary.model_dump(mode="json")


@app.post("/v1/bills/{bill_id}/diff", tags=["bills"])
async def diff_bill(bill_id: UUID, req: BillDiffRequest) -> dict[str, Any]:
    cap: DocumentComparatorCapability = app.state.document_comparator
    from .domain import Citation
    citations = [Citation(**c) for c in req.citations]
    before = BillVersionInput(
        bill_id=bill_id,
        version_label=req.before["version_label"],
        text=req.before["text"],
        clauses=req.before.get("clauses", []),
    )
    after = BillVersionInput(
        bill_id=bill_id,
        version_label=req.after["version_label"],
        text=req.after["text"],
        clauses=req.after.get("clauses", []),
    )
    diff = cap.diff(before, after, citations)
    return diff.model_dump(mode="json")


@app.post("/v1/bills/{bill_id}/timeline", tags=["bills"])
async def extract_timeline(bill_id: UUID, req: TimelineRequest) -> dict[str, Any]:
    cap: TimelineExtractorCapability = app.state.timeline_extractor
    from .domain import Citation
    citations = [Citation(**c) for c in req.citations]
    timeline = await cap.extract(bill_id, req.events, citations)
    return {
        "bill_id": str(bill_id),
        "events": [
            {
                "event_type": e.event_type,
                "date": e.date.isoformat() if e.date else None,
                "date_is_approximate": e.date_is_approximate,
                "house": e.house,
                "description": e.description,
                "source_url": e.source_url,
                "confidence": e.confidence.value,
                "note": e.note,
            }
            for e in timeline.ordered()
        ],
    }


@app.post("/v1/stages/explain", tags=["stages"])
async def explain_stage(req: StageExplainRequest) -> dict[str, Any]:
    cap: StageExplainerCapability = app.state.stage_explainer
    stage = StageDefinition(
        code=req.code,
        name=req.name,
        simple_explanation=req.simple_explanation,
        official_definition=req.official_definition,
        country=req.country,
        allowed_next=req.allowed_next,
        sources=req.sources,
    )
    explanation = await cap.explain(stage, bill_title=req.bill_title)
    return {"stage": stage.name, "code": stage.code, "explanation": explanation}


@app.post("/v1/terminology/explain", tags=["terminology"])
async def explain_term(req: TerminologyExplainRequest) -> dict[str, Any]:
    cap: TerminologyExplainerCapability = app.state.terminology_explainer
    term = TermDefinition(
        term=req.term,
        simple_explanation=req.simple_explanation,
        official_definition=req.official_definition,
        stage_code=req.stage_code,
        country=req.country,
        sources=req.sources,
    )
    return {"term": term.term, "explanation": await cap.explain(term)}


@app.post("/v1/questions", tags=["qa"])
async def answer_question(req: QuestionRequest) -> dict[str, Any]:
    cap: CivicQuestionAnswererCapability = app.state.qa
    question = Question(
        text=req.text,
        bill_id=req.bill_id,
        country=req.country,
        impact_lens=req.impact_lens,
    )
    response = await cap.answer(question)
    return response.model_dump(mode="json")


@app.post("/v1/questions/stream", tags=["qa"])
async def stream_question(req: QuestionRequest) -> StreamingResponse:
    """SSE endpoint that streams the response as it's built.

    The first chunk is the validated answer; subsequent chunks are the citations
    and validation metadata. Real per-token streaming requires provider support
    (OpenAI streaming API). The stub provider returns the full answer at once.
    """
    cap: CivicQuestionAnswererCapability = app.state.qa
    question = Question(
        text=req.text,
        bill_id=req.bill_id,
        country=req.country,
        impact_lens=req.impact_lens,
    )

    async def gen():
        import json
        response = await cap.answer(question)
        yield f"event: answer\ndata: {json.dumps({'answer': response.answer, 'validated': response.validated})}\n\n"
        yield f"event: citations\ndata: {json.dumps([c.model_dump(mode='json') for c in response.citations])}\n\n"
        yield f"event: done\ndata: {json.dumps({'response_id': str(response.id), 'validation_status': response.validation_status.value, 'validation_failures': response.validation_failures})}\n\n"

    return StreamingResponse(gen(), media_type="text/event-stream")


@app.post("/v1/impact/analyze", tags=["impact"])
async def analyze_impact(req: ImpactRequest) -> dict[str, Any]:
    cap: ImpactAnalyzerCapability = app.state.impact_analyzer
    from .domain import Citation
    citations = [Citation(**c) for c in req.citations]
    assessment = cap.analyze(req.lens, req.bill_text, citations)
    return {
        "lens": assessment.lens,
        "summary": assessment.summary,
        "strength": assessment.strength.value,
        "citations": [c.model_dump(mode="json") for c in assessment.citations],
        "caveats": assessment.caveats,
    }


@app.post("/v1/briefing/generate", tags=["briefing"])
async def generate_briefing(req: BriefingRequest) -> dict[str, Any]:
    cap: BriefingGeneratorCapability = app.state.briefing_generator
    briefing = await cap.generate(req.country, req.events)
    return briefing.model_dump(mode="json")


@app.post("/v1/entities/extract", tags=["entities"])
async def extract_entities(req: EntityRequest) -> dict[str, Any]:
    cap: EntityExtractorCapability = app.state.entity_extractor
    entities = await cap.extract(req.text)
    return {"entities": [e.__dict__ for e in entities]}


@app.post("/v1/topics/classify", tags=["topics"])
async def classify_topics(req: TopicRequest) -> dict[str, Any]:
    cap: TopicClassifierCapability = app.state.topic_classifier
    result = cap.classify(req.text)
    return {"topics": result.topics, "confidence": result.confidence}


@app.post("/v1/contradictions/detect", tags=["contradictions"])
async def detect_contradiction(req: ContradictionRequest) -> dict[str, Any]:
    cap: ContradictionDetectorCapability = app.state.contradiction_detector
    conflict = cap.detect(req.claim, req.source_a, req.source_b)
    if conflict is None:
        return {"conflict": None}
    return {
        "conflict": {
            "id": str(conflict.id),
            "claim": conflict.claim,
            "source_a": {
                "id": str(conflict.source_a_id),
                "value": conflict.source_a_value,
                "url": conflict.source_a_url,
                "retrieved_at": conflict.source_a_retrieved_at.isoformat(),
            },
            "source_b": {
                "id": str(conflict.source_b_id),
                "value": conflict.source_b_value,
                "url": conflict.source_b_url,
                "retrieved_at": conflict.source_b_retrieved_at.isoformat(),
            },
            "resolution_strategy": conflict.resolution_strategy,
            "resolved_at": conflict.resolved_at.isoformat() if conflict.resolved_at else None,
        }
    }


@app.exception_handler(Exception)
async def unhandled_exception_handler(request, exc: Exception):
    log = get_logger(__name__)
    log.exception("unhandled_error", extra={"event": "unhandled_error", "path": request.url.path})
    raise HTTPException(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        detail={"error": "internal_error", "message": str(exc)},
    )
