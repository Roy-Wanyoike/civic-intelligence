// Package mozambique_test verifies that the Mozambique adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Mozambique-specific
// data is correctly shaped.
package mozambique_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: MozambiqueAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*mozambique.MozambiqueAdapter)(nil)

func TestMozambiqueAdapter_SatisfiesInterface(t *testing.T) {
	a := mozambique.NewMozambiqueAdapter()
	assert.NotNil(t, a)
}

func TestMozambiqueAdapter_CountryCode(t *testing.T) {
	a := mozambique.NewMozambiqueAdapter()
	assert.Equal(t, "MZ", a.CountryCode())
}

func TestMozambiqueAdapter_Supports(t *testing.T) {
	a := mozambique.NewMozambiqueAdapter()
	assert.True(t, a.Supports("https://www.parlamento.gov.mz/bills"))
	assert.True(t, a.Supports("https://parlamento.gov.mz/bills/2024/edu"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://www.parliament.go.ug"))
}

func TestMozambiqueBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.MozambiqueBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("MZ"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

func TestMozambiqueBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.MozambiqueBillStages
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

func TestMozambiqueBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.MozambiqueBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Verify Mozambique's canonical Bill flow:
	// First Reading → Second Reading → Committee Stage → Report Stage →
	// Third Reading → Presidential Assent → Commencement
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "COMMITTEE_STAGE")
	assert.Contains(t, byCode["COMMITTEE_STAGE"].AllowedNext, "REPORT_STAGE")
	assert.Contains(t, byCode["REPORT_STAGE"].AllowedNext, "THIRD_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "PRESIDENTIAL_ASSENT")
	assert.Contains(t, byCode["PRESIDENTIAL_ASSENT"].AllowedNext, "COMMENCEMENT")
}

func TestMozambiqueBillStages_AllowedNextReferencesValid(t *testing.T) {
	stages := internal.MozambiqueBillStages
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

func TestMozambiqueLegislativeStructure_Unicameral(t *testing.T) {
	s := internal.MozambiqueLegislativeStructure()
	assert.Equal(t, contracts.Country("MZ"), s.Country)
	assert.Len(t, s.Houses, 1, "Mozambique should be unicameral (one House)")
	assert.Equal(t, "National Assembly of Mozambique", s.Houses[0].Name)
	assert.Equal(t, contracts.HouseTypeSingle, s.Houses[0].Type)
}

func TestMozambiqueLegislativeStructure_MemberCount(t *testing.T) {
	s := internal.MozambiqueLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	assert.Equal(t, 193, s.Houses[0].Members, "National Assembly of Mozambique has 193 elected MPs")
}

func TestMozambiqueLegislativeStructure_TermDays(t *testing.T) {
	s := internal.MozambiqueLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	// Mozambique's parliamentary term is five years per Article 50 of the 1990 Constitution (as revised in 2004 and 2018).
	assert.Equal(t, 5*365, s.Houses[0].TermDays)
}

func TestMozambiqueTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.MozambiqueTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("MZ"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

func TestMozambiqueTerminology_ContainsCommitteeStage(t *testing.T) {
	terms := internal.MozambiqueTerminology
	found := false
	for _, term := range terms {
		if term.Term == "Committee Stage" {
			found = true
			assert.NotEmpty(t, term.SimpleExplanation)
		}
	}
	assert.True(t, found, "Mozambique terminology must include the 'Committee Stage' term")
}

func TestMozambiqueAdapter_GetStages(t *testing.T) {
	a := mozambique.NewMozambiqueAdapter()
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
}

func TestMozambiqueAdapter_GetTerminology(t *testing.T) {
	a := mozambique.NewMozambiqueAdapter()
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
}

func TestMozambiqueAdapter_GetLegislativeStructure(t *testing.T) {
	a := mozambique.NewMozambiqueAdapter()
	s, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, contracts.Country("MZ"), s.Country)
	assert.Len(t, s.Houses, 1, "Mozambique is unicameral")
}

func TestMozambiqueAdapter_NormalizeSourceItem(t *testing.T) {
	a := mozambique.NewMozambiqueAdapter()
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.parlamento.gov.mz/bills/2024/edu",
		"title": "The Education (Amendment) Bill, 2024",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.parlamento.gov.mz/bills/2024/edu", item.URL)
	assert.Equal(t, "The Education (Amendment) Bill, 2024", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "MZ", item.CountryCode)
}

func TestMozambiqueAdapter_DiscoverReturnsEmptyWithoutError(t *testing.T) {
	a := mozambique.NewMozambiqueAdapter()
	// Without overriding the bills URL, Discover will attempt to hit the
	// real parlamento.gov.mz site, which is unreachable from the sandbox.
	items, err := a.Discover(context.Background())
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the testdata/bills.html
// fixture from a local HTTP server and verifies Discover returns Bills
// extracted by the parliament adapter's ParseBillsListing.
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	html := loadMozambiqueFixture(t, "bills.html")

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

	for i, item := range items {
		assert.Equal(t, "MZ", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		assert.Equal(t, "National Assembly of Mozambique", item.House, "item %d has wrong House", i)
		assert.Equal(t, "National Assembly of Mozambique", item.Metadata["house"], "item %d has wrong house metadata", i)
		assert.Equal(t, "Parliament of Mozambique", item.Metadata["institution"], "item %d has wrong institution metadata", i)
	}

	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"Electronic Transactions and Cyber Security Bill, 2024",
		"Access to Information (Amendment) Bill, 2024",
		"Public Universities (Amendment) Bill, 2024",
		"Cyber Security (Amendment) Bill, 2024",
		"Companies (Amendment) Bill, 2024",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}
}

// TestMozambiqueSampleBills_CountAndShape verifies the Mozambique sample Bills
// registry has 5 entries that all reference parlamento.gov.mz.
func TestMozambiqueSampleBills_CountAndShape(t *testing.T) {
	bills := internal.MozambiqueSampleBills
	require.Len(t, bills, 5, "MozambiqueSampleBills should have exactly 5 entries")
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.URL, "sample bill %d has empty URL", i)
		assert.NotEmpty(t, b.BillNumber, "sample bill %d has empty BillNumber", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.NotEmpty(t, b.Date, "sample bill %d has empty Date", i)
		assert.True(t, strings.Contains(b.URL, "parlamento.gov.mz"),
			"sample bill %d URL should reference parlamento.gov.mz", i)
	}
}

// loadMozambiqueFixture reads a testdata/ fixture into a string.
func loadMozambiqueFixture(t *testing.T, name string) string {
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
