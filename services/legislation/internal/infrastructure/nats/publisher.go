// Package natspub contains the NATS-backed implementation of the
// legislation service's EventPublisher. Domain code never imports NATS;
// only this package does.
package natspub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	nats "github.com/nats-io/nats.go"
)

// Publisher implements domain.EventPublisher over NATS.
type Publisher struct {
	nc     *nats.Conn
	source string
}

// New constructs a Publisher.
func New(nc *nats.Conn, source string) *Publisher {
	return &Publisher{nc: nc, source: source}
}

// Publish marshals the event envelope and publishes it to the canonical
// subject derived from the event type. The publisher sets Source if empty
// and ensures OccurredAt is set.
func (p *Publisher) Publish(ctx context.Context, event contracts.Event) error {
	if p == nil || p.nc == nil {
		return fmt.Errorf("nats publisher: not connected")
	}
	if event.Source == "" {
		event.Source = p.source
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	if event.Subject == "" {
		event.Subject = "civic." + string(event.Type)
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	// Use the request context's deadline if any. NATS Go client doesn't
	// accept contexts in Publish, so we honor context cancellation by
	// checking Done before publishing.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return p.nc.Publish(event.Subject, payload)
}
