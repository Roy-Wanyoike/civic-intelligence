// Package main — RSS 2.0 feed generator (issue #280).
//
// This file implements the platform's RSS feeds: a small, stdlib-only
// generator (Feed → renderRSS → string) plus four HTTP handlers that
// reuse the JSON endpoints' data sources but emit application/rss+xml
// instead of application/json. RSS readers (Feedly, Inoreader, NetNewsWire)
// can subscribe without creating an account — the feeds are public, the
// same way the underlying /api/v1/bills, /api/v1/what-changed,
// /api/v1/brief/today, and /api/v1/people/{id} endpoints are.
//
// Routes (registered in main.go):
//
//      GET /api/v1/feed/bills.rss            — latest Bills (Kenya Law adapter)
//      GET /api/v1/feed/what-changed.rss     — latest verified civic changes
//      GET /api/v1/feed/brief.rss            — latest Civic Daily Brief
//      GET /api/v1/feed/people/{id}.rss      — per-MP activity (scorecard timeline)
//
// The generator deliberately uses only encoding/xml (no external deps).
// RSS 2.0 spec: https://www.rssboard.org/rss-specification
package main

import (
        "context"
        "encoding/xml"
        "fmt"
        "log"
        "net/http"
        "os"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
)

// --- Public types (consumed by callers + tests) ---

// Feed is the flat, renderer-agnostic RSS 2.0 feed shape. Callers populate
// Title / Description / Link / Items without touching encoding/xml — the
// <rss version="2.0"><channel>…</channel></rss> envelope is the renderer's
// concern, not the caller's.
type Feed struct {
        Title       string
        Description string
        Link        string
        Items       []FeedItem
}

// FeedItem mirrors a single RSS <item> element. PubDate is the
// RFC-822-formatted timestamp required by RSS 2.0 §4.3. When GUID is left
// empty the renderer falls back to the Link (RSS 2.0 §4.3.9 lets a guid
// equal the link with isPermaLink="true" — that's the default behaviour).
type FeedItem struct {
        Title       string
        Description string
        Link        string
        PubDate     string
        GUID        string
        Category    string
}

// --- Internal encoding/xml shape (private) ---

// rssXML is the on-the-wire RSS 2.0 root element. Kept private so callers
// cannot break the <rss version="2.0"><channel>…</channel></rss> envelope
// by constructing an invalid top-level shape.
type rssXML struct {
        XMLName xml.Name   `xml:"rss"`
        Version string     `xml:"version,attr"`
        Channel rssChannel `xml:"channel"`
}

// rssChannel is the <channel> element required by RSS 2.0 §4. The spec
// requires title / link / description as the first three child elements
// (in any order — encoding/xml respects the struct field order here).
type rssChannel struct {
        Title       string    `xml:"title"`
        Description string    `xml:"description"`
        Link        string    `xml:"link"`
        Items       []rssItem `xml:"item"`
}

// rssItem is a single <item> inside <channel>. omitempty tags keep empty
// fields out of the rendered XML so the feed stays compact for items that
// omit a category or pubDate.
type rssItem struct {
        Title       string `xml:"title"`
        Description string `xml:"description"`
        Link        string `xml:"link"`
        PubDate     string `xml:"pubDate,omitempty"`
        GUID        string `xml:"guid,omitempty"`
        Category    string `xml:"category,omitempty"`
}

// rssContentType is the documented RSS 2.0 content type. Set verbatim on
// every /api/v1/feed/*.rss response so RSS readers parse without
// content-type negotiation. The charset suffix matches what Feedly +
// Inoreader send in their Accept headers.
const rssContentType = "application/rss+xml; charset=utf-8"

