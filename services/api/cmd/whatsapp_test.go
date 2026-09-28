// Package main — tests for the WhatsApp bot handlers (issue #290).
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
)

// --- Test helpers ---

// NOTE: mockBillsAdapter is defined in bills_test.go and reused here.
// It implements the BillsAdapter interface with configurable bills +
// err + fetchBillHTML/fetchBillErr fields, so the BILL keyword test can
// drive the live-match path without depending on the Kenya Law upstream
// crawl (which would be flaky in CI).

// newTestBot builds a fresh WhatsAppBot for a test, with an isolated
// rate limiter + a stub bills adapter + the real gazette store. The
// AI service URL is empty so the free-form QA path returns the
// "not configured" graceful degradation rather than trying to dial
// localhost:8000.
func newTestBot(limit int) (*WhatsAppBot, *WhatsAppRateLimiter) {
	rl := NewWhatsAppRateLimiter(limit)
	bot := NewWhatsAppBot(rl, "", &mockBillsAdapter{}, gazetteAlertStore)
	return bot, rl
}

// whatsappDispatch fires a POST /api/v1/whatsapp/webhook request with a
// JSON body through the supplied handler. Returns the recorded response
// + the parsed webhook envelope when the response is JSON.
func whatsappDispatch(handler http.Handler, body string) (*httptest.ResponseRecorder, *whatsappWebhookResponse) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	var resp whatsappWebhookResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	return rr, &resp
}

// whatsappMsg builds a JSON webhook request body for the supplied
// phone + message text. The timestamp + message_id are deterministic
// fixtures so tests can assert on them.
func whatsappMsg(phone, body string) string {
	b, _ := json.Marshal(WhatsAppMessage{
		From:      phone,
		Body:      body,
		Timestamp: "1700000000",
		MessageID: "wamid.test_" + body,
	})
	return string(b)
}

// --- WhatsAppRateLimiter ---

// TestWhatsAppRateLimiter_AllowsUnderLimit verifies that the limiter
// allows up to `limit` messages per phone per UTC day, then rejects
// the next one. The counter is consumed on each Allow call so a
// concurrent second message can't race past the ceiling.
func TestWhatsAppRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := NewWhatsAppRateLimiter(3)
	phone := "+254700000001"
	for i := 0; i < 3; i++ {
		if !rl.Allow(phone) {
			t.Fatalf("call %d: expected Allow=true, got false (limit=3)", i+1)
		}
	}
	// 4th call should be rejected.
	if rl.Allow(phone) {
		t.Fatal("call 4: expected Allow=false (limit reached), got true")
	}
	// Remaining should be 0.
	if got := rl.Remaining(phone); got != 0 {
		t.Errorf("Remaining after limit hit: expected 0, got %d", got)
	}
}

// TestWhatsAppRateLimiter_PerPhoneIsolation verifies that the limit is
// per-phone — phone B's messages don't count against phone A's limit.
func TestWhatsAppRateLimiter_PerPhoneIsolation(t *testing.T) {
	rl := NewWhatsAppRateLimiter(2)
	if !rl.Allow("+254700000001") {
		t.Fatal("phone A call 1: expected Allow=true")
	}
	if !rl.Allow("+254700000001") {
		t.Fatal("phone A call 2: expected Allow=true")
	}
	// Phone A is at the limit, but phone B should still get its full quota.
	if !rl.Allow("+254700000002") {
		t.Fatal("phone B call 1: expected Allow=true (per-phone isolation)")
	}
	if !rl.Allow("+254700000002") {
		t.Fatal("phone B call 2: expected Allow=true (per-phone isolation)")
	}
	// Phone B is now also at the limit.
	if rl.Allow("+254700000002") {
		t.Fatal("phone B call 3: expected Allow=false (limit reached)")
	}
}

// TestWhatsAppRateLimiter_DisabledWhenLimitZeroOrNegative verifies
// that a limit <= 0 disables rate limiting entirely — Allow always
// returns true without touching the counters. Used by tests that want
// to exercise the keyword routes without hitting the ceiling.
func TestWhatsAppRateLimiter_DisabledWhenLimitZeroOrNegative(t *testing.T) {
	rl := NewWhatsAppRateLimiter(0)
	for i := 0; i < 10; i++ {
		if !rl.Allow("+254700000001") {
			t.Fatalf("call %d: expected Allow=true (limit disabled)", i+1)
		}
	}
	if got := rl.Remaining("+254700000001"); got != -1 {
		t.Errorf("Remaining when disabled: expected -1 (unlimited), got %d", got)
	}
}

