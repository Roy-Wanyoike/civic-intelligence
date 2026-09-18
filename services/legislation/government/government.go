// Package government contains the domain model for constitution, government,
// presidential terms, and government periods.
//
// Design rules (Constitution + Government spec, sections 30-31):
//   1. No Kenyan president is hard-coded into application logic. Presidents,
//      administrations, and terms are database entities.
//   2. The architecture supports any country: Presidential, Parliamentary,
//      Semi-Presidential, Monarchical, Transitional.
//   3. Historical truth is never overwritten. Government transitions are
//      first-class events.
//   4. The Constitution is treated as a first-class civic domain, not
//      merely another document.
package government

import "time"

// ID is the platform-wide identifier type.
type ID string

// CountryCode is ISO 3166-1 alpha-2.
type CountryCode string

// GovernmentSystem classifies a country's executive structure. Spec §31.
type GovernmentSystem string

const (
        GovernmentSystemPresidential      GovernmentSystem = "PRESIDENTIAL"
        GovernmentSystemParliamentary     GovernmentSystem = "PARLIAMENTARY"
        GovernmentSystemSemiPresidential  GovernmentSystem = "SEMI_PRESIDENTIAL"
        GovernmentSystemMonarchical       GovernmentSystem = "MONARCHICAL"
        GovernmentSystemTransitional      GovernmentSystem = "TRANSITIONAL"
        GovernmentSystemOther             GovernmentSystem = "OTHER"
)

// Constitution is the first-class civic domain. Spec §1.
type Constitution struct {
        ID            ID              `json:"id"`
        CountryCode   CountryCode     `json:"country_code"`
        Title         string          `json:"title"`
        PromulgatedAt time.Time      `json:"promulgated_at"`
        AssentedAt    *time.Time      `json:"assented_at"`
        Version       string          `json:"version"`
        SourceURL     string          `json:"source_url"`
        Chapters      []ConstitutionChapter `json:"chapters"`
        CreatedAt     time.Time      `json:"created_at"`
        UpdatedAt     time.Time      `json:"updated_at"`
}

// ConstitutionChapter groups related articles. Spec §1.
type ConstitutionChapter struct {
        ID             ID                     `json:"id"`
        ConstitutionID ID                     `json:"constitution_id"`
        Number         int                    `json:"number"`
        Title          string                 `json:"title"`
        Articles       []ConstitutionArticle  `json:"articles"`
}

// ConstitutionArticle is a single article of the constitution. Spec §1.
type ConstitutionArticle struct {
        ID              ID                            `json:"id"`
        ChapterID       ID                            `json:"chapter_id"`
        Number          string                        `json:"number"` // e.g. "Article 1", "Article 10"
        Title           string                        `json:"title"`
        Text            string                        `json:"text"`
        SourceURL       string                        `json:"source_url"`
        CrossReferences []ConstitutionCrossReference `json:"cross_references"`
}

// ConstitutionCrossReferenceType classifies a cross-reference. Spec §6.
type ConstitutionCrossReferenceType string

const (
        CrossRefExplicit    ConstitutionCrossReferenceType = "EXPLICIT"    // Bill cites Article X
        CrossRefInferred   ConstitutionCrossReferenceType = "INFERRED"    // system identifies possible relationship
        CrossRefJudicial   ConstitutionCrossReferenceType = "JUDICIAL"     // court-confirmed interpretation
        CrossRefUnknown     ConstitutionCrossReferenceType = "UNKNOWN"     // no reliable relationship
)

// ConstitutionCrossReference links a constitutional article to another civic
// entity (Bill, Act, Court Decision, Institution). Spec §6.
type ConstitutionCrossReference struct {
        ID             ID
        ArticleID      ID
        TargetType     string // "BILL", "ACT", "REGULATION", "COURT_DECISION", "INSTITUTION"
        TargetID       ID
        ReferenceType  ConstitutionCrossReferenceType
        SourceURL      string
        Notes          string
        CreatedAt      time.Time
}

// President is the head of state (or head of government in some systems).
// Spec §7.
type President struct {
        ID           ID         `json:"id"`
        CountryCode  CountryCode `json:"country_code"`
        FullName     string     `json:"full_name"`
        DisplayName  string     `json:"display_name"`
        BornAt       *time.Time `json:"born_at"`
        BiographyURL string     `json:"biography_url"`
        PhotoURL     string     `json:"photo_url"`
        CreatedAt    time.Time  `json:"created_at"`
}

// Administration is a presidential administration. Spec §7.
type Administration struct {
        ID               ID              `json:"id"`
        CountryCode      CountryCode      `json:"country_code"`
        PresidentID      ID              `json:"president_id"`
        Name             string          `json:"name"`
        StartDate        time.Time       `json:"start_date"`
        EndDate          *time.Time      `json:"end_date"`
        GovernmentSystem GovernmentSystem `json:"government_system"`
        SourceURL        string          `json:"source_url"`
        CreatedAt        time.Time       `json:"created_at"`
}

// PresidentialTerm is a single term of a president. Spec §8.
//
// A president may have multiple terms. The data model supports N terms —
// it does NOT assume every administration has exactly two terms.
type PresidentialTerm struct {
        ID               ID         `json:"id"`
        AdministrationID ID         `json:"administration_id"`
        PresidentID      ID         `json:"president_id"`
        TermNumber       int        `json:"term_number"`
        ElectionDate     *time.Time `json:"election_date"`
        SwearingInDate   *time.Time `json:"swearing_in_date"`
        StartDate        time.Time  `json:"start_date"`
        EndDate          *time.Time `json:"end_date"`
        Status           string     `json:"status"`
        SourceURL        string     `json:"source_url"`
        CreatedAt        time.Time  `json:"created_at"`
}

// GovernmentPeriod represents the operational period of a government.
// Spec §9. Combines administration + parliament + cabinet.
type GovernmentPeriod struct {
        ID              ID
        AdministrationID ID
        PresidentialTermID *ID
        ParliamentID    *ID
        StartDate       time.Time
        EndDate         *time.Time
        Cabinet         []CabinetMember
        CreatedAt       time.Time
}

// CabinetMember is a member of the cabinet for a government period.
type CabinetMember struct {
        ID              ID
        GovernmentPeriodID ID
        PersonID        ID
        Role            string // e.g. "Cabinet Secretary for Finance"
        Ministry        string
        StartDate       time.Time
        EndDate         *time.Time
        SourceURL       string
}

// GovernmentTransition is a first-class civic event. Spec §28.
type GovernmentTransition struct {
        ID                   ID         `json:"id"`
        CountryCode          CountryCode `json:"country_code"`
        OutgoingAdminID      *ID        `json:"outgoing_admin_id"`
        IncomingAdminID      ID         `json:"incoming_admin_id"`
        TransitionDate       time.Time  `json:"transition_date"`
        OutgoingPresidentID  *ID        `json:"outgoing_president_id"`
        IncomingPresidentID  ID         `json:"incoming_president_id"`
        SourceURL            string     `json:"source_url"`
        Notes                string     `json:"notes"`
        CreatedAt            time.Time  `json:"created_at"`
}
