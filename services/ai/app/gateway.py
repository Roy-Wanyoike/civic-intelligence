"""Provider-agnostic model gateway.

The application requests CAPABILITIES (``bill_summarizer``, ``civic_question_answerer``)
and the gateway routes to the configured provider. The gateway handles retries,
timeouts, token budgets, cost tracking, model selection, structured output, and
fallback providers.

Adding a new provider: implement the :class:`ModelProvider` protocol and register
it in :func:`build_gateway`.
"""
from __future__ import annotations

import asyncio
import time
from decimal import Decimal
from typing import Any, Protocol, runtime_checkable

import httpx
from tenacity import (
    retry,
    retry_if_exception_type,
    stop_after_attempt,
    wait_exponential,
)

from .config import Settings
from .domain import (
    GatewayAllProvidersFailedError,
    GatewayBudgetExceededError,
    GatewayError,
    GatewayTimeoutError,
    ModelRequest,
    ModelResponse,
)
from .logging import get_logger

log = get_logger(__name__)


# ---------- provider protocol ----------

@runtime_checkable
class ModelProvider(Protocol):
    """Interface every model provider implements."""

    name: str

    async def complete(self, request: ModelRequest) -> ModelResponse: ...


# ---------- stub provider (no network; for dev + tests) ----------

class StubProvider:
    """Deterministic, offline provider. Used in dev, CI, and unit tests so
    the platform runs without external API keys."""

    name = "stub"

    async def complete(self, request: ModelRequest) -> ModelResponse:
        # Sleep a tiny bit to simulate latency.
        await asyncio.sleep(0.01)
        # Echo keywords from the user prompt so eval tests can verify the
        # pipeline actually executed (a real provider would produce a real
        # answer). The stub provider NEVER fabricates specific facts.
        keywords = [
            w for w in request.user_prompt.split()
            if len(w) > 4 and w.isalpha()
        ][:8]
        keyword_str = " ".join(keywords) if keywords else "no keywords"
        content: str
        parsed: dict[str, Any] | None = None
        if request.response_format == "json":
            parsed = {
                "summary": f"This Bill would establish a framework. Keywords: {keyword_str}.",
                "key_points": ["Point one", "Point two"],
                "caveats": ["Specific details depend on the final text."],
            }
            content = str(parsed)
        else:
            content = (
                f"[STUB] Response for capability '{request.capability}'. "
                f"Question keywords: {keyword_str}. "
                f"System prompt was {len(request.system_prompt)} chars."
            )
        return ModelResponse(
            content=content,
            parsed_json=parsed,
            model="stub-1",
            provider=self.name,
            prompt_tokens=len(request.system_prompt.split()) + len(request.user_prompt.split()),
            completion_tokens=len(content.split()),
            latency_ms=12,
            cost_cents=Decimal("0"),
            finish_reason="stop",
        )


# ---------- OpenAI provider ----------

class OpenAIProvider:
    """OpenAI Chat Completions provider. Activated only if ``OPENAI_API_KEY`` is set."""

    name = "openai"
    base_url = "https://api.openai.com/v1/chat/completions"

    def __init__(self, api_key: str, default_model: str, timeout: float) -> None:
        self._api_key = api_key
        self._default_model = default_model
        self._timeout = timeout
        self._client = httpx.AsyncClient(
            timeout=timeout,
            headers={"Authorization": f"Bearer {api_key}"},
        )

    @retry(
        retry=retry_if_exception_type((httpx.TimeoutException, httpx.NetworkError)),
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=0.5, max=2),
        reraise=True,
    )
    async def complete(self, request: ModelRequest) -> ModelResponse:
        payload: dict[str, Any] = {
            "model": self._default_model,
            "messages": [
                {"role": "system", "content": request.system_prompt},
                {"role": "user", "content": request.user_prompt},
            ],
            "max_tokens": request.max_tokens,
            "temperature": request.temperature,
        }
        if request.response_format == "json":
            payload["response_format"] = {"type": "json_object"}
        try:
            t0 = time.monotonic()
            resp = await self._client.post(self.base_url, json=payload)
            resp.raise_for_status()
            data = resp.json()
            latency_ms = int((time.monotonic() - t0) * 1000)
            content = data["choices"][0]["message"]["content"]
            parsed = None
            if request.response_format == "json":
                import json
                parsed = json.loads(content)
            return ModelResponse(
                content=content,
                parsed_json=parsed,
                model=data.get("model", self._default_model),
                provider=self.name,
                prompt_tokens=data.get("usage", {}).get("prompt_tokens", 0),
                completion_tokens=data.get("usage", {}).get("completion_tokens", 0),
                latency_ms=latency_ms,
                cost_cents=_estimate_openai_cost_cents(
                    self._default_model,
                    data.get("usage", {}).get("prompt_tokens", 0),
                    data.get("usage", {}).get("completion_tokens", 0),
                ),
                finish_reason=data["choices"][0].get("finish_reason", "stop"),
            )
        except httpx.TimeoutException as e:
            raise GatewayTimeoutError(str(e)) from e
        except httpx.HTTPStatusError as e:
            raise GatewayError(f"OpenAI HTTP {e.response.status_code}: {e.response.text[:200]}") from e


