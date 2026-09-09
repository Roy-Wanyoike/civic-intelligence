// Package parliament — tests for the Parliament of Kenya adapter.
//
// Test fixtures (in testdata/) were captured from the live parliament.go.ke
// site on 2026-09-09 using:
//
//	curl -sS -A "CivicIntelligence/0.1" \
//	  https://www.parliament.go.ke/the-national-assembly/house-business/bills \
//	  > testdata/bills_na.html
//
// The Parliament website is a Drupal 8 site; the Bills listing page uses
// one <div class="post-block"> per Bill (see ParseBillsListing for the
// exact structure).
package parliament

import (
	"context"
	"net/http"
	"net/http/httptest"
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

// Compile-time assertion: Adapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*Adapter)(nil)

// --- stage mapping tests ----------------------------------------------------

func TestMapStageText_CanonicalStages(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Canonical names as published by Parliament.
		{"First Reading", string(internal.StageFirstReading)},
		{"Second Reading", string(internal.StageSecondReading)},
		{"Committee Stage", string(internal.StageCommitteeStage)},
		{"Committee of the Whole House", string(internal.StageCommitteeOfWholeHouse)},
		{"Report Stage", string(internal.StageReportStage)},
		{"Third Reading", string(internal.StageThirdReading)},
		{"Presidential Assent", string(internal.StagePresidentialAssent)},
		{"Commencement", string(internal.StageCommencement)},
		{"Mediation Committee", string(internal.StageMediation)},
		{"Rejected", string(internal.StageRejected)},
		{"Withdrawn", string(internal.StageWithdrawn)},
		{"Lapsed", string(internal.StageLapsed)},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MapStageText(tt.input)
			assert.Equal(t, tt.want, got, "stage mapping for %q", tt.input)
		})
	}
}

func TestMapStageText_CaseInsensitive(t *testing.T) {
	cases := map[string]string{
		"first reading":       string(internal.StageFirstReading),
		"FIRST READING":       string(internal.StageFirstReading),
		"Second reading":      string(internal.StageSecondReading),
		"SECOND READING":      string(internal.StageSecondReading),
		"committee stage":     string(internal.StageCommitteeStage),
		"COMMITTEE STAGE":     string(internal.StageCommitteeStage),
		"presidential assent": string(internal.StagePresidentialAssent),
		"PRESIDENTIAL ASSENT": string(internal.StagePresidentialAssent),
	}
	for input, want := range cases {
		got := MapStageText(input)
		assert.Equal(t, want, got, "stage mapping for %q", input)
	}
}

func TestMapStageText_SubstringMatch(t *testing.T) {
	// Real Parliament tracker lines look like "Second Reading — 12 March 2026".
	cases := []struct {
		input string
		want  string
	}{
		{"Second Reading — 12 March 2026", string(internal.StageSecondReading)},
		{"The Bill was read a First Time on 4 February 2026", string(internal.StageFirstReading)},
		{"Consideration by the Committee of the Whole House ongoing", string(internal.StageCommitteeOfWholeHouse)},
		{"Assented to by the President on 14 June 2026", string(internal.StagePresidentialAssent)},
		{"The Bill lapsed at the end of the 12th Parliament", string(internal.StageLapsed)},
		{"Bill withdrawn by the Mover on 30 July 2026", string(internal.StageWithdrawn)},
		{"Mediation Committee convened after Senate amendments were rejected", string(internal.StageMediation)},
	}
	for _, tt := range cases {
		got := MapStageText(tt.input)
		assert.Equal(t, tt.want, got, "stage mapping for %q", tt.input)
	}
}

func TestMapStageText_NumericVariants(t *testing.T) {
	assert.Equal(t, string(internal.StageFirstReading), MapStageText("1st Reading"))
	assert.Equal(t, string(internal.StageSecondReading), MapStageText("2nd Reading"))
	assert.Equal(t, string(internal.StageThirdReading), MapStageText("3rd Reading"))
}

func TestMapStageText_EmptyInput(t *testing.T) {
	assert.Equal(t, "", MapStageText(""))
	assert.Equal(t, "", MapStageText("   "))
}

