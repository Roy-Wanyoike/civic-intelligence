// Package registry — wrapper types that adapt the existing
// contracts.LegislativeSourceAdapter implementations (kenya.KenyaAdapter,
// uganda.UgandaAdapter, ...) to the CountryAdapter interface defined in
// registry.go.
//
// Why wrappers? The existing adapters ship with two slightly different
// method shapes:
//   - LegislativeSourceAdapter.GetStages returns ([]StageDefinition, error)
//   - LegislativeSourceAdapter.GetLegislativeStructure returns (*LegislativeStructure, error)
//
// The CountryAdapter interface collapses these to value-only returns because
// (a) the underlying adapters always return nil errors for these data
// lookups (the data is statically defined per country) and (b) the registry
// pattern is read-heavy and benefits from non-erroring accessors.
//
// The wrappers also add the metadata methods (CountryName, FlagEmoji,
// ParliamentName, LegislatureType, GetOfficialSources) that do not exist on
// the underlying adapters — these are stable per-country strings that
// belong with the adapter but have not yet been promoted to the
// contracts.LegislativeSourceAdapter interface.
//
// Each wrapper is intentionally a thin pass-through so the existing adapter
// packages (and their tests) remain the single source of truth.
package registry

import (
        "context"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// ---------------------------------------------------------------------------
// Generic helpers
// ---------------------------------------------------------------------------

// billsFromSourceItems projects the discovery output of an existing adapter
// (which returns []contracts.SourceItem) into the country-agnostic
// BillCandidate shape used by the registry. Only items whose SourceType is
// SourceItemBill are surfaced — gazettes, hansards, committee reports etc.
// are filtered out so the returned slice is purely a list of Bills.
func billsFromSourceItems(countryCode string, items []contracts.SourceItem) []BillCandidate {
        out := make([]BillCandidate, 0, len(items))
        for _, item := range items {
                if item.SourceType != contracts.SourceItemBill && item.DocumentType != "bill" {
                        continue
                }
                out = append(out, BillCandidate{
                        Title:       item.Title,
                        Number:      item.ExternalID,
                        Stage:       "",
                        House:       item.House,
                        SourceURL:   item.URL,
                        CountryCode: countryCode,
                })
        }
        return out
}

// ---------------------------------------------------------------------------
// Kenya wrapper
// ---------------------------------------------------------------------------

// kenyaWrapper adapts kenya.KenyaAdapter to CountryAdapter. Country-specific
// metadata (flag emoji, parliament name, legislature type, official sources)
// is hard-coded here because it is stable per country and not yet promoted
// to the contracts.LegislativeSourceAdapter interface.
type kenyaWrapper struct {
        inner *kenya.KenyaAdapter
}

func (w kenyaWrapper) CountryCode() string  { return "KE" }
func (w kenyaWrapper) CountryName() string  { return "Kenya" }
func (w kenyaWrapper) FlagEmoji() string    { return "🇰🇪" }
func (w kenyaWrapper) ParliamentName() string {
        return "Parliament of Kenya"
}
func (w kenyaWrapper) LegislatureType() string { return "bicameral" }

func (w kenyaWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w kenyaWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w kenyaWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "Parliament of Kenya", URL: "https://www.parliament.go.ke/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Kenya Law", URL: "https://www.kenyalaw.org/", AuthorityLevel: "primary", ItemType: "act"},
                {Name: "Kenya Gazette", URL: "https://gazettes.africa/kenya/", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "Central Bank of Kenya", URL: "https://www.centralbank.go.ke/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "The National Treasury", URL: "https://www.treasury.go.ke/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w kenyaWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("KE", items), nil
}

// ---------------------------------------------------------------------------
// Uganda wrapper
// ---------------------------------------------------------------------------

type ugandaWrapper struct {
        inner *uganda.UgandaAdapter
}

func (w ugandaWrapper) CountryCode() string  { return "UG" }
func (w ugandaWrapper) CountryName() string  { return "Uganda" }
func (w ugandaWrapper) FlagEmoji() string    { return "🇺🇬" }
func (w ugandaWrapper) ParliamentName() string {
        return "Parliament of Uganda"
}
func (w ugandaWrapper) LegislatureType() string { return "unicameral" }

func (w ugandaWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w ugandaWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w ugandaWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "Parliament of Uganda", URL: "https://www.parliament.go.ug/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Uganda Gazette", URL: "https://ulii.org/", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "Bank of Uganda", URL: "https://www.bou.or.ug/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "Ministry of Finance, Planning and Economic Development", URL: "https://www.finance.go.ug/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w ugandaWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("UG", items), nil
}

// ---------------------------------------------------------------------------
// Tanzania wrapper
// ---------------------------------------------------------------------------

type tanzaniaWrapper struct {
        inner *tanzania.TanzaniaAdapter
}

func (w tanzaniaWrapper) CountryCode() string  { return "TZ" }
func (w tanzaniaWrapper) CountryName() string  { return "Tanzania" }
func (w tanzaniaWrapper) FlagEmoji() string    { return "🇹🇿" }
func (w tanzaniaWrapper) ParliamentName() string {
        return "Bunge la Tanzania"
}
func (w tanzaniaWrapper) LegislatureType() string { return "unicameral" }

func (w tanzaniaWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w tanzaniaWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w tanzaniaWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "Parliament of Tanzania", URL: "https://www.parliament.go.tz/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Government Gazette of Tanzania", URL: "https://www.parliament.go.tz/gazette", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "Bank of Tanzania", URL: "https://www.bot.go.tz/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "Ministry of Finance and Planning", URL: "https://www.mof.go.tz/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w tanzaniaWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("TZ", items), nil
}

// ---------------------------------------------------------------------------
// Ghana wrapper
// ---------------------------------------------------------------------------

type ghanaWrapper struct {
        inner *ghana.GhanaAdapter
}

func (w ghanaWrapper) CountryCode() string  { return "GH" }
func (w ghanaWrapper) CountryName() string  { return "Ghana" }
func (w ghanaWrapper) FlagEmoji() string    { return "🇬🇭" }
func (w ghanaWrapper) ParliamentName() string {
        return "Parliament of Ghana"
}
func (w ghanaWrapper) LegislatureType() string { return "unicameral" }

func (w ghanaWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w ghanaWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w ghanaWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "Parliament of Ghana", URL: "https://parliament.gh/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Ghana Gazette", URL: "https://www.egl.gov.gh/", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "Bank of Ghana", URL: "https://www.bog.gov.gh/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "Ministry of Finance", URL: "https://mofep.gov.gh/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w ghanaWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("GH", items), nil
}

// ---------------------------------------------------------------------------
// Nigeria wrapper
// ---------------------------------------------------------------------------

type nigeriaWrapper struct {
        inner *nigeria.NigeriaAdapter
}

func (w nigeriaWrapper) CountryCode() string  { return "NG" }
func (w nigeriaWrapper) CountryName() string  { return "Nigeria" }
func (w nigeriaWrapper) FlagEmoji() string    { return "🇳🇬" }
func (w nigeriaWrapper) ParliamentName() string {
        return "National Assembly of Nigeria"
}
func (w nigeriaWrapper) LegislatureType() string { return "bicameral" }

func (w nigeriaWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w nigeriaWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w nigeriaWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "National Assembly of Nigeria", URL: "https://nass.gov.ng/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Nigeria Gazette", URL: "https://www.federalgovernmentpress.gov.ng/", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "Central Bank of Nigeria", URL: "https://www.cbn.gov.ng/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "Federal Ministry of Finance", URL: "https://www.fmf.gov.ng/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w nigeriaWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("NG", items), nil
}

// ---------------------------------------------------------------------------
// South Africa wrapper
// ---------------------------------------------------------------------------

type southAfricaWrapper struct {
        inner *south_africa.SouthAfricaAdapter
}

func (w southAfricaWrapper) CountryCode() string  { return "ZA" }
func (w southAfricaWrapper) CountryName() string  { return "South Africa" }
func (w southAfricaWrapper) FlagEmoji() string    { return "🇿🇦" }
func (w southAfricaWrapper) ParliamentName() string {
        return "Parliament of South Africa"
}
func (w southAfricaWrapper) LegislatureType() string { return "bicameral" }

func (w southAfricaWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w southAfricaWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w southAfricaWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "Parliament of South Africa", URL: "https://www.parliament.gov.za/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Government Gazette of South Africa", URL: "https://www.gov.za/documents", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "South African Reserve Bank", URL: "https://www.resbank.co.za/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "National Treasury", URL: "https://www.treasury.gov.za/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w southAfricaWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("ZA", items), nil
}

// ---------------------------------------------------------------------------
// Rwanda wrapper (added in Wave 12 / ENG-L1)
// ---------------------------------------------------------------------------

type rwandaWrapper struct {
        inner *rwanda.RwandaAdapter
}

func (w rwandaWrapper) CountryCode() string  { return "RW" }
func (w rwandaWrapper) CountryName() string  { return "Rwanda" }
func (w rwandaWrapper) FlagEmoji() string    { return "🇷🇼" }
func (w rwandaWrapper) ParliamentName() string {
        return "Parliament of Rwanda"
}
func (w rwandaWrapper) LegislatureType() string { return "bicameral" }

func (w rwandaWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w rwandaWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w rwandaWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "Parliament of Rwanda", URL: "https://www.parliament.gov.rw/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Official Gazette of the Republic of Rwanda", URL: "https://www.parliament.gov.rw/gazette", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "National Bank of Rwanda", URL: "https://www.bnr.rw/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "Ministry of Finance and Economic Planning", URL: "https://www.minecofin.gov.rw/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w rwandaWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("RW", items), nil
}

// ---------------------------------------------------------------------------
// Zambia wrapper (added in Wave 12 / ENG-L1)
// ---------------------------------------------------------------------------

type zambiaWrapper struct {
        inner *zambia.ZambiaAdapter
}

func (w zambiaWrapper) CountryCode() string  { return "ZM" }
func (w zambiaWrapper) CountryName() string  { return "Zambia" }
func (w zambiaWrapper) FlagEmoji() string    { return "🇿🇲" }
func (w zambiaWrapper) ParliamentName() string {
        return "National Assembly of Zambia"
}
func (w zambiaWrapper) LegislatureType() string { return "unicameral" }

func (w zambiaWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w zambiaWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w zambiaWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "National Assembly of Zambia", URL: "https://www.parliament.gov.zm/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Government Gazette of Zambia", URL: "https://www.parliament.gov.zm/gazette", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "Bank of Zambia", URL: "https://www.boz.zm/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "Ministry of Finance and National Planning", URL: "https://www.mofnp.gov.zm/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w zambiaWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("ZM", items), nil
}

// ---------------------------------------------------------------------------
// Senegal wrapper (added in Wave 12 / ENG-L1)
// ---------------------------------------------------------------------------

type senegalWrapper struct {
        inner *senegal.SenegalAdapter
}

func (w senegalWrapper) CountryCode() string  { return "SN" }
func (w senegalWrapper) CountryName() string  { return "Senegal" }
func (w senegalWrapper) FlagEmoji() string    { return "🇸🇳" }
func (w senegalWrapper) ParliamentName() string {
        return "Assemblée Nationale du Sénégal"
}
func (w senegalWrapper) LegislatureType() string { return "unicameral" }

func (w senegalWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w senegalWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w senegalWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "Assemblée Nationale du Sénégal", URL: "https://www.assemblee-nationale.sn/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Journal Officiel du Sénégal", URL: "https://www.jo.gouv.sn/", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "Banque Centrale des États de l'Afrique de l'Ouest (BCEAO)", URL: "https://www.bceao.int/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "Ministère de l'Économie et des Finances", URL: "https://www.finances.gouv.sn/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w senegalWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("SN", items), nil
}

// ---------------------------------------------------------------------------
// Egypt wrapper (added in Wave 12 / ENG-L1)
// ---------------------------------------------------------------------------

type egyptWrapper struct {
        inner *egypt.EgyptAdapter
}

func (w egyptWrapper) CountryCode() string  { return "EG" }
func (w egyptWrapper) CountryName() string  { return "Egypt" }
func (w egyptWrapper) FlagEmoji() string    { return "🇪🇬" }
func (w egyptWrapper) ParliamentName() string {
        return "Egyptian Parliament"
}
func (w egyptWrapper) LegislatureType() string { return "bicameral" }

func (w egyptWrapper) GetStages() []contracts.StageDefinition {
        stages, err := w.inner.GetStages(context.Background())
        if err != nil {
                return nil
        }
        return stages
}

func (w egyptWrapper) GetLegislativeStructure() contracts.LegislativeStructure {
        s, err := w.inner.GetLegislativeStructure(context.Background())
        if err != nil || s == nil {
                return contracts.LegislativeStructure{}
        }
        return *s
}

func (w egyptWrapper) GetOfficialSources() []SourceDefinition {
        return []SourceDefinition{
                {Name: "Egyptian Parliament", URL: "https://www.parliament.eg/", AuthorityLevel: "primary", ItemType: "bill"},
                {Name: "Official Gazette of the Arab Republic of Egypt", URL: "https://www.parliament.eg/gazette", AuthorityLevel: "official", ItemType: "gazette"},
                {Name: "Central Bank of Egypt", URL: "https://www.cbe.org.eg/", AuthorityLevel: "primary", ItemType: "policy"},
                {Name: "Ministry of Finance", URL: "https://www.mof.gov.eg/", AuthorityLevel: "primary", ItemType: "policy"},
        }
}

func (w egyptWrapper) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
        items, err := w.inner.Discover(ctx)
        if err != nil {
                return nil, err
        }
        return billsFromSourceItems("EG", items), nil
}

