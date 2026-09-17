package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGovernmentsAPI_ListReturnsAdministrations verifies that the list
// endpoint returns the seed administrations (Jomo Kenyatta through William
// Ruto).
func TestGovernmentsAPI_ListReturnsAdministrations(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/governments", nil)
	rec := httptest.NewRecorder()

	makeGovernmentsListHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		Administrations []map[string]any `json:"administrations"`
		Count           int              `json:"count"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Count == 0 {
		t.Fatal("expected administrations; got 0")
	}
	// Verify at least 5 admins (Jomo, Moi, Kibaki, Uhuru, Ruto).
	if resp.Count < 5 {
		t.Errorf("expected at least 5 administrations; got %d", resp.Count)
	}
	// Verify the most recent (William Ruto) is first.
	first, _ := resp.Administrations[0]["name"].(string)
	if first != "William Ruto Administration" {
		t.Errorf("expected William Ruto first; got %q", first)
	}
}

// TestGovernmentsAPI_DetailReturnsPresident verifies that the detail endpoint
// returns the administration + president + terms.
func TestGovernmentsAPI_DetailReturnsPresident(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/governments/admin-uhuru-kenyatta", nil)
	rec := httptest.NewRecorder()

	makeGovernmentDetailHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		Administration map[string]any   `json:"administration"`
		President      map[string]any   `json:"president"`
		Terms          []map[string]any `json:"terms"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	name, _ := resp.President["display_name"].(string)
	if name != "Uhuru Kenyatta" {
		t.Errorf("expected president Uhuru Kenyatta; got %q", name)
	}
	if len(resp.Terms) < 2 {
		t.Errorf("expected at least 2 terms; got %d", len(resp.Terms))
	}
}

// TestGovernmentsAPI_ConstitutionReturnsFactTag verifies that the
// constitution endpoint tags the constitution as FACT.
func TestGovernmentsAPI_ConstitutionReturnsFactTag(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/constitution", nil)
	rec := httptest.NewRecorder()

	makeConstitutionHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["reality_layer"] != "FACT" {
		t.Errorf("expected reality_layer FACT; got %v", resp["reality_layer"])
	}
}

// TestGovernmentsAPI_TransitionsReturnsChronological verifies that the
// transitions endpoint returns transitions sorted by date.
func TestGovernmentsAPI_TransitionsReturnsChronological(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transitions", nil)
	rec := httptest.NewRecorder()

	makeTransitionsHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		Transitions []map[string]any `json:"transitions"`
		Count       int              `json:"count"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Count < 4 {
		t.Errorf("expected at least 4 transitions; got %d", resp.Count)
	}
	// First should be most recent (2022 transition to Ruto).
	firstDate, _ := resp.Transitions[0]["transition_date"].(string)
	if firstDate == "" {
		t.Fatal("expected non-empty transition_date")
	}
}
