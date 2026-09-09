// Package documents owns the structural interpretation of raw documents.
// Given a RawDocument produced by ingestion, the documents service extracts
// pages, sections and chunks. It is country-agnostic: parsing is driven by
// the document's MIME type, never by which country the document came from.
package documents

import "time"

// Document is the parsed representation of a raw document fetched by
// ingestion. It owns the page/section/chunk structure that downstream
// services (evidence, intelligence, search) use to cite and retrieve text.
type Document struct {
	ID            string
	SourceID      string
	CountryCode   string
	Title         string
	MimeType      string
	ContentHash   string
	Pages         []DocumentPage
	Sections      []DocumentSection
	Chunks        []DocumentChunk
	ParsedAt      time.Time
	SourceDocumentID string // documents service may ingest from another parsed document
}

// DocumentPage is one page of a paginated document (PDF, DOCX, etc.). HTML
// documents may have a single synthetic page with PageNumber=1.
type DocumentPage struct {
	ID         string
	DocumentID string
	PageNumber int
	Text       string
	Image      []byte // optional; for OCR
	BoundingBoxes []BoundingBox
}

// BoundingBox is a rectangular region on a page (for OCR / tables).
type BoundingBox struct {
	X0, Y0, X1, Y1 float64
	Type           string // "text", "table", "image"
}

// DocumentSection is a logical subdivision of a document (a heading, a
// clause, a Q&A). Sections may span multiple pages; the OffsetWithinPage
// array records where on each page the section appears.
type DocumentSection struct {
	ID              string
	DocumentID      string
	Title           string
	Type            string // "clause", "heading", "qa", "paragraph", "schedule"
	PageNumbers     []int
	OffsetWithinPage map[int]int // page -> byte offset
	Text            string
}

// DocumentChunk is a fixed-size window of text suitable for embedding by
// the intelligence service. Chunks preserve the page_number + section +
// offset triple so that AI explanations can cite precise locations.
type DocumentChunk struct {
	ID              string
	DocumentID      string
	PageNumber      int
	SectionID       string
	Offset          int // byte offset within the page
	Text            string
	TokenEstimate   int
	EmbeddingVector []float32 // populated by intelligence; nil until then
}

// Extraction is the structured result of running a parser over a raw
// document. It is the value object that parsers return to the application
// layer.
type Extraction struct {
	DocumentID    string
	Title          string
	Pages          []DocumentPage
	Sections       []DocumentSection
	Confidence     float64
	ExtractorName  string
	ExtractedAt    time.Time
	Metadata       map[string]string
}

// OCRResult is the output of running OCR on a page image. The documents
// service has an OCR pipeline (concrete impl in infrastructure) but the
// domain type lives here so the application layer can be OCR-provider
// agnostic.
type OCRResult struct {
	PageNumber int
	Text       string
	Confidence float64
	Language   string
	Provider   string // "tesseract", "cloud-vision", etc.
}
