package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// correctionDispatch runs a request through an OptionalAuth chain populated
// with the given principal, mirroring how the production router wires the
// corrections endpoints.
func correctionDispatch(handler http.Handler, method, url string, body []byte, p auth.Principal) *httptest.ResponseRecorder {
	var br *bytes.Reader
	if body != nil {
		br = bytes.NewReader(body)
	} else {
		br = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, url, br)
	if p.IsAuthenticated() {
		req.Header.Set("Authorization", "Bearer test-token")
	}
	v := auth.StaticVerifier{Principal: p}
	wrapped := middleware.OptionalAuth(v)(handler)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)
	return rr
}

// adminPrincipal is a principal carrying the user:admin scope (used to assert
// admin-only access to GET /api/v1/corrections).
func adminPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "admin-1",
		Scopes: auth.Scopes{auth.ScopeUserAdmin},
	}
}

// editorPrincipal is a principal carrying the evidence:write scope (also
// allowed to list corrections).
func editorPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "editor-1",
		Scopes: auth.Scopes{auth.ScopeEvidenceWrite},
	}
}

// citizenPrincipal is an authenticated non-admin user.
func citizenPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "citizen-1",
		Scopes: auth.Scopes{auth.ScopeBillRead},
	}
}

// validSubmission returns a correction request body with all required fields
// filled in correctly.
func validSubmission() correctionRequest {
	return correctionRequest{
		TargetType:     "bill",
		TargetID:       "00000000-0000-0000-0000-000000000001",
		Category:       CorrectionWrongFact,
		Reason:         "The bill title is incorrect — should be 'Housing Bill, 2024' not 'Housing Bill 2024'.",
		PageURL:        "https://civic-intelligence.vercel.app/bills/00000000-0000-0000-0000-000000000001",
		Evidence:       "https://www.parliament.go.ke/bills/housing-2024",
		SubmittedEmail: "citizen@example.com",
	}
}

// TestCorrectionStore_Submit_Anonymous verifies an anonymous submission
// succeeds when submitted_email is provided.
func TestCorrectionStore_Submit_Anonymous(t *testing.T) {
	store := NewCorrectionStore()
	req := validSubmission()
	rec, err := store.Submit(req, "")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if rec.ID == "" || !strings.HasPrefix(rec.ID, "crt_") {
		t.Errorf("expected crt_ prefixed ID, got %q", rec.ID)
	}
	if rec.Status != "pending" {
		t.Errorf("expected status=pending, got %s", rec.Status)
	}
	if rec.SubmittedEmail != "citizen@example.com" {
		t.Errorf("expected email to be stored, got %q", rec.SubmittedEmail)
	}
	if rec.SubmittedBy != "" {
		t.Errorf("expected empty submitted_by for anonymous submit, got %q", rec.SubmittedBy)
	}
}

// TestCorrectionStore_Submit_Authenticated verifies an authenticated
// submission sets submitted_by and does not require email.
func TestCorrectionStore_Submit_Authenticated(t *testing.T) {
	store := NewCorrectionStore()
	req := validSubmission()
	req.SubmittedEmail = "" // authenticated users don't need to provide email
	rec, err := store.Submit(req, "user-1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if rec.SubmittedBy != "user-1" {
		t.Errorf("expected submitted_by=user-1, got %q", rec.SubmittedBy)
	}
}

