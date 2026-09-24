// Package mozambique implements the Mozambique country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// Mozambique is a PRESIDENTIAL republic with a UNICAMERAL Parliament — the
// National Assembly, with 250 members directly elected for 5-year terms.
//
// The President (Filipe Nyusi, since 2015) is both head of state and
// head of government, elected by universal suffrage. The President assents
// to Bills passed by Parliament.
//
// All Mozambique-specific knowledge lives here:
//   - Bill stages (First Reading → Second Reading → Committee Stage → Report
//     Stage → Third Reading → Presidential Assent → Commencement)
//   - Parliamentary terminology (National Assembly, Minister, Hansard,
//     Committee of the Whole, Order Paper, etc.)
//   - Source URLs for parlamento.gov.mz
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Mozambique means writing adapters/mozambique/ — the legislation
// service code is unchanged.
package mozambique

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// MozambiqueAdapter implements contracts.LegislativeSourceAdapter.
type MozambiqueAdapter struct {
	parliament *parliament.Adapter
}

// NewMozambiqueAdapter constructs a Mozambique adapter.
func NewMozambiqueAdapter() *MozambiqueAdapter {
	return &MozambiqueAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for Mozambique.
func (a *MozambiqueAdapter) CountryCode() string { return "MZ" }

// Supports reports whether this adapter can handle the given URL.
// Mozambique's Parliament sources are served under hosts containing
// "parlamento.gov.mz".
func (a *MozambiqueAdapter) Supports(url string) bool {
	return strings.Contains(url, "parlamento.gov.mz")
}

// Discover returns the items currently available from this adapter's sources.
func (a *MozambiqueAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *MozambiqueAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *MozambiqueAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *MozambiqueAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "MZ",
	}, nil
}

// GetLegislativeStructure returns Mozambique's institutions, houses, and committees.
func (a *MozambiqueAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.MozambiqueLegislativeStructure()
	return &s, nil
}

// GetStages returns Mozambique's Bill stages (with allowed transitions).
func (a *MozambiqueAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.MozambiqueBillStages, nil
}

// GetTerminology returns Mozambique's parliamentary terminology registry.
func (a *MozambiqueAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.MozambiqueTerminology, nil
}

// Compile-time assertion: MozambiqueAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*MozambiqueAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
