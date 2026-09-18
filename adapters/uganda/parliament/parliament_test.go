// Package parliament — tests for the Parliament of Uganda adapter.
//
// The HTML fixture in testdata/bills.html is a realistic reconstruction of the
// Parliament of Uganda Bills listing page
// (https://www.parliament.go.ug/business/bills), structured as a
// <table class="bills-listing"> with one <tr> per Bill. The five Bills in the
// fixture mirror adapters/uganda/internal.UgandaSampleBills.
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

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// Compile-time assertion: Adapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*Adapter)(nil)

// --- adapter contract tests ------------------------------------------------

func TestAdapter_CountryCode(t *testing.T) {
        a := NewAdapter(nil, "")
        assert.Equal(t, "UG", a.CountryCode())
}

func TestAdapter_Supports(t *testing.T) {
        a := NewAdapter(nil, "")
        // Accepts the official Parliament of Uganda domain.
        assert.True(t, a.Supports("https://www.parliament.go.ug/business/bills"))
        assert.True(t, a.Supports("https://parliament.go.ug/business/bills/national-coffee-bill-2024"))
        // Accepts the civic aggregator domain.
        assert.True(t, a.Supports("https://parliamentwatch.ug/bills"))
        // Rejects foreign / unrelated URLs.
        assert.False(t, a.Supports("https://www.parliament.go.ke/the-national-assembly/house-business/bills"))
        assert.False(t, a.Supports("https://new.kenyalaw.org/bills/"))
        assert.False(t, a.Supports("https://example.com/"))
}

func TestAdapter_DefaultUserAgent(t *testing.T) {
        a := NewAdapter(nil, "")
        ua := a.UserAgent()
        assert.NotEmpty(t, ua, "UserAgent must be non-empty")
        assert.Contains(t, ua, "CivicIntelligence")
        assert.Contains(t, ua, "github.com/Roy-Wanyoike/civic-intelligence")
}

func TestAdapter_GetOfficialSources(t *testing.T) {
        a := NewAdapter(nil, "")
        sources := a.GetOfficialSources()
        require.NotEmpty(t, sources, "should define at least the Bills source")

        // At least one source must point at the canonical Bills listing URL.
        var billsURLFound bool
        for _, s := range sources {
                assert.Equal(t, "UG", s.Country)
                assert.Equal(t, "primary", s.Authority)
                assert.Equal(t, "uganda.parliament", s.Adapter)
                assert.NotEmpty(t, s.URL)
                assert.Contains(t, s.DocumentTypes, "bill")
                assert.Greater(t, s.CrawlFrequency, 0)
                if strings.Contains(s.URL, "parliament.go.ug") && strings.Contains(s.URL, "bills") {
                        billsURLFound = true
                }
        }
        assert.True(t, billsURLFound, "should define a source for the Parliament Bills listing URL")
}

func TestAdapter_GetStages(t *testing.T) {
        a := NewAdapter(nil, "")
        stages, err := a.GetStages(context.Background())
        require.NoError(t, err)
        require.NotEmpty(t, stages)

        // Uganda's Bill lifecycle has at least 7 stages
        // (First Reading → … → Commencement), plus REJECTED as a terminal state.
        assert.GreaterOrEqual(t, len(stages), 7, "should return at least 7 Uganda stages")

        codes := make(map[string]bool, len(stages))
        for _, s := range stages {
                assert.Equal(t, contracts.Country("UG"), s.Country)
                codes[s.Code] = true
        }
        // Verify the 7 lifecycle stages are all present.
        for _, want := range []string{
                "FIRST_READING", "SECOND_READING", "COMMITTEE_STAGE",
                "REPORT_STAGE", "THIRD_READING", "PRESIDENTIAL_ASSENT",
                "COMMENCEMENT",
        } {
                assert.True(t, codes[want], "missing stage %q", want)
        }
}

// --- stage mapping tests ----------------------------------------------------

func TestMapStageText_CanonicalStages(t *testing.T) {
        tests := []struct {
                input string
                want  string
        }{
                {"First Reading", "FIRST_READING"},
                {"Second Reading", "SECOND_READING"},
                {"Committee Stage", "COMMITTEE_STAGE"},
                {"Report Stage", "REPORT_STAGE"},
                {"Third Reading", "THIRD_READING"},
                {"Presidential Assent", "PRESIDENTIAL_ASSENT"},
                {"Commencement", "COMMENCEMENT"},
                {"Rejected", "REJECTED"},
                // Tolerates trailing dates / context.
                {"Second Reading — 12 March 2024", "SECOND_READING"},
                // Numeric variants.
                {"1st Reading", "FIRST_READING"},
                {"2nd Reading", "SECOND_READING"},
                {"3rd Reading", "THIRD_READING"},
        }
        for _, tc := range tests {
                assert.Equal(t, tc.want, MapStageText(tc.input),
                        "MapStageText(%q) should be %q", tc.input, tc.want)
        }
}

func TestMapStageText_EmptyAndUnknown(t *testing.T) {
        assert.Equal(t, "", MapStageText(""))
        assert.Equal(t, "", MapStageText("   "))
        assert.Equal(t, "", MapStageText("The Bill is scheduled for tabling next week"))
}

// --- Discover via mock HTTP server -----------------------------------------

