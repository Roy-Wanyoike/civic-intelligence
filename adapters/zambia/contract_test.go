// Package zambia_test verifies that the Zambia adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Zambia-specific
// data is correctly shaped.
package zambia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: ZambiaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*zambia.ZambiaAdapter)(nil)

// Test 1: The adapter implements the interface and can be constructed.
func TestZambiaAdapter_SatisfiesInterface(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	assert.NotNil(t, a)
}

// Test 2: CountryCode returns the ISO 3166-1 alpha-2 code for Zambia.
func TestZambiaAdapter_CountryCode(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	assert.Equal(t, "ZM", a.CountryCode())
}

// Test 3: Supports recognises parliament.gov.zm URLs and rejects others.
func TestZambiaAdapter_Supports(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	assert.True(t, a.Supports("https://www.parliament.gov.zm/business/bills"))
	assert.True(t, a.Supports("https://parliament.gov.zm/bills/2024/health"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://parliament.go.ug"))
	assert.False(t, a.Supports("https://example.com"))
}

// Test 4: Every Zambia stage carries the country code "ZM" and the canonical
// flow stages are present.
func TestZambiaBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.ZambiaBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("ZM"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

// Test 5: All Zambia stages from the task description are present in the
// expected order:
//   First Reading → Second Reading → Committee Stage → Report Stage →
//   Third Reading → Presidential Assent → Commencement.
func TestZambiaBillStages_FullStageChain(t *testing.T) {
	stages := internal.ZambiaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	expected := []string{
		"FIRST_READING",
		"SECOND_READING",
		"COMMITTEE_STAGE",
		"REPORT_STAGE",
		"THIRD_READING",
		"PRESIDENTIAL_ASSENT",
		"COMMENCEMENT",
	}
	for _, code := range expected {
		s, ok := byCode[code]
		assert.True(t, ok, "missing expected stage %s", code)
		assert.NotEmpty(t, s.Name)
	}
}

// Test 6: Terminal stages exist and have IsTerminal=true.
func TestZambiaBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.ZambiaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	for _, terminal := range []string{"COMMENCEMENT", "REJECTED", "WITHDRAWN"} {
		s, ok := byCode[terminal]
		assert.True(t, ok, "missing terminal stage %s", terminal)
		assert.True(t, s.IsTerminal, "stage %s should be terminal", terminal)
		assert.Empty(t, s.AllowedNext, "terminal stage %s should have no AllowedNext", terminal)
	}
}

// Test 7: Stage transitions are valid — every AllowedNext code references a
// known stage, and the canonical Zambia flow is encoded correctly.
func TestZambiaBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.ZambiaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	for _, s := range stages {
		for _, next := range s.AllowedNext {
			_, ok := byCode[next]
			assert.True(t, ok, "stage %q allows transition to unknown stage %q", s.Code, next)
		}
	}
	// Verify the canonical Zambia Bill flow.
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "COMMITTEE_STAGE")
	assert.Contains(t, byCode["COMMITTEE_STAGE"].AllowedNext, "REPORT_STAGE")
	assert.Contains(t, byCode["REPORT_STAGE"].AllowedNext, "THIRD_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "PRESIDENTIAL_ASSENT")
	assert.Contains(t, byCode["PRESIDENTIAL_ASSENT"].AllowedNext, "COMMENCEMENT")
}

// Test 8: Legislative structure is unicameral with the correct member count
// (National Assembly of Zambia: 167 members).
func TestZambiaLegislativeStructure_Unicameral(t *testing.T) {
	s := internal.ZambiaLegislativeStructure()
	assert.Equal(t, contracts.Country("ZM"), s.Country)
	assert.Equal(t, "ZM", s.CountryCode)
	assert.Equal(t, "Zambia", s.CountryName)
	assert.Len(t, s.Houses, 1, "Zambia should be unicameral (one House)")
	assert.Equal(t, "National Assembly of Zambia", s.Houses[0].Name)
	assert.Equal(t, contracts.HouseTypeSingle, s.Houses[0].Type)
}

// Test 9: Member count of the National Assembly is 167.
func TestZambiaLegislativeStructure_MemberCount(t *testing.T) {
	s := internal.ZambiaLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	assert.Equal(t, 167, s.Houses[0].Members, "National Assembly of Zambia has 167 members per Article 63")
}

// Test 10: Terminology registry has at least 20 terms, each with country="ZM".
func TestZambiaTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.ZambiaTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("ZM"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

// Test 11: Specific terms from the task description are present.
func TestZambiaTerminology_ExpectedTermsExist(t *testing.T) {
	terms := internal.ZambiaTerminology
	byTerm := map[string]bool{}
	for _, term := range terms {
		byTerm[term.Term] = true
	}
	for _, want := range []string{
		"First Reading",
		"Second Reading",
		"Committee Stage",
		"Report Stage",
		"Third Reading",
		"Presidential Assent",
		"Commencement",
		"Hansard",
		"Order Paper",
		"National Assembly",
	} {
		assert.True(t, byTerm[want], "expected term %q is missing", want)
	}
}

// Test 12: Adapter's GetStages returns the same number of stages as the
// internal registry.
func TestZambiaAdapter_GetStages(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
	assert.Len(t, stages, len(internal.ZambiaBillStages))
	for _, s := range stages {
		assert.Equal(t, contracts.Country("ZM"), s.Country)
	}
}

// Test 13: Adapter's GetTerminology returns at least 20 terms.
func TestZambiaAdapter_GetTerminology(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
	assert.Len(t, terms, len(internal.ZambiaTerminology))
	for _, tm := range terms {
		assert.Equal(t, contracts.Country("ZM"), tm.Country)
	}
}

// Test 14: Adapter's GetLegislativeStructure returns the unicameral structure.
func TestZambiaAdapter_GetLegislativeStructure(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	structure, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, structure)
	assert.Equal(t, contracts.Country("ZM"), structure.Country)
	assert.Len(t, structure.Houses, 1, "Zambia is unicameral")
}

// Test 15: NormalizeSourceItem maps a raw metadata map into a SourceItem.
func TestZambiaAdapter_NormalizeSourceItem(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.parliament.gov.zm/business/bills/public-health-amendment-bill-2024",
		"title": "Public Health (Amendment) Bill, 2024",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.parliament.gov.zm/business/bills/public-health-amendment-bill-2024", item.URL)
	assert.Equal(t, "Public Health (Amendment) Bill, 2024", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "ZM", item.CountryCode)
}

// Test 16: Discover does not panic when invoked without a mock server.
func TestZambiaAdapter_Discover(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	items, err := a.Discover(context.Background())
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the testdata/bills.html
// fixture from a local HTTP server and verifies Discover returns Bills
// extracted by the parliament adapter's ParseBillsListing.
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	html := loadZambiaFixture(t, "bills.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/business/bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := parliament.NewAdapter(srv.Client(), "CivicIntelligence/0.1-test")
	a.SetBillsURLForTest(srv.URL + "/business/bills")

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items, "Discover should return Bills from the fixture")

	for i, item := range items {
		assert.Equal(t, "ZM", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		assert.Equal(t, "National Assembly of Zambia", item.House, "item %d has wrong House", i)
		assert.Equal(t, "National Assembly of Zambia", item.Metadata["house"], "item %d has wrong house metadata", i)
		assert.Equal(t, "National Assembly of Zambia", item.Metadata["institution"], "item %d has wrong institution metadata", i)
	}

	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"Public Health (Amendment) Bill, 2024",
		"Cyber Security and Cyber Crimes Bill, 2024",
		"Public Order (Amendment) Bill, 2024",
		"Public Finance Management (Amendment) Bill, 2024",
		"Data Protection Bill, 2023",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}

	// Verify the stage mapping for the first Bill.
	assert.Equal(t, "SECOND_READING", items[0].Metadata["stage"])
}

// TestZambiaSampleBills_CountAndShape verifies the Zambia sample Bills
// registry has 5 entries that all reference parliament.gov.zm.
func TestZambiaSampleBills_CountAndShape(t *testing.T) {
	bills := internal.ZambiaSampleBills
	require.Len(t, bills, 5, "ZambiaSampleBills should have exactly 5 entries")
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.SourceURL, "sample bill %d has empty SourceURL", i)
		assert.NotEmpty(t, b.Number, "sample bill %d has empty Number", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.True(t, strings.Contains(b.SourceURL, "parliament.gov.zm"),
			"sample bill %d SourceURL should reference parliament.gov.zm", i)
	}
}

