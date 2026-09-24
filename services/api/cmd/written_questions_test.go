package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// writtenQuestionJSON is the JSON-decoded form of a single WrittenQuestion
// item. Decoding into a struct (rather than map[string]any) lets the
// tests reference fields by name + type, which makes failures easier
// to diagnose when a refactor renames a field or flips an omitempty
// tag.
type writtenQuestionJSON struct {
	ID           string `json:"id"`
	MPID         string `json:"mp_id"`
	MPName       string `json:"mp_name"`
	Minister     string `json:"minister"`
	Ministry     string `json:"ministry"`
	QuestionText string `json:"question_text"`
	AskedAt      string `json:"asked_at"`
	ResponseText string `json:"response_text,omitempty"`
	RespondedAt  string `json:"responded_at,omitempty"`
	Deadline     string `json:"deadline"`
	Status       string `json:"status"`
	SourceURL    string `json:"source_url"`
}

// writtenQuestionsListPayload is the JSON envelope returned by
// GET /api/v1/questions/written. Only the fields the tests assert on
// are decoded — the rest are ignored.
type writtenQuestionsListPayload struct {
	Items  []writtenQuestionJSON `json:"items"`
	Total  int                   `json:"total"`
	Source string                `json:"source"`
	MPID   string                `json:"mp_id,omitempty"`
	Status string                `json:"status,omitempty"`
}

// writtenQuestionsByPersonPayload is the JSON envelope returned by
// GET /api/v1/people/{id}/questions.
type writtenQuestionsByPersonPayload struct {
	MPID         string                `json:"mp_id"`
	Name         string                `json:"name"`
	Items        []writtenQuestionJSON `json:"items"`
	Total        int                   `json:"total"`
	Source       string                `json:"source"`
	ScorecardURL string                `json:"scorecard_url"`
}

// validWrittenQuestionStatuses is the canonical set of
// WrittenQuestionStatus wire values. Used by the tests to assert
// every response item carries one of the 3 allowed statuses (no
// typos, no silently-introduced "open" / "closed" / "late" variants).
// Kept in the test file (rather than reading
// allWrittenQuestionStatuses from written_questions.go) so the test
// fails loudly if a new status is added without a corresponding test
// update.
var validWrittenQuestionStatuses = map[string]bool{
	"pending":  true,
	"answered": true,
	"overdue":  true,
}

// expectedWrittenQuestions returns the seed-matrix written questions
// for the given MP, sorted most-recent-first by asked_at. The test
// uses this to cross-check the response items against the seed slice
// (catches a refactor that loses the sort or drops a row).
func expectedWrittenQuestions(t *testing.T, mpID string) []writtenQuestionJSON {
	t.Helper()
	want := make([]writtenQuestionJSON, 0, 3)
	for _, q := range seedWrittenQuestionsSorted {
		if q.MPID != mpID {
			continue
		}
		want = append(want, writtenQuestionJSON{
			ID:           q.ID,
			MPID:         q.MPID,
			MPName:       q.MPName,
			Minister:     q.Minister,
			Ministry:     q.Ministry,
			QuestionText: q.QuestionText,
			AskedAt:      q.AskedAt,
			ResponseText: q.ResponseText,
			RespondedAt:  q.RespondedAt,
			Deadline:     q.Deadline,
			Status:       string(q.Status),
			SourceURL:    q.SourceURL,
		})
	}
	if len(want) == 0 {
		t.Fatalf("expected seed slice to have ≥1 written question for %s", mpID)
	}
	return want
}