// --- WhatsAppSender selection ---

// TestNewWhatsAppSender_StubWhenNoAPIKey verifies that NewWhatsAppSender
// returns a *StubWhatsAppSender when WHATSAPP_API_KEY is unset. This is
// the dev / test default; production sets the env var to opt into real
// delivery via Meta's Cloud API.
func TestNewWhatsAppSender_StubWhenNoAPIKey(t *testing.T) {
	t.Setenv("WHATSAPP_API_KEY", "")
	t.Setenv("WHATSAPP_PHONE_NUMBER_ID", "")
	s := NewWhatsAppSender(nil)
	if _, ok := s.(*StubWhatsAppSender); !ok {
		t.Fatalf("expected *StubWhatsAppSender, got %T", s)
	}
	if s.Name() != "stub" {
		t.Errorf("expected name=stub, got %s", s.Name())
	}
}

// TestNewWhatsAppSender_CloudWhenAPIKeySet verifies that
// NewWhatsAppSender returns a *CloudWhatsAppSender when both env vars
// are set. We do NOT exercise the live HTTP path here — the stub
// contract is sufficient for unit tests.
func TestNewWhatsAppSender_CloudWhenAPIKeySet(t *testing.T) {
	t.Setenv("WHATSAPP_API_KEY", "test-fake-token-EAAxxxx")
	t.Setenv("WHATSAPP_PHONE_NUMBER_ID", "1234567890")
	s := NewWhatsAppSender(nil)
	cws, ok := s.(*CloudWhatsAppSender)
	if !ok {
		t.Fatalf("expected *CloudWhatsAppSender, got %T", s)
	}
	if cws.apiKey != "test-fake-token-EAAxxxx" {
		t.Errorf("apiKey: expected test-fake-token-EAAxxxx, got %s", cws.apiKey)
	}
	if cws.phoneNumberID != "1234567890" {
		t.Errorf("phoneNumberID: expected 1234567890, got %s", cws.phoneNumberID)
	}
	if s.Name() != "cloud" {
		t.Errorf("expected name=cloud, got %s", s.Name())
	}
}

// TestNewWhatsAppSender_StubWhenOnlyAPIKeySet verifies the sender
// falls back to the stub when the phone number ID is missing (both
// env vars are required for the cloud sender — a key alone can't
// address the messages endpoint).
func TestNewWhatsAppSender_StubWhenOnlyAPIKeySet(t *testing.T) {
	t.Setenv("WHATSAPP_API_KEY", "test-fake-token")
	t.Setenv("WHATSAPP_PHONE_NUMBER_ID", "")
	s := NewWhatsAppSender(nil)
	if _, ok := s.(*StubWhatsAppSender); !ok {
		t.Fatalf("expected *StubWhatsAppSender when phone_number_id is missing, got %T", s)
	}
}

// --- StubWhatsAppSender ---

// TestStubWhatsAppSender_CapturesSend verifies the stub records the
// last to + message so tests can assert what was "sent" without
// parsing log output. Send never returns an error.
func TestStubWhatsAppSender_CapturesSend(t *testing.T) {
	stub := &StubWhatsAppSender{}
	delivery, err := stub.Send(context.Background(), "+254712345678", "Hello from Civic Intelligence")
	if err != nil {
		t.Fatalf("stub.Send returned error: %v", err)
	}
	if delivery.Status != WhatsAppStatusSent {
		t.Errorf("expected status=sent, got %s", delivery.Status)
	}
	if delivery.To != "+254712345678" {
		t.Errorf("delivery to: expected +254712345678, got %s", delivery.To)
	}
	if delivery.Message != "Hello from Civic Intelligence" {
		t.Errorf("delivery message mismatch")
	}
	if !strings.HasPrefix(delivery.ID, "wa_") {
		t.Errorf("expected delivery ID prefix wa_, got %s", delivery.ID)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.LastTo != "+254712345678" {
		t.Errorf("LastTo: %s", stub.LastTo)
	}
	if stub.LastMessage != "Hello from Civic Intelligence" {
		t.Errorf("LastMessage: %s", stub.LastMessage)
	}
	if stub.SendCount != 1 {
		t.Errorf("SendCount: expected 1, got %d", stub.SendCount)
	}
}

// --- WhatsAppBot.Handle (keyword routing) ---

// TestWhatsAppBot_Handle_HelpKeyword verifies the HELP keyword returns
// the menu + the help route. The menu must mention every keyword
// (BILL, MP, GAZETTE, HELP) so a citizen can discover the bot's
// capabilities from their first message.
func TestWhatsAppBot_Handle_HelpKeyword(t *testing.T) {
	bot, _ := newTestBot(5)
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000001", Body: "HELP",
	})
	if route != whatsappRouteHelp {
		t.Errorf("route: expected %q, got %q", whatsappRouteHelp, route)
	}
	for _, kw := range []string{"BILL", "MP", "GAZETTE", "HELP"} {
		if !strings.Contains(reply, kw) {
			t.Errorf("HELP reply missing keyword %q; reply=%q", kw, reply)
		}
	}
}

