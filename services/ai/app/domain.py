"""Domain types shared across the AI service.

These mirror the contracts in ``packages/contracts`` (Go) so the Python and Go
halves of the platform speak the same language. They are deliberately pure —
no infrastructure imports.
"""
from __future__ import annotations

from datetime import datetime
from decimal import Decimal
from enum import Enum
from typing import Any, Literal
from uuid import UUID, uuid4

from pydantic import BaseModel, Field, HttpUrl


# ---------- enums ----------

class ClaimType(str, Enum):
    FACT = "fact"
    STAGE = "stage"
    DATE = "date"
    ENTITY = "entity"
    DEFINITION = "definition"
    INTERPRETATION = "interpretation"


class Confidence(str, Enum):
    """How strongly the evidence supports the claim."""

    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"
    UNKNOWN = "unknown"


class SourceType(str, Enum):
    PARLIAMENT = "parliament"
    GAZETTE = "gazette"
    KENYA_LAW = "kenya_law"
    HANSARD = "hansard"
    COMMITTEE_REPORT = "committee_report"
    ORDER_PAPER = "order_paper"
    VOTES_PROCEEDINGS = "votes_proceedings"


class ValidationStatus(str, Enum):
    PENDING = "pending"
    PASSED = "passed"
    FAILED = "failed"
    PARTIAL = "partial"


# ---------- evidence ----------

class Citation(BaseModel):
    """A pointer to a specific span in an authoritative document."""

    document_id: UUID
    source_url: HttpUrl
    page_number: int | None = None
    section: str | None = None
    snippet: str = Field(..., description="Verbatim text from the source")
    snippet_offset_start: int | None = None
    snippet_offset_end: int | None = None
    source_type: SourceType
    retrieved_at: datetime


class Claim(BaseModel):
    """A single factual assertion made by the AI. Each claim MUST be backed by
    at least one citation, otherwise the response fails validation."""

    id: UUID = Field(default_factory=uuid4)
    type: ClaimType
    text: str
    citations: list[Citation] = Field(default_factory=list)
    confidence: Confidence = Confidence.UNKNOWN

    def is_supported(self) -> bool:
        return bool(self.citations) and self.confidence != Confidence.UNKNOWN


class CandidateFact(BaseModel):
    """A fact the AI PROPOSES. The Go ``services/intelligence`` consumes these
    via NATS and decides whether to accept them into canonical state. AI never
    writes to canonical state directly."""

    id: UUID = Field(default_factory=uuid4)
    payload: dict[str, Any]
    evidence: list[Citation]
    proposed_at: datetime = Field(default_factory=datetime.utcnow)
    proposed_by: str = "ai-service"


# ---------- RAG ----------

class RetrievedChunk(BaseModel):
    """A chunk returned from hybrid (FTS + pgvector) search."""

    document_id: UUID
    chunk_id: UUID
    text: str
    page_number: int | None
    section: str | None
    source_url: HttpUrl
    source_type: SourceType
    semantic_score: float
    keyword_score: float
    combined_score: float


class RAGContext(BaseModel):
    """The context window built from retrieved chunks and passed to the LLM."""

    chunks: list[RetrievedChunk]
    citations: list[Citation]
    token_count: int


class Question(BaseModel):
    """An incoming citizen question."""

    text: str
    bill_id: UUID | None = None
    country: str = "KE"
    impact_lens: Literal[
        "citizen",
        "student",
        "small_business",
        "employee",
        "farmer",
        "driver",
        "tenant",
        "homeowner",
        "employer",
        "digital_worker",
    ] | None = None


# ---------- AI responses ----------

class AIResponse(BaseModel):
    """A response from the AI gateway. Always carries citations and a
    validation status. ``validated=False`` responses must NOT be surfaced to
    citizens as factual claims."""

    id: UUID = Field(default_factory=uuid4)
    question_id: UUID | None = None
    answer: str
    claims: list[Claim] = Field(default_factory=list)
    citations: list[Citation] = Field(default_factory=list)
    rag_context: RAGContext | None = None
    validated: bool = False
    validation_status: ValidationStatus = ValidationStatus.PENDING
    validation_failures: list[str] = Field(default_factory=list)
    model: str
    provider: str
    prompt_tokens: int = 0
    completion_tokens: int = 0
    latency_ms: int = 0
    cost_cents: Decimal = Decimal("0")
    created_at: datetime = Field(default_factory=datetime.utcnow)


class BillSummary(BaseModel):
    """Plain-language summary of a Bill."""

    bill_id: UUID
    short_title: str
    one_line_summary: str
    plain_language_explanation: str
    current_stage_explained: str
    what_it_would_do: list[str]
    who_it_affects: list[str]
    what_happens_next: list[str]
    citations: list[Citation]
    confidence: Confidence
    validated: bool
    validation_failures: list[str] = Field(default_factory=list)


class BillDiff(BaseModel):
    """A plain-language explanation of what changed between two Bill versions."""

    bill_id: UUID
    from_version: str
    to_version: str
    additions: list[str] = Field(default_factory=list, description="New clause text")
    removals: list[str] = Field(default_factory=list, description="Removed clause text")
    modifications: list[str] = Field(default_factory=list)
    affected_clauses: list[str] = Field(default_factory=list)
    plain_language_explanation: str
    citations: list[Citation]
    # We never explain WHY a change was made unless an authoritative source
    # states the reason. This is a hard architectural rule.
    stated_reason: str | None = None
    validated: bool


class BriefingItem(BaseModel):
    """A single item in the daily civic briefing."""

    kind: Literal["new_bill", "stage_change", "amendment", "new_document", "enacted", "committee"]
    title: str
    description: str
    bill_id: UUID | None = None
    citations: list[Citation]
    significance: str  # plain-language "why this matters"
    confidence: Confidence


class DailyBriefing(BaseModel):
    """Kenya Civic Brief — generated daily from verified events."""

    country: str
    date: datetime
    headline: str
    items: list[BriefingItem]
    generated_at: datetime = Field(default_factory=datetime.utcnow)


# ---------- model gateway ----------

class ModelRequest(BaseModel):
    """A request to the provider-agnostic model gateway."""

    capability: str  # e.g. "bill_summarizer", "civic_question_answerer"
    system_prompt: str
    user_prompt: str
    max_tokens: int = 1024
    temperature: float = 0.0
    response_format: Literal["text", "json"] = "text"
    json_schema: dict[str, Any] | None = None


class ModelResponse(BaseModel):
    """A response from the model gateway — provider-agnostic."""

    content: str
    parsed_json: dict[str, Any] | None = None
    model: str
    provider: str
    prompt_tokens: int = 0
    completion_tokens: int = 0
    latency_ms: int = 0
    cost_cents: Decimal = Decimal("0")
    finish_reason: str = "stop"


class GatewayError(Exception):
    """Base error for model gateway failures."""


class GatewayTimeoutError(GatewayError):
    pass


class GatewayBudgetExceededError(GatewayError):
    pass


class GatewayAllProvidersFailedError(GatewayError):
    pass
