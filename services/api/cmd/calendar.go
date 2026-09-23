// Package main — Civic Calendar handlers (task ENG-K1 — Feature 1).
//
// Implements an in-memory CivicCalendar store keyed by event ID. The store
// is seeded at package init with 26 sample events spanning the next ~3
// months of a realistic Kenya Parliament schedule (parliament sessions,
// committee meetings, bill readings, public-participation deadlines,
// gazette publications, court hearings, and budget presentations).
//
// Endpoints (registered in main.go):
//
//      GET /api/v1/calendar         ?month=2026-09&country=KE
//      GET /api/v1/calendar/today  ?country=KE
//      GET /api/v1/calendar/upcoming?country=KE&limit=10
//      GET /api/v1/calendar/live   ?country=KE          (issue #283)
//
// Each event carries: id, title, date, type, description, source_url,
// country, institution. Dates are RFC3339 (date-only is accepted on input
// but the response always returns YYYY-MM-DD for compactness).
//
// The sample data uses a fixed calendarAnchor (first day of next month at
// UTC) so the events stay reproducible under tests. In production the
// anchor would be wall-clock today; the in-memory store would be replaced
// by a SQL repository (e.g. civic_calendar.events) populated by the
// ingestion service from parliament.go.ke + kenyalaw.go.ke.
package main

import (
        "crypto/rand"
        "encoding/hex"
        "encoding/json"
        "fmt"
        "net/http"
        "sort"
        "strconv"
        "strings"
        "sync"
        "time"
)

// --- Event types ---

// CalendarEventType is the canonical event-type taxonomy. The string values
// are part of the public HTTP contract — frontend filters + i18n map over
// them — so they must not be renamed without a coordinated frontend change.
type CalendarEventType string

const (
        CalParliamentSession    CalendarEventType = "parliament_session"
        CalCommitteeMeeting     CalendarEventType = "committee_meeting"
        CalBillReading          CalendarEventType = "bill_reading"
        CalPublicParticipation  CalendarEventType = "public_participation"
        CalGazettePublication   CalendarEventType = "gazette_publication"
        CalCourtHearing         CalendarEventType = "court_hearing"
        CalBudgetPresentation   CalendarEventType = "budget_presentation"
)

// calValidEventTypes is the allow-list the filter parameter accepts.
// Anything outside this set returns 400 (bad_request) so the frontend
// can rely on the contract. Named with the cal prefix to avoid colliding
// with notifications.validEventTypes (a separate allow-list for the
// notification event_type column).
var calValidEventTypes = map[CalendarEventType]bool{
        CalParliamentSession:    true,
        CalCommitteeMeeting:     true,
        CalBillReading:          true,
        CalPublicParticipation:  true,
        CalGazettePublication:   true,
        CalCourtHearing:         true,
        CalBudgetPresentation:   true,
}

// CalendarEvent is the JSON shape returned by every calendar endpoint.
type CalendarEvent struct {
        ID          string            `json:"id"`
        Title       string            `json:"title"`
        Date        string            `json:"date"` // YYYY-MM-DD (RFC3339 date)
        Type        CalendarEventType `json:"type"`
        Description string            `json:"description,omitempty"`
        SourceURL   string            `json:"source_url,omitempty"`
        Country     string            `json:"country"`
        Institution string            `json:"institution,omitempty"`
}

// --- In-memory store ---

// CalendarStore is an in-memory store of CalendarEvent rows keyed by event ID.
// It is safe for concurrent use. Reads (EventsForMonth / Today / Upcoming)
// return defensively-copied slices so callers cannot mutate the seed data.
type CalendarStore struct {
        mu      sync.RWMutex
        events  map[string]CalendarEvent
        anchor  time.Time // wall-clock reference used by Today()
        country string    // default country code applied when the request omits one
}

// NewCalendarStore returns an empty store. Use SeedCalendarSampleData to
// populate it with the realistic Kenya Parliament sample schedule.
//
// `anchor` is the wall-clock value used as "today" by Today() and Upcoming().
// Tests pass a deterministic value; production passes time.Now().UTC().
// `country` is the default country code applied when the request omits one.
func NewCalendarStore(anchor time.Time, country string) *CalendarStore {
        if country == "" {
                country = "KE"
        }
        return &CalendarStore{
                events:  make(map[string]CalendarEvent),
                anchor:  anchor,
                country: strings.ToUpper(country),
        }
}

