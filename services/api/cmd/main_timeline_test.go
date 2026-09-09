package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
)

// TestBuildTimelineForBill_WithPublicationDate verifies that when a Bill has
// a publication date, the timeline contains exactly one "publication" event
// sourced from the Akoma Ntoso URL on kenyalaw.org.
func TestBuildTimelineForBill_WithPublicationDate(t *testing.T) {
	pubDate, _ := time.Parse("2006-01-02", "2026-09-07")
	bill := &kenya_law.BillCandidate{
		URL:             "https://new.kenyalaw.org/akn/ke/bill/na/2026-09-07/the-housing-bill-2026/eng@2026-09-07",
		Slug:            "the-housing-bill-2026",
		Title:           "The Housing Bill 2026",
		House:           "National Assembly",
		PublicationDate: pubDate,
		SourceID:        "ke-bill-2026-09-07-the-housing-bill-2026",
	}

	events := buildTimelineForBill(bill)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.EventType != "publication" {
		t.Errorf("expected event_type=publication, got %q", ev.EventType)
	}
	if ev.House != "National Assembly" {
		t.Errorf("expected house=National Assembly, got %q", ev.House)
	}
	if ev.Confidence != "high" {
		t.Errorf("expected confidence=high, got %q", ev.Confidence)
	}
	if ev.SourceURL != bill.URL {
		t.Errorf("expected source_url=%s, got %s", bill.URL, ev.SourceURL)
	}
	if ev.BillID != bill.SourceID {
		t.Errorf("expected bill_id=%s, got %s", bill.SourceID, ev.BillID)
	}
	if ev.DateIsApproximate {
		t.Errorf("expected date_is_approximate=false, got true")
	}
	if ev.Note == nil || *ev.Note == "" {
		t.Errorf("expected a non-empty note explaining the inference")
	}
	// Date should be RFC3339-formatted.
	if _, err := time.Parse(time.RFC3339, ev.Date); err != nil {
		t.Errorf("expected RFC3339 date, got %q: %v", ev.Date, err)
	}
}

// TestBuildTimelineForBill_NilBill ensures that calling with nil returns an
// empty (not nil) slice — preventing null dereferences downstream.
func TestBuildTimelineForBill_NilBill(t *testing.T) {
	events := buildTimelineForBill(nil)
	if events == nil {
		t.Fatal("expected non-nil slice for nil bill")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for nil bill, got %d", len(events))
	}
}

// TestBuildTimelineForBill_ZeroPublicationDate verifies the function never
// invents events when the publication date is missing.
func TestBuildTimelineForBill_ZeroPublicationDate(t *testing.T) {
	bill := &kenya_law.BillCandidate{
		URL:      "https://new.kenyalaw.org/akn/ke/bill/na/2026-09-07/the-housing-bill-2026/eng@2026-09-07",
		Title:    "The Housing Bill 2026",
		House:    "National Assembly",
		SourceID: "ke-bill-2026-09-07-the-housing-bill-2026",
	}
	events := buildTimelineForBill(bill)
	if len(events) != 0 {
		t.Errorf("expected 0 events for bill without publication date, got %d", len(events))
	}
}

// --- HTTP handler tests using a mock HTTPClient ---

// mockKenyaLawClient is a stub HTTPClient that returns canned responses
// keyed on URL path. It lets us exercise the full adapter pipeline without
// hitting the live new.kenyalaw.org site.
type mockKenyaLawClient struct {
	billsListHTML  string
	billDetailHTML string
}

func (m *mockKenyaLawClient) Do(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Path, "/bills") && !strings.Contains(req.URL.Path, "/akn/") {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(m.billsListHTML)),
			Header:     make(http.Header),
		}, nil
	}
	if strings.Contains(req.URL.Path, "/akn/ke/bill/") {
		body := m.billDetailHTML
		if body == "" {
			body = "<html><body>bill detail</body></html>"
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("not found")),
		Header:     make(http.Header),
	}, nil
}

// newTestAdapter returns a kenya_law.Adapter wired to a mock client that
// returns the provided bills listing HTML.
func newTestAdapter(billsListHTML string) *kenya_law.Adapter {
	client := &mockKenyaLawClient{billsListHTML: billsListHTML}
	return kenya_law.NewAdapter(client, "CivicIntelligence-test/0.1")
}

