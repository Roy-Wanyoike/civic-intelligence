// Package legislation — post-assent legislative lifecycle domain.
//
// Phase: Post-Assent Legislative Lifecycle spec.
//
// This file extends the Bill lifecycle BEYOND Presidential Assent. The full
// lifecycle is:
//
//      Bill → Introduction → First Reading → Committee → Public Participation →
//      Second Reading → Committee of Whole House → Third Reading →
//      Other House → Concurrence → Presidential Assent → ACT →
//      Commencement → Regulations → Administrative Actions →
//      Judicial Challenges → Judicial Decisions → Amendments → Repeal/Replacement
//
// CRITICAL INVARIANT: Presidential assent is NOT the end of the lifecycle.
// The platform tracks every post-assent event with evidence.
package domain

import (
        "context"
        "time"
)

// PresidentialAssentStatus records the outcome of a presidential assent
// event. Spec section 14.
type PresidentialAssentStatus string

const (
        AssentStatusAssented PresidentialAssentStatus = "ASSENTED"
        AssentStatusReturned PresidentialAssentStatus = "RETURNED"
        AssentStatusWithheld PresidentialAssentStatus = "WITHHELD"
        AssentStatusUnknown  PresidentialAssentStatus = "UNKNOWN"
)

// PresidentialAssentEvent is a first-class event representing the
// president's action on a Bill. Spec section 14.
//
// This replaces the anti-pattern of mutating `bill.status = PASSED`
// without preserving the historical event.
type PresidentialAssentEvent struct {
        ID                  ID
        BillID              ID
        PresidentID         ID
        AdministrationID    ID
        PresidentialTermID  *ID
        Date                time.Time
        OfficialSource      string
        DocumentID          *ID
        Evidence            []EvidenceRef
        AssentStatus        PresidentialAssentStatus
        CreatedAt           time.Time
}

// ActStatus is the lifecycle state of an Act of Parliament.
type ActStatus string

const (
        ActStatusAssented      ActStatus = "ASSENTED"
        ActStatusPublished     ActStatus = "PUBLISHED"
        ActStatusCommenced     ActStatus = "COMMENCED"
        ActStatusAmended       ActStatus = "AMENDED"
        ActStatusRepealed      ActStatus = "REPEALED"
        ActStatusUnknown       ActStatus = "UNKNOWN"
)

// ActVersion is an immutable snapshot of an Act at a point in time.
// Spec section 20. Used for amendments — the historical text is preserved.
type ActVersion struct {
        ID              ID
        ActID           ID
        Version         int
        Text            string
        AmendmentID     *ID // if this version is the result of an amendment
        EffectiveFrom   time.Time
        EffectiveUntil  *time.Time
        SourceURL       string
        CreatedAt       time.Time
}

// ActAuditStatus tracks the platform's coverage of an Act's post-assent
// lifecycle. Spec section 18.
//
// IMPORTANT: "No authoritative record found" is NOT the same as "nothing
// happened". The platform distinguishes NOT_VERIFIED from negative findings.
type ActAuditStatus string

const (
        ActAuditAssentConfirmed         ActAuditStatus = "ASSENT_CONFIRMED"
        ActAuditPublicationConfirmed    ActAuditStatus = "PUBLICATION_CONFIRMED"
        ActAuditCommencementConfirmed   ActAuditStatus = "COMMENCEMENT_CONFIRMED"
        ActAuditImplementationTracked   ActAuditStatus = "IMPLEMENTATION_TRACKED"
        ActAuditRegulationsTracked      ActAuditStatus = "REGULATIONS_TRACKED"
        ActAuditJudicialHistoryTracked  ActAuditStatus = "JUDICIAL_HISTORY_TRACKED"
        ActAuditAmendmentsTracked       ActAuditStatus = "AMENDMENTS_TRACKED"
        ActAuditRepealStatusTracked     ActAuditStatus = "REPEAL_STATUS_TRACKED"
        ActAuditComplete                ActAuditStatus = "AUDIT_COMPLETE"
        ActAuditPartiallyTracked        ActAuditStatus = "PARTIALLY_TRACKED"
        ActAuditDataGap                 ActAuditStatus = "DATA_GAP"
        ActAuditConflictingSources      ActAuditStatus = "CONFLICTING_SOURCES"
        ActAuditNotVerified             ActAuditStatus = "NOT_VERIFIED"
)

// LegislativeLifecycleAudit aggregates the audit status of an Act across
// every post-assent dimension. Spec section 16.
type LegislativeLifecycleAudit struct {
        ActID                ID
        AssentRecorded       bool
        ActPublished         bool
        ActNumber            string
        CommencementNotice   bool
        RegulationsRequired  bool
        RegulationsIssued    bool
        InstitutionalAction  bool
        ImplementationDocs   bool
        Amended              bool
        CourtChallenged      bool
        JudicialDecision     bool
        Repealed             bool
        CurrentStatus        ActStatus
        AuditStatuses        []ActAuditStatus
        DataGaps             []string
        LastReviewedAt       time.Time
}

