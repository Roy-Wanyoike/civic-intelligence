// Package kenya implements the Kenyan country adapter for the Civic Intelligence
// Platform. It implements contracts.LegislativeSourceAdapter.
//
// ALL Kenya-specific knowledge lives here:
//   - The 2010 Constitution's bicameral Parliament (National Assembly + Senate)
//   - Kenyan Bill stages (First Reading → ... → Presidential Assent → Commencement)
//   - Kenyan parliamentary terminology (Second Reading, Hansard, Order Paper, etc.)
//   - Source URLs for parliament.go.ke, kenyalaw.org, the Kenya Gazette
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Uganda means writing adapters/uganda/ — the legislation
// service code is unchanged.
package kenya

import (
	"context"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/gazette"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// KenyaAdapter implements contracts.LegislativeSourceAdapter.
// Compile-time assertion at the bottom guarantees interface compliance.
type KenyaAdapter struct {
	parliament *parliament.Adapter
	kenyaLaw   *kenya_law.Adapter
	gazette    *gazette.Adapter
}

// Dependencies holds the collaborators the adapter needs.
type Dependencies struct {
	HTTPClient parliament.HTTPClient
	UserAgent  string
}

// NewKenyaAdapter constructs a Kenya adapter.
func NewKenyaAdapter(deps Dependencies) *KenyaAdapter {
	if deps.UserAgent == "" {
		deps.UserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"
	}
	return &KenyaAdapter{
		parliament: parliament.NewAdapter(deps.HTTPClient, deps.UserAgent),
		kenyaLaw:   kenya_law.NewAdapter(deps.HTTPClient, deps.UserAgent),
		gazette:    gazette.NewAdapter(deps.HTTPClient, deps.UserAgent),
	}
}

// Discover delegates to all three source adapters and merges results.
func (a *KenyaAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	var out []contracts.SourceItem
	for _, src := range []interface {
		Discover(ctx context.Context) ([]contracts.SourceItem, error)
	}{a.parliament, a.kenyaLaw, a.gazette} {
		items, err := src.Discover(ctx)
		if err != nil {
			continue // a single failing source does not abort the whole discovery
		}
		out = append(out, items...)
	}
	return out, nil
}

func (a *KenyaAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	switch item.DocumentType {
	case "act", "regulation", "legal_notice":
		return a.kenyaLaw.Fetch(ctx, item)
	case "gazette_notice":
		return a.gazette.Fetch(ctx, item)
	default:
		return a.parliament.Fetch(ctx, item)
	}
}

func (a *KenyaAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// GetLegislativeStructure returns Kenya's institutions, legislature, houses,
// and committees. This is ADAPTER DATA — these strings are values, not
// constants in the global domain model.
func (a *KenyaAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.KenyaLegislativeStructure()
	return &s, nil
}

// GetStages returns the Kenyan Bill stages per the 2010 Constitution and
// Parliament's Standing Orders. The internal package stores these as a richer
// []KenyaStage type; we project them down to []contracts.StageDefinition
// via each stage's ToContract() method.
func (a *KenyaAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	stages := internal.KenyaBillStages
	out := make([]contracts.StageDefinition, len(stages))
	for i, s := range stages {
		out[i] = s.ToContract()
	}
	return out, nil
}

// GetTerminology returns the Kenyan parliamentary terminology registry.
// Same projection pattern as GetStages.
func (a *KenyaAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	terms := internal.KenyaTerminology
	out := make([]contracts.TermDefinition, len(terms))
	for i, t := range terms {
		out[i] = t.ToContract()
	}
	return out, nil
}

// Compile-time assertion: KenyaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*KenyaAdapter)(nil)
