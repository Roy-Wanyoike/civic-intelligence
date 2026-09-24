// Package niger implements the Niger country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// Niger is a PRESIDENTIAL republic with a UNICAMERAL Parliament — the
// National Assembly, with 171 members directly elected for 5-year terms.
//
// The President (Abdourahamane Tchiani, since 2023) is both head of state and
// head of government, elected by universal suffrage. The President assents
// to Bills passed by Parliament.
//
// All Niger-specific knowledge lives here:
//   - Bill stages (First Reading → Second Reading → Committee Stage → Report
//     Stage → Third Reading → Presidential Assent → Commencement)
//   - Parliamentary terminology (National Assembly, Minister, Hansard,
//     Committee of the Whole, Order Paper, etc.)
//   - Source URLs for assemblee.ne
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Niger means writing adapters/niger/ — the legislation
// service code is unchanged.
package niger

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/niger/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/niger/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// NigerAdapter implements contracts.LegislativeSourceAdapter.
type NigerAdapter struct {
	parliament *parliament.Adapter
}

// NewNigerAdapter constructs a Niger adapter.
func NewNigerAdapter() *NigerAdapter {
	return &NigerAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for Niger.
func (a *NigerAdapter) CountryCode() string { return "NE" }

// Supports reports whether this adapter can handle the given URL.
// Niger's Parliament sources are served under hosts containing
// "assemblee.ne".
func (a *NigerAdapter) Supports(url string) bool {
	return strings.Contains(url, "assemblee.ne")
}

// Discover returns the items currently available from this adapter's sources.
func (a *NigerAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *NigerAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *NigerAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *NigerAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "NE",
	}, nil
}

// GetLegislativeStructure returns Niger's institutions, houses, and committees.
func (a *NigerAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.NigerLegislativeStructure()
	return &s, nil
}

// GetStages returns Niger's Bill stages (with allowed transitions).
func (a *NigerAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.NigerBillStages, nil
}

// GetTerminology returns Niger's parliamentary terminology registry.
func (a *NigerAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.NigerTerminology, nil
}

// Compile-time assertion: NigerAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*NigerAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
