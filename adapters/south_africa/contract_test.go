// Package south_africa_test verifies that the South Africa adapter satisfies
// the global contracts.LegislativeSourceAdapter interface and that all
// South Africa-specific data is correctly shaped.
package south_africa_test

import (
        "context"
        "testing"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa/internal"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa/parliament"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/stretchr/testify/assert"
)

// Compile-time assertion: SouthAfricaAdapter satisfies
// contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*south_africa.SouthAfricaAdapter)(nil)

// TestSouthAfricaAdapter_SatisfiesInterface verifies that the adapter can be
// constructed and is non-nil. The compile-time assertion above already
// guarantees interface compliance.
func TestSouthAfricaAdapter_SatisfiesInterface(t *testing.T) {
        a := south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{})
        assert.NotNil(t, a)
}

// TestSouthAfricaAdapter_CountryCode verifies the ISO 3166-1 alpha-2 code.
func TestSouthAfricaAdapter_CountryCode(t *testing.T) {
        a := south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{})
        assert.Equal(t, "ZA", a.CountryCode())
}

// TestSouthAfricaAdapter_Supports verifies URL routing for the South Africa
// adapter. It must accept URLs containing "parliament.gov.za" and reject URLs
// from other countries (parliament.go.ke, parliament.go.ug).
func TestSouthAfricaAdapter_Supports(t *testing.T) {
        a := south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{})
        assert.True(t, a.Supports("https://www.parliament.gov.za/bills-and-laws"))
        assert.True(t, a.Supports("https://parliament.gov.za/bill/42"))
        assert.False(t, a.Supports("https://parliament.go.ke/"))
        assert.False(t, a.Supports("https://www.parliament.go.ug/bills"))
}

// TestSouthAfricaBillStages_AllStagesHaveCountry verifies that every stage
// projects down to a contracts.StageDefinition with country = ZA and all
// required string fields populated.
func TestSouthAfricaBillStages_AllStagesHaveCountry(t *testing.T) {
        stages := internal.SouthAfricaBillStages // var, not func
        assert.NotEmpty(t, stages)
        for _, s := range stages {
                c := s.ToContract()
                assert.Equal(t, contracts.Country("ZA"), c.Country, "stage %q has wrong country", c.Code)
                assert.NotEmpty(t, c.Code)
                assert.NotEmpty(t, c.Name)
                assert.NotEmpty(t, c.SimpleExplanation)
        }
}

