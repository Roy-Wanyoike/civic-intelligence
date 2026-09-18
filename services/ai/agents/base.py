"""Base classes for the Civic Agent Network.

Defines the contract every agent must satisfy:

  * :class:`Task`  — the input envelope (question + context + budget).
  * :class:`Result` — the output envelope (always marked ``ai_generated=True``).
  * :class:`Budget` — token + cost cap; checked before every model call.
  * :class:`BaseAgent` — abstract base. Concrete agents implement :meth:`run`.

The base class wires structured logging, budget enforcement, and provenance
stamping so individual agents don't have to repeat that boilerplate.
"""
from __future__ import annotations

import abc
import hashlib
import time
from dataclasses import dataclass, field
from decimal import Decimal
from enum import Flag, auto
from typing import Any, Protocol
from uuid import UUID, uuid4

from app.logging import get_logger

log = get_logger(__name__)


# ---------------------------------------------------------------------------
# Errors
# ---------------------------------------------------------------------------

class AgentError(Exception):
    """Base error for any agent failure."""


class BudgetExceeded(AgentError):
    """Raised when an agent would exceed its configured token/cost budget.

    Agents MUST check the budget before every external (model) call and raise
    this proactively rather than discovering the breach after the fact.
    """


class PermissionDenied(AgentError):
    """Raised when an agent is asked to do something outside its permissions."""


# ---------------------------------------------------------------------------
# Permissions — what an agent is allowed to do
# ---------------------------------------------------------------------------

class Permission(Flag):
    """Capability flags. Agents declare the union they need.

    The hard contract is that NO agent has :attr:`WRITE_CIVIC_TRUTH` — the
    registry refuses to register any agent claiming it. Canonical state is
    owned by the Go ``services/intelligence`` service exclusively.
    """

    READ_EVIDENCE = auto()
    SEARCH_DOCUMENTS = auto()
    READ_LEGISLATION = auto()
    READ_SCENARIOS = auto()
    CALL_MODEL = auto()
    WRITE_AUDIT_LOG = auto()
    # WRITE_CIVIC_TRUTH deliberately omitted — no agent may write canonical
    # state. If you need to mutate legislation.*, do it via NATS events to
    # the Go intelligence service.


# ---------------------------------------------------------------------------
# Budget
# ---------------------------------------------------------------------------

@dataclass
class Budget:
    """Per-task budget. Defaults are tight; callers can override.

    ``remaining_tokens`` is mutated as the agent runs. Mutations are NOT
    thread-safe — each Task gets its own Budget instance.
    """

    max_tokens: int = 4_000
    max_cost_cents: Decimal = Decimal("2.00")
    remaining_tokens: int = field(init=False)
    spent_cost_cents: Decimal = field(default_factory=lambda: Decimal("0"))

    def __post_init__(self) -> None:
        self.remaining_tokens = self.max_tokens

    def charge(self, tokens: int, cost_cents: Decimal) -> None:
        """Record token + cost usage from a model call.

        Raises :class:`BudgetExceeded` if the call would push us over either
        limit. Callers MUST call :meth:`would_exceed` BEFORE invoking the model
        to avoid the wasted round-trip.
        """
        if tokens < 0:
            raise AgentError(f"budget: negative tokens ({tokens})")
        self.remaining_tokens -= tokens
        self.spent_cost_cents += cost_cents
        if self.remaining_tokens < 0:
            raise BudgetExceeded(
                f"token budget exceeded: used {-self.remaining_tokens} over limit "
                f"{self.max_tokens}"
            )
        if self.spent_cost_cents > self.max_cost_cents:
            raise BudgetExceeded(
                f"cost budget exceeded: {self.spent_cost_cents}¢ > "
                f"{self.max_cost_cents}¢"
            )

    def would_exceed(self, tokens: int, cost_cents: Decimal) -> bool:
        """True if a call using ``tokens`` / ``cost_cents`` would exceed."""
        return (
            tokens > self.remaining_tokens
            or self.spent_cost_cents + cost_cents > self.max_cost_cents
        )

    def remaining(self) -> tuple[int, Decimal]:
        """Return (remaining_tokens, remaining_cost_cents)."""
        return self.remaining_tokens, self.max_cost_cents - self.spent_cost_cents


