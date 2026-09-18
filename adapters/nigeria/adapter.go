// Package nigeria implements the Nigerian country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// ALL Nigeria-specific knowledge lives here:
//   - The 1999 Constitution's bicameral National Assembly (House of
//     Representatives + Senate)
//   - Nigerian Bill stages (First Reading → ... → Concurrence → Assent →
//     Commencement)
//   - Nigerian parliamentary terminology (Second Reading, Public Hearing,
//     Concurrence, Hansard, Order Paper, etc.)
//   - Source URLs for nass.gov.ng
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Nigeria means writing adapters/nigeria/ — the legislation
// service code is unchanged.
package nigeria

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// NigeriaAdapter implements contracts.LegislativeSourceAdapter.
// Compile-time assertion at the bottom guarantees interface compliance.
type NigeriaAdapter struct {
	parliament *parliament.Adapter
}

// Dependencies holds the collaborators the adapter needs.
type Dependencies struct {
	HTTPClient parliament.HTTPClient
	UserAgent  string
}

// NewNigeriaAdapter constructs a Nigeria adapter.
func NewNigeriaAdapter(deps Dependencies) *NigeriaAdapter {
	if deps.UserAgent == "" {
		deps.UserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"
	}
	return &NigeriaAdapter{
		parliament: parliament.NewAdapter(deps.HTTPClient, deps.UserAgent),
	}
}

// CountryCode returns "NG" — the ISO 3166-1 alpha-2 code for Nigeria.
func (a *NigeriaAdapter) CountryCode() string { return "NG" }

// Supports reports whether this adapter can handle the given URL. The Nigeria
// adapter handles URLs from nass.gov.ng (the National Assembly of Nigeria
// portal).
func (a *NigeriaAdapter) Supports(url string) bool {
	return strings.Contains(url, "nass.gov.ng")
}

// NormalizeSourceItem converts raw Nigeria-source metadata into the platform's
// canonical SourceItem shape. Delegates to the internal normalizer which
// knows Nigeria-specific field names ("bill_no", "chamber", etc.).
func (a *NigeriaAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return internal.NormalizeSourceItem(raw)
}

// Discover delegates to the parliament source adapter.
//
// The current parliament adapter is a SKELETON (see parliament.go) and
// returns an empty slice. The structure is in place so that the ingestion
// service can be wired up; the actual nass.gov.ng HTML parsing is implemented
// in a follow-up issue.
func (a *NigeriaAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch delegates to the parliament source adapter.
func (a *NigeriaAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse delegates to the parliament source adapter.
func (a *NigeriaAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// GetLegislativeStructure returns Nigeria's institutions, legislature, houses,
// and committees. This is ADAPTER DATA — these strings are values, not
// constants in the global domain model.
func (a *NigeriaAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.NigeriaLegislativeStructure()
	return &s, nil
}

// GetStages returns the Nigerian Bill stages per the 1999 Constitution and
// the Standing Orders of the Senate and House of Representatives. The
// internal package stores these as a richer []NigeriaStage type; we project
// them down to []contracts.StageDefinition via each stage's ToContract()
// method.
func (a *NigeriaAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	stages := internal.NigeriaBillStages
	out := make([]contracts.StageDefinition, len(stages))
	for i, s := range stages {
		out[i] = s.ToContract()
	}
	return out, nil
}

// GetTerminology returns the Nigerian parliamentary terminology registry.
// Same projection pattern as GetStages.
func (a *NigeriaAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	terms := internal.NigeriaTerminology
	out := make([]contracts.TermDefinition, len(terms))
	for i, t := range terms {
		out[i] = t.ToContract()
	}
	return out, nil
}

// Compile-time assertion: NigeriaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*NigeriaAdapter)(nil)