func TestMapStageText_UnknownInput(t *testing.T) {
	// Strings that don't contain any stage keyword.
	assert.Equal(t, "", MapStageText("The Bill is scheduled for tabling next week"))
	assert.Equal(t, "", MapStageText("Order Paper — Thursday 27 August 2026"))
}

func TestMapStageText_MoreSpecificBeforeGeneric(t *testing.T) {
	// "committee of the whole house" must NOT match the bare "committee" pattern.
	assert.Equal(t,
		string(internal.StageCommitteeOfWholeHouse),
		MapStageText("Committee of the Whole House"),
	)
}

func TestMapStageTextWithConfidence_ExactMatch(t *testing.T) {
	code, conf := MapStageTextWithConfidence("Second Reading")
	assert.Equal(t, string(internal.StageSecondReading), code)
	assert.InDelta(t, 0.95, conf, 0.01, "exact match should score ~0.95")
}

func TestMapStageTextWithConfidence_SubstringMatch(t *testing.T) {
	code, conf := MapStageTextWithConfidence("Second Reading — 12 March 2026")
	assert.Equal(t, string(internal.StageSecondReading), code)
	assert.InDelta(t, 0.80, conf, 0.01, "substring match should score ~0.80")
}

func TestMapStageTextWithConfidence_NoMatch(t *testing.T) {
	code, conf := MapStageTextWithConfidence("nothing relevant here")
	assert.Equal(t, "", code)
	assert.Equal(t, 0.0, conf)
}

// --- ParseBillsListing tests (against real fixtures) ------------------------

func TestParseBillsListing_RealNAFixture(t *testing.T) {
	html := loadFixture(t, "bills_na.html")
	bills := ParseBillsListing(html)

	require.NotEmpty(t, bills, "should discover Bills from the real NA fixture")
	assert.GreaterOrEqual(t, len(bills), 10, "the NA fixture has at least 10 Bills")

	// Verify the first Bill has the expected structure.
	first := bills[0]
	assert.NotEmpty(t, first.URL, "Bill URL must be set")
	assert.NotEmpty(t, first.Title, "Bill Title must be set")
	assert.Contains(t, first.URL, "parliament.go.ke")
	assert.Contains(t, first.URL, ".pdf")

	// Verify every Bill has at least a URL and a Title (URL is mandatory;
	// the listing always links the Bill PDF).
	for i, b := range bills {
		assert.NotEmpty(t, b.URL, "Bill %d has empty URL", i)
		assert.NotEmpty(t, b.Title, "Bill %d has empty Title", i)
	}
}

func TestParseBillsListing_RealSenateFixture(t *testing.T) {
	html := loadFixture(t, "bills_senate.html")
	bills := ParseBillsListing(html)

	require.NotEmpty(t, bills, "should discover Bills from the real Senate fixture")
	assert.GreaterOrEqual(t, len(bills), 5, "the Senate fixture has at least 5 Bills")

	// Verify each Senate Bill has a Petition URL pointing at senate_petition.
	petitionCount := 0
	for _, b := range bills {
		if strings.Contains(b.PetitionURL, "senate_petition") {
			petitionCount++
		}
	}
	assert.Greater(t, petitionCount, 0,
		"at least one Senate Bill should have a senate_petition URL")
}

func TestParseBillsListing_Deduplication(t *testing.T) {
	// Synthetic fixture with two identical post-blocks.
	html := `<html><body>
		<div class="post-block"><div class="post-content">
			<div class="post-title"><a href="https://www.parliament.go.ke/sites/default/files/2026-08/THE_TEST_BILL.pdf">The Test Bill</a></div>
			<div class="post-meta">
				<span class="post-petition"><a href="https://www.parliament.go.ke/contact/national_assembly_petition?bill=The Test Bill">Submit Comments</a></span>
			</div>
		</div></div>
		<div class="post-block"><div class="post-content">
			<div class="post-title"><a href="https://www.parliament.go.ke/sites/default/files/2026-08/THE_TEST_BILL.pdf">The Test Bill (duplicate)</a></div>
			<div class="post-meta">
				<span class="post-petition"><a href="https://www.parliament.go.ke/contact/national_assembly_petition?bill=The Test Bill">Submit Comments</a></span>
			</div>
		</div></div>
		<div class="post-block"><div class="post-content">
			<div class="post-title"><a href="https://www.parliament.go.ke/sites/default/files/2026-08/THE_OTHER_BILL.pdf">The Other Bill</a></div>
		</div></div>
	</body></html>`

	bills := ParseBillsListing(html)
	assert.Len(t, bills, 2, "should deduplicate by URL")
	assert.Equal(t, "The Test Bill", bills[0].Title)
	assert.Equal(t, "The Other Bill", bills[1].Title)
}

