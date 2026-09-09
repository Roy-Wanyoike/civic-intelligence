package president

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- fixtures ----------------------------------------------------------------

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "fixture %s not found", name)
	return string(data)
}

// --- ParseAssentList ---------------------------------------------------------

// TestParseAssentList_RealRSSFixture tests the RSS parser against the actual
// president.go.ke search-as-RSS feed captured on 2026-09-09.
func TestParseAssentList_RealRSSFixture(t *testing.T) {
	xml := loadFixture(t, "assent_list.rss")
	candidates := ParseAssentList(xml)

	require.NotEmpty(t, candidates, "should discover announcements from the real RSS feed")
	assert.GreaterOrEqual(t, len(candidates), 5, "the fixture has at least 5 announcements")

	first := candidates[0]
	assert.NotEmpty(t, first.SourceURL)
	assert.Contains(t, first.SourceURL, "president.go.ke")
	assert.NotEmpty(t, first.BillName, "should carry announcement headline as hint")
	assert.NotEmpty(t, first.SourceID)
	assert.False(t, first.AssentDate.IsZero(), "assent date should be parsed from RSS pubDate")

	// SourceID must be stable across runs and derived from the URL slug.
	for _, c := range candidates {
		assert.True(t, strings.HasPrefix(c.SourceID, "ke-assent-"),
			"SourceID should start with ke-assent- prefix; got %q", c.SourceID)
	}
}

// TestParseAssentList_SyntheticFixture tests against the synthetic RSS feed
// that includes the Sep 8 2026 four-bill scenario described in the task brief.
func TestParseAssentList_SyntheticFixture(t *testing.T) {
	xml := loadFixture(t, "assent_list_synthetic.rss")
	candidates := ParseAssentList(xml)

	require.NotEmpty(t, candidates)
	assert.GreaterOrEqual(t, len(candidates), 5)

	// Find the Sep 8 2026 announcement.
	var sep8 *AssentCandidate
	for i := range candidates {
		if strings.Contains(candidates[i].SourceURL, "four-bills-at-state-house") {
			sep8 = &candidates[i]
			break
		}
	}
	require.NotNil(t, sep8, "should find the Sep 8 2026 four-bills announcement")

	expected, _ := time.Parse(time.RFC1123Z, "Tue, 08 Sep 2026 14:30:00 +0000")
	assert.Equal(t, expected, sep8.AssentDate)
	assert.Equal(t, "PRESIDENT RUTO ASSENTS TO FOUR BILLS AT STATE HOUSE", sep8.BillName)
	assert.Equal(t, "https://www.president.go.ke/president-ruto-assents-to-four-bills-at-state-house/", sep8.SourceURL)
	assert.Equal(t, "ke-assent-president-ruto-assents-to-four-bills-at-state-house", sep8.SourceID)
}

// TestParseAssentList_HTMLFallback verifies that ParseAssentList accepts HTML
// (the search results page) as a fallback when RSS is unavailable. The HTML
// path is used when /?s=assent is fetched instead of /search/assent/feed/rss2/.
func TestParseAssentList_HTMLFallback(t *testing.T) {
	html := `
	<html><body>
		<a href="https://www.president.go.ke/president-ruto-assents-to-parliamentary-bills/">Parliamentary Bills</a>
		<a href="https://www.president.go.ke/president-ruto-assents-to-parliamentary-bills/">Parliamentary Bills (dup)</a>
		<a href="https://www.president.go.ke/some-other-press-release/">Other Release</a>
	</body></html>`
	candidates := ParseAssentList(html)
	require.Len(t, candidates, 1, "should dedupe and only keep assent URLs")
	assert.Contains(t, candidates[0].SourceURL, "parliamentary-bills")
	assert.Equal(t, "Parliamentary Bills", candidates[0].BillName)
}

func TestParseAssentList_Empty(t *testing.T) {
	assert.Empty(t, ParseAssentList(""))
	assert.Empty(t, ParseAssentList("<html><body>no items</body></html>"))
}

// --- ParseAssentDetail -------------------------------------------------------

