// Package zambia implements the Zambia country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// ALL Zambia-specific knowledge lives here:
//   - The unicameral National Assembly of Zambia (167 members) per Article 63
//     of the Constitution (as amended by Act No. 2 of 2016).
//   - Zambian Bill stages (First Reading → Second Reading → Committee Stage →
//     Report Stage → Third Reading → Presidential Assent → Commencement).
//   - Zambian parliamentary terminology (Hansard, Order Paper, Caucus, etc.).
//   - Source URLs for parliament.gov.zm.
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Zambia means writing adapters/zambia/ — the legislation
// service code is unchanged.
package zambia

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// ZambiaAdapter implements contracts.LegislativeSourceAdapter.
// Compile-time assertion at the bottom guarantees interface compliance.
type ZambiaAdapter struct {
	parliament *parliament.Adapter
}

// Dependencies holds the collaborators the adapter needs.
type Dependencies struct {
	HTTPClient parliament.HTTPClient
	UserAgent  string
}

// NewZambiaAdapter constructs a Zambia adapter.
func NewZambiaAdapter(deps Dependencies) *ZambiaAdapter {
	if deps.UserAgent == "" {
		deps.UserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"
	}
	return &ZambiaAdapter{
		parliament: parliament.NewAdapter(deps.HTTPClient, deps.UserAgent),
	}
}

// CountryCode returns "ZM" — the ISO 3166-1 alpha-2 code for Zambia.
func (a *ZambiaAdapter) CountryCode() string { return "ZM" }

// Supports reports whether this adapter can handle the given URL. The Zambia
// adapter handles URLs from parliament.gov.zm.
func (a *ZambiaAdapter) Supports(url string) bool {
	return strings.Contains(url, "parliament.gov.zm")
}

// Discover delegates to the parliament source adapter.
func (a *ZambiaAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch delegates to the parliament source adapter.
func (a *ZambiaAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse delegates to the parliament source adapter.
func (a *ZambiaAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts raw Zambia-source metadata into the platform's
// canonical SourceItem shape.
func (a *ZambiaAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "ZM",
	}, nil
}

// GetLegislativeStructure returns Zambia's institutions, legislature, houses,
// and committees. This is ADAPTER DATA — these strings are values, not
// constants in the global domain model.
func (a *ZambiaAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.ZambiaLegislativeStructure()
	return &s, nil
}

// GetStages returns the Zambian Bill stages per Article 78 of the
// Constitution and the Standing Orders of the National Assembly.
func (a *ZambiaAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.ZambiaBillStages, nil
}

// GetTerminology returns the Zambian parliamentary terminology registry.
func (a *ZambiaAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.ZambiaTerminology, nil
}

// Compile-time assertion: ZambiaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*ZambiaAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
