// Package main — tests for the SMS alerts handlers (issue #289).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"net/http"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
)

// === SMSSubscriberStore ===

// TestSMSSubscriberStore_Subscribe_New verifies that Subscribe creates a
// new subscriber with normalised phone (kept as-is when already E.164),
// upper-cased country, lower-cased + de-duplicated keywords, and the
// Active flag set to true.
func TestSMSSubscriberStore_Subscribe_New(t *testing.T) {
	s := NewSMSSubscriberStore()
	rec, created, err := s.Subscribe("+254712345678", "ke", []string{"TENDER", "tender", "Health"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if !created {
		t.Fatal("expected created=true for a new subscriber")
	}
	if rec.Phone != "+254712345678" {
		t.Errorf("Phone: expected +254712345678, got %s", rec.Phone)
	}
	if rec.Country != "KE" {
		t.Errorf("Country: expected KE, got %s", rec.Country)
	}
	want := []string{"tender", "health"}
	if len(rec.Keywords) != len(want) {
		t.Fatalf("Keywords: expected %v, got %v", want, rec.Keywords)
	}
	for i, k := range rec.Keywords {
		if k != want[i] {
			t.Errorf("Keywords[%d]: expected %q, got %q", i, want[i], k)
		}
	}
	if !rec.Active {
		t.Error("expected Active=true")
	}
	if rec.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

// TestSMSSubscriberStore_Subscribe_Duplicate returns the existing
// record + created=false — the HTTP handler maps this to 409.
func TestSMSSubscriberStore_Subscribe_Duplicate(t *testing.T) {
	s := NewSMSSubscriberStore()
	_, _, err := s.Subscribe("+254700000099", "KE", []string{"tender"})
	if err != nil {
		t.Fatalf("first Subscribe: %v", err)
	}
	rec, created, err := s.Subscribe("+254700000099", "KE", []string{"health"})
	if err != nil {
		t.Fatalf("second Subscribe: %v", err)
	}
	if created {
		t.Error("expected created=false for a duplicate phone")
	}
	if len(rec.Keywords) != 1 || rec.Keywords[0] != "tender" {
		t.Errorf("duplicate should return the existing record unchanged; got keywords %v", rec.Keywords)
	}
}

// TestSMSSubscriberStore_Subscribe_Validation covers the five
// validation errors: empty phone, non-E.164 phone, missing keywords,
// only-blank keywords, and an over-long keyword.
func TestSMSSubscriberStore_Subscribe_Validation(t *testing.T) {
	s := NewSMSSubscriberStore()
	long := strings.Repeat("a", 65)
	cases := []struct {
		name     string
		phone    string
		country  string
		keywords []string
		wantErr  string
	}{
		{"empty phone", "", "KE", []string{"tender"}, "phone is required"},
		{"non-E.164 phone", "254712345678", "KE", []string{"tender"}, "E.164"},
		{"missing keywords", "+254712345678", "KE", nil, "at least one keyword"},
		{"only blank keywords", "+254712345678", "KE", []string{"  ", ""}, "at least one keyword"},
		{"over-long keyword", "+254712345678", "KE", []string{long}, "exceeds 64 characters"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := s.Subscribe(c.phone, c.country, c.keywords)
			if err == nil {
				t.Fatalf("expected error %q, got nil", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("expected error %q, got %q", c.wantErr, err.Error())
			}
		})
	}
}

// TestSMSSubscriberStore_List_SeededWith10Subscribers verifies the
// package-level store has been seeded with exactly 10 subscribers
// spanning 5 countries (KE, UG, TZ, GH, NG).
func TestSMSSubscriberStore_List_SeededWith10Subscribers(t *testing.T) {
	subs := smsSubscriberStore.List()
	if len(subs) != 10 {
		t.Fatalf("expected 10 seeded subscribers, got %d", len(subs))
	}
	countries := make(map[string]bool, 5)
	for _, s := range subs {
		countries[s.Country] = true
		if len(s.Keywords) == 0 {
			t.Errorf("subscriber %s has no keywords", s.Phone)
		}
		if !strings.HasPrefix(s.Phone, "+") {
			t.Errorf("subscriber %s is not E.164", s.Phone)
		}
	}
	wantCountries := []string{"KE", "UG", "TZ", "GH", "NG"}
	for _, c := range wantCountries {
		if !countries[c] {
			t.Errorf("expected seeded subscriber from %s", c)
		}
	}
}

// TestSMSSubscriberStore_ActiveSubscribers_FilteredByKeyword verifies
// the broadcast recipient list is correctly filtered by keyword.
func TestSMSSubscriberStore_ActiveSubscribers_FilteredByKeyword(t *testing.T) {
	s := NewSMSSubscriberStore()
	_, _, _ = s.Subscribe("+254700000010", "KE", []string{"tender"})
	_, _, _ = s.Subscribe("+254700000011", "KE", []string{"health"})
	_, _, _ = s.Subscribe("+254700000012", "KE", []string{"tender", "health"})

	all := s.ActiveSubscribers(nil)
	if len(all) != 3 {
		t.Fatalf("expected 3 active subscribers, got %d", len(all))
	}
	tender := s.ActiveSubscribers([]string{"tender"})
	if len(tender) != 2 {
		t.Errorf("expected 2 tender subscribers, got %d", len(tender))
	}
	health := s.ActiveSubscribers([]string{"health"})
	if len(health) != 2 {
		t.Errorf("expected 2 health subscribers, got %d", len(health))
	}
	nonMatch := s.ActiveSubscribers([]string{"tax"})
	if len(nonMatch) != 0 {
		t.Errorf("expected 0 tax subscribers, got %d", len(nonMatch))
	}
}

// TestSMSSubscriberStore_SetActive verifies the mute/unmute path.
func TestSMSSubscriberStore_SetActive(t *testing.T) {
	s := NewSMSSubscriberStore()
	_, _, _ = s.Subscribe("+254700000020", "KE", []string{"tender"})
	if !s.SetActive("+254700000020", false) {
		t.Fatal("expected SetActive to return true for an existing subscriber")
	}
	active := s.ActiveSubscribers(nil)
	if len(active) != 0 {
		t.Errorf("expected 0 active subscribers after mute, got %d", len(active))
	}
	if !s.SetActive("+254700000020", true) {
		t.Fatal("expected SetActive to return true when reactivating")
	}
	active = s.ActiveSubscribers(nil)
	if len(active) != 1 {
		t.Errorf("expected 1 active subscriber after unmute, got %d", len(active))
	}
	if s.SetActive("+999999999999", true) {
		t.Error("expected SetActive to return false for unknown phone")
	}
}

// === SMSSender selection ===

// TestNewSMSSender_StubWhenNoAPIKey verifies that NewSMSSender returns a
// *StubSMSSender when AFRICAS_TALKING_API_KEY is unset. This is the
// dev / test default; production sets the env var to opt into real
// delivery via Africa's Talking.
func TestNewSMSSender_StubWhenNoAPIKey(t *testing.T) {
	t.Setenv("AFRICAS_TALKING_API_KEY", "")
	t.Setenv("AFRICAS_TALKING_USERNAME", "")
	s := NewSMSSender(nil)
	if _, ok := s.(*StubSMSSender); !ok {
		t.Fatalf("expected *StubSMSSender, got %T", s)
	}
	if s.Name() != "stub" {
		t.Errorf("expected name=stub, got %s", s.Name())
	}
}

// TestNewSMSSender_AfricasTalkingWhenAPIKeySet verifies that
// NewSMSSender returns an *AfricasTalkingSMSSender when the env var is
// set. We do NOT exercise the live HTTP path here — the stub contract
// is sufficient for unit tests.
func TestNewSMSSender_AfricasTalkingWhenAPIKeySet(t *testing.T) {
	t.Setenv("AFRICAS_TALKING_API_KEY", "test-fake-key-ATShXXXX")
	t.Setenv("AFRICAS_TALKING_USERNAME", "test-workspace")
	s := NewSMSSender(nil)
	ats, ok := s.(*AfricasTalkingSMSSender)
	if !ok {
		t.Fatalf("expected *AfricasTalkingSMSSender, got %T", s)
	}
	if ats.apiKey != "test-fake-key-ATShXXXX" {
		t.Errorf("apiKey: expected test-fake-key-ATShXXXX, got %s", ats.apiKey)
	}
	if s.Name() != "africas_talking" {
		t.Errorf("expected name=africas_talking, got %s", s.Name())
	}
}

// TestNewSMSSender_SandboxBaseURLWhenUsernameSandbox verifies the
// sandbox vs production base-URL switch is driven by the username.
func TestNewSMSSender_SandboxBaseURLWhenUsernameSandbox(t *testing.T) {
	t.Setenv("AFRICAS_TALKING_API_KEY", "test-fake-key")
	t.Setenv("AFRICAS_TALKING_USERNAME", "sandbox")
	s := NewSMSSender(nil)
	ats, ok := s.(*AfricasTalkingSMSSender)
	if !ok {
		t.Fatalf("expected *AfricasTalkingSMSSender, got %T", s)
	}
	if ats.baseURL != "https://sandbox.africastalking.com" {
		t.Errorf("sandbox baseURL: expected sandbox.africastalking.com, got %s", ats.baseURL)
	}
}

// === StubSMSSender ===

// TestStubSMSSender_CapturesSend verifies the stub records the last
// phone + message so tests can assert what was "sent" without parsing
// log output. Send never returns an error.
func TestStubSMSSender_CapturesSend(t *testing.T) {
	stub := &StubSMSSender{}
	alert, err := stub.Send(context.Background(), "+254712345678", "Hello from Civic Intelligence")
	if err != nil {
		t.Fatalf("stub.Send returned error: %v", err)
	}
	if alert.Status != SMSStatusSent {
		t.Errorf("expected status=sent, got %s", alert.Status)
	}
	if alert.Phone != "+254712345678" {
		t.Errorf("alert phone: expected +254712345678, got %s", alert.Phone)
	}
	if alert.Message != "Hello from Civic Intelligence" {
		t.Errorf("alert message mismatch")
	}
	if !strings.HasPrefix(alert.ID, "sms_") {
		t.Errorf("expected alert ID prefix sms_, got %s", alert.ID)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.LastPhone != "+254712345678" {
		t.Errorf("LastPhone: %s", stub.LastPhone)
	}
	if stub.LastMessage != "Hello from Civic Intelligence" {
		t.Errorf("LastMessage: %s", stub.LastMessage)
	}
	if stub.SendCount != 1 {
		t.Errorf("SendCount: expected 1, got %d", stub.SendCount)
	}
}

// TestStubSMSSender_ConcurrentSafe fans out 50 concurrent Send calls
// and verifies the counter matches — the broadcast handler shares a
// single sender across goroutines, so the stub MUST be concurrency-safe.
func TestStubSMSSender_ConcurrentSafe(t *testing.T) {
	stub := &StubSMSSender{}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = stub.Send(context.Background(), "+254700000000", "msg")
		}()
	}
	wg.Wait()
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.SendCount != 50 {
		t.Errorf("SendCount after 50 concurrent sends: expected 50, got %d", stub.SendCount)
	}
}

// === urlValues helper ===

// TestURLValues_Encode verifies the form-encoder produces
// application/x-www-form-urlencoded output that net/url.ParseQuery can
// decode (so Africa's Talking's gateway will accept it).
func TestURLValues_Encode(t *testing.T) {
	v := urlValues{
		"username": "sandbox",
		"to":       "+254712345678",
		"message":  "Hello World",
		"apiKey":   "ATSh_test_key",
	}
	encoded := v.Encode()
	if !strings.Contains(encoded, "username=sandbox") {
		t.Errorf("missing username=sandbox in %q", encoded)
	}
	if !strings.Contains(encoded, "message=Hello+World") {
		t.Errorf("expected message=Hello+World (space as +), got %q", encoded)
	}
	// Round-trip through net/url.ParseQuery to confirm the encoding is
	// wire-compatible with the gateway's expected parsing.
	parsed, err := url.ParseQuery(encoded)
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	if parsed.Get("message") != "Hello World" {
		t.Errorf("round-trip message: expected 'Hello World', got %q", parsed.Get("message"))
	}
	if parsed.Get("to") != "+254712345678" {
		t.Errorf("round-trip to: expected '+254712345678', got %q", parsed.Get("to"))
	}
	if parsed.Get("username") != "sandbox" {
		t.Errorf("round-trip username: expected 'sandbox', got %q", parsed.Get("username"))
	}
}

// === HTTP handlers ===

// TestSMSSubscribe_CreatesSubscriber verifies that POST
// /api/v1/alerts/sms/subscribe creates a new subscriber and returns
// 201 Created with the subscriber record. The endpoint is PUBLIC
// (no auth required) — a citizen subscribes from the website form
// without signing in.
func TestSMSSubscribe_CreatesSubscriber(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeSMSSubscribeHandler(store)
	body := `{"phone":"+254700000077","country":"KE","keywords":["tender","health"]}`
	rr := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/subscribe", []byte(body), auth.Anonymous())
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var rec SMSSubscriber
	if err := json.Unmarshal(rr.Body.Bytes(), &rec); err != nil {
		t.Fatalf("invalid JSON: %v (body=%s)", err, rr.Body.String())
	}
	if rec.Phone != "+254700000077" {
		t.Errorf("Phone: expected +254700000077, got %s", rec.Phone)
	}
	if rec.Country != "KE" {
		t.Errorf("Country: expected KE, got %s", rec.Country)
	}
	if len(rec.Keywords) != 2 {
		t.Fatalf("expected 2 keywords, got %d (%v)", len(rec.Keywords), rec.Keywords)
	}
	if rec.Keywords[0] != "tender" || rec.Keywords[1] != "health" {
		t.Errorf("keywords: expected [tender health], got %v", rec.Keywords)
	}
	if !rec.Active {
		t.Error("expected Active=true by default")
	}
	if rec.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	// Verify the store actually persisted the record.
	subs := store.List()
	if len(subs) != 1 {
		t.Errorf("expected 1 subscriber in store, got %d", len(subs))
	}
}

// TestSMSSubscribe_DuplicateReturns409 verifies that subscribing the
// same phone twice returns 409 Conflict (NOT 201 Created) with the
// existing record in the body. The duplicate path is NOT an error at
// the store level (idempotency is a concern of the HTTP layer).
func TestSMSSubscribe_DuplicateReturns409(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeSMSSubscribeHandler(store)
	body := `{"phone":"+254700000088","country":"KE","keywords":["tender"]}`
	// First request: 201 Created.
	rr1 := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/subscribe", []byte(body), auth.Anonymous())
	if rr1.Code != http.StatusCreated {
		t.Fatalf("first POST: expected 201, got %d (body=%s)", rr1.Code, rr1.Body.String())
	}
	// Second request: 409 Conflict.
	body2 := `{"phone":"+254700000088","country":"KE","keywords":["health","tax"]}`
	rr2 := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/subscribe", []byte(body2), auth.Anonymous())
	if rr2.Code != http.StatusConflict {
		t.Fatalf("second POST: expected 409, got %d (body=%s)", rr2.Code, rr2.Body.String())
	}
	var resp struct {
		Error      string        `json:"error"`
		Message    string        `json:"message"`
		Subscriber SMSSubscriber `json:"subscriber"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v (body=%s)", err, rr2.Body.String())
	}
	if resp.Error != "already_subscribed" {
		t.Errorf("error code: expected already_subscribed, got %s", resp.Error)
	}
	// The 409 response must echo back the ORIGINAL record (not the new
	// keywords) — the subscriber's keywords are immutable on duplicate.
	if len(resp.Subscriber.Keywords) != 1 || resp.Subscriber.Keywords[0] != "tender" {
		t.Errorf("expected original keywords [tender], got %v", resp.Subscriber.Keywords)
	}
	// Verify the store wasn't mutated by the duplicate request.
	subs := store.List()
	if len(subs) != 1 {
		t.Errorf("expected 1 subscriber in store (no mutation on dup), got %d", len(subs))
	}
}

// TestSMSSubscribe_ValidationErrors covers the five 400 paths.
func TestSMSSubscribe_ValidationErrors(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeSMSSubscribeHandler(store)
	cases := []struct {
		name string
		body string
	}{
		{"invalid JSON", "not json"},
		{"missing phone", `{"keywords":["tender"]}`},
		{"non-E.164 phone", `{"phone":"254712345678","keywords":["tender"]}`},
		{"missing keywords", `{"phone":"+254712345678"}`},
		{"only blank keywords", `{"phone":"+254712345678","keywords":["  ",""]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/subscribe", []byte(c.body), auth.Anonymous())
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d (body=%s)", rr.Code, rr.Body.String())
			}
		})
	}
}

