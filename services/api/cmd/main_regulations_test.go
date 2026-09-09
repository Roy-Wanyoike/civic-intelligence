package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRegulationsList_Default verifies the /api/v1/regulations endpoint returns
// the expected sample regulations, every row carrying the documented fields.
func TestRegulationsList_Default(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/regulations", nil)
	rr := httptest.NewRecorder()
	handleRegulationsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Items  []regulationResponse `json:"items"`
		Total  int                  `json:"total"`
		Source string               `json:"source"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != len(sampleRegulations) {
		t.Errorf("expected total=%d, got %d", len(sampleRegulations), resp.Total)
	}
	if len(resp.Items) != len(sampleRegulations) {
		t.Fatalf("expected %d items, got %d", len(sampleRegulations), len(resp.Items))
	}

	for _, r := range resp.Items {
		if r.ID == "" || r.Title == "" || r.ParentAct == "" || r.SourceURL == "" || r.Status == "" {
			t.Errorf("regulation missing required field: %+v", r)
		}
		switch r.Status {
		case "in_force", "amended", "repealed":
		default:
			t.Errorf("regulation %q has invalid status %q", r.ID, r.Status)
		}
		if r.Country != "KE" {
			t.Errorf("regulation %q country=%q, want KE", r.ID, r.Country)
		}
		if !strings.HasPrefix(r.SourceURL, "https://") {
			t.Errorf("regulation %q source_url must be https: %s", r.ID, r.SourceURL)
		}
		if !strings.Contains(r.SourceURL, "kenyalaw.org") && !strings.Contains(r.SourceURL, "centralbank.go.ke") {
			t.Errorf("regulation %q source_url not from kenyalaw.org or centralbank.go.ke: %s", r.ID, r.SourceURL)
		}
		if r.GazetteNotice == "" {
			t.Errorf("regulation %q missing gazette_notice", r.ID)
		}
		if r.EffectiveDate == "" {
			t.Errorf("regulation %q missing effective_date", r.ID)
		}
	}
}

// TestRegulationsList_FilterByStatus verifies the ?status= query filter.
func TestRegulationsList_FilterByStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/regulations?status=in_force", nil)
	rr := httptest.NewRecorder()
	handleRegulationsList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Items []regulationResponse `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, r := range resp.Items {
		if r.Status != "in_force" {
			t.Errorf("expected all items in_force, got %q for %s", r.Status, r.ID)
		}
	}
	if len(resp.Items) == 0 {
		t.Error("expected at least one in_force regulation, got 0")
	}
}

// TestRegulationsList_FilterByParentAct verifies the ?parent_act= query filter.
func TestRegulationsList_FilterByParentAct(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/regulations?parent_act=data+protection", nil)
	rr := httptest.NewRecorder()
	handleRegulationsList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Items []regulationResponse `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 regulations under Data Protection Act, got %d", len(resp.Items))
	}
	for _, r := range resp.Items {
		if !strings.Contains(r.ParentAct, "Data Protection Act") {
			t.Errorf("regulation %q has wrong parent_act: %s", r.ID, r.ParentAct)
		}
	}
}

// TestRegulationsList_SearchQuery verifies the ?q= query filter searches title + parent_act + summary.
func TestRegulationsList_SearchQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/regulations?q=central+bank", nil)
	rr := httptest.NewRecorder()
	handleRegulationsList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Items []regulationResponse `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 match for 'central bank', got %d", len(resp.Items))
	}
	if resp.Items[0].ID != "ke-reg-cbk-prudential-2013" {
		t.Errorf("expected CBK Prudential Regulations, got %s", resp.Items[0].ID)
	}
}

// TestRegulationDetail_Found verifies a known regulation ID returns the full record.
func TestRegulationDetail_Found(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/regulations/ke-reg-data-protection-general-2021", nil)
	rr := httptest.NewRecorder()
	handleRegulationDetail(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var r regulationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if r.ID != "ke-reg-data-protection-general-2021" {
		t.Errorf("expected Data Protection General Regulations, got %s", r.ID)
	}
	if r.ParentAct == "" {
		t.Errorf("regulation %q missing parent_act", r.ID)
	}
}

// TestRegulationDetail_NotFound verifies an unknown ID returns 404.
func TestRegulationDetail_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/regulations/nonexistent-id", nil)
	rr := httptest.NewRecorder()
	handleRegulationDetail(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// TestRegulationsList_HasThreeToFive verifies the task requirement of 3-5 sample regulations.
func TestRegulationsList_HasThreeToFive(t *testing.T) {
	if len(sampleRegulations) < 3 {
		t.Errorf("expected at least 3 sample regulations, got %d", len(sampleRegulations))
	}
	if len(sampleRegulations) > 5 {
		t.Errorf("sample list grew beyond the documented 3-5 range (%d) — update the contract note", len(sampleRegulations))
	}
}

// TestRegulationsList_IncludesDataProtectionGeneral verifies the task-required regulation is present.
func TestRegulationsList_IncludesDataProtectionGeneral(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/regulations", nil)
	rr := httptest.NewRecorder()
	handleRegulationsList(rr, req)
	var resp struct {
		Items []regulationResponse `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, r := range resp.Items {
		if strings.Contains(strings.ToLower(r.Title), "data protection") && strings.Contains(strings.ToLower(r.Title), "general") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Data Protection (General) Regulations 2021 in the sample regulations, not found")
	}
}

// TestRegulationsList_IncludesCBK verifies the Central Bank Prudential Regulations are present.
func TestRegulationsList_IncludesCBK(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/regulations", nil)
	rr := httptest.NewRecorder()
	handleRegulationsList(rr, req)
	var resp struct {
		Items []regulationResponse `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, r := range resp.Items {
		if strings.Contains(strings.ToLower(r.Title), "central bank") && strings.Contains(strings.ToLower(r.Title), "prudential") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Central Bank Prudential Regulations in the sample regulations, not found")
	}
}
