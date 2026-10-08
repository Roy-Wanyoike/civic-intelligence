"""Tests for the model gateway: budget enforcement, retries, stub provider."""
from __future__ import annotations

from decimal import Decimal
from unittest.mock import AsyncMock

import pytest

from app.domain import GatewayBudgetExceededError, ModelRequest, ModelResponse
from app.gateway import ModelGateway, StubProvider, _normalize_qwen_base_url


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


# ---------- Qwen URL normaliser ----------

# The normaliser accepts both the OpenAI-style base URL and the full
# endpoint URL so users can paste whatever ModelScope's dashboard gives
# them. This locks the contract in so a future refactor can't silently
# break either form.

def test_normalize_qwen_base_url_appends_path_when_missing():
    """The OpenAI-style base URL (without /chat/completions) gets the
    path appended. This is the form the user-facing dashboard surfaces."""
    result = _normalize_qwen_base_url("https://api-inference.modelscope.ai/v1")
    assert result == "https://api-inference.modelscope.ai/v1/chat/completions"


def test_normalize_qwen_base_url_passes_through_when_present():
    """The full endpoint URL passes through unchanged. This preserves
    backward compatibility with .env files that already use the full URL."""
    full = "https://api-inference.modelscope.ai/v1/chat/completions"
    assert _normalize_qwen_base_url(full) == full


def test_normalize_qwen_base_url_strips_trailing_slash():
    """Trailing slashes are stripped before the path check so both
    ``/v1`` and ``/v1/`` normalise identically."""
    assert _normalize_qwen_base_url("https://api-inference.modelscope.ai/v1/") == (
        "https://api-inference.modelscope.ai/v1/chat/completions"
    )


def test_normalize_qwen_base_url_handles_empty_string():
    """An empty string falls back to the bare path. This is a defensive
    default — Settings' field validator catches empty strings before
    they reach the gateway in practice."""
    assert _normalize_qwen_base_url("") == "chat/completions"


def test_normalize_qwen_base_url_handles_custom_host():
    """Self-hosted ModelScope deployments (rare but possible) are
    accepted as long as they follow the same path convention."""
    assert _normalize_qwen_base_url("https://my-modelscope.internal/api/v1") == (
        "https://my-modelscope.internal/api/v1/chat/completions"
    )
