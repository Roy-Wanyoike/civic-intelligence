package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestAPIRouter builds a minimal ServeMux that registers the people,
// committees, and institutions routes the same way main() does — both
// with and without the trailing slash. This is required to assert that
// Go's http.ServeMux itself does not auto-301 the no-slash form to the
// with-slash form (issue #266).
//
// The handler-level dispatch behaviour (issue #264) is verified by the
// other tests that call handlePeople / handleCommittees / handleInstitutions
// directly with a URL.Path set by httptest.NewRequest.
func newTestAPIRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/people", handlePeople)
	mux.HandleFunc("/api/v1/people/", handlePeople)
	mux.HandleFunc("/api/v1/committees", handleCommittees)
	mux.HandleFunc("/api/v1/committees/", handleCommittees)
	mux.HandleFunc("/api/v1/institutions", handleInstitutions)
	mux.HandleFunc("/api/v1/institutions/", handleInstitutions)
	return mux
}

// doRequest is a small helper that fires a GET against the supplied
// handler and returns the recorded response. It exists purely to keep
// the trailing-slash regression tests below compact.
func doRequest(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rr := httptest.NewRecorder()
	// Disable the default redirect-following behaviour so we observe the
	// raw 301 (if any) the ServeMux emits.
	h.ServeHTTP(rr, req)
	return rr
}

// === Issue #264: GET /api/v1/people (no trailing slash) ===

// TestPeopleHandler_NoTrailingSlash_ReturnsList verifies that
// handlePeople returns the people list (not a 404 "person not found")
// when called with /api/v1/people (no trailing slash). This is the
// regression test for the original bug.
func TestPeopleHandler_NoTrailingSlash_ReturnsList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/people", nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/people (no slash), got %d body=%q", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total == 0 || len(resp.Items) == 0 {
		t.Fatalf("expected non-empty people list, got total=%d", resp.Total)
	}
}

// TestPeopleHandler_TrailingSlash_ReturnsList verifies the with-slash
// form still works (the previously-working URL form). The payload must
// match the no-slash form exactly.
func TestPeopleHandler_TrailingSlash_ReturnsList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/people/", nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/people/, got %d body=%q", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total == 0 || len(resp.Items) == 0 {
		t.Fatalf("expected non-empty people list, got total=%d", resp.Total)
	}
}

// TestPeopleHandler_BothURLFormsReturnSamePayload asserts the with-slash
// and no-slash forms of /api/v1/people return byte-for-byte identical
// bodies. This protects REST clients that round-trip a URL returned by
// the API without normalising it.
func TestPeopleHandler_BothURLFormsReturnSamePayload(t *testing.T) {
	noSlash := doRequest(t, http.HandlerFunc(handlePeople), "/api/v1/people")
	withSlash := doRequest(t, http.HandlerFunc(handlePeople), "/api/v1/people/")

	if noSlash.Code != http.StatusOK {
		t.Fatalf("no-slash: expected 200, got %d", noSlash.Code)
	}
	if withSlash.Code != http.StatusOK {
		t.Fatalf("with-slash: expected 200, got %d", withSlash.Code)
	}
	if noSlash.Body.String() != withSlash.Body.String() {
		t.Errorf("expected identical bodies, got no-slash=%q with-slash=%q",
			noSlash.Body.String(), withSlash.Body.String())
	}
}

// TestPeopleHandler_PersonDetailStillWorks verifies the person detail
// path (GET /api/v1/people/{id}) was not broken by the path
// normalisation change.
func TestPeopleHandler_PersonDetailStillWorks(t *testing.T) {
	// person-001 is in sampleScorecards with KE visibility.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/people/person-001", nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for person detail, got %d body=%q", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] != "person-001" {
		t.Errorf("expected id=person-001, got %v", resp["id"])
	}
	if resp["scorecard_url"] != "/api/v1/people/person-001/scorecard" {
		t.Errorf("expected scorecard_url, got %v", resp["scorecard_url"])
	}
}

// TestPeopleHandler_ScorecardSubResourceStillWorks verifies the
// /scorecard sub-resource (task ENG-K2) still dispatches correctly
// after the path normalisation change.
func TestPeopleHandler_ScorecardSubResourceStillWorks(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/people/person-001/scorecard", nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for scorecard, got %d body=%q", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["person_id"] != "person-001" {
		t.Errorf("expected person_id=person-001, got %v", resp["person_id"])
	}
	if resp["reality_layer"] != "FACT" {
		t.Errorf("expected reality_layer=FACT, got %v", resp["reality_layer"])
	}
}

// === Issue #266: /institutions and /committees no-slash URLs ===

