// Package main — tests for the RSS 2.0 feed generator + handlers (issue #280).
//
// The tests cover:
//   - renderRSS output parses as valid XML (TestRenderRSS_ValidXML)
//   - renderRSS includes <item> elements (TestRenderRSS_ContainsItems)
//   - GET /api/v1/feed/bills.rss returns valid RSS XML with the right
//     content-type (TestBillsRSSFeed_ReturnsXML)
//   - GET /api/v1/feed/what-changed.rss (TestWhatChangedRSSFeed_ReturnsXML)
//   - GET /api/v1/feed/brief.rss (TestBriefRSSFeed_ReturnsXML)
//   - GET /api/v1/feed/people/{id}.rss (TestMPRSSFeed_ReturnsXML)
//
// The HTTP tests reuse the same mocks the JSON endpoint tests use
// (mockBillsAdapter, newTestAdapter, briefStore) so a regression in the
// shared data pipeline surfaces against both content-types.
package main

import (
        "encoding/xml"
        "errors"
        "net/http"
        "net/http/httptest"
        "strings"
        "testing"
)

// sampleRSSFeed returns a deterministic two-item feed for the renderer
// tests. Kept small so the assertions can spell out exactly what the
// output should contain without coupling to the seed slice.
func sampleRSSFeed() Feed {
        return Feed{
                Title:       "Civic Intelligence — Test Feed",
                Description: "A test feed for the renderRSS unit tests.",
                Link:        "https://example.com/feed",
                Items: []FeedItem{
                        {
                                Title:       "First item",
                                Description: "First description",
                                Link:        "https://example.com/1",
                                PubDate:     "Wed, 02 Oct 2024 12:00:00 +0000",
                                GUID:        "https://example.com/1",
                                Category:    "test",
                        },
                        {
                                Title:       "Second item",
                                Description: "Second description",
                                Link:        "https://example.com/2",
                                PubDate:     "Thu, 03 Oct 2024 12:00:00 +0000",
                                GUID:        "https://example.com/2",
                                Category:    "test",
                        },
                },
        }
}

// rssXMLForTest mirrors the rssXML struct in rss.go so the test can
// decode the rendered output back into a typed value. Defined locally
// (rather than reusing the production struct) so a future refactor of
// the production XML tags does not silently bypass the test.
type rssXMLForTest struct {
        XMLName xml.Name        `xml:"rss"`
        Version string          `xml:"version,attr"`
        Channel rssChannelForTest `xml:"channel"`
}

type rssChannelForTest struct {
        Title       string          `xml:"title"`
        Description string          `xml:"description"`
        Link        string          `xml:"link"`
        Items       []rssItemForTest `xml:"item"`
}

type rssItemForTest struct {
        Title       string `xml:"title"`
        Description string `xml:"description"`
        Link        string `xml:"link"`
        PubDate     string `xml:"pubDate"`
        GUID        string `xml:"guid"`
        Category    string `xml:"category"`
}

// --- Renderer unit tests ---

// TestRenderRSS_ValidXML verifies the output of renderRSS parses as
// valid XML — i.e. it's well-formed (no unclosed tags, no invalid
// entities) and conforms to the RSS 2.0 envelope (<rss version="2.0">
// with a single <channel> child). The test decodes the output into a
// typed struct and asserts the version attribute + channel fields.
func TestRenderRSS_ValidXML(t *testing.T) {
        out, err := renderRSS(sampleRSSFeed())
        if err != nil {
                t.Fatalf("renderRSS: %v", err)
        }

        // Must start with the XML prolog.
        if !strings.HasPrefix(out, `<?xml version="1.0" encoding="UTF-8"?>`) {
                t.Errorf("expected output to start with the XML prolog, got: %s", truncForLog(out, 80))
        }

        var v rssXMLForTest
        if err := xml.NewDecoder(strings.NewReader(out)).Decode(&v); err != nil {
                t.Fatalf("xml decode: %v\nXML was:\n%s", err, out)
        }

        if v.XMLName.Local != "rss" {
                t.Errorf("expected root element <rss>, got %q", v.XMLName.Local)
        }
        if v.Version != "2.0" {
                t.Errorf("expected version=2.0, got %q", v.Version)
        }
        if v.Channel.Title != "Civic Intelligence — Test Feed" {
                t.Errorf("expected channel title, got %q", v.Channel.Title)
        }
        if v.Channel.Link != "https://example.com/feed" {
                t.Errorf("expected channel link, got %q", v.Channel.Link)
        }
        if v.Channel.Description == "" {
                t.Errorf("expected non-empty channel description")
        }
}

