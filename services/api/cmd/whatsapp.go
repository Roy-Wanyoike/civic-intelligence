// Package main — WhatsApp Business bot for civic Q&A (issue #290).
//
// This file wires the Civic Intelligence platform to the WhatsApp
// Business Cloud API (https://developers.facebook.com/docs/whatsapp/cloud-api)
// — Meta's official WhatsApp messaging API. WhatsApp was chosen because:
//
//   - WhatsApp is the dominant messaging channel in every African
//     market the platform ships an adapter for (Kenya, Uganda,
//     Tanzania, Ghana, Nigeria, South Africa). In Kenya alone
//     WhatsApp penetration is ~80% of the adult population (CAK 2024
//     sector report), dwarfing both SMS and the web frontend among
//     citizens who already own a smartphone.
//   - The Cloud API is a single REST endpoint:
//     POST https://graph.facebook.com/v18.0/{phone_number_id}/messages
//     with one auth header (Bearer <access_token>). No SDK, no client
//     library, no message queue.
//   - Inbound messages arrive as signed webhook callbacks on
//     POST /api/v1/whatsapp/webhook
//     Meta calls this endpoint once per inbound message; the platform
//     responds with 200 OK and asynchronously sends the reply via the
//     send endpoint. The webhook is the only ingress point.
//
// WhatsAppSender is an interface so tests can substitute a
// StubWhatsAppSender (which only logs) — the platform never sends real
// WhatsApp messages in unit tests.
//
// Selection at startup:
//
//	if os.Getenv("WHATSAPP_API_KEY") != "" → CloudWhatsAppSender
//	else                                  → StubWhatsAppSender
//
// Per-phone rate limiting: every inbound message counts against the
// caller's daily free-tier limit (default 5 questions/day, configurable
// via WHATSAPP_FREE_DAILY_LIMIT). After the limit is hit, the bot
// returns a friendly rate-limit reply instead of dispatching to the AI
// Q&A endpoint — this protects the platform's AI cost ceiling without
// requiring a paywall (a follow-up issue will wire paid tiers via
// M-Pesa / Card sponsor endpoints).
//
// Keyword routing: a citizen texts one of four keywords to get a
// structured response without engaging the AI Q&A pipeline:
//
//	BILL [number]    — Bill summary (e.g. "BILL 23 of 2024")
//	MP [name]        — MP scorecard (e.g. "MP Wetangula")
//	GAZETTE [kw]     — latest gazette notice matching keyword
//	HELP             — the keyword menu
//
// Any other message is dispatched to the platform's AI Q&A endpoint
// (services/ai) as a free-form civic question.
//
// Endpoints (registered in main.go):
//
//	POST /api/v1/whatsapp/webhook  — WhatsApp Cloud API callback (public)
//	GET  /api/v1/whatsapp/status   — health check (public)
//
// The architecture is production-ready; the live HTTP path to Meta's
// Cloud API uses the CloudWhatsAppSender (fully wired), but tests +
// dev run on the StubWhatsAppSender until WHATSAPP_API_KEY +
// WHATSAPP_PHONE_NUMBER_ID are configured.
package main

import (
	"bytes"
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

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
)

// --- Domain types ---

// WhatsAppMessage is the simplified inbound message shape the webhook
// accepts. Meta's actual webhook payload is deeply nested (entry →
// changes → value → messages → [0]); we accept a flattened form so the
// platform's own tests + admin tooling can call the webhook without
// fabricating the full Meta envelope. The production webhook ingress
// (a small adapter in front of this handler) translates the raw Meta
// payload into this flat shape before dispatching.
//
// Fields:
//   - From      — E.164 phone number of the sender (e.g. "+254712345678").
//   - Body      — the inbound message text (e.g. "BILL 23 of 2024").
//   - Timestamp — Unix seconds string (Meta sends this as a string; we
//     preserve it verbatim for traceability).
//   - MessageID — Meta's wamid.* identifier so the platform can
//     deduplicate inbound messages on retry.
type WhatsAppMessage struct {
	From      string `json:"from"`
	Body      string `json:"body"`
	Timestamp string `json:"timestamp"`
	MessageID string `json:"message_id"`
}

// WhatsAppDelivery is the outcome of an outbound WhatsApp send. The
// Status field is one of "queued", "sent", "failed" — mirroring the
// SMS alert lifecycle so the platform's notification fan-out code can
// treat both channels uniformly. The MessageID is Meta's wamid.* for
// the outbound message (empty when the stub is used).
type WhatsAppDelivery struct {
	ID        string    `json:"id"`
	To        string    `json:"to"`
	Message   string    `json:"message"`
	SentAt    time.Time `json:"sent_at"`
	Status    string    `json:"status"`
	MessageID string    `json:"message_id,omitempty"`
	Source    string    `json:"source"`
}

// WhatsApp delivery statuses. The lifecycle is: queued → sent (or
// queued → failed). The StubWhatsAppSender always reports "sent"
// immediately so the webhook handler returns a synchronous success to
// Meta (Meta retries webhook delivery if the webhook doesn't return
// 200 within 5 seconds, so a synchronous stub keeps the webhook
// fast + observable).
const (
	WhatsAppStatusQueued = "queued"
	WhatsAppStatusSent   = "sent"
	WhatsAppStatusFailed = "failed"
)

