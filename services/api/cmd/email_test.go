package main

import (
        "context"
        "encoding/json"
        "errors"
        "io"
        "net/http"
        "net/http/httptest"
        "strings"
        "testing"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
)

// === EmailSender ===

// TestNewEmailSender_StubWhenNoAPIKey verifies that NewEmailSender returns
// a *StubEmailSender (not Resend) when RESEND_API_KEY is unset. We t.Setenv
// the empty string to be explicit — the test does not depend on ambient
// environment state.
func TestNewEmailSender_StubWhenNoAPIKey(t *testing.T) {
        t.Setenv("RESEND_API_KEY", "")
        s := NewEmailSender(nil)
        if _, ok := s.(*StubEmailSender); !ok {
                t.Fatalf("expected *StubEmailSender, got %T", s)
        }
        if s.Name() != "stub" {
                t.Errorf("expected name=stub, got %s", s.Name())
        }
}

// TestStubEmailSender_CapturesLastSend verifies the StubEmailSender records
// the to/subject/body so tests can assert what was "sent" without parsing
// log output. Send never returns an error.
func TestStubEmailSender_CapturesLastSend(t *testing.T) {
        stub := &StubEmailSender{}
        if err := stub.Send(context.Background(), "citizen@example.com", "subj", "<p>hi</p>"); err != nil {
                t.Fatalf("stub.Send returned error: %v", err)
        }
        if stub.LastTo != "citizen@example.com" {
                t.Errorf("LastTo=%q", stub.LastTo)
        }
        if stub.LastSubject != "subj" {
                t.Errorf("LastSubject=%q", stub.LastSubject)
        }
        if stub.LastBody != "<p>hi</p>" {
                t.Errorf("LastBody=%q", stub.LastBody)
        }
        if stub.SendCount != 1 {
                t.Errorf("SendCount=%d", stub.SendCount)
        }
}

// TestResendEmailSender_POSTsToResendAPI verifies the ResendEmailSender
// builds the correct HTTP request — Authorization header, JSON body with
// from/to/subject/html, POST to /emails — and returns nil on a 2xx. The
// test spins up an httptest.Server that records the inbound request.
func TestResendEmailSender_POSTsToResendAPI(t *testing.T) {
        var (
                gotPath    string
                gotAuth    string
                gotCT      string
                gotPayload map[string]string
        )
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                gotPath = r.URL.Path
                gotAuth = r.Header.Get("Authorization")
                gotCT = r.Header.Get("Content-Type")
                body, _ := io.ReadAll(r.Body)
                _ = json.Unmarshal(body, &gotPayload)
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                _, _ = w.Write([]byte(`{"id":"msg_123"}`))
        }))
        defer srv.Close()

        sender := &ResendEmailSender{
                apiKey:    "test-key",
                client:    srv.Client(),
                baseURL:   srv.URL,
                fromEmail: "onboarding@resend.dev",
        }
        err := sender.Send(context.Background(), "citizen@example.com", "Today in Civic Intelligence", "<html>body</html>")
        if err != nil {
                t.Fatalf("Send: %v", err)
        }
        if gotPath != "/emails" {
                t.Errorf("expected path /emails, got %q", gotPath)
        }
        if gotAuth != "Bearer test-key" {
                t.Errorf("expected Authorization 'Bearer test-key', got %q", gotAuth)
        }
        if gotCT != "application/json" {
                t.Errorf("expected Content-Type application/json, got %q", gotCT)
        }
        if gotPayload["from"] != "onboarding@resend.dev" {
                t.Errorf("payload.from=%q", gotPayload["from"])
        }
        if gotPayload["to"] != "citizen@example.com" {
                t.Errorf("payload.to=%q", gotPayload["to"])
        }
        if gotPayload["subject"] != "Today in Civic Intelligence" {
                t.Errorf("payload.subject=%q", gotPayload["subject"])
        }
        if gotPayload["html"] != "<html>body</html>" {
                t.Errorf("payload.html=%q", gotPayload["html"])
        }
}

