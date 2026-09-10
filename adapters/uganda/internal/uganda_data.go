// Package internal holds Uganda-specific legislative data.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// UgandaBillStages defines Uganda's Bill lifecycle.
// Uganda is unicameral — the Parliament of Uganda is the sole legislative body.
// Stages: First Reading → Second Reading → Committee → Report → Third Reading → Assent
var UgandaBillStages = []contracts.StageDefinition{
	{
		Code:              "FIRST_READING",
		Name:              "First Reading",
		SimpleExplanation: "The Bill is read for the first time in Parliament. No debate yet.",
		Country:           "UG",
		AllowedNext:       []string{"SECOND_READING"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Second Reading",
		SimpleExplanation: "MPs debate the principles of the Bill. A vote decides whether it proceeds.",
		Country:           "UG",
		AllowedNext:       []string{"COMMITTEE_STAGE", "REJECTED"},
	},
	{
		Code:              "COMMITTEE_STAGE",
		Name:              "Committee Stage",
		SimpleExplanation: "A sectoral committee examines the Bill clause-by-clause.",
		Country:           "UG",
		AllowedNext:       []string{"REPORT_STAGE"},
	},
	{
		Code:              "REPORT_STAGE",
		Name:              "Report Stage",
		SimpleExplanation: "The committee reports back to Parliament. Further amendments may be proposed.",
		Country:           "UG",
		AllowedNext:       []string{"THIRD_READING"},
	},
	{
		Code:              "THIRD_READING",
		Name:              "Third Reading",
		SimpleExplanation: "Final debate and vote on whether to pass the Bill.",
		Country:           "UG",
		AllowedNext:       []string{"PRESIDENTIAL_ASSENT", "REJECTED"},
	},
	{
		Code:              "PRESIDENTIAL_ASSENT",
		Name:              "Presidential Assent",
		SimpleExplanation: "The President signs the Bill into law. May refer it back once.",
		Country:           "UG",
		AllowedNext:       []string{"COMMENCEMENT"},
	},
	{
		Code:              "COMMENCEMENT",
		Name:              "Commencement",
		SimpleExplanation: "The Act comes into force.",
		Country:           "UG",
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejected",
		SimpleExplanation: "The Bill was defeated at a vote.",
		Country:           "UG",
		IsTerminal:        true,
	},
}

// UgandaTerminology defines Ugandan parliamentary terms.
var UgandaTerminology = []contracts.TermDefinition{
	{Term: "First Reading", SimpleExplanation: "The Bill is introduced to Parliament.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Second Reading", SimpleExplanation: "MPs debate the principles of the Bill.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Third Reading", SimpleExplanation: "Final vote on the Bill.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Committee Stage", SimpleExplanation: "A sectoral committee examines the Bill clause-by-clause.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Presidential Assent", SimpleExplanation: "The President signs the Bill into law.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Hansard", SimpleExplanation: "The official record of parliamentary debates.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Order Paper", SimpleExplanation: "The daily agenda of Parliament.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Bill Tracker", SimpleExplanation: "A tool to track the progress of a Bill through Parliament.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Sectoral Committee", SimpleExplanation: "A committee that examines Bills related to a specific sector.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Act of Parliament", SimpleExplanation: "A Bill that has been passed by Parliament and assented to by the President.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Commencement", SimpleExplanation: "The date an Act comes into force.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Motion", SimpleExplanation: "A formal proposal put before Parliament for discussion and decision.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Petition", SimpleExplanation: "A formal request to Parliament by citizens.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Caucus", SimpleExplanation: "A meeting of members of a political party.", Country: "UG"},
	{Term: "Division", SimpleExplanation: "A formal vote where members' names are recorded.", Country: "UG"},
	{Term: "Quorum", SimpleExplanation: "The minimum number of members required for Parliament to conduct business.", Country: "UG"},
	{Term: "Speaker", SimpleExplanation: "The presiding officer of Parliament.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Clerk to Parliament", SimpleExplanation: "The senior administrative officer of Parliament.", Country: "UG", Sources: []string{"https://www.parliament.go.ug"}},
	{Term: "Deputy Speaker", SimpleExplanation: "Deputises the Speaker in presiding over Parliament.", Country: "UG"},
	{Term: "Leader of Government Business", SimpleExplanation: "The MP responsible for guiding government business in Parliament.", Country: "UG"},
	{Term: "Leader of the Opposition", SimpleExplanation: "The MP who leads the official opposition in Parliament.", Country: "UG"},
	{Term: "Backbencher", SimpleExplanation: "An MP who does not hold a ministerial or opposition frontbench position.", Country: "UG"},
	{Term: "Whip", SimpleExplanation: "An MP responsible for party discipline and attendance.", Country: "UG"},
	{Term: "Reading", SimpleExplanation: "A stage in the Bill process where the Bill is formally presented to Parliament.", Country: "UG"},
	{Term: "Royal Assent", SimpleExplanation: "Not applicable in Uganda — see Presidential Assent.", Country: "UG"},
}

// UgandaLegislativeStructure returns Uganda's institutional structure.
// KEY: Uganda is UNICAMERAL — only one House (Parliament), unlike Kenya's
// bicameral (National Assembly + Senate).
func UgandaLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "UG",
		CountryCode: "UG",
		CountryName: "Uganda",
		Houses: []contracts.HouseDefinition{
			{
				Code:     "PARLIAMENT",
				Name:     "Parliament of Uganda",
				Type:     contracts.HouseTypeSingle,
				Members:  556,
				TermDays: 5 * 365,
			},
		},
	}
}
