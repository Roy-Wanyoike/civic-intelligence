// Package contracts — event types published/consumed across services.
//
// Naming convention: <aggregate>.<action> (dotted, lowercase).
// Commands say "do something". Events say "something happened".
// Don't use events as hidden commands.
package contracts

import "time"

// EventType is a string identifying an event in the NATS subject hierarchy.
type EventType string

const (
	EventSourceDiscovered        EventType = "source.discovered"
	EventSourceChanged           EventType = "source.changed"
	EventDocumentDiscovered      EventType = "document.discovered"
	EventDocumentDownloaded      EventType = "document.downloaded"
	EventDocumentParsed          EventType = "document.parsed"
	EventDocumentOCRCompleted    EventType = "document.ocr_completed"
	EventBillDiscovered          EventType = "bill.discovered"
	EventBillUpdated             EventType = "bill.updated"
	EventBillVersionCreated      EventType = "bill.version_created"
	EventBillStageChanged        EventType = "bill.stage_changed"
	EventAmendmentDiscovered     EventType = "amendment.discovered"
	EventCommitteeUpdated        EventType = "committee.updated"
	EventHansardPublished         EventType = "hansard.published"
	EventCitationCreated         EventType = "citation.created"
	EventEvidenceAttached        EventType = "evidence.attached"
	EventAIExplanationGenerated  EventType = "ai.explanation.generated"
	EventAIValidationFailed      EventType = "ai.validation.failed"
	EventNotificationCreated     EventType = "notification.created"
)

// EventEnvelope wraps every event with a correlation ID and timestamp.
type EventEnvelope struct {
	CorrelationID string          `json:"correlation_id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	EventType     EventType       `json:"event_type"`
	Payload       map[string]any  `json:"payload"`
}

// NewEnvelope constructs an envelope with a fresh timestamp.
func NewEnvelope(et EventType, payload map[string]any) EventEnvelope {
	return EventEnvelope{
		OccurredAt: time.Now().UTC(),
		EventType:  et,
		Payload:    payload,
	}
}
