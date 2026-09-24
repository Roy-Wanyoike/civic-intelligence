// Package burkina_faso implements the Burkina Faso country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// Burkina Faso is a PRESIDENTIAL republic with a UNICAMERAL Parliament — the
// National Assembly, with 71 members directly elected for 5-year terms.
//
// The President (Ibrahim Traoré, since 2022) is both head of state and
// head of government, elected by universal suffrage. The President assents
// to Bills passed by Parliament.
//
// All Burkina Faso-specific knowledge lives here:
//   - Bill stages (First Reading → Second Reading → Committee Stage → Report
//     Stage → Third Reading → Presidential Assent → Commencement)
//   - Parliamentary terminology (National Assembly, Minister, Hansard,
//     Committee of the Whole, Order Paper, etc.)
//   - Source URLs for assemblee.bf
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Burkina Faso means writing adapters/burkina_faso/ — the legislation
// service code is unchanged.
package burkina_faso

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/burkina_faso/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/burkina_faso/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// BurkinaFasoAdapter implements contracts.LegislativeSourceAdapter.
type BurkinaFasoAdapter struct {
	parliament *parliament.Adapter
}

// NewBurkinaFasoAdapter constructs a Burkina Faso adapter.
func NewBurkinaFasoAdapter() *BurkinaFasoAdapter {
	return &BurkinaFasoAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for Burkina Faso.
func (a *BurkinaFasoAdapter) CountryCode() string { return "BF" }

// Supports reports whether this adapter can handle the given URL.
// Burkina Faso's Parliament sources are served under hosts containing
// "assemblee.bf".
func (a *BurkinaFasoAdapter) Supports(url string) bool {
	return strings.Contains(url, "assemblee.bf")
}

// Discover returns the items currently available from this adapter's sources.
func (a *BurkinaFasoAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *BurkinaFasoAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *BurkinaFasoAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *BurkinaFasoAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "BF",
	}, nil
}

// GetLegislativeStructure returns Burkina Faso's institutions, houses, and committees.
func (a *BurkinaFasoAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.BurkinaFasoLegislativeStructure()
	return &s, nil
}

// GetStages returns Burkina Faso's Bill stages (with allowed transitions).
func (a *BurkinaFasoAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.BurkinaFasoBillStages, nil
}

// GetTerminology returns Burkina Faso's parliamentary terminology registry.
func (a *BurkinaFasoAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.BurkinaFasoTerminology, nil
}

// Compile-time assertion: BurkinaFasoAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*BurkinaFasoAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
