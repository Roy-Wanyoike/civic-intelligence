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
        EntityAct         = "act"
)

// validEntityTypes is the allow-list of entity types that may be followed.
var validEntityTypes = map[string]bool{
        EntityBill:        true,
        EntityCommittee:   true,
        EntityTopic:       true,
        EntityInstitution: true,
        EntityPerson:      true,
        EntityAct:         true,
}

// FollowRecord mirrors a row in notifications.follows.
//
// Channels (issue #279) controls which delivery mechanisms fire when a
// matching notification is emitted. The default is ["in_app"] (the
// notification row is always written so the /notifications page has the
// alert). When "email" is present in the slice, the daily digest pipeline
// also sends the recipient an email. The slice is stored as a slice (not
// a bitfield) so the OpenAPI contract can extend to "sms", "push", and
// "webhook" later without a schema migration.
type FollowRecord struct {
        ID         string    `json:"id"`
        UserID     string    `json:"user_id"`
        EntityType string    `json:"entity_type"`
        EntityID   string    `json:"entity_id"`
        Channels   []string  `json:"channels"`
        CreatedAt  time.Time `json:"created_at"`
}

// followRequest is the JSON body for POST /api/v1/subscriptions.
type followRequest struct {
        EntityType string   `json:"entity_type"`
        EntityID   string   `json:"entity_id"`
        Channels   []string `json:"channels,omitempty"`
}

