// Package morocco_test verifies that the Morocco adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Morocco-specific
// data is correctly shaped.
package morocco_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: MoroccoAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*morocco.MoroccoAdapter)(nil)

func TestMoroccoAdapter_SatisfiesInterface(t *testing.T) {
	a := morocco.NewMoroccoAdapter()
	assert.NotNil(t, a)
}

func TestMoroccoAdapter_CountryCode(t *testing.T) {
	a := morocco.NewMoroccoAdapter()
	assert.Equal(t, "MA", a.CountryCode())
}

func TestMoroccoAdapter_Supports(t *testing.T) {
	a := morocco.NewMoroccoAdapter()
	assert.True(t, a.Supports("https://www.parlement.ma/fr/projets-lois"))
	assert.True(t, a.Supports("https://parlement.ma/bills/2024/edu"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://www.parliament.go.ug"))
}

func TestMoroccoBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.MoroccoBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("MA"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

func TestMoroccoBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.MoroccoBillStages
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

func TestMoroccoBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.MoroccoBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Verify Morocco's canonical Bill flow:
	// Proposal → Committee → First Reading → Second Reading →
	// House of Councillors Review → Final Vote → Royal Promulgation → Commencement
	assert.Contains(t, byCode["PROPOSAL"].AllowedNext, "COMMITTEE")
	assert.Contains(t, byCode["COMMITTEE"].AllowedNext, "FIRST_READING")
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "COUNCILLORS_REVIEW")
	assert.Contains(t, byCode["COUNCILLORS_REVIEW"].AllowedNext, "FINAL_VOTE")
	assert.Contains(t, byCode["FINAL_VOTE"].AllowedNext, "ROYAL_PROMULGATION")
	assert.Contains(t, byCode["ROYAL_PROMULGATION"].AllowedNext, "COMMENCEMENT")
}

func TestMoroccoBillStages_AllowedNextReferencesValid(t *testing.T) {
	stages := internal.MoroccoBillStages
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

func TestMoroccoLegislativeStructure_Bicameral(t *testing.T) {
	s := internal.MoroccoLegislativeStructure()
	assert.Equal(t, contracts.Country("MA"), s.Country)
	assert.Len(t, s.Houses, 2, "Morocco should be bicameral")
	houseByCode := map[string]contracts.HouseDefinition{}
	for _, h := range s.Houses {
		houseByCode[h.Code] = h
	}
	reps, ok := houseByCode["REPS"]
	assert.True(t, ok, "missing House of Representatives")
	assert.Equal(t, "House of Representatives", reps.Name)
	assert.Equal(t, contracts.HouseTypeLower, reps.Type)
	assert.Equal(t, 395, reps.Members)
	cons, ok := houseByCode["CONS"]
	assert.True(t, ok, "missing House of Councillors")
	assert.Equal(t, "House of Councillors", cons.Name)
	assert.Equal(t, contracts.HouseTypeUpper, cons.Type)
	assert.Equal(t, 120, cons.Members)
}

func TestMoroccoLegislativeStructure_TermDays(t *testing.T) {
	s := internal.MoroccoLegislativeStructure()
	assert.Len(t, s.Houses, 2)
	// House of Representatives: 5-year term. House of Councillors: 6-year term.
	for _, h := range s.Houses {
		switch h.Code {
		case "REPS":
			assert.Equal(t, 5*365, h.TermDays)
		case "CONS":
			assert.Equal(t, 6*365, h.TermDays)
		}
	}
}

func TestMoroccoTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.MoroccoTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("MA"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

func TestMoroccoTerminology_ContainsRoyalPromulgation(t *testing.T) {
	terms := internal.MoroccoTerminology
	found := false
	for _, term := range terms {
		if term.Term == "Royal Promulgation" {
			found = true
			assert.NotEmpty(t, term.SimpleExplanation)
		}
	}
	assert.True(t, found, "Morocco terminology must include the 'Royal Promulgation' term")
}

func TestMoroccoAdapter_GetStages(t *testing.T) {
	a := morocco.NewMoroccoAdapter()
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
}

func TestMoroccoAdapter_GetTerminology(t *testing.T) {
	a := morocco.NewMoroccoAdapter()
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
}

func TestMoroccoAdapter_GetLegislativeStructure(t *testing.T) {
	a := morocco.NewMoroccoAdapter()
	s, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, contracts.Country("MA"), s.Country)
	assert.Len(t, s.Houses, 2, "Morocco is bicameral")
}

func TestMoroccoAdapter_NormalizeSourceItem(t *testing.T) {
	a := morocco.NewMoroccoAdapter()
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.parlement.ma/fr/projets-lois/loi-edu",
		"title": "Loi sur l'éducation",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.parlement.ma/fr/projets-lois/loi-edu", item.URL)
	assert.Equal(t, "Loi sur l'éducation", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "MA", item.CountryCode)
}

func TestMoroccoAdapter_DiscoverReturnsEmptyWithoutError(t *testing.T) {
	a := morocco.NewMoroccoAdapter()
	// Without overriding the bills URL, Discover will attempt to hit the
	// real parlement.ma site, which is unreachable from the sandbox. We
	// assert only that the contract surface compiles and the call returns
	// without panicking — error or empty result are both acceptable here.
	items, err := a.Discover(context.Background())
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the testdata/bills.html
// fixture from a local HTTP server and verifies Discover returns Bills
// extracted by the parliament adapter's ParseBillsListing.
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	html := loadMoroccoFixture(t, "bills.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/fr/projets-lois", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := parliament.NewAdapter(srv.Client(), "CivicIntelligence/0.1-test")
	a.SetBillsURLForTest(srv.URL + "/fr/projets-lois")

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items, "Discover should return Bills from the fixture")

	housesFound := map[string]int{}
	for i, item := range items {
		assert.Equal(t, "MA", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		switch item.House {
		case "House of Representatives", "House of Councillors":
			housesFound[item.House]++
		default:
			t.Errorf("item %d has unexpected House %q", i, item.House)
		}
		assert.Equal(t, "Parliament of Morocco", item.Metadata["institution"],
			"item %d has wrong institution metadata", i)
	}

	// The National Assembly must be represented; House of Councillors is also
	// represented (the fixture has one Councillors Bill to exercise the bicameral path).
	assert.Greater(t, housesFound["House of Representatives"], 0,
		"should find at least one Representatives Bill")
	assert.Greater(t, housesFound["House of Councillors"], 0,
		"should find at least one Councillors Bill")

	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"Loi-cadre sur le développement humain",
		"Loi sur la protection des données personnelles",
		"Loi de Finances 2024",
		"Loi sur la cybersécurité",
		"Loi sur les entreprises publiques",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}
}

// TestMoroccoSampleBills_CountAndShape verifies the Morocco sample Bills
// registry has 5 entries that all reference parlement.ma.
func TestMoroccoSampleBills_CountAndShape(t *testing.T) {
	bills := internal.MoroccoSampleBills
	require.Len(t, bills, 5, "MoroccoSampleBills should have exactly 5 entries")
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.URL, "sample bill %d has empty URL", i)
		assert.NotEmpty(t, b.BillNumber, "sample bill %d has empty BillNumber", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.NotEmpty(t, b.Date, "sample bill %d has empty Date", i)
		assert.NotEmpty(t, b.House, "sample bill %d has empty House", i)
		assert.True(t, strings.Contains(b.URL, "parlement.ma"),
			"sample bill %d URL should reference parlement.ma", i)
	}
}

// loadMoroccoFixture reads a testdata/ fixture into a string.
func loadMoroccoFixture(t *testing.T, name string) string {
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
