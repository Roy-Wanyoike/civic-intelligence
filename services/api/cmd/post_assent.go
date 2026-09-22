// Package main — post-assent legislative lifecycle API endpoints.
//
// Endpoints:
//
//      GET  /api/v1/acts/{id}/audit     -- full lifecycle audit for an Act
//      GET  /api/v1/acts/{id}/events    -- post-assent events (commencement, regulations, courts, amendments)
//      POST /api/v1/acts/{id}/follow    -- follow a law (issue #193 flagship experience)
//      GET  /api/v1/acts/{id}/lineage   -- full legal lineage (Bill → Act → Amendments → Regulations → Court Decisions)
package main

import (
        "context"
        "net/http"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
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
        Repealed            bool     `json:"repealed"`
        CurrentStatus      string   `json:"current_status"`
        AuditStatuses      []string `json:"audit_statuses"`
        DataGaps           []string `json:"data_gaps"`
        Disclaimer         string   `json:"disclaimer"`
}

// LineageStep is a single stage in an Act's full legal lineage.
//
// Each step is independently sourced — missing evidence is reported as
// NOT_VERIFIED (never inferred). Spec section 20, 37.
type LineageStep struct {
        Step        string `json:"step"`
        Title       string `json:"title"`
        Description string `json:"description"`
        Status      string `json:"status"` // "VERIFIED" or "NOT_VERIFIED"
        Date        string `json:"date,omitempty"`
        SourceURL   string `json:"source_url,omitempty"`
}

// followLawMonitors is the eight-category notification contract for the
// Follow-a-Law flagship experience (issue #193 / #216). When a user follows
// an Act, the platform will notify them when authoritative evidence
// indicates a change in any of these dimensions. The platform NEVER
// fabricates notifications — each monitor fires only when a verified
// post-assent event is ingested.
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

// samplePostAssentEvents is a cached map built from actRepo at startup.
// It is keyed by Act ID and contains the same events the repository
// exposes via ListPostAssentEvents. The handlers query actRepo directly;
// this map is kept for any legacy callers and as a stable in-process
// snapshot for tests that want a deterministic view of seed events.
var samplePostAssentEvents = buildSamplePostAssentEvents()

func buildSamplePostAssentEvents() map[string][]PostAssentEventResponse {
        acts, err := actRepo.ListActs(context.Background(), legislation.ActFilter{})
        if err != nil {
                return map[string][]PostAssentEventResponse{}
        }
        out := map[string][]PostAssentEventResponse{}
        for _, a := range acts {
                events, _ := actRepo.ListPostAssentEvents(context.Background(), legislation.ID(a.ID))
                resps := make([]PostAssentEventResponse, 0, len(events))
                for _, e := range events {
                        resps = append(resps, toPostAssentEventResponse(e))
                }
                out[string(a.ID)] = resps
        }
        return out
}

// toPostAssentEventResponse converts a domain PostAssentEvent into the JSON
// response shape.
func toPostAssentEventResponse(e legislation.PostAssentEvent) PostAssentEventResponse {
        return PostAssentEventResponse{
                ID:          string(e.ID),
                ActID:       string(e.ActID),
                EventType:   string(e.EventType),
                EventDate:   e.EventDate,
                Title:       e.Title,
                Description: e.Description,
                SourceURL:   e.SourceURL,
        }
}

