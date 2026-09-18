"""Tests for the agent registry and the BaseAgent contract."""
from __future__ import annotations

from decimal import Decimal

import pytest

from agents.base import (
    AgentError,
    BaseAgent,
    Budget,
    BudgetExceeded,
    Permission,
    Result,
    Task,
)
from agents.registry import (
    AgentAlreadyRegisteredError,
    AgentNotFoundError,
    AgentRegistry,
    get_registry,
    reset_registry,
)


# ---------------------------------------------------------------------------
# Test fixtures — a tiny stub agent + stub model client
# ---------------------------------------------------------------------------

class _NoopAgent(BaseAgent):
    """Echoes the question back. No external calls."""

    name = "noop"
    version = "0.0.1"
    description = "Test stub."
    permissions = Permission.READ_EVIDENCE

    async def _run(self, task: Task) -> Result:
        return Result(
            agent_name=self.name,
            agent_version=self.version,
            answer=f"echo: {task.question}",
            confidence=1.0,
            metadata={"echoed": True},
        )


@pytest.fixture
def registry() -> AgentRegistry:
    reset_registry()
    return get_registry()


# ---------------------------------------------------------------------------
# Registry: register / get / list / contains
# ---------------------------------------------------------------------------

class TestRegistryRegistration:
    def test_register_then_get(self, registry: AgentRegistry) -> None:
        agent = _NoopAgent()
        meta = registry.register(agent, capabilities=["echo"])
        assert meta.name == "noop"
        assert meta.version == "0.0.1"
        assert "echo" in meta.capabilities

        assert registry.contains("noop")
        assert registry.get("noop") is agent

    def test_duplicate_register_without_overwrite_errors(self, registry: AgentRegistry) -> None:
        registry.register(_NoopAgent())
        with pytest.raises(AgentAlreadyRegisteredError):
            registry.register(_NoopAgent())

    def test_duplicate_register_with_overwrite_replaces(self, registry: AgentRegistry) -> None:
        first = _NoopAgent()
        registry.register(first)
        second = _NoopAgent()
        registry.register(second, overwrite=True)
        assert registry.get("noop") is second

    def test_register_requires_nonempty_name(self, registry: AgentRegistry) -> None:
        agent = _NoopAgent()
        agent.name = ""
        with pytest.raises(ValueError):
            registry.register(agent)

    def test_unregister_removes(self, registry: AgentRegistry) -> None:
        registry.register(_NoopAgent())
        registry.unregister("noop")
        assert not registry.contains("noop")

    def test_get_unknown_raises_typed_error(self, registry: AgentRegistry) -> None:
        with pytest.raises(AgentNotFoundError) as excinfo:
            registry.get("does-not-exist")
        assert excinfo.value.name == "does-not-exist"

    def test_metadata_unknown_raises_typed_error(self, registry: AgentRegistry) -> None:
        with pytest.raises(AgentNotFoundError):
            registry.metadata("does-not-exist")

    def test_list_returns_sorted_metadata(self, registry: AgentRegistry) -> None:
        a = _NoopAgent(); a.name = "zebra"
        b = _NoopAgent(); b.name = "alpha"
        registry.register(a)
        registry.register(b)
        names = [m.name for m in registry.list()]
        assert names == ["alpha", "zebra"]

    def test_metadata_to_dict_shape(self, registry: AgentRegistry) -> None:
        registry.register(_NoopAgent(), budget_tokens=1234, capabilities=["echo"])
        d = registry.metadata("noop").to_dict()
        assert d["name"] == "noop"
        assert d["version"] == "0.0.1"
        assert "READ_EVIDENCE" in d["permissions"]
        assert d["budget"]["max_tokens"] == 1234
        assert d["capabilities"] == ["echo"]

    def test_default_budget_for(self, registry: AgentRegistry) -> None:
        registry.register(
            _NoopAgent(),
            budget_tokens=999,
            budget_cost_cents=Decimal("1.23"),
        )
        tokens, cost = registry.default_budget_for("noop")
        assert tokens == 999
        assert cost == Decimal("1.23")

    def test_len(self, registry: AgentRegistry) -> None:
        assert len(registry) == 0
        registry.register(_NoopAgent())
        assert len(registry) == 1


# ---------------------------------------------------------------------------
# Budget
# ---------------------------------------------------------------------------