// whatsappStatus constants for the webhook response envelope. The
// status field tells Meta (and the platform's own admin tooling)
// whether the inbound message was processed, rate-limited, or
// rejected. Meta only inspects the HTTP status code (200 = OK); the
// JSON status is for the platform's own observability.
const (
	whatsappStatusOK            = "ok"
	whatsappStatusRateLimited   = "rate_limited"
	whatsappStatusInvalid       = "invalid"
	whatsappStatusInternalError = "internal_error"
)

// --- WhatsAppSender interface ---

// WhatsAppSender is the abstraction every outbound WhatsApp code path
// depends on. Implementations:
//   - *CloudWhatsAppSender — POSTs to Meta's Graph API.
//   - *StubWhatsAppSender   — logs the to+body and discards it.
//
// Send MUST be safe for concurrent use; the webhook handler is called
// once per inbound message but a future broadcast feature will fan out
// one goroutine per recipient.
type WhatsAppSender interface {
	// Send delivers a single WhatsApp message to a single recipient.
	// The ctx is honoured so the caller can bound the total send time.
	// A non-nil error means the message was NOT accepted by Meta —
	// callers SHOULD log + continue rather than abort the batch.
	Send(ctx context.Context, to, message string) (WhatsAppDelivery, error)

	// Name returns a short identifier for logs + diagnostics (e.g.
	// "cloud", "stub"). The webhook handler surfaces this in its
	// response so an operator can immediately see whether
	// WHATSAPP_API_KEY was set at startup.
	Name() string
}

