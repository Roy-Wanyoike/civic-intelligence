// Package internal holds Tanzania-specific legislative data.
//
// Tanzania is UNICAMERAL — the Bunge la Tanzania (Parliament of Tanzania) is
// the sole legislative body at the Union level. This mirrors the Uganda
// adapter's structure (also unicameral) rather than the Kenya adapter
// (bicameral: National Assembly + Senate).
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// TanzaniaBillStages defines the lifecycle of a Bill in the Bunge la Tanzania.
// Stages (per the task #154 spec):
//
//	First Reading → Second Reading → Committee → Report →
//	Third Reading → Assent → Commencement
//
// Tanzania is unicameral — the Bunge is the sole legislative chamber.
// Reference: https://www.parliament.go.tz
var TanzaniaBillStages = []contracts.StageDefinition{
	{
		Code:              "FIRST_READING",
		Name:              "First Reading",
		SimpleExplanation: "The Bill is read for the first time in the Bunge. No debate occurs at this stage; the Bill is committed to the appropriate committee.",
		Country:           "TZ",
		Order:             1,
		AllowedNext:       []string{"SECOND_READING"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Second Reading",
		SimpleExplanation: "Members of Parliament debate the principles and policy of the Bill. A vote decides whether the Bill proceeds to committee scrutiny.",
		Country:           "TZ",
		Order:             2,
		AllowedNext:       []string{"COMMITTEE_STAGE", "REJECTED"},
	},
	{
		Code:              "COMMITTEE_STAGE",
		Name:              "Committee",
		SimpleExplanation: "A standing or sectoral committee examines the Bill clause-by-clause and may propose amendments.",
		Country:           "TZ",
		Order:             3,
		AllowedNext:       []string{"REPORT_STAGE"},
	},
	{
		Code:              "REPORT_STAGE",
		Name:              "Report",
		SimpleExplanation: "The committee reports back to the Bunge. Members may propose further amendments to the committee's report.",
		Country:           "TZ",
		Order:             4,
		AllowedNext:       []string{"THIRD_READING"},
	},
	{
		Code:              "THIRD_READING",
		Name:              "Third Reading",
		SimpleExplanation: "Final debate and vote on whether to pass the Bill. No substantial amendments are made at this stage.",
		Country:           "TZ",
		Order:             5,
		AllowedNext:       []string{"PRESIDENTIAL_ASSENT", "REJECTED"},
	},
	{
		Code:              "PRESIDENTIAL_ASSENT",
		Name:              "Assent",
		SimpleExplanation: "The President of the United Republic signs the Bill into law. The President may refer the Bill back once for reconsideration.",
		Country:           "TZ",
		Order:             6,
		AllowedNext:       []string{"COMMENCEMENT"},
	},
	{
		Code:              "COMMENCEMENT",
		Name:              "Commencement",
		SimpleExplanation: "The Act comes into force on the date of publication in the Government Gazette, or on a date specified within the Act.",
		Country:           "TZ",
		Order:             7,
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejected",
		SimpleExplanation: "The Bill was defeated at a vote in the Bunge.",
		Country:           "TZ",
		Order:             99,
		IsTerminal:        true,
	},
}

// TanzaniaTerminology defines Tanzanian parliamentary terms. All entries
// include a source URL pointing at the Parliament of Tanzania's official
// website (parliament.go.tz) where applicable.
//
// Per issue #154: at least 20 parliamentary terms must be defined.
var TanzaniaTerminology = []contracts.TermDefinition{
	{Term: "Bunge", SimpleExplanation: "The Swahili word for Parliament — the national legislature of the United Republic of Tanzania.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Bunge la Tanzania", SimpleExplanation: "The Parliament of Tanzania — the unicameral legislative body of the United Republic.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "First Reading", SimpleExplanation: "The Bill is formally introduced to the Bunge; no debate is held at this stage.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Second Reading", SimpleExplanation: "Members debate the principles and policy of the Bill before a vote on committal.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Third Reading", SimpleExplanation: "The final debate and vote on whether to pass the Bill.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Committee Stage", SimpleExplanation: "A standing or sectoral committee scrutinises the Bill clause-by-clause and proposes amendments.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Report Stage", SimpleExplanation: "The committee presents its report to the Bunge; members may move further amendments.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Assent", SimpleExplanation: "The President of the United Republic signs a passed Bill into law.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Commencement", SimpleExplanation: "The date an Act of Parliament comes into force, usually upon gazettement.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Hansard", SimpleExplanation: "The official verbatim record of debates in the Bunge.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Order Paper", SimpleExplanation: "The formal daily agenda of business before the Bunge.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Standing Orders", SimpleExplanation: "The written rules that govern the procedure and conduct of business in the Bunge.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Government Bill", SimpleExplanation: "A Bill introduced by a Minister or the Attorney General on behalf of the Government.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Private Member's Bill", SimpleExplanation: "A Bill introduced by a Member of Parliament who is not a Minister.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Act of Parliament", SimpleExplanation: "A Bill that has been passed by the Bunge and assented to by the President.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Speaker", SimpleExplanation: "The presiding officer of the Bunge, elected by Members from among themselves or from outside the House.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Deputy Speaker", SimpleExplanation: "Deputises the Speaker and presides over the Bunge in the Speaker's absence.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Clerk of the National Assembly", SimpleExplanation: "The senior administrative officer of the Bunge, responsible for procedural and record-keeping functions.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Attorney General", SimpleExplanation: "The principal legal adviser to the Government, who may attend and speak in the Bunge ex officio.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Leader of Government Business", SimpleExplanation: "The Minister responsible for arranging and guiding Government business in the Bunge.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Leader of the Official Opposition", SimpleExplanation: "The MP who leads the Official Opposition shadow frontbench in the Bunge.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Backbencher", SimpleExplanation: "An MP who does not hold a ministerial, opposition frontbench, or presiding office role.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Whip", SimpleExplanation: "An MP responsible for party discipline and attendance in the Bunge.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Quorum", SimpleExplanation: "The minimum number of Members required to be present for the Bunge to transact business.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Division", SimpleExplanation: "A formal recorded vote in which Members' positions are individually counted.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Special Seats", SimpleExplanation: "Constitutionally reserved seats for women allocated to political parties proportionally to ensure at least 30% female representation.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Union Matters", SimpleExplanation: "Matters affecting the United Republic of Tanzania that fall under the jurisdiction of the Union Parliament (as distinct from Zanzibar's own House of Representatives).", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Constituency", SimpleExplanation: "An electoral area represented by a single elected Member of Parliament.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Government Gazette", SimpleExplanation: "The official publication in which Acts, notices, and statutory instruments of the United Republic are published.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
	{Term: "Bill Tracker", SimpleExplanation: "A tool to follow the progress of a Bill through the Bunge's stages.", Country: "TZ", Sources: []string{"https://www.parliament.go.tz"}},
}

// TanzaniaLegislativeStructure returns Tanzania's institutional structure.
//
// KEY: Tanzania is UNICAMERAL — only one House (the Bunge la Tanzania), unlike
// Kenya's bicameral (National Assembly + Senate) setup. The Bunge has 393
// members: 266 elected from constituencies + 117 special seats (women's
// reserved seats plus Zanzibar and presidential appointees).
//
// Reference: https://www.parliament.go.tz
func TanzaniaLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "TZ",
		CountryCode: "TZ",
		CountryName: "Tanzania",
		Houses: []contracts.HouseDefinition{
			{
				Code:     "BUNGE",
				Name:     "Bunge la Tanzania",
				Type:     contracts.HouseTypeSingle,
				Members:  393,
				TermDays: 5 * 365,
			},
		},
	}
}
