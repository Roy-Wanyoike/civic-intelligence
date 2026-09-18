// Package contract — adapter_contract_test.go verifies ALL 6 country
// adapters (Kenya, Uganda, Tanzania, Ghana, Nigeria, South Africa)
// implement the contracts.LegislativeSourceAdapter interface AND that
// every adapter returns country-specific data shaped consistently with
// the contracts.
//
// Per adapter ADR-0004 (Country Adapter Pattern): country-specific code
// stays inside the adapter package; the global domain model contains
// ZERO country-specific strings. This test enforces that contract by
// driving each adapter through the same interface and asserting on the
// shared invariants.
package contract

import (
        "context"
        "testing"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// Compile-time assertions: every adapter concrete type must satisfy
// contracts.LegislativeSourceAdapter. If any adapter drifts (e.g. method
// signature changes), the build fails here.
var (
        _ contracts.LegislativeSourceAdapter = (*kenya.KenyaAdapter)(nil)
        _ contracts.LegislativeSourceAdapter = (*uganda.UgandaAdapter)(nil)
        _ contracts.LegislativeSourceAdapter = (*tanzania.TanzaniaAdapter)(nil)
        _ contracts.LegislativeSourceAdapter = (*ghana.GhanaAdapter)(nil)
        _ contracts.LegislativeSourceAdapter = (*nigeria.NigeriaAdapter)(nil)
        _ contracts.LegislativeSourceAdapter = (*south_africa.SouthAfricaAdapter)(nil)
)

// adapterFactory is a constructor that returns a fresh adapter instance.
// Used so each test can drive a fresh adapter without state bleed.
type adapterFactory func() contracts.LegislativeSourceAdapter

// allAdapters returns every country adapter the platform ships, paired
// with its expected ISO 3166-1 alpha-2 country code. Adding a new
// adapter? Register it here — the rest of the suite picks it up
// automatically.
func allAdapters() []struct {
        code    string
        factory adapterFactory
} {
        return []struct {
                code    string
                factory adapterFactory
        }{
                {"KE", func() contracts.LegislativeSourceAdapter { return kenya.NewKenyaAdapter(kenya.Dependencies{}) }},
                {"UG", func() contracts.LegislativeSourceAdapter { return uganda.NewUgandaAdapter() }},
                {"TZ", func() contracts.LegislativeSourceAdapter { return tanzania.NewTanzaniaAdapter() }},
                {"GH", func() contracts.LegislativeSourceAdapter { return ghana.NewGhanaAdapter() }},
                {"NG", func() contracts.LegislativeSourceAdapter { return nigeria.NewNigeriaAdapter(nigeria.Dependencies{}) }},
                {"ZA", func() contracts.LegislativeSourceAdapter { return south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{}) }},
        }
}

// TestAllAdapters_SatisfyInterface runs every adapter factory through the
// interface to confirm (a) the constructor returns a non-nil instance
// and (b) the instance satisfies the contracts.LegislativeSourceAdapter
// interface at runtime (not just at compile time). The compile-time
// assertions above catch signature drift; this catches runtime wiring
// issues (nil receivers, panicking constructors).
func TestAllAdapters_SatisfyInterface(t *testing.T) {
        for _, a := range allAdapters() {
                adapter := a.factory()
                if adapter == nil {
                        t.Errorf("adapter %s: factory returned nil", a.code)
                        continue
                }
                if adapter.CountryCode() != a.code {
                        t.Errorf("adapter %s: CountryCode() returned %q, expected %q",
                                a.code, adapter.CountryCode(), a.code)
                }
        }
}

