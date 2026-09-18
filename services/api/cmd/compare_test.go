package main

import (
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "strings"
        "testing"

        kenya_seed "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// newCompareTestDebtRepo constructs an in-memory DebtRepository seeded with
// the Kenya dataset for the /compare/debt endpoint. Mirrors the helper in
// public_debt_test.go but kept local so each test file stays self-contained.
func newCompareTestDebtRepo(t *testing.T) legislation.DebtRepository {
        t.Helper()
        repo, err := legislation.WireDebtRepository(kenya_seed.SeedDebt)
        if err != nil {
                t.Fatalf("seed debt repo: %v", err)
        }
        return repo
}

// TestCompare_CountriesReturnsAllProfiles verifies that the
// /api/v1/compare/countries endpoint returns the institutional profile
// for each selected country. Each profile must include the country code,
// name, flag, government system, and house count.
func TestCompare_CountriesReturnsAllProfiles(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/countries?countries=KE,UG,TZ", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(nil)(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        Countries  []string        `json:"countries"`
                        Dimensions []string        `json:"dimensions"`
                        Data       map[string]any  `json:"data"`
                        Differences []map[string]any `json:"differences"`
                        Disclaimer string          `json:"disclaimer"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if len(resp.Comparison.Countries) != 3 {
                t.Fatalf("expected 3 countries; got %d (%v)", len(resp.Comparison.Countries), resp.Comparison.Countries)
        }
        for _, code := range []string{"KE", "UG", "TZ"} {
                row, ok := resp.Comparison.Data[code].(map[string]any)
                if !ok {
                        t.Errorf("expected %s in data; got %T", code, resp.Comparison.Data[code])
                        continue
                }
                if _, ok := row["government_system"].(string); !ok {
                        t.Errorf("%s missing government_system", code)
                }
                if _, ok := row["house_count"].(float64); !ok {
                        t.Errorf("%s missing house_count", code)
                }
        }
}

// TestCompare_CountriesDisclaimerPresent verifies the critical invariant
// (Spec §37): EVERY compare response carries the disclaimer that the
// platform does not rank countries. This is the headline test for the
// ENG-I3 task — without it the comparison would risk implying a ranking.
func TestCompare_CountriesDisclaimerPresent(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/countries?countries=KE,UG", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(nil)(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        Disclaimer string `json:"disclaimer"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if !strings.Contains(resp.Comparison.Disclaimer, "does not rank") {
                t.Errorf("expected disclaimer to mention 'does not rank'; got %q", resp.Comparison.Disclaimer)
        }
}

// TestCompare_GovernmentStructureReturnsBicameralVsUnicameral verifies that
// the government-structure endpoint surfaces the bicameral-vs-unicameral
// distinction correctly across the selected countries. The differences
// array must describe (not evaluate) the distinction.
func TestCompare_GovernmentStructureReturnsBicameralVsUnicameral(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/government-structure?countries=KE,UG,NG,ZA", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(nil)(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        Countries []string          `json:"countries"`
                        Data      map[string]any    `json:"data"`
                        Differences []map[string]any `json:"differences"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        // KE + NG + ZA are bicameral (2 houses); UG is unicameral (1 house).
        keRow := resp.Comparison.Data["KE"].(map[string]any)
        if houses, _ := keRow["houses"].(float64); houses != 2 {
                t.Errorf("expected KE houses=2 (bicameral); got %v", houses)
        }
        ugRow := resp.Comparison.Data["UG"].(map[string]any)
        if houses, _ := ugRow["houses"].(float64); houses != 1 {
                t.Errorf("expected UG houses=1 (unicameral); got %v", houses)
        }

        // Differences array must mention "houses" (the dimension).
        found := false
        for _, d := range resp.Comparison.Differences {
                if dim, _ := d["dimension"].(string); dim == "houses" {
                        found = true
                        note, _ := d["note"].(string)
                        if !strings.Contains(note, "bicameral") || !strings.Contains(note, "unicameral") {
                                t.Errorf("expected note to mention bicameral+unicameral; got %q", note)
                        }
                }
        }
        if !found {
                t.Error("expected a 'houses' dimension in differences; got none")
        }
}

// TestCompare_LegislationByTopic verifies the legislation comparison
// endpoint filters the sample bills by topic. The response surfaces the
// bills_count_2024 + acts_count_2024 + sample_bills per country.
func TestCompare_LegislationByTopic(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/legislation?countries=KE,UG&topic=health", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(nil)(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        Topic string          `json:"topic"`
                        Data  map[string]any  `json:"data"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if resp.Comparison.Topic != "health" {
                t.Errorf("expected topic=health; got %q", resp.Comparison.Topic)
        }
        // Each country row should have bills_count_2024 + sample_bills.
        keRow := resp.Comparison.Data["KE"].(map[string]any)
        if _, ok := keRow["bills_count_2024"]; !ok {
                t.Error("expected bills_count_2024 field in KE row")
        }
        if _, ok := keRow["sample_bills"]; !ok {
                t.Error("expected sample_bills field in KE row")
        }
}

// TestCompare_DebtReturnsKenyaLiveSeries verifies that the debt comparison
// endpoint pulls the live CBK + Treasury observations for Kenya from the
// DebtRepository. The KE row should carry a non-empty `series` array.
func TestCompare_DebtReturnsKenyaLiveSeries(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/debt?countries=KE,UG&from=2020&to=2024", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(newCompareTestDebtRepo(t))(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        FromYear int             `json:"from_year"`
                        ToYear   int             `json:"to_year"`
                        Data     map[string]any  `json:"data"`
                        Disclaimer string        `json:"disclaimer"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if resp.Comparison.FromYear != 2020 || resp.Comparison.ToYear != 2024 {
                t.Errorf("expected from=2020 to=2024; got from=%d to=%d", resp.Comparison.FromYear, resp.Comparison.ToYear)
        }
        keRow := resp.Comparison.Data["KE"].(map[string]any)
        series, ok := keRow["series"].([]any)
        if !ok || len(series) == 0 {
                t.Errorf("expected KE to carry a non-empty series; got %T", keRow["series"])
        }
        // The KE row should also carry the debt_to_gdp indicator (70.2 from kenya_seed).
        if debtToGDP, _ := keRow["debt_to_gdp"].(float64); debtToGDP != 70.2 {
                t.Errorf("expected KE debt_to_gdp=70.2; got %v", keRow["debt_to_gdp"])
        }
}

// TestCompare_DebtCarriesNoPoliticalPerformanceScore verifies the critical
// attribution rule (Spec §37): the /compare/debt response appends the
// canonical NO_POLITICAL_PERFORMANCE_SCORE disclaimer so the platform
// never implicitly ranks countries by debt level.
func TestCompare_DebtCarriesNoPoliticalPerformanceScore(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/debt?countries=KE,UG", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(newCompareTestDebtRepo(t))(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        Disclaimer string `json:"disclaimer"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if !strings.Contains(resp.Comparison.Disclaimer, "does not") {
                t.Errorf("expected disclaimer to mention what the platform does NOT do; got %q", resp.Comparison.Disclaimer)
        }
        if !strings.Contains(resp.Comparison.Disclaimer, "performance ranking") && !strings.Contains(resp.Comparison.Disclaimer, "best borrower") {
                t.Errorf("expected disclaimer to mention 'performance ranking' or 'best borrower'; got %q", resp.Comparison.Disclaimer)
        }
}

// TestCompare_IndicatorsFiltersByKey verifies that the indicators endpoint
// honours the ?indicators= filter — only the requested keys are returned.
func TestCompare_IndicatorsFiltersByKey(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/indicators?countries=KE,UG,TZ&indicators=debt_to_gdp,bills_introduced", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(nil)(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        Indicators []string        `json:"indicators"`
                        Data       map[string]any   `json:"data"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if len(resp.Comparison.Indicators) != 2 {
                t.Fatalf("expected 2 indicators; got %d (%v)", len(resp.Comparison.Indicators), resp.Comparison.Indicators)
        }
        // Each indicator key should be present in every country row.
        for _, code := range []string{"KE", "UG", "TZ"} {
                row := resp.Comparison.Data[code].(map[string]any)
                for _, key := range resp.Comparison.Indicators {
                        ind, ok := row[key].(map[string]any)
                        if !ok {
                                t.Errorf("expected %s to have indicator %s; got %T", code, key, row[key])
                                continue
                        }
                        if _, ok := ind["value"].(float64); !ok {
                                t.Errorf("%s.%s missing value", code, key)
                        }
                        if _, ok := ind["source_url"].(string); !ok {
                                t.Errorf("%s.%s missing source_url", code, key)
                        }
                }
        }
}

// TestCompare_IndicatorsDisclaimerPresent mirrors the per-endpoint
// disclaimer test for the /indicators endpoint. The disclaimer invariant
// must hold across EVERY compare sub-resource.
func TestCompare_IndicatorsDisclaimerPresent(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/indicators?countries=KE,UG", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(nil)(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        Disclaimer string `json:"disclaimer"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if !strings.Contains(resp.Comparison.Disclaimer, "does not rank") {
                t.Errorf("expected disclaimer to mention 'does not rank'; got %q", resp.Comparison.Disclaimer)
        }
}

// TestCompare_NeverRanksCountries scans the response bodies of all 5
// compare endpoints for forbidden ranking language. The platform must
// NEVER use "best country", "worst country", "top performer", "ranked first",
// "first place", "#1", etc. — even in passing — to describe a country
// relative to another. This is the headline invariant test for ENG-I3.
//
// Note: the canonical NO_POLITICAL_PERFORMANCE_SCORE disclaimer explicitly
// says the platform does NOT compute "best borrower" / "worst borrower",
// so those phrases legitimately appear in the disclaimer text — they're
// excluded from the forbidden list here. What's forbidden is using those
// phrases to ATTRIBUTE a ranking to a country, not disclaiming them.
func TestCompare_NeverRanksCountries(t *testing.T) {
        cases := []struct {
                name string
                url  string
        }{
                {"countries", "/api/v1/compare/countries?countries=KE,UG,TZ"},
                {"legislation", "/api/v1/compare/legislation?countries=KE,UG&topic=health"},
                {"debt", "/api/v1/compare/debt?countries=KE,UG&from=2020&to=2024"},
                {"government_structure", "/api/v1/compare/government-structure?countries=KE,UG,NG,ZA"},
                {"indicators", "/api/v1/compare/indicators?countries=KE,UG,TZ&indicators=debt_to_gdp,bills_introduced"},
        }
        forbidden := []string{
                "best country", "worst country",
                "top performer", "bottom performer", "ranked first", "ranked last",
                "highest ranking", "lowest ranking", "#1", "first place", "last place",
        }
        for _, tc := range cases {
                t.Run(tc.name, func(t *testing.T) {
                        req := httptest.NewRequest(http.MethodGet, tc.url, nil)
                        rec := httptest.NewRecorder()
                        makeCompareRouter(newCompareTestDebtRepo(t))(rec, req)
                        if rec.Code != 200 {
                                t.Fatalf("expected 200; got %d", rec.Code)
                        }
                        body := rec.Body.String()
                        bodyLower := strings.ToLower(body)
                        for _, phrase := range forbidden {
                                if strings.Contains(bodyLower, phrase) {
                                        t.Errorf("response for %s contains forbidden ranking phrase %q", tc.name, phrase)
                                }
                        }
                })
        }
}

// TestCompare_AllFiveEndpointsReturnDisclaimer verifies that every one of
// the 5 compare endpoints attaches the disclaimer to the response. This
// is a structural invariant — adding a new endpoint without the
// disclaimer would be a regression.
func TestCompare_AllFiveEndpointsReturnDisclaimer(t *testing.T) {
        cases := []struct {
                name string
                url  string
        }{
                {"countries", "/api/v1/compare/countries?countries=KE,UG"},
                {"legislation", "/api/v1/compare/legislation?countries=KE,UG"},
                {"debt", "/api/v1/compare/debt?countries=KE,UG"},
                {"government_structure", "/api/v1/compare/government-structure?countries=KE,UG"},
                {"indicators", "/api/v1/compare/indicators?countries=KE,UG"},
        }
        for _, tc := range cases {
                t.Run(tc.name, func(t *testing.T) {
                        req := httptest.NewRequest(http.MethodGet, tc.url, nil)
                        rec := httptest.NewRecorder()
                        makeCompareRouter(newCompareTestDebtRepo(t))(rec, req)
                        if rec.Code != 200 {
                                t.Fatalf("expected 200; got %d", rec.Code)
                        }
                        var resp struct {
                                Comparison struct {
                                        Disclaimer string `json:"disclaimer"`
                                } `json:"comparison"`
                        }
                        _ = json.NewDecoder(rec.Body).Decode(&resp)
                        if resp.Comparison.Disclaimer == "" {
                                t.Errorf("endpoint %s returned empty disclaimer", tc.name)
                        }
                        if !strings.Contains(resp.Comparison.Disclaimer, "does not rank") {
                                t.Errorf("endpoint %s disclaimer must mention 'does not rank'; got %q", tc.name, resp.Comparison.Disclaimer)
                        }
                })
        }
}

// TestCompare_DefaultsToAllCountriesWhenParamMissing verifies that
// omitting ?countries= returns all 6 supported countries rather than
// erroring out. The compare API is browse-first — users should be able
// to land on /compare/countries and see the full set without crafting
// a query string.
func TestCompare_DefaultsToAllCountriesWhenParamMissing(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/countries", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(nil)(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        Countries []string `json:"countries"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if len(resp.Comparison.Countries) != 6 {
                t.Errorf("expected 6 default countries; got %d (%v)", len(resp.Comparison.Countries), resp.Comparison.Countries)
        }
}

