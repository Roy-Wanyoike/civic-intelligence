// Package egypt_test verifies that the Egypt adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Egypt-specific
// data is correctly shaped.
package egypt_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: EgyptAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*egypt.EgyptAdapter)(nil)

// Test 1: The adapter implements the interface and can be constructed.
func TestEgyptAdapter_SatisfiesInterface(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	assert.NotNil(t, a)
}

// Test 2: CountryCode returns the ISO 3166-1 alpha-2 code for Egypt.
func TestEgyptAdapter_CountryCode(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	assert.Equal(t, "EG", a.CountryCode())
}

// Test 3: Supports recognises parliament.eg URLs and rejects others.
func TestEgyptAdapter_Supports(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	assert.True(t, a.Supports("https://www.parliament.eg/house/bills"))
	assert.True(t, a.Supports("https://parliament.eg/senate/bills"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://example.com"))
}

// Test 4: Every Egypt stage carries the country code "EG".
func TestEgyptBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.EgyptBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("EG"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

// Test 5: All Egypt stages from the task description are present.
// Canonical flow: Proposal → Committee Review → First Reading → Second Reading →
// Senate Review → Third Reading → Presidential Ratification → Publication.
func TestEgyptBillStages_FullStageChain(t *testing.T) {
	stages := internal.EgyptBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	expected := []string{
		"PROPOSAL",
		"COMMITTEE_REVIEW",
		"FIRST_READING",
		"SECOND_READING",
		"SENATE_REVIEW",
		"THIRD_READING",
		"PRESIDENTIAL_RATIFICATION",
		"PUBLICATION",
	}
	for _, code := range expected {
		s, ok := byCode[code]
		assert.True(t, ok, "missing expected stage %s", code)
		assert.NotEmpty(t, s.Name)
	}
}

// Test 6: Terminal stages exist and have IsTerminal=true.
func TestEgyptBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.EgyptBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	for _, terminal := range []string{"PUBLICATION", "REJECTED", "WITHDRAWN"} {
		s, ok := byCode[terminal]
		assert.True(t, ok, "missing terminal stage %s", terminal)
		assert.True(t, s.IsTerminal, "stage %s should be terminal", terminal)
		assert.Empty(t, s.AllowedNext, "terminal stage %s should have no AllowedNext", terminal)
	}
}

// Test 7: Stage transitions are valid — every AllowedNext code references a
// known stage, and the canonical Egypt flow is encoded correctly.
func TestEgyptBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.EgyptBillStages
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
	// Verify the canonical Egypt Bill flow.
	assert.Contains(t, byCode["PROPOSAL"].AllowedNext, "COMMITTEE_REVIEW")
	assert.Contains(t, byCode["COMMITTEE_REVIEW"].AllowedNext, "FIRST_READING")
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "SENATE_REVIEW")
	assert.Contains(t, byCode["SENATE_REVIEW"].AllowedNext, "THIRD_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "PRESIDENTIAL_RATIFICATION")
	assert.Contains(t, byCode["PRESIDENTIAL_RATIFICATION"].AllowedNext, "PUBLICATION")
}

// Test 8: Legislative structure is bicameral with the correct houses and
// member counts (Senate: 300, House of Representatives: 596).
func TestEgyptLegislativeStructure_Bicameral(t *testing.T) {
	s := internal.EgyptLegislativeStructure()
	assert.Equal(t, contracts.Country("EG"), s.Country)
	assert.Equal(t, "EG", s.CountryCode)
	assert.Equal(t, "Egypt", s.CountryName)
	assert.Len(t, s.Houses, 2, "Egypt should be bicameral per the 2014 Constitution (as amended in 2019)")
	byName := map[string]contracts.HouseDefinition{}
	for _, h := range s.Houses {
		byName[h.Name] = h
	}
	sen, ok := byName["Senate"]
	assert.True(t, ok, "missing Senate")
	assert.Equal(t, 300, sen.Members, "Senate should have 300 members")
	assert.Equal(t, contracts.HouseTypeUpper, sen.Type, "Senate is the upper house")
	hor, ok := byName["House of Representatives"]
	assert.True(t, ok, "missing House of Representatives")
	assert.Equal(t, 596, hor.Members, "House of Representatives should have 596 members")
	assert.Equal(t, contracts.HouseTypeLower, hor.Type, "House of Representatives is the lower house")
}

// Test 9: Terminology registry has at least 20 terms, each with country="EG".
func TestEgyptTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.EgyptTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("EG"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

// Test 10: Specific terms from the task description are present.
func TestEgyptTerminology_ExpectedTermsExist(t *testing.T) {
	terms := internal.EgyptTerminology
	byTerm := map[string]bool{}
	for _, term := range terms {
		byTerm[term.Term] = true
	}
	for _, want := range []string{
		"Proposal",
		"Committee Review",
		"First Reading",
		"Second Reading",
		"Senate Review",
		"Third Reading",
		"Presidential Ratification",
		"Publication",
		"Hansard",
		"Order Paper",
		"House of Representatives",
		"Senate",
	} {
		assert.True(t, byTerm[want], "expected term %q is missing", want)
	}
}

// Test 11: Adapter's GetStages returns the same number of stages as the
// internal registry.
func TestEgyptAdapter_GetStages(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
	assert.Len(t, stages, len(internal.EgyptBillStages))
	for _, s := range stages {
		assert.Equal(t, contracts.Country("EG"), s.Country)
	}
}

// Test 12: Adapter's GetTerminology returns at least 20 terms.
func TestEgyptAdapter_GetTerminology(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
	assert.Len(t, terms, len(internal.EgyptTerminology))
	for _, tm := range terms {
		assert.Equal(t, contracts.Country("EG"), tm.Country)
	}
}

// Test 13: Adapter's GetLegislativeStructure returns the bicameral structure.
func TestEgyptAdapter_GetLegislativeStructure(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	structure, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, structure)
	assert.Equal(t, contracts.Country("EG"), structure.Country)
	assert.Len(t, structure.Houses, 2, "Egypt should be bicameral")
}

// Test 14: NormalizeSourceItem maps a raw metadata map into a SourceItem.
func TestEgyptAdapter_NormalizeSourceItem(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.parliament.eg/house/bills/new-investment-law-2024",
		"title": "New Investment Law",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.parliament.eg/house/bills/new-investment-law-2024", item.URL)
	assert.Equal(t, "New Investment Law", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "EG", item.CountryCode)
}

// Test 15: Discover does not panic when invoked without a mock server.
func TestEgyptAdapter_Discover(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	items, err := a.Discover(context.Background())
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the House of Representatives
// and Senate Bills fixtures from a local HTTP server and verifies the
// parliament adapter discovers Bills from BOTH chambers (Egypt is bicameral).
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	houseHTML := loadEgyptFixture(t, "bills.html")
	senateHTML := loadEgyptFixture(t, "bills_senate.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/house/bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(houseHTML))
	})
	mux.HandleFunc("/senate/bills", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(senateHTML))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := parliament.NewAdapter(srv.Client(), "CivicIntelligence/0.1-test")
	a.SetHouseBillsURLForTest(srv.URL + "/house/bills")
	a.SetSenateBillsURLForTest(srv.URL + "/senate/bills")

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items, "Discover should return Bills from the fixtures")

	housesFound := map[string]int{}
	for i, item := range items {
		assert.Equal(t, "EG", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		switch item.House {
		case "House of Representatives", "Senate":
			housesFound[item.House]++
		default:
			t.Errorf("item %d has unexpected House %q", i, item.House)
		}
		assert.Equal(t, "Egyptian Parliament", item.Metadata["institution"],
			"item %d has wrong institution metadata", i)
	}

	// Verify BOTH chambers are represented.
	assert.Greater(t, housesFound["House of Representatives"], 0,
		"should find at least one House of Representatives Bill")
	assert.Greater(t, housesFound["Senate"], 0,
		"should find at least one Senate Bill")

	// Verify the sample Bills (by title) — confirms the parser extracted every
	// <div class="bill-card"> in BOTH fixtures.
	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"New Investment Law",
		"Digital Citizenship Rights Law",
		"Personal Data Protection Law",
		"Unified Labour Law",
		"Public Universities Governance Law",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}
}

