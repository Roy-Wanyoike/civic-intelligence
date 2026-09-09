"""Vercel serverless function entry point for the AI service.

Vercel's Python runtime looks for .py files in an api/ directory. Each file
becomes a serverless function at the route matching the filename. This file
re-exports the FastAPI app from app/main.py so Vercel can serve it as an
ASGI application.

The vercel.json rewrites /api/ai/* to this service, so:
  /api/ai/v1/questions  →  this function  →  app.main.app  →  /v1/questions
"""
from app.main import app

# Vercel Python runtime expects an ASGI app named `app` at module level.
# The FastAPI instance is already named `app` in app/main.py, so this import
# is sufficient.
