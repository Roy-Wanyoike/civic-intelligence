// Package main — notifications handlers (issue #112 — Notification Engine).
//
// Implements an in-memory notification store keyed by user_id, with a small
// set of sample notifications seeded on first access (so the /notifications
// page has something to render before any verified events are flowing).
//
// The notifications schema in migration 014_notifications.up.sql already
// owns the canonical tables:
//   - notifications.notifications
//   - notifications.delivery_attempts
// Once the identity service + Postgres connection are wired, this in-memory
// store will be replaced by a SQL-backed repository. The HTTP contract is
// intentionally stable so the frontend does not change.
//
// Endpoints:
//   GET  /api/v1/notifications         — list the caller's notifications
//   POST /api/v1/notifications/{id}/read — mark a notification as read
package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// NotificationEventType constants. These mirror the event_type column on
// notifications.notifications in the canonical schema.
const (
	EventBillPublished      = "bill_published"
	EventBillStageChange    = "bill_stage_change"
	EventBillAmended        = "bill_amended"
	EventBillAssented       = "bill_assented"
	EventNewDocument         = "new_document"
	EventCommitteeReport    = "committee_report"
	EventPublicParticipation = "public_participation"
)

// validEventTypes is the allow-list of event types we currently emit.
var validEventTypes = map[string]bool{
	EventBillPublished:       true,
	EventBillStageChange:     true,
	EventBillAmended:         true,
	EventBillAssented:        true,
	EventNewDocument:         true,
	EventCommitteeReport:     true,
	EventPublicParticipation: true,
}

// NotificationRecord mirrors a row in notifications.notifications.
type NotificationRecord struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id"`
	EventType   string                 `json:"event_type"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Payload     map[string]any         `json:"payload,omitempty"`
	ReadAt      *time.Time             `json:"read_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// NotificationStore is an in-memory store of notifications keyed by user_id
// and notification_id. In production this is replaced by
// notifications.notifications in Postgres.
//
// All methods are safe for concurrent use.
type NotificationStore struct {
	mu             sync.RWMutex
	notifications  map[string]NotificationRecord // notif_id → record
	seededForUsers map[string]bool                // user_id → seeded?
}

// NewNotificationStore returns an empty in-memory store.
func NewNotificationStore() *NotificationStore {
	return &NotificationStore{
		notifications:  make(map[string]NotificationRecord),
		seededForUsers: make(map[string]bool),
	}
}

// seedSampleNotifications populates the store with a small set of sample
// notifications for the given user — these mirror the shape of real
// notifications that the engine will emit when verified events are flowing
// (e.g., a Bill the user follows changed stage). The intent is to give the
// frontend something to render and the QA path something to verify before
// the full pipeline (NATS subscription → matcher → delivery) is wired.
//
// Idempotent: seeding a user that's already been seeded is a no-op.
func (s *NotificationStore) seedSampleNotifications(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seededForUsers[userID] {
		return
	}
	s.seededForUsers[userID] = true

	now := time.Now().UTC()
	samples := []NotificationRecord{
		{
			ID:         newNotificationID(),
			UserID:     userID,
			EntityType: EntityBill,
			EntityID:   "00000000-0000-0000-0000-000000000001",
			EventType:  EventBillStageChange,
			Title:      "Housing Bill, 2024 advanced to Committee Stage",
			Body:        "The Housing Bill, 2024 has moved from First Reading to Committee Stage. The Departmental Committee on Lands will now examine it clause-by-clause.",
			Payload:     map[string]any{"prev_stage": "first_reading", "new_stage": "committee_stage"},
			CreatedAt:   now.Add(-2 * time.Hour),
		},
		{
			ID:         newNotificationID(),
			UserID:     userID,
			EntityType: EntityBill,
			EntityID:   "00000000-0000-0000-0000-000000000002",
			EventType:  EventNewDocument,
			Title:      "New version published: Data Protection (Amendment) Bill",
			Body:        "A new version of the Data Protection (Amendment) Bill, 2024 was published by the Senate Committee on ICT.",
			Payload:     map[string]any{"version_no": 2, "source_url": "https://new.kenyalaw.org/akn/ke/bill/senate/2024-09-07/data-protection-amendment"},
			CreatedAt:   now.Add(-26 * time.Hour),
			ReadAt:      nil,
		},
		{
			ID:         newNotificationID(),
			UserID:     userID,
			EntityType: EntityBill,
			EntityID:   "00000000-0000-0000-0000-000000000003",
			EventType:  EventPublicParticipation,
			Title:      "Public participation call: Public Finance Management (Amendment) Bill",
			Body:        "The Departmental Committee on Finance and National Planning is inviting public submissions on the Public Finance Management (Amendment) Bill, 2024. Submissions close in 14 days.",
			Payload:     map[string]any{"deadline_days": 14, "committee": "Finance and National Planning"},
			CreatedAt:   now.Add(-72 * time.Hour),
			ReadAt:      ptrTime(now.Add(-12 * time.Hour)),
		},
	}
	for _, n := range samples {
		s.notifications[n.ID] = n
	}
}