// patchChannelsRequest is the JSON body for PATCH /api/v1/subscriptions/{id}.
// Only the channels field is mutable — entity_type / entity_id are immutable
// after creation (a follow is for a specific entity). The body MUST carry a
// non-empty channels array; pass ["in_app"] to disable email delivery while
// keeping the in-app channel on.
type patchChannelsRequest struct {
        Channels []string `json:"channels"`
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
//
// The channels slice controls which delivery mechanisms fire when a matching
// notification is emitted (issue #279). When channels is empty, the default
// (["in_app"]) is applied — the in-app channel is always on so the
// /notifications page renders the alert. Pass "email" in the slice to opt
// the recipient into email delivery via the daily digest pipeline.
func (s *SubscriptionStore) Follow(userID, entityType, entityID string, channels []string) (FollowRecord, error) {
        if userID == "" {
                return FollowRecord{}, errors.New("user_id required")
        }
        if !validEntityTypes[entityType] {
                return FollowRecord{}, errors.New("invalid entity_type: " + entityType)
        }
        if entityID == "" {
                return FollowRecord{}, errors.New("entity_id required")
        }
        normChannels, err := normaliseChannels(channels)
        if err != nil {
                return FollowRecord{}, err
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
                Channels:   normChannels,
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

// Get returns the follow record for the given ID, scoped to userID. Returns
// (FollowRecord{}, false) when the record does not exist or belongs to
// another user. Used by the PATCH handler (issue #279) to look up the
// record before mutating channels.
func (s *SubscriptionStore) Get(userID, followID string) (FollowRecord, bool) {
        if userID == "" || followID == "" {
                return FollowRecord{}, false
        }
        s.mu.RLock()
        defer s.mu.RUnlock()
        rec, ok := s.follows[followID]
        if !ok || rec.UserID != userID {
                return FollowRecord{}, false
        }
        return rec, true
}

// UpdateChannels replaces the channels slice on the follow record with the
// supplied value, scoped to userID. The channels array is validated + the
// in_app channel is always preserved (a subscription with no in-app channel
// would silently swallow alerts). Returns the updated record, or
// (FollowRecord{}, false) when the record does not exist / does not belong
// to the caller.
//
// Used by the PATCH /api/v1/subscriptions/{id} handler (issue #279).
func (s *SubscriptionStore) UpdateChannels(userID, followID string, channels []string) (FollowRecord, bool) {
        norm, err := normaliseChannels(channels)
        if err != nil {
                return FollowRecord{}, false
        }
        s.mu.Lock()
        defer s.mu.Unlock()
        rec, ok := s.follows[followID]
        if !ok || rec.UserID != userID {
                return FollowRecord{}, false
        }
        rec.Channels = norm
        s.follows[followID] = rec
        return rec, true
}

// ListAll returns every follow record across every user. Used by the daily
// digest pipeline (issue #279) to fan out an email to every subscriber who
// has "email" in their channels. In production this query is replaced by
// SELECT … WHERE 'email' = ANY(channels) on notifications.follows.
func (s *SubscriptionStore) ListAll() []FollowRecord {
        s.mu.RLock()
        defer s.mu.RUnlock()
        out := make([]FollowRecord, 0, len(s.follows))
        for _, f := range s.follows {
                out = append(out, f)
        }
        return out
}

// EmailSubscribers returns the distinct user IDs whose channels slice
// includes "email". Used by the refresh handler (issue #279) to compute
// the digest recipient list without re-scanning the store per-user.
func (s *SubscriptionStore) EmailSubscribers() []string {
        s.mu.RLock()
        defer s.mu.RUnlock()
        seen := make(map[string]bool)
        for _, f := range s.follows {
                if !hasChannel(f.Channels, ChannelEmail) {
                        continue
                }
                seen[f.UserID] = true
        }
        out := make([]string, 0, len(seen))
        for uid := range seen {
                out = append(out, uid)
        }
        return out
}

// IsEmailSubscriber reports whether the given user has at least one
// follow whose channels slice includes "email". Used by tests + the
// PATCH handler's response logging.
func (s *SubscriptionStore) IsEmailSubscriber(userID string) bool {
        s.mu.RLock()
        defer s.mu.RUnlock()
        for _, f := range s.follows {
                if f.UserID != userID {
                        continue
                }
                if hasChannel(f.Channels, ChannelEmail) {
                        return true
                }
        }
        return false
}

// normaliseChannels validates + canonicalises a channels slice. Empty /
// nil input returns the default channel set (["in_app"]). Whitespace is
// trimmed and lower-cased. Duplicates are removed. Unknown channel names
// produce an error so a typo (e.g. "emial") does not silently disable
// delivery.
//
// The in_app channel is ALWAYS present in the returned slice — a follow
// with no in-app channel would never surface on the /notifications page,
// which would be a silent footgun. If the caller omits it, normalisation
// adds it.
func normaliseChannels(channels []string) ([]string, error) {
        if len(channels) == 0 {
                // Defensive copy — the caller may mutate the package-level
                // defaultChannels slice otherwise.
                out := make([]string, len(defaultChannels))
                copy(out, defaultChannels)
                return out, nil
        }
        seen := make(map[string]bool, len(channels))
        out := make([]string, 0, len(channels))
        for _, c := range channels {
                c = strings.TrimSpace(strings.ToLower(c))
                if c == "" {
                        continue
                }
                if !validChannels[c] {
                        return nil, errors.New("invalid channel: " + c)
                }
                if seen[c] {
                        continue
                }
                seen[c] = true
                out = append(out, c)
        }
        if !seen[ChannelInApp] {
                out = append(out, ChannelInApp)
        }
        return out, nil
}

// hasChannel reports whether the channels slice contains the given channel
// name. Case-insensitive on the channel name to be forgiving against callers
// that pass uppercase variants.
func hasChannel(channels []string, want string) bool {
        for _, c := range channels {
                if strings.EqualFold(c, want) {
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
                        "invalid entity_type; must be one of: bill, committee, topic, institution, person, act")
                return
        }
        if req.EntityID == "" {
                writeError(w, http.StatusBadRequest, "bad_request", "entity_id is required")
                return
        }

        rec, err := store.Follow(p.UserID, req.EntityType, req.EntityID, req.Channels)
        if err != nil {
                // Channel validation errors are 400s; anything else is a 500.
                if strings.HasPrefix(err.Error(), "invalid channel") {
                        writeError(w, http.StatusBadRequest, "bad_request", err.Error())
                        return
                }
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
// PATCH (update channels, issue #279) and DELETE (unfollow) on
// /api/v1/subscriptions/{id}.
//
// The optional userEmails registry is populated by the PATCH handler when
// the caller's principal carries an email claim — the daily digest
// pipeline (see digest.go) needs that email address to deliver mail.
// Pass nil when the handler is wired in tests that do not exercise the
// email fan-out.
func makeSubscriptionDetailHandler(store *SubscriptionStore, userEmails *UserEmailStore) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                switch r.Method {
                case http.MethodPatch:
                        handlePatchSubscription(w, r, store, userEmails)
                case http.MethodDelete:
                        handleUnsubscribe(w, r, store)
                default:
                        w.Header().Set("Allow", "DELETE, PATCH")
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
                                "only DELETE and PATCH are supported on /api/v1/subscriptions/{id}")
                }
        }
}

// handlePatchSubscription handles PATCH /api/v1/subscriptions/{id}.
//
// The request body MUST be a JSON object of the form
//
//      {"channels": ["in_app", "email"]}
//
// The channels array replaces the existing one. The in_app channel is
// always preserved by normalisation — a follow with no in-app channel
// would silently swallow alerts (the /notifications page would never
// render them). Unknown channel names (e.g. "sms", "emial") return 400.
//
// When the caller's principal carries an email claim, it is captured into
// the userEmails registry so the daily digest pipeline can deliver mail
// to that address. This captures the email opportunistically on every
// PATCH — the citizen's address travels with the request that opts them
// into the email channel.
func handlePatchSubscription(w http.ResponseWriter, r *http.Request, store *SubscriptionStore, userEmails *UserEmailStore) {
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

        var req patchChannelsRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
                return
        }
        if len(req.Channels) == 0 {
                writeError(w, http.StatusBadRequest, "bad_request", "channels is required")
                return
        }

        rec, ok := store.UpdateChannels(p.UserID, followID, req.Channels)
        if !ok {
                writeError(w, http.StatusNotFound, "not_found",
                        "subscription not found, does not belong to caller, or channels array invalid")
                return
        }

        // Opportunistic email capture: when the caller has an email claim AND
        // they just enabled the email channel, remember the address so the
        // daily digest pipeline can reach them. We store the email regardless
        // of whether "email" was just added — the registry is idempotent and
        // keeps the address fresh in case it changes.
        if userEmails != nil && p.Email != "" {
                userEmails.Set(p.UserID, p.Email)
        }
        writeJSON(w, http.StatusOK, rec)
}

// handleUnsubscribe handles DELETE /api/v1/subscriptions/{id}.
func handleUnsubscribe(w http.ResponseWriter, r *http.Request, store *SubscriptionStore) {
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
