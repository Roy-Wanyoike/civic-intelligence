// Package natspub contains the candidate-fact consumer for the legislation
// service. It listens to the EventCandidateFactProposed stream and dispatches
// accepted facts to the AcceptCandidateFactHandler.
package natspub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/application"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
	nats "github.com/nats-io/nats.go"
)

// CandidateFactConsumer subscribes to the candidate fact proposed subject
// and invokes the accept handler. Rejected candidates are not retried by
// default; redelivery is delegated to NATS JetStream.
type CandidateFactConsumer struct {
	nc      *nats.Conn
	accept  *application.AcceptCandidateFactHandler
}

// NewCandidateFactConsumer constructs the consumer.
func NewCandidateFactConsumer(nc *nats.Conn, accept *application.AcceptCandidateFactHandler) *CandidateFactConsumer {
	return &CandidateFactConsumer{nc: nc, accept: accept}
}

// Start subscribes and returns immediately. The subscription runs in the
// background; the returned function unsubscribes.
func (c *CandidateFactConsumer) Start(ctx context.Context) (func(), error) {
	if c == nil || c.nc == nil {
		return nil, fmt.Errorf("candidate fact consumer: not connected")
	}
	sub, err := c.nc.Subscribe("civic.ai.candidate_fact.proposed", func(m *nats.Msg) {
		c.handle(ctx, m)
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe candidate facts: %w", err)
	}
	return func() { _ = sub.Unsubscribe() }, nil
}

// handle processes a single message. It never panics on a malformed payload;
// it acks-nak's instead, which lets NATS redeliver.
func (c *CandidateFactConsumer) handle(ctx context.Context, m *nats.Msg) {
	var event contracts.Event
	if err := json.Unmarshal(m.Data, &event); err != nil {
		_ = m.Nak()
		return
	}
	var payload contracts.CandidateFactProposedEvent
	raw, _ := json.Marshal(event.Data)
	if err := json.Unmarshal(raw, &payload); err != nil {
		_ = m.Nak()
		return
	}
	if payload.BillID == "" {
		// No bill target — nothing for legislation to do. Ack to drop.
		_ = m.Ack()
		return
	}
	err := c.accept.Handle(ctx, application.AcceptCandidateFactCommand{
		CandidateFactID: payload.CandidateFactID,
		BillID:          domain.ID(payload.BillID),
		Subject:         payload.Subject,
		Claim:           payload.Claim,
		Validator:       "evidence_v1",
	})
	if err != nil {
		_ = m.Nak()
		return
	}
	_ = m.Ack()
}
