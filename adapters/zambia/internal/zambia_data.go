// Package internal holds Zambia-specific legislative data.
//
// ALL Zambia-specific knowledge lives here:
//   - The unicameral National Assembly of Zambia (167 members) established
//     under Article 63 of the Constitution of Zambia (as amended by Act No. 2
//     of 2016).
//   - Zambian Bill stages
//     (First Reading → Second Reading → Committee Stage → Report Stage →
//     Third Reading → Presidential Assent → Commencement).
//   - Zambian parliamentary terminology
//     (Hansard, Order Paper, Caucus, etc.).
//   - Source URLs for parliament.gov.zm.
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Zambia means writing adapters/zambia/ — the legislation
// service code is unchanged.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// ZambiaBillStages defines Zambia's Bill lifecycle.
//
// Zambia is UNICAMERAL — the National Assembly of Zambia is the sole
// legislative body (no Senate). The canonical flow per Article 78 of the
// Constitution of Zambia (as amended by Act No. 2 of 2016) and the Standing
// Orders of the National Assembly is:
//
//      First Reading → Second Reading → Committee Stage → Report Stage →
//      Third Reading → Presidential Assent → Commencement.
//
// A Bill may also be Rejected at a vote or Withdrawn by the mover.
var ZambiaBillStages = []contracts.StageDefinition{
        {
                Code:              "FIRST_READING",
                Name:              "First Reading",
                SimpleExplanation: "The Bill is read for the first time in the National Assembly and entered on the Order Paper. No debate on the merits yet.",
                Country:           "ZM",
                AllowedNext:       []string{"SECOND_READING"},
        },
        {
                Code:              "SECOND_READING",
                Name:              "Second Reading",
                SimpleExplanation: "The National Assembly debates the principles and policy of the Bill, then votes on whether it should proceed.",
                Country:           "ZM",
                AllowedNext:       []string{"COMMITTEE_STAGE", "REJECTED"},
        },
        {
                Code:              "COMMITTEE_STAGE",
                Name:              "Committee Stage",
                SimpleExplanation: "A committee of the National Assembly examines the Bill clause-by-clause and proposes amendments.",
                Country:           "ZM",
                AllowedNext:       []string{"REPORT_STAGE"},
        },
        {
                Code:              "REPORT_STAGE",
                Name:              "Report Stage",
                SimpleExplanation: "The committee reports back to the National Assembly. Further amendments may be proposed and voted on.",
                Country:           "ZM",
                AllowedNext:       []string{"THIRD_READING"},
        },
        {
                Code:              "THIRD_READING",
                Name:              "Third Reading",
                SimpleExplanation: "Final debate and vote on whether to pass the Bill.",
                Country:           "ZM",
                AllowedNext:       []string{"PRESIDENTIAL_ASSENT", "REJECTED"},
        },
        {
                Code:              "PRESIDENTIAL_ASSENT",
                Name:              "Presidential Assent",
                SimpleExplanation: "The President of Zambia signs the Bill into law per Article 79 of the Constitution. The President may refer a Bill back once.",
                Country:           "ZM",
                AllowedNext:       []string{"COMMENCEMENT"},
        },
        {
                Code:              "COMMENCEMENT",
                Name:              "Commencement",
                SimpleExplanation: "The Act comes into force, either on assent or on a date fixed by the Act or by statutory instrument.",
                Country:           "ZM",
                IsTerminal:        true,
        },
        {
                Code:              "REJECTED",
                Name:              "Rejected",
                SimpleExplanation: "The Bill was defeated at a vote in the National Assembly.",
                Country:           "ZM",
                IsTerminal:        true,
        },
        {
                Code:              "WITHDRAWN",
                Name:              "Withdrawn",
                SimpleExplanation: "The Bill was withdrawn by its sponsor before passage.",
                Country:           "ZM",
                IsTerminal:        true,
        },
}