// TestCorrectionStore_Submit_Validation covers all validation rules.
func TestCorrectionStore_Submit_Validation(t *testing.T) {
	store := NewCorrectionStore()

	cases := []struct {
		name    string
		mutate  func(r correctionRequest) correctionRequest
		wantErr string
	}{
		{
			name:    "invalid target_type",
			mutate:  func(r correctionRequest) correctionRequest { r.TargetType = "spaceship"; return r },
			wantErr: "invalid target_type",
		},
		{
			name:    "missing target_id",
			mutate:  func(r correctionRequest) correctionRequest { r.TargetID = ""; return r },
			wantErr: "target_id required",
		},
		{
			name:    "invalid category",
			mutate:  func(r correctionRequest) correctionRequest { r.Category = "wrong"; return r },
			wantErr: "invalid category",
		},
		{
			name:    "missing reason",
			mutate:  func(r correctionRequest) correctionRequest { r.Reason = ""; return r },
			wantErr: "reason required",
		},
		{
			name:    "missing page_url",
			mutate:  func(r correctionRequest) correctionRequest { r.PageURL = ""; return r },
			wantErr: "page_url required",
		},
		{
			name:    "page_url not http(s)",
			mutate:  func(r correctionRequest) correctionRequest { r.PageURL = "ftp://example.com"; return r },
			wantErr: "page_url must be a valid http(s) URL",
		},
		{
			name:    "evidence not http(s)",
			mutate:  func(r correctionRequest) correctionRequest { r.Evidence = "not-a-url"; return r },
			wantErr: "evidence must be a valid http(s) URL",
		},
		{
			name:    "anonymous without email",
			mutate:  func(r correctionRequest) correctionRequest { r.SubmittedEmail = ""; return r },
			wantErr: "submitted_email required for anonymous submissions",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := c.mutate(validSubmission())
			_, err := store.Submit(req, "")
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("expected error %q, got %v", c.wantErr, err)
			}
		})
	}
}

// TestCorrectionStore_List_SortedDesc verifies List returns newest-first.
func TestCorrectionStore_List_SortedDesc(t *testing.T) {
	store := NewCorrectionStore()
	r1, _ := store.Submit(validSubmission(), "")
	r2, _ := store.Submit(validSubmission(), "")
	r3, _ := store.Submit(validSubmission(), "")

	got := store.List("")
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	if got[0].ID != r3.ID || got[1].ID != r2.ID || got[2].ID != r1.ID {
		t.Errorf("expected newest-first (r3,r2,r1), got %s,%s,%s",
			got[0].ID, got[1].ID, got[2].ID)
	}
}

// TestCorrectionStore_List_FilterByStatus verifies the status filter.
func TestCorrectionStore_List_FilterByStatus(t *testing.T) {
	store := NewCorrectionStore()
	r1, _ := store.Submit(validSubmission(), "")
	// Transition r1 to accepted.
	accepted, err := store.Transition(r1.ID, "accepted", "admin-1")
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if accepted.Status != "accepted" {
		t.Errorf("expected status=accepted, got %s", accepted.Status)
	}
	if accepted.ReviewedBy != "admin-1" {
		t.Errorf("expected reviewed_by=admin-1, got %s", accepted.ReviewedBy)
	}
	if accepted.ReviewedAt == nil {
		t.Error("expected reviewed_at to be set")
	}

	_, _ = store.Submit(validSubmission(), "")

	pending := store.List("pending")
	acceptedList := store.List("accepted")
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}
	if len(acceptedList) != 1 {
		t.Errorf("expected 1 accepted, got %d", len(acceptedList))
	}
	if acceptedList[0].ID != r1.ID {
		t.Errorf("filter returned wrong record: %s", acceptedList[0].ID)
	}
}

// TestHandleCorrectionSubmit_AnonymousSuccess verifies POST /api/v1/corrections
// returns 201 for a valid anonymous submission.
func TestHandleCorrectionSubmit_AnonymousSuccess(t *testing.T) {
	store := NewCorrectionStore()
	handler := makeCorrectionsHandler(store)

	body, _ := json.Marshal(validSubmission())
	rr := correctionDispatch(handler, http.MethodPost, "/api/v1/corrections", body, auth.Anonymous())
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var rec CorrectionRecord
	if err := json.Unmarshal(rr.Body.Bytes(), &rec); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if rec.Status != "pending" {
		t.Errorf("expected status=pending, got %s", rec.Status)
	}
	if rec.SubmittedEmail != "citizen@example.com" {
		t.Errorf("expected email to be stored, got %q", rec.SubmittedEmail)
	}
}

