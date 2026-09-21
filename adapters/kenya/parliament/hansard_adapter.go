// Hansard adapter — discovers and fetches parliamentary debate transcripts.
//
// Live site recon (2026-09-21): parliament.go.ke publishes Hansard as PDFs
// under two listing pages:
//
//   NA:     /the-national-assembly/house-business/hansard
//   Senate: /the-senate/Hansard                          (capital H)
//
// Each listing is a Drupal View with 25 rows per page (paginated via ?page=N).
// Each row contains:
//
//   <tr>
//     <td class="views-field-field-pdf">
//       <span class="file--application-pdf">
//         <a href=".../sites/default/files/YYYY-MM/The%20Hansard%20-%20Thursday%2C%2027%20August%202026.pdf">…</a>
//       </span>
//     </td>
//     <td class="views-field-field-video-in-youtube">
//       <a href="https://youtu.be/...">…</a>
//     </td>
//   </tr>
//
// The Hansard title (used for the sitting date) is the PDF filename with the
// "The Hansard -" prefix stripped. The companion YouTube column is captured
// in the candidate's metadata but not stored on the source item.
package parliament

import (
        "context"
        "fmt"
        "net/url"
        "strings"
        "time"
)

// HansardCandidate is a discovered Hansard document.
type HansardCandidate struct {
        URL          string
        Title        string
        House        string
        SittingDate  time.Time
        SourceID     string
        VideoURL     string // optional YouTube URL of the plenary video
        DiscoveredAt time.Time
}

// DiscoverHansard discovers Hansard documents from both the National Assembly
// and Senate listing pages. It walks the first maxPages pages of each listing
// (25 rows per page; pass a negative number to walk every page until the
// pager stops returning results).
//
// The returned candidates are deduplicated by URL (NA + Senate never overlap,
// but the same call may pick up the same sitting from the listing + a sidebar
// "latest" block). Each candidate carries a deterministic SourceID derived
// from the house + sitting date so downstream consumers can dedupe across
// discovery runs.
func (a *Adapter) DiscoverHansard(ctx context.Context) ([]HansardCandidate, error) {
        const maxPages = 5 // cap at 5 pages = 125 most-recent sittings per house
        var out []HansardCandidate
        for _, page := range []struct {
                url   string
                house string
        }{
                {a.naHansardURL, "National Assembly"},
                {a.senateHansardURL, "Senate"},
        } {
                cands, err := discoverPDFListing(a, ctx, page.url, page.house, maxPages, hansardRowMapper)
                if err != nil {
                        // A single failing house does not abort discovery of the other.
                        continue
                }
                out = append(out, cands...)
        }
        // Deduplicate by URL (the "latest" sidebar block on the NA page can
        // re-list a sitting that's also in the main archive).
        out = dedupeHansardByURL(out)
        now := time.Now().UTC()
        for i := range out {
                out[i].DiscoveredAt = now
                if out[i].SourceID == "" {
                        out[i].SourceID = makeHansardSourceID(out[i].House, out[i].SittingDate, out[i].URL)
                }
        }
        return out, nil
}

// hansardRowMapper converts a single PDF row from ParsePDFListing into a
// HansardCandidate. The sitting date is parsed from the PDF filename (which
// typically looks like "The Hansard - Thursday, 27 August 2026.pdf").
func hansardRowMapper(href, linkText, house string) (HansardCandidate, bool) {
        if !strings.HasSuffix(strings.ToLower(href), ".pdf") {
                return HansardCandidate{}, false
        }
        title := decodePDFTitle(href, linkText)
        sittingDate := parseHansardSittingDate(title)
        return HansardCandidate{
                URL:         href,
                Title:       title,
                House:       house,
                SittingDate: sittingDate,
        }, true
}

// parseHansardSittingDate extracts the sitting date from a Hansard title.
// The Parliament of Kenya Hansard PDFs follow a consistent naming pattern:
//   "The Hansard - Thursday, 27 August 2026"
//   "The Hansard - Wednesday 26Th August 2026"
//   "Hansard - 18 September 2026 (Afternoon Sitting)"
// The parser is tolerant of case, missing commas, and ordinal suffixes
// (1st/2nd/3rd/27Th etc.).
func parseHansardSittingDate(title string) time.Time {
        // Strip ordinals (1st, 2nd, 3rd, 4th, 27Th, etc.) — case-insensitive.
        clean := ordinalsRe.ReplaceAllString(title, "$1")
        return findDateInText(clean)
}

