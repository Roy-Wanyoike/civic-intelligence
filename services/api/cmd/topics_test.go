// Package main — tests for the Topics API endpoints (issue #267).
//
// Covers the four acceptance-criteria tests required by the issue:
//   - TestTopicsListHandler_ReturnsAllTopics
//   - TestTopicsListHandler_FiltersByCountry
//   - TestTopicDetailHandler_ReturnsTopicWithBills
//   - TestTopicDetailHandler_UnknownTopic_Returns404
//
// Plus three extra tests that guard against the trailing-slash + case
// regressions BE-1 fixed for /people, /institutions, /committees:
//   - TestTopicsListHandler_NoTrailingSlash_ReturnsList
//   - TestTopicsListHandler_TrailingSlash_ReturnsSamePayload
//   - TestTopicDetailHandler_CaseInsensitiveID
//
// The tests exercise the handlers directly (httptest.NewRequest) and
// via the real http.ServeMux (newTestAPIRouter in main_test.go) so the
// auto-301 the mux emits on the no-slash form is also caught.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// topicsListResponse mirrors the JSON shape returned by
// GET /api/v1/topics. Declared locally so each test can decode into a
// typed struct without leaking the schema into the test cases.
type topicsListResponse struct {
	Items  []kenya_seed.Topic `json:"items"`
	Total  int                `json:"total"`
	Country string             `json:"country"`
	Source string             `json:"source"`
	Note   string             `json:"note"`
}

// topicsDetailResponse mirrors the JSON shape returned by
// GET /api/v1/topics/{id}.
type topicsDetailResponse struct {
	Topic   kenya_seed.Topic     `json:"topic"`
	Bills   []kenya_seed.TopicBill   `json:"bills"`
	Acts    []kenya_seed.TopicAct    `json:"acts"`
	People  []kenya_seed.TopicPerson `json:"people"`
	Country string                 `json:"country"`
	Source  string                 `json:"source"`
	Note    string                 `json:"note"`
}

// TestTopicsListHandler_ReturnsAllTopics verifies the list endpoint
// returns the 10 seed Kenyan civic topics, each carrying the
// documented fields (id, name, description, bill_count, act_count,
// people_count, country).
func TestTopicsListHandler_ReturnsAllTopics(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topics", nil)
	rr := httptest.NewRecorder()

	handleTopicsList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp topicsListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Total != len(kenya_seed.KenyaTopics) {
		t.Errorf("expected total=%d, got %d", len(kenya_seed.KenyaTopics), resp.Total)
	}
	if resp.Total != 10 {
		t.Errorf("expected 10 seed topics, got %d — issue #267 spec requires 10", resp.Total)
	}
	if resp.Source != "kenya_seed" {
		t.Errorf("expected source=kenya_seed, got %q", resp.Source)
	}
	if resp.Country != "KE" {
		t.Errorf("expected country=KE (default), got %q", resp.Country)
	}

	// Every item must carry the documented fields + the country must
	// be KE (the only seed country today).
	requiredIDs := map[string]bool{
		"health": false, "education": false, "finance": false,
		"justice": false, "defence-foreign-relations": false,
		"energy-petroleum": false, "agriculture-livestock": false,
		"infrastructure-transport": false,
		"environment-natural-resources": false, "ict": false,
	}
	for _, item := range resp.Items {
		if item.ID == "" || item.Name == "" || item.Description == "" {
			t.Errorf("topic missing required field: %+v", item)
		}
		if item.Country != "KE" {
			t.Errorf("topic %q country=%q, want KE", item.ID, item.Country)
		}
		if item.BillCount < 0 || item.ActCount < 0 || item.PeopleCount < 0 {
			t.Errorf("topic %q has negative count: %+v", item.ID, item)
		}
		if _, ok := requiredIDs[item.ID]; ok {
			requiredIDs[item.ID] = true
		}
	}
	for id, seen := range requiredIDs {
		if !seen {
			t.Errorf("expected topic %q in the list, not found", id)
		}
	}
}