// TestSMSSubscribe_MethodNotAllowed verifies GET returns 405.
func TestSMSSubscribe_MethodNotAllowed(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeSMSSubscribeHandler(store)
	rr := dispatch(handler, http.MethodGet, "/api/v1/alerts/sms/subscribe", nil, auth.Anonymous())
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Allow"), "POST") {
		t.Errorf("Allow header: expected POST, got %s", rr.Header().Get("Allow"))
	}
}

// TestSMSSubscribers_AdminList_Authorized verifies that an admin
// principal (with user:admin scope) can list subscribers.
func TestSMSSubscribers_AdminList_Authorized(t *testing.T) {
	store := NewSMSSubscriberStore()
	_, _, _ = store.Subscribe("+254700000050", "KE", []string{"tender"})
	_, _, _ = store.Subscribe("+256700000050", "UG", []string{"health"})
	handler := makeSMSSubscribersHandler(store, &StubSMSSender{})
	p := auth.Principal{UserID: "admin-1", Scopes: auth.Scopes{auth.ScopeUserAdmin}}
	rr := dispatch(handler, http.MethodGet, "/api/v1/alerts/sms/subscribers", nil, p)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items  []SMSSubscriber `json:"items"`
		Total  int             `json:"total"`
		Sender string          `json:"sender"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v (body=%s)", err, rr.Body.String())
	}
	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
	if resp.Sender != "stub" {
		t.Errorf("expected sender=stub, got %s", resp.Sender)
	}
}

// TestSMSSubscribers_AnonymousRejected verifies the admin scope check
// rejects anonymous callers with 401.
func TestSMSSubscribers_AnonymousRejected(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeSMSSubscribersHandler(store, &StubSMSSender{})
	rr := dispatch(handler, http.MethodGet, "/api/v1/alerts/sms/subscribers", nil, auth.Anonymous())
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for anonymous, got %d", rr.Code)
	}
}

// TestSMSSubscribers_AuthenticatedButNotAdminRejected verifies the
// admin scope check rejects authenticated callers without user:admin
// or notification:write scope with 403.
func TestSMSSubscribers_AuthenticatedButNotAdminRejected(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeSMSSubscribersHandler(store, &StubSMSSender{})
	p := auth.Principal{UserID: "citizen-1"} // no admin scope
	rr := dispatch(handler, http.MethodGet, "/api/v1/alerts/sms/subscribers", nil, p)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rr.Code)
	}
}

// TestSMSSubscribers_FilterByActiveAndCountry verifies the ?active=
// and ?country= filters narrow the result set.
func TestSMSSubscribers_FilterByActiveAndCountry(t *testing.T) {
	store := NewSMSSubscriberStore()
	_, _, _ = store.Subscribe("+254700000060", "KE", []string{"tender"})
	_, _, _ = store.Subscribe("+256700000060", "UG", []string{"tender"})
	_, _, _ = store.Subscribe("+254700000061", "KE", []string{"tender"})
	_ = store.SetActive("+254700000061", false)
	handler := makeSMSSubscribersHandler(store, &StubSMSSender{})
	p := auth.Principal{UserID: "admin-1", Scopes: auth.Scopes{auth.ScopeUserAdmin}}

	// Filter by country=KE.
	rr := dispatch(handler, http.MethodGet, "/api/v1/alerts/sms/subscribers?country=KE", nil, p)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for country=KE, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items []SMSSubscriber `json:"items"`
		Total int             `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("country=KE: expected total=2, got %d", resp.Total)
	}

	// Filter by active=false.
	rr = dispatch(handler, http.MethodGet, "/api/v1/alerts/sms/subscribers?active=false", nil, p)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for active=false, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("active=false: expected total=1, got %d", resp.Total)
	}
}

