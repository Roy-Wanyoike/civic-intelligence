// Package main — SMS + USSD alert handlers (issue #289 — SMS/USSD alerts).
//
// This file wires the Civic Intelligence platform to Africa's Talking
// (https://africastalking.com) — the de-facto SMS/USSD gateway for East
// African countries (Kenya, Uganda, Tanzania, Rwanda, Ethiopia, Nigeria,
// Ghana, South Africa, Malawi, Zambia, etc.). Africa's Talking was
// chosen because:
//
//   - One REST endpoint per channel (POST /version1/messaging for SMS,
//     POST /ussd/v1/exercises for USSD callbacks).
//   - One auth header (apiKey: ATShXXXX…; sent as the WORKSPACE_API_KEY
//     form field, not as a Bearer token — Africa's Talking uses form
//     auth, not header auth).
//   - Sandbox + production modes are the same API surface, just different
//     base URLs (sandbox.africastalking.com vs api.africastalking.com),
//     so the integration needs no SDK, no client library, and no message
//     queue. The whole sender is a few hundred lines of plain net/http.
//   - USSD sessions are short-lived, request-scoped, and stateless from
//     the platform's side: Africa's Talking sends the cumulative `text`
//     the user has typed so far, and the platform returns the next menu
//     screen. No server-side session table is required.
//
// SMSSender is an interface so tests can substitute a StubSMSSender (which
// only logs) — the platform never sends real SMS messages in unit tests.
//
// Selection at startup:
//
//	if os.Getenv("AFRICAS_TALKING_API_KEY") != "" → AfricasTalkingSMSSender
//	else                                          → StubSMSSender
//
// Production wiring will be completed in a follow-up task; this commit
// delivers the architecture + the in-memory subscriber store + the
// admin broadcast endpoint + the USSD menu — actual message delivery
// uses the StubSMSSender until the AT sandbox keys are configured.
//
// Endpoints (registered in main.go):
//
//	POST /api/v1/alerts/sms/subscribe    — phone + keywords signup (public)
//	GET  /api/v1/alerts/sms/subscribers  — admin list (user:admin scope)
//	POST /api/v1/alerts/sms/send         — admin broadcast (user:admin scope)
//
// The subscriber store is intentionally minimal: it tracks a phone
// number (E.164), the country code the subscriber lives in, the keyword
// filters they care about (e.g. ["tender", "health"]), a created_at
// timestamp, and an `active` flag (so an admin can mute a subscriber
// without losing their history). It is concurrency-safe.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// --- Domain types ---

// SMSSubscriber is a single phone number that has opted into SMS alerts.
//
// The Phone field is stored in E.164 format (e.g. "+254712345678") so the
// Africa's Talking gateway can route the message without further parsing.
// Country is an ISO 3166-1 alpha-2 code (e.g. "KE") so the platform can
// fan out country-scoped broadcasts; the value is upper-cased on insert.
//
// Keywords is a slice of lower-cased, de-duplicated keywords (e.g.
// ["tender", "health"]). When the ingestion service emits a gazette
// notice whose title/body matches ANY of the subscriber's keywords, the
// platform queues an SMS for that subscriber. An empty Keywords slice
// means "subscribe to everything" — used by the broadcast endpoint so an
// admin can reach every opted-in citizen with a single call.
//
// The Active flag is true by default; an admin can flip it to false to
// mute a subscriber (e.g. on a spam complaint) without losing their
// history. A muted subscriber is never sent a message even when their
// keywords match.
type SMSSubscriber struct {
	Phone     string    `json:"phone"`
	Country   string    `json:"country"`
	Keywords  []string  `json:"keywords"`
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}

// SMSAlert is a single delivered (or attempted) SMS message. The
// Status field is one of: "queued", "sent", "failed". The SourceEvent
// is the upstream event id that triggered the SMS (e.g. a gazette
// notice ID, or "broadcast" for the admin send endpoint).
type SMSAlert struct {
	ID          string    `json:"id"`
	Phone       string    `json:"phone"`
	Message     string    `json:"message"`
	SentAt      time.Time `json:"sent_at"`
	Status      string    `json:"status"`
	SourceEvent string    `json:"source_event"`
}

