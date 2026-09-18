// Package senegal_test verifies that the Senegal adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Senegal-specific
// data is correctly shaped.
package senegal_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time assertion: SenegalAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*senegal.SenegalAdapter)(nil)

// Test 1: The adapter implements the interface and can be constructed.
func TestSenegalAdapter_SatisfiesInterface(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	assert.NotNil(t, a)
}

// Test 2: CountryCode returns the ISO 3166-1 alpha-2 code for Senegal.
func TestSenegalAdapter_CountryCode(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	assert.Equal(t, "SN", a.CountryCode())
}

// Test 3: Supports recognises assemblee-nationale.sn URLs and rejects others.
func TestSenegalAdapter_Supports(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	assert.True(t, a.Supports("https://www.assemblee-nationale.sn/travaux/lois"))
	assert.True(t, a.Supports("https://assemblee-nationale.sn/lois/loi-de-finances-2024"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://example.com"))
}

// Test 4: Every Senegal stage carries the country code "SN".
func TestSenegalBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.SenegalBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("SN"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

// Test 5: All Senegal stages from the task description are present, with the
// canonical flow:
//   Dépôt → Commission → Première Lecture → Deuxième Lecture →
//   Adoption → Promulgation.
//
// Stage codes are in English (for cross-country consistency); display names
// are in French (as published by the Assemblée Nationale).
func TestSenegalBillStages_FullStageChain(t *testing.T) {
	stages := internal.SenegalBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Verify every canonical stage is present.
	expected := []string{
		"DEPOT",
		"COMMISSION",
		"FIRST_READING",
		"SECOND_READING",
		"ADOPTION",
		"PROMULGATION",
		"COMMENCEMENT",
	}
	for _, code := range expected {
		s, ok := byCode[code]
		assert.True(t, ok, "missing expected stage %s", code)
		assert.NotEmpty(t, s.Name)
	}
	// Verify the French display names.
	assert.Equal(t, "Dépôt", byCode["DEPOT"].Name)
	assert.Equal(t, "Commission", byCode["COMMISSION"].Name)
	assert.Equal(t, "Première Lecture", byCode["FIRST_READING"].Name)
	assert.Equal(t, "Deuxième Lecture", byCode["SECOND_READING"].Name)
	assert.Equal(t, "Adoption", byCode["ADOPTION"].Name)
	assert.Equal(t, "Promulgation", byCode["PROMULGATION"].Name)
}

// Test 6: Terminal stages exist and have IsTerminal=true.
func TestSenegalBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.SenegalBillStages
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
// known stage, and the canonical Senegal flow is encoded correctly.
func TestSenegalBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.SenegalBillStages
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
	// Verify the canonical Senegal Bill flow (codes are in English).
	assert.Contains(t, byCode["DEPOT"].AllowedNext, "COMMISSION")
	assert.Contains(t, byCode["COMMISSION"].AllowedNext, "FIRST_READING")
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "ADOPTION")
	assert.Contains(t, byCode["ADOPTION"].AllowedNext, "PROMULGATION")
	assert.Contains(t, byCode["PROMULGATION"].AllowedNext, "COMMENCEMENT")
}

// Test 8: Legislative structure is unicameral with the correct member count
// (Assemblée Nationale du Sénégal: 165 members).
func TestSenegalLegislativeStructure_Unicameral(t *testing.T) {
	s := internal.SenegalLegislativeStructure()
	assert.Equal(t, contracts.Country("SN"), s.Country)
	assert.Equal(t, "SN", s.CountryCode)
	assert.Equal(t, "Senegal", s.CountryName)
	assert.Len(t, s.Houses, 1, "Senegal should be unicameral (one House)")
	assert.Equal(t, "Assemblée Nationale du Sénégal", s.Houses[0].Name)
	assert.Equal(t, contracts.HouseTypeSingle, s.Houses[0].Type)
}

// Test 9: Member count of the National Assembly is 165.
func TestSenegalLegislativeStructure_MemberCount(t *testing.T) {
	s := internal.SenegalLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	assert.Equal(t, 165, s.Houses[0].Members, "Assemblée Nationale du Sénégal has 165 members per Article 55")
}

// Test 10: Terminology registry has at least 20 terms, each with country="SN".
func TestSenegalTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.SenegalTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("SN"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

// Test 11: Specific terms from the task description are present.
func TestSenegalTerminology_ExpectedTermsExist(t *testing.T) {
	terms := internal.SenegalTerminology
	byTerm := map[string]bool{}
	for _, term := range terms {
		byTerm[term.Term] = true
	}
	for _, want := range []string{
		"Dépôt",
		"Commission",
		"Première Lecture",
		"Deuxième Lecture",
		"Adoption",
		"Promulgation",
		"Hansard",
		"Ordre du Jour",
		"Assemblée Nationale",
		"Loi de Finances",
	} {
		assert.True(t, byTerm[want], "expected term %q is missing", want)
	}
}

// Test 12: Adapter's GetStages returns the same number of stages as the
// internal registry.
func TestSenegalAdapter_GetStages(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
	assert.Len(t, stages, len(internal.SenegalBillStages))
	for _, s := range stages {
		assert.Equal(t, contracts.Country("SN"), s.Country)
	}
}

// Test 13: Adapter's GetTerminology returns at least 20 terms.
func TestSenegalAdapter_GetTerminology(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
	assert.Len(t, terms, len(internal.SenegalTerminology))
	for _, tm := range terms {
		assert.Equal(t, contracts.Country("SN"), tm.Country)
	}
}

// Test 14: Adapter's GetLegislativeStructure returns the unicameral structure.
func TestSenegalAdapter_GetLegislativeStructure(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	structure, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, structure)
	assert.Equal(t, contracts.Country("SN"), structure.Country)
	assert.Len(t, structure.Houses, 1, "Senegal is unicameral")
}

// Test 15: NormalizeSourceItem maps a raw metadata map into a SourceItem.
func TestSenegalAdapter_NormalizeSourceItem(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://www.assemblee-nationale.sn/travaux/lois/loi-de-finances-2024",
		"title": "Loi de finances 2024",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://www.assemblee-nationale.sn/travaux/lois/loi-de-finances-2024", item.URL)
	assert.Equal(t, "Loi de finances 2024", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "SN", item.CountryCode)
}

// Test 16: Discover does not panic when invoked without a mock server.
func TestSenegalAdapter_Discover(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	items, err := a.Discover(context.Background())
	_ = items
	_ = err
}

// TestAdapter_DiscoverBills_ViaMockServer serves the testdata/bills.html
// fixture from a local HTTP server and verifies Discover returns Bills
// extracted by the parliament adapter's ParseBillsListing.
func TestAdapter_DiscoverBills_ViaMockServer(t *testing.T) {
	html := loadSenegalFixture(t, "bills.html")

	mux := http.NewServeMux()
	mux.HandleFunc("/travaux/lois", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	a := parliament.NewAdapter(srv.Client(), "CivicIntelligence/0.1-test")
	a.SetBillsURLForTest(srv.URL + "/travaux/lois")

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items, "Discover should return Bills from the fixture")

	for i, item := range items {
		assert.Equal(t, "SN", item.CountryCode, "item %d has wrong CountryCode", i)
		assert.Equal(t, "bill", item.DocumentType, "item %d has wrong DocumentType", i)
		assert.Equal(t, contracts.SourceItemBill, item.SourceType, "item %d has wrong SourceType", i)
		assert.NotEmpty(t, item.URL, "item %d has empty URL", i)
		assert.NotEmpty(t, item.Title, "item %d has empty Title", i)
		assert.False(t, item.DiscoveredAt.IsZero(), "item %d has empty DiscoveredAt", i)
		assert.Equal(t, "Assemblée Nationale du Sénégal", item.House, "item %d has wrong House", i)
		assert.Equal(t, "Assemblée Nationale du Sénégal", item.Metadata["house"], "item %d has wrong house metadata", i)
		assert.Equal(t, "Assemblée Nationale du Sénégal", item.Metadata["institution"], "item %d has wrong institution metadata", i)
	}

	titles := map[string]bool{}
	for _, item := range items {
		titles[item.Title] = true
	}
	for _, want := range []string{
		"Loi de finances 2024",
		"Loi sur la protection des données",
		"Loi d'orientation de l'Éducation nationale",
		"Loi portant Code minier",
		"Loi sur la cybersécurité",
	} {
		assert.True(t, titles[want], "expected Bill %q in discovered items", want)
	}

	// Verify the stage mapping for the first Bill — "Adoption" → ADOPTION.
	assert.Equal(t, "ADOPTION", items[0].Metadata["stage"])
}

// TestSenegalSampleBills_CountAndShape verifies the Senegal sample Bills
// registry has 5 entries that all reference assemblee-nationale.sn.
func TestSenegalSampleBills_CountAndShape(t *testing.T) {
	bills := internal.SenegalSampleBills
	require.Len(t, bills, 5, "SenegalSampleBills should have exactly 5 entries")
	for i, b := range bills {
		assert.NotEmpty(t, b.Title, "sample bill %d has empty Title", i)
		assert.NotEmpty(t, b.SourceURL, "sample bill %d has empty SourceURL", i)
		assert.NotEmpty(t, b.Number, "sample bill %d has empty Number", i)
		assert.NotEmpty(t, b.Sponsor, "sample bill %d has empty Sponsor", i)
		assert.NotEmpty(t, b.Stage, "sample bill %d has empty Stage", i)
		assert.True(t, strings.Contains(b.SourceURL, "assemblee-nationale.sn"),
			"sample bill %d SourceURL should reference assemblee-nationale.sn", i)
	}
}

// Test 17: Fetch rejects an empty URL with a clear error.
func TestSenegalAdapter_Fetch_EmptyURL(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	_, err := a.Fetch(context.Background(), contracts.SourceItem{URL: ""})
	assert.Error(t, err)
}

// Test 18: Parse of a PDF document returns a low-confidence placeholder
// ExtractedRecord (the documents service handles full extraction).
func TestSenegalAdapter_Parse_PDF(t *testing.T) {
	a := senegal.NewSenegalAdapter(senegal.Dependencies{})
	recs, err := a.Parse(context.Background(), contracts.RawDocument{
		URL:      "https://www.assemblee-nationale.sn/travaux/lois/loi-de-finances-2024.pdf",
		Bytes:    []byte("%PDF-1.4"),
		MimeType: "application/pdf",
	})
	assert.NoError(t, err)
	assert.Len(t, recs, 1)
	assert.Equal(t, "bill", recs[0].Kind)
	assert.Equal(t, "senegal.parliament.PDFPlaceholder", recs[0].ExtractorName)
	assert.Less(t, recs[0].Confidence, 0.5)
}

// TestMapStageText_FrenchAndEnglish verifies the stage mapping accepts both
// French and English substrings. Stage codes are always English.
func TestMapStageText_FrenchAndEnglish(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Dépôt", "DEPOT"},
		{"depot", "DEPOT"},
		{"Commission", "COMMISSION"},
		{"Première Lecture", "FIRST_READING"},
		{"premiere lecture", "FIRST_READING"},
		{"First Reading", "FIRST_READING"},
		{"Deuxième Lecture", "SECOND_READING"},
		{"deuxieme lecture", "SECOND_READING"},
		{"Second Reading", "SECOND_READING"},
		{"Adoption", "ADOPTION"},
		{"Promulgation", "PROMULGATION"},
		{"Entrée en vigueur", "COMMENCEMENT"},
		{"entree en vigueur", "COMMENCEMENT"},
		{"Rejeté", "REJECTED"},
		{"Retiré", "WITHDRAWN"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, parliament.MapStageText(tc.input),
			"MapStageText(%q) should be %q", tc.input, tc.want)
	}
}

// loadSenegalFixture reads a testdata/ fixture into a string.
func loadSenegalFixture(t *testing.T, name string) string {
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