// TestWhatsAppBot_Handle_BillKeyword verifies the BILL keyword looks up
// a Bill by number and returns a summary. The lookup uses the mock
// bills adapter so the test is deterministic (no live crawl).
func TestWhatsAppBot_Handle_BillKeyword(t *testing.T) {
	rl := NewWhatsAppRateLimiter(5)
	bot := NewWhatsAppBot(rl, "", &mockBillsAdapter{
		bills: []kenya_law.BillCandidate{
			{
				SourceID:        "ke-bill-2024-09-07-statutory-instruments-amendment",
				Title:           "NA Bill No. 23 of 2024 — Statutory Instruments (Amendment)",
				House:           "National Assembly",
				PublicationDate: time.Date(2024, 9, 7, 0, 0, 0, 0, time.UTC),
				URL:             "https://kenyalaw.org/bills/2024/statutory-instruments-amendment",
				Slug:            "statutory-instruments-amendment-2024",
			},
		},
	}, gazetteAlertStore)

	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000001", Body: "BILL 23 of 2024",
	})
	if route != whatsappRouteBill {
		t.Errorf("route: expected %q, got %q", whatsappRouteBill, route)
	}
	// The reply MUST surface the Bill's title + house.
	if !strings.Contains(reply, "Statutory Instruments") {
		t.Errorf("reply missing Bill title; reply=%q", reply)
	}
	if !strings.Contains(reply, "National Assembly") {
		t.Errorf("reply missing House; reply=%q", reply)
	}
	// The reply MUST link to the full Bill detail page.
	if !strings.Contains(reply, "/api/v1/bills/") {
		t.Errorf("reply missing Bill detail link; reply=%q", reply)
	}
}

// TestWhatsAppBot_Handle_BillKeyword_NoMatch verifies the BILL keyword
// returns a friendly "no match" reply when the number doesn't match
// any Bill (rather than erroring or returning an empty reply).
func TestWhatsAppBot_Handle_BillKeyword_NoMatch(t *testing.T) {
	bot, _ := newTestBot(5)
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000001", Body: "BILL 999 of 2099",
	})
	if route != whatsappRouteBill {
		t.Errorf("route: expected %q, got %q", whatsappRouteBill, route)
	}
	if !strings.Contains(strings.ToLower(reply), "no bill matched") && !strings.Contains(strings.ToLower(reply), "no bill") {
		t.Errorf("expected a 'no Bill matched' reply, got %q", reply)
	}
}

// TestWhatsAppBot_Handle_BillKeyword_EmptyNumber verifies the BILL
// keyword with no number returns a usage prompt.
func TestWhatsAppBot_Handle_BillKeyword_EmptyNumber(t *testing.T) {
	bot, _ := newTestBot(5)
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000001", Body: "BILL",
	})
	if route != whatsappRouteBill {
		t.Errorf("route: expected %q, got %q", whatsappRouteBill, route)
	}
	if !strings.Contains(strings.ToLower(reply), "bill number") {
		t.Errorf("expected a usage prompt mentioning 'bill number', got %q", reply)
	}
}

// TestWhatsAppBot_Handle_MPKeyword verifies the MP keyword looks up an
// MP by name and returns a scorecard summary. The lookup matches the
// samplePeople slice (Speakers + Presidents across the 6 supported
// countries) so "Wetangula" resolves to "Rt. Hon. Moses Wetangula".
func TestWhatsAppBot_Handle_MPKeyword(t *testing.T) {
	bot, _ := newTestBot(5)
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000001", Body: "MP Wetangula",
	})
	if route != whatsappRouteMP {
		t.Errorf("route: expected %q, got %q", whatsappRouteMP, route)
	}
	// The reply MUST surface the MP's name.
	if !strings.Contains(reply, "Wetangula") {
		t.Errorf("reply missing MP name; reply=%q", reply)
	}
	// The reply MUST link to the full scorecard page.
	if !strings.Contains(reply, "/scorecard") {
		t.Errorf("reply missing scorecard link; reply=%q", reply)
	}
}

