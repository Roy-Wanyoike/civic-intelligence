package main

import (
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "net/url"
        "strings"
        "testing"
        "time"
)

// fixedCalendarAnchor is a deterministic "today" used by tests so the
// sample data (which is computed relative to the anchor) is reproducible.
// 2026-09-15 is a Tuesday; chosen so the weekly Friday Gazette events
// land on real Fridays (2026-09-18, 2026-10-02, ...).
var fixedCalendarAnchor = time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC)

// newSeededCalendarStore returns a fresh CalendarStore seeded with the
// 26 sample events anchored at fixedCalendarAnchor.
func newSeededCalendarStore(t *testing.T) *CalendarStore {
        t.Helper()
        s := NewCalendarStore(fixedCalendarAnchor, "KE")
        SeedCalendarSampleData(s, fixedCalendarAnchor)
        return s
}

func TestCalendarStore_SeededWithSampleData(t *testing.T) {
        s := newSeededCalendarStore(t)
        all := s.List("KE")
        // The spec calls for 20-30 sample events; the seeder ships 26.
        if len(all) < 20 || len(all) > 30 {
                t.Fatalf("expected 20-30 seeded events, got %d", len(all))
        }
        // Every event must carry the canonical fields.
        for _, e := range all {
                if e.ID == "" || e.Title == "" || e.Date == "" || e.Type == "" {
                        t.Errorf("event %+v missing required field", e)
                }
                if !calValidEventTypes[e.Type] {
                        t.Errorf("event %s has invalid type %q", e.ID, e.Type)
                }
                if e.Country != "KE" {
                        t.Errorf("event %s country: expected KE, got %s", e.ID, e.Country)
                }
                if _, err := time.Parse("2006-01-02", e.Date); err != nil {
                        t.Errorf("event %s has invalid date %q: %v", e.ID, e.Date, err)
                }
        }
}

func TestCalendarStore_EventsForMonth(t *testing.T) {
        s := newSeededCalendarStore(t)
        events, err := s.EventsForMonth("2026-09", "KE")
        if err != nil {
                t.Fatalf("EventsForMonth: %v", err)
        }
        // Every event in September 2026 must fall within the month.
        for _, e := range events {
                d, err := time.Parse("2006-01-02", e.Date)
                if err != nil {
                        t.Fatalf("invalid date: %v", err)
                }
                if d.Year() != 2026 || d.Month() != time.September {
                        t.Errorf("event %s dated %s should be in September 2026", e.ID, e.Date)
                }
        }
        if len(events) == 0 {
                t.Fatal("expected at least one September event, got 0")
        }
}

func TestCalendarStore_EventsForMonth_NormalisesInputs(t *testing.T) {
        s := newSeededCalendarStore(t)
        cases := []string{"2026-09", "2026-9", "2026/09", "202609"}
        for _, in := range cases {
                events, err := s.EventsForMonth(in, "KE")
                if err != nil {
                        t.Errorf("EventsForMonth(%q): %v", in, err)
                        continue
                }
                if len(events) == 0 {
                        t.Errorf("EventsForMonth(%q): expected events, got 0", in)
                }
        }
}

func TestCalendarStore_EventsForMonth_InvalidInput(t *testing.T) {
        s := newSeededCalendarStore(t)
        _, err := s.EventsForMonth("not-a-month", "KE")
        if err == nil {
                t.Fatal("expected error for invalid month")
        }
        if !strings.Contains(err.Error(), "invalid month") {
                t.Errorf("expected 'invalid month' in error, got %v", err)
        }
}

func TestCalendarStore_EventsForMonth_EmptyUsesAnchor(t *testing.T) {
        s := newSeededCalendarStore(t)
        events, err := s.EventsForMonth("", "KE")
        if err != nil {
                t.Fatalf("EventsForMonth(''): %v", err)
        }
        if len(events) == 0 {
                t.Fatal("expected events for anchor's month, got 0")
        }
}

func TestCalendarStore_Today(t *testing.T) {
        s := newSeededCalendarStore(t)
        today := s.Today("KE")
        for _, e := range today {
                if e.Date != fixedCalendarAnchor.Format("2006-01-02") {
                        t.Errorf("today event %s dated %s, expected %s", e.ID, e.Date, fixedCalendarAnchor.Format("2006-01-02"))
                }
        }
}