// TestSMSSend_AdminBroadcast verifies the admin broadcast endpoint
// delivers to every active subscriber and returns a summary with the
// per-recipient delivery log. The StubSMSSender is used so the test
// doesn't hit the Africa's Talking gateway.
func TestSMSSend_AdminBroadcast(t *testing.T) {
	store := NewSMSSubscriberStore()
	_, _, _ = store.Subscribe("+254700000070", "KE", []string{"tender"})
	_, _, _ = store.Subscribe("+254700000071", "KE", []string{"tender"})
	_, _, _ = store.Subscribe("+254700000072", "KE", []string{"health"})

	stub := &StubSMSSender{}
	handler := makeSMSSendHandler(store, stub)
	p := auth.Principal{UserID: "admin-1", Scopes: auth.Scopes{auth.ScopeUserAdmin}}
	body := `{"message":"New tender published: hospital equipment","keywords":["tender"]}`
	rr := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/send", []byte(body), p)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Sent       int        `json:"sent"`
		Failed     int        `json:"failed"`
		Total      int        `json:"total"`
		Sender     string     `json:"sender"`
		Deliveries []SMSAlert `json:"deliveries"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v (body=%s)", err, rr.Body.String())
	}
	if resp.Sent != 2 {
		t.Errorf("expected sent=2 (tender subscribers), got %d", resp.Sent)
	}
	if resp.Failed != 0 {
		t.Errorf("expected failed=0, got %d", resp.Failed)
	}
	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
	if resp.Sender != "stub" {
		t.Errorf("expected sender=stub, got %s", resp.Sender)
	}
	if len(resp.Deliveries) != 2 {
		t.Errorf("expected 2 deliveries, got %d", len(resp.Deliveries))
	}
	alerts := store.Alerts()
	if len(alerts) != 2 {
		t.Errorf("expected 2 alerts in store log, got %d", len(alerts))
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.SendCount != 2 {
		t.Errorf("expected stub.SendCount=2, got %d", stub.SendCount)
	}
}

// TestSMSSend_AnonymousRejected verifies the admin scope check.
func TestSMSSend_AnonymousRejected(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeSMSSendHandler(store, &StubSMSSender{})
	body := `{"message":"hi"}`
	rr := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/send", []byte(body), auth.Anonymous())
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for anonymous, got %d", rr.Code)
	}
}

// TestSMSSend_NonAdminRejected verifies the admin scope check on a
// non-admin authenticated caller.
func TestSMSSend_NonAdminRejected(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeSMSSendHandler(store, &StubSMSSender{})
	p := auth.Principal{UserID: "citizen-1"}
	body := `{"message":"hi"}`
	rr := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/send", []byte(body), p)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rr.Code)
	}
}

// TestSMSSend_ValidationErrors covers the 400 paths.
func TestSMSSend_ValidationErrors(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeSMSSendHandler(store, &StubSMSSender{})
	p := auth.Principal{UserID: "admin-1", Scopes: auth.Scopes{auth.ScopeUserAdmin}}
	cases := []struct {
		name string
		body string
	}{
		{"invalid JSON", "not json"},
		{"empty message", `{"message":""}`},
		{"missing message", `{}`},
		{"over-long message", `{"message":"` + strings.Repeat("x", 161) + `"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/send", []byte(c.body), p)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d (body=%s)", rr.Code, rr.Body.String())
			}
		})
	}
}

