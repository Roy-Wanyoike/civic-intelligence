// Package events is the home for shared event helper utilities. Concrete
// event type constants and payloads live in packages/contracts so that
// domain layers may reference them without pulling in any message-bus driver
// dependencies. This package provides the small Publisher/Subscriber
// interfaces that every service's domain layer depends on; concrete NATS
// implementations live in infrastructure.
package events

import (
	"context"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// Publisher is the abstraction that every domain service uses to emit events.
// The concrete implementation (NATSPublisher) lives in infrastructure.
// Domain code never imports the NATS client library.
type Publisher interface {
	// Publish marshals the event envelope and writes it to the underlying
	// bus. The subject is derived from contracts.Event.Subject; if empty,
	// implementations should default to the event type.
	Publish(ctx context.Context, event contracts.Event) error
}

// Subscriber is the abstraction that every domain service uses to consume
// events. Handlers are registered by event type; the implementation routes
// each delivered message to the appropriate handler.
type Subscriber interface {
	// Subscribe registers a handler for the given event type. Multiple
	// handlers per type are permitted.
	Subscribe(eventType contracts.EventType, handler Handler) (unsubscribe func())
}

// Handler processes a single event. Implementations MUST return a non-nil
// error to trigger redelivery (subject to the bus's redelivery policy).
type Handler func(ctx context.Context, event contracts.Event) error

// NopPublisher is a no-op Publisher useful for tests.
type NopPublisher struct{}

// Publish implements Publisher.
func (NopPublisher) Publish(_ context.Context, _ contracts.Event) error { return nil }

// NopSubscriber is a no-op Subscriber useful for tests.
type NopSubscriber struct{}

// Subscribe implements Subscriber.
func (NopSubscriber) Subscribe(_ contracts.EventType, _ Handler) func() { return func() {} }

// SubjectFor returns the canonical NATS subject for an event type. The
// subject format is "civic.<domain>.<action>" derived from the event type's
// dot-separated segments (e.g. "bill.stage_changed" -> "civic.bill.stage_changed").
func SubjectFor(t contracts.EventType) string {
	return "civic." + string(t)
}