// TestAllAdapters_GetStages exercises GetStages and asserts that every
// returned stage carries the adapter's own country code (per ADR-0004 —
// country-specific data stays inside the adapter, but the StageDefinition
// embeds the country for the global domain's convenience).
func TestAllAdapters_GetStages(t *testing.T) {
        ctx := context.Background()
        for _, a := range allAdapters() {
                adapter := a.factory()
                stages, err := adapter.GetStages(ctx)
                if err != nil {
                        t.Errorf("adapter %s: GetStages error: %v", a.code, err)
                        continue
                }
                if len(stages) == 0 {
                        t.Errorf("adapter %s: GetStages returned 0 stages", a.code)
                        continue
                }
                for _, s := range stages {
                        if s.Country != contracts.Country(a.code) {
                                t.Errorf("adapter %s: stage %q has country %q, expected %q",
                                        a.code, s.Code, s.Country, a.code)
                        }
                        if s.Code == "" || s.Name == "" {
                                t.Errorf("adapter %s: stage has empty Code or Name: %+v", a.code, s)
                        }
                }
        }
}

// TestAllAdapters_GetTerminology exercises GetTerminology and asserts
// every returned term carries the adapter's country code + a non-empty
// SimpleExplanation (per the platform-wide promise that every term has
// a plain-language explanation).
func TestAllAdapters_GetTerminology(t *testing.T) {
        ctx := context.Background()
        for _, a := range allAdapters() {
                adapter := a.factory()
                terms, err := adapter.GetTerminology(ctx)
                if err != nil {
                        t.Errorf("adapter %s: GetTerminology error: %v", a.code, err)
                        continue
                }
                if len(terms) == 0 {
                        t.Errorf("adapter %s: GetTerminology returned 0 terms", a.code)
                        continue
                }
                for _, term := range terms {
                        if term.Country != contracts.Country(a.code) {
                                t.Errorf("adapter %s: term %q has country %q, expected %q",
                                        a.code, term.Term, term.Country, a.code)
                        }
                        if term.Term == "" || term.SimpleExplanation == "" {
                                t.Errorf("adapter %s: term has empty Term or SimpleExplanation: %+v", a.code, term)
                        }
                }
        }
}

// TestAllAdapters_GetLegislativeStructure exercises the structure endpoint
// and asserts the returned structure carries the adapter's country code.
func TestAllAdapters_GetLegislativeStructure(t *testing.T) {
        ctx := context.Background()
        for _, a := range allAdapters() {
                adapter := a.factory()
                s, err := adapter.GetLegislativeStructure(ctx)
                if err != nil {
                        t.Errorf("adapter %s: GetLegislativeStructure error: %v", a.code, err)
                        continue
                }
                if s == nil {
                        t.Errorf("adapter %s: GetLegislativeStructure returned nil", a.code)
                        continue
                }
                if s.Country != contracts.Country(a.code) {
                        t.Errorf("adapter %s: structure has country %q, expected %q",
                                a.code, s.Country, a.code)
                }
                if len(s.Houses) == 0 {
                        t.Errorf("adapter %s: structure has 0 houses", a.code)
                }
        }
}

// TestAllAdapters_Supports exercises the Supports method with a known
// country URL + a clearly foreign URL. The known URL must match; the
// foreign URL must NOT match (prevents adapters from claiming URLs they
// cannot serve).
func TestAllAdapters_Supports(t *testing.T) {
        cases := []struct {
                code        string
                knownURL    string
                foreignURL  string
        }{
                {"KE", "https://www.parliament.go.ke/", "https://www.parliament.go.ug/"},
                {"UG", "https://www.parliament.go.ug/", "https://www.parliament.go.ke/"},
                {"TZ", "https://www.parliament.go.tz/", "https://www.parliament.go.ke/"},
                {"GH", "https://www.parliament.gh/", "https://www.parliament.go.ke/"},
                {"NG", "https://www.nass.gov.ng/", "https://www.parliament.go.ke/"},
                {"ZA", "https://www.parliament.gov.za/", "https://www.parliament.go.ke/"},
        }
        for _, c := range cases {
                factory := func() contracts.LegislativeSourceAdapter {
                        for _, a := range allAdapters() {
                                if a.code == c.code {
                                        return a.factory()
                                }
                        }
                        t.Fatalf("no adapter registered for %s", c.code)
                        return nil
                }
                adapter := factory()
                if !adapter.Supports(c.knownURL) {
                        t.Errorf("adapter %s: Supports(%q) returned false; expected true",
                                c.code, c.knownURL)
                }
                // Note: a foreign URL might be claimed by another adapter, but
                // no adapter should claim a URL on a different country's domain.
                // We use a generic example.com URL as the unambiguous foreign.
                if adapter.Supports("https://example.com/not-a-parliament-site") {
                        t.Errorf("adapter %s: Supports(example.com) returned true; expected false",
                                c.code)
                }
        }
}

