package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestActsList_Default verifies the /api/v1/acts endpoint returns the expected
// sample Acts of Parliament, every row carrying the documented fields.
func TestActsList_Default(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/acts", nil)
	rr := httptest.NewRecorder()
	handleActsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Items  []actResponse `json:"items"`
		Total  int           `json:"total"`
		Source string        `json:"source"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != len(sampleActs) {
		t.Errorf("expected total=%d, got %d", len(sampleActs), resp.Total)
	}
	if resp.Source != "kenyalaw.org" {
		t.Errorf("expected source=kenyalaw.org, got %q", resp.Source)
	}
	if len(resp.Items) != len(sampleActs) {
		t.Fatalf("expected %d items, got %d", len(sampleActs), len(resp.Items))
	}

	// Every item must carry the documented fields.
	for _, a := range resp.Items {
		if a.ID == "" || a.Title == "" || a.Citation == "" || a.SourceURL == "" || a.Status == "" {
			t.Errorf("act missing required field: %+v", a)
		}
		switch a.Status {
		case "in_force", "amended", "repealed":
		default:
			t.Errorf("act %q has invalid status %q", a.ID, a.Status)
		}
		if a.Country != "KE" {
			t.Errorf("act %q country=%q, want KE", a.ID, a.Country)
		}
		if !strings.HasPrefix(a.SourceURL, "https://www.kenyalaw.org/") {
			t.Errorf("act %q source_url is not a kenyalaw.org URL: %s", a.ID, a.SourceURL)
		}
		if a.AssentDate == "" {
			t.Errorf("act %q missing assent_date", a.ID)
		}
		if a.CommencementDate == "" {
			t.Errorf("act %q missing commencement_date", a.ID)
		}
	}
}

// TestActsList_FilterByStatus verifies the ?status= query filter.
func TestActsList_FilterByStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/acts?status=in_force", nil)
	rr := httptest.NewRecorder()
	handleActsList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Items []actResponse `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, a := range resp.Items {
		if a.Status != "in_force" {
			t.Errorf("expected all items in_force, got %q for %s", a.Status, a.ID)
		}
	}
	if len(resp.Items) == 0 {
		t.Error("expected at least one in_force act, got 0")
	}
}

// TestActsList_SearchQuery verifies the ?q= query filter searches title + citation + summary.
func TestActsList_SearchQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/acts?q=data", nil)
	rr := httptest.NewRecorder()
	handleActsList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Items []actResponse `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 match for 'data', got %d", len(resp.Items))
	}
	if resp.Items[0].ID != "ke-act-data-protection-2019" {
		t.Errorf("expected Data Protection Act, got %s", resp.Items[0].ID)
	}
}

// TestActsList_UnknownStatusReturnsEmpty verifies an unknown status filter
// yields an empty list (not an error).
func TestActsList_UnknownStatusReturnsEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/acts?status=nonexistent", nil)
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
		t.Errorf("expected 0 items for unknown status, got %d", resp.Total)
	}
}

// TestActDetail_Found verifies a known act ID returns the full record.
func TestActDetail_Found(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/acts/ke-act-constitution-2010", nil)
	rr := httptest.NewRecorder()
	handleActDetail(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var a actResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &a); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if a.ID != "ke-act-constitution-2010" {
		t.Errorf("expected constitution, got %s", a.ID)
	}
	if a.Status != "in_force" {
		t.Errorf("expected in_force status, got %s", a.Status)
	}
}

// TestActDetail_NotFound verifies an unknown ID returns 404.
func TestActDetail_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/acts/nonexistent-id", nil)
	rr := httptest.NewRecorder()
	handleActDetail(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// TestActsList_IncludesConstitution verifies the Constitution of Kenya 2010 is present.
func TestActsList_IncludesConstitution(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/acts", nil)
	rr := httptest.NewRecorder()
	handleActsList(rr, req)
	var resp struct {
		Items []actResponse `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, a := range resp.Items {
		if strings.Contains(strings.ToLower(a.Title), "constitution") && strings.Contains(a.AssentDate, "2010") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Constitution of Kenya 2010 in the sample acts, not found")
	}
}

// TestActsList_HasThreeToFive verifies the task requirement of 3-5 real Kenyan Acts.
func TestActsList_HasThreeToFive(t *testing.T) {
	if len(sampleActs) < 3 {
		t.Errorf("expected at least 3 sample acts, got %d", len(sampleActs))
	}
	if len(sampleActs) > 5 {
		t.Errorf("sample list grew beyond the documented 3-5 range (%d) — update the contract note", len(sampleActs))
	}
}