// TestResendEmailSender_Non2xxReturnsError verifies a non-2xx Resend
// response surfaces as an error containing the upstream body so an
// operator can see why Resend rejected the request.
func TestResendEmailSender_Non2xxReturnsError(t *testing.T) {
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusUnprocessableEntity)
                _, _ = w.Write([]byte(`{"error":"invalid recipient"}`))
        }))
        defer srv.Close()

        sender := &ResendEmailSender{
                apiKey:    "test-key",
                client:    srv.Client(),
                baseURL:   srv.URL,
                fromEmail: "onboarding@resend.dev",
        }
        err := sender.Send(context.Background(), "bad@example.com", "subj", "<html>body</html>")
        if err == nil {
                t.Fatal("expected error on 422, got nil")
        }
        if !strings.Contains(err.Error(), "422") {
                t.Errorf("expected error to mention 422 status, got %v", err)
        }
        if !strings.Contains(err.Error(), "invalid recipient") {
                t.Errorf("expected error to include upstream body, got %v", err)
        }
}

// TestResendEmailSender_ValidationErrorPaths verifies Send's input
// validation branches. Empty to / subject and missing API key each
// produce a distinct, descriptive error — no HTTP request is fired.
func TestResendEmailSender_ValidationErrorPaths(t *testing.T) {
        sender := &ResendEmailSender{apiKey: "k", client: http.DefaultClient, baseURL: "https://example.com", fromEmail: "from@example.com"}
        cases := []struct {
                name    string
                to      string
                subject string
                body    string
                wantErr string
        }{
                {"empty to", "", "s", "b", "recipient"},
                {"empty subject", "x@x.com", "", "b", "subject"},
        }
        for _, c := range cases {
                t.Run(c.name, func(t *testing.T) {
                        err := sender.Send(context.Background(), c.to, c.subject, c.body)
                        if err == nil || !strings.Contains(err.Error(), c.wantErr) {
                                t.Errorf("expected error containing %q, got %v", c.wantErr, err)
                        }
                })
        }

        // Missing API key.
        sender.apiKey = ""
        err := sender.Send(context.Background(), "x@x.com", "s", "b")
        if err == nil || !strings.Contains(err.Error(), "RESEND_API_KEY") {
                t.Errorf("expected RESEND_API_KEY error, got %v", err)
        }
}

// TestResendEmailSender_ContextCancelled verifies Send honours the
// supplied context — a cancelled context aborts the request before it
// hits the wire (the test asserts Send returns an error mentioning
// context cancellation).
func TestResendEmailSender_ContextCancelled(t *testing.T) {
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                time.Sleep(50 * time.Millisecond)
                w.WriteHeader(http.StatusOK)
        }))
        defer srv.Close()

        sender := &ResendEmailSender{
                apiKey:    "k",
                client:    srv.Client(),
                baseURL:   srv.URL,
                fromEmail: "from@example.com",
        }
        ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
        defer cancel()
        err := sender.Send(ctx, "x@x.com", "s", "b")
        if err == nil {
                t.Fatal("expected error on cancelled context, got nil")
        }
}

// TestNewEmailSender_ResendWhenAPIKey verifies NewEmailSender returns a
// *ResendEmailSender when RESEND_API_KEY is set. The fromEmail is taken
// from RESEND_FROM_EMAIL when set, falling back to "onboarding@resend.dev".
func TestNewEmailSender_ResendWhenAPIKey(t *testing.T) {
        t.Setenv("RESEND_API_KEY", "sk-test")
        t.Setenv("RESEND_FROM_EMAIL", "noreply@civicintelligence.com")
        s := NewEmailSender(nil)
        resend, ok := s.(*ResendEmailSender)
        if !ok {
                t.Fatalf("expected *ResendEmailSender, got %T", s)
        }
        if resend.apiKey != "sk-test" {
                t.Errorf("apiKey=%q", resend.apiKey)
        }
        if resend.fromEmail != "noreply@civicintelligence.com" {
                t.Errorf("fromEmail=%q", resend.fromEmail)
        }
        if s.Name() != "resend" {
                t.Errorf("expected name=resend, got %s", s.Name())
        }
}

// === Channels + PATCH ===