// actResponseToDomain adapts the API response shape (actResponse) back into
// the legislation domain's Act type. The conversion preserves the
// authoritative fields (assent date, commencement date, citation) so that
// the domain AuditForAct function can compute the lifecycle audit against
// the SAME data the user just saw rendered (issue #215).
//
// Dates are stored as strings ("2006-01-02") in actResponse; we parse them
// back into time.Time so domain.AuditForAct can evaluate IsZero() / nil
// checks. A malformed or empty date is treated as missing — the audit will
// surface it as a data gap, NOT as "the event never happened".
func actResponseToDomain(act actResponse) legislation.Act {
        out := legislation.Act{
                ID:          legislation.ID(act.ID),
                ActNumber:   act.Citation,
                ActName:     act.Title,
                Status:      legislation.ActStatus(act.Status),
                CountryID:   legislation.ID(act.Country),
                Description: act.Summary,
                SourceURL:   act.SourceURL,
        }
        if act.AssentDate != "" {
                if t, err := time.Parse("2006-01-02", act.AssentDate); err == nil {
                        out.AssentedAt = t
                }
        }
        if act.CommencementDate != "" {
                if t, err := time.Parse("2006-01-02", act.CommencementDate); err == nil {
                        out.CommencementDate = &t
                }
        }
        return out
}

// eventsResponseToDomain adapts a slice of PostAssentEventResponse back into
// domain PostAssentEvent values. Used so the domain AuditForAct can consume
// the same events the /events endpoint returned (issue #215). Only the
// fields AuditForAct inspects are populated.
func eventsResponseToDomain(events []PostAssentEventResponse) []legislation.PostAssentEvent {
        out := make([]legislation.PostAssentEvent, 0, len(events))
        for _, e := range events {
                out = append(out, legislation.PostAssentEvent{
                        ID:        legislation.ID(e.ID),
                        ActID:     legislation.ID(e.ActID),
                        EventType: legislation.PostAssentEventType(e.EventType),
                        EventDate: e.EventDate,
                        Title:     e.Title,
                        SourceURL: e.SourceURL,
                })
        }
        return out
}

