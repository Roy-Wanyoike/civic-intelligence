package main

import (
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"
)

// TestScorecard_ReturnsFactualRecord verifies the scorecard endpoint returns
// the factual record for a known sample person. The platform must NEVER
// rank MPs or imply political approval — this test enforces the
// NO_POLITICAL_PERFORMANCE_SCORE invariant (task ENG-K2):
//   - attendance_rate is a 0.0–1.0 rate, NOT a "performance score"
//   - bills_sponsored, questions_asked, votes_recorded, statements_made are
//     raw integer counts, NOT weighted aggregates
//   - every metric carries a source_url
//   - the canonical disclaimer string is present
//   - reality_layer is FACT
func TestScorecard_ReturnsFactualRecord(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/people/person-001/scorecard", nil)
        rec := httptest.NewRecorder()

        makeScorecardHandler()(rec, req)

        if rec.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rec.Code)
        }

        var resp map[string]any
        if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }

        // 1. Identity fields present.
        if resp["person_id"] != "person-001" {
                t.Errorf("expected person_id=person-001, got %v", resp["person_id"])
        }
        if resp["name"] != "Kimani Ichung'wah" {
                t.Errorf("expected name Kimani Ichung'wah, got %v", resp["name"])
        }
        if resp["role"] == "" {
                t.Error("role is required, got empty")
        }
        if resp["constituency"] == "" {
                t.Error("constituency is required, got empty")
        }

        // 2. The canonical disclaimer is present verbatim — guards against
        // silent drift in editorial posture.
        got := resp["disclaimer"].(string)
        if got != scorecardDisclaimer {
                t.Errorf("disclaimer drifted: got %q", got)
        }
        if resp["reality_layer"] != "FACT" {
                t.Errorf("expected reality_layer FACT, got %v", resp["reality_layer"])
        }

        // 3. attendance_rate is a 0.0–1.0 rate. The test asserts the value
        // is within [0.0, 1.0] AND is a float — never a 0-100 score.
        ar, ok := resp["attendance_rate"].(float64)
        if !ok {
                t.Fatalf("attendance_rate must be float64, got %T", resp["attendance_rate"])
        }
        if ar < 0.0 || ar > 1.0 {
                t.Errorf("attendance_rate must be in [0.0, 1.0], got %v", ar)
        }
        // attendance_source_url must be present so a citizen can verify.
        if resp["attendance_source_url"] == nil || resp["attendance_source_url"] == "" {
                t.Error("attendance_source_url is required")
        }

        // 4. The four raw count metrics must be integers (JSON numbers),
        // NOT weighted aggregates or scores.
        for _, key := range []string{"bills_sponsored", "questions_asked", "votes_recorded", "statements_made"} {
                v, ok := resp[key].(float64)
                if !ok {
                        t.Errorf("%s must be a number, got %T", key, resp[key])
                        continue
                }
                if v < 0 {
                        t.Errorf("%s must be non-negative, got %v", key, v)
                }
                // Must be a whole number (count), not a fractional value.
                if v != float64(int(v)) {
                        t.Errorf("%s must be a whole-number count, got %v", key, v)
                }
        }

        // 5. NO composite "score", "rating", "performance", "grade", "rank"
        // or any other aggregation field exists anywhere in the response.
        forbiddenKeys := []string{
                "score", "rating", "performance", "performance_score",
                "grade", "rank", "overall_score", "mp_rating", "approval",
                "approval_rating", "approval_score",
        }
        for _, key := range forbiddenKeys {
                if _, exists := resp[key]; exists {
                        t.Errorf("response MUST NOT contain key %q — the platform does not rank MPs (task ENG-K2)", key)
                }
        }

        // 6. Every metric in `metrics` map carries a source_url.
        metrics, ok := resp["metrics"].(map[string]any)
        if !ok {
                t.Fatal("metrics map missing")
        }
        for k, mv := range metrics {
                m, ok := mv.(map[string]any)
                if !ok {
                        t.Errorf("metric %q is not an object", k)
                        continue
                }
                if m["source_url"] == nil || m["source_url"] == "" {
                        t.Errorf("metric %q missing source_url", k)
                }
        }

        // 7. bills_sponsored_list and committee_memberships are non-empty for
        // the sample people and each item carries a source_url.
        bills, _ := resp["bills_sponsored_list"].([]any)
        if len(bills) == 0 {
                t.Error("expected at least one bill in bills_sponsored_list")
        }
        for _, b := range bills {
                bm := b.(map[string]any)
                if bm["source_url"] == nil || bm["source_url"] == "" {
                        t.Errorf("bill missing source_url: %v", bm)
                }
        }
        cmtes, _ := resp["committee_memberships"].([]any)
        if len(cmtes) == 0 {
                t.Error("expected at least one committee membership")
        }
        for _, c := range cmtes {
                cm := c.(map[string]any)
                if cm["source_url"] == nil || cm["source_url"] == "" {
                        t.Errorf("committee membership missing source_url: %v", cm)
                }
        }

        // 8. recent_activity items each carry a source_url.
        activity, _ := resp["recent_activity"].([]any)
        if len(activity) == 0 {
                t.Error("expected recent_activity timeline")
        }
        for _, a := range activity {
                am := a.(map[string]any)
                if am["source_url"] == nil || am["source_url"] == "" {
                        t.Errorf("activity item missing source_url: %v", am)
                }
        }

        // 9. source_agencies lists the official source families a citizen can
        // cross-check against (parliament.go.ke, Hansard, Votes and Proceedings).
        agencies, _ := resp["source_agencies"].([]any)
        if len(agencies) < 2 {
                t.Errorf("expected at least 2 source_agencies, got %d", len(agencies))
        }
}

