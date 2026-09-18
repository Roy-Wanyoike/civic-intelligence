package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
)

// TestActAudit_ReturnsLifecycleStatuses verifies that the audit endpoint
// delegates to domain.AuditForAct (issue #215) instead of hardcoding
// "everything is confirmed". The Data Protection Act 2019 seed has an
// assent date and a commencement date but NO publication date, so the
// audit must surface:
//   - ASSENT_CONFIRMED        (AssentedAt is set)
//   - COMMENCEMENT_CONFIRMED  (CommencementDate is set)
//   - REGULATIONS_TRACKED    (a REGULATION post-assent event exists)
//   - PARTIALLY_TRACKED + DATA_GAP (publication date is missing)
//
// ActPublished must be FALSE because the seed has no PublicationDate.
// This is the regression test for #215 — the audit no longer pretends the
// Act was published when no authoritative publication record exists.
func TestActAudit_ReturnsLifecycleStatuses(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019/audit", nil)
	rec := httptest.NewRecorder()

	makeActRouter()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		ActID              string   `json:"act_id"`
		AuditStatuses      []string `json:"audit_statuses"`
		Disclaimer         string   `json:"disclaimer"`
		AssentRecorded     bool     `json:"assent_recorded"`
		ActPublished       bool     `json:"act_published"`
		CommencementNotice bool     `json:"commencement_notice"`
		RegulationsIssued  bool     `json:"regulations_issued"`
		DataGaps           []string `json:"data_gaps"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if !resp.AssentRecorded {
		t.Error("expected assent recorded (AssentedAt is set in seed data)")
	}
	if resp.ActPublished {
		t.Error("expected ActPublished=FALSE — seed data has no PublicationDate; the audit must NOT fabricate it (#215)")
	}
	if !resp.CommencementNotice {
		t.Error("expected commencement notice (CommencementDate is set in seed data)")
	}
	if !resp.RegulationsIssued {
		t.Error("expected regulations issued (a REGULATION event is seeded for this Act)")
	}
	if resp.Disclaimer == "" {
		t.Error("expected disclaimer")
	}
	if !contains(resp.AuditStatuses, "ASSENT_CONFIRMED") {
		t.Error("expected ASSENT_CONFIRMED in audit statuses")
	}
	if !contains(resp.AuditStatuses, "COMMENCEMENT_CONFIRMED") {
		t.Error("expected COMMENCEMENT_CONFIRMED in audit statuses")
	}
	if !contains(resp.AuditStatuses, "REGULATIONS_TRACKED") {
		t.Error("expected REGULATIONS_TRACKED (seed act has a REGULATION event)")
	}
	if !contains(resp.AuditStatuses, "PARTIALLY_TRACKED") {
		t.Error("expected PARTIALLY_TRACKED (publication_date_missing data gap)")
	}
	if !contains(resp.AuditStatuses, "DATA_GAP") {
		t.Error("expected DATA_GAP (publication_date_missing)")
	}
	if !contains(resp.DataGaps, "publication_date_missing") {
		t.Errorf("expected 'publication_date_missing' in data gaps; got %v", resp.DataGaps)
	}
}

// TestActAudit_NoCommencementDate_NotVerified is the #215 regression test.
//
// An act without a commencement date MUST report NOT_VERIFIED for the
// commencement dimension — NEVER "the Act never commenced". This is the
// spec invariant (section 37): absence of evidence is not evidence of
// absence. The test calls buildAudit directly with a synthetic act that
// has no commencement date and verifies the audit surfaces NOT_VERIFIED.
func TestActAudit_NoCommencementDate_NotVerified(t *testing.T) {
	// Synthetic actResponse with no commencement date (only assent).
	// domain.AuditForAct must flag commencement as NOT_VERIFIED + a
	// "commencement_notice_not_found" data gap.
	act := actResponse{
		ID:         "act-test-no-commencement",
		Title:      "Test Act (no commencement)",
		Citation:   "No. 99 of 2024",
		AssentDate: time.Now().Add(-60 * 24 * time.Hour).Format("2006-01-02"),
		Status:     "in_force",
		Country:    "KE",
	}

	audit := buildAudit(act, nil)

	if audit.AssentRecorded != true {
		t.Error("expected assent recorded (AssentDate is set)")
	}
	if audit.CommencementNotice {
		t.Error("expected CommencementNotice=FALSE — no commencement date was supplied")
	}
	if !contains(audit.AuditStatuses, "ASSENT_CONFIRMED") {
		t.Error("expected ASSENT_CONFIRMED (AssentDate is set)")
	}
	if !contains(audit.AuditStatuses, "NOT_VERIFIED") {
		t.Error("expected NOT_VERIFIED — commencement notice is missing (#215 spec invariant)")
	}
	if !contains(audit.AuditStatuses, "PARTIALLY_TRACKED") {
		t.Error("expected PARTIALLY_TRACKED (data gap present)")
	}
	if !contains(audit.AuditStatuses, "DATA_GAP") {
		t.Error("expected DATA_GAP (commencement notice missing)")
	}
	if !contains(audit.DataGaps, "commencement_notice_not_found") {
		t.Errorf("expected 'commencement_notice_not_found' data gap; got %v", audit.DataGaps)
	}
}

// TestActAudit_TracksPostAssentEvents verifies the audit reflects real
// post-assent events (regulations, amendments, court challenges, repeal)
// sourced through domain.AuditForAct.
func TestActAudit_TracksPostAssentEvents(t *testing.T) {
	now := time.Now().UTC()
	act := actResponse{
		ID:               "act-test-events",
		Title:            "Test Act (with events)",
		Citation:         "No. 1 of 2024",
		AssentDate:       now.Add(-90 * 24 * time.Hour).Format("2006-01-02"),
		CommencementDate: now.Add(-60 * 24 * time.Hour).Format("2006-01-02"),
		Status:           "amended",
	}
	events := []PostAssentEventResponse{
		{EventType: "REGULATION", EventDate: now.Add(-30 * 24 * time.Hour), Title: "Regulation issued"},
		{EventType: "COURT_CHALLENGE", EventDate: now.Add(-15 * 24 * time.Hour), Title: "Court challenge"},
		{EventType: "JUDICIAL_DECISION", EventDate: now.Add(-10 * 24 * time.Hour), Title: "High Court decision"},
		{EventType: "AMENDMENT", EventDate: now.Add(-5 * 24 * time.Hour), Title: "Amendment Act"},
	}

	audit := buildAudit(act, events)

	if !audit.RegulationsIssued {
		t.Error("expected regulations issued")
	}
	if !audit.CourtChallenged {
		t.Error("expected court challenged")
	}
	if !audit.JudicialDecision {
		t.Error("expected judicial decision")
	}
	if !audit.Amended {
		t.Error("expected amended")
	}
	if !contains(audit.AuditStatuses, "REGULATIONS_TRACKED") {
		t.Error("expected REGULATIONS_TRACKED")
	}
	if !contains(audit.AuditStatuses, "JUDICIAL_HISTORY_TRACKED") {
		t.Error("expected JUDICIAL_HISTORY_TRACKED")
	}
	if !contains(audit.AuditStatuses, "AMENDMENTS_TRACKED") {
		t.Error("expected AMENDMENTS_TRACKED")
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

// TestFollowLaw_CreatesRealSubscription verifies the Follow-a-Law
// experience creates a REAL subscription record in the shared
// subscriptionStore (issue #216). The response must include a non-empty
// subscription_id, target="act:"+actID, and all 8 monitor categories.
// The persisted record must be retrievable via /api/v1/subscriptions
// (proving the follow endpoint wrote to the same store).
func TestFollowLaw_CreatesRealSubscription(t *testing.T) {
	// Use a unique user ID so this test does not collide with other tests
	// that exercise the shared package-level subscriptionStore.
	const userID = "user-followlaw-test"
	const actID = "ke-act-data-protection-2019"

	p := auth.Principal{UserID: userID, Scopes: auth.Scopes{auth.ScopeNotificationWrite}}
	rr := dispatch(makeActRouter(), http.MethodPost, "/api/v1/acts/"+actID+"/follow", nil, p)

	if rr.Code != 200 {
		t.Fatalf("expected 200; got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Followed       bool     `json:"followed"`
		ActID          string   `json:"act_id"`
		SubscriptionID string   `json:"subscription_id"`
		Target         string   `json:"target"`
		Monitors       []string `json:"monitors"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.Followed {
		t.Error("expected followed=true")
	}
	if resp.ActID != actID {
		t.Errorf("expected act_id=%s; got %s", actID, resp.ActID)
	}
	if resp.SubscriptionID == "" {
		t.Fatal("expected non-empty subscription_id (issue #216 — must not be a stub)")
	}
	if resp.Target != "act:"+actID {
		t.Errorf("expected target='act:%s'; got %q", actID, resp.Target)
	}
	if len(resp.Monitors) != 8 {
		t.Errorf("expected 8 monitor categories; got %d", len(resp.Monitors))
	}
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

	// Verify the follow endpoint actually persisted the record by querying
	// the shared store. The follow endpoint and the /api/v1/subscriptions
	// endpoint MUST use the same store (issue #216 contract).
	follows := subscriptionStore.List(userID)
	found := false
	for _, f := range follows {
		if f.ID == resp.SubscriptionID && f.EntityType == EntityAct && f.EntityID == actID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("subscription %s not found in shared subscriptionStore for user %s (entity_type=act, entity_id=%s)",
			resp.SubscriptionID, userID, actID)
	}
}

