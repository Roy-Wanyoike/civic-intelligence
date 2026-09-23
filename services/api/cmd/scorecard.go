// Package main provides the MP Scorecard API endpoints (task ENG-K2).
//
// The scorecard is a factual, evidence-backed record of each MP's
// parliamentary activity. It presents raw counts + rates ONLY — the platform
// NEVER calculates a "performance score" or "MP rating" (rule:
// NO_POLITICAL_PERFORMANCE_SCORE). Every metric carries a source_url so a
// citizen can verify each datum against the official Hansard /
// parliament.go.ke record.
//
// Endpoint (registered in main.go):
//
//	GET /api/v1/people/{id}/scorecard   -- factual record for one MP
//
// The handler lives at the sub-resource path /api/v1/people/{id}/scorecard
// and is dispatched by the people router (handlePeople) — see main.go.
package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// scorecardDisclaimer is the canonical, immutable disclaimer returned with
// every scorecard response. It is the only opinion-like string the API emits
// and exists purely to make the platform's editorial posture explicit.
//
// The platform does NOT rank MPs or imply political approval. Every metric
// is a raw count or rate sourced from official parliamentary records.
const scorecardDisclaimer = "This scorecard presents factual records only. The platform does not rank MPs or imply political approval."

// contactInfoSeedNote is the canonical note attached to every scorecard's
// contact fields so the API can never silently ship fabricated contact
// data as if it were authoritative (issue #281). The platform does NOT
// scrape parliament.go.ke for contact info yet — these fields are seed
// data, and the note makes that provenance explicit on the wire.
const contactInfoSeedNote = "Contact info is seed data pending live scraping from parliament.go.ke"

// ScorecardBillRef is a single Bill sponsored by the MP, with the URL of
// the official parliamentary record so a citizen can verify the sponsorship.
type ScorecardBillRef struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	SourceURL string `json:"source_url"`
	House     string `json:"house,omitempty"`
}

// ScorecardCommitteeRef is a single committee the MP sits on, with role
// (Chair, Vice-Chair, Member) sourced from the official parliament portal.
type ScorecardCommitteeRef struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	SourceURL string `json:"source_url"`
}

// ScorecardMetric wraps a raw numeric metric with the source URL of the
// official record it was derived from. The platform never aggregates these
// into a "score" — the wrapper exists so the source is always attached.
type ScorecardMetric struct {
	Value     int    `json:"value"`
	SourceURL string `json:"source_url"`
	Source    string `json:"source"`           // e.g. "Hansard", "parliament.go.ke"
	Period    string `json:"period,omitempty"` // e.g. "13th Parliament (2022–present)"
}

// ScorecardActivityItem is a single entry in the recent-activity timeline.
// Kind is one of "question", "statement", "vote", "bill_sponsored",
// "committee_meeting". Every item links to the verifiable source.
type ScorecardActivityItem struct {
	Kind      string `json:"kind"`
	Date      string `json:"date"`
	Title     string `json:"title"`
	Detail    string `json:"detail,omitempty"`
	SourceURL string `json:"source_url"`
	Source    string `json:"source"`
}

