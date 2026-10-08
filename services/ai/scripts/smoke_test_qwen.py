"""End-to-end smoke test for the QwenProvider.

Runs from anywhere in the repo so the import paths resolve correctly:

    cd services/ai
    python3 scripts/smoke_test_qwen.py

The script:
1. Loads Settings from services/ai/.env (the gitignored file with the
   real API key).
2. Builds the QwenProvider via the same `build_gateway` factory the AI
   service uses.
3. Sends a minimal chat completion request.
4. Prints the response (model, content, latency, token usage).
5. Exits 0 on success, 1 on failure.

This is NOT a unit test — it makes a real network call to ModelScope.
The AI service's pytest suite has stub-based unit tests for the gateway;
this script is the manual integration check we run before declaring the
Qwen credentials "live".
"""

from __future__ import annotations

import asyncio
import os
import sys
from pathlib import Path

# Add services/ai to sys.path so the `app` package imports resolve.
# This script lives at services/ai/scripts/smoke_test_qwen.py, so the
# parent of the parent is services/ai.
AI_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(AI_ROOT))

# Set the working directory to services/ai so Settings' env_file=.env
# picks up the gitignored .env file.
os.chdir(AI_ROOT)

from app.config import get_settings  # noqa: E402
from app.domain import ModelRequest  # noqa: E402
from app.gateway import (  # noqa: E402
    QWEN_CHAT_COMPLETIONS_PATH,
    _normalize_qwen_base_url,
    build_gateway,
)


async def main() -> int:
    settings = get_settings()

    print("=" * 60)
    print("QwenProvider end-to-end smoke test")
    print("=" * 60)
    print(f"  provider:        {settings.model_gateway_default_provider}")
    print(f"  base_url (cfg):  {settings.qwen_base_url}")
    print(f"  endpoint (norm): {_normalize_qwen_base_url(settings.qwen_base_url)}")
    print(f"  model:           {settings.qwen_default_model}")
    print(f"  api_key prefix:  {settings.qwen_api_key[:10]}…{settings.qwen_api_key[-4:]}")
    print(f"  timeout:         {settings.model_gateway_timeout_seconds}s")
    print(f"  max_retries:     {settings.model_gateway_max_retries}")
    print()

    # Sanity-check the URL normaliser behaves correctly.
    assert _normalize_qwen_base_url("https://api-inference.modelscope.ai/v1") == (
        f"https://api-inference.modelscope.ai/v1{QWEN_CHAT_COMPLETIONS_PATH}"
    ), "normaliser failed for base URL form"
    assert _normalize_qwen_base_url(
        "https://api-inference.modelscope.ai/v1/chat/completions"
    ) == "https://api-inference.modelscope.ai/v1/chat/completions", (
        "normaliser failed for full endpoint form"
    )
    print("✓ URL normaliser handles both base + full endpoint forms")
    print()

    if not settings.qwen_api_key:
        print("✗ AI_QWEN_API_KEY is empty — set it in services/ai/.env")
        return 1

    gateway = build_gateway(settings)

    request = ModelRequest(
        capability="civic_question_answerer",
        system_prompt=(
            "You are a helpful assistant. Answer the user's question in "
            "one short sentence."
        ),
        user_prompt="What is the capital of Kenya?",
        max_tokens=64,
        temperature=0.0,
        response_format="text",
    )

    print("Sending chat completion request…")
    print()

    try:
        response = await gateway.complete(request)
    except Exception as exc:
        print(f"✗ Request failed: {type(exc).__name__}: {exc}")
        return 1

    print("✓ Response received")
    print(f"  provider:          {response.provider}")
    print(f"  model:             {response.model}")
    print(f"  latency_ms:        {response.latency_ms}")
    print(f"  prompt_tokens:     {response.prompt_tokens}")
    print(f"  completion_tokens: {response.completion_tokens}")
    print(f"  cost_cents:        {response.cost_cents}")
    print(f"  finish_reason:     {response.finish_reason}")
    print()
    print("--- content ---")
    print(response.content)
    print("--- /content ---")

    return 0


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
