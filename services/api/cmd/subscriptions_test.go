package main

import (
        "bytes"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "strings"
        "testing"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// dispatch runs a request through a chain that injects the principal, then
// returns the recorded response. It uses middleware.OptionalAuth with a
// StaticVerifier to populate the request context the same way the real
// router does in production. For an anonymous principal, no Authorization
// header is added (mirroring real anonymous requests).
func dispatch(handler http.Handler, method, url string, body []byte, p auth.Principal) *httptest.ResponseRecorder {
        var bodyReader *bytes.Reader
        if body != nil {
                bodyReader = bytes.NewReader(body)
        } else {
                bodyReader = bytes.NewReader(nil)
        }
        req := httptest.NewRequest(method, url, bodyReader)
        if p.IsAuthenticated() {
                // OptionalAuth only invokes the verifier when a Bearer token is present.
                // StaticVerifier returns the configured principal for ANY token, so the
                // actual value here is irrelevant — but the header MUST be set.
                req.Header.Set("Authorization", "Bearer test-token")
        }
        v := auth.StaticVerifier{Principal: p}
        wrapped := middleware.OptionalAuth(v)(handler)
        rr := httptest.NewRecorder()
        wrapped.ServeHTTP(rr, req)
        return rr
}

func TestSubscriptionStore_FollowIdempotent(t *testing.T) {
        store := NewSubscriptionStore()

        rec1, err := store.Follow("user-1", EntityBill, "bill-123")
        if err != nil {
                t.Fatalf("first Follow: %v", err)
        }
        if rec1.ID == "" {
                t.Fatal("expected non-empty follow ID")
        }
        if rec1.UserID != "user-1" || rec1.EntityType != EntityBill || rec1.EntityID != "bill-123" {
                t.Fatalf("unexpected record: %+v", rec1)
        }

        // Second Follow for the same (user, type, id) must return the existing record.
        rec2, err := store.Follow("user-1", EntityBill, "bill-123")
        if err != nil {
                t.Fatalf("second Follow: %v", err)
        }
        if rec2.ID != rec1.ID {
                t.Errorf("idempotency: expected same ID %s, got %s", rec1.ID, rec2.ID)
        }

        // Different user following the same entity gets a different record.
        rec3, err := store.Follow("user-2", EntityBill, "bill-123")
        if err != nil {
                t.Fatalf("third Follow: %v", err)
        }
        if rec3.ID == rec1.ID {
                t.Errorf("different user should get different ID, got %s", rec3.ID)
        }
}

func TestSubscriptionStore_Validation(t *testing.T) {
        store := NewSubscriptionStore()

        cases := []struct {
                name       string
                userID     string
                entityType string
                entityID   string
                wantErr    string
        }{
                {"empty user_id", "", EntityBill, "bill-1", "user_id required"},
                {"invalid entity_type", "user-1", "spaceship", "x", "invalid entity_type"},
                {"empty entity_id", "user-1", EntityBill, "", "entity_id required"},
        }
        for _, c := range cases {
                t.Run(c.name, func(t *testing.T) {
                        _, err := store.Follow(c.userID, c.entityType, c.entityID)
                        if err == nil || !strings.Contains(err.Error(), c.wantErr) {
                                t.Errorf("expected error %q, got %v", c.wantErr, err)
                        }
                })
        }
}

func TestSubscriptionStore_ListAndUnfollow(t *testing.T) {
        store := NewSubscriptionStore()
        r1, _ := store.Follow("user-1", EntityBill, "bill-a")
        r2, _ := store.Follow("user-1", EntityTopic, "topic-x")
        _, _ = store.Follow("user-2", EntityBill, "bill-b") // different user

        got := store.List("user-1")
        if len(got) != 2 {
                t.Fatalf("expected 2 follows for user-1, got %d", len(got))
        }

        // Order should be newest-first (rec2 was created after rec1).
        if got[0].ID != r2.ID {
                t.Errorf("expected newest first; got[0].ID=%s, want %s", got[0].ID, r2.ID)
        }

        // Unfollow by another user must NOT delete user-1's follow.
        if store.Unfollow("user-2", r1.ID) {
                t.Error("user-2 should not be able to unfollow user-1's subscription")
        }
        // Now unfollow as the owner.
        if !store.Unfollow("user-1", r1.ID) {
                t.Error("expected Unfollow to return true for owner")
        }
        if store.IsFollowing("user-1", EntityBill, "bill-a") {
                t.Error("IsFollowing should return false after Unfollow")
        }
        if store.Unfollow("user-1", r1.ID) {
                t.Error("Unfollow on already-removed record should return false")
        }
}

func TestSubscriptionStore_IsFollowing(t *testing.T) {
        store := NewSubscriptionStore()
        _, _ = store.Follow("user-1", EntityBill, "bill-a")
        if !store.IsFollowing("user-1", EntityBill, "bill-a") {
                t.Error("expected IsFollowing=true for existing follow")
        }
        if store.IsFollowing("user-1", EntityBill, "bill-b") {
                t.Error("expected IsFollowing=false for non-existent follow")
        }
        if store.IsFollowing("user-2", EntityBill, "bill-a") {
                t.Error("expected IsFollowing=false for different user")
        }
}

func TestHandleSubscribe_Success(t *testing.T) {
        store := NewSubscriptionStore()
        handler := makeSubscriptionsHandler(store)

        body := `{"entity_type":"bill","entity_id":"00000000-0000-0000-0000-000000000001"}`
        p := auth.Principal{UserID: "user-1", Scopes: auth.Scopes{auth.ScopeNotificationWrite}}
        rr := dispatch(handler, http.MethodPost, "/api/v1/subscriptions", []byte(body), p)

        if rr.Code != http.StatusCreated {
                t.Fatalf("expected 201 Created, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var rec FollowRecord
        if err := json.Unmarshal(rr.Body.Bytes(), &rec); err != nil {
                t.Fatalf("invalid JSON: %v", err)
        }
        if rec.UserID != "user-1" {
                t.Errorf("expected UserID 'user-1', got '%s'", rec.UserID)
        }
        if rec.EntityType != "bill" {
                t.Errorf("expected EntityType 'bill', got '%s'", rec.EntityType)
        }
        if rec.EntityID != "00000000-0000-0000-0000-000000000001" {
                t.Errorf("unexpected EntityID: %s", rec.EntityID)
        }
        if !strings.HasPrefix(rec.ID, "flw_") {
                t.Errorf("expected ID prefix 'flw_', got %s", rec.ID)
        }
        if rec.CreatedAt.IsZero() || rec.CreatedAt.After(time.Now().Add(time.Second)) {
                t.Errorf("CreatedAt not set correctly: %s", rec.CreatedAt)
        }
}

func TestHandleSubscribe_AnonymousRejected(t *testing.T) {
        store := NewSubscriptionStore()
        handler := makeSubscriptionsHandler(store)

        body := `{"entity_type":"bill","entity_id":"bill-x"}`
        // No token in request → OptionalAuth yields Anonymous.
        rr := dispatch(handler, http.MethodPost, "/api/v1/subscriptions", []byte(body), auth.Anonymous())
        if rr.Code != http.StatusUnauthorized {
                t.Errorf("expected 401 for anonymous POST, got %d", rr.Code)
        }
}

func TestHandleSubscribe_InvalidJSON(t *testing.T) {
        store := NewSubscriptionStore()
        handler := makeSubscriptionsHandler(store)

        p := auth.Principal{UserID: "user-1"}
        rr := dispatch(handler, http.MethodPost, "/api/v1/subscriptions", []byte("not json"), p)
        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
        }
}

func TestHandleSubscribe_MissingFields(t *testing.T) {
        store := NewSubscriptionStore()
        handler := makeSubscriptionsHandler(store)
        p := auth.Principal{UserID: "user-1"}

        cases := []struct {
                name string
                body string
        }{
                {"missing entity_type", `{"entity_id":"bill-x"}`},
                {"missing entity_id", `{"entity_type":"bill"}`},
                {"empty entity_id", `{"entity_type":"bill","entity_id":""}`},
        }
        for _, c := range cases {
                t.Run(c.name, func(t *testing.T) {
                        rr := dispatch(handler, http.MethodPost, "/api/v1/subscriptions", []byte(c.body), p)
                        if rr.Code != http.StatusBadRequest {
                                t.Errorf("expected 400 for %s, got %d (body=%s)", c.name, rr.Code, rr.Body.String())
                        }
                })
        }
}

func TestHandleSubscribe_InvalidEntityType(t *testing.T) {
        store := NewSubscriptionStore()
        handler := makeSubscriptionsHandler(store)
        p := auth.Principal{UserID: "user-1"}

        body := `{"entity_type":"spaceship","entity_id":"x"}`
        rr := dispatch(handler, http.MethodPost, "/api/v1/subscriptions", []byte(body), p)
        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for invalid entity_type, got %d", rr.Code)
        }
        if !strings.Contains(rr.Body.String(), "invalid entity_type") {
                t.Errorf("expected error to mention invalid entity_type, got %s", rr.Body.String())
        }
}

func TestHandleSubscribe_Idempotent(t *testing.T) {
        store := NewSubscriptionStore()
        handler := makeSubscriptionsHandler(store)
        p := auth.Principal{UserID: "user-1"}

        body := `{"entity_type":"bill","entity_id":"bill-xyz"}`

        rr1 := dispatch(handler, http.MethodPost, "/api/v1/subscriptions", []byte(body), p)
        if rr1.Code != http.StatusCreated {
                t.Fatalf("first POST: expected 201, got %d", rr1.Code)
        }
        var rec1 FollowRecord
        _ = json.Unmarshal(rr1.Body.Bytes(), &rec1)

        rr2 := dispatch(handler, http.MethodPost, "/api/v1/subscriptions", []byte(body), p)
        if rr2.Code != http.StatusCreated {
                t.Fatalf("second POST: expected 201, got %d", rr2.Code)
        }
        var rec2 FollowRecord
        _ = json.Unmarshal(rr2.Body.Bytes(), &rec2)

        if rec1.ID != rec2.ID {
                t.Errorf("idempotency: expected same ID, got %s vs %s", rec1.ID, rec2.ID)
        }
        if len(store.List("user-1")) != 1 {
                t.Errorf("idempotency: expected 1 record in store, got %d", len(store.List("user-1")))
        }
}

func TestHandleListSubscriptions_Success(t *testing.T) {
        store := NewSubscriptionStore()
        _, _ = store.Follow("user-1", EntityBill, "bill-a")
        _, _ = store.Follow("user-1", EntityTopic, "topic-x")
        // Other user's follows should NOT appear.
        _, _ = store.Follow("user-2", EntityBill, "bill-b")

        handler := makeSubscriptionsHandler(store)
        p := auth.Principal{UserID: "user-1"}
        rr := dispatch(handler, http.MethodGet, "/api/v1/subscriptions", nil, p)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }

        var resp struct {
                Items []FollowRecord `json:"items"`
                Total int            `json:"total"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("invalid JSON: %v", err)
        }
        if resp.Total != 2 {
                t.Errorf("expected total=2, got %d", resp.Total)
        }
        for _, f := range resp.Items {
                if f.UserID != "user-1" {
                        t.Errorf("leak: saw follow for user %s while listing user-1", f.UserID)
                }
        }
}

func TestHandleListSubscriptions_FilterByEntityType(t *testing.T) {
        store := NewSubscriptionStore()
        _, _ = store.Follow("user-1", EntityBill, "bill-a")
        _, _ = store.Follow("user-1", EntityTopic, "topic-x")
        _, _ = store.Follow("user-1", EntityCommittee, "committee-1")

        handler := makeSubscriptionsHandler(store)
        p := auth.Principal{UserID: "user-1"}
        rr := dispatch(handler, http.MethodGet, "/api/v1/subscriptions?entity_type=bill", nil, p)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []FollowRecord `json:"items"`
                Total int            `json:"total"`
        }
        _ = json.Unmarshal(rr.Body.Bytes(), &resp)
        if resp.Total != 1 {
                t.Fatalf("filter: expected 1 bill, got %d", resp.Total)
        }
        if resp.Items[0].EntityType != EntityBill {
                t.Errorf("filter: expected EntityType 'bill', got '%s'", resp.Items[0].EntityType)
        }
}

func TestHandleListSubscriptions_AnonymousRejected(t *testing.T) {
        store := NewSubscriptionStore()
        handler := makeSubscriptionsHandler(store)
        rr := dispatch(handler, http.MethodGet, "/api/v1/subscriptions", nil, auth.Anonymous())
        if rr.Code != http.StatusUnauthorized {
                t.Errorf("expected 401 for anonymous GET, got %d", rr.Code)
        }
}

func TestHandleUnsubscribe_Success(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a")
        handler := makeSubscriptionDetailHandler(store)

        p := auth.Principal{UserID: "user-1"}
        rr := dispatch(handler, http.MethodDelete, "/api/v1/subscriptions/"+rec.ID, nil, p)
        if rr.Code != http.StatusNoContent {
                t.Fatalf("expected 204 No Content, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        if store.IsFollowing("user-1", EntityBill, "bill-a") {
                t.Error("expected follow to be removed after DELETE")
        }
}

func TestHandleUnsubscribe_NotOwner(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a")
        handler := makeSubscriptionDetailHandler(store)

        // user-2 attempts to delete user-1's follow.
        p := auth.Principal{UserID: "user-2"}
        rr := dispatch(handler, http.MethodDelete, "/api/v1/subscriptions/"+rec.ID, nil, p)
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404 for non-owner DELETE, got %d", rr.Code)
        }
        if !store.IsFollowing("user-1", EntityBill, "bill-a") {
                t.Error("follow should still exist after non-owner DELETE attempt")
        }
}

func TestHandleUnsubscribe_NotFound(t *testing.T) {
        store := NewSubscriptionStore()
        handler := makeSubscriptionDetailHandler(store)
        p := auth.Principal{UserID: "user-1"}
        rr := dispatch(handler, http.MethodDelete, "/api/v1/subscriptions/flw_does_not_exist", nil, p)
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404, got %d", rr.Code)
        }
}

func TestHandleUnsubscribe_AnonymousRejected(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a")
        handler := makeSubscriptionDetailHandler(store)

        rr := dispatch(handler, http.MethodDelete, "/api/v1/subscriptions/"+rec.ID, nil, auth.Anonymous())
        if rr.Code != http.StatusUnauthorized {
                t.Errorf("expected 401 for anonymous DELETE, got %d", rr.Code)
        }
}

func TestSubscriptionsHandler_MethodNotAllowed(t *testing.T) {
        store := NewSubscriptionStore()
        handler := makeSubscriptionsHandler(store)
        p := auth.Principal{UserID: "user-1"}

        rr := dispatch(handler, http.MethodPut, "/api/v1/subscriptions", []byte("{}"), p)
        if rr.Code != http.StatusMethodNotAllowed {
                t.Errorf("expected 405 for PUT, got %d", rr.Code)
        }
        if rr.Header().Get("Allow") != "GET, POST" {
                t.Errorf("expected Allow header 'GET, POST', got %q", rr.Header().Get("Allow"))
        }
}

func TestSubscriptionDetailHandler_MethodNotAllowed(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a")
        handler := makeSubscriptionDetailHandler(store)
        p := auth.Principal{UserID: "user-1"}

        rr := dispatch(handler, http.MethodGet, "/api/v1/subscriptions/"+rec.ID, nil, p)
        if rr.Code != http.StatusMethodNotAllowed {
                t.Errorf("expected 405 for GET on detail handler, got %d", rr.Code)
        }
        if rr.Header().Get("Allow") != "DELETE" {
                t.Errorf("expected Allow header 'DELETE', got %q", rr.Header().Get("Allow"))
        }
}

func TestNewFollowID_UniqueAndPrefixed(t *testing.T) {
        seen := make(map[string]bool, 100)
        for i := 0; i < 100; i++ {
                id := newFollowID()
                if !strings.HasPrefix(id, "flw_") {
                        t.Fatalf("id %s missing flw_ prefix", id)
                }
                if seen[id] {
                        t.Fatalf("collision at iteration %d: %s", i, id)
                }
                seen[id] = true
        }
}
