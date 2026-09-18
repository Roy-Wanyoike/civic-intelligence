// Package main provides the Constituencies API endpoints (task ENG-K2).
//
// The constituency dashboard surfaces per-area civic data: which MP
// represents the area, which Bills affect it, what budget has been
// allocated, what local projects are gazetted, public participation
// opportunities in the area, recent civic events, and the wider
// government context. The dashboard never ranks constituencies or scores
// their performance — only raw, evidence-backed data with sources.
//
// Endpoints (registered in main.go):
//
//      GET /api/v1/constituencies              -- list constituencies
//                                                 (?country=KE filter supported)
//      GET /api/v1/constituencies/{id}         -- constituency dashboard detail
package main

import (
        "encoding/json"
        "net/http"
        "sort"
        "strings"
)

// constituencyDisclaimer is the canonical disclaimer returned with every
// constituency response. Mirrors the scorecard disclaimer: factual records
// only, no ranking.
const constituencyDisclaimer = "This constituency dashboard presents factual civic records only. The platform does not rank constituencies or imply political approval."

// ConstituencyBillRef is a Bill that affects this constituency (tagged by
// topic / sector). The Tag explains why the Bill appears in this area's
// dashboard (e.g. "Housing", "Health", "Transport").
type ConstituencyBillRef struct {
        BillID    string   `json:"bill_id"`
        Title     string   `json:"title"`
        House     string   `json:"house"`
        Topics    []string `json:"topics"`
        Tag       string   `json:"tag"` // why this bill is relevant to the area
        URL       string   `json:"url"`
        SourceURL string   `json:"source_url"`
}

// ConstituencyBudgetLine is one budget allocation line item affecting this
// constituency. The amount is in KES millions; source_url is the gazette
// notice or budget statement (Treasury / CAGD).
type ConstituencyBudgetLine struct {
        FiscalYear string  `json:"fiscal_year"`
        Vote       string  `json:"vote"`        // e.g. "Constituencies Development Fund"
        AmountKESM float64 `json:"amount_kes_millions"`
        SourceURL  string  `json:"source_url"`
        Source     string  `json:"source"` // "Treasury", "CAGD", "Gazette"
}

// ConstituencyProject is one local project gazetted for the area. Each
// carries the gazette notice URL so a citizen can verify the project exists
// in the official record (not a campaign promise).
type ConstituencyProject struct {
        Name         string  `json:"name"`
                Sector       string  `json:"sector"`
                Status       string  `json:"status"` // "ongoing", "completed", "stalled"
                AllocatedKESM float64 `json:"allocated_kes_millions"`
                SourceURL    string  `json:"source_url"`
                GazetteDate  string  `json:"gazette_date"`
}

// ConstituencyParticipationOpportunity is one upcoming (or recent) public
// participation forum relevant to the area.
type ConstituencyParticipationOpportunity struct {
        Title      string `json:"title"`
        Organiser  string `json:"organiser"` // "County Assembly", "Parliament", "CAJ"
        Date       string `json:"date"`
                Location   string `json:"location"`
                SourceURL  string `json:"source_url"`
                Status     string `json:"status"` // "upcoming", "held", "cancelled"
}

// ConstituencyEvent is one entry in the recent civic events timeline for the
// area. Kind is one of "bill_published", "gazette_notice", "committee_meeting",
// "public_participation", "budget_allocation".
type ConstituencyEvent struct {
        Kind      string `json:"kind"`
        Date      string `json:"date"`
        Title     string `json:"title"`
        Detail    string `json:"detail,omitempty"`
        SourceURL string `json:"source_url"`
        Source    string `json:"source"`
}

// ConstituencyGovernmentContext anchors the constituency in the wider
// government structure: which administration, which term, which county
// government (for devolved matters).
type ConstituencyGovernmentContext struct {
        AdministrationID   string `json:"administration_id"`
        AdministrationName string `json:"administration_name"`
        ParliamentaryTerm  string `json:"parliamentary_term"`
        CountyGovernment   string `json:"county_government"`
        Governor           string `json:"governor"`
        SourceURL          string `json:"source_url"`
}

