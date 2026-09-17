"""FastAPI entrypoint for Vercel multi-service deployment.

Vercel's Python runtime looks for: app.py, main.py, index.py, server.py,
wsgi.py, asgi.py in the service root directory.

This file imports the FastAPI app from app/main.py so Vercel can detect it.
"""
from app.main import app

# The 'app' variable is the ASGI application Vercel serves.
