package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// RawDocumentInput is the input contract for a parser. It carries the raw
// bytes (or a reader) and enough metadata to make parsing decisions.
type RawDocumentInput struct {
	DocumentID   string
	SourceID     string
	CountryCode  string
	MimeType     string
	URL          string
	ContentHash  string
	Bytes        []byte
}

// Extractor is the country-agnostic interface every parser must satisfy.
// Concrete HTML, PDF, DOCX and OCR extractors live in infrastructure.
type Extractor interface {
	// Name returns the extractor's identifier (used in audit logs).
	Name() string
	// Supports reports whether the extractor can handle the given MIME.
	Supports(mimeType string) bool
	// Extract parses the input into pages, sections and (later) chunks.
	Extract(ctx context.Context, in RawDocumentInput) (Extraction, error)
}

// ExtractorRegistry routes a raw document to the right Extractor by MIME
// type. If no extractor matches, an error is returned (rather than silently
// dropping the document).
type ExtractorRegistry interface {
	ForMime(mime string) (Extractor, error)
	Register(e Extractor)
}

// DefaultRegistry is the in-memory implementation.
type DefaultRegistry struct {
	extractors []Extractor
}

// NewDefaultRegistry constructs the registry.
func NewDefaultRegistry(extractors ...Extractor) *DefaultRegistry {
	return &DefaultRegistry{extractors: extractors}
}

// Register adds an extractor.
func (r *DefaultRegistry) Register(e Extractor) { r.extractors = append(r.extractors, e) }

// ForMime returns the first registered extractor that supports the MIME type.
func (r *DefaultRegistry) ForMime(mime string) (Extractor, error) {
	for _, e := range r.extractors {
		if e.Supports(mime) {
			return e, nil
		}
	}
	return nil, fmt.Errorf("documents: no extractor for mime %q", mime)
}

// ChunkingStrategy decides how to slice a page's text into chunks suitable
// for embedding. The strategy MUST preserve page_number + section + offset
// so that downstream citations are precise.
type ChunkingStrategy interface {
	Chunk(doc Document, sections []DocumentSection) []DocumentChunk
}

// SlidingWindowChunker splits page text into N-token windows with M-token
// overlap. It is the default strategy because it preserves citation
// precision while keeping chunks small enough for embedding models.
type SlidingWindowChunker struct {
	TargetTokens  int
	OverlapTokens int
}

// NewSlidingWindowChunker constructs a chunker with sensible defaults.
func NewSlidingWindowChunker(target, overlap int) *SlidingWindowChunker {
	if target <= 0 {
		target = 512
	}
	if overlap < 0 || overlap >= target {
		overlap = target / 4
	}
	return &SlidingWindowChunker{TargetTokens: target, OverlapTokens: overlap}
}

// Chunk implements ChunkingStrategy.
func (c *SlidingWindowChunker) Chunk(doc Document, sections []DocumentSection) []DocumentChunk {
	out := make([]DocumentChunk, 0, len(doc.Pages))
	sectionByPage := indexSectionsByPage(sections)
	for _, page := range doc.Pages {
		words := strings.Fields(page.Text)
		if len(words) == 0 {
			continue
		}
		step := c.TargetTokens - c.OverlapTokens
		if step <= 0 {
			step = 1
		}
		for start := 0; start < len(words); start += step {
			end := start + c.TargetTokens
			if end > len(words) {
				end = len(words)
			}
			window := strings.Join(words[start:end], " ")
			sectionID := ""
			if secs, ok := sectionByPage[page.PageNumber]; ok && len(secs) > 0 {
				sectionID = secs[0].ID
			}
			out = append(out, DocumentChunk{
				ID:            fmt.Sprintf("%s_p%d_c%d", doc.ID, page.PageNumber, start),
				DocumentID:    doc.ID,
				PageNumber:    page.PageNumber,
				SectionID:     sectionID,
				Offset:        offsetOfWindow(page.Text, words, start),
				Text:          window,
				TokenEstimate: end - start,
			})
			if end == len(words) {
				break
			}
		}
	}
	return out
}

// indexSectionsByPage returns a map from page number to the sections on that
// page (in order of appearance).
func indexSectionsByPage(sections []DocumentSection) map[int][]DocumentSection {
	out := make(map[int][]DocumentSection)
	for _, s := range sections {
		for _, p := range s.PageNumbers {
			out[p] = append(out[p], s)
		}
	}
	return out
}

// offsetOfWindow returns the byte offset in pageText at which the window
// starting at word index `start` begins. This is approximate (word-based)
// but good enough for citation precision.
func offsetOfWindow(pageText string, words []string, start int) int {
	if start == 0 {
		return 0
	}
	// Count characters in the first `start` words plus their separating
	// spaces.
	total := 0
	for i := 0; i < start && i < len(words); i++ {
		total += len(words[i]) + 1 // +1 for the separating space
	}
	if total > len(pageText) {
		return len(pageText)
	}
	return total
}

// DocumentRepository persists parsed documents.
type DocumentRepository interface {
	Save(ctx context.Context, d Document) error
	Get(ctx context.Context, id string) (*Document, error)
	ListBySource(ctx context.Context, sourceID string) ([]Document, error)
}

// EventPublisher is the abstract event bus.
type EventPublisher interface {
	Publish(ctx context.Context, event contracts.Event) error
}

// Clock abstracts time.
type Clock interface{ Now() time.Time }

// SystemClock is the production Clock.
type SystemClock struct{}

// Now implements Clock.
func (SystemClock) Now() time.Time { return time.Now().UTC() }