// List returns the user's notifications, ordered by CreatedAt descending
// (newest first). If `unreadOnly=true`, only unread notifications are
// returned.
func (s *NotificationStore) List(userID string, unreadOnly bool) []NotificationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]NotificationRecord, 0)
	for _, n := range s.notifications {
		if n.UserID != userID {
			continue
		}
		if unreadOnly && n.ReadAt != nil {
			continue
		}
		out = append(out, n)
	}
	// Sort by CreatedAt descending.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].CreatedAt.After(out[j-1].CreatedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// MarkRead marks a notification as read by setting ReadAt to now. Returns
// true if a notification was updated, false if it was already read or doesn't
// exist / belongs to another user.
func (s *NotificationStore) MarkRead(userID, notifID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notifications[notifID]
	if !ok || n.UserID != userID {
		return false
	}
	if n.ReadAt != nil {
		return false
	}
	now := time.Now().UTC()
	n.ReadAt = &now
	s.notifications[notifID] = n
	return true
}

// CountUnread returns the number of unread notifications for the user.
func (s *NotificationStore) CountUnread(userID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, n := range s.notifications {
		if n.UserID == userID && n.ReadAt == nil {
			count++
		}
	}
	return count
}

// Emit creates a new notification for the given user. Used by the (future)
// event matcher. Returns an error if the event_type is not in the allow-list.
func (s *NotificationStore) Emit(userID, entityType, entityID, eventType, title, body string, payload map[string]any) (NotificationRecord, error) {
	if userID == "" {
		return NotificationRecord{}, errors.New("user_id required")
	}
	if !validEntityTypes[entityType] {
		return NotificationRecord{}, errors.New("invalid entity_type: " + entityType)
	}
	if !validEventTypes[eventType] {
		return NotificationRecord{}, errors.New("invalid event_type: " + eventType)
	}
	if title == "" {
		return NotificationRecord{}, errors.New("title required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Idempotent: if a notification with the same (user, entity, event_type)
	// already exists, return it. This matches the dedup constraint the SQL
	// schema will eventually carry (once we add a unique index on
	// (user_id, entity_type, entity_id, event_type, created_at::date)).
	for _, n := range s.notifications {
		if n.UserID == userID && n.EntityType == entityType && n.EntityID == entityID && n.EventType == eventType {
			return n, nil
		}
	}

	rec := NotificationRecord{
		ID:         newNotificationID(),
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
		EventType:  eventType,
		Title:       title,
		Body:        body,
		Payload:     payload,
		CreatedAt:   time.Now().UTC(),
	}
	s.notifications[rec.ID] = rec
	return rec, nil
}

// newNotificationID generates a 16-byte hex ID. In production this is a UUID
// from uuid_generate_v4() in Postgres.
func newNotificationID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return "ntf_" + hex.EncodeToString(b)
}

// ptrTime returns a pointer to the given time. Useful for nullable fields.
func ptrTime(t time.Time) *time.Time {
	return &t
}

// --- HTTP handlers ---

// makeNotificationsHandler returns an http.HandlerFunc that handles GET
// /api/v1/notifications (list user's notifications).
func makeNotificationsHandler(store *NotificationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET is supported on /api/v1/notifications")
			return
		}
		p := middleware.PrincipalFromRequest(r)
		if p.IsAnonymous() {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}

		// Seed sample notifications on first access (issue #112: "For now,
		// return sample notifications (since no events flowing yet)").
		store.seedSampleNotifications(p.UserID)

		unreadOnly := strings.EqualFold(r.URL.Query().Get("unread"), "true") ||
			r.URL.Query().Get("unread") == "1"
		items := store.List(p.UserID, unreadOnly)
		unread := store.CountUnread(p.UserID)

		writeJSON(w, http.StatusOK, map[string]any{
			"items":  items,
			"total":  len(items),
			"unread": unread,
		})
	}
}

// makeNotificationDetailHandler returns an http.HandlerFunc that handles
// POST /api/v1/notifications/{id}/read (mark as read).
func makeNotificationDetailHandler(store *NotificationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only POST is supported on /api/v1/notifications/{id}/read")
			return
		}
		p := middleware.PrincipalFromRequest(r)
		if p.IsAnonymous() {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}

		// Extract path: /api/v1/notifications/{id}/read
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/notifications/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "notification id required")
			return
		}
		notifID := parts[0]

		// Only the "/read" sub-route is supported on the detail handler today.
		if len(parts) < 2 || parts[1] != "read" {
			writeError(w, http.StatusNotFound, "not_found",
				"unknown sub-route; only /read is supported")
			return
		}

		if !store.MarkRead(p.UserID, notifID) {
			writeError(w, http.StatusNotFound, "not_found",
				"notification not found, already read, or does not belong to caller")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