// TestHandleCorrectionSubmit_AuthenticatedSuccess verifies POST works for
// authenticated users and sets submitted_by.
func TestHandleCorrectionSubmit_AuthenticatedSuccess(t *testing.T) {
	store := NewCorrectionStore()
	handler := makeCorrectionsHandler(store)

	req := validSubmission()
	req.SubmittedEmail = "" // authenticated users don't need email
	body, _ := json.Marshal(req)
	rr := correctionDispatch(handler, http.MethodPost, "/api/v1/corrections", body, citizenPrincipal())
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var rec CorrectionRecord
	_ = json.Unmarshal(rr.Body.Bytes(), &rec)
	if rec.SubmittedBy != "citizen-1" {
		t.Errorf("expected submitted_by=citizen-1, got %q", rec.SubmittedBy)
	}
}

// TestHandleCorrectionSubmit_InvalidJSON verifies a malformed body 400s.
func TestHandleCorrectionSubmit_InvalidJSON(t *testing.T) {
	store := NewCorrectionStore()
	handler := makeCorrectionsHandler(store)
	rr := correctionDispatch(handler, http.MethodPost, "/api/v1/corrections", []byte("not json"), auth.Anonymous())
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// TestHandleCorrectionSubmit_InvalidCategory verifies an unknown category 400s.
func TestHandleCorrectionSubmit_InvalidCategory(t *testing.T) {
	store := NewCorrectionStore()
	handler := makeCorrectionsHandler(store)

	req := validSubmission()
	req.Category = "wrong"
	body, _ := json.Marshal(req)
	rr := correctionDispatch(handler, http.MethodPost, "/api/v1/corrections", body, auth.Anonymous())
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid category, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid category") {
		t.Errorf("expected error to mention category, got %s", rr.Body.String())
	}
}

// TestHandleCorrectionsList_AdminAllowed verifies that a user:admin scope
// can list corrections.
func TestHandleCorrectionsList_AdminAllowed(t *testing.T) {
	store := NewCorrectionStore()
	_, _ = store.Submit(validSubmission(), "")
	handler := makeCorrectionsHandler(store)

	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections", nil, adminPrincipal())
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items []CorrectionRecord `json:"items"`
		Total int               `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("expected total=1, got %d", resp.Total)
	}
}

// TestHandleCorrectionsList_EditorAllowed verifies that the evidence:write
// scope can also list corrections (editors review corrections as part of
// the trust pipeline).
func TestHandleCorrectionsList_EditorAllowed(t *testing.T) {
	store := NewCorrectionStore()
	_, _ = store.Submit(validSubmission(), "")
	handler := makeCorrectionsHandler(store)

	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections", nil, editorPrincipal())
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for editor, got %d", rr.Code)
	}
}

// TestHandleCorrectionsList_AnonymousRejected verifies anonymous callers
// get 401 on the list endpoint.
func TestHandleCorrectionsList_AnonymousRejected(t *testing.T) {
	store := NewCorrectionStore()
	handler := makeCorrectionsHandler(store)
	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections", nil, auth.Anonymous())
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for anonymous list, got %d", rr.Code)
	}
}

// TestHandleCorrectionsList_CitizenForbidden verifies a non-admin authenticated
// caller gets 403 on the list endpoint.
func TestHandleCorrectionsList_CitizenForbidden(t *testing.T) {
	store := NewCorrectionStore()
	handler := makeCorrectionsHandler(store)
	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections", nil, citizenPrincipal())
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin authenticated list, got %d", rr.Code)
	}
}

// TestHandleCorrectionsList_FilterByStatus verifies the ?status= filter.
func TestHandleCorrectionsList_FilterByStatus(t *testing.T) {
	store := NewCorrectionStore()
	r1, _ := store.Submit(validSubmission(), "")
	_, _ = store.Transition(r1.ID, "accepted", "admin-1")
	_, _ = store.Submit(validSubmission(), "")

	handler := makeCorrectionsHandler(store)
	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections?status=accepted", nil, adminPrincipal())
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Items []CorrectionRecord `json:"items"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 accepted, got %d", len(resp.Items))
	}
}

