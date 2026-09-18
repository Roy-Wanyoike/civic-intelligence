// Package internal holds Ghana-specific legislative data.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// GhanaBillStages defines Ghana's Bill lifecycle.
// Ghana is unicameral — the Parliament of Ghana is the sole legislative body.
// Stages: First Reading → Second Reading → Consideration Stage → Third Reading → Assent → Commencement
//
// Note: Ghana uses "Consideration Stage" in place of the Commonwealth-wide
// "Committee Stage" — it is a stage of the whole House, taken in Committee of
// the Whole, where the Bill is examined clause-by-clause.
var GhanaBillStages = []contracts.StageDefinition{
	{
		Code:              "FIRST_READING",
		Name:              "First Reading",
		SimpleExplanation: "The Bill is read for the first time in Parliament. No debate yet.",
		Country:           "GH",
		AllowedNext:       []string{"SECOND_READING"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Second Reading",
		SimpleExplanation: "MPs debate the principles and policy of the Bill. A vote decides whether it proceeds.",
		Country:           "GH",
		AllowedNext:       []string{"CONSIDERATION_STAGE", "REJECTED"},
	},
	{
		Code:              "CONSIDERATION_STAGE",
		Name:              "Consideration Stage",
		SimpleExplanation: "Parliament, sitting as a Committee of the Whole, examines the Bill clause-by-clause and considers amendments.",
		Country:           "GH",
		AllowedNext:       []string{"THIRD_READING"},
	},
	{
		Code:              "THIRD_READING",
		Name:              "Third Reading",
		SimpleExplanation: "Final debate and vote on whether to pass the Bill.",
		Country:           "GH",
		AllowedNext:       []string{"ASSENT", "REJECTED"},
	},
	{
		Code:              "ASSENT",
		Name:              "Assent",
		SimpleExplanation: "The President assents to the Bill, signifying agreement to enact it into law.",
		Country:           "GH",
		AllowedNext:       []string{"COMMENCEMENT"},
	},
	{
		Code:              "COMMENCEMENT",
		Name:              "Commencement",
		SimpleExplanation: "The Act comes into force, either on assent or on a date specified in the Act.",
		Country:           "GH",
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejected",
		SimpleExplanation: "The Bill was defeated at a vote.",
		Country:           "GH",
		IsTerminal:        true,
	},
	{
		Code:              "WITHDRAWN",
		Name:              "Withdrawn",
		SimpleExplanation: "The Bill was withdrawn by its sponsor before passage.",
		Country:           "GH",
		IsTerminal:        true,
	},
}

// GhanaTerminology defines Ghanaian parliamentary terms.
var GhanaTerminology = []contracts.TermDefinition{
	{Term: "First Reading", SimpleExplanation: "The Bill is introduced and read for the first time in Parliament.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Second Reading", SimpleExplanation: "MPs debate the principles and policy of the Bill before a vote.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Consideration Stage", SimpleExplanation: "Parliament, in Committee of the Whole, examines the Bill clause-by-clause and considers amendments.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Third Reading", SimpleExplanation: "Final debate and vote on whether to pass the Bill.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Assent", SimpleExplanation: "The President assents to the Bill, enacting it into law.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Commencement", SimpleExplanation: "The date an Act comes into force.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Act of Parliament", SimpleExplanation: "A Bill that has been passed by Parliament and assented to by the President.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Hansard", SimpleExplanation: "The official verbatim record of parliamentary debates.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Order Paper", SimpleExplanation: "The daily agenda of business before Parliament.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Constitutional Instrument", SimpleExplanation: "A subsidiary instrument made under authority of the Constitution.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Legislative Instrument", SimpleExplanation: "Subsidiary legislation made by a person or body under authority of an Act of Parliament.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Committee of the Whole", SimpleExplanation: "A committee comprising all MPs, used during the Consideration Stage to examine a Bill clause-by-clause.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Select Committee", SimpleExplanation: "A permanent committee of Parliament that examines Bills and issues within a specific subject area.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Standing Committee", SimpleExplanation: "A committee established under the Standing Orders of Parliament for the duration of a Parliament.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Ad Hoc Committee", SimpleExplanation: "A committee set up for a specific task and dissolved once it reports.", Country: "GH"},
	{Term: "Speaker", SimpleExplanation: "The presiding officer of Parliament, elected by MPs from among themselves or from outside.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Deputy Speaker", SimpleExplanation: "Assists the Speaker and presides in the Speaker's absence; comprises First and Second Deputy Speakers.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Clerk to Parliament", SimpleExplanation: "The senior administrative officer and chief advisor on parliamentary procedure.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Majority Leader", SimpleExplanation: "The MP who leads the party or coalition with the most seats in Parliament and guides government business.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Minority Leader", SimpleExplanation: "The MP who leads the largest party or coalition not in government.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Majority Chief Whip", SimpleExplanation: "An MP of the governing side responsible for party discipline and member attendance.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Minority Chief Whip", SimpleExplanation: "An MP of the opposition side responsible for party discipline and member attendance.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Leader of Government Business", SimpleExplanation: "The MP, usually the Majority Leader, responsible for scheduling and guiding government business in Parliament.", Country: "GH"},
	{Term: "Backbencher", SimpleExplanation: "An MP who does not hold a frontbench or ministerial position.", Country: "GH"},
	{Term: "Minister", SimpleExplanation: "A member of the executive, usually an MP or senator, responsible for a government ministry.", Country: "GH"},
	{Term: "Motion", SimpleExplanation: "A formal proposal put before Parliament for debate and decision.", Country: "GH"},
	{Term: "Division", SimpleExplanation: "A formal vote in which MPs' names and votes are recorded.", Country: "GH"},
	{Term: "Quorum", SimpleExplanation: "The minimum number of MPs (one-third, excluding the presiding officer) required for Parliament to transact business.", Country: "GH", Sources: []string{"https://parliament.gh"}},
	{Term: "Prorogation", SimpleExplanation: "The end of a parliamentary session, after which Parliament must be summoned anew.", Country: "GH"},
	{Term: "Dissolution of Parliament", SimpleExplanation: "The end of a Parliament's term before a general election; all seats become vacant.", Country: "GH"},
}

// GhanaSampleBills is a representative sample of recent Ghana Parliament
// Bills, used to seed local fixtures (testdata/bills.html) and contract
// tests without touching the network. Titles are realistic Ghanaian Bill
// titles drawn from publicly published Bills.
//
// These records are values, not constants — they exist so test fixtures can
// be regenerated deterministically and so the adapter's Parse step has a
// known-shape input to extract from.
var GhanaSampleBills = []GhanaSampleBill{
	{
		Title:      "The Right to Information (Amendment) Bill, 2024",
		BillNumber: "Bill No. 12 of 2024",
		Sponsor:    "Minister for Justice and Attorney-General",
		Stage:      "Consideration Stage",
		Date:       "12 March 2024",
		URL:        "https://parliament.ghana.gov.gh/business/bills/rti-amendment-2024.pdf",
	},
	{
		Title:      "The Minerals Income Tax (Amendment) Bill, 2024",
		BillNumber: "Bill No. 18 of 2024",
		Sponsor:    "Minister for Finance",
		Stage:      "Second Reading",
		Date:       "27 February 2024",
		URL:        "https://parliament.ghana.gov.gh/business/bills/minerals-income-tax-2024.pdf",
	},
	{
		Title:      "The Public Universities Bill, 2024",
		BillNumber: "Bill No. 22 of 2024",
		Sponsor:    "Minister for Education",
		Stage:      "First Reading",
		Date:       "06 May 2024",
		URL:        "https://parliament.ghana.gov.gh/business/bills/public-universities-2024.pdf",
	},
	{
		Title:      "The Cyber Security (Amendment) Bill, 2024",
		BillNumber: "Bill No. 27 of 2024",
		Sponsor:    "Minister for Communications and Digitalisation",
		Stage:      "Third Reading",
		Date:       "18 July 2024",
		URL:        "https://parliament.ghana.gov.gh/business/bills/cyber-security-amendment-2024.pdf",
	},
	{
		Title:      "The Companies (Amendment) Bill, 2024",
		BillNumber: "Bill No. 31 of 2024",
		Sponsor:    "Minister for Justice and Attorney-General",
		Stage:      "Assent",
		Date:       "02 October 2024",
		URL:        "https://parliament.ghana.gov.gh/business/bills/companies-amendment-2024.pdf",
	},
}

// GhanaSampleBill is a single sample Bill record. Field names mirror the
// bill-card HTML structure parsed by parliament.ParseBillsListing.
type GhanaSampleBill struct {
	Title      string
	BillNumber string
	Sponsor    string
	Stage      string
	Date       string
	URL        string
}

// GhanaLegislativeStructure returns Ghana's institutional structure.
// KEY: Ghana is UNICAMERAL — only one House (Parliament), unlike Kenya's
// bicameral (National Assembly + Senate). The Parliament of Ghana has 275
// elected Members, each representing one of the 275 constituencies, serving
// four-year terms under Article 97 of the 1992 Constitution.
func GhanaLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "GH",
		CountryCode: "GH",
		CountryName: "Ghana",
		Houses: []contracts.HouseDefinition{
			{
				Code:     "PARLIAMENT",
				Name:     "Parliament of Ghana",
				Type:     contracts.HouseTypeSingle,
				Members:  275,
				TermDays: 4 * 365,
			},
		},
	}
}