// TestWhatsAppBot_Handle_MPKeyword_ScorecardMatch verifies the MP
// keyword matches against the sampleScorecards slice (which carries
// richer data — attendance, bills sponsored, etc.) when the name
// matches a scorecard MP (e.g. "Ichung'wah").
func TestWhatsAppBot_Handle_MPKeyword_ScorecardMatch(t *testing.T) {
	bot, _ := newTestBot(5)
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000001", Body: "MP Ichung'wah",
	})
	if route != whatsappRouteMP {
		t.Errorf("route: expected %q, got %q", whatsappRouteMP, route)
	}
	if !strings.Contains(reply, "Ichung'wah") {
		t.Errorf("reply missing MP name; reply=%q", reply)
	}
	// The scorecard reply MUST surface raw counts (bills sponsored,
	// questions asked) — the platform NEVER aggregates into a score.
	if !strings.Contains(reply, "Bills sponsored") {
		t.Errorf("reply missing 'Bills sponsored' metric; reply=%q", reply)
	}
	// The disclaimer MUST be present so the citizen knows the
	// platform doesn't rank MPs.
	if !strings.Contains(reply, scorecardDisclaimer) {
		t.Errorf("reply missing scorecard disclaimer; reply=%q", reply)
	}
}

// TestWhatsAppBot_Handle_MPKeyword_NoMatch verifies the MP keyword
// returns a friendly "no match" reply when the name doesn't match.
func TestWhatsAppBot_Handle_MPKeyword_NoMatch(t *testing.T) {
	bot, _ := newTestBot(5)
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000001", Body: "MP Nonexistentperson",
	})
	if route != whatsappRouteMP {
		t.Errorf("route: expected %q, got %q", whatsappRouteMP, route)
	}
	if !strings.Contains(strings.ToLower(reply), "no mp matched") && !strings.Contains(strings.ToLower(reply), "no mp") {
		t.Errorf("expected a 'no MP matched' reply, got %q", reply)
	}
}

// TestWhatsAppBot_Handle_GazetteKeyword verifies the GAZETTE keyword
// looks up the latest gazette notice matching the supplied keyword.
// The lookup uses a freshly-seeded gazette store (the package-level
// gazetteAlertStore is only seeded in main(), not at init, so tests
// must seed their own).
func TestWhatsAppBot_Handle_GazetteKeyword(t *testing.T) {
	rl := NewWhatsAppRateLimiter(5)
	bot := NewWhatsAppBot(rl, "", &mockBillsAdapter{}, newSeededGazetteStore(t))

	// "tender" is the most-subscribed keyword in the seed data — 4 of
	// the 13 seeded notices contain "tender" in the title/body.
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000077", Body: "GAZETTE tender",
	})
	if route != whatsappRouteGazette {
		t.Fatalf("route: expected %q, got %q", whatsappRouteGazette, route)
	}
	if !strings.Contains(reply, "Source:") {
		t.Errorf("expected a gazette notice reply with Source: line; reply=%q", reply)
	}
	if !strings.Contains(strings.ToLower(reply), "tender") {
		t.Errorf("reply missing 'tender' keyword; reply=%q", reply)
	}
}

// TestWhatsAppBot_Handle_QADispatch_GracefulWhenNoAIService verifies
// the free-form Q&A path returns a graceful degradation when the AI
// service URL is not configured (rather than erroring or hanging).
func TestWhatsAppBot_Handle_QADispatch_GracefulWhenNoAIService(t *testing.T) {
	bot, _ := newTestBot(5)
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000001", Body: "What is the Affordable Housing Bill?",
	})
	if route != whatsappRouteQA {
		t.Errorf("route: expected %q, got %q", whatsappRouteQA, route)
	}
	if !strings.Contains(strings.ToLower(reply), "not configured") && !strings.Contains(strings.ToLower(reply), "unavailable") {
		t.Errorf("expected a graceful degradation reply, got %q", reply)
	}
}

