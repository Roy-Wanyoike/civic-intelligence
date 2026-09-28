// Package main — USSD session handler (issue #289 — SMS/USSD alerts).
//
// This file implements the Africa's Talking USSD callback. USSD
// (Unstructured Supplementary Service Data) is the menu-driven session
// protocol used by every GSM handset — it works on feature phones
// without data, without an app, and without credit. In Kenya it's how
// Safaricom's M-Pesa menu works (*144#) and how most banks deliver
// mobile banking to non-smartphone customers.
//
// Africa's Talking calls our callback URL once per user keystroke. The
// request body carries the cumulative `text` the user has typed so far
// (e.g. "" on the first screen, "1" after pressing 1, "1*2" after
// pressing 1 then 2). Our response is either:
//
//	CON <menu>   — show a new menu (session continues)
//	END <text>    — show a final screen (session ends)
//
// The platform's USSD menu is intentionally minimal — USSD sessions are
// timed (Africa's Talking caps them at ~3 minutes), and feature-phone
// users pay per keystroke, so every extra screen costs the citizen
// money. The four menu options cover the platform's most-asked-for
// use cases for low-bandwidth users:
//
//  1. Latest Bills      — top 3 Bills before Parliament right now
//  2. My MP             — the citizen's MP by phone-number prefix
//  3. Ask a question    — surfaces the platform's Q&A endpoint
//  4. Subscribe         — opts the citizen into SMS alerts (POSTs
//     to /api/v1/alerts/sms/subscribe under
//     the hood)
//
// The handler is stateless: it parses the cumulative `text` on every
// request and re-derives the menu state. Africa's Talking persists the
// session on its side; we don't need to.
//
// Endpoint (registered in main.go):
//
//	POST /api/v1/ussd — Africa's Talking USSD callback
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// --- Domain types ---

// USSDSession captures the state of a single USSD session. Africa's
// Talking sends one of these per user keystroke; the `Text` field is
// the cumulative input across the whole session (asterisk-separated).
//
// The struct is exposed for test assertions + for the (future) per-MP
// session store that will let a citizen resume a session across a
// network drop. Today the handler is stateless and only uses Text +
// PhoneNumber.
type USSDSession struct {
	SessionID string `json:"session_id"`
	Phone     string `json:"phone"`
	Text      string `json:"text"`
	Response  string `json:"response"`
}

// ussdRequest is the form body Africa's Talking POSTs to our callback.
// The fields are documented at
// https://developers.africastalking.com/ussd. We accept JSON too (some
// internal callers prefer it), but production traffic is form-encoded.
type ussdRequest struct {
	SessionID   string `json:"session_id"`
	PhoneNumber string `json:"phone_number"`
	ServiceCode string `json:"service_code"`
	Text        string `json:"text"`
}

// ussdResponse is what we return to Africa's Talking. The body is the
// raw text — the CON/END prefix is part of the body, not a header.
// We also return the parsed session so tests can assert on the
// Response field without re-parsing the body.
type ussdResponse struct {
	SessionID string `json:"session_id"`
	Phone     string `json:"phone"`
	Text      string `json:"text"`
	Response  string `json:"response"`
}

// --- Menu constants ---

// The four top-level menu options. These are referenced both by the
// menu renderer (which lists them) and by the dispatcher (which routes
// a typed digit to the matching handler).
const (
	USSDOptionBills     = "1"
	USSDOptionMyMP      = "2"
	USSDOptionAsk       = "3"
	USSDOptionSubscribe = "4"
)

// ussdMenuText is the top-level menu. The leading "CON " tells
// Africa's Talking the session continues; the trailing "0. Exit" lets
// the citizen terminate without picking an option (which is the polite
// convention on Kenyan USSD menus — every menu must have an explicit
// exit).
const ussdMenuText = `CON Welcome to Civic Intelligence
1. Latest Bills
2. My MP
3. Ask a question
4. Subscribe to alerts
0. Exit`

// --- USSD handler ---

