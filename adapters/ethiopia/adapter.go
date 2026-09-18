// Package ethiopia implements the Ethiopia country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// Ethiopia is a PARLIAMENTARY federal republic with a BICAMERAL Parliament:
//   - House of Peoples' Representatives (lower house, 547 members, directly
//     elected for 5-year terms)
//   - House of Federation (upper house, 153 members, elected by state
//     councils for 5-year terms)
//
// The Prime Minister (Abiy Ahmed, since 2018) is head of government, elected
// by the House of Peoples' Representatives. The President (Sahle-Work Zewde,
// since 2018) is ceremonial and promulgates Acts on the advice of the PM.
//
// All Ethiopia-specific knowledge lives here:
//   - Bill stages (Proposal → Committee Review → First Reading → Second
//     Reading → House of Federation Review → Final Vote → Promulgation)
//   - Parliamentary terminology (House of Peoples' Representatives, House of
//     Federation, Proclamation, etc.)
//   - Source URLs for parliament.gov.et
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Ethiopia means writing adapters/ethiopia/ — the legislation
// service code is unchanged.
package ethiopia

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// EthiopiaAdapter implements contracts.LegislativeSourceAdapter.
type EthiopiaAdapter struct {
	parliament *parliament.Adapter
}

// NewEthiopiaAdapter constructs an Ethiopia adapter.
func NewEthiopiaAdapter() *EthiopiaAdapter {
	return &EthiopiaAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for Ethiopia.
func (a *EthiopiaAdapter) CountryCode() string { return "ET" }

// Supports reports whether this adapter can handle the given URL.
// Ethiopia's Parliament sources are served under hosts containing
// "parliament.gov.et".
func (a *EthiopiaAdapter) Supports(url string) bool {
	return strings.Contains(url, "parliament.gov.et")
}

// Discover returns the items currently available from this adapter's sources.
func (a *EthiopiaAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *EthiopiaAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *EthiopiaAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *EthiopiaAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "ET",
	}, nil
}

// GetLegislativeStructure returns Ethiopia's institutions, houses, and committees.
func (a *EthiopiaAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.EthiopiaLegislativeStructure()
	return &s, nil
}

// GetStages returns Ethiopia's Bill stages (with allowed transitions).
func (a *EthiopiaAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.EthiopiaBillStages, nil
}

// GetTerminology returns Ethiopia's parliamentary terminology registry.
func (a *EthiopiaAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.EthiopiaTerminology, nil
}

// Compile-time assertion: EthiopiaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*EthiopiaAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
