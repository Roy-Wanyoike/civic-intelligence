// Package events provides typed event definitions + helpers for publishing
// to NATS JetStream. Each service that publishes events imports this package.
package events

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// Type alias for convenience — the canonical EventType lives in contracts.
type EventType = contracts.EventType

// Subject returns the dotted NATS subject for an event type.
func Subject(et contracts.EventType) string {
	return string(et)
}