// TestWhatsAppBot_Handle_RateLimit verifies that after the daily limit
// is hit, subsequent messages return the rate-limit reply + the
// rate_limited route. The rate-limit reply MUST mention the limit +
// the reset time so the citizen knows when to try again.
func TestWhatsAppBot_Handle_RateLimit(t *testing.T) {
	bot, _ := newTestBot(2) // limit=2 for a fast test
	phone := "+254700000001"

	// First two messages should be allowed.
	for i := 0; i < 2; i++ {
		reply, route := bot.Handle(context.Background(), WhatsAppMessage{
			From: phone, Body: "HELP",
		})
		if route == whatsappRouteRateLimit {
			t.Fatalf("call %d: expected non-rate-limit route, got %q (reply=%q)", i+1, route, reply)
		}
	}

	// Third message should be rate-limited.
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: phone, Body: "HELP",
	})
	if route != whatsappRouteRateLimit {
		t.Fatalf("call 3: expected route=%q, got %q (reply=%q)", whatsappRouteRateLimit, route, reply)
	}
	// The reply MUST mention the limit + the reset time.
	if !strings.Contains(reply, "daily free limit") {
		t.Errorf("rate-limit reply missing 'daily free limit'; reply=%q", reply)
	}
	if !strings.Contains(reply, "resets at") {
		t.Errorf("rate-limit reply missing 'resets at'; reply=%q", reply)
	}
	// The reply MUST link to the sponsor page so the citizen can
	// upgrade to a paid tier.
	if !strings.Contains(reply, "sponsor") {
		t.Errorf("rate-limit reply missing sponsor link; reply=%q", reply)
	}
}

// TestWhatsAppBot_Handle_EmptyBodyReturnsHelp verifies that an empty
// body (which Meta sends as a test ping during webhook verification)
// returns the HELP menu rather than erroring.
func TestWhatsAppBot_Handle_EmptyBodyReturnsHelp(t *testing.T) {
	bot, _ := newTestBot(5)
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "+254700000001", Body: "",
	})
	if route != whatsappRouteHelp {
		t.Errorf("route: expected %q, got %q", whatsappRouteHelp, route)
	}
	if !strings.Contains(reply, "Civic Intelligence") {
		t.Errorf("HELP reply missing 'Civic Intelligence'; reply=%q", reply)
	}
}

// TestWhatsAppBot_Handle_EmptyPhoneReturnsInvalid verifies that a
// missing phone number returns the invalid route (not a crash).
func TestWhatsAppBot_Handle_EmptyPhoneReturnsInvalid(t *testing.T) {
	bot, _ := newTestBot(5)
	reply, route := bot.Handle(context.Background(), WhatsAppMessage{
		From: "", Body: "HELP",
	})
	if route != whatsappRouteInvalid {
		t.Errorf("route: expected %q, got %q", whatsappRouteInvalid, route)
	}
	if !strings.Contains(strings.ToLower(reply), "phone number") {
		t.Errorf("expected a 'phone number' error reply, got %q", reply)
	}
}

// --- HTTP handlers: webhook ---

