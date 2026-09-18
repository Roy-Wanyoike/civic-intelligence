// Package integration — api_bills_test.go exercises the /api/v1/bills
// endpoint family. The bills endpoint reads live data from the Kenya Law
// adapter (new.kenyalaw.org). When the network is available, the test
// verifies the success response shape + pagination params; when the
// network is unreachable (typical in CI sandboxes), the test verifies the
// 503 adapter_error shape so the test still asserts a documented contract
// rather than silently skipping.
//
// Endpoints covered:
//
//	GET /api/v1/bills                  -- list Bills (pagination, country header)
//	GET /api/v1/bills/{id}/timeline   -- Bill timeline (publication event)
//	GET /api/v1/bills/{id}/summary    -- AI summary (or fallback stub)
//	GET /api/v1/bills/{id}/changes    -- version diff (empty until #19)
//	GET /api/v1/bills/{id}/versions   -- version list (empty until #19)
//	GET /api/v1/bills/{id}/documents  -- document list (empty until #19)
package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// billsListResponse is the documented /api/v1/bills success shape.
type billsListResponse struct {
	Items    []map[string]any `json:"items"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Source   string          `json:"source"`
	Country  string          `json:"country"`
}

// errorResponse is the documented error shape returned by writeError().
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// TestBills_HealthzReachable confirms the server is up before any bills
// endpoint is exercised. This catches a regression where TestMain's
// waitForReady returned successfully but the server crashed between then
// and the first bills test.
func TestBills_HealthzReachable(t *testing.T) {
	var resp map[string]string
	status := mustGet(t, rootURL("/api/v1/healthz"), &resp)
	assertStatus(t, "/api/v1/healthz", http.StatusOK, status)
	if resp["status"] != "ok" {
		t.Errorf("healthz: expected status=ok, got %q", resp["status"])
	}
	if resp["service"] != "api" {
		t.Errorf("healthz: expected service=api, got %q", resp["service"])
	}
}

// TestBills_ListReturnsDocumentedShape verifies /api/v1/bills returns the
// documented JSON envelope. The Kenya Law adapter makes a live HTTP call
// to new.kenyalaw.org; in sandboxed CI this returns 503 adapter_error. We
// accept either branch and assert on the corresponding documented shape.
func TestBills_ListReturnsDocumentedShape(t *testing.T) {
	var list billsListResponse
	status := mustGet(t, apiURL("/bills"), &list)

	switch status {
	case http.StatusOK:
		// Success path: Kenya Law adapter returned real Bill candidates.
		if list.Total < 0 {
			t.Errorf("bills: total should be >= 0, got %d", list.Total)
		}
		if list.Total != len(list.Items) {
			t.Errorf("bills: total (%d) != len(items) (%d)", list.Total, len(list.Items))
		}
		if list.Page != 1 {
			t.Errorf("bills: expected page=1, got %d", list.Page)
		}
		if list.PageSize < 0 {
			t.Errorf("bills: page_size should be >= 0, got %d", list.PageSize)
		}
		if !strings.Contains(list.Source, "kenyalaw") {
			t.Errorf("bills: source %q should mention kenyalaw", list.Source)
		}
		// Country comes from X-Civic-Country header (default KE).
		if list.Country != "KE" {
			t.Errorf("bills: expected country=KE, got %q", list.Country)
		}
		// Every returned Bill must carry the documented fields.
		for i, b := range list.Items {
			for _, field := range []string{"id", "identifier", "title", "house", "status", "country", "source_url"} {
				v, ok := b[field]
				if !ok || v == nil || v == "" {
					t.Errorf("bills[%d]: missing required field %q (got %v)", i, field, v)
				}
			}
		}
	case http.StatusServiceUnavailable:
		// Sandbox path: Kenya Law adapter unreachable. The handler returns
		// a 503 with the documented adapter_error envelope so clients can
		// distinguish "no Bills found" from "upstream down".
		var errResp errorResponse
		// Re-decode: mustGet above tried to decode into billsListResponse
		// which would have left the error fields unset. Decode again into
		// the error shape.
		status2 := mustGet(t, apiURL("/bills"), &errResp)
		if status2 != http.StatusServiceUnavailable {
			t.Fatalf("bills: re-request returned %d (expected 503)", status2)
		}
		if errResp.Error != "adapter_error" {
			t.Errorf("bills 503: expected error=adapter_error, got %q", errResp.Error)
		}
		if errResp.Message == "" {
			t.Error("bills 503: expected non-empty message")
		}
	default:
		t.Fatalf("bills: unexpected status %d (expected 200 or 503)", status)
	}
}

// TestBills_ListAcceptsCountryHeader verifies the X-Civic-Country header
// is reflected in the response. The handler defaults to "KE" when no
// header is set; with an explicit header the country field echoes it.
func TestBills_ListAcceptsCountryHeader(t *testing.T) {
	req, err := http.NewRequest("GET", apiURL("/bills"), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-Civic-Country", "UG")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("bills: unexpected status %d (expected 200 or 503)", resp.StatusCode)
	}
	var list billsListResponse
	_ = json.NewDecoder(resp.Body).Decode(&list)
	// Country is only present on the 200 path (503 returns the error envelope).
	if resp.StatusCode == http.StatusOK && list.Country != "UG" {
		t.Errorf("bills: X-Civic-Country=UG should be echoed in response; got %q", list.Country)
	}
}

// TestBills_DetailRequiresID verifies that /api/v1/bills/ (with no ID
// segment) returns 400 bad_request. The handler explicitly rejects empty
// IDs rather than returning an empty list.
func TestBills_DetailRequiresID(t *testing.T) {
	// /api/v1/bills/ with trailing slash but no ID segment. The handler
	// splits the path and finds an empty billID, returning 400.
	var errResp errorResponse
	status := mustGet(t, apiURL("/bills/"), &errResp)
	if status != http.StatusBadRequest {
		t.Errorf("bills/: expected 400, got %d", status)
	}
}

// TestBills_DetailUnknownReturns404 verifies that a non-existent bill ID
// returns 404 not_found (not 500, not an empty 200).
func TestBills_DetailUnknownReturns404(t *testing.T) {
	// The handler tries to discover Bills from the Kenya Law adapter; if
	// the adapter is unreachable (503) we skip this assertion because the
	// 503 happens before the not_found branch. When the adapter works, an
	// unknown ID must yield 404.
	var dummy map[string]any
	firstStatus := mustGet(t, apiURL("/bills/does-not-exist-"+strings.Repeat("x", 12)), &dummy)
	if firstStatus == http.StatusServiceUnavailable {
		t.Skip("bills detail: Kenya Law adapter unreachable in sandbox; skipping 404 assertion")
	}
	if firstStatus != http.StatusNotFound {
		t.Errorf("bills/unknown: expected 404, got %d", firstStatus)
	}
}

// TestBills_SubRoutesReturnDocumentedShape verifies the /bills/{id}/timeline,
// /changes, /versions, /documents sub-routes return 200 with the documented
// envelope when the parent bill exists, OR 404/503 when it doesn't / when
// the adapter is unreachable. The test picks a known Bill ID from the
// list endpoint if available; otherwise it skips.
func TestBills_SubRoutesReturnDocumentedShape(t *testing.T) {
	// Discover a real Bill ID first.
	var list billsListResponse
	listStatus := mustGet(t, apiURL("/bills"), &list)
	if listStatus != http.StatusOK || len(list.Items) == 0 {
		t.Skip("bills: adapter unavailable or empty list; cannot test sub-routes")
	}
	firstID, _ := list.Items[0]["id"].(string)
	if firstID == "" {
		t.Skip("bills: first item has no id; cannot test sub-routes")
	}

	for _, sub := range []struct {
		path   string
		expect int
	}{
		{"timeline", http.StatusOK},
		{"changes", http.StatusOK},
		{"versions", http.StatusOK},
		{"documents", http.StatusOK},
		{"summary", http.StatusOK},
	} {
		var resp map[string]any
		status := mustGet(t, apiURL("/bills/"+firstID+"/"+sub.path), &resp)
		if status != sub.expect {
			t.Errorf("bills/%s: expected %d, got %d", sub.path, sub.expect, status)
			continue
		}
		// Every sub-route response carries a bill_id field tying the
		// payload back to the parent Bill.
		if _, ok := resp["bill_id"]; !ok {
			t.Errorf("bills/%s: response missing bill_id field", sub.path)
		}
	}
}

// TestBills_UnknownSubRouteReturns404 verifies that an undocumented
// /bills/{id}/{sub} returns 404 not_found (not 500).
func TestBills_UnknownSubRouteReturns404(t *testing.T) {
	var list billsListResponse
	listStatus := mustGet(t, apiURL("/bills"), &list)
	if listStatus != http.StatusOK || len(list.Items) == 0 {
		t.Skip("bills: adapter unavailable; cannot test sub-route 404")
	}
	firstID, _ := list.Items[0]["id"].(string)
	var errResp errorResponse
	status := mustGet(t, apiURL("/bills/"+firstID+"/unknown-sub-route"), &errResp)
	if status != http.StatusNotFound {
		t.Errorf("bills/unknown-sub-route: expected 404, got %d", status)
	}
}
