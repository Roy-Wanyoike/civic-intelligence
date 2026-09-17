// Package ghana_test verifies that the Ghana adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Ghana-specific
// data is correctly shaped.
package ghana_test

import (
	"context"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
)

// Compile-time assertion: GhanaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*ghana.GhanaAdapter)(nil)

func TestGhanaAdapter_SatisfiesInterface(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	assert.NotNil(t, a)
}

func TestGhanaAdapter_CountryCode(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	assert.Equal(t, "GH", a.CountryCode())
}

func TestGhanaAdapter_Supports(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	assert.True(t, a.Supports("https://parliament.gh/bills"))
	assert.True(t, a.Supports("https://www.parliament.gh/bills/2024/edu"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
	assert.False(t, a.Supports("https://www.parliament.go.ug"))
}

func TestGhanaBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.GhanaBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("GH"), s.Country, "stage %q has wrong country", s.Code)
		assert.NotEmpty(t, s.Code)
		assert.NotEmpty(t, s.Name)
		assert.NotEmpty(t, s.SimpleExplanation)
	}
}

func TestGhanaBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.GhanaBillStages
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

func TestGhanaBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.GhanaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	// Verify Ghana's canonical Bill flow:
	// First Reading → Second Reading → Consideration Stage → Third Reading → Assent → Commencement
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "CONSIDERATION_STAGE")
	assert.Contains(t, byCode["CONSIDERATION_STAGE"].AllowedNext, "THIRD_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "ASSENT")
	assert.Contains(t, byCode["ASSENT"].AllowedNext, "COMMENCEMENT")
}

func TestGhanaBillStages_AllowedNextReferencesValid(t *testing.T) {
	stages := internal.GhanaBillStages
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

func TestGhanaLegislativeStructure_Unicameral(t *testing.T) {
	s := internal.GhanaLegislativeStructure()
	assert.Equal(t, contracts.Country("GH"), s.Country)
	assert.Len(t, s.Houses, 1, "Ghana should be unicameral (one House)")
	assert.Equal(t, "Parliament of Ghana", s.Houses[0].Name)
	assert.Equal(t, contracts.HouseTypeSingle, s.Houses[0].Type)
}

func TestGhanaLegislativeStructure_MemberCount(t *testing.T) {
	s := internal.GhanaLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	assert.Equal(t, 275, s.Houses[0].Members, "Parliament of Ghana has 275 elected MPs")
}

func TestGhanaLegislativeStructure_TermDays(t *testing.T) {
	s := internal.GhanaLegislativeStructure()
	assert.Len(t, s.Houses, 1)
	// Ghana's parliamentary term is four years per Article 97 of the 1992 Constitution.
	assert.Equal(t, 4*365, s.Houses[0].TermDays)
}

func TestGhanaTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.GhanaTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("GH"), term.Country, "term %q has wrong country", term.Term)
		assert.NotEmpty(t, term.Term)
		assert.NotEmpty(t, term.SimpleExplanation)
	}
}

func TestGhanaTerminology_ContainsConsiderationStage(t *testing.T) {
	terms := internal.GhanaTerminology
	found := false
	for _, term := range terms {
		if term.Term == "Consideration Stage" {
			found = true
			assert.NotEmpty(t, term.SimpleExplanation)
		}
	}
	assert.True(t, found, "Ghana terminology must include the 'Consideration Stage' term")
}

func TestGhanaAdapter_GetStages(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
}

func TestGhanaAdapter_GetTerminology(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
}

func TestGhanaAdapter_GetLegislativeStructure(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	s, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, contracts.Country("GH"), s.Country)
	assert.Len(t, s.Houses, 1, "Ghana is unicameral")
}

func TestGhanaAdapter_NormalizeSourceItem(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	item, err := a.NormalizeSourceItem(map[string]any{
		"url":   "https://parliament.gh/bills/2024/edu",
		"title": "The Education (Amendment) Bill, 2024",
	})
	assert.NoError(t, err)
	assert.Equal(t, "https://parliament.gh/bills/2024/edu", item.URL)
	assert.Equal(t, "The Education (Amendment) Bill, 2024", item.Title)
	assert.Equal(t, "bill", item.DocumentType)
	assert.Equal(t, contracts.SourceItemBill, item.SourceType)
	assert.Equal(t, "GH", item.CountryCode)
}

func TestGhanaAdapter_DiscoverReturnsEmptyWithoutError(t *testing.T) {
	a := ghana.NewGhanaAdapter()
	items, err := a.Discover(context.Background())
	assert.NoError(t, err)
	// Skeleton: discover returns an empty slice until the upstream crawler is wired.
	assert.NotNil(t, items)
}