// TestSMSSend_NoMatchingSubscribers verifies the broadcast returns a
// 200 with sent=0 when no subscribers match the supplied keywords
// (instead of 404 — the broadcast endpoint is conceptually a
// fire-and-forget).
func TestSMSSend_NoMatchingSubscribers(t *testing.T) {
	store := NewSMSSubscriberStore()
	_, _, _ = store.Subscribe("+254700000080", "KE", []string{"tender"})
	handler := makeSMSSendHandler(store, &StubSMSSender{})
	p := auth.Principal{UserID: "admin-1", Scopes: auth.Scopes{auth.ScopeUserAdmin}}
	body := `{"message":"test","keywords":["nonexistent_keyword"]}`
	rr := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/send", []byte(body), p)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Sent  int `json:"sent"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Sent != 0 || resp.Total != 0 {
		t.Errorf("expected sent=0 total=0, got sent=%d total=%d", resp.Sent, resp.Total)
	}
}

// === CountingSMSSender (mock that fails on demand) ===

// countingSMSSender is a test-only SMSSender that lets tests assert on
// the number of Send calls AND simulate failures (e.g. to exercise the
// "failed" counter in the broadcast response).
type countingSMSSender struct {
	mu        sync.Mutex
	calls     int
	failOnNth int // 1-indexed; calls at or above this index fail
}