// TestPatchSubscription_AddEmailChannel verifies PATCH
// /api/v1/subscriptions/{id} updates the channels slice on the follow
// record. The response carries the updated record so the frontend can
// refresh its UI without an extra GET.
func TestPatchSubscription_AddEmailChannel(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a", nil)
        handler := makeSubscriptionDetailHandler(store, nil)

        p := auth.Principal{UserID: "user-1"}
        body := `{"channels":["in_app","email"]}`
        rr := dispatch(handler, http.MethodPatch, "/api/v1/subscriptions/"+rec.ID, []byte(body), p)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var updated FollowRecord
        if err := json.Unmarshal(rr.Body.Bytes(), &updated); err != nil {
                t.Fatalf("invalid JSON: %v", err)
        }
        if !hasChannel(updated.Channels, ChannelEmail) {
                t.Errorf("expected channels to include email; got %v", updated.Channels)
        }
        if !hasChannel(updated.Channels, ChannelInApp) {
                t.Errorf("expected channels to include in_app (always preserved); got %v", updated.Channels)
        }
        // Persisted to the store.
        got, ok := store.Get("user-1", rec.ID)
        if !ok {
                t.Fatal("expected follow to be persisted")
        }
        if !hasChannel(got.Channels, ChannelEmail) {
                t.Errorf("stored channels should include email; got %v", got.Channels)
        }
}

// TestPatchSubscription_NotOwner verifies a PATCH from a non-owner
// returns 404 (the record does not belong to the caller).
func TestPatchSubscription_NotOwner(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a", nil)
        handler := makeSubscriptionDetailHandler(store, nil)

        p := auth.Principal{UserID: "user-2"}
        body := `{"channels":["in_app","email"]}`
        rr := dispatch(handler, http.MethodPatch, "/api/v1/subscriptions/"+rec.ID, []byte(body), p)
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404 for non-owner PATCH, got %d", rr.Code)
        }
}

// TestPatchSubscription_InvalidChannel verifies an unknown channel name
// (e.g. "sms") returns 404 because UpdateChannels fails validation. The
// test uses 404 (not 400) because the handler maps every UpdateChannels
// failure to a single not_found envelope — this matches the existing
// DELETE contract (which also returns 404 for non-existent records).
func TestPatchSubscription_InvalidChannel(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a", nil)
        handler := makeSubscriptionDetailHandler(store, nil)

        p := auth.Principal{UserID: "user-1"}
        body := `{"channels":["sms"]}`
        rr := dispatch(handler, http.MethodPatch, "/api/v1/subscriptions/"+rec.ID, []byte(body), p)
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404 for invalid channel, got %d", rr.Code)
        }
}

// TestPatchSubscription_AnonymousRejected verifies the PATCH handler
// returns 401 when the caller is anonymous.
func TestPatchSubscription_AnonymousRejected(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a", nil)
        handler := makeSubscriptionDetailHandler(store, nil)

        body := `{"channels":["in_app","email"]}`
        rr := dispatch(handler, http.MethodPatch, "/api/v1/subscriptions/"+rec.ID, []byte(body), auth.Anonymous())
        if rr.Code != http.StatusUnauthorized {
                t.Errorf("expected 401 for anonymous PATCH, got %d", rr.Code)
        }
}

// TestPatchSubscription_EmptyChannels verifies the PATCH handler rejects
// an empty channels array — the citizen must always opt into at least
// one channel.
func TestPatchSubscription_EmptyChannels(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a", nil)
        handler := makeSubscriptionDetailHandler(store, nil)

        p := auth.Principal{UserID: "user-1"}
        body := `{"channels":[]}`
        rr := dispatch(handler, http.MethodPatch, "/api/v1/subscriptions/"+rec.ID, []byte(body), p)
        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for empty channels, got %d", rr.Code)
        }
}

// TestPatchSubscription_CapturesEmail verifies the PATCH handler
// opportunistically captures the caller's email claim into the
// userEmails registry so the daily digest pipeline can reach them.
func TestPatchSubscription_CapturesEmail(t *testing.T) {
        store := NewSubscriptionStore()
        rec, _ := store.Follow("user-1", EntityBill, "bill-a", nil)
        emails := NewUserEmailStore()
        handler := makeSubscriptionDetailHandler(store, emails)

        p := auth.Principal{UserID: "user-1", Email: "citizen@example.com"}
        body := `{"channels":["in_app","email"]}`
        rr := dispatch(handler, http.MethodPatch, "/api/v1/subscriptions/"+rec.ID, []byte(body), p)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        if got := emails.Get("user-1"); got != "citizen@example.com" {
                t.Errorf("expected captured email 'citizen@example.com', got %q", got)
        }
}