// Upsert adds or replaces an event by ID. It is safe for concurrent use.
// Returns the stored event (with normalised date + country fields).
func (s *CalendarStore) Upsert(e CalendarEvent) CalendarEvent {
        if e.ID == "" {
                e.ID = newCalendarEventID()
        }
        // Normalise date to YYYY-MM-DD. Accept RFC3339, full-date, or anything
        // time.Parse can handle. Reject garbage by leaving the field as-is —
        // the caller can detect by re-parsing.
        if t, err := time.Parse("2006-01-02", e.Date); err == nil {
                e.Date = t.Format("2006-01-02")
        } else if t, err := time.Parse(time.RFC3339, e.Date); err == nil {
                e.Date = t.UTC().Format("2006-01-02")
        }
        if e.Country == "" {
                e.Country = s.country
        } else {
                e.Country = strings.ToUpper(e.Country)
        }
        s.mu.Lock()
        s.events[e.ID] = e
        s.mu.Unlock()
        return e
}

// List returns all events matching the country filter, sorted by date
// ascending. If country is empty, events for every country are returned.
func (s *CalendarStore) List(country string) []CalendarEvent {
        country = strings.ToUpper(strings.TrimSpace(country))
        s.mu.RLock()
        out := make([]CalendarEvent, 0, len(s.events))
        for _, e := range s.events {
                if country != "" && e.Country != country {
                        continue
                }
                out = append(out, e)
        }
        s.mu.RUnlock()
        sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
        return out
}

// EventsForMonth returns events whose Date falls inside the YYYY-MM month.
// The month string is parsed leniently: "2026-09", "2026-9", "202609" all
// work; an empty month returns events for the anchor's month.
func (s *CalendarStore) EventsForMonth(month, country string) ([]CalendarEvent, error) {
        start, end, err := parseMonthRange(month, s.anchor)
        if err != nil {
                return nil, err
        }
        all := s.List(country)
        out := make([]CalendarEvent, 0)
        for _, e := range all {
                t, err := time.Parse("2006-01-02", e.Date)
                if err != nil {
                        continue
                }
                if !t.Before(start) && t.Before(end) {
                        out = append(out, e)
                }
        }
        return out, nil
}

// Today returns events whose Date equals the anchor's YYYY-MM-DD.
func (s *CalendarStore) Today(country string) []CalendarEvent {
        today := s.anchor.Format("2006-01-02")
        all := s.List(country)
        out := make([]CalendarEvent, 0)
        for _, e := range all {
                if e.Date == today {
                        out = append(out, e)
                }
        }
        return out
}

// Upcoming returns the next `limit` events whose Date is today or later.
// limit is clamped to [1, 100]. Country filters; "" means all countries.
func (s *CalendarStore) Upcoming(limit int, country string) []CalendarEvent {
        if limit < 1 {
                limit = 10
        }
        if limit > 100 {
                limit = 100
        }
        today := s.anchor.Format("2006-01-02")
        all := s.List(country)
        out := make([]CalendarEvent, 0, limit)
        for _, e := range all {
                if e.Date >= today {
                        out = append(out, e)
                        if len(out) >= limit {
                                break
                        }
                }
        }
        return out
}

// FilterByTypes returns the subset of events whose Type is in types. If
// types is empty, the input is returned unchanged.
func FilterByTypes(events []CalendarEvent, types []CalendarEventType) []CalendarEvent {
        if len(types) == 0 {
                return events
        }
        wanted := make(map[CalendarEventType]bool, len(types))
        for _, t := range types {
                wanted[t] = true
        }
        out := events[:0]
        for _, e := range events {
                if wanted[e.Type] {
                        out = append(out, e)
                }
        }
        return out
}

// parseMonthRange converts "2026-09" → [2026-09-01, 2026-10-01). Empty
// input uses the anchor's month. Accepted formats: "2006-01", "2006-1",
// "2006/01", "200601".
func parseMonthRange(month string, anchor time.Time) (time.Time, time.Time, error) {
        month = strings.TrimSpace(month)
        var ref time.Time
        if month == "" {
                ref = anchor
        } else {
                // Normalise separators.
                s := strings.ReplaceAll(month, "/", "-")
                for _, layout := range []string{"2006-01", "2006-1", "200601"} {
                        if t, err := time.Parse(layout, s); err == nil {
                                ref = t
                                break
                        }
                }
        }
        if ref.IsZero() {
                return time.Time{}, time.Time{}, fmt.Errorf("invalid month %q (expected YYYY-MM, e.g. 2026-09)", month)
        }
        start := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, time.UTC)
        end := start.AddDate(0, 1, 0)
        return start, end, nil
}