func (c *countingSMSSender) Name() string { return "counting" }

func (c *countingSMSSender) Send(_ context.Context, phone, message string) (SMSAlert, error) {
	c.mu.Lock()
	c.calls++
	n := c.calls
	c.mu.Unlock()
	if c.failOnNth > 0 && n >= c.failOnNth {
		return SMSAlert{
			ID: newSMSAlertID(), Phone: phone, Message: message,
			SentAt:      time.Now().UTC(),
			Status:      SMSStatusFailed,
			SourceEvent: "test",
		}, errors.New("simulated gateway failure")
	}
	return SMSAlert{
		ID: newSMSAlertID(), Phone: phone, Message: message,
		SentAt:      time.Now().UTC(),
		Status:      SMSStatusSent,
		SourceEvent: "test",
	}, nil
}

// TestSMSSend_PartialFailure verifies that when the SMSSender fails on
// one recipient, the broadcast continues to deliver to the others and
// reports both counts in the response. This is the invariant that lets
// us trust the broadcast endpoint under partial outages.
func TestSMSSend_PartialFailure(t *testing.T) {
	store := NewSMSSubscriberStore()
	_, _, _ = store.Subscribe("+254700000090", "KE", []string{"tender"})
	_, _, _ = store.Subscribe("+254700000091", "KE", []string{"tender"})
	_, _, _ = store.Subscribe("+254700000092", "KE", []string{"tender"})
	sender := &countingSMSSender{failOnNth: 2}
	handler := makeSMSSendHandler(store, sender)
	p := auth.Principal{UserID: "admin-1", Scopes: auth.Scopes{auth.ScopeUserAdmin}}
	body := `{"message":"partial failure test","keywords":["tender"]}`
	rr := dispatch(handler, http.MethodPost, "/api/v1/alerts/sms/send", []byte(body), p)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Sent   int `json:"sent"`
		Failed int `json:"failed"`
		Total  int `json:"total"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	if resp.Sent+resp.Failed != 3 {
		t.Errorf("sent+failed: expected 3, got %d", resp.Sent+resp.Failed)
	}
	if resp.Failed == 0 {
		t.Errorf("expected at least 1 failure, got 0")
	}
}
