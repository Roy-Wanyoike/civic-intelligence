// Package dr_congo implements the DR Congo country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// DR Congo (Democratic Republic of the Congo) is a SEMI-PRESIDENTIAL republic
// with a BICAMERAL Parliament:
//   - National Assembly (lower house, 500 members, directly elected for 5-year terms)
//   - Senate (upper house, 109 members, indirectly elected by provincial
//     legislatures for 5-year terms)
//
// The President (Félix Tshisekedi, since 2019) is head of state and
// promulgates Acts. The Prime Minister is head of government, appointed by
// the President from the majority coalition in the National Assembly.
//
// All DR Congo-specific knowledge lives here:
//   - Bill stages (Proposition → Commission → Première Lecture → Lecture au
//     Sénat → Commission Mixte → Deuxième Lecture → Adoption → Promulgation)
//   - Parliamentary terminology (Assemblée Nationale, Sénat, Promulgation, etc.)
//   - Source URLs for assemblee-nationale.cd
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding DR Congo means writing adapters/dr_congo/ — the legislation
// service code is unchanged.
package dr_congo

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// DRCongoAdapter implements contracts.LegislativeSourceAdapter.
type DRCongoAdapter struct {
	parliament *parliament.Adapter
}

// NewDRCongoAdapter constructs a DR Congo adapter.
func NewDRCongoAdapter() *DRCongoAdapter {
	return &DRCongoAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for DR Congo.
func (a *DRCongoAdapter) CountryCode() string { return "CD" }

// Supports reports whether this adapter can handle the given URL.
// DR Congo's Parliament sources are served under hosts containing
// "assemblee-nationale.cd" or "senat.cd".
func (a *DRCongoAdapter) Supports(url string) bool {
	for _, host := range []string{
		"assemblee-nationale.cd",
		"senat.cd",
		"www.assemblee-nationale.cd",
		"www.senat.cd",
	} {
		if strings.Contains(url, host) {
			return true
		}
	}
	return false
}

// Discover returns the items currently available from this adapter's sources.
func (a *DRCongoAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *DRCongoAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *DRCongoAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *DRCongoAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "CD",
	}, nil
}

// GetLegislativeStructure returns DR Congo's institutions, houses, and committees.
func (a *DRCongoAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.DRCongoLegislativeStructure()
	return &s, nil
}

// GetStages returns DR Congo's Bill stages (with allowed transitions).
func (a *DRCongoAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.DRCongoBillStages, nil
}

// GetTerminology returns DR Congo's parliamentary terminology registry.
func (a *DRCongoAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.DRCongoTerminology, nil
}

// Compile-time assertion: DRCongoAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*DRCongoAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
