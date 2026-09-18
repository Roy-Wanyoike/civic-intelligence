"""Conftest for the Civic Agent Network tests.

Adds ``services/ai`` to sys.path so ``from agents.xxx import ...`` and
``from app.xxx import ...`` both work, and seeds the env vars the app
logging/config modules expect at import time.
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]  # services/ai/
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

# Deterministic env vars BEFORE importing the app.
os.environ.setdefault("AI_MODEL_GATEWAY_DEFAULT_PROVIDER", "stub")
os.environ.setdefault("AI_ENV", "dev")
os.environ.setdefault("AI_LOG_FORMAT", "console")
os.environ.setdefault("AI_CITATION_VALIDATION_REQUIRED", "true")
