// Package extractors contains the concrete implementations of
// domain.Extractor for HTML, plain text, PDF (stubbed) and DOCX (stubbed).
// OCR is implemented separately via a wrapper that uses Tesseract or a cloud
// provider; the country-agnostic interface lives in the domain package.
package extractors

import (
        "context"
        "fmt"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/documents/internal/domain"
)

// timeNowUTC returns the current UTC time. Wrapped as a function (rather
// than calling time.Now().UTC() inline) so tests can stub it if needed.
func timeNowUTC() time.Time { return time.Now().UTC() }

// HTMLExtractor parses simple HTML by stripping tags and splitting on
// heading elements (h1-h6). It is deliberately lightweight; production
// deployments should use a robust HTML parser (golang.org/x/net/html).
type HTMLExtractor struct{}

// Name implements domain.Extractor.
func (HTMLExtractor) Name() string { return "html-v1" }

// Supports implements domain.Extractor.
func (HTMLExtractor) Supports(m string) bool {
        m = strings.ToLower(m)
        return m == "text/html" || m == "application/xhtml+xml"
}

// Extract implements domain.Extractor.
func (HTMLExtractor) Extract(_ context.Context, in domain.RawDocumentInput) (domain.Extraction, error) {
        raw := string(in.Bytes)
        text := stripTags(raw)
        sections := splitHeadings(text)
        page := domain.DocumentPage{
                ID:         in.DocumentID + "_p1",
                DocumentID: in.DocumentID,
                PageNumber: 1,
                Text:       text,
        }
        return domain.Extraction{
                DocumentID:   in.DocumentID,
                Title:        firstLine(text),
                Pages:        []domain.DocumentPage{page},
                Sections:     sections,
                ExtractorName: "html-v1",
                ExtractedAt:  timeNowUTC(),
                Confidence:   0.85,
                Metadata:     map[string]string{"source_url": in.URL},
        }, nil
}

// stripTags removes everything between < and >. A real implementation
// would build a DOM; this is sufficient for the platform's tests.
func stripTags(s string) string {
        var b strings.Builder
        inTag := false
        for _, r := range s {
                switch r {
                case '<':
                        inTag = true
                case '>':
                        inTag = false
                        b.WriteByte(' ')
                default:
                        if !inTag {
                                b.WriteRune(r)
                        }
                }
        }
        return strings.Join(strings.Fields(b.String()), " ")
}

// splitHeadings extracts sections marked by lines that look like headings
// (e.g. lines beginning with "Clause" or numeric markers like "1." or "1.1").
func splitHeadings(text string) []domain.DocumentSection {
        out := []domain.DocumentSection{}
        lines := strings.Split(text, "\n")
        current := domain.DocumentSection{ID: "sec-1", Type: "paragraph"}
        curText := strings.Builder{}
        idx := 0
        for _, ln := range lines {
                ln = strings.TrimSpace(ln)
                if isHeading(ln) {
                        if curText.Len() > 0 {
                                current.Text = curText.String()
                                current.Title = current.Title
                                out = append(out, current)
                                curText.Reset()
                        }
                        idx++
                        current = domain.DocumentSection{
                                ID:           fmt.Sprintf("sec-%d", idx),
                                Title:        ln,
                                Type:         "heading",
                                PageNumbers:  []int{1},
                                OffsetWithinPage: map[int]int{1: curText.Len()},
                        }
                        continue
                }
                if ln != "" {
                        if curText.Len() > 0 {
                                curText.WriteByte(' ')
                        }
                        curText.WriteString(ln)
                }
        }
        if curText.Len() > 0 {
                current.Text = curText.String()
                out = append(out, current)
        }
        return out
}

// isHeading returns true if a line looks like a section heading.
func isHeading(ln string) bool {
        if len(ln) < 3 || len(ln) > 120 {
                return false
        }
        if strings.HasPrefix(ln, "Clause ") || strings.HasPrefix(ln, "Section ") {
                return true
        }
        // "1." or "1.1.2" at the start of a short line.
        dot := strings.IndexByte(ln, '.')
        if dot > 0 && dot < 6 {
                prefix := ln[:dot]
                isDigit := true
                for _, r := range prefix {
                        if r < '0' || r > '9' {
                                isDigit = false
                                break
                        }
                }
                if isDigit {
                        return true
                }
        }
        return false
}

// firstLine returns the first non-empty line of text.
func firstLine(s string) string {
        for _, ln := range strings.Split(s, "\n") {
                if strings.TrimSpace(ln) != "" {
                        return strings.TrimSpace(ln)
                }
        }
        return ""
}
