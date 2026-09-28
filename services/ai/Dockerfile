# Dockerfile for the Civic Intelligence Python AI service (services/ai).
#
# Vercel's built-in Python runtime already runs this service via
# `vercel.json` (entrypoint: main.py). This Dockerfile is the ALTERNATIVE
# path for self-hosting on Railway / Render / Fly.io — useful when you
# want both the Go BFF and the Python AI service on the same platform.
#
# Build from the repo root:
#
#   docker build -f services/ai/Dockerfile -t civic-ai .
#
# Run:
#
#   docker run -p 8000:8000 civic-ai

# ---- Stage 1: build ----------------------------------------------------
FROM python:3.12-slim AS builder

# Install build deps for any C-extension packages in requirements.txt
# (e.g. tiktoken, cryptography). Keep this layer thin.
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Cache pip downloads. Copy just requirements first.
COPY services/ai/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir --user -r requirements.txt

# ---- Stage 2: runtime --------------------------------------------------
FROM python:3.12-slim

# Runtime deps: curl for the healthcheck + ca-certificates for HTTPS.
RUN apt-get update && apt-get install -y --no-install-recommends \
        curl ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && addgroup --system civic && adduser --system --ingroup civic civic

# Copy the pip-installed packages from the builder stage.
COPY --from=builder --chown=civic:civic /root/.local /home/civic/.local

WORKDIR /app
# Copy the application source. Vercel's Python runtime auto-detects main.py
# at the service root — we keep the same layout here for consistency.
COPY --chown=civic:civic services/ai/ ./

# Make the user-installed pip packages importable.
ENV PATH=/home/civic/.local/bin:$PATH \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

USER civic

EXPOSE 8000

# Healthcheck — FastAPI exposes /healthz via app/main.py.
HEALTHCHECK --interval=30s --timeout=2s --start-period=5s --retries=3 \
    CMD curl -fsS http://localhost:8000/healthz || exit 1

# uvicorn serves the FastAPI app. The module path is `app.main:app`
# because services/ai/main.py imports `from app.main import app`.
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
