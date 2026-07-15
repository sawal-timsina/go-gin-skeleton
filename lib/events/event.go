// Package events provides an async messaging abstraction for publishing and
// consuming domain events between microservices. A NATS JetStream backend is
// provided for durable, at-least-once delivery; a no-op backend is selected
// automatically when no broker is configured so the service runs standalone.
//
// The envelope is CloudEvents-inspired and carries a W3C traceparent so a
// consumer can continue the producer's distributed trace.
package events

import (
	"context"
	"encoding/json"
	"time"
)

// Event is the wire envelope. Data holds the domain-specific payload as raw
// JSON so the transport stays schema-agnostic; consumers unmarshal it with
// UnmarshalData into their expected type.
type Event struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Source      string          `json:"source"`
	Time        time.Time       `json:"time"`
	TraceParent string          `json:"traceparent,omitempty"`
	Data        json.RawMessage `json:"data"`
}

// UnmarshalData decodes the event payload into v.
func (e Event) UnmarshalData(v any) error {
	return json.Unmarshal(e.Data, v)
}

// Handler processes a single delivered event. Returning an error signals the
// broker to negatively-acknowledge so the message is redelivered; returning
// nil acknowledges successful processing.
type Handler func(ctx context.Context, e Event) error

// Subscription is a live consumer that can be torn down on shutdown.
type Subscription interface {
	Close() error
}

// Broker is the async messaging contract. Implementations MUST be safe for
// concurrent use.
type Broker interface {
	// Publish wraps data in an Event of the given type and publishes it. The
	// concrete subject is derived from the configured prefix and eventType
	// (e.g. "gin-skeleton.user.created").
	Publish(ctx context.Context, eventType string, data any) error

	// Subscribe consumes events whose subject matches subjectPattern (NATS
	// wildcards such as "gin-skeleton.user.>" are supported). durable names a
	// persistent consumer so redeliveries survive restarts and scale across
	// replicas as a queue group.
	Subscribe(ctx context.Context, subjectPattern, durable string, handler Handler) (Subscription, error)

	// Close releases the underlying connection.
	Close() error
}