// ---------------------------------------------------------------------------
// Default-adapter constructors
// ---------------------------------------------------------------------------

// NewKenyaWrapper returns a CountryAdapter for the supplied Kenya adapter.
// Exposed so callers (e.g. main.go) can construct a fresh adapter with a
// custom HTTP client and immediately register it via Register(...).
func NewKenyaWrapper(inner *kenya.KenyaAdapter) CountryAdapter {
        return kenyaWrapper{inner: inner}
}

// NewUgandaWrapper returns a CountryAdapter for the supplied Uganda adapter.
func NewUgandaWrapper(inner *uganda.UgandaAdapter) CountryAdapter {
        return ugandaWrapper{inner: inner}
}

// NewTanzaniaWrapper returns a CountryAdapter for the supplied Tanzania adapter.
func NewTanzaniaWrapper(inner *tanzania.TanzaniaAdapter) CountryAdapter {
        return tanzaniaWrapper{inner: inner}
}

// NewGhanaWrapper returns a CountryAdapter for the supplied Ghana adapter.
func NewGhanaWrapper(inner *ghana.GhanaAdapter) CountryAdapter {
        return ghanaWrapper{inner: inner}
}

// NewNigeriaWrapper returns a CountryAdapter for the supplied Nigeria adapter.
func NewNigeriaWrapper(inner *nigeria.NigeriaAdapter) CountryAdapter {
        return nigeriaWrapper{inner: inner}
}