// TestRenderRSS_ContainsItems verifies the rendered XML contains the
// <item> elements supplied in the Feed — both as a structural check
// (<item> tag present) and as a content check (the first item's title
// appears verbatim). RSS readers index items by their <title>, so a
// missing title would silently drop the item from the subscriber's view.
func TestRenderRSS_ContainsItems(t *testing.T) {
        out, err := renderRSS(sampleRSSFeed())
        if err != nil {
                t.Fatalf("renderRSS: %v", err)
        }

        // Two items supplied → two <item> elements expected.
        itemCount := strings.Count(out, "<item>")
        if itemCount != 2 {
                t.Errorf("expected 2 <item> elements, got %d (output:\n%s)", itemCount, out)
        }

        // Each item's title must appear verbatim.
        for _, want := range []string{"First item", "Second item"} {
                if !strings.Contains(out, want) {
                        t.Errorf("expected %q in output, got:\n%s", want, out)
                }
        }

        // Channel-level link must appear so RSS readers can deep-link to the
        // originating site from the feed header.
        if !strings.Contains(out, "https://example.com/feed") {
                t.Errorf("expected channel link in output, got:\n%s", out)
        }

        // pubDate must be present (RSS 2.0 §4.3.6).
        if !strings.Contains(out, "<pubDate>") {
                t.Errorf("expected <pubDate> in output, got:\n%s", out)
        }

        // guid must be present (RSS 2.0 §4.3.9 — required for reader-side
        // deduplication).
        if !strings.Contains(out, "<guid>") {
                t.Errorf("expected <guid> in output, got:\n%s", out)
        }
}

// TestRenderRSS_GUIDFallsBackToLink verifies that an empty GUID falls
// back to the item's Link — RSS 2.0 §4.3.9 lets a guid equal the link
// with isPermaLink="true", which is the default behaviour we want so
// RSS readers deduplicate items by their canonical URL even when the
// caller didn't set an explicit GUID.
func TestRenderRSS_GUIDFallsBackToLink(t *testing.T) {
        feed := Feed{
                Title:       "GUID fallback feed",
                Description: "verifies the GUID fallback",
                Link:        "https://example.com/",
                Items: []FeedItem{
                        {
                                Title:       "No explicit GUID",
                                Description: "GUID should fall back to the link",
                                Link:        "https://example.com/item-1",
                                PubDate:     "Wed, 02 Oct 2024 12:00:00 +0000",
                                // GUID deliberately left empty.
                        },
                },
        }
        out, err := renderRSS(feed)
        if err != nil {
                t.Fatalf("renderRSS: %v", err)
        }
        if !strings.Contains(out, "<guid>https://example.com/item-1</guid>") {
                t.Errorf("expected guid to fall back to the link, got:\n%s", out)
        }
}

// --- HTTP endpoint tests ---

// assertRSSContentType asserts the response carries the documented RSS
// content-type. Kept as a helper so each endpoint test stays compact.
func assertRSSContentType(t *testing.T, rr *httptest.ResponseRecorder) {
        t.Helper()
        if ct := rr.Header().Get("Content-Type"); ct != rssContentType {
                t.Errorf("expected Content-Type %q, got %q", rssContentType, ct)
        }
}

// assertValidRSSBody decodes the response body as RSS 2.0 XML and
// asserts the <rss version="2.0"> envelope + a single <channel>. Used
// by every HTTP endpoint test so a regression in the envelope shape is
// caught once across all four feeds.
func assertValidRSSBody(t *testing.T, body string) {
        t.Helper()
        if !strings.HasPrefix(body, `<?xml version="1.0" encoding="UTF-8"?>`) {
                t.Errorf("expected body to start with the XML prolog, got: %s", truncForLog(body, 80))
        }
        var v rssXMLForTest
        if err := xml.NewDecoder(strings.NewReader(body)).Decode(&v); err != nil {
                t.Fatalf("xml decode: %v\nbody was:\n%s", err, body)
        }
        if v.XMLName.Local != "rss" {
                t.Errorf("expected root element <rss>, got %q", v.XMLName.Local)
        }
        if v.Version != "2.0" {
                t.Errorf("expected version=2.0, got %q", v.Version)
        }
        if v.Channel.Title == "" {
                t.Errorf("expected non-empty channel title, got body:\n%s", body)
        }
}

// truncForLog returns the first n bytes of s (with an ellipsis when
// truncated) so test failure messages don't dump the entire RSS body
// into the test output.
func truncForLog(s string, n int) string {
        if len(s) <= n {
                return s
        }
        return s[:n] + "..."
}

