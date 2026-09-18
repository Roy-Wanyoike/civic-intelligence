"""Structured logging. JSON in staging/prod, human-readable in dev."""
from __future__ import annotations

import logging
import sys
from typing import Any

from pythonjsonlogger import jsonlogger

from .config import get_settings


def configure_logging() -> None:
    """Configure the root logger. Idempotent."""
    settings = get_settings()
    level = getattr(logging, settings.log_level.upper(), logging.INFO)

    handler = logging.StreamHandler(sys.stdout)
    if settings.log_format == "json" and settings.env != "dev":
        formatter: logging.Formatter = jsonlogger.JsonFormatter(
            "%(asctime)s %(name)s %(levelname)s %(message)s %(filename)s:%(lineno)d",
            rename_fields={"asctime": "ts", "levelname": "level", "name": "logger"},
        )
    else:
        formatter = logging.Formatter(
            "%(asctime)s %(levelname)-8s %(name)s :: %(message)s"
        )
    handler.setFormatter(formatter)

    root = logging.getLogger()
    root.handlers.clear()
    root.addHandler(handler)
    root.setLevel(level)
    # Quiet noisy libraries.
    logging.getLogger("httpx").setLevel(logging.WARNING)
    logging.getLogger("httpcore").setLevel(logging.WARNING)
    logging.getLogger("uvicorn.access").setLevel(logging.WARNING)


def get_logger(name: str) -> logging.Logger:
    """Convenience accessor used by every module."""
    return logging.getLogger(name)


def log_event(logger: logging.Logger, event: str, **fields: Any) -> None:
    """Emit a structured event log line. Fields become top-level JSON keys."""
    logger.info(event, extra={"event": event, **fields})
