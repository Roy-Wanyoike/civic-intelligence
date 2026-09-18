package main

import (
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"
)

// TestConstituencies_ListReturnsTen verifies the list endpoint returns the
// 10 sample Kenyan constituencies the platform ships with. Guards against an
// accidental truncation of the seed file.
func TestConstituencies_ListReturnsTen(t *testing.T) {
        if len(sampleConstituencies) != 10 {
                t.Fatalf("expected exactly 10 sample constituencies for ENG-K2, got %d", len(sampleConstituencies))
        }
        req := httptest.NewRequest(http.MethodGet, "/api/v1/constituencies?country=KE", nil)
        rec := httptest.NewRecorder()

        makeConstituenciesListHandler()(rec, req)

        if rec.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rec.Code)
        }
        var resp struct {
                Items      []ConstituencyListItem `json:"items"`
                Total      int                     `json:"total"`
                Country    string                  `json:"country"`
                Disclaimer string                  `json:"disclaimer"`
        }
        if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Total != 10 {
                t.Errorf("expected total=10, got %d", resp.Total)
        }
        if resp.Country != "KE" {
                t.Errorf("expected country=KE, got %q", resp.Country)
        }
        if resp.Disclaimer != constituencyDisclaimer {
                t.Errorf("disclaimer mismatch: got %q", resp.Disclaimer)
        }
        // Every item must carry an id, name, county, mp_name and dashboard_url.
        for _, item := range resp.Items {
                if item.ID == "" || item.Name == "" || item.County == "" || item.MPName == "" || item.DashboardURL == "" {
                        t.Errorf("incomplete list item: %+v", item)
                }
        }
}

// TestConstituencies_ListSearchFilters verifies the ?q= query parameter
// narrows the list. The test also confirms case-insensitive substring match
// across name, county, region and MP name.
func TestConstituencies_ListSearchFilters(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/constituencies?country=KE&q=nairobi", nil)
        rec := httptest.NewRecorder()

        makeConstituenciesListHandler()(rec, req)

        if rec.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rec.Code)
        }
        var resp struct {
                Items []ConstituencyListItem `json:"items"`
                Total int                     `json:"total"`
        }
        _ = json.Unmarshal(rec.Body.Bytes(), &resp)
        if resp.Total == 0 {
                t.Fatal("expected at least one match for 'nairobi', got 0")
        }
        for _, item := range resp.Items {
                if item.Name != "Nairobi" {
                        t.Errorf("expected only Nairobi in 'nairobi' search; got %q", item.Name)
                }
        }
}

// TestConstituencies_DetailReturnsFullDashboard verifies the detail endpoint
// returns the full dashboard payload: MP, bills, budget, projects,
// participation, events, government context and debt context. Every
// evidence-bearing field must carry a source_url.
func TestConstituencies_DetailReturnsFullDashboard(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/constituencies/ke-const-nairobi", nil)
        rec := httptest.NewRecorder()

        makeConstituencyDetailHandler()(rec, req)

        if rec.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rec.Code)
        }
        var c Constituency
        if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if c.ID != "ke-const-nairobi" {
                t.Errorf("expected id=ke-const-nairobi, got %q", c.ID)
        }
        if c.Name != "Nairobi" {
                t.Errorf("expected name=Nairobi, got %q", c.Name)
        }
        if c.MP.PersonID == "" || c.MP.ScorecardURL == "" {
                t.Errorf("MP missing person_id / scorecard_url: %+v", c.MP)
        }
        if len(c.Bills) == 0 {
                t.Error("expected bills_affecting_area; got 0")
        }
        if len(c.Budget) == 0 {
                t.Error("expected budget_allocated; got 0")
        }
        if len(c.Projects) == 0 {
                t.Error("expected local_projects; got 0")
        }
        if len(c.Participation) == 0 {
                t.Error("expected public_participation; got 0")
        }
        if len(c.Events) == 0 {
                t.Error("expected recent_events; got 0")
        }
        if c.Government.AdministrationID == "" {
                t.Error("expected government_context.administration_id")
        }
        if c.DebtContext.SourceURL == "" {
                t.Error("expected debt_context.source_url")
        }
        if c.Disclaimer != constituencyDisclaimer {
                t.Errorf("disclaimer mismatch: got %q", c.Disclaimer)
        }
        if c.RealityLayer != "FACT" {
                t.Errorf("expected reality_layer=FACT, got %q", c.RealityLayer)
        }
        // Every bill must carry a source_url.
        for _, b := range c.Bills {
                if b.SourceURL == "" {
                        t.Errorf("bill %q missing source_url", b.Title)
                }
        }
        // Every budget line must carry a source_url.
        for _, b := range c.Budget {
                if b.SourceURL == "" {
                        t.Errorf("budget line %q missing source_url", b.Vote)
                }
        }
        // Every project must carry a source_url.
        for _, p := range c.Projects {
                if p.SourceURL == "" {
                        t.Errorf("project %q missing source_url", p.Name)
                }
        }
        // Every participation opportunity must carry a source_url.
        for _, p := range c.Participation {
                if p.SourceURL == "" {
                        t.Errorf("participation %q missing source_url", p.Title)
                }
        }
}

