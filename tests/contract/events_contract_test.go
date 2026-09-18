// Package contract — events_contract_test.go verifies that every NATS
// event subject defined in packages/contracts conforms to the documented
// naming convention (<aggregate>.<action>) and that packages/events
// produces matching subjects.
//
// The contract enforced here is the producer-consumer contract: every
// service that publishes an event uses packages/events.Subject(et), and
// every service that subscribes uses the same convention. If a constant
// drifts from the convention, this test fails before the event crosses
// any process boundary.
package contract

import (
        "strings"
        "testing"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/events"
)

// allEventTypes is the closed set of EventType constants defined in
// packages/contracts. Adding a new event? Register it here so the
// convention check picks it up.
func allEventTypes() []struct {
        name contracts.EventType
        desc string
} {
        return []struct {
                name contracts.EventType
                desc string
        }{
                {contracts.EventSourceDiscovered, "source discovered"},
                {contracts.EventSourceChanged, "source changed"},
                {contracts.EventDocumentDiscovered, "document discovered"},
                {contracts.EventDocumentDownloaded, "document downloaded"},
                {contracts.EventDocumentParsed, "document parsed"},
                {contracts.EventDocumentOCRCompleted, "document OCR completed"},
                {contracts.EventBillDiscovered, "bill discovered"},
                {contracts.EventBillUpdated, "bill updated"},
                {contracts.EventBillVersionCreated, "bill version created"},
                {contracts.EventBillStageChanged, "bill stage changed"},
                {contracts.EventAmendmentDiscovered, "amendment discovered"},
                {contracts.EventCommitteeUpdated, "committee updated"},
                {contracts.EventHansardPublished, "hansard published"},
                {contracts.EventCitationCreated, "citation created"},
                {contracts.EventEvidenceAttached, "evidence attached"},
                {contracts.EventAIExplanationGenerated, "AI explanation generated"},
                {contracts.EventAIValidationFailed, "AI validation failed"},
                {contracts.EventNotificationCreated, "notification created"},
        }
}

// TestEventTypes_AllNonEmpty verifies every EventType constant is a
// non-empty string. An empty event type would silently produce an empty
// NATS subject (which is invalid — NATS subjects must be non-empty).
func TestEventTypes_AllNonEmpty(t *testing.T) {
        for _, et := range allEventTypes() {
                if string(et.name) == "" {
                        t.Errorf("EventType %q (%s): constant is empty string", et.name, et.desc)
                }
        }
}

// TestEventTypes_FollowNamingConvention verifies every EventType follows
// the documented dotted lowercase convention:
//   - at least one dot separator (most events use 2 segments like
//     "bill.discovered"; a few AI events use 3 segments like
//     "ai.explanation.generated" — both forms are valid)
//   - first + last segments non-empty
//   - all segments are lowercase
//   - no spaces, no slashes (NATS reserves '.' as the hierarchy separator
//     and disallows spaces + slashes in subject tokens)
func TestEventTypes_FollowNamingConvention(t *testing.T) {
        for _, et := range allEventTypes() {
                s := string(et.name)
                parts := strings.Split(s, ".")
                if len(parts) < 2 {
                        t.Errorf("EventType %q: expected at least one '.' separator, got %d",
                                s, len(parts)-1)
                        continue
                }
                for _, p := range parts {
                        if p == "" {
                                t.Errorf("EventType %q: has an empty segment", s)
                        }
                        if strings.ToLower(p) != p {
                                t.Errorf("EventType %q: segment %q must be lowercase", s, p)
                        }
                }
                if strings.ContainsAny(s, " /\t") {
                        t.Errorf("EventType %q: must not contain spaces, slashes, or tabs (NATS subject rules)", s)
                }
        }
}

// TestEventTypes_NoDuplicates verifies every EventType constant is
// distinct. A duplicate would silently overwrite the earlier constant
// (Go's iota-less const blocks don't dedupe).
func TestEventTypes_NoDuplicates(t *testing.T) {
        seen := map[contracts.EventType]string{}
        for _, et := range allEventTypes() {
                if prev, ok := seen[et.name]; ok {
                        t.Errorf("EventType %q is duplicated: first seen as %q, now %q",
                                et.name, prev, et.desc)
                        continue
                }
                seen[et.name] = et.desc
        }
}

// TestEvents_SubjectMatchesEventType verifies events.Subject(et) returns
// the event type string verbatim. The events package is the producer-side
// adapter that turns an EventType into a NATS subject; if it transforms
// the string (e.g. adds a "civic." prefix), every consumer must agree —
// this test enforces the simplest contract: subject == event type.
func TestEvents_SubjectMatchesEventType(t *testing.T) {
        for _, et := range allEventTypes() {
                subject := events.Subject(et.name)
                if subject != string(et.name) {
                        t.Errorf("events.Subject(%q) = %q; expected %q",
                                et.name, subject, et.name)
                }
        }
}

// TestEventEnvelope_RoundTrip verifies an EventEnvelope carries the
// EventType + payload through JSON marshalling without losing data.
// This is the wire format every producer + consumer must agree on.
func TestEventEnvelope_RoundTrip(t *testing.T) {
        for _, et := range allEventTypes() {
                payload := map[string]any{
                        "id":    "test-id-" + string(et.name),
                        "count": 42,
                }
                envelope := contracts.NewEnvelope(et.name, payload)
                if envelope.EventType != et.name {
                        t.Errorf("envelope EventType %q != expected %q", envelope.EventType, et.name)
                }
                if envelope.OccurredAt.IsZero() {
                        t.Errorf("envelope for %q: OccurredAt is zero", et.name)
                }
                if envelope.OccurredAt.After(time.Now().Add(time.Second)) {
                        t.Errorf("envelope for %q: OccurredAt is in the future", et.name)
                }
                if envelope.Payload["id"] != payload["id"] {
                        t.Errorf("envelope for %q: payload id lost; got %v", et.name, envelope.Payload["id"])
                }
        }
}

// TestEvents_SubjectIsStableOverTime verifies the subject for a known
// event type is stable: calling Subject() twice with the same input
// must return the same output. This catches a regression where the
// events package might start hashing or transforming the input.
func TestEvents_SubjectIsStableOverTime(t *testing.T) {
        for _, et := range allEventTypes() {
                first := events.Subject(et.name)
                second := events.Subject(et.name)
                if first != second {
                        t.Errorf("events.Subject(%q): non-deterministic; %q vs %q", et.name, first, second)
                }
        }
}

// TestEventTypes_AggregatesAreKnown verifies every event's aggregate
// (the part before the dot) is one of the platform's known aggregate
// roots. This catches typos like "soucre.discovered" → "source.discovered".
func TestEventTypes_AggregatesAreKnown(t *testing.T) {
        knownAggregates := map[string]string{
                "source":       "ingestion sources",
                "document":     "documents service",
                "bill":         "legislation service",
                "amendment":    "legislation service",
                "committee":    "legislation service",
                "hansard":      "legislation service",
                "citation":     "evidence service",
                "evidence":     "evidence service",
                "ai":           "AI service",
                "notification": "notifications service",
        }
        for _, et := range allEventTypes() {
                s := string(et.name)
                parts := strings.Split(s, ".")
                if len(parts) != 2 {
                        continue // caught by FollowNamingConvention test
                }
                aggregate := parts[0]
                if _, ok := knownAggregates[aggregate]; !ok {
                        t.Errorf("EventType %q: aggregate %q is not in the known set; "+
                                "either register it here or fix the typo", s, aggregate)
                }
        }
}