// TestParseAssentDetail_RealFixture parses the real president.go.ke
// "PRESIDENT RUTO ASSENTS TO PARLIAMENTARY BILLS" page captured 2024-12-04.
// The page mentions 3 bills: Division of Revenue (Amendment) Bill 2024,
// National Rating Bill 2022, Water (Amendment) Bill 2024.
func TestParseAssentDetail_RealFixture(t *testing.T) {
	html := loadFixture(t, "assent_detail.html")
	url := "https://www.president.go.ke/president-ruto-assents-to-parliamentary-bills/"
	candidates := ParseAssentDetail(html, url)

	require.Len(t, candidates, 3, "should extract 3 bills from the real page")

	// Verify each candidate has the expected structure.
	for _, c := range candidates {
		assert.NotEmpty(t, c.BillName)
		assert.Contains(t, c.BillName, "Bill")
		assert.Equal(t, url, c.SourceURL)
		assert.False(t, c.AssentDate.IsZero(), "assent date should be parsed from post-meta")
		assert.True(t, strings.HasPrefix(c.SourceID, "ke-assent-"))
	}

	// Verify the assent date — the post-meta span says "December 4, 2024".
	expectedDate, _ := time.Parse("January 2, 2006", "December 4, 2024")
	assert.Equal(t, expectedDate, candidates[0].AssentDate)

	// Collect bill names for assertion.
	names := make([]string, len(candidates))
	for i, c := range candidates {
		names[i] = c.BillName
	}
	t.Logf("Extracted bills: %v", names)

	// Each known bill name must be present (order-independent — the parser
	// returns first-seen order, which we don't constrain beyond that).
	//
	// NOTE: the year separator is preserved from the source text. "National
	// Rating Bill 2022" has no comma; the other two use the comma form.
	assert.Contains(t, names, "Division of Revenue (Amendment) Bill, 2024")
	assert.Contains(t, names, "National Rating Bill 2022")
	assert.Contains(t, names, "Water (Amendment) Bill, 2024")
}

// TestParseAssentDetail_IncomeTaxBills parses the real president.go.ke
// "PRESIDENT RUTO ASSENTS TO INCOME TAX, SEZ AND TECHNOPOLIS BILLS" page
// captured 2026-05-11. This page has bills WITHOUT years in the headline
// — the parser must still capture them.
func TestParseAssentDetail_IncomeTaxBills(t *testing.T) {
	html := loadFixture(t, "assent_income_tax.html")
	url := "https://www.president.go.ke/president-ruto-assents-to-income-tax-sez-and-technopolis-bills/"
	candidates := ParseAssentDetail(html, url)

	require.Len(t, candidates, 3, "should extract 3 bills (Income Tax, SEZ, Technopolis)")

	expectedDate, _ := time.Parse("January 2, 2006", "May 11, 2026")
	assert.Equal(t, expectedDate, candidates[0].AssentDate)

	names := make([]string, len(candidates))
	for i, c := range candidates {
		names[i] = c.BillName
	}
	t.Logf("Extracted bills: %v", names)

	assert.Contains(t, names, "Income Tax Bill")
	assert.Contains(t, names, "Special Economic Zones (Amendment) Bill")
	assert.Contains(t, names, "Technopolis Bill")
}

// TestParseAssentDetail_FourBills parses a synthetic page that mirrors the
// Sep 8 2026 State House ceremony described in the task brief (4 Bills).
// This is the canonical use-case the platform was built to detect.
func TestParseAssentDetail_FourBills(t *testing.T) {
	html := loadFixture(t, "assent_four_bills_synthetic.html")
	url := "https://www.president.go.ke/president-ruto-assents-to-four-bills-at-state-house/"
	candidates := ParseAssentDetail(html, url)

	require.Len(t, candidates, 4, "the Sep 8 2026 ceremony covered 4 bills")

	expectedDate, _ := time.Parse("January 2, 2006", "September 8, 2026")
	assert.Equal(t, expectedDate, candidates[0].AssentDate)

	names := make([]string, len(candidates))
	for i, c := range candidates {
		names[i] = c.BillName
	}
	t.Logf("Extracted bills: %v", names)

	// The four bills named in the task brief.
	assert.Contains(t, names, "Kenya National Council for Population and Development Bill, 2026")
	assert.Contains(t, names, "Public Health (Amendment) Bill, 2026")
	assert.Contains(t, names, "Cooperatives (Amendment) Bill, 2026")
	assert.Contains(t, names, "Climate Change (Amendment) Bill, 2026")

	// All four candidates should share the same SourceURL.
	for _, c := range candidates {
		assert.Equal(t, url, c.SourceURL)
	}

	// Each candidate should have a unique SourceID (because the bill name
	// differs).
	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		assert.False(t, seen[c.SourceID], "duplicate SourceID: %s", c.SourceID)
		seen[c.SourceID] = true
	}
}