# ---------------------------------------------------------------------------
# Task / Result envelopes
# ---------------------------------------------------------------------------

@dataclass
class Task:
    """Input to an agent.

    ``question`` is the natural-language ask.
    ``context`` is arbitrary structured context (bill_id, scenario_id, ...).
    ``budget`` caps token + cost usage for the whole task.
    ``correlation_id`` propagates through logs.
    """

    question: str
    context: dict[str, Any] = field(default_factory=dict)
    budget: Budget = field(default_factory=Budget)
    correlation_id: str = field(default_factory=lambda: uuid4().hex)
    requested_by: str = "anonymous"

    def input_hash(self) -> str:
        """Stable short hash of the question + context for log dedup."""
        h = hashlib.sha256()
        h.update(self.question.encode("utf-8"))
        for k in sorted(self.context):
            h.update(f"{k}={self.context[k]}".encode("utf-8"))
        return h.hexdigest()[:12]


@dataclass
class Result:
    """Output of an agent.

    ``ai_generated`` is ALWAYS True — the base class sets it; agents cannot
    unset it. This is the architectural stamp that downstream consumers
    (and the citation validator) rely on.
    """

    agent_name: str
    agent_version: str
    answer: str
    citations: list[dict[str, Any]] = field(default_factory=list)
    evidence_refs: list[str] = field(default_factory=list)
    assumptions: list[str] = field(default_factory=list)
    confidence: float = 0.0
    metadata: dict[str, Any] = field(default_factory=dict)
    ai_generated: bool = True  # always True — see class docstring
    correlation_id: str = ""
    duration_ms: int = 0

    def to_dict(self) -> dict[str, Any]:
        """Serialise for logging / event publication."""
        return {
            "agent": self.agent_name,
            "version": self.agent_version,
            "answer": self.answer,
            "citations": self.citations,
            "evidence_refs": self.evidence_refs,
            "assumptions": self.assumptions,
            "confidence": self.confidence,
            "metadata": self.metadata,
            "ai_generated": self.ai_generated,
            "correlation_id": self.correlation_id,
            "duration_ms": self.duration_ms,
        }


# ---------------------------------------------------------------------------
# ModelClient + EvidenceClient protocols — what agents may call
# ---------------------------------------------------------------------------

class ModelClient(Protocol):
    """Minimal model-gateway interface agents may use.

    Real impl = ``app.gateway.ModelGateway``. Tests substitute a stub.
    """

    async def complete(
        self,
        system_prompt: str,
        user_prompt: str,
        max_tokens: int = 512,
        temperature: float = 0.0,
    ) -> tuple[str, int, int, Decimal]:
        """Return (content, prompt_tokens, completion_tokens, cost_cents)."""
        ...


class EvidenceClient(Protocol):
    """Minimal evidence-retrieval interface agents may use.

    Real impl = ``app.evidence_store.EvidenceStore``. Tests substitute a stub.
    """

    async def retrieve(
        self, question: str, bill_id: str | None = None, top_k: int = 8
    ) -> list[dict[str, Any]]:
        """Return a list of evidence chunks (dicts with at least `text`
        and `source_url`)."""
        ...


class LegislationClient(Protocol):
    """Read-only client over the legislation service."""

    async def get_bill(self, bill_id: str) -> dict[str, Any] | None: ...
    async def search_bills(self, query: str, limit: int = 10) -> list[dict[str, Any]]: ...


class ScenarioClient(Protocol):
    """Read-only client over the simulation service."""

    async def list_scenarios(
        self, bill_id: str | None = None, limit: int = 20
    ) -> list[dict[str, Any]]: ...


# ---------------------------------------------------------------------------
# BaseAgent
# ---------------------------------------------------------------------------

