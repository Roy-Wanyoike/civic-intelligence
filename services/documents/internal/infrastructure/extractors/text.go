package extractors

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/services/documents/internal/domain"
)

// TextExtractor handles plain-text documents. It treats the entire document
// as a single page (PageNumber=1) and splits on headings heuristically.
type TextExtractor struct{}

// Name implements domain.Extractor.
func (TextExtractor) Name() string { return "text-v1" }

// Supports implements domain.Extractor.
func (TextExtractor) Supports(m string) bool {
	m = strings.ToLower(m)
	return m == "text/plain" || m == "text/markdown"
}

// Extract implements domain.Extractor.
func (TextExtractor) Extract(_ context.Context, in domain.RawDocumentInput) (domain.Extraction, error) {
	text := string(in.Bytes)
	page := domain.DocumentPage{
		ID:         in.DocumentID + "_p1",
		DocumentID: in.DocumentID,
		PageNumber: 1,
		Text:       text,
	}
	return domain.Extraction{
		DocumentID:    in.DocumentID,
		Title:         firstLine(text),
		Pages:         []domain.DocumentPage{page},
		Sections:      splitHeadings(text),
		ExtractorName: "text-v1",
		ExtractedAt:   timeNowUTC(),
		Confidence:    0.95,
	}, nil
}

// PDFExtractor is a stub for the production PDF extractor. The real impl
// would use a library like pdfcpu or unidoc; here we delegate to the
// text extractor to keep the build hermetic. A todo references the
// integration ticket.
type PDFExtractor struct {
	Fallback TextExtractor
}

// Name implements domain.Extractor.
func (PDFExtractor) Name() string { return "pdf-v1-stub" }

// Supports implements domain.Extractor.
func (PDFExtractor) Supports(m string) bool {
	m = strings.ToLower(m)
	return m == "application/pdf"
}

// Extract implements domain.Extractor. It calls the text extractor on the
// raw bytes (which will produce low-quality results for binary PDFs but
// allows the pipeline to complete and emit a low-confidence event so the
// orchestrator can choose to run OCR).
func (p PDFExtractor) Extract(ctx context.Context, in domain.RawDocumentInput) (domain.Extraction, error) {
	ext, err := p.Fallback.Extract(ctx, in)
	if err != nil {
		return ext, err
	}
	ext.ExtractorName = p.Name()
	ext.Confidence = 0.40 // low confidence; OCR pipeline should follow
	ext.Metadata = map[string]string{"note": "PDF extractor is a stub; OCR may be needed"}
	return ext, nil
}

// DOCXExtractor is a stub analogous to PDFExtractor.
type DOCXExtractor struct {
	Fallback TextExtractor
}

// Name implements domain.Extractor.
func (DOCXExtractor) Name() string { return "docx-v1-stub" }

// Supports implements domain.Extractor.
func (DOCXExtractor) Supports(m string) bool {
	m = strings.ToLower(m)
	return m == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
		m == "application/msword"
}

// Extract implements domain.Extractor.
func (d DOCXExtractor) Extract(ctx context.Context, in domain.RawDocumentInput) (domain.Extraction, error) {
	ext, err := d.Fallback.Extract(ctx, in)
	if err != nil {
		return ext, err
	}
	ext.ExtractorName = d.Name()
	ext.Confidence = 0.50
	ext.Metadata = map[string]string{"note": "DOCX extractor is a stub; production should use unioffice"}
	return ext, nil
}
