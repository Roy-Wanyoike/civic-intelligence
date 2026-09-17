package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestActAudit_ReturnsLifecycleStatuses verifies that the audit endpoint
// returns the full set of lifecycle statuses.
func TestActAudit_ReturnsLifecycleStatuses(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019/audit", nil)
	rec := httptest.NewRecorder()

	makeActRouter()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		ActID          string   `json:"act_id"`
		AuditStatuses  []string `json:"audit_statuses"`
		Disclaimer     string   `json:"disclaimer"`
		AssentRecorded bool     `json:"assent_recorded"`
		ActPublished   bool     `json:"act_published"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if !resp.AssentRecorded {
		t.Error("expected assent recorded")
	}
	if !resp.ActPublished {
		t.Error("expected act published")
	}
	if resp.Disclaimer == "" {
		t.Error("expected disclaimer")
	}
	if !contains(resp.AuditStatuses, "ASSENT_CONFIRMED") {
		t.Error("expected ASSENT_CONFIRMED in audit statuses")
	}
	if !contains(resp.AuditStatuses, "REGULATIONS_TRACKED") {
		t.Error("expected REGULATIONS_TRACKED (sample act has regulations)")
	}
}

// TestActEvents_ReturnsPostAssentEvents verifies that the events endpoint
// returns the post-assent events for an Act.
func TestActEvents_ReturnsPostAssentEvents(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019/events", nil)
	rec := httptest.NewRecorder()

	makeActRouter()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		ActID string                   `json:"act_id"`
		Events []map[string]any       `json:"events"`
		Count int                     `json:"count"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if resp.ActID != "ke-act-data-protection-2019" {
		t.Errorf("expected ke-act-data-protection-2019; got %s", resp.ActID)
	}
	if resp.Count == 0 {
		t.Error("expected at least one event")
	}
}

// TestFollowLaw_ReturnsMonitorList verifies the Follow-a-Law experience.
func TestFollowLaw_ReturnsMonitorList(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/acts/ke-act-data-protection-2019/follow", nil)
	rec := httptest.NewRecorder()

	makeActRouter()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		Followed  bool     `json:"followed"`
		Monitors  []string `json:"monitors"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if !resp.Followed {
		t.Error("expected followed=true")
	}
	// Should monitor at least: COMMENCEMENT, REGULATIONS, AMENDMENTS, COURT_CASES.
	expected := map[string]bool{
		"COMMENCEMENT":  false,
		"REGULATIONS":   false,
		"AMENDMENTS":    false,
		"COURT_CASES":   false,
	}
	for _, m := range resp.Monitors {
		if _, ok := expected[m]; ok {
			expected[m] = true
		}
	}
	for k, v := range expected {
		if !v {
			t.Errorf("expected monitor %s to be present", k)
		}
	}
}

// TestLineage_ReturnsFullChain verifies that the lineage endpoint returns
// the full Bill → Act → Amendments → ... → Current Status chain.
func TestLineage_ReturnsFullChain(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019/lineage", nil)
	rec := httptest.NewRecorder()

	makeActRouter()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		ActID    string                   `json:"act_id"`
		Lineage  []map[string]any         `json:"lineage"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Lineage) < 8 {
		t.Errorf("expected at least 8 lineage steps; got %d", len(resp.Lineage))
	}
	// First step should be origin_bill; last should be current_status.
	first, _ := resp.Lineage[0]["step"].(string)
	last, _ := resp.Lineage[len(resp.Lineage)-1]["step"].(string)
	if first != "origin_bill" {
		t.Errorf("expected first step origin_bill; got %q", first)
	}
	if last != "current_status" {
		t.Errorf("expected last step current_status; got %q", last)
	}
}

// TestActRouter_ActDetailStillWorks verifies that the existing /acts/{id}
// detail endpoint still works after the router was added.
func TestActRouter_ActDetailStillWorks(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019", nil)
	rec := httptest.NewRecorder()

	makeActRouter()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