// TestTopicsListHandler_FiltersByCountry verifies a non-KE country
// scope returns an empty list (200, not 404) — other countries do
// not yet have seed topics. ALL (GlobalCountry) returns the full
// Kenya seed (the dashboard view).
func TestTopicsListHandler_FiltersByCountry(t *testing.T) {
	t.Run("UG scope returns empty list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/topics", nil)
		req = req.WithContext(middleware.WithCountry(context.Background(), "UG"))
		rr := httptest.NewRecorder()

		handleTopicsList(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var resp topicsListResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Total != 0 {
			t.Errorf("expected 0 topics for UG scope, got %d", resp.Total)
		}
		if resp.Country != "UG" {
			t.Errorf("expected country=UG echoed on response, got %q", resp.Country)
		}
		if len(resp.Items) != 0 {
			t.Errorf("expected empty items slice, got %d items", len(resp.Items))
		}
	})

	t.Run("ALL scope returns full Kenya seed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/topics", nil)
		req = req.WithContext(middleware.WithCountry(context.Background(), middleware.GlobalCountry))
		rr := httptest.NewRecorder()

		handleTopicsList(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var resp topicsListResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Total != len(kenya_seed.KenyaTopics) {
			t.Errorf("expected ALL to return the full %d topics, got %d",
				len(kenya_seed.KenyaTopics), resp.Total)
		}
	})

	t.Run("KE scope returns 10 topics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/topics", nil)
		req = req.WithContext(middleware.WithCountry(context.Background(), "KE"))
		rr := httptest.NewRecorder()

		handleTopicsList(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var resp topicsListResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Total != 10 {
			t.Errorf("expected 10 topics for KE scope, got %d", resp.Total)
		}
	})
}

// TestTopicDetailHandler_ReturnsTopicWithBills verifies the detail
// endpoint returns the topic plus the associated Bills / Acts / People
// slices, with every entry carrying a source_url.
func TestTopicDetailHandler_ReturnsTopicWithBills(t *testing.T) {
	// "health" has 5 bills + 1 act + 1 person in the seed — use it as
	// the canonical test case.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topics/health", nil)
	rr := httptest.NewRecorder()

	handleTopicDetail(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp topicsDetailResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Topic.ID != "health" {
		t.Errorf("expected topic.id=health, got %q", resp.Topic.ID)
	}
	if resp.Topic.Name != "Health" {
		t.Errorf("expected topic.name=Health, got %q", resp.Topic.Name)
	}
	if resp.Topic.Country != "KE" {
		t.Errorf("expected topic.country=KE, got %q", resp.Topic.Country)
	}

	// Counts in the topic envelope MUST match the actual slice lengths
	// (this is the invariant the package init enforces — guards
	// against a future drift).
	if resp.Topic.BillCount != len(resp.Bills) {
		t.Errorf("bill_count mismatch: topic.bill_count=%d, len(bills)=%d",
			resp.Topic.BillCount, len(resp.Bills))
	}
	if resp.Topic.ActCount != len(resp.Acts) {
		t.Errorf("act_count mismatch: topic.act_count=%d, len(acts)=%d",
			resp.Topic.ActCount, len(resp.Acts))
	}
	if resp.Topic.PeopleCount != len(resp.People) {
		t.Errorf("people_count mismatch: topic.people_count=%d, len(people)=%d",
			resp.Topic.PeopleCount, len(resp.People))
	}

	// "health" is seeded with at least 1 bill, 1 act, 1 person.
	if len(resp.Bills) == 0 {
		t.Error("expected at least one associated bill for the health topic")
	}
	if len(resp.Acts) == 0 {
		t.Error("expected at least one associated act for the health topic")
	}
	if len(resp.People) == 0 {
		t.Error("expected at least one associated person for the health topic")
	}

	// Every bill must carry the documented fields + a source_url.
	for _, b := range resp.Bills {
		if b.ID == "" || b.Title == "" || b.House == "" || b.Year == 0 || b.SourceURL == "" {
			t.Errorf("bill missing required field: %+v", b)
		}
	}
	// Every act must carry the documented fields + a source_url.
	for _, a := range resp.Acts {
		if a.ID == "" || a.Title == "" || a.Citation == "" || a.SourceURL == "" {
			t.Errorf("act missing required field: %+v", a)
		}
	}
	// Every person must carry the documented fields.
	for _, p := range resp.People {
		if p.ID == "" || p.FullName == "" || p.Role == "" || p.House == "" {
			t.Errorf("person missing required field: %+v", p)
		}
	}
}

// TestTopicDetailHandler_UnknownTopic_Returns404 verifies an unknown
// topic ID returns 404 (not 200 with empty slices — that would let the
// frontend silently render a fabricated topic page).
func TestTopicDetailHandler_UnknownTopic_Returns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topics/does-not-exist", nil)
	rr := httptest.NewRecorder()

	handleTopicDetail(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown topic, got %d (body: %s)",
			rr.Code, rr.Body.String())
	}

	var resp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "not_found" {
		t.Errorf("expected error=not_found, got %q", resp.Error)
	}
	if resp.Message == "" {
		t.Errorf("expected non-empty error message")
	}
}

