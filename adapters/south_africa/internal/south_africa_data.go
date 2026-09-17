// Package internal holds South Africa-specific legislative data that must NEVER
// leak into the global platform types. The contracts package defines only
// country-agnostic shapes (StageDefinition, TermDefinition, HouseDefinition).
// Every South African string — "National Assembly", "National Council of
// Provinces", "NCOP Concurrence", "Presidential Assent", "Hansard" — lives
// in this package.
//
// The South Africa adapter's adapter.go projects these richer records down into
// the plain contracts.* types when the ingestion service asks for them.
//
// Sources used to compile these definitions:
//   - Constitution of the Republic of South Africa, 1996 (Chapters 4–6)
//     https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996
//   - Parliament of South Africa — Bills & Laws
//     https://www.parliament.gov.za/bills-and-laws
//   - Parliament of South Africa — How a Bill becomes law
//     https://www.parliament.gov.za/project-vote/education
//   - The Rules of the National Assembly (9th edition)
//   - The Rules of the National Council of Provinces
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// ---------------------------------------------------------------------------
// House codes
// ---------------------------------------------------------------------------

// House codes used throughout the South Africa adapter. They appear in
// RawMetadata and SourceItem.House fields, but never as bare strings in the
// global domain model.
const (
	HouseCodeNationalAssembly           = "NA"  // lower house
	HouseCodeNationalCouncilOfProvinces = "NCOP" // upper house
)

// ---------------------------------------------------------------------------
// Stage codes
// ---------------------------------------------------------------------------

// StageCode is the stable identifier South Africa uses for a Bill lifecycle
// stage. These codes are the ONLY stage-related strings that ever cross into
// the global domain model (they are case-normalised SCREAMING_SNAKE strings,
// namespaced implicitly by the adapter's CountryCode()).
//
// The codes follow the conventions of the Constitution of the Republic of
// South Africa, 1996 (Chapter 4) and the Rules of the National Assembly and
// the National Council of Provinces.
type StageCode string

const (
	// StageIntroduction is publication + introduction (First Reading). The Bill
	// is introduced in the National Assembly and published in the Government
	// Gazette.
	StageIntroduction StageCode = "INTRODUCTION"

	// StageCommittee is the portfolio/standing committee stage where the Bill
	// is scrutinised clause-by-clause and public submissions are invited.
	StageCommittee StageCode = "COMMITTEE"

	// StagePublicParticipation is the constitutionally-mandated public
	// participation process (section 59 / 72 of the Constitution).
	StagePublicParticipation StageCode = "PUBLIC_PARTICIPATION"

	// StageNAVote is the vote on the Bill in the National Assembly after the
	// committee has reported.
	StageNAVote StageCode = "NA_VOTE"

	// StageNCOPConcurrence is the concurrence vote by the National Council of
	// Provinces. Section 75 Bills require an NCOP vote; Section 76 Bills require
	// each provincial delegation to mandate a vote.
	StageNCOPConcurrence StageCode = "NCOP_CONCURRENCE"

	// StageMediation is used when the NA and NCOP disagree and a Mediation
	// Committee is convened per section 76 of the Constitution.
	StageMediation StageCode = "MEDIATION"

	// StagePresidentialAssent is assent by the President per section 79 of the
	// Constitution.
	StagePresidentialAssent StageCode = "PRESIDENTIAL_ASSENT"

	// StageCommencement is when an Act is brought into force by a
	// commencement notice in the Government Gazette.
	StageCommencement StageCode = "COMMENCEMENT"

	// StageRejected is the terminal state for a Bill that fails to pass a vote.
	StageRejected StageCode = "REJECTED"

	// StageWithdrawn is the terminal state for a Bill that is withdrawn by the
	// member or Minister who introduced it.
	StageWithdrawn StageCode = "WITHDRAWN"

	// StageLapsed is the terminal state for a Bill that lapses at the end of a
	// Parliament (or at the end of the annual session, per NA Rule 333) without
	// concluding.
	StageLapsed StageCode = "LAPSED"
)

// ---------------------------------------------------------------------------
// Stage records
// ---------------------------------------------------------------------------

// SouthAfricaStage is the richer, South Africa-specific stage record. It
// carries the country code, the official definition (with citation to the
// Constitution or the Rules), and the canonical source URLs for the stage.
//
// The platform NEVER receives a SouthAfricaStage directly. adapter.go projects
// it down to a contracts.StageDefinition when the ingestion service asks for
// the BillStages.
type SouthAfricaStage struct {
	Code                StageCode
	Name                string
	SimpleExplanation   string
	OfficialDefinition  string
	Country             string // always "ZA"
	Order               int
	AllowedNext         []StageCode
	RequiresEvidence    bool
	RequiresVote        bool
	TypicalDurationDays int
	Sources             []string
}

