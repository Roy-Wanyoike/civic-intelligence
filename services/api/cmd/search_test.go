package main

import (
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"
)

// TestSearch_ReturnsResults verifies that GET /api/v1/search?q=data returns
// non-empty results across Acts and Bills (the Data Protection Act 2019 and
// its originating bill both contain "data" in their title). GAP-49-1: the
// previous handler returned `{"items":[],"total":0}` for every query.
func TestSearch_ReturnsResults(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=data", nil)
        rec := httptest.NewRecorder()

        handleSearch(rec, req)

        if rec.Code != http.StatusOK {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Q     string        `json:"q"`
                Items []searchItem  `json:"items"`
                Total int           `json:"total"`
                Note  string        `json:"note"`
        }
        if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Q != "data" {
                t.Errorf("expected q=data echoed back; got %q", resp.Q)
        }
        if resp.Total == 0 || len(resp.Items) == 0 {
                t.Fatalf("expected non-empty results; got total=%d items=%d", resp.Total, len(resp.Items))
        }
        if resp.Note == "" {
                t.Error("expected note field to surface stopgap status")
        }

        // Every item must carry the documented fields.
        for i, it := range resp.Items {
                if it.Type == "" {
                        t.Errorf("item %d missing type", i)
                }
                if it.ID == "" {
                        t.Errorf("item %d missing id", i)
                }
                if it.Title == "" {
                        t.Errorf("item %d missing title", i)
                }
                if it.URL == "" {
                        t.Errorf("item %d missing url", i)
                }
                switch it.Type {
                case "bill", "act", "constitution_article":
                        // ok
                default:
                        t.Errorf("item %d has unknown type %q", i, it.Type)
                }
        }

        // The Data Protection Act 2019 (and its originating bill) must
        // appear in the results — this is the canonical "data" hit.
        foundAct := false
        foundBill := false
        for _, it := range resp.Items {
                if it.Type == "act" && it.ID == "ke-act-data-protection-2019" {
                        foundAct = true
                }
                if it.Type == "bill" && it.ID == "ke-bill-data-protection-2018" {
                        foundBill = true
                }
        }
        if !foundAct {
                t.Errorf("expected Data Protection Act in results; items=%+v", resp.Items)
        }
        if !foundBill {
                t.Errorf("expected Data Protection Bill in results; items=%+v", resp.Items)
        }
}

// TestSearch_EmptyQuery verifies that an empty query returns 200 with an
// empty items array (not 400). The frontend search page renders before
// the user types anything; a 200 + empty response lets it render "no
// results yet" without an error state.
func TestSearch_EmptyQuery(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=", nil)
        rec := httptest.NewRecorder()

        handleSearch(rec, req)

        if rec.Code != http.StatusOK {
                t.Fatalf("expected 200 for empty query; got %d (the frontend renders before the user types, so 200+empty is more friendly than 400)", rec.Code)
        }
        var resp struct {
                Q     string         `json:"q"`
                Items []searchItem   `json:"items"`
                Total int            `json:"total"`
        }
        if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Total != 0 {
                t.Errorf("expected total=0 for empty query; got %d", resp.Total)
        }
        if len(resp.Items) != 0 {
                t.Errorf("expected 0 items for empty query; got %d", len(resp.Items))
        }
}

// TestSearch_Ranking_TitleBeatsBody verifies that a title match outranks a
// body-only match. The stopgap ranks by (title hit, position) > (body hit,
// position). Two Acts that both contain the needle but only one in the
// title should surface the title-match first.
func TestSearch_Ranking_TitleBeatsBody(t *testing.T) {
        // "constitution" appears in the Constitution of Kenya 2010 Act's
        // title (high rank) and in the description of other Acts (low
        // rank). The constitution-titled Act should appear before any
        // body-only match.
        req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=constitution", nil)
        rec := httptest.NewRecorder()

        handleSearch(rec, req)

        if rec.Code != http.StatusOK {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Items []searchItem `json:"items"`
        }
        if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if len(resp.Items) == 0 {
                t.Fatal("expected non-empty results for 'constitution'")
        }
        // The first hit must be the Constitution Act (title match).
        if resp.Items[0].Type != "act" || resp.Items[0].ID != "ke-act-constitution-2010" {
                t.Errorf("expected Constitution Act as top result; got %+v", resp.Items[0])
        }
}

// TestSearch_CapsAt20 verifies that the result cap is enforced. We use a
// short common needle ("a") which matches almost every Act and Article;
// the handler must return at most 20 items.
func TestSearch_CapsAt20(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=a", nil)
        rec := httptest.NewRecorder()

        handleSearch(rec, req)

        if rec.Code != http.StatusOK {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Items []searchItem `json:"items"`
                Total int          `json:"total"`
        }
        if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if len(resp.Items) > maxSearchResults {
                t.Errorf("expected at most %d items; got %d", maxSearchResults, len(resp.Items))
        }
        if resp.Total > maxSearchResults {
                t.Errorf("expected total <= %d; got %d", maxSearchResults, resp.Total)
        }
        if resp.Total != len(resp.Items) {
                t.Errorf("total (%d) must match items length (%d)", resp.Total, len(resp.Items))
        }
}

// TestSearch_MethodNotAllowed verifies that non-GET methods are rejected.
// The endpoint is documented as GET-only; POST/PUT/DELETE return 405.
func TestSearch_MethodNotAllowed(t *testing.T) {
        req := httptest.NewRequest(http.MethodPost, "/api/v1/search?q=data", nil)
        rec := httptest.NewRecorder()

        handleSearch(rec, req)

        if rec.Code != http.StatusMethodNotAllowed {
                t.Errorf("expected 405 for POST; got %d", rec.Code)
        }
}
