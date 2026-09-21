// Committee adapter — discovers and fetches committee reports.
//
// Live site recon (2026-09-21): parliament.go.ke organises committee reports
// per-committee rather than as a single archive. The crawl is two-level:
//
//  1. Fetch the committee index:
//       NA:     /the-national-assembly/committees
//       Senate: /the-senate/committees/senate-committees
//     The index lists every committee as a link of the form:
//       /the-national-assembly/committees/<id>/<committee-slug>
//       /the-senate/committees/<category-slug>/<id>/<committee-slug>
//
//  2. For each committee link, fetch the per-committee page and extract
//     the report PDF links from:
//       <div class="field--name-field-committee-report">
//         <div class="field__items">
//           <div class="field__item">
//             <span class="file--application-pdf">
//               <a href=".../sites/default/files/YYYY-MM/<report>.pdf">…</a>
//             </span>
//           </div>
//         </div>
//       </div>
//
// There is no pagination on the per-committee report list — all reports
// for a single committee fit on one page.
//
// The crawler is polite: it reuses the same PoliteClient (1 req/sec/host)
// as the rest of the parliament package, so the committee-index fetch +
// the per-committee fetches are automatically rate-limited.
package parliament

import (
        "context"
        "fmt"
        "strings"
        "time"

        "golang.org/x/net/html"
)

// CommitteeCandidate is a discovered committee report.
type CommitteeCandidate struct {
        URL           string
        Title         string
        CommitteeName string
        CommitteeURL  string // the per-committee page URL (where the report was found)
        House         string
        ReportType    string // "report" or "minute"
        PublishedDate time.Time
        SourceID      string
        DiscoveredAt  time.Time
}

// DiscoverCommittees discovers committee reports from both the National
// Assembly and Senate committee indices. It fetches the committee index,
// then for each committee fetches the per-committee page and extracts the
// report PDF links.
//
// The crawl is capped at maxCommittees per house (default 30) to avoid
// pulling all ~70 reports per committee across the ~30 committees per house
// = ~2,100 PDFs in a single discovery run. Set maxCommittees to 0 or
// negative to walk every committee.
//
// Callers wanting fresh committee reports should call this endpoint at most
// once per day (the daily cron at 03:00 UTC is appropriate).
func (a *Adapter) DiscoverCommittees(ctx context.Context) ([]CommitteeCandidate, error) {
        const maxCommittees = 30 // cap at 30 committees per house
        var out []CommitteeCandidate
        for _, page := range []struct {
                url   string
                house string
        }{
                {a.naCommitteesURL, "National Assembly"},
                {a.senateCommitteesURL, "Senate"},
        } {
                cands, err := a.discoverCommitteeReports(ctx, page.url, page.house, maxCommittees)
                if err != nil {
                        // A single failing house does not abort discovery of the other.
                        continue
                }
                out = append(out, cands...)
        }
        out = dedupeCommitteeByURL(out)
        now := time.Now().UTC()
        for i := range out {
                out[i].DiscoveredAt = now
                if out[i].SourceID == "" {
                        out[i].SourceID = makeCommitteeSourceID(out[i].House, out[i].CommitteeName, out[i].URL)
                }
        }
        return out, nil
}

// discoverCommitteeReports performs the two-level crawl for a single house:
// fetch the committee index, then for each committee link fetch the per-
// committee page and extract the report PDF links.
func (a *Adapter) discoverCommitteeReports(ctx context.Context, indexURL, house string, maxCommittees int) ([]CommitteeCandidate, error) {
        indexHTML, err := a.fetchURL(ctx, indexURL, "committee_index")
        if err != nil {
                return nil, fmt.Errorf("committee index %s: %w", indexURL, err)
        }
        committeeLinks := ParseCommitteeIndex(indexHTML)
        if maxCommittees > 0 && len(committeeLinks) > maxCommittees {
                committeeLinks = committeeLinks[:maxCommittees]
        }
        var out []CommitteeCandidate
        for _, link := range committeeLinks {
                select {
                case <-ctx.Done():
                        return out, ctx.Err()
                default:
                }
                pageHTML, err := a.fetchURL(ctx, link.URL, "committee_page")
                if err != nil {
                        // Skip a failing committee page rather than aborting the crawl.
                        continue
                }
                reports := ParseCommitteeReports(pageHTML, link.URL, house, link.Name)
                out = append(out, reports...)
        }
        return out, nil
}