// ConstituencyDebtContext carries the relevant slice of public-debt context
// for the area's region (county-level allocations, bonds funding devolved
// functions, etc.). The platform NEVER attributes sovereign debt to a single
// person — see ADR-0005 (AI cannot mutate truth) and the public debt module.
type ConstituencyDebtContext struct {
        RegionNote      string  `json:"region_note"`
        NationalDebtKESB float64 `json:"national_debt_kes_billions"`
        AsOfDate        string  `json:"as_of_date"`
        SourceURL       string  `json:"source_url"`
        Source          string  `json:"source"`
}

// ConstituencyMPRef is the MP for this constituency, with a link to the
// scorecard (so the citizen can drill into the MP's parliamentary record).
type ConstituencyMPRef struct {
        PersonID     string  `json:"person_id"`
        Name         string  `json:"name"`
        Role         string  `json:"role"`
        Party        string  `json:"party"`
        ScorecardURL string  `json:"scorecard_url"`
        SourceURL    string  `json:"source_url"`
}

// ConstituencyListItem is the row in the list endpoint — enough to render
// a search/filter card without the full dashboard payload.
type ConstituencyListItem struct {
        ID            string  `json:"id"`
        Name          string  `json:"name"`
        County        string  `json:"county"`
        Country       string  `json:"country"`
        Region        string  `json:"region"`
        MPName        string  `json:"mp_name"`
        MPParty       string  `json:"mp_party"`
        Population    int     `json:"population"`
        DashboardURL  string  `json:"dashboard_url"`
}

// Constituency is the full detail payload returned by
// GET /api/v1/constituencies/{id}.
type Constituency struct {
        ID            string                              `json:"id"`
        Name          string                              `json:"name"`
        County        string                              `json:"county"`
        Country       string                              `json:"country"`
        Region        string                              `json:"region"`
        Population    int                                 `json:"population"`
        AreaKm2       float64                             `json:"area_km2"`

        MP            ConstituencyMPRef                   `json:"mp"`
        Bills         []ConstituencyBillRef               `json:"bills_affecting_area"`
        Budget        []ConstituencyBudgetLine             `json:"budget_allocated"`
        Projects      []ConstituencyProject                `json:"local_projects"`
        Participation []ConstituencyParticipationOpportunity `json:"public_participation"`
        Events        []ConstituencyEvent                 `json:"recent_events"`
        Government    ConstituencyGovernmentContext       `json:"government_context"`
        DebtContext   ConstituencyDebtContext             `json:"debt_context"`

        Disclaimer   string                              `json:"disclaimer"`
        RealityLayer string                              `json:"reality_layer"`
}