// newCalendarEventID generates a 16-byte hex ID. Production would use a
// UUID; the prefix "cal_" keeps calendar event IDs distinct from other
// platform IDs (flw_, src_, evi_) in logs.
func newCalendarEventID() string {
        b := make([]byte, 16)
        if _, err := rand.Read(b); err != nil {
                return "cal_" + time.Now().UTC().Format("20060102150405.000000000")
        }
        return "cal_" + hex.EncodeToString(b)
}

// --- HTTP handlers ---

// makeCalendarHandler returns an http.HandlerFunc that routes between
// the calendar collection endpoints.
//
//      GET /api/v1/calendar         — events for a month (?month=YYYY-MM)
//      GET /api/v1/calendar/today   — today's events
//      GET /api/v1/calendar/upcoming— upcoming events (?limit=N, default 10)
//      GET /api/v1/calendar/live    — "is parliament sitting right now?" (issue #283)
//
// All three honour ?country=KE (defaults to store's country) and an
// optional ?types=parliament_session,committee_meeting filter (comma list).
func makeCalendarHandler(store *CalendarStore) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        w.Header().Set("Allow", "GET")
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
                                "only GET is supported on /api/v1/calendar")
                        return
                }

                // Sub-route dispatch. The outer mux registers /api/v1/calendar,
                // /api/v1/calendar/today, /api/v1/calendar/upcoming,
                // /api/v1/calendar/live as separate patterns, but this handler
                // is also robust to being mounted at /api/v1/calendar/ — it
                // inspects the trailing path segment.
                switch strings.TrimPrefix(r.URL.Path, "/api/v1/calendar") {
                case "", "/":
                        handleCalendarMonth(w, r, store)
                case "/today", "/today/":
                        handleCalendarToday(w, r, store)
                case "/upcoming", "/upcoming/":
                        handleCalendarUpcoming(w, r, store)
                case "/live", "/live/":
                        handleCalendarLive(w, r, store)
                default:
                        writeError(w, http.StatusNotFound, "not_found",
                                "unknown calendar sub-path: "+r.URL.Path)
                }
        }
}

// handleCalendarMonth handles GET /api/v1/calendar?month=2026-09&country=KE.
func handleCalendarMonth(w http.ResponseWriter, r *http.Request, store *CalendarStore) {
        country := countryQuery(r, store.country)
        month := r.URL.Query().Get("month")
        events, err := store.EventsForMonth(month, country)
        if err != nil {
                writeError(w, http.StatusBadRequest, "bad_request", err.Error())
                return
        }
        types, ok := parseTypesQuery(r)
        if !ok {
                writeError(w, http.StatusBadRequest, "bad_request",
                        "invalid event type in ?types= filter")
                return
        }
        events = FilterByTypes(events, types)
        writeJSON(w, http.StatusOK, map[string]any{
                "month":  normaliseMonth(month, store.anchor),
                "items":  events,
                "total":  len(events),
        })
}

// handleCalendarToday handles GET /api/v1/calendar/today?country=KE.
func handleCalendarToday(w http.ResponseWriter, r *http.Request, store *CalendarStore) {
        country := countryQuery(r, store.country)
        events := store.Today(country)
        types, ok := parseTypesQuery(r)
        if !ok {
                writeError(w, http.StatusBadRequest, "bad_request",
                        "invalid event type in ?types= filter")
                return
        }
        events = FilterByTypes(events, types)
        writeJSON(w, http.StatusOK, map[string]any{
                "date":   store.anchor.Format("2006-01-02"),
                "items":  events,
                "total":  len(events),
        })
}

// handleCalendarUpcoming handles GET /api/v1/calendar/upcoming?country=KE&limit=10.
func handleCalendarUpcoming(w http.ResponseWriter, r *http.Request, store *CalendarStore) {
        country := countryQuery(r, store.country)
        limit := parseIntDefault(r.URL.Query().Get("limit"), 10)
        types, ok := parseTypesQuery(r)
        if !ok {
                writeError(w, http.StatusBadRequest, "bad_request",
                        "invalid event type in ?types= filter")
                return
        }
        events := store.Upcoming(limit, country)
        events = FilterByTypes(events, types)
        writeJSON(w, http.StatusOK, map[string]any{
                "items":  events,
                "total":  len(events),
                "limit":  limit,
        })
}

// --- Plenary livestream (issue #283) ---