// NewSouthAfricaWrapper returns a CountryAdapter for the supplied South Africa adapter.
func NewSouthAfricaWrapper(inner *south_africa.SouthAfricaAdapter) CountryAdapter {
        return southAfricaWrapper{inner: inner}
}

// NewRwandaWrapper returns a CountryAdapter for the supplied Rwanda adapter.
func NewRwandaWrapper(inner *rwanda.RwandaAdapter) CountryAdapter {
        return rwandaWrapper{inner: inner}
}

// NewZambiaWrapper returns a CountryAdapter for the supplied Zambia adapter.
func NewZambiaWrapper(inner *zambia.ZambiaAdapter) CountryAdapter {
        return zambiaWrapper{inner: inner}
}

// NewSenegalWrapper returns a CountryAdapter for the supplied Senegal adapter.
func NewSenegalWrapper(inner *senegal.SenegalAdapter) CountryAdapter {
        return senegalWrapper{inner: inner}
}

// NewEgyptWrapper returns a CountryAdapter for the supplied Egypt adapter.
func NewEgyptWrapper(inner *egypt.EgyptAdapter) CountryAdapter {
        return egyptWrapper{inner: inner}
}

// MustRegisterDefault registers the 10 default country adapters (Kenya,
// Uganda, Tanzania, Ghana, Nigeria, South Africa, Rwanda, Zambia, Senegal,
// Egypt) using fresh adapter instances constructed with zero-value
// Dependencies. It is safe to call multiple times — subsequent calls are
// no-ops once the adapters are registered.
//
// Production wiring: call this once at startup in services/api/cmd/main.go.
// Test wiring: the country-isolation test calls ResetForTest() and then
// MustRegisterDefault() so it sees a known registry state.
//
// Wave 12 (ENG-L1, 2026) added the final 4 adapters — Rwanda (RW), Zambia
// (ZM), Senegal (SN), and Egypt (EG) — bringing the platform from 6 to 10
// supported countries.
func MustRegisterDefault() {
        registryMu.Lock()
        defer registryMu.Unlock()
        if _, ok := registry["KE"]; !ok {
                registry["KE"] = kenyaWrapper{inner: kenya.NewKenyaAdapter(kenya.Dependencies{})}
        }
        if _, ok := registry["UG"]; !ok {
                registry["UG"] = ugandaWrapper{inner: uganda.NewUgandaAdapter()}
        }
        if _, ok := registry["TZ"]; !ok {
                registry["TZ"] = tanzaniaWrapper{inner: tanzania.NewTanzaniaAdapter()}
        }
        if _, ok := registry["GH"]; !ok {
                registry["GH"] = ghanaWrapper{inner: ghana.NewGhanaAdapter()}
        }
        if _, ok := registry["NG"]; !ok {
                registry["NG"] = nigeriaWrapper{inner: nigeria.NewNigeriaAdapter(nigeria.Dependencies{})}
        }
        if _, ok := registry["ZA"]; !ok {
                registry["ZA"] = southAfricaWrapper{inner: south_africa.NewSouthAfricaAdapter(south_africa.Dependencies{})}
        }
        if _, ok := registry["RW"]; !ok {
                registry["RW"] = rwandaWrapper{inner: rwanda.NewRwandaAdapter(rwanda.Dependencies{})}
        }
        if _, ok := registry["ZM"]; !ok {
                registry["ZM"] = zambiaWrapper{inner: zambia.NewZambiaAdapter(zambia.Dependencies{})}
        }
        if _, ok := registry["SN"]; !ok {
                registry["SN"] = senegalWrapper{inner: senegal.NewSenegalAdapter(senegal.Dependencies{})}
        }
        if _, ok := registry["EG"]; !ok {
                registry["EG"] = egyptWrapper{inner: egypt.NewEgyptAdapter(egypt.Dependencies{})}
        }
}