// TestParseAssentDetail_OgDescriptionExtraction verifies the regex picks up
// bill names from the og:description meta tag when the body uses the verb
// "assented to" (which my introRe catches).
func TestParseAssentDetail_OgDescriptionExtraction(t *testing.T) {
	html := `<html><head>
		<meta property="og:title" content="PRESIDENT RUTO ASSENTS TO THE SUGAR BILL" />
		<meta property="og:description" content="President Ruto assented to the Sugar Bill, 2026 at State House Nairobi." />
		</head><body>
		<div class="brxe-post-meta post-meta"><span class="item">July 15, 2026</span></div>
		<div class="brxe-post-content">
			<p class="wp-block-paragraph">President Ruto assented to the Sugar Bill, 2026 at State House Nairobi.</p>
		</div></body></html>`
	candidates := ParseAssentDetail(html, "https://www.president.go.ke/sugar-bill-assent/")
	require.Len(t, candidates, 1)
	assert.Equal(t, "Sugar Bill, 2026", candidates[0].BillName)

	expectedDate, _ := time.Parse("January 2, 2006", "July 15, 2026")
	assert.Equal(t, expectedDate, candidates[0].AssentDate)
}

// TestParseAssentDetail_FallbackToTitle verifies the fallback path: when no
// bill names can be extracted from the body or og:description, the parser
// falls back to using the announcement's headline (cleaned) as a single
// candidate. This handles short announcements that only mention the bill in
// the title without using the "assented to" verb.
func TestParseAssentDetail_FallbackToTitle(t *testing.T) {
	html := `<html><head>
		<meta property="og:title" content="PRESIDENT RUTO ADDRESSES THE NATION" />
		<meta property="og:description" content="President Ruto addressed the nation from State House." />
		</head><body>
		<div class="brxe-post-meta post-meta"><span class="item">July 15, 2026</span></div>
		<div class="brxe-post-content">
			<p class="wp-block-paragraph">The President addressed the nation from State House.</p>
		</div></body></html>`
	candidates := ParseAssentDetail(html, "https://www.president.go.ke/address/")
	// No "Bill" mention anywhere — the parser falls back to the og:title.
	require.Len(t, candidates, 1)
	assert.Equal(t, "PRESIDENT RUTO ADDRESSES THE NATION", candidates[0].BillName)
}

func TestParseAssentDetail_Empty(t *testing.T) {
	assert.Empty(t, ParseAssentDetail("", "https://www.president.go.ke/x/"))
	assert.Empty(t, ParseAssentDetail("<html><body>no bills</body></html>", "https://www.president.go.ke/x/"))
}

// --- Stage mapping -----------------------------------------------------------

// TestAssentStageMapping verifies the core invariant of this adapter: every
// record produced by ParseAssentDetail is tagged with the PRESIDENTIAL_ASSENT
// stage code, AND that code matches the Kenya internal stage table.
//
// This is the cross-boundary test: it imports the Kenya internal package
// (which is allowed here because adapter tests live inside adapters/kenya)
// and asserts that the constant duplicated in president.StagePresidentialAssent
// is identical to internal.StagePresidentialAssent. If either side changes,
// this test fails loudly.
func TestAssentStageMapping(t *testing.T) {
	// Constant duplication: the president adapter must NOT import the kenya
	// internal package, so it carries a local copy of the stage code string.
	// This assertion guarantees the two strings stay in sync.
	assert.Equal(t,
		string(internal.StagePresidentialAssent),
		StagePresidentialAssent,
		"president.StagePresidentialAssent must match internal.StagePresidentialAssent",
	)

	// An AssentCandidate, projected to an ExtractedRecord, carries the
	// PRESIDENTIAL_ASSENT stage code.
	candidate := AssentCandidate{
		BillName:   "The Sugar Bill, 2026",
		AssentDate: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
		SourceURL:  "https://www.president.go.ke/sugar-bill-assent/",
		SourceID:   "ke-assent-sugar-bill-assent",
	}
	record := candidate.ToExtractedRecord()

	assert.Equal(t, "PRESIDENTIAL_ASSENT", record.Stage)
	assert.Equal(t, StagePresidentialAssent, record.Stage)
	assert.Equal(t, "assent", record.Kind)
	assert.Equal(t, candidate.BillName, record.Title)
	assert.Equal(t, candidate.AssentDate, record.PublishedAt)
	assert.Equal(t, candidate.SourceURL, record.SourceURL)
	assert.Greater(t, record.Confidence, 0.9, "official source should carry high confidence")
	assert.False(t, record.RetrievedAt.IsZero(), "RetrievedAt should be set")

	// The stage must exist in the Kenya stage table — i.e., the adapter is
	// not inventing a stage code that the rest of the platform doesn't know.
	stage := internal.FindStage(internal.StageCode(record.Stage))
	require.NotNil(t, stage, "stage %s must exist in the Kenya stage table", record.Stage)
	assert.Equal(t, "Presidential Assent", stage.Name)
	assert.Equal(t, 8, stage.Order)
}