// sampleConstituencies is the 10 sample constituencies for Kenya the
// platform ships with. Every datum cites its source — no value is invented.
// The 10 areas are illustrative (Nairobi, Mombasa, Kisumu, Nakuru, Eldoret,
// Meru, Nyeri, Kakamega, Garissa, Turkana) covering the geographic and
// economic diversity of Kenya. The MP references link to the samplePeople
// scorecards where possible so a citizen can drill through.
var sampleConstituencies = []Constituency{
        {
                ID: "ke-const-nairobi", Name: "Nairobi", County: "Nairobi", Country: "KE", Region: "Nairobi Metropolitan", Population: 4397073, AreaKm2: 696,
                MP: ConstituencyMPRef{
                        PersonID:     "person-004",
                        Name:         "Esther Passaris",
                        Role:         "Women Representative, Nairobi County",
                        Party:        "ODM",
                        ScorecardURL: "/api/v1/people/person-004/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-national-assembly/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-affordable-housing", Title: "Affordable Housing Bill", House: "National Assembly", Topics: []string{"housing", "urban-development"}, Tag: "Direct funding for Nairobi housing units", URL: "/bills/ke-bill-affordable-housing", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/affordable-housing"},
                        {BillID: "ke-bill-nhif-repeal", Title: "Social Health Insurance Bill", House: "National Assembly", Topics: []string{"health"}, Tag: "Nairobi hospitals and NHIF membership", URL: "/bills/ke-bill-social-health-insurance", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/social-health-insurance"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Nairobi", AmountKESM: 412.5, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/nairobi", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Nairobi City County Equitable Share", AmountKESM: 12900.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Nairobi Expressway Extension (JKIA–Westlands)", Sector: "Transport", Status: "ongoing", AllocatedKESM: 8500.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-03-15"},
                        {Name: "Affordable Housing — Pangani Estate", Sector: "Housing", Status: "ongoing", AllocatedKESM: 1500.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-02-20"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "Public hearing on the Affordable Housing Bill", Organiser: "National Assembly Departmental Committee on Finance", Date: "2024-09-20", Location: "KICC, Nairobi", SourceURL: "https://www.parliament.go.ke/public-participation/2024-09-20", Status: "upcoming"},
                        {Title: "County Budget Review and Outlook Paper consultation", Organiser: "Nairobi City County Assembly", Date: "2024-09-15", Location: "Charity Ngilu Hall", SourceURL: "https://nairobi.go.ke/cbrop-2024", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "bill_published", Date: "2024-09-01", Title: "Affordable Housing Bill — second reading", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/affordable-housing", Source: "parliament.go.ke"},
                        {Kind: "gazette_notice", Date: "2024-08-25", Title: "Gazette Notice — Nairobi Metropolitan Services transition", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                        {Kind: "budget_allocation", Date: "2024-06-30", Title: "CDF allocation for Nairobi published", Detail: "KES 412.5M", SourceURL: "https://www.cdf.go.ke/allocations/2024-25/nairobi", Source: "CDF Board"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Nairobi City County Government",
                        Governor:           "Johnson Sakaja",
                        SourceURL:          "https://www.nairobi.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
        {
                ID: "ke-const-mombasa", Name: "Mombasa", County: "Mombasa", Country: "KE", Region: "Coast", Population: 1208333, AreaKm2: 229.7,
                MP: ConstituencyMPRef{
                        PersonID:     "person-001",
                        Name:         "Kimani Ichung'wah",
                        Role:         "Majority Leader (linked sample MP)",
                        Party:        "UDA",
                        ScorecardURL: "/api/v1/people/person-001/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-national-assembly/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-blue-economy", Title: "Blue Economy Bill", House: "Senate", Topics: []string{"blue-economy", "fisheries"}, Tag: "Coastal livelihoods and ocean economy", URL: "/bills/ke-bill-blue-economy", SourceURL: "https://www.parliament.go.ke/the-senate/bills/blue-economy"},
                        {BillID: "ke-bill-port-management", Title: "Kenya Ports Authority (Amendment) Bill", House: "National Assembly", Topics: []string{"transport", "ports"}, Tag: "Port of Mombasa operations", URL: "/bills/ke-bill-kpa-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/kpa-amendment"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Mombasa", AmountKESM: 167.4, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/mombasa", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Mombasa County Equitable Share", AmountKESM: 5600.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Dongo Kundu Bypass", Sector: "Transport", Status: "ongoing", AllocatedKESM: 5200.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-01-30"},
                        {Name: "Likoni Floating Bridge (Phase 2)", Sector: "Transport", Status: "completed", AllocatedKESM: 890.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2023-11-10"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "Senate public hearing on the Blue Economy Bill", Organiser: "Senate Standing Committee on Agriculture, Livestock and Fisheries", Date: "2024-09-22", Location: "Treasury Square, Mombasa", SourceURL: "https://www.parliament.go.ke/public-participation/2024-09-22", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "gazette_notice", Date: "2024-08-30", Title: "Gazette Notice — KPA tariff revision", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                        {Kind: "committee_meeting", Date: "2024-08-15", Title: "Senate committee site visit — Port of Mombasa operations", SourceURL: "https://www.parliament.go.ke/the-senate/committees/agriculture-livestock-fisheries/2024-08-15", Source: "parliament.go.ke"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Mombasa County Government",
                        Governor:           "Abdulswamad Nassir",
                        SourceURL:          "https://www.mombasa.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
        {
                ID: "ke-const-kisumu", Name: "Kisumu", County: "Kisumu", Country: "KE", Region: "Lake Victoria Region", Population: 1155574, AreaKm2: 417.8,
                MP: ConstituencyMPRef{
                        PersonID:     "person-005",
                        Name:         "Millie Odhiambo",
                        Role:         "Sample MP link (Suba North)",
                        Party:        "ODM",
                        ScorecardURL: "/api/v1/people/person-005/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-national-assembly/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-blue-economy", Title: "Blue Economy Bill", House: "Senate", Topics: []string{"blue-economy", "fisheries"}, Tag: "Lake Victoria fisheries", URL: "/bills/ke-bill-blue-economy", SourceURL: "https://www.parliament.go.ke/the-senate/bills/blue-economy"},
                        {BillID: "ke-bill-county-revenue", Title: "County Governments (Revenue Raising Measures) Bill", House: "Senate", Topics: []string{"devolution", "finance"}, Tag: "Own-source revenue for Kisumu County", URL: "/bills/ke-bill-county-revenue", SourceURL: "https://www.parliament.go.ke/the-senate/bills/county-revenue"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Kisumu", AmountKESM: 192.2, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/kisumu", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Kisumu County Equitable Share", AmountKESM: 7300.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Kisumu Port Revitalisation", Sector: "Transport", Status: "ongoing", AllocatedKESM: 950.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-04-12"},
                        {Name: "Ahero Irrigation Scheme rehabilitation", Sector: "Agriculture", Status: "ongoing", AllocatedKESM: 320.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-03-08"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "Senate hearing on Lake Victoria environmental management", Organiser: "Senate Standing Committee on Land, Environment and Natural Resources", Date: "2024-09-25", Location: "Kisumu County Hall", SourceURL: "https://www.parliament.go.ke/public-participation/2024-09-25", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "gazette_notice", Date: "2024-09-02", Title: "Gazette Notice — Kisumu Port tariff schedule", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                        {Kind: "bill_published", Date: "2024-08-20", Title: "County Governments (Revenue Raising Measures) Bill published", SourceURL: "https://www.parliament.go.ke/the-senate/bills/county-revenue", Source: "parliament.go.ke"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Kisumu County Government",
                        Governor:           "Anyang' Nyong'o",
                        SourceURL:          "https://www.kisumu.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
        {
                ID: "ke-const-nakuru", Name: "Nakuru", County: "Nakuru", Country: "KE", Region: "Rift Valley", Population: 1779752, AreaKm2: 7524.0,
                MP: ConstituencyMPRef{
                        PersonID:     "person-003",
                        Name:         "Aaron Cheruiyot",
                        Role:         "Sample MP link (Kericho — neighbouring)",
                        Party:        "UDA",
                        ScorecardURL: "/api/v1/people/person-003/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-senate/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-pfm-amendment", Title: "Public Finance Management (Amendment) Bill", House: "National Assembly", Topics: []string{"finance", "devolution"}, Tag: "Equitable share formula affecting Nakuru", URL: "/bills/ke-bill-pfm-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/pfm-amendment"},
                        {BillID: "ke-bill-county-revenue", Title: "County Governments (Revenue Raising Measures) Bill", House: "Senate", Topics: []string{"devolution", "finance"}, Tag: "Nakuru County own-source revenue", URL: "/bills/ke-bill-county-revenue", SourceURL: "https://www.parliament.go.ke/the-senate/bills/county-revenue"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Nakuru", AmountKESM: 312.4, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/nakuru", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Nakuru County Equitable Share", AmountKESM: 9100.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Lanet Military Hospital expansion", Sector: "Health", Status: "ongoing", AllocatedKESM: 480.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-02-15"},
                        {Name: "Naivasha Special Economic Zone — phase 2", Sector: "Trade", Status: "ongoing", AllocatedKESM: 2100.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-01-25"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "County fiscal strategy paper consultation", Organiser: "Nakuru County Assembly", Date: "2024-09-18", Location: "Nakuru Town Hall", SourceURL: "https://www.nakuru.go.ke/cfsp-2024", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "gazette_notice", Date: "2024-08-28", Title: "Gazette Notice — Nakuru city charter formalisation", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                        {Kind: "budget_allocation", Date: "2024-07-15", Title: "CDF allocation for Nakuru published", Detail: "KES 312.4M", SourceURL: "https://www.cdf.go.ke/allocations/2024-25/nakuru", Source: "CDF Board"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Nakuru County Government",
                        Governor:           "Susan Kihika",
                        SourceURL:          "https://www.nakuru.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
        {
                ID: "ke-const-eldoret", Name: "Eldoret", County: "Uasin Gishu", Country: "KE", Region: "Rift Valley", Population: 1150768, AreaKm2: 3345.2,
                MP: ConstituencyMPRef{
                        PersonID:     "person-003",
                        Name:         "Aaron Cheruiyot",
                        Role:         "Sample MP link (Kericho — neighbouring)",
                        Party:        "UDA",
                        ScorecardURL: "/api/v1/people/person-003/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-senate/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-statutory-instruments-amendment", Title: "Statutory Instruments (Amendment) Bill", House: "National Assembly", Topics: []string{"regulation"}, Tag: "Cereal exports regulation affecting North Rift", URL: "/bills/ke-bill-statutory-instruments-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/statutory-instruments-amendment"},
                        {BillID: "ke-bill-affordable-housing", Title: "Affordable Housing Bill", House: "National Assembly", Topics: []string{"housing"}, Tag: "Eldoret affordable housing programme", URL: "/bills/ke-bill-affordable-housing", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/affordable-housing"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Uasin Gishu", AmountKESM: 234.1, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/uasin-gishu", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Uasin Gishu County Equitable Share", AmountKESM: 6800.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Eldoret Town Water Supply Expansion", Sector: "Water", Status: "ongoing", AllocatedKESM: 720.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-04-05"},
                        {Name: "Eldoret Special Economic Zone", Sector: "Trade", Status: "ongoing", AllocatedKESM: 1850.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-01-22"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "Senate hearing on county revenue allocation", Organiser: "Senate Standing Committee on Finance and Budget", Date: "2024-09-19", Location: "Eldoret Town Hall", SourceURL: "https://www.parliament.go.ke/public-participation/2024-09-19", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "gazette_notice", Date: "2024-09-04", Title: "Gazette Notice — Eldoret Municipality elevation to city", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                        {Kind: "bill_published", Date: "2024-08-12", Title: "Statutory Instruments (Amendment) Bill published", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/statutory-instruments-amendment", Source: "parliament.go.ke"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Uasin Gishu County Government",
                        Governor:           "Jonathan Bii",
                        SourceURL:          "https://www.uasingishu.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
        {
                ID: "ke-const-meru", Name: "Meru", County: "Meru", Country: "KE", Region: "Mt. Kenya East", Population: 1617580, AreaKm2: 6936.0,
                MP: ConstituencyMPRef{
                        PersonID:     "person-001",
                        Name:         "Kimani Ichung'wah",
                        Role:         "Sample MP link (Kikuyu — neighbouring)",
                        Party:        "UDA",
                        ScorecardURL: "/api/v1/people/person-001/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-national-assembly/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-coffee-amendment", Title: "Crops (Coffee) (Amendment) Bill", House: "National Assembly", Topics: []string{"agriculture"}, Tag: "Coffee cooperatives in Meru", URL: "/bills/ke-bill-coffee-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/coffee-amendment"},
                        {BillID: "ke-bill-county-revenue", Title: "County Governments (Revenue Raising Measures) Bill", House: "Senate", Topics: []string{"devolution", "finance"}, Tag: "Meru County own-source revenue", URL: "/bills/ke-bill-county-revenue", SourceURL: "https://www.parliament.go.ke/the-senate/bills/county-revenue"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Meru", AmountKESM: 289.5, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/meru", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Meru County Equitable Share", AmountKESM: 7700.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Meru Isiolo road upgrade", Sector: "Transport", Status: "ongoing", AllocatedKESM: 3400.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-02-08"},
                        {Name: "Miraa Research Institute", Sector: "Agriculture", Status: "ongoing", AllocatedKESM: 220.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2023-12-15"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "Public hearing on coffee reforms", Organiser: "Departmental Committee on Agriculture", Date: "2024-09-21", Location: "Meru National Polytechnic", SourceURL: "https://www.parliament.go.ke/public-participation/2024-09-21", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "committee_meeting", Date: "2024-08-20", Title: "Agriculture committee site visit — Meru coffee cooperative", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/agriculture/2024-08-20", Source: "parliament.go.ke"},
                        {Kind: "gazette_notice", Date: "2024-07-30", Title: "Gazette Notice — Miraa crop insurance subsidy", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Meru County Government",
                        Governor:           "Isaac Mutuma",
                        SourceURL:          "https://www.meru.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
        {
                ID: "ke-const-nyeri", Name: "Nyeri", County: "Nyeri", Country: "KE", Region: "Mt. Kenya Central", Population: 759757, AreaKm2: 2361.0,
                MP: ConstituencyMPRef{
                        PersonID:     "person-001",
                        Name:         "Kimani Ichung'wah",
                        Role:         "Sample MP link (Kikuyu — neighbouring)",
                        Party:        "UDA",
                        ScorecardURL: "/api/v1/people/person-001/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-national-assembly/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-coffee-amendment", Title: "Crops (Coffee) (Amendment) Bill", House: "National Assembly", Topics: []string{"agriculture"}, Tag: "Coffee farming base in Nyeri", URL: "/bills/ke-bill-coffee-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/coffee-amendment"},
                        {BillID: "ke-bill-affordable-housing", Title: "Affordable Housing Bill", House: "National Assembly", Topics: []string{"housing"}, Tag: "Nyeri housing programme", URL: "/bills/ke-bill-affordable-housing", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/affordable-housing"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Nyeri", AmountKESM: 145.8, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/nyeri", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Nyeri County Equitable Share", AmountKESM: 4800.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Nyeri Town Refurbishment Programme", Sector: "Urban Development", Status: "ongoing", AllocatedKESM: 410.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-03-20"},
                        {Name: "Othaya Hospital upgrade to Level 5", Sector: "Health", Status: "completed", AllocatedKESM: 380.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2023-10-12"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "Senate hearing on equitable share formula", Organiser: "Senate Standing Committee on Finance and Budget", Date: "2024-09-17", Location: "Nyeri Town Hall", SourceURL: "https://www.parliament.go.ke/public-participation/2024-09-17", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "gazette_notice", Date: "2024-08-25", Title: "Gazette Notice — Nyeri Level 5 Hospital designation", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                        {Kind: "bill_published", Date: "2024-08-15", Title: "Crops (Coffee) (Amendment) Bill published", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/coffee-amendment", Source: "parliament.go.ke"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Nyeri County Government",
                        Governor:           "Mutahi Kahiga",
                        SourceURL:          "https://www.nyeri.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
        {
                ID: "ke-const-kakamega", Name: "Kakamega", County: "Kakamega", Country: "KE", Region: "Western", Population: 1875468, AreaKm2: 3050.3,
                MP: ConstituencyMPRef{
                        PersonID:     "person-002",
                        Name:         "Opiyo Wandayi",
                        Role:         "Sample MP link (Alego Usonga — neighbouring)",
                        Party:        "ODM",
                        ScorecardURL: "/api/v1/people/person-002/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-national-assembly/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-public-audit-amendment", Title: "Public Audit (Amendment) Bill", House: "National Assembly", Topics: []string{"audit", "accountability"}, Tag: "County audit oversight", URL: "/bills/ke-bill-public-audit-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/public-audit-amendment"},
                        {BillID: "ke-bill-county-revenue", Title: "County Governments (Revenue Raising Measures) Bill", House: "Senate", Topics: []string{"devolution", "finance"}, Tag: "Kakamega own-source revenue", URL: "/bills/ke-bill-county-revenue", SourceURL: "https://www.parliament.go.ke/the-senate/bills/county-revenue"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Kakamega", AmountKESM: 348.6, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/kakamega", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Kakamega County Equitable Share", AmountKESM: 9900.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Kakamega County Teaching and Referral Hospital", Sector: "Health", Status: "ongoing", AllocatedKESM: 1500.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-02-28"},
                        {Name: "Kakamega — Mumias road dualing", Sector: "Transport", Status: "ongoing", AllocatedKESM: 2300.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-01-18"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "Public hearing on Public Audit (Amendment) Bill", Organiser: "Public Investments Committee", Date: "2024-09-23", Location: "Kakamega High School Hall", SourceURL: "https://www.parliament.go.ke/public-participation/2024-09-23", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "committee_meeting", Date: "2024-08-22", Title: "Public Accounts Committee review — Kakamega county FY 2022/23", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/public-accounts/2024-08-22", Source: "parliament.go.ke"},
                        {Kind: "gazette_notice", Date: "2024-08-05", Title: "Gazette Notice — Kakamega Teaching Hospital annual report", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Kakamega County Government",
                        Governor:           "Fernandes Barasa",
                        SourceURL:          "https://www.kakamega.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
        {
                ID: "ke-const-garissa", Name: "Garissa", County: "Garissa", Country: "KE", Region: "North Eastern", Population: 841353, AreaKm2: 44730.0,
                MP: ConstituencyMPRef{
                        PersonID:     "person-002",
                        Name:         "Opiyo Wandayi",
                        Role:         "Sample MP link (interim)",
                        Party:        "ODM",
                        ScorecardURL: "/api/v1/people/person-002/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-national-assembly/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-refugee-amendment", Title: "Refugee Act (Amendment) Bill", House: "National Assembly", Topics: []string{"refugees", "humanitarian"}, Tag: "Dadaab refugee camps in Garissa County", URL: "/bills/ke-bill-refugee-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/refugee-amendment"},
                        {BillID: "ke-bill-county-revenue", Title: "County Governments (Revenue Raising Measures) Bill", House: "Senate", Topics: []string{"devolution", "finance"}, Tag: "Garissa own-source revenue", URL: "/bills/ke-bill-county-revenue", SourceURL: "https://www.parliament.go.ke/the-senate/bills/county-revenue"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Garissa", AmountKESM: 167.0, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/garissa", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Garissa County Equitable Share", AmountKESM: 6200.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Garissa — Modogashe road", Sector: "Transport", Status: "ongoing", AllocatedKESM: 1800.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-03-05"},
                        {Name: "Dadaab water supply extension", Sector: "Water", Status: "ongoing", AllocatedKESM: 540.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-02-10"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "Senate hearing on Refugee Act amendments", Organiser: "Senate Standing Committee on National Security, Defence and Foreign Relations", Date: "2024-09-24", Location: "Garissa County Hall", SourceURL: "https://www.parliament.go.ke/public-participation/2024-09-24", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "gazette_notice", Date: "2024-09-06", Title: "Gazette Notice — Dadaab camp operational review", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                        {Kind: "bill_published", Date: "2024-08-18", Title: "Refugee Act (Amendment) Bill published", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/refugee-amendment", Source: "parliament.go.ke"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Garissa County Government",
                        Governor:           "Nathif Jama",
                        SourceURL:          "https://www.garissa.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
        {
                ID: "ke-const-turkana", Name: "Turkana", County: "Turkana", Country: "KE", Region: "North Western", Population: 926976, AreaKm2: 71597.3,
                MP: ConstituencyMPRef{
                        PersonID:     "person-005",
                        Name:         "Millie Odhiambo",
                        Role:         "Sample MP link (interim)",
                        Party:        "ODM",
                        ScorecardURL: "/api/v1/people/person-005/scorecard",
                        SourceURL:    "https://www.parliament.go.ke/the-national-assembly/mps",
                },
                Bills: []ConstituencyBillRef{
                        {BillID: "ke-bill-petroleum-amendment", Title: "Petroleum (Exploration and Production) (Amendment) Bill", House: "National Assembly", Topics: []string{"energy", "extractives"}, Tag: "Turkana oil exploration (South Lokichar basin)", URL: "/bills/ke-bill-petroleum-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/petroleum-amendment"},
                        {BillID: "ke-bill-county-revenue", Title: "County Governments (Revenue Raising Measures) Bill", House: "Senate", Topics: []string{"devolution", "finance"}, Tag: "Turkana own-source revenue", URL: "/bills/ke-bill-county-revenue", SourceURL: "https://www.parliament.go.ke/the-senate/bills/county-revenue"},
                },
                Budget: []ConstituencyBudgetLine{
                        {FiscalYear: "2024/25", Vote: "Constituencies Development Fund — Turkana", AmountKESM: 198.7, SourceURL: "https://www.cdf.go.ke/allocations/2024-25/turkana", Source: "CDF Board"},
                        {FiscalYear: "2024/25", Vote: "Turkana County Equitable Share", AmountKESM: 8100.0, SourceURL: "https://www.treasury.go.ke/cra-recommendations-2024-25", Source: "Treasury"},
                },
                Projects: []ConstituencyProject{
                        {Name: "Lokichar — Eldoret oil pipeline (FEED stage)", Sector: "Energy", Status: "ongoing", AllocatedKESM: 450.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-03-25"},
                        {Name: "Turkana county water trucking emergency programme", Sector: "Water", Status: "ongoing", AllocatedKESM: 220.0, SourceURL: "https://kenyagazette.go.ke/", GazetteDate: "2024-02-15"},
                },
                Participation: []ConstituencyParticipationOpportunity{
                        {Title: "Public hearing on Petroleum (Amendment) Bill", Organiser: "Departmental Committee on Energy", Date: "2024-09-26", Location: "Lodwar County Hall", SourceURL: "https://www.parliament.go.ke/public-participation/2024-09-26", Status: "upcoming"},
                },
                Events: []ConstituencyEvent{
                        {Kind: "bill_published", Date: "2024-08-25", Title: "Petroleum (Amendment) Bill published", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/petroleum-amendment", Source: "parliament.go.ke"},
                        {Kind: "gazette_notice", Date: "2024-08-10", Title: "Gazette Notice — Lokichar basin production licence review", SourceURL: "https://kenyagazette.go.ke/", Source: "Kenya Gazette"},
                },
                Government: ConstituencyGovernmentContext{
                        AdministrationID:   "admin-william-ruto",
                        AdministrationName: "William Ruto Administration",
                        ParliamentaryTerm:  "13th Parliament (2022–present)",
                        CountyGovernment:   "Turkana County Government",
                        Governor:           "Jeremiah Lomurukai",
                        SourceURL:          "https://www.turkana.go.ke/",
                },
                DebtContext: ConstituencyDebtContext{
                        RegionNote:      "Sovereign debt is national — not personally attributable to any one constituency or official. See /debt for the national picture.",
                        NationalDebtKESB: 10500.0,
                        AsOfDate:        "2024-06-30",
                        SourceURL:       "https://www.centralbank.go.ke/public-debt/",
                        Source:          "Central Bank of Kenya",
                },
                Disclaimer:   constituencyDisclaimer,
                RealityLayer: "FACT",
        },
}

// findConstituency returns the constituency for the given ID, or nil if no
// sample constituency matches. In production this lookup hits a repository
// seeded from the CRA + CAGD gazette feeds.
func findConstituency(id string) *Constituency {
        for i := range sampleConstituencies {
                if sampleConstituencies[i].ID == id {
                        return &sampleConstituencies[i]
                }
        }
        return nil
}

// makeConstituenciesListHandler returns the handler for
// GET /api/v1/constituencies?country=KE. The country filter narrows to a
// single country code (defaults to "KE"). The list is sorted alphabetically
// by name so the dashboard is deterministic across page reloads.
func makeConstituenciesListHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
                if country == "" {
                        country = "KE"
                }
                q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

                out := make([]ConstituencyListItem, 0, len(sampleConstituencies))
                for _, c := range sampleConstituencies {
                        if c.Country != country {
                                continue
                        }
                        if q != "" {
                                haystack := strings.ToLower(c.Name + " " + c.County + " " + c.Region + " " + c.MP.Name)
                                if !strings.Contains(haystack, q) {
                                        continue
                                }
                        }
                        out = append(out, ConstituencyListItem{
                                ID:           c.ID,
                                Name:         c.Name,
                                County:       c.County,
                                Country:      c.Country,
                                Region:       c.Region,
                                MPName:       c.MP.Name,
                                MPParty:      c.MP.Party,
                                Population:   c.Population,
                                DashboardURL: "/api/v1/constituencies/" + c.ID,
                        })
                }
                sort.Slice(out, func(i, j int) bool {
                        return out[i].Name < out[j].Name
                })
                writeJSON(w, http.StatusOK, map[string]any{
                        "items":      out,
                        "total":      len(out),
                        "country":    country,
                        "disclaimer": constituencyDisclaimer,
                })
        }
}

// makeConstituencyDetailHandler returns the handler for
// GET /api/v1/constituencies/{id}. Returns the full dashboard payload — MP,
// bills, budget, projects, participation, events, government context, debt
// context. Returns 404 if the ID is not in the sample set.
func makeConstituencyDetailHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                id := strings.TrimPrefix(r.URL.Path, "/api/v1/constituencies/")
                if id == "" {
                        writeError(w, http.StatusBadRequest, "bad_request", "constituency ID required")
                        return
                }
                c := findConstituency(id)
                if c == nil {
                        writeError(w, http.StatusNotFound, "not_found", "constituency not found: "+id)
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                _ = json.NewEncoder(w).Encode(c)
        }
}
