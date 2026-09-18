// Package rwanda_test verifies that the Rwanda adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Rwanda-specific
// data is correctly shaped.
package rwanda_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: RwandaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*rwanda.RwandaAdapter)(nil)

// Test 1: The adapter implements the interface and can be constructed.
func TestRwandaAdapter_SatisfiesInterface(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	assert.NotNil(t, a)
}

// Test 2: CountryCode returns the ISO 3166-1 alpha-2 code for Rwanda.
func TestRwandaAdapter_CountryCode(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	assert.Equal(t, "RW", a.CountryCode())
}

// Test 3: Supports recognises parliament.gov.rw URLs and rejects others.
func TestRwandaAdapter_Supports(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	assert.True(t, a.Supports("https://www.parliament.gov.rw/chamber-of-deputies/bills"))
	assert.True(t, a.Supports("https://parliament.gov.rw/senate/bills"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://parliament.go.ug"))
	assert.False(t, a.Supports("https://example.com"))
}

// Test 4: Every Rwanda stage carries the country code "RW" and the canonical
// flow stages are present.
func TestRwandaBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.RwandaBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("RW"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

// Test 5: All Rwanda stages from the task description are present.
// Canonical flow: First Reading → Committee → Second Reading → Senate Review →
// Third Reading → Presidential Assent → Commencement.
func TestRwandaBillStages_FullStageChain(t *testing.T) {
	stages := internal.RwandaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	expected := []string{
		"FIRST_READING",
		"COMMITTEE",
		"SECOND_READING",
		"SENATE_REVIEW",
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
func TestRwandaBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.RwandaBillStages
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
// known stage, and the canonical Rwanda flow is encoded correctly.
func TestRwandaBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.RwandaBillStages
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
	// Verify the canonical Rwanda Bill flow.
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "COMMITTEE")
	assert.Contains(t, byCode["COMMITTEE"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "SENATE_REVIEW")
	assert.Contains(t, byCode["SENATE_REVIEW"].AllowedNext, "THIRD_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "PRESIDENTIAL_ASSENT")
	assert.Contains(t, byCode["PRESIDENTIAL_ASSENT"].AllowedNext, "COMMENCEMENT")
}

// Test 8: Legislative structure is bicameral with the correct houses and
// member counts (Senate: 26, Chamber of Deputies: 80).
func TestRwandaLegislativeStructure_Bicameral(t *testing.T) {
	s := internal.RwandaLegislativeStructure()
	assert.Equal(t, contracts.Country("RW"), s.Country)
	assert.Equal(t, "RW", s.CountryCode)
	assert.Equal(t, "Rwanda", s.CountryName)
	assert.Len(t, s.Houses, 2, "Rwanda should be bicameral per the 2003 Constitution")
	byName := map[string]contracts.HouseDefinition{}
	for _, h := range s.Houses {
		byName[h.Name] = h
	}
	sen, ok := byName["Senate"]
	assert.True(t, ok, "missing Senate")
	assert.Equal(t, 26, sen.Members, "Senate should have 26 members")
	assert.Equal(t, contracts.HouseTypeUpper, sen.Type, "Senate is the upper house")
	cod, ok := byName["Chamber of Deputies"]
	assert.True(t, ok, "missing Chamber of Deputies")
	assert.Equal(t, 80, cod.Members, "Chamber of Deputies should have 80 members")
	assert.Equal(t, contracts.HouseTypeLower, cod.Type, "Chamber of Deputies is the lower house")
}

// Test 9: Terminology registry has at least 20 terms, each with country="RW".
func TestRwandaTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.RwandaTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("RW"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

// Test 10: Specific terms from the task description are present.
func TestRwandaTerminology_ExpectedTermsExist(t *testing.T) {
	terms := internal.RwandaTerminology
	byTerm := map[string]bool{}
	for _, term := range terms {
		byTerm[term.Term] = true
	}
	for _, want := range []string{
		"First Reading",
		"Committee",
		"Second Reading",
		"Senate Review",
		"Third Reading",
		"Presidential Assent",
		"Commencement",
		"Hansard",
		"Order Paper",
		"Chamber of Deputies",
		"Senate",
	} {
		assert.True(t, byTerm[want], "expected term %q is missing", want)
	}
}

// Test 11: Adapter's GetStages returns the same number of stages as the
// internal registry.
func TestRwandaAdapter_GetStages(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
	assert.Len(t, stages, len(internal.RwandaBillStages))
	for _, s := range stages {
		assert.Equal(t, contracts.Country("RW"), s.Country)
	}
}

// Test 12: Adapter's GetTerminology returns at least 20 terms.
func TestRwandaAdapter_GetTerminology(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
	assert.Len(t, terms, len(internal.RwandaTerminology))
	for _, tm := range terms {
		assert.Equal(t, contracts.Country("RW"), tm.Country)
	}
}

// Test 13: Adapter's GetLegislativeStructure returns the bicameral structure.
func TestRwandaAdapter_GetLegislativeStructure(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	structure, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, structure)
	assert.Equal(t, contracts.Country("RW"), structure.Country)
	assert.Len(t, structure.Houses, 2, "Rwanda should be bicameral")
}

// Test 14: NormalizeSourceItem maps a raw metadata map into a SourceItem.
func TestRwandaAdapter_NormalizeSourceItem(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.parliament.gov.rw/laws/personal-data-privacy-2021",
		"title": "Law on the Protection of Personal Data and Privacy",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.parliament.gov.rw/laws/personal-data-privacy-2021", item.URL)
	assert.Equal(t, "Law on the Protection of Personal Data and Privacy", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "RW", item.CountryCode)
}

// Test 15: Discover does not panic when invoked without a mock server.
func TestRwandaAdapter_Discover(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	items, err := a.Discover(context.Background())
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the Chamber of Deputies
// and Senate Bills fixtures from a local HTTP server and verifies the
// parliament adapter discovers Bills from BOTH chambers (Rwanda is bicameral).
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	deputiesHTML := loadRwandaFixture(t, "bills.html")
	senateHTML := loadRwandaFixture(t, "bills_senate.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/chamber-of-deputies/bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(deputiesHTML))
	})
	mux.HandleFunc("/senate/bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(senateHTML))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := parliament.NewAdapter(srv.Client(), "CivicIntelligence/0.1-test")
	a.SetDeputiesBillsURLForTest(srv.URL + "/chamber-of-deputies/bills")
	a.SetSenateBillsURLForTest(srv.URL + "/senate/bills")

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items, "Discover should return Bills from the fixtures")

	housesFound := map[string]int{}
	for i, item := range items {
		assert.Equal(t, "RW", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		switch item.House {
		case "Chamber of Deputies", "Senate":
			housesFound[item.House]++
		default:
			t.Errorf("item %d has unexpected House %q", i, item.House)
		}
		assert.Equal(t, "Parliament of Rwanda", item.Metadata["institution"],
			"item %d has wrong institution metadata", i)
	}

	// Verify BOTH chambers are represented.
	assert.Greater(t, housesFound["Chamber of Deputies"], 0,
		"should find at least one Chamber of Deputies Bill")
	assert.Greater(t, housesFound["Senate"], 0,
		"should find at least one Senate Bill")

	// Verify the sample Bills (by title) — confirms the parser extracted every
	// <div class="bill-card"> in BOTH fixtures.
	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"Law on the Protection of Personal Data and Privacy",
		"Law Governing ICT",
		"Law on Public Procurement",
		"Law on the Prevention and Punishment of Gender-Based Violence",
		"Law on the Organization of Tourism",
		"Law on Mining and Quarry Operations",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}
}

// TestRwandaSampleBills_CountAndShape verifies the Rwanda sample Bills
// registry has 5 entries with both chambers represented.
func TestRwandaSampleBills_CountAndShape(t *testing.T) {
	bills := internal.RwandaSampleBills
	require.Len(t, bills, 5, "RwandaSampleBills should have exactly 5 entries")
	deputiesCount, senateCount := 0, 0
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.SourceURL, "sample bill %d has empty SourceURL", i)
		assert.NotEmpty(t, b.Number, "sample bill %d has empty Number", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.NotEmpty(t, b.House, "sample bill %d has empty House", i)
		assert.True(t, strings.Contains(b.SourceURL, "parliament.gov.rw"),
			"sample bill %d SourceURL should reference parliament.gov.rw", i)
		switch b.House {
		case "Chamber of Deputies":
			deputiesCount++
		case "Senate":
			senateCount++
		}
	}
	assert.Greater(t, deputiesCount, 0, "should have at least one Chamber of Deputies sample Bill")
	assert.Greater(t, senateCount, 0, "should have at least one Senate sample Bill")
}

// Test 16: Fetch rejects an empty URL with a clear error.
func TestRwandaAdapter_Fetch_EmptyURL(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	_, err := a.Fetch(context.Background(), contracts.SourceItem{URL: ""})
	assert.Error(t, err)
}

// Test 17: Parse of a PDF document returns a low-confidence placeholder
// ExtractedRecord (the documents service handles full extraction).
func TestRwandaAdapter_Parse_PDF(t *testing.T) {
	a := rwanda.NewRwandaAdapter(rwanda.Dependencies{})
	recs, err := a.Parse(context.Background(), contracts.RawDocument{
		URL:      "https://www.parliament.gov.rw/laws/personal-data-privacy-2021.pdf",
		Bytes:    []byte("%PDF-1.4"),
		MimeType: "application/pdf",
	})
	assert.NoError(t, err)
	assert.Len(t, recs, 1)
	assert.Equal(t, "bill", recs[0].Kind)
	assert.Equal(t, "rwanda.parliament.PDFPlaceholder", recs[0].ExtractorName)
	assert.Less(t, recs[0].Confidence, 0.5)
}

// loadRwandaFixture reads a testdata/ fixture into a string.
func loadRwandaFixture(t *testing.T, name string) string {
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
