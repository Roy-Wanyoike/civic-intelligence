// Package egypt implements the Egypt country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// ALL Egypt-specific knowledge lives here:
//   - The 2014 Constitution's bicameral Parliament (Senate + House of
//     Representatives, re-established by the 2019 amendment).
//   - Egyptian Bill stages (Proposal → Committee Review → First Reading →
//     Second Reading → Senate Review → Third Reading → Presidential
//     Ratification → Publication).
//   - Egyptian parliamentary terminology (Hansard, Order Paper, etc.).
//   - Source URLs for parliament.eg.
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Egypt means writing adapters/egypt/ — the legislation
// service code is unchanged.
package egypt

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// EgyptAdapter implements contracts.LegislativeSourceAdapter.
// Compile-time assertion at the bottom guarantees interface compliance.
type EgyptAdapter struct {
	parliament *parliament.Adapter
}

// Dependencies holds the collaborators the adapter needs.
type Dependencies struct {
	HTTPClient parliament.HTTPClient
	UserAgent  string
}

// NewEgyptAdapter constructs an Egypt adapter.
func NewEgyptAdapter(deps Dependencies) *EgyptAdapter {
	if deps.UserAgent == "" {
		deps.UserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"
	}
	return &EgyptAdapter{
		parliament: parliament.NewAdapter(deps.HTTPClient, deps.UserAgent),
	}
}

// CountryCode returns "EG" — the ISO 3166-1 alpha-2 code for Egypt.
func (a *EgyptAdapter) CountryCode() string { return "EG" }

// Supports reports whether this adapter can handle the given URL. The Egypt
// adapter handles URLs from parliament.eg.
func (a *EgyptAdapter) Supports(url string) bool {
	return strings.Contains(url, "parliament.eg")
}

// Discover delegates to the parliament source adapter.
func (a *EgyptAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch delegates to the parliament source adapter.
func (a *EgyptAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse delegates to the parliament source adapter.
func (a *EgyptAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts raw Egypt-source metadata into the platform's
// canonical SourceItem shape.
func (a *EgyptAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "EG",
	}, nil
}

// GetLegislativeStructure returns Egypt's institutions, legislature, houses,
// and committees. This is ADAPTER DATA — these strings are values, not
// constants in the global domain model.
func (a *EgyptAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.EgyptLegislativeStructure()
	return &s, nil
}

// GetStages returns the Egyptian Bill stages per the 2014 Constitution (as
// amended in 2019) and the internal regulations of each House.
func (a *EgyptAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.EgyptBillStages, nil
}

// GetTerminology returns the Egyptian parliamentary terminology registry.
func (a *EgyptAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.EgyptTerminology, nil
}

// Compile-time assertion: EgyptAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*EgyptAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
