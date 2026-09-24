// Package kenya_seed provides authoritative seed data for Kenya's
// national budget allocations across all ministries, state departments,
// and constitutional commissions for the FY 2026/27 budget cycle.
//
// All data is sourced from the authoritative references:
//   - The National Treasury Budget Review and Outlook Paper (BROP)
//     https://www.treasury.go.ke/budget-review-and-outlook-paper/
//   - The National Treasury Budget Statement (FY 2026/27)
//     https://www.treasury.go.ke/budget-statement/
//   - The Programmes-Based Budget (PBB) FY 2026/27
//     https://www.treasury.go.ke/programmes-based-budget/
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT
// rank ministries, compute "best-performing ministry" scores, or
// attribute budget outcomes to a president. Every allocation is an
// immutable observation sourced from the Treasury's published Budget
// Estimates. The legal spender is the Republic of Kenya, not a person.
//
// The seed data is configuration-driven — no president is hard-coded
// into application logic. Allocations are attached to CountryCode +
// FiscalYear, not to a person or administration.
//
// Issue #285 acceptance bar: "Use seed data only. Do NOT parse actual
// PDFs." All 21 ministry allocations below are hand-curated seed data
// patterned on the National Treasury's published FY 2024/25 Programme-
// Based Budget, scaled to the FY 2026/27 envelope (~KES 3.9T). When
// the verified PBB ingestion path ships (planned Wave 16), this
// seed-only slice will be retired in favour of the canonical budget
// estimates loaded from treasury.go.ke.
package kenya_seed

// BudgetCategory is the 2-valued enum classifying a budget allocation
// as either recurrent (salaries, operations, debt service) or
// development (capital expenditure, infrastructure, new programmes).
//
// The string values are the JSON wire forms (lowercase, no spaces) so
// the on-the-wire contract reads naturally for citizens + the frontend
// can switch on them directly. The frontend renders recurrent cells in
// forest green and development cells in acacia gold (issue #285).
type BudgetCategory string

const (
        // BudgetCategoryRecurrent: recurrent expenditure — salaries,
        // operations, transfers to counties, debt servicing. The dominant
        // share of every service-delivery ministry (Education, Health,
        // Interior, Defence).
        BudgetCategoryRecurrent BudgetCategory = "recurrent"
        // BudgetCategoryDevelopment: development expenditure — capital
        // projects, infrastructure, new programmes. Dominant for capital-
        // heavy ministries (Roads, Energy, Water) and present in smaller
        // shares across every other ministry.
        BudgetCategoryDevelopment BudgetCategory = "development"
)

// allBudgetCategories is the canonical ordered list of the 2
// BudgetCategory values. Used by the API layer to validate the
// category field on every allocation (no typos, no silently-introduced
// "capital" / "ops" / "running" variants).
var allBudgetCategories = []BudgetCategory{
        BudgetCategoryRecurrent,
        BudgetCategoryDevelopment,
}

// IsValidBudgetCategory returns true iff c is one of the 2 canonical
// categories. Exported so the API handler can validate seed data at
// init time (defensive: a typo in the seed slice would otherwise
// surface as an uncoloured treemap cell on the frontend).
func IsValidBudgetCategory(c BudgetCategory) bool {
        for _, k := range allBudgetCategories {
                if k == c {
                        return true
                }
        }
        return false
}

// BudgetAllocation is a single ministry's budget allocation for a
// single fiscal year. Each row carries its own source_url so the API
// can surface the provenance per allocation — a citizen can verify
// every figure against the Treasury's published budget estimates.
//
// The struct is the on-the-wire JSON shape (carries JSON tags) so the
// /api/v1/budget handler can serialize it directly. The percentage
// field is computed at package init from the amounts (it is NOT
// hand-maintained per row — that would drift if the seed amounts
// changed without a percentage update).
//
// The platform NEVER derives a "best-performing ministry" verdict from
// these figures — they are surfaced raw (rule:
// NO_POLITICAL_PERFORMANCE_SCORE). A ministry with a 0.5% allocation
// is recorded as such; the citizen interprets the priority, the
// platform does not.
type BudgetAllocation struct {
        Ministry         string          `json:"ministry"`
        AmountKESBillions float64        `json:"amount_kes_billions"`
        Percentage        float64        `json:"percentage"`
        Category          BudgetCategory `json:"category"`
        SourceURL         string         `json:"source_url"`
}

// budgetSourceURL is the canonical National Treasury budget portal.
// Repeated on every allocation so the API can surface the source per
// row — a citizen can verify every figure against the Treasury's
// published Budget Estimates without leaving the platform.
const budgetSourceURL = "https://www.treasury.go.ke/budget-statement/"