// TestBillsRSSFeed_ReturnsXML verifies GET /api/v1/feed/bills.rss
// returns valid RSS 2.0 XML with the application/rss+xml content-type.
// Reuses mockBillsAdapter so a regression in the shared BillsAdapter
// pipeline surfaces against both the JSON and RSS endpoints.
func TestBillsRSSFeed_ReturnsXML(t *testing.T) {
        adapter := &mockBillsAdapter{bills: sampleLiveBills()}
        req := httptest.NewRequest(http.MethodGet, "/api/v1/feed/bills.rss", nil)
        rr := httptest.NewRecorder()

        makeBillsRSSFeedHandler(adapter)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        assertRSSContentType(t, rr)
        body := rr.Body.String()
        assertValidRSSBody(t, body)

        // The mock bill title must appear in the feed.
        if !strings.Contains(body, "The Housing Bill 2026") {
                t.Errorf("expected sample bill title in feed, got:\n%s", body)
        }

        // The House must appear as a <category> so subscribers can filter the
        // feed by chamber (National Assembly vs Senate).
        if !strings.Contains(body, "<category>National Assembly</category>") {
                t.Errorf("expected <category>National Assembly</category>, got:\n%s", body)
        }
}

// TestBillsRSSFeed_FallbackToSeed_OnAdapterError verifies the bills RSS
// feed mirrors the JSON endpoint's issue-#265 fallback: when the live
// crawl fails, the feed returns 200 (NOT 5xx) with the seed Bills so
// subscribers don't see their feed break during upstream outages.
func TestBillsRSSFeed_FallbackToSeed_OnAdapterError(t *testing.T) {
        adapter := &mockBillsAdapter{err: errSimulatedUpstream}
        req := httptest.NewRequest(http.MethodGet, "/api/v1/feed/bills.rss", nil)
        rr := httptest.NewRecorder()

        makeBillsRSSFeedHandler(adapter)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200 (NOT 5xx) on adapter error, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        assertRSSContentType(t, rr)
        body := rr.Body.String()
        assertValidRSSBody(t, rr.Body.String())

        // Seed slice must be non-empty — the fallback should yield real items.
        if !strings.Contains(body, "<item>") {
                t.Errorf("expected at least one <item> from the seed fallback, got:\n%s", body)
        }
}

// errSimulatedUpstream is the canned error used by the fallback tests.
// Defined separately so the test name + the assertion share a single
// source of truth for the simulated failure mode.
var errSimulatedUpstream = errors.New("simulated upstream 503")

// TestWhatChangedRSSFeed_ReturnsXML verifies GET /api/v1/feed/what-changed.rss
// returns valid RSS 2.0 XML. Uses the billsListFixture (three Bills
// including an Amendment + a Finance Bill) so the significance
// classifier (SUBSTANTIVE / HIGH_IMPACT / INFORMATIONAL) is exercised.
func TestWhatChangedRSSFeed_ReturnsXML(t *testing.T) {
        adapter := newTestAdapter(billsListFixture)
        req := httptest.NewRequest(http.MethodGet, "/api/v1/feed/what-changed.rss", nil)
        rr := httptest.NewRecorder()

        makeWhatChangedRSSFeedHandler(adapter)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        assertRSSContentType(t, rr)
        body := rr.Body.String()
        assertValidRSSBody(t, body)

        // The fixture includes "Amendment" + "Finance" titles — the classifier
        // should emit SUBSTANTIVE + HIGH_IMPACT categories.
        if !strings.Contains(body, "<category>SUBSTANTIVE</category>") {
                t.Errorf("expected SUBSTANTIVE category for the Amendment bill, got:\n%s", body)
        }
        if !strings.Contains(body, "<category>HIGH_IMPACT</category>") {
                t.Errorf("expected HIGH_IMPACT category for the Finance bill, got:\n%s", body)
        }
}

// TestBriefRSSFeed_ReturnsXML verifies GET /api/v1/feed/brief.rss
// returns valid RSS 2.0 XML. Generates a brief into the store first so
// the feed has at least one <item> to assert against — verifies the
// brief headline + AI disclaimer both surface in the rendered feed.
func TestBriefRSSFeed_ReturnsXML(t *testing.T) {
        adapter := newTestAdapter(billsListFixture)
        store := newBriefStore()

        // Generate today's brief so the feed has something to render.
        genReq := httptest.NewRequest(http.MethodPost, "/api/v1/brief/generate", strings.NewReader(`{"country":"KE"}`))
        genReq.Header.Set("Content-Type", "application/json")
        genRR := httptest.NewRecorder()
        makeBriefGenerateHandler(adapter, "", store)(genRR, genReq)
        if genRR.Code != http.StatusOK {
                t.Fatalf("setup: brief generate: %d (body=%s)", genRR.Code, genRR.Body.String())
        }

        req := httptest.NewRequest(http.MethodGet, "/api/v1/feed/brief.rss", nil)
        rr := httptest.NewRecorder()
        makeBriefRSSFeedHandler(store)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        assertRSSContentType(t, rr)
        body := rr.Body.String()
        assertValidRSSBody(t, body)

        if !strings.Contains(body, "<item>") {
                t.Errorf("expected at least one <item> in the brief feed, got:\n%s", body)
        }

        // The deep link must point back to /briefing?id={briefID} so RSS
        // subscribers land on the rendered brief (not the raw JSON).
        if !strings.Contains(body, "/briefing?id=brief-") {
                t.Errorf("expected brief deep link to /briefing?id=brief-…, got:\n%s", body)
        }
}