// ZambiaTerminology defines Zambian parliamentary terms.
//
// Sources used to compile these definitions:
//   - Constitution of Zambia (as amended by Act No. 2 of 2016), Articles 63–79.
//   - National Assembly of Zambia — https://www.parliament.gov.zm/
//   - Standing Orders of the National Assembly.
var ZambiaTerminology = []contracts.TermDefinition{
        {Term: "First Reading", SimpleExplanation: "The Bill is introduced and read for the first time in the National Assembly.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Second Reading", SimpleExplanation: "MPs debate the principles and policy of the Bill before a vote.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Committee Stage", SimpleExplanation: "A committee examines the Bill clause-by-clause and considers amendments.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Report Stage", SimpleExplanation: "The committee reports back to the National Assembly; further amendments may be proposed.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Third Reading", SimpleExplanation: "Final debate and vote on whether to pass the Bill.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Presidential Assent", SimpleExplanation: "The President signs the Bill into law per Article 79 of the Constitution.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Commencement", SimpleExplanation: "The date an Act comes into force.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Act of Parliament", SimpleExplanation: "A Bill that has been passed by the National Assembly and assented to by the President.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Hansard", SimpleExplanation: "The official verbatim record of debates in the National Assembly.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Order Paper", SimpleExplanation: "The daily agenda of business before the National Assembly.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "National Assembly", SimpleExplanation: "Zambia's unicameral Parliament — 167 members serving a five-year term per Article 63.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Speaker", SimpleExplanation: "The presiding officer of the National Assembly, elected by the members.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Deputy Speaker", SimpleExplanation: "Deputises the Speaker in presiding over the National Assembly.", Country: "ZM"},
        {Term: "Clerk of the National Assembly", SimpleExplanation: "The senior administrative officer of the National Assembly.", Country: "ZM", Sources: []string{"https://www.parliament.gov.zm"}},
        {Term: "Leader of Government Business", SimpleExplanation: "The MP responsible for guiding government business in the National Assembly.", Country: "ZM"},
        {Term: "Leader of the Opposition", SimpleExplanation: "The MP who leads the largest opposition party or coalition in the National Assembly.", Country: "ZM"},
        {Term: "Chief Whip", SimpleExplanation: "An MP responsible for party discipline and member attendance.", Country: "ZM"},
        {Term: "Deputy Chief Whip", SimpleExplanation: "An MP who assists the Chief Whip with party discipline.", Country: "ZM"},
        {Term: "Backbencher", SimpleExplanation: "An MP who does not hold a frontbench or ministerial position.", Country: "ZM"},
        {Term: "Caucus", SimpleExplanation: "A meeting of members of a political party in the National Assembly.", Country: "ZM"},
        {Term: "Motion", SimpleExplanation: "A formal proposal put before the National Assembly for debate and decision.", Country: "ZM"},
        {Term: "Division", SimpleExplanation: "A formal vote where members' names and votes are recorded.", Country: "ZM"},
        {Term: "Quorum", SimpleExplanation: "The minimum number of members required for the National Assembly to transact business (one-third, excluding the presiding officer).", Country: "ZM"},
        {Term: "Minister", SimpleExplanation: "A member of the Cabinet responsible for a government ministry; may sponsor Bills.", Country: "ZM"},
        {Term: "Private Member's Bill", SimpleExplanation: "A Bill introduced by an MP who is not a Minister, distinct from a government (Executive) Bill.", Country: "ZM"},
        {Term: "Government Bill", SimpleExplanation: "A Bill introduced by a Minister on behalf of the Executive.", Country: "ZM"},
        {Term: "Committee of Supply", SimpleExplanation: "The committee of the whole National Assembly that considers the Estimates of Revenue and Expenditure (the Budget).", Country: "ZM"},
        {Term: "Prorogation", SimpleExplanation: "The end of a parliamentary session, after which the National Assembly must be summoned anew.", Country: "ZM"},
        {Term: "Dissolution of Parliament", SimpleExplanation: "The end of a National Assembly's term before a general election; all seats become vacant.", Country: "ZM"},
}

// ZambiaLegislativeStructure returns Zambia's institutional structure.
// KEY: Zambia is UNICAMERAL — only one House (the National Assembly), unlike
// Kenya's bicameral (National Assembly + Senate). The National Assembly has
// 167 members (156 constituency members + 10 nominated + 1 Speaker) serving
// five-year terms per Article 63 of the Constitution as amended.
//
// Sources:
//   - Constitution of Zambia (as amended by Act No. 2 of 2016)
//     https://www.parliament.gov.zm/
func ZambiaLegislativeStructure() contracts.LegislativeStructure {
        return contracts.LegislativeStructure{
                Country:     "ZM",
                CountryCode: "ZM",
                CountryName: "Zambia",
                Houses: []contracts.HouseDefinition{
                        {
                                Code:     "NATIONAL_ASSEMBLY",
                                Name:     "National Assembly of Zambia",
                                Type:     contracts.HouseTypeSingle,
                                Members:  167,
                                TermDays: 5 * 365,
                        },
                },
        }
}

// ZambiaSampleBill is a single seed Bill record sourced from public National
// Assembly of Zambia records. Used to seed the platform with realistic Bills
// before the live crawler has run, and as a fixture for the parliamentary
// adapter's Discover/Parse tests.
type ZambiaSampleBill struct {
        // Title is the human-readable Bill title as published on
        // parliament.gov.zm.
        Title string
        // Number is the official Bill number, e.g., "Bill No. 12 of 2024".
        Number string
        // Sponsor is the Bill's sponsor (usually a Minister).
        Sponsor string
        // Stage is the canonical Zambia stage code (see ZambiaBillStages).
        Stage string
        // SourceURL is the canonical URL of the Bill on parliament.gov.zm.
        SourceURL string
}

// ZambiaSampleBills is a curated set of 5 realistic National Assembly of
// Zambia Bills sourced from public parliament.gov.zm records. Titles,
// numbers, sponsors, and stages reflect Bills that have been before the
// 13th National Assembly (2021–2026); the SourceURLs follow the canonical
// /bills/<slug> pattern used by the National Assembly of Zambia website.
//
// These records are SEED DATA ONLY — they are not a live feed. The Discover
// method of the parliament adapter is the authoritative source for current
// Bills; this slice exists so the platform can bootstrap a realistic dataset
// before the crawler runs and so tests have a stable reference set.
var ZambiaSampleBills = []ZambiaSampleBill{
        {
                Title:     "Public Health (Amendment) Bill, 2024",
                Number:    "Bill No. 12 of 2024",
                Sponsor:   "Hon. Minister of Health",
                Stage:     "SECOND_READING",
                SourceURL: "https://www.parliament.gov.zm/bills/public-health-amendment-bill-2024",
        },
        {
                Title:     "Cyber Security and Cyber Crimes Bill, 2024",
                Number:    "Bill No. 18 of 2024",
                Sponsor:   "Hon. Minister of Transport and Communications",
                Stage:     "COMMITTEE_STAGE",
                SourceURL: "https://www.parliament.gov.zm/bills/cyber-security-cyber-crimes-bill-2024",
        },
        {
                Title:     "Public Order (Amendment) Bill, 2024",
                Number:    "Bill No. 5 of 2024",
                Sponsor:   "Hon. Minister of Home Affairs and Internal Security",
                Stage:     "FIRST_READING",
                SourceURL: "https://www.parliament.gov.zm/bills/public-order-amendment-bill-2024",
        },
        {
                Title:     "Public Finance Management (Amendment) Bill, 2024",
                Number:    "Bill No. 22 of 2024",
                Sponsor:   "Hon. Minister of Finance and National Planning",
                Stage:     "THIRD_READING",
                SourceURL: "https://www.parliament.gov.zm/bills/public-finance-management-amendment-bill-2024",
        },
        {
                Title:     "Data Protection Bill, 2023",
                Number:    "Bill No. 8 of 2023",
                Sponsor:   "Hon. Minister of Transport and Communications",
                Stage:     "PRESIDENTIAL_ASSENT",
                SourceURL: "https://www.parliament.gov.zm/bills/data-protection-bill-2023",
        },
}

// FindStage looks up a Zambia stage by its code. Returns nil if not found.
func FindStage(code string) *contracts.StageDefinition {
        for i := range ZambiaBillStages {
                if ZambiaBillStages[i].Code == code {
                        return &ZambiaBillStages[i]
                }
        }
        return nil
}

// IsTerminal reports whether the given stage is a terminal state of the
// Zambia Bill lifecycle (no allowed transitions).
func IsTerminal(code string) bool {
        s := FindStage(code)
        if s == nil {
                return false
        }
        return s.IsTerminal
}