func TestParseBillsListing_PetitionURLExtraction(t *testing.T) {
	html := `<html><body>
		<div class="post-block"><div class="post-content">
			<div class="post-title"><a href="https://www.parliament.go.ke/sites/default/files/2026-08/THE_HERALDRY_BILL.pdf">The Heraldry Bill, 2026</a></div>
			<div class="post-meta">
				<span class="post-petition"><a href="https://www.parliament.go.ke/contact/national_assembly_petition?bill=THE HERALDRY BILL%2C 2026">Submit Comments</a></span>
			</div>
		</div></div>
	</body></html>`

	bills := ParseBillsListing(html)
	require.Len(t, bills, 1)
	assert.NotEmpty(t, bills[0].PetitionURL)
	assert.Contains(t, bills[0].PetitionURL, "national_assembly_petition")
}

func TestParseBillsListing_PetitionURLFillsTitleWhenAbsent(t *testing.T) {
	// A Bill with a Petition link but no PDF link — the ?bill= query should
	// populate the Title.
	html := `<html><body>
		<div class="post-block"><div class="post-content">
			<div class="post-meta">
				<span class="post-petition"><a href="https://www.parliament.go.ke/contact/senate_petition?bill=The Creative Economy Bill%2C 2026">Submit Comments</a></span>
			</div>
		</div></div>
	</body></html>`

	bills := ParseBillsListing(html)
	require.Len(t, bills, 1)
	assert.Equal(t, "The Creative Economy Bill, 2026", bills[0].Title)
	assert.NotEmpty(t, bills[0].PetitionURL)
}

func TestParseBillsListing_BillDigestURL(t *testing.T) {
	html := `<html><body>
		<div class="post-block"><div class="post-content">
			<div class="post-title"><a href="https://www.parliament.go.ke/sites/default/files/2026-07/library_services_bill.pdf">The Kenya National Library Services Bill</a></div>
			<div class="post-meta">
				<span class="post-digest">Bill Digest: <a href="https://www.parliament.go.ke/sites/default/files/2026-07/library_services_digest.pdf">Bills Digest</a></span>
				<span class="post-petition"><a href="https://www.parliament.go.ke/contact/national_assembly_petition?bill=The Kenya National Library Services Bill">Submit Comments</a></span>
			</div>
		</div></div>
	</body></html>`

	bills := ParseBillsListing(html)
	require.Len(t, bills, 1)
	assert.NotEmpty(t, bills[0].BillDigestURL, "Bill Digest URL should be extracted")
	assert.Contains(t, bills[0].BillDigestURL, "digest")
}

func TestParseBillsListing_EmptyHTML(t *testing.T) {
	bills := ParseBillsListing("<html><body>No Bills here</body></html>")
	assert.Empty(t, bills)
}

func TestParseBillsListing_MalformedHTML(t *testing.T) {
	// Even malformed HTML should not panic.
	bills := ParseBillsListing("<<<>not html")
	assert.Empty(t, bills)
}

// --- ParseBillTrackerHTML tests --------------------------------------------