// renderRSS renders the supplied Feed as RSS 2.0 XML (with the standard
// <?xml version="1.0" encoding="UTF-8"?> prolog). Returns an error if
// the underlying encoding/xml encoder fails — Go's encoder is permissive
// (it auto-escapes &, <, >, ", '), so an error here would only surface
// from a custom Marshaler or an I/O failure, neither of which the stdlib
// types trigger. The error path is still wired so callers can route
// failures to writeError instead of panicking.
//
// Empty Item.GUID values fall back to Item.Link — RSS 2.0 §4.3.9 lets the
// guid equal the link with isPermaLink="true" (the default), so RSS
// readers deduplicate items by their canonical URL.
func renderRSS(feed Feed) (string, error) {
        items := make([]rssItem, 0, len(feed.Items))
        for _, it := range feed.Items {
                guid := it.GUID
                if guid == "" {
                        guid = it.Link
                }
                items = append(items, rssItem{
                        Title:       it.Title,
                        Description: it.Description,
                        Link:        it.Link,
                        PubDate:     it.PubDate,
                        GUID:        guid,
                        Category:    it.Category,
                })
        }

        out := rssXML{
                Version: "2.0",
                Channel: rssChannel{
                        Title:       feed.Title,
                        Description: feed.Description,
                        Link:        feed.Link,
                        Items:       items,
                },
        }

        var buf strings.Builder
        buf.WriteString(xml.Header)
        enc := xml.NewEncoder(&buf)
        enc.Indent("", "  ")
        if err := enc.Encode(out); err != nil {
                return "", fmt.Errorf("encode rss: %w", err)
        }
        if err := enc.Flush(); err != nil {
                return "", fmt.Errorf("flush rss: %w", err)
        }
        return buf.String(), nil
}

// writeRSS renders the feed, sets the RSS content type, and writes the
// body. On render failure it delegates to writeError so the caller still
// gets a structured JSON error (the response is already in the
// application/rss+xml content-type at that point, but the body remains a
// valid error envelope — RSS readers will surface the failure).
func writeRSS(w http.ResponseWriter, feed Feed) error {
        body, err := renderRSS(feed)
        if err != nil {
                return err
        }
        w.Header().Set("Content-Type", rssContentType)
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(body))
        return nil
}

// rssPublicLinkBase is the public-facing web origin prepended to deep
// links in the feed channel Link field. RSS readers display this as the
// "website" link alongside each item. Overridable via WEB_BASE_URL so
// staging deployments don't link to the production domain.
var rssPublicLinkBase = func() string {
        if v := envOr("WEB_BASE_URL", ""); v != "" {
                return v
        }
        return "https://civicintelligence.com"
}()

// envOr returns os.Getenv(key) or the fallback when the env var is empty
// / unset. Kept local to rss.go so the file stays self-contained.
func envOr(key, fallback string) string {
        if v := strings.TrimSpace(osGetenv(key)); v != "" {
                return v
        }
        return fallback
}

// osGetenv is a thin wrapper around os.Getenv. It exists so rss.go can be
// tested without importing "os" directly — keeps the test surface
// minimal and lets a future refactor swap to a config struct without
// touching every call site.
func osGetenv(key string) string {
        return os.Getenv(key)
}

// formatPubDate converts a time.Time into the RFC-822-formatted pubDate
// RSS 2.0 requires (e.g. "Wed, 02 Oct 2024 12:00:00 +0000"). Returns an
// empty string for the zero value so empty dates don't pollute the feed
// with "Mon, 01 Jan 0001 00:00:00 +0000" garbage.
func formatPubDate(t time.Time) string {
        if t.IsZero() {
                return ""
        }
        return t.UTC().Format(time.RFC1123Z)
}

// --- HTTP handlers ---

// makeBillsRSSFeedHandler returns the handler for
// GET /api/v1/feed/bills.rss. It reuses the same BillsAdapter the JSON
// /api/v1/bills endpoint uses (issue #265) — same data source, same
// degraded-fallback contract — but renders the result as RSS 2.0 XML.
//
// Each Bill becomes a single <item> whose GUID is the Bill's stable
// SourceID (so RSS readers deduplicate across re-fetches even when the
// upstream slug changes) and whose Category is the House (National
// Assembly vs Senate).
func makeBillsRSSFeedHandler(adapter BillsAdapter) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
                        return
                }
                ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
                defer cancel()

                // Issue #265: same live-or-seed fallback as the JSON endpoint.
                bills, err := adapter.DiscoverBills(ctx)
                if err != nil {
                        log.Printf("bills.rss: live crawl failed, falling back to seed data: %v", err)
                        bills = kenya_seed.SampleBills
                }

                items := make([]FeedItem, 0, len(bills))
                for _, b := range bills {
                        pubDate := formatPubDate(b.PublicationDate)
                        if pubDate == "" {
                                // Bills without a publication date still need a pubDate so
                                // RSS readers sort them after dated items. Use now() so the
                                // reader surfaces them at the top of the inbox on first
                                // contact, then they settle into chronological order.
                                pubDate = time.Now().UTC().Format(time.RFC1123Z)
                        }
                        items = append(items, FeedItem{
                                Title:       b.Title,
                                Description: fmt.Sprintf("Bill %s published in the %s. Source: %s", b.Slug, b.House, b.URL),
                                Link:        b.URL,
                                PubDate:     pubDate,
                                GUID:        b.SourceID,
                                Category:    b.House,
                        })
                }

                feed := Feed{
                        Title:       "Civic Intelligence — Bills",
                        Description: "Latest Bills before Kenya Parliament, sourced from new.kenyalaw.org.",
                        Link:        rssPublicLinkBase + "/bills",
                        Items:       items,
                }
                if err := writeRSS(w, feed); err != nil {
                        log.Printf("bills.rss: render failed: %v", err)
                        writeError(w, http.StatusInternalServerError, "rss_error", "failed to render feed")
                }
        }
}