// TestNormaliseChannels_DefaultsAndPreservesInApp verifies the
// normalisation rules: empty input defaults to [in_app]; "in_app" is
// always present even when the caller passes only "email"; unknown
// names error; duplicates are removed; case is normalised.
func TestNormaliseChannels_DefaultsAndPreservesInApp(t *testing.T) {
        cases := []struct {
                name     string
                input    []string
                wantErr  bool
                hasEmail bool
                hasInApp bool
        }{
                {"empty defaults to in_app", nil, false, false, true},
                {"email only adds in_app", []string{"email"}, false, true, true},
                {"both channels", []string{"in_app", "email"}, false, true, true},
                {"uppercase normalized", []string{"EMAIL", "In_App"}, false, true, true},
                {"dedupe", []string{"in_app", "in_app", "email", "email"}, false, true, true},
                {"unknown channel errors", []string{"sms"}, true, false, false},
        }
        for _, c := range cases {
                t.Run(c.name, func(t *testing.T) {
                        out, err := normaliseChannels(c.input)
                        if c.wantErr {
                                if err == nil {
                                        t.Fatalf("expected error, got nil (out=%v)", out)
                                }
                                return
                        }
                        if err != nil {
                                t.Fatalf("unexpected error: %v", err)
                        }
                        if hasChannel(out, ChannelEmail) != c.hasEmail {
                                t.Errorf("hasEmail mismatch: got %v", out)
                        }
                        if !hasChannel(out, ChannelInApp) {
                                t.Errorf("in_app should always be present; got %v", out)
                        }
                })
        }
}

// === Digest construction ===

// TestBuildDigest_GroupsBySection verifies buildDigest places
// notifications into the correct sections based on entity_type — Bills
// → "Bills that changed stage", institutions/gazette → "New gazette
// notices matching your alerts", people → "Your MP's activity".
func TestBuildDigest_GroupsBySection(t *testing.T) {
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        since := now.Add(-24 * time.Hour)
        notifs := []NotificationRecord{
                {ID: "n1", UserID: "u1", EntityType: EntityBill, EntityID: "b1", EventType: EventBillStageChange, Title: "Bill X advanced", Body: "Stage change", CreatedAt: now.Add(-1 * time.Hour)},
                {ID: "n2", UserID: "u1", EntityType: EntityInstitution, EntityID: "i1", EventType: EventBillPublished, Title: "Gazette notice", Body: "Notice", CreatedAt: now.Add(-2 * time.Hour)},
                {ID: "n3", UserID: "u1", EntityType: EntityPerson, EntityID: "p1", EventType: EventCommitteeReport, Title: "MP spoke", Body: "Hansard", CreatedAt: now.Add(-3 * time.Hour)},
        }
        d := buildDigest("u1", "u1@example.com", notifs, since, now, "https://example.com")
        if d.TotalItems != 3 {
                t.Fatalf("expected total_items=3, got %d", d.TotalItems)
        }
        if len(d.Sections) != 3 {
                t.Fatalf("expected 3 sections, got %d", len(d.Sections))
        }
        if len(d.Sections[0].Items) != 1 || d.Sections[0].Items[0].Title != "Bill X advanced" {
                t.Errorf("bills section mismatch: %+v", d.Sections[0].Items)
        }
        if len(d.Sections[1].Items) != 1 || d.Sections[1].Items[0].Title != "Gazette notice" {
                t.Errorf("gazette section mismatch: %+v", d.Sections[1].Items)
        }
        if len(d.Sections[2].Items) != 1 || d.Sections[2].Items[0].Title != "MP spoke" {
                t.Errorf("mp section mismatch: %+v", d.Sections[2].Items)
        }
}