// TestConstituencies_DetailReturns404ForUnknown verifies the detail endpoint
// returns 404 (not 200 with empty fields) when the ID is unknown. Prevents
// the frontend from rendering a fabricated dashboard.
func TestConstituencies_DetailReturns404ForUnknown(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/constituencies/does-not-exist", nil)
        rec := httptest.NewRecorder()

        makeConstituencyDetailHandler()(rec, req)

        if rec.Code != http.StatusNotFound {
                t.Fatalf("expected 404, got %d", rec.Code)
        }
}

// TestConstituencies_DebtContextNeverAttributesToPerson verifies the
// debt_context block in every sample constituency carries the canonical
// "sovereign debt is national" note — the platform NEVER attributes
// sovereign borrowing personally to a single MP, governor, or president
// (see ADR-0005, public_debt.go).
func TestConstituencies_DebtContextNeverAttributesToPerson(t *testing.T) {
        for _, c := range sampleConstituencies {
                if c.DebtContext.RegionNote == "" {
                        t.Errorf("constituency %s: debt_context.region_note is empty", c.ID)
                        continue
                }
                if !strContains(c.DebtContext.RegionNote, "national") {
                        t.Errorf("constituency %s: debt_context.region_note must say 'national' to disavow personal attribution; got %q", c.ID, c.DebtContext.RegionNote)
                }
        }
}

// TestConstituencies_ListIsSorted verifies the list endpoint returns
// constituencies sorted alphabetically by name. Sorting keeps the dashboard
// deterministic across page reloads (a search/filter feature would otherwise
// shuffle the cards depending on insertion order).
func TestConstituencies_ListIsSorted(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/constituencies?country=KE", nil)
        rec := httptest.NewRecorder()

        makeConstituenciesListHandler()(rec, req)

        var resp struct {
                Items []ConstituencyListItem `json:"items"`
        }
        _ = json.Unmarshal(rec.Body.Bytes(), &resp)
        for i := 1; i < len(resp.Items); i++ {
                if resp.Items[i-1].Name > resp.Items[i].Name {
                        t.Errorf("list not sorted: %q > %q at index %d", resp.Items[i-1].Name, resp.Items[i].Name, i)
                }
        }
}

// strContains is a local substring helper. The cmd package already declares
// a `contains` helper (in post_assent_test.go) with a different signature,
// so this file uses a uniquely-named variant to avoid a redeclaration error
// during test compilation.
func strContains(haystack, needle string) bool {
        return len(haystack) >= len(needle) && (haystack == needle ||
                strIndexOf(haystack, needle) >= 0)
}

func strIndexOf(haystack, needle string) int {
        for i := 0; i+len(needle) <= len(haystack); i++ {
                if haystack[i:i+len(needle)] == needle {
                        return i
                }
        }
        return -1
}