// TestScorecard_Returns404ForUnknownPerson verifies the endpoint does NOT
// invent a scorecard for an unknown person_id. Returning 404 (not 200 with
// empty fields) prevents the frontend from rendering a fabricated record.
func TestScorecard_Returns404ForUnknownPerson(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/people/does-not-exist/scorecard", nil)
        rec := httptest.NewRecorder()

        makeScorecardHandler()(rec, req)

        if rec.Code != http.StatusNotFound {
                t.Fatalf("expected 404, got %d", rec.Code)
        }
}

// TestScorecard_AllFiveSamplePeopleHaveScorecards verifies that all 5 sample
// people return a non-empty scorecard with every required field. This guards
// against an accidental truncation of the samplePeople seed (e.g. if someone
// deletes a row while editing the file).
func TestScorecard_AllFiveSamplePeopleHaveScorecards(t *testing.T) {
        if len(samplePeople) != 5 {
                t.Fatalf("expected exactly 5 sample people for ENG-K2, got %d", len(samplePeople))
        }
        for _, p := range samplePeople {
                req := httptest.NewRequest(http.MethodGet, "/api/v1/people/"+p.PersonID+"/scorecard", nil)
                rec := httptest.NewRecorder()

                makeScorecardHandler()(rec, req)

                if rec.Code != http.StatusOK {
                        t.Errorf("person %s: expected 200, got %d", p.PersonID, rec.Code)
                        continue
                }
                var resp map[string]any
                _ = json.Unmarshal(rec.Body.Bytes(), &resp)
                if resp["disclaimer"] != scorecardDisclaimer {
                        t.Errorf("person %s: disclaimer mismatch", p.PersonID)
                }
                if resp["reality_layer"] != "FACT" {
                        t.Errorf("person %s: reality_layer not FACT", p.PersonID)
                }
                // attendance rate in [0, 1]
                ar, _ := resp["attendance_rate"].(float64)
                if ar < 0.0 || ar > 1.0 {
                        t.Errorf("person %s: attendance_rate %v out of [0,1]", p.PersonID, ar)
                }
        }
}

// TestScorecard_DisclaimerWordingIsStable verifies the canonical disclaimer
// wording. The string is the platform's editorial posture — any change must
// be deliberate and breaking. If you find yourself editing this constant,
// please consult the docs/NO_FAKE_COMPLETION.md policy first.
func TestScorecard_DisclaimerWordingIsStable(t *testing.T) {
        want := "This scorecard presents factual records only. The platform does not rank MPs or imply political approval."
        if scorecardDisclaimer != want {
                t.Errorf("disclaimer drifted to %q", scorecardDisclaimer)
        }
}

// TestPeopleList_ReturnsSamplePeopleWithScorecardLinks verifies the people
// list endpoint returns the 5 sample people each with a scorecard_url link.
// This is the entry point that lets a citizen find their MP's scorecard.
func TestPeopleList_ReturnsSamplePeopleWithScorecardLinks(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/people", nil)
        rec := httptest.NewRecorder()

        makePeopleListHandler()(rec, req)

        if rec.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rec.Code)
        }
        var resp struct {
                Items []scorecardListItem `json:"items"`
                Total int                  `json:"total"`
        }
        if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Total != len(samplePeople) {
                t.Errorf("expected total=%d, got %d", len(samplePeople), resp.Total)
        }
        for _, item := range resp.Items {
                if item.ScorecardURL == "" {
                        t.Errorf("person %s missing scorecard_url", item.PersonID)
                }
        }
}