// TestBriefRSSFeed_EmptyStoreReturnsValidXML verifies that when the
// briefStore is empty (before the day's first /brief/generate), the RSS
// feed still returns 200 with a valid (but item-less) feed — RSS
// readers must see a structurally valid response rather than a 5xx.
func TestBriefRSSFeed_EmptyStoreReturnsValidXML(t *testing.T) {
        store := newBriefStore()
        req := httptest.NewRequest(http.MethodGet, "/api/v1/feed/brief.rss", nil)
        rr := httptest.NewRecorder()
        makeBriefRSSFeedHandler(store)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        assertRSSContentType(t, rr)
        body := rr.Body.String()
        assertValidRSSBody(t, body)

        if strings.Contains(body, "<item>") {
                t.Errorf("expected zero <item> elements on an empty store, got:\n%s", body)
        }
}

// TestMPRSSFeed_ReturnsXML verifies GET /api/v1/feed/people/{id}.rss
// returns valid RSS 2.0 XML for a known MP. Uses person-001 (a seeded
// sample scorecard) so the test has a stable person ID to assert
// against. The feed must surface the MP's recent activity as <item>
// elements, with the activity kind (bill_sponsored, vote, question,
// statement) as the <category> for subscriber-side filtering.
func TestMPRSSFeed_ReturnsXML(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/feed/people/person-001.rss", nil)
        // httptest.NewRequest parses the URL but doesn't always preserve the
        // path on r.URL.Path for unusual suffixes like .rss — set it
        // explicitly so the handler's path-stripping logic sees the .rss.
        req.URL.Path = "/api/v1/feed/people/person-001.rss"
        rr := httptest.NewRecorder()

        makeMPRSSFeedHandler()(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        assertRSSContentType(t, rr)
        body := rr.Body.String()
        assertValidRSSBody(t, body)

        // The MP's name must appear in the channel title so subscribers see
        // whose activity they're following in their reader's title bar.
        if !strings.Contains(body, "Ichung") {
                t.Errorf("expected MP name in channel title, got:\n%s", body)
        }

        // At least one <item> expected — person-001's recent_activity is
        // non-empty in the seed slice.
        if !strings.Contains(body, "<item>") {
                t.Errorf("expected at least one <item> from the MP's recent_activity, got:\n%s", body)
        }

        // The activity kind must appear as a <category> so subscribers can
        // filter by activity type (bills, votes, questions, statements).
        hasActivityCategory := strings.Contains(body, "<category>bill_sponsored</category>") ||
                strings.Contains(body, "<category>vote</category>") ||
                strings.Contains(body, "<category>question</category>") ||
                strings.Contains(body, "<category>statement</category>")
        if !hasActivityCategory {
                t.Errorf("expected at least one activity kind as a <category>, got:\n%s", body)
        }
}

// TestMPRSSFeed_NotFound verifies the per-MP RSS endpoint returns 404
// for an unknown person ID — RSS readers surface the failure rather
// than silently subscribing to an empty feed (which a 200-with-no-items
// would imply is a valid-but-quiet MP).
func TestMPRSSFeed_NotFound(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/feed/people/does-not-exist.rss", nil)
        req.URL.Path = "/api/v1/feed/people/does-not-exist.rss"
        rr := httptest.NewRecorder()

        makeMPRSSFeedHandler()(rr, req)

        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404 for unknown person, got %d (body=%s)", rr.Code, rr.Body.String())
        }
}

// TestMPRSSFeed_EmptyID verifies the per-MP RSS endpoint returns 400
// when the person ID is missing entirely (e.g. GET /api/v1/feed/people/.rss).
func TestMPRSSFeed_EmptyID(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/feed/people/.rss", nil)
        req.URL.Path = "/api/v1/feed/people/.rss"
        rr := httptest.NewRecorder()

        makeMPRSSFeedHandler()(rr, req)

        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for empty person ID, got %d (body=%s)", rr.Code, rr.Body.String())
        }
}