// TestSouthAfricaBillStages_TerminalStagesExist verifies that the canonical
// terminal stages (REJECTED, WITHDRAWN, LAPSED, COMMENCEMENT) are present
// and flagged as terminal with no AllowedNext.
func TestSouthAfricaBillStages_TerminalStagesExist(t *testing.T) {
        stages := internal.SouthAfricaBillStages
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

// TestSouthAfricaBillStages_StageChainIsValid verifies the canonical South
// Africa Bill flow:
//
//      INTRODUCTION → COMMITTEE → PUBLIC_PARTICIPATION → NA_VOTE →
//      NCOP_CONCURRENCE → PRESIDENTIAL_ASSENT → COMMENCEMENT
//
// Every "allowed next" must also reference an existing stage.
func TestSouthAfricaBillStages_StageChainIsValid(t *testing.T) {
        stages := internal.SouthAfricaBillStages
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
        // Verify the canonical South African Bill flow.
        assert.Contains(t, byCode["INTRODUCTION"].AllowedNext, "COMMITTEE")
        assert.Contains(t, byCode["COMMITTEE"].AllowedNext, "PUBLIC_PARTICIPATION")
        assert.Contains(t, byCode["PUBLIC_PARTICIPATION"].AllowedNext, "NA_VOTE")
        assert.Contains(t, byCode["NA_VOTE"].AllowedNext, "NCOP_CONCURRENCE")
        assert.Contains(t, byCode["NCOP_CONCURRENCE"].AllowedNext, "PRESIDENTIAL_ASSENT")
        assert.Contains(t, byCode["PRESIDENTIAL_ASSENT"].AllowedNext, "COMMENCEMENT")
}

// TestSouthAfricaBillStages_CanTransition exercises the internal CanTransition
// helper for both permitted and forbidden transitions.
func TestSouthAfricaBillStages_CanTransition(t *testing.T) {
        // Permitted forward transitions.
        assert.True(t, internal.CanTransition(internal.StageIntroduction, internal.StageCommittee))
        assert.True(t, internal.CanTransition(internal.StageNAVote, internal.StageNCOPConcurrence))
        assert.True(t, internal.CanTransition(internal.StageNCOPConcurrence, internal.StageMediation))
        assert.True(t, internal.CanTransition(internal.StageMediation, internal.StagePresidentialAssent))
        assert.True(t, internal.CanTransition(internal.StagePresidentialAssent, internal.StageCommencement))
        // Forbidden transitions (skipping a stage).
        assert.False(t, internal.CanTransition(internal.StageIntroduction, internal.StageCommencement))
        assert.False(t, internal.CanTransition(internal.StageCommittee, internal.StagePresidentialAssent))
        // Terminal stages permit no transitions.
        assert.False(t, internal.CanTransition(internal.StageCommencement, internal.StagePresidentialAssent))
        assert.False(t, internal.CanTransition(internal.StageRejected, internal.StagePresidentialAssent))
}

// TestSouthAfricaTerminology_AllHaveCountryAndAtLeastOneSource verifies that
// every terminology entry has country = ZA, a non-empty term, a non-empty
// simple explanation, and at least 20 terms are defined.
func TestSouthAfricaTerminology_AllHaveCountryAndAtLeastOneSource(t *testing.T) {
        terms := internal.SouthAfricaTerminology // var, not func
        assert.GreaterOrEqual(t, len(terms), 20, "should have at least 20 terms")
        for _, term := range terms {
                c := term.ToContract()
                assert.Equal(t, contracts.Country("ZA"), c.Country, "term %q has wrong country", c.Term)
                assert.NotEmpty(t, c.Term)
                assert.NotEmpty(t, c.SimpleExplanation)
        }
}

// TestSouthAfricaTerminology_KeyTermsPresent verifies that the most
// South-Africa-specific terms (the ones that uniquely identify this adapter)
// are present in the registry.
func TestSouthAfricaTerminology_KeyTermsPresent(t *testing.T) {
        want := []string{
                "national_assembly",
                "national_council_of_provinces",
                "ncop_concurrence",
                "section_75_bill",
                "section_76_bill",
                "provincial_mandate",
                "mediation_committee",
                "presidential_assent",
                "public_participation",
                "portfolio_committee",
        }
        for _, k := range want {
                term := internal.FindTerm(k)
                assert.NotNil(t, term, "missing key term %q", k)
        }
}

// TestSouthAfricaLegislativeStructure_Bicameral verifies that South Africa's
// Parliament is bicameral (per section 42 of the 1996 Constitution) and that
// both houses are present with their canonical member counts: NA = 400, NCOP =
// 90 (10 per province × 9 provinces).
func TestSouthAfricaLegislativeStructure_Bicameral(t *testing.T) {
        s := internal.SouthAfricaLegislativeStructure()
        assert.Equal(t, contracts.Country("ZA"), s.Country)
        assert.Len(t, s.Houses, 2, "South Africa should be bicameral per the 1996 Constitution")
        houseByCode := map[string]contracts.HouseDefinition{}
        for _, h := range s.Houses {
                houseByCode[h.Code] = h
        }
        na, ok := houseByCode[internal.HouseCodeNationalAssembly]
        assert.True(t, ok, "missing National Assembly")
        assert.Equal(t, "National Assembly", na.Name)
        assert.Equal(t, contracts.HouseTypeLower, na.Type)
        assert.Equal(t, 400, na.Members, "NA should have 400 members per section 46")
        ncop, ok := houseByCode[internal.HouseCodeNationalCouncilOfProvinces]
        assert.True(t, ok, "missing NCOP")
        assert.Equal(t, "National Council of Provinces", ncop.Name)
        assert.Equal(t, contracts.HouseTypeUpper, ncop.Type)
        assert.Equal(t, 90, ncop.Members, "NCOP should have 90 members (10 per province × 9 provinces)")
}

// TestSouthAfricaLegislativeStructure_HasCommittees verifies the structure
// includes committees from both houses.
func TestSouthAfricaLegislativeStructure_HasCommittees(t *testing.T) {
        s := internal.SouthAfricaLegislativeStructure()
        assert.NotEmpty(t, s.Committees, "committees should be defined")
        byHouse := map[string]int{}
        for _, c := range s.Committees {
                byHouse[c.House]++
        }
        assert.Greater(t, byHouse[internal.HouseCodeNationalAssembly], 0, "should have NA portfolio/standing committees")
        assert.Greater(t, byHouse[internal.HouseCodeNationalCouncilOfProvinces], 0, "should have NCOP select committees")
}

// TestSouthAfricaAdapter_GetStages verifies the top-level adapter returns the
// stages as contracts.StageDefinition.
func TestSouthAfricaAdapter_GetStages(t *testing.T) {
        a := south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{})
        stages, err := a.GetStages(context.Background())
        assert.NoError(t, err)
        assert.NotEmpty(t, stages)
        for _, s := range stages {
                assert.Equal(t, contracts.Country("ZA"), s.Country)
        }
}

// TestSouthAfricaAdapter_GetTerminology verifies the top-level adapter returns
// the terminology as contracts.TermDefinition and meets the 20-term minimum.
func TestSouthAfricaAdapter_GetTerminology(t *testing.T) {
        a := south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{})
        terms, err := a.GetTerminology(context.Background())
        assert.NoError(t, err)
        assert.GreaterOrEqual(t, len(terms), 20)
        for _, term := range terms {
                assert.Equal(t, contracts.Country("ZA"), term.Country)
        }
}

