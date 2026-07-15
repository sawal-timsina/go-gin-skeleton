package events

import (
	"context"

	"boilerplate-api/lib/config"
)

// noopBroker is selected when NATS_URL is empty. Publishes are logged at debug
// and dropped; subscriptions are inert. This lets the service boot and run its
// HTTP/gRPC surfaces without a broker present.
type noopBroker struct {
	logger config.Logger
}

// NewNoopBroker returns a Broker that discards events.
func NewNoopBroker(logger config.Logger) Broker {
	return &noopBroker{logger: logger}
}

func (b *noopBroker) Publish(_ context.Context, eventType string, _ any) error {
	b.logger.Debugf("events(noop): dropping %q — no broker configured", eventType)
	return nil
}

func (b *noopBroker) Subscribe(_ context.Context, subjectPattern, _ string, _ Handler) (Subscription, error) {
	b.logger.Debugf("events(noop): ignoring subscription to %q — no broker configured", subjectPattern)
	return noopSubscription{}, nil
}

func (b *noopBroker) Close() error { return nil }

type noopSubscription struct{}

func (noopSubscription) Close() error { return nil }
