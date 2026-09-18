// Package malawi implements the Malawi country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// Malawi is a PRESIDENTIAL republic with a UNICAMERAL Parliament — the
// National Assembly, with 193 members directly elected for 5-year terms.
//
// The President (Lazarus Chakwera, since 2020) is both head of state and
// head of government, elected by universal suffrage. The President assents
// to Bills passed by Parliament.
//
// All Malawi-specific knowledge lives here:
//   - Bill stages (First Reading → Second Reading → Committee Stage → Report
//     Stage → Third Reading → Presidential Assent → Commencement)
//   - Parliamentary terminology (National Assembly, Minister, Hansard,
//     Committee of the Whole, Order Paper, etc.)
//   - Source URLs for parliament.gov.mw
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Malawi means writing adapters/malawi/ — the legislation
// service code is unchanged.
package malawi

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/malawi/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/malawi/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// MalawiAdapter implements contracts.LegislativeSourceAdapter.
type MalawiAdapter struct {
	parliament *parliament.Adapter
}

// NewMalawiAdapter constructs a Malawi adapter.
func NewMalawiAdapter() *MalawiAdapter {
	return &MalawiAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for Malawi.
func (a *MalawiAdapter) CountryCode() string { return "MW" }

// Supports reports whether this adapter can handle the given URL.
// Malawi's Parliament sources are served under hosts containing
// "parliament.gov.mw".
func (a *MalawiAdapter) Supports(url string) bool {
	return strings.Contains(url, "parliament.gov.mw")
}

// Discover returns the items currently available from this adapter's sources.
func (a *MalawiAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *MalawiAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *MalawiAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *MalawiAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "MW",
	}, nil
}

// GetLegislativeStructure returns Malawi's institutions, houses, and committees.
func (a *MalawiAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.MalawiLegislativeStructure()
	return &s, nil
}

// GetStages returns Malawi's Bill stages (with allowed transitions).
func (a *MalawiAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.MalawiBillStages, nil
}

// GetTerminology returns Malawi's parliamentary terminology registry.
func (a *MalawiAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.MalawiTerminology, nil
}

// Compile-time assertion: MalawiAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*MalawiAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