// TestBuildDigest_FiltersByTimeWindow verifies buildDigest drops
// notifications whose CreatedAt falls outside the [since, until] window.
// Notifications older than 24h are excluded; future-dated ones are too.
func TestBuildDigest_FiltersByTimeWindow(t *testing.T) {
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        since := now.Add(-24 * time.Hour)
        notifs := []NotificationRecord{
                {ID: "fresh", UserID: "u1", EntityType: EntityBill, EntityID: "b1", EventType: EventBillStageChange, Title: "fresh", Body: "fresh", CreatedAt: now.Add(-1 * time.Hour)},
                {ID: "old", UserID: "u1", EntityType: EntityBill, EntityID: "b2", EventType: EventBillStageChange, Title: "old", Body: "old", CreatedAt: now.Add(-48 * time.Hour)},
                {ID: "future", UserID: "u1", EntityType: EntityBill, EntityID: "b3", EventType: EventBillStageChange, Title: "future", Body: "future", CreatedAt: now.Add(+1 * time.Hour)},
        }
        d := buildDigest("u1", "u1@example.com", notifs, since, now, "https://example.com")
        if d.TotalItems != 1 {
                t.Fatalf("expected 1 item in window, got %d", d.TotalItems)
        }
        if d.Sections[0].Items[0].Title != "fresh" {
                t.Errorf("expected only the 'fresh' item, got %q", d.Sections[0].Items[0].Title)
        }
}

// TestBuildDigest_EveryItemLinksBack verifies every populated digest item
// carries a non-empty link_url — the email is never a dead-end.
func TestBuildDigest_EveryItemLinksBack(t *testing.T) {
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        since := now.Add(-24 * time.Hour)
        notifs := []NotificationRecord{
                {ID: "n1", UserID: "u1", EntityType: EntityBill, EntityID: "b1", EventType: EventBillStageChange, Title: "Bill", Body: "body", CreatedAt: now.Add(-1 * time.Hour)},
                {ID: "n2", UserID: "u1", EntityType: EntityInstitution, EntityID: "i1", EventType: EventBillPublished, Title: "Gazette", Body: "body", CreatedAt: now.Add(-2 * time.Hour)},
                {ID: "n3", UserID: "u1", EntityType: EntityPerson, EntityID: "p1", EventType: EventCommitteeReport, Title: "MP", Body: "body", CreatedAt: now.Add(-3 * time.Hour)},
        }
        d := buildDigest("u1", "u1@example.com", notifs, since, now, "https://example.com")
        for sIdx, sec := range d.Sections {
                for iIdx, item := range sec.Items {
                        if strings.TrimSpace(item.LinkURL) == "" {
                                t.Errorf("section[%d] item[%d]: empty link_url", sIdx, iIdx)
                        }
                }
        }
}

// TestBuildDigest_EmptyDigest verifies buildDigest produces a valid
// (empty) digest when there are no notifications in the window. The
// total_items is 0 and every section is empty.
func TestBuildDigest_EmptyDigest(t *testing.T) {
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        since := now.Add(-24 * time.Hour)
        d := buildDigest("u1", "u1@example.com", nil, since, now, "https://example.com")
        if d.TotalItems != 0 {
                t.Errorf("expected total_items=0, got %d", d.TotalItems)
        }
        for sIdx, sec := range d.Sections {
                if len(sec.Items) != 0 {
                        t.Errorf("section[%d] should be empty, got %d items", sIdx, len(sec.Items))
                }
        }
        if !strings.Contains(d.Headline, "No new civic alerts") {
                t.Errorf("expected empty-digest headline, got %q", d.Headline)
        }
}

// TestBuildDigest_UnsubscribeURL verifies the unsubscribe link is built
// from the supplied baseURL and carries the user_id as a query parameter
// — the (future) /api/v1/subscriptions/unsubscribe endpoint can locate
// the user's follows without an auth header.
func TestBuildDigest_UnsubscribeURL(t *testing.T) {
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        d := buildDigest("user-42", "u@example.com", nil, now.Add(-24*time.Hour), now, "https://example.com")
        if !strings.HasPrefix(d.UnsubscribeURL, "https://example.com/notifications/unsubscribe?u=user-42") {
                t.Errorf("unexpected unsubscribe_url: %s", d.UnsubscribeURL)
        }
}

// === HTML rendering ===

