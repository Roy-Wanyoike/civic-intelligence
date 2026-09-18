// Package registry is the central registry of country adapters for the Civic
// Intelligence Platform. It maps an ISO 3166-1 alpha-2 country code (e.g.
// "KE", "UG", "TZ") to the CountryAdapter implementation for that country.
//
// Architectural role (ADR-0004 — Country Adapter Pattern):
//   - The global domain model in services/legislation/ contains ZERO
//     country-specific strings. Country-specific code lives in
//     adapters/{country}/ packages.
//   - The ingestion service and the API layer look up an adapter by country
//     code via this registry rather than hard-coding switch statements. That
//     keeps the API layer free of country knowledge — adding a new country
//     means writing its adapter package + registering it here + nothing else.
//
// Data isolation contract:
//   - Each CountryAdapter implementation lives in its own Go module
//     (adapters/{country}/go.mod). Changing data in one adapter CANNOT affect
//     any other adapter — the packages are physically separate.
//   - The registry stores adapters by country code; look-ups are O(1) and
//     always return the exact adapter for the requested country. There is no
//     fallback, no merge, no cross-country defaulting.
//   - A contributor working in adapters/tanzania/ cannot accidentally change
//     Kenyan data — the Kenya adapter code path is in a different directory
//     with its own go.mod and its own test suite.
package registry

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// CountryAdapter is the high-level interface every country adapter exposes
// to the platform. It extends the data-lookup surface of
// contracts.LegislativeSourceAdapter with metadata helpers (country name,
// flag, parliament name, legislature type) and convenience accessors that
// return values without an error — the underlying adapters always return
// nil errors for these data lookups because the data is statically defined
// per country.
//
// The wrapper types in wrappers.go adapt the existing
// contracts.LegislativeSourceAdapter implementations (kenya.KenyaAdapter,
// uganda.UgandaAdapter, ...) to this interface without modifying the
// underlying adapter packages.
type CountryAdapter interface {
	// CountryCode returns the ISO 3166-1 alpha-2 code (e.g. "KE").
	CountryCode() string

	// CountryName returns the human-readable name (e.g. "Kenya").
	CountryName() string

	// FlagEmoji returns the country's flag emoji (e.g. "🇰🇪").
	FlagEmoji() string

	// ParliamentName returns the human-readable name of the country's
	// national legislature (e.g. "Parliament of Kenya", "Bunge la Tanzania",
	// "National Assembly of Nigeria").
	ParliamentName() string

	// LegislatureType returns "unicameral" or "bicameral".
	LegislatureType() string

	// GetStages returns the country's Bill stages. The returned slice carries
	// the country's own country code on every stage — the platform NEVER
	// mixes stages across countries.
	GetStages() []contracts.StageDefinition

	// GetLegislativeStructure returns the country's institutions, houses, and
	// committees. The returned structure carries the country's own country
	// code — the platform NEVER mixes structures across countries.
	GetLegislativeStructure() contracts.LegislativeStructure

	// GetOfficialSources returns the country's authoritative upstream sources
	// (parliament website, official gazette, official law reports, etc.). Each
	// source carries an authority_level so consumers can prioritise.
	GetOfficialSources() []SourceDefinition

	// DiscoverBills returns the country's currently-discovered Bills. The
	// returned slice contains ONLY Bills from this country — the registry
	// never cross-contaminates Bills across countries.
	DiscoverBills(ctx context.Context) ([]BillCandidate, error)
}

// SourceDefinition describes an authoritative upstream source for a country.
type SourceDefinition struct {
	Name           string
	URL            string
	AuthorityLevel string // "primary" | "official" | "aggregator"
	ItemType       string // "bill" | "hansard" | "gazette" | "act" | "regulation" | "policy"
}

// BillCandidate is a discovered Bill awaiting ingestion into the platform.
type BillCandidate struct {
	Title       string
	Number      string
	Sponsor     string
	Stage       string
	House       string
	SourceURL   string
	CountryCode string
}

// CountryInfo is the public-facing metadata for a supported country.
type CountryInfo struct {
	Code            string
	Name            string
	FlagEmoji       string
	ParliamentName  string
	LegislatureType string
}

// ErrUnsupportedCountry is returned by GetAdapter when the requested country
// code is not registered. Callers may use errors.Is(err, ErrUnsupportedCountry)
// to detect this condition and surface a 404 / 400 to the client.
var ErrUnsupportedCountry = errors.New("unsupported country")

var (
	registryMu sync.RWMutex
	registry   = map[string]CountryAdapter{}
)

// Register adds an adapter to the registry. Panics on a duplicate country
// code — duplicate registration is a programming error caught at startup, not
// a runtime error to be recovered from.
//
// Safe for concurrent use; concurrent calls serialise on the registry mutex.
func Register(adapter CountryAdapter) {
	if adapter == nil {
		panic("registry.Register: nil adapter")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	code := adapter.CountryCode()
	if code == "" {
		panic("registry.Register: adapter returned empty CountryCode()")
	}
	if _, exists := registry[code]; exists {
		panic(fmt.Sprintf("registry.Register: duplicate country code %q (already registered)", code))
	}
	registry[code] = adapter
}

// GetAdapter returns the adapter registered for the given country code.
// Returns ErrUnsupportedCountry (wrapped) when the code is unknown so HTTP
// handlers can map it to a 404 without further inspection.
func GetAdapter(countryCode string) (CountryAdapter, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	a, ok := registry[countryCode]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedCountry, countryCode)
	}
	return a, nil
}

// MustGetAdapter is the convenience form of GetAdapter — it panics when the
// code is unknown. Use only in startup paths or tests where an unknown code
// is a programming error, not a runtime error.
func MustGetAdapter(countryCode string) CountryAdapter {
	a, err := GetAdapter(countryCode)
	if err != nil {
		panic(err)
	}
	return a
}

// SupportedCountries returns metadata for every registered country, sorted by
// ISO 3166-1 alpha-2 code so the output is stable across runs (and across
// concurrent registrations). Used by /api/v1/countries and the frontend
// government selector.
func SupportedCountries() []CountryInfo {
	registryMu.RLock()
	codes := make([]string, 0, len(registry))
	for code := range registry {
		codes = append(codes, code)
	}
	adaptersByCode := make(map[string]CountryAdapter, len(codes))
	for code, a := range registry {
		adaptersByCode[code] = a
	}
	registryMu.RUnlock()

	sort.Strings(codes)
	out := make([]CountryInfo, 0, len(codes))
	for _, code := range codes {
		a := adaptersByCode[code]
		if a == nil {
			continue
		}
		out = append(out, CountryInfo{
			Code:             a.CountryCode(),
			Name:             a.CountryName(),
			FlagEmoji:        a.FlagEmoji(),
			ParliamentName:   a.ParliamentName(),
			LegislatureType:  a.LegislatureType(),
		})
	}
	return out
}

// IsSupported reports whether a country code is registered.
func IsSupported(countryCode string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	_, ok := registry[countryCode]
	return ok
}

// ResetForTest clears the registry. Test-only — production code MUST NOT
// call this. Used by the isolation tests to register a clean set of
// adapters and verify isolation behaviour without interference from other
// test files that may have registered adapters.
func ResetForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]CountryAdapter{}
}