// Test 17: Fetch rejects an empty URL with a clear error.
func TestZambiaAdapter_Fetch_EmptyURL(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	_, err := a.Fetch(context.Background(), contracts.SourceItem{URL: ""})
	assert.Error(t, err)
}

// Test 18: Parse of a PDF document returns a low-confidence placeholder
// ExtractedRecord (the documents service handles full extraction).
func TestZambiaAdapter_Parse_PDF(t *testing.T) {
	a := zambia.NewZambiaAdapter(zambia.Dependencies{})
	recs, err := a.Parse(context.Background(), contracts.RawDocument{
		URL:      "https://www.parliament.gov.zm/business/bills/public-health-amendment-bill-2024.pdf",
		Bytes:    []byte("%PDF-1.4"),
		MimeType: "application/pdf",
	})
	assert.NoError(t, err)
	assert.Len(t, recs, 1)
	assert.Equal(t, "bill", recs[0].Kind)
	assert.Equal(t, "zambia.parliament.PDFPlaceholder", recs[0].ExtractorName)
	assert.Less(t, recs[0].Confidence, 0.5)
}

// loadZambiaFixture reads a testdata/ fixture into a string.
func loadZambiaFixture(t *testing.T, name string) string {
	t.Helper()
	candidates := []string{
		filepath.Join("parliament", "testdata", name),
		filepath.Join("testdata", name),
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			return string(data)
		}
	}
	t.Fatalf("fixture %s not found under testdata/", name)
	return ""
}