class BaseAgent(abc.ABC):
    """Abstract base for every Civic Agent.

    Subclasses implement :meth:`_run` (the actual work). The public
    :meth:`run` wrapper:

      1. Logs the task start (correlation_id, input_hash, budget).
      2. Calls ``_run``.
      3. Stamps the ``Result`` with ``ai_generated=True`` and timing.
      4. Logs the task end (success/failure, tokens, cost).
      5. Re-raises any :class:`AgentError` subclass; converts unexpected
         exceptions to :class:`AgentError`.

    Subclasses never touch ``Result.ai_generated`` directly.
    """

    #: Agent name (matches registry key).
    name: str = "base"
    #: Semver-style version string.
    version: str = "0.1.0"
    #: Short human description shown in /agents introspection endpoint.
    description: str = ""
    #: Permissions the agent needs. The registry checks this against a
    #: deny-list on registration.
    permissions: Permission = Permission(0)

    def __init__(
        self,
        *,
        model: ModelClient | None = None,
        evidence: EvidenceClient | None = None,
        legislation: LegislationClient | None = None,
        scenarios: ScenarioClient | None = None,
    ) -> None:
        self._model = model
        self._evidence = evidence
        self._legislation = legislation
        self._scenarios = scenarios
        self._log = get_logger(f"agents.{self.name}")

    async def run(self, task: Task) -> Result:
        """Execute the task. See class docstring for the wrapper contract."""
        start = time.monotonic()
        self._log.info(
            "agent.task.start",
            extra={
                "agent": self.name,
                "version": self.version,
                "correlation_id": task.correlation_id,
                "input_hash": task.input_hash(),
                "budget_tokens": task.budget.max_tokens,
                "requested_by": task.requested_by,
            },
        )

        try:
            result = await self._run(task)
        except AgentError:
            self._log.warning(
                "agent.task.failed",
                extra={
                    "agent": self.name,
                    "correlation_id": task.correlation_id,
                    "elapsed_ms": int((time.monotonic() - start) * 1000),
                },
            )
            raise
        except Exception as exc:  # noqa: BLE001
            self._log.error(
                "agent.task.crashed",
                extra={
                    "agent": self.name,
                    "correlation_id": task.correlation_id,
                    "error": str(exc),
                    "elapsed_ms": int((time.monotonic() - start) * 1000),
                },
            )
            raise AgentError(f"{self.name}: {exc}") from exc

        # Stamp the wrapper fields. ``ai_generated`` is always True.
        result.agent_name = self.name
        result.agent_version = self.version
        result.ai_generated = True
        result.correlation_id = task.correlation_id
        result.duration_ms = int((time.monotonic() - start) * 1000)

        self._log.info(
            "agent.task.end",
            extra={
                "agent": self.name,
                "correlation_id": task.correlation_id,
                "elapsed_ms": result.duration_ms,
                "tokens_remaining": task.budget.remaining()[0],
                "cost_spent_cents": str(task.budget.spent_cost_cents),
                "citations": len(result.citations),
            },
        )
        return result

    @abc.abstractmethod
    async def _run(self, task: Task) -> Result:
        """Subclass implements the actual work. Must return a Result.

        The wrapper stamps ``agent_name``, ``agent_version``,
        ``ai_generated``, ``correlation_id``, and ``duration_ms`` — subclasses
        need not set those.
        """
        ...

    # ---- convenience helpers for subclasses ----

    def _charge_model(
        self, task: Task, prompt_tokens: int, completion_tokens: int, cost_cents: Decimal
    ) -> None:
        """Record a model call against the task's budget.

        Raises :class:`BudgetExceeded` if the call would breach the cap.
        Subclasses call this AFTER every model invocation; the base class
        expects subclasses to check :meth:`Budget.would_exceed` BEFORE the
        call (so they don't waste a round-trip).
        """
        task.budget.charge(prompt_tokens + completion_tokens, cost_cents)
