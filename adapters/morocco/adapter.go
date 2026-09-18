// Package morocco implements the Morocco country adapter for the Civic
// Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// Morocco is a CONSTITUTIONAL MONARCHY with a BICAMERAL Parliament:
//   - House of Representatives (lower house, 395 members, directly elected)
//   - House of Councillors (upper house, 120 members, indirectly elected)
//
// The King (Mohammed VI, since 1999) is head of state and promulgates laws;
// the Head of Government (Aziz Akhannouch, since 2021) leads the executive.
//
// All Morocco-specific knowledge lives here:
//   - Bill stages (Proposal → Committee → First Reading → Second Reading →
//     House of Councillors Review → Final Vote → Royal Promulgation)
//   - Parliamentary terminology (Chambre des Représentants, Chambre des
//     Conseillers, Promulgation Royale, etc.)
//   - Source URLs for parlement.ma
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Morocco means writing adapters/morocco/ — the legislation
// service code is unchanged.
package morocco

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco/parliament"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// MoroccoAdapter implements contracts.LegislativeSourceAdapter.
type MoroccoAdapter struct {
	parliament *parliament.Adapter
}

// NewMoroccoAdapter constructs a Morocco adapter.
func NewMoroccoAdapter() *MoroccoAdapter {
	return &MoroccoAdapter{
		parliament: parliament.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"),
	}
}

// CountryCode returns the ISO 3166-1 alpha-2 country code for Morocco.
func (a *MoroccoAdapter) CountryCode() string { return "MA" }

// Supports reports whether this adapter can handle the given URL.
// Morocco's Parliament sources are served under hosts containing "parlement.ma".
func (a *MoroccoAdapter) Supports(url string) bool {
	return strings.Contains(url, "parlement.ma")
}

// Discover returns the items currently available from this adapter's sources.
func (a *MoroccoAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes.
func (a *MoroccoAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *MoroccoAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return a.parliament.Parse(ctx, doc)
}

// NormalizeSourceItem converts country-specific raw metadata into the
// platform's canonical SourceItem shape.
func (a *MoroccoAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "MA",
	}, nil
}

// GetLegislativeStructure returns Morocco's institutions, houses, and committees.
func (a *MoroccoAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.MoroccoLegislativeStructure()
	return &s, nil
}

// GetStages returns Morocco's Bill stages (with allowed transitions).
func (a *MoroccoAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return internal.MoroccoBillStages, nil
}

// GetTerminology returns Morocco's parliamentary terminology registry.
func (a *MoroccoAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return internal.MoroccoTerminology, nil
}

// Compile-time assertion: MoroccoAdapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*MoroccoAdapter)(nil)

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