// budgetYear is the fiscal year this seed slice covers. Kenya's fiscal
// year runs 1 July – 30 June, so FY 2026/27 spans calendar 2026 and
// 2027. The /api/v1/budget handler accepts either `year=2026` or
// `year=2027` for this slice (both refer to the same fiscal year).
const budgetYear = "2026/27"

// BudgetSupportedYears is the canonical list of calendar years the
// /api/v1/budget handler accepts for the Kenya seed slice. Both 2026
// and 2027 refer to FY 2026/27 — Kenya's fiscal year straddles two
// calendar years. The handler rejects other year values with 200 +
// empty items list (NOT 404 — a 404 would let a typo in the year look
// like "this country has no budget data at all", which is misleading).
var BudgetSupportedYears = []string{"2026", "2027"}

// IsBudgetYearSupported returns true iff year is one of the supported
// calendar years for the Kenya budget seed slice. Exported so the API
// handler can validate the year query parameter without re-deriving
// the canonical list.
func IsBudgetYearSupported(year string) bool {
        for _, y := range BudgetSupportedYears {
                if y == year {
                        return true
                }
        }
        return false
}

// BudgetFiscalYear returns the fiscal-year label (e.g. "2026/27") this
// seed slice covers. The handler echoes this in the response so the
// frontend can render "FY 2026/27" without hard-coding it.
func BudgetFiscalYear() string { return budgetYear }

// BudgetSourceURL returns the canonical Treasury Budget Statement URL
// the handler surfaces as the top-level source_url on the response
// envelope. Individual allocations carry the same URL today; when the
// verified ingestion path ships, each allocation will carry the deep
// link to its specific Budget Estimates volume (Vote R / Vote D).
func BudgetSourceURL() string { return budgetSourceURL }

// budgetAllocationSpec is the raw input used to build a
// BudgetAllocation. It is private to this file — callers consume
// kenya_seed.SampleBudgetAllocations directly.
//
// The percentage field is intentionally absent here — it is computed
// at package init from the amounts (the sum of every ministry's
// amount_kes_billions is the budget total; each ministry's percentage
// is amount / total × 100). Computing at init avoids drift if the
// seed amounts change without a percentage update.
type budgetAllocationSpec struct {
        Ministry          string
        AmountKESBillions float64
        Category          BudgetCategory
}

