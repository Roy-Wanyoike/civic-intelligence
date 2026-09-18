package domain

import (
        "context"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// EventPublisher is the abstract event bus the application layer uses to
// emit legislation events (bill.created, bill.stage_changed, ...). Domain
// code never imports the NATS client.
type EventPublisher interface {
        Publish(ctx context.Context, event contracts.Event) error
}

// Clock abstracts time so handlers can be deterministic in tests.
type Clock interface {
        Now() time.Time
}

// SystemClock is the production Clock.
type SystemClock struct{}

// Now implements Clock.
func (SystemClock) Now() time.Time { return time.Now().UTC() }

// IDGenerator abstracts ID creation so handlers can be deterministic in tests.
type IDGenerator interface {
        New() ID
}

// Logger is the domain logger interface (avoids importing zerolog directly).
type Logger interface {
        Info(msg string, fields ...Field)
        Error(msg string, err error, fields ...Field)
}

// Field is a structured log field carried by the domain logger.
type Field struct {
        Key   string
        Value any
}

// NopLogger is a no-op Logger.
type NopLogger struct{}

// Info implements Logger.
func (NopLogger) Info(string, ...Field) {}
// Error implements Logger.
func (NopLogger) Error(string, error, ...Field) {}
