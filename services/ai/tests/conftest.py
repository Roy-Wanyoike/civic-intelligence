"""Test configuration and shared fixtures."""
from __future__ import annotations

import os
import sys
from pathlib import Path

# Ensure the app package is importable.
ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

# Set deterministic env vars BEFORE importing the app.
os.environ.setdefault("AI_MODEL_GATEWAY_DEFAULT_PROVIDER", "stub")
os.environ.setdefault("AI_ENV", "dev")
os.environ.setdefault("AI_LOG_FORMAT", "console")
os.environ.setdefault("AI_CITATION_VALIDATION_REQUIRED", "true")
os.environ.setdefault("AI_CITATION_VALIDATION_FAIL_ON_UNSUPPORTED", "true")
