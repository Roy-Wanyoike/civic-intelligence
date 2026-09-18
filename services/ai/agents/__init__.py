"""Civic Agent Network — a small multi-agent system for evidence-grounded
civic reasoning.

Each agent is a thin, single-purpose component:

  * :class:`ResearchAgent`            — retrieves evidence for a claim/question.
  * :class:`HistoricalComparator`     — finds comparable past bills/scenarios.
  * :class:`AssumptionAuditor`        — flags unsupported assertions.
  * :class:`ScenarioSynthesizer`      — composes an evidence-grounded explanation.

Hard invariants (enforced here, not by convention):

1. Agents **never** write to canonical civic state. They emit ``Result`` objects
   marked ``ai_generated=True``; the Go ``services/intelligence`` consumes them
   via NATS and is the only entity allowed to mutate ``legislation.*`` /
   ``government.*``.
2. Every agent **logs its actions** (input hash, model, tokens, latency, exit
   status) via the shared structured logger.
3. Every agent **respects a token/cost budget**. Calls that would exceed the
   budget raise :class:`BudgetExceeded` before they happen.
4. Every output is explicitly marked **AI-generated** (``Result.ai_generated``).

The registry (:mod:`agents.registry`) is the entry point: register agents at
startup, look them up by name at request time.
"""
from __future__ import annotations

from .base import (
    AgentError,
    BaseAgent,
    Budget,
    BudgetExceeded,
    Permission,
    Result,
    Task,
)
from .registry import (
    AgentMetadata,
    AgentRegistry,
    AgentNotFoundError,
    get_registry,
)

__all__ = [
    "AgentError",
    "AgentMetadata",
    "AgentNotFoundError",
    "AgentRegistry",
    "BaseAgent",
    "Budget",
    "BudgetExceeded",
    "Permission",
    "Result",
    "Task",
    "get_registry",
]

__version__ = "0.1.0"
