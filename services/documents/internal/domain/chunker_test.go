package domain

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSlidingWindowChunker_PreservesPageNumberAndOffset is the headline
// test for the documents service's chunking strategy: every chunk MUST carry
// a non-zero PageNumber and a valid Offset so that downstream AI
// explanations can cite precise locations.
func TestSlidingWindowChunker_PreservesPageNumberAndOffset(t *testing.T) {
	c := NewSlidingWindowChunker(10, 2) // small windows for the test
	doc := Document{
		ID: "doc-1",
		Pages: []DocumentPage{
			{ID: "p1", DocumentID: "doc-1", PageNumber: 1,
				Text: strings.Repeat("the quick brown fox jumps over the lazy dog ", 30)},
			{ID: "p2", DocumentID: "doc-1", PageNumber: 2,
				Text: strings.Repeat("a b c d e f g h i j k l m n o p q r s t ", 30)},
		},
		Sections: []DocumentSection{
			{ID: "sec1", DocumentID: "doc-1", Title: "Section One", PageNumbers: []int{1}},
			{ID: "sec2", DocumentID: "doc-1", Title: "Section Two", PageNumbers: []int{2}},
		},
	}
	chunks := c.Chunk(doc, doc.Sections)
	assert.NotEmpty(t, chunks)
	for _, ch := range chunks {
		assert.NotEmpty(t, ch.ID)
		assert.Equal(t, "doc-1", ch.DocumentID)
		assert.GreaterOrEqual(t, ch.PageNumber, 1, "every chunk must carry a page number")
		assert.GreaterOrEqual(t, ch.Offset, 0)
		assert.NotEmpty(t, ch.Text)
		assert.Greater(t, ch.TokenEstimate, 0)
		// Each chunk's section ID must correspond to the page it lives on.
		if ch.PageNumber == 1 {
			assert.Equal(t, "sec1", ch.SectionID)
		} else if ch.PageNumber == 2 {
			assert.Equal(t, "sec2", ch.SectionID)
		}
	}
}

// TestSlidingWindowChunker_Overlaps ensures successive chunks on the same
// page share words (the overlap is what gives downstream retrievers
// continuity when slicing a long clause).
func TestSlidingWindowChunker_Overlaps(t *testing.T) {
	c := NewSlidingWindowChunker(4, 2)
	doc := Document{
		ID: "doc-2",
		Pages: []DocumentPage{
			{ID: "p1", DocumentID: "doc-2", PageNumber: 1,
				Text: "alpha beta gamma delta epsilon zeta eta theta iota kappa lambda"},
		},
	}
	chunks := c.Chunk(doc, nil)
	assert.GreaterOrEqual(t, len(chunks), 2)
	// The first chunk's tail should appear in the second chunk's head
	// because overlap > 0.
	tail := lastNWords(chunks[0].Text, 2)
	assert.True(t, strings.HasPrefix(chunks[1].Text, tail[0]) || strings.Contains(chunks[1].Text, tail[0]+tail[1]),
		"chunk 0 tail %v should appear at chunk 1 head", tail)
}

// lastNWords returns the last n space-separated words of s.
func lastNWords(s string, n int) []string {
	words := strings.Fields(s)
	if len(words) <= n {
		return words
	}
	return words[len(words)-n:]
}

// TestExtractorRegistry_RoutesByMime verifies the extractor registry routes
// by MIME type and errors when no extractor matches (no silent drop).
func TestExtractorRegistry_RoutesByMime(t *testing.T) {
	r := NewDefaultRegistry(&fakeExtractor{name: "html", mime: "text/html"})
	got, err := r.ForMime("text/html")
	assert.NoError(t, err)
	assert.Equal(t, "html", got.Name())
	_, err = r.ForMime("application/pdf")
	assert.Error(t, err)
}

// fakeExtractor is a minimal Extractor for tests.
type fakeExtractor struct {
	name string
	mime string
}

func (f *fakeExtractor) Name() string                                  { return f.name }
func (f *fakeExtractor) Supports(m string) bool                        { return m == f.mime }
func (f *fakeExtractor) Extract(ctx context.Context, _ RawDocumentInput) (Extraction, error) {
	return Extraction{ExtractorName: f.name}, nil
}

// Keep imports honest.
var _ = context.Background