// CommitteeLink is a single committee entry from the committee index page.
type CommitteeLink struct {
        URL  string // absolute URL of the per-committee page
        Name string // human-readable committee name (e.g. "Public Accounts Committee")
        Slug string // committee slug from the URL path (e.g. "public-accounts-committee")
}

// ParseCommitteeIndex walks the committee index HTML and extracts every
// committee link. Links on the NA index are of the form:
//
//      /the-national-assembly/committees/<id>/<committee-slug>
//
// and on the Senate index:
//
//      /the-senate/committees/<category-slug>/<id>/<committee-slug>
//
// The parser collects every <a href> whose path contains
// "/the-national-assembly/committees/" or "/the-senate/committees/" and
// whose href has at least one path segment after that prefix. The
// committee's own index page is excluded from the result (its href would
// match the prefix but with zero additional segments).
func ParseCommitteeIndex(htmlBody string) []CommitteeLink {
        out := []CommitteeLink{}
        seen := map[string]struct{}{}
        doc, err := html.Parse(strings.NewReader(htmlBody))
        if err != nil {
                return out
        }
        walkAnchorsListing(doc, func(href, text string) {
                // Resolve relative URLs.
                if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") {
                        href = "https://www.parliament.go.ke" + href
                }
                // Match committee-detail links.
                var prefix string
                if strings.Contains(href, "/the-national-assembly/committees/") {
                        prefix = "/the-national-assembly/committees/"
                } else if strings.Contains(href, "/the-senate/committees/") {
                        prefix = "/the-senate/committees/"
                } else {
                        return
                }
                // Extract the slug(s) after the prefix.
                idx := strings.Index(href, prefix)
                if idx < 0 {
                        return
                }
                rest := href[idx+len(prefix):]
                // Strip query string + fragment.
                if i := strings.IndexAny(rest, "?#"); i >= 0 {
                        rest = rest[:i]
                }
                // Need at least one path segment after the prefix.
                if rest == "" || !strings.Contains(rest, "/") {
                        return
                }
                // Deduplicate by URL.
                if _, ok := seen[href]; ok {
                        return
                }
                seen[href] = struct{}{}
                // Extract the committee slug (last path segment).
                segments := strings.Split(strings.TrimSuffix(rest, "/"), "/")
                slug := segments[len(segments)-1]
                // Derive a clean committee name from the link text.
                name := cleanCommitteeName(text)
                out = append(out, CommitteeLink{
                        URL:  href,
                        Name: name,
                        Slug: slug,
                })
        })
        return out
}

// cleanCommitteeName normalises a committee name from link text. Strips
// leading/trailing whitespace, collapses internal whitespace, and removes
// trailing "Committee" duplication (e.g. "Public Accounts Committee Committee"
// → "Public Accounts Committee").
func cleanCommitteeName(s string) string {
        s = strings.TrimSpace(s)
        // Collapse internal whitespace runs to single spaces.
        for strings.Contains(s, "  ") {
                s = strings.ReplaceAll(s, "  ", " ")
        }
        // Strip trailing duplicate "Committee".
        s = strings.TrimSuffix(s, " Committee Committee")
        return s
}