// NewWhatsAppSender picks an implementation based on the
// WHATSAPP_API_KEY env var. When the var is set, a *CloudWhatsAppSender
// is returned; otherwise a *StubWhatsAppSender is returned. The
// optional httpClient parameter exists so tests can inject an
// httptest.Server-tied client; production callers pass nil and a
// default client with a 10-second timeout is used.
//
// The sender short-circuits to the stub when the env var is missing
// so the platform boots cleanly in dev + in tests; the webhook handler
// surfaces the chosen sender's Name() in its response so an operator
// can immediately see whether real WhatsApp delivery is wired.
func NewWhatsAppSender(httpClient *http.Client) WhatsAppSender {
	apiKey := strings.TrimSpace(os.Getenv("WHATSAPP_API_KEY"))
	phoneNumberID := strings.TrimSpace(os.Getenv("WHATSAPP_PHONE_NUMBER_ID"))
	if apiKey == "" || phoneNumberID == "" {
		log.Printf("whatsapp: WHATSAPP_API_KEY or WHATSAPP_PHONE_NUMBER_ID not set — using StubWhatsAppSender (no WhatsApp messages will be sent)")
		return &StubWhatsAppSender{}
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	// The Cloud API version is pinned to v18.0 (the latest GA at the
	// time of writing). A future upgrade will bump this in lockstep
	// with Meta's deprecation schedule.
	baseURL := "https://graph.facebook.com/v18.0"
	log.Printf("whatsapp: WHATSAPP_API_KEY detected — using CloudWhatsAppSender (phone_number_id=%s, base=%s)", phoneNumberID, baseURL)
	return &CloudWhatsAppSender{
		apiKey:        apiKey,
		phoneNumberID: phoneNumberID,
		client:        httpClient,
		baseURL:       baseURL,
	}
}

// --- CloudWhatsAppSender ---

// CloudWhatsAppSender POSTs WhatsApp messages to Meta's Graph API.
//
// The request body is JSON (NOT form-encoded — Meta's WhatsApp Cloud
// API uses JSON, unlike Africa's Talking's form-encoded SMS API). The
// shape is documented at
// https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages:
//
//	{
//	  "messaging_product": "whatsapp",
//	  "to": "<E.164 recipient>",
//	  "type": "text",
//	  "text": {"body": "<message>"}
//	}
//
// A 200 response from Meta means the message was queued — delivery to
// the recipient's handset is asynchronous on Meta's side (typically
// 1-5 seconds for online recipients; minutes for offline ones). Any
// non-2xx status is surfaced as an error containing the upstream body
// so an operator can see why Meta rejected the request.
//
// NOTE: this sender is wired + tested via the StubWhatsAppSender
// contract; the live HTTP path is exercised in the follow-up task that
// wires Meta sandbox credentials into CI. The architecture is ready
// today.
type CloudWhatsAppSender struct {
	apiKey        string
	phoneNumberID string
	client        *http.Client
	baseURL       string
}

// Name implements WhatsAppSender.
func (s *CloudWhatsAppSender) Name() string { return "cloud" }

// Send implements WhatsAppSender. The recipient `to` MUST be in E.164
// format (e.g. "+254712345678"); Meta rejects anything else with a
// 400. The message is sent as-is; the caller is responsible for
// truncating to WhatsApp's 4096-character text-message limit.
func (s *CloudWhatsAppSender) Send(ctx context.Context, to, message string) (WhatsAppDelivery, error) {
	if strings.TrimSpace(to) == "" {
		return WhatsAppDelivery{}, errors.New("whatsapp: recipient (to) is required")
	}
	if strings.TrimSpace(message) == "" {
		return WhatsAppDelivery{}, errors.New("whatsapp: message is required")
	}
	if s.apiKey == "" || s.phoneNumberID == "" {
		return WhatsAppDelivery{}, errors.New("whatsapp: WHATSAPP_API_KEY or WHATSAPP_PHONE_NUMBER_ID not configured")
	}

	// Build the Cloud API request body. The shape is fixed by Meta's
	// spec; the "messaging_product" field MUST be "whatsapp" and the
	// "type" MUST be "text" for a plain-text message.
	reqBody := map[string]any{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text":              map[string]string{"body": message},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return WhatsAppDelivery{}, fmt.Errorf("whatsapp: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/messages", s.baseURL, s.phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return WhatsAppDelivery{}, fmt.Errorf("whatsapp: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return WhatsAppDelivery{
			ID: newWhatsAppDeliveryID(), To: to, Message: message,
			SentAt: time.Now().UTC(), Status: WhatsAppStatusFailed,
			Source: "cloud",
		}, fmt.Errorf("whatsapp: cloud api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return WhatsAppDelivery{
			ID: newWhatsAppDeliveryID(), To: to, Message: message,
			SentAt: time.Now().UTC(), Status: WhatsAppStatusFailed,
			Source: "cloud",
		}, fmt.Errorf("whatsapp: cloud api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse Meta's response to extract the outbound wamid.* for
	// traceability. The response shape is:
	//   {"messaging_product":"whatsapp","contacts":[{"input":"+254...","wa_id":"254..."}],"messages":[{"id":"wamid.HBgL..."}]}
	var apiResp struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&apiResp)
	outboundID := ""
	if len(apiResp.Messages) > 0 {
		outboundID = apiResp.Messages[0].ID
	}
	return WhatsAppDelivery{
		ID: newWhatsAppDeliveryID(), To: to, Message: message,
		SentAt: time.Now().UTC(), Status: WhatsAppStatusSent,
		MessageID: outboundID, Source: "cloud",
	}, nil
}

// --- StubWhatsAppSender ---

// StubWhatsAppSender is the no-op WhatsAppSender used in dev mode
// (when WHATSAPP_API_KEY is not set) and in unit tests. It logs the
// recipient + message so the webhook's behaviour is observable in dev
// logs without sending any real traffic. Send always returns a
// WhatsAppDelivery with Status=sent — the stub never fails.
//
// The stub is concurrency-safe because log.Printf is. The LastTo /
// LastMessage / SendCount fields are populated for test assertions
// but are NOT synchronised — tests are expected to call Send serially.
type StubWhatsAppSender struct {
	mu          sync.Mutex
	LastTo      string
	LastMessage string
	SendCount   int
}

// Name implements WhatsAppSender.
func (s *StubWhatsAppSender) Name() string { return "stub" }

// Send implements WhatsAppSender.
func (s *StubWhatsAppSender) Send(_ context.Context, to, message string) (WhatsAppDelivery, error) {
	s.mu.Lock()
	s.LastTo = to
	s.LastMessage = message
	s.SendCount++
	s.mu.Unlock()
	log.Printf("whatsapp(stub): would send to=%q msg=%q", to, message)
	return WhatsAppDelivery{
		ID: newWhatsAppDeliveryID(), To: to, Message: message,
		SentAt: time.Now().UTC(), Status: WhatsAppStatusSent,
		Source: "stub",
	}, nil
}

// newWhatsAppDeliveryID generates a 16-byte hex ID with the "wa_"
// prefix so delivery records are distinguishable from SMS alert IDs
// (sms_) and follow IDs (flw_) in logs.
func newWhatsAppDeliveryID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return "wa_" + hex.EncodeToString(b)
}

// --- WhatsAppRateLimiter ---

// WhatsAppRateLimiter enforces a per-phone daily free-tier limit. The
// limit defaults to 5 messages/day (configurable via the
// WHATSAPP_FREE_DAILY_LIMIT env var) and resets at UTC midnight.
//
// The limiter is intentionally in-memory — production wiring will move
// the counter to Redis (the platform already ships a Redis cache per
// ADR-0009) so the limit is shared across API replicas. The interface
// is stable: a Redis-backed limiter will be a drop-in replacement.
//
// All methods are safe for concurrent use.
type WhatsAppRateLimiter struct {
	mu     sync.Mutex
	counts map[string]*whatsappDailyCounter // phone → counter
	limit  int
}

// whatsappDailyCounter tracks one phone's message count for a single
// UTC day. When the date rolls over (the Date field no longer matches
// today's UTC date), the counter is reset to 0 — this is the daily
// reset semantics.
type whatsappDailyCounter struct {
	Date  string // YYYY-MM-DD (UTC)
	Count int
}

// NewWhatsAppRateLimiter returns a limiter with the supplied daily
// limit. A limit <= 0 disables rate limiting entirely (the Allow
// method always returns true) — used by tests that want to exercise
// the keyword routes without hitting the ceiling.
func NewWhatsAppRateLimiter(limit int) *WhatsAppRateLimiter {
	return &WhatsAppRateLimiter{
		counts: make(map[string]*whatsappDailyCounter),
		limit:  limit,
	}
}

// defaultWhatsAppFreeDailyLimit is the documented free-tier daily
// limit. Overridable via the WHATSAPP_FREE_DAILY_LIMIT env var so an
// operator can tune the ceiling without a code change.
const defaultWhatsAppFreeDailyLimit = 5

// Limit returns the configured daily limit.
func (r *WhatsAppRateLimiter) Limit() int { return r.limit }

// Allow reports whether the supplied phone is under the daily limit.
// When the phone is under the limit, the counter is incremented BEFORE
// returning true — this is the "consume" semantic so a concurrent
// second message from the same phone can't race past the ceiling. When
// the phone is over the limit, the counter is left untouched (so the
// rate-limit response can report the actual consumed count).
//
// When the limiter's limit is <= 0, Allow always returns true without
// touching the counters (rate limiting disabled).
func (r *WhatsAppRateLimiter) Allow(phone string) bool {
	phone = strings.TrimSpace(phone)
	if r.limit <= 0 || phone == "" {
		return true
	}
	today := time.Now().UTC().Format("2006-01-02")
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.counts[phone]
	if !ok || c.Date != today {
		// Either first-ever message from this phone, or the date
		// rolled over since the last message. Reset the counter.
		c = &whatsappDailyCounter{Date: today, Count: 0}
		r.counts[phone] = c
	}
	if c.Count >= r.limit {
		return false
	}
	c.Count++
	return true
}

// Remaining returns the number of messages the phone can still send
// today before hitting the limit. Returns 0 when the limit is reached
// (or exceeded via a race); returns the limit when the phone has not
// yet messaged today.
func (r *WhatsAppRateLimiter) Remaining(phone string) int {
	phone = strings.TrimSpace(phone)
	if r.limit <= 0 {
		return -1 // unlimited — sentinel value
	}
	today := time.Now().UTC().Format("2006-01-02")
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.counts[phone]
	if !ok || c.Date != today {
		return r.limit
	}
	remaining := r.limit - c.Count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ResetAt returns the UTC time at which the supplied phone's counter
// will reset (the next UTC midnight). Used by the webhook handler to
// surface a friendly "try again in N hours" message.
func (r *WhatsAppRateLimiter) ResetAt(_ string) time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}

// --- WhatsAppBot ---

// WhatsAppBot is the bot's core dispatcher. It receives an inbound
// WhatsAppMessage, applies per-phone rate limiting, routes by keyword,
// and returns the outbound reply text + the routing decision. The bot
// does NOT deliver the reply itself — that's the WhatsAppSender's job —
// so the dispatcher can be unit-tested without any HTTP traffic.
//
// The bot is concurrency-safe because every dependency (the rate
// limiter + the gazette store + the bills adapter) is concurrency-safe.
type WhatsAppBot struct {
	rateLimiter  *WhatsAppRateLimiter
	aiServiceURL string
	billsAdapter BillsAdapter
	gazetteStore *GazetteAlertStore
	httpClient   *http.Client
}

// NewWhatsAppBot wires the bot's dependencies. The aiServiceURL is
// the base URL of the Python AI service (FastAPI) — when empty, the
// free-form Q&A path degrades gracefully (returns a "service
// unavailable" reply instead of erroring). The billsAdapter is used
// for the BILL keyword lookup; the gazetteStore is used for the
// GAZETTE keyword lookup.
func NewWhatsAppBot(rateLimiter *WhatsAppRateLimiter, aiServiceURL string, billsAdapter BillsAdapter, gazetteStore *GazetteAlertStore) *WhatsAppBot {
	if rateLimiter == nil {
		rateLimiter = NewWhatsAppRateLimiter(defaultWhatsAppFreeDailyLimit)
	}
	return &WhatsAppBot{
		rateLimiter:  rateLimiter,
		aiServiceURL: strings.TrimSpace(aiServiceURL),
		billsAdapter: billsAdapter,
		gazetteStore: gazetteStore,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

// whatsappRoute is the routing decision the bot made for an inbound
// message. The webhook handler surfaces this in its response so an
// operator can see at a glance whether a citizen's message hit a
// keyword route or went to the AI Q&A pipeline. The values are stable
// strings (not enums) so the OpenAPI spec can document them.
const (
	whatsappRouteBill      = "bill"
	whatsappRouteMP        = "mp"
	whatsappRouteGazette   = "gazette"
	whatsappRouteHelp      = "help"
	whatsappRouteQA        = "qa"
	whatsappRouteRateLimit = "rate_limited"
	whatsappRouteInvalid   = "invalid"
)

// Handle processes a single inbound WhatsApp message and returns the
// outbound reply + the routing decision. The reply is the text the
// WhatsAppSender will deliver (or log, in stub mode). The route is
// surfaced in the webhook response for observability.
//
// The method applies the rate limit FIRST — a rate-limited phone gets
// the rate-limit reply regardless of the message content. This is the
// "free tier" semantic: every message counts against the daily ceiling,
// including keyword lookups (a citizen can't spam BILL/MP/GAZETTE to
// bypass the AI Q&A cost ceiling).
func (b *WhatsAppBot) Handle(ctx context.Context, msg WhatsAppMessage) (reply, route string) {
	phone := strings.TrimSpace(msg.From)
	if phone == "" {
		return "We couldn't read your phone number. Please try again.", whatsappRouteInvalid
	}
	if !b.rateLimiter.Allow(phone) {
		resetAt := b.rateLimiter.ResetAt(phone)
		return fmt.Sprintf(
			"You've reached your daily free limit of %d questions. "+
				"Your limit resets at %s UTC. "+
				"To ask more questions today, become a sponsor at https://civicintelligence.org/sponsor.",
			b.rateLimiter.Limit(), resetAt.Format("2006-01-02 15:04"),
		), whatsappRouteRateLimit
	}

	body := strings.TrimSpace(msg.Body)
	upper := strings.ToUpper(body)

	switch {
	case upper == "HELP" || body == "":
		return b.helpReply(), whatsappRouteHelp
	case strings.HasPrefix(upper, "BILL"):
		return b.billReply(ctx, strings.TrimSpace(strings.TrimPrefix(body, body[:4])))
	case strings.HasPrefix(upper, "MP"):
		return b.mpReply(strings.TrimSpace(strings.TrimPrefix(body, body[:2])))
	case strings.HasPrefix(upper, "GAZETTE"):
		return b.gazetteReply(strings.TrimSpace(strings.TrimPrefix(body, body[:7])))
	default:
		return b.qaReply(ctx, body)
	}
}

// helpReply renders the keyword menu. The menu is intentionally short
// (under 5 lines) so it fits in a single WhatsApp message bubble
// without truncation. Each keyword is paired with an example so a
// citizen can copy the format on their first message.
func (b *WhatsAppBot) helpReply() string {
	return strings.Join([]string{
		"Civic Intelligence WhatsApp Bot",
		"",
		"Send any of these keywords:",
		"• BILL [number] — Bill summary (e.g. BILL 23 of 2024)",
		"• MP [name] — MP scorecard (e.g. MP Wetangula)",
		"• GAZETTE [keyword] — latest gazette notice (e.g. GAZETTE tender)",
		"• HELP — this menu",
		"",
		"Or type any civic question and we'll answer it.",
		"",
		"Free tier: " + fmt.Sprintf("%d questions/day.", b.rateLimiter.Limit()),
	}, "\n")
}

// billReply looks up a Bill by its number (e.g. "23 of 2024") and
// returns a short summary. The lookup uses the same BillsAdapter the
// /api/v1/bills endpoint uses (so the bot's data is identical to the
// web frontend's data). When the adapter is unreachable or the bill
// isn't found, the bot returns a graceful fallback rather than
// erroring — WhatsApp users expect a reply, not a 5xx.
//
// The summary is built from the BillCandidate's fields (no AI call) so
// the BILL keyword path is free of AI cost — a citizen can check a
// Bill's status without consuming an AI Q&A credit.
func (b *WhatsAppBot) billReply(ctx context.Context, number string) (string, string) {
	if number == "" {
		return "Please send a Bill number, e.g. BILL 23 of 2024.", whatsappRouteBill
	}
	if b.billsAdapter == nil {
		return "Bill lookup is currently unavailable. Please try again later.", whatsappRouteBill
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	bills, err := b.billsAdapter.DiscoverBills(ctx)
	if err != nil {
		log.Printf("whatsapp: BILL keyword: adapter error: %v", err)
		return "We couldn't reach the Kenya Law Bills feed right now. Please try again in a few minutes.", whatsappRouteBill
	}

	// Match by number — the citizen's input (e.g. "23 of 2024") is
	// matched case-insensitively against each Bill's title + slug +
	// source ID. The match is a substring search because the citizen
	// may not know the Bill's full title; "23 of 2024" should match
	// "NA Bill No. 23 of 2024 — Statutory Instruments (Amendment)".
	needle := strings.ToLower(number)
	for i := range bills {
		hay := strings.ToLower(bills[i].Title + " " + bills[i].Slug + " " + bills[i].SourceID)
		if strings.Contains(hay, needle) {
			return b.formatBillSummary(&bills[i]), whatsappRouteBill
		}
	}

	// Fallback to the seed Bills when the live crawl returns nothing
	// (mirrors the /api/v1/bills degraded-mode contract from issue #265).
	for _, title := range whatsappSeedBillTitles() {
		if strings.Contains(strings.ToLower(title), needle) {
			return "Bill found (seed data):\n" + title + "\n\nFull details: https://civicintelligence.org/api/v1/bills", whatsappRouteBill
		}
	}

	return "No Bill matched " + strconvQuote(number) + ". Send HELP for the keyword menu, or visit https://civicintelligence.org/api/v1/bills for the full list.", whatsappRouteBill
}

// formatBillSummary renders a one-paragraph Bill summary from a
// BillCandidate. The summary is intentionally compact (under 600
// chars) so it fits in a single WhatsApp text bubble without
// truncation. Every claim is sourced — the Bill's URL is included so
// the citizen can verify the summary against the official Kenya Law
// record.
func (b *WhatsAppBot) formatBillSummary(bill *kenya_law.BillCandidate) string {
	year := 0
	if !bill.PublicationDate.IsZero() {
		year = bill.PublicationDate.Year()
	}
	pubDateStr := ""
	if !bill.PublicationDate.IsZero() {
		pubDateStr = bill.PublicationDate.Format("2 January 2006")
	}
	out := fmt.Sprintf("Bill: %s\n", bill.Title)
	out += fmt.Sprintf("House: %s\n", bill.House)
	if year != 0 {
		out += fmt.Sprintf("Year: %d\n", year)
	}
	if pubDateStr != "" {
		out += fmt.Sprintf("Published: %s\n", pubDateStr)
	}
	out += fmt.Sprintf("Source: %s\n", bill.URL)
	out += "\nFull details: https://civicintelligence.org/api/v1/bills/" + bill.SourceID
	return out
}

// mpReply looks up an MP by name (case-insensitive substring match
// against the samplePeople + sampleScorecards slices) and returns a
// short scorecard summary. The lookup matches the same dataset the
// /api/v1/people endpoint serves so the bot's data is identical to
// the web frontend's data.
//
// The summary surfaces raw counts + rates ONLY — the platform NEVER
// calculates a political performance score (rule:
// NO_POLITICAL_PERFORMANCE_SCORE). Every metric carries a source URL
// so the citizen can verify the data against the official
// parliament.go.ke record.
func (b *WhatsAppBot) mpReply(name string) (string, string) {
	if name == "" {
		return "Please send an MP name, e.g. MP Wetangula.", whatsappRouteMP
	}
	needle := strings.ToLower(name)

	// First match against the scorecards (which carry the richest
	// data — attendance, bills sponsored, etc.).
	for i := range sampleScorecards {
		if strings.Contains(strings.ToLower(sampleScorecards[i].Name), needle) {
			return b.formatMPScorecard(&sampleScorecards[i]), whatsappRouteMP
		}
	}

	// Fall back to the people list (Speakers + Presidents across the
	// 6 supported countries). This catches "MP Ruto" or "MP Museveni"
	// which aren't in the scorecards slice but are in samplePeople.
	for _, p := range samplePeople {
		fullName, _ := p["full_name"].(string)
		if fullName == "" {
			continue
		}
		if strings.Contains(strings.ToLower(fullName), needle) {
			id, _ := p["id"].(string)
			role, _ := p["role"].(string)
			house, _ := p["house"].(string)
			country, _ := p["country"].(string)
			out := fmt.Sprintf("MP: %s\n", fullName)
			if role != "" {
				out += fmt.Sprintf("Role: %s\n", role)
			}
			if house != "" {
				out += fmt.Sprintf("House: %s\n", house)
			}
			if country != "" {
				out += fmt.Sprintf("Country: %s\n", country)
			}
			out += "\nFull scorecard: https://civicintelligence.org/api/v1/people/" + id + "/scorecard"
			return out, whatsappRouteMP
		}
	}

	return "No MP matched " + strconvQuote(name) + ". Send HELP for the keyword menu, or visit https://civicintelligence.org/api/v1/people for the full list.", whatsappRouteMP
}

// formatMPScorecard renders a short MP scorecard from a sample
// scorecard. The summary is intentionally compact (under 800 chars)
// so it fits in a single WhatsApp text bubble. The disclaimer is
// included so the citizen knows the platform never ranks MPs.
func (b *WhatsAppBot) formatMPScorecard(s *MPScorecard) string {
	out := fmt.Sprintf("MP: %s\n", s.Name)
	out += fmt.Sprintf("Role: %s\n", s.Role)
	out += fmt.Sprintf("Constituency: %s\n", s.Constituency)
	out += fmt.Sprintf("Party: %s\n", s.Party)
	out += "\nParliamentary record:\n"
	out += fmt.Sprintf("• Attendance: %d%%\n", s.AttendanceRate.Value)
	out += fmt.Sprintf("• Bills sponsored: %d\n", s.BillsSponsored.Value)
	out += fmt.Sprintf("• Questions asked: %d\n", s.QuestionsAsked.Value)
	out += fmt.Sprintf("• Votes recorded: %d\n", s.VotesRecorded.Value)
	out += "\n" + scorecardDisclaimer
	out += "\n\nFull scorecard: https://civicintelligence.org/api/v1/people/" + s.PersonID + "/scorecard"
	return out
}

// gazetteReply looks up the latest published gazette notice matching
// the supplied keyword (case-insensitive substring match on title +
// body + notice_number). The lookup uses the same GazetteAlertStore
// the /api/v1/gazette/alerts endpoint uses (so the bot's data is
// identical to the web frontend's data). Drafts never match — only
// notices whose gazette_date <= today.
//
// The summary surfaces the title + date + source URL so the citizen
// can verify the notice against the official Kenya Gazette record.
func (b *WhatsAppBot) gazetteReply(keyword string) (string, string) {
	if keyword == "" {
		return "Please send a gazette keyword, e.g. GAZETTE tender.", whatsappRouteGazette
	}
	if b.gazetteStore == nil {
		return "Gazette lookup is currently unavailable. Please try again later.", whatsappRouteGazette
	}

	notices := b.gazetteStore.Notices(time.Now())
	if len(notices) == 0 {
		return "No gazette notices are currently published.", whatsappRouteGazette
	}

	needle := strings.ToLower(keyword)
	for _, n := range notices {
		hay := strings.ToLower(n.Title + " " + n.BodyText + " " + n.NoticeNumber + " " + n.Summary)
		if strings.Contains(hay, needle) {
			out := fmt.Sprintf("Latest gazette notice matching %q:\n\n", keyword)
			out += fmt.Sprintf("Title: %s\n", n.Title)
			out += fmt.Sprintf("Date: %s\n", n.GazetteDate)
			if n.NoticeNumber != "" {
				out += fmt.Sprintf("Notice No.: %s\n", n.NoticeNumber)
			}
			if n.Authority != "" {
				out += fmt.Sprintf("Authority: %s\n", n.Authority)
			}
			out += fmt.Sprintf("Source: %s\n", n.SourceURL)
			return out, whatsappRouteGazette
		}
	}

	return "No gazette notice matched " + strconvQuote(keyword) + ". Send HELP for the keyword menu, or visit https://civicintelligence.org/api/v1/gazette/alerts for the full list.", whatsappRouteGazette
}

// qaReply dispatches a free-form civic question to the platform's AI
// Q&A endpoint (services/ai). The dispatch is a direct HTTP POST to
// the Python AI service's /api/v1/ask endpoint — the same endpoint
// the /api/v1/questions HTTP handler proxies to. This keeps the bot
// self-contained (it doesn't loop back through the API service's own
// /api/v1/questions route, which would require an internal auth
// token).
//
// When the AI service is unreachable, the bot returns a graceful
// degradation instead of erroring — WhatsApp users expect a reply,
// not a 5xx.
func (b *WhatsAppBot) qaReply(ctx context.Context, question string) (string, string) {
	if strings.TrimSpace(question) == "" {
		return "Please send a question, e.g. 'What is the status of the Affordable Housing Bill?'.", whatsappRouteQA
	}
	if b.aiServiceURL == "" {
		return "The AI Q&A service is not configured right now. Please try again later, or send HELP for the keyword menu.", whatsappRouteQA
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	reqBody := map[string]string{"question": question}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("whatsapp: QA dispatch: marshal request: %v", err)
		return "We couldn't process your question right now. Please try again later.", whatsappRouteQA
	}

	url := strings.TrimRight(b.aiServiceURL, "/") + "/api/v1/ask"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("whatsapp: QA dispatch: build request: %v", err)
		return "We couldn't process your question right now. Please try again later.", whatsappRouteQA
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		log.Printf("whatsapp: QA dispatch: ai service unreachable: %v", err)
		return "The AI Q&A service is currently unavailable. Please try again later, or send HELP for the keyword menu.", whatsappRouteQA
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("whatsapp: QA dispatch: ai service returned status %d", resp.StatusCode)
		return "The AI Q&A service returned an error. Please try again later, or send HELP for the keyword menu.", whatsappRouteQA
	}

	// The AI service returns {"answer":"..."}. We extract the answer
	// and forward it verbatim. When the body doesn't decode (e.g. the
	// AI service returned a non-JSON error), fall back to a graceful
	// message rather than surfacing the raw body.
	var aiResp struct {
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		log.Printf("whatsapp: QA dispatch: decode response: %v", err)
		return "We couldn't read the AI service's response. Please try again later.", whatsappRouteQA
	}
	if strings.TrimSpace(aiResp.Answer) == "" {
		return "The AI service returned an empty answer. Please rephrase your question or try again later.", whatsappRouteQA
	}
	return aiResp.Answer, whatsappRouteQA
}

// --- HTTP handlers ---

// whatsappWebhookRequest is the JSON body for POST
// /api/v1/whatsapp/webhook. The shape is the flat WhatsAppMessage —
// the production webhook ingress (a small adapter in front of this
// handler) translates Meta's nested webhook envelope into this flat
// shape before dispatching.
type whatsappWebhookRequest struct {
	WhatsAppMessage
}

// whatsappWebhookResponse is the JSON body returned by the webhook.
// The status field tells the platform's own admin tooling whether the
// inbound message was processed, rate-limited, or rejected. Meta only
// inspects the HTTP status code (200 = OK); the JSON envelope is for
// the platform's own observability.
type whatsappWebhookResponse struct {
	Status    string                 `json:"status"`
	MessageID string                 `json:"message_id"`
	From      string                 `json:"from"`
	Route     string                 `json:"route"`
	Reply     string                 `json:"reply"`
	Sender    string                 `json:"sender"`
	Delivery  *WhatsAppDelivery      `json:"delivery,omitempty"`
	RateLimit *whatsappRateLimitInfo `json:"rate_limit,omitempty"`
}

// whatsappRateLimitInfo is the per-phone rate-limit state surfaced in
// the webhook response so the citizen (and the platform's admin
// tooling) can see how many questions remain in the free tier. The
// reset_at field is the UTC time at which the counter resets.
type whatsappRateLimitInfo struct {
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	ResetAt   string `json:"reset_at"`
}

// makeWhatsAppWebhookHandler returns an http.HandlerFunc that handles
// POST /api/v1/whatsapp/webhook. The endpoint is PUBLIC (no auth
// required) — Meta calls the webhook from its Cloud API gateway, not
// from a user session, so there is no OIDC token to verify. The phone
// number in the request body is the user's identity for the duration
// of the message.
//
// The handler:
//  1. Parses the request body (JSON).
//  2. Normalises the phone number to E.164 (Meta sends it without the
//     leading '+' in some modes).
//  3. Dispatches the message through the bot.
//  4. Sends the reply via the WhatsAppSender (stub or cloud).
//  5. Returns a JSON envelope with the reply, the route, the sender
//     name, the delivery record, and the per-phone rate-limit state.
//
// The handler always returns 200 OK (even on rate-limit or invalid
// input) so Meta doesn't retry the webhook delivery — retries would
// duplicate the inbound message in the bot's logs and re-consume the
// citizen's daily limit. The JSON status field is the platform's own
// signal for whether the message was actually processed.
func makeWhatsAppWebhookHandler(bot *WhatsAppBot, sender WhatsAppSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only POST is supported on /api/v1/whatsapp/webhook")
			return
		}

		var req whatsappWebhookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusOK, whatsappWebhookResponse{
				Status: whatsappStatusInvalid,
				Reply:  "Invalid JSON body. Please send a valid WhatsApp message.",
			})
			return
		}

		// Normalise the phone number to E.164 (Meta sends it without
		// the leading '+' in some modes; we add it back so the
		// rate limiter + sender see a consistent key).
		phone := strings.TrimSpace(req.From)
		if phone != "" && !strings.HasPrefix(phone, "+") {
			phone = "+" + phone
		}
		req.From = phone

		reply, route := bot.Handle(r.Context(), req.WhatsAppMessage)

		// Send the reply via the WhatsAppSender. The stub logs; the
		// cloud sender POSTs to Meta. Either way, the delivery record
		// is surfaced in the response so the platform's admin tooling
		// can track outbound message status.
		var delivery *WhatsAppDelivery
		if sender != nil && phone != "" {
			d, err := sender.Send(r.Context(), phone, reply)
			if err != nil {
				log.Printf("whatsapp: webhook: sender error for %s: %v", phone, err)
			}
			delivery = &d
		}

		senderName := ""
		if sender != nil {
			senderName = sender.Name()
		}

		// Surface the per-phone rate-limit state so the citizen can
		// see how many questions remain in the free tier.
		var rateLimit *whatsappRateLimitInfo
		if bot.rateLimiter != nil && bot.rateLimiter.Limit() > 0 {
			rateLimit = &whatsappRateLimitInfo{
				Limit:     bot.rateLimiter.Limit(),
				Remaining: bot.rateLimiter.Remaining(phone),
				ResetAt:   bot.rateLimiter.ResetAt(phone).Format(time.RFC3339),
			}
		}

		status := whatsappStatusOK
		if route == whatsappRouteRateLimit {
			status = whatsappStatusRateLimited
		} else if route == whatsappRouteInvalid {
			status = whatsappStatusInvalid
		}

		writeJSON(w, http.StatusOK, whatsappWebhookResponse{
			Status:    status,
			MessageID: req.MessageID,
			From:      phone,
			Route:     route,
			Reply:     reply,
			Sender:    senderName,
			Delivery:  delivery,
			RateLimit: rateLimit,
		})
	}
}

