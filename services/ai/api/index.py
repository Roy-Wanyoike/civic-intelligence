"""Vercel serverless function entry point for the AI service.

Vercel's Python runtime expects each .py file in the api/ directory to be a
serverless function. The function receives an ASGI-compatible request and
must return an ASGI-compatible response.

This file wraps the FastAPI app from app/main.py so Vercel can serve it.
The vercel.json rewrites /api/ai/* to this service, so:
  /api/ai/v1/questions  →  this function  →  app.main.app  →  /v1/questions

Vercel Python runtime: https://vercel.com/docs/functions/runtimes/python
"""
from app.main import app

# Vercel's Python runtime detects ASGI apps exported as `app` at module level.
# The FastAPI instance imported above IS the ASGI app — no wrapper needed.
# This file exists so Vercel has an entry point in the api/ directory.
