// Package internal holds Kenya-specific knowledge that must NEVER leak into the
// global platform types. The contracts package defines only country-agnostic
// shapes (StageDefinition, TermDefinition, ...). Every Kenyan string —
// "Second Reading", "Committee of the Whole House", "Presidential Assent",
// "Senate", "National Assembly", "Kenya Gazette" — lives in this package.
//
// The Kenya adapter's adapter.go projects these richer records down into the
// plain contracts.* types when the ingestion service asks for them.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// StageCode is the stable identifier Kenya uses for a bill lifecycle stage.
// These codes are the ONLY stage-related strings that ever cross into the
// global domain model (they are case-normalised SCREAMING_SNAKE strings,
// namespaced implicitly by the adapter's CountryCode()).
//
// The codes match the codes used by the Kenya Parliament Bill Tracker export
// when available; otherwise they follow the conventions of the 2010
// Constitution and the Standing Orders of the National Assembly and Senate.
type StageCode string

const (
	// StageFirstReading is publication + introduction. The Bill is read a
	// first time and published in the Kenya Gazette.
	StageFirstReading StageCode = "FIRST_READING"

	// StageSecondReading is the principle debate: the House debates whether
	// the Bill's policy and principles should be approved.
	StageSecondReading StageCode = "SECOND_READING"

	// StageCommitteeStage is the departmental committee stage where the
	// Bill is scrutinised clause-by-clause and public participation is held.
	StageCommitteeStage StageCode = "COMMITTEE_STAGE"

	// StageCommitteeOfWholeHouse is the Committee of the Whole House
	// (the House sitting in committee, chaired by the Chair of Committees)
	// where clause-by-clause amendments are considered on the floor.
	StageCommitteeOfWholeHouse StageCode = "COMMITTEE_OF_WHOLE_HOUSE"

	// StageReportStage is the report stage: the committee reports its
	// amendments back to the House, which may further amend.
	StageReportStage StageCode = "REPORT_STAGE"

	// StageThirdReading is the final reading in a house, after which the
	// Bill is passed by that house.
	StageThirdReading StageCode = "THIRD_READING"

	// StagePresidentialAssent is the assent by the President per Article 115
	// of the Constitution.
	StagePresidentialAssent StageCode = "PRESIDENTIAL_ASSENT"

	// StageCommencement is when an Act is brought into force by a
	// commencement notice in the Kenya Gazette.
	StageCommencement StageCode = "COMMENCEMENT"

	// StageRejected is the terminal state for a Bill that fails to pass.
	StageRejected StageCode = "REJECTED"

	// StageWithdrawn is the terminal state for a Bill that is withdrawn by
	// the Mover.
	StageWithdrawn StageCode = "WITHDRAWN"

	// StageLapsed is the terminal state for a Bill that lapses at the end of
	// a Parliament without concluding.
	StageLapsed StageCode = "LAPSED"

	// StageMediation is used when the two houses disagree and a Mediation
	// Committee is convened per Article 113 of the Constitution.
	StageMediation StageCode = "MEDIATION"
)

// KenyaStage is the richer, Kenya-specific stage record. It carries the
// country code, the official definition (with citation to the Standing Orders
// or the Constitution), and the canonical source URLs for the stage.
//
// The platform NEVER receives a KenyaStage directly. adapter.go projects it
// down to a contracts.StageDefinition when the ingestion service asks for the
// BillStages.
type KenyaStage struct {
	Code                StageCode
	Name                string
	SimpleExplanation   string
	OfficialDefinition  string
	Country             string // always "KE"
	Order               int
	AllowedNext         []StageCode
	RequiresEvidence    bool
	RequiresVote        bool
	TypicalDurationDays int
	Sources             []string
}

