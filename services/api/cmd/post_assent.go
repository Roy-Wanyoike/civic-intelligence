// Package main — post-assent legislative lifecycle API endpoints.
//
// Endpoints:
//
//	GET  /api/v1/acts/{id}/audit     -- full lifecycle audit for an Act
//	GET  /api/v1/acts/{id}/events    -- post-assent events (commencement, regulations, courts, amendments)
//	POST /api/v1/acts/{id}/follow    -- follow a law (issue #193 flagship experience)
//	GET  /api/v1/acts/{id}/lineage   -- full legal lineage (Bill → Act → Amendments → Regulations → Court Decisions)
//
// Invariants (issues #215, #216, #217):
//
//   - #215: The audit endpoint delegates to domain.AuditForAct so the audit
//     actually inspects the Act's fields. An act without a commencement date
//     is reported as NOT_VERIFIED, never as "commenced".
//   - #216: The follow endpoint creates a real subscription in the
//     SubscriptionStore. No stub responses.
//   - #217: The lineage endpoint is data-driven: every step is sourced from
//     the Act's fields or a matching post-assent event. Steps with no
//     evidence are flagged NOT_VERIFIED, never inferred.
package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/postassent"
)

// PostAssentEventResponse is the JSON representation of a post-assent event.
type PostAssentEventResponse struct {
	ID          string    `json:"id"`
	ActID       string    `json:"act_id"`
	EventType   string    `json:"event_type"`
	EventDate   time.Time `json:"event_date"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	SourceURL   string    `json:"source_url"`
}

// ActAuditResponse is the JSON representation of a full lifecycle audit.
type ActAuditResponse struct {
	ActID              string   `json:"act_id"`
	AssentRecorded     bool     `json:"assent_recorded"`
	ActPublished       bool     `json:"act_published"`
	ActNumber          string   `json:"act_number"`
	CommencementNotice bool     `json:"commencement_notice"`
	RegulationsIssued  bool     `json:"regulations_issued"`
	Amended            bool     `json:"amended"`
	CourtChallenged    bool     `json:"court_challenged"`
	JudicialDecision   bool     `json:"judicial_decision"`
	Repealed           bool     `json:"repealed"`
	CurrentStatus      string   `json:"current_status"`
	AuditStatuses      []string `json:"audit_statuses"`
	DataGaps           []string `json:"data_gaps"`
	Disclaimer         string   `json:"disclaimer"`
}

// samplePostAssentEvents is a placeholder for the in-memory store. In
// production these come from the legislation service's ActRepository.
var samplePostAssentEvents = map[string][]PostAssentEventResponse{
	"ke-act-data-protection-2019": {
		{
			ID:          "ev-dpa-1",
			ActID:       "ke-act-data-protection-2019",
			EventType:   "COMMENCEMENT",
			EventDate:   time.Date(2019, 11, 25, 0, 0, 0, 0, time.UTC),
			Title:       "Commencement Notice",
			Description: "The Data Protection Act, 2019 commenced on 25 November 2019.",
			SourceURL:   "https://www.kenyalaw.org/kl/index.php?id=4639",
		},
		{
			ID:          "ev-dpa-2",
			ActID:       "ke-act-data-protection-2019",
			EventType:   "REGULATION",
			EventDate:   time.Date(2021, 2, 12, 0, 0, 0, 0, time.UTC),
			Title:       "Data Protection (General) Regulations, 2021",
			Description: "Regulations issued under section 71 of the Act.",
			SourceURL:   "https://www.kenyalaw.org/kl/index.php?id=10675",
		},
	},
}

// followLawMonitors is the fixed set of event categories a "Follow a Law"
// subscription monitors. Spec section 41. These mirror the post-assent event
// taxonomy the platform tracks for every Act.
var followLawMonitors = []string{
	"COMMENCEMENT",
	"REGULATIONS",
	"IMPLEMENTATION",
	"AMENDMENTS",
	"COURT_CASES",
	"GOVERNMENT_NOTICES",
	"INSTITUTIONAL_ACTIONS",
	"RELATED_BILLS",
}

// makeActRouter routes /api/v1/acts/{id} and sub-resources to the
// appropriate handler. This is needed because Go's http.ServeMux doesn't
// support multiple handlers on the same path prefix.
//
// The store is the in-memory SubscriptionStore that the follow endpoint
// writes to (issue #216). Other endpoints (audit, events, lineage) ignore it.
func makeActRouter(store *SubscriptionStore) http.HandlerFunc {
	auditH := makeActAuditHandler()
	eventsH := makeActEventsHandler()
	followH := makeFollowLawHandler(store)
	lineageH := makeLineageHandler()
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/acts/")
		// path is now: {id} or {id}/audit etc.
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 1 {
			// Just /acts/{id}
			handleActDetail(w, r)
			return
		}
		sub := strings.TrimSuffix(parts[1], "/")
		switch sub {
		case "audit":
			auditH(w, r)
		case "events":
			eventsH(w, r)
		case "follow":
			followH(w, r)
		case "lineage":
			lineageH(w, r)
		default:
			writeError(w, http.StatusNotFound, "not_found", "unknown sub-resource: "+sub)
		}
	}
}

// makeActAuditHandler handles GET /api/v1/acts/{id}/audit.
//
// Returns the full lifecycle audit for an Act. Spec section 16, 43.
//
// CRITICAL: missing commencement notice is reported as NOT_VERIFIED, NOT
// as "never commenced". The audit delegates to domain.AuditForAct so the
// act's fields (assent date, publication date, commencement date) drive the
// status — the API never hardcodes "everything is confirmed" (issue #215).
func makeActAuditHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/acts/")
		id := strings.TrimSuffix(path, "/audit")
		// Find the act in sampleActs.
		var found *actResponse
		for i := range sampleActs {
			if sampleActs[i].ID == id {
				found = &sampleActs[i]
				break
			}
		}
		if found == nil {
			writeError(w, http.StatusNotFound, "not_found", "act not found: "+id)
			return
		}
		events := samplePostAssentEvents[id]
		audit := buildAudit(*found, events)
		writeJSON(w, http.StatusOK, audit)
	}
}

// makeActEventsHandler handles GET /api/v1/acts/{id}/events.
func makeActEventsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/acts/")
		id := strings.TrimSuffix(path, "/events")
		events := samplePostAssentEvents[id]
		if events == nil {
			events = []PostAssentEventResponse{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"act_id":     id,
			"events":     events,
			"count":      len(events),
			"disclaimer": "Events sourced from authoritative material (Kenya Law, Kenya Gazette).",
		})
	}
}

// makeFollowLawHandler handles POST /api/v1/acts/{id}/follow.
//
// Spec section 41 — flagship "Follow a Law" experience. The user follows
// an Act and receives notifications when authoritative evidence indicates
// something changed (commencement, regulations, amendments, court cases,
// institutional actions, related Bills).
//
// Issue #216: this handler now creates a REAL subscription in the
// SubscriptionStore. The persisted record carries:
//
//   - target = "act:" + actID
//   - monitors = the 8 followLawMonitors categories
//   - the calling user's ID (from the auth middleware)
//
// The response includes the subscription_id so the frontend can later
// unsubscribe or inspect the subscription.
func makeFollowLawHandler(store *SubscriptionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		p := middleware.PrincipalFromRequest(r)
		if p.IsAnonymous() {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required to follow a law")
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/acts/")
		id := strings.TrimSuffix(path, "/follow")
		if id == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "act ID required")
			return
		}

		// Persist the subscription. The store is idempotent on
		// (user_id, target) so a double-follow returns the same ID.
		monitors := append([]string(nil), followLawMonitors...)
		target := "act:" + id
		sub, err := store.CreateSubscription(p.UserID, target, monitors)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "subscribe_failed", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, http.StatusOK, map[string]any{
			"followed":        true,
			"subscription_id": sub.ID,
			"act_id":          id,
			"target":          sub.Target,
			"monitors":        sub.Monitors,
			"disclaimer":      "You will be notified when authoritative evidence indicates this Act changed. The platform does not fabricate notifications.",
		})
	}
}

// makeLineageHandler handles GET /api/v1/acts/{id}/lineage.
//
// Spec section 20 — full legal lineage:
// Bill → Bill Version → Amendment → Passed Version → Assent → Act → Act
// Version → Amendment → Regulation → Judicial Interpretation → Repeal/Replacement.
//
// Issue #217: this handler is data-driven. For each lineage step, it looks
// for evidence in the Act's fields or in the post-assent event list. If
// evidence exists, the step is reported as VERIFIED with the date,
// description, and source URL. If no evidence exists, the step is reported
// as NOT_VERIFIED with the description "No authoritative record found
// yet." — the platform never infers that nothing happened.
func makeLineageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/acts/")
		id := strings.TrimSuffix(path, "/lineage")
		var found *actResponse
		for i := range sampleActs {
			if sampleActs[i].ID == id {
				found = &sampleActs[i]
				break
			}
		}
		if found == nil {
			writeError(w, http.StatusNotFound, "not_found", "act not found: "+id)
			return
		}
		events := samplePostAssentEvents[id]
		lineage := buildLineage(*found, events)
		writeJSON(w, http.StatusOK, map[string]any{
			"act_id":     id,
			"lineage":    lineage,
			"disclaimer": "Each step is independently sourced. Missing steps are flagged NOT_VERIFIED, not inferred.",
		})
	}
}

// lineageStep is one row in the lineage response.
type lineageStep struct {
	Step        string `json:"step"`
	Title       string `json:"title"`
	Status      string `json:"status"` // "VERIFIED" or "NOT_VERIFIED"
	Date        string `json:"date,omitempty"`
	Description string `json:"description"`
	SourceURL   string `json:"source_url,omitempty"`
}

const lineageNotVerifiedDescription = "No authoritative record found yet."

// buildLineage constructs a data-driven lineage slice for the given Act +
// events. Each step is sourced from the act's fields or the matching
// post-assent event; steps without evidence are NOT_VERIFIED.
func buildLineage(act actResponse, events []PostAssentEventResponse) []lineageStep {
	steps := make([]lineageStep, 0, 9)

	// origin_bill — we don't track the originating Bill on the Act (yet).
	steps = append(steps, lineageStep{
		Step:        "origin_bill",
		Title:       act.Title + " (origin Bill)",
		Status:      "NOT_VERIFIED",
		Description: lineageNotVerifiedDescription,
	})

	// parliamentary_journey — bill-stage data, not tracked on the Act.
	steps = append(steps, lineageStep{
		Step:        "parliamentary_journey",
		Title:       "Parliamentary journey",
		Status:      "NOT_VERIFIED",
		Description: lineageNotVerifiedDescription,
	})

	// presidential_assent — from the Act's AssentDate field.
	if act.AssentDate != "" {
		steps = append(steps, lineageStep{
			Step:        "presidential_assent",
			Title:       "Presidential Assent",
			Status:      "VERIFIED",
			Date:        act.AssentDate,
			Description: "The President assented to the Bill on " + act.AssentDate + ", creating the Act.",
			SourceURL:   act.SourceURL,
		})
	} else {
		steps = append(steps, lineageStep{
			Step:        "presidential_assent",
			Title:       "Presidential Assent",
			Status:      "NOT_VERIFIED",
			Description: lineageNotVerifiedDescription,
		})
	}

	// act_publication — would come from a PUBLICATION event or
	// PublicationDate field (not yet on actResponse). The platform does
	// NOT infer a publication date from assent or commencement.
	if pub := findEvent(events, "PUBLICATION"); pub != nil {
		steps = append(steps, lineageStep{
			Step:        "act_publication",
			Title:       "Act publication",
			Status:      "VERIFIED",
			Date:        pub.EventDate.Format("2006-01-02"),
			Description: pub.Description,
			SourceURL:   pub.SourceURL,
		})
	} else {
		steps = append(steps, lineageStep{
			Step:        "act_publication",
			Title:       "Act publication",
			Status:      "NOT_VERIFIED",
			Description: lineageNotVerifiedDescription,
		})
	}

	// commencement — prefer the explicit COMMENCEMENT event; fall back to
	// the Act's CommencementDate field (which itself comes from the
	// commencement notice on kenyalaw.org).
	if comm := findEvent(events, "COMMENCEMENT"); comm != nil {
		steps = append(steps, lineageStep{
			Step:        "commencement",
			Title:       "Commencement",
			Status:      "VERIFIED",
			Date:        comm.EventDate.Format("2006-01-02"),
			Description: comm.Description,
			SourceURL:   comm.SourceURL,
		})
	} else if act.CommencementDate != "" {
		steps = append(steps, lineageStep{
			Step:        "commencement",
			Title:       "Commencement",
			Status:      "VERIFIED",
			Date:        act.CommencementDate,
			Description: "The Act commenced on " + act.CommencementDate + " per the commencement notice.",
			SourceURL:   act.SourceURL,
		})
	} else {
		steps = append(steps, lineageStep{
			Step:        "commencement",
			Title:       "Commencement",
			Status:      "NOT_VERIFIED",
			Description: lineageNotVerifiedDescription,
		})
	}

	// regulations — from the most recent REGULATION event.
	if reg := findLatestEvent(events, "REGULATION"); reg != nil {
		steps = append(steps, lineageStep{
			Step:        "regulations",
			Title:       "Regulations",
			Status:      "VERIFIED",
			Date:        reg.EventDate.Format("2006-01-02"),
			Description: reg.Description,
			SourceURL:   reg.SourceURL,
		})
	} else {
		steps = append(steps, lineageStep{
			Step:        "regulations",
			Title:       "Regulations",
			Status:      "NOT_VERIFIED",
			Description: lineageNotVerifiedDescription,
		})
	}

	// amendments — from the most recent AMENDMENT event.
	if amm := findLatestEvent(events, "AMENDMENT"); amm != nil {
		steps = append(steps, lineageStep{
			Step:        "amendments",
			Title:       "Amendments",
			Status:      "VERIFIED",
			Date:        amm.EventDate.Format("2006-01-02"),
			Description: amm.Description,
			SourceURL:   amm.SourceURL,
		})
	} else {
		steps = append(steps, lineageStep{
			Step:        "amendments",
			Title:       "Amendments",
			Status:      "NOT_VERIFIED",
			Description: lineageNotVerifiedDescription,
		})
	}

	// court_decisions — from COURT_CHALLENGE or JUDICIAL_DECISION events.
	if cd := findLatestEvent(events, "COURT_CHALLENGE", "JUDICIAL_DECISION"); cd != nil {
		steps = append(steps, lineageStep{
			Step:        "court_decisions",
			Title:       "Court decisions",
			Status:      "VERIFIED",
			Date:        cd.EventDate.Format("2006-01-02"),
			Description: cd.Description,
			SourceURL:   cd.SourceURL,
		})
	} else {
		steps = append(steps, lineageStep{
			Step:        "court_decisions",
			Title:       "Court decisions",
			Status:      "NOT_VERIFIED",
			Description: lineageNotVerifiedDescription,
		})
	}

	// current_status — derived from the Act's Status field (always present
	// in the sample data; in production this comes from the canonical Act
	// record).
	steps = append(steps, lineageStep{
		Step:        "current_status",
		Title:       "Current status",
		Status:      "VERIFIED",
		Description: "The current legal status of the Act is: " + act.Status + ".",
		SourceURL:   act.SourceURL,
	})

	return steps
}

// findEvent returns the first event of any of the given types, or nil.
func findEvent(events []PostAssentEventResponse, types ...string) *PostAssentEventResponse {
	for i := range events {
		for _, t := range types {
			if events[i].EventType == t {
				return &events[i]
			}
		}
	}
	return nil
}

// findLatestEvent returns the most recent event of any of the given types
// (sorted by EventDate), or nil. Used for steps like "regulations" /
// "amendments" / "court_decisions" where multiple events may exist and the
// lineage shows the latest.
func findLatestEvent(events []PostAssentEventResponse, types ...string) *PostAssentEventResponse {
	var best *PostAssentEventResponse
	for i := range events {
		matches := false
		for _, t := range types {
			if events[i].EventType == t {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}
		if best == nil || events[i].EventDate.After(best.EventDate) {
			best = &events[i]
		}
	}
	return best
}

// buildAudit constructs an ActAuditResponse by delegating to
// domain.AuditForAct (issue #215). The actResponse + PostAssentEventResponse
// shapes are first adapted to the domain types; the resulting audit is
// then mapped back to the JSON response shape.
//
// CRITICAL: This used to hardcode "everything is confirmed". It now reflects
// the actual state of the Act's fields — an act without a commencement date
// reports NOT_VERIFIED for commencement, not "commenced".
func buildAudit(act actResponse, events []PostAssentEventResponse) ActAuditResponse {
	domainAct := actResponseToDomain(act)
	domainEvents := eventsResponseToDomain(events)
	audit := postassent.AuditForAct(domainAct, domainEvents)

	out := ActAuditResponse{
		ActID:              string(audit.ActID),
		AssentRecorded:     audit.AssentRecorded,
		ActPublished:       audit.ActPublished,
		ActNumber:          audit.ActNumber,
		CommencementNotice: audit.CommencementNotice,
		RegulationsIssued:  audit.RegulationsIssued,
		Amended:            audit.Amended,
		CourtChallenged:    audit.CourtChallenged,
		JudicialDecision:   audit.JudicialDecision,
		Repealed:           audit.Repealed,
		CurrentStatus:      string(audit.CurrentStatus),
		AuditStatuses:      make([]string, 0, len(audit.AuditStatuses)),
		DataGaps:           audit.DataGaps,
		Disclaimer:         "Missing records are reported as NOT_VERIFIED, never as 'never happened'. The platform does not infer absence.",
	}
	if out.DataGaps == nil {
		out.DataGaps = []string{}
	}
	for _, s := range audit.AuditStatuses {
		out.AuditStatuses = append(out.AuditStatuses, string(s))
	}
	return out
}

// actResponseToDomain maps the API's actResponse (string dates, no
// PublicationDate field) to the domain.Act value object that
// domain.AuditForAct consumes. Empty strings become zero times / nil
// pointers so the audit reports the corresponding data gaps accurately.
//
// NOTE: the actResponse struct doesn't currently carry a publication_date
// field, so PublicationDate is always nil here. This is honest — the sample
// data does not include gazette publication dates, so the audit reports
// "publication_date_missing". When the ActRepository is wired in (ENG-B1),
// the publication date will flow through and the gap will close.
func actResponseToDomain(act actResponse) postassent.Act {
	out := postassent.Act{
		ID:        postassent.ID(act.ID),
		ActNumber: act.Citation,
		ActName:   act.Title,
		SourceURL: act.SourceURL,
	}
	if t, ok := parseDate(act.AssentDate); ok {
		out.AssentedAt = t
	}
	if t, ok := parseDate(act.CommencementDate); ok {
		comm := t
		out.CommencementDate = &comm
	}
	out.Status = mapActStatus(act.Status)
	return out
}

// eventsResponseToDomain maps a slice of API PostAssentEventResponse to the
// domain.PostAssentEvent shape that AuditForAct iterates.
func eventsResponseToDomain(events []PostAssentEventResponse) []postassent.PostAssentEvent {
	out := make([]postassent.PostAssentEvent, 0, len(events))
	for _, e := range events {
		out = append(out, postassent.PostAssentEvent{
			ID:          postassent.ID(e.ID),
			ActID:       postassent.ID(e.ActID),
			EventType:   mapEventType(e.EventType),
			EventDate:   e.EventDate,
			Title:       e.Title,
			Description: e.Description,
			SourceURL:   e.SourceURL,
		})
	}
	return out
}

// mapEventType converts the API string event type to the domain
// PostAssentEventType. Unknown strings map to the empty string and are
// ignored by AuditForAct's switch.
func mapEventType(s string) postassent.PostAssentEventType {
	switch s {
	case "ASSENT":
		return postassent.EventAssent
	case "PUBLICATION":
		return postassent.EventPublication
	case "COMMENCEMENT":
		return postassent.EventCommencement
	case "REGULATION":
		return postassent.EventRegulation
	case "IMPLEMENTATION":
		return postassent.EventImplementation
	case "COURT_CHALLENGE":
		return postassent.EventCourtChallenge
	case "JUDICIAL_DECISION":
		return postassent.EventJudicialDecision
	case "AMENDMENT":
		return postassent.EventAmendment
	case "REPEAL":
		return postassent.EventRepeal
	}
	return ""
}

// mapActStatus converts the API's free-text status string (e.g. "in_force",
// "amended") to the domain ActStatus enum. Unknown values map to
// ActStatusUnknown so the audit surfaces them rather than silently lying.
func mapActStatus(s string) postassent.ActStatus {
	switch s {
	case "in_force":
		return postassent.StatusCommenced
	case "amended":
		return postassent.StatusAmended
	case "repealed":
		return postassent.StatusRepealed
	case "assented":
		return postassent.StatusAssented
	case "published":
		return postassent.StatusPublished
	}
	return postassent.StatusUnknown
}

// parseDate parses a YYYY-MM-DD string into a time.Time. Returns ok=false for
// empty strings or unparseable input; callers treat ok=false as "no date".
func parseDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}
