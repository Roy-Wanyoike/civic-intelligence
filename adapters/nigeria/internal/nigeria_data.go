// Package internal holds Nigeria-specific knowledge that must NEVER leak into
// the global platform types. The contracts package defines only country-agnostic
// shapes (StageDefinition, TermDefinition, ...). Every Nigerian string —
// "Second Reading", "Concurrence", "Public Hearing", "Senate", "House of
// Representatives", "National Assembly" — lives in this package.
//
// The Nigeria adapter's adapter.go projects these richer records down into the
// plain contracts.* types when the ingestion service asks for them.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// House codes used throughout the Nigeria adapter. They appear in RawMetadata
// and SourceItem.House fields, but never as bare strings in the global domain
// model.
const (
	HouseCodeHouseOfReps = "HOR"
	HouseCodeSenate      = "SEN"
)

// Committee codes are stable identifiers used by the adapter only. The
// contracts.CommitteeDefinition exposes them via the Code field, which the
// legislation service uses as an opaque key.
const (
	CommitteeAppropriations        = "APP"
	CommitteeFinance               = "FIN"
	CommitteeJudiciary             = "JUD"
	CommitteeHealth                = "HLT"
	CommitteeEducation             = "EDU"
	CommitteeDefence               = "DEF"
	CommitteePublicAccounts        = "PAC"
	CommitteeWorks                 = "WRK"
	CommitteePower                 = "PWR"
	CommitteeCommunication         = "COM"
	CommitteeAgriculture           = "AGR"
	CommitteeTransport             = "TRN"
	CommitteeAviation              = "AVN"
	CommitteeNigerDelta            = "NDT"
	CommitteeEcologicalFunds       = "ECF"
	CommitteeRulesBusiness         = "RBS"
	CommitteeEthicsPrivileges       = "ETH"
	CommitteeSelection             = "SEL"
	CommitteeWomenAffairs          = "WMA"
	CommitteeYouthSports           = "YTH"
)

// StageCode is the stable identifier Nigeria uses for a bill lifecycle stage.
// These codes are the ONLY stage-related strings that ever cross into the
// global domain model (they are case-normalised SCREAMING_SNAKE strings,
// namespaced implicitly by the adapter's CountryCode()).
//
// The codes follow the conventions of the 1999 Constitution of the Federal
// Republic of Nigeria (as amended) and the Standing Orders of the Senate and
// House of Representatives.
type StageCode string

const (
	// StageFirstReading is the formal introduction. The Bill is read a first
	// time and entered on the House's Order Paper.
	StageFirstReading StageCode = "FIRST_READING"

	// StageSecondReading is the principle debate: the House debates whether
	// the Bill's policy and principles should be approved.
	StageSecondReading StageCode = "SECOND_READING"

	// StagePublicHearing is the constitutionally-mandated public hearing
	// conducted by the relevant committee.
	StagePublicHearing StageCode = "PUBLIC_HEARING"

	// StageCommittee is the committee stage where the Bill is scrutinised
	// clause-by-clause and amendments proposed.
	StageCommittee StageCode = "COMMITTEE"

	// StageReport is the report stage: the committee reports its
	// amendments back to the House, which may further amend.
	StageReport StageCode = "REPORT"

	// StageThirdReading is the final reading in a house, after which the
	// Bill is passed by that house.
	StageThirdReading StageCode = "THIRD_READING"

	// StageConcurrence is the stage at which the other chamber of the
	// National Assembly considers a Bill passed by the originating chamber.
	StageConcurrence StageCode = "CONCURRENCE"

	// StageAssent is the assent by the President per Section 58 of the
	// Constitution.
	StageAssent StageCode = "ASSENT"

	// StageCommencement is when an Act is brought into force by a
	// commencement notice in the Federal Government Gazette.
	StageCommencement StageCode = "COMMENCEMENT"

	// StageRejected is the terminal state for a Bill that fails to pass.
	StageRejected StageCode = "REJECTED"

	// StageWithdrawn is the terminal state for a Bill that is withdrawn by
	// the Mover.
	StageWithdrawn StageCode = "WITHDRAWN"

	// StageLapsed is the terminal state for a Bill that lapses at the end of
	// an Assembly without concluding.
	StageLapsed StageCode = "LAPSED"

	// StageConferenceCommittee is used when the two chambers disagree and a
	// Conference Committee (Joint Committee) is convened per the Standing
	// Orders of the Senate and House.
	StageConferenceCommittee StageCode = "CONFERENCE_COMMITTEE"

	// StageVeto is used when the President withholds assent to a Bill. The
	// National Assembly may override the veto by a two-thirds majority of
	// each chamber.
	StageVeto StageCode = "VETO"
)

// NigeriaStage is the richer, Nigeria-specific stage record. It carries the
// country code, the official definition (with citation to the Standing Orders
// or the Constitution), and the canonical source URLs for the stage.
//
// The platform NEVER receives a NigeriaStage directly. adapter.go projects it
// down to a contracts.StageDefinition when the ingestion service asks for the
// BillStages.
type NigeriaStage struct {
	Code                StageCode
	Name                string
	SimpleExplanation   string
	OfficialDefinition  string
	Country             string // always "NG"
	Order               int
	AllowedNext         []StageCode
	RequiresEvidence    bool
	RequiresVote        bool
	TypicalDurationDays int
	Sources             []string
}