// TestAdapter_GetStages verifies the adapter exposes its stage via the
// contracts.LegislativeSourceAdapter interface.
func TestAdapter_GetStages(t *testing.T) {
	a := NewAdapter(nil, "")
	stages, err := a.GetStages(context.Background())
	require.NoError(t, err)
	require.Len(t, stages, 1)

	s := stages[0]
	assert.Equal(t, "PRESIDENTIAL_ASSENT", s.Code)
	assert.Equal(t, "Presidential Assent", s.Name)
	assert.Equal(t, "KE", string(s.Country))
	assert.Equal(t, 8, s.Order)
	assert.True(t, s.RequiresEvidence)
	assert.Contains(t, s.AllowedNext, "COMMENCEMENT")
}

// --- Adapter structure ------------------------------------------------------

func TestAdapter_CountryCode(t *testing.T) {
	a := NewAdapter(nil, "")
	assert.Equal(t, "KE", a.CountryCode())
}

func TestAdapter_Supports(t *testing.T) {
	a := NewAdapter(nil, "")
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.president.go.ke/president-ruto-assents-to-parliamentary-bills/", true},
		{"https://president.go.ke/speeches_remarks/during-the-assent/", true},
		{"https://new.kenyalaw.org/bills/", false},
		{"https://parliament.go.ke/bills", false},
		{"https://example.com", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.expected, a.Supports(tt.url))
		})
	}
}

func TestAdapter_GetOfficialSources(t *testing.T) {
	a := NewAdapter(nil, "")
	sources := a.GetOfficialSources()
	require.NotEmpty(t, sources)

	s := sources[0]
	assert.Equal(t, "KE", s.Country)
	assert.Equal(t, "primary", s.Authority)
	assert.Contains(t, s.URL, "president.go.ke")
	assert.Contains(t, s.DocumentTypes, "assent_announcement")
	assert.Equal(t, "kenya.president", s.Adapter)
	assert.Greater(t, s.CrawlFrequency, time.Duration(0))
}

// TestAdapter_Defaults verifies NewAdapter fills in sane defaults when called
// with nil/empty arguments.
func TestAdapter_Defaults(t *testing.T) {
	a := NewAdapter(nil, "")
	require.NotNil(t, a.client, "client should default to a non-nil HTTP client")
	assert.NotEmpty(t, a.userAgent)
	assert.Contains(t, a.userAgent, "CivicIntelligence")
	assert.Equal(t, "https://www.president.go.ke", a.baseURL)
	assert.Equal(t, "https://www.president.go.ke/search/assent/feed/rss2/", a.feedURL)
}

// --- Compile-time interface compliance --------------------------------------

// TestAdapter_ImplementsLegislativeSourceAdapter is a runtime check that
// complements the compile-time var _ contracts.LegislativeSourceAdapter
// assertion in president.go. It guards against accidental interface drift
// (e.g., adding a method parameter) that the compile-time check would also
// catch, but in a way that emits a friendly test failure.
func TestAdapter_ImplementsLegislativeSourceAdapter(t *testing.T) {
	var _ contracts.LegislativeSourceAdapter = (*Adapter)(nil)
	var _ contracts.LegislativeSourceAdapter = NewAdapter(nil, "")
}

// --- Live HTTP test ---------------------------------------------------------

// mockAssentClient is an in-memory HTTPClient that returns canned responses
// based on the URL pattern. Used by TestDiscoverAssents_Pipeline to exercise
// the full discover → fetch → parse pipeline without hitting president.go.ke.
type mockAssentClient struct {
	rssFixture    string
	detailFixture string
	calls         int
}

func (m *mockAssentClient) Do(req *http.Request) (*http.Response, error) {
	m.calls++
	url := req.URL.String()
	var body, contentType string
	switch {
	case strings.Contains(url, "/search/assent/feed/rss2/"):
		body = m.rssFixture
		contentType = "application/rss+xml"
	case strings.Contains(url, "president-ruto-assents-to-four-bills-at-state-house"):
		body = m.detailFixture
		contentType = "text/html"
	default:
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       http.NoBody,
			Header:     make(http.Header),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{contentType}},
	}, nil
}