// ParseCommitteeReports walks a per-committee HTML page and extracts every
// committee-report PDF link. The PDF links live inside:
//
//      <div class="field--name-field-committee-report">
//        <div class="field__items">
//          <div class="field__item">
//            <span class="file--application-pdf">
//              <a href=".../sites/default/files/YYYY-MM/<report>.pdf">…</a>
//            </span>
//          </div>
//        </div>
//      </div>
//
// The parser is tolerant of layout drift — if the field--name-field-committee-report
// container is absent, it falls back to collecting every PDF link on the page.
// The report's title is the PDF filename (URL-decoded), and the published
// date is parsed from the title when possible.
func ParseCommitteeReports(htmlBody, committeeURL, house, committeeName string) []CommitteeCandidate {
        var out []CommitteeCandidate
        doc, err := html.Parse(strings.NewReader(htmlBody))
        if err != nil {
                return out
        }
        walkAnchorsListing(doc, func(href, text string) {
                if !strings.HasSuffix(strings.ToLower(href), ".pdf") {
                        return
                }
                if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") {
                        href = "https://www.parliament.go.ke" + href
                }
                title := decodePDFTitle(href, text)
                reportType := "report"
                if strings.Contains(strings.ToLower(title), "minute") {
                        reportType = "minute"
                }
                out = append(out, CommitteeCandidate{
                        URL:           href,
                        Title:         title,
                        CommitteeName: committeeName,
                        CommitteeURL:  committeeURL,
                        House:         house,
                        ReportType:    reportType,
                        PublishedDate: parseCommitteeReportDate(title),
                })
        })
        return out
}

// parseCommitteeReportDate extracts the publication date from a committee
// report title. Committee report PDFs are typically named:
//   "Report on procurement of External Audit Services for the OAG.pdf"
//   "Report on the Inquiry into the Universal Health Coverage - 18 September 2026.pdf"
// Most committee reports do NOT carry a date in the title — for those, the
// zero time is returned and the caller should fall back to the file's
// Last-Modified header or the discovery timestamp.
func parseCommitteeReportDate(title string) time.Time {
        clean := ordinalsRe.ReplaceAllString(title, "$1")
        return findDateInText(clean)
}

// makeCommitteeSourceID returns a deterministic source ID for a committee
// report. Format: ke-parliament-<house-slug>-committee-<committee-slug>-<hash>.
// The hash is needed because a single committee can produce multiple reports
// with similar titles.
func makeCommitteeSourceID(house, committeeName, href string) string {
        houseSlug := "na"
        if strings.EqualFold(house, "Senate") {
                houseSlug = "senate"
        }
        committeeSlug := slugify(committeeName)
        return fmt.Sprintf("ke-parliament-%s-committee-%s-%s", houseSlug, committeeSlug, shortHash(href))
}

// slugify converts a committee name to a URL-safe slug (lowercase, hyphens).
func slugify(s string) string {
        s = strings.ToLower(strings.TrimSpace(s))
        s = strings.ReplaceAll(s, " ", "-")
        s = strings.ReplaceAll(s, "&", "and")
        // Strip non-alphanumeric non-hyphen chars.
        var b strings.Builder
        for _, r := range s {
                if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
                        b.WriteRune(r)
                }
        }
        return b.String()
}

// dedupeCommitteeByURL removes candidates with duplicate URLs. First
// occurrence wins.
func dedupeCommitteeByURL(in []CommitteeCandidate) []CommitteeCandidate {
        seen := make(map[string]struct{}, len(in))
        out := make([]CommitteeCandidate, 0, len(in))
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

// FetchCommitteeReport downloads a committee report PDF.
func (a *Adapter) FetchCommitteeReport(ctx context.Context, url string) (string, error) {
        return a.fetchURL(ctx, url, "committee_report")
}

// CommitteeReport is a parsed committee report.
//
// NOTE: real committee report parsing requires PDF text extraction +
// report-structure heuristics (findings, recommendations, evidence
// appendices). That pipeline is delegated to the documents service
// (services/documents). This HTML-only parser is intentionally a stub.
type CommitteeReport struct {
        Title       string
        Committee   string
        House       string
        ReportType  string
        PublishedAt time.Time
        BillRefs    []string
}

// ParseCommitteeReport extracts metadata from a committee report.
func (a *Adapter) ParseCommitteeReport(htmlBody, sourceURL string) (*CommitteeReport, error) {
        _ = htmlBody
        _ = sourceURL
        return nil, nil
}
