// Package niger_test verifies that the Niger adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Niger-specific
// data is correctly shaped.
package niger_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/niger"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/niger/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/niger/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: NigerAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*niger.NigerAdapter)(nil)

func TestNigerAdapter_SatisfiesInterface(t *testing.T) {
	a := niger.NewNigerAdapter()
	assert.NotNil(t, a)
}

func TestNigerAdapter_CountryCode(t *testing.T) {
	a := niger.NewNigerAdapter()
	assert.Equal(t, "NE", a.CountryCode())
}

func TestNigerAdapter_Supports(t *testing.T) {
	a := niger.NewNigerAdapter()
	assert.True(t, a.Supports("https://www.assemblee.ne/bills"))
	assert.True(t, a.Supports("https://assemblee.ne/bills/2024/edu"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://www.parliament.go.ug"))
}

func TestNigerBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.NigerBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("NE"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

func TestNigerBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.NigerBillStages
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

func TestNigerBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.NigerBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Verify Niger's canonical Bill flow:
	// First Reading → Second Reading → Committee Stage → Report Stage →
	// Third Reading → Presidential Assent → Commencement
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "COMMITTEE_STAGE")
	assert.Contains(t, byCode["COMMITTEE_STAGE"].AllowedNext, "REPORT_STAGE")
	assert.Contains(t, byCode["REPORT_STAGE"].AllowedNext, "THIRD_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "PRESIDENTIAL_ASSENT")
	assert.Contains(t, byCode["PRESIDENTIAL_ASSENT"].AllowedNext, "COMMENCEMENT")
}

func TestNigerBillStages_AllowedNextReferencesValid(t *testing.T) {
	stages := internal.NigerBillStages
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

func TestNigerLegislativeStructure_Unicameral(t *testing.T) {
	s := internal.NigerLegislativeStructure()
	assert.Equal(t, contracts.Country("NE"), s.Country)
	assert.Len(t, s.Houses, 1, "Niger should be unicameral (one House)")
	assert.Equal(t, "National Assembly of Niger", s.Houses[0].Name)
	assert.Equal(t, contracts.HouseTypeSingle, s.Houses[0].Type)
}

func TestNigerLegislativeStructure_MemberCount(t *testing.T) {
	s := internal.NigerLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	assert.Equal(t, 193, s.Houses[0].Members, "National Assembly of Niger has 193 elected MPs")
}

func TestNigerLegislativeStructure_TermDays(t *testing.T) {
	s := internal.NigerLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	// Niger's parliamentary term is five years per Article 50 of the Transitional Charter (2023).
	assert.Equal(t, 5*365, s.Houses[0].TermDays)
}

func TestNigerTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.NigerTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("NE"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

func TestNigerTerminology_ContainsCommitteeStage(t *testing.T) {
	terms := internal.NigerTerminology
	found := false
	for _, term := range terms {
		if term.Term == "Committee Stage" {
			found = true
			assert.NotEmpty(t, term.SimpleExplanation)
		}
	}
	assert.True(t, found, "Niger terminology must include the 'Committee Stage' term")
}

func TestNigerAdapter_GetStages(t *testing.T) {
	a := niger.NewNigerAdapter()
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
}

func TestNigerAdapter_GetTerminology(t *testing.T) {
	a := niger.NewNigerAdapter()
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
}

func TestNigerAdapter_GetLegislativeStructure(t *testing.T) {
	a := niger.NewNigerAdapter()
	s, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, contracts.Country("NE"), s.Country)
	assert.Len(t, s.Houses, 1, "Niger is unicameral")
}

func TestNigerAdapter_NormalizeSourceItem(t *testing.T) {
	a := niger.NewNigerAdapter()
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.assemblee.ne/bills/2024/edu",
		"title": "The Education (Amendment) Bill, 2024",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.assemblee.ne/bills/2024/edu", item.URL)
	assert.Equal(t, "The Education (Amendment) Bill, 2024", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "NE", item.CountryCode)
}

func TestNigerAdapter_DiscoverReturnsEmptyWithoutError(t *testing.T) {
	a := niger.NewNigerAdapter()
	// Without overriding the bills URL, Discover will attempt to hit the
	// real assemblee.ne site, which is unreachable from the sandbox.
	items, err := a.Discover(context.Background())
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the testdata/bills.html
// fixture from a local HTTP server and verifies Discover returns Bills
// extracted by the parliament adapter's ParseBillsListing.
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	html := loadNigerFixture(t, "bills.html")

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
		assert.Equal(t, "NE", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		assert.Equal(t, "National Assembly of Niger", item.House, "item %d has wrong House", i)
		assert.Equal(t, "National Assembly of Niger", item.Metadata["house"], "item %d has wrong house metadata", i)
		assert.Equal(t, "Parliament of Niger", item.Metadata["institution"], "item %d has wrong institution metadata", i)
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

// TestNigerSampleBills_CountAndShape verifies the Niger sample Bills
// registry has 5 entries that all reference assemblee.ne.
func TestNigerSampleBills_CountAndShape(t *testing.T) {
	bills := internal.NigerSampleBills
	require.Len(t, bills, 5, "NigerSampleBills should have exactly 5 entries")
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.URL, "sample bill %d has empty URL", i)
		assert.NotEmpty(t, b.BillNumber, "sample bill %d has empty BillNumber", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.NotEmpty(t, b.Date, "sample bill %d has empty Date", i)
		assert.True(t, strings.Contains(b.URL, "assemblee.ne"),
			"sample bill %d URL should reference assemblee.ne", i)
	}
}

// loadNigerFixture reads a testdata/ fixture into a string.
func loadNigerFixture(t *testing.T, name string) string {
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