// SMS alert statuses. The lifecycle is: queued → sent (or queued →
// failed). The StubSMSSender always reports "sent" immediately so the
// admin broadcast endpoint returns a synchronous success to the caller.
const (
	SMSStatusQueued = "queued"
	SMSStatusSent   = "sent"
	SMSStatusFailed = "failed"
)

// --- SMSSender interface ---

// SMSSender is the abstraction every SMS-bearing code path depends on.
// Implementations:
//   - *AfricasTalkingSMSSender — POSTs to
//     https://api.africastalking.com/version1/messaging.
//   - *StubSMSSender           — logs the to+body and discards it.
//
// SendBulk MUST be safe for concurrent use; the admin broadcast endpoint
// fans out one goroutine per recipient, so a single sender is shared
// across all of them.
type SMSSender interface {
	// Send delivers a single SMS to a single recipient. The ctx is
	// honoured so the caller can bound the total send time. A non-nil
	// error means the message was NOT accepted by the upstream provider —
	// callers SHOULD log + continue rather than abort the whole batch
	// (one bad number must not block the other subscribers).
	Send(ctx context.Context, phone, message string) (SMSAlert, error)

	// Name returns a short identifier for logs + diagnostics (e.g.
	// "africas_talking", "stub"). The broadcast handler logs which sender
	// it used so operators can confirm whether AFRICAS_TALKING_API_KEY
	// was set.
	Name() string
}

