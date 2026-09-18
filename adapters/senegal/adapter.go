// Package senegal implements the Senegal country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// ALL Senegal-specific knowledge lives here:
//   - The unicameral National Assembly of Senegal (165 members) per Article 55
//     of the Constitution (as revised in 2016 and 2019).
//   - Senegalese Bill stages (Dépôt → Commission → Première Lecture →
//     Deuxième Lecture → Adoption → Promulgation). Stage codes are in English
//     for cross-country consistency; display names are in French as published.
//   - Senegalese parliamentary terminology (Hansard, Ordonnance, etc.).
//   - Source URLs for assemblee-nationale.sn.
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Senegal means writing adapters/senegal/ — the legislation
// service code is unchanged.
package senegal

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// SenegalAdapter implements contracts.LegislativeSourceAdapter.
// Compile-time assertion at the bottom guarantees interface compliance.
type SenegalAdapter struct {
	parliament *parliament.Adapter
}

// Dependencies holds the collaborators the adapter needs.
type Dependencies struct {
	HTTPClient parliament.HTTPClient
	UserAgent  string
}

// NewSenegalAdapter constructs a Senegal adapter.
func NewSenegalAdapter(deps Dependencies) *SenegalAdapter {
	if deps.UserAgent == "" {
		deps.UserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"
	}
	return &SenegalAdapter{
		parliament: parliament.NewAdapter(deps.HTTPClient, deps.UserAgent),
	}
}

// CountryCode returns "SN" — the ISO 3166-1 alpha-2 code for Senegal.
func (a *SenegalAdapter) CountryCode() string { return "SN" }

// Supports reports whether this adapter can handle the given URL. The Senegal
// adapter handles URLs from assemblee-nationale.sn.
func (a *SenegalAdapter) Supports(url string) bool {
	return strings.Contains(url, "assemblee-nationale.sn")
}

// Discover delegates to the parliament source adapter.
func (a *SenegalAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch delegates to the parliament source adapter.
func (a *SenegalAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse delegates to the parliament source adapter.
func (a *SenegalAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts raw Senegal-source metadata into the platform's
// canonical SourceItem shape.
func (a *SenegalAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "SN",
	}, nil
}

// GetLegislativeStructure returns Senegal's institutions, legislature, houses,
// and committees. This is ADAPTER DATA — these strings are values, not
// constants in the global domain model.
func (a *SenegalAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.SenegalLegislativeStructure()
	return &s, nil
}

// GetStages returns the Senegalese Bill stages per Article 60 of the
// Constitution and the Règlement Intérieur de l'Assemblée Nationale. Stage
// codes are in English; display names are in French.
func (a *SenegalAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.SenegalBillStages, nil
}

// GetTerminology returns the Senegalese parliamentary terminology registry.
func (a *SenegalAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.SenegalTerminology, nil
}

// Compile-time assertion: SenegalAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*SenegalAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