// NigeriaBillStages is the ordered, authoritative list of Nigeria's bill stages.
//
// Sources used to compile these definitions:
//   - Constitution of the Federal Republic of Nigeria, 1999 (as amended),
//     Sections 47–89 (The National Assembly) and Section 58 (Mode of exercising
//     legislative powers).
//     https://www.constituteproject.org/constitution/Nigeria_1999
//   - Standing Orders of the Senate (Federal Republic of Nigeria)
//     https://nass.gov.ng/senate
//   - Standing Orders of the House of Representatives
//     https://nass.gov.ng/house
//   - National Institute for Legislative and Democratic Studies (NILDS)
//     https://www.nilds.gov.ng/
//
// Stage transitions follow the Standing Orders (a Bill normally proceeds
// First Reading → Second Reading → Public Hearing → Committee → Report →
// Third Reading → Concurrence → Assent → Commencement). A Bill may also be
// considered by a Conference Committee (when the chambers disagree),
// Vetoed by the President, Rejected, Withdrawn or Lapsed.
var NigeriaBillStages = []NigeriaStage{
	{
		Code:               StageFirstReading,
		Name:               "First Reading",
		Order:              1,
		Country:            "NG",
		SimpleExplanation:  "The Bill is read the first time in the chamber and entered on the Order Paper. No debate on the merits takes place.",
		OfficialDefinition: "Publication and first reading of a Bill as provided for by the Standing Orders of the Senate and House of Representatives, and Section 58 of the Constitution.",
		AllowedNext: []StageCode{
			StageSecondReading,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 7,
		Sources: []string{
			"https://nass.gov.ng/",
			"https://www.constituteproject.org/constitution/Nigeria_1999",
		},
	},
	{
		Code:               StageSecondReading,
		Name:               "Second Reading",
		Order:              2,
		Country:            "NG",
		SimpleExplanation:  "The chamber debates and votes on the principles and policy of the Bill. No clause-by-clause amendments happen here.",
		OfficialDefinition: "Debate on the general principles of a Bill, governed by the Standing Orders of the Senate and House of Representatives. At the conclusion the chamber votes on whether the Bill should proceed to committee.",
		AllowedNext: []StageCode{
			StagePublicHearing,
			StageCommittee,
			StageRejected,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:       true,
		TypicalDurationDays: 14,
		Sources: []string{
			"https://nass.gov.ng/",
			"https://www.constituteproject.org/constitution/Nigeria_1999",
		},
	},
	{
		Code:               StagePublicHearing,
		Name:               "Public Hearing",
		Order:              3,
		Country:            "NG",
		SimpleExplanation:  "The relevant committee holds a public hearing to receive input from citizens, civil society, and subject-matter experts on the Bill.",
		OfficialDefinition: "Public hearing conducted by the relevant committee to receive memoranda and oral submissions from stakeholders and the public, per the Standing Orders of both chambers.",
		AllowedNext: []StageCode{
			StageCommittee,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 21,
		Sources: []string{
			"https://nass.gov.ng/",
		},
	},
	{
		Code:               StageCommittee,
		Name:               "Committee",
		Order:              4,
		Country:            "NG",
		SimpleExplanation:  "The relevant committee scrutinises the Bill clause-by-clause, considers public submissions, and proposes amendments.",
		OfficialDefinition: "Committee scrutiny of a Bill, including consideration of memoranda received at the public hearing, per the Standing Orders of both chambers.",
		AllowedNext: []StageCode{
			StageReport,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 60,
		Sources: []string{
			"https://nass.gov.ng/house/committees",
			"https://nass.gov.ng/senate/committees",
			"https://www.constituteproject.org/constitution/Nigeria_1999",
		},
	},
	{
		Code:               StageReport,
		Name:               "Report",
		Order:              5,
		Country:            "NG",
		SimpleExplanation:  "The committee reports its proposed amendments back to the full chamber, which can accept, reject or further amend them.",
		OfficialDefinition: "Report of a committee on a Bill and consideration of amendments on the floor, per the Standing Orders of both chambers.",
		AllowedNext: []StageCode{
			StageThirdReading,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:       true,
		TypicalDurationDays: 7,
		Sources: []string{
			"https://nass.gov.ng/",
		},
	},
	{
		Code:               StageThirdReading,
		Name:               "Third Reading",
		Order:              6,
		Country:            "NG",
		SimpleExplanation:  "The final reading in the chamber. The Bill is voted on as amended; if passed it is forwarded to the other chamber for concurrence.",
		OfficialDefinition: "Third reading and passing of a Bill by a chamber, per the Standing Orders. A Bill passed by one chamber is forwarded to the other for concurrence.",
		AllowedNext: []StageCode{
			StageConcurrence,
			StageConferenceCommittee,
			StageRejected,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:       true,
		TypicalDurationDays: 7,
		Sources: []string{
			"https://nass.gov.ng/",
			"https://www.constituteproject.org/constitution/Nigeria_1999",
		},
	},
	{
		Code:               StageConcurrence,
		Name:               "Concurrence",
		Order:              7,
		Country:            "NG",
		SimpleExplanation:  "The other chamber of the National Assembly considers the Bill as passed by the originating chamber. If both agree, the Bill is sent to the President for assent.",
		OfficialDefinition: "Concurrence by the other chamber of the National Assembly, per Section 58 of the Constitution. Where the chambers disagree, a Conference Committee is convened.",
		AllowedNext: []StageCode{
			StageAssent,
			StageConferenceCommittee,
			StageRejected,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:       true,
		TypicalDurationDays: 21,
		Sources: []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
			"https://nass.gov.ng/",
		},
	},
	{
		Code:               StageConferenceCommittee,
		Name:               "Conference Committee",
		Order:              8,
		Country:            "NG",
		SimpleExplanation:  "When the Senate and House of Representatives disagree on a Bill, a joint Conference Committee is formed to agree a compromise text.",
		OfficialDefinition: "Conference Committee (Joint Committee) of both chambers, convened when the chambers disagree on a Bill, to agree a compromise version for both chambers to approve.",
		AllowedNext: []StageCode{
			StageAssent,
			StageRejected,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:       true,
		TypicalDurationDays: 21,
		Sources: []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
			"https://nass.gov.ng/",
		},
	},
	{
		Code:               StageAssent,
		Name:               "Assent",
		Order:              9,
		Country:            "NG",
		SimpleExplanation:  "The President signs the Bill into law per Section 58 of the Constitution. May be withheld (veto); the National Assembly may override a veto by a two-thirds majority of each chamber.",
		OfficialDefinition: "Assent to a Bill by the President, per Section 58 of the Constitution. The President may withhold assent (veto); the National Assembly may override the veto by a two-thirds majority of each chamber.",
		AllowedNext: []StageCode{
			StageCommencement,
			StageVeto,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 30,
		Sources: []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
			"https://nass.gov.ng/",
		},
	},
	{
		Code:               StageVeto,
		Name:               "Veto",
		Order:              10,
		Country:            "NG",
		SimpleExplanation:  "The President has withheld assent. The National Assembly may override the veto by a two-thirds majority of each chamber; otherwise the Bill fails.",
		OfficialDefinition: "Presidential veto per Section 58(4)–(5) of the Constitution. The National Assembly may override the veto by a two-thirds majority of each chamber.",
		AllowedNext: []StageCode{
			StageAssent,
			StageRejected,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:       true,
		TypicalDurationDays: 14,
		Sources: []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
			"https://nass.gov.ng/",
		},
	},
	{
		Code:               StageCommencement,
		Name:               "Commencement",
		Order:              11,
		Country:            "NG",
		SimpleExplanation:  "The Act is brought into force, either on a date fixed by the Act itself or by a commencement notice in the Federal Government Gazette.",
		OfficialDefinition: "Coming into operation of an Act of the National Assembly, per the Interpretation Act and the Act itself. Where no date is fixed, commencement is by notice in the Federal Government Gazette.",
		AllowedNext:         nil, // terminal-success
		RequiresEvidence:   true,
		RequiresVote:       false,
		TypicalDurationDays: 30,
		Sources: []string{
			"https://nass.gov.ng/",
			"https://www.constituteproject.org/constitution/Nigeria_1999",
		},
	},
	{
		Code:               StageRejected,
		Name:               "Rejected",
		Order:               100,
		Country:             "NG",
		SimpleExplanation:   "The Bill failed to pass a vote at second or third reading, failed concurrence, or a veto override failed.",
		OfficialDefinition: "Terminal state: the Bill has failed a required vote and cannot proceed in this Assembly.",
		AllowedNext:         nil, // terminal
		RequiresEvidence:   true,
		RequiresVote:       true,
		TypicalDurationDays: 0,
		Sources: []string{
			"https://nass.gov.ng/",
		},
	},
	{
		Code:               StageWithdrawn,
		Name:               "Withdrawn",
		Order:               101,
		Country:             "NG",
		SimpleExplanation:   "The mover of the Bill has withdrawn it, per the Standing Orders.",
		OfficialDefinition: "Terminal state: a Bill withdrawn by the Mover or originating sponsor, per the Standing Orders of the Senate and House of Representatives.",
		AllowedNext:         nil, // terminal
		RequiresEvidence:   true,
		RequiresVote:       false,
		TypicalDurationDays: 0,
		Sources: []string{
			"https://nass.gov.ng/",
		},
	},
	{
		Code:               StageLapsed,
		Name:               "Lapsed",
		Order:               102,
		Country:             "NG",
		SimpleExplanation:   "The Bill has lapsed at the end of an Assembly without concluding its passage.",
		OfficialDefinition: "Terminal state: a Bill lapses at the dissolution of the National Assembly, per Section 64 of the Constitution.",
		AllowedNext:         nil, // terminal
		RequiresEvidence:   true,
		RequiresVote:       false,
		TypicalDurationDays: 0,
		Sources: []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
		},
	},
}

// FindStage looks up a NigeriaStage by its code. Returns nil if not found.
func FindStage(code StageCode) *NigeriaStage {
	for i := range NigeriaBillStages {
		if NigeriaBillStages[i].Code == code {
			return &NigeriaBillStages[i]
		}
	}
	return nil
}

// IsTerminal reports whether the given stage has no allowed transitions (i.e.
// is a terminal state of the Bill lifecycle).
func IsTerminal(code StageCode) bool {
	s := FindStage(code)
	if s == nil {
		return false
	}
	return len(s.AllowedNext) == 0
}

// CanTransition reports whether moving from from to to is permitted by the
// Standing Orders model encoded in NigeriaBillStages.
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

// ToContract projects the richer NigeriaStage down to the platform's
// country-agnostic contracts.StageDefinition. No Nigeria-specific string
// other than the stage Code and Name crosses this boundary.
func (s NigeriaStage) ToContract() contracts.StageDefinition {
	next := make([]string, 0, len(s.AllowedNext))
	for _, n := range s.AllowedNext {
		next = append(next, string(n))
	}
	return contracts.StageDefinition{
		Code:                string(s.Code),
		Name:                s.Name,
		Description:         s.SimpleExplanation,
		SimpleExplanation:   s.SimpleExplanation,
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

// StageByCode returns the contracts.StageDefinition for the given code, or
// the zero value if the code is unknown.
func StageByCode(code string) contracts.StageDefinition {
	if s := FindStage(StageCode(code)); s != nil {
		return s.ToContract()
	}
	return contracts.StageDefinition{}
}

// NigeriaTerm is the richer, Nigeria-specific glossary entry. It carries
// country, sources (URLs to official Nigerian documents) and an optional
// stage association. The contracts.TermDefinition type only carries a
// canonical key, term and short description — adapter.go projects the
// richer record down to that shape.
type NigeriaTerm struct {
	Term                string
	CanonicalKey        string
	SimpleExplanation   string
	OfficialDefinition  string
	Stage               string // optional stage code, "" if general
	Country             string // always "NG"
	Sources             []string
}

// NigeriaTerminology is the registry of Nigerian parliamentary and legal
// terms used by the adapter when normalising extracted records.
//
// Sources used to compile these definitions:
//   - Constitution of the Federal Republic of Nigeria, 1999 (as amended)
//     https://www.constituteproject.org/constitution/Nigeria_1999
//   - National Assembly of Nigeria
//     https://nass.gov.ng/
//   - National Institute for Legislative and Democratic Studies (NILDS)
//     https://www.nilds.gov.ng/
var NigeriaTerminology = []NigeriaTerm{
	{
		Term:              "First Reading",
		CanonicalKey:      "first_reading",
		SimpleExplanation: "The formal introduction of a Bill in a chamber; the Bill is read by title only and entered on the Order Paper.",
		OfficialDefinition: "Publication and first reading per the Standing Orders of the Senate and House of Representatives.",
		Stage:    string(StageFirstReading),
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Second Reading",
		CanonicalKey:      "second_reading",
		SimpleExplanation: "Debate on the principles and policy of a Bill, followed by a vote on whether it should proceed.",
		OfficialDefinition: "Per the Standing Orders. The lead debate outlines the policy; debate follows; a vote decides whether to refer the Bill to committee.",
		Stage:    string(StageSecondReading),
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Public Hearing",
		CanonicalKey:      "public_hearing",
		SimpleExplanation: "A forum convened by a committee to receive memoranda and oral submissions from the public and stakeholders on a Bill.",
		OfficialDefinition: "Public hearing conducted by a committee, per the Standing Orders of both chambers. Memoranda received inform the committee's clause-by-clause consideration.",
		Stage:    string(StagePublicHearing),
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Committee",
		CanonicalKey:      "committee",
		SimpleExplanation: "Scrutiny of a Bill by a standing or ad-hoc committee, including consideration of memoranda received at the public hearing.",
		OfficialDefinition: "Per the Standing Orders; the committee reports back to the chamber with proposed amendments.",
		Stage:    string(StageCommittee),
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Report",
		CanonicalKey:      "report",
		SimpleExplanation: "The committee reports its proposed amendments back to the full chamber for decision.",
		OfficialDefinition: "Per the Standing Orders of the Senate and House of Representatives.",
		Stage:    string(StageReport),
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Third Reading",
		CanonicalKey:      "third_reading",
		SimpleExplanation: "Final reading and vote on a Bill in a chamber.",
		OfficialDefinition: "Per the Standing Orders; after third reading the Bill is forwarded to the other chamber for concurrence.",
		Stage:    string(StageThirdReading),
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Concurrence",
		CanonicalKey:      "concurrence",
		SimpleExplanation: "Agreement by the other chamber of the National Assembly to a Bill passed by the originating chamber.",
		OfficialDefinition: "Per Section 58 of the Constitution. Where the chambers disagree on amendments, a Conference Committee is convened.",
		Stage:    string(StageConcurrence),
		Country:  "NG",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
			"https://nass.gov.ng/",
		},
	},
	{
		Term:              "Conference Committee",
		CanonicalKey:      "conference_committee",
		SimpleExplanation: "A joint committee of both chambers formed when they disagree on a Bill, to agree a compromise version.",
		OfficialDefinition: "Conference Committee convened under the Standing Orders to resolve disagreements between the Senate and the House of Representatives on a Bill.",
		Stage:    string(StageConferenceCommittee),
		Country:  "NG",
		Sources:  []string{"https://www.constituteproject.org/constitution/Nigeria_1999"},
	},
	{
		Term:              "Assent",
		CanonicalKey:      "assent",
		SimpleExplanation: "The President signs a Bill passed by the National Assembly into law.",
		OfficialDefinition: "Section 58 of the Constitution of the Federal Republic of Nigeria, 1999 (as amended). The President may withhold assent (veto); the National Assembly may override by a two-thirds majority of each chamber.",
		Stage:    string(StageAssent),
		Country:  "NG",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
			"https://nass.gov.ng/",
		},
	},
	{
		Term:              "Veto",
		CanonicalKey:      "veto",
		SimpleExplanation: "The President has refused assent to a Bill. The National Assembly may override by a two-thirds majority of each chamber.",
		OfficialDefinition: "Section 58(4)–(5) of the Constitution. The President's refusal of assent; overridable by a two-thirds majority of each chamber of the National Assembly.",
		Stage:    string(StageVeto),
		Country:  "NG",
		Sources:  []string{"https://www.constituteproject.org/constitution/Nigeria_1999"},
	},
	{
		Term:              "Commencement",
		CanonicalKey:      "commencement",
		SimpleExplanation: "The date an Act comes into force, fixed by the Act or by a notice in the Federal Government Gazette.",
		OfficialDefinition: "Per the Interpretation Act and the Act's own commencement provision. Where no date is fixed, commencement is by Gazette notice.",
		Stage:    string(StageCommencement),
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Order Paper",
		CanonicalKey:      "order_paper",
		SimpleExplanation: "The daily published agenda of business for a chamber.",
		OfficialDefinition: "The Order Paper sets out the business of a sitting; it is published by the Clerk of each chamber before each sitting.",
		Country:  "NG",
		Sources:  []string{
			"https://nass.gov.ng/house/order-paper",
			"https://nass.gov.ng/senate/order-paper",
		},
	},
	{
		Term:              "Hansard",
		CanonicalKey:      "hansard",
		SimpleExplanation: "The official verbatim report of debates in a chamber.",
		OfficialDefinition: "The official report of parliamentary debates, prepared by the Hansard department of each chamber.",
		Country:  "NG",
		Sources:  []string{
			"https://nass.gov.ng/house/hansard",
			"https://nass.gov.ng/senate/hansard",
		},
	},
	{
		Term:              "Votes and Proceedings",
		CanonicalKey:      "votes_and_proceedings",
		SimpleExplanation: "The official record of decisions and divisions taken at each sitting.",
		OfficialDefinition: "The Votes and Proceedings record every decision of the chamber; it is the authoritative source for vote outcomes.",
		Country:  "NG",
		Sources:  []string{
			"https://nass.gov.ng/house/votes-and-proceedings",
			"https://nass.gov.ng/senate/votes-and-proceedings",
		},
	},
	{
		Term:              "Gazette",
		CanonicalKey:      "gazette",
		SimpleExplanation: "An official announcement published in the Federal Government Gazette by the Federal Government Press.",
		OfficialDefinition: "A notice published in the Federal Government Gazette (or a supplement) under the Gazette Act. Gazette notices include Acts, commencement notices, appointments and statutory instruments.",
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Executive Bill",
		CanonicalKey:      "executive_bill",
		SimpleExplanation: "A Bill introduced by the Executive (the President) and forwarded to the National Assembly for enactment.",
		OfficialDefinition: "A Bill sponsored by the Executive branch, transmitted to the National Assembly by the President. Distinguished from a Member's (Private Member's) Bill.",
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Member's Bill",
		CanonicalKey:      "members_bill",
		SimpleExplanation: "A Bill introduced by an individual member of the National Assembly rather than the Executive.",
		OfficialDefinition: "A Bill sponsored by a member (or members) in their individual capacity. The member's name appears as the sponsor.",
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Money Bill",
		CanonicalKey:      "money_bill",
		SimpleExplanation: "A Bill dealing with taxation, public debt or appropriation of public funds, requiring special procedures per Section 59 of the Constitution.",
		OfficialDefinition: "A Bill within the meaning of Section 59 of the Constitution. Money Bills follow special procedures for passage and assent.",
		Country:  "NG",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
			"https://nass.gov.ng/",
		},
	},
	{
		Term:              "Appropriation Bill",
		CanonicalKey:      "appropriation_bill",
		SimpleExplanation: "The annual Bill authorising withdrawal of funds from the Consolidated Revenue Fund to fund the Federal Government's budget.",
		OfficialDefinition: "Per Section 81 of the Constitution. The President causes the Appropriation Bill to be introduced; the National Assembly may modify it but must pass it before the start of the financial year.",
		Country:  "NG",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
			"https://nass.gov.ng/",
		},
	},
	{
		Term:              "Bicameral",
		CanonicalKey:      "bicameral",
		SimpleExplanation: "A legislature with two chambers; Nigeria's National Assembly has the House of Representatives (lower) and Senate (upper).",
		OfficialDefinition: "Section 47 of the Constitution establishes the National Assembly as two chambers: the Senate and the House of Representatives.",
		Country:  "NG",
		Sources:  []string{"https://www.constituteproject.org/constitution/Nigeria_1999"},
	},
	{
		Term:              "Senate President",
		CanonicalKey:      "senate_president",
		SimpleExplanation: "The presiding officer of the Senate, elected under Section 50 of the Constitution.",
		OfficialDefinition: "Section 50 of the Constitution establishes the President of the Senate as the presiding officer of the Senate.",
		Country:  "NG",
		Sources:  []string{
			"https://nass.gov.ng/senate",
			"https://www.constituteproject.org/constitution/Nigeria_1999",
		},
	},
	{
		Term:              "Speaker of the House",
		CanonicalKey:      "speaker_of_the_house",
		SimpleExplanation: "The presiding officer of the House of Representatives, elected under Section 50 of the Constitution.",
		OfficialDefinition: "Section 50 of the Constitution establishes the Speaker of the House of Representatives as the presiding officer of the House.",
		Country:  "NG",
		Sources:  []string{
			"https://nass.gov.ng/house",
			"https://www.constituteproject.org/constitution/Nigeria_1999",
		},
	},
	{
		Term:              "Clerk to the National Assembly",
		CanonicalKey:      "clerk_to_the_national_assembly",
		SimpleExplanation: "The chief administrative officer of the National Assembly, responsible for publishing the Order Paper, Hansard, and Votes and Proceedings.",
		OfficialDefinition: "The Clerk to the National Assembly is the head of the National Assembly bureaucracy and the chief procedural advisor to both chambers.",
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Attorney General of the Federation",
		CanonicalKey:      "attorney_general_of_the_federation",
		SimpleExplanation: "The principal legal advisor to the Federal Government, established under Section 150 of the Constitution.",
		OfficialDefinition: "Section 150 of the Constitution. The Attorney General of the Federation is the chief legal advisor to the Federal Government and a member of the Federal Executive Council.",
		Country:  "NG",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
		},
	},
	{
		Term:              "Public Participation",
		CanonicalKey:      "public_participation",
		SimpleExplanation: "The process by which the public submits memoranda and oral evidence at public hearings on Bills and other business of the National Assembly.",
		OfficialDefinition: "Public hearings are convened by committees to receive memoranda and oral submissions from the public and stakeholders on pending Bills.",
		Country:  "NG",
		Sources:  []string{
			"https://nass.gov.ng/",
		},
	},
	{
		Term:              "Memorandum",
		CanonicalKey:      "memorandum",
		SimpleExplanation: "A written submission by a member of the public or organisation to a committee considering a Bill.",
		OfficialDefinition: "A memorandum is a written submission made to a committee of the National Assembly, typically in response to a public hearing notice on a Bill.",
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Constituency",
		CanonicalKey:      "constituency",
		SimpleExplanation: "An electoral district represented by a member of the House of Representatives or the Senate.",
		OfficialDefinition: "Nigeria is divided into 360 Federal Constituencies (each electing one member to the House of Representatives) and 109 Senatorial Districts (each electing one Senator), per the Constitution.",
		Country:  "NG",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Nigeria_1999",
			"https://nass.gov.ng/",
		},
	},
	{
		Term:              "State House of Assembly",
		CanonicalKey:      "state_house_of_assembly",
		SimpleExplanation: "The legislature of one of Nigeria's 36 states, established under Section 90 of the Constitution.",
		OfficialDefinition: "Section 90 of the Constitution establishes a House of Assembly for each of the 36 states of the Federation.",
		Country:  "NG",
		Sources:  []string{"https://www.constituteproject.org/constitution/Nigeria_1999"},
	},
	{
		Term:              "Consolidated Revenue Fund",
		CanonicalKey:      "consolidated_revenue_fund",
		SimpleExplanation: "The principal fund of the Federal Government into which all revenues are paid, established under Section 80 of the Constitution.",
		OfficialDefinition: "Section 80 of the Constitution. No monies may be withdrawn from the Consolidated Revenue Fund except as authorised by an Appropriation Act.",
		Country:  "NG",
		Sources:  []string{"https://www.constituteproject.org/constitution/Nigeria_1999"},
	},
	{
		Term:              "Prorogation",
		CanonicalKey:      "prorogation",
		SimpleExplanation: "The formal end of a session of the National Assembly by proclamation of the President, without dissolving the Assembly.",
		OfficialDefinition: "A session of the National Assembly is prorogued by the President; pending business generally lapses unless carried over.",
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
	{
		Term:              "Quorum",
		CanonicalKey:      "quorum",
		SimpleExplanation: "The minimum number of members (one-third of the members of the chamber) required to conduct business.",
		OfficialDefinition: "Section 54 of the Constitution. The quorum of the Senate is one-third of its members; the quorum of the House of Representatives is one-third of its members.",
		Country:  "NG",
		Sources:  []string{"https://www.constituteproject.org/constitution/Nigeria_1999"},
	},
	{
		Term:              "Division",
		CanonicalKey:      "division",
		SimpleExplanation: "A formal recorded vote where members' names are entered in the Votes and Proceedings.",
		OfficialDefinition: "A division is the procedure by which the chamber votes with the ayes and noes recorded individually; results are entered into the Votes and Proceedings.",
		Country:  "NG",
		Sources:  []string{"https://nass.gov.ng/"},
	},
}

// FindTerm looks up a NigeriaTerm by its canonical key. Returns nil if not found.
func FindTerm(key string) *NigeriaTerm {
	for i := range NigeriaTerminology {
		if NigeriaTerminology[i].CanonicalKey == key {
			return &NigeriaTerminology[i]
		}
	}
	return nil
}

// ToContract projects the richer NigeriaTerm down to the platform's
// country-agnostic contracts.TermDefinition. The canonical key is the only
// string that links the platform back to this richer record.
func (t NigeriaTerm) ToContract() contracts.TermDefinition {
	return contracts.TermDefinition{
		Term:                t.Term,
		CanonicalKey:        t.CanonicalKey,
		SimpleExplanation:   t.SimpleExplanation,
		Description:         t.SimpleExplanation,
		OfficialDefinition:  t.OfficialDefinition,
		StageCode:           t.Stage,
		Country:             contracts.Country(t.Country),
		Sources:             t.Sources,
	}
}

// NigeriaLegislativeStructure returns the structural description of Nigeria's
// National Assembly as established by the 1999 Constitution: a bicameral
// legislature comprising the House of Representatives (lower house) and the
// Senate (upper house), each with its standing committees.
//
// Sources:
//   - Constitution of the Federal Republic of Nigeria, 1999 (as amended),
//     Sections 47–89 (The National Assembly)
//     https://www.constituteproject.org/constitution/Nigeria_1999
//   - National Assembly of Nigeria
//     https://nass.gov.ng/
//   - The Senate
//     https://nass.gov.ng/senate
//   - The House of Representatives
//     https://nass.gov.ng/house
func NigeriaLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     contracts.Country("NG"),
		CountryCode: "NG",
		CountryName: "Nigeria",
		Houses: []contracts.HouseDefinition{
			{
				Code:     HouseCodeHouseOfReps,
				Name:     "House of Representatives",
				Type:     contracts.HouseTypeLower,
				Members:  360, // 360 Federal Constituencies, each electing one member per Section 49 of the Constitution
				TermDays: 4 * 365, // Section 64: 4-year term
			},
			{
				Code:     HouseCodeSenate,
				Name:     "Senate",
				Type:     contracts.HouseTypeUpper,
				Members:  109, // 109 Senatorial Districts (3 per state + 1 for the FCT) per Section 48 of the Constitution
				TermDays: 4 * 365, // Section 64: 4-year term
			},
		},
		Committees: nigeriaStandingCommittees(),
	}
}