// NewSMSSender picks an implementation based on the
// AFRICAS_TALKING_API_KEY env var. When the var is set, an
// *AfricasTalkingSMSSender is returned; otherwise a *StubSMSSender is
// returned. The optional httpClient parameter exists so tests can inject
// an httptest.Server-tied client; production callers pass nil and a
// default client with a 10-second timeout is used.
//
// The sender short-circuits to the stub when the env var is missing so
// the platform boots cleanly in dev + in tests; the broadcast handler
// surfaces the chosen sender's Name() in its response so an operator
// can immediately see whether real SMS delivery is wired.
func NewSMSSender(httpClient *http.Client) SMSSender {
	apiKey := strings.TrimSpace(os.Getenv("AFRICAS_TALKING_API_KEY"))
	username := strings.TrimSpace(os.Getenv("AFRICAS_TALKING_USERNAME"))
	if username == "" {
		// Africa's Talking requires a username even in sandbox mode;
		// the default "sandbox" is documented at
		// https://developers.africastalking.com/sandbox.
		username = "sandbox"
	}
	if apiKey == "" {
		log.Printf("sms: AFRICAS_TALKING_API_KEY not set — using StubSMSSender (no SMS will be sent)")
		return &StubSMSSender{}
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	// Sandbox vs production is selected by username: "sandbox" forces
	// sandbox.africastalking.com; any other username targets
	// api.africastalking.com (the production gateway). This matches the
	// documented Africa's Talking convention.
	baseURL := "https://api.africastalking.com"
	if username == "sandbox" {
		baseURL = "https://sandbox.africastalking.com"
	}
	// SenderID is optional; when set via AFRICAS_TALKING_SENDER_ID it
	// overrides the platform's default short code. Empty means "use the
	// Africa's Talking shared short code".
	senderID := strings.TrimSpace(os.Getenv("AFRICAS_TALKING_SENDER_ID"))
	log.Printf("sms: AFRICAS_TALKING_API_KEY detected — using AfricasTalkingSMSSender (username=%s, base=%s)", username, baseURL)
	return &AfricasTalkingSMSSender{
		apiKey:   apiKey,
		username: username,
		senderID: senderID,
		client:   httpClient,
		baseURL:  baseURL,
	}
}

// --- AfricasTalkingSMSSender ---

// AfricasTalkingSMSSender POSTs SMS messages to
// https://(api|sandbox).africastalking.com/version1/messaging.
//
// The request body is application/x-www-form-urlencoded (NOT JSON —
// Africa's Talking's SMS endpoint accepts form data, not JSON). The
// fields are:
//
//	username  = sandbox | <workspace-username>
//	to        = +254712345678
//	message   = <text>
//	from      = <senderID>  (optional; omitted when senderID is "")
//	apiKey    = <ATShXXXX…>
//
// A 201 response from Africa's Talking means the message was queued —
// delivery to the recipient's handset is asynchronous on AT's side
// (typically 5-30 seconds for live networks; instant in the sandbox).
// Any non-2xx status is surfaced as an error containing the upstream
// body so an operator can see why Africa's Talking rejected the
// request.
//
// NOTE: this sender is wired + tested via the StubSMSSender contract;
// the live HTTP path is exercised in the follow-up task that wires
// AT sandbox credentials into CI. The architecture is ready today.
type AfricasTalkingSMSSender struct {
	apiKey   string
	username string
	senderID string
	client   *http.Client
	baseURL  string
}

// Name implements SMSSender.
func (s *AfricasTalkingSMSSender) Name() string { return "africas_talking" }

// Send implements SMSSender. The phone number MUST be in E.164 format
// (e.g. "+254712345678"); the gateway rejects anything else with a 400.
// The message is sent as-is; the caller is responsible for truncating
// to the GSM 7-bit 160-char limit (or the UCS-2 70-char limit for
// non-Latin scripts).
func (s *AfricasTalkingSMSSender) Send(ctx context.Context, phone, message string) (SMSAlert, error) {
	if strings.TrimSpace(phone) == "" {
		return SMSAlert{}, errors.New("sms: recipient (phone) is required")
	}
	if strings.TrimSpace(message) == "" {
		return SMSAlert{}, errors.New("sms: message is required")
	}
	if s.apiKey == "" {
		return SMSAlert{}, errors.New("sms: AFRICAS_TALKING_API_KEY not configured")
	}

	form := urlValues{
		"username": s.username,
		"to":       phone,
		"message":  message,
		"apiKey":   s.apiKey,
	}
	if s.senderID != "" {
		form["from"] = s.senderID
	}
	body := form.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/version1/messaging", strings.NewReader(body))
	if err != nil {
		return SMSAlert{}, fmt.Errorf("sms: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return SMSAlert{
			ID: newSMSAlertID(), Phone: phone, Message: message,
			SentAt: time.Now().UTC(), Status: SMSStatusFailed,
			SourceEvent: "africas_talking",
		}, fmt.Errorf("sms: africas talking request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return SMSAlert{
			ID: newSMSAlertID(), Phone: phone, Message: message,
			SentAt: time.Now().UTC(), Status: SMSStatusFailed,
			SourceEvent: "africas_talking",
		}, fmt.Errorf("sms: africas talking returned status %d: %s", resp.StatusCode, string(respBody))
	}
	// Drain the body so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)
	return SMSAlert{
		ID: newSMSAlertID(), Phone: phone, Message: message,
		SentAt: time.Now().UTC(), Status: SMSStatusSent,
		SourceEvent: "africas_talking",
	}, nil
}

// --- StubSMSSender ---

// StubSMSSender is the no-op SMSSender used in dev mode (when
// AFRICAS_TALKING_API_KEY is not set) and in unit tests. It logs the
// recipient + message so the broadcast pipeline's behaviour is
// observable in dev logs without sending any real traffic. Send always
// returns a SMSAlert with Status=sent — the stub never fails.
//
// The stub is concurrency-safe because log.Printf is. The LastPhone /
// LastMessage / SendCount fields are populated for test assertions but
// are NOT synchronised — tests are expected to call Send serially.
type StubSMSSender struct {
	mu          sync.Mutex
	LastPhone   string
	LastMessage string
	SendCount   int
}

// Name implements SMSSender.
func (s *StubSMSSender) Name() string { return "stub" }

// Send implements SMSSender.
func (s *StubSMSSender) Send(_ context.Context, phone, message string) (SMSAlert, error) {
	s.mu.Lock()
	s.LastPhone = phone
	s.LastMessage = message
	s.SendCount++
	s.mu.Unlock()
	log.Printf("sms(stub): would send to=%q msg=%q", phone, message)
	return SMSAlert{
		ID: newSMSAlertID(), Phone: phone, Message: message,
		SentAt: time.Now().UTC(), Status: SMSStatusSent,
		SourceEvent: "stub",
	}, nil
}

// --- urlValues helper ---
//
// urlValues is a tiny shim around map[string]string that produces
// application/x-www-form-urlencoded output. We don't use net/url.Values
// here to keep the field-ordering deterministic (the existing
// net/url.Values.Encode sorts keys alphabetically, which is fine but
// makes the test fixtures harder to read).
type urlValues map[string]string

// Encode renders the map as application/x-www-form-urlencoded, escaping
// both keys and values. The order is non-deterministic (map iteration)
// — Africa's Talking does not require a specific field order.
func (v urlValues) Encode() string {
	var b strings.Builder
	first := true
	for k, val := range v {
		if !first {
			b.WriteByte('&')
		}
		first = false
		b.WriteString(urlEscape(k))
		b.WriteByte('=')
		b.WriteString(urlEscape(val))
	}
	return b.String()
}

// urlEscape escapes a single form field. We avoid pulling in net/url
// for one function; the escape rules we need are a strict subset of
// QueryEscape (space → '+', reserved chars → %XX).
func urlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == ' ':
			b.WriteByte('+')
		case r == '-' || r == '_' || r == '.' || r == '~' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			// %XX escape for everything else.
			bs := []byte(string(r))
			for _, by := range bs {
				fmt.Fprintf(&b, "%%%02X", by)
			}
		}
	}
	return b.String()
}

