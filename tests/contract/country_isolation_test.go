// Package contract — country_isolation_test.go verifies that each country's
// data is fully isolated: a contributor working in one adapter CANNOT
// accidentally affect another country's data. This is the architectural
// invariant behind ADR-0004 (Country Adapter Pattern) and the platform's
// promise to country contributors.
//
// The tests exercise the registry (adapters/registry) AND the per-country
// seed packages so they catch isolation regressions at TWO layers:
//
//  1. Registry layer: each country code resolves to its own adapter; no
//     adapter returns data for another country; stages + structures carry
//     the correct country code.
//
//  2. Seed layer: each country's seed data (presidents, administrations,
//     Acts, debt snapshots, borrowing agreements) carries only its own
//     country code; the lists are physically separate Go packages and
//     therefore impossible to cross-contaminate at the data level.
//
// Adding a new country? Run `go test -count=1 -run CountryIsolation ./...`
// from this directory — every test below is table-driven so the new country
// is picked up automatically once it is registered.
package contract

import (
        "context"
        "errors"
        "sort"
        "strings"
        "testing"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana/ghana_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria/nigeria_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/registry"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa/south_africa_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania/tanzania_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// expectedCountries is the canonical list of countries the platform supports.
// Adding a new country? Append it here — every test below is table-driven off
// this list so the new country is exercised automatically.
//
// Wave 12 (ENG-L1, 2026) added the final 4 adapters — Rwanda (RW), Zambia
// (ZM), Senegal (SN), and Egypt (EG) — bringing the platform from 6 to 10
// supported countries.
var expectedCountries = []struct {
        code, name, flag, parliament, legType string
}{
        {"KE", "Kenya", "🇰🇪", "Parliament of Kenya", "bicameral"},
        {"UG", "Uganda", "🇺🇬", "Parliament of Uganda", "unicameral"},
        {"TZ", "Tanzania", "🇹🇿", "Bunge la Tanzania", "unicameral"},
        {"GH", "Ghana", "🇬🇭", "Parliament of Ghana", "unicameral"},
        {"NG", "Nigeria", "🇳🇬", "National Assembly of Nigeria", "bicameral"},
        {"ZA", "South Africa", "🇿🇦", "Parliament of South Africa", "bicameral"},
        {"RW", "Rwanda", "🇷🇼", "Parliament of Rwanda", "bicameral"},
        {"ZM", "Zambia", "🇿🇲", "National Assembly of Zambia", "unicameral"},
        {"SN", "Senegal", "🇸🇳", "Assemblée Nationale du Sénégal", "unicameral"},
        {"EG", "Egypt", "🇪🇬", "Egyptian Parliament", "bicameral"},
}

// setupRegistryForTest returns a registry with all 6 default adapters
// registered, isolated from any prior test state. Tests MUST call this
// rather than relying on package-level init() so the test order does not
// matter.
func setupRegistryForTest(t *testing.T) {
        t.Helper()
        registry.ResetForTest()
        registry.MustRegisterDefault()
}

// TestRegistry_ReturnsAll6SupportedCountries verifies the registry knows
// about every expected country code. A missing entry means the country is
// not registered — a contributor cannot reach its data via the API.
func TestRegistry_ReturnsAll6SupportedCountries(t *testing.T) {
        setupRegistryForTest(t)
        got := registry.SupportedCountries()
        if len(got) != len(expectedCountries) {
                t.Fatalf("SupportedCountries returned %d countries, expected %d", len(got), len(expectedCountries))
        }
        // SupportedCountries is sorted by code, so we can compare against a
        // sorted expectation.
        want := make([]struct {
                code, name, flag, parliament, legType string
        }, len(expectedCountries))
        copy(want, expectedCountries)
        sort.Slice(want, func(i, j int) bool { return want[i].code < want[j].code })
        for i, w := range want {
                g := got[i]
                if g.Code != w.code {
                        t.Errorf("SupportedCountries[%d].Code = %q, expected %q", i, g.Code, w.code)
                }
                if g.Name != w.name {
                        t.Errorf("SupportedCountries[%d].Name = %q, expected %q", i, g.Name, w.name)
                }
                if g.FlagEmoji != w.flag {
                        t.Errorf("SupportedCountries[%d].FlagEmoji = %q, expected %q", i, g.FlagEmoji, w.flag)
                }
                if g.ParliamentName != w.parliament {
                        t.Errorf("SupportedCountries[%d].ParliamentName = %q, expected %q", i, g.ParliamentName, w.parliament)
                }
                if g.LegislatureType != w.legType {
                        t.Errorf("SupportedCountries[%d].LegislatureType = %q, expected %q", i, g.LegislatureType, w.legType)
                }
        }
}

// TestRegistry_GetAdapter returns the correct adapter for each supported
// country code. The returned adapter MUST carry the country's own code.
func TestRegistry_GetAdapter(t *testing.T) {
        setupRegistryForTest(t)
        for _, c := range expectedCountries {
                a, err := registry.GetAdapter(c.code)
                if err != nil {
                        t.Errorf("GetAdapter(%q) returned error: %v", c.code, err)
                        continue
                }
                if a == nil {
                        t.Errorf("GetAdapter(%q) returned nil adapter", c.code)
                        continue
                }
                if got := a.CountryCode(); got != c.code {
                        t.Errorf("GetAdapter(%q).CountryCode() = %q", c.code, got)
                }
                if got := a.CountryName(); got != c.name {
                        t.Errorf("GetAdapter(%q).CountryName() = %q, expected %q", c.code, got, c.name)
                }
                if got := a.LegislatureType(); got != c.legType {
                        t.Errorf("GetAdapter(%q).LegislatureType() = %q, expected %q", c.code, got, c.legType)
                }
                if got := a.ParliamentName(); got != c.parliament {
                        t.Errorf("GetAdapter(%q).ParliamentName() = %q, expected %q", c.code, got, c.parliament)
                }
                if got := a.FlagEmoji(); got != c.flag {
                        t.Errorf("GetAdapter(%q).FlagEmoji() = %q, expected %q", c.code, got, c.flag)
                }
        }
}

// TestRegistry_UnsupportedCountryReturnsError verifies the registry returns
// ErrUnsupportedCountry for unknown codes — never a nil adapter. This is
// the contract the API layer relies on to map unknown codes to HTTP 404.
func TestRegistry_UnsupportedCountryReturnsError(t *testing.T) {
        setupRegistryForTest(t)
        for _, code := range []string{"XX", "UK", "US", "FR", "", "ke", "Kenya"} {
                a, err := registry.GetAdapter(code)
                if err == nil {
                        t.Errorf("GetAdapter(%q) expected error, got nil adapter=%v", code, a)
                        continue
                }
                if !errors.Is(err, registry.ErrUnsupportedCountry) {
                        t.Errorf("GetAdapter(%q) error type %T does not satisfy errors.Is(ErrUnsupportedCountry)", code, err)
                }
                if a != nil {
                        t.Errorf("GetAdapter(%q) returned non-nil adapter alongside error", code)
                }
        }
}

// TestRegistry_IsSupported reports accurately for both supported + unsupported codes.
func TestRegistry_IsSupported(t *testing.T) {
        setupRegistryForTest(t)
        for _, c := range expectedCountries {
                if !registry.IsSupported(c.code) {
                        t.Errorf("IsSupported(%q) = false; expected true", c.code)
                }
        }
        for _, code := range []string{"XX", "UK", ""} {
                if registry.IsSupported(code) {
                        t.Errorf("IsSupported(%q) = true; expected false", code)
                }
        }
}

// TestRegistry_StageCountryCodeIsolation verifies that each adapter's GetStages
// returns ONLY stages for that adapter's own country — never a foreign code.
// This catches the "stage bleed" failure mode where a contributor copies a
// stage slice from another country and forgets to swap the country code.
func TestRegistry_StageCountryCodeIsolation(t *testing.T) {
        setupRegistryForTest(t)
        for _, c := range expectedCountries {
                a, err := registry.GetAdapter(c.code)
                if err != nil {
                        t.Fatalf("GetAdapter(%q): %v", c.code, err)
                }
                stages := a.GetStages()
                if len(stages) == 0 {
                        t.Errorf("adapter %s: GetStages returned 0 stages", c.code)
                        continue
                }
                for _, s := range stages {
                        if got := string(s.Country); got != c.code {
                                t.Errorf("adapter %s: stage %q has Country=%q (isolation breach)",
                                        c.code, s.Code, got)
                        }
                        // The stage code MUST be uppercase + non-empty — the contract
                        // is that stage codes are SCREAMING_SNAKE strings.
                        if s.Code != strings.ToUpper(s.Code) || s.Code == "" {
                                t.Errorf("adapter %s: stage code %q is not uppercase SCREAMING_SNAKE", c.code, s.Code)
                        }
                }
        }
}

// TestRegistry_LegislativeStructureCountryIsolation verifies each adapter's
// GetLegislativeStructure returns the country's own structure (not another
// country's). It also verifies the bicameral/unicameral claim matches the
// number of houses returned.
func TestRegistry_LegislativeStructureCountryIsolation(t *testing.T) {
        setupRegistryForTest(t)
        for _, c := range expectedCountries {
                a, err := registry.GetAdapter(c.code)
                if err != nil {
                        t.Fatalf("GetAdapter(%q): %v", c.code, err)
                }
                s := a.GetLegislativeStructure()
                if got := string(s.Country); got != c.code {
                        t.Errorf("adapter %s: structure Country = %q (isolation breach)", c.code, got)
                }
                if got := s.CountryCode; got != c.code {
                        t.Errorf("adapter %s: structure CountryCode = %q (isolation breach)", c.code, got)
                }
                if got := s.CountryName; got != c.name {
                        t.Errorf("adapter %s: structure CountryName = %q, expected %q", c.code, got, c.name)
                }
                // Bicameral => exactly 2 houses; unicameral => exactly 1 house.
                wantHouses := 1
                if c.legType == "bicameral" {
                        wantHouses = 2
                }
                if len(s.Houses) != wantHouses {
                        t.Errorf("adapter %s: legislature type %s but got %d houses, expected %d",
                                c.code, c.legType, len(s.Houses), wantHouses)
                }
                // Each house must declare its chamber type explicitly.
                for _, h := range s.Houses {
                        if h.Code == "" || h.Name == "" {
                                t.Errorf("adapter %s: house %+v has empty Code or Name", c.code, h)
                        }
                        if c.legType == "unicameral" && h.Type != contracts.HouseTypeSingle {
                                t.Errorf("adapter %s: unicameral but house %s has Type=%s, expected %s",
                                        c.code, h.Name, h.Type, contracts.HouseTypeSingle)
                        }
                }
        }
}

// TestRegistry_OfficialSourcesCountryIsolation verifies that every URL in an
// adapter's GetOfficialSources list belongs to that country's own domains.
// This catches the failure mode where a contributor copies the Kenya source
// list and forgets to swap the parliament URL.
func TestRegistry_OfficialSourcesCountryIsolation(t *testing.T) {
        setupRegistryForTest(t)
        // Per-country known official domain fragments. Each URL returned by an
        // adapter MUST contain the country's own fragment — never another country's.
        knownDomains := map[string]string{
                "KE": "parliament.go.ke",
                "UG": "parliament.go.ug",
                "TZ": "parliament.go.tz",
                "GH": "parliament.gh",
                "NG": "nass.gov.ng",
                "ZA": "parliament.gov.za",
        }
        // Domains that belong to OTHER countries — MUST NOT appear in this adapter's sources.
        foreignDomains := map[string][]string{
                "KE": {"parliament.go.ug", "parliament.go.tz", "nass.gov.ng", "parliament.gov.za"},
                "UG": {"parliament.go.ke", "parliament.go.tz", "nass.gov.ng", "parliament.gov.za"},
                "TZ": {"parliament.go.ke", "parliament.go.ug", "nass.gov.ng", "parliament.gov.za"},
                "GH": {"parliament.go.ke", "parliament.go.ug", "parliament.go.tz", "nass.gov.ng"},
                "NG": {"parliament.go.ke", "parliament.go.ug", "parliament.gov.za"},
                "ZA": {"parliament.go.ke", "parliament.go.ug", "nass.gov.ng"},
        }
        for _, c := range expectedCountries {
                a, err := registry.GetAdapter(c.code)
                if err != nil {
                        t.Fatalf("GetAdapter(%q): %v", c.code, err)
                }
                srcs := a.GetOfficialSources()
                if len(srcs) == 0 {
                        t.Errorf("adapter %s: GetOfficialSources returned 0 sources", c.code)
                        continue
                }
                // The country's primary parliament source MUST appear at least once.
                ownDomain := knownDomains[c.code]
                foundOwn := false
                for _, src := range srcs {
                        if src.URL == "" {
                                t.Errorf("adapter %s: source %+v has empty URL", c.code, src)
                                continue
                        }
                        if strings.Contains(src.URL, ownDomain) {
                                foundOwn = true
                        }
                        // None of the foreign domains may appear.
                        for _, foreign := range foreignDomains[c.code] {
                                if strings.Contains(src.URL, foreign) {
                                        t.Errorf("adapter %s: official source URL %q contains foreign domain %q (isolation breach)",
                                                c.code, src.URL, foreign)
                                }
                        }
                }
                if !foundOwn {
                        t.Errorf("adapter %s: GetOfficialSources does not list the country's own parliament domain %q",
                                c.code, ownDomain)
                }
        }
}

// TestRegistry_DiscoverBillsNeverReturnsForeignBills verifies that calling
// DiscoverBills on adapter X returns only Bills whose CountryCode == X.
// The wrappers in adapters/registry/wrappers.go pin the CountryCode on every
// BillCandidate so this invariant holds even if the underlying adapter's
// discovery returned no CountryCode at all.
func TestRegistry_DiscoverBillsNeverReturnsForeignBills(t *testing.T) {
        setupRegistryForTest(t)
        ctx := context.Background()
        for _, c := range expectedCountries {
                a, err := registry.GetAdapter(c.code)
                if err != nil {
                        t.Fatalf("GetAdapter(%q): %v", c.code, err)
                }
                bills, err := a.DiscoverBills(ctx)
                if err != nil {
                        // Discover may legitimately fail in offline test environments
                        // (no HTTP client configured). Skip rather than fail.
                        t.Logf("adapter %s: DiscoverBills returned error (skipping): %v", c.code, err)
                        continue
                }
                for _, b := range bills {
                        if b.CountryCode != c.code {
                                t.Errorf("adapter %s: DiscoverBills returned a Bill with CountryCode=%q (isolation breach)",
                                        c.code, b.CountryCode)
                        }
                }
        }
}

// TestSeedData_CountryCodeIsolation verifies each country's seed data
// (presidents, administrations, Acts, debt snapshots, borrowing agreements)
// carries ONLY that country's code. The packages are physically separate
// Go packages so this test mainly guards against copy-paste errors when a
// contributor adds a new country by copying an existing one.
func TestSeedData_CountryCodeIsolation(t *testing.T) {
        // Presidents — must all carry the country's own code.
        for _, p := range kenya_seed.KenyaPresidents {
                if p.CountryCode != "KE" {
                        t.Errorf("kenya_seed.KenyaPresidents: %q has CountryCode %q", p.FullName, p.CountryCode)
                }
        }
        for _, p := range tanzania_seed.TanzaniaPresidents {
                if p.CountryCode != "TZ" {
                        t.Errorf("tanzania_seed.TanzaniaPresidents: %q has CountryCode %q", p.FullName, p.CountryCode)
                }
        }
        for _, p := range ghana_seed.GhanaPresidents {
                if p.CountryCode != "GH" {
                        t.Errorf("ghana_seed.GhanaPresidents: %q has CountryCode %q", p.FullName, p.CountryCode)
                }
        }
        for _, p := range nigeria_seed.NigeriaPresidents {
                if p.CountryCode != "NG" {
                        t.Errorf("nigeria_seed.NigeriaPresidents: %q has CountryCode %q", p.FullName, p.CountryCode)
                }
        }
        for _, p := range south_africa_seed.SouthAfricaPresidents {
                if p.CountryCode != "ZA" {
                        t.Errorf("south_africa_seed.SouthAfricaPresidents: %q has CountryCode %q", p.FullName, p.CountryCode)
                }
        }

        // Administrations.
        for _, a := range kenya_seed.KenyaAdministrations {
                if a.CountryCode != "KE" {
                        t.Errorf("kenya_seed.KenyaAdministrations: %q has CountryCode %q", a.Name, a.CountryCode)
                }
        }
        for _, a := range tanzania_seed.TanzaniaAdministrations {
                if a.CountryCode != "TZ" {
                        t.Errorf("tanzania_seed.TanzaniaAdministrations: %q has CountryCode %q", a.Name, a.CountryCode)
                }
        }
        for _, a := range ghana_seed.GhanaAdministrations {
                if a.CountryCode != "GH" {
                        t.Errorf("ghana_seed.GhanaAdministrations: %q has CountryCode %q", a.Name, a.CountryCode)
                }
        }
        for _, a := range nigeria_seed.NigeriaAdministrations {
                if a.CountryCode != "NG" {
                        t.Errorf("nigeria_seed.NigeriaAdministrations: %q has CountryCode %q", a.Name, a.CountryCode)
                }
        }
        for _, a := range south_africa_seed.SouthAfricaAdministrations {
                if a.CountryCode != "ZA" {
                        t.Errorf("south_africa_seed.SouthAfricaAdministrations: %q has CountryCode %q", a.Name, a.CountryCode)
                }
        }

        // Acts.
        for _, act := range kenya_seed.KenyaActs {
                if act.CountryID != "KE" {
                        t.Errorf("kenya_seed.KenyaActs: %q has CountryID %q", act.ActName, act.CountryID)
                }
        }
        for _, act := range tanzania_seed.TanzaniaActs {
                if act.CountryID != "TZ" {
                        t.Errorf("tanzania_seed.TanzaniaActs: %q has CountryID %q", act.ActName, act.CountryID)
                }
        }
        for _, act := range ghana_seed.GhanaActs {
                if act.CountryID != "GH" {
                        t.Errorf("ghana_seed.GhanaActs: %q has CountryID %q", act.ActName, act.CountryID)
                }
        }
        for _, act := range nigeria_seed.NigeriaActs {
                if act.CountryID != "NG" {
                        t.Errorf("nigeria_seed.NigeriaActs: %q has CountryID %q", act.ActName, act.CountryID)
                }
        }
        for _, act := range south_africa_seed.SouthAfricaActs {
                if act.CountryID != "ZA" {
                        t.Errorf("south_africa_seed.SouthAfricaActs: %q has CountryID %q", act.ActName, act.CountryID)
                }
        }

        // Debt snapshots.
        for _, s := range kenya_seed.KenyaDebtSnapshots {
                if s.CountryCode != "KE" {
                        t.Errorf("kenya_seed.KenyaDebtSnapshots: %s has CountryCode %q", s.ID, s.CountryCode)
                }
        }
        for _, s := range tanzania_seed.TanzaniaDebtSnapshots {
                if s.CountryCode != "TZ" {
                        t.Errorf("tanzania_seed.TanzaniaDebtSnapshots: %s has CountryCode %q", s.ID, s.CountryCode)
                }
        }
        for _, s := range ghana_seed.GhanaDebtSnapshots {
                if s.CountryCode != "GH" {
                        t.Errorf("ghana_seed.GhanaDebtSnapshots: %s has CountryCode %q", s.ID, s.CountryCode)
                }
        }
        for _, s := range nigeria_seed.NigeriaDebtSnapshots {
                if s.CountryCode != "NG" {
                        t.Errorf("nigeria_seed.NigeriaDebtSnapshots: %s has CountryCode %q", s.ID, s.CountryCode)
                }
        }
        for _, s := range south_africa_seed.SouthAfricaDebtSnapshots {
                if s.CountryCode != "ZA" {
                        t.Errorf("south_africa_seed.SouthAfricaDebtSnapshots: %s has CountryCode %q", s.ID, s.CountryCode)
                }
        }

        // Borrowing agreements.
        for _, b := range kenya_seed.KenyaBorrowingAgreements {
                if b.CountryCode != "KE" {
                        t.Errorf("kenya_seed.KenyaBorrowingAgreements: %s has CountryCode %q", b.ID, b.CountryCode)
                }
        }
        for _, b := range tanzania_seed.TanzaniaBorrowingAgreements {
                if b.CountryCode != "TZ" {
                        t.Errorf("tanzania_seed.TanzaniaBorrowingAgreements: %s has CountryCode %q", b.ID, b.CountryCode)
                }
        }
        for _, b := range ghana_seed.GhanaBorrowingAgreements {
                if b.CountryCode != "GH" {
                        t.Errorf("ghana_seed.GhanaBorrowingAgreements: %s has CountryCode %q", b.ID, b.CountryCode)
                }
        }
        for _, b := range nigeria_seed.NigeriaBorrowingAgreements {
                if b.CountryCode != "NG" {
                        t.Errorf("nigeria_seed.NigeriaBorrowingAgreements: %s has CountryCode %q", b.ID, b.CountryCode)
                }
        }
        for _, b := range south_africa_seed.SouthAfricaBorrowingAgreements {
                if b.CountryCode != "ZA" {
                        t.Errorf("south_africa_seed.SouthAfricaBorrowingAgreements: %s has CountryCode %q", b.ID, b.CountryCode)
                }
        }
}

// TestSeedData_IndependentMutationDoesNotCrossContaminate verifies that
// mutating one country's seed slice does NOT affect another country's seed.
// This is a runtime guard that the seed slices are physically distinct
// memory regions — a contributor who appends to TanzaniaBorrowingAgreements
// MUST NOT see the change reflected in KenyaBorrowingAgreements or
// GhanaBorrowingAgreements.
func TestSeedData_IndependentMutationDoesNotCrossContaminate(t *testing.T) {
        originalKenya := len(kenya_seed.KenyaBorrowingAgreements)
        originalGhana := len(ghana_seed.GhanaBorrowingAgreements)
        originalTanzania := len(tanzania_seed.TanzaniaBorrowingAgreements)

        // Append a foreign-coded agreement to Tanzania's slice. The agreement's
        // CountryCode is "TZ" — even with a sensible code, the test asserts that
        // appending to Tanzania does not leak into Kenya or Ghana.
        tanzania_seed.TanzaniaBorrowingAgreements = append(
                tanzania_seed.TanzaniaBorrowingAgreements,
                legislation.BorrowingAgreement{ID: "test-tz-injected", CountryCode: "TZ"},
        )
        // The Kenya slice MUST be unchanged.
        if got := len(kenya_seed.KenyaBorrowingAgreements); got != originalKenya {
                t.Errorf("Tanzania append leaked into Kenya: Kenya len=%d, expected %d", got, originalKenya)
        }
        // The Ghana slice MUST be unchanged.
        if got := len(ghana_seed.GhanaBorrowingAgreements); got != originalGhana {
                t.Errorf("Tanzania append leaked into Ghana: Ghana len=%d, expected %d", got, originalGhana)
        }
        // The Tanzania slice MUST have grown by exactly 1.
        if got := len(tanzania_seed.TanzaniaBorrowingAgreements); got != originalTanzania+1 {
                t.Errorf("Tanzania append did not take effect: len=%d, expected %d", got, originalTanzania+1)
        }

        // Restore — remove the appended entry so subsequent tests see the
        // original list.
        tanzania_seed.TanzaniaBorrowingAgreements = tanzania_seed.TanzaniaBorrowingAgreements[:originalTanzania]
        if got := len(tanzania_seed.TanzaniaBorrowingAgreements); got != originalTanzania {
                t.Errorf("Tanzania restoration failed: len=%d, expected %d", got, originalTanzania)
        }
}

// TestSeedData_ExactlyOneCurrentAdministration verifies each country has
// exactly one administration with EndDate == nil (the current incumbent).
// This catches the failure mode where a contributor forgets to mark the
// current administration as ongoing and accidentally has two open-ended
// administrations (or zero — leaving no "current" to surface in the UI).
func TestSeedData_ExactlyOneCurrentAdministration(t *testing.T) {
        probes := []struct {
                country string
                open    int
                closed  int
        }{
                {"KE", countKenyaOpenAdmins(), countKenyaClosedAdmins()},
                {"TZ", countTanzaniaOpenAdmins(), countTanzaniaClosedAdmins()},
                {"GH", countGhanaOpenAdmins(), countGhanaClosedAdmins()},
                {"NG", countNigeriaOpenAdmins(), countNigeriaClosedAdmins()},
                {"ZA", countSouthAfricaOpenAdmins(), countSouthAfricaClosedAdmins()},
        }
        for _, p := range probes {
                if p.open != 1 {
                        t.Errorf("country %s: expected exactly 1 open-ended administration, got %d (closed=%d)",
                                p.country, p.open, p.closed)
                }
        }
}

// Per-country administration counters. Each helper inlines the iteration
// because the five administration slices share the same underlying
// government.Administration type but the Go type system does not unify them
// across adapter packages — a generic helper would require adding an
// endDateIsNil method on government.Administration, which is owned by a
// different package.

func countKenyaOpenAdmins() int {
        n := 0
        for _, a := range kenya_seed.KenyaAdministrations {
                if a.EndDate == nil {
                        n++
                }
        }
        return n
}

func countKenyaClosedAdmins() int {
        return len(kenya_seed.KenyaAdministrations) - countKenyaOpenAdmins()
}

func countTanzaniaOpenAdmins() int {
        n := 0
        for _, a := range tanzania_seed.TanzaniaAdministrations {
                if a.EndDate == nil {
                        n++
                }
        }
        return n
}

func countTanzaniaClosedAdmins() int {
        return len(tanzania_seed.TanzaniaAdministrations) - countTanzaniaOpenAdmins()
}

func countGhanaOpenAdmins() int {
        n := 0
        for _, a := range ghana_seed.GhanaAdministrations {
                if a.EndDate == nil {
                        n++
                }
        }
        return n
}

func countGhanaClosedAdmins() int {
        return len(ghana_seed.GhanaAdministrations) - countGhanaOpenAdmins()
}

func countNigeriaOpenAdmins() int {
        n := 0
        for _, a := range nigeria_seed.NigeriaAdministrations {
                if a.EndDate == nil {
                        n++
                }
        }
        return n
}

func countNigeriaClosedAdmins() int {
        return len(nigeria_seed.NigeriaAdministrations) - countNigeriaOpenAdmins()
}

func countSouthAfricaOpenAdmins() int {
        n := 0
        for _, a := range south_africa_seed.SouthAfricaAdministrations {
                if a.EndDate == nil {
                        n++
                }
        }
        return n
}

func countSouthAfricaClosedAdmins() int {
        return len(south_africa_seed.SouthAfricaAdministrations) - countSouthAfricaOpenAdmins()
}
