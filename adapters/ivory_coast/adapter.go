// Package ivory_coast implements the Ivory Coast country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// Ivory Coast is a PRESIDENTIAL republic with a UNICAMERAL Parliament — the
// National Assembly, with 255 members directly elected for 5-year terms.
//
// The President (Alassane Ouattara, since 2010) is both head of state and
// head of government, elected by universal suffrage. The President assents
// to Bills passed by Parliament.
//
// All Ivory Coast-specific knowledge lives here:
//   - Bill stages (First Reading → Second Reading → Committee Stage → Report
//     Stage → Third Reading → Presidential Assent → Commencement)
//   - Parliamentary terminology (National Assembly, Minister, Hansard,
//     Committee of the Whole, Order Paper, etc.)
//   - Source URLs for assemblee-nationale.ci
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Ivory Coast means writing adapters/ivory_coast/ — the legislation
// service code is unchanged.
package ivory_coast

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ivory_coast/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ivory_coast/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// IvoryCoastAdapter implements contracts.LegislativeSourceAdapter.
type IvoryCoastAdapter struct {
	parliament *parliament.Adapter
}

// NewIvoryCoastAdapter constructs a Ivory Coast adapter.
func NewIvoryCoastAdapter() *IvoryCoastAdapter {
	return &IvoryCoastAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for Ivory Coast.
func (a *IvoryCoastAdapter) CountryCode() string { return "CI" }

// Supports reports whether this adapter can handle the given URL.
// Ivory Coast's Parliament sources are served under hosts containing
// "assemblee-nationale.ci".
func (a *IvoryCoastAdapter) Supports(url string) bool {
	return strings.Contains(url, "assemblee-nationale.ci")
}

// Discover returns the items currently available from this adapter's sources.
func (a *IvoryCoastAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *IvoryCoastAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *IvoryCoastAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *IvoryCoastAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "CI",
	}, nil
}

// GetLegislativeStructure returns Ivory Coast's institutions, houses, and committees.
func (a *IvoryCoastAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.IvoryCoastLegislativeStructure()
	return &s, nil
}

// GetStages returns Ivory Coast's Bill stages (with allowed transitions).
func (a *IvoryCoastAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.IvoryCoastBillStages, nil
}

// GetTerminology returns Ivory Coast's parliamentary terminology registry.
func (a *IvoryCoastAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.IvoryCoastTerminology, nil
}

// Compile-time assertion: IvoryCoastAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*IvoryCoastAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
