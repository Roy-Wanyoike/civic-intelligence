// Package ghana implements the Ghana country adapter for the Civic Intelligence
// Platform. It implements contracts.LegislativeSourceAdapter.
//
// Ghana's Parliament is UNICAMERAL (no Senate) — like Uganda's, but unlike
// Kenya's bicameral Parliament. The adapter handles this without changing the
// global domain model.
package ghana

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// GhanaAdapter implements contracts.LegislativeSourceAdapter.
type GhanaAdapter struct {
	parliament *parliament.Adapter
}

// NewGhanaAdapter constructs a Ghana adapter.
func NewGhanaAdapter() *GhanaAdapter {
	return &GhanaAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for Ghana.
func (a *GhanaAdapter) CountryCode() string { return "GH" }

// Supports reports whether this adapter can handle the given URL.
// Parliament of Ghana sources are served under hosts containing "parliament.gh".
func (a *GhanaAdapter) Supports(url string) bool {
	return strings.Contains(url, "parliament.gh")
}

// Discover returns the items currently available from this adapter's sources.
func (a *GhanaAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *GhanaAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *GhanaAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *GhanaAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "GH",
	}, nil
}

// GetLegislativeStructure returns Ghana's institutions, houses, and committees.
func (a *GhanaAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.GhanaLegislativeStructure()
	return &s, nil
}

// GetStages returns Ghana's Bill stages (with allowed transitions).
func (a *GhanaAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.GhanaBillStages, nil
}

// GetTerminology returns Ghana's parliamentary terminology registry.
func (a *GhanaAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.GhanaTerminology, nil
}

// Compile-time assertion: GhanaAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*GhanaAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
