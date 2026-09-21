// Votes & Proceedings adapter — discovers and fetches official vote records.
//
// Live site recon (2026-09-21): parliament.go.ke publishes Votes & Proceedings
// as PDFs under two listing pages:
//
//   NA:     /the-national-assembly/house-business/votes-proceeding
//           (note the singular form — parliament.go.ke uses "votes-proceeding",
//            NOT "votes-proceedings" — the old TODO at /votes-and-proceedings
//            returns HTTP 404)
//   Senate: /the-senate/house-business/votes-proceeding
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
// The V&P title (used for the sitting date) is the PDF filename, typically
// of the form "Thursday, August 27, 2026 at 2.30pm.pdf".
package parliament

import (
        "context"
        "fmt"
        "strings"
        "time"
)

// VotesCandidate is a discovered Votes & Proceedings document.
type VotesCandidate struct {
        URL          string
        Title        string
        House        string
        SittingDate  time.Time
        SourceID     string
        DiscoveredAt time.Time
}

// DiscoverVotesProceedings discovers V&P documents from both the National
// Assembly and Senate listing pages. It walks the first maxPages pages of
// each listing (25 rows per page). The returned candidates are deduplicated
// by URL.
func (a *Adapter) DiscoverVotesProceedings(ctx context.Context) ([]VotesCandidate, error) {
        const maxPages = 5 // cap at 5 pages = 125 most-recent sittings per house
        var out []VotesCandidate
        for _, page := range []struct {
                url   string
                house string
        }{
                {a.naVotesURL, "National Assembly"},
                {a.senateVotesURL, "Senate"},
        } {
                cands, err := discoverPDFListing(a, ctx, page.url, page.house, maxPages, votesRowMapper)
                if err != nil {
                        // A single failing house does not abort discovery of the other.
                        continue
                }
                out = append(out, cands...)
        }
        out = dedupeVotesByURL(out)
        now := time.Now().UTC()
        for i := range out {
                out[i].DiscoveredAt = now
                if out[i].SourceID == "" {
                        out[i].SourceID = makeVotesSourceID(out[i].House, out[i].SittingDate, out[i].URL)
                }
        }
        return out, nil
}

// votesRowMapper converts a single PDF row from ParsePDFListing into a
// VotesCandidate. The sitting date is parsed from the PDF filename
// (which typically contains "August 27, 2026 at 2.30pm" or similar).
func votesRowMapper(href, linkText, house string) (VotesCandidate, bool) {
        if !strings.HasSuffix(strings.ToLower(href), ".pdf") {
                return VotesCandidate{}, false
        }
        title := decodePDFTitle(href, linkText)
        sittingDate := parseVotesSittingDate(title)
        return VotesCandidate{
                URL:         href,
                Title:       title,
                House:       house,
                SittingDate: sittingDate,
        }, true
}

// parseVotesSittingDate extracts the sitting date from a V&P title. V&P PDFs
// are typically named:
//   "Thursday, August 27, 2026 at 2.30pm"
//   "Wednesday 26Th August 2026"
//   "9 September 2026 (Afternoon Sitting)"
// The parser is tolerant of case, ordinals (26Th), and trailing time-of-day
// or sitting-label annotations.
func parseVotesSittingDate(title string) time.Time {
        clean := ordinalsRe.ReplaceAllString(title, "$1")
        return findDateInText(clean)
}

// makeVotesSourceID returns a deterministic source ID for a V&P document.
// Format: ke-parliament-<house-slug>-votes-YYYY-MM-DD. Falls back to a URL
// hash when the sitting date is unknown.
func makeVotesSourceID(house string, sitting time.Time, href string) string {
        houseSlug := "na"
        if strings.EqualFold(house, "Senate") {
                houseSlug = "senate"
        }
        if sitting.IsZero() {
                return "ke-parliament-" + houseSlug + "-votes-" + shortHash(href)
        }
        return fmt.Sprintf("ke-parliament-%s-votes-%04d-%02d-%02d",
                houseSlug, sitting.Year(), int(sitting.Month()), sitting.Day())
}

// dedupeVotesByURL removes candidates with duplicate URLs. First occurrence
// wins (the main archive is preferred over the "latest" sidebar block).
func dedupeVotesByURL(in []VotesCandidate) []VotesCandidate {
        seen := make(map[string]struct{}, len(in))
        out := make([]VotesCandidate, 0, len(in))
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

// FetchVotesProceedings downloads a V&P PDF.
func (a *Adapter) FetchVotesProceedings(ctx context.Context, url string) (string, error) {
        return a.fetchURL(ctx, url, "votes_proceedings")
}

// VoteRecord is a single vote in a V&P document.
//
// NOTE: real V&P parsing requires PDF text extraction + division-by-division
// or voice-vote heuristics. That pipeline is delegated to the documents
// service (services/documents). This HTML-only parser is intentionally a stub.
type VoteRecord struct {
        PersonID   string
        PersonName string
        Vote       string // aye, nay, abstain, absent
        BillRef    string
}

// ParseVotesProceedings extracts vote records from a V&P document.
func (a *Adapter) ParseVotesProceedings(htmlBody, sourceURL string) ([]VoteRecord, error) {
        _ = htmlBody
        _ = sourceURL
        return []VoteRecord{}, nil
}
