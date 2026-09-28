// Package main — tests for the USSD session handler (issue #289).
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// --- Helpers ---

// ussdJSONDispatch fires a USSD request with a JSON body through the
// handler, bypassing OptionalAuth (USSD is anonymous by design — AT
// calls the callback without a bearer token). Returns the raw body +
// the JSON-parsed session when the response is JSON.
func ussdJSONDispatch(handler http.Handler, body string) (*httptest.ResponseRecorder, *ussdResponse) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ussd", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	var session ussdResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &session)
	return rr, &session
}

// ussdFormDispatch fires a USSD request with a form-encoded body
// through the handler — the production Content-Type Africa's Talking
// uses. Returns the raw response body (a CON/END-prefixed text string).
func ussdFormDispatch(handler http.Handler, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ussd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// === Required tests ===

// TestUSSDHandler_ReturnsMenu verifies that an empty `text` (the first
// screen of a USSD session) returns the top-level menu with the four
// documented options. The response MUST start with "CON " so
// Africa's Talking knows the session continues.
func TestUSSDHandler_ReturnsMenu(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeUSSDHandler(store)
	body := `{"session_id":"ATPid_123","phone_number":"+254712345678","service_code":"*384*99#","text":""}`
	rr, session := ussdJSONDispatch(handler, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	// The response body MUST be JSON for application/json callers.
	if !strings.Contains(rr.Header().Get("Content-Type"), "application/json") {
		t.Errorf("expected JSON Content-Type, got %s", rr.Header().Get("Content-Type"))
	}
	// The Response field must start with "CON " — Africa's Talking's
	// session-continue marker.
	if !strings.HasPrefix(session.Response, "CON ") {
		t.Errorf("expected response to start with 'CON ', got %q", session.Response)
	}
	// Every documented menu option must appear.
	wantOptions := []string{
		"1. Latest Bills",
		"2. My MP",
		"3. Ask a question",
		"4. Subscribe to alerts",
		"0. Exit",
	}
	for _, opt := range wantOptions {
		if !strings.Contains(session.Response, opt) {
			t.Errorf("expected response to contain %q, got %q", opt, session.Response)
		}
	}
	// Session-scoped fields echo back.
	if session.Phone != "+254712345678" {
		t.Errorf("expected Phone +254712345678, got %s", session.Phone)
	}
	if session.Text != "" {
		t.Errorf("expected Text '', got %q", session.Text)
	}
}

// TestUSSDHandler_BillsOption verifies that selecting option 1 (Latest
// Bills) returns an END-terminated response with the latest Bill
// titles. The session MUST end (END prefix) because there's no further
// input to collect — the citizen reads the list and exits.
func TestUSSDHandler_BillsOption(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeUSSDHandler(store)
	body := `{"session_id":"ATPid_456","phone_number":"+254712345678","text":"1"}`
	rr, session := ussdJSONDispatch(handler, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	// MUST be END-terminated.
	if !strings.HasPrefix(session.Response, "END ") {
		t.Errorf("expected response to start with 'END ', got %q", session.Response)
	}
	// MUST mention "Bills" (the menu's title).
	if !strings.Contains(session.Response, "Bill") {
		t.Errorf("expected response to mention 'Bill', got %q", session.Response)
	}
	// MUST contain at least one numbered entry (1., 2., 3.).
	if !strings.Contains(session.Response, "1.") {
		t.Errorf("expected response to contain a numbered Bill list, got %q", session.Response)
	}
}

// === Form-encoded path (production Content-Type) ===

// TestUSSDHandler_FormEncoded_ReturnsMenu verifies the production path
// — Africa's Talking sends a form-encoded body, and we return a
// text/plain response with the CON/END prefix.
func TestUSSDHandler_FormEncoded_ReturnsMenu(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeUSSDHandler(store)
	form := url.Values{
		"sessionId":   {"ATPid_789"},
		"phoneNumber": {"+254712345678"},
		"serviceCode": {"*384*99#"},
		"text":        {""},
	}
	rr := ussdFormDispatch(handler, form)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("expected text/plain Content-Type, got %s", rr.Header().Get("Content-Type"))
	}
	body := rr.Body.String()
	if !strings.HasPrefix(body, "CON ") {
		t.Errorf("expected body to start with 'CON ', got %q", body)
	}
	if !strings.Contains(body, "1. Latest Bills") {
		t.Errorf("expected body to contain '1. Latest Bills', got %q", body)
	}
}

// TestUSSDHandler_FormEncoded_BillsOption verifies the form-encoded
// path for option 1 (Latest Bills).
func TestUSSDHandler_FormEncoded_BillsOption(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeUSSDHandler(store)
	form := url.Values{
		"sessionId":   {"ATPid_form_bills"},
		"phoneNumber": {"+254712345678"},
		"serviceCode": {"*384*99#"},
		"text":        {"1"},
	}
	rr := ussdFormDispatch(handler, form)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.HasPrefix(rr.Body.String(), "END ") {
		t.Errorf("expected END-prefixed response, got %q", rr.Body.String())
	}
}

// === Dispatch state machine (covers all 4 menu options + exit) ===

// TestDispatchUSSD_TopLevelMenu verifies that an empty text returns
// the top-level menu (CON-prefixed).
func TestDispatchUSSD_TopLevelMenu(t *testing.T) {
	store := NewSMSSubscriberStore()
	resp := dispatchUSSD("", "+254712345678", store)
	if !strings.HasPrefix(resp, "CON ") {
		t.Errorf("expected CON prefix, got %q", resp)
	}
	for _, opt := range []string{"1. Latest Bills", "2. My MP", "3. Ask a question", "4. Subscribe to alerts"} {
		if !strings.Contains(resp, opt) {
			t.Errorf("expected menu to contain %q, got %q", opt, resp)
		}
	}
}

// TestDispatchUSSD_MyMPOption verifies that option 2 returns the
// citizen's MP info based on their phone number's country prefix.
func TestDispatchUSSD_MyMPOption(t *testing.T) {
	store := NewSMSSubscriberStore()
	// Kenya prefix → KE Speaker + President.
	resp := dispatchUSSD("2", "+254712345678", store)
	if !strings.HasPrefix(resp, "END ") {
		t.Errorf("expected END prefix, got %q", resp)
	}
	if !strings.Contains(resp, "Moses Wetangula") {
		t.Errorf("expected Kenyan Speaker, got %q", resp)
	}
	if !strings.Contains(resp, "William Ruto") {
		t.Errorf("expected Kenyan President, got %q", resp)
	}
	// Uganda prefix → UG Speaker + President.
	resp = dispatchUSSD("2", "+256712345678", store)
	if !strings.Contains(resp, "Anita Among") {
		t.Errorf("expected Ugandan Speaker, got %q", resp)
	}
}

// TestDispatchUSSD_AskOption verifies option 3 prompts for a question,
// then acknowledges receipt when the citizen types their question.
func TestDispatchUSSD_AskOption(t *testing.T) {
	store := NewSMSSubscriberStore()
	// First screen: prompt for the question.
	resp := dispatchUSSD("3", "+254712345678", store)
	if !strings.HasPrefix(resp, "CON ") {
		t.Errorf("expected CON prefix for first screen, got %q", resp)
	}
	if !strings.Contains(resp, "question") {
		t.Errorf("expected prompt to mention 'question', got %q", resp)
	}
	// Second screen: citizen types "What is the status of Bill 23?".
	resp = dispatchUSSD("3*What is the status of Bill 23", "+254712345678", store)
	if !strings.HasPrefix(resp, "END ") {
		t.Errorf("expected END prefix for acknowledgement, got %q", resp)
	}
	if !strings.Contains(resp, "Question received") {
		t.Errorf("expected acknowledgement, got %q", resp)
	}
}

// TestDispatchUSSD_SubscribeOption verifies option 4 opts the citizen
// into SMS alerts via the SMSSubscriberStore, and that selecting it
// twice returns the "already subscribed" message (no duplicate row).
func TestDispatchUSSD_SubscribeOption(t *testing.T) {
	store := NewSMSSubscriberStore()
	// First call: subscribes the citizen.
	resp := dispatchUSSD("4", "+254712345678", store)
	if !strings.HasPrefix(resp, "END ") {
		t.Errorf("expected END prefix, got %q", resp)
	}
	if !strings.Contains(resp, "Subscribed") {
		t.Errorf("expected 'Subscribed' message, got %q", resp)
	}
	// Verify the store has the subscriber.
	subs := store.List()
	if len(subs) != 1 {
		t.Errorf("expected 1 subscriber after USSD subscribe, got %d", len(subs))
	}
	if subs[0].Phone != "+254712345678" {
		t.Errorf("expected subscriber phone +254712345678, got %s", subs[0].Phone)
	}
	// Second call: returns "already subscribed".
	resp = dispatchUSSD("4", "+254712345678", store)
	if !strings.Contains(resp, "already subscribed") {
		t.Errorf("expected 'already subscribed' message, got %q", resp)
	}
	// Store must NOT have a duplicate.
	subs = store.List()
	if len(subs) != 1 {
		t.Errorf("expected 1 subscriber after duplicate USSD subscribe, got %d", len(subs))
	}
}

// TestDispatchUSSD_ExitOption verifies option 0 returns the
// "Thank you" END message.
func TestDispatchUSSD_ExitOption(t *testing.T) {
	store := NewSMSSubscriberStore()
	resp := dispatchUSSD("0", "+254712345678", store)
	if !strings.HasPrefix(resp, "END ") {
		t.Errorf("expected END prefix, got %q", resp)
	}
	if !strings.Contains(resp, "Thank you") {
		t.Errorf("expected 'Thank you' in exit message, got %q", resp)
	}
}

// TestDispatchUSSD_InvalidOption verifies that an unrecognised option
// returns an END-terminated "Invalid option" message (so the citizen
// isn't trapped in a loop paying per keystroke).
func TestDispatchUSSD_InvalidOption(t *testing.T) {
	store := NewSMSSubscriberStore()
	resp := dispatchUSSD("9", "+254712345678", store)
	if !strings.HasPrefix(resp, "END ") {
		t.Errorf("expected END prefix for invalid option, got %q", resp)
	}
	if !strings.Contains(resp, "Invalid option") {
		t.Errorf("expected 'Invalid option' message, got %q", resp)
	}
}

// === Helpers ===

// TestCountryFromPhonePrefix verifies the coarse phone-prefix → country
// mapping for every country the platform ships an adapter for.
func TestCountryFromPhonePrefix(t *testing.T) {
	cases := []struct {
		phone string
		want  string
	}{
		{"+254712345678", "KE"},
		{"+256712345678", "UG"},
		{"+255712345678", "TZ"},
		{"+233200000000", "GH"},
		{"+234800000000", "NG"},
		{"+27820000000", "ZA"},
		{"+99999999", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := countryFromPhonePrefix(c.phone)
		if got != c.want {
			t.Errorf("countryFromPhonePrefix(%q): expected %s, got %s", c.phone, c.want, got)
		}
	}
}

// TestUSSDHandler_MethodNotAllowed verifies GET returns 405.
func TestUSSDHandler_MethodNotAllowed(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeUSSDHandler(store)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ussd", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Allow"), "POST") {
		t.Errorf("Allow header: expected POST, got %s", rr.Header().Get("Allow"))
	}
}

// TestUSSDHandler_InvalidJSON verifies that a malformed JSON body
// returns 400 (the form-encoded path also returns 400 for malformed
// form bodies).
func TestUSSDHandler_InvalidJSON(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeUSSDHandler(store)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ussd", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestUSSDHandler_PhoneNormalisedWithoutPlus verifies the handler
// prepends '+' when Africa's Talking omits it (sandbox behaviour).
func TestUSSDHandler_PhoneNormalisedWithoutPlus(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeUSSDHandler(store)
	body := `{"session_id":"ATPid_1","phone_number":"254712345678","text":"2"}`
	rr, session := ussdJSONDispatch(handler, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if session.Phone != "+254712345678" {
		t.Errorf("expected phone normalised to +254712345678, got %s", session.Phone)
	}
	// The Kenya MP info should appear (proving the normalised prefix
	// was used for the country lookup).
	if !strings.Contains(session.Response, "Moses Wetangula") {
		t.Errorf("expected Kenyan Speaker in response, got %q", session.Response)
	}
}

// TestUSSDHandler_EmptyQuestion verifies option 3 with an empty
// question segment returns a friendly END message instead of accepting
// a blank question.
func TestUSSDHandler_EmptyQuestion(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeUSSDHandler(store)
	body := `{"session_id":"ATPid_empty","phone_number":"+254712345678","text":"3*  "}`
	rr, session := ussdJSONDispatch(handler, body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.HasPrefix(session.Response, "END ") {
		t.Errorf("expected END prefix for empty question, got %q", session.Response)
	}
	if !strings.Contains(session.Response, "empty") {
		t.Errorf("expected 'empty' message, got %q", session.Response)
	}
}

// TestUSSDHandler_AnonymousNoAuthRequired verifies the handler does
// NOT require an Authorization header — Africa's Talking calls the
// callback without one. We confirm by NOT setting the Bearer token
// in dispatch (the production form-encoded path doesn't go through
// OptionalAuth).
func TestUSSDHandler_AnonymousNoAuthRequired(t *testing.T) {
	store := NewSMSSubscriberStore()
	handler := makeUSSDHandler(store)
	// Anonymous dispatch (no OptionalAuth wrapping — the USSD endpoint
	// is registered on apiHandler without auth middleware in main.go).
	form := url.Values{
		"sessionId":   {"ATPid_anon"},
		"phoneNumber": {"+254712345678"},
		"text":        {""},
	}
	rr := ussdFormDispatch(handler, form)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for anonymous USSD, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestLatestSeedBills verifies the seed Bill titles are returned
// (the function is the source for the USSD Bills menu).
func TestLatestSeedBills(t *testing.T) {
	bills := latestSeedBills(3)
	if len(bills) != 3 {
		t.Fatalf("expected 3 bills, got %d", len(bills))
	}
	for _, b := range bills {
		if !strings.Contains(b, "Bill") {
			t.Errorf("expected bill title to contain 'Bill', got %q", b)
		}
	}
	// n=0 returns nil.
	if latestSeedBills(0) != nil {
		t.Error("expected nil for n=0")
	}
	// n > len(bills) returns the full slice (no panic).
	all := latestSeedBills(100)
	if len(all) != 3 {
		t.Errorf("expected 3 bills (clamped), got %d", len(all))
	}
}
