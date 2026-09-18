// Package internal holds Senegal-specific legislative data.
//
// ALL Senegal-specific knowledge lives here:
//   - The unicameral National Assembly of Senegal (165 members) per Article 55
//     of the Constitution of Senegal (as revised in 2016 and 2019).
//   - Senegalese Bill stages (Dépôt → Commission → Première Lecture →
//     Deuxième Lecture → Adoption → Promulgation). The stage CODES are in
//     English (DEPOT, COMMISSION, FIRST_READING, SECOND_READING, ADOPTION,
//     PROMULGATION) for consistency with the rest of the platform; the
//     stage DISPLAY NAMES are in French as published by the Assemblée
//     Nationale on assemblee-nationale.sn.
//   - Senegalese parliamentary terminology (Hansard, Ordonnance, etc.).
//   - Source URLs for assemblee-nationale.sn.
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Senegal means writing adapters/senegal/ — the legislation
// service code is unchanged.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// SenegalBillStages defines Senegal's Bill lifecycle.
//
// Senegal is UNICAMERAL — the National Assembly of Senegal is the sole
// legislative body (no Senate). The canonical flow per Article 60 of the
// Constitution of Senegal (as revised in 2016 and 2019) and the Règlement
// Intérieur de l'Assemblée Nationale is:
//
//	Dépôt → Commission → Première Lecture → Deuxième Lecture →
//	Adoption → Promulgation.
//
// Stage CODES are in English (for consistency with the rest of the platform);
// stage NAMES are in French (matching the source documents on
// assemblee-nationale.sn).
//
// A Bill may also be Rejeté (Rejected) at a vote or Retiré (Withdrawn) by
// the mover.
var SenegalBillStages = []contracts.StageDefinition{
	{
		Code:              "DEPOT",
		Name:              "Dépôt",
		SimpleExplanation: "Le projet ou la proposition de loi est déposé sur le Bureau de l'Assemblée Nationale.",
		Country:           "SN",
		AllowedNext:       []string{"COMMISSION"},
	},
	{
		Code:              "COMMISSION",
		Name:              "Commission",
		SimpleExplanation: "Une commission de l'Assemblée Nationale examine le texte article par article et propose des amendements.",
		Country:           "SN",
		AllowedNext:       []string{"FIRST_READING"},
	},
	{
		Code:              "FIRST_READING",
		Name:              "Première Lecture",
		SimpleExplanation: "L'Assemblée Nationale débat des principes et de la politique du texte, puis vote sur son adoption en première lecture.",
		Country:           "SN",
		AllowedNext:       []string{"SECOND_READING", "REJECTED"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Deuxième Lecture",
		SimpleExplanation: "L'Assemblée Nationale examine le texte amendé en deuxième lecture, à la suite d'éventuelles révisions.",
		Country:           "SN",
		AllowedNext:       []string{"ADOPTION"},
	},
	{
		Code:              "ADOPTION",
		Name:              "Adoption",
		SimpleExplanation: "L'Assemblée Nationale adopte définitivement le texte par vote.",
		Country:           "SN",
		AllowedNext:       []string{"PROMULGATION", "REJECTED"},
	},
	{
		Code:              "PROMULGATION",
		Name:              "Promulgation",
		SimpleExplanation: "Le Président de la République promulgue la loi dans les conditions fixées par l'article 68 de la Constitution.",
		Country:           "SN",
		AllowedNext:       []string{"COMMENCEMENT"},
	},
	{
		Code:              "COMMENCEMENT",
		Name:              "Entrée en Vigueur",
		SimpleExplanation: "La loi entre en vigueur à la date de sa promulgation ou à la date fixée par ses dispositions.",
		Country:           "SN",
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejeté",
		SimpleExplanation: "Le texte a été rejeté à l'issue d'un vote de l'Assemblée Nationale.",
		Country:           "SN",
		IsTerminal:        true,
	},
	{
		Code:              "WITHDRAWN",
		Name:              "Retiré",
		SimpleExplanation: "Le texte a été retiré par son auteur avant son adoption.",
		Country:           "SN",
		IsTerminal:        true,
	},
}

// SenegalTerminology defines Senegalese parliamentary terms. Stage-related
// terms carry the French term as published by the Assemblée Nationale; the
// canonical keys are in English for cross-country consistency.
//
// Sources used to compile these definitions:
//   - Constitution of Senegal (as revised by the 2016 and 2019 referenda),
//     Articles 55–87.
//   - Assemblée Nationale du Sénégal — https://www.assemblee-nationale.sn/
//   - Règlement Intérieur de l'Assemblée Nationale.
var SenegalTerminology = []contracts.TermDefinition{
	{Term: "Dépôt", SimpleExplanation: "Le projet ou la proposition de loi est déposé sur le Bureau de l'Assemblée Nationale.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Commission", SimpleExplanation: "Une commission permanente examine le texte article par article et propose des amendements.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Première Lecture", SimpleExplanation: "L'Assemblée Nationale débat des principes et de la politique du texte, puis vote sur son adoption.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Deuxième Lecture", SimpleExplanation: "L'Assemblée Nationale examine le texte amendé en deuxième lecture.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Adoption", SimpleExplanation: "L'Assemblée Nationale adopte définitivement le texte par vote.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Promulgation", SimpleExplanation: "Le Président de la République promulgue la loi, la rendant exécutoire.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Entrée en Vigueur", SimpleExplanation: "La date à laquelle la loi devient applicable.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Loi", SimpleExplanation: "Règle de droit votée par l'Assemblée Nationale et promulguée par le Président de la République.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Projet de Loi", SimpleExplanation: "Texte présenté par le Gouvernement à l'Assemblée Nationale pour adoption.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Proposition de Loi", SimpleExplanation: "Texte présenté par un ou plusieurs députés à l'Assemblée Nationale pour adoption.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Loi de Finances", SimpleExplanation: "Loi annuelle qui autorise le Gouvernement à percevoir les impôts et à engager les dépenses de l'État.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Hansard", SimpleExplanation: "Le compte rendu officiel in extenso des débats de l'Assemblée Nationale.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Ordre du Jour", SimpleExplanation: "Le programme quotidien des travaux de l'Assemblée Nationale.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Assemblée Nationale", SimpleExplanation: "Le parlement unicaméral du Sénégal — 165 députés élus pour un mandat de cinq ans.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Commission Permanente", SimpleExplanation: "Une commission permanente de l'Assemblée Nationale responsable d'un domaine thématique.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Bureau de l'Assemblée", SimpleExplanation: "L'organe directeur de l'Assemblée Nationale, comprenant le Président et les Vice-Présidents.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Président de l'Assemblée Nationale", SimpleExplanation: "Le député élu pour présider les séances de l'Assemblée Nationale.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Député", SimpleExplanation: "Membre élu de l'Assemblée Nationale du Sénégal.", Country: "SN"},
	{Term: "Gouvernement", SimpleExplanation: "Le pouvoir exécutif, dirigé par le Premier Ministre et composé des Ministres; peut déposer des projets de loi.", Country: "SN"},
	{Term: "Ministre", SimpleExplanation: "Membre du Gouvernement responsable d'un département ministériel; peut présenter des projets de loi.", Country: "SN"},
	{Term: "Motion", SimpleExplanation: "Proposition formelle soumise à l'Assemblée Nationale pour débat et décision.", Country: "SN"},
	{Term: "Vote", SimpleExplanation: "Décision formelle de l'Assemblée Nationale exprimée par les députés.", Country: "SN"},
	{Term: "Scrutin Public", SimpleExplanation: "Vote public où les noms et les voix des députés sont enregistrés.", Country: "SN"},
	{Term: "Quorum", SimpleExplanation: "Le nombre minimum de députés requis pour que l'Assemblée Nationale puisse valablement délibérer.", Country: "SN"},
	{Term: "Ordonnance", SimpleExplanation: "Acte pris par le Président de la République dans les domaines réservés à la loi, sous réserve de ratification par l'Assemblée Nationale.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
	{Term: "Dissolution de l'Assemblée", SimpleExplanation: "La fin anticipée du mandat de l'Assemblée Nationale par décision du Président de la République, entraînant de nouvelles élections.", Country: "SN"},
	{Term: "Promulgation Présidentielle", SimpleExplanation: "L'acte par lequel le Président de la République promulgue la loi adoptée par l'Assemblée Nationale, dans les 15 jours de la transmission.", Country: "SN", Sources: []string{"https://www.assemblee-nationale.sn"}},
}

// SenegalLegislativeStructure returns Senegal's institutional structure.
// KEY: Senegal is UNICAMERAL — only one House (the National Assembly), unlike
// Kenya's bicameral (National Assembly + Senate). The National Assembly has
// 165 members serving five-year terms per Article 55 of the Constitution as
// revised.
//
// Sources:
//   - Constitution of Senegal (as revised by the 2016 and 2019 referenda),
//     Articles 55–87.
//   - https://www.assemblee-nationale.sn/
func SenegalLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "SN",
		CountryCode: "SN",
		CountryName: "Senegal",
		Houses: []contracts.HouseDefinition{
			{
				Code:     "NATIONAL_ASSEMBLY",
				Name:     "Assemblée Nationale du Sénégal",
				Type:     contracts.HouseTypeSingle,
				Members:  165,
				TermDays: 5 * 365,
			},
		},
	}
}