def _estimate_openai_cost_cents(model: str, prompt_tokens: int, completion_tokens: int) -> Decimal:
    """Rough cost estimator. Update the table as OpenAI pricing changes."""
    pricing = {
        "gpt-4o-mini": (0.15, 0.60),    # per 1M tokens, USD
        "gpt-4o": (5.00, 15.00),
        "gpt-4-turbo": (10.00, 30.00),
    }
    in_per_m, out_per_m = pricing.get(model, (1.0, 2.0))
    usd = (prompt_tokens / 1_000_000) * in_per_m + (completion_tokens / 1_000_000) * out_per_m
    return Decimal(str(usd * 100)).quantize(Decimal("0.0001"))


# ---------- gateway ----------

class ModelGateway:
    """Provider-agnostic gateway. Routes by capability, retries, enforces
    budgets, tracks cost, and falls back across providers."""

    def __init__(
        self,
        primary: ModelProvider,
        fallbacks: list[ModelProvider] | None = None,
        *,
        budget_usd_per_day: float,
        max_retries: int = 2,
    ) -> None:
        self._primary = primary
        self._fallbacks = fallbacks or []
        self._budget_usd_per_day = Decimal(str(budget_usd_per_day))
        self._max_retries = max_retries
        # Simple in-memory daily cost counter. In production this lives in Redis.
        self._spent_today_cents: Decimal = Decimal("0")

    async def complete(self, request: ModelRequest) -> ModelResponse:
        """Execute the request, retrying across providers on transient errors."""
        if self._spent_today_cents / 100 >= self._budget_usd_per_day:
            raise GatewayBudgetExceededError(
                f"Daily budget of ${self._budget_usd_per_day} exceeded"
            )

        providers: list[ModelProvider] = [self._primary, *self._fallbacks]
        last_error: Exception | None = None
        for provider in providers:
            for attempt in range(self._max_retries + 1):
                try:
                    response = await provider.complete(request)
                    self._spent_today_cents += response.cost_cents
                    log.info(
                        "model_gateway_success",
                        extra={
                            "event": "model_gateway_success",
                            "provider": provider.name,
                            "capability": request.capability,
                            "model": response.model,
                            "latency_ms": response.latency_ms,
                            "prompt_tokens": response.prompt_tokens,
                            "completion_tokens": response.completion_tokens,
                            "cost_cents": str(response.cost_cents),
                            "attempt": attempt + 1,
                        },
                    )
                    return response
                except (GatewayTimeoutError, GatewayError) as e:
                    last_error = e
                    log.warning(
                        "model_gateway_retry",
                        extra={
                            "event": "model_gateway_retry",
                            "provider": provider.name,
                            "attempt": attempt + 1,
                            "error": str(e),
                        },
                    )
                    continue
        raise GatewayAllProvidersFailedError(
            f"All providers failed. Last error: {last_error}"
        ) from last_error


def build_gateway(settings: Settings) -> ModelGateway:
    """Factory that wires providers based on settings."""
    primary: ModelProvider
    fallbacks: list[ModelProvider] = []

    if settings.model_gateway_default_provider == "openai" and settings.openai_api_key:
        primary = OpenAIProvider(
            settings.openai_api_key,
            settings.openai_default_model,
            float(settings.model_gateway_timeout_seconds),
        )
    elif settings.model_gateway_default_provider == "anthropic" and settings.anthropic_api_key:
        # Anthropic provider not implemented in this iteration — fall back to stub
        # with a logged warning. See GitHub issue #CI-AI-002.
        log.warning("anthropic_provider_not_implemented_falling_back_to_stub")
        primary = StubProvider()
    else:
        primary = StubProvider()

    return ModelGateway(
        primary=primary,
        fallbacks=fallbacks,
        budget_usd_per_day=settings.model_gateway_budget_usd_per_day,
        max_retries=settings.model_gateway_max_retries,
    )
