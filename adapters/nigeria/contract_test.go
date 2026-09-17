// Package nigeria_test verifies that the Nigeria adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Nigeria-specific
// data is correctly shaped.
package nigeria_test

import (
	"context"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
)

// Compile-time assertion: NigeriaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*nigeria.NigeriaAdapter)(nil)

// Test 1: The adapter implements the interface and can be constructed.
func TestNigeriaAdapter_SatisfiesInterface(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	assert.NotNil(t, a)
}

// Test 2: CountryCode returns the ISO 3166-1 alpha-2 code for Nigeria.
func TestNigeriaAdapter_CountryCode(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	assert.Equal(t, "NG", a.CountryCode())
}

// Test 3: Supports recognises nass.gov.ng URLs and rejects others.
func TestNigeriaAdapter_Supports(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	assert.True(t, a.Supports("https://nass.gov.ng/house/bills"))
	assert.True(t, a.Supports("https://www.nass.gov.ng/senate/bills/HB-2024-012"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://parliament.go.ug"))
	assert.False(t, a.Supports("https://example.com"))
}

// Test 4: Every Nigeria stage carries the country code "NG" and the canonical
// flow stages (FIRST_READING, SECOND_READING, THIRD_READING, ASSENT,
// COMMENCEMENT) are present.
func TestNigeriaBillStages_AllStagesHaveCountry(t *testing.T) {
	stages := internal.NigeriaBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		// Project to the contracts type to verify the global contract is met.
		c := s.ToContract()
		assert.Equal(t, contracts.Country("NG"), c.Country, "stage %q has wrong country", c.Code)
		assert.NotEmpty(t, c.Code)
		assert.NotEmpty(t, c.Name)
		assert.NotEmpty(t, c.SimpleExplanation)
	}
}

// Test 5: All Nigeria stages from the task description are present in the
// expected order:
//   First Reading → Second Reading → Public Hearing → Committee → Report →
//   Third Reading → Concurrence → Assent → Commencement.
func TestNigeriaBillStages_FullStageChain(t *testing.T) {
	stages := internal.NigeriaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[string(s.Code)] = s.ToContract()
	}
	expected := []string{
		"FIRST_READING",
		"SECOND_READING",
		"PUBLIC_HEARING",
		"COMMITTEE",
		"REPORT",
		"THIRD_READING",
		"CONCURRENCE",
		"ASSENT",
		"COMMENCEMENT",
	}
	for _, code := range expected {
		s, ok := byCode[code]
		assert.True(t, ok, "missing expected stage %s", code)
		assert.NotEmpty(t, s.Name)
	}
}

// Test 6: Terminal stages exist and have empty AllowedNext + IsTerminal=true.
func TestNigeriaBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.NigeriaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[string(s.Code)] = s.ToContract()
	}
	for _, terminal := range []string{"REJECTED", "WITHDRAWN", "LAPSED", "COMMENCEMENT"} {
		s, ok := byCode[terminal]
		assert.True(t, ok, "missing terminal stage %s", terminal)
		assert.True(t, s.IsTerminal, "stage %s should be terminal", terminal)
		assert.Empty(t, s.AllowedNext, "terminal stage %s should have no AllowedNext", terminal)
	}
}

// Test 7: Stage transitions are valid — every AllowedNext code references a
// known stage, and the canonical Nigeria flow is encoded correctly.
func TestNigeriaBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.NigeriaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[string(s.Code)] = s.ToContract()
	}
	// Every "allowed next" must reference an existing stage.
	for _, s := range stages {
		for _, next := range s.AllowedNext {
			nextStr := string(next)
			_, ok := byCode[nextStr]
			assert.True(t, ok, "stage %q allows transition to unknown stage %q", s.Code, nextStr)
		}
	}
	// Verify the canonical Nigeria Bill flow.
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "PUBLIC_HEARING")
	assert.Contains(t, byCode["PUBLIC_HEARING"].AllowedNext, "COMMITTEE")
	assert.Contains(t, byCode["COMMITTEE"].AllowedNext, "REPORT")
	assert.Contains(t, byCode["REPORT"].AllowedNext, "THIRD_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "CONCURRENCE")
	assert.Contains(t, byCode["CONCURRENCE"].AllowedNext, "ASSENT")
	assert.Contains(t, byCode["ASSENT"].AllowedNext, "COMMENCEMENT")
}

