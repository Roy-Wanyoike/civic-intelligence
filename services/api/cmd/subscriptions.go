// Package main — subscriptions handlers (issue #110 — Following).
//
// Implements an in-memory follow/subscription store keyed by user_id.
// The notifications schema in migration 014_notifications.up.sql already
// owns the canonical tables (notifications.follows, notifications.subscriptions,
// notifications.notifications, notifications.delivery_attempts). Once the
// identity service + Postgres connection are wired, this in-memory store will
// be replaced by a SQL-backed repository that writes to those tables; the
// HTTP contract (request/response shapes) is intentionally stable so the
// frontend does not change.
//
// Endpoints:
//   POST   /api/v1/subscriptions      — follow an entity (requires auth)
//   GET    /api/v1/subscriptions      — list the caller's follows (requires auth)
//   DELETE /api/v1/subscriptions/{id} — unfollow (requires auth)
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// EntityType constants for follows. These match the entity_type column on
// notifications.follows in the canonical schema.
const (
	EntityBill        = "bill"
	EntityCommittee   = "committee"
	EntityTopic       = "topic"
	EntityInstitution = "institution"
	EntityPerson      = "person"
)

// validEntityTypes is the allow-list of entity types that may be followed.
var validEntityTypes = map[string]bool{
	EntityBill:        true,
	EntityCommittee:   true,
	EntityTopic:       true,
	EntityInstitution: true,
	EntityPerson:      true,
}

// FollowRecord mirrors a row in notifications.follows.
type FollowRecord struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// followRequest is the JSON body for POST /api/v1/subscriptions.
type followRequest struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

// SubscriptionStore is an in-memory store of follows keyed by follow ID.
// In production this is replaced by notifications.follows in Postgres.
//
// All methods are safe for concurrent use.
type SubscriptionStore struct {
	mu      sync.RWMutex
	follows map[string]FollowRecord // follow_id → record
}

// NewSubscriptionStore returns an empty in-memory store.
func NewSubscriptionStore() *SubscriptionStore {
	return &SubscriptionStore{follows: make(map[string]FollowRecord)}
}

// Follow creates a follow record for the given user + entity. If the user
// already follows the entity, the existing record is returned (idempotent —
// matches the UNIQUE (user_id, entity_type, entity_id) constraint in the SQL
// schema).
func (s *SubscriptionStore) Follow(userID, entityType, entityID string) (FollowRecord, error) {
	if userID == "" {
		return FollowRecord{}, errors.New("user_id required")
	}
	if !validEntityTypes[entityType] {
		return FollowRecord{}, errors.New("invalid entity_type: " + entityType)
	}
	if entityID == "" {
		return FollowRecord{}, errors.New("entity_id required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Idempotent: return existing follow if one already exists for this user+entity.
	for _, f := range s.follows {
		if f.UserID == userID && f.EntityType == entityType && f.EntityID == entityID {
			return f, nil
		}
	}

	rec := FollowRecord{
		ID:         newFollowID(),
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  time.Now().UTC(),
	}
	s.follows[rec.ID] = rec
	return rec, nil
}

// Unfollow removes a follow record by ID, scoped to the given user. Returns
// true if a record was removed, false otherwise.
func (s *SubscriptionStore) Unfollow(userID, followID string) bool {
	if userID == "" || followID == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.follows[followID]
	if !ok || rec.UserID != userID {
		return false
	}
	delete(s.follows, followID)
	return true
}

// List returns all follows for the given user, ordered by CreatedAt descending.
func (s *SubscriptionStore) List(userID string) []FollowRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]FollowRecord, 0)
	for _, f := range s.follows {
		if f.UserID == userID {
			out = append(out, f)
		}
	}
	// Sort by CreatedAt descending (newest first).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].CreatedAt.After(out[j-1].CreatedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// IsFollowing reports whether the user follows the given entity.
func (s *SubscriptionStore) IsFollowing(userID, entityType, entityID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, f := range s.follows {
		if f.UserID == userID && f.EntityType == entityType && f.EntityID == entityID {
			return true
		}
	}
	return false
}

// newFollowID generates a 16-byte hex ID. In production this is a UUID from
// uuid_generate_v4() in Postgres.
func newFollowID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Should never happen with crypto/rand; fall back to timestamp.
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return "flw_" + hex.EncodeToString(b)
}

// --- HTTP handlers ---

// makeSubscriptionsHandler returns an http.HandlerFunc that routes between
// POST (create follow) and GET (list follows) for /api/v1/subscriptions.
func makeSubscriptionsHandler(store *SubscriptionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleSubscribe(w, r, store)
		case http.MethodGet:
			handleListSubscriptions(w, r, store)
		default:
			w.Header().Set("Allow", "GET, POST")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET and POST are supported on /api/v1/subscriptions")
		}
	}
}

// handleSubscribe handles POST /api/v1/subscriptions.
func handleSubscribe(w http.ResponseWriter, r *http.Request, store *SubscriptionStore) {
	p := middleware.PrincipalFromRequest(r)
	if p.IsAnonymous() {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req followRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	req.EntityType = strings.TrimSpace(strings.ToLower(req.EntityType))
	req.EntityID = strings.TrimSpace(req.EntityID)

	if req.EntityType == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "entity_type is required")
		return
	}
	if !validEntityTypes[req.EntityType] {
		writeError(w, http.StatusBadRequest, "bad_request",
			"invalid entity_type; must be one of: bill, committee, topic, institution, person")
		return
	}
	if req.EntityID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "entity_id is required")
		return
	}

	rec, err := store.Follow(p.UserID, req.EntityType, req.EntityID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "follow_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

// handleListSubscriptions handles GET /api/v1/subscriptions.
func handleListSubscriptions(w http.ResponseWriter, r *http.Request, store *SubscriptionStore) {
	p := middleware.PrincipalFromRequest(r)
	if p.IsAnonymous() {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	// Optional filter: ?entity_type=bill
	entityType := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("entity_type")))
	if entityType != "" && !validEntityTypes[entityType] {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid entity_type filter")
		return
	}

	follows := store.List(p.UserID)
	if entityType != "" {
		filtered := follows[:0]
		for _, f := range follows {
			if f.EntityType == entityType {
				filtered = append(filtered, f)
			}
		}
		follows = filtered
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": follows,
		"total": len(follows),
	})
}

// makeSubscriptionDetailHandler returns an http.HandlerFunc that handles
// DELETE /api/v1/subscriptions/{id}.
func makeSubscriptionDetailHandler(store *SubscriptionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", "DELETE")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only DELETE is supported on /api/v1/subscriptions/{id}")
			return
		}
		p := middleware.PrincipalFromRequest(r)
		if p.IsAnonymous() {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}

		// Extract the follow ID from the path: /api/v1/subscriptions/{id}
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/subscriptions/")
		followID := strings.Trim(path, "/")
		if followID == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "subscription id required")
			return
		}

		if !store.Unfollow(p.UserID, followID) {
			writeError(w, http.StatusNotFound, "not_found",
				"subscription not found or does not belong to caller")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