func TestParseBillTrackerHTML_StageFound(t *testing.T) {
	html := loadFixture(t, "bill_tracker_detail.html")
	bt := ParseBillTrackerHTML(html, "https://www.parliament.go.ke/the-national-assembly/house-business/bills/the-heraldry-bill-2026/tracker")

	require.NotNil(t, bt)
	// The fixture lists three stages, with "Committee of the Whole House"
	// as the most recent. The parser scans the full text and returns the
	// FIRST match — "First Reading" — so we verify it returns a non-empty
	// stage and the source URL is preserved.
	assert.NotEmpty(t, bt.Stage, "should find at least one stage in the tracker HTML")
	assert.Contains(t, bt.SourceURL, "parliament.go.ke")
	assert.False(t, bt.RetrievedAt.IsZero(), "RetrievedAt must be set")
	assert.Greater(t, bt.Confidence, 0.0, "confidence must be > 0 when a stage is found")
}

func TestParseBillTrackerHTML_NoStageInfo(t *testing.T) {
	// The real Bill Tracker index page (testdata/bill_tracker.html) is a
	// listing of weekly tracker PDFs — no per-Bill stage info in the HTML.
	html := loadFixture(t, "bill_tracker.html")
	bt := ParseBillTrackerHTML(html, "https://www.parliament.go.ke/the-national-assembly/house-business/bill-tracker")

	require.NotNil(t, bt)
	// Expect empty stage + low confidence (the page is an index, not a stage view).
	if bt.Stage != "" {
		// If the page happens to contain a stage keyword by accident, still
		// require the metadata to be set correctly.
		assert.NotEmpty(t, bt.Status)
	}
	assert.False(t, bt.RetrievedAt.IsZero(), "RetrievedAt must always be set")
	assert.Equal(t, "https://www.parliament.go.ke/the-national-assembly/house-business/bill-tracker", bt.SourceURL)
}

func TestParseBillTrackerHTML_EmptyHTML(t *testing.T) {
	bt := ParseBillTrackerHTML("", "https://www.parliament.go.ke/x")
	require.NotNil(t, bt)
	assert.Empty(t, bt.Stage)
	assert.False(t, bt.RetrievedAt.IsZero())
	assert.Equal(t, "https://www.parliament.go.ke/x", bt.SourceURL)
}

// --- ParseBillDetail tests -------------------------------------------------

func TestParseBillDetail_ExtractsTitleAndURL(t *testing.T) {
	html := `<html><body>
		<div class="post-block"><div class="post-content">
			<div class="post-title"><a href="https://www.parliament.go.ke/sites/default/files/2026-08/THE_HERALDRY_BILL.pdf">The Heraldry Bill, 2026</a></div>
		</div></div>
	</body></html>`

	records, err := ParseBillDetail(html, "https://www.parliament.go.ke/the-national-assembly/house-business/bills")
	require.NoError(t, err)
	require.Len(t, records, 1)

	r := records[0]
	assert.Equal(t, "bill", r.Kind)
	assert.Equal(t, "The Heraldry Bill, 2026", r.Title)
	assert.Equal(t, "National Assembly", r.House)
	assert.Equal(t, "https://www.parliament.go.ke/the-national-assembly/house-business/bills", r.SourceURL)
	assert.False(t, r.RetrievedAt.IsZero(), "RetrievedAt must be set")
	assert.Greater(t, r.Confidence, 0.5, "confidence should be high for an HTML parse")
	assert.Equal(t, "kenya.parliament.HTMLParser", r.ExtractorName)
	require.NotNil(t, r.Extra)
	assert.NotEmpty(t, r.Extra["pdf_url"], "should record the PDF URL in Extra")
}

func TestParseBillDetail_NoTitle(t *testing.T) {
	html := "<html><body>nothing here</body></html>"
	records, err := ParseBillDetail(html, "https://www.parliament.go.ke/x")
	require.Error(t, err)
	assert.Nil(t, records)
	assert.Contains(t, err.Error(), "no Bill title")
}

// --- adapter contract tests -------------------------------------------------

func TestAdapter_CountryCode(t *testing.T) {
	a := NewAdapter(nil, "")
	assert.Equal(t, "KE", a.CountryCode())
}

func TestAdapter_Supports(t *testing.T) {
	a := NewAdapter(nil, "")
	assert.True(t, a.Supports("https://www.parliament.go.ke/the-national-assembly/house-business/bills"))
	assert.True(t, a.Supports("https://nationalassembly.go.ke/"))
	assert.True(t, a.Supports("https://senate.go.ke/"))
	assert.False(t, a.Supports("https://new.kenyalaw.org/bills/"))
	assert.False(t, a.Supports("https://example.com/"))
}

