package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestScenariosAPI_ListReturnsGoldenDataset verifies that GET /api/v1/scenarios
// returns the golden dataset and tags every scenario HYPOTHETICAL.
func TestScenariosAPI_ListReturnsGoldenDataset(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios", nil)
	req.Header.Set("X-Tenant-ID", "tenant-golden")
	rec := httptest.NewRecorder()

	makeScenariosHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		Scenarios []map[string]any `json:"scenarios"`
		Count     int              `json:"count"`
		Disclaimer string           `json:"disclaimer"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Count == 0 {
		t.Fatal("expected scenarios; got 0")
	}
	if !strings.Contains(resp.Disclaimer, "HYPOTHETICAL") {
		t.Errorf("expected disclaimer to mention HYPOTHETICAL; got %q", resp.Disclaimer)
	}
	// Every scenario must be tagged HYPOTHETICAL.
	for _, s := range resp.Scenarios {
		layer, _ := s["reality_layer"].(string)
		if layer != "HYPOTHETICAL" {
			t.Errorf("expected reality_layer HYPOTHETICAL; got %q", layer)
		}
	}
}

// TestScenariosAPI_GetDetailReturnsDisclaimer verifies that GET /scenarios/{id}
// returns a disclaimer labelling the scenario HYPOTHETICAL.
func TestScenariosAPI_GetDetailReturnsDisclaimer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/scn-simple-deterministic", nil)
	req.Header.Set("X-Tenant-ID", "tenant-golden")
	rec := httptest.NewRecorder()

	makeScenarioDetailHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	disclaimer, _ := resp["disclaimer"].(string)
	if !strings.Contains(disclaimer, "HYPOTHETICAL") {
		t.Errorf("expected disclaimer to mention HYPOTHETICAL; got %q", disclaimer)
	}
}

// TestScenariosAPI_RunProducesModeledResults verifies that POST
// /scenarios/{id}?action=run returns a result tagged MODELED.
func TestScenariosAPI_RunProducesModeledResults(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/scn-simple-deterministic?action=run", nil)
	req.Header.Set("X-Tenant-ID", "tenant-golden")
	rec := httptest.NewRecorder()

	makeScenarioDetailHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	layer, _ := resp["reality_layer"].(string)
	if layer != "MODELED" {
		t.Errorf("expected reality_layer MODELED; got %q", layer)
	}
	disclaimer, _ := resp["disclaimer"].(string)
	if !strings.Contains(disclaimer, "SIMULATED") {
		t.Errorf("expected disclaimer to mention SIMULATED; got %q", disclaimer)
	}
}

// TestScenariosAPI_TenantIsolation verifies Gate G — a tenant cannot access
// another tenant's scenarios.
func TestScenariosAPI_TenantIsolation(t *testing.T) {
	// tenant-golden owns scn-simple-deterministic.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/scn-simple-deterministic", nil)
	req.Header.Set("X-Tenant-ID", "tenant-other")
	rec := httptest.NewRecorder()

	makeScenarioDetailHandler()(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404 for cross-tenant access; got %d", rec.Code)
	}
}

// TestScenariosAPI_CompareReturnsDisclaimer verifies that the comparison
// endpoint produces a disclaimer and does NOT produce political rankings.
func TestScenariosAPI_CompareReturnsDisclaimer(t *testing.T) {
	body := `{"scenario_ids":["scn-comparison-a","scn-comparison-b"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/compare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-golden")
	rec := httptest.NewRecorder()

	makeScenarioCompareHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	disclaimer, _ := resp["disclaimer"].(string)
	if !strings.Contains(disclaimer, "does not rank") && !strings.Contains(disclaimer, "does not") {
		t.Errorf("expected disclaimer about not ranking; got %q", disclaimer)
	}
}

// TestScenariosAPI_MethodologyReturnsLimitations verifies Gate E — Model
// Transparency. Every published result identifies the model/methodology
// and limitations.
func TestScenariosAPI_MethodologyReturnsLimitations(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/scn-simple-deterministic/methodology", nil)
	req.Header.Set("X-Tenant-ID", "tenant-golden")
	rec := httptest.NewRecorder()

	makeScenarioDetailHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	lims, ok := resp["limitations"].([]any)
	if !ok || len(lims) == 0 {
		t.Errorf("expected non-empty limitations; got %v", resp["limitations"])
	}
}

// TestScenariosAPI_TimelineReturnsLabels verifies Gate K — UI Clarity.
// Timeline events carry OBSERVED / ASSUMED / MODELED labels.
func TestScenariosAPI_TimelineReturnsLabels(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/scn-simple-deterministic/timeline", nil)
	req.Header.Set("X-Tenant-ID", "tenant-golden")
	rec := httptest.NewRecorder()

	makeScenarioDetailHandler()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		Timeline []map[string]any `json:"timeline"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	labels := map[string]bool{}
	for _, ev := range resp.Timeline {
		if l, ok := ev["label"].(string); ok {
			labels[l] = true
		}
	}
	if !labels["OBSERVED"] {
		t.Error("expected at least one OBSERVED timeline event")
	}
	if !labels["ASSUMED"] {
		t.Error("expected at least one ASSUMED timeline event")
	}
	if !labels["MODELED"] {
		t.Error("expected at least one MODELED timeline event")
	}
}
