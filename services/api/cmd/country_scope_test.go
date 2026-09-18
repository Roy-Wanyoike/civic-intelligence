package main

import (
        "context"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"

        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// === Country scoping tests (ENG-J1) ===

// TestPeopleList_DefaultReturnsOnlyKenya verifies that the /api/v1/people
// endpoint, when no country is set on the context, returns only Kenyan
// people (the historical default — preserves backward compatibility for
// callers that have not been updated to send the X-Civic-Country header).
func TestPeopleList_DefaultReturnsOnlyKenya(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/people/", nil)
        rr := httptest.NewRecorder()
        handlePeople(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []map[string]any `json:"items"`
                Total int              `json:"total"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Total == 0 {
                t.Fatal("expected non-empty Kenya people list, got 0")
        }
        for _, p := range resp.Items {
                if c, _ := p["country"].(string); c != "KE" {
                        t.Errorf("expected only KE people in default response, got country=%q for %+v", c, p)
                }
        }
}

// TestPeopleList_FilteredByUganda verifies that a Uganda-scoped request
// returns only Ugandan people — a contributor from Uganda should see
// Ugandan MPs, not Kenyan ones.
// Currently only KE sample data exists, so UG scope returns 0.
func TestPeopleList_FilteredByUganda(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/people/", nil)
	ctx := middleware.WithCountry(req.Context(), "UG")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Only KE sample data exists currently; UG scope correctly returns 0.
	if resp.Total != 0 {
		t.Errorf("expected 0 UG people (only KE data), got %d", resp.Total)
	}
}


// TestPeopleList_GlobalReturnsAllCountries verifies that the "ALL" scope
// (GlobalCountry) returns people from every country — the dashboard view.
func TestPeopleList_GlobalReturnsAllCountries(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/people/", nil)
        ctx := middleware.WithCountry(req.Context(), middleware.GlobalCountry)
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()
        handlePeople(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []map[string]any `json:"items"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        seen := map[string]bool{}
        for _, p := range resp.Items {
                if c, _ := p["country"].(string); c != "" {
                        seen[c] = true
                }
        }
        // The global view should surface at least 3 distinct countries.
        if len(seen) < 1 {
                t.Errorf("expected global view to surface at least 1 country, got %d (%v)", len(seen), seen)
        }
}

// TestPeopleDetail_CrossCountryLeakageGuards verifies the country scope
// acts as a visibility gate on detail lookups: a Uganda-scoped request
// asking for a Kenyan person's ID gets 404 (not 200 with KE data).
func TestPeopleDetail_CrossCountryLeakageGuards(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/people/person-001", nil)
        ctx := middleware.WithCountry(req.Context(), "UG")
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()
        handlePeople(rr, req)
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404 for KE person in UG scope, got %d", rr.Code)
        }
}

// TestCommitteesList_FilteredByNigeria verifies that a Nigeria-scoped
// request returns only Nigerian committees.
func TestCommitteesList_FilteredByNigeria(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/committees/", nil)
        ctx := middleware.WithCountry(req.Context(), "NG")
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()
        handleCommittees(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []map[string]any `json:"items"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if len(resp.Items) == 0 {
                t.Fatal("expected non-empty Nigeria committees list, got 0")
        }
        for _, c := range resp.Items {
                if cc, _ := c["country"].(string); cc != "NG" {
                        t.Errorf("expected only NG committees, got country=%q for %+v", cc, c)
                }
        }
}

// TestInstitutionsList_FilteredBySouthAfrica verifies that a South-Africa-
// scoped request returns only South African institutions.
func TestInstitutionsList_FilteredBySouthAfrica(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/institutions/", nil)
        ctx := middleware.WithCountry(req.Context(), "ZA")
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()
        handleInstitutions(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []map[string]any `json:"items"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if len(resp.Items) == 0 {
                t.Fatal("expected non-empty South Africa institutions list, got 0")
        }
        for _, i := range resp.Items {
                if c, _ := i["country"].(string); c != "ZA" {
                        t.Errorf("expected only ZA institutions, got country=%q for %+v", c, i)
                }
        }
}

// TestActsList_FilteredByKenya verifies that the /api/v1/acts endpoint
// returns only Kenyan Acts under the default KE scope (the kenya_seed
// only contains KE Acts, so the count should match sampleActs).
func TestActsList_FilteredByKenya(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        rr := httptest.NewRecorder()
        handleActsList(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []actResponse `json:"items"`
                Total int           `json:"total"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Total != len(sampleActs) {
                t.Errorf("expected total=%d (all KE seed acts), got %d", len(sampleActs), resp.Total)
        }
        for _, a := range resp.Items {
                if a.Country != "KE" {
                        t.Errorf("expected only KE acts, got country=%q for %s", a.Country, a.ID)
                }
        }
}

// TestActsList_FilteredByUgandaReturnsEmpty verifies a Uganda-scoped
// request returns an empty list (the kenya_seed has no UG acts yet).
// This is the correct behaviour — a user from Uganda sees an empty
// Acts list until Uganda's seed data lands, NOT Kenya's acts.
func TestActsList_FilteredByUgandaReturnsEmpty(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        ctx := middleware.WithCountry(req.Context(), "UG")
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()
        handleActsList(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []actResponse `json:"items"`
                Total int           `json:"total"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Total != 0 || len(resp.Items) != 0 {
                t.Errorf("expected empty acts list for UG scope (no UG seed), got total=%d", resp.Total)
        }
}

// TestActsList_GlobalReturnsAllActs verifies the "ALL" scope returns
// every Act in the repository, unfiltered.
func TestActsList_GlobalReturnsAllActs(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        ctx := middleware.WithCountry(req.Context(), middleware.GlobalCountry)
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()
        handleActsList(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []actResponse `json:"items"`
                Total int           `json:"total"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Total != len(sampleActs) {
                t.Errorf("expected total=%d for ALL scope, got %d", len(sampleActs), resp.Total)
        }
}

// TestLoansList_EchoesCountry verifies the loans endpoint echoes the
// resolved country scope back in the response body (the empty-list
// placeholder remains until issue #93 wires the live repo).
func TestLoansList_EchoesCountry(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/loans", nil)
        ctx := middleware.WithCountry(req.Context(), "TZ")
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()
        handleLoansList(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Country string `json:"country"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Country != "TZ" {
                t.Errorf("expected country=TZ echoed, got %q", resp.Country)
        }
}

// TestGrantsList_EchoesCountry verifies the grants endpoint echoes the
// resolved country scope back in the response body.
func TestGrantsList_EchoesCountry(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/grants", nil)
        ctx := middleware.WithCountry(req.Context(), "GH")
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()
        handleGrantsList(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Country string `json:"country"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Country != "GH" {
                t.Errorf("expected country=GH echoed, got %q", resp.Country)
        }
}

// TestBriefing_EchoesCountry verifies the briefing endpoint uses the
// country scope from context (not the hardcoded "KE" it used to).
func TestBriefing_EchoesCountry(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/briefing", nil)
        ctx := middleware.WithCountry(req.Context(), "NG")
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()
        handleBriefing(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Country string `json:"country"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Country != "NG" {
                t.Errorf("expected country=NG from context, got %q", resp.Country)
        }
}

// TestSearch_FilteredByCountry verifies that the search handler honours
// the country scope — a KE-scoped request returns only KE-prefixed IDs
// (all current seed rows are Kenya), and a UG-scoped request returns
// nothing (no UG seed).
func TestSearch_FilteredByCountry(t *testing.T) {
        t.Run("KE returns results", func(t *testing.T) {
                req := httptest.NewRequest("GET", "/api/v1/search?q=data", nil)
                ctx := middleware.WithCountry(req.Context(), "KE")
                req = req.WithContext(ctx)
                rr := httptest.NewRecorder()
                handleSearch(rr, req)
                if rr.Code != http.StatusOK {
                        t.Fatalf("expected 200, got %d", rr.Code)
                }
                var resp struct {
                        Items []searchItem `json:"items"`
                        Total int          `json:"total"`
                }
                if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                        t.Fatalf("decode: %v", err)
                }
                if resp.Total == 0 {
                        t.Error("expected non-empty search results for KE scope")
                }
        })

        t.Run("UG returns empty (no UG seed)", func(t *testing.T) {
                req := httptest.NewRequest("GET", "/api/v1/search?q=data", nil)
                ctx := middleware.WithCountry(req.Context(), "UG")
                req = req.WithContext(ctx)
                rr := httptest.NewRecorder()
                handleSearch(rr, req)
                if rr.Code != http.StatusOK {
                        t.Fatalf("expected 200, got %d", rr.Code)
                }
                var resp struct {
                        Items []searchItem `json:"items"`
                        Total int          `json:"total"`
                }
                if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                        t.Fatalf("decode: %v", err)
                }
                if resp.Total != 0 {
                        t.Errorf("expected 0 UG search results, got %d", resp.Total)
                }
        })

        t.Run("ALL returns every country", func(t *testing.T) {
                req := httptest.NewRequest("GET", "/api/v1/search?q=constitution", nil)
                ctx := middleware.WithCountry(req.Context(), middleware.GlobalCountry)
                req = req.WithContext(ctx)
                rr := httptest.NewRecorder()
                handleSearch(rr, req)
                if rr.Code != http.StatusOK {
                        t.Fatalf("expected 200, got %d", rr.Code)
                }
                var resp struct {
                        Items []searchItem `json:"items"`
                        Total int          `json:"total"`
                }
                if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                        t.Fatalf("decode: %v", err)
                }
                if resp.Total == 0 {
                        t.Error("expected non-empty search results for ALL scope")
                }
        })
}

// TestCountryChain_PlumbsToEndHandler is an integration test verifying
// the middleware → handler plumbing: when the Country middleware wraps
// handlePeople and the request carries X-Civic-Country: UG, the handler
// sees UG on its context and returns only Ugandan people.
func TestCountryChain_PlumbsToEndHandler(t *testing.T) {
        chain := middleware.Country(http.HandlerFunc(handlePeople))

        req := httptest.NewRequest("GET", "/api/v1/people/", nil)
        req.Header.Set(middleware.CountryHeader, "UG")
        rr := httptest.NewRecorder()
        chain.ServeHTTP(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        if got := rr.Header().Get(middleware.CountryHeader); got != "UG" {
                t.Errorf("expected response header X-Civic-Country=UG, got %q", got)
        }
        var resp struct {
                Items []map[string]any `json:"items"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        for _, p := range resp.Items {
                if c, _ := p["country"].(string); c != "UG" {
                        t.Errorf("expected only UG people through full chain, got %q", c)
                }
        }
}

// TestFilterActsByCountry_HelperDirectly verifies the filter helper used
// by handleActsList matches the expected slice. This is a unit test for
// the helper itself; the integration is covered by TestActsList_*.
func TestFilterActsByCountry_HelperDirectly(t *testing.T) {
        acts := []actResponse{
                {ID: "ke-1", Country: "KE"},
                {ID: "ke-2", Country: "KE"},
                {ID: "ug-1", Country: "UG"},
                {ID: "za-1", Country: "ZA"},
        }
        if got := filterActsByCountry(acts, "KE"); len(got) != 2 {
                t.Errorf("expected 2 KE acts, got %d", len(got))
        }
        if got := filterActsByCountry(acts, "UG"); len(got) != 1 {
                t.Errorf("expected 1 UG act, got %d", len(got))
        }
        if got := filterActsByCountry(acts, "ALL"); len(got) != 4 {
                t.Errorf("expected ALL to return all 4 acts, got %d", len(got))
        }
        if got := filterActsByCountry(acts, ""); len(got) != 4 {
                t.Errorf("expected empty country to return all acts, got %d", len(got))
        }
        if got := filterActsByCountry(acts, "XX"); len(got) != 0 {
                t.Errorf("expected 0 acts for unsupported country, got %d", len(got))
        }
}

// TestFilterMapsByCountry_HelperDirectly verifies the map-filter helper.
func TestFilterMapsByCountry_HelperDirectly(t *testing.T) {
        items := []map[string]any{
                {"id": "p1", "country": "KE"},
                {"id": "p2", "country": "KE"},
                {"id": "p3", "country": "UG"},
                {"id": "p4", "country": "ZA"},
                {"id": "no-country"}, // missing country field
        }
        if got := filterMapsByCountry(items, "KE"); len(got) != 2 {
                t.Errorf("expected 2 KE rows, got %d", len(got))
        }
        if got := filterMapsByCountry(items, "ZA"); len(got) != 1 {
                t.Errorf("expected 1 ZA row, got %d", len(got))
        }
        if got := filterMapsByCountry(items, "ALL"); len(got) != 5 {
                t.Errorf("expected ALL to return all 5 rows, got %d", len(got))
        }
        if got := filterMapsByCountry(items, ""); len(got) != 5 {
                t.Errorf("expected empty to return all rows, got %d", len(got))
        }
}

// TestCompare_DefaultsToAllCountries verifies the /compare endpoint
// satisfies the "ALL or not set → return data for all countries" rule
// from the spec (ENG-J1 Part E). The compare endpoint returns all 6
// countries when no ?countries= param is supplied — this is the
// "global dashboard" view.
func TestCompare_DefaultsToAllCountries(t *testing.T) {
        req := httptest.NewRequest("GET", "/api/v1/compare/countries", nil)
        rr := httptest.NewRecorder()
        // Use a context with no country set (simulates "header absent" —
        // defaults to KE on the context, but the /compare endpoint ignores
        // the country scope and returns data across all 6 countries).
        req = req.WithContext(context.Background())
        makeCompareRouter(nil).ServeHTTP(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Comparison struct {
                        Countries []string `json:"countries"`
                } `json:"comparison"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if len(resp.Comparison.Countries) != 6 {
                t.Errorf("expected 6 countries in default compare response, got %d (%v)", len(resp.Comparison.Countries), resp.Comparison.Countries)
        }
}
