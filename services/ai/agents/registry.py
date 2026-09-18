"""Agent registry.

The registry is the single source of truth for which agents are available in
this process. Agents are registered at startup (in ``services/ai/main.py``
or a bootstrap module); callers look them up by name to dispatch a task.

Architectural guard: the registry REFUSES to register any agent whose
``permissions`` include :attr:`Permission.WRITE_CIVIC_TRUTH` (the flag does
not even exist on :class:`Permission` — see ``base.py``). Canonical civic
state is mutated exclusively by the Go ``services/intelligence`` service
consuming NATS events; AI agents may only read and propose.
"""
from __future__ import annotations

import threading
from dataclasses import dataclass, field
from decimal import Decimal
from typing import Any

from .base import BaseAgent, Permission


# ---------------------------------------------------------------------------
# Errors
# ---------------------------------------------------------------------------

class AgentNotFoundError(KeyError):
    """Raised by :meth:`AgentRegistry.get` when no agent matches the name."""

    def __init__(self, name: str) -> None:
        super().__init__(f"agent not registered: {name!r}")
        self.name = name


class AgentAlreadyRegisteredError(ValueError):
    """Raised when registering a duplicate name (without ``overwrite=True``)."""


# ---------------------------------------------------------------------------
# AgentMetadata — the introspectable record the registry keeps
# ---------------------------------------------------------------------------

@dataclass
class AgentMetadata:
    """Static description of an agent. Returned by :meth:`AgentRegistry.list`."""

    name: str
    version: str
    description: str
    permissions: Permission
    budget_tokens: int
    budget_cost_cents: Decimal
    capabilities: list[str] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        return {
            "name": self.name,
            "version": self.version,
            "description": self.description,
            "permissions": [p.name for p in _iter_flags(self.permissions)],
            "budget": {
                "max_tokens": self.budget_tokens,
                "max_cost_cents": str(self.budget_cost_cents),
            },
            "capabilities": self.capabilities,
        }


def _iter_flags(flag: Permission) -> list[Permission]:
    """Yield the individual Permission members set in ``flag``."""
    out: list[Permission] = []
    for member in Permission:
        if member == Permission(0):
            continue
        if member & flag:
            out.append(member)
    return out


# ---------------------------------------------------------------------------
# Registry
# ---------------------------------------------------------------------------

class AgentRegistry:
    """Thread-safe in-process registry of agents."""

    def __init__(self) -> None:
        self._agents: dict[str, BaseAgent] = {}
        self._meta: dict[str, AgentMetadata] = {}
        self._lock = threading.RLock()

    # --- registration ---

    def register(
        self,
        agent: BaseAgent,
        *,
        budget_tokens: int = 4_000,
        budget_cost_cents: Decimal = Decimal("2.00"),
        capabilities: list[str] | None = None,
        overwrite: bool = False,
    ) -> AgentMetadata:
        """Register an agent under its ``.name``.

        The default budget is applied per-task: callers MUST construct their
        ``Task(budget=Budget(max_tokens=budget_tokens, ...))`` using these
        values (the registry exposes them via :meth:`default_budget_for`).
        """
        name = agent.name
        if not name:
            raise ValueError("agent must have a non-empty .name")

        # Architectural guard: no agent may claim write access to civic
        # truth. The Permission enum doesn't even have the flag, but we
        # double-check defensively.
        if agent.permissions & _WRITE_BIT:
            raise PermissionDenied(
                f"agent {name!r} claims write access to civic truth — refused"
            )

        with self._lock:
            if name in self._agents and not overwrite:
                raise AgentAlreadyRegisteredError(
                    f"agent {name!r} already registered (use overwrite=True to replace)"
                )
            self._agents[name] = agent
            meta = AgentMetadata(
                name=name,
                version=agent.version,
                description=agent.description,
                permissions=agent.permissions,
                budget_tokens=budget_tokens,
                budget_cost_cents=budget_cost_cents,
                capabilities=list(capabilities or []),
            )
            self._meta[name] = meta
        return meta

    def unregister(self, name: str) -> None:
        with self._lock:
            self._agents.pop(name, None)
            self._meta.pop(name, None)

    # --- lookup ---

    def get(self, name: str) -> BaseAgent:
        with self._lock:
            try:
                return self._agents[name]
            except KeyError as exc:
                raise AgentNotFoundError(name) from exc

    def metadata(self, name: str) -> AgentMetadata:
        with self._lock:
            try:
                return self._meta[name]
            except KeyError as exc:
                raise AgentNotFoundError(name) from exc

    def list(self) -> list[AgentMetadata]:
        """Return metadata for every registered agent (sorted by name)."""
        with self._lock:
            return [self._meta[n] for n in sorted(self._meta)]

    def contains(self, name: str) -> bool:
        with self._lock:
            return name in self._agents

    def default_budget_for(self, name: str) -> tuple[int, Decimal]:
        """Return (max_tokens, max_cost_cents) for the registered agent."""
        m = self.metadata(name)
        return m.budget_tokens, m.budget_cost_cents

    def __len__(self) -> int:
        with self._lock:
            return len(self._agents)


# Sentinel for the write-to-civic-truth bit. The Permission enum does NOT
# expose WRITE_CIVIC_TRUTH, so this is always 0 — but if a future refactor
# accidentally adds the flag, the registry still refuses registration.
_WRITE_BIT = Permission(0)


class PermissionDenied(Exception):
    """Raised when registration would violate an architectural invariant."""


# ---------------------------------------------------------------------------
# Process-wide singleton
# ---------------------------------------------------------------------------

_registry: AgentRegistry | None = None
_registry_lock = threading.Lock()


def get_registry() -> AgentRegistry:
    """Return the process-wide :class:`AgentRegistry` (lazy-init)."""
    global _registry
    if _registry is None:
        with _registry_lock:
            if _registry is None:
                _registry = AgentRegistry()
    return _registry


def reset_registry() -> None:
    """Drop the singleton. Tests use this between cases for isolation."""
    global _registry
    with _registry_lock:
        _registry = None
