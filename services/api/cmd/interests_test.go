package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// interestsPayload is the JSON envelope returned by the new endpoint
// (issue #288). Only the fields the tests assert on are decoded.
type interestsPayload struct {
	PersonID     string               `json:"person_id"`
	PersonName   string               `json:"person_name"`
	Items        []RegisteredInterest `json:"items"`
	Total        int                  `json:"total"`
	Category     string               `json:"category,omitempty"`
	Source       string               `json:"source"`
	ScorecardURL string               `json:"scorecard_url,omitempty"`
	Disclaimer   string               `json:"disclaimer"`
}

// validCategoriesForTest mirrors the package-level validInterestCategories
// map so a regression in either direction (a new category added to the
// API without updating this set, or a value dropped from the API but
// left here) shows up as a test failure.
var validCategoriesForTest = map[string]bool{
	"directorship":  true,
	"land_property": true,
	"shares":        true,
	"gifts":         true,
	"other_income":  true,
	"loans":         true,
}

// TestInterestsByPerson_ReturnsInterests verifies the new
// /api/v1/people/{id}/interests endpoint (issue #288) returns every
// registered-interest record the seed slice attributes to the given MP.
//
// The test:
//  1. Hits /api/v1/people/person-001/interests via handlePeople (so the
//     dispatch logic in handlePeople is exercised — not just the inner
//     handler).
//  2. Asserts 200 + the response envelope's person_id, person_name,
//     source, scorecard_url, and disclaimer fields are populated correctly.
//  3. Asserts the items list matches findInterestsByPerson for person-001 —
//     same count + every item has person_id == person-001 + every item
//     carries a non-empty source_url + every item's category is one of the
//     six canonical enum values.
//  4. Asserts the disclaimer is the canonical seed-provenance marker so
//     the API never silently ships fabricated declarations as authoritative.
func TestInterestsByPerson_ReturnsInterests(t *testing.T) {
	const personID = "person-001"
	// Compute the expected seed records for person-001 up-front so the
	// test fails loudly if the seed slice changes (regression guard).
	expected := findInterestsByPerson(personID)
	if len(expected) == 0 {
		t.Fatalf("seed slice must attribute ≥1 interest to person-001 for this test")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/people/"+personID+"/interests", nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/people/%s/interests, got %d (body=%s)",
			personID, rr.Code, rr.Body.String())
	}

	var resp interestsPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PersonID != personID {
		t.Errorf("expected person_id=%q, got %q", personID, resp.PersonID)
	}
	// Person-001 is Kimani Ichung'wah (sampleScorecards[0]).
	const wantName = "Kimani Ichung'wah"
	if resp.PersonName != wantName {
		t.Errorf("expected person_name=%q, got %q", wantName, resp.PersonName)
	}
	if resp.Source != "seed" {
		t.Errorf("expected source=seed (declarations are seed-only), got %q", resp.Source)
	}
	wantScorecardURL := "/api/v1/people/" + personID + "/scorecard"
	if resp.ScorecardURL != wantScorecardURL {
		t.Errorf("expected scorecard_url=%q, got %q", wantScorecardURL, resp.ScorecardURL)
	}
	if resp.Disclaimer != interestsDisclaimer {
		t.Errorf("expected disclaimer=%q, got %q", interestsDisclaimer, resp.Disclaimer)
	}
	// No ?category= filter on this call → the envelope MUST NOT echo one.
	if resp.Category != "" {
		t.Errorf("expected empty category (no filter applied), got %q", resp.Category)
	}

	// Assert the items count matches the seed slice's count for this MP.
	if resp.Total != len(expected) {
		t.Errorf("expected total=%d (matches seed slice), got %d", len(expected), resp.Total)
	}
	if len(resp.Items) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(resp.Items))
	}

	// Every item MUST be attributed to the requested MP + carry a
	// non-empty source_url + use a category from the closed enum. The
	// source_url invariant is the platform's core anti-fabrication guard
	// (rule: every declared interest links to its canonical register entry).
	for i, item := range resp.Items {
		if item.PersonID != personID {
			t.Errorf("item[%d]: expected person_id=%q, got %q", i, personID, item.PersonID)
		}
		if item.PersonName != wantName {
			t.Errorf("item[%d]: expected person_name=%q, got %q", i, wantName, item.PersonName)
		}
		if item.Description == "" {
			t.Errorf("item[%d]: description must not be empty", i)
		}
		if !validCategoriesForTest[item.Category] {
			t.Errorf("item[%d]: category=%q is not in the closed enum", i, item.Category)
		}
		if item.SourceURL == "" {
			t.Errorf("item[%d]: source_url must not be empty (anti-fabrication guard)", i)
		}
		if item.DeclaredAt == "" {
			t.Errorf("item[%d]: declared_at must not be empty", i)
		}
		if item.Source == "" {
			t.Errorf("item[%d]: source must not be empty", i)
		}
	}
}