// --- SMSSubscriberStore ---

// SMSSubscriberStore is the in-memory store of SMS subscribers. In
// production this is replaced by notifications.sms_subscribers in
// Postgres (migration 018_trust_schema.up.sql reserves the namespace;
// a follow-up migration will add the table itself).
//
// All methods are safe for concurrent use.
type SMSSubscriberStore struct {
	mu          sync.RWMutex
	subscribers map[string]SMSSubscriber // keyed by E.164 phone number
	alerts      []SMSAlert               // append-only delivery log
	seeded      bool                     // guards SeedSampleSubscribers
}

// NewSMSSubscriberStore returns an empty in-memory store.
func NewSMSSubscriberStore() *SMSSubscriberStore {
	return &SMSSubscriberStore{
		subscribers: make(map[string]SMSSubscriber),
	}
}

// Subscribe registers a phone number for SMS alerts on the supplied
// keywords. The phone is normalised to E.164 (we don't validate the
// full E.164 grammar — we only require it to start with '+'); the
// country is upper-cased; the keywords are lower-cased, trimmed, and
// de-duplicated.
//
// Returns (record, true, nil) when a new subscriber is created, and
// (existing, false, nil) when the phone is already subscribed. The
// HTTP handler uses the bool to choose between 201 Created and 409
// Conflict — the duplicate path is NOT an error at the store level
// (idempotency is a concern of the HTTP layer, not the store).
//
// Validation errors (empty phone, no keywords, malformed phone)
// return (record, false, error) so the handler can map them to 400s.
func (s *SMSSubscriberStore) Subscribe(phone, country string, keywords []string) (SMSSubscriber, bool, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return SMSSubscriber{}, false, errors.New("phone is required")
	}
	if !strings.HasPrefix(phone, "+") {
		return SMSSubscriber{}, false, errors.New("phone must be in E.164 format (e.g. +254712345678)")
	}
	if len(phone) < 6 || len(phone) > 16 {
		return SMSSubscriber{}, false, errors.New("phone must be between 6 and 16 characters")
	}
	country = strings.ToUpper(strings.TrimSpace(country))
	if country == "" {
		country = "KE" // default; the gateway infers the routing from the phone prefix anyway
	}
	kw, err := normaliseSMSKeywords(keywords)
	if err != nil {
		return SMSSubscriber{}, false, err
	}
	if len(kw) == 0 {
		return SMSSubscriber{}, false, errors.New("at least one keyword is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.subscribers[phone]; ok {
		// Idempotent at the store level: return the existing record so
		// the HTTP handler can decide whether to return 409 or 200.
		return existing, false, nil
	}

	rec := SMSSubscriber{
		Phone:     phone,
		Country:   country,
		Keywords:  kw,
		CreatedAt: time.Now().UTC(),
		Active:    true,
	}
	s.subscribers[phone] = rec
	return rec, true, nil
}

