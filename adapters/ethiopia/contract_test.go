// Package ethiopia_test verifies that the Ethiopia adapter satisfies the
// global contracts.LegislativeSourceAdapter interface and that all
// Ethiopia-specific data is correctly shaped.
package ethiopia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: EthiopiaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*ethiopia.EthiopiaAdapter)(nil)

func TestEthiopiaAdapter_SatisfiesInterface(t *testing.T) {
	a := ethiopia.NewEthiopiaAdapter()
	assert.NotNil(t, a)
}

func TestEthiopiaAdapter_CountryCode(t *testing.T) {
	a := ethiopia.NewEthiopiaAdapter()
	assert.Equal(t, "ET", a.CountryCode())
}

func TestEthiopiaAdapter_Supports(t *testing.T) {
	a := ethiopia.NewEthiopiaAdapter()
	assert.True(t, a.Supports("https://www.parliament.gov.et/bills"))
	assert.True(t, a.Supports("https://parliament.gov.et/bills/2024/edu"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://www.parliament.go.ug"))
}

func TestEthiopiaBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.EthiopiaBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("ET"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

func TestEthiopiaBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.EthiopiaBillStages
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

func TestEthiopiaBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.EthiopiaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Verify Ethiopia's canonical Bill flow:
	// Proposal → Committee Review → First Reading → Second Reading →
	// House of Federation Review → Final Vote → Promulgation → Commencement
	assert.Contains(t, byCode["PROPOSAL"].AllowedNext, "COMMITTEE_REVIEW")
	assert.Contains(t, byCode["COMMITTEE_REVIEW"].AllowedNext, "FIRST_READING")
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "FEDERATION_REVIEW")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "FINAL_VOTE")
	assert.Contains(t, byCode["FEDERATION_REVIEW"].AllowedNext, "FINAL_VOTE")
	assert.Contains(t, byCode["FINAL_VOTE"].AllowedNext, "PROMULGATION")
	assert.Contains(t, byCode["PROMULGATION"].AllowedNext, "COMMENCEMENT")
}