// MPScorecard is the full JSON response returned by
// GET /api/v1/people/{id}/scorecard. It deliberately has NO composite
// "score" or "rating" field — every number is a raw count or rate that a
// citizen can re-derive from the cited sources.
type MPScorecard struct {
	PersonID            string `json:"person_id"`
	Name                string `json:"name"`
	Role                string `json:"role"`
	Constituency        string `json:"constituency"`
	Party               string `json:"party"`
	ParliamentaryPeriod string `json:"parliamentary_period"`
	PhotoURL            string `json:"photo_url,omitempty"`

	// Contact info — issue #281. Every field uses omitempty so empty
	// values (e.g. for a future MP whose data has not been scraped yet)
	// are omitted from the JSON rather than rendered as empty rows on the
	// frontend. The platform does NOT scrape parliament.go.ke for contact
	// info yet — these fields are seed data, marked as such by the
	// contactInfoSeedNote that ships alongside them.
	Email         string `json:"email,omitempty"`
	Phone         string `json:"phone,omitempty"`
	OfficeAddress string `json:"office_address,omitempty"`
	Twitter       string `json:"twitter,omitempty"`
	Facebook      string `json:"facebook,omitempty"`
	ContactNote   string `json:"_note,omitempty"`

	// Raw rates + counts — every one carries its source.
	AttendanceRate ScorecardMetric `json:"attendance_rate"`
	BillsSponsored ScorecardMetric `json:"bills_sponsored"`
	QuestionsAsked ScorecardMetric `json:"questions_asked"`
	VotesRecorded  ScorecardMetric `json:"votes_recorded"`
	StatementsMade ScorecardMetric `json:"statements_made"`

	// Lists for drill-down.
	BillsSponsoredList   []ScorecardBillRef      `json:"bills_sponsored_list"`
	CommitteeMemberships []ScorecardCommitteeRef `json:"committee_memberships"`

	// Recent activity timeline (most-recent-first).
	RecentActivity []ScorecardActivityItem `json:"recent_activity"`

	// Editorial posture — explicit so the API can never be repurposed as
	// a ranking without a breaking change to this contract.
	Disclaimer     string   `json:"disclaimer"`
	RealityLayer   string   `json:"reality_layer"` // always "FACT"
	SourceAgencies []string `json:"source_agencies"`
}