// makeActRouter routes /api/v1/acts/{id} and sub-resources to the
// appropriate handler. This is needed because Go's http.ServeMux doesn't
// support multiple handlers on the same path prefix.
func makeActRouter() http.HandlerFunc {
        auditH := makeActAuditHandler()
        eventsH := makeActEventsHandler()
        followH := makeFollowLawHandler()
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
// as "never commenced". The audit is computed by domain.AuditForAct (issue
// #215) — the platform NEVER hardcodes "everything is confirmed".
func makeActAuditHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/acts/")
                id := strings.TrimSuffix(path, "/audit")
                // Query the repository for the act.
                a, err := actRepo.GetAct(r.Context(), legislation.ID(id))
                if err != nil {
                        writeError(w, http.StatusNotFound, "not_found", "act not found: "+id)
                        return
                }
                // Query the repository for the act's post-assent events.
                domainEvents, _ := actRepo.ListPostAssentEvents(r.Context(), legislation.ID(id))
                events := make([]PostAssentEventResponse, 0, len(domainEvents))
                for _, e := range domainEvents {
                        events = append(events, toPostAssentEventResponse(e))
                }
                audit := buildAudit(toActResponse(*a), events)
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
                // Query the repository for the act's post-assent events.
                domainEvents, _ := actRepo.ListPostAssentEvents(r.Context(), legislation.ID(id))
                events := make([]PostAssentEventResponse, 0, len(domainEvents))
                for _, e := range domainEvents {
                        events = append(events, toPostAssentEventResponse(e))
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
// Spec section 41 — flagship "Follow a Law" experience (issue #193, #216).
// The user follows an Act and receives notifications when authoritative
// evidence indicates something changed (commencement, regulations,
// amendments, court cases, institutional actions, related Bills).
//
// #216: This handler is NO LONGER a stub. It enforces authentication,
// creates a real subscription record in the shared subscriptionStore with
// target="act:"+actID (entity_type=act, entity_id=actID), and returns the
// real subscription_id so the frontend can later DELETE it via
// /api/v1/subscriptions/{id}. The platform NEVER fabricates a follow —
// if the user is anonymous, the request is rejected with 401.
func makeFollowLawHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                // Authentication is required — follows are per-user records. The
                // /api/v1/* tree is wrapped in middleware.OptionalAuth, so the
                // principal (anonymous or authenticated) is already in the context.
                p := middleware.PrincipalFromRequest(r)
                if p.IsAnonymous() {
                        writeError(w, http.StatusUnauthorized, "unauthorized",
                                "authentication required to follow a law")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/acts/")
                id := strings.TrimSuffix(path, "/follow")
                if id == "" {
                        writeError(w, http.StatusBadRequest, "bad_request", "act ID required")
                        return
                }

                // Verify the act exists — refuses to follow a non-existent Act so
                // the subscription store cannot accumulate ghost follows.
                if _, err := actRepo.GetAct(r.Context(), legislation.ID(id)); err != nil {
                        writeError(w, http.StatusNotFound, "not_found", "act not found: "+id)
                        return
                }

                // Create a real subscription. EntityAct ("act") + the Act's ID is
                // the canonical target; the response exposes the composite
                // "act:"+actID form for the frontend.
                rec, err := subscriptionStore.Follow(p.UserID, EntityAct, id, nil)
                if err != nil {
                        writeError(w, http.StatusInternalServerError, "follow_failed", err.Error())
                        return
                }

                writeJSON(w, http.StatusOK, map[string]any{
                        "followed":         true,
                        "act_id":           id,
                        "subscription_id":  rec.ID,
                        "target":           "act:" + id,
                        "monitors":         followLawMonitors,
                        "subscription_url": "/api/v1/subscriptions/" + rec.ID,
                        "disclaimer": "You will be notified when authoritative evidence indicates this Act changed. The platform does not fabricate notifications.",
                })
        }
}

// makeLineageHandler handles GET /api/v1/acts/{id}/lineage.
//
// Spec section 20 — full legal lineage:
// Bill → Bill Version → Amendment → Passed Version → Assent → Act → Act
// Version → Amendment → Regulation → Judicial Interpretation → Repeal/Replacement.
//
// #217: This handler is NO LONGER a hardcoded 9-step list. It sources each
// step from the act's authoritative fields and post-assent events; steps
// with no evidence are reported as NOT_VERIFIED (never inferred).
func makeLineageHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/acts/")
                id := strings.TrimSuffix(path, "/lineage")
                // Query the repository for the act.
                a, err := actRepo.GetAct(r.Context(), legislation.ID(id))
                if err != nil {
                        writeError(w, http.StatusNotFound, "not_found", "act not found: "+id)
                        return
                }
                // Query the repository for the act's post-assent events.
                domainEvents, _ := actRepo.ListPostAssentEvents(r.Context(), legislation.ID(id))
                events := make([]PostAssentEventResponse, 0, len(domainEvents))
                for _, e := range domainEvents {
                        events = append(events, toPostAssentEventResponse(e))
                }
                lineage := buildLineage(toActResponse(*a), events)
                writeJSON(w, http.StatusOK, map[string]any{
                        "act_id":     id,
                        "lineage":    lineage,
                        "disclaimer": "Each step is independently sourced. Missing steps are flagged NOT_VERIFIED, not inferred.",
                })
        }
}

// buildAudit constructs an ActAuditResponse from an actResponse + events by
// delegating to domain.AuditForAct (issue #215). The platform NEVER
// hardcodes "everything is confirmed" — every audit dimension (assent,
// publication, commencement, regulations, judicial history, amendments,
// repeal) is computed from the act's authoritative fields and the
// post-assent events. An act without a commencement date is reported as
// NOT_VERIFIED (not "never commenced").
func buildAudit(act actResponse, events []PostAssentEventResponse) ActAuditResponse {
        audit := legislation.AuditForAct(actResponseToDomain(act), eventsResponseToDomain(events))

        out := ActAuditResponse{
                ActID:              act.ID,
                ActNumber:          audit.ActNumber,
                CurrentStatus:      string(audit.CurrentStatus),
                AssentRecorded:     audit.AssentRecorded,
                ActPublished:       audit.ActPublished,
                CommencementNotice: audit.CommencementNotice,
                RegulationsIssued:  audit.RegulationsIssued,
                Amended:            audit.Amended,
                CourtChallenged:    audit.CourtChallenged,
                JudicialDecision:   audit.JudicialDecision,
                Repealed:            audit.Repealed,
                DataGaps:           audit.DataGaps,
        }
        if out.DataGaps == nil {
                out.DataGaps = []string{}
        }
        out.AuditStatuses = make([]string, 0, len(audit.AuditStatuses))
        for _, s := range audit.AuditStatuses {
                out.AuditStatuses = append(out.AuditStatuses, string(s))
        }
        out.Disclaimer = "Missing records are reported as NOT_VERIFIED, never as 'never happened'. The platform does not infer absence."
        return out
}

