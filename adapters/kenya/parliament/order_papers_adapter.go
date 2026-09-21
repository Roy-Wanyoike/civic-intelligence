// Order Paper adapter — discovers and fetches daily House agendas.
//
// Live site recon (2026-09-21): parliament.go.ke publishes Order Papers as
// PDFs under two listing pages:
//
//   NA:     /the-national-assembly/house-business/order-paper
//   Senate: /the-senate/house-business/order-paper
//
// Each listing is a Drupal View with 25 rows per page (paginated via ?page=N).
// Each row contains a single PDF link inside
//
//   <td class="views-field-field-pdf">
//     <span class="file--application-pdf">
//       <a href=".../sites/default/files/YYYY-MM/<name>.pdf">…</a>
//     </span>
//   </td>
//
// The Order Paper title (used for the sitting date) is the PDF filename,
// typically of the form "ORDER PAPER FOR THURSDAY 27TH AUGUST 2026.pdf".
// Supplementary Order Papers are also listed and are kept as separate
// candidates (the URL uniquely identifies them).
package parliament

import (
        "context"
        "fmt"
        "strings"
        "time"
)

// OrderPaperCandidate is a discovered Order Paper.
type OrderPaperCandidate struct {
        URL          string
        Title        string
        House        string
        SittingDate  time.Time
        SourceID     string
        IsSupplementary bool // true when the title contains "Supplementary"
        DiscoveredAt time.Time
}

// DiscoverOrderPapers discovers Order Papers from both the National Assembly
// and Senate listing pages. It walks the first maxPages pages of each listing
// (25 rows per page). The returned candidates are deduplicated by URL.
func (a *Adapter) DiscoverOrderPapers(ctx context.Context) ([]OrderPaperCandidate, error) {
        const maxPages = 5 // cap at 5 pages = 125 most-recent sittings per house
        var out []OrderPaperCandidate
        for _, page := range []struct {
                url   string
                house string
        }{
                {a.naOrderPaperURL, "National Assembly"},
                {a.senateOrderPaperURL, "Senate"},
        } {
                cands, err := discoverPDFListing(a, ctx, page.url, page.house, maxPages, orderPaperRowMapper)
                if err != nil {
                        // A single failing house does not abort discovery of the other.
                        continue
                }
                out = append(out, cands...)
        }
        out = dedupeOrderByURL(out)
        now := time.Now().UTC()
        for i := range out {
                out[i].DiscoveredAt = now
                if out[i].SourceID == "" {
                        out[i].SourceID = makeOrderPaperSourceID(out[i].House, out[i].SittingDate, out[i].URL)
                }
        }
        return out, nil
}

// orderPaperRowMapper converts a single PDF row from ParsePDFListing into an
// OrderPaperCandidate. The sitting date is parsed from the PDF filename
// (which typically contains "FOR THURSDAY 27TH AUGUST 2026" or similar).
// Supplementary Order Papers are flagged via IsSupplementary.
func orderPaperRowMapper(href, linkText, house string) (OrderPaperCandidate, bool) {
        if !strings.HasSuffix(strings.ToLower(href), ".pdf") {
                return OrderPaperCandidate{}, false
        }
        title := decodePDFTitle(href, linkText)
        sittingDate := parseOrderPaperSittingDate(title)
        return OrderPaperCandidate{
                URL:             href,
                Title:           title,
                House:           house,
                SittingDate:     sittingDate,
                IsSupplementary: isSupplementaryOrderPaper(title),
        }, true
}

// parseOrderPaperSittingDate extracts the sitting date from an Order Paper
// title. Order Paper PDFs are typically named:
//   "ORDER PAPER FOR THURSDAY 27TH AUGUST 2026"
//   "Supplementary Order Paper - 18 September 2026"
//   "Order Paper for Wednesday 26Th August 2026"
// The parser is tolerant of case, ordinals (27Th), and missing commas.
func parseOrderPaperSittingDate(title string) time.Time {
        clean := ordinalsRe.ReplaceAllString(title, "$1")
        return findDateInText(clean)
}

// isSupplementaryOrderPaper reports whether the title indicates a
// Supplementary Order Paper (a per-sitting addendum to the main Order Paper).
// The check is case-insensitive and tolerant of abbreviated forms.
func isSupplementaryOrderPaper(title string) bool {
        lower := strings.ToLower(title)
        return strings.Contains(lower, "supplementary") || strings.Contains(lower, "supp order")
}

// makeOrderPaperSourceID returns a deterministic source ID for an Order Paper.
// Format: ke-parliament-<house-slug>-order-paper-YYYY-MM-DD[-supp-<hash>] when
// the sitting date is known; falls back to a URL hash otherwise. The
// supplementary flag appends a -supp suffix (with a short URL hash for
// disambiguation, since a single sitting can have multiple supplementary
// order papers).
func makeOrderPaperSourceID(house string, sitting time.Time, href string) string {
        houseSlug := "na"
        if strings.EqualFold(house, "Senate") {
                houseSlug = "senate"
        }
        if sitting.IsZero() {
                return "ke-parliament-" + houseSlug + "-order-paper-" + shortHash(href)
        }
        base := fmt.Sprintf("ke-parliament-%s-order-paper-%04d-%02d-%02d",
                houseSlug, sitting.Year(), int(sitting.Month()), sitting.Day())
        if isSupplementaryOrderPaper(href) {
                return base + "-supp-" + shortHash(href)
        }
        return base
}

// dedupeOrderByURL removes candidates with duplicate URLs. First occurrence
// wins (the main archive is preferred over the "latest" sidebar block).
func dedupeOrderByURL(in []OrderPaperCandidate) []OrderPaperCandidate {
        seen := make(map[string]struct{}, len(in))
        out := make([]OrderPaperCandidate, 0, len(in))
        for _, c := range in {
                if c.URL == "" {
                        continue
                }
                if _, ok := seen[c.URL]; ok {
                        continue
                }
                seen[c.URL] = struct{}{}
                out = append(out, c)
        }
        return out
}

// FetchOrderPaper downloads an Order Paper PDF.
func (a *Adapter) FetchOrderPaper(ctx context.Context, url string) (string, error) {
        return a.fetchURL(ctx, url, "order_paper")
}

// OrderPaperItem is a single agenda item in an Order Paper.
//
// NOTE: real Order Paper parsing requires PDF text extraction + agenda-item
// structure heuristics. That pipeline is delegated to the documents service
// (services/documents). This HTML-only parser is intentionally a stub.
type OrderPaperItem struct {
        Title       string
        Description string
        BillRef     string // optional Bill identifier
}

// ParseOrderPaper extracts agenda items from an Order Paper.
func (a *Adapter) ParseOrderPaper(htmlBody, sourceURL string) ([]OrderPaperItem, error) {
        _ = htmlBody
        _ = sourceURL
        return []OrderPaperItem{}, nil
}