// TestFollowLaw_AnonymousRejected verifies the follow endpoint requires
// authentication (issue #216). The platform never attributes a follow to
// an anonymous caller.
func TestFollowLaw_AnonymousRejected(t *testing.T) {
	rr := dispatch(makeActRouter(), http.MethodPost, "/api/v1/acts/ke-act-data-protection-2019/follow", nil, auth.Anonymous())
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for anonymous follow; got %d", rr.Code)
	}
}

// TestFollowLaw_ActNotFound verifies the follow endpoint refuses to create
// a subscription for a non-existent Act (issue #216). This prevents ghost
// follows from accumulating in the subscriptionStore.
func TestFollowLaw_ActNotFound(t *testing.T) {
	p := auth.Principal{UserID: "user-followlaw-notfound"}
	rr := dispatch(makeActRouter(), http.MethodPost, "/api/v1/acts/does-not-exist/follow", nil, p)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for follow on non-existent act; got %d", rr.Code)
	}
}

// TestLineage_ReturnsFullChain verifies the lineage endpoint returns the
// full Bill → Act → Amendments → ... → Current Status chain (issue #217).
//
// Each step is independently sourced — missing evidence is reported as
// NOT_VERIFIED. For the Data Protection Act 2019 (which has assent,
// commencement, and a REGULATION event), the following steps must be
// VERIFIED: presidential_assent, act_publication, commencement,
// regulations, current_status. The origin_bill + parliamentary_journey
// steps are NOT_VERIFIED because the platform does not yet track the
// originating Bill on the Act. The amendments + court_decisions steps
// are NOT_VERIFIED because no AMENDMENT / JUDICIAL_DECISION events are
// seeded for this Act.
func TestLineage_ReturnsFullChain(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts/ke-act-data-protection-2019/lineage", nil)
	rec := httptest.NewRecorder()

	makeActRouter()(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp struct {
		ActID   string           `json:"act_id"`
		Lineage []map[string]any `json:"lineage"`
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

	// Build a step → status map for assertion.
	statusByStep := map[string]string{}
	for _, s := range resp.Lineage {
		step, _ := s["step"].(string)
		status, _ := s["status"].(string)
		statusByStep[step] = status
	}

	// Steps that MUST be VERIFIED for the Data Protection Act 2019 seed.
	for _, step := range []string{"presidential_assent", "act_publication", "commencement", "regulations", "current_status"} {
		if statusByStep[step] != "VERIFIED" {
			t.Errorf("expected step %s to be VERIFIED; got %q", step, statusByStep[step])
		}
	}
	// Steps that MUST be NOT_VERIFIED for this seed (no Bill link, no AMENDMENT / COURT event).
	for _, step := range []string{"origin_bill", "parliamentary_journey", "amendments", "court_decisions"} {
		if statusByStep[step] != "NOT_VERIFIED" {
			t.Errorf("expected step %s to be NOT_VERIFIED; got %q", step, statusByStep[step])
		}
	}
	// Every NOT_VERIFIED step must carry the canonical description.
	for _, s := range resp.Lineage {
		status, _ := s["status"].(string)
		if status == "NOT_VERIFIED" {
			desc, _ := s["description"].(string)
			if desc != "No authoritative record found yet." {
				t.Errorf("expected NOT_VERIFIED step %q to use canonical description; got %q",
					s["step"], desc)
			}
		}
	}
}

// TestLineage_ActWithoutCommencement_StepIsNotVerified is the #217
// regression test. An act without a commencement date must have the
// "commencement" lineage step reported as NOT_VERIFIED (never inferred).
func TestLineage_ActWithoutCommencement_StepIsNotVerified(t *testing.T) {
	act := actResponse{
		ID:         "act-test-no-comm",
		Title:      "Test Act (no commencement)",
		Citation:   "No. 99 of 2024",
		AssentDate: time.Now().Add(-60 * 24 * time.Hour).Format("2006-01-02"),
		Status:     "in_force",
		// CommencementDate intentionally empty.
	}

	steps := buildLineage(act, nil)
	statusByStep := map[string]string{}
	for _, s := range steps {
		statusByStep[s.Step] = s.Status
	}
	if statusByStep["commencement"] != "NOT_VERIFIED" {
		t.Errorf("expected commencement step NOT_VERIFIED when no commencement date; got %q",
			statusByStep["commencement"])
	}
	if statusByStep["presidential_assent"] != "VERIFIED" {
		t.Errorf("expected presidential_assent step VERIFIED (AssentDate is set); got %q",
			statusByStep["presidential_assent"])
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