func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
        html := loadFixture(t, "bills.html")

        // Capture the User-Agent header sent by Discover so we can assert it
        // outside the handler goroutine (testify assertions are safe for
        // concurrent use, but recording + asserting in the main goroutine is
        // cleaner and avoids any race with srv.Close()).
        var receivedUA string
        var uaSeen bool

        mux := http.NewServeMux()
        mux.HandleFunc("/business/bills", func(w http.ResponseWriter, r *http.Request) {
                receivedUA = r.Header.Get("User-Agent")
                uaSeen = true
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                _, _ = w.Write([]byte(html))
        })
        srv := httptest.NewServer(mux)
        defer srv.Close()

        a := NewAdapter(srv.Client(), "")
        // Override the Bills URL to point at the mock server.
        a.billsURL = srv.URL + "/business/bills"

        items, err := a.Discover(context.Background())
        require.NoError(t, err)
        require.NotEmpty(t, items, "should discover Bills from the fixture")

        // Verify the User-Agent header was sent on the HTTP request.
        assert.True(t, uaSeen, "the mock server should have received a request")
        assert.NotEmpty(t, receivedUA, "Discover must send a User-Agent header")
        assert.Contains(t, receivedUA, "CivicIntelligence",
                "User-Agent should identify the platform")

        // Verify every SourceItem is well-formed.
        for i, item := range items {
                assert.Equal(t, "UG", item.CountryCode, "item %d has wrong CountryCode", i)
                assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
                assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
                assert.Equal(t, "Parliament of Uganda", item.House, "item %d has wrong House", i)
                assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
                assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
                assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
                assert.Equal(t, "Parliament of Uganda", item.Metadata["institution"],
                        "item %d has wrong institution metadata", i)
        }

        // Should discover all 5 fixture Bills.
        assert.Len(t, items, 5, "fixture has 5 Bills")

        // Verify the first Bill has the expected fields.
        first := items[0]
        assert.Equal(t, "The National Coffee Bill, 2024", first.Title)
        assert.Equal(t, "The Bill No. 12 of 2024", first.ExternalID)
        assert.Contains(t, first.URL, "national-coffee-bill-2024")
        assert.Contains(t, first.SourceID, "ug-parliament-bill-national-coffee-bill-2024")
        assert.Equal(t, "SECOND_READING", first.Metadata["stage"])
}

// --- Parse tests ------------------------------------------------------------

func TestAdapter_Parse_ExtractsBill(t *testing.T) {
        html := loadFixture(t, "bills.html")
        doc := contracts.RawDocument{
                URL:         "https://www.parliament.go.ug/business/bills",
                Bytes:       []byte(html),
                MimeType:    "text/html",
                RetrievedAt: time.Now().UTC(),
        }

        a := NewAdapter(nil, "")
        records, err := a.Parse(context.Background(), doc)
        require.NoError(t, err)
        require.Len(t, records, 5, "fixture has 5 Bills")

        // Verify the first extracted record has the expected Bill fields.
        r := records[0]
        assert.Equal(t, "bill", r.Kind)
        assert.Equal(t, "The National Coffee Bill, 2024", r.Title)
        assert.Equal(t, "The Bill No. 12 of 2024", r.Identifier)
        assert.Equal(t, "Parliament of Uganda", r.House)
        assert.Equal(t, "SECOND_READING", r.Stage)
        assert.Contains(t, r.Sponsor, "Minister of Agriculture")
        assert.Equal(t, "uganda.parliament.HTMLParser", r.ExtractorName)
        assert.False(t, r.RetrievedAt.IsZero(), "RetrievedAt must be set")
        assert.Greater(t, r.Confidence, 0.5, "confidence should be high for a fully-parsed Bill")
        // The Extra map should carry the raw stage text + bill URL.
        require.NotNil(t, r.Extra)
        assert.Equal(t, "Second Reading", r.Extra["stage_raw"])
        assert.Equal(t, "/business/bills/national-coffee-bill-2024", r.Extra["bill_url"])

        // Verify the stage mapping for all 5 records.
        expectedStages := []string{
                "SECOND_READING",       // National Coffee Bill
                "COMMITTEE_STAGE",      // Traffic and Road Safety (Amendment) Bill
                "FIRST_READING",        // Anti-Corruption (Amendment) Bill
                "THIRD_READING",        // Public Finance Management (Amendment) Bill
                "PRESIDENTIAL_ASSENT",  // Data Protection and Privacy (Amendment) Bill
        }
        for i, want := range expectedStages {
                assert.Equal(t, want, records[i].Stage,
                        "record %d (%s) has wrong stage", i, records[i].Title)
        }
}

func TestAdapter_Parse_NoBillsReturnsError(t *testing.T) {
        a := NewAdapter(nil, "")
        _, err := a.Parse(context.Background(), contracts.RawDocument{
                URL:      "https://www.parliament.go.ug/business/bills",
                Bytes:    []byte("<html><body>No Bills here</body></html>"),
                MimeType: "text/html",
        })
        require.Error(t, err)
        assert.Contains(t, err.Error(), "no Bills found")
}

func TestAdapter_Fetch_EmptyURLError(t *testing.T) {
        a := NewAdapter(nil, "")
        _, err := a.Fetch(context.Background(), contracts.SourceItem{})
        require.Error(t, err)
        assert.Contains(t, err.Error(), "empty URL")
}

// --- helpers ---------------------------------------------------------------

// loadFixture reads a testdata/ fixture into a string. Used by every test
// that needs realistic Parliament HTML.
func loadFixture(t *testing.T, name string) string {
        t.Helper()
        path := filepath.Join("testdata", name)
        data, err := os.ReadFile(path)
        require.NoError(t, err, "fixture %s not found under testdata/", name)
        return string(data)
}