func TestCalendarStore_UpcomingSortedAndLimited(t *testing.T) {
        s := newSeededCalendarStore(t)
        events := s.Upcoming(5, "KE")
        if len(events) > 5 {
                t.Fatalf("expected at most 5 events, got %d", len(events))
        }
        today := fixedCalendarAnchor.Format("2006-01-02")
        for _, e := range events {
                if e.Date < today {
                        t.Errorf("upcoming event %s dated %s is in the past (today=%s)", e.ID, e.Date, today)
                }
        }
        // Verify ascending order.
        for i := 1; i < len(events); i++ {
                if events[i].Date < events[i-1].Date {
                        t.Errorf("upcoming events not sorted: %s before %s", events[i-1].Date, events[i].Date)
                }
        }
}

func TestCalendarStore_Upcoming_LimitClamping(t *testing.T) {
        s := newSeededCalendarStore(t)
        if got := s.Upcoming(-1, "KE"); len(got) != 10 {
                t.Errorf("Upcoming(-1): expected default 10, got %d", len(got))
        }
        if got := s.Upcoming(1000, "KE"); len(got) > 100 {
                t.Errorf("Upcoming(1000): expected clamp to 100, got %d", len(got))
        }
}

func TestCalendarStore_CountryFilter(t *testing.T) {
        s := newSeededCalendarStore(t)
        // Add a non-KE event and confirm it is excluded by the country filter.
        s.Upsert(CalendarEvent{
                ID:      "cal_tz_test",
                Title:   "Tanzania Parliament Sitting",
                Date:    fixedCalendarAnchor.AddDate(0, 0, 1).Format("2006-01-02"),
                Type:    CalParliamentSession,
                Country: "TZ",
        })
        keEvents := s.List("KE")
        for _, e := range keEvents {
                if e.Country != "KE" {
                        t.Errorf("country filter leaked event %s (country=%s)", e.ID, e.Country)
                }
        }
        tzEvents := s.List("TZ")
        if len(tzEvents) == 0 {
                t.Fatal("expected at least 1 TZ event")
        }
        for _, e := range tzEvents {
                if e.Country != "TZ" {
                        t.Errorf("TZ filter leaked event %s (country=%s)", e.ID, e.Country)
                }
        }
}

func TestFilterByTypes(t *testing.T) {
        s := newSeededCalendarStore(t)
        all := s.List("KE")
        filtered := FilterByTypes(all, []CalendarEventType{CalBillReading, CalCourtHearing})
        for _, e := range filtered {
                if e.Type != CalBillReading && e.Type != CalCourtHearing {
                        t.Errorf("filter leaked event %s of type %s", e.ID, e.Type)
                }
        }
        if len(filtered) == 0 {
                t.Fatal("expected at least one filtered event, got 0")
        }
}

func TestParseTypesQuery(t *testing.T) {
        cases := []struct {
                name    string
                query   string
                wantLen int
                ok      bool
        }{
                {"empty", "", 0, true},
                {"single", "parliament_session", 1, true},
                {"multi", "parliament_session,bill_reading", 2, true},
                {"spaces", "  parliament_session , bill_reading ", 2, true},
                {"invalid", "spaceship", 0, false},
                {"mixed-invalid", "parliament_session,spaceship", 0, false},
        }
        for _, c := range cases {
                t.Run(c.name, func(t *testing.T) {
                        // Build the URL with url.Values so spaces are percent-encoded
                        // (httptest.NewRequest rejects raw spaces in the path).
                        v := url.Values{}
                        v.Set("types", c.query)
                        req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?"+v.Encode(), nil)
                        got, ok := parseTypesQuery(req)
                        if ok != c.ok {
                                t.Errorf("ok: expected %v, got %v", c.ok, ok)
                        }
                        if c.ok && len(got) != c.wantLen {
                                t.Errorf("len: expected %d, got %d", c.wantLen, len(got))
                        }
                })
        }
}

func TestParseMonthRange(t *testing.T) {
        start, end, err := parseMonthRange("2026-09", fixedCalendarAnchor)
        if err != nil {
                t.Fatalf("parseMonthRange: %v", err)
        }
        wantStart := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
        wantEnd := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
        if !start.Equal(wantStart) {
                t.Errorf("start: expected %v, got %v", wantStart, start)
        }
        if !end.Equal(wantEnd) {
                t.Errorf("end: expected %v, got %v", wantEnd, end)
        }
}

// --- HTTP handler tests ---