// livePlaceholderVideoURL is the YouTube URL returned by /api/v1/calendar/live
// when parliament is in session. The Hansard crawler already discovers a
// per-sitting YouTube URL (HansardCandidate.VideoURL) — in production the
// live endpoint would surface that URL directly. Until the live Hansard
// discovery pipeline is wired into the calendar store, we return this
// stable placeholder so the frontend can exercise the embed path.
//
// TODO: wire to live Hansard discovery — replace with the sitting's
// HansardCandidate.VideoURL once the ingestion worker writes calendar
// events + their video URLs into the store.
const livePlaceholderVideoURL = "https://youtu.be/dQw4w9WgXcQ"

// CalendarLiveResponse is the JSON envelope returned by
// GET /api/v1/calendar/live. When `is_live` is true, the response also
// carries the live-stream video URL, a human-readable sitting title, and
// the house (e.g., "National Assembly") so the frontend can render a
// "🔴 Parliament is live — Watch now" banner + modal without a second
// round-trip. When `is_live` is false, the remaining fields are omitted
// so the wire payload stays minimal (the frontend only needs the flag).
type CalendarLiveResponse struct {
        IsLive   bool   `json:"is_live"`
        VideoURL string `json:"video_url,omitempty"`
        Title    string `json:"title,omitempty"`
        House    string `json:"house,omitempty"`
        // EventID is the calendar event ID of the in-session sitting so the
        // frontend can deep-link to the calendar entry. Omitted when not live.
        EventID string `json:"event_id,omitempty"`
        // Date echoes the anchor's YYYY-MM-DD so the frontend can show
        // "Live — [date]" without parsing the video URL. Omitted when not
        // live.
        Date string `json:"date,omitempty"`
}

// LiveSession returns the in-session parliament_session event for "today"
// (the anchor's date), filtered by country. Returns (zero, false) when no
// parliament_session event is scheduled for today.
//
// "Today" follows the same anchor contract as Today() — production uses
// wall-clock UTC; tests inject a deterministic anchor so the seeded
// sample schedule is reproducible.
func (s *CalendarStore) LiveSession(country string) (CalendarEvent, bool) {
        today := s.anchor.Format("2006-01-02")
        for _, e := range s.Today(country) {
                if e.Type == CalParliamentSession && e.Date == today {
                        return e, true
                }
        }
        return CalendarEvent{}, false
}

// handleCalendarLive handles GET /api/v1/calendar/live?country=KE (issue #283).
//
// Returns 200 with `{ "is_live": true, "video_url": ..., "title": ...,
// "house": ... }` when a parliament_session event is scheduled for today,
// or 200 with `{ "is_live": false }` when parliament is not sitting.
//
// The video URL is currently the placeholder constant above — in
// production it would come from the Hansard crawl (HansardCandidate.VideoURL).
func handleCalendarLive(w http.ResponseWriter, r *http.Request, store *CalendarStore) {
        country := countryQuery(r, store.country)
        event, ok := store.LiveSession(country)
        if !ok {
                writeJSON(w, http.StatusOK, CalendarLiveResponse{IsLive: false})
                return
        }
        // TODO: wire to live Hansard discovery — replace livePlaceholderVideoURL
        // with the sitting's HansardCandidate.VideoURL once the ingestion worker
        // writes calendar events + their video URLs into the store.
        title := event.Title
        if title == "" {
                title = "National Assembly Sitting — " + event.Date
        }
        house := event.Institution
        if house == "" {
                house = calendarEventHouse(event)
                if house == "" {
                        house = "National Assembly"
                }
        }
        writeJSON(w, http.StatusOK, CalendarLiveResponse{
                IsLive:   true,
                VideoURL: livePlaceholderVideoURL,
                Title:    title,
                House:    house,
                EventID:  event.ID,
                Date:     event.Date,
        })
}

// calendarEventHouse returns the parliamentary house implied by the event's
// Institution or Title. Used as a fallback when callers want a short house
// label ("National Assembly" / "Senate") and the event has Institution
// populated. Returns "" when the house cannot be inferred.
//
// Named calendarEventHouse (rather than CalendarEvent.House) so it does
// not collide with a future struct field of the same name; it is a pure
// helper that operates on the event's existing Institution/Title strings.
func calendarEventHouse(e CalendarEvent) string {
        switch {
        case strings.Contains(strings.ToLower(e.Institution), "senate"):
                return "Senate"
        case strings.Contains(strings.ToLower(e.Institution), "national assembly"):
                return "National Assembly"
        case strings.Contains(strings.ToLower(e.Institution), "parliament"):
                return "Parliament"
        case strings.Contains(strings.ToLower(e.Title), "senate"):
                return "Senate"
        case strings.Contains(strings.ToLower(e.Title), "national assembly"):
                return "National Assembly"
        }
        return ""
}