// TestWhatsAppWebhook_Returns200 verifies that POST
// /api/v1/whatsapp/webhook returns 200 OK for a valid inbound message.
// The endpoint MUST always return 200 (even on rate-limit or invalid
// input) so Meta doesn't retry the webhook delivery — retries would
// duplicate the inbound message in the bot's logs and re-consume the
// citizen's daily limit.
func TestWhatsAppWebhook_Returns200(t *testing.T) {
	bot, _ := newTestBot(5)
	stub := &StubWhatsAppSender{}
	handler := makeWhatsAppWebhookHandler(bot, stub)
	rr, resp := whatsappDispatch(handler, whatsappMsg("+254700000001", "HELP"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if resp.Status != whatsappStatusOK {
		t.Errorf("status: expected %q, got %q", whatsappStatusOK, resp.Status)
	}
	if resp.From != "+254700000001" {
		t.Errorf("From: expected +254700000001, got %s", resp.From)
	}
	if resp.Route != whatsappRouteHelp {
		t.Errorf("route: expected %q, got %q", whatsappRouteHelp, resp.Route)
	}
	if resp.Sender != "stub" {
		t.Errorf("sender: expected stub, got %s", resp.Sender)
	}
	if resp.Reply == "" {
		t.Error("expected non-empty reply")
	}
	if resp.Delivery == nil {
		t.Error("expected non-nil delivery record")
	} else if resp.Delivery.Status != WhatsAppStatusSent {
		t.Errorf("delivery status: expected sent, got %s", resp.Delivery.Status)
	}
	if resp.RateLimit == nil {
		t.Error("expected non-nil rate_limit info")
	} else {
		if resp.RateLimit.Limit != 5 {
			t.Errorf("rate_limit.limit: expected 5, got %d", resp.RateLimit.Limit)
		}
		if resp.RateLimit.Remaining != 4 {
			t.Errorf("rate_limit.remaining: expected 4 (after 1 message), got %d", resp.RateLimit.Remaining)
		}
	}
}

// TestWhatsAppWebhook_BillKeyword verifies the BILL keyword routing
// through the full HTTP webhook path. The webhook MUST parse the JSON
// body, dispatch through the bot, send the reply via the stub sender,
// and surface the delivery record + rate-limit state in the response.
func TestWhatsAppWebhook_BillKeyword(t *testing.T) {
	rl := NewWhatsAppRateLimiter(5)
	bot := NewWhatsAppBot(rl, "", &mockBillsAdapter{
		bills: []kenya_law.BillCandidate{
			{
				SourceID:        "ke-bill-2024-09-07-statutory-instruments-amendment",
				Title:           "NA Bill No. 23 of 2024 — Statutory Instruments (Amendment)",
				House:           "National Assembly",
				PublicationDate: time.Date(2024, 9, 7, 0, 0, 0, 0, time.UTC),
				URL:             "https://kenyalaw.org/bills/2024/statutory-instruments-amendment",
				Slug:            "statutory-instruments-amendment-2024",
			},
		},
	}, gazetteAlertStore)
	stub := &StubWhatsAppSender{}
	handler := makeWhatsAppWebhookHandler(bot, stub)

	rr, resp := whatsappDispatch(handler, whatsappMsg("+254700000001", "BILL 23 of 2024"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if resp.Status != whatsappStatusOK {
		t.Errorf("status: expected %q, got %q", whatsappStatusOK, resp.Status)
	}
	if resp.Route != whatsappRouteBill {
		t.Errorf("route: expected %q, got %q", whatsappRouteBill, resp.Route)
	}
	if !strings.Contains(resp.Reply, "Statutory Instruments") {
		t.Errorf("reply missing Bill title; reply=%q", resp.Reply)
	}
	// The stub sender MUST have been called with the reply.
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.SendCount != 1 {
		t.Errorf("stub SendCount: expected 1, got %d", stub.SendCount)
	}
	if stub.LastTo != "+254700000001" {
		t.Errorf("stub LastTo: expected +254700000001, got %s", stub.LastTo)
	}
	if !strings.Contains(stub.LastMessage, "Statutory Instruments") {
		t.Errorf("stub LastMessage missing Bill title; msg=%q", stub.LastMessage)
	}
}

// TestWhatsAppWebhook_MPKeyword verifies the MP keyword routing through
// the full HTTP webhook path. The webhook MUST surface the MP's name +
// the scorecard link in the reply.
func TestWhatsAppWebhook_MPKeyword(t *testing.T) {
	bot, _ := newTestBot(5)
	stub := &StubWhatsAppSender{}
	handler := makeWhatsAppWebhookHandler(bot, stub)

	rr, resp := whatsappDispatch(handler, whatsappMsg("+254700000001", "MP Wetangula"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if resp.Status != whatsappStatusOK {
		t.Errorf("status: expected %q, got %q", whatsappStatusOK, resp.Status)
	}
	if resp.Route != whatsappRouteMP {
		t.Errorf("route: expected %q, got %q", whatsappRouteMP, resp.Route)
	}
	if !strings.Contains(resp.Reply, "Wetangula") {
		t.Errorf("reply missing MP name; reply=%q", resp.Reply)
	}
	if !strings.Contains(resp.Reply, "/scorecard") {
		t.Errorf("reply missing scorecard link; reply=%q", resp.Reply)
	}
	// The stub sender MUST have been called with the reply.
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.SendCount != 1 {
		t.Errorf("stub SendCount: expected 1, got %d", stub.SendCount)
	}
	if !strings.Contains(stub.LastMessage, "Wetangula") {
		t.Errorf("stub LastMessage missing MP name; msg=%q", stub.LastMessage)
	}
}

// TestWhatsAppWebhook_RateLimit verifies the webhook returns 200 OK
// (NOT 429) when a phone hits its daily limit, with status=
// "rate_limited" in the JSON body. The 200 is required so Meta
// doesn't retry the webhook delivery (retries would re-consume the
// citizen's limit + duplicate the inbound message in logs).
func TestWhatsAppWebhook_RateLimit(t *testing.T) {
	bot, _ := newTestBot(2) // limit=2 for a fast test
	stub := &StubWhatsAppSender{}
	handler := makeWhatsAppWebhookHandler(bot, stub)
	phone := "+254700000001"

	// First two messages: 200 OK with status=ok.
	for i := 0; i < 2; i++ {
		rr, resp := whatsappDispatch(handler, whatsappMsg(phone, "HELP"))
		if rr.Code != http.StatusOK {
			t.Fatalf("call %d: expected 200, got %d (body=%s)", i+1, rr.Code, rr.Body.String())
		}
		if resp.Status != whatsappStatusOK {
			t.Errorf("call %d: status: expected %q, got %q", i+1, whatsappStatusOK, resp.Status)
		}
		if resp.Route == whatsappRouteRateLimit {
			t.Fatalf("call %d: did not expect rate_limited route yet (reply=%q)", i+1, resp.Reply)
		}
	}

	// Third message: 200 OK with status=rate_limited + route=rate_limited.
	rr, resp := whatsappDispatch(handler, whatsappMsg(phone, "HELP"))
	if rr.Code != http.StatusOK {
		t.Fatalf("call 3: expected 200 (NOT 429 — Meta would retry), got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if resp.Status != whatsappStatusRateLimited {
		t.Errorf("call 3: status: expected %q, got %q", whatsappStatusRateLimited, resp.Status)
	}
	if resp.Route != whatsappRouteRateLimit {
		t.Errorf("call 3: route: expected %q, got %q", whatsappRouteRateLimit, resp.Route)
	}
	if !strings.Contains(resp.Reply, "daily free limit") {
		t.Errorf("rate-limit reply missing 'daily free limit'; reply=%q", resp.Reply)
	}
	// The rate-limit info MUST show 0 remaining.
	if resp.RateLimit == nil {
		t.Fatal("expected non-nil rate_limit info")
	}
	if resp.RateLimit.Remaining != 0 {
		t.Errorf("rate_limit.remaining: expected 0, got %d", resp.RateLimit.Remaining)
	}
	if resp.RateLimit.Limit != 2 {
		t.Errorf("rate_limit.limit: expected 2, got %d", resp.RateLimit.Limit)
	}
	// The stub sender MUST still have been called with the rate-limit
	// reply (the citizen needs to see the limit message on WhatsApp).
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.SendCount != 3 {
		t.Errorf("stub SendCount: expected 3 (one per webhook call, including the rate-limit reply), got %d", stub.SendCount)
	}
	if !strings.Contains(stub.LastMessage, "daily free limit") {
		t.Errorf("stub LastMessage missing 'daily free limit'; msg=%q", stub.LastMessage)
	}
}

// TestWhatsAppWebhook_MethodNotAllowed verifies GET returns 405.
func TestWhatsAppWebhook_MethodNotAllowed(t *testing.T) {
	bot, _ := newTestBot(5)
	stub := &StubWhatsAppSender{}
	handler := makeWhatsAppWebhookHandler(bot, stub)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/whatsapp/webhook", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Allow"), "POST") {
		t.Errorf("Allow header: expected POST, got %s", rr.Header().Get("Allow"))
	}
}

// TestWhatsAppWebhook_InvalidJSONReturns200 verifies the webhook
// returns 200 OK (NOT 400) with status="invalid" when the JSON body
// is malformed. The 200 is required so Meta doesn't retry the webhook
// delivery on a bad payload.
func TestWhatsAppWebhook_InvalidJSONReturns200(t *testing.T) {
	bot, _ := newTestBot(5)
	stub := &StubWhatsAppSender{}
	handler := makeWhatsAppWebhookHandler(bot, stub)
	rr, resp := whatsappDispatch(handler, "not json")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 (NOT 400 — Meta would retry), got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if resp.Status != whatsappStatusInvalid {
		t.Errorf("status: expected %q, got %q", whatsappStatusInvalid, resp.Status)
	}
}

// TestWhatsAppWebhook_PhoneNormalisedWithoutPlus verifies the webhook
// normalises a phone number without the leading '+' to E.164 (Meta
// sends it without the '+' in some modes). The normalised phone is
// used as the rate-limit key + the sender's `to` field so the
// citizen's limit + delivery record are consistent across modes.
func TestWhatsAppWebhook_PhoneNormalisedWithoutPlus(t *testing.T) {
	bot, _ := newTestBot(5)
	stub := &StubWhatsAppSender{}
	handler := makeWhatsAppWebhookHandler(bot, stub)
	// Send without the leading '+'.
	body := `{"from":"254700000001","body":"HELP","timestamp":"1700000000","message_id":"wamid.test"}`
	rr, resp := whatsappDispatch(handler, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if resp.From != "+254700000001" {
		t.Errorf("From: expected +254700000001 (normalised), got %s", resp.From)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.LastTo != "+254700000001" {
		t.Errorf("stub LastTo: expected +254700000001 (normalised), got %s", stub.LastTo)
	}
}

// --- HTTP handlers: status ---

// TestWhatsAppStatus_ReturnsOK verifies GET /api/v1/whatsapp/status
// returns 200 OK with the sender name + daily limit + keyword list.
func TestWhatsAppStatus_ReturnsOK(t *testing.T) {
	rl := NewWhatsAppRateLimiter(5)
	stub := &StubWhatsAppSender{}
	handler := makeWhatsAppStatusHandler(stub, rl)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/whatsapp/status", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Status     string   `json:"status"`
		Sender     string   `json:"sender"`
		DailyLimit int      `json:"daily_limit"`
		Keywords   []string `json:"keywords"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v (body=%s)", err, rr.Body.String())
	}
	if resp.Status != "ok" {
		t.Errorf("status: expected ok, got %s", resp.Status)
	}
	if resp.Sender != "stub" {
		t.Errorf("sender: expected stub, got %s", resp.Sender)
	}
	if resp.DailyLimit != 5 {
		t.Errorf("daily_limit: expected 5, got %d", resp.DailyLimit)
	}
	// Every documented keyword MUST appear.
	for _, kw := range []string{"BILL", "MP", "GAZETTE", "HELP"} {
		found := false
		for _, k := range resp.Keywords {
			if strings.Contains(k, kw) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("keywords list missing %q; got %v", kw, resp.Keywords)
		}
	}
}

// TestWhatsAppStatus_MethodNotAllowed verifies POST returns 405.
func TestWhatsAppStatus_MethodNotAllowed(t *testing.T) {
	rl := NewWhatsAppRateLimiter(5)
	stub := &StubWhatsAppSender{}
	handler := makeWhatsAppStatusHandler(stub, rl)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/status", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Allow"), "GET") {
		t.Errorf("Allow header: expected GET, got %s", rr.Header().Get("Allow"))
	}
}

// --- Configurable daily limit (env var override) ---

// TestWhatsAppFreeDailyLimit_EnvOverride verifies the
// WHATSAPP_FREE_DAILY_LIMIT env var overrides the default limit of 5.
// This lets an operator tune the free-tier ceiling without a code
// change (e.g. to 10 for a promotional period, or to 0 to disable
// the bot entirely).
func TestWhatsAppFreeDailyLimit_EnvOverride(t *testing.T) {
	t.Setenv("WHATSAPP_FREE_DAILY_LIMIT", "10")
	limit := parseIntDefaultEnv("WHATSAPP_FREE_DAILY_LIMIT", defaultWhatsAppFreeDailyLimit)
	if limit != 10 {
		t.Errorf("expected limit=10 from env override, got %d", limit)
	}

	t.Setenv("WHATSAPP_FREE_DAILY_LIMIT", "not-a-number")
	limit = parseIntDefaultEnv("WHATSAPP_FREE_DAILY_LIMIT", defaultWhatsAppFreeDailyLimit)
	if limit != defaultWhatsAppFreeDailyLimit {
		t.Errorf("expected default %d on invalid env value, got %d", defaultWhatsAppFreeDailyLimit, limit)
	}

	t.Setenv("WHATSAPP_FREE_DAILY_LIMIT", "")
	limit = parseIntDefaultEnv("WHATSAPP_FREE_DAILY_LIMIT", defaultWhatsAppFreeDailyLimit)
	if limit != defaultWhatsAppFreeDailyLimit {
		t.Errorf("expected default %d on empty env value, got %d", defaultWhatsAppFreeDailyLimit, limit)
	}
}