// KenyaBillStages is the ordered, authoritative list of Kenya's bill stages.
//
// Sources used to compile these definitions:
//   - Constitution of Kenya, 2010, Articles 109–115 (legislative process)
//     https://www.constituteproject.org/constitution/Kenya_2010
//   - The Standing Orders of the National Assembly (current edition)
//     https://www.parliament.go.ke/sites/default/files/2021-09/National-Assembly-Standing-Orders.pdf
//   - The Standing Orders of the Senate
//     https://www.parliament.go.ke/sites/default/files/2021-09/Senate-Standing-Orders.pdf
//   - Kenya Law Reform Commission, "Guide to the Legislative Process"
//     https://www.klrc.go.ke/
//
// Stage transitions follow the Standing Orders (a Bill normally proceeds
// First Reading → Second Reading → Committee Stage → Committee of the Whole
// House → Report Stage → Third Reading → Presidential Assent → Commencement).
// A Bill may also be Mediated (between houses), Rejected, Withdrawn or Lapsed.
var KenyaBillStages = []KenyaStage{
	{
		Code:      StageFirstReading,
		Name:      "First Reading",
		Order:     1,
		Country:   "KE",
		SimpleExplanation: "The Bill is read the first time in the house and published in the Kenya Gazette. No debate on the merits takes place.",
		OfficialDefinition: "Publication and first reading of a Bill as provided for by Standing Order 117 (National Assembly) / Standing Order 134 (Senate) and Article 109 of the Constitution.",
		AllowedNext: []StageCode{
			StageSecondReading,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:         false,
		TypicalDurationDays: 14,
		Sources: []string{
			"https://www.parliament.go.ke/",
			"https://www.constituteproject.org/constitution/Kenya_2010",
		},
	},
	{
		Code:      StageSecondReading,
		Name:      "Second Reading",
		Order:     2,
		Country:   "KE",
		SimpleExplanation: "The house debates and votes on the principles and policy of the Bill. No clause-by-clause amendments happen here.",
		OfficialDefinition: "Debate on the principles and policy of a Bill, governed by Standing Order 118 (National Assembly) and Standing Order 135 (Senate). At the conclusion the House votes on whether the Bill should proceed to committee.",
		AllowedNext: []StageCode{
			StageCommitteeStage,
			StageCommitteeOfWholeHouse,
			StageRejected,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        true,
		TypicalDurationDays: 30,
		Sources: []string{
			"https://www.parliament.go.ke/",
			"https://www.constituteproject.org/constitution/Kenya_2010",
		},
	},
	{
		Code:      StageCommitteeStage,
		Name:      "Committee Stage",
		Order:     3,
		Country:   "KE",
		SimpleExplanation: "The relevant departmental committee scrutinises the Bill clause-by-clause, takes public submissions, and proposes amendments.",
		OfficialDefinition: "Departmental committee scrutiny of a Bill, including public participation, per Article 118 of the Constitution and Standing Orders 119 and 136.",
		AllowedNext: []StageCode{
			StageReportStage,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 90,
		Sources: []string{
			"https://www.parliament.go.ke/the-national-assembly/departmental-committees",
			"https://www.parliament.go.ke/the-senate/standing-committees",
			"https://www.constituteproject.org/constitution/Kenya_2010",
		},
	},
	{
		Code:      StageCommitteeOfWholeHouse,
		Name:      "Committee of the Whole House",
		Order:     4,
		Country:   "KE",
		SimpleExplanation: "The house sits as a committee (chaired by the Chair of Committees) and considers the Bill clause-by-clause on the floor, voting on each clause and any amendments.",
		OfficialDefinition: "Consideration of a Bill clause-by-clause by the House in committee, per Standing Orders 120 and 137. The Chair of Committees presides.",
		AllowedNext: []StageCode{
			StageReportStage,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        true,
		TypicalDurationDays: 7,
		Sources: []string{
			"https://www.parliament.go.ke/",
		},
	},
	{
		Code:      StageReportStage,
		Name:      "Report Stage",
		Order:     5,
		Country:   "KE",
		SimpleExplanation: "The committee reports its proposed amendments back to the full house, which can accept, reject or further amend them.",
		OfficialDefinition: "Report of a committee on a Bill and consideration of amendments on the floor, per Standing Orders 121 and 138.",
		AllowedNext: []StageCode{
			StageThirdReading,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        true,
		TypicalDurationDays: 7,
		Sources: []string{
			"https://www.parliament.go.ke/",
		},
	},
	{
		Code:      StageThirdReading,
		Name:      "Third Reading",
		Order:     6,
		Country:   "KE",
		SimpleExplanation: "The final reading in the house. The Bill is voted on as amended; if passed it is forwarded to the other house (or, if agreed by both, to the President).",
		OfficialDefinition: "Third reading and passing of a Bill by a house, per Standing Orders 122 and 139. A Bill concerning counties must pass both houses; an ordinary Bill may originate in either.",
		AllowedNext: []StageCode{
			StagePresidentialAssent,
			StageMediation,
			StageRejected,
			StageWithdrawn,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        true,
		TypicalDurationDays: 7,
		Sources: []string{
			"https://www.parliament.go.ke/",
			"https://www.constituteproject.org/constitution/Kenya_2010",
		},
	},
	{
		Code:      StageMediation,
		Name:      "Mediation Committee",
		Order:     7,
		Country:   "KE",
		SimpleExplanation: "When the National Assembly and Senate disagree on a Bill concerning counties, a Mediation Committee is formed to agree a compromise text.",
		OfficialDefinition: "Mediation between the two houses per Article 113 of the Constitution. A Mediation Committee prepares a version of the Bill for both houses to approve.",
		AllowedNext: []StageCode{
			StagePresidentialAssent,
			StageRejected,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        true,
		TypicalDurationDays: 30,
		Sources: []string{
			"https://www.constituteproject.org/constitution/Kenya_2010",
			"https://www.parliament.go.ke/",
		},
	},
	{
		Code:      StagePresidentialAssent,
		Name:      "Presidential Assent",
		Order:     8,
		Country:   "KE",
		SimpleExplanation: "The President signs the Bill into law per Article 115 of the Constitution. May be referred back once; otherwise it becomes law.",
		OfficialDefinition: "Assent to a Bill by the President, per Article 115 of the Constitution. The President may refer a Bill back once; if returned and re-passed, assent is mandatory (subject to a constitutional referral to the Supreme Court).",
		AllowedNext: []StageCode{
			StageCommencement,
			StageLapsed,
		},
		RequiresEvidence:    true,
		RequiresVote:        false,
		TypicalDurationDays: 14,
		Sources: []string{
			"https://www.constituteproject.org/constitution/Kenya_2010",
			"https://www.parliament.go.ke/",
			"https://www.kenyalaw.org/",
		},
	},
	{
		Code:      StageCommencement,
		Name:      "Commencement",
		Order:     9,
		Country:   "KE",
		SimpleExplanation: "The Act is brought into force, either on a date fixed by the Act itself or by a commencement notice in the Kenya Gazette.",
		OfficialDefinition: "Coming into operation of an Act of Parliament, per the Interpretation and General Provisions Act (Cap 2) and the Act itself. Where no date is fixed, commencement is by Legal Notice in the Kenya Gazette.",
		AllowedNext:        nil, // terminal-success
		RequiresEvidence:   true,
		RequiresVote:       false,
		TypicalDurationDays: 30,
		Sources: []string{
			"https://www.kenyalaw.org/",
			"https://gazettes.africa/",
		},
	},
	{
		Code:      StageRejected,
		Name:      "Rejected",
		Order:     100,
		Country:   "KE",
		SimpleExplanation: "The Bill failed to pass a vote at second or third reading, or failed Mediation Committee consideration.",
		OfficialDefinition: "Terminal state: the Bill has failed a required vote and cannot proceed in this Parliament.",
		AllowedNext:        nil, // terminal
		RequiresEvidence:   true,
		RequiresVote:       true,
		TypicalDurationDays: 0,
		Sources: []string{
			"https://www.parliament.go.ke/",
		},
	},
	{
		Code:      StageWithdrawn,
		Name:      "Withdrawn",
		Order:     101,
		Country:   "KE",
		SimpleExplanation: "The mover of the Bill has withdrawn it, per the Standing Orders.",
		OfficialDefinition: "Terminal state: a Bill withdrawn by the Mover or originating department, per Standing Orders 124 and 141.",
		AllowedNext:        nil, // terminal
		RequiresEvidence:   true,
		RequiresVote:       false,
		TypicalDurationDays: 0,
		Sources: []string{
			"https://www.parliament.go.ke/",
		},
	},
	{
		Code:      StageLapsed,
		Name:      "Lapsed",
		Order:     102,
		Country:   "KE",
		SimpleExplanation: "The Bill has lapsed at the end of a Parliament without concluding its passage.",
		OfficialDefinition: "Terminal state: a Bill lapses at the dissolution of Parliament per Article 109(5) of the Constitution.",
		AllowedNext:        nil, // terminal
		RequiresEvidence:   true,
		RequiresVote:       false,
		TypicalDurationDays: 0,
		Sources: []string{
			"https://www.constituteproject.org/constitution/Kenya_2010",
		},
	},
}

// FindStage looks up a KenyaStage by its code. Returns nil if not found.
func FindStage(code StageCode) *KenyaStage {
	for i := range KenyaBillStages {
		if KenyaBillStages[i].Code == code {
			return &KenyaBillStages[i]
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
// Standing Orders model encoded in KenyaBillStages.
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

// ToContract projects the richer KenyaStage down to the platform's
// country-agnostic contracts.StageDefinition. No Kenya-specific string
// other than the stage Code and Name crosses this boundary.
func (s KenyaStage) ToContract() contracts.StageDefinition {
	next := make([]string, 0, len(s.AllowedNext))
	for _, n := range s.AllowedNext {
		next = append(next, string(n))
	}
	return contracts.StageDefinition{
		Code:                string(s.Code),
		Name:                s.Name,
		Description:         s.SimpleExplanation,
		Order:               s.Order,
		AllowedTransitions:  next,
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
