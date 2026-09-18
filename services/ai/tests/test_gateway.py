"""Tests for the model gateway: budget enforcement, retries, stub provider."""
from __future__ import annotations

from decimal import Decimal
from unittest.mock import AsyncMock

import pytest

from app.domain import GatewayBudgetExceededError, ModelRequest, ModelResponse
from app.gateway import ModelGateway, StubProvider


@pytest.mark.asyncio
async def test_stub_provider_returns_response():
    provider = StubProvider()
    req = ModelRequest(
        capability="bill_summarizer",
        system_prompt="you are helpful",
        user_prompt="summarize this bill",
        response_format="text",
    )
    resp = await provider.complete(req)
    assert resp.provider == "stub"
    assert resp.model == "stub-1"
    assert resp.content  # non-empty
    assert resp.finish_reason == "stop"


@pytest.mark.asyncio
async def test_gateway_enforces_daily_budget():
    # Set an absurdly low budget so the very first request is rejected.
    gw = ModelGateway(
        primary=StubProvider(),
        fallbacks=[],
        budget_usd_per_day=0.0,
        max_retries=0,
    )
    req = ModelRequest(
        capability="bill_summarizer",
        system_prompt="x",
        user_prompt="y",
    )
    with pytest.raises(GatewayBudgetExceededError):
        await gw.complete(req)


@pytest.mark.asyncio
async def test_gateway_tracks_spent_budget():
    gw = ModelGateway(
        primary=StubProvider(),
        fallbacks=[],
        budget_usd_per_day=10.0,
        max_retries=0,
    )
    req = ModelRequest(
        capability="bill_summarizer",
        system_prompt="hello",
        user_prompt="world",
    )
    # Stub provider returns cost_cents=0, so this should not raise.
    resp = await gw.complete(req)
    assert resp.provider == "stub"
    # Stub provider has zero cost, so budget should still be intact.
    assert gw._spent_today_cents == Decimal("0")
