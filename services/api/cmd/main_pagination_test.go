package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestParsePagination_Defaults verifies that omitting both page and
// page_size yields the OpenAPI defaults (1 / 20).
func TestParsePagination_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts", nil)
	page, pageSize := parsePagination(req)
	if page != 1 {
		t.Errorf("expected page=1, got %d", page)
	}
	if pageSize != defaultPageSize {
		t.Errorf("expected page_size=%d, got %d", defaultPageSize, pageSize)
	}
}

// TestParsePagination_Explicit verifies that valid page + page_size
// values are parsed verbatim.
func TestParsePagination_Explicit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts?page=3&page_size=50", nil)
	page, pageSize := parsePagination(req)
	if page != 3 {
		t.Errorf("expected page=3, got %d", page)
	}
	if pageSize != 50 {
		t.Errorf("expected page_size=50, got %d", pageSize)
	}
}

// TestParsePagination_CapsPageSizeAt100 verifies the documented
// maximum (100) is enforced — anything larger is clamped, not rejected.
func TestParsePagination_CapsPageSizeAt100(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts?page_size=10000", nil)
	_, pageSize := parsePagination(req)
	if pageSize != maxPageSize {
		t.Errorf("expected page_size clamped to %d, got %d", maxPageSize, pageSize)
	}
}

// TestParsePagination_RejectsZeroOrNegative verifies zero/negative
// values fall back to the defaults rather than producing empty pages.
func TestParsePagination_RejectsZeroOrNegative(t *testing.T) {
	cases := []struct {
		query    string
		wantPage int
		wantSize int
	}{
		{"?page=0&page_size=0", 1, defaultPageSize},
		{"?page=-5&page_size=-10", 1, defaultPageSize},
		{"?page=abc&page_size=xyz", 1, defaultPageSize},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/acts"+tc.query, nil)
		page, pageSize := parsePagination(req)
		if page != tc.wantPage {
			t.Errorf("%q: expected page=%d, got %d", tc.query, tc.wantPage, page)
		}
		if pageSize != tc.wantSize {
			t.Errorf("%q: expected page_size=%d, got %d", tc.query, tc.wantSize, pageSize)
		}
	}
}

// TestPaginate_Basic verifies the slice window + hasNext flag for
// several page/page_size combinations over a fixed 7-item slice.
func TestPaginate_Basic(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7}
	cases := []struct {
		name     string
		page     int
		pageSize int
		wantLen  int
		wantNext bool
	}{
		{"first page full", 1, 3, 3, true},
		{"second page partial", 3, 3, 1, false},
		{"single page all", 1, 10, 7, false},
		{"page beyond end", 5, 3, 0, false},
		{"empty slice", 1, 10, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var src []int
			if tc.name != "empty slice" {
				src = items
			}
			paged, hasNext := paginate(src, tc.page, tc.pageSize)
			if len(paged) != tc.wantLen {
				t.Errorf("expected %d items, got %d (%v)", tc.wantLen, len(paged), paged)
			}
			if hasNext != tc.wantNext {
				t.Errorf("expected hasNext=%v, got %v", tc.wantNext, hasNext)
			}
		})
	}
}

// TestHandleActsList_Pagination verifies the /api/v1/acts endpoint
// honours ?page_size= and returns the documented pagination metadata.
// Closes GAP-67-2 — pagination was documented in OpenAPI but ignored.
func TestHandleActsList_Pagination(t *testing.T) {
	// page_size=2 should slice the sample list down to 2 items on page 1.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts?page=1&page_size=2", nil)
	rr := httptest.NewRecorder()
	handleActsList(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Items    []actResponse `json:"items"`
		Total    int           `json:"total"`
		Page     int           `json:"page"`
		PageSize int           `json:"page_size"`
		HasNext  bool          `json:"has_next"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Page != 1 {
		t.Errorf("expected page=1, got %d", resp.Page)
	}
	if resp.PageSize != 2 {
		t.Errorf("expected page_size=2, got %d", resp.PageSize)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items on page 1, got %d", len(resp.Items))
	}
	if resp.Total != len(sampleActs) {
		t.Errorf("expected total=%d (full sample size), got %d", len(sampleActs), resp.Total)
	}
	if !resp.HasNext {
		t.Errorf("expected has_next=true when sample has more than page_size items")
	}
}

// TestHandleActsList_PaginationCap verifies page_size > 100 is clamped
// to 100 even if the caller asks for a larger value.
func TestHandleActsList_PaginationCap(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts?page_size=9999", nil)
	rr := httptest.NewRecorder()
	handleActsList(rr, req)

	var resp struct {
		PageSize int `json:"page_size"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PageSize != maxPageSize {
		t.Errorf("expected page_size clamped to %d, got %d", maxPageSize, resp.PageSize)
	}
}