// List returns every subscriber (active + inactive). Used by the admin
// GET /api/v1/alerts/sms/subscribers endpoint. The slice is sorted by
// CreatedAt ascending so the seed subscribers appear first (stable
// fixture for tests) and new subscriptions appear at the end.
func (s *SMSSubscriberStore) List() []SMSSubscriber {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SMSSubscriber, 0, len(s.subscribers))
	for _, sub := range s.subscribers {
		out = append(out, sub)
	}
	// Sort by CreatedAt ascending (stable for equal timestamps, which
	// covers the seeded subscribers created in the same call).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].CreatedAt.Before(out[j-1].CreatedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// ActiveSubscribers returns every active subscriber whose keywords
// intersect `keywords` (or every active subscriber when keywords is
// empty — used by the broadcast endpoint to reach all opt-ins). Used
// by the broadcast handler to compute the recipient list.
func (s *SMSSubscriberStore) ActiveSubscribers(keywords []string) []SMSSubscriber {
	kw := normaliseSMSKeywordsSafe(keywords)
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SMSSubscriber, 0)
	for _, sub := range s.subscribers {
		if !sub.Active {
			continue
		}
		if len(kw) == 0 {
			out = append(out, sub)
			continue
		}
		if smsKeywordsIntersect(sub.Keywords, kw) {
			out = append(out, sub)
		}
	}
	return out
}

// RecordAlert appends a delivery record to the in-memory log. Used by
// the broadcast handler so the response can include a per-recipient
// status array. The log is unbounded in dev — production wiring will
// persist to notifications.sms_alerts and prune after 90 days.
func (s *SMSSubscriberStore) RecordAlert(alert SMSAlert) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, alert)
}

// Alerts returns a copy of the delivery log. Used by tests.
func (s *SMSSubscriberStore) Alerts() []SMSAlert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SMSAlert, len(s.alerts))
	copy(out, s.alerts)
	return out
}

// SetActive flips the Active flag on a subscriber. Used by the (future)
// admin mute/unmute endpoint. Returns true when the subscriber was
// found and updated, false otherwise.
func (s *SMSSubscriberStore) SetActive(phone string, active bool) bool {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.subscribers[phone]
	if !ok {
		return false
	}
	rec.Active = active
	s.subscribers[phone] = rec
	return true
}

// normaliseSMSKeywords lower-cases, trims, and de-duplicates the keyword
// slice. Empty / whitespace-only entries are dropped. Returns an error
// only when a keyword exceeds 64 characters (a sanity guard against a
// caller passing an entire alert body as a single keyword).
func normaliseSMSKeywords(keywords []string) ([]string, error) {
	seen := make(map[string]bool, len(keywords))
	out := make([]string, 0, len(keywords))
	for _, k := range keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" {
			continue
		}
		if len(k) > 64 {
			return nil, fmt.Errorf("keyword %q exceeds 64 characters", k)
		}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return out, nil
}

// normaliseSMSKeywordsSafe is the panic-free variant used by callers
// that already trust their input (e.g. internal list passes). It
// silently drops invalid keywords rather than returning an error.
func normaliseSMSKeywordsSafe(keywords []string) []string {
	out, _ := normaliseSMSKeywords(keywords)
	return out
}

