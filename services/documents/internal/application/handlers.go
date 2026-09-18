// Package application contains the documents service's use cases.
package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/Roy-Wanyoike/civic-intelligence/services/documents/internal/domain"
)

// ParseDocumentHandler is invoked when ingestion publishes
// document.discovered. It loads the raw bytes (via the injected raw
// fetcher), routes them to the right extractor, runs the chunking strategy
// and persists the parsed Document. It then publishes document.parsed.
type ParseDocumentHandler struct {
	registry  domain.ExtractorRegistry
	chunker   domain.ChunkingStrategy
	docs      domain.DocumentRepository
	publisher domain.EventPublisher
	clock     domain.Clock
	idGen     IDGenerator
	rawLoader RawDocumentLoader
}

// RawDocumentLoader abstracts the network call to ingestion's document
// storage. In production this hits the S3-compatible store; in tests it is
// a stub.
type RawDocumentLoader interface {
	Load(ctx context.Context, documentID string) (domain.RawDocumentInput, error)
}

// IDGenerator generates string IDs.
type IDGenerator interface {
	New() string
}

// NewParseDocumentHandler constructs the handler.
func NewParseDocumentHandler(
	registry domain.ExtractorRegistry,
	chunker domain.ChunkingStrategy,
	docs domain.DocumentRepository,
	publisher domain.EventPublisher,
	clock domain.Clock,
	idGen IDGenerator,
	rawLoader RawDocumentLoader,
) *ParseDocumentHandler {
	return &ParseDocumentHandler{
		registry: registry, chunker: chunker, docs: docs,
		publisher: publisher, clock: clock, idGen: idGen,
		rawLoader: rawLoader,
	}
}

// Handle executes the parse pipeline.
func (h *ParseDocumentHandler) Handle(ctx context.Context, rawDocumentID string) (*domain.Document, error) {
	raw, err := h.rawLoader.Load(ctx, rawDocumentID)
	if err != nil {
		return nil, fmt.Errorf("load raw: %w", err)
	}
	ext, err := h.registry.ForMime(raw.MimeType)
	if err != nil {
		return nil, err
	}
	extraction, err := ext.Extract(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}
	now := h.clock.Now()
	doc := domain.Document{
		ID:           h.idGen.New(),
		SourceID:     raw.SourceID,
		CountryCode:  raw.CountryCode,
		Title:        extraction.Title,
		MimeType:     raw.MimeType,
		ContentHash:  raw.ContentHash,
		Pages:        extraction.Pages,
		Sections:     extraction.Sections,
		ParsedAt:     now,
		SourceDocumentID: rawDocumentID,
	}
	doc.Chunks = h.chunker.Chunk(doc, doc.Sections)
	if err := h.docs.Save(ctx, doc); err != nil {
		return nil, err
	}

	_ = h.publisher.Publish(ctx, contracts.Event{
		ID:         h.idGen.New(),
		Type:       contracts.EventDocumentParsed,
		Source:     "documents",
		Subject:    "civic.document.parsed",
		OccurredAt: now,
		Data: map[string]any{
			"document_id": doc.ID,
			"pages":       len(doc.Pages),
			"sections":    len(doc.Sections),
			"chunks":      len(doc.Chunks),
		},
	})
	return &doc, nil
}

// GetDocumentHandler loads a single parsed document.
type GetDocumentHandler struct {
	docs domain.DocumentRepository
}

// NewGetDocumentHandler constructs the handler.
func NewGetDocumentHandler(docs domain.DocumentRepository) *GetDocumentHandler {
	return &GetDocumentHandler{docs: docs}
}

// Handle executes the query.
func (h *GetDocumentHandler) Handle(ctx context.Context, id string) (*domain.Document, error) {
	return h.docs.Get(ctx, id)
}

// GetChunksForPageHandler returns all chunks for a given page of a document.
// Used by the intelligence service when retrieving citation candidates.
type GetChunksForPageHandler struct {
	docs domain.DocumentRepository
}

// NewGetChunksForPageHandler constructs the handler.
func NewGetChunksForPageHandler(docs domain.DocumentRepository) *GetChunksForPageHandler {
	return &GetChunksForPageHandler{docs: docs}
}

// Handle executes the query.
func (h *GetChunksForPageHandler) Handle(ctx context.Context, documentID string, pageNumber int) ([]domain.DocumentChunk, error) {
	doc, err := h.docs.Get(ctx, documentID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.DocumentChunk, 0)
	for _, c := range doc.Chunks {
		if c.PageNumber == pageNumber {
			out = append(out, c)
		}
	}
	return out, nil
}

// DefaultTitleFromText infers a document title from its first non-empty page
// if the extractor did not return one. Pure helper; no side effects.
func DefaultTitleFromText(pages []domain.DocumentPage) string {
	for _, p := range pages {
		t := strings.TrimSpace(p.Text)
		if t == "" {
			continue
		}
		if i := strings.IndexByte(t, '\n'); i > 0 {
			return t[:i]
		}
		if len(t) > 200 {
			return t[:200]
		}
		return t
	}
	return ""
}

// time is imported here to satisfy the time.Now usage in tests; the handler
// itself relies on Clock.
var _ = time.Now
