package kenya_law

import (
        "context"
        "fmt"
        "io"
        "regexp"
        "strings"
        "time"

        "golang.org/x/net/html"
)

// RegulationEntry is a parsed record from the kenyalaw.org subsidiary
// legislation index. Subsidiary legislation (often called "Legal Notices" or
// "Subsidiary Legislation") covers regulations, rules and orders made under
// the authority of an Act of Parliament.
type RegulationEntry struct {
        LegalNoticeNo string // e.g. "Legal Notice No. 45 of 2023"
        ParentAct     string // The Act under which the regulation is made
        Title         string
        PublishedAt   time.Time
        URL           string
}

// DiscoverRegulations fetches the kenyalaw.org subsidiary legislation index
// and parses it into a slice of RegulationEntry records.
func (k *KenyaLawAdapter) DiscoverRegulations(ctx context.Context) ([]RegulationEntry, error) {
        body, err := k.fetchSource(ctx, SourceSubsidiaryIndex, "subsidiary")
        if err != nil {
                return nil, err
        }
        if body == nil {
                return nil, nil
        }
        entries, err := ParseRegulationsHTML(strings.NewReader(string(body)))
        if err != nil {
                return nil, fmt.Errorf("%w: %v", ErrParse, err)
        }
        return entries, nil
}

// ParseRegulationsHTML parses the kenyalaw.org subsidiary legislation page.
//
// Each entry is typically a link with text like "Legal Notice No. 45 of 2023 —
// The Foo (Bar) Regulations, 2023". The parser splits that into a notice
// number, a parent Act (when discoverable from the title) and the title.
func ParseRegulationsHTML(r io.Reader) ([]RegulationEntry, error) {
        doc, err := html.Parse(r)
        if err != nil {
                return nil, fmt.Errorf("kenya_law parser: %w", err)
        }
        var out []RegulationEntry
        walkAnchors(doc, func(href, text string) {
                if !looksLikeRegulation(href, text) {
                        return
                }
                out = append(out, RegulationEntry{
                        LegalNoticeNo: extractLegalNoticeNo(text),
                        ParentAct:     extractParentAct(text),
                        Title:         strings.TrimSpace(text),
                        PublishedAt:   parseDateInText(text),
                        URL:           href,
                })
        })
        return out, nil
}

// looksLikeRegulation returns true if the href/text suggests a Legal Notice
// or subsidiary legislation record.
func looksLikeRegulation(href, text string) bool {
        lc := strings.ToLower(href + " " + text)
        return strings.Contains(lc, "legal notice") ||
                strings.Contains(lc, "subsidiary") ||
                strings.Contains(lc, "regulations") ||
                strings.Contains(lc, "ln_") ||
                strings.Contains(lc, "legal_notice")
}

var legalNoticeNoRe = regexp.MustCompile(`(?i)\blegal\s+notice\s+no\.?\s*(\d+)\s+of\s+(\d{4})\b`)

func extractLegalNoticeNo(text string) string {
        m := legalNoticeNoRe.FindStringSubmatch(text)
        if len(m) >= 3 {
                return "Legal Notice No. " + m[1] + " of " + m[2]
        }
        return ""
}

// extractParentAct tries to identify the parent Act from a regulation title.
// Kenyan regulation titles follow patterns like "The Foo Act (Cap XX)
// (Bar) Regulations, 2023". The parser returns "Foo Act (Cap XX)" where
// possible.
func extractParentAct(text string) string {
        // Look for "... Act (Cap XX) ..." patterns.
        if idx := strings.Index(strings.ToLower(text), "act"); idx > 0 {
                // Trim back to the start of the Act name.
                start := strings.LastIndexAny(text[:idx], "(,;")
                if start == -1 {
                        start = 0
                }
                return strings.TrimSpace(text[start:])
        }
        return ""
}

// parseDateInText extracts a year-only date from a regulation title for the
// published_at field. Real Legal Notices carry full dates; we fall back to
// the year if that's all we can find.
func parseDateInText(text string) time.Time {
        m := yearRe.FindString(text)
        if m == "" {
                return time.Time{}
        }
        var y int
        _, err := fmt.Sscanf(m, "%d", &y)
        if err != nil {
                return time.Time{}
        }
        return time.Date(y, time.January, 1, 0, 0, 0, 0, time.UTC)
}