// countryQuery returns the ?country= query value, upper-cased, defaulting
// to def when empty.
func countryQuery(r *http.Request, def string) string {
        c := strings.TrimSpace(r.URL.Query().Get("country"))
        if c == "" {
                return def
        }
        return strings.ToUpper(c)
}

// parseTypesQuery parses ?types=parliament_session,committee_meeting into
// a []CalendarEventType. Returns ok=false if any token is not in the
// allow-list (the caller rejects with 400). An empty query yields an empty
// slice + ok=true (no filter applied).
func parseTypesQuery(r *http.Request) ([]CalendarEventType, bool) {
        raw := strings.TrimSpace(r.URL.Query().Get("types"))
        if raw == "" {
                return nil, true
        }
        parts := strings.Split(raw, ",")
        out := make([]CalendarEventType, 0, len(parts))
        for _, p := range parts {
                p = strings.TrimSpace(p)
                if p == "" {
                        continue
                }
                t := CalendarEventType(strings.ToLower(p))
                if !calValidEventTypes[t] {
                        return nil, false
                }
                out = append(out, t)
        }
        return out, true
}

// normaliseMonth returns the input month parsed + reformatted as YYYY-MM.
// Empty input yields the anchor's month. Invalid input yields "".
func normaliseMonth(month string, anchor time.Time) string {
        start, _, err := parseMonthRange(month, anchor)
        if err != nil {
                return ""
        }
        return start.Format("2006-01")
}

// --- Sample data ---

// calendarStore is the package-level in-memory CalendarStore. main.go
// reuses it for the HTTP handlers. The seed data spans the next ~3 months
// from calendarAnchor, so the calendar always shows events a citizen can
// actually attend / watch / act on.
var calendarStore = NewCalendarStore(calendarAnchor, "KE")

// calendarAnchor is the wall-clock "today" for the seeded sample data. It
// is computed at package init from time.Now().UTC() so the calendar always
// shows future events. Tests that need determinism construct their own
// store via NewCalendarStore with a fixed anchor.
var calendarAnchor = time.Now().UTC()