// TestSouthAfricaAdapter_GetLegislativeStructure verifies the top-level
// adapter returns a non-nil bicameral structure.
func TestSouthAfricaAdapter_GetLegislativeStructure(t *testing.T) {
        a := south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{})
        structure, err := a.GetLegislativeStructure(context.Background())
        assert.NoError(t, err)
        assert.NotNil(t, structure)
        assert.Equal(t, contracts.Country("ZA"), structure.Country)
        assert.Len(t, structure.Houses, 2, "South Africa is bicameral")
}

// TestSouthAfricaAdapter_NormalizeSourceItem_RequiredFields verifies the
// normalizer rejects records missing the required fields.
func TestSouthAfricaAdapter_NormalizeSourceItem_RequiredFields(t *testing.T) {
        a := south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{})

        _, err := a.NormalizeSourceItem(nil)
        assert.Error(t, err)

        _, err = a.NormalizeSourceItem(map[string]any{})
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "external_id")

        _, err = a.NormalizeSourceItem(map[string]any{
                "external_id": "B 12—2026",
        })
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "url")

        // Fully-populated record should normalize without error.
        item, err := a.NormalizeSourceItem(map[string]any{
                "external_id": "B 12—2026",
                "url":         "https://www.parliament.gov.za/bills-and-laws/b12-2026",
                "title":       "The National Rail Bill, 2026",
                "house":       "National Assembly",
                "published_at": "2026-03-01",
        })
        assert.NoError(t, err)
        assert.Equal(t, "ZA", item.CountryCode)
        assert.Equal(t, "B 12—2026", item.ExternalID)
        assert.Equal(t, "The National Rail Bill, 2026", item.Title)
        assert.Equal(t, internal.HouseCodeNationalAssembly, item.House)
        assert.False(t, item.PublishedAt.IsZero())
        assert.False(t, item.DiscoveredAt.IsZero())
        assert.Equal(t, "za-bill:B 12—2026", item.SourceID)
}

// TestParliamentAdapter_SupportsParliamentGovZa verifies that the parliament
// sub-adapter recognises parliament.gov.za URLs and rejects foreign
// parliamentary URLs.
func TestParliamentAdapter_SupportsParliamentGovZa(t *testing.T) {
        p := parliament.NewAdapter(nil, "")
        assert.True(t, p.Supports("https://www.parliament.gov.za/bills-and-laws"))
        assert.True(t, p.Supports("https://parliament.gov.za/anything"))
        assert.False(t, p.Supports("https://parliament.go.ke/"))
        assert.False(t, p.Supports("https://example.com/"))
}

// TestParliamentAdapter_ParseStage_MapsCanonicalStages verifies that the
// parliament adapter's stage-text mapper maps all the canonical stage labels
// published on parliament.gov.za to the platform's stage codes.
func TestParliamentAdapter_ParseStage_MapsCanonicalStages(t *testing.T) {
        p := parliament.NewAdapter(nil, "")
        ctx := context.Background()
        cases := map[string]string{
                "Introduced":                          string(internal.StageIntroduction),
                "Introduction":                        string(internal.StageIntroduction),
                "Referred to Portfolio Committee":      string(internal.StageCommittee),
                "In Committee":                        string(internal.StageCommittee),
                "Public Participation":                string(internal.StagePublicParticipation),
                "Public comment invited":              string(internal.StagePublicParticipation),
                "Passed by the National Assembly":     string(internal.StageNAVote),
                "Passed by NA":                        string(internal.StageNAVote),
                "Passed by the NCOP":                  string(internal.StageNCOPConcurrence),
                "National Council of Provinces":       string(internal.StageNCOPConcurrence),
                "In Mediation Committee":              string(internal.StageMediation),
                "Assented to":                         string(internal.StagePresidentialAssent),
                "Signed by the President":             string(internal.StagePresidentialAssent),
                "In force":                            string(internal.StageCommencement),
                "Commenced":                           string(internal.StageCommencement),
                "Rejected":                             string(internal.StageRejected),
                "Withdrawn":                            string(internal.StageWithdrawn),
                "Lapsed":                               string(internal.StageLapsed),
                "unknown status":                      "", // unknown → empty
        }
        for input, want := range cases {
                got := p.ParseStage(ctx, input)
                assert.Equal(t, want, got, "input %q mapped to %q, want %q", input, got, want)
        }
}

// TestSouthAfricaAdapter_DiscoverReturnsEmpty verifies the skeleton Discover
// returns an empty slice (not nil-error) so callers can wire it into the
// ingestion pipeline today.
func TestSouthAfricaAdapter_DiscoverReturnsEmpty(t *testing.T) {
        a := south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{})
        items, err := a.Discover(context.Background())
        assert.NoError(t, err)
        assert.Empty(t, items)
}
