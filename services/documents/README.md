# documents service

The **documents** service owns the structural interpretation of raw documents
produced by ingestion. It is country-agnostic: parsing is driven by MIME
type, never by which country a document came from.

## Bounded context

Owns:
- Document, DocumentPage, DocumentSection, DocumentChunk
- Extraction pipeline (HTML / text / PDF / DOCX / OCR)
- Chunking strategy that preserves page_number + section + offset for citations
- OCRResult

Does NOT own:
- Raw bytes (→ ingestion)
- Country-specific field mapping (→ adapters/kenya)
- AI explanations / embeddings (→ intelligence)
- Search indexes (→ search)
- Notifications (→ notifications)

## Architectural rules enforced by this service

1. **Country-agnostic.** The parser choice is made by MIME type only. The
   Kenya adapter does NOT tell the documents service how to parse a Hansard.
2. **Citation precision is mandatory.** Every chunk carries page_number,
   section_id and offset. Without these, downstream AI explanations cannot
   cite precise locations.
3. **Repository interfaces live in domain.** Concrete PostgreSQL impls
   live in infrastructure/postgres. Domain has zero `database/sql` imports.

## Running

```bash
go build ./...
go test ./...
```