// AuditForAct inspects an Act and returns its lifecycle audit. Spec
// section 16.
//
// "No commencement notice found" does NOT mean "the Act never commenced".
// Instead, the audit returns ActAuditNotVerified for commencement.
func AuditForAct(act Act, events []PostAssentEvent) LegislativeLifecycleAudit {
        out := LegislativeLifecycleAudit{
                ActID:          act.ID,
                ActNumber:      act.ActNumber,
                CurrentStatus:  act.Status,
        }
        if act.AssentedAt.IsZero() {
                out.DataGaps = append(out.DataGaps, "assent_date_missing")
        } else {
                out.AssentRecorded = true
                out.AuditStatuses = append(out.AuditStatuses, ActAuditAssentConfirmed)
        }
        if act.PublicationDate != nil {
                out.ActPublished = true
                out.AuditStatuses = append(out.AuditStatuses, ActAuditPublicationConfirmed)
        } else {
                out.DataGaps = append(out.DataGaps, "publication_date_missing")
        }
        if act.CommencementDate != nil {
                out.CommencementNotice = true
                out.AuditStatuses = append(out.AuditStatuses, ActAuditCommencementConfirmed)
        } else {
                out.AuditStatuses = append(out.AuditStatuses, ActAuditNotVerified)
                out.DataGaps = append(out.DataGaps, "commencement_notice_not_found")
        }

        for _, e := range events {
                switch e.EventType {
                case PostAssentEventRegulation:
                        out.RegulationsIssued = true
                case PostAssentEventImplementation:
                        out.ImplementationDocs = true
                case PostAssentEventCourtChallenge:
                        out.CourtChallenged = true
                case PostAssentEventJudicialDecision:
                        out.JudicialDecision = true
                case PostAssentEventAmendment:
                        out.Amended = true
                case PostAssentEventRepeal:
                        out.Repealed = true
                }
        }

        if out.RegulationsIssued {
                out.AuditStatuses = append(out.AuditStatuses, ActAuditRegulationsTracked)
        }
        if out.JudicialDecision {
                out.AuditStatuses = append(out.AuditStatuses, ActAuditJudicialHistoryTracked)
        }
        if out.Amended {
                out.AuditStatuses = append(out.AuditStatuses, ActAuditAmendmentsTracked)
        }
        if out.Repealed {
                out.AuditStatuses = append(out.AuditStatuses, ActAuditRepealStatusTracked)
        }

        if len(out.DataGaps) > 0 {
                out.AuditStatuses = append(out.AuditStatuses, ActAuditPartiallyTracked, ActAuditDataGap)
        } else if out.AssentRecorded && out.ActPublished && out.CommencementNotice {
                out.AuditStatuses = append(out.AuditStatuses, ActAuditComplete)
        }
        return out
}

// PostAssentEventType enumerates the kinds of events tracked after a Bill
// becomes an Act.
type PostAssentEventType string

const (
        PostAssentEventAssent           PostAssentEventType = "ASSENT"
        PostAssentEventPublication       PostAssentEventType = "PUBLICATION"
        PostAssentEventCommencement      PostAssentEventType = "COMMENCEMENT"
        PostAssentEventRegulation        PostAssentEventType = "REGULATION"
        PostAssentEventImplementation    PostAssentEventType = "IMPLEMENTATION"
        PostAssentEventCourtChallenge    PostAssentEventType = "COURT_CHALLENGE"
        PostAssentEventJudicialDecision  PostAssentEventType = "JUDICIAL_DECISION"
        PostAssentEventAmendment         PostAssentEventType = "AMENDMENT"
        PostAssentEventRepeal            PostAssentEventType = "REPEAL"
)

// PostAssentEvent is a single event in the post-assent lifecycle of an Act.
type PostAssentEvent struct {
        ID           ID
        ActID        ID
        BillID       *ID
        EventType    PostAssentEventType
        EventDate    time.Time
        Title        string
        Description  string
        SourceURL    string
        DocumentID   *ID
        Evidence     []EvidenceRef
        CreatedAt    time.Time
}

// EvidenceRef is a reference to a verified civic fact or document.
// (Defined here to avoid an import cycle with the simulation domain.)
type EvidenceRef struct {
        Kind     string
        ID       string
        SourceURL string
}

// ActFilter is a query filter for acts.
type ActFilter struct {
        CountryID    *ID
        Status       []ActStatus
        Limit        int
        Offset       int
}

// ActRepository persists Acts and their post-assent events.
type ActRepository interface {
        CreateAct(ctx context.Context, a Act) error
        GetAct(ctx context.Context, id ID) (*Act, error)
        ListActs(ctx context.Context, filter ActFilter) ([]Act, error)
        UpdateAct(ctx context.Context, a Act) error

        AppendVersion(ctx context.Context, v ActVersion) error
        ListVersions(ctx context.Context, actID ID) ([]ActVersion, error)

        AppendPostAssentEvent(ctx context.Context, e PostAssentEvent) error
        ListPostAssentEvents(ctx context.Context, actID ID) ([]PostAssentEvent, error)

        RecordAssent(ctx context.Context, ev PresidentialAssentEvent) (*Act, error)
}