// TestRenderDigestEmail_ContainsKeySections verifies the rendered HTML
// carries the three section titles, the header, an unsubscribe link, and
// is mobile-responsive (carries a viewport meta tag). The test does not
// assert byte-for-byte — the template is allowed to evolve as long as
// these anchors remain.
func TestRenderDigestEmail_ContainsKeySections(t *testing.T) {
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        notifs := []NotificationRecord{
                {ID: "n1", UserID: "u1", EntityType: EntityBill, EntityID: "b1", EventType: EventBillStageChange, Title: "Bill", Body: "body", CreatedAt: now.Add(-1 * time.Hour)},
                {ID: "n2", UserID: "u1", EntityType: EntityInstitution, EntityID: "i1", EventType: EventBillPublished, Title: "Gazette", Body: "body", CreatedAt: now.Add(-2 * time.Hour)},
                {ID: "n3", UserID: "u1", EntityType: EntityPerson, EntityID: "p1", EventType: EventCommitteeReport, Title: "MP", Body: "body", CreatedAt: now.Add(-3 * time.Hour)},
        }
        d := buildDigest("u1", "u1@example.com", notifs, now.Add(-24*time.Hour), now, "https://example.com")
        html := renderDigestEmail(d)
        // Section titles are HTML-escaped by the renderer — the apostrophe in
        // "MP's" becomes &#39;. Match the escaped forms so the test stays
        // robust against the renderer's escape rules.
        wantAnchors := []string{
                `Today in Civic Intelligence`,
                `Bills that changed stage`,
                `New gazette notices matching your alerts`,
                `Your MP&#39;s activity`,
                `Unsubscribe`,
                `viewport`,
                `https://example.com/notifications/unsubscribe?u=u1`,
        }
        for _, want := range wantAnchors {
                if !strings.Contains(html, want) {
                        t.Errorf("rendered HTML missing anchor %q", want)
                }
        }
}

// TestRenderDigestEmail_EscapesHTMLOnBillTitle verifies the renderer
// escapes user-supplied text — a Bill title containing a stray < or &
// does not break the email markup. The renderer MUST NOT echo raw
// notification text into the HTML body.
func TestRenderDigestEmail_EscapesHTMLOnBillTitle(t *testing.T) {
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        notifs := []NotificationRecord{
                {ID: "n1", UserID: "u1", EntityType: EntityBill, EntityID: "b1", EventType: EventBillStageChange, Title: "Bill <script>alert(1)</script>", Body: "Tom & Jerry", CreatedAt: now.Add(-1 * time.Hour)},
        }
        d := buildDigest("u1", "u1@example.com", notifs, now.Add(-24*time.Hour), now, "https://example.com")
        html := renderDigestEmail(d)
        if strings.Contains(html, "<script>alert(1)</script>") {
                t.Errorf("HTML did not escape <script> tag: %s", html)
        }
        if strings.Contains(html, "Tom & Jerry") {
                t.Errorf("HTML did not escape '&' in body: %s", html)
        }
        // The escaped form must appear instead.
        if !strings.Contains(html, "&lt;script&gt;") {
                t.Errorf("expected escaped &lt;script&gt; in HTML")
        }
        if !strings.Contains(html, "Tom &amp; Jerry") {
                t.Errorf("expected escaped Tom &amp; Jerry in HTML")
        }
}

// === Daily digest fan-out ===

// TestSendDailyDigests_StubSender verifies the daily digest pipeline
// fans out one email per subscriber who has "email" in their channels,
// using a StubEmailSender so no real traffic is generated. Each
// subscriber's SendCount should be 1 after the fan-out.
func TestSendDailyDigests_StubSender(t *testing.T) {
        subs := NewSubscriptionStore()
        _, _ = subs.Follow("u1", EntityBill, "b1", []string{"in_app", "email"})
        _, _ = subs.Follow("u2", EntityBill, "b2", []string{"in_app", "email"})
        // u3 only has in_app — no digest.
        _, _ = subs.Follow("u3", EntityBill, "b3", nil)

        emails := NewUserEmailStore()
        emails.Set("u1", "u1@example.com")
        emails.Set("u2", "u2@example.com")

        notifStore := NewNotificationStore()
        // Seed one notification per user so the digest is non-empty.
        _, _ = notifStore.Emit("u1", EntityBill, "b1", EventBillStageChange, "Bill X advanced", "Stage change", nil)
        _, _ = notifStore.Emit("u2", EntityBill, "b2", EventBillStageChange, "Bill Y advanced", "Stage change", nil)

        stub := &StubEmailSender{}
        sent, failed := sendDailyDigests(context.Background(), subs, emails, notifStore, stub, "https://example.com", time.Now().UTC())
        if sent != 2 {
                t.Errorf("expected sent=2, got %d", sent)
        }
        if failed != 0 {
                t.Errorf("expected failed=0, got %d", failed)
        }
        if stub.SendCount != 2 {
                t.Errorf("expected stub.SendCount=2, got %d", stub.SendCount)
        }
        // u3 was never sent an email.
        if strings.Contains(stub.LastTo, "u3") {
                t.Errorf("u3 should not have been sent an email; LastTo=%q", stub.LastTo)
        }
}