// buildLineage constructs the 9-step legal lineage for an Act from the
// act's authoritative fields and its post-assent events (issue #217).
// Each step is independently sourced — when no evidence is available, the
// step is reported with status "NOT_VERIFIED" and description
// "No authoritative record found yet.". The platform NEVER infers a step
// from absence.
func buildLineage(act actResponse, events []PostAssentEventResponse) []LineageStep {
        // Helpers.
        findLatestEvent := func(et string) (PostAssentEventResponse, bool) {
                var latest PostAssentEventResponse
                found := false
                for _, e := range events {
                        if e.EventType != et {
                                continue
                        }
                        if !found || e.EventDate.After(latest.EventDate) {
                                latest = e
                                found = true
                        }
                }
                return latest, found
        }
        assentDate := func() string {
                if act.AssentDate != "" {
                        return act.AssentDate
                }
                return ""
        }
        commencementDate := func() string {
                if act.CommencementDate != "" {
                        return act.CommencementDate
                }
                return ""
        }

        steps := make([]LineageStep, 0, 9)

        // 1. Origin Bill. The platform tracks the originating Bill via
        //    Act.BillID — currently NOT exposed on actResponse, so this step
        //    is NOT_VERIFIED until the Act↔Bill link is surfaced to the API.
        //    (When the BillID is backfilled onto actResponse, this becomes
        //    VERIFIED with the Bill's identifier.)
        steps = append(steps, LineageStep{
                Step:        "origin_bill",
                Title:       act.Title + " (origin Bill)",
                Description: "No authoritative record found yet.",
                Status:      "NOT_VERIFIED",
        })

        // 2. Parliamentary journey. The platform does not yet track
        //    first-reading / committee / public-participation milestones as
        //    discrete events on the Act; until it does, this is NOT_VERIFIED.
        steps = append(steps, LineageStep{
                Step:        "parliamentary_journey",
                Title:       "Parliamentary journey",
                Description: "No authoritative record found yet.",
                Status:      "NOT_VERIFIED",
        })

        // 3. Presidential Assent.
        if d := assentDate(); d != "" {
                steps = append(steps, LineageStep{
                        Step:        "presidential_assent",
                        Title:       "Presidential Assent",
                        Description:  "The President assented to the Bill on " + d + ", creating the Act.",
                        Status:      "VERIFIED",
                        Date:        d,
                        SourceURL:   act.SourceURL,
                })
        } else {
                steps = append(steps, LineageStep{
                        Step:        "presidential_assent",
                        Title:       "Presidential Assent",
                        Description: "No authoritative record found yet.",
                        Status:      "NOT_VERIFIED",
                })
        }

        // 4. Act publication (Kenya Gazette).
        // NOTE: the API doesn't yet expose PublicationDate separately; we
        // treat the gazette source URL as evidence the Act was published.
        // This becomes VERIFIED with a real PublicationDate when that field
        // is backfilled onto actResponse.
        if act.SourceURL != "" {
                steps = append(steps, LineageStep{
                        Step:        "act_publication",
                        Title:       "Act publication",
                        Description:  "The Act was published in the Kenya Gazette.",
                        Status:      "VERIFIED",
                        SourceURL:   act.SourceURL,
                })
        } else {
                steps = append(steps, LineageStep{
                        Step:        "act_publication",
                        Title:       "Act publication",
                        Description: "No authoritative record found yet.",
                        Status:      "NOT_VERIFIED",
                })
        }

        // 5. Commencement.
        if d := commencementDate(); d != "" {
                desc := "The Act commenced on " + d + " per the commencement notice."
                if ev, ok := findLatestEvent("COMMENCEMENT"); ok && ev.SourceURL != "" {
                        steps = append(steps, LineageStep{
                                Step:        "commencement",
                                Title:       "Commencement",
                                Description: desc,
                                Status:      "VERIFIED",
                                Date:        d,
                                SourceURL:   ev.SourceURL,
                        })
                } else {
                        steps = append(steps, LineageStep{
                                Step:        "commencement",
                                Title:       "Commencement",
                                Description: desc,
                                Status:      "VERIFIED",
                                Date:        d,
                        })
                }
        } else {
                steps = append(steps, LineageStep{
                        Step:        "commencement",
                        Title:       "Commencement",
                        Description: "No authoritative record found yet.",
                        Status:      "NOT_VERIFIED",
                })
        }

        // 6. Regulations issued under the Act.
        if ev, ok := findLatestEvent("REGULATION"); ok {
                steps = append(steps, LineageStep{
                        Step:        "regulations",
                        Title:       "Regulations",
                        Description:  ev.Title,
                        Status:      "VERIFIED",
                        Date:        ev.EventDate.Format("2006-01-02"),
                        SourceURL:   ev.SourceURL,
                })
        } else {
                steps = append(steps, LineageStep{
                        Step:        "regulations",
                        Title:       "Regulations",
                        Description: "No authoritative record found yet.",
                        Status:      "NOT_VERIFIED",
                })
        }

        // 7. Amendments.
        if ev, ok := findLatestEvent("AMENDMENT"); ok {
                steps = append(steps, LineageStep{
                        Step:        "amendments",
                        Title:       "Amendments",
                        Description:  ev.Title,
                        Status:      "VERIFIED",
                        Date:        ev.EventDate.Format("2006-01-02"),
                        SourceURL:   ev.SourceURL,
                })
        } else {
                steps = append(steps, LineageStep{
                        Step:        "amendments",
                        Title:       "Amendments",
                        Description: "No authoritative record found yet.",
                        Status:      "NOT_VERIFIED",
                })
        }

        // 8. Court decisions (challenges + judgments).
        if ev, ok := findLatestEvent("JUDICIAL_DECISION"); ok {
                steps = append(steps, LineageStep{
                        Step:        "court_decisions",
                        Title:       "Court decisions",
                        Description:  ev.Title,
                        Status:      "VERIFIED",
                        Date:        ev.EventDate.Format("2006-01-02"),
                        SourceURL:   ev.SourceURL,
                })
        } else if ev, ok := findLatestEvent("COURT_CHALLENGE"); ok {
                steps = append(steps, LineageStep{
                        Step:        "court_decisions",
                        Title:       "Court decisions",
                        Description:  ev.Title,
                        Status:      "VERIFIED",
                        Date:        ev.EventDate.Format("2006-01-02"),
                        SourceURL:   ev.SourceURL,
                })
        } else {
                steps = append(steps, LineageStep{
                        Step:        "court_decisions",
                        Title:       "Court decisions",
                        Description: "No authoritative record found yet.",
                        Status:      "NOT_VERIFIED",
                })
        }

        // 9. Current status — always present (the act has a Status field).
        if act.Status == "" {
                steps = append(steps, LineageStep{
                        Step:        "current_status",
                        Title:       "Current status",
                        Description: "No authoritative record found yet.",
                        Status:      "NOT_VERIFIED",
                })
        } else {
                steps = append(steps, LineageStep{
                        Step:        "current_status",
                        Title:       "Current status",
                        Description:  "The Act's current legal status is " + act.Status + ".",
                        Status:      "VERIFIED",
                })
        }

        return steps
}
