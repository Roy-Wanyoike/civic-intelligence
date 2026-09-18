"""Application configuration loaded from environment.

All settings are pydantic-settings models. The service refuses to start if a
required setting is missing — fail-fast is preferable to silent misbehavior.
"""
from __future__ import annotations

from functools import lru_cache
from typing import Literal

from pydantic import Field, field_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Runtime configuration. Loaded once, cached via :func:`get_settings`."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_prefix="AI_",
        env_nested_delimiter="__",
        case_sensitive=False,
        extra="ignore",
        protected_namespaces=("settings_",),
    )

    # --- service ---
    service_name: str = "ai"
    service_port: int = 8000
    env: Literal["dev", "staging", "prod"] = "dev"
    log_level: str = "INFO"
    log_format: Literal["json", "console"] = "json"

    # --- upstream services ---
    legislation_service_url: str = "http://localhost:9001"
    documents_service_url: str = "http://localhost:9002"
    evidence_service_url: str = "http://localhost:9003"
    search_service_url: str = "http://localhost:9004"

    # --- database (read-only consumer) ---
    database_url: str = "postgresql://civic:civic@localhost:5432/civic_intelligence"
    database_pool_size: int = 10
    database_statement_timeout_ms: int = 5000

    # --- model gateway ---
    model_gateway_default_provider: Literal["stub", "openai", "anthropic"] = "stub"
    model_gateway_timeout_seconds: int = 30
    model_gateway_max_retries: int = 2
    model_gateway_budget_usd_per_day: float = 20.0

    openai_api_key: str = ""
    openai_default_model: str = "gpt-4o-mini"
    anthropic_api_key: str = ""
    anthropic_default_model: str = "claude-3-5-haiku-latest"

    # --- RAG ---
    rag_top_k: int = 8
    rag_min_score: float = 0.35
    rag_rerank_top_k: int = 4
    rag_max_context_tokens: int = 6000

    # --- citation validation ---
    citation_validation_required: bool = True
    citation_validation_fail_on_unsupported: bool = True

    # --- telemetry ---
    otel_exporter_otlp_endpoint: str = ""
    otel_service_name: str = "civic-ai"

    @field_validator("model_gateway_budget_usd_per_day")
    @classmethod
    def _budget_positive(cls, v: float) -> float:
        if v <= 0:
            raise ValueError("model_gateway_budget_usd_per_day must be positive")
        return v


@lru_cache
def get_settings() -> Settings:
    """Return a cached Settings instance. Use this everywhere — never call
    ``Settings()`` directly outside of tests.
    """
    return Settings()