// TestDiscoverAssents_Pipeline verifies the full discover → fetch → parse
// pipeline using a mock HTTPClient. This is the closest we can get to a live
// test without hitting president.go.ke from CI.
//
// Pipeline exercised:
//  1. DiscoverAssents → fetches RSS → ParseAssentList → []AssentCandidate
//  2. FetchAssent (for the Sep 8 2026 four-bills candidate)
//  3. ParseAssent → ParseAssentDetail → 4 per-bill AssentCandidates
//
// Then via the contracts.LegislativeSourceAdapter interface:
//  4. Discover → []SourceItem
//  5. Fetch → *RawDocument
//  6. Parse → []ExtractedRecord (each carrying PRESIDENTIAL_ASSENT)
func TestDiscoverAssents_Pipeline(t *testing.T) {
	mock := &mockAssentClient{
		rssFixture:    loadFixture(t, "assent_list_synthetic.rss"),
		detailFixture: loadFixture(t, "assent_four_bills_synthetic.html"),
	}
	a := NewAdapter(mock, "CivicIntelligence/0.1 (test)")

	// 1. Discover.
	candidates, err := a.DiscoverAssents(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, candidates)
	assert.Greater(t, mock.calls, 0, "DiscoverAssents should have called the HTTP client")

	// 2. Find the Sep 8 2026 four-bills announcement.
	var fourBillsURL string
	for _, c := range candidates {
		if strings.Contains(c.SourceURL, "four-bills-at-state-house") {
			fourBillsURL = c.SourceURL
			break
		}
	}
	require.NotEmpty(t, fourBillsURL)

	// 3. Fetch + Parse the announcement.
	html, err := a.FetchAssent(context.Background(), fourBillsURL)
	require.NoError(t, err)
	require.Contains(t, html, "PRESIDENT RUTO ASSENTS TO FOUR BILLS AT STATE HOUSE")

	bills, err := a.ParseAssent(html, fourBillsURL)
	require.NoError(t, err)
	require.Len(t, bills, 4, "should extract 4 bills from the fetched page")

	// 4. Verify the full pipeline via the contracts.LegislativeSourceAdapter
	//    methods (Discover → Fetch → Parse). This exercises the projection
	//    from AssentCandidate → SourceItem → RawDocument → ExtractedRecord.
	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, items)

	// Find the four-bills item, fetch it, parse it.
	var item contracts.SourceItem
	for _, it := range items {
		if strings.Contains(it.URL, "four-bills-at-state-house") {
			item = it
			break
		}
	}
	require.NotEmpty(t, item.URL)

	doc, err := a.Fetch(context.Background(), item)
	require.NoError(t, err)
	assert.Equal(t, item.URL, doc.URL)
	assert.False(t, doc.RetrievedAt.IsZero(), "RetrievedAt must be set on RawDocument")

	records, err := a.Parse(context.Background(), *doc)
	require.NoError(t, err)
	require.Len(t, records, 4)

	for _, r := range records {
		assert.Equal(t, "PRESIDENTIAL_ASSENT", r.Stage)
		assert.Equal(t, "assent", r.Kind)
		assert.Equal(t, item.URL, r.SourceURL)
		assert.False(t, r.PublishedAt.IsZero())
		assert.Greater(t, r.Confidence, 0.9)
	}
}

// --- Helpers ----------------------------------------------------------------

// TestSourceIDFromURL verifies the SourceID is derived from the URL slug
// (the part of the URL after the last "/").
func TestSourceIDFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{
			"https://www.president.go.ke/president-ruto-assents-to-parliamentary-bills/",
			"ke-assent-president-ruto-assents-to-parliamentary-bills",
		},
		{
			"https://www.president.go.ke/president-ruto-assents-to-four-bills-at-state-house/",
			"ke-assent-president-ruto-assents-to-four-bills-at-state-house",
		},
		{
			"https://president.go.ke/speeches_remarks/during-the-assent-to-the-sovereign-wealth-fund-bill-2026/",
			"ke-assent-during-the-assent-to-the-sovereign-wealth-fund-bill-2026",
		},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.expected, sourceIDFromURL(tt.url))
		})
	}
}

// TestDecodeHTMLEntities verifies WordPress's smart-quote / em-dash entities
// are decoded before parsing (otherwise bill names containing apostrophes
// would be mangled).
func TestDecodeHTMLEntities(t *testing.T) {
	in := "Kenya&#8217;s Constitution &#8211; Article 115 &#8230;"
	out := decodeHTMLEntities(in)
	assert.Equal(t, "Kenya's Constitution - Article 115 ...", out)
}