// TestWrittenQuestionsList_ReturnsQuestions verifies the
// /api/v1/questions/written list endpoint returns all 15 seed
// questions, every item carrying all required fields, every status
// one of the 3 allowed values, and the items list sorted
// most-recent-first by asked_at.
//
// The test drives the list handler directly. The 15-item invariant
// guards against an accidental truncation of the seed slice (e.g. if
// someone deletes a row while editing the file).
func TestWrittenQuestionsList_ReturnsQuestions(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/written", nil)
	rr := httptest.NewRecorder()

	makeWrittenQuestionsListHandler()(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	var resp writtenQuestionsListPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Issue #286 acceptance: 15 written questions across 5 sample MPs.
	if resp.Total != len(sampleWrittenQuestions) {
		t.Errorf("expected total=%d (matches seed slice), got %d", len(sampleWrittenQuestions), resp.Total)
	}
	if resp.Total != 15 {
		t.Errorf("expected 15 seed written questions (issue #286 acceptance bar), got %d", resp.Total)
	}
	if len(resp.Items) != resp.Total {
		t.Errorf("len(items)=%d but total=%d — the two MUST match", len(resp.Items), resp.Total)
	}
	if resp.Source != "seed" {
		t.Errorf("expected source=seed (issue #286 acceptance: seed data only), got %q", resp.Source)
	}
	// No filters were supplied → echo fields are empty.
	if resp.MPID != "" {
		t.Errorf("expected mp_id echo to be empty when no filter supplied, got %q", resp.MPID)
	}
	if resp.Status != "" {
		t.Errorf("expected status echo to be empty when no filter supplied, got %q", resp.Status)
	}

	// Assert every item carries all 10 required fields + every status
	// is one of the 3 allowed values + items are sorted
	// most-recent-first by asked_at.
	previousDate := ""
	for i, item := range resp.Items {
		if item.ID == "" {
			t.Errorf("item[%d]: expected id to be non-empty", i)
		}
		if item.MPID == "" {
			t.Errorf("item[%d]: expected mp_id to be non-empty", i)
		}
		if item.MPName == "" {
			t.Errorf("item[%d]: expected mp_name to be non-empty", i)
		}
		if item.Minister == "" {
			t.Errorf("item[%d]: expected minister to be non-empty", i)
		}
		if item.Ministry == "" {
			t.Errorf("item[%d]: expected ministry to be non-empty", i)
		}
		if item.QuestionText == "" {
			t.Errorf("item[%d]: expected question_text to be non-empty", i)
		}
		if item.AskedAt == "" {
			t.Errorf("item[%d]: expected asked_at to be non-empty", i)
		}
		if item.Deadline == "" {
			t.Errorf("item[%d]: expected deadline to be non-empty", i)
		}
		if item.SourceURL == "" {
			t.Errorf("item[%d]: expected source_url to be non-empty", i)
		}
		if !validWrittenQuestionStatuses[item.Status] {
			t.Errorf("item[%d]: expected status to be one of pending/answered/overdue, got %q", i, item.Status)
		}
		// answered rows MUST carry response_text + responded_at;
		// pending + overdue MUST NOT.
		switch item.Status {
		case "answered":
			if item.ResponseText == "" {
				t.Errorf("item[%d]: answered row missing response_text", i)
			}
			if item.RespondedAt == "" {
				t.Errorf("item[%d]: answered row missing responded_at", i)
			}
		case "pending", "overdue":
			if item.ResponseText != "" {
				t.Errorf("item[%d]: %s row must NOT carry response_text", i, item.Status)
			}
			if item.RespondedAt != "" {
				t.Errorf("item[%d]: %s row must NOT carry responded_at", i, item.Status)
			}
		}
		// Items sorted most-recent-first (asked_at descending).
		if previousDate != "" && item.AskedAt > previousDate {
			t.Errorf("item[%d]: asked_at not sorted most-recent-first (previous=%q, current=%q)", i, previousDate, item.AskedAt)
		}
		previousDate = item.AskedAt
	}

	// Status distribution: 5 pending, 5 answered, 5 overdue (the seed
	// slice is hand-balanced across the 3 statuses).
	statusCounts := map[string]int{}
	for _, item := range resp.Items {
		statusCounts[item.Status]++
	}
	for _, s := range []string{"pending", "answered", "overdue"} {
		if statusCounts[s] != 5 {
			t.Errorf("expected 5 %s questions (seed balance), got %d", s, statusCounts[s])
		}
	}
}

// TestWrittenQuestionsList_FiltersByMP verifies the list endpoint
// filters by the mp_id query param. Every returned item must belong
// to the requested MP, the total must match the seed slice's count
// for that MP (3 per MP), and the mp_id echo field must be populated
// so the frontend can verify the filter was honoured.
//
// The test also covers the /api/v1/people/{id}/questions sub-resource
// (which routes through handlePeople → handleWrittenQuestionsByPerson)
// by asserting that the per-MP filter + the per-MP endpoint return the
// same item set (catches a refactor that breaks the filter / dispatch
// divergence between the two paths).
func TestWrittenQuestionsList_FiltersByMP(t *testing.T) {
	const personID = "person-001"
	want := expectedWrittenQuestions(t, personID)

	// 1. /api/v1/questions/written?mp_id=person-001
	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/written?mp_id="+personID, nil)
	rr := httptest.NewRecorder()
	makeWrittenQuestionsListHandler()(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	var resp writtenQuestionsListPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.MPID != personID {
		t.Errorf("expected mp_id echo=%q, got %q", personID, resp.MPID)
	}
	if resp.Total != len(want) {
		t.Errorf("expected total=%d (matches seed slice for %s), got %d", len(want), personID, resp.Total)
	}
	if resp.Total != 3 {
		t.Errorf("expected 3 questions per MP (seed balance), got %d", resp.Total)
	}
	if len(resp.Items) != len(want) {
		t.Fatalf("expected %d items, got %d", len(want), len(resp.Items))
	}
	// Every item belongs to the requested MP.
	for i, item := range resp.Items {
		if item.MPID != personID {
			t.Errorf("item[%d]: expected mp_id=%q, got %q (every item in this filtered list MUST belong to the requested MP)", i, personID, item.MPID)
		}
		if item.MPName == "" {
			t.Errorf("item[%d]: expected mp_name to be non-empty", i)
		}
	}
	// Cross-check the response items against the seed slice (catches
	// a refactor that loses a row or scrambles the per-MP filter).
	wantByID := map[string]writtenQuestionJSON{}
	for _, w := range want {
		wantByID[w.ID] = w
	}
	for _, got := range resp.Items {
		w, ok := wantByID[got.ID]
		if !ok {
			t.Errorf("response item with id=%q not in seed slice for %s", got.ID, personID)
			continue
		}
		if got.Ministry != w.Ministry {
			t.Errorf("id=%q: expected ministry=%q, got %q", got.ID, w.Ministry, got.Ministry)
		}
		if got.Minister != w.Minister {
			t.Errorf("id=%q: expected minister=%q, got %q", got.ID, w.Minister, got.Minister)
		}
		if got.Status != w.Status {
			t.Errorf("id=%q: expected status=%q, got %q", got.ID, w.Status, got.Status)
		}
	}

	// 2. Cross-check against the per-MP endpoint
	// /api/v1/people/{id}/questions. The two endpoints MUST return the
	// same item set — the list endpoint with an mp_id filter is a
	// superset view that happens to filter to one MP, while the per-MP
	// endpoint is the canonical MP-scoped view. A divergence between
	// the two is a contract regression.
	perMPReq := httptest.NewRequest(http.MethodGet, "/api/v1/people/"+personID+"/questions", nil)
	perMPRR := httptest.NewRecorder()
	handlePeople(perMPRR, perMPReq)

	if perMPRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/people/%s/questions, got %d (body=%s)", personID, perMPRR.Code, perMPRR.Body.String())
	}
	var perMPResp writtenQuestionsByPersonPayload
	if err := json.Unmarshal(perMPRR.Body.Bytes(), &perMPResp); err != nil {
		t.Fatalf("decode per-MP: %v", err)
	}
	if perMPResp.MPID != personID {
		t.Errorf("per-MP: expected mp_id=%q, got %q", personID, perMPResp.MPID)
	}
	// person-001 is Kimani Ichung'wah (sampleScorecards[0]).
	if perMPResp.Name != "Kimani Ichung'wah" {
		t.Errorf("per-MP: expected name=%q, got %q", "Kimani Ichung'wah", perMPResp.Name)
	}
	if perMPResp.Source != "seed" {
		t.Errorf("per-MP: expected source=seed, got %q", perMPResp.Source)
	}
	wantScorecardURL := "/api/v1/people/" + personID + "/scorecard"
	if perMPResp.ScorecardURL != wantScorecardURL {
		t.Errorf("per-MP: expected scorecard_url=%q, got %q", wantScorecardURL, perMPResp.ScorecardURL)
	}
	if perMPResp.Total != resp.Total {
		t.Errorf("per-MP endpoint returned total=%d but list endpoint returned total=%d — the two MUST agree", perMPResp.Total, resp.Total)
	}

	// 3. Filter with an unknown mp_id returns 200 + an empty items
	// list (NOT 404 — the list endpoint is a search-style filter, so
	// an unknown mp_id is a legitimate "no matches" result rather
	// than a "resource not found" error). This is consistent with
	// the /api/v1/bills endpoint's behaviour for filters that match
	// no rows.
	unknownReq := httptest.NewRequest(http.MethodGet, "/api/v1/questions/written?mp_id=does-not-exist", nil)
	unknownRR := httptest.NewRecorder()
	makeWrittenQuestionsListHandler()(unknownRR, unknownReq)
	if unknownRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for unknown mp_id filter, got %d (body=%s)", unknownRR.Code, unknownRR.Body.String())
	}
	var unknownResp writtenQuestionsListPayload
	if err := json.Unmarshal(unknownRR.Body.Bytes(), &unknownResp); err != nil {
		t.Fatalf("decode unknown mp_id: %v", err)
	}
	if unknownResp.Total != 0 || len(unknownResp.Items) != 0 {
		t.Errorf("expected empty items list for unknown mp_id, got total=%d items=%d", unknownResp.Total, len(unknownResp.Items))
	}

	// 4. /api/v1/people/{id}/questions for an unknown MP returns 404
	// (NOT 200 with an empty items list) — the per-MP endpoint is a
	// resource-style URL, so an unknown person_id is a 404, not a
	// "no matches" result. This guards the "every datum is sourced"
	// invariant (a typo in a person ID must 404, not silently look
	// like a legitimate "no questions" response).
	notFoundReq := httptest.NewRequest(http.MethodGet, "/api/v1/people/does-not-exist/questions", nil)
	notFoundRR := httptest.NewRecorder()
	handlePeople(notFoundRR, notFoundReq)
	if notFoundRR.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown person_id on per-MP endpoint, got %d (body=%s)", notFoundRR.Code, notFoundRR.Body.String())
	}
}