// makeWhatChangedRSSFeedHandler returns the handler for
// GET /api/v1/feed/what-changed.rss. Mirrors the JSON /what-changed
// endpoint's data pipeline (DiscoverBills → ChangeItem) but emits RSS.
//
// Each item carries the change's significance (INFORMATIONAL / SUBSTANTIVE
// / HIGH_IMPACT) as the RSS <category> so subscribers can filter their
// feed reader by severity.
func makeWhatChangedRSSFeedHandler(kenyaLaw *kenya_law.Adapter) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
                        return
                }
                ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
                defer cancel()

                bills, err := kenyaLaw.DiscoverBills(ctx)
                if err != nil {
                        // Mirror the JSON endpoint's empty-list contract — RSS readers
                        // should still see a valid (if empty) feed rather than a 5xx.
                        log.Printf("what-changed.rss: adapter error: %v", err)
                        bills = []kenya_law.BillCandidate{}
                }

                items := make([]FeedItem, 0, len(bills))
                for _, b := range bills {
                        significance := classifyBillSignificance(b.Title)
                        pubDate := formatPubDate(b.PublicationDate)
                        if pubDate == "" {
                                pubDate = time.Now().UTC().Format(time.RFC1123Z)
                        }
                        items = append(items, FeedItem{
                                Title:       b.Title,
                                Description: fmt.Sprintf("New Bill published on Kenya Law. Significance: %s. House: %s.", significance, b.House),
                                Link:        b.URL,
                                PubDate:     pubDate,
                                GUID:        b.URL,
                                Category:    significance,
                        })
                }

                feed := Feed{
                        Title:       "Civic Intelligence — What Changed",
                        Description: "Proactive feed of verified civic changes. Every item links to evidence.",
                        Link:        rssPublicLinkBase + "/what-changed",
                        Items:       items,
                }
                if err := writeRSS(w, feed); err != nil {
                        log.Printf("what-changed.rss: render failed: %v", err)
                        writeError(w, http.StatusInternalServerError, "rss_error", "failed to render feed")
                }
        }
}

// classifyBillSignificance mirrors the heuristic used by the JSON
// /what-changed endpoint (issue #186). Factored out so the RSS handler
// can apply the same classification without duplicating the strings.
func classifyBillSignificance(title string) string {
        tl := strings.ToLower(title)
        switch {
        case strings.Contains(tl, "amendment"):
                return "SUBSTANTIVE"
        case strings.Contains(tl, "finance") || strings.Contains(tl, "appropriation"):
                return "HIGH_IMPACT"
        default:
                return "INFORMATIONAL"
        }
}