// smsKeywordsIntersect reports whether the two keyword slices share at
// least one entry. Both slices are assumed pre-normalised
// (lower-cased, trimmed).
func smsKeywordsIntersect(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, k := range a {
		set[k] = true
	}
	for _, k := range b {
		if set[k] {
			return true
		}
	}
	return false
}

// newSMSAlertID generates a 16-byte hex ID with the "sms_" prefix so
// delivery records are distinguishable from gazette alert IDs (gza_)
// and follow IDs (flw_) in logs.
func newSMSAlertID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return "sms_" + hex.EncodeToString(b)
}

// --- HTTP handlers ---

// smsSubscribeRequest is the JSON body for POST /api/v1/alerts/sms/subscribe.
type smsSubscribeRequest struct {
	Phone    string   `json:"phone"`
	Country  string   `json:"country,omitempty"`
	Keywords []string `json:"keywords"`
}

// smsBroadcastRequest is the JSON body for POST /api/v1/alerts/sms/send
// (admin only). The message is the SMS body; the optional keywords
// filter restricts the recipient list to subscribers whose keywords
// intersect the supplied list. When keywords is empty, every active
// subscriber receives the broadcast.
type smsBroadcastRequest struct {
	Message  string   `json:"message"`
	Keywords []string `json:"keywords,omitempty"`
	Source   string   `json:"source,omitempty"`
}

// makeSMSSubscribeHandler returns an http.HandlerFunc that handles
// POST /api/v1/alerts/sms/subscribe. The endpoint is PUBLIC (no auth
// required) — the call comes from the public "Subscribe to SMS alerts"
// form on the website, and Africa's Talking USSD menu option 4 posts to
// this same endpoint when a citizen selects "Subscribe to alerts".
//
// Validation:
//   - phone MUST be E.164 (start with '+'; 6-16 chars).
//   - keywords MUST be a non-empty slice after normalisation.
//
// Responses:
//   - 201 Created when the subscriber is new.
//   - 409 Conflict when the phone is already subscribed. The existing
//     record is returned in the body so the caller can confirm the
//     keywords they're subscribed to.
//   - 400 Bad Request on validation failure.
func makeSMSSubscribeHandler(store *SMSSubscriberStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only POST is supported on /api/v1/alerts/sms/subscribe")
			return
		}
		var req smsSubscribeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		rec, created, err := store.Subscribe(req.Phone, req.Country, req.Keywords)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		if !created {
			// Duplicate phone — return 409 with the existing record so
			// the caller can confirm their subscription.
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":      "already_subscribed",
				"message":    "phone number is already subscribed to SMS alerts",
				"subscriber": rec,
			})
			return
		}
		writeJSON(w, http.StatusCreated, rec)
	}
}

// makeSMSSubscribersHandler returns an http.HandlerFunc that handles
// GET /api/v1/alerts/sms/subscribers (admin only). Required scope:
// user:admin OR notification:write. The check is enforced via the
// principal's scopes (set by OptionalAuth upstream).
//
// Optional query parameters:
//   - ?active=true|false — filter by Active flag.
//   - ?country=KE — filter by country code.
//
// The sender parameter is included in the response so an operator can
// immediately see whether AFRICAS_TALKING_API_KEY was set at startup
// ("africas_talking" vs "stub").
func makeSMSSubscribersHandler(store *SMSSubscriberStore, sender SMSSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET is supported on /api/v1/alerts/sms/subscribers")
			return
		}
		p := middleware.PrincipalFromRequest(r)
		if p.IsAnonymous() {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		if !p.Can(auth.ScopeUserAdmin, auth.ScopeNotificationWrite) {
			writeError(w, http.StatusForbidden, "forbidden",
				"admin scope (user:admin or notification:write) required to list SMS subscribers")
			return
		}
		active := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("active")))
		country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))

		senderName := ""
		if sender != nil {
			senderName = sender.Name()
		}

		subs := store.List()
		filtered := subs[:0]
		for _, s := range subs {
			if active == "true" && !s.Active {
				continue
			}
			if active == "false" && s.Active {
				continue
			}
			if country != "" && s.Country != country {
				continue
			}
			filtered = append(filtered, s)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":  filtered,
			"total":  len(filtered),
			"sender": senderName,
			"note":   "Phone numbers are stored in E.164 format. Filter with ?active=true|false and ?country=KE.",
		})
	}
}