// TestHandleCorrectionsList_InvalidStatus verifies an unknown ?status= 400s.
func TestHandleCorrectionsList_InvalidStatus(t *testing.T) {
	store := NewCorrectionStore()
	handler := makeCorrectionsHandler(store)
	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections?status=banana", nil, adminPrincipal())
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// TestHandleCorrectionsList_MethodNotAllowed verifies PUT 405s with Allow header.
func TestHandleCorrectionsList_MethodNotAllowed(t *testing.T) {
	store := NewCorrectionStore()
	handler := makeCorrectionsHandler(store)
	rr := correctionDispatch(handler, http.MethodPut, "/api/v1/corrections", []byte("{}"), adminPrincipal())
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
	if rr.Header().Get("Allow") != "GET, POST" {
		t.Errorf("expected Allow=GET, POST, got %q", rr.Header().Get("Allow"))
	}
}

// TestHandleCorrectionDetail_AdminSuccess verifies GET /api/v1/corrections/{id}
// returns the full record for an admin caller.
func TestHandleCorrectionDetail_AdminSuccess(t *testing.T) {
	store := NewCorrectionStore()
	rec, _ := store.Submit(validSubmission(), "")
	handler := makeCorrectionDetailHandler(store)

	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections/"+rec.ID, nil, adminPrincipal())
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var got CorrectionRecord
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got.ID != rec.ID {
		t.Errorf("expected id=%s, got %s", rec.ID, got.ID)
	}
}

// TestHandleCorrectionDetail_AnonymousRejected verifies anonymous detail 401s.
func TestHandleCorrectionDetail_AnonymousRejected(t *testing.T) {
	store := NewCorrectionStore()
	rec, _ := store.Submit(validSubmission(), "")
	handler := makeCorrectionDetailHandler(store)

	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections/"+rec.ID, nil, auth.Anonymous())
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// TestHandleCorrectionDetail_CitizenForbidden verifies a non-admin 403s.
func TestHandleCorrectionDetail_CitizenForbidden(t *testing.T) {
	store := NewCorrectionStore()
	rec, _ := store.Submit(validSubmission(), "")
	handler := makeCorrectionDetailHandler(store)

	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections/"+rec.ID, nil, citizenPrincipal())
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for citizen detail, got %d", rr.Code)
	}
}

// TestHandleCorrectionDetail_NotFound verifies 404 on unknown ID.
func TestHandleCorrectionDetail_NotFound(t *testing.T) {
	store := NewCorrectionStore()
	handler := makeCorrectionDetailHandler(store)
	rr := correctionDispatch(handler, http.MethodGet, "/api/v1/corrections/crt_does_not_exist", nil, adminPrincipal())
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// TestCorrectionCategories_CoverAllSpecValues verifies the 8 documented
// categories from issue #166 are all valid.
func TestCorrectionCategories_CoverAllSpecValues(t *testing.T) {
	want := []string{
		"wrong_fact", "wrong_source", "wrong_citation", "outdated",
		"incorrect_interpretation", "missing_information",
		"conflicting_sources", "broken_document",
	}
	for _, c := range want {
		if !validCorrectionCategories[c] {
			t.Errorf("category %q is not in the allow-list", c)
		}
	}
}

// TestIsValidURL verifies the URL validator accepts http(s) and rejects others.
func TestIsValidURL(t *testing.T) {
	yes := []string{
		"https://example.com",
		"http://example.com/path?q=1",
		"https://parliament.go.ke/bills/x",
	}
	for _, s := range yes {
		if !isValidURL(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	no := []string{"", "not-a-url", "ftp://example.com", "javascript:alert(1)", "//example.com"}
	for _, s := range no {
		if isValidURL(s) {
			t.Errorf("expected %q to be INVALID", s)
		}
	}
}

// TestNewCorrectionID_UniqueAndPrefixed verifies ID generation.
func TestNewCorrectionID_UniqueAndPrefixed(t *testing.T) {
	seen := make(map[string]bool, 50)
	for i := 0; i < 50; i++ {
		id := newCorrectionID()
		if !strings.HasPrefix(id, "crt_") {
			t.Fatalf("id %s missing crt_ prefix", id)
		}
		if seen[id] {
			t.Fatalf("collision at iteration %d: %s", i, id)
		}
		seen[id] = true
	}
}