class TestBudget:
    def test_default_budget(self) -> None:
        b = Budget()
        assert b.max_tokens == 4_000
        assert b.remaining_tokens == 4_000
        assert b.spent_cost_cents == Decimal("0")

    def test_charge_within_limit(self) -> None:
        b = Budget(max_tokens=100, max_cost_cents=Decimal("1.00"))
        b.charge(40, Decimal("0.40"))
        assert b.remaining_tokens == 60
        assert b.spent_cost_cents == Decimal("0.40")

    def test_charge_exactly_at_limit_ok(self) -> None:
        b = Budget(max_tokens=100, max_cost_cents=Decimal("1.00"))
        b.charge(100, Decimal("1.00"))
        assert b.remaining_tokens == 0

    def test_charge_over_token_limit_raises(self) -> None:
        b = Budget(max_tokens=100, max_cost_cents=Decimal("100"))
        with pytest.raises(BudgetExceeded):
            b.charge(101, Decimal("0"))

    def test_charge_over_cost_limit_raises(self) -> None:
        b = Budget(max_tokens=10_000, max_cost_cents=Decimal("0.50"))
        with pytest.raises(BudgetExceeded):
            b.charge(1, Decimal("0.51"))

    def test_would_exceed(self) -> None:
        b = Budget(max_tokens=100, max_cost_cents=Decimal("1.00"))
        b.charge(50, Decimal("0.50"))
        assert b.would_exceed(60, Decimal("0")) is True   # would breach tokens
        assert b.would_exceed(50, Decimal("0")) is False  # exactly at limit OK
        assert b.would_exceed(0, Decimal("0.51")) is True  # would breach cost

    def test_charge_negative_tokens_raises(self) -> None:
        b = Budget()
        with pytest.raises(AgentError):
            b.charge(-1, Decimal("0"))


# ---------------------------------------------------------------------------
# BaseAgent.run contract
# ---------------------------------------------------------------------------

class TestBaseAgentRun:
    @pytest.mark.asyncio
    async def test_run_stamps_provenance(self, registry: AgentRegistry) -> None:
        agent = _NoopAgent()
        registry.register(agent)
        task = Task(question="hello", context={"bill_id": "B-1"})

        result = await agent.run(task)
        assert result.agent_name == "noop"
        assert result.agent_version == "0.0.1"
        assert result.ai_generated is True
        assert result.correlation_id == task.correlation_id
        assert result.duration_ms >= 0
        assert "hello" in result.answer

    @pytest.mark.asyncio
    async def test_run_converts_unexpected_exception_to_agent_error(self) -> None:
        class BoomAgent(BaseAgent):
            name = "boom"
            version = "0.0.1"
            permissions = Permission(0)

            async def _run(self, task: Task) -> Result:
                raise RuntimeError("unexpected")

        agent = BoomAgent()
        with pytest.raises(AgentError) as excinfo:
            await agent.run(Task(question="x"))
        assert "unexpected" in str(excinfo.value)

    @pytest.mark.asyncio
    async def test_agent_error_propagates_unwrapped(self) -> None:
        class BudgetAgent(BaseAgent):
            name = "budget"
            version = "0.0.1"
            permissions = Permission(0)

            async def _run(self, task: Task) -> Result:
                raise BudgetExceeded("simulated")

        agent = BudgetAgent()
        with pytest.raises(BudgetExceeded):
            await agent.run(Task(question="x"))


# ---------------------------------------------------------------------------
# Architectural invariant: no agent may claim write access to civic truth
# ---------------------------------------------------------------------------

class TestNoWriteToCivicTruth:
    def test_permission_enum_has_no_write_civic_truth(self) -> None:
        # The Permission enum must NOT have a WRITE_CIVIC_TRUTH member.
        assert not hasattr(Permission, "WRITE_CIVIC_TRUTH")

    def test_every_result_is_marked_ai_generated(self) -> None:
        # Even if a subclass tries to set ai_generated=False, the wrapper
        # overrides it. Verify by constructing a Result with False and
        # checking what the wrapper produces.
        class LyingAgent(BaseAgent):
            name = "lying"
            version = "0.0.1"
            permissions = Permission(0)

            async def _run(self, task: Task) -> Result:
                r = Result(agent_name="lying", agent_version="0.0.1", answer="x")
                r.ai_generated = False  # attempt to lie
                return r

        import asyncio

        agent = LyingAgent()
        result = asyncio.run(agent.run(Task(question="x")))
        assert result.ai_generated is True