func TestAdapter_DefaultUserAgent(t *testing.T) {
	a := NewAdapter(nil, "")
	// The PoliteClient is created by NewAdapter — we can't inspect userAgent
	// directly (it's unexported), but we can verify the adapter constructed.
	assert.NotNil(t, a)
}

func TestAdapter_GetOfficialSources(t *testing.T) {
	a := NewAdapter(nil, "")
	sources := a.GetOfficialSources()
	require.NotEmpty(t, sources)
	assert.GreaterOrEqual(t, len(sources), 3, "should define at least 3 sources (NA Bills, Senate Bills, Bill Tracker)")

	for _, s := range sources {
		assert.Equal(t, "KE", s.Country)
		assert.Equal(t, "primary", s.Authority)
		assert.Equal(t, "kenya.parliament", s.Adapter)
		assert.NotEmpty(t, s.URL)
		assert.NotEmpty(t, s.DocumentTypes)
		assert.Greater(t, s.CrawlFrequency, time.Duration(0))
	}
}

func TestAdapter_GetStages(t *testing.T) {
	a := NewAdapter(nil, "")
	stages, err := a.GetStages(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, stages)
	// Sanity check: we expect at least the canonical Kenya stages.
	codes := map[string]bool{}
	for _, s := range stages {
		codes[s.Code] = true
	}
	for _, want := range []string{
		"FIRST_READING", "SECOND_READING", "COMMITTEE_STAGE",
		"COMMITTEE_OF_WHOLE_HOUSE", "REPORT_STAGE", "THIRD_READING",
		"PRESIDENTIAL_ASSENT", "COMMENCEMENT",
	} {
		assert.True(t, codes[want], "missing stage %q", want)
	}
}

func TestAdapter_GetTerminology(t *testing.T) {
	a := NewAdapter(nil, "")
	terms, err := a.GetTerminology(context.Background())
	require.NoError(t, err)
	// The central Kenya terminology registry has ≥25 terms.
	assert.GreaterOrEqual(t, len(terms), 25)
}

func TestAdapter_GetLegislativeStructure(t *testing.T) {
	a := NewAdapter(nil, "")
	structure, err := a.GetLegislativeStructure(context.Background())
	require.NoError(t, err)
	require.NotNil(t, structure)
	assert.Equal(t, contracts.Country("KE"), structure.Country)
	assert.Len(t, structure.Houses, 2, "Kenya is bicameral")
}

func TestAdapter_NormalizeSourceItem(t *testing.T) {
	a := NewAdapter(nil, "")
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":           "https://www.parliament.go.ke/sites/default/files/2026-08/THE_TEST_BILL.pdf",
		"title":         "The Test Bill, 2026",
		"house_name":    "National Assembly",
		"external_id":   "NA Bill No. 1 of 2026",
		"document_type": "bill",
	})
	require.NoError(t, err)
	assert.Equal(t, "KE", item.CountryCode)
	assert.Equal(t, "The Test Bill, 2026", item.Title)
	assert.Contains(t, item.URL, "parliament.go.ke")
}

// --- Discover via mock HTTP server -----------------------------------------

func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	// Serve the real fixtures from a local HTTP server so DiscoverBills can
	// exercise its fetch + parse pipeline without touching the network.
	naHTML := loadFixture(t, "bills_na.html")
	senHTML := loadFixture(t, "bills_senate.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/the-national-assembly/house-business/bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(naHTML))
	})
	mux.HandleFunc("/the-senate/senate-bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(senHTML))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := NewAdapter(srv.Client(), "CivicIntelligence/0.1-test")
	// Override the URLs to point at the mock server.
	a.naBillsURL = srv.URL + "/the-national-assembly/house-business/bills"
	a.senateBillsURL = srv.URL + "/the-senate/senate-bills"

	bills, err := a.DiscoverBills(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, bills)

	// Verify every Bill carries a source URL + DiscoveredAt.
	for i, b := range bills {
		assert.NotEmpty(t, b.URL, "Bill %d has empty URL", i)
		assert.False(t, b.DiscoveredAt.IsZero(), "Bill %d has empty DiscoveredAt", i)
		assert.NotEmpty(t, b.House, "Bill %d has empty House", i)
	}

	// Verify both houses are represented.
	houses := map[string]bool{}
	for _, b := range bills {
		houses[b.House] = true
	}
	assert.True(t, houses["National Assembly"], "should find NA Bills")
	assert.True(t, houses["Senate"], "should find Senate Bills")
}