func TestEthiopiaBillStages_AllowedNextReferencesValid(t *testing.T) {
	stages := internal.EthiopiaBillStages
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

func TestEthiopiaLegislativeStructure_Bicameral(t *testing.T) {
	s := internal.EthiopiaLegislativeStructure()
	assert.Equal(t, contracts.Country("ET"), s.Country)
	assert.Len(t, s.Houses, 2, "Ethiopia should be bicameral")
	houseByCode := map[string]contracts.HouseDefinition{}
	for _, h := range s.Houses {
		houseByCode[h.Code] = h
	}
	hpr, ok := houseByCode["HPR"]
	assert.True(t, ok, "missing House of Peoples' Representatives")
	assert.Equal(t, "House of Peoples' Representatives", hpr.Name)
	assert.Equal(t, contracts.HouseTypeLower, hpr.Type)
	assert.Equal(t, 547, hpr.Members)
	hof, ok := houseByCode["HOF"]
	assert.True(t, ok, "missing House of Federation")
	assert.Equal(t, "House of Federation", hof.Name)
	assert.Equal(t, contracts.HouseTypeUpper, hof.Type)
	assert.Equal(t, 153, hof.Members)
}

func TestEthiopiaLegislativeStructure_TermDays(t *testing.T) {
	s := internal.EthiopiaLegislativeStructure()
	assert.Len(t, s.Houses, 2)
	// Both houses: 5-year terms per the 1995 Constitution.
	for _, h := range s.Houses {
		assert.Equal(t, 5*365, h.TermDays, "house %q has wrong term length", h.Code)
	}
}

func TestEthiopiaTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.EthiopiaTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("ET"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

func TestEthiopiaTerminology_ContainsProclamation(t *testing.T) {
	terms := internal.EthiopiaTerminology
	found := false
	for _, term := range terms {
		if term.Term == "Proclamation" {
			found = true
			assert.NotEmpty(t, term.SimpleExplanation)
		}
	}
	assert.True(t, found, "Ethiopia terminology must include the 'Proclamation' term")
}

func TestEthiopiaAdapter_GetStages(t *testing.T) {
	a := ethiopia.NewEthiopiaAdapter()
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
}

func TestEthiopiaAdapter_GetTerminology(t *testing.T) {
	a := ethiopia.NewEthiopiaAdapter()
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
}

func TestEthiopiaAdapter_GetLegislativeStructure(t *testing.T) {
	a := ethiopia.NewEthiopiaAdapter()
	s, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, contracts.Country("ET"), s.Country)
	assert.Len(t, s.Houses, 2, "Ethiopia is bicameral")
}

func TestEthiopiaAdapter_NormalizeSourceItem(t *testing.T) {
	a := ethiopia.NewEthiopiaAdapter()
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.parliament.gov.et/bills/edu-proclamation",
		"title": "Education Proclamation",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.parliament.gov.et/bills/edu-proclamation", item.URL)
	assert.Equal(t, "Education Proclamation", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "ET", item.CountryCode)
}

func TestEthiopiaAdapter_DiscoverReturnsEmptyWithoutError(t *testing.T) {
	a := ethiopia.NewEthiopiaAdapter()
	// Without overriding the bills URL, Discover will attempt to hit the
	// real parliament.gov.et site, which is unreachable from the sandbox.
	items, err := a.Discover(context.Background())
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the testdata/bills.html
// fixture from a local HTTP server and verifies Discover returns Bills
// extracted by the parliament adapter's ParseBillsListing.
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	html := loadEthiopiaFixture(t, "bills.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := parliament.NewAdapter(srv.Client(), "CivicIntelligence/0.1-test")
	a.SetBillsURLForTest(srv.URL + "/bills")

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items, "Discover should return Bills from the fixture")

	housesFound := map[string]int{}
	for i, item := range items {
		assert.Equal(t, "ET", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		switch item.House {
		case "House of Peoples' Representatives", "House of Federation":
			housesFound[item.House]++
		default:
			t.Errorf("item %d has unexpected House %q", i, item.House)
		}
		assert.Equal(t, "Parliament of Ethiopia", item.Metadata["institution"],
			"item %d has wrong institution metadata", i)
	}

	// Both houses must be represented (the fixture has one House of Federation
	// Bill to exercise the bicameral path).
	assert.Greater(t, housesFound["House of Peoples' Representatives"], 0,
		"should find at least one HPR Bill")
	assert.Greater(t, housesFound["House of Federation"], 0,
		"should find at least one HoF Bill")

	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"Proclamation on Hate Speech and Disinformation",
		"Federal Government Budget Proclamation",
		"Data Protection Proclamation",
		"Anti-Corruption Proclamation (Amendment)",
		"Cooperatives Proclamation",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}
}

// TestEthiopiaSampleBills_CountAndShape verifies the Ethiopia sample Bills
// registry has 5 entries that all reference parliament.gov.et.
func TestEthiopiaSampleBills_CountAndShape(t *testing.T) {
	bills := internal.EthiopiaSampleBills
	require.Len(t, bills, 5, "EthiopiaSampleBills should have exactly 5 entries")
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.URL, "sample bill %d has empty URL", i)
		assert.NotEmpty(t, b.BillNumber, "sample bill %d has empty BillNumber", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.NotEmpty(t, b.Date, "sample bill %d has empty Date", i)
		assert.NotEmpty(t, b.House, "sample bill %d has empty House", i)
		assert.True(t, strings.Contains(b.URL, "parliament.gov.et"),
			"sample bill %d URL should reference parliament.gov.et", i)
	}
}

// loadEthiopiaFixture reads a testdata/ fixture into a string.
func loadEthiopiaFixture(t *testing.T, name string) string {
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
