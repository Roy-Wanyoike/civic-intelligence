package kenya

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAdapter_BillStagesReturnsKenyaLifecycle is the contract test that the
// Kenya adapter exposes the canonical Kenyan bill lifecycle to the
// platform. Any change here means the legislation service's stage validator
// will see new transitions.
func TestAdapter_BillStagesReturnsKenyaLifecycle(t *testing.T) {
	a := New()
	stages := a.BillStages()

	if assert.Len(t, stages, 12, "expected 12 Kenyan stages") {
		// First stage must be the First Reading.
		assert.Equal(t, "FIRST_READING", stages[0].Code)
		// Final positive terminal is Commencement; REJECTED/WITHDRAWN/LAPSED
		// are negative terminals. Spot-check Commencement.
		var commencement *struct {
			Code string
			Name string
		}
		for _, s := range stages {
			if s.Code == "COMMENCEMENT" {
				commencement = &struct {
					Code string
					Name string
				}{Code: s.Code, Name: s.Name}
				break
			}
		}
		if assert.NotNil(t, commencement) {
			assert.Equal(t, "Commencement", commencement.Name)
		}
		// The Second Reading stage must permit transition to Committee Stage.
		for _, s := range stages {
			if s.Code == "SECOND_READING" {
				assert.Contains(t, s.AllowedTransitions, "COMMITTEE_STAGE")
				assert.True(t, s.RequiresVote, "Second Reading requires a vote")
			}
		}
	}
}

// TestAdapter_NormalizeSourceItem_MapsKenyaFields verifies that
// country-specific raw keys are mapped into the platform's SourceItem shape
// without leaking any Kenyan strings into the output beyond the agreed
// CountryCode/House codes.
func TestAdapter_NormalizeSourceItem_MapsKenyaFields(t *testing.T) {
	a := New()
	out, err := a.NormalizeSourceItem(map[string]any{
		"external_id": "NA-BILL-2024-123",
		"title":        "The Tax Laws (Amendment) Bill, 2024",
		"url":          "https://www.parliament.go.ke/bills/NA-BILL-2024-123",
		"type":         "bill",
		"chamber":      "National Assembly",
		"published_at": "2024-03-15",
		"sponsor":      "Hon. Mover",
	})
	assert.NoError(t, err)
	assert.Equal(t, "KE", out.CountryCode)
	assert.Equal(t, "NA", out.House)
	assert.Equal(t, "NA-BILL-2024-123", out.ExternalID)
	assert.Equal(t, "The Tax Laws (Amendment) Bill, 2024", out.Title)
	assert.Equal(t, "bill", string(out.Type))
	assert.Contains(t, out.RawMetadata, "sponsor")
	assert.False(t, out.PublishedAt.IsZero(), "date should be parsed")
}

// TestAdapter_Supports verifies the URL allowlist.
func TestAdapter_Supports(t *testing.T) {
	a := New()
	cases := []struct {
		url string
		ok  bool
	}{
		{"https://www.parliament.go.ke/bills/123", true},
		{"https://parliament.go.ke/the-national-assembly", true},
		{"https://www.kenyalaw.org/kl/gazette", true},
		{"https://gazettes.africa/ke/2024", true},
		{"https://example.com/", false},
		{"", false},
		{"not a url", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.ok, a.Supports(c.url), "url=%q", c.url)
	}
}