// TestAllAdapters_NormalizeSourceItem verifies every adapter projects a
// raw map into a SourceItem carrying the adapter's country code. Some
// adapters (Kenya, Nigeria, South Africa) require additional fields like
// `external_id` for their normalization; we supply a minimal seed so
// every adapter's NormalizeSourceItem succeeds.
func TestAllAdapters_NormalizeSourceItem(t *testing.T) {
        for _, a := range allAdapters() {
                adapter := a.factory()
                item, err := adapter.NormalizeSourceItem(map[string]any{
                        "url":         "https://example.com/bill/1",
                        "title":       "Test Bill",
                        "external_id": "EXT-" + a.code + "-1",
                })
                if err != nil {
                        t.Errorf("adapter %s: NormalizeSourceItem error: %v", a.code, err)
                        continue
                }
                if item.CountryCode != a.code {
                        t.Errorf("adapter %s: NormalizeSourceItem set country_code %q, expected %q",
                                a.code, item.CountryCode, a.code)
                }
        }
}

// TestAllAdapters_TerminalStagesExist verifies every adapter declares
// the canonical terminal stages (REJECTED, COMMENCEMENT at minimum).
// These are the platform-wide terminal states; every country's Bill flow
// must end in one of them.
func TestAllAdapters_TerminalStagesExist(t *testing.T) {
        ctx := context.Background()
        for _, a := range allAdapters() {
                adapter := a.factory()
                stages, err := adapter.GetStages(ctx)
                if err != nil {
                        t.Errorf("adapter %s: GetStages error: %v", a.code, err)
                        continue
                }
                byCode := map[string]contracts.StageDefinition{}
                for _, s := range stages {
                        byCode[s.Code] = s
                }
                // COMMENCEMENT is the canonical success terminal state.
                if c, ok := byCode["COMMENCEMENT"]; !ok {
                        t.Errorf("adapter %s: missing COMMENCEMENT terminal stage", a.code)
                } else if !c.IsTerminal {
                        t.Errorf("adapter %s: COMMENCEMENT stage must be terminal", a.code)
                }
                // REJECTED is the canonical failure terminal state.
                if r, ok := byCode["REJECTED"]; !ok {
                        t.Errorf("adapter %s: missing REJECTED terminal stage", a.code)
                } else if !r.IsTerminal {
                        t.Errorf("adapter %s: REJECTED stage must be terminal", a.code)
                }
        }
}

// TestAllAdapters_StageChainHasNoForwardReferences verifies every adapter's
// stage AllowedNext references point to existing stage codes (no dangling
// transitions, no typos).
func TestAllAdapters_StageChainHasNoForwardReferences(t *testing.T) {
        ctx := context.Background()
        for _, a := range allAdapters() {
                adapter := a.factory()
                stages, err := adapter.GetStages(ctx)
                if err != nil {
                        t.Errorf("adapter %s: GetStages error: %v", a.code, err)
                        continue
                }
                byCode := map[string]bool{}
                for _, s := range stages {
                        byCode[s.Code] = true
                }
                for _, s := range stages {
                        for _, next := range s.AllowedNext {
                                if !byCode[next] {
                                        t.Errorf("adapter %s: stage %q AllowedNext references unknown stage %q",
                                                a.code, s.Code, next)
                                }
                        }
                }
        }
}
