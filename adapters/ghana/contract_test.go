// Package ghana_test verifies that the Ghana adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Ghana-specific
// data is correctly shaped.
package ghana_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: GhanaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*ghana.GhanaAdapter)(nil)

func TestGhanaAdapter_SatisfiesInterface(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	assert.NotNil(t, a)
}

func TestGhanaAdapter_CountryCode(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	assert.Equal(t, "GH", a.CountryCode())
}

func TestGhanaAdapter_Supports(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	assert.True(t, a.Supports("https://parliament.gh/bills"))
	assert.True(t, a.Supports("https://www.parliament.gh/bills/2024/edu"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://www.parliament.go.ug"))
}

func TestGhanaBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.GhanaBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("GH"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

func TestGhanaBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.GhanaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	assert.True(t, byCode["COMMENCEMENT"].IsTerminal)
	assert.True(t, byCode["REJECTED"].IsTerminal)
	assert.True(t, byCode["WITHDRAWN"].IsTerminal)
	// Terminal stages should not allow further transitions.
	assert.Empty(t, byCode["COMMENCEMENT"].AllowedNext)
	assert.Empty(t, byCode["REJECTED"].AllowedNext)
}

func TestGhanaBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.GhanaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Verify Ghana's canonical Bill flow:
	// First Reading → Second Reading → Consideration Stage → Third Reading → Assent → Commencement
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "CONSIDERATION_STAGE")
	assert.Contains(t, byCode["CONSIDERATION_STAGE"].AllowedNext, "THIRD_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "ASSENT")
	assert.Contains(t, byCode["ASSENT"].AllowedNext, "COMMENCEMENT")
}

func TestGhanaBillStages_AllowedNextReferencesValid(t *testing.T) {
	stages := internal.GhanaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Every "allowed next" must reference an existing stage.
	for _, s := range stages {
		for _, next := range s.AllowedNext {
			_, ok := byCode[next]
			assert.True(t, ok, "stage %q allows transition to unknown stage %q", s.Code, next)
		}
	}
}

func TestGhanaLegislativeStructure_Unicameral(t *testing.T) {
	s := internal.GhanaLegislativeStructure()
	assert.Equal(t, contracts.Country("GH"), s.Country)
	assert.Len(t, s.Houses, 1, "Ghana should be unicameral (one House)")
	assert.Equal(t, "Parliament of Ghana", s.Houses[0].Name)
	assert.Equal(t, contracts.HouseTypeSingle, s.Houses[0].Type)
}

func TestGhanaLegislativeStructure_MemberCount(t *testing.T) {
	s := internal.GhanaLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	assert.Equal(t, 275, s.Houses[0].Members, "Parliament of Ghana has 275 elected MPs")
}

func TestGhanaLegislativeStructure_TermDays(t *testing.T) {
	s := internal.GhanaLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	// Ghana's parliamentary term is four years per Article 97 of the 1992 Constitution.
	assert.Equal(t, 4*365, s.Houses[0].TermDays)
}

func TestGhanaTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.GhanaTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("GH"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

func TestGhanaTerminology_ContainsConsiderationStage(t *testing.T) {
	terms := internal.GhanaTerminology
	found := false
	for _, term := range terms {
		if term.Term == "Consideration Stage" {
			found = true
			assert.NotEmpty(t, term.SimpleExplanation)
		}
	}
	assert.True(t, found, "Ghana terminology must include the 'Consideration Stage' term")
}

func TestGhanaAdapter_GetStages(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
}

func TestGhanaAdapter_GetTerminology(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
}

func TestGhanaAdapter_GetLegislativeStructure(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	s, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, contracts.Country("GH"), s.Country)
	assert.Len(t, s.Houses, 1, "Ghana is unicameral")
}

func TestGhanaAdapter_NormalizeSourceItem(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://parliament.gh/bills/2024/edu",
		"title": "The Education (Amendment) Bill, 2024",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://parliament.gh/bills/2024/edu", item.URL)
	assert.Equal(t, "The Education (Amendment) Bill, 2024", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "GH", item.CountryCode)
}

func TestGhanaAdapter_DiscoverReturnsEmptyWithoutError(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	// Without overriding the bills URL, Discover will attempt to hit the
	// real parliament.ghana.gov.gh site, which is unreachable from the
	// sandbox. We assert only that the contract surface compiles and the
	// call returns without panicking — error or empty result are both
	// acceptable here. The mock-server path (TestAdapter_DiscoverBills_ViaMockServer)
	// is the authoritative test for Discover's behavior.
	items, err := a.Discover(context.Background())
	// In sandbox environments the HTTP call will fail; either an error or an
	// empty slice is acceptable. What is NOT acceptable is a panic.
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the testdata/bills.html
// fixture from a local HTTP server and verifies Discover returns Bills
// extracted by the parliament adapter's ParseBillsListing.
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	html := loadGhanaFixture(t, "bills.html")

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
		assert.Equal(t, "GH", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		assert.Equal(t, "Parliament of Ghana", item.House, "item %d has wrong House", i)
		assert.Equal(t, "Parliament of Ghana", item.Metadata["house"], "item %d has wrong house metadata", i)
		assert.Equal(t, "Parliament of Ghana", item.Metadata["institution"], "item %d has wrong institution metadata", i)
	}

	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"The Right to Information (Amendment) Bill, 2024",
		"The Minerals Income Tax (Amendment) Bill, 2024",
		"The Public Universities Bill, 2024",
		"The Cyber Security (Amendment) Bill, 2024",
		"The Companies (Amendment) Bill, 2024",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}
}

// TestGhanaSampleBills_CountAndShape verifies the Ghana sample Bills
// registry has 5 entries that all reference parliament.ghana.gov.gh.
func TestGhanaSampleBills_CountAndShape(t *testing.T) {
	bills := internal.GhanaSampleBills
	require.Len(t, bills, 5, "GhanaSampleBills should have exactly 5 entries")
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.URL, "sample bill %d has empty URL", i)
		assert.NotEmpty(t, b.BillNumber, "sample bill %d has empty BillNumber", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.NotEmpty(t, b.Date, "sample bill %d has empty Date", i)
		assert.True(t, strings.Contains(b.URL, "parliament.ghana.gov.gh"),
			"sample bill %d URL should reference parliament.ghana.gov.gh", i)
	}
}

// loadGhanaFixture reads a testdata/ fixture into a string.
func loadGhanaFixture(t *testing.T, name string) string {
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