// Test 8: Stage ordering is monotonically increasing in the canonical flow
// (FIRST_READING has the lowest Order; COMMENCEMENT has the highest among
// the success path).
func TestNigeriaBillStages_OrderIsMonotonic(t *testing.T) {
	stages := internal.NigeriaBillStages
	byCode := map[string]int{}
	for _, s := range stages {
		byCode[string(s.Code)] = s.Order
	}
	// The success path's Order must be strictly increasing.
	successPath := []string{
		"FIRST_READING",
		"SECOND_READING",
		"PUBLIC_HEARING",
		"COMMITTEE",
		"REPORT",
		"THIRD_READING",
		"CONCURRENCE",
		"ASSENT",
		"COMMENCEMENT",
	}
	for i := 0; i+1 < len(successPath); i++ {
		assert.Less(t,
			byCode[successPath[i]],
			byCode[successPath[i+1]],
			"%s should be ordered before %s", successPath[i], successPath[i+1],
		)
	}
	// Terminal failure stages come after COMMENCEMENT.
	assert.Greater(t, byCode["REJECTED"], byCode["COMMENCEMENT"])
	assert.Greater(t, byCode["WITHDRAWN"], byCode["COMMENCEMENT"])
	assert.Greater(t, byCode["LAPSED"], byCode["COMMENCEMENT"])
}

// Test 9: Legislative structure is bicameral with the correct houses and
// member counts (House of Representatives: 360, Senate: 109).
func TestNigeriaLegislativeStructure_Bicameral(t *testing.T) {
	s := internal.NigeriaLegislativeStructure()
	assert.Equal(t, contracts.Country("NG"), s.Country)
	assert.Equal(t, "NG", s.CountryCode)
	assert.Equal(t, "Nigeria", s.CountryName)
	assert.Len(t, s.Houses, 2, "Nigeria should be bicameral per the 1999 Constitution")
	byName := map[string]contracts.HouseDefinition{}
	for _, h := range s.Houses {
		byName[h.Name] = h
	}
	hor, ok := byName["House of Representatives"]
	assert.True(t, ok, "missing House of Representatives")
	assert.Equal(t, 360, hor.Members, "House of Representatives should have 360 members")
	assert.Equal(t, contracts.HouseTypeLower, hor.Type, "House of Representatives is the lower house")
	sen, ok := byName["Senate"]
	assert.True(t, ok, "missing Senate")
	assert.Equal(t, 109, sen.Members, "Senate should have 109 members")
	assert.Equal(t, contracts.HouseTypeUpper, sen.Type, "Senate is the upper house")
}

// Test 10: Every committee belongs to one of the two houses and has a name.
func TestNigeriaLegislativeStructure_CommitteesValid(t *testing.T) {
	s := internal.NigeriaLegislativeStructure()
	assert.NotEmpty(t, s.Committees, "Nigeria should have standing committees")
	knownHouseCodes := map[string]bool{
		internal.HouseCodeHouseOfReps: true,
		internal.HouseCodeSenate:      true,
	}
	for _, c := range s.Committees {
		assert.NotEmpty(t, c.Code, "committee should have a code")
		assert.NotEmpty(t, c.Name, "committee should have a name")
		assert.True(t, knownHouseCodes[c.House], "committee %q has unknown house %q", c.Name, c.House)
		assert.Greater(t, c.Members, 0, "committee %q should have positive members", c.Name)
	}
}

// Test 11: Terminology registry has at least 20 terms, each with country="NG".
func TestNigeriaTerminology_AllHaveCountryAndAtLeastOneSource(t *testing.T) {
	terms := internal.NigeriaTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		c := term.ToContract()
		assert.Equal(t, contracts.Country("NG"), c.Country, "term %q has wrong country", c.Term)
		assert.NotEmpty(t, c.Term)
		assert.NotEmpty(t, c.SimpleExplanation)
		assert.NotEmpty(t, c.CanonicalKey)
	}
}

// Test 12: Specific terms from the task description are present.
func TestNigeriaTerminology_ExpectedTermsExist(t *testing.T) {
	terms := internal.NigeriaTerminology
	byKey := map[string]*internal.NigeriaTerm{}
	for i := range terms {
		byKey[terms[i].CanonicalKey] = &terms[i]
	}
	expectedKeys := []string{
		"first_reading",
		"second_reading",
		"public_hearing",
		"committee",
		"report",
		"third_reading",
		"concurrence",
		"assent",
		"commencement",
		"hansard",
		"order_paper",
	}
	for _, k := range expectedKeys {
		tm, ok := byKey[k]
		assert.True(t, ok, "expected term with key %q is missing", k)
		if ok {
			assert.NotEmpty(t, tm.Term)
		}
	}
}

// Test 13: Adapter's GetStages returns the same number of stages as the
// internal registry.
func TestNigeriaAdapter_GetStages(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
	assert.Len(t, stages, len(internal.NigeriaBillStages))
	for _, s := range stages {
		assert.Equal(t, contracts.Country("NG"), s.Country)
	}
}

// Test 14: Adapter's GetTerminology returns the same number of terms as the
// internal registry (>= 20).
func TestNigeriaAdapter_GetTerminology(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
	assert.Len(t, terms, len(internal.NigeriaTerminology))
	for _, tm := range terms {
		assert.Equal(t, contracts.Country("NG"), tm.Country)
	}
}

// Test 15: Adapter's GetLegislativeStructure returns the bicameral structure.
func TestNigeriaAdapter_GetLegislativeStructure(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	structure, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, structure)
	assert.Equal(t, contracts.Country("NG"), structure.Country)
	assert.Len(t, structure.Houses, 2, "Nigeria should be bicameral")
}