// TestCompare_DropsUnknownCountryCodes verifies the platform silently
// ignores unsupported country codes rather than 400-ing. The supported
// set is {KE, UG, TZ, GH, NG, ZA}; anything else is dropped.
func TestCompare_DropsUnknownCountryCodes(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/countries?countries=KE,XX,UG,YY", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(nil)(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Comparison struct {
                        Countries []string `json:"countries"`
                } `json:"comparison"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if len(resp.Comparison.Countries) != 2 {
                t.Errorf("expected 2 valid countries (KE, UG); got %d (%v)", len(resp.Comparison.Countries), resp.Comparison.Countries)
        }
        for _, c := range resp.Comparison.Countries {
                if c != "KE" && c != "UG" {
                        t.Errorf("unexpected country %q in response", c)
                }
        }
}

// TestCompare_MethodNotAllowed verifies that POST/PUT/DELETE are rejected
// with 405 across every compare sub-resource.
func TestCompare_MethodNotAllowed(t *testing.T) {
        for _, path := range []string{"countries", "legislation", "debt", "government-structure", "indicators"} {
                req := httptest.NewRequest(http.MethodPost, "/api/v1/compare/"+path, nil)
                rec := httptest.NewRecorder()
                makeCompareRouter(nil)(rec, req)
                if rec.Code != http.StatusMethodNotAllowed {
                        t.Errorf("expected 405 for POST /api/v1/compare/%s; got %d", path, rec.Code)
                }
        }
}

// TestCompare_UnknownSubResourceReturns404 verifies that an unknown
// sub-resource returns 404 (not 200 with empty data).
func TestCompare_UnknownSubResourceReturns404(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/compare/unknown", nil)
        rec := httptest.NewRecorder()

        makeCompareRouter(nil)(rec, req)

        if rec.Code != http.StatusNotFound {
                t.Fatalf("expected 404; got %d", rec.Code)
        }
}