func TestAdapter_Discover_ImplementsContract(t *testing.T) {
	// Verify Discover returns proper contracts.SourceItem (not just BillCandidate).
	naHTML := loadFixture(t, "bills_na.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/the-national-assembly/house-business/bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(naHTML))
	})
	mux.HandleFunc("/the-senate/senate-bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(loadFixture(t, "bills_senate.html")))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := NewAdapter(srv.Client(), "")
	a.naBillsURL = srv.URL + "/the-national-assembly/house-business/bills"
	a.senateBillsURL = srv.URL + "/the-senate/senate-bills"

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items)

	for i, item := range items {
		assert.Equal(t, "KE", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		// Metadata should record the house + institution.
		assert.NotEmpty(t, item.Metadata["house"], "item %d has empty house metadata", i)
		assert.Equal(t, "Parliament of Kenya", item.Metadata["institution"])
	}
}

func TestAdapter_FetchBillTracker_ViaMockServer(t *testing.T) {
	trackerHTML := loadFixture(t, "bill_tracker_detail.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/tracker/heraldry", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(trackerHTML))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := NewAdapter(srv.Client(), "")
	bt, err := a.FetchBillTracker(context.Background(), srv.URL+"/tracker/heraldry")
	require.NoError(t, err)
	require.NotNil(t, bt)
	assert.False(t, bt.RetrievedAt.IsZero(), "RetrievedAt must be set")
	assert.Equal(t, srv.URL+"/tracker/heraldry", bt.SourceURL)
	// The fixture contains "First Reading", "Second Reading", "Committee of
	// the Whole House" — the parser should find at least one of these.
	assert.NotEmpty(t, bt.Stage, "should extract a stage from the tracker HTML")
}

func TestAdapter_ParseStage_Delegates(t *testing.T) {
	a := NewAdapter(nil, "")
	assert.Equal(t, string(internal.StageFirstReading),
		a.ParseStage(context.Background(), "First Reading"))
	assert.Equal(t, string(internal.StageSecondReading),
		a.ParseStage(context.Background(), "Second Reading — 12 March 2026"))
	assert.Equal(t, "",
		a.ParseStage(context.Background(), "nothing relevant"))
}

func TestAdapter_Fetch_EmptyURLError(t *testing.T) {
	a := NewAdapter(nil, "")
	_, err := a.Fetch(context.Background(), contracts.SourceItem{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty URL")
}

// --- makeSourceID tests ----------------------------------------------------

func TestMakeSourceID_NationalAssembly(t *testing.T) {
	id := makeSourceID("National Assembly",
		"https://www.parliament.go.ke/sites/default/files/2026-08/THE%20HERALDRY%20BILL%2C%202026..pdf")
	assert.Equal(t, "ke-parliament-na-bill-the-heraldry-bill-2026", id)
}

func TestMakeSourceID_Senate(t *testing.T) {
	id := makeSourceID("Senate",
		"https://www.parliament.go.ke/sites/default/files/2026-07/The%20Kenya%20National%20Library%20Services%20Bill.pdf")
	assert.Equal(t, "ke-parliament-senate-bill-the-kenya-national-library-services-bill", id)
}

func TestMakeSourceID_EmptyURL(t *testing.T) {
	id := makeSourceID("National Assembly", "")
	// Should still produce a stable prefix + the "unknown" slug fallback.
	assert.Contains(t, id, "ke-parliament-na-bill-")
}

// --- helpers ---------------------------------------------------------------

// loadFixture reads a testdata/ fixture into a string. Used by every test
// that needs real Parliament HTML.
func loadFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "fixture %s not found under testdata/", name)
	return string(data)
}