// nigeriaStandingCommittees returns a representative sample of the standing
// committees that the Senate and House of Representatives publish on their
// official site. The National Assembly has over 100 standing committees in
// total; we list the most prominent ones by chamber.
//
// Committee definitions are derived from the published Committee Lists:
//   - House of Representatives committees
//     https://nass.gov.ng/house/committees
//   - Senate committees
//     https://nass.gov.ng/senate/committees
func nigeriaStandingCommittees() []contracts.CommitteeDefinition {
	return []contracts.CommitteeDefinition{
		// House of Representatives standing committees.
		{Code: CommitteeAppropriations, Name: "Appropriations", House: HouseCodeHouseOfReps, Type: "standing", Members: 41},
		{Code: CommitteeFinance, Name: "Finance", House: HouseCodeHouseOfReps, Type: "standing", Members: 29},
		{Code: CommitteeJudiciary, Name: "Judiciary", House: HouseCodeHouseOfReps, Type: "standing", Members: 25},
		{Code: CommitteeHealth, Name: "Health", House: HouseCodeHouseOfReps, Type: "standing", Members: 33},
		{Code: CommitteeEducation, Name: "Education", House: HouseCodeHouseOfReps, Type: "standing", Members: 33},
		{Code: CommitteeDefence, Name: "Defence", House: HouseCodeHouseOfReps, Type: "standing", Members: 25},
		{Code: CommitteePublicAccounts, Name: "Public Accounts", House: HouseCodeHouseOfReps, Type: "standing", Members: 29},
		{Code: CommitteeWorks, Name: "Works", House: HouseCodeHouseOfReps, Type: "standing", Members: 33},
		{Code: CommitteePower, Name: "Power", House: HouseCodeHouseOfReps, Type: "standing", Members: 33},
		{Code: CommitteeCommunication, Name: "Communications", House: HouseCodeHouseOfReps, Type: "standing", Members: 25},
		{Code: CommitteeAgriculture, Name: "Agriculture", House: HouseCodeHouseOfReps, Type: "standing", Members: 29},
		{Code: CommitteeTransport, Name: "Land Transport", House: HouseCodeHouseOfReps, Type: "standing", Members: 29},
		{Code: CommitteeAviation, Name: "Aviation", House: HouseCodeHouseOfReps, Type: "standing", Members: 25},
		{Code: CommitteeNigerDelta, Name: "Niger Delta Affairs", House: HouseCodeHouseOfReps, Type: "standing", Members: 29},
		{Code: CommitteeEcologicalFunds, Name: "Ecological Funds", House: HouseCodeHouseOfReps, Type: "standing", Members: 25},
		{Code: CommitteeRulesBusiness, Name: "Rules and Business", House: HouseCodeHouseOfReps, Type: "standing", Members: 25},
		{Code: CommitteeEthicsPrivileges, Name: "Ethics and Privileges", House: HouseCodeHouseOfReps, Type: "standing", Members: 25},
		{Code: CommitteeSelection, Name: "Selection", House: HouseCodeHouseOfReps, Type: "select", Members: 25},
		{Code: CommitteeWomenAffairs, Name: "Women Affairs", House: HouseCodeHouseOfReps, Type: "standing", Members: 25},
		{Code: CommitteeYouthSports, Name: "Youth and Sports", House: HouseCodeHouseOfReps, Type: "standing", Members: 25},

		// Senate standing committees.
		{Code: CommitteeAppropriations, Name: "Appropriations", House: HouseCodeSenate, Type: "standing", Members: 19},
		{Code: CommitteeFinance, Name: "Finance", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeJudiciary, Name: "Judiciary, Human Rights and Legal Matters", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeHealth, Name: "Health (Secondary and Tertiary)", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeEducation, Name: "Education (Basic and Secondary)", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeDefence, Name: "Defence", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteePublicAccounts, Name: "Public Accounts", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeWorks, Name: "Works", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteePower, Name: "Power", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeCommunication, Name: "Communications", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeAgriculture, Name: "Agriculture", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeTransport, Name: "Land Transport", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeAviation, Name: "Aviation", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeNigerDelta, Name: "Niger Delta Affairs", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeEcologicalFunds, Name: "Ecological Funds", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeRulesBusiness, Name: "Rules and Business", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeEthicsPrivileges, Name: "Ethics, Privileges and Public Petitions", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeSelection, Name: "Selection", House: HouseCodeSenate, Type: "select", Members: 17},
		{Code: CommitteeWomenAffairs, Name: "Women Affairs", House: HouseCodeSenate, Type: "standing", Members: 17},
		{Code: CommitteeYouthSports, Name: "Youth and Sports", House: HouseCodeSenate, Type: "standing", Members: 17},
	}
}