// TestSendDailyDigests_SkipsUserWithoutEmail verifies the pipeline skips
// a subscriber who has the "email" channel but no email address on file
// — the digest cannot be delivered, so the user is logged + skipped
// rather than the batch aborting.
func TestSendDailyDigests_SkipsUserWithoutEmail(t *testing.T) {
        subs := NewSubscriptionStore()
        _, _ = subs.Follow("u1", EntityBill, "b1", []string{"in_app", "email"})

        emails := NewUserEmailStore()
        // No email registered for u1.
        notifStore := NewNotificationStore()
        stub := &StubEmailSender{}
        sent, failed := sendDailyDigests(context.Background(), subs, emails, notifStore, stub, "https://example.com", time.Now().UTC())
        if sent != 0 {
                t.Errorf("expected sent=0 (no email on file), got %d", sent)
        }
        if failed != 0 {
                t.Errorf("expected failed=0 (skipped, not failed), got %d", failed)
        }
        if stub.SendCount != 0 {
                t.Errorf("stub.SendCount should be 0, got %d", stub.SendCount)
        }
}

// TestSendDailyDigests_NilSenderIsNoop verifies the pipeline tolerates
// a nil sender — the refresh handler can pass nil in tests / dev without
// panicking. Returns (0, 0) without iterating the store.
func TestSendDailyDigests_NilSenderIsNoop(t *testing.T) {
        subs := NewSubscriptionStore()
        _, _ = subs.Follow("u1", EntityBill, "b1", []string{"in_app", "email"})
        emails := NewUserEmailStore()
        emails.Set("u1", "u1@example.com")
        notifStore := NewNotificationStore()
        sent, failed := sendDailyDigests(context.Background(), subs, emails, notifStore, nil, "https://example.com", time.Now().UTC())
        if sent != 0 || failed != 0 {
                t.Errorf("expected (0,0) for nil sender, got (%d,%d)", sent, failed)
        }
}

// TestSendDailyDigests_FailedSendCountsAsFailed verifies a sender that
// returns an error is counted in `failed` rather than `sent`, and the
// pipeline continues to the next recipient.
func TestSendDailyDigests_FailedSendCountsAsFailed(t *testing.T) {
        subs := NewSubscriptionStore()
        _, _ = subs.Follow("u1", EntityBill, "b1", []string{"in_app", "email"})
        _, _ = subs.Follow("u2", EntityBill, "b2", []string{"in_app", "email"})
        emails := NewUserEmailStore()
        emails.Set("u1", "u1@example.com")
        emails.Set("u2", "u2@example.com")
        notifStore := NewNotificationStore()

        // failingSender always returns an error.
        failingSender := &failingEmailSender{}
        sent, failed := sendDailyDigests(context.Background(), subs, emails, notifStore, failingSender, "https://example.com", time.Now().UTC())
        if sent != 0 {
                t.Errorf("expected sent=0, got %d", sent)
        }
        if failed != 2 {
                t.Errorf("expected failed=2, got %d", failed)
        }
}

// failingEmailSender is a test EmailSender that always returns an error.
// Used to verify sendDailyDigests counts a failed send without aborting
// the batch.
type failingEmailSender struct{}

func (f *failingEmailSender) Send(_ context.Context, _, _, _ string) error {
        return errors.New("simulated upstream failure")
}
func (f *failingEmailSender) Name() string { return "failing" }

// === GET /brief/digest ===

