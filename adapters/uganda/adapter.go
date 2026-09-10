// Package uganda implements the Uganda country adapter for the Civic Intelligence
// Platform. It implements contracts.LegislativeSourceAdapter.
//
// Uganda's Parliament is UNICAMERAL (no Senate) — unlike Kenya's bicameral
// Parliament. The adapter handles this without changing the global domain model.
package uganda

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// UgandaAdapter implements contracts.LegislativeSourceAdapter.
type UgandaAdapter struct {
	parliament *parliament.Adapter
}

// NewUgandaAdapter constructs a Uganda adapter.
func NewUgandaAdapter() *UgandaAdapter {
	return &UgandaAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

func (a *UgandaAdapter) CountryCode() string { return "UG" }

func (a *UgandaAdapter) Supports(url string) bool {
	return strings.Contains(url, "parliament.go.ug") || strings.Contains(url, "parliamentwatch.ug")
}

func (a *UgandaAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

func (a *UgandaAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

func (a *UgandaAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

func (a *UgandaAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "UG",
	}, nil
}

func (a *UgandaAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.UgandaLegislativeStructure()
	return &s, nil
}

func (a *UgandaAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.UgandaBillStages, nil
}

func (a *UgandaAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.UgandaTerminology, nil
}

// Compile-time assertion.
var _ contracts.LegislativeSourceAdapter = (*UgandaAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