// SenegalSampleBill is a single seed Bill record sourced from public Assemblée
// Nationale du Sénégal records. Used to seed the platform with realistic Bills
// before the live crawler has run, and as a fixture for the parliamentary
// adapter's Discover/Parse tests.
type SenegalSampleBill struct {
	// Title is the human-readable Bill title as published on
	// assemblee-nationale.sn.
	Title string
	// Number is the official Bill number, e.g., "Projet de loi n° 12/2024".
	Number string
	// Sponsor is the Bill's sponsor (usually a Minister).
	Sponsor string
	// Stage is the canonical Senegal stage code (see SenegalBillStages).
	Stage string
	// SourceURL is the canonical URL of the Bill on assemblee-nationale.sn.
	SourceURL string
}

// SenegalSampleBills is a curated set of 5 realistic National Assembly of
// Senegal Bills sourced from public assemblee-nationale.sn records. Titles,
// numbers, sponsors, and stages reflect Bills that have been before the 15th
// Législature (2022–2027); the SourceURLs follow the canonical /lois/<slug>
// pattern used by the Assemblée Nationale du Sénégal website.
//
// These records are SEED DATA ONLY — they are not a live feed. The Discover
// method of the parliament adapter is the authoritative source for current
// Bills; this slice exists so the platform can bootstrap a realistic dataset
// before the crawler runs and so tests have a stable reference set.
var SenegalSampleBills = []SenegalSampleBill{
	{
		Title:     "Loi de finances 2024",
		Number:    "Projet de loi n° 12/2023",
		Sponsor:   "Hon. Ministre de l'Économie et des Finances",
		Stage:     "ADOPTION",
		SourceURL: "https://www.assemblee-nationale.sn/lois/loi-de-finances-2024",
	},
	{
		Title:     "Loi sur la protection des données",
		Number:    "Projet de loi n° 18/2023",
		Sponsor:   "Hon. Ministre de la Communication, des Télécommunications et du Numérique",
		Stage:     "SECOND_READING",
		SourceURL: "https://www.assemblee-nationale.sn/lois/protection-donnees-2023",
	},
	{
		Title:     "Loi d'orientation de l'Éducation nationale",
		Number:    "Projet de loi n° 05/2024",
		Sponsor:   "Hon. Ministre de l'Éducation nationale",
		Stage:     "FIRST_READING",
		SourceURL: "https://www.assemblee-nationale.sn/lois/orientation-education-nationale-2024",
	},
	{
		Title:     "Loi portant Code minier",
		Number:    "Projet de loi n° 22/2024",
		Sponsor:   "Hon. Ministre des Mines et de la Géologie",
		Stage:     "COMMISSION",
		SourceURL: "https://www.assemblee-nationale.sn/lois/code-minier-2024",
	},
	{
		Title:     "Loi sur la cybersécurité",
		Number:    "Projet de loi n° 08/2023",
		Sponsor:   "Hon. Ministre de la Communication, des Télécommunications et du Numérique",
		Stage:     "PROMULGATION",
		SourceURL: "https://www.assemblee-nationale.sn/lois/cybersecurite-2023",
	},
}

// FindStage looks up a Senegal stage by its code. Returns nil if not found.
func FindStage(code string) *contracts.StageDefinition {
	for i := range SenegalBillStages {
		if SenegalBillStages[i].Code == code {
			return &SenegalBillStages[i]
		}
	}
	return nil
}

// IsTerminal reports whether the given stage is a terminal state of the
// Senegal Bill lifecycle (no allowed transitions).
func IsTerminal(code string) bool {
	s := FindStage(code)
	if s == nil {
		return false
	}
	return s.IsTerminal
}
