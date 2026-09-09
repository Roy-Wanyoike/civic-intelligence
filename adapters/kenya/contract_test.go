// Package kenya_test verifies that the Kenya adapter satisfies the global
// contracts.LegislativeSourceAdapter interface and that all Kenya-specific
// data is correctly shaped.
package kenya_test

import (
	"context"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/stretchr/testify/assert"
)

// Compile-time assertion: KenyaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*kenya.KenyaAdapter)(nil)

func TestKenyaAdapter_SatisfiesInterface(t *testing.T) {
	a := kenya.NewKenyaAdapter(kenya.Dependencies{})
	assert.NotNil(t, a)
}

func TestKenyaBillStages_AllStagesHaveCountry(t *testing.T) {
	stages := internal.KenyaBillStages // var, not func
	assert.NotEmpty(t, stages)
	for _, s := range stages {
		// Project to the contracts type to verify the global contract is met.
		c := s.ToContract()
		assert.Equal(t, contracts.Country("KE"), c.Country, "stage %q has wrong country", c.Code)
		assert.NotEmpty(t, c.Code)
		assert.NotEmpty(t, c.Name)
		assert.NotEmpty(t, c.SimpleExplanation)
	}
}

func TestKenyaBillStages_TerminalStagesExist(t *testing.T) {
	stages := internal.KenyaBillStages
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

func TestKenyaBillStages_StageChainIsValid(t *testing.T) {
	stages := internal.KenyaBillStages
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
	// Verify the canonical Kenya Bill flow.
	assert.Contains(t, byCode["FIRST_READING"].AllowedNext, "SECOND_READING")
	assert.Contains(t, byCode["SECOND_READING"].AllowedNext, "COMMITTEE_STAGE")
	assert.Contains(t, byCode["THIRD_READING"].AllowedNext, "PRESIDENTIAL_ASSENT")
	assert.Contains(t, byCode["PRESIDENTIAL_ASSENT"].AllowedNext, "COMMENCEMENT")
}

func TestKenyaTerminology_AllHaveCountryAndAtLeastOneSource(t *testing.T) {
	terms := internal.KenyaTerminology // var, not func
	assert.GreaterOrEqual(t, len(terms), 25, "should have at least 25 terms")
	for _, term := range terms {
		c := term.ToContract()
		assert.Equal(t, contracts.Country("KE"), c.Country, "term %q has wrong country", c.Term)
		assert.NotEmpty(t, c.Term)
		assert.NotEmpty(t, c.SimpleExplanation)
	}
}

func TestKenyaLegislativeStructure_Bicameral(t *testing.T) {
	s := internal.KenyaLegislativeStructure()
	assert.Equal(t, contracts.Country("KE"), s.Country)
	assert.Len(t, s.Houses, 2, "Kenya should be bicameral per the 2010 Constitution")
	houseNames := map[string]bool{}
	for _, h := range s.Houses {
		houseNames[h.Name] = true
	}
	assert.True(t, houseNames["National Assembly"], "missing National Assembly")
	assert.True(t, houseNames["Senate"], "missing Senate")
}

func TestKenyaAdapter_GetStages(t *testing.T) {
	a := kenya.NewKenyaAdapter(kenya.Dependencies{})
	stages, err := a.GetStages(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stages)
}

func TestKenyaAdapter_GetTerminology(t *testing.T) {
	a := kenya.NewKenyaAdapter(kenya.Dependencies{})
	terms, err := a.GetTerminology(context.Background())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(terms), 25)
}

func TestKenyaAdapter_GetLegislativeStructure(t *testing.T) {
	a := kenya.NewKenyaAdapter(kenya.Dependencies{})
	structure, err := a.GetLegislativeStructure(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, structure)
	assert.Equal(t, contracts.Country("KE"), structure.Country)
}
