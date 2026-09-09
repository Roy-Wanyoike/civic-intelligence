"""Civic Intelligence — AI service package.

This package implements the provider-agnostic AI gateway and the evidence-grounded
RAG pipeline. The architectural invariant is: AI proposes candidate facts; it never
mutates canonical legislative state. Every response passes citation validation.
"""

__version__ = "0.1.0"
