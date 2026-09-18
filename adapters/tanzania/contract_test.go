// Package tanzania_test verifies that the Tanzania adapter satisfies the
// global contracts.LegislativeSourceAdapter interface and that all
// Tanzania-specific data (stages, terminology, legislative structure) is
// correctly shaped.
//
// Issue #154: at least 10 contract tests required.
package tanzania_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoke/civic-intelligence/adapters/tanzania"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: TanzaniaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*tanzania.TanzaniaAdapter)(nil)

// 1. The adapter can be constructed and satisfies the interface.
func TestTanzaniaAdapter_SatisfiesInterface(t *testing.T) {
	a := tanzania.NewTanzaniaAdapter()
	assert.NotNil(t, a)
}

// 2. CountryCode returns the ISO code for Tanzania.
func TestTanzaniaAdapter_CountryCode(t *testing.T) {
	a := tanzania.NewTanzaniaAdapter()
	assert.Equal(t, "TZ", a.CountryCode())
}

// 3. Supports claims parliament.go.tz URLs and rejects other countries' domains.
func TestTanzaniaAdapter_Supports(t *testing.T) {
	a := tanzania.NewTanzaniaAdapter()
	assert.True(t, a.Supports("https://www.parliament.go.tz/bunge/bills"))
	assert.True(t, a.Supports("https://parliament.go.tz/index.php/bills"))
	assert.False(t, a.Supports("https://www.parliament.go.ug/bills"), "Uganda URLs must not be claimed")
	assert.False(t, a.Supports("https://www.parliament.go.ke/bills-tracker"), "Kenya URLs must not be claimed")
	assert.False(t, a.Supports("https://example.com/tanzania"))
}

// 4. Every stage carries the TZ country code.
func TestTanzaniaBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.TanzaniaBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("TZ"), s.Country, "stage %q has wrong country", s.Code)
	}
}

// 5. The terminal stages COMMENCEMENT and REJECTED exist and are marked terminal.
func TestTanzaniaBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.TanzaniaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	assert.True(t, byCode["COMMENCEMENT"].IsTerminal, "COMMENCEMENT must be terminal")
	assert.True(t, byCode["REJECTED"].IsTerminal, "REJECTED must be terminal")
}

// 6. The canonical Tanzania Bill flow is intact per the #154 spec:
//    First Reading → Second Reading → Committee → Report →
//    Third Reading → Assent → Commencement
func TestTanzaniaBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.TanzaniaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Every AllowedNext must reference an existing stage.
	for _, s := range stages {
		for _, next := range s.AllowedNext {
			_, ok := byCode[next]
			assert.True(t, ok, "stage %q allows transition to unknown stage %q", s.Code, next)
		}
	}
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "COMMITTEE_STAGE")
	assert.Contains(t, byCode["COMMITTEE_STAGE"].AllowedNext, "REPORT_STAGE")
	assert.Contains(t, byCode["REPORT_STAGE"].AllowedNext, "THIRD_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "PRESIDENTIAL_ASSENT")
	assert.Contains(t, byCode["PRESIDENTIAL_ASSENT"].AllowedNext, "COMMENCEMENT")
}

// 7. The stage Names follow the issue #154 spec literally.
func TestTanzaniaBillStages_StageNamesPerSpec(t *testing.T) {
	stages := internal.TanzaniaBillStages
	byCode := map[string]string{}
	for _, s := range stages {
		byCode[s.Code] = s.Name
	}
	assert.Equal(t, "First Reading", byCode["FIRST_READING"])
	assert.Equal(t, "Second Reading", byCode["SECOND_READING"])
	assert.Equal(t, "Committee", byCode["COMMITTEE_STAGE"])
	assert.Equal(t, "Report", byCode["REPORT_STAGE"])
	assert.Equal(t, "Third Reading", byCode["THIRD_READING"])
	assert.Equal(t, "Assent", byCode["PRESIDENTIAL_ASSENT"])
	assert.Equal(t, "Commencement", byCode["COMMENCEMENT"])
}

// 8. Tanzania is unicameral — exactly one House, named Bunge la Tanzania,
//    with 393 members and the Single house type.
func TestTanzaniaLegislativeStructure_Unicameral(t *testing.T) {
	s := internal.TanzaniaLegislativeStructure()
	assert.Equal(t, contracts.Country("TZ"), s.Country)
	assert.Equal(t, "TZ", s.CountryCode)
	assert.Equal(t, "Tanzania", s.CountryName)
	assert.Len(t, s.Houses, 1, "Tanzania must be unicameral (one House)")
	assert.Equal(t, "Bunge la Tanzania", s.Houses[0].Name)
	assert.Equal(t, contracts.HouseTypeSingle, s.Houses[0].Type)
	assert.Equal(t, 393, s.Houses[0].Members, "Bunge must have 393 members per the task spec")
	assert.Equal(t, 5*365, s.Houses[0].TermDays, "Bunge term length is 5 years")
}

// 9. Every terminology entry carries the TZ country code and at least 20 exist.
func TestTanzaniaTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.TanzaniaTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "issue #154 requires at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("TZ"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term, "term has an empty Term field")
		assert.NotEmpty(t, term.SimpleExplanation, "term %q has an empty explanation", term.Term)
	}
}