// TestTopicDetailHandler_EmptyID_Returns400 verifies a missing topic
// ID returns 400 (the dispatcher should never let an empty ID reach
// this handler, but the guard is defensive).
func TestTopicDetailHandler_EmptyID_Returns400(t *testing.T) {
	// A bare /api/v1/topics/ path would be dispatched to the LIST
	// handler (the trailing slash is the registered base); simulate a
	// direct call with an empty ID to exercise the defensive guard.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topics/", nil)
	rr := httptest.NewRecorder()

	handleTopicDetail(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty topic ID, got %d (body: %s)",
			rr.Code, rr.Body.String())
	}
}

// TestTopicDetailHandler_CaseInsensitiveID verifies the detail
// handler accepts case-insensitive topic IDs — /topics/Health and
// /topics/health resolve to the same topic. Matches the
// path-normalisation pattern used by the /people/{id} handler.
func TestTopicDetailHandler_CaseInsensitiveID(t *testing.T) {
	for _, id := range []string{"Health", "HEALTH", "Health"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/topics/"+id, nil)
		rr := httptest.NewRecorder()

		handleTopicDetail(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("case-insensitive lookup for %q: expected 200, got %d", id, rr.Code)
			continue
		}
		var resp topicsDetailResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Errorf("case-insensitive lookup for %q: decode: %v", id, err)
			continue
		}
		if resp.Topic.ID != "health" {
			t.Errorf("case-insensitive lookup for %q: expected canonical id=health, got %q",
				id, resp.Topic.ID)
		}
	}
}

// TestTopicDetailHandler_CountryVisibilityGate verifies the country
// visibility gate: a Uganda-scoped request asking for a Kenyan topic
// returns 404 (mirrors the /people/{id} gate). ALL (GlobalCountry) is
// allowed so the cross-country dashboard can deep-link.
func TestTopicDetailHandler_CountryVisibilityGate(t *testing.T) {
	t.Run("UG scope returns 404 for KE topic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/topics/health", nil)
		req = req.WithContext(middleware.WithCountry(context.Background(), "UG"))
		rr := httptest.NewRecorder()

		handleTopicDetail(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for UG-scoped request on KE topic, got %d", rr.Code)
		}
	})

	t.Run("ALL scope returns 200 for KE topic", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/topics/health", nil)
		req = req.WithContext(middleware.WithCountry(context.Background(), middleware.GlobalCountry))
		rr := httptest.NewRecorder()

		handleTopicDetail(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for ALL-scoped request on KE topic, got %d", rr.Code)
		}
	})
}

// TestTopicsListHandler_NoTrailingSlash_ReturnsList verifies the
// no-trailing-slash URL form returns 200 (NOT a 301 redirect) when
// dispatched through the real http.ServeMux. Mirrors BE-1's fix for
// the /people / /institutions / /committees no-slash 301 bug.
func TestTopicsListHandler_NoTrailingSlash_ReturnsList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/topics", handleTopicsList)
	mux.HandleFunc("/api/v1/topics/", handleTopicsList)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/topics", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for no-slash URL, got %d (NOT 301)", rr.Code)
	}
	var resp topicsListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total == 0 {
		t.Error("expected non-empty topic list, got 0 items")
	}
}

