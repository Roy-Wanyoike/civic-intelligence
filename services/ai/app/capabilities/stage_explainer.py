"""StageExplainer — explains what a parliamentary stage means in plain language.

The stage definitions come from the country adapter (Kenya). This capability
NEVER hard-codes stage names — it receives a ``StageDefinition`` and produces
a plain-language explanation that fits the user's question context.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from ..config import Settings
from ..domain import Citation, Confidence
from ..gateway import ModelGateway


@dataclass
class StageDefinition:
    code: str           # e.g., "SECOND_READING" — country-specific
    name: str           # e.g., "Second Reading"
    simple_explanation: str
    official_definition: str | None
    country: str        # ISO code
    allowed_next: list[str]
    sources: list[str]  # URLs


class StageExplainerCapability:
    capability_name = "stage_explainer"

    def __init__(self, settings: Settings, gateway: ModelGateway) -> None:
        self._settings = settings
        self._gateway = gateway

    async def explain(self, stage: StageDefinition, *, bill_title: str | None = None) -> str:
        # For the stub provider we return the simple_explanation directly.
        # For real providers we ask the LLM to weave in bill context.
        if self._settings.model_gateway_default_provider == "stub":
            base = stage.simple_explanation
            if bill_title:
                base = f"For the Bill \"{bill_title}\", the current stage is {stage.name}. {base}"
            return base
        # Real LLM path
        request_text = (
            f"Explain the parliamentary stage '{stage.name}' ({stage.code}) for country {stage.country} "
            f"in plain language for a citizen. Bill context: {bill_title or 'general'}. "
            f"Official definition: {stage.official_definition or 'not available'}."
        )
        from ..domain import ModelRequest
        resp = await self._gateway.complete(
            ModelRequest(
                capability=self.capability_name,
                system_prompt="You explain parliamentary stages in plain Kenyan English. Never fabricate.",
                user_prompt=request_text,
                max_tokens=300,
                temperature=0.0,
            )
        )
        return resp.content