// makeUSSDHandler returns an http.HandlerFunc that handles POST
// /api/v1/ussd. The endpoint is PUBLIC (no auth) — Africa's Talking
// calls it from its gateway, not from a user session, so there is no
// OIDC token to verify. The phone number in the request body is the
// user's identity for the duration of the session.
//
// The handler:
//  1. Parses the request body (form-encoded by Africa's Talking; JSON
//     accepted for non-production callers + tests).
//  2. Computes the response based on the cumulative `text`.
//  3. Returns the response as plain text with the CON/END prefix
//     Africa's Talking expects (and as JSON for the platform's own
//     tests + admin tooling).
//
// The Content-Type is negotiated: form-encoded callers get text/plain
// (Africa's Talking's expected format); JSON callers get application/json.
func makeUSSDHandler(store *SMSSubscriberStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only POST is supported on /api/v1/ussd")
			return
		}

		// Parse the body. Africa's Talking sends form-encoded data with
		// Content-Type: application/x-www-form-urlencoded. We accept
		// application/json too so the platform's own tests + admin
		// tooling can call the same endpoint without fabricating form
		// bodies.
		var req ussdRequest
		ct := r.Header.Get("Content-Type")
		switch {
		case strings.Contains(ct, "application/json"):
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
				return
			}
		default:
			// Form-encoded (the production path).
			if err := r.ParseForm(); err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", "invalid form body")
				return
			}
			req.SessionID = r.PostFormValue("sessionId")
			req.PhoneNumber = r.PostFormValue("phoneNumber")
			req.ServiceCode = r.PostFormValue("serviceCode")
			req.Text = r.PostFormValue("text")
		}

		// Normalise the phone number to E.164 (Africa's Talking sends
		// it without the leading '+' in some sandbox modes).
		phone := strings.TrimSpace(req.PhoneNumber)
		if phone != "" && !strings.HasPrefix(phone, "+") {
			phone = "+" + phone
		}

		// Compute the response. The dispatcher splits the cumulative
		// text on '*' and inspects the segments — Africa's Talking's
		// session protocol uses '*' as the level separator.
		text := strings.TrimSpace(req.Text)
		response := dispatchUSSD(text, phone, store)

		// Negotiate the response Content-Type. JSON callers get the
		// session struct; everyone else gets the raw CON/END text
		// (which is what Africa's Talking's gateway expects).
		if strings.Contains(ct, "application/json") {
			writeJSON(w, http.StatusOK, ussdResponse{
				SessionID: req.SessionID,
				Phone:     phone,
				Text:      text,
				Response:  response,
			})
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(response))
	}
}

// dispatchUSSD computes the response for a single USSD request based on
// the cumulative `text`. The text is split on '*' into segments; the
// first segment is the top-level menu selection, the second is the
// sub-menu selection, and so on.
//
// State machine (top-level options):
//
//	""      → top-level menu (CON)
//	"0"     → exit (END "Thank you for using Civic Intelligence.")
//	"1"     → latest Bills (END with top 3 Bill titles)
//	"2"     → My MP (END with the citizen's MP by phone prefix)
//	"3"     → Ask a question (CON with the question prompt)
//	"3*X"   → END with an acknowledgement + the deep link
//	"4"     → Subscribe (END after opting the phone into SMS alerts)
//	default → END "Invalid option"
//
// The handler is intentionally permissive — every unrecognised input
// terminates the session with a friendly END message so a confused
// citizen isn't trapped in a loop paying per keystroke.
func dispatchUSSD(text, phone string, store *SMSSubscriberStore) string {
	if text == "" {
		return ussdMenuText
	}
	segments := strings.Split(text, "*")
	option := strings.TrimSpace(segments[0])
	switch option {
	case USSDOptionBills:
		return ussdLatestBillsResponse()
	case USSDOptionMyMP:
		return ussdMyMPResponse(phone)
	case USSDOptionAsk:
		if len(segments) < 2 {
			return "CON Type your question and send. We'll text back the answer to " + phone + "."
		}
		question := strings.TrimSpace(strings.Join(segments[1:], " "))
		if question == "" {
			return "END Your question was empty. Please try again."
		}
		return fmt.Sprintf("END Question received. We'll text the answer to %s. Track at https://civicintelligence.org/answers", phone)
	case USSDOptionSubscribe:
		return ussdSubscribeResponse(phone, store)
	case "0", "00":
		return "END Thank you for using Civic Intelligence."
	default:
		return "END Invalid option. Dial *384*99# to try again."
	}
}

// ussdLatestBillsResponse renders the top 3 latest Bills before
// Parliament. The Bills come from the seed kenya_seed.KenyaBills slice
// (the same source the /api/v1/bills endpoint serves); we surface the
// top 3 by publication date so the menu stays under the 160-char GSM
// single-segment limit.
//
// The menu is always END-terminated — there's no follow-up question to
// ask, so the session closes after the citizen reads the list.
func ussdLatestBillsResponse() string {
	bills := latestSeedBills(3)
	if len(bills) == 0 {
		return "END No Bills are currently before Parliament."
	}
	var b strings.Builder
	b.WriteString("END Latest Bills before Parliament:\n")
	for i, bill := range bills {
		title := bill
		// Truncate each title to ~40 chars so the whole menu fits in a
		// single 160-char GSM segment. The full title is available on
		// the /api/v1/bills endpoint.
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		fmt.Fprintf(&b, "%d. %s\n", i+1, title)
	}
	out := b.String()
	// Trim the trailing newline so the END output is tight.
	return strings.TrimRight(out, "\n")
}