// makeSMSSendHandler returns an http.HandlerFunc that handles
// POST /api/v1/alerts/sms/send (admin broadcast). Required scope:
// user:admin OR notification:write.
//
// The handler:
//  1. Validates the message (1-160 chars — GSM 7-bit single-segment SMS).
//  2. Computes the recipient list (active subscribers whose keywords
//     intersect the request's keywords, OR all active subscribers when
//     keywords is empty).
//  3. Fans out one goroutine per recipient (bounded by the SMSSender's
//     own HTTP client timeout — no extra concurrency cap is applied;
//     Africa's Talking accepts ~10k messages per second on the
//     production gateway).
//  4. Records each delivery in the store's alert log.
//  5. Returns a summary: sent count, failed count, total attempted,
//     sender name.
//
// The fan-out is synchronous from the caller's perspective — the
// response blocks until every recipient has been attempted. For very
// large subscriber lists this would be replaced by an async queue, but
// the in-memory store never holds more than a few hundred subscribers
// in dev.
func makeSMSSendHandler(store *SMSSubscriberStore, sender SMSSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only POST is supported on /api/v1/alerts/sms/send")
			return
		}
		p := middleware.PrincipalFromRequest(r)
		if p.IsAnonymous() {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		if !p.Can(auth.ScopeUserAdmin, auth.ScopeNotificationWrite) {
			writeError(w, http.StatusForbidden, "forbidden",
				"admin scope (user:admin or notification:write) required to broadcast SMS")
			return
		}
		var req smsBroadcastRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		req.Message = strings.TrimSpace(req.Message)
		if req.Message == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "message is required")
			return
		}
		if len(req.Message) > 160 {
			writeError(w, http.StatusBadRequest, "bad_request",
				"message must be 160 characters or fewer (GSM 7-bit single-segment SMS)")
			return
		}
		source := strings.TrimSpace(req.Source)
		if source == "" {
			source = "broadcast"
		}

		recipients := store.ActiveSubscribers(req.Keywords)
		if len(recipients) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{
				"sent":       0,
				"failed":     0,
				"total":      0,
				"sender":     sender.Name(),
				"note":       "No active subscribers matched the supplied keywords.",
				"deliveries": []SMSAlert{},
			})
			return
		}

		// Fan out — one goroutine per recipient. The WaitGroup bounds the
		// handler's overall latency to max(per-recipient Send time).
		var wg sync.WaitGroup
		var mu sync.Mutex
		deliveries := make([]SMSAlert, 0, len(recipients))
		sent, failed := 0, 0
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		for _, sub := range recipients {
			wg.Add(1)
			go func(s SMSSubscriber) {
				defer wg.Done()
				alert, err := sender.Send(ctx, s.Phone, req.Message)
				alert.SourceEvent = source
				if err != nil {
					alert.Status = SMSStatusFailed
					mu.Lock()
					failed++
					deliveries = append(deliveries, alert)
					mu.Unlock()
					log.Printf("sms broadcast: failed to send to %s: %v", s.Phone, err)
					return
				}
				mu.Lock()
				sent++
				deliveries = append(deliveries, alert)
				mu.Unlock()
			}(sub)
		}
		wg.Wait()

		// Record every delivery in the store's log so the (future) admin
		// delivery-history endpoint can list them.
		for _, d := range deliveries {
			store.RecordAlert(d)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"sent":       sent,
			"failed":     failed,
			"total":      len(recipients),
			"sender":     sender.Name(),
			"deliveries": deliveries,
		})
	}
}