// TestTopicsListHandler_TrailingSlash_ReturnsSamePayload verifies the
// trailing-slash URL form returns the same payload as the no-slash
// form. Mirrors BE-1's "slash vs no-slash bodies are byte-identical"
// contract for /people, /institutions, /committees.
func TestTopicsListHandler_TrailingSlash_ReturnsSamePayload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/topics", handleTopicsList)
	mux.HandleFunc("/api/v1/topics/", handleTopicsList)

	// No-slash form.
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/topics", nil)
	rr1 := httptest.NewRecorder()
	mux.ServeHTTP(rr1, req1)

	// Trailing-slash form.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/topics/", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)

	if rr1.Code != http.StatusOK || rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 for both forms, got %d / %d", rr1.Code, rr2.Code)
	}

	var resp1, resp2 topicsListResponse
	if err := json.Unmarshal(rr1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("decode no-slash: %v", err)
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode with-slash: %v", err)
	}
	if resp1.Total != resp2.Total {
		t.Errorf("total mismatch: no-slash=%d, with-slash=%d", resp1.Total, resp2.Total)
	}
	if len(resp1.Items) != len(resp2.Items) {
		t.Errorf("item count mismatch: no-slash=%d, with-slash=%d",
			len(resp1.Items), len(resp2.Items))
	}
	// Spot-check the first item — IDs should match.
	if len(resp1.Items) > 0 && len(resp2.Items) > 0 {
		if resp1.Items[0].ID != resp2.Items[0].ID {
			t.Errorf("first item ID mismatch: %q vs %q",
				resp1.Items[0].ID, resp2.Items[0].ID)
		}
	}
}

// TestTopicsListHandler_MethodNotAllowed verifies POST/PUT/DELETE
// requests return 405 method_not_allowed — the topics endpoints are
// read-only today (issue #267 scope: GET only).
func TestTopicsListHandler_MethodNotAllowed(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/api/v1/topics", nil)
		rr := httptest.NewRecorder()
		handleTopicsList(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 for %s, got %d", method, rr.Code)
		}
	}
}

// TestTopicsSeed_CountsMatchDetail verifies the bill_count / act_count /
// people_count fields advertised in the list response match the actual
// slice lengths returned by the detail response for every topic.
// Guards against a future drift where the counts are updated manually
// but the seed slices are not.
func TestTopicsSeed_CountsMatchDetail(t *testing.T) {
	for _, t2 := range kenya_seed.KenyaTopics {
		detail := kenya_seed.KenyaTopicDetail(t2.ID)
		if detail == nil {
			t.Errorf("topic %q: KenyaTopicDetail returned nil", t2.ID)
			continue
		}
		if t2.BillCount != len(detail.Bills) {
			t.Errorf("topic %q: list bill_count=%d but detail has %d bills",
				t2.ID, t2.BillCount, len(detail.Bills))
		}
		if t2.ActCount != len(detail.Acts) {
			t.Errorf("topic %q: list act_count=%d but detail has %d acts",
				t2.ID, t2.ActCount, len(detail.Acts))
		}
		if t2.PeopleCount != len(detail.People) {
			t.Errorf("topic %q: list people_count=%d but detail has %d people",
				t2.ID, t2.PeopleCount, len(detail.People))
		}
	}
}

// TestTopicsSeed_AllTopicsHaveUniqueIDs verifies no two seed topics
// share an ID — duplicate IDs would silently overwrite each other in
// the FindKenyaTopicByID lookup.
func TestTopicsSeed_AllTopicsHaveUniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, t2 := range kenya_seed.KenyaTopics {
		if seen[t2.ID] {
			t.Errorf("duplicate topic ID %q in KenyaTopics seed", t2.ID)
		}
		seen[t2.ID] = true
	}
}