// SeedCalendarSampleData populates the store with 26 realistic Kenya
// Parliament events. It is idempotent: re-running it does NOT create
// duplicate IDs (the sample event IDs are deterministic). It is called
// from main.go after the package-level store is constructed.
//
// The sample schedule mixes:
//   - 4 parliament_session events (NA + Senate sittings)
//   - 6 committee_meeting events (Budget, Justice, Health, Education, Energy, Defence)
//   - 4 bill_reading events (1st / 2nd / 3rd readings of pending Bills)
//   - 3 public_participation events (memorandum submission deadlines)
//   - 4 gazette_publication events (weekly Friday gazette)
//   - 3 court_hearing events (petitions before the High / Supreme Court)
//   - 2 budget_presentation events (Budget Statement + Medium-Term Debt Strategy)
//
// Total: 26 events spanning ~95 days.
func SeedCalendarSampleData(store *CalendarStore, anchor time.Time) {
        // anchorDay returns the YYYY-MM-DD string for `dayOffset` days from anchor.
        anchorDay := func(dayOffset int) string {
                return anchor.AddDate(0, 0, dayOffset).Format("2006-01-02")
        }
        // anchorFriday returns the YYYY-MM-DD of the Friday in the week that
        // is `weekOffset` weeks from anchor's week.
        anchorFriday := func(weekOffset int) string {
                t := anchor.AddDate(0, 0, weekOffset*7)
                // Move forward to the next Friday (weekday 5).
                delta := (5 - int(t.Weekday()) + 7) % 7
                return t.AddDate(0, 0, delta).Format("2006-01-02")
        }

        samples := []CalendarEvent{
                // === Parliament sessions (NA + Senate) ===
                {
                        ID:          "cal_parl_na_wk1",
                        Title:       "National Assembly — Morning Sitting",
                        Date:        anchorDay(2),
                        Type:        CalParliamentSession,
                        Description: "Order of the Day: Second Reading of the Public Finance Management (Amendment) Bill, 2026. Live broadcast on Parliament TV.",
                        SourceURL:   "https://www.parliament.go.ke/the-national-assembly/order-paper",
                        Country:     "KE",
                        Institution: "National Assembly of Kenya",
                },
                {
                        ID:          "cal_parl_sen_wk2",
                        Title:       "Senate — Afternoon Sitting",
                        Date:        anchorDay(9),
                        Type:        CalParliamentSession,
                        Description: "Senate considers County Governments Additional Allocations Bill. Senators may table amendments.",
                        SourceURL:   "https://www.parliament.go.ke/the-senate/order-paper",
                        Country:     "KE",
                        Institution: "Senate of Kenya",
                },
                {
                        ID:          "cal_parl_na_wk4",
                        Title:       "National Assembly — Special Sitting (Budget)",
                        Date:        anchorDay(28),
                        Type:        CalParliamentSession,
                        Description: "Special sitting convened for the Budget Speech and tabling of the Appropriations Bill, 2026.",
                        SourceURL:   "https://www.parliament.go.ke/the-national-assembly/order-paper",
                        Country:     "KE",
                        Institution: "National Assembly of Kenya",
                },
                {
                        ID:          "cal_parl_joint_wk8",
                        Title:       "Joint Sitting — Presidential Address",
                        Date:        anchorDay(56),
                        Type:        CalParliamentSession,
                        Description: "His Excellency the President addresses a Joint Sitting of Parliament per Article 132 of the Constitution.",
                        SourceURL:   "https://www.parliament.go.ke/news/joint-sitting",
                        Country:     "KE",
                        Institution: "Parliament of Kenya",
                },

                // === Committee meetings ===
                {
                        ID:          "cal_cmte_budget_wk1",
                        Title:       "Budget & Appropriations Committee — Pre-Budget Hearings",
                        Date:        anchorDay(4),
                        Type:        CalCommitteeMeeting,
                        Description: "Public hearings on the 2026/27 Budget Policy Statement. Submissions from the National Treasury, CRA, and CBK.",
                        SourceURL:   "https://www.parliament.go.ke/committees/budget-appropriations",
                        Country:     "KE",
                        Institution: "Budget & Appropriations Committee",
                },
                {
                        ID:          "cal_cmte_justice_wk2",
                        Title:       "Justice & Legal Affairs Committee — Inquiry",
                        Date:        anchorDay(11),
                        Type:        CalCommitteeMeeting,
                        Description: "Continued inquiry into the status of the Building Bridges Initiative (BBI) court judgments. AG + DPP invited.",
                        SourceURL:   "https://www.parliament.go.ke/committees/justice-legal-affairs",
                        Country:     "KE",
                        Institution: "Justice & Legal Affairs Committee",
                },
                {
                        ID:          "cal_cmte_health_wk3",
                        Title:       "Health Committee — Social Health Authority Oversight",
                        Date:        anchorDay(18),
                        Type:        CalCommitteeMeeting,
                        Description: "Oversight hearing on SHA fund disbursement to county hospitals. Cabinet Secretary for Health to appear.",
                        SourceURL:   "https://www.parliament.go.ke/committees/health",
                        Country:     "KE",
                        Institution: "Health Committee",
                },
                {
                        ID:          "cal_cmte_education_wk4",
                        Title:       "Education Committee — Universities Funding Model",
                        Date:        anchorDay(25),
                        Type:        CalCommitteeMeeting,
                        Description: "Review of the new differentiated unit-cost funding model for public universities. Vice-Chancellors invited.",
                        SourceURL:   "https://www.parliament.go.ke/committees/education",
                        Country:     "KE",
                        Institution: "Education & Research Committee",
                },
                {
                        ID:          "cal_cmte_energy_wk6",
                        Title:       "Energy Committee — Geothermal Tariff Review",
                        Date:        anchorDay(42),
                        Type:        CalCommitteeMeeting,
                        Description: "Inquiry into the Geothermal Development Company tariff review and its impact on the cost of power.",
                        SourceURL:   "https://www.parliament.go.ke/committees/energy",
                        Country:     "KE",
                        Institution: "Energy Committee",
                },
                {
                        ID:          "cal_cmte_defence_wk9",
                        Title:       "Defence & Foreign Relations Committee — KDF Deployment",
                        Date:        anchorDay(63),
                        Type:        CalCommitteeMeeting,
                        Description: "Closed session: review of KDF deployment under Article 241 for approval by Parliament.",
                        SourceURL:   "https://www.parliament.go.ke/committees/defence-foreign-relations",
                        Country:     "KE",
                        Institution: "Defence & Foreign Relations Committee",
                },

                // === Bill readings ===
                {
                        ID:          "cal_bill_pfm_1r",
                        Title:       "Public Finance Management (Amendment) Bill, 2026 — First Reading",
                        Date:        anchorDay(2),
                        Type:        CalBillReading,
                        Description: "First Reading of the Public Finance Management (Amendment) Bill, 2026. Bill committed to the Budget & Appropriations Committee.",
                        SourceURL:   "https://kenyalaw.org/bills/2026/pfm-amendment",
                        Country:     "KE",
                        Institution: "National Assembly of Kenya",
                },
                {
                        ID:          "cal_bill_data_2r",
                        Title:       "Data Protection (Amendment) Bill, 2026 — Second Reading",
                        Date:        anchorDay(15),
                        Type:        CalBillReading,
                        Description: "Second Reading: debate on the principles of the Data Protection (Amendment) Bill, 2026. Committee of the Whole to follow.",
                        SourceURL:   "https://kenyalaw.org/bills/2026/data-protection-amendment",
                        Country:     "KE",
                        Institution: "National Assembly of Kenya",
                },
                {
                        ID:          "cal_bill_county_3r",
                        Title:       "County Governments Additional Allocations Bill — Third Reading",
                        Date:        anchorDay(22),
                        Type:        CalBillReading,
                        Description: "Third Reading of the County Governments Additional Allocations Bill. Vote on the Bill as amended.",
                        SourceURL:   "https://kenyalaw.org/bills/2026/county-additional-allocations",
                        Country:     "KE",
                        Institution: "Senate of Kenya",
                },
                {
                        ID:          "cal_bill_elections_2r",
                        Title:       "Elections (Amendment) Bill, 2026 — Second Reading",
                        Date:        anchorDay(40),
                        Type:        CalBillReading,
                        Description: "Second Reading of the Elections (Amendment) Bill, 2026 — campaign finance caps and dispute resolution timelines.",
                        SourceURL:   "https://kenyalaw.org/bills/2026/elections-amendment",
                        Country:     "KE",
                        Institution: "National Assembly of Kenya",
                },

                // === Public participation deadlines ===
                {
                        ID:          "cal_pp_pfm_deadline",
                        Title:       "Public Participation — PFM (Amendment) Bill, 2026",
                        Date:        anchorDay(7),
                        Type:        CalPublicParticipation,
                        Description: "Deadline for submitting memoranda on the PFM (Amendment) Bill, 2026. Submissions to the Clerk of the National Assembly.",
                        SourceURL:   "https://www.parliament.go.ke/public-participation",
                        Country:     "KE",
                        Institution: "National Assembly of Kenya",
                },
                {
                        ID:          "cal_pp_data_deadline",
                        Title:       "Public Participation — Data Protection (Amendment) Bill",
                        Date:        anchorDay(20),
                        Type:        CalPublicParticipation,
                        Description: "Deadline for memoranda on the Data Protection (Amendment) Bill, 2026. Submissions accepted via portal + email.",
                        SourceURL:   "https://www.parliament.go.ke/public-participation",
                        Country:     "KE",
                        Institution: "National Assembly of Kenya",
                },
                {
                        ID:          "cal_pp_budget_deadline",
                        Title:       "Public Participation — 2026/27 Budget Policy Statement",
                        Date:        anchorDay(27),
                        Type:        CalPublicParticipation,
                        Description: "Final deadline for public submissions on the Budget Policy Statement. Hearings precede the Budget Speech.",
                        SourceURL:   "https://www.treasury.go.ke/bps-2026-27",
                        Country:     "KE",
                        Institution: "National Treasury",
                },

                // === Gazette publications (weekly Friday) ===
                {
                        ID:          "cal_gazette_wk1",
                        Title:       "Kenya Gazette — Special Issue (Legal Notices)",
                        Date:        anchorFriday(0),
                        Type:        CalGazettePublication,
                        Description: "Weekly Friday Gazette: 12 Legal Notices including the Energy (Petroleum Pricing) Regulations, 2026.",
                        SourceURL:   "https://www.gazettes.africa/kenya",
                        Country:     "KE",
                        Institution: "Kenya Gazette",
                },
                {
                        ID:          "cal_gazette_wk3",
                        Title:       "Kenya Gazette — Vol. CXXVIII No. 32",
                        Date:        anchorFriday(2),
                        Type:        CalGazettePublication,
                        Description: "Weekly Friday Gazette: appointments, transfer of officers, and the Public Procurement (Preference) Regulations, 2026.",
                        SourceURL:   "https://www.gazettes.africa/kenya",
                        Country:     "KE",
                        Institution: "Kenya Gazette",
                },
                {
                        ID:          "cal_gazette_wk5",
                        Title:       "Kenya Gazette — Special Issue (Tenders)",
                        Date:        anchorFriday(4),
                        Type:        CalGazettePublication,
                        Description: "Special issue: tender notices from MDAs, the State Department for Public Works, and county governments.",
                        SourceURL:   "https://www.gazettes.africa/kenya",
                        Country:     "KE",
                        Institution: "Kenya Gazette",
                },
                {
                        ID:          "cal_gazette_wk8",
                        Title:       "Kenya Gazette — Vol. CXXVIII No. 35",
                        Date:        anchorFriday(7),
                        Type:        CalGazettePublication,
                        Description: "Weekly Friday Gazette: notice of assent to two Acts, ministerial appointments, and the Health Insurance (Rates) Order, 2026.",
                        SourceURL:   "https://www.gazettes.africa/kenya",
                        Country:     "KE",
                        Institution: "Kenya Gazette",
                },

                // === Court hearings ===
                {
                        ID:          "cal_court_petition_wk3",
                        Title:       "High Court — Constitutional Petition No. E012 of 2026",
                        Date:        anchorDay(17),
                        Type:        CalCourtHearing,
                        Description: "Mention: petition challenging the constitutionality of the Computer Misuse and Cybercrimes (Amendment) Act, 2026.",
                        SourceURL:   "https://www.judiciary.go.ke/cause-lists",
                        Country:     "KE",
                        Institution: "High Court of Kenya (Constitutional & Human Rights Division)",
                },
                {
                        ID:          "cal_court_appeal_wk5",
                        Title:       "Court of Appeal — Civil Appeal No. E087 of 2025",
                        Date:        anchorDay(33),
                        Type:        CalCourtHearing,
                        Description: "Hearing of appeal against the High Court decision on the Public Procurement (Preference) Regulations.",
                        SourceURL:   "https://www.judiciary.go.ke/cause-lists",
                        Country:     "KE",
                        Institution: "Court of Appeal of Kenya",
                },
                {
                        ID:          "cal_court_sc_wk10",
                        Title:       "Supreme Court — Advisory Opinion No. E003 of 2026",
                        Date:        anchorDay(70),
                        Type:        CalCourtHearing,
                        Description: "Advisory opinion on the constitutionality of the Senate's revenue allocation formula. Live stream on Judiciary portal.",
                        SourceURL:   "https://www.judiciary.go.ke/cause-lists",
                        Country:     "KE",
                        Institution: "Supreme Court of Kenya",
                },

                // === Budget presentations ===
                {
                        ID:          "cal_budget_statement",
                        Title:       "Budget Statement — Financial Year 2026/27",
                        Date:        anchorDay(28),
                        Type:        CalBudgetPresentation,
                        Description: "The Cabinet Secretary for the National Treasury presents the Budget Statement for FY 2026/27 to the National Assembly.",
                        SourceURL:   "https://www.treasury.go.ke/budget-statement-2026-27",
                        Country:     "KE",
                        Institution: "National Treasury",
                },
                {
                        ID:          "cal_mtds",
                        Title:       "Medium-Term Debt Management Strategy — Tabling",
                        Date:        anchorDay(35),
                        Type:        CalBudgetPresentation,
                        Description: "Tabling of the 2026 Medium-Term Debt Management Strategy Statement (MTDS) before Parliament.",
                        SourceURL:   "https://www.treasury.go.ke/mtds-2026",
                        Country:     "KE",
                        Institution: "National Treasury",
                },
        }

        store.mu.Lock()
        defer store.mu.Unlock()
        for _, e := range samples {
                // Normalise country + date via Upsert (but call it directly to
                // avoid the per-call Lock/Unlock thrash).
                if e.Country == "" {
                        e.Country = store.country
                } else {
                        e.Country = strings.ToUpper(e.Country)
                }
                store.events[e.ID] = e
        }
}

// --- JSON helpers re-used by calendar (kept here so the file is
// self-contained for review; main.go defines the canonical writeJSON /
// writeError but we re-export a private alias to avoid an import cycle in
// tests that exercise CalendarStore directly without spinning up a
// server). ---

// writeCalendarJSON is exported via a lowercase alias so the calendar_test
// helpers can decode responses without depending on net/http internals.
func writeCalendarJSON(w http.ResponseWriter, status int, body any) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        _ = json.NewEncoder(w).Encode(body)
}

// EncodeInt is a tiny helper exposed for tests that want to assert the
// numeric shape of the "total" field without importing encoding/json
// themselves.
func EncodeInt(n int) string { return strconv.Itoa(n) }
