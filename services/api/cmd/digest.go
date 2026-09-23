// Package main — daily digest email pipeline (issue #279 — Email alert
// subscriptions).
//
// This file owns the digest data shape, the HTML email template, the
// per-user digest builder, and the daily fan-out pipeline.
//
// The pipeline is invoked by the refresh handler
// (services/api/cmd/main.go:makeRefreshHandler) when the Vercel daily
// cron hits /api/v1/refresh at 03:00 UTC (see vercel.json). When
// RESEND_API_KEY is set, every subscriber who has "email" in their
// channels gets a personalised digest of the past 24h's notifications;
// when the key is absent, the pipeline is a no-op (StubEmailSender logs
// the would-be sends in dev).
//
// Routes:
//
//      GET /api/v1/brief/digest — returns the past 24h digest for the
//                                  authenticated caller. Used by the
//                                  /notifications frontend so a citizen can
//                                  preview the same content the email will
//                                  carry.
//
// The HTML template is inline-CSS, mobile-responsive, and renders three
// sections — Bills that changed stage, New gazette notices matching your
// alerts, Your MP's activity — plus an unsubscribe link at the bottom.
package main

import (
        "context"
        "fmt"
        "log"
        "net/http"
        "strings"
        "sync"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// --- Types ---

// Digest is the per-user daily digest payload. The same struct is returned
// by GET /api/v1/brief/digest (JSON) and rendered into the email HTML by
// renderDigestEmail.
type Digest struct {
        UserID        string            `json:"user_id"`
        Email         string            `json:"email,omitempty"`
        Headline      string            `json:"headline"`
        GeneratedAt   time.Time         `json:"generated_at"`
        Since         time.Time         `json:"since"`
        Until         time.Time         `json:"until"`
        Sections      []DigestSection   `json:"sections"`
        TotalItems    int               `json:"total_items"`
        UnsubscribeURL string           `json:"unsubscribe_url"`
}

// DigestSection groups digest items under one of the three fixed headings.
type DigestSection struct {
        Title  string        `json:"title"`
        Items  []DigestItem  `json:"items"`
}

// DigestItem is a single line in the digest. LinkURL is required so every
// item links back to the platform — the email is never a dead-end.
type DigestItem struct {
        Title       string `json:"title"`
        Description string `json:"description,omitempty"`
        LinkURL     string `json:"link_url"`
        Kind        string `json:"kind"` // bill_stage_change, gazette_notice, mp_activity
        Date        string `json:"date,omitempty"`
}

// The three fixed section titles. The template renders these in order.
const (
        digestSectionBills   = "Bills that changed stage"
        digestSectionGazette = "New gazette notices matching your alerts"
        digestSectionMP      = "Your MP's activity"

        // digestSubject is the email subject line. It intentionally mentions
        // the platform name so the recipient can build a filter rule.
        digestSubject = "Today in Civic Intelligence"

        // digestWindow is how far back the digest reaches. 24h matches the
        // daily cron cadence — a shorter window would let alerts fall through
        // the cracks when the cron is delayed; a longer one would surface
        // stale items the citizen has already seen in-app.
        digestWindow = 24 * time.Hour
)

// --- User email registry (in-memory) ---
//
// The refresh handler runs as a cron job WITHOUT an authenticated
// principal, so it cannot read the user's email from the request
// context. The UserEmailStore captures the email the FIRST time the
// user authenticates with one (via any authenticated request that
// touches the subscriptions PATCH handler today; in production this
// becomes a column on identity.users). When the daily digest pipeline
// fans out, it looks up the email here; users without a known email are
// skipped (logged) rather than dropped silently.
type UserEmailStore struct {
        mu     sync.RWMutex
        emails map[string]string // user_id → email
}

// NewUserEmailStore returns an empty registry.
func NewUserEmailStore() *UserEmailStore {
        return &UserEmailStore{emails: make(map[string]string)}
}

// Set records the email for the given user. Empty values are ignored so
// an unauthenticated PATCH does not clobber a previously-stored address.
// Idempotent — re-registering the same email is a no-op.
func (s *UserEmailStore) Set(userID, email string) {
        userID = strings.TrimSpace(userID)
        email = strings.TrimSpace(strings.ToLower(email))
        if userID == "" || email == "" {
                return
        }
        s.mu.Lock()
        defer s.mu.Unlock()
        s.emails[userID] = email
}

// Get returns the stored email for the user, or "" when unknown.
func (s *UserEmailStore) Get(userID string) string {
        s.mu.RLock()
        defer s.mu.RUnlock()
        return s.emails[userID]
}

// --- Digest construction ---

// buildDigest assembles a Digest for the given user from their notification
// stream. The function is pure — it does not touch the EmailSender; the
// caller decides whether to render the result to HTML + send an email or
// return it as JSON (the GET /brief/digest endpoint does the latter).
//
// The function scans the supplied notifications and groups them into the
// three fixed sections by event_type + entity_type. The since/until
// window is supplied by the caller — typically time.Now().Add(-24h) and
// time.Now(), but tests pass fixed timestamps.
func buildDigest(
        userID, email string,
        allNotifs []NotificationRecord,
        since, until time.Time,
        baseURL string,
) Digest {
        sections := []DigestSection{
                {Title: digestSectionBills},
                {Title: digestSectionGazette},
                {Title: digestSectionMP},
        }

        total := 0
        for _, n := range allNotifs {
                if n.CreatedAt.Before(since) || n.CreatedAt.After(until) {
                        continue
                }
                item := DigestItem{
                        Title:       n.Title,
                        Description: n.Body,
                        LinkURL:     itemLinkURL(n, baseURL),
                        Kind:        string(n.EventType),
                        Date:        n.CreatedAt.UTC().Format("2006-01-02"),
                }
                switch {
                case n.EntityType == EntityBill:
                        sections[0].Items = append(sections[0].Items, item)
                case n.EntityType == EntityInstitution || strings.Contains(strings.ToLower(n.EventType), "gazette"):
                        sections[1].Items = append(sections[1].Items, item)
                case n.EntityType == EntityPerson:
                        sections[2].Items = append(sections[2].Items, item)
                default:
                        // Anything we cannot classify lands in the first section so it is
                        // not lost — the citizen can still see + click through.
                        sections[0].Items = append(sections[0].Items, item)
                }
                total++
        }

        return Digest{
                UserID:         userID,
                Email:          email,
                Headline:       digestHeadline(sections, total),
                GeneratedAt:    until,
                Since:          since,
                Until:          until,
                Sections:       sections,
                TotalItems:     total,
                UnsubscribeURL: unsubscribeURL(userID, baseURL),
        }
}

// digestHeadline composes a one-line summary like "3 Bills changed stage,
// 2 new gazette notices, 1 MP update". The pluralisation is intentionally
// simple — zero/one/many — to keep the template readable.
func digestHeadline(sections []DigestSection, total int) string {
        if total == 0 {
                return "No new civic alerts today."
        }
        parts := make([]string, 0, 3)
        for _, s := range sections {
                if len(s.Items) == 0 {
                        continue
                }
                label := strings.TrimSuffix(strings.ToLower(s.Title), "s")
                // "Bills that changed stage" → "bills changed stage"
                // "New gazette notices matching your alerts" → "new gazette notices"
                // "Your MP's activity" → "your mp's activity"
                switch {
                case strings.Contains(label, "changed stage"):
                        label = "bills changed stage"
                case strings.Contains(label, "gazette"):
                        label = "new gazette notices"
                case strings.Contains(label, "mp"):
                        label = "MP updates"
                }
                parts = append(parts, fmt.Sprintf("%d %s", len(s.Items), label))
        }
        return strings.Join(parts, ", ") + "."
}

// itemLinkURL returns the deep-link URL for a notification's entity on the
// web app. The baseURL is configurable so tests can pass an httptest
// server; production passes the public web origin (e.g.
// https://civicintelligence.com).
func itemLinkURL(n NotificationRecord, baseURL string) string {
        base := strings.TrimRight(baseURL, "/")
        if base == "" {
                base = "https://civicintelligence.com"
        }
        switch n.EntityType {
        case EntityBill:
                return fmt.Sprintf("%s/bills/%s", base, n.EntityID)
        case EntityAct:
                return fmt.Sprintf("%s/acts/%s", base, n.EntityID)
        case EntityPerson:
                return fmt.Sprintf("%s/people/%s", base, n.EntityID)
        case EntityCommittee:
                return fmt.Sprintf("%s/committees#%s", base, n.EntityID)
        case EntityInstitution:
                return fmt.Sprintf("%s/institutions#%s", base, n.EntityID)
        case EntityTopic:
                return fmt.Sprintf("%s/topics#%s", base, n.EntityID)
        default:
                return base
        }
}

// unsubscribeURL builds the per-user unsubscribe link. The link carries
// the user_id as a query parameter so the (future) /api/v1/subscriptions/unsubscribe
// endpoint can locate the user's follows without an auth header — the
// email client clicks through to a landing page that confirms the choice.
func unsubscribeURL(userID, baseURL string) string {
        base := strings.TrimRight(baseURL, "/")
        if base == "" {
                base = "https://civicintelligence.com"
        }
        return fmt.Sprintf("%s/notifications/unsubscribe?u=%s", base, userID)
}

// --- HTML rendering ---

// renderDigestEmail produces the mobile-responsive HTML email body for
// the supplied Digest. All CSS is inline so the email renders correctly
// in Gmail, Outlook, and Apple Mail (which strip <style> blocks). The
// layout uses a single 600px column with a 16px gutter — the de-facto
// mobile email width that scales down on narrow viewports.
//
// The template intentionally avoids JavaScript, external images, and
// external stylesheets — every email client blocks at least one of
// those. The only network fetch is the unsubscribe link.
func renderDigestEmail(d Digest) string {
        var b strings.Builder
        b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>`)
        b.WriteString(digestSubject)
        b.WriteString(`</title>
</head>
<body style="margin:0;padding:0;background:#f5f5f4;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#1c1917;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f5f5f4;">
    <tr>
      <td align="center" style="padding:24px 12px;">
        <table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;background:#ffffff;border-radius:12px;overflow:hidden;border:1px solid #e7e5e4;">
          <tr>
            <td style="padding:32px 24px;background:#064e3b;color:#ffffff;">
              <h1 style="margin:0;font-size:24px;font-weight:600;line-height:1.25;">Today in Civic Intelligence</h1>
              <p style="margin:8px 0 0;font-size:14px;color:#d1fae5;">`)
        b.WriteString(d.Headline)
        b.WriteString(`</p>
              <p style="margin:4px 0 0;font-size:12px;color:#a7f3d0;">`)
        b.WriteString(d.GeneratedAt.UTC().Format("Monday, 02 Jan 2006"))
        b.WriteString(`</p>
            </td>
          </tr>
`)

        if d.TotalItems == 0 {
                b.WriteString(`          <tr>
            <td style="padding:32px 24px;text-align:center;color:#57534e;font-size:14px;">
              <p style="margin:0 0 8px;">No new civic alerts in the past 24 hours.</p>
              <p style="margin:0;font-size:13px;color:#78716c;">Follow a Bill, committee, or topic to start receiving alerts.</p>
            </td>
          </tr>
`)
        } else {
                for _, sec := range d.Sections {
                        if len(sec.Items) == 0 {
                                continue
                        }
                        b.WriteString(`          <tr>
            <td style="padding:24px 24px 8px;border-top:1px solid #f5f5f4;">
              <h2 style="margin:0;font-size:16px;font-weight:600;color:#1c1917;">`)
                        b.WriteString(escapeHTML(sec.Title))
                        b.WriteString(`</h2>
            </td>
          </tr>
          <tr>
            <td style="padding:0 24px 8px;">
              <table role="presentation" width="100%" cellpadding="0" cellspacing="0">
`)
                        for _, item := range sec.Items {
                                b.WriteString(`                <tr>
                  <td style="padding:12px 0;border-top:1px solid #f5f5f4;">
                    <a href="`)
                                b.WriteString(escapeHTML(item.LinkURL))
                                b.WriteString(`" style="color:#064e3b;text-decoration:none;font-size:15px;font-weight:600;line-height:1.35;display:block;">`)
                                b.WriteString(escapeHTML(item.Title))
                                b.WriteString(`</a>
                    <p style="margin:4px 0 0;font-size:13px;color:#57534e;line-height:1.5;">`)
                                b.WriteString(escapeHTML(item.Description))
                                b.WriteString(`</p>
                    <p style="margin:4px 0 0;font-size:12px;color:#78716c;">`)
                                b.WriteString(escapeHTML(item.Date))
                                b.WriteString(`</p>
                  </td>
                </tr>
`)
                        }
                        b.WriteString(`              </table>
            </td>
          </tr>
`)
                }
        }

        b.WriteString(`          <tr>
            <td style="padding:24px;border-top:1px solid #f5f5f4;text-align:center;">
              <p style="margin:0 0 8px;font-size:13px;color:#78716c;">
                <a href="`)
        b.WriteString(escapeHTML(d.UnsubscribeURL))
        b.WriteString(`" style="color:#064e3b;text-decoration:underline;">Unsubscribe</a>
                ·
                <a href="https://civicintelligence.com/notifications" style="color:#064e3b;text-decoration:underline;">View in app</a>
              </p>
              <p style="margin:0;font-size:11px;color:#a8a29e;line-height:1.5;">
                You are receiving this email because you enabled email alerts on Civic Intelligence.
                Replies to this address are not monitored.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`)
        return b.String()
}