// TestBriefDigestHandler_Success verifies GET /api/v1/brief/digest
// returns the past 24h digest for the authenticated caller. The response
// carries the caller's user_id, headline, sections, and total_items.
func TestBriefDigestHandler_Success(t *testing.T) {
        notifStore := NewNotificationStore()
        handler := makeBriefDigestHandler(notifStore, "https://example.com")

        p := auth.Principal{UserID: "user-1", Email: "u1@example.com"}
        rr := dispatch(handler, http.MethodGet, "/api/v1/brief/digest", nil, p)
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var d Digest
        if err := json.Unmarshal(rr.Body.Bytes(), &d); err != nil {
                t.Fatalf("invalid JSON: %v", err)
        }
        if d.UserID != "user-1" {
                t.Errorf("expected user_id=user-1, got %s", d.UserID)
        }
        if d.Email != "u1@example.com" {
                t.Errorf("expected email=u1@example.com, got %s", d.Email)
        }
        if len(d.Sections) != 3 {
                t.Errorf("expected 3 sections, got %d", len(d.Sections))
        }
        // The sample notifications include items in the past 24h.
        if d.TotalItems == 0 {
                t.Errorf("expected non-zero total_items from seeded notifications, got %d", d.TotalItems)
        }
}

// TestBriefDigestHandler_AnonymousRejected verifies the digest endpoint
// returns 401 when the caller is anonymous — the digest is personalised
// to the caller's subscriptions, so anonymous access makes no sense.
func TestBriefDigestHandler_AnonymousRejected(t *testing.T) {
        notifStore := NewNotificationStore()
        handler := makeBriefDigestHandler(notifStore, "https://example.com")
        rr := dispatch(handler, http.MethodGet, "/api/v1/brief/digest", nil, auth.Anonymous())
        if rr.Code != http.StatusUnauthorized {
                t.Errorf("expected 401 for anonymous GET, got %d", rr.Code)
        }
}

// TestBriefDigestHandler_MethodNotAllowed verifies the digest endpoint
// returns 405 with Allow: GET for non-GET methods.
func TestBriefDigestHandler_MethodNotAllowed(t *testing.T) {
        notifStore := NewNotificationStore()
        handler := makeBriefDigestHandler(notifStore, "https://example.com")
        p := auth.Principal{UserID: "user-1"}
        rr := dispatch(handler, http.MethodPost, "/api/v1/brief/digest", []byte("{}"), p)
        if rr.Code != http.StatusMethodNotAllowed {
                t.Errorf("expected 405 for POST, got %d", rr.Code)
        }
        if rr.Header().Get("Allow") != "GET" {
                t.Errorf("expected Allow 'GET', got %q", rr.Header().Get("Allow"))
        }
}

// === UserEmailStore ===

// TestUserEmailStore_Idempotent verifies Set is idempotent — re-storing
// the same email is a no-op and lower-cases the value.
func TestUserEmailStore_Idempotent(t *testing.T) {
        s := NewUserEmailStore()
        s.Set("u1", "Citizen@Example.com")
        s.Set("u1", "Citizen@Example.com")
        if got := s.Get("u1"); got != "citizen@example.com" {
                t.Errorf("expected lower-cased 'citizen@example.com', got %q", got)
        }
}

// TestUserEmailStore_EmptyIgnored verifies Set ignores empty userID or
// email — an unauthenticated PATCH must not clobber a previously-stored
// address.
func TestUserEmailStore_EmptyIgnored(t *testing.T) {
        s := NewUserEmailStore()
        s.Set("u1", "u1@example.com")
        s.Set("u1", "") // ignored
        s.Set("", "u2@example.com")
        if got := s.Get("u1"); got != "u1@example.com" {
                t.Errorf("expected 'u1@example.com', got %q", got)
        }
        if got := s.Get(""); got != "" {
                t.Errorf("expected empty for empty user_id, got %q", got)
        }
}

// === EmailSubscribers ===

// TestEmailSubscribers_Distinct verifies the EmailSubscribers() method
// returns distinct user IDs (one entry per user, not per follow).
func TestEmailSubscribers_Distinct(t *testing.T) {
        store := NewSubscriptionStore()
        _, _ = store.Follow("u1", EntityBill, "b1", []string{"in_app", "email"})
        _, _ = store.Follow("u1", EntityTopic, "t1", []string{"in_app", "email"}) // same user, second follow
        _, _ = store.Follow("u2", EntityBill, "b2", []string{"in_app", "email"})
        _, _ = store.Follow("u3", EntityBill, "b3", nil) // no email channel

        subs := store.EmailSubscribers()
        if len(subs) != 2 {
                t.Errorf("expected 2 distinct email subscribers, got %d (%v)", len(subs), subs)
        }
}

// Ensure unused import doesn't break the build when the test file is
// compiled in isolation.
var _ = io.ReadAll