// 10. Every term cites parliament.go.tz as its source, per issue #154.
func TestTanzaniaTerminology_AllHaveParliamentGoTzSource(t *testing.T) {
	terms := internal.TanzaniaTerminology
	assert.NotEmpty(t, terms)
	for _, term := range terms {
		assert.NotEmpty(t, term.Sources, "term %q has no sources", term.Term)
		found := false
		for _, src := range term.Sources {
			if strings.Contains(src, "parliament.go.tz") {
				found = true
				break
			}
		}
		assert.True(t, found, "term %q must cite parliament.go.tz", term.Term)
	}
}

// 11. The Tanzania-specific term "Bunge" is present and well-formed.
func TestTanzaniaTerminology_IncludesBungeTerm(t *testing.T) {
	terms := internal.TanzaniaTerminology
	byTerm := map[string]contracts.TermDefinition{}
	for _, term := range terms {
		byTerm[term.Term] = term
	}
	b, ok := byTerm["Bunge"]
	assert.True(t, ok, "expected a 'Bunge' terminology entry")
	assert.Equal(t, contracts.Country("TZ"), b.Country)
	assert.NotEmpty(t, b.SimpleExplanation)
}

// 12. GetStages returns the seeded stages without error.
func TestTanzaniaAdapter_GetStages(t *testing.T) {
	a := tanzania.NewTanzaniaAdapter()
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("TZ"), s.Country)
	}
}

// 13. GetTerminology returns at least 20 terms without error.
func TestTanzaniaAdapter_GetTerminology(t *testing.T) {
	a := tanzania.NewTanzaniaAdapter()
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
}

// 14. GetLegislativeStructure returns Tanzania's unicameral structure.
func TestTanzaniaAdapter_GetLegislativeStructure(t *testing.T) {
	a := tanzania.NewTanzaniaAdapter()
	s, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, contracts.Country("TZ"), s.Country)
	assert.Len(t, s.Houses, 1, "Tanzania is unicameral")
	assert.Equal(t, "Bunge la Tanzania", s.Houses[0].Name)
}

// 15. NormalizeSourceItem maps raw metadata to a canonical SourceItem pinned to TZ.
func TestTanzaniaAdapter_NormalizeSourceItem(t *testing.T) {
	a := tanzania.NewTanzaniaAdapter()
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.parliament.go.tz/bunge/bills/123",
		"title": "Written Laws (Miscellaneous Amendments) Bill, 2024",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.parliament.go.tz/bunge/bills/123", item.URL)
	assert.Equal(t, "Written Laws (Miscellaneous Amendments) Bill, 2024", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "TZ", item.CountryCode)
}

// 16. Discover returns the Bunge Bills served by a mock parliament.go.tz
//     server. The fixture (testdata/bills.html) is parsed by the parliament
//     adapter's ParseBillsListing; every returned SourceItem carries a URL,
//     a Title, the Bunge house code, and a DiscoveredAt timestamp.
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	html := loadTanzaniaFixture(t, "bills.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/bunge/bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := parliament.NewAdapter(srv.Client(), "CivicIntelligence/0.1-test")
	// Override the bills URL to point at the mock server.
	a.SetBillsURLForTest(srv.URL + "/bunge/bills")

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items, "Discover should return Bills from the fixture")

	// Verify every item carries the Tanzania country code, the Bunge house,
	// and a non-empty URL + DiscoveredAt timestamp.
	for i, item := range items {
		assert.Equal(t, "TZ", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		assert.Equal(t, "Bunge la Tanzania", item.House, "item %d has wrong House", i)
		// Metadata should record the house + institution.
		assert.Equal(t, "Bunge la Tanzania", item.Metadata["house"], "item %d has wrong house metadata", i)
		assert.Equal(t, "Parliament of Tanzania", item.Metadata["institution"], "item %d has wrong institution metadata", i)
	}

	// Verify the sample Bills are present (by title) — confirms the parser
	// extracted every <div class="bill-card"> in the fixture.
	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"The Written Laws (Miscellaneous Amendments) Act, 2023",
		"The Public Finance Act (Amendment) Bill, 2023",
		"The Local Government (Urban Authorities) Act (Amendment) Bill, 2023",
		"The Mining Act (Amendment) Bill, 2023",
		"The Electronic and Postal Communications Act (Amendment) Bill, 2023",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}
}

// 17. The Tanzania-specific sample Bills registry has 5 entries — enough to
//     seed a local fixture without touching the network.
func TestTanzaniaSampleBills_CountAndShape(t *testing.T) {
	bills := internal.TanzaniaSampleBills
	require.Len(t, bills, 5, "TanzaniaSampleBills should have exactly 5 entries")
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.URL, "sample bill %d has empty URL", i)
		assert.NotEmpty(t, b.BillNumber, "sample bill %d has empty BillNumber", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.NotEmpty(t, b.Date, "sample bill %d has empty Date", i)
		assert.True(t, strings.Contains(b.URL, "parliament.go.tz"),
			"sample bill %d URL should reference parliament.go.tz", i)
	}
}

// loadTanzaniaFixture reads a testdata/ fixture into a string.
func loadTanzaniaFixture(t *testing.T, name string) string {
	t.Helper()
	// Try the parliament sub-package's testdata/ first; fall back to the
	// tanzania package root for callers that placed fixtures there.
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
