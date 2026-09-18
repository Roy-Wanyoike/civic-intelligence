package uganda_test

import (
        "context"
        "testing"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda/internal"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
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

// --- ENG-F1: extended interface + unicameral contract tests ----------------

// TestUgandaAdapter_SatisfiesFullInterface exercises every method of the
// contracts.LegislativeSourceAdapter interface through the adapter instance
// (the compile-time assertion in TestUgandaAdapter_SatisfiesInterface only
// guarantees method-set compatibility, not that each method is callable
// without panicking).
func TestUgandaAdapter_SatisfiesFullInterface(t *testing.T) {
        a := uganda.NewUgandaAdapter()
        require.NotNil(t, a)

        // CountryCode + Supports — pure functions, must not panic.
        assert.Equal(t, "UG", a.CountryCode())
        assert.True(t, a.Supports("https://www.parliament.go.ug/business/bills"))
        assert.False(t, a.Supports("https://example.com/"))

        // Discover — allowed to return an empty slice or an error (the live site
        // may be unreachable in CI); must NOT panic.
        items, err := a.Discover(context.Background())
        // We don't assert on err or items content — the live site may be
        // unreachable in CI. We only assert the call didn't panic.
        _ = items

        // Fetch — empty URL must error, not panic.
        _, err = a.Fetch(context.Background(), contracts.SourceItem{})
        assert.Error(t, err)

        // Parse — must not panic on an empty document.
        _, _ = a.Parse(context.Background(), contracts.RawDocument{
                URL:      "https://www.parliament.go.ug/business/bills",
                Bytes:    []byte("<html></html>"),
                MimeType: "text/html",
        })

        // NormalizeSourceItem — projects a raw map into a SourceItem.
        item, err := a.NormalizeSourceItem(map[string]any{
                "url":   "https://www.parliament.go.ug/business/bills/x",
                "title": "The Test Bill, 2024",
        })
        assert.NoError(t, err)
        assert.Equal(t, "UG", item.CountryCode)
        assert.Equal(t, "The Test Bill, 2024", item.Title)

        // GetLegislativeStructure — must return a non-nil structure.
        structure, err := a.GetLegislativeStructure(context.Background())
        assert.NoError(t, err)
        require.NotNil(t, structure)
        assert.Equal(t, contracts.Country("UG"), structure.Country)

        // GetStages — must return a non-empty slice.
        stages, err := a.GetStages(context.Background())
        assert.NoError(t, err)
        assert.NotEmpty(t, stages)

        // GetTerminology — must return a non-empty slice.
        terms, err := a.GetTerminology(context.Background())
        assert.NoError(t, err)
        assert.NotEmpty(t, terms)
}

// TestUgandaAdapter_GetStages_NoSenate verifies that the Uganda Bill stages
// contain no Senate-related stage. Uganda is UNICAMERAL — unlike Kenya, there
// is no "mediation" stage (which exists only in bicameral systems where the
// two houses must reconcile their versions of a Bill).
func TestUgandaAdapter_GetStages_NoSenate(t *testing.T) {
        a := uganda.NewUgandaAdapter()
        stages, err := a.GetStages(context.Background())
        require.NoError(t, err)

        for _, s := range stages {
                // No stage code may reference "SENATE" or "MEDIATION" — those exist
                // only in bicameral systems (e.g., Kenya).
                assert.NotContains(t, s.Code, "SENATE",
                        "Uganda is unicameral — stage %q must not reference Senate", s.Code)
                assert.NotContains(t, s.Code, "MEDIATION",
                        "Uganda is unicameral — stage %q must not reference mediation", s.Code)
                assert.NotContains(t, s.Name, "Senate",
                        "Uganda is unicameral — stage %q must not reference Senate", s.Code)
        }
}

// TestUgandaAdapter_GetLegislativeStructure_SingleHouse verifies that
// GetLegislativeStructure returns EXACTLY one house named "Parliament of
// Uganda" of type HouseTypeSingle. This is the unicameral contract: there
// must be no "Senate" or "National Assembly" (those are bicameral/Kenya terms).
func TestUgandaAdapter_GetLegislativeStructure_SingleHouse(t *testing.T) {
        a := uganda.NewUgandaAdapter()
        s, err := a.GetLegislativeStructure(context.Background())
        require.NoError(t, err)
        require.NotNil(t, s)

        assert.Len(t, s.Houses, 1, "Uganda must have exactly one House (unicameral)")

        house := s.Houses[0]
        assert.Equal(t, "Parliament of Uganda", house.Name,
                "the single House must be 'Parliament of Uganda'")
        assert.Equal(t, contracts.HouseTypeSingle, house.Type,
                "the single House must be of type HouseTypeSingle")
        assert.Equal(t, "PARLIAMENT", house.Code)

        // Explicitly verify no Senate / National Assembly leaks in.
        for _, h := range s.Houses {
                assert.NotContains(t, h.Name, "Senate",
                        "Uganda has no Senate (unicameral)")
                assert.NotEqual(t, "National Assembly", h.Name,
                        "Uganda's House is 'Parliament of Uganda', not 'National Assembly'")
        }
}

// TestUgandaSampleBills verifies the seed-data slice is well-formed and that
// every record carries the expected fields. These records are the bootstrap
// dataset used by the platform before the live crawler runs.
func TestUgandaSampleBills(t *testing.T) {
        bills := internal.UgandaSampleBills
        require.Len(t, bills, 5, "should have exactly 5 seed Bills")

        // Collect every stage code referenced by the sample Bills.
        stageCodes := map[string]bool{}
        for _, code := range []string{
                "FIRST_READING", "SECOND_READING", "COMMITTEE_STAGE",
                "REPORT_STAGE", "THIRD_READING", "PRESIDENTIAL_ASSENT",
                "COMMENCEMENT", "REJECTED",
        } {
                stageCodes[code] = true
        }

        for i, b := range bills {
                assert.NotEmpty(t, b.Title, "bill %d has empty Title", i)
                assert.NotEmpty(t, b.Number, "bill %d has empty Number", i)
                assert.Contains(t, b.Number, "Bill No.",
                        "bill %d Number should look like 'The Bill No. X of YYYY'", i)
                assert.NotEmpty(t, b.Sponsor, "bill %d has empty Sponsor", i)
                assert.True(t, stageCodes[b.Stage],
                        "bill %d has unknown stage code %q", i, b.Stage)
                assert.NotEmpty(t, b.SourceURL, "bill %d has empty SourceURL", i)
                assert.Contains(t, b.SourceURL, "parliament.go.ug",
                        "bill %d SourceURL should be on parliament.go.ug", i)
        }
}
