"""Evidence store — fetches evidence for a question via the search service.

This is a thin client over the Go ``services/search`` HTTP API. It performs
hybrid search (FTS + pgvector) and reranks results client-side.

In a future iteration this will move into a dedicated reranker microservice
once corpus size justifies it.
"""
from __future__ import annotations

import httpx
from tenacity import retry, stop_after_attempt, wait_exponential

from .config import Settings
from .domain import Citation, RetrievedChunk, RAGContext, SourceType
from .logging import get_logger

log = get_logger(__name__)


class EvidenceStore:
    """Retrieves and reranks evidence chunks for a question."""

    def __init__(self, settings: Settings) -> None:
        self._settings = settings
        self._client = httpx.AsyncClient(
            base_url=settings.search_service_url,
            timeout=10.0,
        )

    @retry(
        stop=stop_after_attempt(2),
        wait=wait_exponential(multiplier=0.5, max=2),
        reraise=True,
    )
    async def retrieve(self, question: str, bill_id: str | None = None) -> list[RetrievedChunk]:
        """Hybrid search against the search service."""
        params: dict[str, str | int] = {
            "q": question,
            "top_k": self._settings.rag_top_k,
            "min_score": self._settings.rag_min_score,
        }
        if bill_id:
            params["bill_id"] = bill_id
        try:
            resp = await self._client.get("/internal/search/hybrid", params=params)
            resp.raise_for_status()
            data = resp.json()
        except httpx.HTTPError as e:
            log.warning("evidence_retrieve_failed", extra={"event": "evidence_retrieve_failed", "error": str(e)})
            # In dev/CI the search service is offline — return empty so the
            # pipeline still completes (with low confidence).
            return []

        chunks: list[RetrievedChunk] = []
        for item in data.get("chunks", []):
            chunks.append(
                RetrievedChunk(
                    document_id=item["document_id"],
                    chunk_id=item["chunk_id"],
                    text=item["text"],
                    page_number=item.get("page_number"),
                    section=item.get("section"),
                    source_url=item["source_url"],
                    source_type=SourceType(item.get("source_type", "parliament")),
                    semantic_score=float(item.get("semantic_score", 0.0)),
                    keyword_score=float(item.get("keyword_score", 0.0)),
                    combined_score=float(item.get("combined_score", 0.0)),
                )
            )
        return chunks

    def rerank(self, chunks: list[RetrievedChunk], *, top_k: int | None = None) -> list[RetrievedChunk]:
        """Simple reranker: weight semantic + keyword score, prefer parliament/hansard sources."""
        if not chunks:
            return []
        k = top_k or self._settings.rag_rerank_top_k
        # Boost authoritative sources.
        source_boost = {
            SourceType.PARLIAMENT: 1.15,
            SourceType.HANSARD: 1.10,
            SourceType.GAZETTE: 1.05,
            SourceType.KENYA_LAW: 1.10,
        }
        scored = sorted(
            chunks,
            key=lambda c: c.combined_score * source_boost.get(c.source_type, 1.0),
            reverse=True,
        )
        return scored[:k]

    def build_context(self, chunks: list[RetrievedChunk]) -> RAGContext:
        """Build the RAG context window from reranked chunks."""
        # Rough token estimate: 1 token ≈ 4 chars.
        max_chars = self._settings.rag_max_context_tokens * 4
        used_chars = 0
        chosen: list[RetrievedChunk] = []
        citations: list[Citation] = []
        for chunk in chunks:
            if used_chars + len(chunk.text) > max_chars:
                break
            chosen.append(chunk)
            used_chars += len(chunk.text)
            citations.append(
                Citation(
                    document_id=chunk.document_id,
                    source_url=chunk.source_url,
                    page_number=chunk.page_number,
                    section=chunk.section,
                    snippet=chunk.text[:280],
                    source_type=chunk.source_type,
                    retrieved_at=__import__("datetime").datetime.utcnow(),
                )
            )
        token_count = used_chars // 4
        return RAGContext(chunks=chosen, citations=citations, token_count=token_count)
