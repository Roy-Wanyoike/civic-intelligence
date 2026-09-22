// Package main — email alert delivery (issue #279 — Email alert subscriptions).
//
// This file wires the Civic Intelligence platform to the Resend email API
// (https://resend.com) — the simplest email API. Resend was chosen because
// it exposes a single REST endpoint (POST https://api.resend.com/emails)
// with one auth header (Bearer RESEND_API_KEY) and one JSON body, so the
// integration needs no SDK, no client library, and no message queue. The
// whole sender is a few hundred lines of plain net/http.
//
// EmailSender is an interface so tests can substitute a StubEmailSender
// (which only logs) — the platform never sends real emails in unit tests.
//
// Selection at startup:
//
//	if os.Getenv("RESEND_API_KEY") != "" → ResendEmailSender
//	else                                  → StubEmailSender
//
// The daily digest pipeline (see digest.go) calls EmailSender.Send with
// the HTML body produced by renderDigestEmail. The refresh handler
// (services/api/cmd/main.go:makeRefreshHandler) is the only production
// caller of the digest pipeline today — Vercel's daily cron hits
// /api/v1/refresh at 03:00 UTC (see vercel.json).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Notification channels. A subscription's Channels slice controls which
// delivery mechanisms fire when a notification matches. The platform
// always writes the in-app notification row (so the /notifications page
// has the alert); the email channel additionally sends an email when
// "email" is present in the slice.
//
// Channels is stored as a slice (not a bitfield) so the OpenAPI contract
// can extend to "sms", "push", "webhook" etc. without a schema migration.
const (
	ChannelInApp  = "in_app"
	ChannelEmail  = "email"
)

// validChannels is the allow-list of channel names a subscription may
// carry. Unknown names are rejected by the PATCH handler so a typo (e.g.
// "emial") does not silently disable delivery.
var validChannels = map[string]bool{
	ChannelInApp: true,
	ChannelEmail: true,
}

// defaultChannels is the channel set assigned to a brand-new follow that
// does not specify one explicitly. The in-app channel is always on (the
// notification row is always written); the email channel is opt-in.
var defaultChannels = []string{ChannelInApp}

// EmailSender is the abstraction every email-bearing code path depends on.
// Implementations:
//   - *ResendEmailSender — POSTs to https://api.resend.com/emails.
//   - *StubEmailSender   — logs the to+subject and discards the body.
//
// Send MUST be safe for concurrent use; the daily digest pipeline fans out
// one goroutine per recipient, so a single sender is shared across all of
// them.
type EmailSender interface {
	// Send delivers an HTML email to a single recipient. The ctx is
	// honoured so the caller (e.g. the refresh handler) can bound the
	// total digest send time. A non-nil error means the email was NOT
	// accepted by the upstream provider — callers SHOULD log + continue
	// rather than abort the whole batch (one bad address must not block
	// the other subscribers).
	Send(ctx context.Context, to, subject, htmlBody string) error

	// Name returns a short identifier for logs + diagnostics (e.g.
	// "resend", "stub"). The refresh handler logs which sender it used
	// so operators can confirm whether RESEND_API_KEY was set.
	Name() string
}

// NewEmailSender picks an implementation based on the RESEND_API_KEY env
// var. When the var is set, a *ResendEmailSender is returned; otherwise
// a *StubEmailSender is returned. The optional httpClient parameter
// exists so tests can inject an httptest.Server-tied client; production
// callers pass nil and a default client with a 10-second timeout is used.
func NewEmailSender(httpClient *http.Client) EmailSender {
	apiKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	if apiKey == "" {
		log.Printf("email: RESEND_API_KEY not set — using StubEmailSender (no emails will be sent)")
		return &StubEmailSender{}
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	log.Printf("email: RESEND_API_KEY detected — using ResendEmailSender")
	return &ResendEmailSender{
		apiKey:  apiKey,
		client:  httpClient,
		baseURL: "https://api.resend.com",
		// FromEmail defaults to the documented Resend sandbox address
		// ("onboarding@resend.dev") so the platform can send mail before
		// a verified sending domain is configured. Override with the
		// RESEND_FROM_EMAIL env var in production.
		fromEmail: firstNonEmpty(os.Getenv("RESEND_FROM_EMAIL"), "onboarding@resend.dev"),
	}
}

// --- ResendEmailSender ---

// ResendEmailSender POSTs emails to https://api.resend.com/emails. The
// request body is JSON of the form:
//
//	{
//	  "from":    "onboarding@resend.dev",
//	  "to":      "citizen@example.com",
//	  "subject": "Today in Civic Intelligence",
//	  "html":    "<html>…</html>"
//	}
//
// The Authorization header carries "Bearer <RESEND_API_KEY>". A 200/201
// response from Resend means the message was queued — delivery to the
// recipient's inbox is asynchronous on Resend's side. Any non-2xx status
// is surfaced as an error containing the upstream body so an operator can
// see why Resend rejected the request.
type ResendEmailSender struct {
	apiKey    string
	client    *http.Client
	baseURL   string
	fromEmail string
}

// Name implements EmailSender.
func (s *ResendEmailSender) Name() string { return "resend" }

// Send implements EmailSender.
func (s *ResendEmailSender) Send(ctx context.Context, to, subject, htmlBody string) error {
	if strings.TrimSpace(to) == "" {
		return errors.New("email: recipient (to) is required")
	}
	if strings.TrimSpace(subject) == "" {
		return errors.New("email: subject is required")
	}
	if s.apiKey == "" {
		return errors.New("email: RESEND_API_KEY not configured")
	}

	payload := map[string]string{
		"from":    s.fromEmail,
		"to":      to,
		"subject": subject,
		"html":    htmlBody,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("email: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("email: resend request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("email: resend returned status %d: %s", resp.StatusCode, string(respBody))
	}
	// Drain the body so the connection can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// --- StubEmailSender ---

// StubEmailSender is the no-op EmailSender used in dev mode (when
// RESEND_API_KEY is not set) and in unit tests. It logs the recipient +
// subject so the digest pipeline's behaviour is observable in dev logs
// without sending any real traffic. Send always returns nil — the stub
// never fails.
//
// The stub is concurrency-safe because log.Printf is.
type StubEmailSender struct {
	// LastTo / LastSubject are populated by Send so tests can assert on
	// what was "sent" without parsing log output. Access to these fields
	// is NOT synchronised — tests are expected to call Send serially.
	LastTo      string
	LastSubject string
	LastBody    string
	// SendCount is the number of times Send was called. Useful for
	// asserting the digest pipeline fanned out N emails for N
	// subscribers.
	SendCount int
}

// Name implements EmailSender.
func (s *StubEmailSender) Name() string { return "stub" }

// Send implements EmailSender.
func (s *StubEmailSender) Send(_ context.Context, to, subject, htmlBody string) error {
	s.LastTo = to
	s.LastSubject = subject
	s.LastBody = htmlBody
	s.SendCount++
	log.Printf("email(stub): would send to=%q subject=%q (body %d bytes)", to, subject, len(htmlBody))
	return nil
}

// firstNonEmpty returns the first non-empty argument, or "" if all are
// empty. Used to pick a from-email with env-var override + default.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
