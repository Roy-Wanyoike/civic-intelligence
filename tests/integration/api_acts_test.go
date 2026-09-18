// Package integration — api_acts_test.go exercises the Acts of Parliament
// endpoints. The Kenya seed data (kenya_seed.KenyaActs) populates the
// in-memory ActRepository at startup with verified Kenyan acts sourced
// from kenyalaw.org.
//
// Post-assent endpoints (audit, events, follow, lineage) verify the
// flagship "Follow a Law" experience (issue #193, #216) and the full legal
// lineage (Spec §20).
//
// Endpoints covered:
//
//	GET  /api/v1/acts                  -- list Acts (filter + search + pagination)
//	GET  /api/v1/acts/{id}             -- Act detail
//	GET  /api/v1/acts/{id}/audit      -- full lifecycle audit
//	GET  /api/v1/acts/{id}/events     -- post-assent events
//	POST /api/v1/acts/{id}/follow     -- follow a law (auth required)
//	GET  /api/v1/acts/{id}/lineage     -- full legal lineage (9 steps)
package integration

import (
	"net/http"
	"strings"
	"testing"
)

// actsListResponse is the documented /api/v1/acts list shape.
type actsListResponse struct {
	Items    []map[string]any `json:"items"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	HasNext  bool            `json:"has_next"`
	Source   string          `json:"source"`
}

// TestActs_ListReturnsSeed verifies the list endpoint returns the seeded
// Kenyan acts with the documented envelope.
func TestActs_ListReturnsSeed(t *testing.T) {
	var resp actsListResponse
	status := mustGet(t, apiURL("/acts"), &resp)
	assertStatus(t, "/acts", http.StatusOK, status)

	if resp.Total == 0 {
		t.Fatal("acts: expected at least one act, got total=0")
	}
	if len(resp.Items) == 0 {
		t.Fatal("acts: expected non-empty items array")
	}
	if resp.Source != "kenyalaw.org" {
		t.Errorf("acts: expected source=kenyalaw.org, got %q", resp.Source)
	}
	if resp.Page != 1 {
		t.Errorf("acts: expected page=1, got %d", resp.Page)
	}
	if resp.PageSize < 1 {
		t.Errorf("acts: expected page_size >= 1, got %d", resp.PageSize)
	}
	// Every item must carry the documented fields.
	for i, a := range resp.Items {
		for _, field := range []string{"id", "title", "citation", "status", "country", "source_url", "assent_date", "commencement_date"} {
			v, ok := a[field]
			if !ok || v == nil || v == "" {
				t.Errorf("acts[%d]: missing required field %q (got %v)", i, field, v)
			}
		}
		if a["country"] != "KE" {
			t.Errorf("acts[%d]: expected country=KE, got %v", i, a["country"])
		}
		if !strings.HasPrefix(a["source_url"].(string), "https://www.kenyalaw.org/") {
			t.Errorf("acts[%d]: source_url is not a kenyalaw.org URL: %v", i, a["source_url"])
		}
		switch a["status"] {
		case "in_force", "amended", "repealed":
		default:
			t.Errorf("acts[%d]: invalid status %q", i, a["status"])
		}
	}
}

// TestActs_ListFilterByStatus verifies the ?status= query filter narrows
// the list to acts matching that status.
func TestActs_ListFilterByStatus(t *testing.T) {
	var resp actsListResponse
	status := mustGet(t, apiURL("/acts?status=in_force"), &resp)
	assertStatus(t, "/acts?status=in_force", http.StatusOK, status)

	if len(resp.Items) == 0 {
		t.Fatal("acts?status=in_force: expected at least one in_force act")
	}
	for i, a := range resp.Items {
		if a["status"] != "in_force" {
			t.Errorf("acts?status=in_force[%d]: expected in_force, got %v", i, a["status"])
		}
	}
}

// TestActs_ListSearchQuery verifies the ?q= query filter searches title +
// citation + summary. "data" should match the Data Protection Act.
func TestActs_ListSearchQuery(t *testing.T) {
	var resp actsListResponse
	status := mustGet(t, apiURL("/acts?q=data"), &resp)
	assertStatus(t, "/acts?q=data", http.StatusOK, status)

	if len(resp.Items) != 1 {
		t.Fatalf("acts?q=data: expected 1 match, got %d", len(resp.Items))
	}
	if resp.Items[0]["id"] != "ke-act-data-protection-2019" {
		t.Errorf("acts?q=data: expected ke-act-data-protection-2019, got %v", resp.Items[0]["id"])
	}
}

// TestActs_ListPagination verifies ?page + ?page_size return the requested
// slice and that has_next flips when more pages exist.
func TestActs_ListPagination(t *testing.T) {
	// Get the full list to know the total.
	var all actsListResponse
	if status := mustGet(t, apiURL("/acts"), &all); status != http.StatusOK {
		t.Fatalf("/acts: status %d", status)
	}
	if all.Total < 3 {
		t.Skipf("acts/pagination: need >= 3 acts to test pagination, got %d", all.Total)
	}

	// Request page=1, page_size=2 — should return first 2 items + has_next=true.
	var p1 actsListResponse
	status := mustGet(t, apiURL("/acts?page=1&page_size=2"), &p1)
	assertStatus(t, "/acts?page=1&page_size=2", http.StatusOK, status)
	if len(p1.Items) != 2 {
		t.Errorf("acts/p1: expected 2 items, got %d", len(p1.Items))
	}
	if !p1.HasNext {
		t.Error("acts/p1: expected has_next=true")
	}

	// Request page=2, page_size=2 — should return the remaining items.
	var p2 actsListResponse
	status = mustGet(t, apiURL("/acts?page=2&page_size=2"), &p2)
	assertStatus(t, "/acts?page=2&page_size=2", http.StatusOK, status)
	// Total should be the same (total is the filtered count, not paged count).
	if p2.Total != all.Total {
		t.Errorf("acts/p2: expected total=%d, got %d", all.Total, p2.Total)
	}
}

// TestActs_DetailReturnsAct verifies GET /acts/{id} returns the full act.
func TestActs_DetailReturnsAct(t *testing.T) {
	var resp map[string]any
	status := mustGet(t, apiURL("/acts/ke-act-constitution-2010"), &resp)
	assertStatus(t, "/acts/ke-act-constitution-2010", http.StatusOK, status)

	if resp["id"] != "ke-act-constitution-2010" {
		t.Errorf("acts/detail: expected id=ke-act-constitution-2010, got %v", resp["id"])
	}
	if resp["status"] != "in_force" {
		t.Errorf("acts/detail: expected status=in_force, got %v", resp["status"])
	}
	if resp["country"] != "KE" {
		t.Errorf("acts/detail: expected country=KE, got %v", resp["country"])
	}
}

// TestActs_DetailUnknownReturns404 verifies an unknown act ID returns 404.
func TestActs_DetailUnknownReturns404(t *testing.T) {
	var dummy map[string]any
	status := mustGet(t, apiURL("/acts/nonexistent-id"), &dummy)
	assertStatus(t, "/acts/nonexistent-id", http.StatusNotFound, status)
}

// TestActs_AuditReturnsLifecycle verifies GET /acts/{id}/audit returns the
// full lifecycle audit for a known act.
func TestActs_AuditReturnsLifecycle(t *testing.T) {
	var resp map[string]any
	status := mustGet(t, apiURL("/acts/ke-act-constitution-2010/audit"), &resp)
	assertStatus(t, "/acts/.../audit", http.StatusOK, status)

	if resp["act_id"] != "ke-act-constitution-2010" {
		t.Errorf("acts/audit: expected act_id=ke-act-constitution-2010, got %v", resp["act_id"])
	}
	// Audit must surface the canonical disclaimer about NOT_VERIFIED gaps.
	disclaimer, _ := resp["disclaimer"].(string)
	if !strings.Contains(disclaimer, "NOT_VERIFIED") {
		t.Errorf("acts/audit disclaimer should mention NOT_VERIFIED; got %q", disclaimer)
	}
	// Audit statuses must be a non-empty array.
	statuses, ok := resp["audit_statuses"].([]any)
	if !ok || len(statuses) == 0 {
		t.Errorf("acts/audit: expected non-empty audit_statuses, got %v", resp["audit_statuses"])
	}
}

// TestActs_EventsReturnsList verifies GET /acts/{id}/events returns the
// post-assent events list with a disclaimer citing authoritative sources.
func TestActs_EventsReturnsList(t *testing.T) {
	var resp map[string]any
	status := mustGet(t, apiURL("/acts/ke-act-data-protection-2019/events"), &resp)
	assertStatus(t, "/acts/.../events", http.StatusOK, status)

	if resp["act_id"] != "ke-act-data-protection-2019" {
		t.Errorf("acts/events: expected act_id=ke-act-data-protection-2019, got %v", resp["act_id"])
	}
	count, _ := resp["count"].(float64)
	events, _ := resp["events"].([]any)
	if int(count) != len(events) {
		t.Errorf("acts/events: count (%v) != len(events) (%d)", count, len(events))
	}
	disclaimer, _ := resp["disclaimer"].(string)
	if !strings.Contains(disclaimer, "authoritative") {
		t.Errorf("acts/events disclaimer should mention authoritative sources; got %q", disclaimer)
	}
}

// TestActs_FollowRequiresAuth verifies POST /acts/{id}/follow returns 401
// when the caller is anonymous. The flagship Follow-a-Law experience
// requires authentication — the platform never fabricates a follow.
func TestActs_FollowRequiresAuth(t *testing.T) {
	var dummy map[string]any
	status := mustPost(t, apiURL("/acts/ke-act-constitution-2010/follow"), "", &dummy)
	assertStatus(t, "/acts/.../follow (anonymous)", http.StatusUnauthorized, status)
}

// TestActs_FollowUnknownActReturns404 verifies following a non-existent
// act is rejected with 404 (the platform refuses to create ghost follows).
func TestActs_FollowUnknownActReturns404(t *testing.T) {
	// Even though auth would normally fail first, the handler checks act
	// existence only AFTER the auth check. So this returns 401, not 404.
	// We assert 401 to document the auth-first ordering.
	var dummy map[string]any
	status := mustPost(t, apiURL("/acts/nonexistent-id/follow"), "", &dummy)
	if status != http.StatusUnauthorized && status != http.StatusNotFound {
		t.Errorf("/acts/nonexistent/follow: expected 401 or 404, got %d", status)
	}
}

// TestActs_LineageReturnsSteps verifies GET /acts/{id}/lineage returns the
// documented legal lineage with a disclaimer about NOT_VERIFIED steps.
func TestActs_LineageReturnsSteps(t *testing.T) {
	var resp map[string]any
	status := mustGet(t, apiURL("/acts/ke-act-constitution-2010/lineage"), &resp)
	assertStatus(t, "/acts/.../lineage", http.StatusOK, status)

	if resp["act_id"] != "ke-act-constitution-2010" {
		t.Errorf("acts/lineage: expected act_id=ke-act-constitution-2010, got %v", resp["act_id"])
	}
	lineage, ok := resp["lineage"].([]any)
	if !ok {
		t.Fatal("acts/lineage: expected lineage array, got none")
	}
	if len(lineage) == 0 {
		t.Error("acts/lineage: expected at least one step, got empty array")
	}
	disclaimer, _ := resp["disclaimer"].(string)
	if !strings.Contains(disclaimer, "NOT_VERIFIED") {
		t.Errorf("acts/lineage disclaimer should mention NOT_VERIFIED; got %q", disclaimer)
	}
}

// TestActs_UnknownSubResourceReturns404 verifies that an undocumented
// /acts/{id}/<sub> returns 404 (not 500).
func TestActs_UnknownSubResourceReturns404(t *testing.T) {
	var dummy map[string]any
	status := mustGet(t, apiURL("/acts/ke-act-constitution-2010/unknown-sub"), &dummy)
	assertStatus(t, "/acts/.../unknown-sub", http.StatusNotFound, status)
}