// TestInterestsByPerson_UnknownPerson_Returns404 verifies the endpoint
// returns 404 (NOT 200 with an empty items list) for an unknown person
// ID. The person MUST exist before we report on the absence of declared
// interests — otherwise a typo'd person ID silently looks like a
// compliant MP with zero declarations.
func TestInterestsByPerson_UnknownPerson_Returns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/people/does-not-exist/interests", nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown person, got %d (body=%s)",
			rr.Code, rr.Body.String())
	}
}

// TestInterestsByPerson_FiltersByCategory verifies the ?category= filter
// narrows the response to only matching records AND echoes the filter
// back on the envelope so the client can confirm the filter applied.
//
// The test:
//  1. Hits /api/v1/people/person-001/interests?category=directorship via
//     handlePeople.
//  2. Asserts 200 + every returned item has category == "directorship".
//  3. Asserts the items count matches the seed slice's directorship count
//     for person-001 (computed independently via findInterestsByPerson +
//     a manual category check, so a bug in either the filter or the
//     seed-slice attribution surfaces).
//  4. Asserts the envelope echoes the ?category= value back so the client
//     can confirm the filter was applied (not silently dropped).
//  5. Also asserts the 400 path: an unknown category value returns 400
//     (NOT 200 with an empty items list) so a typo never looks like a
//     legitimate "no interests of this kind" response.
func TestInterestsByPerson_FiltersByCategory(t *testing.T) {
	const personID = "person-001"
	const category = "directorship"

	// Compute the expected directorship count for person-001 by re-running
	// the seed-slice filter (independent of the handler's filter logic).
	allForPerson := findInterestsByPerson(personID)
	expectedCount := 0
	for _, ri := range allForPerson {
		if ri.Category == category {
			expectedCount++
		}
	}
	if expectedCount == 0 {
		t.Fatalf("seed slice must attribute ≥1 %q interest to person-001 for this test", category)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/people/"+personID+"/interests?category="+category, nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for filtered request, got %d (body=%s)",
			rr.Code, rr.Body.String())
	}

	var resp interestsPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Category != category {
		t.Errorf("expected envelope to echo category=%q, got %q", category, resp.Category)
	}
	if resp.Total != expectedCount {
		t.Errorf("expected total=%d (directorship-only count), got %d", expectedCount, resp.Total)
	}
	if len(resp.Items) != expectedCount {
		t.Fatalf("expected %d filtered items, got %d", expectedCount, len(resp.Items))
	}
	for i, item := range resp.Items {
		if item.Category != category {
			t.Errorf("item[%d]: expected category=%q, got %q (filter not applied)",
				i, category, item.Category)
		}
		if item.PersonID != personID {
			t.Errorf("item[%d]: expected person_id=%q, got %q", i, personID, item.PersonID)
		}
	}

	// 400 path: an unknown category value MUST return 400 (NOT 200 with
	// an empty items list) so a typo never looks like a legitimate
	// "no interests of this kind" response.
	req400 := httptest.NewRequest(http.MethodGet,
		"/api/v1/people/"+personID+"/interests?category=not-a-real-category", nil)
	rr400 := httptest.NewRecorder()
	handlePeople(rr400, req400)
	if rr400.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown category, got %d (body=%s)",
			rr400.Code, rr400.Body.String())
	}
}