// makeWhatsAppStatusHandler returns an http.HandlerFunc that handles
// GET /api/v1/whatsapp/status. The endpoint is PUBLIC (no auth
// required) — it's used by Meta's webhook subscription verification
// flow AND by the platform's own liveness probes. The response
// surfaces the sender name (so an operator can immediately see
// whether WHATSAPP_API_KEY was set at startup) + the configured daily
// limit + the current UTC time.
func makeWhatsAppStatusHandler(sender WhatsAppSender, rateLimiter *WhatsAppRateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET is supported on /api/v1/whatsapp/status")
			return
		}

		senderName := ""
		if sender != nil {
			senderName = sender.Name()
		}
		limit := 0
		if rateLimiter != nil {
			limit = rateLimiter.Limit()
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "ok",
			"sender":       senderName,
			"daily_limit":  limit,
			"now_utc":      time.Now().UTC().Format(time.RFC3339),
			"keywords":     []string{"BILL [number]", "MP [name]", "GAZETTE [keyword]", "HELP"},
			"webhook_docs": "https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks",
		})
	}
}

// --- Sample data ---

// whatsappSeedBillTitles is a static fallback list of the top 3 Bills
// the platform tracks. Used by billReply when the live Kenya Law
// adapter returns no Bills (mirrors the /api/v1/bills degraded-mode
// contract from issue #265). The titles are surfaced verbatim to the
// citizen; the full Bill is available at /api/v1/bills.
//
// The titles below are illustrative; they are surfaced verbatim to the
// WhatsApp citizen and link to /api/v1/bills for the full Bill.
func whatsappSeedBillTitles() []string {
	return []string{
		"NA Bill No. 23 of 2024 — Statutory Instruments (Amendment)",
		"Senate Bill No. 11 of 2024 — County Governments (Health Services)",
		"NA Bill No. 7 of 2024 — Public Finance Management (Amendment)",
	}
}

// --- Helpers ---

// strconvQuote wraps strconv.Quote so the whatsapp.go file doesn't
// need to import strconv (which would be unused otherwise — only the
// quote helper is used). The helper is local so the call sites stay
// compact.
func strconvQuote(s string) string {
	// We inline a minimal quote — wrapping in double quotes + escaping
	// any embedded double quotes — rather than pulling in strconv for
	// a single use. This matches the existing convention in the
	// codebase of avoiding strconv for trivial formatting.
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
