// Package cameroon implements the Cameroon country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// Cameroon is a PRESIDENTIAL republic with a UNICAMERAL Parliament — the
// National Assembly, with 180 members directly elected for 5-year terms.
//
// The President (Paul Biya, since 1982) is both head of state and
// head of government, elected by universal suffrage. The President assents
// to Bills passed by Parliament.
//
// All Cameroon-specific knowledge lives here:
//   - Bill stages (First Reading → Second Reading → Committee Stage → Report
//     Stage → Third Reading → Presidential Assent → Commencement)
//   - Parliamentary terminology (National Assembly, Minister, Hansard,
//     Committee of the Whole, Order Paper, etc.)
//   - Source URLs for parliament.cm
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Cameroon means writing adapters/cameroon/ — the legislation
// service code is unchanged.
package cameroon

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/cameroon/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/cameroon/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// CameroonAdapter implements contracts.LegislativeSourceAdapter.
type CameroonAdapter struct {
	parliament *parliament.Adapter
}

// NewCameroonAdapter constructs a Cameroon adapter.
func NewCameroonAdapter() *CameroonAdapter {
	return &CameroonAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for Cameroon.
func (a *CameroonAdapter) CountryCode() string { return "CM" }

// Supports reports whether this adapter can handle the given URL.
// Cameroon's Parliament sources are served under hosts containing
// "parliament.cm".
func (a *CameroonAdapter) Supports(url string) bool {
	return strings.Contains(url, "parliament.cm")
}

// Discover returns the items currently available from this adapter's sources.
func (a *CameroonAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *CameroonAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *CameroonAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *CameroonAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "CM",
	}, nil
}

// GetLegislativeStructure returns Cameroon's institutions, houses, and committees.
func (a *CameroonAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.CameroonLegislativeStructure()
	return &s, nil
}

// GetStages returns Cameroon's Bill stages (with allowed transitions).
func (a *CameroonAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.CameroonBillStages, nil
}

// GetTerminology returns Cameroon's parliamentary terminology registry.
func (a *CameroonAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.CameroonTerminology, nil
}

// Compile-time assertion: CameroonAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*CameroonAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