// TestWrittenQuestionsList_FiltersByStatus verifies the list endpoint
// filters by the status query param. Every returned item must carry
// the requested status, and the status echo field must be populated
// so the frontend can verify the filter was honoured. An invalid
// status value returns 400 (NOT 200 with an empty items list) so a
// typo in the status does not silently look like a legitimate "no
// matches" result.
//
// This test is named per the issue spec ("filterable by mp_id +
// status") — the Filter-by-Status coverage is required by the issue
// acceptance bar even though the explicit test list in the spec
// names only FiltersByMP. The two filter tests are complementary: a
// refactor that breaks the status filter would slip past
// TestWrittenQuestionsList_FiltersByMP.
func TestWrittenQuestionsList_FiltersByStatus(t *testing.T) {
	for _, status := range []string{"pending", "answered", "overdue"} {
		t.Run(status, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/written?status="+status, nil)
			rr := httptest.NewRecorder()
			makeWrittenQuestionsListHandler()(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200 for status=%s, got %d (body=%s)", status, rr.Code, rr.Body.String())
			}
			var resp writtenQuestionsListPayload
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode status=%s: %v", status, err)
			}
			if resp.Status != status {
				t.Errorf("expected status echo=%q, got %q", status, resp.Status)
			}
			if resp.Total != 5 {
				t.Errorf("expected 5 %s questions (seed balance), got %d", status, resp.Total)
			}
			for i, item := range resp.Items {
				if item.Status != status {
					t.Errorf("item[%d]: expected status=%q, got %q (every item in a status-filtered list MUST carry the requested status)", i, status, item.Status)
				}
			}
		})
	}

	// Invalid status → 400 (NOT 200 with an empty items list).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/written?status=garbage", nil)
	rr := httptest.NewRecorder()
	makeWrittenQuestionsListHandler()(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status filter, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestWrittenQuestionDetail_ReturnsQuestion verifies the
// /api/v1/questions/written/{id} detail endpoint returns the full
// WrittenQuestion JSON for a known id. Every required field must be
// present + every status-aware field (response_text + responded_at)
// must match the seed slice's expectation for that status.
func TestWrittenQuestionDetail_ReturnsQuestion(t *testing.T) {
	const id = "wq-001" // person-001 / Kimani Ichung'wah / answered
	want := findWrittenQuestionByID(id)
	if want == nil {
		t.Fatalf("expected seed slice to contain %s", id)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/written/"+id, nil)
	rr := httptest.NewRecorder()
	makeWrittenQuestionDetailHandler()(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	var resp writtenQuestionJSON
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != want.ID {
		t.Errorf("expected id=%q, got %q", want.ID, resp.ID)
	}
	if resp.MPID != want.MPID {
		t.Errorf("expected mp_id=%q, got %q", want.MPID, resp.MPID)
	}
	if resp.MPName != want.MPName {
		t.Errorf("expected mp_name=%q, got %q", want.MPName, resp.MPName)
	}
	if resp.Minister != want.Minister {
		t.Errorf("expected minister=%q, got %q", want.Minister, resp.Minister)
	}
	if resp.Ministry != want.Ministry {
		t.Errorf("expected ministry=%q, got %q", want.Ministry, resp.Ministry)
	}
	if resp.QuestionText != want.QuestionText {
		t.Errorf("expected question_text=%q, got %q", want.QuestionText, resp.QuestionText)
	}
	if resp.AskedAt != want.AskedAt {
		t.Errorf("expected asked_at=%q, got %q", want.AskedAt, resp.AskedAt)
	}
	if resp.Deadline != want.Deadline {
		t.Errorf("expected deadline=%q, got %q", want.Deadline, resp.Deadline)
	}
	if resp.Status != string(want.Status) {
		t.Errorf("expected status=%q, got %q", string(want.Status), resp.Status)
	}
	if resp.SourceURL != want.SourceURL {
		t.Errorf("expected source_url=%q, got %q", want.SourceURL, resp.SourceURL)
	}
	// wq-001 is an answered row → response_text + responded_at MUST
	// be populated.
	if want.Status == WrittenQuestionAnswered {
		if resp.ResponseText == "" {
			t.Errorf("answered row missing response_text")
		}
		if resp.RespondedAt == "" {
			t.Errorf("answered row missing responded_at")
		}
	}
}

// TestWrittenQuestionDetail_UnknownID_Returns404 verifies the detail
// endpoint returns 404 (NOT 200 with an empty body) for an unknown
// id. This guards the "every datum is sourced" invariant — a typo in
// the id must 404, not silently look like a legitimate "question not
// yet answered" response.
func TestWrittenQuestionDetail_UnknownID_Returns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/questions/written/wq-does-not-exist", nil)
	rr := httptest.NewRecorder()
	makeWrittenQuestionDetailHandler()(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown id, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	// The error envelope MUST be the canonical {error, message} shape
	// so the frontend can render it uniformly with every other 404
	// in the API.
	var errResp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if errResp["error"] != "not_found" {
		t.Errorf("expected error=not_found, got %q", errResp["error"])
	}
	if errResp["message"] == "" {
		t.Errorf("expected non-empty message, got %q", errResp["message"])
	}
}