// Test 16: NormalizeSourceItem rejects records with missing required fields.
func TestNigeriaAdapter_NormalizeSourceItem_RequiredFields(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	// Missing url and title.
	_, err := a.NormalizeSourceItem(map[string]any{
		"external_id": "HB-2024-012",
	})
	assert.Error(t, err)

	// Missing url only.
	_, err = a.NormalizeSourceItem(map[string]any{
		"external_id": "HB-2024-012",
		"title":       "Some Bill",
	})
	assert.Error(t, err)

	// Missing external_id only.
	_, err = a.NormalizeSourceItem(map[string]any{
		"url":   "https://nass.gov.ng/bill/HB-2024-012",
		"title": "Some Bill",
	})
	assert.Error(t, err)

	// All required fields present.
	item, err := a.NormalizeSourceItem(map[string]any{
		"external_id": "HB-2024-012",
		"url":         "https://nass.gov.ng/bill/HB-2024-012",
		"title":       "Some Bill",
	})
	assert.NoError(t, err)
	assert.Equal(t, "NG", item.CountryCode)
	assert.Equal(t, "HB-2024-012", item.ExternalID)
	assert.Equal(t, "Some Bill", item.Title)
}

// Test 17: NormalizeSourceItem maps the "House of Representatives" chamber
// string to the canonical HOR house code, and "Senate" to SEN.
func TestNigeriaAdapter_NormalizeSourceItem_HouseMapping(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	item, err := a.NormalizeSourceItem(map[string]any{
		"external_id": "HB-2024-012",
		"url":         "https://nass.gov.ng/house/bill/HB-2024-012",
		"title":       "Some House Bill",
		"chamber":     "House of Representatives",
	})
	assert.NoError(t, err)
	assert.Equal(t, internal.HouseCodeHouseOfReps, item.House)

	item2, err := a.NormalizeSourceItem(map[string]any{
		"external_id": "SB-2024-008",
		"url":         "https://nass.gov.ng/senate/bill/SB-2024-008",
		"title":       "Some Senate Bill",
		"house":       "Senate",
	})
	assert.NoError(t, err)
	assert.Equal(t, internal.HouseCodeSenate, item2.House)
}

// Test 18: Discover returns an empty slice without error (skeleton behaviour).
func TestNigeriaAdapter_Discover(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	items, err := a.Discover(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, items)
	// Skeleton returns an empty slice; full crawling is a follow-up issue.
	assert.Empty(t, items)
}

// Test 19: Fetch rejects an empty URL with a clear error.
func TestNigeriaAdapter_Fetch_EmptyURL(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	_, err := a.Fetch(context.Background(), contracts.SourceItem{URL: ""})
	assert.Error(t, err)
}

// Test 20: Parse of a PDF document returns a low-confidence placeholder
// ExtractedRecord (the documents service handles full extraction).
func TestNigeriaAdapter_Parse_PDF(t *testing.T) {
	a := nigeria.NewNigeriaAdapter(nigeria.Dependencies{})
	recs, err := a.Parse(context.Background(), contracts.RawDocument{
		URL:      "https://nass.gov.ng/documents/HB-2024-012.pdf",
		Bytes:    []byte("%PDF-1.4"),
		MimeType: "application/pdf",
	})
	assert.NoError(t, err)
	assert.Len(t, recs, 1)
	assert.Equal(t, "bill", recs[0].Kind)
	assert.Equal(t, "nigeria.parliament.PDFPlaceholder", recs[0].ExtractorName)
	assert.Less(t, recs[0].Confidence, 0.5)
}

// Test 21: FindStage / IsTerminal / CanTransition behave consistently with
// the NigeriaBillStages slice.
func TestNigeriaStage_Helpers(t *testing.T) {
	// FindStage: known code returns non-nil; unknown returns nil.
	s := internal.FindStage(internal.StageSecondReading)
	assert.NotNil(t, s)
	assert.Equal(t, "Second Reading", s.Name)
	assert.Nil(t, internal.FindStage(internal.StageCode("NON_EXISTENT_STAGE")))

	// IsTerminal: COMMENCEMENT is terminal; SECOND_READING is not.
	assert.True(t, internal.IsTerminal(internal.StageCommencement))
	assert.True(t, internal.IsTerminal(internal.StageRejected))
	assert.False(t, internal.IsTerminal(internal.StageSecondReading))

	// CanTransition: FIRST_READING -> SECOND_READING is permitted.
	assert.True(t, internal.CanTransition(internal.StageFirstReading, internal.StageSecondReading))
	// FIRST_READING -> ASSENT is NOT permitted.
	assert.False(t, internal.CanTransition(internal.StageFirstReading, internal.StageAssent))
	// From an unknown code, nothing is permitted.
	assert.False(t, internal.CanTransition(internal.StageCode("UNKNOWN"), internal.StageSecondReading))
}
