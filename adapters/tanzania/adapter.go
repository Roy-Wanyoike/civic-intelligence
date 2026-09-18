// Package tanzania implements the Tanzania country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// Tanzania's Parliament (Bunge la Tanzania) is UNICAMERAL — like Uganda's
// Parliament and unlike Kenya's bicameral setup. This adapter therefore
// follows the Uganda adapter pattern (single House) rather than the Kenya
// adapter pattern (National Assembly + Senate).
//
// Country code: TZ. Source domain: parliament.go.tz.
package tanzania

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// TanzaniaAdapter implements contracts.LegislativeSourceAdapter for the
// United Republic of Tanzania.
type TanzaniaAdapter struct {
	parliament *parliament.Adapter
}

// NewTanzaniaAdapter constructs a Tanzania adapter wired to the
// parliament.go.tz source adapter.
func NewTanzaniaAdapter() *TanzaniaAdapter {
	return &TanzaniaAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 code for Tanzania.
func (a *TanzaniaAdapter) CountryCode() string { return "TZ" }

// Supports reports whether this adapter can handle the given URL. The
// Tanzania adapter claims any URL on the parliament.go.tz domain.
func (a *TanzaniaAdapter) Supports(url string) bool {
	return strings.Contains(url, "parliament.go.tz")
}

// Discover delegates to the parliament.go.tz adapter.
func (a *TanzaniaAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch delegates to the parliament.go.tz adapter.
func (a *TanzaniaAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse delegates to the parliament.go.tz adapter.
func (a *TanzaniaAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts a country-specific raw metadata map into the
// platform's canonical SourceItem shape. The Tanzania adapter maps the
// "url"/"title" keys to SourceItem fields and pins the country code to TZ.
func (a *TanzaniaAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "TZ",
	}, nil
}

// GetLegislativeStructure returns Tanzania's unicameral Bunge structure.
func (a *TanzaniaAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	_ = ctx
	s := internal.TanzaniaLegislativeStructure()
	return &s, nil
}

// GetStages returns the Tanzanian Bill stages (with allowed transitions).
func (a *TanzaniaAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	_ = ctx
	return internal.TanzaniaBillStages, nil
}

// GetTerminology returns the Tanzanian parliamentary terminology registry.
func (a *TanzaniaAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	_ = ctx
	return internal.TanzaniaTerminology, nil
}

// Compile-time assertion: TanzaniaAdapter satisfies the global contract.
var _ contracts.LegislativeSourceAdapter = (*TanzaniaAdapter)(nil)

// getString is a tiny helper for safely pulling a string out of a map[string]any.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