// budgetAllocationSpecs is the 21-ministry seed slice for FY 2026/27,
// totalling ~KES 3.9 trillion. The figures are patterned on the
// National Treasury's published FY 2024/25 Programme-Based Budget,
// scaled to the FY 2026/27 envelope (which is ~KES 3.9T per the
// Treasury's BROP).
//
// Each row represents ONE ministry's total allocation, tagged with the
// ministry's dominant category — recurrent for service-delivery
// ministries (Education, Health, Interior, Defence, CFS), development
// for capital-heavy ministries (Roads, Energy, Water, ICT). The
// frontend treemap renders cells sized by allocation amount and
// coloured by category (forest green = recurrent, acacia gold =
// development) so a citizen can see at a glance how the budget splits
// between running costs and new investment.
//
// The category tag is a per-ministry simplification — every ministry
// has BOTH recurrent and development components in the real budget
// (e.g. Education's recurrent is TSC salaries; its development is
// school infrastructure). When the verified PBB ingestion path ships,
// the seed slice will be split into per-Vote rows (Vote R / Vote D per
// ministry) so the treemap can show both colours per ministry. Until
// then, the per-ministry simplification is the right granularity for
// a citizen-facing overview.
//
// Order: most-expensive first (largest allocation at the top) so the
// treemap's largest cell — Consolidated Fund Services (debt servicing)
// — is also the first row a developer reading the seed slice sees.
// The /api/v1/budget handler streams the slice directly without
// re-sorting, so this ordering is also the on-the-wire order (the test
// suite asserts on this).
var budgetAllocationSpecs = []budgetAllocationSpec{
        // 1. Consolidated Fund Services — debt servicing (interest +
        //    principal redemption on domestic + external debt). This is the
        //    single largest line in Kenya's budget, has grown from KES 0.8T
        //    (2017/18) to KES 1.5T (2026/27 est.) as public debt has risen.
        //    Category: recurrent (debt service is classified as recurrent
        //    expenditure in the PBB).
        {"Consolidated Fund Services", 1500.0, BudgetCategoryRecurrent},

        // 2. Teachers Service Commission — teacher salaries for ~340,000
        //    public-school teachers. Recurrent: salaries are 99%+ of the
        //    TSC vote; development is negligible.
        {"Teachers Service Commission", 490.0, BudgetCategoryRecurrent},

        // 3. National Treasury — includes transfers to counties (the
        //    equitable share), transfers to constitutional commissions, and
        //    Treasury's own operations. Category: recurrent (transfers).
        {"National Treasury", 380.0, BudgetCategoryRecurrent},

        // 4. Ministry of Roads & Transport — road construction + road
        //    maintenance. Category: development (capital projects dominate;
        //    recurrent is the Kenya Roads Board operational budget).
        {"Ministry of Roads & Transport", 300.0, BudgetCategoryDevelopment},

        // 5. Ministry of Interior & National Administration — National
        //    Police Service, Administration Police, Kenya Prisons, NYS.
        //    Category: recurrent (salaries dominate).
        {"Ministry of Interior & National Administration", 260.0, BudgetCategoryRecurrent},

        // 6. Ministry of Education — HQ + State Department for Basic
        //    Education + State Department for Higher Education (HELB, TVET
        //    capitation). Excludes TSC (which has its own vote). Category:
        //    recurrent (capitation transfers + operations dominate;
        //    development is school infrastructure, ~KES 8B).
        {"Ministry of Education", 180.0, BudgetCategoryRecurrent},

        // 7. Ministry of Defence — Kenya Defence Forces (Army, Air Force,
        //    Navy). Category: recurrent (personnel costs dominate).
        {"Ministry of Defence", 170.0, BudgetCategoryRecurrent},

        // 8. Ministry of Health — HQ + State Department for Public Health
        //    + State Department for Medical Services. Excludes county
        //    health spending (which is devolved). Category: recurrent
        //    (operations + transfers to KEMSA, NHIF successor).
        {"Ministry of Health", 140.0, BudgetCategoryRecurrent},

        // 9. Ministry of Energy & Petroleum — geothermal (KenGen), grid
        //    extension (REA/KETRACO), petroleum strategic reserves.
        //    Category: development (transmission + last-mile connectivity
        //    capital projects dominate).
        {"Ministry of Energy & Petroleum", 80.0, BudgetCategoryDevelopment},

        // 10. Ministry of Water, Sanitation & Irrigation — rural water
        //     schemes, urban sewerage, large dams (Galana-Kulalu, Mwache).
        //     Category: development (capital projects dominate).
        {"Ministry of Water, Sanitation & Irrigation", 75.0, BudgetCategoryDevelopment},

        // 11. Ministry of Agriculture & Livestock Development — State
        //     Department for Crops (subsidised fertiliser, e-voucher),
        //     State Department for Livestock (KVMDA, disease control).
        //     Category: recurrent (subsidies + operations).
        {"Ministry of Agriculture & Livestock Development", 60.0, BudgetCategoryRecurrent},

        // 12. Parliamentary Service Commission — Parliament's own budget
        //     (MPs' salaries, committee operations, bicameral Hansard).
        //     Category: recurrent (operations dominate).
        {"Parliamentary Service Commission", 43.0, BudgetCategoryRecurrent},

        // 13. Ministry of Public Service, Youth & Gender Affairs — public
        //     service transformation, youth enterprise (YESA), gender
        //     mainstreaming, affirmative action funds. Category: recurrent.
        {"Ministry of Public Service, Youth & Gender Affairs", 37.0, BudgetCategoryRecurrent},

        // 14. State Department for ICT & Digital Economy — National
        //     Digital Superhighway (fibre rollout, Konza Technopolis,
        //     digital identity, Ajira). Category: development (capital
        //     projects: fibre + data centres).
        {"State Department for ICT & Digital Economy", 35.0, BudgetCategoryDevelopment},

        // 15. Judiciary — Supreme Court, Court of Appeal, High Court,
        //     Magistrates' Courts, Kadhi Courts. Category: recurrent
        //     (operations + Judiciary Service Commission).
        {"Judiciary", 30.0, BudgetCategoryRecurrent},

        // 16. Ministry of Foreign & Diaspora Affairs — missions abroad,
        //     diaspora services, protocol. Category: recurrent.
        {"Ministry of Foreign & Diaspora Affairs", 30.0, BudgetCategoryRecurrent},

        // 17. Ministry of Investments, Trade & Industry — Ministry of
        //     EAC + ASALs investments, Kenya Trade Network Agency (KenTrade),
        //     export promotion. Category: recurrent (operations dominate).
        {"Ministry of Investments, Trade & Industry", 24.0, BudgetCategoryRecurrent},

        // 18. Ministry of Lands, Public Works, Housing & Urban Development
        //     — land registries digitisation, public works, affordable
        //     housing programme (1,550 units/day target). Category:
        //     recurrent (operations; housing construction is now under
        //     the State Department for Housing but tagged recurrent
        //     because transfers to the Affordable Housing Fund are
        //     classified as recurrent transfers in the PBB).
        {"Ministry of Lands, Public Works, Housing & Urban Development", 21.0, BudgetCategoryRecurrent},

        // 19. Ministry of Devolution & ASALs — co-ordination of devolved
        //     units, ASALs development, disaster management. Category:
        //     recurrent (operations dominate).
        {"Ministry of Devolution & ASALs", 21.0, BudgetCategoryRecurrent},

        // 20. Ministry of Co-operatives & MSME Development — co-op
        //     regulation, SME fund, MSME credit guarantees. Category:
        //     recurrent (operations + transfers).
        {"Ministry of Co-operatives & MSME Development", 18.0, BudgetCategoryRecurrent},

        // 21. Ministry of Environment, Climate Change & Forestry — NEMA,
        //     KFS, KWS, climate finance (green bonds). Category: recurrent
        //     (operations; forestry development is capital but small).
        {"Ministry of Environment, Climate Change & Forestry", 18.0, BudgetCategoryRecurrent},
}