// escapeHTML replaces &, <, >, ", ' with their HTML entities. Used by the
// template renderer so a Bill title containing a stray < or & doesn't
// break the email markup. We avoid html.EscapeString here so the
// renderer stays dependency-free at the call site.
func escapeHTML(s string) string {
        r := strings.NewReplacer(
                `&`, "&amp;",
                `<`, "&lt;",
                `>`, "&gt;",
                `"`, "&quot;",
                `'`, "&#39;",
        )
        return r.Replace(s)
}

// --- Daily digest fan-out ---

// sendDailyDigests iterates over every subscriber who has "email" in
// their channels, builds a per-user digest from their past-24h
// notifications, renders the HTML, and calls EmailSender.Send. The
// function is the production entrypoint invoked by the refresh handler
// (issue #279).
//
// The function is intentionally resilient: a single recipient's failure
// is logged and the loop continues, so one bad address never blocks the
// whole batch. The ctx bounds the total time — each Send call inherits
// the ctx, so a slow Resend response propagates the cancellation.
//
// The baseURL is the public web origin (e.g. https://civicintelligence.com)
// used for the per-item deep links + the unsubscribe link. Empty defaults
// to https://civicintelligence.com.
//
// Returns the number of emails successfully queued + the number that
// failed. Callers (the refresh handler) include these counts in the
// refresh response so an operator can see at a glance whether the cron
// sent any mail.
func sendDailyDigests(
        ctx context.Context,
        subs *SubscriptionStore,
        emails *UserEmailStore,
        notifStore *NotificationStore,
        sender EmailSender,
        baseURL string,
        now time.Time,
) (sent, failed int) {
        if sender == nil {
                return 0, 0
        }
        subscribers := subs.EmailSubscribers()
        if len(subscribers) == 0 {
                return 0, 0
        }
        since := now.Add(-digestWindow)

        for _, userID := range subscribers {
                email := emails.Get(userID)
                if email == "" {
                        log.Printf("digest: skipping user %s — no email on file (PATCH /subscriptions/{id} with email channel captures it from the auth principal)", userID)
                        continue
                }
                notifs := notifStore.List(userID, false)
                digest := buildDigest(userID, email, notifs, since, now, baseURL)
                if digest.TotalItems == 0 {
                        // Still send the email — an empty digest tells the citizen the
                        // pipeline is alive (and shows the unsubscribe link) without
                        // inventing items.
                        log.Printf("digest: user %s — 0 items in past 24h, sending empty digest", userID)
                }
                body := renderDigestEmail(digest)
                if err := sender.Send(ctx, email, digestSubject, body); err != nil {
                        log.Printf("digest: send to %s failed: %v", email, err)
                        failed++
                        continue
                }
                sent++
        }
        return sent, failed
}

// --- HTTP handler ---

// makeBriefDigestHandler returns the handler for GET /api/v1/brief/digest.
// The endpoint returns the same JSON the email pipeline renders to HTML,
// so the /notifications page can preview today's digest without waiting
// for the cron.
func makeBriefDigestHandler(
        notifStore *NotificationStore,
        baseURL string,
) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        w.Header().Set("Allow", "GET")
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
                                "only GET is supported on /api/v1/brief/digest")
                        return
                }
                p := middleware.PrincipalFromRequest(r)
                if p.IsAnonymous() {
                        writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
                        return
                }
                now := time.Now().UTC()
                since := now.Add(-digestWindow)
                // Seed sample notifications so the digest has content during local
                // dev + tests (the live event matcher is not yet wired — issue #112).
                notifStore.seedSampleNotifications(p.UserID)
                notifs := notifStore.List(p.UserID, false)
                digest := buildDigest(p.UserID, p.Email, notifs, since, now, baseURL)
                writeJSON(w, http.StatusOK, digest)
        }
}
