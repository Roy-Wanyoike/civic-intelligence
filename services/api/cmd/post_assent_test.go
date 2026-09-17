package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// dispatchActRouter runs a request through makeActRouter with the
// subscriptionStore injected and the principal populated the same way the
// real router does (via middleware.OptionalAuth + a StaticVerifier). This
// mirrors the dispatch helper in subscriptions_test.go but is scoped to the
// acts router. An anonymous principal skips the Authorization header.
func dispatchActRouter(t *testing.T, method, url string, p auth.Principal) *httptest.ResponseRecorder {
	t.Helper()
	store := NewSubscriptionStore()
	router := makeActRouter(store)
	req := httptest.NewRequest(method, url, nil)
	if p.IsAuthenticated() {
		req.Header.Set("Authorization", "Bearer test-token")
	}
	v := auth.StaticVerifier{Principal: p}
	wrapped := middleware.OptionalAuth(v)(router)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	return rr
}

// TestActAudit_ReturnsLifecycleStatuses verifies that the audit endpoint
// returns the full set of lifecycle statuses.
//
// Issue #215: the audit now delegates to domain.AuditForAct, so its output
// reflects the actual state of the act's fields. For
// ke-act-data-protection-2019 (which has AssentDate + CommencementDate but
// no PublicationDate), the audit reports:
//   - AssentRecorded   = true  (AssentDate is set)
//   - ActPublished     = false (sample data has no PublicationDate)
//   - CommencementNotice = true (CommencementDate is set + COMMENCEMENT event)
//   - AuditStatuses contains ASSENT_CONFIRMED, COMMENCEMENT_CONFIRMED,
//     REGULATIONS_TRACKED (REGULATION event exists), PARTIALLY_TRACKED,
//     DATA_GAP (publication_date_missing)
func TestActAudit_ReturnsLifecycleStatuses(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019/audit", nil)
	rec := httptest.NewRecorder()

	makeActRouter(NewSubscriptionStore())(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		ActID              string   `json:"act_id"`
		AssentRecorded     bool     `json:"assent_recorded"`
		ActPublished       bool     `json:"act_published"`
		CommencementNotice bool     `json:"commencement_notice"`
		AuditStatuses      []string `json:"audit_statuses"`
		DataGaps           []string `json:"data_gaps"`
		Disclaimer         string   `json:"disclaimer"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if !resp.AssentRecorded {
		t.Error("expected assent recorded (sample act has AssentDate)")
	}
	// Issue #215: actResponse does not carry a PublicationDate field, so
	// the audit must report ActPublished=false rather than the previous
	// hardcoded true. This is the honest state of the sample data.
	if resp.ActPublished {
		t.Error("expected ActPublished=false (sample data has no publication date; was hardcoded true before #215)")
	}
	if !resp.CommencementNotice {
		t.Error("expected commencement notice (sample act has CommencementDate + COMMENCEMENT event)")
	}
	if resp.Disclaimer == "" {
		t.Error("expected disclaimer")
	}
	if !contains(resp.AuditStatuses, "ASSENT_CONFIRMED") {
		t.Error("expected ASSENT_CONFIRMED in audit statuses")
	}
	if !contains(resp.AuditStatuses, "REGULATIONS_TRACKED") {
		t.Error("expected REGULATIONS_TRACKED (sample act has a REGULATION event)")
	}
	if !contains(resp.AuditStatuses, "COMMENCEMENT_CONFIRMED") {
		t.Error("expected COMMENCEMENT_CONFIRMED (sample act has CommencementDate)")
	}
	if !contains(resp.DataGaps, "publication_date_missing") {
		t.Error("expected publication_date_missing data gap (sample data has no publication date)")
	}
}

// TestActAudit_WithoutCommencementDate_ReportsNotVerified is the regression
// test for issue #215. Before #215, buildAudit hardcoded
// CommencementNotice=true for every act. After delegating to
// domain.AuditForAct, an act without a commencement date correctly reports
// NOT_VERIFIED for commencement.
func TestActAudit_WithoutCommencementDate_ReportsNotVerified(t *testing.T) {
	// Construct a synthetic actResponse with no commencement date — the
	// same shape as the sample data but with CommencementDate empty.
	actsNoCommencement := actResponse{
		ID:         "ke-act-test-no-commencement",
		Title:      "Test Act (No Commencement)",
		Citation:   "No. 99 of 2099",
		AssentDate: "2099-01-01",
		// CommencementDate intentionally empty.
		SourceURL: "https://example.test/act/99",
		Status:    "assented",
		Country:   "KE",
		Summary:   "Synthetic act with no commencement date for regression testing.",
	}
	audit := buildAudit(actsNoCommencement, nil)

	if audit.CommencementNotice {
		t.Error("CommencementNotice must be false for an act without a commencement date")
	}
	if !contains(audit.AuditStatuses, "NOT_VERIFIED") {
		t.Errorf("expected NOT_VERIFIED in audit statuses for missing commencement; got %v", audit.AuditStatuses)
	}
	if !contains(audit.DataGaps, "commencement_notice_not_found") {
		t.Errorf("expected commencement_notice_not_found data gap; got %v", audit.DataGaps)
	}
	if !audit.AssentRecorded {
		t.Error("expected assent recorded (AssentDate is set)")
	}
}

// TestActEvents_ReturnsPostAssentEvents verifies that the events endpoint
// returns the post-assent events for an Act.
func TestActEvents_ReturnsPostAssentEvents(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019/events", nil)
	rec := httptest.NewRecorder()

	makeActRouter(NewSubscriptionStore())(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		ActID  string           `json:"act_id"`
		Events []map[string]any `json:"events"`
		Count  int              `json:"count"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if resp.ActID != "ke-act-data-protection-2019" {
		t.Errorf("expected ke-act-data-protection-2019; got %s", resp.ActID)
	}
	if resp.Count == 0 {
		t.Error("expected at least one event")
	}
}

// TestFollowLaw_CreatesRealSubscription verifies the Follow-a-Law experience
// persists a real subscription (issue #216). Before #216, the handler
// returned a stub "followed: true" without writing anything. After #216, it
// creates a subscription in the SubscriptionStore and returns the
// subscription ID.
func TestFollowLaw_CreatesRealSubscription(t *testing.T) {
	store := NewSubscriptionStore()

	// Wrap the router with OptionalAuth so PrincipalFromRequest sees the
	// authenticated user (mirrors the real /api/v1/ mux).
	router := makeActRouter(store)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/acts/ke-act-data-protection-2019/follow", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	p := auth.Principal{UserID: "user-1"}
	wrapped := middleware.OptionalAuth(auth.StaticVerifier{Principal: p})(router)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d (body=%s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Followed       bool     `json:"followed"`
		SubscriptionID string   `json:"subscription_id"`
		ActID          string   `json:"act_id"`
		Target         string   `json:"target"`
		Monitors       []string `json:"monitors"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if !resp.Followed {
		t.Error("expected followed=true")
	}
	// Issue #216: subscription_id must be a real persisted ID, not empty.
	if resp.SubscriptionID == "" {
		t.Error("expected non-empty subscription_id (was a stub before #216)")
	}
	if resp.ActID != "ke-act-data-protection-2019" {
		t.Errorf("expected act_id ke-act-data-protection-2019; got %s", resp.ActID)
	}
	if resp.Target != "act:ke-act-data-protection-2019" {
		t.Errorf("expected target act:ke-act-data-protection-2019; got %s", resp.Target)
	}
	// Should monitor all 8 categories.
	expected := map[string]bool{
		"COMMENCEMENT":          false,
		"REGULATIONS":           false,
		"IMPLEMENTATION":        false,
		"AMENDMENTS":            false,
		"COURT_CASES":           false,
		"GOVERNMENT_NOTICES":    false,
		"INSTITUTIONAL_ACTIONS": false,
		"RELATED_BILLS":         false,
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

	// The subscription must actually be in the store — not a stub response.
	sub, ok := store.GetSubscription(resp.SubscriptionID)
	if !ok {
		t.Fatal("subscription not persisted in store (still a stub?)")
	}
	if sub.UserID != "user-1" {
		t.Errorf("expected persisted subscription UserID=user-1; got %s", sub.UserID)
	}
	if sub.Target != "act:ke-act-data-protection-2019" {
		t.Errorf("expected persisted subscription Target=act:ke-act-data-protection-2019; got %s", sub.Target)
	}
}

// TestFollowLaw_AnonymousRejected verifies the follow endpoint requires
// authentication (mirrors /api/v1/subscriptions).
func TestFollowLaw_AnonymousRejected(t *testing.T) {
	store := NewSubscriptionStore()
	router := makeActRouter(store)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/acts/ke-act-data-protection-2019/follow", nil)
	// No Authorization header → OptionalAuth yields Anonymous.
	wrapped := middleware.OptionalAuth(auth.StaticVerifier{Principal: auth.Anonymous()})(router)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for anonymous follow; got %d (body=%s)", rec.Code, rec.Body.String())
	}
	// Store must remain empty.
	if len(store.List("user-1")) != 0 {
		t.Error("store should be empty after anonymous follow attempt")
	}
}

// TestLineage_DataDrivenWithNotVerified verifies that the lineage endpoint
// is data-driven (issue #217). For ke-act-data-protection-2019:
//   - origin_bill, parliamentary_journey, act_publication, amendments,
//     court_decisions have no evidence → must be NOT_VERIFIED.
//   - presidential_assent (AssentDate set), commencement (COMMENCEMENT
//     event), regulations (REGULATION event), current_status must be VERIFIED.
func TestLineage_DataDrivenWithNotVerified(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019/lineage", nil)
	rec := httptest.NewRecorder()

	makeActRouter(NewSubscriptionStore())(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		ActID   string           `json:"act_id"`
		Lineage []map[string]any `json:"lineage"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Lineage) < 9 {
		t.Errorf("expected at least 9 lineage steps; got %d", len(resp.Lineage))
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

	// Bucket each step by status so we can assert which steps have evidence.
	verified := map[string]bool{}
	notVerified := map[string]bool{}
	for _, s := range resp.Lineage {
		step, _ := s["step"].(string)
		status, _ := s["status"].(string)
		switch status {
		case "VERIFIED":
			verified[step] = true
		case "NOT_VERIFIED":
			notVerified[step] = true
		default:
			t.Errorf("step %s has unexpected status %q (must be VERIFIED or NOT_VERIFIED)", step, status)
		}
	}

	// Steps with evidence in the sample data — these MUST be VERIFIED.
	for _, step := range []string{"presidential_assent", "commencement", "regulations", "current_status"} {
		if !verified[step] {
			t.Errorf("expected %s to be VERIFIED (sample data has evidence)", step)
		}
	}
	// Steps without evidence — these MUST be NOT_VERIFIED (issue #217).
	for _, step := range []string{"origin_bill", "parliamentary_journey", "act_publication", "amendments", "court_decisions"} {
		if !notVerified[step] {
			t.Errorf("expected %s to be NOT_VERIFIED (no evidence in sample data)", step)
		}
	}
}

// TestLineage_ActWithoutCommencement_StepIsNotVerified verifies that for an
// act without a commencement date, the lineage "commencement" step is
// NOT_VERIFIED (issue #217 regression).
func TestLineage_ActWithoutCommencement_StepIsNotVerified(t *testing.T) {
	actsNoCommencement := actResponse{
		ID:         "ke-act-test-no-commencement",
		Title:      "Test Act (No Commencement)",
		Citation:   "No. 99 of 2099",
		AssentDate: "2099-01-01",
		// CommencementDate intentionally empty.
		SourceURL: "https://example.test/act/99",
		Status:    "assented",
	}
	steps := buildLineage(actsNoCommencement, nil)
	var commStep *lineageStep
	for i := range steps {
		if steps[i].Step == "commencement" {
			commStep = &steps[i]
			break
		}
	}
	if commStep == nil {
		t.Fatal("expected a 'commencement' lineage step")
	}
	if commStep.Status != "NOT_VERIFIED" {
		t.Errorf("expected commencement step status NOT_VERIFIED; got %s (status=%s date=%s)", commStep.Status, commStep.Status, commStep.Date)
	}
	if commStep.Description != "No authoritative record found yet." {
		t.Errorf("expected NOT_VERIFIED description 'No authoritative record found yet.'; got %q", commStep.Description)
	}
}

// TestActRouter_ActDetailStillWorks verifies that the existing /acts/{id}
// detail endpoint still works after the router was changed to accept a
// subscription store.
func TestActRouter_ActDetailStillWorks(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019", nil)
	rec := httptest.NewRecorder()

	makeActRouter(NewSubscriptionStore())(rec, req)

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