// TestEgyptSampleBills_CountAndShape verifies the Egypt sample Bills registry
// has 5 entries with both chambers represented.
func TestEgyptSampleBills_CountAndShape(t *testing.T) {
	bills := internal.EgyptSampleBills
	require.Len(t, bills, 5, "EgyptSampleBills should have exactly 5 entries")
	houseCount, senateCount := 0, 0
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.SourceURL, "sample bill %d has empty SourceURL", i)
		assert.NotEmpty(t, b.Number, "sample bill %d has empty Number", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.NotEmpty(t, b.House, "sample bill %d has empty House", i)
		assert.True(t, strings.Contains(b.SourceURL, "parliament.eg"),
			"sample bill %d SourceURL should reference parliament.eg", i)
		switch b.House {
		case "House of Representatives":
			houseCount++
		case "Senate":
			senateCount++
		}
	}
	assert.Greater(t, houseCount, 0, "should have at least one House of Representatives sample Bill")
	assert.Greater(t, senateCount, 0, "should have at least one Senate sample Bill")
}

// Test 16: Fetch rejects an empty URL with a clear error.
func TestEgyptAdapter_Fetch_EmptyURL(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	_, err := a.Fetch(context.Background(), contracts.SourceItem{URL: ""})
	assert.Error(t, err)
}

// Test 17: Parse of a PDF document returns a low-confidence placeholder
// ExtractedRecord (the documents service handles full extraction).
func TestEgyptAdapter_Parse_PDF(t *testing.T) {
	a := egypt.NewEgyptAdapter(egypt.Dependencies{})
	recs, err := a.Parse(context.Background(), contracts.RawDocument{
		URL:      "https://www.parliament.eg/house/bills/new-investment-law-2024.pdf",
		Bytes:    []byte("%PDF-1.4"),
		MimeType: "application/pdf",
	})
	assert.NoError(t, err)
	assert.Len(t, recs, 1)
	assert.Equal(t, "bill", recs[0].Kind)
	assert.Equal(t, "egypt.parliament.PDFPlaceholder", recs[0].ExtractorName)
	assert.Less(t, recs[0].Confidence, 0.5)
}

// loadEgyptFixture reads a testdata/ fixture into a string.
func loadEgyptFixture(t *testing.T, name string) string {
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
