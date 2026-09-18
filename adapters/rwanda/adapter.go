// Package rwanda implements the Rwanda country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// ALL Rwanda-specific knowledge lives here:
//   - The 2003 Constitution's bicameral Parliament (Senate + Chamber of
//     Deputies).
//   - Rwandan Bill stages (First Reading → Committee → Second Reading →
//     Senate Review → Third Reading → Presidential Assent → Commencement).
//   - Rwandan parliamentary terminology (Hansard, Order Paper, Bureau of
//     the Chamber, etc.).
//   - Source URLs for parliament.gov.rw.
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Rwanda means writing adapters/rwanda/ — the legislation
// service code is unchanged.
package rwanda

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// RwandaAdapter implements contracts.LegislativeSourceAdapter.
// Compile-time assertion at the bottom guarantees interface compliance.
type RwandaAdapter struct {
	parliament *parliament.Adapter
}

// Dependencies holds the collaborators the adapter needs.
type Dependencies struct {
	HTTPClient parliament.HTTPClient
	UserAgent  string
}

// NewRwandaAdapter constructs a Rwanda adapter.
func NewRwandaAdapter(deps Dependencies) *RwandaAdapter {
	if deps.UserAgent == "" {
		deps.UserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"
	}
	return &RwandaAdapter{
		parliament: parliament.NewAdapter(deps.HTTPClient, deps.UserAgent),
	}
}

// CountryCode returns "RW" — the ISO 3166-1 alpha-2 code for Rwanda.
func (a *RwandaAdapter) CountryCode() string { return "RW" }

// Supports reports whether this adapter can handle the given URL. The Rwanda
// adapter handles URLs from parliament.gov.rw.
func (a *RwandaAdapter) Supports(url string) bool {
	return strings.Contains(url, "parliament.gov.rw")
}

// Discover delegates to the parliament source adapter.
func (a *RwandaAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch delegates to the parliament source adapter.
func (a *RwandaAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse delegates to the parliament source adapter.
func (a *RwandaAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts raw Rwanda-source metadata into the platform's
// canonical SourceItem shape.
func (a *RwandaAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "RW",
	}, nil
}

// GetLegislativeStructure returns Rwanda's institutions, legislature, houses,
// and committees. This is ADAPTER DATA — these strings are values, not
// constants in the global domain model.
func (a *RwandaAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.RwandaLegislativeStructure()
	return &s, nil
}

// GetStages returns the Rwandan Bill stages per the 2003 Constitution (as
// revised in 2015) and the Rules of Procedure of the Chamber of Deputies.
func (a *RwandaAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.RwandaBillStages, nil
}

// GetTerminology returns the Rwandan parliamentary terminology registry.
func (a *RwandaAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.RwandaTerminology, nil
}

// Compile-time assertion: RwandaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*RwandaAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
