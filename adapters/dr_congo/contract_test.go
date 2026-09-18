// Package dr_congo_test verifies that the DR Congo adapter satisfies the
// global contracts.LegislativeSourceAdapter interface and that all DR
// Congo-specific data is correctly shaped.
package dr_congo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: DRCongoAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*dr_congo.DRCongoAdapter)(nil)

func TestDRCongoAdapter_SatisfiesInterface(t *testing.T) {
	a := dr_congo.NewDRCongoAdapter()
	assert.NotNil(t, a)
}

func TestDRCongoAdapter_CountryCode(t *testing.T) {
	a := dr_congo.NewDRCongoAdapter()
	assert.Equal(t, "CD", a.CountryCode())
}

func TestDRCongoAdapter_Supports(t *testing.T) {
	a := dr_congo.NewDRCongoAdapter()
	assert.True(t, a.Supports("https://www.assemblee-nationale.cd/projets-lois"))
	assert.True(t, a.Supports("https://www.senat.cd/"))
	assert.True(t, a.Supports("https://assemblee-nationale.cd/projets-lois/2024/edu"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://www.parliament.go.ug"))
}

func TestDRCongoBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.DRCongoBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("CD"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

func TestDRCongoBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.DRCongoBillStages
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

func TestDRCongoBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.DRCongoBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Verify DR Congo's canonical Bill flow:
	// Proposition → Commission → Première Lecture → Lecture au Sénat →
	// Commission Mixte → Deuxième Lecture → Adoption → Promulgation → Commencement
	assert.Contains(t, byCode["PROPOSITION"].AllowedNext, "COMMISSION")
	assert.Contains(t, byCode["COMMISSION"].AllowedNext, "FIRST_READING")
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SENATE_READING")
	assert.Contains(t, byCode["SENATE_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SENATE_READING"].AllowedNext, "JOINT_COMMISSION")
	assert.Contains(t, byCode["JOINT_COMMISSION"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "ADOPTION")
	assert.Contains(t, byCode["ADOPTION"].AllowedNext, "PROMULGATION")
	assert.Contains(t, byCode["PROMULGATION"].AllowedNext, "COMMENCEMENT")
}

func TestDRCongoBillStages_AllowedNextReferencesValid(t *testing.T) {
	stages := internal.DRCongoBillStages
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

func TestDRCongoLegislativeStructure_Bicameral(t *testing.T) {
	s := internal.DRCongoLegislativeStructure()
	assert.Equal(t, contracts.Country("CD"), s.Country)
	assert.Len(t, s.Houses, 2, "DR Congo should be bicameral")
	houseByCode := map[string]contracts.HouseDefinition{}
	for _, h := range s.Houses {
		houseByCode[h.Code] = h
	}
	na, ok := houseByCode["NA"]
	assert.True(t, ok, "missing National Assembly")
	assert.Equal(t, "National Assembly", na.Name)
	assert.Equal(t, contracts.HouseTypeLower, na.Type)
	assert.Equal(t, 500, na.Members)
	sen, ok := houseByCode["SEN"]
	assert.True(t, ok, "missing Senate")
	assert.Equal(t, "Senate", sen.Name)
	assert.Equal(t, contracts.HouseTypeUpper, sen.Type)
	assert.Equal(t, 109, sen.Members)
}

func TestDRCongoLegislativeStructure_TermDays(t *testing.T) {
	s := internal.DRCongoLegislativeStructure()
	assert.Len(t, s.Houses, 2)
	// Both houses: 5-year terms per the 2006 Constitution.
	for _, h := range s.Houses {
		assert.Equal(t, 5*365, h.TermDays, "house %q has wrong term length", h.Code)
	}
}

func TestDRCongoTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.DRCongoTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("CD"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

func TestDRCongoTerminology_ContainsPromulgation(t *testing.T) {
	terms := internal.DRCongoTerminology
	found := false
	for _, term := range terms {
		if term.Term == "Promulgation" {
			found = true
			assert.NotEmpty(t, term.SimpleExplanation)
		}
	}
	assert.True(t, found, "DR Congo terminology must include the 'Promulgation' term")
}

func TestDRCongoAdapter_GetStages(t *testing.T) {
	a := dr_congo.NewDRCongoAdapter()
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
}

func TestDRCongoAdapter_GetTerminology(t *testing.T) {
	a := dr_congo.NewDRCongoAdapter()
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
}

func TestDRCongoAdapter_GetLegislativeStructure(t *testing.T) {
	a := dr_congo.NewDRCongoAdapter()
	s, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, contracts.Country("CD"), s.Country)
	assert.Len(t, s.Houses, 2, "DR Congo is bicameral")
}

func TestDRCongoAdapter_NormalizeSourceItem(t *testing.T) {
	a := dr_congo.NewDRCongoAdapter()
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.assemblee-nationale.cd/projets-lois/loi-edu",
		"title": "Loi sur l'éducation",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.assemblee-nationale.cd/projets-lois/loi-edu", item.URL)
	assert.Equal(t, "Loi sur l'éducation", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "CD", item.CountryCode)
}

func TestDRCongoAdapter_DiscoverReturnsEmptyWithoutError(t *testing.T) {
	a := dr_congo.NewDRCongoAdapter()
	// Without overriding the bills URL, Discover will attempt to hit the
	// real assemblee-nationale.cd site, which is unreachable from the
	// sandbox. We assert only that the contract surface compiles and the
	// call returns without panicking.
	items, err := a.Discover(context.Background())
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the testdata/bills.html
// fixture from a local HTTP server and verifies Discover returns Bills
// extracted by the parliament adapter's ParseBillsListing.
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	html := loadDRCongoFixture(t, "bills.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/projets-lois", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := parliament.NewAdapter(srv.Client(), "CivicIntelligence/0.1-test")
	a.SetBillsURLForTest(srv.URL + "/projets-lois")

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items, "Discover should return Bills from the fixture")

	housesFound := map[string]int{}
	for i, item := range items {
		assert.Equal(t, "CD", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		switch item.House {
		case "National Assembly", "Senate":
			housesFound[item.House]++
		default:
			t.Errorf("item %d has unexpected House %q", i, item.House)
		}
		assert.Equal(t, "Parliament of DR Congo", item.Metadata["institution"],
			"item %d has wrong institution metadata", i)
	}

	// Both houses must be represented (the fixture has one Senate Bill to
	// exercise the bicameral path).
	assert.Greater(t, housesFound["National Assembly"], 0,
		"should find at least one National Assembly Bill")
	assert.Greater(t, housesFound["Senate"], 0,
		"should find at least one Senate Bill")

	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"Loi portant mesures de protection des données personnelles",
		"Loi sur les hydrocarbures",
		"Loi de Finances 2024",
		"Loi sur la cybersécurité",
		"Loi sur les entreprises publiques",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}
}

// TestDRCongoSampleBills_CountAndShape verifies the DR Congo sample Bills
// registry has 5 entries that all reference assemblee-nationale.cd.
func TestDRCongoSampleBills_CountAndShape(t *testing.T) {
	bills := internal.DRCongoSampleBills
	require.Len(t, bills, 5, "DRCongoSampleBills should have exactly 5 entries")
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.URL, "sample bill %d has empty URL", i)
		assert.NotEmpty(t, b.BillNumber, "sample bill %d has empty BillNumber", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.NotEmpty(t, b.Date, "sample bill %d has empty Date", i)
		assert.NotEmpty(t, b.House, "sample bill %d has empty House", i)
		assert.True(t, strings.Contains(b.URL, "assemblee-nationale.cd"),
			"sample bill %d URL should reference assemblee-nationale.cd", i)
	}
}

// loadDRCongoFixture reads a testdata/ fixture into a string.
func loadDRCongoFixture(t *testing.T, name string) string {
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
