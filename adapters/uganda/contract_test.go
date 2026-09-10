package uganda_test

import (
	"context"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
)

var _ contracts.LegislativeSourceAdapter = (*uganda.UgandaAdapter)(nil)

func TestUgandaAdapter_SatisfiesInterface(t *testing.T) {
	a := uganda.NewUgandaAdapter()
	assert.NotNil(t, a)
}

func TestUgandaAdapter_CountryCode(t *testing.T) {
	a := uganda.NewUgandaAdapter()
	assert.Equal(t, "UG", a.CountryCode())
}

func TestUgandaAdapter_Supports(t *testing.T) {
	a := uganda.NewUgandaAdapter()
	assert.True(t, a.Supports("https://www.parliament.go.ug/bills"))
	assert.False(t, a.Supports("https://parliament.go.ke"))
}

func TestUgandaBillStages_AllHaveCountry(t *testing.T) {
	stages := internal.UgandaBillStages
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		assert.Equal(t, contracts.Country("UG"), s.Country)
	}
}

func TestUgandaBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.UgandaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	assert.True(t, byCode["COMMENCEMENT"].IsTerminal)
	assert.True(t, byCode["REJECTED"].IsTerminal)
}

func TestUgandaBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.UgandaBillStages
	byCode := map[string]contracts.StageDefinition{}
	for _, s := range stages {
		byCode[s.Code] = s
	}
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "PRESIDENTIAL_ASSENT")
}

func TestUgandaLegislativeStructure_Unicameral(t *testing.T) {
	s := internal.UgandaLegislativeStructure()
	assert.Equal(t, contracts.Country("UG"), s.Country)
	assert.Len(t, s.Houses, 1, "Uganda should be unicameral (one House)")
	assert.Equal(t, "Parliament of Uganda", s.Houses[0].Name)
	assert.Equal(t, contracts.HouseTypeSingle, s.Houses[0].Type)
}

func TestUgandaTerminology_AllHaveCountry(t *testing.T) {
	terms := internal.UgandaTerminology
	assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
	for _, term := range terms {
		assert.Equal(t, contracts.Country("UG"), term.Country)
	}
}

func TestUgandaAdapter_GetStages(t *testing.T) {
	a := uganda.NewUgandaAdapter()
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
}

func TestUgandaAdapter_GetTerminology(t *testing.T) {
	a := uganda.NewUgandaAdapter()
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 20)
}

func TestUgandaAdapter_GetLegislativeStructure(t *testing.T) {
	a := uganda.NewUgandaAdapter()
	s, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.Equal(t, contracts.Country("UG"), s.Country)
	assert.Len(t, s.Houses, 1, "Uganda is unicameral")
}