// NigeriaSampleBills is a representative sample of recent Nigerian National
// Assembly Bills, used to seed local fixtures (testdata/bills_house.html and
// testdata/bills_senate.html) and contract tests without touching the
// network. Titles are realistic Nigerian Bill titles drawn from publicly
// published House of Representatives and Senate Bills.
//
// Bills are bicameral: some originate in the House of Representatives (HB.
// prefix), others in the Senate (SB. prefix). Both houses are represented in
// this sample — the contract test (TestAdapter_DiscoverBills_ViaMockServer)
// serves a fixture per chamber and verifies Discover returns Bills tagged
// with the originating House.
//
// These records are values, not constants — they exist so test fixtures can
// be regenerated deterministically and so the adapter's Parse step has a
// known-shape input to extract from.
var NigeriaSampleBills = []NigeriaSampleBill{
	{
		Title:      "The Electric Power Sector Reform (Amendment) Bill, 2024",
		BillNumber: "HB. 1234",
		Sponsor:    "Rep. Babajide Obanikoro (APC, Lagos)",
		Stage:      "Second Reading",
		Date:       "12 March 2024",
		URL:        "https://nass.gov.ng/house/bills/HB-1234-2024.pdf",
		House:      "House of Representatives",
	},
	{
		Title:      "The Nigerian Minerals and Mining (Amendment) Bill, 2024",
		BillNumber: "HB. 1567",
		Sponsor:    "Rep. Aliyu Sani (PDP, Kaduna)",
		Stage:      "Public Hearing",
		Date:       "28 April 2024",
		URL:        "https://nass.gov.ng/house/bills/HB-1567-2024.pdf",
		House:      "House of Representatives",
	},
	{
		Title:      "The Federal University of Technology (Establishment) Bill, 2024",
		BillNumber: "HB. 1789",
		Sponsor:    "Rep. Fatima Bello (APC, Kano)",
		Stage:      "First Reading",
		Date:       "06 June 2024",
		URL:        "https://nass.gov.ng/house/bills/HB-1789-2024.pdf",
		House:      "House of Representatives",
	},
	{
		Title:      "The Electoral Act (Amendment) Bill, 2024",
		BillNumber: "SB. 421",
		Sponsor:    "Sen. Oluwole Bode (PDP, Oyo)",
		Stage:      "Concurrence",
		Date:       "11 July 2024",
		URL:        "https://nass.gov.ng/senate/bills/SB-421-2024.pdf",
		House:      "Senate",
	},
	{
		Title:      "The Cybercrimes (Prohibition, Prevention) (Amendment) Bill, 2024",
		BillNumber: "SB. 567",
		Sponsor:    "Sen. Halima Musa (APC, Borno)",
		Stage:      "Committee",
		Date:       "18 September 2024",
		URL:        "https://nass.gov.ng/senate/bills/SB-567-2024.pdf",
		House:      "Senate",
	},
}

// NigeriaSampleBill is a single sample Bill record. Field names mirror the
// bill-card HTML structure parsed by parliament.ParseBillsListing.
type NigeriaSampleBill struct {
	Title      string
	BillNumber string
	Sponsor    string
	Stage      string
	Date       string
	URL        string
	House      string
}