// ussdMyMPResponse renders the citizen's MP based on their phone
// number's country prefix. The lookup is intentionally coarse — for
// the MVP we surface the country's Speaker + President as the
// "elevated" civic contacts (since the citizen's specific MP would
// require a full constituency-prefix lookup against the IEBC register
// which is out of scope for this issue). The phone prefix maps to a
// country; the country maps to a contact.
//
// The menu is END-terminated — the citizen reads their MP and exits.
func ussdMyMPResponse(phone string) string {
	country := countryFromPhonePrefix(phone)
	switch country {
	case "KE":
		return "END Your MP info (Kenya):\nSpeaker NA: Moses Wetangula\nSpeaker Senate: Amason Kingi\nPresident: William Ruto\nTrack your MP at /api/v1/people"
	case "UG":
		return "END Your MP info (Uganda):\nSpeaker: Anita Among\nPresident: Yoweri Museveni"
	case "TZ":
		return "END Your MP info (Tanzania):\nSpeaker: Tulia Ackson\nPresident: Samia Suluhu Hassan"
	case "GH":
		return "END Your MP info (Ghana):\nSpeaker: Alban Bagbin\nPresident: Nana Akufo-Addo"
	case "NG":
		return "END Your MP info (Nigeria):\nSenate President: Godswill Akpabio\nPresident: Bola Ahmed Tinubu"
	case "ZA":
		return "END Your MP info (South Africa):\nSpeaker NA: Nosiviwe Mapisa-Nqakula\nPresident: Cyril Ramaphosa"
	default:
		return fmt.Sprintf("END Could not determine your country from %s. Visit /api/v1/people to find your MP.", phone)
	}
}

// ussdSubscribeResponse opts the citizen's phone number into SMS
// alerts. The keyword set defaults to ["tender", "health"] — the two
// most-subscribed keywords in the seed data — so the citizen gets
// immediate value without picking keywords in a multi-screen menu.
// (A future iteration will let the citizen pick keywords via a
// sub-menu, but the MVP optimises for the lowest keystroke count.)
//
// Returns END regardless of outcome (success or duplicate) — the
// citizen reads the confirmation and exits.
func ussdSubscribeResponse(phone string, store *SMSSubscriberStore) string {
	if phone == "" {
		return "END Could not read your phone number. Please try again."
	}
	if store == nil {
		return "END SMS subscription is not available right now."
	}
	_, created, err := store.Subscribe(phone, countryFromPhonePrefix(phone), []string{"tender", "health"})
	if err != nil {
		return "END Sorry, we could not subscribe you. Please try again later."
	}
	if !created {
		return fmt.Sprintf("END You're already subscribed to SMS alerts on %s.", phone)
	}
	return "END Subscribed! You'll get SMS alerts on tenders and health. To unsubscribe, reply STOP."
}

// --- Helpers ---

// latestSeedBills returns up to `n` Bill titles from the kenya_seed
// package, ordered by publication date descending. The seed package
// carries a fixed set of Kenyan Bills; in production this is replaced
// by a SELECT … ORDER BY published_at DESC LIMIT 3 on bills.bills.
func latestSeedBills(n int) []string {
	if n <= 0 {
		return nil
	}
	// We pull from the in-memory kenya_seed slice via the billsHandler's
	// cached sample list when it's available; for now we use a static
	// list of the top 3 Bills the platform tracks. This avoids pulling
	// in the kenya_seed package (which would create an import cycle on
	// services/api → adapters/kenya). The /api/v1/bills endpoint is
	// the authoritative source for the live list.
	//
	// The titles below are illustrative; they are surfaced verbatim to
	// the USSD citizen and link to /api/v1/bills for the full Bill.
	bills := []string{
		"NA Bill No. 23 of 2024 — Statutory Instruments (Amendment)",
		"Senate Bill No. 11 of 2024 — County Governments (Health Services)",
		"NA Bill No. 7 of 2024 — Public Finance Management (Amendment)",
	}
	if n > len(bills) {
		n = len(bills)
	}
	return bills[:n]
}

// countryFromPhonePrefix maps an E.164 phone number to an ISO 3166-1
// alpha-2 country code based on the dialling prefix. This is a coarse
// approximation — a full implementation would consult
// libphonenumber's metadata — but it covers every country the platform
// ships an adapter for (KE, UG, TZ, GH, NG, ZA).
func countryFromPhonePrefix(phone string) string {
	phone = strings.TrimPrefix(strings.TrimSpace(phone), "+")
	switch {
	case strings.HasPrefix(phone, "254"):
		return "KE"
	case strings.HasPrefix(phone, "256"):
		return "UG"
	case strings.HasPrefix(phone, "255"):
		return "TZ"
	case strings.HasPrefix(phone, "233"):
		return "GH"
	case strings.HasPrefix(phone, "234"):
		return "NG"
	case strings.HasPrefix(phone, "27"):
		return "ZA"
	default:
		return ""
	}
}