// SouthAfricaBillStages is the ordered, authoritative list of South Africa's
// Bill stages.
//
// Sources used to compile these definitions:
//   - Constitution of the Republic of South Africa, 1996, Chapter 4 (Parliament)
//     https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996
//   - Parliament of South Africa — How a Law is Made
//     https://www.parliament.gov.za/
//   - The Rules of the National Assembly (current edition)
//   - The Rules of the National Council of Provinces (current edition)
//
// Stage transitions follow the South African legislative process: a Bill is
// introduced in the NA, sent to committee with public participation, voted on
// by the NA, then considered by the NCOP (with a Mediation Committee if the
// two houses disagree on a Section 76 Bill), assented to by the President,
// and brought into force by a commencement notice.
var SouthAfricaBillStages = []SouthAfricaStage{
	{
		Code:               StageIntroduction,
		Name:               "Introduction",
		Order:              1,
		Country:            "ZA",
		SimpleExplanation:  "The Bill is introduced (read for the first time) in the National Assembly and published in the Government Gazette. No debate on the merits takes place.",
		OfficialDefinition: "Publication and introduction of a Bill per NA Rule 279 and section 73 of the Constitution. The Bill is gazetted and referred to the relevant portfolio committee.",
		AllowedNext: []StageCode{
			StageCommittee,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 14,
		Sources: []string{
			"https://www.parliament.gov.za/bills-and-laws",
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		},
	},
	{
		Code:               StageCommittee,
		Name:               "Committee",
		Order:              2,
		Country:            "ZA",
		SimpleExplanation:  "The relevant portfolio or standing committee scrutinises the Bill clause-by-clause, takes written and oral submissions, and proposes amendments.",
		OfficialDefinition: "Committee consideration per NA Rules 284–292. The committee may invite public submissions and must report back to the House.",
		AllowedNext: []StageCode{
			StagePublicParticipation,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 60,
		Sources: []string{
			"https://www.parliament.gov.za/parliamentary-committees",
		},
	},
	{
		Code:               StagePublicParticipation,
		Name:               "Public Participation",
		Order:              3,
		Country:            "ZA",
		SimpleExplanation:  "Members of the public are invited to comment on the Bill, per sections 59 and 72 of the Constitution.",
		OfficialDefinition: "Sections 59 (NA) and 72 (NCOP) of the Constitution mandate that the National Assembly and the NCOP facilitate public involvement in their legislative and other processes.",
		AllowedNext: []StageCode{
			StageNAVote,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 30,
		Sources: []string{
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
			"https://www.parliament.gov.za/public-participation",
		},
	},
	{
		Code:               StageNAVote,
		Name:               "NA Vote",
		Order:              4,
		Country:            "ZA",
		SimpleExplanation:  "The National Assembly debates and votes on the Bill (with any committee amendments). A majority of members is required to pass.",
		OfficialDefinition: "Voting on a Bill in the National Assembly per NA Rule 102 and section 53 of the Constitution. Most Bills pass by a simple majority of the members present; constitutional amendments require a two-thirds majority.",
		AllowedNext: []StageCode{
			StageNCOPConcurrence,
			StageMediation,
			StageRejected,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        true,
		TypicalDurationDays: 14,
		Sources: []string{
			"https://www.parliament.gov.za/",
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		},
	},
	{
		Code:               StageNCOPConcurrence,
		Name:               "NCOP Concurrence",
		Order:              5,
		Country:            "ZA",
		SimpleExplanation:  "The National Council of Provinces considers the Bill. For Section 76 Bills, each provincial delegation votes on its provincial mandate.",
		OfficialDefinition: "NCOP consideration per Chapter 4 of the Constitution. Section 75 Bills (not affecting provinces) are voted on by the NCOP as a whole; Section 76 Bills (affecting provinces) require each provincial delegation to vote in accordance with its provincial mandate.",
		AllowedNext: []StageCode{
			StagePresidentialAssent,
			StageMediation,
			StageRejected,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        true,
		TypicalDurationDays: 30,
		Sources: []string{
			"https://www.parliament.gov.za/ncop",
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		},
	},
	{
		Code:               StageMediation,
		Name:               "Mediation Committee",
		Order:              6,
		Country:            "ZA",
		SimpleExplanation:  "When the NA and NCOP disagree on a Section 76 Bill, a Mediation Committee is formed to agree a compromise text.",
		OfficialDefinition: "Mediation between the two houses per section 76 of the Constitution. A Mediation Committee (composed of NA and NCOP members) prepares a version of the Bill for both houses to consider.",
		AllowedNext: []StageCode{
			StagePresidentialAssent,
			StageRejected,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        true,
		TypicalDurationDays: 30,
		Sources: []string{
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
			"https://www.parliament.gov.za/",
		},
	},
	{
		Code:               StagePresidentialAssent,
		Name:               "Presidential Assent",
		Order:              7,
		Country:            "ZA",
		SimpleExplanation:  "The President signs the Bill into law per section 79 of the Constitution. May be referred back to the NA once for reconsideration.",
		OfficialDefinition: "Assent to a Bill by the President, per section 79 of the Constitution. The President may refer a Bill back to the NA for reconsideration on constitutional grounds; if returned and re-passed, assent is mandatory (subject to referral to the Constitutional Court).",
		AllowedNext: []StageCode{
			StageCommencement,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 14,
		Sources: []string{
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
			"https://www.parliament.gov.za/",
			"https://www.gov.za/",
		},
	},
	{
		Code:               StageCommencement,
		Name:               "Commencement",
		Order:              8,
		Country:            "ZA",
		SimpleExplanation:  "The Act is brought into force, either on a date fixed by the Act itself or by a commencement proclamation in the Government Gazette.",
		OfficialDefinition: "Coming into operation of an Act of Parliament, per the Interpretation Act 33 of 1957 and the Act itself. Where no date is fixed, commencement is by Presidential proclamation in the Government Gazette.",
		AllowedNext:        nil, // terminal-success
		RequiresEvidence:   true,
		RequiresVote:       false,
		TypicalDurationDays: 90,
		Sources: []string{
			"https://www.gov.za/documents",
			"https://www.parliament.gov.za/",
		},
	},
	{
		Code:               StageRejected,
		Name:               "Rejected",
		Order:              100,
		Country:            "ZA",
		SimpleExplanation:  "The Bill failed to pass a vote in the NA or NCOP, or failed to be agreed by the Mediation Committee.",
		OfficialDefinition: "Terminal state: the Bill has failed a required vote and cannot proceed in this Parliament.",
		AllowedNext:        nil, // terminal
		RequiresEvidence:   true,
		RequiresVote:       true,
		TypicalDurationDays: 0,
		Sources: []string{
			"https://www.parliament.gov.za/",
		},
	},
	{
		Code:               StageWithdrawn,
		Name:               "Withdrawn",
		Order:              101,
		Country:            "ZA",
		SimpleExplanation:  "The member or Minister who introduced the Bill has withdrawn it, per the NA Rules.",
		OfficialDefinition: "Terminal state: a Bill withdrawn by the member or Minister in charge, per NA Rule 333 (and corresponding NCOP Rule).",
		AllowedNext:        nil, // terminal
		RequiresEvidence:   true,
		RequiresVote:       false,
		TypicalDurationDays: 0,
		Sources: []string{
			"https://www.parliament.gov.za/",
		},
	},
	{
		Code:               StageLapsed,
		Name:               "Lapsed",
		Order:              102,
		Country:            "ZA",
		SimpleExplanation:  "The Bill has lapsed at the end of a Parliament without concluding its passage.",
		OfficialDefinition: "Terminal state: a Bill lapses at the dissolution of Parliament. A lapsed Bill may be revived in the next Parliament per NA Rule 333(2).",
		AllowedNext:        nil, // terminal
		RequiresEvidence:   true,
		RequiresVote:       false,
		TypicalDurationDays: 0,
		Sources: []string{
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		},
	},
}

// FindStage looks up a SouthAfricaStage by its code. Returns nil if not found.
func FindStage(code StageCode) *SouthAfricaStage {
	for i := range SouthAfricaBillStages {
		if SouthAfricaBillStages[i].Code == code {
			return &SouthAfricaBillStages[i]
		}
	}
	return nil
}

// IsTerminal reports whether the given stage has no allowed transitions
// (i.e. is a terminal state of the Bill lifecycle).
func IsTerminal(code StageCode) bool {
	s := FindStage(code)
	if s == nil {
		return false
	}
	return len(s.AllowedNext) == 0
}

// CanTransition reports whether moving from `from` to `to` is permitted by the
// South African Bill lifecycle model encoded in SouthAfricaBillStages.
func CanTransition(from, to StageCode) bool {
	s := FindStage(from)
	if s == nil {
		return false
	}
	for _, n := range s.AllowedNext {
		if n == to {
			return true
		}
	}
	return false
}

// ToContract projects the richer SouthAfricaStage down to the platform's
// country-agnostic contracts.StageDefinition. No South Africa-specific string
// other than the stage Code and Name crosses this boundary.
func (s SouthAfricaStage) ToContract() contracts.StageDefinition {
	next := make([]string, 0, len(s.AllowedNext))
	for _, n := range s.AllowedNext {
		next = append(next, string(n))
	}
	return contracts.StageDefinition{
		Code:                string(s.Code),
		Name:                s.Name,
		Description:         s.SimpleExplanation,
		SimpleExplanation:    s.SimpleExplanation,
		OfficialDefinition:  s.OfficialDefinition,
		Country:             contracts.Country(s.Country),
		AllowedNext:         next,
		AllowedTransitions:  next,
		Order:               s.Order,
		IsTerminal:          IsTerminal(s.Code),
		RequiresEvidence:    s.RequiresEvidence,
		RequiresVote:        s.RequiresVote,
		TypicalDurationDays: s.TypicalDurationDays,
	}
}

// ---------------------------------------------------------------------------
// Terminology
// ---------------------------------------------------------------------------

// SouthAfricaTerm is the richer, South Africa-specific glossary entry. It
// carries country, sources (URLs to official South African documents) and an
// optional stage association. The contracts.TermDefinition type only carries a
// canonical key, term and short description — adapter.go projects the richer
// record down to that shape.
type SouthAfricaTerm struct {
	Term               string
	CanonicalKey       string
	SimpleExplanation  string
	OfficialDefinition string
	Stage              string // optional stage code, "" if general
	Country            string // always "ZA"
	Sources            []string
}

// SouthAfricaTerminology is the registry of South African parliamentary and
// legal terms used by the adapter when normalising extracted records.
//
// Sources used to compile these definitions:
//   - Constitution of the Republic of South Africa, 1996
//     https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996
//   - Parliament of South Africa — Glossary and Rules
//     https://www.parliament.gov.za/
//   - Government of South Africa — Government Gazette
//     https://www.gov.za/documents
var SouthAfricaTerminology = []SouthAfricaTerm{
	{
		Term:              "Introduction",
		CanonicalKey:      "introduction",
		SimpleExplanation: "The formal introduction (first reading) of a Bill in the National Assembly. The Bill is gazetted and referred to a committee.",
		OfficialDefinition: "NA Rule 279 and section 73 of the Constitution. The Bill is published in the Government Gazette before introduction.",
		Stage:   string(StageIntroduction),
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/bills-and-laws"},
	},
	{
		Term:              "Portfolio Committee",
		CanonicalKey:      "portfolio_committee",
		SimpleExplanation: "A National Assembly committee that shadow a government department and scrutinise Bills affecting that department's portfolio.",
		OfficialDefinition: "NA Chapter 12. Each portfolio committee is named for its department (e.g. Portfolio Committee on Finance) and consists of NA members proportionate to party representation.",
		Stage:   string(StageCommittee),
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/parliamentary-committees"},
	},
	{
		Term:              "Select Committee",
		CanonicalKey:      "select_committee",
		SimpleExplanation: "An NCOP committee that shadow a government department and scrutinise Bills in the NCOP.",
		OfficialDefinition: "NCOP Rules Chapter 12. Select committees are the NCOP's equivalent of the NA's portfolio committees.",
		Stage:   string(StageCommittee),
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/parliamentary-committees"},
	},
	{
		Term:              "Public Participation",
		CanonicalKey:      "public_participation",
		SimpleExplanation: "Constitutionally-mandated process where the public submits views on Bills and other parliamentary business.",
		OfficialDefinition: "Sections 59 (NA) and 72 (NCOP) of the Constitution mandate Parliament to facilitate public involvement in its legislative and other processes.",
		Stage:   string(StagePublicParticipation),
		Country: "ZA",
		Sources: []string{
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
			"https://www.parliament.gov.za/public-participation",
		},
	},
	{
		Term:              "NA Vote",
		CanonicalKey:      "na_vote",
		SimpleExplanation: "The vote on the Bill in the National Assembly after the committee has reported.",
		OfficialDefinition: "Voting in the NA per NA Rule 102 and section 53 of the Constitution. Most Bills pass by simple majority of members present and voting.",
		Stage:   string(StageNAVote),
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/"},
	},
	{
		Term:              "NCOP Concurrence",
		CanonicalKey:      "ncop_concurrence",
		SimpleExplanation: "The vote by the National Council of Provinces on a Bill passed by the National Assembly.",
		OfficialDefinition: "Chapter 4 of the Constitution. Section 75 Bills (not affecting provinces) are voted on by the NCOP as a whole; Section 76 Bills (affecting provinces) require each provincial delegation to vote in accordance with its provincial mandate.",
		Stage:   string(StageNCOPConcurrence),
		Country: "ZA",
		Sources: []string{
			"https://www.parliament.gov.za/ncop",
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		},
	},
	{
		Term:              "Section 75 Bill",
		CanonicalKey:      "section_75_bill",
		SimpleExplanation: "A Bill that does not affect the provinces. The NCOP may pass, reject, or amend within 30 days; the NA decides the final text.",
		OfficialDefinition: "Section 75 of the Constitution. The NCOP's role is to consider, and may pass/reject/amend, but the NA has the final say.",
		Country: "ZA",
		Sources: []string{"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996"},
	},
	{
		Term:              "Section 76 Bill",
		CanonicalKey:      "section_76_bill",
		SimpleExplanation: "A Bill that affects the provinces. The NCOP must vote on it, with each provincial delegation voting on its provincial mandate.",
		OfficialDefinition: "Section 76 of the Constitution. Each provincial delegation in the NCOP has one vote, exercised in accordance with the province's mandate.",
		Country: "ZA",
		Sources: []string{"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996"},
	},
	{
		Term:              "Mediation Committee",
		CanonicalKey:      "mediation_committee",
		SimpleExplanation: "A joint committee of the NA and NCOP convened when the two houses disagree on a Section 76 Bill.",
		OfficialDefinition: "Section 76 of the Constitution. The Mediation Committee prepares a compromise version for both houses to consider.",
		Stage:   string(StageMediation),
		Country: "ZA",
		Sources: []string{"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996"},
	},
	{
		Term:              "Presidential Assent",
		CanonicalKey:      "presidential_assent",
		SimpleExplanation: "The President signs a Bill passed by Parliament into law.",
		OfficialDefinition: "Section 79 of the Constitution. The President must either assent to and sign the Bill, or refer it back to the NA for reconsideration on constitutional grounds.",
		Stage:   string(StagePresidentialAssent),
		Country: "ZA",
		Sources: []string{
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
			"https://www.parliament.gov.za/",
		},
	},
	{
		Term:              "Commencement",
		CanonicalKey:      "commencement",
		SimpleExplanation: "The date an Act comes into force, fixed by the Act or by a Presidential proclamation in the Government Gazette.",
		OfficialDefinition: "Per the Interpretation Act 33 of 1957 and the Act's own commencement provision.",
		Stage:   string(StageCommencement),
		Country: "ZA",
		Sources: []string{
			"https://www.gov.za/documents",
			"https://www.parliament.gov.za/",
		},
	},
	{
		Term:              "National Assembly",
		CanonicalKey:      "national_assembly",
		SimpleExplanation: "The lower house of South Africa's Parliament, with 400 members elected by proportional representation.",
		OfficialDefinition: "Section 42(3) of the Constitution. The National Assembly consists of no fewer than 350 and no more than 400 women and men elected as members in terms of an electoral system.",
		Country: "ZA",
		Sources: []string{
			"https://www.parliament.gov.za/national-assembly",
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		},
	},
	{
		Term:              "National Council of Provinces",
		CanonicalKey:      "national_council_of_provinces",
		SimpleExplanation: "The upper house of South Africa's Parliament, with 90 members (10 per province × 9 provinces) representing the provinces.",
		OfficialDefinition: "Section 42(4) and section 60 of the Constitution. The NCOP consists of a single delegation from each province (10 members each), totalling 90 members.",
		Country: "ZA",
		Sources: []string{
			"https://www.parliament.gov.za/ncop",
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		},
	},
	{
		Term:              "Bicameral",
		CanonicalKey:      "bicameral",
		SimpleExplanation: "A legislature with two houses; South Africa's Parliament has the National Assembly (lower) and the National Council of Provinces (upper).",
		OfficialDefinition: "Section 42(1) of the Constitution establishes Parliament as consisting of the National Assembly and the National Council of Provinces.",
		Country: "ZA",
		Sources: []string{"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996"},
	},
	{
		Term:              "Hansard",
		CanonicalKey:      "hansard",
		SimpleExplanation: "The official verbatim report of debates in the National Assembly and the NCOP.",
		OfficialDefinition: "The official report of parliamentary debates, prepared by the Hansard department of each house.",
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/hansard"},
	},
	{
		Term:              "Order Paper",
		CanonicalKey:      "order_paper",
		SimpleExplanation: "The published agenda of business for a sitting of the National Assembly or NCOP.",
		OfficialDefinition: "The Order Paper sets out the business of a sitting; it is published by the Clerk before each sitting.",
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/order-paper"},
	},
	{
		Term:              "Questions and Replies",
		CanonicalKey:      "questions_and_replies",
		SimpleExplanation: "The published questions put by members to Ministers (and the written replies).",
		OfficialDefinition: "NA Rules Chapter 17. Members may put questions for oral or written reply to Cabinet Ministers, Deputy Ministers and the President.",
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/questions-and-replies"},
	},
	{
		Term:              "Government Bill",
		CanonicalKey:      "government_bill",
		SimpleExplanation: "A Bill introduced by a Cabinet Minister (or Deputy Minister) on behalf of the Executive.",
		OfficialDefinition: "A Bill sponsored by the Government (the Executive), as distinguished from a Private Member's Bill or a Committee Bill.",
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/bills-and-laws"},
	},
	{
		Term:              "Private Member's Bill",
		CanonicalKey:      "private_members_bill",
		SimpleExplanation: "A Bill introduced by an individual member of Parliament rather than the Government.",
		OfficialDefinition: "Per section 73 of the Constitution and NA Rule 276. Only Cabinet members may introduce Money Bills; other members may introduce Bills with the prior approval of the Speaker and after compliance with the NA Rules and Joint Rules.",
		Country: "ZA",
		Sources: []string{
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
			"https://www.parliament.gov.za/bills-and-laws",
		},
	},
	{
		Term:              "Money Bill",
		CanonicalKey:      "money_bill",
		SimpleExplanation: "A Bill dealing with taxation, public debt, or appropriation of public funds. May only be introduced by a Cabinet member.",
		OfficialDefinition: "Section 77 of the Constitution. A Money Bill may only be introduced by a Cabinet member. The Money Bills Amendment Procedure and Related Matters Act 9 of 2009 governs the procedure for amending Money Bills.",
		Country: "ZA",
		Sources: []string{
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
			"https://www.parliament.gov.za/bills-and-laws",
		},
	},
	{
		Term:              "Speaker of the National Assembly",
		CanonicalKey:      "speaker_of_the_national_assembly",
		SimpleExplanation: "The presiding officer of the National Assembly, elected under section 52 of the Constitution.",
		OfficialDefinition: "Section 52 of the Constitution. The Speaker is the presiding officer of the National Assembly.",
		Country: "ZA",
		Sources: []string{
			"https://www.parliament.gov.za/national-assembly",
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		},
	},
	{
		Term:              "Chairperson of the NCOP",
		CanonicalKey:      "chairperson_of_the_ncop",
		SimpleExplanation: "The presiding officer of the National Council of Provinces, elected under section 64 of the Constitution.",
		OfficialDefinition: "Section 64 of the Constitution. The Chairperson of the NCOP is elected by the Council from among its permanent delegates.",
		Country: "ZA",
		Sources: []string{
			"https://www.parliament.gov.za/ncop",
			"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		},
	},
	{
		Term:              "Secretary to Parliament",
		CanonicalKey:      "secretary_to_parliament",
		SimpleExplanation: "The chief administrative officer of Parliament, established under section 13 of the Constitution.",
		OfficialDefinition: "Section 13 of the Constitution. The Secretary to Parliament is appointed by the Houses and is the head of the administration of Parliament.",
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/"},
	},
	{
		Term:              "Government Gazette",
		CanonicalKey:      "government_gazette",
		SimpleExplanation: "The official publication of the Government of South Africa. Bills, Acts and commencement notices are published here.",
		OfficialDefinition: "A notice published in the Government Gazette. Government Gazettes are published by the Government Printing Works.",
		Country: "ZA",
		Sources: []string{"https://www.gov.za/documents"},
	},
	{
		Term:              "Provincial Mandate",
		CanonicalKey:      "provincial_mandate",
		SimpleExplanation: "The instruction given by a Provincial Legislature to its NCOP delegation on how to vote on a Section 76 Bill.",
		OfficialDefinition: "Section 76 of the Constitution. Each provincial delegation in the NCOP votes in accordance with the mandate of its Provincial Legislature.",
		Country: "ZA",
		Sources: []string{"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996"},
	},
	{
		Term:              "Quorum",
		CanonicalKey:      "quorum",
		SimpleExplanation: "The minimum number of members required for a house to conduct business. The NA requires one-third of its members; the NCOP requires one-third of its permanent delegates.",
		OfficialDefinition: "Section 53(2) of the Constitution. The NA requires a quorum of one-third of its members; the NCOP requires a quorum of one-third of its permanent delegates.",
		Country: "ZA",
		Sources: []string{"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996"},
	},
	{
		Term:              "Division",
		CanonicalKey:      "division",
		SimpleExplanation: "A formal recorded vote where members' names are entered in the Record of Proceedings.",
		OfficialDefinition: "NA Rule 102. A division is the procedure by which the House votes with the ayes and noes recorded individually; results are entered in the Record of Proceedings.",
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/"},
	},
	{
		Term:              "Whip",
		CanonicalKey:      "whip",
		SimpleExplanation: "A member responsible for party discipline, attendance, and the management of business in the house.",
		OfficialDefinition: "Each party with seats in a house designates whips. The Chief Whips of the majority party and the largest opposition party are members of the Chief Whips' Forum, which programmes house business.",
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/"},
	},
	{
		Term:              "Memorandum on the Objects of the Bill",
		CanonicalKey:      "memorandum_on_the_objects_of_the_bill",
		SimpleExplanation: "The explanatory statement accompanying a Bill setting out its policy, objects, and reasons for each clause.",
		OfficialDefinition: "NA Rule 279. A Bill must be accompanied by a Memorandum on its Objects when published in the Government Gazette.",
		Country: "ZA",
		Sources: []string{"https://www.parliament.gov.za/bills-and-laws"},
	},
	{
		Term:              "Cabinet",
		CanonicalKey:      "cabinet",
		SimpleExplanation: "The President, the Deputy President and the Ministers, collectively responsible to Parliament per section 92 of the Constitution.",
		OfficialDefinition: "Section 91 and 92 of the Constitution. The Cabinet consists of the President, the Deputy President and Ministers, and is collectively accountable to Parliament.",
		Country: "ZA",
		Sources: []string{"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996"},
	},
	{
		Term: "Constitutional Court",
		CanonicalKey: "constitutional_court",
		SimpleExplanation: "The highest court in South Africa for constitutional matters; it may be called upon to certify the constitutionality of a Bill.",
		OfficialDefinition: "Section 167 of the Constitution. The Constitutional Court is the highest court for constitutional matters.",
		Country: "ZA",
		Sources: []string{"https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996"},
	},
}

// FindTerm looks up a SouthAfricaTerm by its canonical key. Returns nil if not
// found.
func FindTerm(key string) *SouthAfricaTerm {
	for i := range SouthAfricaTerminology {
		if SouthAfricaTerminology[i].CanonicalKey == key {
			return &SouthAfricaTerminology[i]
		}
	}
	return nil
}

// ToContract projects the richer SouthAfricaTerm down to the platform's
// country-agnostic contracts.TermDefinition. The canonical key is the only
// string that links the platform back to this richer record.
func (t SouthAfricaTerm) ToContract() contracts.TermDefinition {
	return contracts.TermDefinition{
		Term:               t.Term,
		CanonicalKey:       t.CanonicalKey,
		SimpleExplanation:  t.SimpleExplanation,
		Description:        t.SimpleExplanation,
		OfficialDefinition: t.OfficialDefinition,
		StageCode:          t.Stage,
		Country:            contracts.Country(t.Country),
		Sources:            t.Sources,
	}
}

// ---------------------------------------------------------------------------
// Legislative structure
// ---------------------------------------------------------------------------

// SouthAfricaLegislativeStructure returns the structural description of
// South Africa's Parliament as established by the 1996 Constitution: a
// bicameral legislature comprising the National Assembly (lower house, 400
// members) and the National Council of Provinces (upper house, 90 members —
// 10 per province × 9 provinces).
//
// Sources:
//   - Constitution of the Republic of South Africa, 1996, Chapter 4 (Parliament)
//     https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996
//   - Parliament of South Africa
//     https://www.parliament.gov.za/
//   - National Assembly
//     https://www.parliament.gov.za/national-assembly
//   - National Council of Provinces
//     https://www.parliament.gov.za/ncop
func SouthAfricaLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     contracts.Country("ZA"),
		CountryCode: "ZA",
		CountryName: "South Africa",
		Houses: []contracts.HouseDefinition{
			{
				Code:     HouseCodeNationalAssembly,
				Name:     "National Assembly",
				Type:     contracts.HouseTypeLower,
				Members:  400, // per section 46 of the Constitution
				TermDays: 5 * 365,
			},
			{
				Code:     HouseCodeNationalCouncilOfProvinces,
				Name:     "National Council of Provinces",
				Type:     contracts.HouseTypeUpper,
				Members:  90, // 10 per province × 9 provinces per section 60 of the Constitution
				TermDays: 5 * 365,
			},
		},
		Committees: southAfricaStandingCommittees(),
	}
}

// southAfricaStandingCommittees returns a representative selection of the
// standing committees published by the National Assembly and the NCOP.
//
// Committee definitions are derived from the published Committee List:
//   - National Assembly portfolio committees
//     https://www.parliament.gov.za/parliamentary-committees
//   - NCOP select committees
//     https://www.parliament.gov.za/parliamentary-committees
func southAfricaStandingCommittees() []contracts.CommitteeDefinition {
	return []contracts.CommitteeDefinition{
		// National Assembly portfolio committees.
		{Code: "PC_FIN", Name: "Portfolio Committee on Finance", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_HLT", Name: "Portfolio Committee on Health", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_HED", Name: "Portfolio Committee on Higher Education", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_BED", Name: "Portfolio Committee on Basic Education", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_JUS", Name: "Portfolio Committee on Justice and Correctional Services", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_TIC", Name: "Portfolio Committee on Trade, Industry and Competition", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_AGR", Name: "Portfolio Committee on Agriculture, Land Reform and Rural Development", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_DEF", Name: "Portfolio Committee on Defence and Military Veterans", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_MRE", Name: "Portfolio Committee on Mineral Resources and Energy", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_CDT", Name: "Portfolio Committee on Communications and Digital Technologies", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_TRA", Name: "Portfolio Committee on Transport", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},
		{Code: "PC_HWS", Name: "Portfolio Committee on Human Settlements, Water and Sanitation", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},

		// National Assembly sessional / standing committees.
		{Code: "SCOPA", Name: "Standing Committee on Public Accounts (SCOPA)", House: HouseCodeNationalAssembly, Type: "standing", Members: 15},
		{Code: "SCFIN", Name: "Standing Committee on Finance", House: HouseCodeNationalAssembly, Type: "standing", Members: 11},
		{Code: "PC_PWI", Name: "Portfolio Committee on Public Works and Infrastructure", House: HouseCodeNationalAssembly, Type: "portfolio", Members: 11},

		// NCOP select committees.
		{Code: "SEL_FIN", Name: "Select Committee on Finance", House: HouseCodeNationalCouncilOfProvinces, Type: "select", Members: 9},
		{Code: "SEL_HSS", Name: "Select Committee on Health and Social Services", House: HouseCodeNationalCouncilOfProvinces, Type: "select", Members: 9},
		{Code: "SEL_EDU", Name: "Select Committee on Education and Technology, Sports, Arts and Culture", House: HouseCodeNationalCouncilOfProvinces, Type: "select", Members: 9},
		{Code: "SEL_SJ", Name: "Select Committee on Security and Justice", House: HouseCodeNationalCouncilOfProvinces, Type: "select", Members: 9},
		{Code: "SEL_TIE", Name: "Select Committee on Trade and Industry, Economic Development, Small Business Development, Tourism, Employment and Labour", House: HouseCodeNationalCouncilOfProvinces, Type: "select", Members: 9},
		{Code: "SEL_LRE", Name: "Select Committee on Land Reform, Environment, Mineral Resources and Energy", House: HouseCodeNationalCouncilOfProvinces, Type: "select", Members: 9},
		{Code: "SEL_CTA", Name: "Select Committee on Cooperative Governance and Traditional Affairs, Water and Sanitation and Human Settlements", House: HouseCodeNationalCouncilOfProvinces, Type: "select", Members: 9},
		{Code: "SEL_TPS", Name: "Select Committee on Transport, Public Service and Administration, Public Works and Infrastructure", House: HouseCodeNationalCouncilOfProvinces, Type: "select", Members: 9},
	}
}