// makeBriefRSSFeedHandler returns the handler for
// GET /api/v1/feed/brief.rss. Renders the latest generated briefs from
// the briefStore as a single-item-per-brief RSS feed — the brief
// headline becomes the item title, the AI summary becomes the
// description, and the deep link points to /briefing?id={briefID}.
//
// When the store is empty (e.g. before the first /brief/generate call of
// the day), the feed renders with zero items so RSS readers see a valid
// but empty feed rather than a 5xx.
func makeBriefRSSFeedHandler(store *briefStore) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
                        return
                }
                briefs := store.list()

                items := make([]FeedItem, 0, len(briefs))
                for _, b := range briefs {
                        // Parse the brief's RFC3339 generated_at timestamp into the
                        // RFC-822 pubDate RSS requires. On parse failure, fall back to
                        // the brief date itself (a YYYY-MM-DD string) — RSS readers
                        // are permissive about pubDate formats even though the spec
                        // mandates RFC-822.
                        pubDate := b.Date
                        if t, err := time.Parse(time.RFC3339, b.GeneratedAt); err == nil {
                                pubDate = t.UTC().Format(time.RFC1123Z)
                        }
                        items = append(items, FeedItem{
                                Title:       b.Headline,
                                Description: b.AISummary,
                                Link:        fmt.Sprintf("%s/briefing?id=%s", rssPublicLinkBase, b.ID),
                                PubDate:     pubDate,
                                GUID:        b.ID,
                                Category:    "Daily Brief",
                        })
                }

                feed := Feed{
                        Title:       "Civic Intelligence — Daily Brief",
                        Description: "AI-grounded, evidence-cited daily summary of civic developments in Kenya.",
                        Link:        rssPublicLinkBase + "/briefing",
                        Items:       items,
                }
                if err := writeRSS(w, feed); err != nil {
                        log.Printf("brief.rss: render failed: %v", err)
                        writeError(w, http.StatusInternalServerError, "rss_error", "failed to render feed")
                }
        }
}

// makeMPRSSFeedHandler returns the handler for
// GET /api/v1/feed/people/{id}.rss. Aggregates per-MP activity — Bills
// sponsored, speeches, votes, questions — sourced from the same
// sampleScorecards dataset the /api/v1/people/{id}/scorecard endpoint
// uses (issue ENG-K2).
//
// Each activity becomes a single <item> whose Category is the activity
// kind ("bill_sponsored", "vote", "question", "statement") so
// subscribers can filter their feed reader by activity type. The
// platform NEVER aggregates these into a political-performance score
// (rule: NO_POLITICAL_PERFORMANCE_SCORE) — the feed is a raw timeline.
func makeMPRSSFeedHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
                        return
                }
                // Path shape: /api/v1/feed/people/{id}.rss
                // Strip the route prefix + the .rss suffix. Handles both with-slash
                // and no-slash registrations (issue #266).
                id := strings.TrimPrefix(r.URL.Path, "/api/v1/feed/people")
                id = strings.TrimPrefix(id, "/")
                id = strings.TrimSuffix(id, ".rss")
                id = strings.TrimSuffix(id, "/")
                if id == "" {
                        writeError(w, http.StatusBadRequest, "bad_request", "person ID required")
                        return
                }

                sc := findScorecard(id)
                if sc == nil {
                        writeError(w, http.StatusNotFound, "not_found", "person not found: "+id)
                        return
                }

                items := make([]FeedItem, 0, len(sc.RecentActivity))
                for _, act := range sc.RecentActivity {
                        items = append(items, FeedItem{
                                Title:       act.Title,
                                Description: act.Detail,
                                Link:        act.SourceURL,
                                PubDate:     formatScorecardDate(act.Date),
                                GUID:        act.SourceURL,
                                Category:    act.Kind,
                        })
                }

                feed := Feed{
                        Title:       fmt.Sprintf("Civic Intelligence — %s activity", sc.Name),
                        Description: fmt.Sprintf("Bills sponsored, speeches, and votes for %s (%s). Source: parliament.go.ke.", sc.Name, sc.Role),
                        Link:        fmt.Sprintf("%s/people/%s", rssPublicLinkBase, sc.PersonID),
                        Items:       items,
                }
                if err := writeRSS(w, feed); err != nil {
                        log.Printf("people/%s.rss: render failed: %v", id, err)
                        writeError(w, http.StatusInternalServerError, "rss_error", "failed to render feed")
                }
        }
}

// formatScorecardDate converts the scorecard date format ("YYYY-MM-DD")
// into the RFC-822 pubDate RSS 2.0 requires. Returns an empty string on
// parse failure rather than a placeholder — RSS readers handle missing
// pubDates gracefully (they sort the item to the top of the inbox).
func formatScorecardDate(s string) string {
        t, err := time.Parse("2006-01-02", s)
        if err != nil {
                return ""
        }
        return t.UTC().Format(time.RFC1123Z)
}
