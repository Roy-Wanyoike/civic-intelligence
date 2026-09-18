package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
)

// TestHandleBillChanges_Found verifies the handler returns 200 OK with an
// empty `changes` array and the documented note when the Bill exists but
// only one version is in the database.
func TestHandleBillChanges_Found(t *testing.T) {
	const html = `<html><body>
<a href="/akn/ke/bill/na/2026-09-07/the-housing-bill-2026/eng@2026-09-07">The Housing Bill 2026</a>
</body></html>`

	client := &mockKenyaLawClient{billsListHTML: html}
	adapter := kenya_law.NewAdapter(client, "CivicIntelligence-test/0.1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/ke-bill-2026-09-07-the-housing-bill-2026/changes", nil)
	rr := httptest.NewRecorder()

	handleBillChanges(rr, req, adapter, "ke-bill-2026-09-07-the-housing-bill-2026")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	var resp struct {
		BillID  string               `json:"bill_id"`
		Changes []billChangeResponse `json:"changes"`
		Total   int                  `json:"total"`
		Note    string               `json:"note"`
		Source  string               `json:"source"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Source != "new.kenyalaw.org" {
		t.Errorf("expected source=new.kenyalaw.org, got %q", resp.Source)
	}
	if resp.Total != 0 {
		t.Errorf("expected 0 changes, got %d", resp.Total)
	}
	if len(resp.Changes) != 0 {
		t.Errorf("expected empty changes array, got %d entries", len(resp.Changes))
	}
	expectedNote := "Version comparison requires multiple Bill versions in the database"
	if resp.Note != expectedNote {
		t.Errorf("expected note %q, got %q", expectedNote, resp.Note)
	}
}

// TestHandleBillChanges_NotFound verifies the handler returns 404 when the
// Bill ID does not match any discovered Bill.
func TestHandleBillChanges_NotFound(t *testing.T) {
	const html = `<html><body>
<a href="/akn/ke/bill/na/2026-09-07/the-housing-bill-2026/eng@2026-09-07">The Housing Bill 2026</a>
</body></html>`

	client := &mockKenyaLawClient{billsListHTML: html}
	adapter := kenya_law.NewAdapter(client, "CivicIntelligence-test/0.1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/does-not-exist/changes", nil)
	rr := httptest.NewRecorder()

	handleBillChanges(rr, req, adapter, "does-not-exist")

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestHandleBillChanges_EmptyID verifies the handler returns 400 when the
// Bill ID is empty.
func TestHandleBillChanges_EmptyID(t *testing.T) {
	client := &mockKenyaLawClient{billsListHTML: "<html></html>"}
	adapter := kenya_law.NewAdapter(client, "CivicIntelligence-test/0.1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills//changes", nil)
	rr := httptest.NewRecorder()

	handleBillChanges(rr, req, adapter, "")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestHandleBillChanges_AdapterError verifies the handler returns 503 when
// the adapter cannot reach the source (e.g., HTTP 500).
func TestHandleBillChanges_AdapterError(t *testing.T) {
	client := &changesErrorClient{}
	adapter := kenya_law.NewAdapter(client, "CivicIntelligence-test/0.1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/whatever/changes", nil)
	rr := httptest.NewRecorder()

	handleBillChanges(rr, req, adapter, "whatever")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// changesErrorClient always returns HTTP 500.
type changesErrorClient struct{}

func (e *changesErrorClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("server error")),
		Header:     make(http.Header),
	}, nil
}

// TestBillChangeResponse_JSONShape verifies the JSON shape of the
// billChangeResponse struct — every field must serialize with the documented
// JSON key so the frontend can rely on a stable contract once versions
// become available.
func TestBillChangeResponse_JSONShape(t *testing.T) {
	c := billChangeResponse{
		Kind:        "modification",
		Section:     "Clause 14(2)",
		Description: "Updated penalty threshold.",
		Before:      "not exceeding KES 50,000",
		After:       "not exceeding KES 100,000",
		SourceURL:   "https://new.kenyalaw.org/akn/ke/bill/na/2026-09-07/test/eng@2026-09-07",
		Confidence:  "high",
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	required := []string{
		`"kind":"modification"`,
		`"section":"Clause 14(2)"`,
		`"description":"Updated penalty threshold."`,
		`"before":"not exceeding KES 50,000"`,
		`"after":"not exceeding KES 100,000"`,
		`"source_url":"https://new.kenyalaw.org/akn/ke/bill/na/2026-09-07/test/eng@2026-09-07"`,
		`"confidence":"high"`,
	}
	for _, key := range required {
		if !strings.Contains(s, key) {
			t.Errorf("expected JSON to contain %q, got %s", key, s)
		}
	}
}