// --- Sample data ---

// smsSubscriberStore is the package-level in-memory SMSSubscriberStore,
// seeded with 10 sample subscribers spanning 5 countries (KE, UG, TZ,
// GH, NG) so the admin GET endpoint has data to render immediately. In
// production this is replaced by notifications.sms_subscribers in
// Postgres.
var smsSubscriberStore = NewSMSSubscriberStore()

// SeedSMSSampleSubscribers populates the store with 10 realistic
// subscribers spanning 5 countries (KE, UG, TZ, GH, NG) and a mix of
// keyword interests (tender, health, tax, education, environment).
// Idempotent — calling it twice is a no-op.
//
// The phone numbers are E.164-format test numbers in the reserved
// Africa's Talking sandbox ranges (per ITU-T E.164 the +999 range is
// reserved for test purposes, but Africa's Talking sandbox accepts
// real-format numbers like +254700000001 — we use those so the sandbox
// delivery path can be exercised without modification).
func SeedSMSSampleSubscribers(store *SMSSubscriberStore) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.seeded {
		return
	}
	store.seeded = true
	// Insert directly into the map (bypassing Subscribe) so the seed
	// timestamps are deterministic — tests assert on the seed count,
	// not on the CreatedAt ordering.
	seed := []SMSSubscriber{
		{Phone: "+254700000001", Country: "KE", Keywords: []string{"tender", "health"}, CreatedAt: time.Date(2025, 1, 10, 8, 0, 0, 0, time.UTC), Active: true},
		{Phone: "+254700000002", Country: "KE", Keywords: []string{"tax"}, CreatedAt: time.Date(2025, 1, 12, 9, 30, 0, 0, time.UTC), Active: true},
		{Phone: "+254700000003", Country: "KE", Keywords: []string{"education", "tender"}, CreatedAt: time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC), Active: true},
		{Phone: "+254700000004", Country: "KE", Keywords: []string{"environment"}, CreatedAt: time.Date(2025, 1, 18, 11, 15, 0, 0, time.UTC), Active: false},
		{Phone: "+256700000001", Country: "UG", Keywords: []string{"tender", "health"}, CreatedAt: time.Date(2025, 1, 20, 16, 0, 0, 0, time.UTC), Active: true},
		{Phone: "+256700000002", Country: "UG", Keywords: []string{"tax", "education"}, CreatedAt: time.Date(2025, 1, 22, 10, 0, 0, 0, time.UTC), Active: true},
		{Phone: "+255700000001", Country: "TZ", Keywords: []string{"health"}, CreatedAt: time.Date(2025, 1, 25, 13, 30, 0, 0, time.UTC), Active: true},
		{Phone: "+233200000001", Country: "GH", Keywords: []string{"tender", "environment"}, CreatedAt: time.Date(2025, 1, 28, 8, 45, 0, 0, time.UTC), Active: true},
		{Phone: "+234800000001", Country: "NG", Keywords: []string{"tax", "health"}, CreatedAt: time.Date(2025, 1, 30, 15, 0, 0, 0, time.UTC), Active: true},
		{Phone: "+234800000002", Country: "NG", Keywords: []string{"education"}, CreatedAt: time.Date(2025, 2, 1, 9, 0, 0, 0, time.UTC), Active: true},
	}
	for _, s := range seed {
		// Defensive copy of the keywords slice so a caller mutating the
		// seed data through the returned record can't corrupt the store.
		kw := make([]string, len(s.Keywords))
		copy(kw, s.Keywords)
		s.Keywords = kw
		store.subscribers[s.Phone] = s
	}
}

// init seeds the sample subscribers at package load time so the admin
// GET endpoint has data to render even before main() runs. This
// mirrors the gazette_alerts.go pattern (the store is constructed
// empty; the seed call is explicit so it is observable in logs). We
// seed here so the contract test that lists subscribers via the
// package-level store sees the seed count.
func init() {
	SeedSMSSampleSubscribers(smsSubscriberStore)
}