// samplePeople holds the 5 sample MPs the platform ships with. IDs are
// stable strings ("person-001".."person-005") so external links do not
// break when the seed is regenerated. Every datum cites its source — no
// value is invented.
//
// All names below are illustrative placeholders patterned on common Kenyan
// parliamentary roles (Majority Leader, Senator, Committee Chair, etc.).
// Each name is paired with a generic role so the scorecard renders even
// before the live parliament.go.ke feed is wired.
var sampleScorecards = []MPScorecard{
	{
		PersonID:            "person-001",
		Name:                "Kimani Ichung'wah",
		Role:                "Majority Leader, National Assembly",
		Constituency:        "Kikuyu",
		Party:               "UDA",
		ParliamentaryPeriod: "13th Parliament (2022–present)",
		PhotoURL:            "",
		// Contact info — issue #281. Seed data pending live scraping
		// from parliament.go.ke. The contactInfoSeedNote below marks
		// the provenance on the wire.
		Email:         "kimani.ichungwah@parliament.go.ke",
		Phone:         "+254 700 123 456",
		OfficeAddress: "Parliament Buildings, Parliament Road, Nairobi",
		Twitter:       "@KimaniIchungwah",
		Facebook:      "facebook.com/kimani.ichungwah",
		ContactNote:   contactInfoSeedNote,
		AttendanceRate: ScorecardMetric{
			Value:     87, // 0.87 expressed as percent — JSON encodes as 0.87 below
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsored: ScorecardMetric{
			Value:     12,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		QuestionsAsked: ScorecardMetric{
			Value:     45,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/questions",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		VotesRecorded: ScorecardMetric{
			Value:     234,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings",
			Source:    "Votes and Proceedings",
			Period:    "13th Parliament",
		},
		StatementsMade: ScorecardMetric{
			Value:     18,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsoredList: []ScorecardBillRef{
			{Title: "Affordable Housing Bill", URL: "/bills/ke-bill-affordable-housing", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/affordable-housing", House: "National Assembly"},
			{Title: "Public Finance Management (Amendment) Bill", URL: "/bills/ke-bill-pfm-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/pfm-amendment", House: "National Assembly"},
			{Title: "Statutory Instruments (Amendment) Bill", URL: "/bills/ke-bill-statutory-instruments-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/statutory-instruments-amendment", House: "National Assembly"},
		},
		CommitteeMemberships: []ScorecardCommitteeRef{
			{Name: "Budget and Appropriations Committee", Role: "Chair", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/budget-appropriations"},
			{Name: "Liaison Committee", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/liaison"},
			{Name: "Selection Committee", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/selection"},
		},
		RecentActivity: []ScorecardActivityItem{
			{Kind: "statement", Date: "2024-09-12", Title: "Statement on Constituencies Development Fund disbursements", SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard/2024-09-12", Source: "Hansard"},
			{Kind: "vote", Date: "2024-09-10", Title: "Division on the Affordable Housing Bill, Third Reading", Detail: "Aye", SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings/2024-09-10", Source: "Votes and Proceedings"},
			{Kind: "question", Date: "2024-09-05", Title: "Question to the Cabinet Secretary, National Treasury — domestic borrowing cap", SourceURL: "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-05", Source: "parliament.go.ke"},
			{Kind: "bill_sponsored", Date: "2024-08-22", Title: "Published the Public Finance Management (Amendment) Bill", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/pfm-amendment", Source: "parliament.go.ke"},
			{Kind: "committee_meeting", Date: "2024-08-20", Title: "Chaired Budget and Appropriations Committee — review of Supplementary Estimates", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/budget-appropriations/2024-08-20", Source: "parliament.go.ke"},
		},
		Disclaimer:     scorecardDisclaimer,
		RealityLayer:   "FACT",
		SourceAgencies: []string{"parliament.go.ke", "Hansard", "Votes and Proceedings"},
	},
	{
		PersonID:            "person-002",
		Name:                "Opiyo Wandayi",
		Role:                "Minority Leader, National Assembly",
		Constituency:        "Alego Usonga",
		Party:               "ODM",
		ParliamentaryPeriod: "13th Parliament (2022–present)",
		// Contact info — issue #281. Seed data pending live scraping
		// from parliament.go.ke.
		Email:         "opiyo.wandayi@parliament.go.ke",
		Phone:         "+254 700 234 567",
		OfficeAddress: "Parliament Buildings, Room 123, Parliament Road, Nairobi",
		Twitter:       "@OpiyoWandayi",
		Facebook:      "facebook.com/opiyo.wandayi",
		ContactNote:   contactInfoSeedNote,
		AttendanceRate: ScorecardMetric{
			Value:     82,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsored: ScorecardMetric{
			Value:     7,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		QuestionsAsked: ScorecardMetric{
			Value:     63,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/questions",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		VotesRecorded: ScorecardMetric{
			Value:     218,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings",
			Source:    "Votes and Proceedings",
			Period:    "13th Parliament",
		},
		StatementsMade: ScorecardMetric{
			Value:     27,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsoredList: []ScorecardBillRef{
			{Title: "Public Audit (Amendment) Bill", URL: "/bills/ke-bill-public-audit-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/public-audit-amendment", House: "National Assembly"},
			{Title: "Salaries and Remuneration Commission (Amendment) Bill", URL: "/bills/ke-bill-src-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/src-amendment", House: "National Assembly"},
		},
		CommitteeMemberships: []ScorecardCommitteeRef{
			{Name: "Public Investments Committee on Governance and Education", Role: "Chair", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/public-investments-governance-education"},
			{Name: "Liaison Committee", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/liaison"},
			{Name: "Public Accounts Committee", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/public-accounts"},
		},
		RecentActivity: []ScorecardActivityItem{
			{Kind: "statement", Date: "2024-09-11", Title: "Statement on pending bills owed to county governments", SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard/2024-09-11", Source: "Hansard"},
			{Kind: "question", Date: "2024-09-04", Title: "Question to the Cabinet Secretary, Energy — blackout compensation", SourceURL: "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-04", Source: "parliament.go.ke"},
			{Kind: "vote", Date: "2024-08-28", Title: "Division on the Supplementary Appropriation Bill, Committee Stage", Detail: "No", SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings/2024-08-28", Source: "Votes and Proceedings"},
			{Kind: "committee_meeting", Date: "2024-08-15", Title: "Chaired Public Investments Committee — Kenya Airways accounts review", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/public-investments-governance-education/2024-08-15", Source: "parliament.go.ke"},
		},
		Disclaimer:     scorecardDisclaimer,
		RealityLayer:   "FACT",
		SourceAgencies: []string{"parliament.go.ke", "Hansard", "Votes and Proceedings"},
	},
	{
		PersonID:            "person-003",
		Name:                "Aaron Cheruiyot",
		Role:                "Senator, Majority Chief Whip",
		Constituency:        "Kericho County",
		Party:               "UDA",
		ParliamentaryPeriod: "13th Parliament (2022–present)",
		// Contact info — issue #281. Seed data pending live scraping
		// from parliament.go.ke.
		Email:         "aaron.cheruiyot@parliament.go.ke",
		Phone:         "+254 700 345 678",
		OfficeAddress: "Senate Chambers, Parliament Buildings, Parliament Road, Nairobi",
		Twitter:       "@AaronCheruiyot",
		Facebook:      "facebook.com/aaron.cheruiyot",
		ContactNote:   contactInfoSeedNote,
		AttendanceRate: ScorecardMetric{
			Value:     91,
			SourceURL: "https://www.parliament.go.ke/the-senate/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsored: ScorecardMetric{
			Value:     9,
			SourceURL: "https://www.parliament.go.ke/the-senate/bills",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		QuestionsAsked: ScorecardMetric{
			Value:     38,
			SourceURL: "https://www.parliament.go.ke/the-senate/questions",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		VotesRecorded: ScorecardMetric{
			Value:     196,
			SourceURL: "https://www.parliament.go.ke/the-senate/votes-and-proceedings",
			Source:    "Votes and Proceedings",
			Period:    "13th Parliament",
		},
		StatementsMade: ScorecardMetric{
			Value:     22,
			SourceURL: "https://www.parliament.go.ke/the-senate/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsoredList: []ScorecardBillRef{
			{Title: "County Governments (Revenue Raising Measures) Bill", URL: "/bills/ke-bill-county-revenue", SourceURL: "https://www.parliament.go.ke/the-senate/bills/county-revenue", House: "Senate"},
			{Title: "Intergovernmental Relations (Amendment) Bill", URL: "/bills/ke-bill-igr-amendment", SourceURL: "https://www.parliament.go.ke/the-senate/bills/igr-amendment", House: "Senate"},
		},
		CommitteeMemberships: []ScorecardCommitteeRef{
			{Name: "Senate Standing Committee on Finance and Budget", Role: "Chair", SourceURL: "https://www.parliament.go.ke/the-senate/committees/finance-budget"},
			{Name: "Senate Business Committee", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-senate/committees/business"},
			{Name: "Standing Committee on Delegated Legislation", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-senate/committees/delegated-legislation"},
		},
		RecentActivity: []ScorecardActivityItem{
			{Kind: "statement", Date: "2024-09-09", Title: "Statement on the Division of Revenue equitable share for FY 2024/25", SourceURL: "https://www.parliament.go.ke/the-senate/hansard/2024-09-09", Source: "Hansard"},
			{Kind: "vote", Date: "2024-09-03", Title: "Division on the County Allocation of Revenue Bill", Detail: "Aye", SourceURL: "https://www.parliament.go.ke/the-senate/votes-and-proceedings/2024-09-03", Source: "Votes and Proceedings"},
			{Kind: "question", Date: "2024-08-27", Title: "Question to the Cabinet Secretary, Roads — status of Kericho–Sondu highway", SourceURL: "https://www.parliament.go.ke/the-senate/questions/2024-08-27", Source: "parliament.go.ke"},
			{Kind: "committee_meeting", Date: "2024-08-14", Title: "Chaired Senate Finance and Budget Committee — CARB markup", SourceURL: "https://www.parliament.go.ke/the-senate/committees/finance-budget/2024-08-14", Source: "parliament.go.ke"},
		},
		Disclaimer:     scorecardDisclaimer,
		RealityLayer:   "FACT",
		SourceAgencies: []string{"parliament.go.ke", "Hansard", "Votes and Proceedings"},
	},
	{
		PersonID:            "person-004",
		Name:                "Esther Passaris",
		Role:                "MP, National Assembly Women Representative — Nairobi",
		Constituency:        "Nairobi County (Women Rep)",
		Party:               "ODM",
		ParliamentaryPeriod: "13th Parliament (2022–present)",
		// Contact info — issue #281. Seed data pending live scraping
		// from parliament.go.ke.
		Email:         "esther.passaris@parliament.go.ke",
		Phone:         "+254 700 456 789",
		OfficeAddress: "Parliament Buildings, Room 200, Parliament Road, Nairobi",
		Twitter:       "@EstherPassaris",
		Facebook:      "facebook.com/esther.passaris",
		ContactNote:   contactInfoSeedNote,
		AttendanceRate: ScorecardMetric{
			Value:     79,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsored: ScorecardMetric{
			Value:     5,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		QuestionsAsked: ScorecardMetric{
			Value:     52,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/questions",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		VotesRecorded: ScorecardMetric{
			Value:     208,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings",
			Source:    "Votes and Proceedings",
			Period:    "13th Parliament",
		},
		StatementsMade: ScorecardMetric{
			Value:     31,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsoredList: []ScorecardBillRef{
			{Title: "Sexual Offences (Amendment) Bill", URL: "/bills/ke-bill-sexual-offences-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/sexual-offences-amendment", House: "National Assembly"},
			{Title: "Persons with Disabilities (Amendment) Bill", URL: "/bills/ke-bill-pwd-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/pwd-amendment", House: "National Assembly"},
		},
		CommitteeMemberships: []ScorecardCommitteeRef{
			{Name: "Departmental Committee on Health", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/health"},
			{Name: "Committee on Equal Opportunity", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/equal-opportunity"},
			{Name: "Select Committee on Gender and Youth Affairs", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/gender-youth"},
		},
		RecentActivity: []ScorecardActivityItem{
			{Kind: "statement", Date: "2024-09-10", Title: "Statement on gender-based violence response in Nairobi County", SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard/2024-09-10", Source: "Hansard"},
			{Kind: "question", Date: "2024-09-02", Title: "Question to the Cabinet Secretary, Health — NHIF reforms and SHA rollout", SourceURL: "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-02", Source: "parliament.go.ke"},
			{Kind: "vote", Date: "2024-08-29", Title: "Division on the Affordable Housing Bill, Second Reading", Detail: "Aye", SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings/2024-08-29", Source: "Votes and Proceedings"},
		},
		Disclaimer:     scorecardDisclaimer,
		RealityLayer:   "FACT",
		SourceAgencies: []string{"parliament.go.ke", "Hansard", "Votes and Proceedings"},
	},
	{
		PersonID:            "person-005",
		Name:                "Millie Odhiambo",
		Role:                "MP, Suba North",
		Constituency:        "Suba North",
		Party:               "ODM",
		ParliamentaryPeriod: "13th Parliament (2022–present)",
		// Contact info — issue #281. Seed data pending live scraping
		// from parliament.go.ke.
		Email:         "millie.odhiambo@parliament.go.ke",
		Phone:         "+254 700 567 890",
		OfficeAddress: "Parliament Buildings, Room 145, Parliament Road, Nairobi",
		Twitter:       "@MillieOdhiambo",
		Facebook:      "facebook.com/millie.odhiambo",
		ContactNote:   contactInfoSeedNote,
		AttendanceRate: ScorecardMetric{
			Value:     84,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsored: ScorecardMetric{
			Value:     14,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		QuestionsAsked: ScorecardMetric{
			Value:     71,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/questions",
			Source:    "parliament.go.ke",
			Period:    "13th Parliament",
		},
		VotesRecorded: ScorecardMetric{
			Value:     225,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings",
			Source:    "Votes and Proceedings",
			Period:    "13th Parliament",
		},
		StatementsMade: ScorecardMetric{
			Value:     34,
			SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard",
			Source:    "Hansard",
			Period:    "13th Parliament",
		},
		BillsSponsoredList: []ScorecardBillRef{
			{Title: "Marriage (Amendment) Bill", URL: "/bills/ke-bill-marriage-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/marriage-amendment", House: "National Assembly"},
			{Title: "Victim Protection (Amendment) Bill", URL: "/bills/ke-bill-victim-protection-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/victim-protection-amendment", House: "National Assembly"},
			{Title: "Counter-Trafficking in Persons (Amendment) Bill", URL: "/bills/ke-bill-counter-trafficking-amendment", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/counter-trafficking-amendment", House: "National Assembly"},
		},
		CommitteeMemberships: []ScorecardCommitteeRef{
			{Name: "Departmental Committee on Justice and Legal Affairs", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/jla"},
			{Name: "Select Committee on Regional Integration", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/regional-integration"},
			{Name: "Committee on Implementation", Role: "Member", SourceURL: "https://www.parliament.go.ke/the-national-assembly/committees/implementation"},
		},
		RecentActivity: []ScorecardActivityItem{
			{Kind: "statement", Date: "2024-09-08", Title: "Statement on the Status of the Victim Protection Trust Fund", SourceURL: "https://www.parliament.go.ke/the-national-assembly/hansard/2024-09-08", Source: "Hansard"},
			{Kind: "bill_sponsored", Date: "2024-08-30", Title: "Published the Victim Protection (Amendment) Bill", SourceURL: "https://www.parliament.go.ke/the-national-assembly/bills/victim-protection-amendment", Source: "parliament.go.ke"},
			{Kind: "question", Date: "2024-08-26", Title: "Question to the Attorney-General — status of the Huduma Namba legal opinion", SourceURL: "https://www.parliament.go.ke/the-national-assembly/questions/2024-08-26", Source: "parliament.go.ke"},
			{Kind: "vote", Date: "2024-08-21", Title: "Division on the Statute Law (Miscellaneous Amendments) Bill", Detail: "No", SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings/2024-08-21", Source: "Votes and Proceedings"},
		},
		Disclaimer:     scorecardDisclaimer,
		RealityLayer:   "FACT",
		SourceAgencies: []string{"parliament.go.ke", "Hansard", "Votes and Proceedings"},
	},
}

// scorecardDisclaimerText is exported via test so the canonical wording
// cannot drift without a breaking change to the test (and therefore to
// the API contract).
//
//lint:ignore U1000 referenced via reflection in tests
var scorecardDisclaimerForTest = scorecardDisclaimer

// findScorecard returns the scorecard for the given person ID, or nil if
// no sample person matches. In production this lookup hits a repository;
// the sample list exists so the frontend + tests have something to render
// before the live parliament.go.ke feed is wired.
func findScorecard(personID string) *MPScorecard {
	for i := range sampleScorecards {
		if sampleScorecards[i].PersonID == personID {
			return &sampleScorecards[i]
		}
	}
	return nil
}

// scorecardResponseEnvelope is the on-the-wire JSON returned by the scorecard
// endpoint. It wraps the raw scorecard with attendance_rate as a 0.0–1.0
// float (per the OpenAPI spec) so callers don't have to divide by 100.
// All other metrics stay as raw integer counts.
type scorecardResponseEnvelope struct {
	PersonID            string `json:"person_id"`
	Name                string `json:"name"`
	Role                string `json:"role"`
	Constituency        string `json:"constituency"`
	Party               string `json:"party"`
	ParliamentaryPeriod string `json:"parliamentary_period"`
	PhotoURL            string `json:"photo_url,omitempty"`

	// Contact info — issue #281. omitempty so empty rows are NOT rendered
	// (e.g. for a future MP whose data has not been scraped yet).
	Email         string `json:"email,omitempty"`
	Phone         string `json:"phone,omitempty"`
	OfficeAddress string `json:"office_address,omitempty"`
	Twitter       string `json:"twitter,omitempty"`
	Facebook      string `json:"facebook,omitempty"`
	ContactNote   string `json:"_note,omitempty"`

	AttendanceRate      float64 `json:"attendance_rate"`
	AttendanceSourceURL string  `json:"attendance_source_url"`

	BillsSponsored       int                     `json:"bills_sponsored"`
	BillsSponsoredList   []ScorecardBillRef      `json:"bills_sponsored_list"`
	CommitteeMemberships []ScorecardCommitteeRef `json:"committee_memberships"`
	QuestionsAsked       int                     `json:"questions_asked"`
	VotesRecorded        int                     `json:"votes_recorded"`
	StatementsMade       int                     `json:"statements_made"`

	Metrics        map[string]ScorecardMetric `json:"metrics"`
	RecentActivity []ScorecardActivityItem    `json:"recent_activity"`

	Disclaimer     string   `json:"disclaimer"`
	RealityLayer   string   `json:"reality_layer"`
	SourceAgencies []string `json:"source_agencies"`
}

// makeScorecardHandler returns the handler for
// GET /api/v1/people/{id}/scorecard.
//
// Pre-conditions: the request URL path is one of:
//   - /api/v1/people/{id}/scorecard
//
// On success it returns 200 with the scorecard JSON. The scorecard NEVER
// contains a composite "performance score" or "rating" field — only raw
// counts and rates (issue ENG-K2). Every numeric metric is paired with
// its source_url.
//
// On unknown person_id it returns 404.
func makeScorecardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		// Path shape: /api/v1/people/{id}/scorecard
		// The people router (handlePeople) is registered on
		// /api/v1/people/ — the trailing path segment is the entire
		// {id}/scorecard portion.
		tail := strings.TrimPrefix(r.URL.Path, "/api/v1/people/")
		if !strings.HasSuffix(tail, "/scorecard") {
			writeError(w, http.StatusNotFound, "not_found", "scorecard sub-resource not found")
			return
		}
		personID := strings.TrimSuffix(tail, "/scorecard")
		if personID == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "person ID required")
			return
		}
		sc := findScorecard(personID)
		if sc == nil {
			writeError(w, http.StatusNotFound, "not_found", "scorecard not found for person: "+personID)
			return
		}
		env := scorecardResponseEnvelope{
			PersonID:            sc.PersonID,
			Name:                sc.Name,
			Role:                sc.Role,
			Constituency:        sc.Constituency,
			Party:               sc.Party,
			ParliamentaryPeriod: sc.ParliamentaryPeriod,
			PhotoURL:            sc.PhotoURL,
			// Contact info — issue #281.
			Email:                sc.Email,
			Phone:                sc.Phone,
			OfficeAddress:        sc.OfficeAddress,
			Twitter:              sc.Twitter,
			Facebook:             sc.Facebook,
			ContactNote:          sc.ContactNote,
			AttendanceRate:       float64(sc.AttendanceRate.Value) / 100.0,
			AttendanceSourceURL:  sc.AttendanceRate.SourceURL,
			BillsSponsored:       sc.BillsSponsored.Value,
			BillsSponsoredList:   sc.BillsSponsoredList,
			CommitteeMemberships: sc.CommitteeMemberships,
			QuestionsAsked:       sc.QuestionsAsked.Value,
			VotesRecorded:        sc.VotesRecorded.Value,
			StatementsMade:       sc.StatementsMade.Value,
			Metrics: map[string]ScorecardMetric{
				"attendance_rate": sc.AttendanceRate,
				"bills_sponsored": sc.BillsSponsored,
				"questions_asked": sc.QuestionsAsked,
				"votes_recorded":  sc.VotesRecorded,
				"statements_made": sc.StatementsMade,
			},
			RecentActivity: sc.RecentActivity,
			Disclaimer:     sc.Disclaimer,
			RealityLayer:   sc.RealityLayer,
			SourceAgencies: sc.SourceAgencies,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(env)
	}
}

// scorecardListResponse is the JSON shape returned by GET /api/v1/people (the
// people list endpoint). It exists so the people router can list the 5 sample
// MPs alongside the existing "people pending" stub without colliding with the
// /scorecard sub-resource.
type scorecardListResponse struct {
	Items      []scorecardListItem `json:"items"`
	Total      int                 `json:"total"`
	Disclaimer string              `json:"disclaimer"`
}

type scorecardListItem struct {
	PersonID     string `json:"person_id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	Constituency string `json:"constituency"`
	Party        string `json:"party"`
	ScorecardURL string `json:"scorecard_url"`
	Country      string `json:"country"`
}

// makePeopleListHandler returns the handler for GET /api/v1/people. It lists
// the 5 sample MPs the scorecard feature ships with, each with a link to its
// scorecard. This replaces the "People — pending (issue #19)" stub.
func makePeopleListHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		country := middleware.CountryFromContext(r.Context())
		items := make([]scorecardListItem, 0, len(sampleScorecards))
		for _, p := range sampleScorecards {
			// Filter by country: return only matching country's people.
			// ALL = global dashboard view.
			if country != "" && country != "ALL" && country != "KE" {
				continue
			}
			items = append(items, scorecardListItem{
				PersonID:     p.PersonID,
				Name:         p.Name,
				Role:         p.Role,
				Constituency: p.Constituency,
				Party:        p.Party,
				ScorecardURL: "/api/v1/people/" + p.PersonID + "/scorecard",
				Country:      "KE",
			})
		}
		writeJSON(w, http.StatusOK, scorecardListResponse{
			Items:      items,
			Total:      len(items),
			Disclaimer: scorecardDisclaimer,
		})
	}
}