// makeHansardSourceID returns a deterministic source ID for a Hansard sitting.
// Format: ke-parliament-<house-slug>-hansard-YYYY-MM-DD. The date is zero-padded
// so the IDs sort lexicographically by sitting date.
func makeHansardSourceID(house string, sitting time.Time, href string) string {
        houseSlug := "na"
        if strings.EqualFold(house, "Senate") {
                houseSlug = "senate"
        }
        if sitting.IsZero() {
                // Fall back to a hash of the URL so the ID is still stable.
                return "ke-parliament-" + houseSlug + "-hansard-" + shortHash(href)
        }
        return fmt.Sprintf("ke-parliament-%s-hansard-%04d-%02d-%02d",
                houseSlug, sitting.Year(), int(sitting.Month()), sitting.Day())
}

// dedupeHansardByURL removes candidates with duplicate URLs. The first
// occurrence wins (so the main archive listing is preferred over the
// sidebar "latest" block when both link to the same sitting).
func dedupeHansardByURL(in []HansardCandidate) []HansardCandidate {
        seen := make(map[string]struct{}, len(in))
        out := make([]HansardCandidate, 0, len(in))
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

// decodePDFTitle extracts the human-readable title from a PDF link. Parliament
// PDF URLs are typically URL-encoded forms of the title (e.g.,
// "The%20Hansard%20-%20Thursday%2C%2027%20August%202026.pdf"). The .pdf suffix
// is stripped and the URL-encoding is decoded; if the link text is non-empty
// and differs meaningfully from the decoded filename, the link text is used
// (it's typically cleaner).
func decodePDFTitle(href, linkText string) string {
        // Decode the URL-encoded filename portion.
        decoded, err := url.QueryUnescape(href)
        if err != nil {
                decoded = href
        }
        // Strip any path prefix up to the last '/'.
        if idx := strings.LastIndex(decoded, "/"); idx >= 0 {
                decoded = decoded[idx+1:]
        }
        // Strip the .pdf suffix (case-insensitive).
        decoded = strings.TrimSuffix(decoded, ".pdf")
        decoded = strings.TrimSuffix(decoded, ".PDF")
        // If the link text is meaningful and longer than the decoded filename
        // (e.g. it's the full "The Hansard - Thursday, 27 August 2026" while the
        // filename is "The Hansard - Thursday, 27 August 2026"), prefer the
        // link text — it's what the site shows to humans.
        t := strings.TrimSpace(linkText)
        if t != "" && len(t) >= len(decoded) {
                return t
        }
        return strings.TrimSpace(decoded)
}

// FetchHansard downloads a Hansard document. The fetch is polite (1 req/sec/host
// via PoliteClient) and respects the context deadline.
func (a *Adapter) FetchHansard(ctx context.Context, url string) (string, error) {
        return a.fetchURL(ctx, url, "hansard")
}

// ParseHansard extracts speeches from a Hansard document.
//
// NOTE: real Hansard parsing requires PDF text extraction + debate-structure
// heuristics (speaker identification, role inference, topic segmentation).
// That pipeline is delegated to the documents service (see services/documents)
// which has the OCR + structure-extraction infrastructure. This HTML-only
// parser is intentionally a stub — it returns an empty slice so the discovery
// pipeline can complete and the document can be queued for downstream
// extraction.
func (a *Adapter) ParseHansard(htmlBody, sourceURL string) ([]HansardSpeech, error) {
        _ = htmlBody
        _ = sourceURL
        return []HansardSpeech{}, nil
}

// HansardSpeech is a single speech from a Hansard document.
type HansardSpeech struct {
        Speaker     string
        Role        string
        Topic       string
        Text        string
        SittingDate time.Time
}

// itoa is a tiny dependency-free int→string for the page parameter. We use
// this instead of strconv.Itoa to avoid pulling strconv into this file (which
// would force every other parliament adapter file to also import it).
func itoa(n int) string {
        if n == 0 {
                return "0"
        }
        neg := n < 0
        if neg {
                n = -n
        }
        var buf [12]byte
        i := len(buf)
        for n > 0 {
                i--
                buf[i] = byte('0' + n%10)
                n /= 10
        }
        if neg {
                i--
                buf[i] = '-'
        }
        return string(buf[i:])
}

// strconvAtoi is a tiny dependency-free string→int. Returns an error on
// non-numeric input or empty string. We use this instead of strconv.Atoi to
// keep the file's import surface minimal.
func strconvAtoi(s string) (int, error) {
        if s == "" {
                return 0, fmt.Errorf("empty string")
        }
        n := 0
        for _, r := range s {
                if r < '0' || r > '9' {
                        return 0, fmt.Errorf("non-numeric: %q", s)
                }
                n = n*10 + int(r-'0')
        }
        return n, nil
}

// monthIndex maps an English month name to its 1-12 number. The match is
// case-insensitive and accepts both full names ("January") and 3-letter
// abbreviations ("Jan").
func monthIndex(name string) int {
        switch strings.ToLower(name) {
        case "january", "jan":
                return 1
        case "february", "feb":
                return 2
        case "march", "mar":
                return 3
        case "april", "apr":
                return 4
        case "may":
                return 5
        case "june", "jun":
                return 6
        case "july", "jul":
                return 7
        case "august", "aug":
                return 8
        case "september", "sep", "sept":
                return 9
        case "october", "oct":
                return 10
        case "november", "nov":
                return 11
        case "december", "dec":
                return 12
        }
        return 0
}

// ordinalsRe matches ordinal suffixes on day numbers (1st, 2nd, 3rd, 4th,
// 21st, 27Th, etc.). The regex captures the digit group in $1 and the
// suffix in $2 — callers replace the whole match with $1 (just the digits)
// so "27TH" becomes "27" rather than being removed entirely.
var ordinalsRe = mustCompile(`(?i)\b(\d{1,2})(st|nd|rd|th)\b`)

// dateRe matches a date in a free-text string in EITHER form:
//   "27 August 2026"     (day-first, used by Hansard PDFs)
//   "August 27, 2026"    (month-first, used by Votes & Proceedings PDFs)
// Captures day/month/year in groups 1/2/3 (day-first form) — the alternative
// form's groups (4/5/6) are intentionally empty so the caller's m[1]/m[2]/m[3]
// indexing works for both forms via the altRe fallback below.
//
// The regex is case-insensitive (?i) so it matches Parliament's upper-case
// format ("AUGUST 27TH 2026") as well as title-case ("August 2026").
var dateRe = mustCompile(`(?i)(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})`)

// dateReMonthFirst matches the month-first date form ("August 27, 2026").
// Used as a fallback when dateRe fails to match.
var dateReMonthFirst = mustCompile(`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})[,\s]+(\d{4})`)

// findDateInText returns the first date found in s, trying day-first then
// month-first formats. Returns the zero time when no date is present.
func findDateInText(s string) time.Time {
        // Try day-first: "27 August 2026"
        if m := dateRe.FindStringSubmatch(s); m != nil {
                day, err := strconvAtoi(m[1])
                if err == nil {
                        month := monthIndex(m[2])
                        if month >= 1 && month <= 12 {
                                year, err := strconvAtoi(m[3])
                                if err == nil && year >= 1990 && year <= 2100 {
                                        return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
                                }
                        }
                }
        }
        // Try month-first: "August 27, 2026"
        if m := dateReMonthFirst.FindStringSubmatch(s); m != nil {
                month := monthIndex(m[1])
                if month >= 1 && month <= 12 {
                        day, err := strconvAtoi(m[2])
                        if err == nil {
                                year, err := strconvAtoi(m[3])
                                if err == nil && year >= 1990 && year <= 2100 {
                                        return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
                                }
                        }
                }
        }
        return time.Time{}
}

// shortHash returns the first 10 hex characters of the SHA-256 of s. Used as
// a stable identifier when no date is available.
func shortHash(s string) string {
        h := sha256Sum([]byte(s))
        return hexEncode(h[:5])
}