// TestInstitutionsHandler_NoTrailingSlash_Returns200 asserts that
// /api/v1/institutions (no trailing slash) returns 200, NOT a 301
// redirect. This test drives the actual ServeMux so it catches the
// auto-redirect that the http.ServeMux emits when only the with-slash
// form is registered.
func TestInstitutionsHandler_NoTrailingSlash_Returns200(t *testing.T) {
	mux := newTestAPIRouter()
	rr := doRequest(t, mux, "/api/v1/institutions")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/institutions (no slash), got %d (Location=%q) body=%q",
			rr.Code, rr.Header().Get("Location"), rr.Body.String())
	}
	// Explicitly assert there is no Location header (which a 301 would set).
	if loc := rr.Header().Get("Location"); loc != "" {
		t.Errorf("expected no Location header, got %q (ServeMux 301-redirected)", loc)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total == 0 || len(resp.Items) == 0 {
		t.Fatalf("expected non-empty institutions list, got total=%d", resp.Total)
	}
}

// TestInstitutionsHandler_TrailingSlash_ReturnsSamePayload asserts the
// with-slash form returns the same payload as the no-slash form.
func TestInstitutionsHandler_TrailingSlash_ReturnsSamePayload(t *testing.T) {
	mux := newTestAPIRouter()
	noSlash := doRequest(t, mux, "/api/v1/institutions")
	withSlash := doRequest(t, mux, "/api/v1/institutions/")

	if noSlash.Code != http.StatusOK {
		t.Fatalf("no-slash: expected 200, got %d", noSlash.Code)
	}
	if withSlash.Code != http.StatusOK {
		t.Fatalf("with-slash: expected 200, got %d", withSlash.Code)
	}
	if noSlash.Body.String() != withSlash.Body.String() {
		t.Errorf("expected identical bodies, got no-slash=%q with-slash=%q",
			noSlash.Body.String(), withSlash.Body.String())
	}
}

// TestCommitteesHandler_NoTrailingSlash_Returns200 asserts that
// /api/v1/committees (no trailing slash) returns 200, NOT a 301
// redirect. Drives the actual ServeMux to catch the auto-redirect.
func TestCommitteesHandler_NoTrailingSlash_Returns200(t *testing.T) {
	mux := newTestAPIRouter()
	rr := doRequest(t, mux, "/api/v1/committees")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/committees (no slash), got %d (Location=%q) body=%q",
			rr.Code, rr.Header().Get("Location"), rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "" {
		t.Errorf("expected no Location header, got %q (ServeMux 301-redirected)", loc)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total == 0 || len(resp.Items) == 0 {
		t.Fatalf("expected non-empty committees list, got total=%d", resp.Total)
	}
}

// TestCommitteesHandler_TrailingSlash_ReturnsSamePayload asserts the
// with-slash form returns the same payload as the no-slash form.
func TestCommitteesHandler_TrailingSlash_ReturnsSamePayload(t *testing.T) {
	mux := newTestAPIRouter()
	noSlash := doRequest(t, mux, "/api/v1/committees")
	withSlash := doRequest(t, mux, "/api/v1/committees/")

	if noSlash.Code != http.StatusOK {
		t.Fatalf("no-slash: expected 200, got %d", noSlash.Code)
	}
	if withSlash.Code != http.StatusOK {
		t.Fatalf("with-slash: expected 200, got %d", withSlash.Code)
	}
	if noSlash.Body.String() != withSlash.Body.String() {
		t.Errorf("expected identical bodies, got no-slash=%q with-slash=%q",
			noSlash.Body.String(), withSlash.Body.String())
	}
}

// TestCommitteesHandler_DetailStillWorks verifies the committee detail
// path (GET /api/v1/committees/{id}) was not broken by the path
// normalisation change.
func TestCommitteesHandler_DetailStillWorks(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/committees/committee-finance", nil)
	rr := httptest.NewRecorder()
	handleCommittees(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for committee detail, got %d body=%q", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] != "committee-finance" {
		t.Errorf("expected id=committee-finance, got %v", resp["id"])
	}
}

// TestInstitutionsHandler_DetailStillWorks verifies the institution
// detail path (GET /api/v1/institutions/{id}) was not broken by the
// path normalisation change.
func TestInstitutionsHandler_DetailStillWorks(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/institutions/institution-parliament-ke", nil)
	rr := httptest.NewRecorder()
	handleInstitutions(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for institution detail, got %d body=%q", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] != "institution-parliament-ke" {
		t.Errorf("expected id=institution-parliament-ke, got %v", resp["id"])
	}
}