// TestHandleBillTimeline_Found verifies the end-to-end handler returns a
// 200 OK with one publication event sourced from kenyalaw.org.
func TestHandleBillTimeline_Found(t *testing.T) {
	// Minimal bills list HTML with a single Bill.
	const html = `<html><body>
<a href="/akn/ke/bill/na/2026-09-07/the-housing-bill-2026/eng@2026-09-07">The Housing Bill 2026</a>
</body></html>`

	adapter := newTestAdapter(html)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/ke-bill-2026-09-07-the-housing-bill-2026/timeline", nil)
	rr := httptest.NewRecorder()

	handleBillTimeline(rr, req, adapter, "ke-bill-2026-09-07-the-housing-bill-2026")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	var resp struct {
		BillID string              `json:"bill_id"`
		Events []billEventResponse `json:"events"`
		Total  int                 `json:"total"`
		Source string              `json:"source"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Source != "new.kenyalaw.org" {
		t.Errorf("expected source=new.kenyalaw.org, got %q", resp.Source)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 event, got %d", resp.Total)
	}
	ev := resp.Events[0]
	if ev.EventType != "publication" {
		t.Errorf("expected event_type=publication, got %q", ev.EventType)
	}
	if ev.House != "National Assembly" {
		t.Errorf("expected house=National Assembly, got %q", ev.House)
	}
	if !strings.HasPrefix(ev.SourceURL, "https://new.kenyalaw.org/akn/ke/bill/") {
		t.Errorf("expected source_url to point at kenyalaw.org, got %q", ev.SourceURL)
	}
}

// TestHandleBillTimeline_NotFound verifies the handler returns 404 when the
// Bill ID does not match any discovered Bill.
func TestHandleBillTimeline_NotFound(t *testing.T) {
	const html = `<html><body>
<a href="/akn/ke/bill/na/2026-09-07/the-housing-bill-2026/eng@2026-09-07">The Housing Bill 2026</a>
</body></html>`

	adapter := newTestAdapter(html)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/does-not-exist/timeline", nil)
	rr := httptest.NewRecorder()

	handleBillTimeline(rr, req, adapter, "does-not-exist")

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestHandleBillTimeline_EmptyID verifies the handler returns 400 when the
// Bill ID is empty.
func TestHandleBillTimeline_EmptyID(t *testing.T) {
	adapter := newTestAdapter("<html></html>")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills//timeline", nil)
	rr := httptest.NewRecorder()

	handleBillTimeline(rr, req, adapter, "")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestHandleBillTimeline_AdapterError verifies the handler returns 503 when
// the adapter cannot reach the source (e.g., HTTP 500).
func TestHandleBillTimeline_AdapterError(t *testing.T) {
	// Use a client that always returns HTTP 500.
	client := &errorKenyaLawClient{}
	adapter := kenya_law.NewAdapter(client, "CivicIntelligence-test/0.1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/whatever/timeline", nil)
	rr := httptest.NewRecorder()

	handleBillTimeline(rr, req, adapter, "whatever")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// errorKenyaLawClient always returns an HTTP 500, simulating a dead source.
type errorKenyaLawClient struct{}

func (e *errorKenyaLawClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("server error")),
		Header:     make(http.Header),
	}, nil
}

// TestFindBillByID_SubstringMatch verifies the helper's substring fallback:
// callers sometimes pass truncated IDs from URLs.
func TestFindBillByID_SubstringMatch(t *testing.T) {
	const html = `<html><body>
<a href="/akn/ke/bill/na/2026-09-07/the-housing-bill-2026/eng@2026-09-07">The Housing Bill 2026</a>
</body></html>`

	adapter := newTestAdapter(html)
	ctx := context.Background()

	// Full ID works.
	bill, err := findBillByID(ctx, adapter, "ke-bill-2026-09-07-the-housing-bill-2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bill == nil {
		t.Fatal("expected to find the bill by full ID")
	}

	// Substring of the ID also works.
	bill, err = findBillByID(ctx, adapter, "the-housing-bill-2026")
	if err != nil {
		t.Fatalf("unexpected error on substring: %v", err)
	}
	if bill == nil {
		t.Fatal("expected to find the bill by substring of ID")
	}

	// Empty ID returns errEmptyBillID.
	_, err = findBillByID(ctx, adapter, "")
	if err != errEmptyBillID {
		t.Errorf("expected errEmptyBillID, got %v", err)
	}

	// Non-matching ID returns (nil, nil).
	bill, err = findBillByID(ctx, adapter, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bill != nil {
		t.Errorf("expected nil bill for unknown ID, got %+v", bill)
	}
}

// TestBillTimelineResponse_JSONShape verifies the JSON shape of the
// billEventResponse struct — every field must serialize with the documented
// JSON key so the frontend can rely on a stable contract.
func TestBillTimelineResponse_JSONShape(t *testing.T) {
	note := "test note"
	ev := billEventResponse{
		ID:                "ev-1",
		BillID:            "ke-bill",
		EventType:         "publication",
		Date:              "2026-09-07T00:00:00Z",
		DateIsApproximate: false,
		House:             "National Assembly",
		Description:       "Published.",
		SourceURL:         "https://new.kenyalaw.org/akn/ke/bill/na/2026-09-07/test/eng@2026-09-07",
		Confidence:        "high",
		Note:              &note,
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	required := []string{
		`"id":"ev-1"`,
		`"bill_id":"ke-bill"`,
		`"event_type":"publication"`,
		`"date":"2026-09-07T00:00:00Z"`,
		`"date_is_approximate":false`,
		`"house":"National Assembly"`,
		`"description":"Published."`,
		`"source_url":"https://new.kenyalaw.org/akn/ke/bill/na/2026-09-07/test/eng@2026-09-07"`,
		`"confidence":"high"`,
		`"note":"test note"`,
	}
	for _, key := range required {
		if !strings.Contains(s, key) {
			t.Errorf("expected JSON to contain %q, got %s", key, s)
		}
	}
}