func TestCalendarHandler_Month(t *testing.T) {
        store := newSeededCalendarStore(t)
        handler := makeCalendarHandler(store)
        req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?month=2026-09&country=KE", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp struct {
                Month string          `json:"month"`
                Items []CalendarEvent `json:"items"`
                Total int             `json:"total"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("invalid JSON: %v", err)
        }
        if resp.Month != "2026-09" {
                t.Errorf("expected month '2026-09', got %q", resp.Month)
        }
        if resp.Total == 0 {
                t.Error("expected non-zero events for September 2026")
        }
}

func TestCalendarHandler_Today(t *testing.T) {
        store := newSeededCalendarStore(t)
        handler := makeCalendarHandler(store)
        req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/today?country=KE", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp struct {
                Date  string          `json:"date"`
                Items []CalendarEvent `json:"items"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("invalid JSON: %v", err)
        }
        if resp.Date != "2026-09-15" {
                t.Errorf("expected date '2026-09-15', got %q", resp.Date)
        }
        for _, e := range resp.Items {
                if e.Date != "2026-09-15" {
                        t.Errorf("today event %s dated %s, expected 2026-09-15", e.ID, e.Date)
                }
        }
}

func TestCalendarHandler_Upcoming(t *testing.T) {
        store := newSeededCalendarStore(t)
        handler := makeCalendarHandler(store)
        req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/upcoming?country=KE&limit=5", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp struct {
                Items []CalendarEvent `json:"items"`
                Total int             `json:"total"`
                Limit int             `json:"limit"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("invalid JSON: %v", err)
        }
        if resp.Limit != 5 {
                t.Errorf("expected limit 5, got %d", resp.Limit)
        }
        if resp.Total > 5 {
                t.Errorf("expected at most 5 items, got %d", resp.Total)
        }
}

func TestCalendarHandler_TypesFilter(t *testing.T) {
        store := newSeededCalendarStore(t)
        handler := makeCalendarHandler(store)
        req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/upcoming?country=KE&limit=20&types=bill_reading", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp struct {
                Items []CalendarEvent `json:"items"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("invalid JSON: %v", err)
        }
        for _, e := range resp.Items {
                if e.Type != CalBillReading {
                        t.Errorf("type filter leaked event %s of type %s", e.ID, e.Type)
                }
        }
}

func TestCalendarHandler_InvalidTypes(t *testing.T) {
        store := newSeededCalendarStore(t)
        handler := makeCalendarHandler(store)
        req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?month=2026-09&types=spaceship", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for invalid types, got %d (body=%s)", rr.Code, rr.Body.String())
        }
}

func TestCalendarHandler_InvalidMonth(t *testing.T) {
        store := newSeededCalendarStore(t)
        handler := makeCalendarHandler(store)
        req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar?month=garbage", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for invalid month, got %d (body=%s)", rr.Code, rr.Body.String())
        }
}

func TestCalendarHandler_MethodNotAllowed(t *testing.T) {
        store := newSeededCalendarStore(t)
        handler := makeCalendarHandler(store)
        req := httptest.NewRequest(http.MethodPost, "/api/v1/calendar", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusMethodNotAllowed {
                t.Errorf("expected 405 for POST, got %d", rr.Code)
        }
        if rr.Header().Get("Allow") != "GET" {
                t.Errorf("expected Allow: GET, got %q", rr.Header().Get("Allow"))
        }
}

func TestCalendarHandler_UnknownSubPath(t *testing.T) {
        store := newSeededCalendarStore(t)
        handler := makeCalendarHandler(store)
        req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/unknown", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404 for unknown sub-path, got %d", rr.Code)
        }
}

func TestNewCalendarEventID_PrefixedAndUnique(t *testing.T) {
        seen := make(map[string]bool, 100)
        for i := 0; i < 100; i++ {
                id := newCalendarEventID()
                if !strings.HasPrefix(id, "cal_") {
                        t.Fatalf("id %s missing cal_ prefix", id)
                }
                if seen[id] {
                        t.Fatalf("collision at iteration %d: %s", i, id)
                }
                seen[id] = true
        }
}

func TestSeedCalendarSampleData_Idempotent(t *testing.T) {
        s := NewCalendarStore(fixedCalendarAnchor, "KE")
        SeedCalendarSampleData(s, fixedCalendarAnchor)
        firstCount := len(s.List("KE"))
        // Re-seeding should NOT duplicate (sample IDs are deterministic).
        SeedCalendarSampleData(s, fixedCalendarAnchor)
        secondCount := len(s.List("KE"))
        if firstCount != secondCount {
                t.Errorf("idempotency: re-seed changed count from %d to %d", firstCount, secondCount)
        }
}