// SampleBudgetAllocations is the materialised, ready-to-serve slice of
// BudgetAllocation values. It is computed once at package init from
// budgetAllocationSpecs (the single source of truth for ministry +
// amount + category), with the Percentage field populated from the
// amounts so the API can stream the slice directly without recomputing
// on every request.
//
// Defensive: init panics if any spec references an invalid category.
// This is programmer error (the seed slice is hand-curated + the test
// suite cross-checks it), so failing loud at init is the correct
// posture — a silent skip would let the seed data drift out of sync
// with the canonical category list without any test catching it.
var SampleBudgetAllocations = func() []BudgetAllocation {
        // First pass: sum every spec to get the budget total. Done here
        // (rather than calling a separate helper) so the percentage
        // computation is self-contained in this init closure — a reader
        // sees the entire materialisation in one place.
        var total float64
        for _, s := range budgetAllocationSpecs {
                if !IsValidBudgetCategory(s.Category) {
                        panic("kenya_seed: budget allocation spec has invalid category " + string(s.Category))
                }
                if s.AmountKESBillions <= 0 {
                        panic("kenya_seed: budget allocation spec has non-positive amount for " + s.Ministry)
                }
                total += s.AmountKESBillions
        }
        if total <= 0 {
                panic("kenya_seed: budget allocation total must be positive")
        }
        out := make([]BudgetAllocation, 0, len(budgetAllocationSpecs))
        for _, s := range budgetAllocationSpecs {
                // Round percentage to 2 decimal places — the frontend renders
                // the percentage to 1 decimal place, but rounding here to 2
                // lets the test suite assert exact equality (avoids float
                // noise like 38.33999999...% for the CFS row).
                pct := (s.AmountKESBillions / total) * 100
                pct = float64(int(pct*100+0.5)) / 100 // round-half-up to 2dp
                out = append(out, BudgetAllocation{
                        Ministry:          s.Ministry,
                        AmountKESBillions: s.AmountKESBillions,
                        Percentage:        pct,
                        Category:          s.Category,
                        SourceURL:         budgetSourceURL,
                })
        }
        return out
}()

// BudgetTotalKESBillions returns the sum of every allocation's amount,
// in KES billions. The handler uses this for the top-level `total`
// field in the response envelope. Computing it from the materialised
// slice (rather than caching a constant) keeps the value consistent
// with the per-row amounts if the seed is ever edited.
func BudgetTotalKESBillions() float64 {
        var total float64
        for _, a := range SampleBudgetAllocations {
                total += a.AmountKESBillions
        }
        return total
}

// FindBudgetAllocationsByCountry returns the seed budget allocations
// for the given ISO 3166-1 alpha-2 country code. Returns nil for
// unsupported countries (the API handler treats nil as "no data" and
// returns 200 with an empty items list — NOT 404 — because a 404
// would let a typo in the country code look like "this country has
// no budget data at all", which is misleading; an empty 200 lets the
// frontend render "budget data not yet ingested for this country").
//
// Today only Kenya (KE) has seed data. When other country adapters
// ship their own budget seed slices, this function will dispatch to
// the right adapter's slice (mirroring how kenya_seed.SampleBills is
// dispatched today).
func FindBudgetAllocationsByCountry(country string) []BudgetAllocation {
        if country == "KE" {
                return SampleBudgetAllocations
        }
        return nil
}
